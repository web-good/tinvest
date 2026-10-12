# drop_bounce Screener Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** `cmd/dropscreen` ranks the tradable RUB share universe by how well drop_bounce fits each ticker on one timeframe and prints a shortlist of calibration candidates for the `drop-bounce-calibrate` skill.

**Architecture:** Clone of the `cmd/zonescreen` pattern. A pure core in `internal/service/backtest/drop_screen.go` holds the grid, the per-ticker engine replay and the aggregator. `drop_screen_report.go` holds the gates, the ranking, the shortlist and the markdown. `cmd/dropscreen` does all the gRPC and file I/O through the shared `screenrun` plumbing. The real drop_bounce engine runs on a fixed 32-configuration grid. Tickers are ranked by the median train profit factor under stressed costs.

**Tech Stack:** Go 1.25, existing packages `internal/domain/backtest` (engine), `internal/service/trading_strategy/drop_bounce/strategy/core` (strategy), `internal/service/backtest/screenrun` (I/O plumbing).

**Spec:** `docs/superpowers/specs/2026-10-12-drop-bounce-screener-design.md`

## Global Constraints

- Timeframes: `Minutes15` (default), `Minutes30`, `Hour1` only; `Minutes5` and the rest are an error.
- Window defaults: `-months 36`, `-holdout-months 6`; trades are assigned to train or holdout by ENTRY time (`splitTrades`); the holdout never ranks or gates.
- Grid, 32 configurations, fixed order: `VolDailyMult` (outer) × `RSIPeriod` {4, 6} × `RSIExit` {55, 75} × `GreenVolExit` {0, 1.5} (inner). `VolDailyMult`: 15m {0.05, 0.1, 0.2, 0.3}; 30m {0.1, 0.2, 0.3, 0.5}; 1h {0.15, 0.25, 0.4, 0.6}. Pinned: `StopDailyATR` 1.0, `DailyATRPeriod` 10, `VolAvgPeriod` 10, `StartHour` 7. Stop is never 0.
- Engine config: `backtest.Config{InitialCash: opts.Cash, Fraction: opts.Fraction, Commission: opts.Commission, Lot: lot, EntryAtNextOpen: true}`. The strategy is built with `dropcore.NewWithParams` and never through `DropBounceLookupOrGeneric`.
- Ranking key: `PFMedStress` descending, then `TradesMed` descending, then `Ticker` ascending.
- Gates (equality passes): `-min-turnover` 50 (million RUB/day), `-max-tick-pct` 0.3 (0 = off; unknown tick passes), `-min-trades` 20 (median train trades). There is no ATR gate.
- Shortlist line: `DROP_BOUNCE_CANDIDATES=<tickers>`, first `-shortlist` (default 10; 0 = whole ranking) rows of the ranking.
- `-workers` default 1 (single-core rule).
- Report path: `<out>/drop_screen_<Interval>_<YYYYMMDD_HHMMSS>.md`, default `-out reports/drop_screen`.
- The branch `feat/drop-bounce` carries unrelated uncommitted edits, including in `CLAUDE.md`, `docs/drop_bounce/strategy.md` and `.claude/skills/drop-bounce-calibrate/SKILL.md`. **Every commit names its files explicitly** (`git commit -- <paths>`). Never `git add -A` / `git commit -a`. Never commit those three files: edit them and leave them uncommitted.
- Docs in `docs/drop_bounce/` describe mechanics only, no run results.
- Build check: `go build ./internal/... ./pkg/... ./cmd/...` (never `./...`: magefiles has no main). Final gate: `./bin/mage ci`.

## Review Focus

- A ticker that never trades on any configuration (thin or flat paper) must land in "no signals", not in the ranking with PF 0 and not in "rejected for trades". Covered in Task 3 (`FilterAndRankDrop` puts `NoSignals` before the trade gate) and Task 2 (all-silent aggregate).
- A config with only winners gives `+Inf` PF. It must be clamped to `-pf-cap` and counted in `Capped`, and must never poison the median. Covered in Task 2.
- Shortlist with fewer ranked rows than `-shortlist`, or an empty ranking, must not panic and must print the key with what exists. Covered in Task 3.
- A registered ticker (GAZP, T) must be graded on the grid config, not on its calibrated literal. Covered in Task 2 (`ScreenDropTicker` matches a direct engine run of the grid config for ticker "GAZP").
- An invalid `-interval` (Minutes5, Day1, typo) must stop the run before any API call. Covered in Task 4 (`validate`).

---

### Task 1: Grid and exit-share helper

**Files:**
- Create: `internal/service/backtest/drop_screen.go`
- Create: `internal/service/backtest/drop_screen_test.go`
- Modify: `internal/service/backtest/zone_screen.go` (function `slShare`, around line 141)

**Interfaces:**
- Consumes: `dropcore.Params` from `tinvest/internal/service/trading_strategy/drop_bounce/strategy/core` (fields `VolDailyMult float64, VolAvgPeriod int, RSIPeriod int, RSIExit float64, StopDailyATR float64, DailyATRPeriod int, StartHour int, GreenVolExit float64`); `enum.Interval`.
- Produces:
  - `func DropGrid(interval enum.Interval) ([]dropcore.Params, error)`
  - `func dropGridVols(interval enum.Interval) ([]float64, bool)` (used by the report header)
  - `func exitShare(trades []backtest.Trade, reason string) float64`

- [ ] **Step 1: Write the failing tests**

Create `internal/service/backtest/drop_screen_test.go`:

```go
package backtest

import (
	"reflect"
	"testing"

	"tinvest/internal/domain/backtest"
	"tinvest/internal/enum"
)

func TestDropGridPerTimeframe(t *testing.T) {
	cases := []struct {
		interval enum.Interval
		vols     []float64
	}{
		{enum.Minutes15, []float64{0.05, 0.1, 0.2, 0.3}},
		{enum.Minutes30, []float64{0.1, 0.2, 0.3, 0.5}},
		{enum.Hour1, []float64{0.15, 0.25, 0.4, 0.6}},
	}
	for _, c := range cases {
		grid, err := DropGrid(c.interval)
		if err != nil {
			t.Fatalf("%s: DropGrid error: %v", c.interval, err)
		}
		if len(grid) != 32 {
			t.Fatalf("%s: len = %d, want 32", c.interval, len(grid))
		}
		seen := make(map[float64]bool)
		uniq := make(map[[4]float64]bool)
		for _, p := range grid {
			if p.StopDailyATR != 1.0 || p.DailyATRPeriod != 10 || p.VolAvgPeriod != 10 || p.StartHour != 7 {
				t.Fatalf("%s: pinned fields drifted: %+v", c.interval, p)
			}
			seen[p.VolDailyMult] = true
			uniq[[4]float64{p.VolDailyMult, float64(p.RSIPeriod), p.RSIExit, p.GreenVolExit}] = true
		}
		var got []float64
		for _, v := range c.vols {
			if seen[v] {
				got = append(got, v)
			}
		}
		if !reflect.DeepEqual(got, c.vols) || len(seen) != len(c.vols) {
			t.Fatalf("%s: VolDailyMult set = %v, want %v", c.interval, seen, c.vols)
		}
		if len(uniq) != 32 {
			t.Fatalf("%s: %d distinct configurations, want 32", c.interval, len(uniq))
		}
	}
}

func TestDropGridOrder(t *testing.T) {
	grid, err := DropGrid(enum.Minutes15)
	if err != nil {
		t.Fatal(err)
	}
	first, second, last := grid[0], grid[1], grid[31]
	if first.VolDailyMult != 0.05 || first.RSIPeriod != 4 || first.RSIExit != 55 || first.GreenVolExit != 0 {
		t.Fatalf("grid[0] = %+v, want vol 0.05 RSI(4)@55 GV 0", first)
	}
	if second.GreenVolExit != 1.5 || second.RSIExit != 55 {
		t.Fatalf("grid[1] = %+v, want GreenVolExit as the innermost axis", second)
	}
	if last.VolDailyMult != 0.3 || last.RSIPeriod != 6 || last.RSIExit != 75 || last.GreenVolExit != 1.5 {
		t.Fatalf("grid[31] = %+v, want vol 0.3 RSI(6)@75 GV 1.5", last)
	}
}

func TestDropGridRejectsOtherTimeframes(t *testing.T) {
	for _, iv := range []enum.Interval{enum.Minutes5, enum.Day1, enum.Hour4} {
		if _, err := DropGrid(iv); err == nil {
			t.Fatalf("DropGrid(%s) = nil error, want an error", iv)
		}
	}
}

func TestExitShare(t *testing.T) {
	trades := []backtest.Trade{{Reason: "SL"}, {Reason: "RSI"}, {Reason: "SL"}, {Reason: "GREENVOL"}}
	for reason, want := range map[string]float64{"SL": 0.5, "RSI": 0.25, "GREENVOL": 0.25, "EOD": 0} {
		if got := exitShare(trades, reason); got != want {
			t.Fatalf("exitShare(%s) = %v, want %v", reason, got, want)
		}
	}
	if got := exitShare(nil, "SL"); got != 0 {
		t.Fatalf("exitShare(nil) = %v, want 0", got)
	}
}
```

- [ ] **Step 2: Run the tests to verify they fail**

Run: `go test ./internal/service/backtest/ -run 'TestDropGrid|TestExitShare' -count=1`
Expected: build failure, `undefined: DropGrid` and `undefined: exitShare`.

- [ ] **Step 3: Implement**

Create `internal/service/backtest/drop_screen.go`:

