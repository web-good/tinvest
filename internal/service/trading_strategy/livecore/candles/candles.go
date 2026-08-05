// Package candles holds the market-data client surface the live runners share and the
// conversion from API candles to domain candles. Assembling a MarketData snapshot stays
// per-strategy: reversion needs an hourly window plus 4H, rsi_pullback a 30-minute window
// plus dailies, and the two fetch policies have nothing in common beyond these primitives.
package candles

import (
	"context"

	"google.golang.org/protobuf/types/known/timestamppb"

	"tinvest/internal/domain/backtest"
	imodel "tinvest/internal/model"
	"tinvest/internal/utils"
)

// CandleClient is the slice of the market-data client the assembler needs.
type CandleClient interface {
	GetCandles(ctx context.Context, instrumentUID *string, interval int32,
		from, to *timestamppb.Timestamp, limit *int32, withHoliday bool) ([]*imodel.CandleItemTechAnalyse, error)
}

// ToCandles converts oldest-first API candles to domain candles. When completedOnly is
// true the still-forming trailing bar (IsComplete=false) is dropped.
func ToCandles(in []*imodel.CandleItemTechAnalyse, completedOnly bool) []backtest.Candle {
	out := make([]backtest.Candle, 0, len(in))
	for _, c := range in {
		if completedOnly && !c.IsComplete {
			continue
		}
		out = append(out, backtest.Candle{
			Time:   c.Time,
			Open:   utils.CombinePrice(c.Open.Units, c.Open.Nano),
			High:   utils.CombinePrice(c.High.Units, c.High.Nano),
			Low:    utils.CombinePrice(c.Low.Units, c.Low.Nano),
			Close:  utils.CombinePrice(c.Close.Units, c.Close.Nano),
			Volume: c.Volume,
		})
	}
	return out
}
