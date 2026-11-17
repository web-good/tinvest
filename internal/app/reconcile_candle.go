package app

import (
	"context"
	"flag"
	"fmt"
	"io"
	"strings"
	"time"

	densitylive "tinvest/internal/service/screener/density/live"
	"tinvest/internal/service_provider"
	marketgrpc "tinvest/pkg/client/grpc"
)

// ReconcileCandle confirms an external close without running any strategy or
// submitting or canceling orders.
func ReconcileCandle(ctx context.Context, args []string, out io.Writer) error {
	flags := flag.NewFlagSet("reconcile-candle", flag.ContinueOnError)
	flags.SetOutput(out)
	tickerFlag := flags.String("ticker", "", "MOEX ticker")
	entryFlag := flags.String("entry-time", "", "exact entryTime from the candle state in RFC3339")
	if err := flags.Parse(args); err != nil {
		return err
	}
	ticker := strings.ToUpper(strings.TrimSpace(*tickerFlag))
	entryTime, err := time.Parse(time.RFC3339Nano, *entryFlag)
	if flags.NArg() != 0 || ticker == "" || err != nil || entryTime.IsZero() {
		return fmt.Errorf("provide -ticker and the exact -entry-time of the externally closed position")
	}
	a := &App{}
	a.initLogger()
	if err := a.initConfig(ctx); err != nil {
		return err
	}
	cfg := a.config.RSIPullback
	if !cfg.Ready() {
		return fmt.Errorf("rsi pullback account and token are required")
	}
	provider := service_provider.GetServiceProvider(ctx, a.config)
	client, err := provider.GetRSIPullbackGrpcClient()
	if err != nil {
		return err
	}
	market := marketgrpc.NewDensityMarketData(client.Connection(), cfg.Token)
	instruments, err := market.InstrumentsByTickers(ctx, []string{ticker})
	if err != nil {
		return err
	}
	id := instruments[0].Instrument.ID
	broker := marketgrpc.NewDensityBroker(client.Connection(), cfg.Token, cfg.AccountID, false, nil)
	err = provider.GetAccountCoordinator().ConfirmClosedCandle(ctx, ticker, id, entryTime, func(ctx context.Context) error {
		account, err := broker.Account(ctx)
		if err != nil {
			return err
		}
		return checkClosedInstrument(account, id)
	})
	if err != nil {
		return err
	}
	_, err = fmt.Fprintf(out, "%s: external candle close recorded\n", ticker)
	return err
}

func checkClosedInstrument(account densitylive.Account, id string) error {
	if account.Holdings[id].Units != 0 {
		return fmt.Errorf("instrument still has broker inventory")
	}
	for _, order := range account.Orders {
		if order.InstrumentID == id {
			return fmt.Errorf("instrument still has an active order")
		}
	}
	for _, instrumentID := range account.Stops {
		if instrumentID == id {
			return fmt.Errorf("instrument still has an active stop order")
		}
	}
	return nil
}
