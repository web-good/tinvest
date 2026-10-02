package main

import (
	"errors"
	"fmt"
	"sync"
	"time"

	domain "tinvest/internal/domain/backtest"
	svc "tinvest/internal/service/backtest"
	"tinvest/pkg/semaphore"
)

// engineCash is the mock account per ticker run; with Fraction 1.0 lot rounding is negligible
// and the pooled PF is taken on per-trade returns anyway.
const engineCash = 1e7

// maxFoldLag is how far the last fold's test window may end before the run date: the window
// start is pulled to a day <= 28, so the grid can end up to three days early, never months.
const maxFoldLag = 7 * 24 * time.Hour

// stressCommissions are the per-side costs of the sensitivity table; gate 4 reads the row at
// svc.GapStressCommission.
var stressCommissions = []float64{0.001, svc.GapStressCommission}

// windowStart returns the start of a months-long window ending at to. The anchor day is capped
// at 28: walkForwardFolds chains AddDate from the start, and a 29th–31st overflows into the next
// month on the way, losing the last fold (2026-05-31 gave 7 of 8 folds, the last ending in March).
func windowStart(to time.Time, months int) time.Time {
	anchor := time.Date(to.Year(), to.Month(), min(to.Day(), 28),
		to.Hour(), to.Minute(), to.Second(), to.Nanosecond(), to.Location())
	return anchor.AddDate(0, -months, 0)
}

// checkFoldsReachTo rejects a fold grid whose newest test window ends well before the run date:
// the newest quarter would be missing and the last-12-months gate would measure a shorter span.
func checkFoldsReachTo(folds []svc.GapFold, to time.Time) error {
	if len(folds) == 0 {
		return errors.New("no walk-forward folds")
	}
	last := folds[len(folds)-1].TestTo
	if to.Sub(last) > maxFoldLag {
		return fmt.Errorf("last fold ends %s, more than 7 days before the window end %s",
			last.Format("2006-01-02"), to.Format("2006-01-02"))
	}
	return nil
}

// evalResult is everything the report needs from the post-load evaluation.
type evalResult struct {
	runs       []svc.GapComboRun
	folds      []svc.GapFold
	sens       []svc.GapCostRow
	exDays     map[string]int
	verdict    svc.GapVerdict
	last12From time.Time
}

// evaluate is the pure part of a run once the data is loaded: every combination on every
// ticker, the pooled walk-forward, the stressed replay of the chosen combinations and the gate.
// The last-12-months window is anchored to the last fold's test end, so gate 2 and gate 6 measure
// exactly the newest twelve months of out-of-sample trades.
func evaluate(data []tickerData, combos []any, cfg runCfg, from, to time.Time) (evalResult, error) {
	var res evalResult
	res.runs, res.exDays = runCombos(data, combos, nil, cfg.commission, cfg.workers)
	folds, err := svc.RunGapWalkForward(res.runs, from, to, cfg.trainMonths, cfg.testMonths, cfg.minTrades)
	if err != nil {
		return res, err
	}
	if err := checkFoldsReachTo(folds, to); err != nil {
		return res, err
	}
	res.folds = folds
	oos := svc.PooledGapOOS(folds)
	selected := svc.SelectedGapCombos(folds)
	res.sens = []svc.GapCostRow{{Commission: cfg.commission, Trades: oos}}
	var stressed []svc.GapTrade
	found := false
	for _, c := range stressCommissions {
		alt, _ := runCombos(data, combos, selected, c, cfg.workers)
		replayed, err := svc.ReplayGapFolds(folds, alt)
		if err != nil {
			return res, err
		}
		res.sens = append(res.sens, svc.GapCostRow{Commission: c, Trades: replayed})
		if c == svc.GapStressCommission {
			stressed, found = replayed, true
		}
	}
	if !found {
		return res, fmt.Errorf("no stressed replay at %v per side for gate 4", svc.GapStressCommission)
	}
	res.last12From = folds[len(folds)-1].TestTo.AddDate(0, -12, 0)
	res.verdict = svc.GapGate(oos, stressed, res.last12From, cfg.minTrades, svc.GapGateMinLast12Trades)
	return res, nil
}

// runCombos runs the engine for every combination (or only those in `only`, when non-nil) on
// every ticker at one commission, drops ex-day trades and returns runs index-aligned with
// combos plus, per ticker, the number of distinct ex-days on which some trade was dropped.
// Tickers run concurrently; the merge is in ticker order, so the output is deterministic.
func runCombos(data []tickerData, combos []any, only map[int]bool, commission float64, workers int) ([]svc.GapComboRun, map[string]int) {
	perTicker := make([][][]svc.GapTrade, len(data))
	exHit := make([]int, len(data))
	sem := semaphore.New(workers)
	var wg sync.WaitGroup
	for ti := range data {
		wg.Add(1)
		sem.Acquire()
		go func(ti int) {
			defer wg.Done()
			defer sem.Release()
			d := data[ti]
			binding := svc.GapFadeLookupOrGeneric(d.ticker)
			cfg := domain.Config{InitialCash: engineCash, Fraction: 1.0, Commission: commission, Lot: d.lot}
			days := map[string]bool{}
			perTicker[ti] = make([][]svc.GapTrade, len(combos))
			for ci, p := range combos {
				if only != nil && !only[ci] {
					continue
				}
				res := domain.Run(binding.Build(p), d.bars, d.daily, nil, cfg)
				kept, dropped := svc.DropExDayTrades(svc.TagGapTrades(d.ticker, res.Trades), d.exDays)
				perTicker[ti][ci] = kept
				for _, day := range dropped {
					days[day] = true
				}
			}
			exHit[ti] = len(days)
		}(ti)
	}
	wg.Wait()

	runs := make([]svc.GapComboRun, len(combos))
	for ci := range combos {
		runs[ci].Params = combos[ci]
		for ti := range data {
			runs[ci].Trades = append(runs[ci].Trades, perTicker[ti][ci]...)
		}
	}
	ex := make(map[string]int, len(data))
	for ti, d := range data {
		ex[d.ticker] = exHit[ti]
	}
	return runs, ex
}
