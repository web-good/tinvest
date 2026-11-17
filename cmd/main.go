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
)

func main() {
	ctx := context.Background()
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
