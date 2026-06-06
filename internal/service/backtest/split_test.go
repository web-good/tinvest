package backtest

import (
	"testing"
	"time"

	"tinvest/internal/domain/backtest"
)

func TestSplitByTime(t *testing.T) {
	base := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	candles := make([]backtest.Candle, 10)
	for i := range candles {
		candles[i] = backtest.Candle{Time: base.AddDate(0, 0, i)}
	}
	boundary := base.AddDate(0, 0, 6) // first 6 days train, rest test

	train, test := SplitByTime(candles, boundary)
	if len(train) != 6 || len(test) != 4 {
		t.Fatalf("split sizes = %d/%d, want 6/4", len(train), len(test))
	}
	if !train[len(train)-1].Time.Before(boundary) {
		t.Error("last train candle must be before boundary")
	}
	if test[0].Time.Before(boundary) {
		t.Error("first test candle must be at/after boundary")
	}
}
