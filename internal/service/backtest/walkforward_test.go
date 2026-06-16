package backtest

import (
	"testing"
	"time"

	"tinvest/internal/domain/backtest"
)

func date(y int, m time.Month, d int) time.Time {
	return time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
}

func TestWalkForwardFolds(t *testing.T) {
	tests := []struct {
		name                    string
		from, to                time.Time
		trainMonths, testMonths int
		wantFolds               int
		wantErr                 bool
	}{
		{
			name: "24m/12train/3test -> 4 folds",
			from: date(2024, time.January, 1), to: date(2026, time.January, 1),
			trainMonths: 12, testMonths: 3, wantFolds: 4,
		},
		{
			name: "9m/3train/3test -> 2 folds",
			from: date(2025, time.January, 1), to: date(2025, time.October, 1),
			trainMonths: 3, testMonths: 3, wantFolds: 2,
		},
		{
			name: "train+test exceed window -> error",
			from: date(2025, time.January, 1), to: date(2025, time.April, 1),
			trainMonths: 3, testMonths: 3, wantErr: true,
		},
		{
			name: "zero train -> error",
			from: date(2025, time.January, 1), to: date(2025, time.October, 1),
			trainMonths: 0, testMonths: 3, wantErr: true,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			folds, err := walkForwardFolds(tc.from, tc.to, tc.trainMonths, tc.testMonths)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("want error, got %d folds", len(folds))
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(folds) != tc.wantFolds {
				t.Fatalf("folds = %d, want %d", len(folds), tc.wantFolds)
			}
		})
	}
}

func TestWalkForwardFoldsBoundaries(t *testing.T) {
	folds, err := walkForwardFolds(date(2025, time.January, 1), date(2025, time.October, 1), 3, 3)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Fold 0: train Jan-Apr, test Apr-Jul. Fold 1: train Apr-Jul, test Jul-Oct.
	if !folds[0].trainFrom.Equal(date(2025, time.January, 1)) || !folds[0].testTo.Equal(date(2025, time.July, 1)) {
		t.Errorf("fold0 = %+v", folds[0])
	}
	if !folds[1].trainFrom.Equal(date(2025, time.April, 1)) || !folds[1].testTo.Equal(date(2025, time.October, 1)) {
		t.Errorf("fold1 = %+v", folds[1])
	}
}

func TestSliceByRange(t *testing.T) {
	mk := func(h int) backtest.Candle {
		return backtest.Candle{Time: date(2025, time.January, 1).Add(time.Duration(h) * time.Hour)}
	}
	candles := []backtest.Candle{mk(0), mk(1), mk(2), mk(3), mk(4)}
	got := sliceByRange(candles, date(2025, time.January, 1).Add(time.Hour), date(2025, time.January, 1).Add(3*time.Hour))
	// half-open [1h, 3h): expect bars at h=1 and h=2.
	if len(got) != 2 || !got[0].Time.Equal(mk(1).Time) || !got[1].Time.Equal(mk(2).Time) {
		t.Fatalf("sliceByRange = %+v", got)
	}
}

func TestTradesEnteredFrom(t *testing.T) {
	tr := func(h int) backtest.Trade {
		return backtest.Trade{EntryTime: date(2025, time.January, 1).Add(time.Duration(h) * time.Hour)}
	}
	trades := []backtest.Trade{tr(0), tr(1), tr(2), tr(3)}
	boundary := date(2025, time.January, 1).Add(2 * time.Hour)
	got := tradesEnteredFrom(trades, boundary)
	// Keep entries at/after boundary: h=2 and h=3.
	if len(got) != 2 || !got[0].EntryTime.Equal(tr(2).EntryTime) {
		t.Fatalf("tradesEnteredFrom = %+v", got)
	}
}

func TestSumPnL(t *testing.T) {
	trades := []backtest.Trade{{PnL: 100}, {PnL: -40}, {PnL: 10}}
	if got := sumPnL(trades); got != 70 {
		t.Fatalf("sumPnL = %v, want 70", got)
	}
}

func TestTradeReplayDrawdownPct(t *testing.T) {
	// Equity from 1000: +200 -> 1200 (peak), -360 -> 840. DD = (1200-840)/1200 = 0.30.
	trades := []backtest.Trade{{PnL: 200}, {PnL: -360}, {PnL: 60}}
	got := tradeReplayDrawdownPct(trades, 1000)
	if got < 0.2999 || got > 0.3001 {
		t.Fatalf("tradeReplayDrawdownPct = %v, want ~0.30", got)
	}
	if tradeReplayDrawdownPct(nil, 1000) != 0 {
		t.Fatalf("empty trades should give 0 drawdown")
	}
	if tradeReplayDrawdownPct(trades, 0) != 0 {
		t.Fatalf("zero cash should give 0 drawdown (guard)")
	}
}

func TestCompoundReturns(t *testing.T) {
	// (1+0.10)(1-0.05)(1+0.20) - 1 = 0.254.
	got := compoundReturns([]float64{0.10, -0.05, 0.20})
	if got < 0.2539 || got > 0.2541 {
		t.Fatalf("compoundReturns = %v, want ~0.254", got)
	}
	if compoundReturns(nil) != 0 {
		t.Fatalf("empty should give 0")
	}
}
