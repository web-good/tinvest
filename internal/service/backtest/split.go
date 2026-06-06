package backtest

import (
	"time"

	"tinvest/internal/domain/backtest"
)

// SplitByTime partitions oldest-first candles into a train slice (strictly before
// boundary) and a test slice (at/after boundary). Either slice may be empty.
func SplitByTime(candles []backtest.Candle, boundary time.Time) (train, test []backtest.Candle) {
	for i, c := range candles {
		if !c.Time.Before(boundary) {
			return candles[:i], candles[i:]
		}
	}
	return candles, nil
}
