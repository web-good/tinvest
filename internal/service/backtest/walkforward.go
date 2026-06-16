package backtest

import (
	"fmt"
	"time"

	"tinvest/internal/domain/backtest"
)

// foldWindow holds the four time boundaries of one rolling walk-forward fold.
// Train and test windows are half-open [from, to).
type foldWindow struct {
	trainFrom, trainTo time.Time
	testFrom, testTo   time.Time
}

// walkForwardFolds enumerates rolling folds over [from, to). The train window has a
// fixed length of trainMonths; it slides forward by testMonths each fold so OOS windows
// abut without overlap. A fold is emitted only while its test window ends at or before to.
func walkForwardFolds(from, to time.Time, trainMonths, testMonths int) ([]foldWindow, error) {
	if trainMonths <= 0 || testMonths <= 0 {
		return nil, fmt.Errorf("backtest: walk-forward needs train-months>0 and test-months>0 (got %d/%d)", trainMonths, testMonths)
	}
	var folds []foldWindow
	for k := 0; ; k++ {
		trainFrom := from.AddDate(0, k*testMonths, 0)
		trainTo := trainFrom.AddDate(0, trainMonths, 0)
		testFrom := trainTo
		testTo := testFrom.AddDate(0, testMonths, 0)
		if testTo.After(to) {
			break
		}
		folds = append(folds, foldWindow{trainFrom, trainTo, testFrom, testTo})
	}
	if len(folds) == 0 {
		return nil, fmt.Errorf("backtest: no full walk-forward fold fits: train-months+test-months (%d) exceeds the -months window", trainMonths+testMonths)
	}
	return folds, nil
}

// sliceByRange returns the candles whose Time falls in the half-open interval [from, to).
func sliceByRange(candles []backtest.Candle, from, to time.Time) []backtest.Candle {
	_, tail := SplitByTime(candles, from) // tail: Time >= from
	head, _ := SplitByTime(tail, to)      // head: Time < to
	return head
}

// tradesEnteredFrom keeps only trades whose entry is at or after t. Used to drop
// warm-up trades whose entry fell inside the train lead-in of an OOS run.
func tradesEnteredFrom(trades []backtest.Trade, t time.Time) []backtest.Trade {
	out := make([]backtest.Trade, 0, len(trades))
	for _, tr := range trades {
		if !tr.EntryTime.Before(t) {
			out = append(out, tr)
		}
	}
	return out
}

// sumPnL totals net PnL across trades.
func sumPnL(trades []backtest.Trade) float64 {
	var s float64
	for _, t := range trades {
		s += t.PnL
	}
	return s
}

// tradeReplayDrawdownPct replays trade PnL as an equity curve starting at initialCash
// and returns the maximum drawdown as a fraction (0–1) of the running peak. Trade-based
// folds have no engine equity curve of their own, so the curve is reconstructed here.
func tradeReplayDrawdownPct(trades []backtest.Trade, initialCash float64) float64 {
	if initialCash <= 0 {
		return 0
	}
	equity, peak, maxDD := initialCash, initialCash, 0.0
	for _, t := range trades {
		equity += t.PnL
		if equity > peak {
			peak = equity
		}
		if dd := (peak - equity) / peak; dd > maxDD {
			maxDD = dd
		}
	}
	return maxDD
}

// compoundReturns chains per-fold fractional returns into one cumulative return:
// prod(1+r_i) - 1. Models reinvesting each fold's OOS result into the next.
func compoundReturns(pcts []float64) float64 {
	factor := 1.0
	for _, p := range pcts {
		factor *= 1 + p
	}
	return factor - 1
}