```go
package backtest

import (
	"fmt"

	"tinvest/internal/domain/backtest"
	"tinvest/internal/enum"
	dropcore "tinvest/internal/service/trading_strategy/drop_bounce/strategy/core"
)

// Pinned drop-screener axes. Like the zone screener, it answers "is this ticker worth
// calibrating", so it sweeps only the axes whose useful range depends on the paper: the volume
// threshold (how often a ticker spikes) and the two opposite exit shapes. See
// docs/superpowers/specs/2026-10-12-drop-bounce-screener-design.md, §4.
var (
	dropGridVolsByInterval = map[enum.Interval][]float64{
		enum.Minutes15: {0.05, 0.1, 0.2, 0.3},
		enum.Minutes30: {0.1, 0.2, 0.3, 0.5},
		enum.Hour1:     {0.15, 0.25, 0.4, 0.6},
	}
	dropGridRSIPeriods = []int{4, 6}
	dropGridRSIExits   = []float64{55, 75}
	dropGridGreenVols  = []float64{0, 1.5}
)

// dropGridVols returns the VolDailyMult axis of a timeframe; ok is false for an unscreened one.
func dropGridVols(interval enum.Interval) ([]float64, bool) {
	v, ok := dropGridVolsByInterval[interval]
	return v, ok
}

// DropGrid returns the 32 configurations every ticker is screened on for one timeframe, in a
// deterministic order (VolDailyMult outermost, GreenVolExit innermost). The stop is pinned at
// 1.0 daily ATR(10): a property of a tuned point, not of the paper, and the value walk-forward
// folds picked on GAZP and T. Only Minutes15, Minutes30 and Hour1 are screened, matching the
// timeframes drop-bounce-calibrate calibrates.
func DropGrid(interval enum.Interval) ([]dropcore.Params, error) {
	vols, ok := dropGridVols(interval)
	if !ok {
		return nil, fmt.Errorf("drop screener runs on Minutes15, Minutes30 or Hour1, not %s", interval)
	}
	out := make([]dropcore.Params, 0, len(vols)*len(dropGridRSIPeriods)*len(dropGridRSIExits)*len(dropGridGreenVols))
	for _, vol := range vols {
		for _, period := range dropGridRSIPeriods {
			for _, exit := range dropGridRSIExits {
				for _, gv := range dropGridGreenVols {
					out = append(out, dropcore.Params{
						VolDailyMult:   vol,
						VolAvgPeriod:   10,
						RSIPeriod:      period,
						RSIExit:        exit,
						StopDailyATR:   1.0,
						DailyATRPeriod: 10,
						StartHour:      7,
						GreenVolExit:   gv,
					})
				}
			}
		}
	}
	return out, nil
}

// exitShare is the fraction of trades closed with the given exit reason code.
func exitShare(trades []backtest.Trade, reason string) float64 {
	if len(trades) == 0 {
		return 0
	}
	var n int
	for _, t := range trades {
		if t.Reason == reason {
			n++
		}
	}
	return float64(n) / float64(len(trades))
}
```

In `internal/service/backtest/zone_screen.go` replace the body of `slShare` with a call to the new helper (keep the doc comment and the name, callers stay unchanged):

```go
// slShare is the fraction of trades closed by the protective stop (exit reason "SL").
func slShare(trades []backtest.Trade) float64 { return exitShare(trades, "SL") }
```

- [ ] **Step 4: Run the tests to verify they pass**

Run: `go test ./internal/service/backtest/ -count=1`
Expected: `ok` (new tests pass, the zone tests that cover `slShare` stay green).

- [ ] **Step 5: Commit**

```bash
git add internal/service/backtest/drop_screen.go internal/service/backtest/drop_screen_test.go
git commit -m "feat(drop_screen): screening grid per timeframe and exit-share helper

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>" -- internal/service/backtest/drop_screen.go internal/service/backtest/drop_screen_test.go internal/service/backtest/zone_screen.go
```

---

### Task 2: Aggregator and per-ticker replay

**Files:**
- Modify: `internal/service/backtest/drop_screen.go`
- Modify: `internal/service/backtest/drop_screen_test.go`

**Interfaces:**
- Consumes (all existing in package `backtest`, `internal/service/backtest`): `profitFactor(trades) (float64, int)`, `splitTrades(trades, split) (train, holdout)`, `medianF([]float64) float64`, `clampPF(pf, limit) (float64, bool)`, `stressTrades(trades, commission, tick) []backtest.Trade`, `tickPct(tick, lastClose) (float64, bool)`, `halfYearResults(trades) []HalfResult`, `halvesStats([]HalfResult) (posShare, topShare float64, hasProfit bool)`, `medianHoldDays(trades) float64`, `MeanDailyATRPct(daily, period) float64`, const `screenDailyATRPeriod` (14), `ScreenOpts{PFCap, PlateauPF, PlateauTrades, Cash, Fraction, Commission}`, `DefaultScreenOpts()`; `backtest.MeanDailyTurnoverM(candles, lot) float64` and `backtest.Run(s, candles, daily, htf, cfg) Result` from `internal/domain/backtest`; test fixture `dailyCandlesMSK(n int, closePrice, rangePct float64) []backtest.Candle` (in `pullback_screen_test.go`, Volume 1000, one bar per calendar day from Monday 2026-01-05 10:00 MSK). Task 1: `DropGrid`, `exitShare`.
- Produces:
  - `type DropConfigResult struct { Params dropcore.Params; Trades []backtest.Trade }`
  - `type DropRow struct` (fields below)
  - `func AggregateDrop(ticker, name string, results []DropConfigResult, split time.Time, tick float64, opts ScreenOpts) DropRow`
  - `type DropTickerInput struct { Ticker, Name string; Interval enum.Interval; Bars, Daily []backtest.Candle; Lot int32; MinPriceIncrement float64 }`
  - `func ScreenDropTicker(in DropTickerInput, cfgs []dropcore.Params, split time.Time, opts ScreenOpts) DropRow`

- [ ] **Step 1: Write the failing tests**

Append to `internal/service/backtest/drop_screen_test.go` (add imports `math`, `time`, `dropcore "tinvest/internal/service/trading_strategy/drop_bounce/strategy/core"`):

