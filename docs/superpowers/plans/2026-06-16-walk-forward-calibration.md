# Rolling Walk-Forward Calibration Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add a rolling walk-forward mode to `cmd/backtest` that re-calibrates a single ticker on a fixed-length sliding train window, runs each fold out-of-sample with warmed indicators, and reports pooled OOS metrics, a per-fold table, and parameter stability.

**Architecture:** New pure helpers and a `RunWalkForward` orchestrator live in a new file `internal/service/backtest/walkforward.go` (package `backtest`, same package as `calibrate.go`/`basket.go`, so they reuse `RunPhases`, `PooledMetrics`, `SplitByTime`, `ParamRows`, `metricValue` directly). A new Markdown renderer produces `<base>_walkforward.md`. `cmd/backtest/main.go` gains a `-train-months` flag that, when `>0` together with `-calibrate`, branches into the walk-forward path.

**Tech Stack:** Go 1.25, standard library only. Tests are table-driven (`golang-testing` conventions), white-box in `package backtest`.

**Spec:** `docs/superpowers/specs/2026-06-16-walk-forward-calibration-design.md`

---

## File Structure

- **Create** `internal/service/backtest/walkforward.go` — fold-window math, slicing/filtering/metric helpers, `WalkForwardFold`/`WalkForwardSummary` types, `RunWalkForward`, `RenderWalkForwardMarkdown`. One responsibility: walk-forward orchestration + rendering. Lives beside `basket.go`, which it mirrors.
- **Create** `internal/service/backtest/walkforward_test.go` — unit tests for pure helpers + one integration test of `RunWalkForward` with a fake `Binding`.
- **Modify** `cmd/backtest/main.go` — add `-train-months` flag (line 44 area), thread it through `run` (line 59-61, 86) and `runCalibration` (line 231-232), add the walk-forward branch + a validation guard.

Reused unchanged: `SplitByTime` (`split.go`), `RunPhases`/`metricValue`/`ParamRows`/`CalibResult`/`Binding`/`Phase`/`validateMetric` (`calibrate.go`, `registry.go`), `PooledMetrics` (`basket.go`), `backtest.Run`/`backtest.Compute`/`backtest.Trade`/`backtest.Metrics`/`backtest.Candle`/`backtest.Config`/`backtest.ParamLine` (domain).

Type/signature facts locked from the codebase:
- `Binding{ DefaultParams func() any; Build func(any) strategy.Strategy; ParseParams func([]byte)(any,error) }` (`registry.go:17`).
- `func RunPhases(b Binding, phases []Phase, candles, dailyCandles, htfCandles []backtest.Candle, cfg backtest.Config, metric string, minTrades int, periodDays float64, onProgress func(PhaseProgress)) ([]CalibResult, error)` (`calibrate.go:127`).
- `CalibResult{ Params any; Metrics backtest.Metrics }` (`calibrate.go:19`).
- `func PooledMetrics(trades []backtest.Trade) backtest.Metrics` (`basket.go:39`).
- `func metricValue(m backtest.Metrics, metric string) float64` (`calibrate.go:233`, unexported, same package).
- `func ParamRows(params any) []backtest.ParamLine` (`registry.go:77`).
- `func backtest.Run(s strategy.Strategy, candles, dailyCandles, htfCandles []backtest.Candle, cfg backtest.Config) backtest.Result` (`engine.go:104`).
- `backtest.Trade.EntryTime time.Time`, `.PnL float64` (`types.go:27,33`).
- `backtest.Config.InitialCash float64` (`types.go`).
- `backtest.Metrics` fields used: `TotalTrades, Wins, Losses, WinRate, GrossProfit, GrossLoss, ProfitFactor, Expectancy, Sortino, BestTrade, WorstTrade` (`types.go:59-79`).
- `strategy.Strategy` = `Ticker() string`, `Lookback() int`, `Decide(strategy.MarketData) model.Signal` (scalping strategy pkg).

