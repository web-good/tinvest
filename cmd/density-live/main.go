// density-live checks an account read-only. Real trading runs in the shared application.
package main

import (
	"context"
	"crypto/tls"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"tinvest/internal/service/screener/density"
	"tinvest/internal/service/screener/density/live"
	marketgrpc "tinvest/pkg/client/grpc"

	"github.com/joho/godotenv"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	if err := run(ctx, os.Args[1:], os.Stdout); err != nil && !errors.Is(err, flag.ErrHelp) {
		fmt.Fprintln(os.Stderr, "density-live:", err)
		os.Exit(1)
	}
}

func run(ctx context.Context, args []string, out io.Writer) (retErr error) {
	fs := flag.NewFlagSet("density-live", flag.ContinueOnError)
	fs.SetOutput(out)
	envFile := fs.String("env-file", "", "optional dotenv file; existing environment wins")
	accountFlag := fs.String("account", "", "account to check; otherwise DENSITY_ACCOUNT_ID")
	tickersFlag := fs.String("tickers", "T,SBER", "MOEX tickers to check; 1..10")
	classic := fs.Bool("tls-classic", false, "TLS X25519/P-256 compatibility; certificate verification stays enabled")
	trade := fs.Bool("trade", false, "retired: real trading must use the shared application runner")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *trade {
		return fmt.Errorf("standalone trading is disabled: use the shared application runner with DENSITY_LIVE_ENABLED and DENSITY_LIVE_TRADE_ENABLED")
	}
	if fs.NArg() != 0 {
		return fmt.Errorf("unexpected positional arguments")
	}
	var tickers []string
	seen := map[string]bool{}
	for _, t := range strings.Split(*tickersFlag, ",") {
		t = strings.ToUpper(strings.TrimSpace(t))
		if t != "" && !seen[t] {
			tickers = append(tickers, t)
			seen[t] = true
		}
	}
	if len(tickers) == 0 || len(tickers) > 10 {
		return fmt.Errorf("tickers must contain 1..10 explicit instruments")
	}
	if *envFile != "" {
		if err := godotenv.Load(*envFile); err != nil {
			return fmt.Errorf("dotenv: %w", err)
		}
	}
	account := strings.TrimSpace(*accountFlag)
	if account == "" {
		account = strings.TrimSpace(os.Getenv("DENSITY_ACCOUNT_ID"))
	}
	if account == "" {
		return fmt.Errorf("DENSITY_ACCOUNT_ID or -account is required for this read-only check")
	}
	token := os.Getenv("DENSITY_LIVE_TOKEN")
	if token == "" {
		token = os.Getenv("RSI_PULLBACK_TOKEN")
	}
	if token == "" {
		token = os.Getenv("T_BANK")
	}
	if token == "" {
		return fmt.Errorf("DENSITY_LIVE_TOKEN, RSI_PULLBACK_TOKEN or T_BANK is required")
	}
	tc := &tls.Config{MinVersion: tls.VersionTLS12}
	if *classic {
		tc.CurvePreferences = []tls.CurveID{tls.X25519, tls.CurveP256}
	}
	conn, err := grpc.NewClient("invest-public-api.tinkoff.ru:443", grpc.WithTransportCredentials(credentials.NewTLS(tc)))
	if err != nil {
		return err
	}
	defer func() { retErr = errors.Join(retErr, conn.Close()) }()
	market := marketgrpc.NewDensityMarketData(conn, token)
	inputs, err := market.InstrumentsByTickers(ctx, tickers)
	if err != nil {
		return err
	}
	var instruments []density.Instrument
	var ids []string
	for _, in := range inputs {
		instruments = append(instruments, in.Instrument)
		ids = append(ids, in.Instrument.ID)
	}
	broker := marketgrpc.NewDensityBroker(conn, token, account, false, instruments)
	if err = broker.CheckAccount(ctx); err != nil {
		return err
	}
	refs, err := market.ReferencePrices(ctx, ids)
	if err != nil {
		return err
	}
	if err = checkAccount(ctx, broker, instruments, refs, out); err != nil {
		return err
	}
	_, err = fmt.Fprintln(out, "CHECK ONLY complete. No orders placed/canceled, no state journal opened. Real density trading runs in the shared application.")
	return err
}

func checkAccount(ctx context.Context, broker live.Broker, instruments []density.Instrument, refs map[string]density.Price, out io.Writer) error {
	snapshot, err := broker.Account(ctx)
	if err != nil {
		return err
	}
	for _, i := range instruments {
		busy := snapshot.Holdings[i.ID].Units != 0
		for _, o := range snapshot.Orders {
			if o.InstrumentID == i.ID {
				busy = true
			}
		}
		for _, uid := range snapshot.Stops {
			if uid == i.ID {
				busy = true
			}
		}
		ref := live.Price(refs[i.ID])
		allowed := false
		if !busy && ref > 0 {
			allowed, err = broker.CanBuy(ctx, i.ID, ref, 1)
			if err != nil {
				return err
			}
		}
		if _, err = fmt.Fprintf(out, "CHECK %s: 1 lot=%d shares, tick=%.9g RUB, reference_lot_cost~=%.2f RUB, busy=%t, own_cash_allows=%t\n", i.Ticker, i.Lot, float64(live.Price(i.PriceStep))/1e9, float64(ref)*float64(i.Lot)/1e9, busy, allowed); err != nil {
			return err
		}
	}
	return nil
}