```go
var dropSplit = time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)

// dropTrade is a trade held two hours at price 100 with 10 shares: the cost stress then removes
// 0.0005·200·10 = 1 of commission plus 2·tick·10 of ticks.
func dropTrade(entry time.Time, pnl float64, reason string) backtest.Trade {
	return backtest.Trade{EntryTime: entry, ExitTime: entry.Add(2 * time.Hour), EntryPrice: 100, ExitPrice: 100,
		Quantity: 10, PnL: pnl, Reason: reason}
}

func trainAt(day int) time.Time { return time.Date(2026, 2, day, 10, 0, 0, 0, time.UTC) }

func holdoutAt(day int) time.Time { return time.Date(2026, 7, day, 10, 0, 0, 0, time.UTC) }

func TestAggregateDropMediansExitsAndSilence(t *testing.T) {
	grid, _ := DropGrid(enum.Minutes15)
	results := []DropConfigResult{
		{Params: grid[0], Trades: []backtest.Trade{
			dropTrade(trainAt(2), 200, "RSI"), dropTrade(trainAt(3), -100, "SL"), dropTrade(holdoutAt(1), 50, "GREENVOL"),
		}},
		{Params: grid[1], Trades: []backtest.Trade{
			dropTrade(trainAt(4), 300, "GREENVOL"), dropTrade(trainAt(5), -100, "SL"), dropTrade(trainAt(6), -100, "SL"),
		}},
		{Params: grid[2]},
	}
	row := AggregateDrop("GAZP", "Газпром", results, dropSplit, 0, DefaultScreenOpts())

	near := func(name string, got, want float64) {
		t.Helper()
		if math.Abs(got-want) > 1e-9 {
			t.Fatalf("%s = %v, want %v", name, got, want)
		}
	}
	near("PFMed", row.PFMed, 1.5)        // PFs 2, 1.5, 0
	near("TradesMed", row.TradesMed, 2) // 2, 3, 0
	near("SLShare", row.SLShare, (0.5+2.0/3)/2)
	near("RSIShare", row.RSIShare, 0.25)        // 0.5 and 0 over trading configs
	near("GreenVolShare", row.GreenVolShare, 1.0/6) // 0 and 1/3
	near("HoldDays", row.HoldDays, 2.0/24)
	near("PFMedHO", row.PFMedHO, 0)   // holdout PFs: +Inf→10, 0, 0
	near("TradesMedHO", row.TradesMedHO, 0)
	if row.SilentCfg != 1 || row.NoSignals {
		t.Fatalf("SilentCfg/NoSignals = %d/%v, want 1/false", row.SilentCfg, row.NoSignals)
	}
	if row.Best != grid[0] || row.BestPF != 2 {
		t.Fatalf("Best/BestPF = %+v/%v, want grid[0]/2", row.Best, row.BestPF)
	}
	if row.Capped != 0 {
		t.Fatalf("Capped = %d, want 0: only train PFs count", row.Capped)
	}
}

func TestAggregateDropStressIgnoresHoldout(t *testing.T) {
	grid, _ := DropGrid(enum.Minutes15)
	train := []backtest.Trade{dropTrade(trainAt(2), 200, "RSI"), dropTrade(trainAt(3), -100, "SL")}
	opts := DefaultScreenOpts()

	row := AggregateDrop("T", "Т", []DropConfigResult{{Params: grid[0], Trades: train}}, dropSplit, 1, opts)
	if row.PFMed != 2 {
		t.Fatalf("PFMed = %v, want 2", row.PFMed)
	}
	want := 179.0 / 121.0 // each trade loses 1 commission + 20 ticks
	if math.Abs(row.PFMedStress-want) > 1e-9 {
		t.Fatalf("PFMedStress = %v, want %v", row.PFMedStress, want)
	}

	withHoldout := append(append([]backtest.Trade(nil), train...), dropTrade(holdoutAt(1), -1000, "SL"))
	row2 := AggregateDrop("T", "Т", []DropConfigResult{{Params: grid[0], Trades: withHoldout}}, dropSplit, 1, opts)
	if row2.PFMedStress != row.PFMedStress || row2.TradesMed != row.TradesMed {
		t.Fatalf("holdout moved the ranking key: %v/%v vs %v/%v", row2.PFMedStress, row2.TradesMed, row.PFMedStress, row.TradesMed)
	}
	if row2.PFMedStressHO != 0 || row2.TradesMedHO != 1 {
		t.Fatalf("PFMedStressHO/TradesMedHO = %v/%v, want 0/1", row2.PFMedStressHO, row2.TradesMedHO)
	}
}

func TestAggregateDropAllSilent(t *testing.T) {
	grid, _ := DropGrid(enum.Minutes15)
	row := AggregateDrop("XXXX", "", []DropConfigResult{{Params: grid[3]}, {Params: grid[4]}}, dropSplit, 0, DefaultScreenOpts())
	if !row.NoSignals || row.SilentCfg != 2 || row.TradesMed != 0 {
		t.Fatalf("NoSignals/SilentCfg/TradesMed = %v/%d/%v, want true/2/0", row.NoSignals, row.SilentCfg, row.TradesMed)
	}
	if row.Best != grid[3] {
		t.Fatalf("Best = %+v, want the first configuration when nothing trades", row.Best)
	}
}

func TestAggregateDropCapsWinnersOnly(t *testing.T) {
	grid, _ := DropGrid(enum.Minutes15)
	opts := DefaultScreenOpts()
	row := AggregateDrop("GAZP", "", []DropConfigResult{
		{Params: grid[0], Trades: []backtest.Trade{dropTrade(trainAt(2), 100, "RSI")}},
	}, dropSplit, 0, opts)
	if row.PFMed != opts.PFCap || row.Capped != 1 || math.IsInf(row.PFMedStress, 0) {
		t.Fatalf("PFMed/Capped/PFMedStress = %v/%d/%v, want %v/1/finite", row.PFMed, row.Capped, row.PFMedStress, opts.PFCap)
	}
}

// dropFixture is 15m bars over Monday 2026-03-09 and Tuesday 2026-03-10, 07:00–23:45 MSK, flat at
// 100 with volume 10, except a red bar at Tuesday 10:00 (100 → 99, volume 100 = 0.1 of the 1000
// daily volume) followed by a steady climb that pushes RSI back up. Monday gives the engine the 64
// bars of lookback the strategy asks for before the signal.
func dropFixture() []backtest.Candle {
	msk, err := time.LoadLocation("Europe/Moscow")
	if err != nil {
		msk = time.UTC
	}
	var out []backtest.Candle
	signal := time.Date(2026, 3, 10, 10, 0, 0, 0, msk)
	for _, day := range []int{9, 10} {
		for m := 7 * 60; m < 24*60; m += 15 {
			ts := time.Date(2026, 3, day, 0, 0, 0, 0, msk).Add(time.Duration(m) * time.Minute)
			c := backtest.Candle{Time: ts, Open: 100, High: 100.1, Low: 99.9, Close: 100, Volume: 10}
			switch {
			case ts.Equal(signal):
				c = backtest.Candle{Time: ts, Open: 100, High: 100.05, Low: 98.95, Close: 99, Volume: 100}
			case ts.After(signal):
				k := float64(ts.Sub(signal) / (15 * time.Minute))
				open := 99 + 0.2*(k-1)
				c = backtest.Candle{Time: ts, Open: open, High: open + 0.25, Low: open - 0.05, Close: open + 0.2, Volume: 10}
			}
			out = append(out, c)
		}
	}
	return out
}

func TestScreenDropTickerRunsTheGridParamsVerbatim(t *testing.T) {
	// GAZP is a REGISTERED drop_bounce ticker with calibrated literals; the screener must grade it
	// on the grid config like every other ticker, so one grid config through ScreenDropTicker must
	// reproduce a direct engine run with that config.
	bars := dropFixture()
	daily := dailyCandlesMSK(60, 100, 2)
	opts := DefaultScreenOpts()
	grid, _ := DropGrid(enum.Minutes15)
	cfg := grid[0] // vol 0.05, RSI(4)@55, GREENVOL off

	want := backtest.Run(dropcore.NewWithParams("GAZP", 15*time.Minute, cfg), bars, daily, nil,
		backtest.Config{InitialCash: opts.Cash, Fraction: opts.Fraction, Commission: opts.Commission, Lot: 1, EntryAtNextOpen: true})
	train, _ := splitTrades(want.Trades, dropSplit)
	wantPF, wantN := profitFactor(train)
	if wantN == 0 {
		t.Fatal("fixture produces no train trades: the comparison would be 0 vs 0")
	}
	wantPF, _ = clampPF(wantPF, opts.PFCap)

	row := ScreenDropTicker(DropTickerInput{
		Ticker: "GAZP", Name: "Газпром", Interval: enum.Minutes15, Bars: bars, Daily: daily, Lot: 1, MinPriceIncrement: 0.01,
	}, []dropcore.Params{cfg}, dropSplit, opts)

	if row.PFMed != wantPF || row.TradesMed != float64(wantN) {
		t.Fatalf("PFMed/TradesMed = %v/%v, want %v/%d — ScreenDropTicker must run the grid config", row.PFMed, row.TradesMed, wantPF, wantN)
	}
	if row.Bars != len(bars) || row.TurnoverM <= 0 || row.DailyATRPct <= 0 {
		t.Fatalf("Bars/TurnoverM/DailyATRPct = %d/%v/%v, want %d/>0/>0", row.Bars, row.TurnoverM, row.DailyATRPct, len(bars))
	}
	if row.TickUnknown || row.TickPct <= 0 {
		t.Fatalf("TickPct/TickUnknown = %v/%v, want >0/false", row.TickPct, row.TickUnknown)
	}
}

func TestScreenDropTickerFlatSeriesProducesNoSignals(t *testing.T) {
	bars := dropFixture()
	for i := range bars {
		bars[i].Open, bars[i].High, bars[i].Low, bars[i].Close = 100, 100, 100, 100
	}
	grid, _ := DropGrid(enum.Minutes15)
	row := ScreenDropTicker(DropTickerInput{
		Ticker: "XXXX", Interval: enum.Minutes15, Bars: bars, Daily: dailyCandlesMSK(60, 100, 2), Lot: 1,
	}, grid, dropSplit, DefaultScreenOpts())
	if !row.NoSignals {
		t.Fatalf("NoSignals = false on a flat series (no red bar), row = %+v", row)
	}
	if !row.TickUnknown {
		t.Fatal("TickUnknown = false with MinPriceIncrement 0, want true")
	}
}
```

- [ ] **Step 2: Run the tests to verify they fail**

Run: `go test ./internal/service/backtest/ -run 'TestAggregateDrop|TestScreenDropTicker' -count=1`
Expected: build failure, `undefined: DropConfigResult`, `undefined: AggregateDrop`, `undefined: ScreenDropTicker`.

- [ ] **Step 3: Implement**

Append to `internal/service/backtest/drop_screen.go` (add import `time`):