---

## Task 1: Fold-window math and range slicing

**Files:**
- Create: `internal/service/backtest/walkforward.go`
- Test: `internal/service/backtest/walkforward_test.go`

- [ ] **Step 1: Write the failing tests**

Create `internal/service/backtest/walkforward_test.go`:

```go
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
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/service/backtest/ -run 'WalkForwardFolds|SliceByRange' -v`
Expected: FAIL — `undefined: walkForwardFolds`, `undefined: sliceByRange`.

- [ ] **Step 3: Write the helpers**

Create `internal/service/backtest/walkforward.go`:

```go
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
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/service/backtest/ -run 'WalkForwardFolds|SliceByRange' -v`
Expected: PASS (all subtests).

- [ ] **Step 5: Commit**

```bash
git add internal/service/backtest/walkforward.go internal/service/backtest/walkforward_test.go
git commit -m "feat(backtest): rolling walk-forward fold math + range slicing"
```

---

## Task 2: Per-fold metric helpers

**Files:**
- Modify: `internal/service/backtest/walkforward.go`
- Test: `internal/service/backtest/walkforward_test.go`

- [ ] **Step 1: Write the failing tests**

Append to `internal/service/backtest/walkforward_test.go`:

```go
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
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/service/backtest/ -run 'TradesEnteredFrom|SumPnL|TradeReplayDrawdownPct|CompoundReturns' -v`
Expected: FAIL — undefined helpers.

- [ ] **Step 3: Write the helpers**

Append to `internal/service/backtest/walkforward.go`:

```go
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
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/service/backtest/ -run 'TradesEnteredFrom|SumPnL|TradeReplayDrawdownPct|CompoundReturns' -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/service/backtest/walkforward.go internal/service/backtest/walkforward_test.go
git commit -m "feat(backtest): walk-forward per-fold metric helpers"
```

---

## Task 3: Types and parameter-stability analysis

**Files:**
- Modify: `internal/service/backtest/walkforward.go`
- Test: `internal/service/backtest/walkforward_test.go`

- [ ] **Step 1: Write the failing test**

Append to `internal/service/backtest/walkforward_test.go`:

```go
func TestParamStability(t *testing.T) {
	folds := []WalkForwardFold{
		{WinnerRows: []backtest.ParamLine{{Name: "RSIPeriod", Value: "6"}, {Name: "StopATRMult", Value: "0.8"}}},
		{WinnerRows: []backtest.ParamLine{{Name: "RSIPeriod", Value: "6"}, {Name: "StopATRMult", Value: "1.2"}}},
		{Note: "0 OOS-сделок"}, // skipped fold: no WinnerRows, must be ignored
	}
	stable, varied := paramStability(folds)
	if stable["RSIPeriod"] != "6" {
		t.Errorf("RSIPeriod should be stable at 6, got %q", stable["RSIPeriod"])
	}
	if _, ok := stable["StopATRMult"]; ok {
		t.Errorf("StopATRMult should not be stable")
	}
	got := varied["StopATRMult"]
	if len(got) != 2 || got[0] != "0.8" || got[1] != "1.2" {
		t.Errorf("StopATRMult varied = %v, want [0.8 1.2]", got)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/service/backtest/ -run 'ParamStability' -v`
Expected: FAIL — `undefined: WalkForwardFold`, `undefined: paramStability`.

- [ ] **Step 3: Write the types and helper**

Append to `internal/service/backtest/walkforward.go`:

```go
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
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/service/backtest/ -run 'ParamStability' -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/service/backtest/walkforward.go internal/service/backtest/walkforward_test.go
git commit -m "feat(backtest): walk-forward fold/summary types + param stability"
```

---

## Task 4: RunWalkForward orchestrator

**Files:**
- Modify: `internal/service/backtest/walkforward.go`
- Test: `internal/service/backtest/walkforward_test.go`

