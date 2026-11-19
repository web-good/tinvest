// bybit-density scans public Bybit books and runs a virtual trading session.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"tinvest/internal/service/screener/bybitdensity/feed"
	"tinvest/internal/service/screener/bybitdensity/journal"
	"tinvest/internal/service/screener/bybitdensity/model"

	"golang.org/x/sync/errgroup"
)

type marketData interface {
	Instruments(context.Context, model.Market, []string, int) ([]model.Instrument, error)
	Stream(context.Context, []model.Instrument, int, func(model.Event) error) error
}

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	if err := run(ctx, os.Args[1:], os.Stdout); err != nil && !errors.Is(err, flag.ErrHelp) {
		fmt.Fprintln(os.Stderr, "bybit-density:", err)
		os.Exit(1)
	}
}

func run(ctx context.Context, args []string, out io.Writer) error {
	op, err := readOptions(args, out)
	if err != nil {
		return err
	}
	market := feed.NewClient(op.restURL, op.wsURL)
	if op.checkFeed {
		return checkFeed(ctx, market, op, out)
	}
	send, err := telegramSender(op.telegramToken)
	if err != nil {
		return err
	}
	screener, simulation := destinations(send, op.screener, op.simulation)
	return serve(ctx, market, op, screener, simulation)
}

func serve(ctx context.Context, market marketData, op options, screener, simulation sendMessage) (retErr error) {
	session, err := journal.Open(op.stateDir)
	if err != nil {
		return fmt.Errorf("open simulation state: %w", err)
	}
	defer func() { retErr = errors.Join(retErr, session.Close()) }()
	hub := newAlertHub(op.strategy, session)
	group, ctx := errgroup.WithContext(ctx)
	group.Go(func() error { return session.Deliver(ctx, simulation) })
	group.Go(func() error { return hub.sendSignals(ctx, screener) })
	group.Go(func() error { return hub.expire(ctx) })
	group.Go(func() error { return runMarket(ctx, market, model.Spot, op.spotSymbols, op, hub) })
	group.Go(func() error { return runMarket(ctx, market, model.Linear, op.linearSymbols, op, hub) })
	return group.Wait()
}

func runMarket(ctx context.Context, market marketData, category model.Market, symbols []string, op options, hub *alertHub) error {
	var instruments []model.Instrument
	var retry time.Duration
	for ctx.Err() == nil {
		if len(instruments) == 0 {
			var err error
			instruments, err = market.Instruments(ctx, category, symbols, op.symbolLimit)
			if errors.Is(err, feed.ErrInvalidSymbols) {
				return fmt.Errorf("%s instrument configuration: %w", category, err)
			}
			if ctx.Err() != nil {
				return nil
			}
			if err != nil || len(instruments) == 0 {
				slog.Warn("Bybit instrument selection failed", "market", category, "error", err)
				retry = nextRetry(retry)
				if !waitRetry(ctx, retry) {
					return nil
				}
				continue
			}
			hub.add(instruments)
			retry = 0
			slog.Info("Bybit instruments selected", "market", category, "count", len(instruments))
		}
		started := time.Now()
		var recordErr error
		err := market.Stream(ctx, instruments, op.depth, func(event model.Event) error {
			recordErr = hub.apply(event, time.Now())
			return recordErr
		})
		hub.reset(instruments)
		if recordErr != nil {
			return fmt.Errorf("save simulated execution: %w", recordErr)
		}
		if ctx.Err() != nil {
			return nil
		}
		if time.Since(started) >= 30*time.Second {
			retry = 0
		}
		retry = nextRetry(retry)
		slog.Warn("Bybit stream disconnected", "market", category, "error", err, "retry", retry)
		if !waitRetry(ctx, retry) {
			return nil
		}
	}
	return nil
}

func nextRetry(previous time.Duration) time.Duration {
	if previous == 0 {
		return time.Second
	}
	return min(previous*2, 30*time.Second)
}

func waitRetry(ctx context.Context, delay time.Duration) bool {
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-timer.C:
		return true
	case <-ctx.Done():
		return false
	}
}
