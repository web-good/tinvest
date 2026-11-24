package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"tinvest/internal/service/screener/bybitdensity"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	if err := bybitdensity.RunBacktest(ctx, os.Args[1:], os.Stdout); err != nil && !errors.Is(err, flag.ErrHelp) {
		fmt.Fprintln(os.Stderr, "bybit-density-backtest:", err)
		os.Exit(1)
	}
}