- [ ] **Step 1: Write the failing integration test**

Append to `internal/service/backtest/walkforward_test.go`. Add `"tinvest/internal/service/trading_strategy/scalping/model"` and `"tinvest/internal/service/trading_strategy/scalping/strategy"` to the import block at the top of the file.

```go
// alternatingStrategy buys whenever flat and sells the next bar — it produces a trade
// roughly every two bars across the whole series, at deterministic entry times. Lookback
// 1 keeps warm-up trivial. It ignores params, so any swept grid value yields the same run.
type alternatingStrategy struct{}

func (alternatingStrategy) Ticker() string { return "TEST" }
func (alternatingStrategy) Lookback() int  { return 1 }
func (alternatingStrategy) Decide(md strategy.MarketData) model.Signal {
	if md.Position == nil {
		return model.Signal{Kind: model.SignalBuy}
	}
	return model.Signal{Kind: model.SignalSell, Reason: "TP"}
}

type fakeParams struct{ Threshold int }

func fakeBinding() Binding {
	return Binding{
		DefaultParams: func() any { return fakeParams{Threshold: 1} },
		Build:         func(any) strategy.Strategy { return alternatingStrategy{} },
		ParseParams:   func([]byte) (any, error) { return fakeParams{}, nil },
	}
}

// genHourly builds 1h candles over [from, to) with a slight up-drift so some trades
// profit (keeps PooledMetrics non-degenerate).
func genHourly(from, to time.Time) []backtest.Candle {
	var out []backtest.Candle
	price := 100.0
	for ts, i := from, 0; ts.Before(to); ts, i = ts.Add(time.Hour), i+1 {
		if i%2 == 0 {
			price += 1
		} else {
			price -= 0.5
		}
		out = append(out, backtest.Candle{Time: ts, Open: price, High: price + 1, Low: price - 1, Close: price, Volume: 1})
	}
	return out
}

func TestRunWalkForward(t *testing.T) {
	from, to := date(2025, time.January, 1), date(2025, time.October, 1) // 9 months
	candles := genHourly(from, to)
	phases := []Phase{{Grid: Grid{"Threshold": {1, 2}}}}
	cfg := backtest.Config{InitialCash: 100000, Fraction: 1, Commission: 0.0005, Lot: 1}

	s, err := RunWalkForward(fakeBinding(), phases, candles, nil, nil, cfg,
		"profit_factor", 0, from, to, 3, 3)
	if err != nil {
		t.Fatalf("RunWalkForward: %v", err)
	}
	if len(s.Folds) != 2 {
		t.Fatalf("folds = %d, want 2", len(s.Folds))
	}
	var oosSum int
	for _, f := range s.Folds {
		if f.Note != "" {
			t.Fatalf("fold %d unexpectedly skipped: %s", f.Index, f.Note)
		}
		if f.OOSTrades == 0 {
			t.Fatalf("fold %d has no OOS trades", f.Index)
		}
		if f.WinnerRows == nil {
			t.Fatalf("fold %d missing winner rows", f.Index)
		}
		oosSum += f.OOSTrades
	}
	if s.PooledOOS.TotalTrades != oosSum {
		t.Fatalf("pooled trades = %d, want sum of folds %d", s.PooledOOS.TotalTrades, oosSum)
	}
}

func TestRunWalkForwardNoFold(t *testing.T) {
	from, to := date(2025, time.January, 1), date(2025, time.April, 1) // 3 months
	candles := genHourly(from, to)
	cfg := backtest.Config{InitialCash: 100000, Fraction: 1, Commission: 0.0005, Lot: 1}
	_, err := RunWalkForward(fakeBinding(), []Phase{{Grid: Grid{"Threshold": {1}}}}, candles, nil, nil, cfg,
		"profit_factor", 0, from, to, 3, 3)
	if err == nil {
		t.Fatal("want error when no fold fits")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/service/backtest/ -run 'RunWalkForward' -v`
