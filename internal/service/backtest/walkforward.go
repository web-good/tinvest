package backtest

import (
	"fmt"
	"sort"
	"strings"
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

// WalkForwardFold is one fold's outcome: the calibration winner from its train window
// and that winner's out-of-sample performance on the following test window.
type WalkForwardFold struct {
	Index          int
	TrainFrom      time.Time
	TrainTo        time.Time
	TestFrom       time.Time
	TestTo         time.Time
	InSampleMetric float64              // ranking-metric value of the train winner
	InSamplePF     float64              // train winner's profit factor (for the train-vs-OOS column)
	OOS            backtest.Metrics     // trade-based metrics over this fold's OOS trades
	OOSNetPnLPct   float64              // sum(OOS PnL) / InitialCash
	OOSMaxDDPct    float64              // drawdown fraction from replaying OOS trades
	OOSTrades      int                  // count of OOS trades
	WinnerParams   any                  // train winner params (typed)
	WinnerRows     []backtest.ParamLine // train winner params rendered for display/stability
	Note           string               // reason when a fold is skipped or has no OOS trades
}

// WalkForwardSummary aggregates all folds: the pooled OOS trade metrics and the
// compounded fold-over-fold return.
type WalkForwardSummary struct {
	Folds               []WalkForwardFold
	PooledOOS           backtest.Metrics // PooledMetrics over every fold's OOS trades
	CompoundedReturnPct float64
}

// paramStability splits the swept parameters into those that held the same winning value
// across every fold with a winner (stable) and those that changed (varied -> per-fold
// value strings in fold order). Folds without a winner (WinnerRows nil) are ignored.
func paramStability(folds []WalkForwardFold) (stable map[string]string, varied map[string][]string) {
	stable, varied = map[string]string{}, map[string][]string{}
	var names []string
	seen := map[string]bool{}
	var perFold []map[string]string
	for _, f := range folds {
		if f.WinnerRows == nil {
			continue
		}
		m := make(map[string]string, len(f.WinnerRows))
		for _, r := range f.WinnerRows {
			m[r.Name] = r.Value
			if !seen[r.Name] {
				seen[r.Name] = true
				names = append(names, r.Name)
			}
		}
		perFold = append(perFold, m)
	}
	for _, name := range names {
		vals := make([]string, 0, len(perFold))
		allSame := true
		for i, m := range perFold {
			vals = append(vals, m[name])
			if i > 0 && m[name] != vals[0] {
				allSame = false
			}
		}
		if allSame && len(vals) > 0 {
			stable[name] = vals[0]
		} else {
			varied[name] = vals
		}
	}
	return stable, varied
}

// RunWalkForward runs a rolling walk-forward for one ticker: for each fold it calibrates
// the grid on the train window and runs that winner out-of-sample on the next window with
// the train slice as indicator warm-up, keeping only trades entered at/after the OOS start.
// It returns per-fold results plus the pooled-OOS aggregate and compounded fold return.
func RunWalkForward(b Binding, phases []Phase, candles, dailyCandles, htfCandles []backtest.Candle,
	cfg backtest.Config, metric string, minTrades int, from, to time.Time, trainMonths, testMonths int,
) (WalkForwardSummary, error) {
	if err := validateMetric(metric); err != nil {
		return WalkForwardSummary{}, err
	}
	windows, err := walkForwardFolds(from, to, trainMonths, testMonths)
	if err != nil {
		return WalkForwardSummary{}, err
	}
	lookback := b.Build(b.DefaultParams()).Lookback()

	var summary WalkForwardSummary
	var pool []backtest.Trade
	var foldPcts []float64

	for i, w := range windows {
		fold := WalkForwardFold{
			Index:     i + 1,
			TrainFrom: w.trainFrom, TrainTo: w.trainTo,
			TestFrom: w.testFrom, TestTo: w.testTo,
		}

		trainSlice := sliceByRange(candles, w.trainFrom, w.trainTo)
		if len(trainSlice) < lookback {
			return WalkForwardSummary{}, fmt.Errorf(
				"backtest: fold %d train window has %d candles, fewer than the strategy lookback %d: "+
					"widen -train-months or fetch more history", i+1, len(trainSlice), lookback)
		}
		trainDays := w.trainTo.Sub(w.trainFrom).Hours() / 24

		results, err := RunPhases(b, phases, trainSlice, dailyCandles, htfCandles, cfg, metric, minTrades, trainDays, nil)
		if err != nil {
			return WalkForwardSummary{}, fmt.Errorf("backtest: fold %d calibrate: %w", i+1, err)
		}
		if len(results) == 0 {
			fold.Note = "калибровка не дала комбинаций"
			summary.Folds = append(summary.Folds, fold)
			continue
		}

		best := results[0]
		fold.InSampleMetric = metricValue(best.Metrics, metric)
		fold.InSamplePF = best.Metrics.ProfitFactor
		fold.WinnerParams = best.Params
		fold.WinnerRows = ParamRows(best.Params)

		// Warm indicators across the train slice, then run through the OOS window.
		warmSlice := sliceByRange(candles, w.trainFrom, w.testTo)
		res := backtest.Run(b.Build(best.Params), warmSlice, dailyCandles, htfCandles, cfg)
		oos := tradesEnteredFrom(res.Trades, w.testFrom)

		fold.OOS = PooledMetrics(oos)
		fold.OOSTrades = len(oos)
		if cfg.InitialCash > 0 {
			fold.OOSNetPnLPct = sumPnL(oos) / cfg.InitialCash
		}
		fold.OOSMaxDDPct = tradeReplayDrawdownPct(oos, cfg.InitialCash)
		if len(oos) == 0 {
			fold.Note = "0 OOS-сделок"
		}

		pool = append(pool, oos...)
		foldPcts = append(foldPcts, fold.OOSNetPnLPct)
		summary.Folds = append(summary.Folds, fold)
	}

	summary.PooledOOS = PooledMetrics(pool)
	summary.CompoundedReturnPct = compoundReturns(foldPcts)
	return summary, nil
}

// PooledMetrics computes aggregate metrics over a flat pool of trades, ignoring
// equity-curve/bar context (used for pooled out-of-sample validation samples).
func PooledMetrics(trades []backtest.Trade) backtest.Metrics {
	return backtest.Compute(backtest.Result{Trades: trades}, 0, 0, 0)
}

// RenderWalkForwardMarkdown renders the pooled-OOS aggregate, the per-fold train-vs-OOS
// table, and the parameter-stability breakdown.
func RenderWalkForwardMarkdown(ticker, metric string, s WalkForwardSummary, trainMonths, testMonths int) string {
	var b strings.Builder
	fmt.Fprintf(&b, "# Walk-forward %s\n\n", ticker)
	fmt.Fprintf(&b, "- Train-окно: %d мес; OOS-фолд: %d мес; шаг: %d мес (встык)\n", trainMonths, testMonths, testMonths)
	fmt.Fprintf(&b, "- Фолдов: %d; калибровка ранжировалась по: %s\n\n", len(s.Folds), metric)

	m := s.PooledOOS
	b.WriteString("## Пул сделок (агрегат OOS)\n\n")
	b.WriteString("| Метрика | Значение |\n|---|---|\n")
	fmt.Fprintf(&b, "| Всего сделок | %d |\n", m.TotalTrades)
	fmt.Fprintf(&b, "| Выигрышных / проигрышных | %d / %d |\n", m.Wins, m.Losses)
	fmt.Fprintf(&b, "| Win rate | %.2f%% |\n", m.WinRate*100)
	fmt.Fprintf(&b, "| Profit factor | %.3f |\n", m.ProfitFactor)
	fmt.Fprintf(&b, "| Gross profit / loss | %.2f / %.2f |\n", m.GrossProfit, m.GrossLoss)
	fmt.Fprintf(&b, "| Expectancy | %.2f |\n", m.Expectancy)
	fmt.Fprintf(&b, "| Sortino | %.3f |\n", m.Sortino)
	fmt.Fprintf(&b, "| Лучшая / худшая сделка | %.2f / %.2f |\n", m.BestTrade, m.WorstTrade)
	fmt.Fprintf(&b, "| Compounded return | %.2f%% |\n\n", s.CompoundedReturnPct*100)

	b.WriteString("## Результаты по фолдам\n\n")
	b.WriteString("| # | Train-окно | Test-окно | In-sample PF | OOS PF | OOS сделок | OOS NetPnL% | OOS MaxDD% |\n")
	b.WriteString("|---|---|---|---|---|---|---|---|\n")
	df := func(t time.Time) string { return t.Format("2006-01-02") }
	for _, f := range s.Folds {
		train := fmt.Sprintf("%s—%s", df(f.TrainFrom), df(f.TrainTo))
		test := fmt.Sprintf("%s—%s", df(f.TestFrom), df(f.TestTo))
		if f.WinnerRows == nil {
			// Skipped fold (no calibration winner): keep the row at 8 columns, surfacing
			// the reason in the last cell instead of numeric OOS stats.
			fmt.Fprintf(&b, "| %d | %s | %s | — | — | — | — | %s |\n", f.Index, train, test, f.Note)
			continue
		}
		fmt.Fprintf(&b, "| %d | %s | %s | %.3f | %.3f | %d | %.2f%% | %.2f%% |\n",
			f.Index, train, test, f.InSamplePF, f.OOS.ProfitFactor, f.OOSTrades, f.OOSNetPnLPct*100, f.OOSMaxDDPct*100)
	}

	b.WriteString("\n## Стабильность параметров\n\n")
	stable, varied := paramStability(s.Folds)
	if len(stable) > 0 {
		names := make([]string, 0, len(stable))
		for n := range stable {
			names = append(names, n)
		}
		sort.Strings(names)
		b.WriteString("**Стабильные** (одинаковы во всех фолдах): ")
		parts := make([]string, 0, len(names))
		for _, n := range names {
			parts = append(parts, fmt.Sprintf("%s=%s", n, stable[n]))
		}
		b.WriteString(strings.Join(parts, ", "))
		b.WriteString("\n\n")
	}
	if len(varied) > 0 {
		names := make([]string, 0, len(varied))
		for n := range varied {
			names = append(names, n)
		}
		sort.Strings(names)
		b.WriteString("**Гуляли** (значение по фолдам):\n\n")
		for _, n := range names {
			fmt.Fprintf(&b, "- %s: %s\n", n, strings.Join(varied[n], ", "))
		}
	}
	return b.String()
}
