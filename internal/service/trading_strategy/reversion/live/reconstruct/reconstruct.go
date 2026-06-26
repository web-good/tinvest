// Package reconstruct rebuilds reversion entry-state from the broker API when the
// local state file is missing but a position is open. EntryPrice uses the broker's
// average purchase price; EntryTime is the most recent BUY fill; EntryATR and MaxFav
// are recomputed from hourly candles around/after entry.
package reconstruct

import (
	"context"
	"fmt"
	"time"

	"google.golang.org/protobuf/types/known/timestamppb"

	"tinvest/internal/enum"
	imodel "tinvest/internal/model"
	"tinvest/internal/service/trading_strategy/reversion/live/marketdata"
	"tinvest/internal/service/trading_strategy/reversion/live/statestore"
	"tinvest/internal/utils"
	grpcmodel "tinvest/pkg/client/grpc/model"
	"tinvest/pkg/indicators"
)

// TradesClient is the slice of the operations client reconstruct needs.
type TradesClient interface {
	GetInstrumentTrades(ctx context.Context, accountID, instrumentID string, from, to time.Time) ([]grpcmodel.Trade, error)
}

// Entry reconstructs the entry-state for an open position.
func Entry(ctx context.Context, tc TradesClient, cc marketdata.CandleClient,
	accountID, instrumentID, ticker string, purchasePrice float64,
	atrPeriod, lookbackBars int, now time.Time) (statestore.Entry, error) {

	trades, err := tc.GetInstrumentTrades(ctx, accountID, instrumentID, now.AddDate(0, 0, -120), now)
	if err != nil {
		return statestore.Entry{}, fmt.Errorf("reconstruct: trades: %w", err)
	}
	var entryTime time.Time
	for _, tr := range trades {
		if tr.IsBuy && tr.Date.After(entryTime) {
			entryTime = tr.Date
		}
	}
	if entryTime.IsZero() {
		return statestore.Entry{}, fmt.Errorf("reconstruct: no BUY fill found for %s", ticker)
	}

	// Hourly candles from before entry (for ATR warm-up) through now (for maxFav).
	from := entryTime.AddDate(0, 0, -(lookbackBars/4 + 10))
	limit := int32(lookbackBars * 6)
	raw, err := cc.GetCandles(ctx, &instrumentID, enum.Hour1.ToNumberInvestApi(),
		timestamppb.New(from), timestamppb.New(now), &limit, true)
	if err != nil {
		return statestore.Entry{}, fmt.Errorf("reconstruct: candles: %w", err)
	}
	candles := completed(raw)

	atr := atrAtEntry(candles, entryTime, atrPeriod, lookbackBars)
	maxFav := purchasePrice
	for _, c := range candles {
		if !c.Time.Before(entryTime) && c.Close > maxFav {
			maxFav = c.Close
		}
	}

	return statestore.Entry{
		Ticker:     ticker,
		EntryTime:  entryTime,
		EntryPrice: purchasePrice,
		EntryATR:   atr,
		MaxFav:     maxFav,
	}, nil
}

type bar struct {
	Time             time.Time
	High, Low, Close float64
}

func completed(raw []*imodel.CandleItemTechAnalyse) []bar {
	out := make([]bar, 0, len(raw))
	for _, c := range raw {
		if !c.IsComplete {
			continue
		}
		out = append(out, bar{
			Time:  c.Time,
			High:  utils.CombinePrice(c.High.Units, c.High.Nano),
			Low:   utils.CombinePrice(c.Low.Units, c.Low.Nano),
			Close: utils.CombinePrice(c.Close.Units, c.Close.Nano),
		})
	}
	return out
}

// atrAtEntry computes ATR over the hourly window ending at the entry bar (the bar at
// or just before entryTime), mirroring how the core stamps EntryATR at entry.
func atrAtEntry(bars []bar, entryTime time.Time, atrPeriod, lookbackBars int) float64 {
	end := -1
	for i, b := range bars {
		if !b.Time.After(entryTime) {
			end = i
		}
	}
	if end < 0 {
		return 0
	}
	start := end - lookbackBars + 1
	if start < 0 {
		start = 0
	}
	window := bars[start : end+1]
	highs := make([]float64, len(window))
	lows := make([]float64, len(window))
	closes := make([]float64, len(window))
	for i, b := range window {
		highs[i], lows[i], closes[i] = b.High, b.Low, b.Close
	}
	return indicators.ATR(highs, lows, closes, atrPeriod)
}