```go
// DropConfigResult is one grid configuration's trade list on one ticker.
type DropConfigResult struct {
	Params dropcore.Params
	Trades []backtest.Trade
}

// DropRow is one ticker's drop-screening result. Medians run across grid configurations, on the
// train window unless the field says holdout.
type DropRow struct {
	Ticker      string
	Name        string
	TurnoverM   float64 // mean daily turnover, millions of RUB
	DailyATRPct float64 // mean weekday daily ATR, percent of close
	Bars        int     // bars replayed on the screened timeframe
	TickPct     float64 // two price ticks as a percentage of the last close
	TickUnknown bool    // the API gave no tick (or no price): TickPct is 0 and unmeasured

	PFMed       float64 // median raw train profit factor
	PFMedStress float64 // median train profit factor under stressed costs: the ranking key
	TradesMed   float64
	Plateau     float64 // share of configurations with PF >= PlateauPF at >= PlateauTrades trades
	Capped      int     // configurations whose raw train PF hit the cap
	SilentCfg   int     // configurations with no train trade

	PFMedHO       float64 // holdout, never a ranking key
	PFMedStressHO float64
	TradesMedHO   float64

	HalvesPos     float64 // median share of profitable half-years (trading configurations only)
	TopHalf       float64 // median share of the best half in summed profitable halves; 0 = none profitable
	SLShare       float64 // median share of stop exits (trading configurations only)
	RSIShare      float64 // median share of RSI exits
	GreenVolShare float64 // median share of GREENVOL exits
	HoldDays      float64 // median of per-configuration median holding time, calendar days

	Best       dropcore.Params // configuration with the highest raw train PF (reference only)
	BestPF     float64
	BestHalves []HalfResult // half-year breakdown of Best's train trades
	NoSignals  bool         // no configuration traded in either window
}

// AggregateDrop reduces one ticker's per-configuration results to a report row. tick is the
// instrument's price increment used by the cost stress (0 = unknown). It mirrors AggregateZone
// and adds the RSI and GREENVOL exit shares.
func AggregateDrop(ticker, name string, results []DropConfigResult, split time.Time, tick float64, opts ScreenOpts) DropRow {
	row := DropRow{Ticker: ticker, Name: name, NoSignals: true}
	n := len(results)
	pfs, pfsStress, counts := make([]float64, 0, n), make([]float64, 0, n), make([]float64, 0, n)
	pfsHO, pfsStressHO, countsHO := make([]float64, 0, n), make([]float64, 0, n), make([]float64, 0, n)
	var halvesPos, topHalf, sl, rsi, gv, hold []float64
	var plateau int
	var haveBest bool
	var bestTrain []backtest.Trade

	for _, r := range results {
		train, holdout := splitTrades(r.Trades, split)

		pf, cnt := profitFactor(train)
		if cnt > 0 {
			row.NoSignals = false
		} else {
			row.SilentCfg++
		}
		// The first configuration claims Best unconditionally, so Best is always a real grid
		// entry even when every raw PF is 0.
		if !haveBest || pf > row.BestPF {
			row.BestPF, row.Best, bestTrain = pf, r.Params, train
			haveBest = true
		}
		pf, capped := clampPF(pf, opts.PFCap)
		if capped {
			row.Capped++
		}
		if pf >= opts.PlateauPF && cnt >= opts.PlateauTrades {
			plateau++
		}
		pfs = append(pfs, pf)
		counts = append(counts, float64(cnt))

		pfS, _ := profitFactor(stressTrades(train, opts.Commission, tick))
		pfS, _ = clampPF(pfS, opts.PFCap)
		pfsStress = append(pfsStress, pfS)

		pfHO, cntHO := profitFactor(holdout)
		if cntHO > 0 {
			row.NoSignals = false
		}
		pfHO, _ = clampPF(pfHO, opts.PFCap)
		pfsHO = append(pfsHO, pfHO)
		countsHO = append(countsHO, float64(cntHO))
		pfSHO, _ := profitFactor(stressTrades(holdout, opts.Commission, tick))
		pfSHO, _ = clampPF(pfSHO, opts.PFCap)
		pfsStressHO = append(pfsStressHO, pfSHO)

		if cnt == 0 {
			continue // a silent configuration has no halves, exits or holding time to judge
		}
		pos, top, hasProfit := halvesStats(halfYearResults(train))
		halvesPos = append(halvesPos, pos)
		if hasProfit {
			topHalf = append(topHalf, top)
		}
		sl = append(sl, exitShare(train, "SL"))
		rsi = append(rsi, exitShare(train, "RSI"))
		gv = append(gv, exitShare(train, "GREENVOL"))
		hold = append(hold, medianHoldDays(train))
	}

	row.BestPF, _ = clampPF(row.BestPF, opts.PFCap)
	row.BestHalves = halfYearResults(bestTrain)
	row.PFMed = medianF(pfs)
	row.PFMedStress = medianF(pfsStress)
	row.TradesMed = medianF(counts)
	row.PFMedHO = medianF(pfsHO)
	row.PFMedStressHO = medianF(pfsStressHO)
	row.TradesMedHO = medianF(countsHO)
	row.HalvesPos = medianF(halvesPos)
	row.TopHalf = medianF(topHalf)
	row.SLShare = medianF(sl)
	row.RSIShare = medianF(rsi)
	row.GreenVolShare = medianF(gv)
	row.HoldDays = medianF(hold)
	if n > 0 {
		row.Plateau = float64(plateau) / float64(n)
	}
	return row
}

// DropTickerInput is everything ScreenDropTicker needs about one instrument.
type DropTickerInput struct {
	Ticker            string
	Name              string
	Interval          enum.Interval     // timeframe of Bars; sets the strategy's bar step
	Bars              []backtest.Candle // bars over the whole window (train + holdout)
	Daily             []backtest.Candle // daily bars with a warm-up lead-in (volume average and ATR)
	Lot               int32
	MinPriceIncrement float64 // 0 when unknown
}

// ScreenDropTicker replays every grid configuration over one ticker and reduces the runs to a
// report row. The strategy is built directly with dropcore.NewWithParams and NOT through
// DropBounceLookupOrGeneric: registered tickers carry calibrated literals, and grading them on
// those would make their rows incomparable with the rest. Entries fill at the next bar's open,
// as cmd/backtest runs drop_bounce.
func ScreenDropTicker(in DropTickerInput, cfgs []dropcore.Params, split time.Time, opts ScreenOpts) DropRow {
	cfg := backtest.Config{
		InitialCash:     opts.Cash,
		Fraction:        opts.Fraction,
		Commission:      opts.Commission,
		Lot:             in.Lot,
		EntryAtNextOpen: true,
	}
	step := in.Interval.ToTimeDuration()
	results := make([]DropConfigResult, 0, len(cfgs))
	for _, p := range cfgs {
		// drop_bounce needs no higher-timeframe series: htfCandles is nil.
		res := backtest.Run(dropcore.NewWithParams(in.Ticker, step, p), in.Bars, in.Daily, nil, cfg)
		results = append(results, DropConfigResult{Params: p, Trades: res.Trades})
	}
	row := AggregateDrop(in.Ticker, in.Name, results, split, in.MinPriceIncrement, opts)
	row.Bars = len(in.Bars)
	row.TurnoverM = backtest.MeanDailyTurnoverM(in.Bars, in.Lot)
	row.DailyATRPct = MeanDailyATRPct(in.Daily, screenDailyATRPeriod)

	var lastClose float64
	if len(in.Bars) > 0 {
		lastClose = in.Bars[len(in.Bars)-1].Close
	}
	row.TickPct, row.TickUnknown = tickPct(in.MinPriceIncrement, lastClose)
	return row
}
```

- [ ] **Step 4: Run the tests to verify they pass**

Run: `go test ./internal/service/backtest/ -count=1`
Expected: `ok`.

If `TestScreenDropTickerRunsTheGridParamsVerbatim` fails with "fixture produces no train trades", the fixture — not the code — is wrong. Do not weaken the assertion. Check against `docs/drop_bounce/strategy.md` §Вход: signal bar red, `Volume ≥ VolDailyMult · VolAvg` over weekday dailies with non-zero volume (`dailyCandlesMSK` volume 1000), ≥ `VolAvgPeriod`+ weekday dailies before the date, ATR computable, fill bar on the same date, ≥ 64 bars of lookback before the signal. Fix the fixture until a trade appears.

- [ ] **Step 5: Commit**

```bash
git commit -m "feat(drop_screen): aggregate grid runs into a ticker row

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>" -- internal/service/backtest/drop_screen.go internal/service/backtest/drop_screen_test.go
```

---

### Task 3: Gates, ranking, shortlist and markdown report

**Files:**
- Create: `internal/service/backtest/drop_screen_report.go`
- Create: `internal/service/backtest/drop_screen_report_test.go`

**Interfaces:**
- Consumes: Task 2 `DropRow`; Task 1 `DropGrid`, `dropGridVols`; existing `distributionOf([]float64) PFDist` (fields `Min, Q1, Median, Q3, Max, ShareAbove15 float64; N int`), `pct0(float64) string` (both in package `backtest`).
- Produces:
  - `type DropGates struct { MinTurnoverM, MaxTickPct, MinTrades float64 }`
  - `func DropRejectReason(r DropRow, g DropGates) string` ("" = passes turnover and trades gates)
  - `func FilterAndRankDrop(rows []DropRow, g DropGates) (ranked, noSignals, tickRejected, rejected []DropRow)`
  - `func DropShortlist(ranked []DropRow, n int) []string`
  - `type DropScreenMeta struct` (fields below)
  - `func RenderDropScreenMarkdown(ranked, noSignals, tickRejected, rejected []DropRow, meta DropScreenMeta) string`
  - const `DropShortlistKey = "DROP_BOUNCE_CANDIDATES"`

- [ ] **Step 1: Write the failing tests**

Create `internal/service/backtest/drop_screen_report_test.go`:

