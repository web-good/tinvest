package main

import (
	"context"
	"errors"
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"
	"tinvest/internal/app"
	"tinvest/internal/service/screener/bybitdensity"
)

func main() {
	ctx := context.Background()
	if len(os.Args) > 1 && os.Args[1] == "bybit-check-feed" {
		ctx, stop := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
		defer stop()
		if err := bybitdensity.CheckFeed(ctx, os.Args[2:], os.Stdout); err != nil && !errors.Is(err, flag.ErrHelp) {
			log.Fatalf("bybit-check-feed: %v", err)
		}
		return
	}
	if len(os.Args) > 1 && os.Args[1] == "reconcile-candle" {
		ctx, stop := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
		defer stop()
		if err := app.ReconcileCandle(ctx, os.Args[2:], os.Stdout); err != nil && !errors.Is(err, flag.ErrHelp) {
			log.Fatalf("reconcile-candle: %v", err)
		}
		return
	}
	a, err := app.InitApp(ctx)

	if err != nil {
		log.Fatalf("failed to init app: %s", err.Error())
	}

	err = a.Run(ctx)

	if err != nil {
		log.Fatalf("failed to run app: %s", err.Error())
	}
}