Expected: FAIL — `undefined: RunWalkForward`.

- [ ] **Step 3: Write RunWalkForward**

Append to `internal/service/backtest/walkforward.go`:

```go
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
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/service/backtest/ -run 'RunWalkForward' -v`
Expected: PASS (both `TestRunWalkForward` and `TestRunWalkForwardNoFold`).

- [ ] **Step 5: Run the full package suite**

Run: `go test ./internal/service/backtest/...`
Expected: ok (no regressions in calibrate/basket/split tests).

- [ ] **Step 6: Commit**

```bash
git add internal/service/backtest/walkforward.go internal/service/backtest/walkforward_test.go
git commit -m "feat(backtest): RunWalkForward rolling orchestrator"
```

---

## Task 5: Walk-forward Markdown report

**Files:**
- Modify: `internal/service/backtest/walkforward.go`
- Test: `internal/service/backtest/walkforward_test.go`

- [ ] **Step 1: Write the failing test**

Append to `internal/service/backtest/walkforward_test.go`:

```go
func TestRenderWalkForwardMarkdown(t *testing.T) {
	s := WalkForwardSummary{
		Folds: []WalkForwardFold{
			{
				Index:     1,
				TrainFrom: date(2025, time.January, 1), TrainTo: date(2025, time.April, 1),
				TestFrom: date(2025, time.April, 1), TestTo: date(2025, time.July, 1),
				InSamplePF: 2.10, OOS: backtest.Metrics{ProfitFactor: 1.30, TotalTrades: 12},
				OOSNetPnLPct: 0.031, OOSMaxDDPct: 0.04, OOSTrades: 12,
				WinnerRows: []backtest.ParamLine{{Name: "RSIPeriod", Value: "6"}, {Name: "StopATRMult", Value: "0.8"}},
			},
			{
				Index:     2,
				TrainFrom: date(2025, time.April, 1), TrainTo: date(2025, time.July, 1),
				TestFrom: date(2025, time.July, 1), TestTo: date(2025, time.October, 1),
				InSamplePF: 1.90, OOS: backtest.Metrics{ProfitFactor: 0.80, TotalTrades: 9},
				OOSNetPnLPct: -0.012, OOSMaxDDPct: 0.06, OOSTrades: 9,
				WinnerRows: []backtest.ParamLine{{Name: "RSIPeriod", Value: "6"}, {Name: "StopATRMult", Value: "1.2"}},
			},
		},
		PooledOOS:           backtest.Metrics{ProfitFactor: 1.05, TotalTrades: 21, WinRate: 0.48},
		CompoundedReturnPct: 0.0186,
	}
	md := RenderWalkForwardMarkdown("NVTK", "profit_factor", s, 3, 3)

	for _, want := range []string{
		"# Walk-forward NVTK",
		"## Пул сделок (агрегат OOS)",
		"Profit factor",
		"Compounded return",
		"## Результаты по фолдам",
		"## Стабильность параметров",
		"RSIPeriod",   // stable param mentioned
		"StopATRMult", // varied param mentioned
	} {
		if !strings.Contains(md, want) {
			t.Errorf("report missing %q", want)
		}
	}
}
```