```go
package backtest

import (
	"strings"
	"testing"
	"time"

	"tinvest/internal/enum"
)

func dropRow(ticker string, pfStress, trades, turnover, tick float64) DropRow {
	return DropRow{Ticker: ticker, PFMedStress: pfStress, PFMed: pfStress + 0.2, TradesMed: trades, TurnoverM: turnover, TickPct: tick}
}

func tickers(rows []DropRow) string {
	out := make([]string, 0, len(rows))
	for _, r := range rows {
		out = append(out, r.Ticker)
	}
	return strings.Join(out, ",")
}

func TestFilterAndRankDrop(t *testing.T) {
	g := DropGates{MinTurnoverM: 50, MaxTickPct: 0.3, MinTrades: 20}
	unknownTick := dropRow("G", 1.1, 30, 100, 0)
	unknownTick.TickUnknown = true
	silent := dropRow("H", 0, 0, 100, 0.1)
	silent.NoSignals = true
	rows := []DropRow{
		dropRow("A", 1.5, 30, 100, 0.1),
		dropRow("B", 1.5, 40, 100, 0.1), // same key as A, more trades: ranks above A
		dropRow("C", 2.0, 25, 100, 0.1),
		dropRow("D", 3.0, 30, 49.9, 0.1), // turnover below the gate
		dropRow("E", 3.0, 19, 100, 0.1),  // too few trades
		dropRow("F", 3.0, 30, 100, 0.31), // ticks too expensive
		unknownTick,                      // unknown tick passes
		silent,                           // no signals: its own bucket, not "too few trades"
		dropRow("I", 1.2, 20, 50, 0.3),   // every gate exactly at its threshold passes
	}
	ranked, noSignals, tickRejected, rejected := FilterAndRankDrop(rows, g)
	if got := tickers(ranked); got != "C,B,A,I,G" {
		t.Fatalf("ranked = %s, want C,B,A,I,G", got)
	}
	if got := tickers(noSignals); got != "H" {
		t.Fatalf("noSignals = %s, want H", got)
	}
	if got := tickers(tickRejected); got != "F" {
		t.Fatalf("tickRejected = %s, want F", got)
	}
	if got := tickers(rejected); got != "D,E" {
		t.Fatalf("rejected = %s, want D,E", got)
	}
}

func TestFilterAndRankDropTickGateOff(t *testing.T) {
	ranked, _, tickRejected, _ := FilterAndRankDrop([]DropRow{dropRow("F", 3.0, 30, 100, 5)}, DropGates{MinTurnoverM: 50, MinTrades: 20})
	if len(tickRejected) != 0 || len(ranked) != 1 {
		t.Fatalf("MaxTickPct 0 must disable the tick gate: ranked=%d tickRejected=%d", len(ranked), len(tickRejected))
	}
}

func TestDropRejectReason(t *testing.T) {
	g := DropGates{MinTurnoverM: 50, MinTrades: 20}
	if got := DropRejectReason(dropRow("A", 1, 30, 100, 0), g); got != "" {
		t.Fatalf("passing row reason = %q, want empty", got)
	}
	got := DropRejectReason(dropRow("D", 1, 19, 49.9, 0), g)
	if !strings.Contains(got, "оборот 49.9 < 50") || !strings.Contains(got, "сделок 19 < 20") {
		t.Fatalf("reason = %q, want both the turnover and the trades failure", got)
	}
}

func TestDropShortlist(t *testing.T) {
	ranked := []DropRow{{Ticker: "A"}, {Ticker: "B"}, {Ticker: "C"}}
	if got := strings.Join(DropShortlist(ranked, 2), ","); got != "A,B" {
		t.Fatalf("n=2: %s, want A,B", got)
	}
	if got := strings.Join(DropShortlist(ranked, 10), ","); got != "A,B,C" {
		t.Fatalf("n beyond the ranking: %s, want A,B,C", got)
	}
	if got := strings.Join(DropShortlist(ranked, 0), ","); got != "A,B,C" {
		t.Fatalf("n=0 means the whole ranking: %s, want A,B,C", got)
	}
	if got := DropShortlist(nil, 10); len(got) != 0 {
		t.Fatalf("empty ranking: %v, want none", got)
	}
}

func dropMeta() DropScreenMeta {
	return DropScreenMeta{
		Interval: enum.Minutes15, Months: 36, HoldoutMonths: 6, TopN: 50, Shortlist: 2,
		Split: time.Date(2026, 4, 12, 0, 0, 0, 0, time.UTC), Gates: DropGates{MinTurnoverM: 50, MaxTickPct: 0.3, MinTrades: 20},
		PFCap: 10, Commission: 0.0005, Cash: 100000, Scanned: 5, Skipped: 1,
	}
}

func TestRenderDropScreenMarkdown(t *testing.T) {
	ranked := []DropRow{dropRow("C", 2.0, 25, 100, 0.1), dropRow("B", 1.5, 40, 100, 0.1), dropRow("A", 1.5, 30, 100, 0.1)}
	ranked[0].BestHalves = []HalfResult{{Label: "2024H1", PnL: 1500, Trades: 3}}
	md := RenderDropScreenMarkdown(ranked, []DropRow{{Ticker: "H", NoSignals: true}},
		[]DropRow{dropRow("F", 3, 30, 100, 0.31)}, []DropRow{dropRow("D", 3, 30, 49.9, 0.1)}, dropMeta())
	for _, want := range []string{
		"# drop_bounce screener",
		"Minutes15",
		"0.05, 0.1, 0.2, 0.3",
		"## Рейтинг",
		"## Детали топа",
		"2024H1 +1.5%",
		"## Нет сигналов",
		"## Отсеяно гейтом шага цены",
		"## Отсеяно гейтами",
		"оборот 49.9 < 50",
		"DROP_BOUNCE_CANDIDATES=C,B\n",
	} {
		if !strings.Contains(md, want) {
			t.Fatalf("report lacks %q:\n%s", want, md)
		}
	}
}

func TestRenderDropScreenMarkdownEmptyRanking(t *testing.T) {
	md := RenderDropScreenMarkdown(nil, nil, nil, nil, dropMeta())
	if !strings.Contains(md, "DROP_BOUNCE_CANDIDATES=\n") {
		t.Fatalf("empty ranking must still print the key with nothing after it:\n%s", md)
	}
}
```

- [ ] **Step 2: Run the tests to verify they fail**

Run: `go test ./internal/service/backtest/ -run 'Drop' -count=1`
Expected: build failure, `undefined: DropGates`, `undefined: FilterAndRankDrop`, etc.

- [ ] **Step 3: Implement**

Create `internal/service/backtest/drop_screen_report.go`:

