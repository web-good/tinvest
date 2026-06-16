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
