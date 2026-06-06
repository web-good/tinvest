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

func TestSplitByTime_EdgeCases(t *testing.T) {
	base := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	candles := make([]backtest.Candle, 5)
	for i := range candles {
		candles[i] = backtest.Candle{Time: base.AddDate(0, 0, i)}
	}

	t.Run("nil input", func(t *testing.T) {
		train, test := SplitByTime(nil, base)
		if train != nil || test != nil {
			t.Errorf("nil input: got train=%v test=%v, want both nil", train, test)
		}
	})

	t.Run("boundary before all candles", func(t *testing.T) {
		boundary := base.AddDate(0, 0, -1) // before day 0
		train, test := SplitByTime(candles, boundary)
		if len(train) != 0 {
			t.Errorf("train len = %d, want 0", len(train))
		}
		if len(test) != len(candles) {
			t.Errorf("test len = %d, want %d", len(test), len(candles))
		}
	})

	t.Run("boundary after all candles", func(t *testing.T) {
		boundary := base.AddDate(0, 0, len(candles)) // after last day
		train, test := SplitByTime(candles, boundary)
		if len(train) != len(candles) {
			t.Errorf("train len = %d, want %d", len(train), len(candles))
		}
		if len(test) != 0 {
			t.Errorf("test len = %d, want 0", len(test))
		}
	})
}