```go
package backtest

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"tinvest/internal/enum"
)

// DropShortlistKey names the shortlist line the report and the console print: the calibration
// queue for the drop-bounce-calibrate skill.
const DropShortlistKey = "DROP_BOUNCE_CANDIDATES"

// DropGates are the drop screener's hard gates. MaxTickPct 0 disables the tick gate. Equality
// with a threshold passes.
type DropGates struct {
	MinTurnoverM float64 // mean daily turnover, millions of RUB
	MaxTickPct   float64 // two price ticks as a percentage of price
	MinTrades    float64 // median train trades across the grid
}

// DropRejectReason names every turnover/trades gate the row fails, "" when it passes both.
func DropRejectReason(r DropRow, g DropGates) string {
	var reasons []string
	if r.TurnoverM < g.MinTurnoverM {
		reasons = append(reasons, fmt.Sprintf("оборот %.1f < %.0f млн ₽", r.TurnoverM, g.MinTurnoverM))
	}
	if r.TradesMed < g.MinTrades {
		reasons = append(reasons, fmt.Sprintf("сделок %.0f < %.0f", r.TradesMed, g.MinTrades))
	}
	return strings.Join(reasons, "; ")
}

// FilterAndRankDrop sends signal-less rows to their own bucket first (a silent ticker is not
// "too few trades"), then applies the price-tick gate, then turnover and trades. Survivors rank by
// the median STRESSED profit factor descending, then median trades descending, then ticker. A row
// whose tick is unknown passes the tick gate: the report flags it instead of guessing a cost.
func FilterAndRankDrop(rows []DropRow, g DropGates) (ranked, noSignals, tickRejected, rejected []DropRow) {
	for _, r := range rows {
		switch {
		case r.NoSignals:
			noSignals = append(noSignals, r)
		case g.MaxTickPct > 0 && !r.TickUnknown && r.TickPct > g.MaxTickPct:
			tickRejected = append(tickRejected, r)
		case DropRejectReason(r, g) != "":
			rejected = append(rejected, r)
		default:
			ranked = append(ranked, r)
		}
	}
	sort.SliceStable(ranked, func(i, j int) bool {
		if ranked[i].PFMedStress != ranked[j].PFMedStress {
			return ranked[i].PFMedStress > ranked[j].PFMedStress
		}
		if ranked[i].TradesMed != ranked[j].TradesMed {
			return ranked[i].TradesMed > ranked[j].TradesMed
		}
		return ranked[i].Ticker < ranked[j].Ticker
	})
	byTicker := func(rs []DropRow) {
		sort.SliceStable(rs, func(i, j int) bool { return rs[i].Ticker < rs[j].Ticker })
	}
	byTicker(noSignals)
	byTicker(tickRejected)
	byTicker(rejected)
	return ranked, noSignals, tickRejected, rejected
}

// DropShortlist returns the first n tickers of the ranking; n <= 0 or beyond the ranking returns
// all of it.
func DropShortlist(ranked []DropRow, n int) []string {
	if n <= 0 || n > len(ranked) {
		n = len(ranked)
	}
	out := make([]string, 0, n)
	for _, r := range ranked[:n] {
		out = append(out, r.Ticker)
	}
	return out
}

// DropScreenMeta carries the run context shown in the report header.
type DropScreenMeta struct {
	Interval      enum.Interval
	Months        int
	HoldoutMonths int
	TopN          int
	Shortlist     int
	Split         time.Time
	Gates         DropGates
	PFCap         float64
	Commission    float64 // per side, fraction of turnover; the stress adds this again per side
	Cash          float64 // mock starting cash; half-year results are shown as a percentage of it
	Scanned       int
	Skipped       int
}

// RenderDropScreenMarkdown renders the drop screening report; the last line is the shortlist.
func RenderDropScreenMarkdown(ranked, noSignals, tickRejected, rejected []DropRow, meta DropScreenMeta) string {
	var b strings.Builder
	grid, _ := DropGrid(meta.Interval)
	gridSize := len(grid)
	vols, _ := dropGridVols(meta.Interval)
	volStr := make([]string, 0, len(vols))
	for _, v := range vols {
		volStr = append(volStr, fmt.Sprintf("%g", v))
	}
	pfs := make([]float64, 0, len(ranked))
	for _, r := range ranked {
		pfs = append(pfs, r.PFMedStress)
	}
	d := distributionOf(pfs)

	b.WriteString("# drop_bounce screener\n\n")
	fmt.Fprintf(&b, "Таймфрейм: %s. Окно: %d мес., holdout: последние %d мес. (срез %s).\n",
		meta.Interval, meta.Months, meta.HoldoutMonths, meta.Split.Format("2006-01-02"))
	fmt.Fprintf(&b, "Сетка: %d конфигураций — VolDailyMult {%s} × RSIPeriod {4, 6} × RSIExit {55, 75} × GreenVolExit {0, 1.5}; "+
		"прибито: стоп 1.0 дневного ATR(10), VolAvgPeriod 10, StartHour 7.\n", gridSize, strings.Join(volStr, ", "))
	fmt.Fprintf(&b, "Гейты: оборот >= %.0f млн ₽/день, медиана сделок на train >= %.0f, два шага цены <= %.2f%% цены (0 — гейт выключен). PF зажат сверху на %.1f.\n",
		meta.Gates.MinTurnoverM, meta.Gates.MinTrades, meta.Gates.MaxTickPct, meta.PFCap)
	fmt.Fprintf(&b, "Стресс издержек (`PFmed×2`, ключ рейтинга): к комиссии %.4f за сторону добавлена ещё одна такая же и два шага цены на круг.\n",
		meta.Commission)
	fmt.Fprintf(&b, "Вселенная: scanned=%d ranked=%d no-signal=%d tick-rejected=%d rejected=%d skipped=%d.\n\n",
		meta.Scanned, len(ranked), len(noSignals), len(tickRejected), len(rejected), meta.Skipped)

	b.WriteString("## Распределение PFmed×2 по ранжированной вселенной\n\n")
	fmt.Fprintf(&b, "min %.2f · Q1 %.2f · медиана %.2f · Q3 %.2f · max %.2f · доля PFmed×2 >= 1.5: %.0f%% (n=%d)\n\n",
		d.Min, d.Q1, d.Median, d.Q3, d.Max, d.ShareAbove15*100, d.N)

	top := ranked
	if meta.TopN > 0 && meta.TopN < len(top) {
		top = top[:meta.TopN]
	}

	b.WriteString("## Рейтинг\n\n")
	b.WriteString("| # | Ticker | Name | PFmed×2 | PFmed | TradesMed | Plateau | Capped | SilentCfg | PFmed×2 HO | Trades HO | Halves+ | Оборот, млн | ATR% дн | Tick% |\n")
	b.WriteString("|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|\n")
	for i, r := range top {
		tick := fmt.Sprintf("%.2f", r.TickPct)
		if r.TickUnknown {
			tick = "?"
		}
		fmt.Fprintf(&b, "| %d | %s | %s | %.2f | %.2f | %.0f | %s | %d/%d | %d/%d | %.2f | %.0f | %s | %.0f | %.2f | %s |\n",
			i+1, r.Ticker, r.Name, r.PFMedStress, r.PFMed, r.TradesMed, pct0(r.Plateau),
			r.Capped, gridSize, r.SilentCfg, gridSize, r.PFMedStressHO, r.TradesMedHO, pct0(r.HalvesPos),
			r.TurnoverM, r.DailyATRPct, tick)
	}
	b.WriteString("\n`Tick%` `?` — API не отдал шаг цены: стресс учитывает только комиссию. ")
	b.WriteString("`PFmed×2 HO` в сортировке не участвует: это красный флаг («работало и развалилось»), а не критерий отбора.\n\n")

	b.WriteString("## Детали топа\n\n")
	b.WriteString("| Ticker | SL% | RSI% | GREENVOL% | HoldD | TopHalf | Лучшая конфигурация | Полугодия лучшей конфигурации |\n")
	b.WriteString("|---|---|---|---|---|---|---|---|\n")
	for _, r := range top {
		topHalf := "—"
		if r.TopHalf > 0 {
			topHalf = pct0(r.TopHalf)
		}
		best := fmt.Sprintf("vol %g, RSI(%d)@%.0f, GV %g", r.Best.VolDailyMult, r.Best.RSIPeriod, r.Best.RSIExit, r.Best.GreenVolExit)
		halves := make([]string, 0, len(r.BestHalves))
		for _, h := range r.BestHalves {
			share := 0.0
			if meta.Cash > 0 {
				share = h.PnL / meta.Cash * 100
			}
			halves = append(halves, fmt.Sprintf("%s %+.1f%%", h.Label, share))
		}
		fmt.Fprintf(&b, "| %s | %s | %s | %s | %.1f | %s | %s | %s |\n",
			r.Ticker, pct0(r.SLShare), pct0(r.RSIShare), pct0(r.GreenVolShare), r.HoldDays, topHalf, best, strings.Join(halves, " · "))
	}
	b.WriteString("\n`SL%`/`RSI%`/`GREENVOL%` — доли выходов (остаток — закрытие в конце окна); `HoldD` — медианная длительность сделки, дней; ")
	b.WriteString("`TopHalf` — доля лучшего полугодия в сумме прибыльных (100% — весь профит из одного отрезка, «—» — прибыльных нет). ")
	b.WriteString("Полугодия лучшей конфигурации — train-сделки в процентах от стартового кэша.\n\n")

	if len(noSignals) > 0 {
		b.WriteString("## Нет сигналов\n\n")
		b.WriteString("Ни одной сделки ни в одной конфигурации ни в одном окне:\n\n")
		fmt.Fprintf(&b, "%s\n\n", strings.Join(DropShortlist(noSignals, 0), ", "))
	}

	if len(tickRejected) > 0 {
		b.WriteString("## Отсеяно гейтом шага цены\n\n")
		b.WriteString("| Ticker | Tick% | PFmed | PFmed×2 |\n|---|---|---|---|\n")
		for _, r := range tickRejected {
			fmt.Fprintf(&b, "| %s | %.2f | %.2f | %.2f |\n", r.Ticker, r.TickPct, r.PFMed, r.PFMedStress)
		}
		b.WriteString("\n")
	}

	if len(rejected) > 0 {
		b.WriteString("## Отсеяно гейтами\n\n")
		b.WriteString("| Ticker | Причина | PFmed×2 |\n|---|---|---|\n")
		for _, r := range rejected {
			fmt.Fprintf(&b, "| %s | %s | %.2f |\n", r.Ticker, DropRejectReason(r, meta.Gates), r.PFMedStress)
		}
		b.WriteString("\n")
	}

	b.WriteString("## Как это читать\n\n")
	b.WriteString("Шортлист — это **очередь на калибровку** (скилл drop-bounce-calibrate), а не доказательство края. ")
	b.WriteString("Верх рейтинга по построению содержит везунчиков: испытаний столько же, сколько тикеров умножить на конфигурации. ")
	b.WriteString("Планка приёмки не меняется — pooled OOS walk-forward калибровки (PF >= 1.2, >= 20 сделок, >= 3 живых фолда).\n\n")
	b.WriteString("## Шортлист\n\n")
	fmt.Fprintf(&b, "%s=%s\n", DropShortlistKey, strings.Join(DropShortlist(ranked, meta.Shortlist), ","))
	return b.String()
}
```

- [ ] **Step 4: Run the tests to verify they pass**

Run: `go test ./internal/service/backtest/ -count=1`
Expected: `ok`.

- [ ] **Step 5: Commit**

```bash
git add internal/service/backtest/drop_screen_report.go internal/service/backtest/drop_screen_report_test.go
git commit -m "feat(drop_screen): gates, ranking, shortlist and markdown report

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>" -- internal/service/backtest/drop_screen_report.go internal/service/backtest/drop_screen_report_test.go
```

---

### Task 4: `cmd/dropscreen`

**Files:**
- Create: `cmd/dropscreen/main.go`
- Create: `cmd/dropscreen/main_test.go`

**Interfaces:**
- Consumes: Tasks 1–3 (`svc.DropGrid`, `svc.ScreenDropTicker`, `svc.DropTickerInput`, `svc.DropRow`, `svc.DropGates`, `svc.FilterAndRankDrop`, `svc.DropShortlist`, `svc.DropShortlistKey`, `svc.DropScreenMeta`, `svc.RenderDropScreenMarkdown`); existing `svc.DefaultScreenOpts`, `svc.NewCandleProvider(marketData, cacheDir)` with `.Load(ctx, ticker, id, interval, from, to, refresh) ([]backtest.Candle, error)`, `screenrun.*` (see `cmd/zonescreen/main.go`), `semaphore.New(n)` with `Acquire/Release`, `logger.Init()`, `grpcclient.NewClientGrpc(addr, token)`.
- Produces: the binary; `validate(runCfg) error`, `parseInterval(string) (enum.Interval, error)`.

- [ ] **Step 1: Write the failing tests**

Create `cmd/dropscreen/main_test.go`:

```go
package main

import (
	"testing"

	"tinvest/internal/enum"
)

func validCfg() runCfg {
	return runCfg{interval: enum.Minutes15, months: 36, holdoutMonths: 6, workers: 1, maxTickPct: 0.3, minTrades: 20, shortlist: 10}
}

func TestValidateAcceptsDefaults(t *testing.T) {
	if err := validate(validCfg()); err != nil {
		t.Fatalf("validate rejected defaults: %v", err)
	}
}

func TestValidateRejects(t *testing.T) {
	cases := map[string]func(*runCfg){
		"holdout == months":     func(c *runCfg) { c.holdoutMonths = c.months },
		"zero workers":          func(c *runCfg) { c.workers = 0 },
		"negative tick gate":    func(c *runCfg) { c.maxTickPct = -0.1 },
		"negative min trades":   func(c *runCfg) { c.minTrades = -1 },
		"negative shortlist":    func(c *runCfg) { c.shortlist = -1 },
		"Minutes5 not screened": func(c *runCfg) { c.interval = enum.Minutes5 },
		"Day1 not screened":     func(c *runCfg) { c.interval = enum.Day1 },
	}
	for name, mutate := range cases {
		c := validCfg()
		mutate(&c)
		if err := validate(c); err == nil {
			t.Fatalf("%s: validate accepted it, want an error", name)
		}
	}
}

func TestParseInterval(t *testing.T) {
	for s, want := range map[string]enum.Interval{"Minutes15": enum.Minutes15, "Minutes30": enum.Minutes30, "Hour1": enum.Hour1} {
		got, err := parseInterval(s)
		if err != nil || got != want {
			t.Fatalf("parseInterval(%q) = %v, %v; want %v", s, got, err, want)
		}
	}
	for _, s := range []string{"Minutes5", "Day1", "15m", ""} {
		if _, err := parseInterval(s); err == nil {
			t.Fatalf("parseInterval(%q) accepted, want an error", s)
		}
	}
}
```

