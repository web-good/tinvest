//go:build ignore

// Research-only complete-candle top-up, using cached instrument identity.
package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/joho/godotenv"
	"tinvest/internal/enum"
	s "tinvest/internal/service/backtest"
	grpc "tinvest/pkg/client/grpc"
	"tinvest/pkg/logger"
)

func main() {
	logger.Init()
	_ = godotenv.Load("./env/local.env")
	_ = godotenv.Load("./env/token.env")
	token := os.Getenv("T_BANK")
	if token == "" {
		panic("T_BANK unavailable")
	}
	client, e := grpc.NewClientGrpc("invest-public-api.tinkoff.ru:443", token)
	if e != nil {
		panic(e)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()
	provider := s.NewCandleProvider(client.MarketDataServiceClient(), "data/candles")
	loc, e := time.LoadLocation("Europe/Moscow")
	if e != nil {
		panic(e)
	}
	from := time.Date(2026, 10, 15, 0, 0, 0, 0, loc)
	to := time.Date(2026, 11, 15, 0, 0, 0, 0, loc)
	for _, iv := range []enum.Interval{enum.Minutes30, enum.Day1} {
		bars, e := provider.Load(ctx, "NLMK", "161eb0d0-aaac-4451-b374-f5d0eeb1b508", iv, from, to, false)
		if e != nil {
			fmt.Fprintln(os.Stderr, e)
			os.Exit(1)
		}
		fmt.Printf("%s: %d complete candles, %s through %s\n", iv.String(), len(bars), bars[0].Time, bars[len(bars)-1].Time)
	}
}