Add `"strings"` to the test file's import block.

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/service/backtest/ -run 'RenderWalkForwardMarkdown' -v`
Expected: FAIL — `undefined: RenderWalkForwardMarkdown`.

- [ ] **Step 3: Write the renderer**

Append to `internal/service/backtest/walkforward.go`. Add `"sort"`, `"strings"` to the file's import block (alongside `"fmt"`, `"time"`, and the domain import).

```go
// RenderWalkForwardMarkdown renders the pooled-OOS aggregate, the per-fold train-vs-OOS
// table, and the parameter-stability breakdown. Mirrors RenderBasketMarkdown's style.
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
			fmt.Fprintf(&b, "| %d | %s | %s | — | — | — | — | — | %s |\n", f.Index, train, test, f.Note)
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
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/service/backtest/ -run 'RenderWalkForwardMarkdown' -v`
Expected: PASS.

- [ ] **Step 5: Run the full package suite**

Run: `go test ./internal/service/backtest/...`
Expected: ok.

- [ ] **Step 6: Commit**

```bash
git add internal/service/backtest/walkforward.go internal/service/backtest/walkforward_test.go
git commit -m "feat(backtest): walk-forward Markdown report"
```

---

## Task 6: Wire `-train-months` into the CLI

**Files:**
- Modify: `cmd/backtest/main.go`

No unit test (CLI wiring over the gRPC candle provider needs a live token). Verified by `go build`/`go vet` plus a documented manual run.

- [ ] **Step 1: Add the flag**

In `cmd/backtest/main.go`, in the `flag` block, immediately after the `testMonths` line (currently line 44):

```go
		testMonths   = flag.Int("test-months", 0, "walk-forward: calibrate on the earlier window, report best on the last N months")
		trainMonths  = flag.Int("train-months", 0, "rolling walk-forward (with -calibrate): fixed train window in months; step = -test-months")
```

- [ ] **Step 2: Pass it into run()**

Change the `run(...)` call (currently lines 59-61) to include `*trainMonths` right after `*testMonths`:

```go
	if err := run(*ticker, *strategyName, interval, *months, *cash, *fraction, *commission,
		*paramsPath, *calibrate, *metric, *minTrades, *testMonths, *trainMonths, *outDir, *refresh, *explain,
		*basket, *gridDir); err != nil {
		log.Fatalf("backtest: %v", err)
	}