- [ ] **Step 2: Run the tests to verify they fail**

Run: `go test ./cmd/dropscreen/ -count=1`
Expected: build failure (no non-test files / `undefined: runCfg`).

- [ ] **Step 3: Implement**

Create `cmd/dropscreen/main.go`:

```go
// Command dropscreen ranks the tradable RUB share universe by how well the drop_bounce strategy
// fits each ticker on one timeframe. Like cmd/zonescreen it replays the real strategy engine over a
// fixed grid (32 configurations) and ranks by the median profit factor under STRESSED costs
// (double commission plus two price ticks). The report ends with a shortlist line — the queue for
// the drop-bounce-calibrate skill. All gRPC/file I/O lives here; the scoring is pure
// (internal/service/backtest/drop_screen*.go). See docs/drop_bounce/screener.md.
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"tinvest/internal/enum"
	svc "tinvest/internal/service/backtest"
	"tinvest/internal/service/backtest/screenrun"
	grpcclient "tinvest/pkg/client/grpc"
	"tinvest/pkg/logger"
	"tinvest/pkg/semaphore"
)

func main() {
	var (
		intervalS     = flag.String("interval", "Minutes15", "timeframe: Minutes15|Minutes30|Hour1")
		months        = flag.Int("months", 36, "lookback period in months")
		holdoutMonths = flag.Int("holdout-months", 6, "trailing months held out of the ranking window")
		minTurnoverM  = flag.Float64("min-turnover", 50, "gate: minimum mean daily turnover in millions of RUB")
		maxTickPct    = flag.Float64("max-tick-pct", 0.3, "gate: maximum two price ticks as a percentage of the last close (0 = off)")
		minTrades     = flag.Float64("min-trades", 20, "gate: minimum median train trades across the grid")
		topN          = flag.Int("top", 50, "ranking rows in the report (0 = all)")
		shortlist     = flag.Int("shortlist", 10, "tickers in the DROP_BOUNCE_CANDIDATES line (0 = whole ranking)")
		workers       = flag.Int("workers", 1, "concurrent tickers (default 1: single-core rule)")
		pfCap         = flag.Float64("pf-cap", 10, "cap on profit factor for ranking (0 = no cap)")
		cash          = flag.Float64("cash", 100000, "starting mock cash")
		fraction      = flag.Float64("fraction", 1.0, "fraction of cash per entry")
		commission    = flag.Float64("commission", 0.0005, "commission as a fraction of turnover, per side")
		tickersCSV    = flag.String("tickers", "", "comma-separated tickers instead of the full universe (diagnostics/smoke runs)")
		outDir        = flag.String("out", "reports/drop_screen", "report output directory")
		refresh       = flag.Bool("refresh", false, "force candle refetch (ignore cache)")
		pause         = flag.Duration("pause", 0, "idle time after each ticker, e.g. 500ms or 2s: trades wall-clock for a cooler CPU")
	)
	flag.Parse()
	logger.Init()

	interval, err := parseInterval(*intervalS)
	if err != nil {
		log.Fatalf("dropscreen: %v", err)
	}
	opts := svc.DefaultScreenOpts()
	opts.PFCap = *pfCap
	opts.Cash = *cash
	opts.Fraction = *fraction
	opts.Commission = *commission

	if err := run(context.Background(), runCfg{
		interval: interval, months: *months, holdoutMonths: *holdoutMonths, topN: *topN, shortlist: *shortlist,
		workers: *workers, minTurnoverM: *minTurnoverM, maxTickPct: *maxTickPct, minTrades: *minTrades,
		tickers: screenrun.SplitCSV(*tickersCSV), outDir: *outDir, refresh: *refresh, pause: *pause, opts: opts,
	}); err != nil {
		log.Fatalf("dropscreen: %v", err)
	}
}

type runCfg struct {
	interval                                 enum.Interval
	months, holdoutMonths, topN, shortlist   int
	workers                                  int
	minTurnoverM, maxTickPct, minTrades      float64
	tickers                                  []string
	outDir                                   string
	refresh                                  bool
	pause                                    time.Duration
	opts                                     svc.ScreenOpts
}

// parseInterval accepts exactly the timeframes the screener grids exist for.
func parseInterval(s string) (enum.Interval, error) {
	for _, iv := range []enum.Interval{enum.Minutes15, enum.Minutes30, enum.Hour1} {
		if strings.EqualFold(s, iv.String()) {
			return iv, nil
		}
	}
	return 0, fmt.Errorf("unknown -interval %q (want Minutes15|Minutes30|Hour1; Minutes5 is never calibrated)", s)
}

// validate rejects flag combinations that would silently produce a meaningless run.
func validate(cfg runCfg) error {
	if _, err := svc.DropGrid(cfg.interval); err != nil {
		return err
	}
	if cfg.holdoutMonths >= cfg.months {
		return fmt.Errorf("-holdout-months (%d) must be smaller than -months (%d)", cfg.holdoutMonths, cfg.months)
	}
	if cfg.workers < 1 {
		return fmt.Errorf("-workers must be at least 1")
	}
	if cfg.maxTickPct < 0 {
		return fmt.Errorf("-max-tick-pct must be >= 0 (0 disables the gate)")
	}
	if cfg.minTrades < 0 {
		return fmt.Errorf("-min-trades must be >= 0")
	}
	if cfg.shortlist < 0 {
		return fmt.Errorf("-shortlist must be >= 0 (0 = whole ranking)")
	}
	return nil
}

func run(ctx context.Context, cfg runCfg) error {
	if err := validate(cfg); err != nil {
		return err
	}
	if workers, capped := screenrun.EffectiveWorkers(cfg.workers, cfg.refresh); capped {
		fmt.Printf("dropscreen: -refresh forces -workers %d -> %d to avoid punching holes in the local candle cache (see screenrun.RefreshWorkerCap)\n",
			cfg.workers, workers)
		cfg.workers = workers
	}
	token, err := screenrun.LoadToken()
	if err != nil {
		return err
	}
	client, err := grpcclient.NewClientGrpc(screenrun.APIAddress, token)
	if err != nil {
		return fmt.Errorf("grpc client: %w", err)
	}
	universe, err := screenrun.LoadUniverse(ctx, client, cfg.tickers)
	if err != nil {
		return err
	}

	to := time.Now()
	from := to.AddDate(0, -cfg.months, 0)
	// A year of lead-in warms the daily volume average and ATR, exactly as cmd/backtest does.
	dailyFrom := from.AddDate(-1, 0, 0)
	split := to.AddDate(0, -cfg.holdoutMonths, 0)

	provider := svc.NewCandleProvider(client.MarketDataServiceClient(), screenrun.CacheDir)
	grid, err := svc.DropGrid(cfg.interval)
	if err != nil {
		return err
	}

	sem := semaphore.New(cfg.workers)
	var wg sync.WaitGroup
	var mu sync.Mutex
	var rows []svc.DropRow
	var done, skipped int32

	for _, u := range universe {
		wg.Add(1)
		sem.Acquire()
		go func(u screenrun.ShareInfo) {
			defer wg.Done()
			defer sem.Release()

			bars, err := provider.Load(ctx, u.Ticker, u.ID, cfg.interval, from, to, cfg.refresh)
			if err != nil {
				atomic.AddInt32(&skipped, 1)
				fmt.Printf("dropscreen %s: skip (load %s: %v)\n", u.Ticker, cfg.interval, err)
				return
			}
			daily, err := provider.Load(ctx, u.Ticker, u.ID, enum.Day1, dailyFrom, to, cfg.refresh)
			if err != nil {
				atomic.AddInt32(&skipped, 1)
				fmt.Printf("dropscreen %s: skip (load daily: %v)\n", u.Ticker, err)
				return
			}

			row := svc.ScreenDropTicker(svc.DropTickerInput{
				Ticker: u.Ticker, Name: u.Name, Interval: cfg.interval, Bars: bars, Daily: daily,
				Lot: u.Lot, MinPriceIncrement: u.MinPriceIncrement,
			}, grid, split, cfg.opts)
			n := atomic.AddInt32(&done, 1)
			fmt.Printf("dropscreen [%d/%d] %s: PFmed=%.2f PFmed×2=%.2f trades=%.0f tick=%.2f%% turnover=%.0fM atr=%.2f%% bars=%d\n",
				n, len(universe), u.Ticker, row.PFMed, row.PFMedStress, row.TradesMed, row.TickPct, row.TurnoverM, row.DailyATRPct, row.Bars)

			mu.Lock()
			rows = append(rows, row)
			mu.Unlock()

			// Held inside the semaphore slot on purpose: releasing first would let the next
			// ticker start while this one idles, and the pool would stay just as hot.
			screenrun.PauseAfterTicker(ctx, cfg.pause)
		}(u)
	}
	wg.Wait()

	gates := svc.DropGates{MinTurnoverM: cfg.minTurnoverM, MaxTickPct: cfg.maxTickPct, MinTrades: cfg.minTrades}
	ranked, noSignals, tickRejected, rejected := svc.FilterAndRankDrop(rows, gates)
	meta := svc.DropScreenMeta{
		Interval: cfg.interval, Months: cfg.months, HoldoutMonths: cfg.holdoutMonths, TopN: cfg.topN,
		Shortlist: cfg.shortlist, Split: split, Gates: gates, PFCap: cfg.opts.PFCap,
		Commission: cfg.opts.Commission, Cash: cfg.opts.Cash, Scanned: len(universe), Skipped: int(skipped),
	}

	if err := os.MkdirAll(cfg.outDir, 0o755); err != nil {
		return fmt.Errorf("mkdir out dir: %w", err)
	}
	path := filepath.Join(cfg.outDir, fmt.Sprintf("drop_screen_%s_%s.md", cfg.interval, time.Now().Format("20060102_150405")))
	if err := os.WriteFile(path, []byte(svc.RenderDropScreenMarkdown(ranked, noSignals, tickRejected, rejected, meta)), 0o644); err != nil {
		return fmt.Errorf("write %s: %w", path, err)
	}
	fmt.Printf("dropscreen report: %s (scanned=%d ranked=%d no-signal=%d tick-rejected=%d rejected=%d skipped=%d)\n",
		path, len(universe), len(ranked), len(noSignals), len(tickRejected), len(rejected), skipped)
	fmt.Printf("%s=%s\n", svc.DropShortlistKey, strings.Join(svc.DropShortlist(ranked, cfg.shortlist), ","))
	return nil
}
```