```

- [ ] **Step 3: Update run() signature and add the guard**

Change the `run` signature (currently line 86-89) to add `trainMonths int` after `testMonths int`:

```go
func run(ticker, strategyName string, interval enum.Interval, months int, cash, fraction, commission float64,
	paramsPath, calibratePath, metric string, minTrades, testMonths, trainMonths int, outDir string, refresh bool, explain string,
	basketCSV, gridDir string,
) error {
```

Then, right after the existing `if paramsPath != "" && calibratePath != ""` / `if basketCSV != "" && calibratePath != ""` guards near the top of `run` (currently around lines 90-95), add:

```go
	if trainMonths > 0 && calibratePath == "" {
		return fmt.Errorf("-train-months requires -calibrate (walk-forward re-calibrates each fold from a grid)")
	}
```

- [ ] **Step 4: Pass from/trainMonths into runCalibration and branch**

The `runCalibration` call is currently (lines 184-187):

```go
	if calibratePath != "" {
		return runCalibration(binding, calibratePath, candles, dailyCandles, htfCandles, cfg, metric, minTrades, testMonths, periodDays, base,
			metaCommon(ticker, interval, from, to, cfg), to)
	}
```

Change it to also pass `from` and `trainMonths`:

```go
	if calibratePath != "" {
		return runCalibration(binding, calibratePath, candles, dailyCandles, htfCandles, cfg, metric, minTrades, testMonths, trainMonths, periodDays, base,
			metaCommon(ticker, interval, from, to, cfg), from, to)
	}
```

Update the `runCalibration` signature (currently lines 231-233) to accept `trainMonths int` and `from time.Time`:

```go
func runCalibration(b svc.Binding, gridPath string, candles []domain.Candle, dailyCandles, htfCandles []domain.Candle,
	cfg domain.Config, metric string, minTrades, testMonths, trainMonths int, periodDays float64, base string, meta domain.Meta, from, to time.Time,
) error {
```

Add the walk-forward branch as the FIRST statement in `runCalibration` body (before reading the grid file at line 234), so the holdout path is untouched when `trainMonths == 0`:

```go
	if trainMonths > 0 {
		raw, err := os.ReadFile(gridPath)
		if err != nil {
			return fmt.Errorf("read grid: %w", err)
		}
		phases, err := svc.ParsePhases(raw)
		if err != nil {
			return err
		}
		summary, err := svc.RunWalkForward(b, phases, candles, dailyCandles, htfCandles, cfg, metric, minTrades, from, to, trainMonths, testMonths)
		if err != nil {
			return err
		}
		wfPath := base + "_walkforward.md"
		if err := writeFile(wfPath, svc.RenderWalkForwardMarkdown(meta.Ticker, metric, summary, trainMonths, testMonths)); err != nil {
			return err
		}
		fmt.Printf("walk-forward: %s (folds=%d, pooled PF=%.3f, compounded=%.2f%%)\n",
			wfPath, len(summary.Folds), summary.PooledOOS.ProfitFactor, summary.CompoundedReturnPct*100)
		return nil
	}
```

- [ ] **Step 5: Build and vet**

Run: `go build ./... && go vet ./cmd/backtest/ ./internal/service/backtest/`
Expected: no output (clean build + vet).

- [ ] **Step 6: Run the full backtest test suite**

Run: `go test ./internal/service/backtest/...`
Expected: ok.

- [ ] **Step 7: Commit**

```bash
git add cmd/backtest/main.go
git commit -m "feat(backtest): -train-months flag for rolling walk-forward"
```

---

## Task 7: Manual verification run (no code)

**Files:** none (operator step; requires `T_BANK` token + network).

- [ ] **Step 1: Run walk-forward on NVTK**

Run:
```bash
go run ./cmd/backtest -ticker NVTK -strategy reversion \
  -calibrate data/params/nvtk/reversion_grid.json \
  -out ./reports/NVTK -months 24 -train-months 12 -test-months 3 \
  -min-trades 10 -metric profit_factor
```
Expected: stdout prints `walk-forward: reports/NVTK/..._walkforward.md (folds=4, pooled PF=..., compounded=...)`; file `<...>_walkforward.md` exists with the three sections.

- [ ] **Step 2: Sanity-check the report**

Open the generated `_walkforward.md` and confirm:
- 4 fold rows, each with distinct, abutting train/test windows.
- In-sample PF vs OOS PF columns populated; pooled PF present.
- Parameter-stability section lists stable and/or varied knobs.

- [ ] **Step 3: Confirm backward compatibility**

Run the same command WITHOUT `-train-months` and confirm the old single-holdout output (`_best.md` + `_calibration.md`) is still produced unchanged.

---

## Out of scope (optional follow-up, not part of this plan)

The existing single-holdout path (`runCalibration`, `main.go:287`) cold-starts its OOS run at `boundary`, suffering the same warm-up bias this plan fixes for walk-forward. Folding that path onto a shared `tradesEnteredFrom`-based helper is a worthwhile follow-up but is intentionally excluded here to keep the diff focused.

---

## Self-Review Notes

- **Spec coverage:** window algorithm (Task 1), warm-up lead-in + entry-time filter (Task 2 helper + Task 4 use), types/summary (Task 3), pooled + compounded + RunWalkForward (Task 4), 3-block report incl. param stability (Tasks 3+5), `-train-months` CLI + guard + backward compat (Task 6), manual validation (Task 7). All spec sections map to a task.
- **Type consistency:** `WalkForwardFold`/`WalkForwardSummary` field names are identical across Tasks 3, 4, 5. Helper names (`walkForwardFolds`, `sliceByRange`, `tradesEnteredFrom`, `sumPnL`, `tradeReplayDrawdownPct`, `compoundReturns`, `paramStability`, `RunWalkForward`, `RenderWalkForwardMarkdown`) are used identically where referenced.
- **Signatures matched to codebase:** `RunPhases`, `PooledMetrics`, `metricValue`, `ParamRows`, `backtest.Run`, `Binding`, `CalibResult`, `Trade.EntryTime/.PnL`, `Config.InitialCash` all verified against current source before writing.