Run `gofmt -w cmd/dropscreen/main.go` (aligns the `runCfg` fields).

- [ ] **Step 4: Run the tests and the build**

Run: `go test ./cmd/dropscreen/ -count=1 && go build ./internal/... ./pkg/... ./cmd/...`
Expected: `ok`, build without errors.

- [ ] **Step 5: Commit**

```bash
git add cmd/dropscreen/main.go cmd/dropscreen/main_test.go
git commit -m "feat(dropscreen): CLI ranking the universe for drop_bounce calibration

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>" -- cmd/dropscreen/main.go cmd/dropscreen/main_test.go
```

---

### Task 5: Documentation and lint gate

**Files:**
- Create: `docs/drop_bounce/screener.md`
- Modify (leave UNCOMMITTED, they carry unrelated pending edits): `docs/drop_bounce/strategy.md`, `CLAUDE.md`, `.claude/skills/drop-bounce-calibrate/SKILL.md`

**Interfaces:**
- Consumes: Tasks 1–4 (flag names, defaults, report sections, shortlist key).
- Produces: documentation only.

- [ ] **Step 1: Write `docs/drop_bounce/screener.md`**

```markdown
# drop_bounce — скринер тикеров под калибровку

Команда: `cmd/dropscreen`. Ядро: `internal/service/backtest/drop_screen.go`, отчёт —
`drop_screen_report.go`, обвязка — `internal/service/backtest/screenrun` (общая с zonescreen).
Спека: `docs/superpowers/specs/2026-10-12-drop-bounce-screener-design.md`.

## Зачем

Калибровка drop_bounce идёт по тикеру и стоит дорого (скилл `drop-bounce-calibrate`: три
таймфрейма, фазовый walk-forward на одном ядре). Скринер отвечает на один вопрос: на какие тикеры
потратить калибровку и в каком порядке. Это не доказательство края — судья остаётся прежним:
pooled OOS walk-forward калибровки.

## Принцип

Настоящий движок drop_bounce прогоняется по каждому тикеру торгуемой RUB-вселенной на
фиксированной сетке одного таймфрейма. Тикеры ранжируются по медиане profit factor при стрессе
издержек на train-окне. Стратегия строится через `core.NewWithParams` в обход реестра: тикеры с
пакетом оцениваются на той же сетке, что и все.

## Сетка

32 конфигурации: `VolDailyMult` × `RSIPeriod` {4, 6} × `RSIExit` {55, 75} × `GreenVolExit` {0, 1.5}.

| таймфрейм | `VolDailyMult` |
|---|---|
| Minutes15 | 0.05, 0.1, 0.2, 0.3 |
| Minutes30 | 0.1, 0.2, 0.3, 0.5 |
| Hour1 | 0.15, 0.25, 0.4, 0.6 |

Прибито: `StopDailyATR` 1.0, `DailyATRPeriod` 10, `VolAvgPeriod` 10, `StartHour` 7. Двигаются только
оси, чей полезный диапазон зависит от бумаги: порог объёма (как часто бумага даёт всплеск) и две
противоположные формы выхода — быстрая на откате и удерживающая. Стоп — свойство настроенной точки,
поэтому в сетку не входит.

## Метрики

Медианы по конфигурациям, на train (`-months` минус `-holdout-months`; сделка относится к окну по
времени входа):

- **PFmed×2** — PF при стрессе: ещё одна комиссия на сторону и два шага цены на круг. Ключ рейтинга;
  при равенстве — медиана сделок, затем тикер.
- **PFmed**, **TradesMed**, **Plateau** (доля конфигураций с PF ≥ 1.3 на ≥ 10 сделках), **Capped**,
  **SilentCfg**.
- **Holdout** — PFmed×2 и сделки на последних `-holdout-months`; справочно, в рейтинге не участвует.
- **Halves+**, **TopHalf** — доля прибыльных полугодий и доля лучшего в прибыли.
- **SL% / RSI% / GREENVOL%** — доли выходов; **HoldD** — медианное удержание, дней.
- Структура бумаги: оборот, дневной ATR %, два шага цены %.
- Лучшая конфигурация и её полугодия — справочно.

## Гейты

| Флаг | Дефолт | Отсеивает, если |
|---|---|---|
| `-min-turnover` | 50 | средний дневной оборот ниже, млн ₽ |
| `-max-tick-pct` | 0.3 | два шага дороже, % от последней цены; 0 — гейт выключен; неизвестный шаг не отсеивается |
| `-min-trades` | 20 | медиана сделок на train ниже |

Равенство порогу проходит. Тикер, не давший ни одной сделки ни в одной конфигурации, идёт в «Нет
сигналов», а не в отсеянные. Гейта по ATR нет: стоп и порог относительные, низкую волатильность
против издержек ловит гейт шага.

## Запуск

Собранный бинарь на одном ядре:

```bash
GOMAXPROCS=1 nice -n 19 go build -o reports/drop_screen/dropscreen ./cmd/dropscreen
taskset -c 0 nice -n 19 env GOMAXPROCS=1 reports/drop_screen/dropscreen -interval Minutes15
```

Флаги: `-interval` (Minutes15 | Minutes30 | Hour1; 5m не скринится), `-months` (36),
`-holdout-months` (6), гейты выше, `-top` (50; 0 — все), `-shortlist` (10; 0 — весь рейтинг),
`-workers` (1), `-pf-cap` (10), `-cash`, `-fraction`, `-commission` (0.0005), `-tickers` (пробный
прогон по списку), `-pause`, `-refresh` (перекачать свечи; ограничивает `-workers` до 2), `-out`
(`reports/drop_screen`).

Отчёт `reports/drop_screen/drop_screen_<таймфрейм>_<время>.md`: шапка, распределение PFmed×2,
рейтинг, детали топа, нет сигналов, отсеянные шагом, отсеянные гейтами с причиной, шортлист
`DROP_BOUNCE_CANDIDATES=...`. Та же строка печатается в консоль.

## Дальше

Тикеры шортлиста по очереди идут в `/drop-bounce-calibrate <TICKER>`. Скринер гоняется отдельно на
каждом таймфрейме; тикер, попавший в шортлист хотя бы одного, — кандидат на калибровку.
```

- [ ] **Step 2: Link it from the strategy doc, CLAUDE.md and the calibration skill (do not commit these three)**

In `docs/drop_bounce/strategy.md`, after the `## Пакеты тикеров` section's paragraph, add:

```markdown
Кандидатов на калибровку подбирает скринер `cmd/dropscreen` — `docs/drop_bounce/screener.md`.
```

In `CLAUDE.md`, in the `drop_bounce` paragraph, right after `starter grid \`data/params/drop_bounce/grid_example.json\``, insert:

```
; подбор тикеров под калибровку — `cmd/dropscreen` (`-interval Minutes15|Minutes30|Hour1`, шортлист `DROP_BOUNCE_CANDIDATES`), docs/drop_bounce/screener.md
```

In `.claude/skills/drop-bounce-calibrate/SKILL.md`, after the line `Аргументы: \`<TICKER> [заметки владельца]\`...` paragraph (the first paragraph under the title), add:

```markdown
Кандидатов брать из шортлиста скринера `cmd/dropscreen` (`docs/drop_bounce/screener.md`).
```

- [ ] **Step 3: Run the full gate**

Run: `./bin/mage ci`
Expected: lint clean, `go test -race ./...` green, no mock drift. Fix any lint finding in the files of Tasks 1–4 (not in unrelated files) and re-run.

- [ ] **Step 4: Commit only the new doc (and any lint fixes to files from Tasks 1–4)**

```bash
git add docs/drop_bounce/screener.md
git commit -m "docs(drop_bounce): screener reference

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>" -- docs/drop_bounce/screener.md
```
Then `git status --short` must still show `CLAUDE.md`, `docs/drop_bounce/strategy.md` and `.claude/skills/drop-bounce-calibrate/SKILL.md` as modified and uncommitted.

---

### Task 6: First run and sanity check (controller, not a subagent)

**Files:** none in git; report under `reports/drop_screen/`.

- [ ] **Step 1: Build and run on 15m, single core, in the background**

```bash
GOMAXPROCS=1 nice -n 19 go build -o reports/drop_screen/dropscreen ./cmd/dropscreen
taskset -c 0 nice -n 19 env GOMAXPROCS=1 reports/drop_screen/dropscreen -interval Minutes15 > reports/drop_screen/run_Minutes15.log 2>&1
```
Expected: ~45 min, last lines are the report path and `DROP_BOUNCE_CANDIDATES=...`.

- [ ] **Step 2: Sanity check**

Find GAZP and T in the report (ranked, rejected or no signals) and compare with the known walk-forward results: T 15m pooled 1.184/166, GAZP 15m 1.756/12 (12 trades — expected to sit low on trades). Report the shortlist, where GAZP and T landed, and whether the ordering contradicts the walk-forward. This checks that the numbers make sense; it does not validate the screener. Do not tune the grid or gates on this result without the owner's request.
