# Momentum Strategy (EMA200 + MACD + Volume + Daily-ATR) Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add a long-only hourly trend-momentum strategy (close>EMA200, bullish MACD cross, above-average volume, remaining daily-ATR room) on top of the existing backtest engine, ticker-agnostic with per-ticker calibrated params, reporting SL/TP/ATR and a human-readable entry reason.

**Architecture:** Mirror the `levels` package: a pure ticker-agnostic `momentum/strategy/core` implementing `strategy.Strategy`, per-ticker packages supplying calibrated `Params`, a registry binding, and a `-strategy momentum` switch in `cmd/backtest`. Add a reusable MACD indicator to `pkg/indicators`. Extend the shared engine additively (daily OHLC + today's intraday extent + EntryReason + TP fill) without breaking `levels`/`scalping`.

**Tech Stack:** Go 1.25, existing `internal/domain/backtest` engine, `pkg/indicators` (ATR, VolumeConfirmed, new MACD), `internal/domain/ema`.

**Source spec:** `docs/superpowers/specs/2026-06-08-momentum-macd-strategy-design.md`
**Strategy explainer (keep in sync):** `docs/momentum/strategy.md`

**Conventions (read once):**
- All `Params` fields are `int`/`float64` (flags as `int` 0/1) so reflection grid calibration can sweep them.
- `Decide` must be pure: compute indicators from `MarketData`, no I/O.
- Series are oldest-first; `md.Price` is the last close.
- Run `gofmt`/`go build ./...` before each commit. Run `go test ./...` where noted.

---

## File Structure

- Create `pkg/indicators/macd.go` — reusable MACD (fast/slow/signal EMA lines). Local EMA to keep `pkg` free of `internal` deps.
- Create `pkg/indicators/macd_test.go`.
- Modify `internal/service/trading_strategy/scalping/model/signal.go` — add `EntryReason`.
- Modify `internal/service/trading_strategy/scalping/strategy/strategy.go` — add `DailyHighs`, `DailyLows`, `TodayHigh`, `TodayLow` to `MarketData`.
- Modify `internal/domain/backtest/types.go` — add `Trade.EntryReason`.
- Modify `internal/domain/backtest/portfolio.go` — thread `entryReason` through `open`/`close`.
- Modify `internal/domain/backtest/engine.go` — TP fill, daily highs/lows, today's extent, pass `EntryReason` to `open`.
- Modify `internal/domain/backtest/report.go` — add "Причина входа" column (md + csv).
- Modify `internal/domain/backtest/engine_test.go` — tests for new engine behavior.
- Create `internal/service/trading_strategy/momentum/strategy/core/core.go` — pure strategy.
- Create `internal/service/trading_strategy/momentum/strategy/core/core_test.go`.
- Create `internal/service/trading_strategy/momentum/strategy/rusal/rusal.go` — RUAL ticker + DefaultParams.
- Create `internal/service/backtest/momentum_registry.go` — `MomentumLookupOrGeneric` + `genericMomentumDefaults`.
- Create `internal/service/backtest/momentum_registry_test.go`.
- Modify `cmd/backtest/main.go` — `-strategy momentum` switch.
- Create `data/params/rusal/momentum_grid.json` — calibration grid.

---

## Task 1: MACD indicator in pkg/indicators

**Files:**
- Create: `pkg/indicators/macd.go`
- Test: `pkg/indicators/macd_test.go`

- [ ] **Step 1: Write the failing test**

```go
package indicators

import (
	"math"
	"testing"
)

func approx(a, b float64) bool { return math.Abs(a-b) < 1e-9 }

func TestMACDInsufficientHistory(t *testing.T) {
	m, s := MACD([]float64{1, 2, 3}, 12, 26, 9)
	if m != nil || s != nil {
		t.Fatalf("want nil,nil for short history; got %v,%v", m, s)
	}
}

func TestMACDInvalidPeriods(t *testing.T) {
	closes := make([]float64, 100)
	for i := range closes {
		closes[i] = float64(i)
	}
	if m, s := MACD(closes, 26, 12, 9); m != nil || s != nil { // fast>=slow
		t.Fatalf("want nil for fast>=slow")
	}
	if m, s := MACD(closes, 0, 26, 9); m != nil || s != nil {
		t.Fatalf("want nil for non-positive period")
	}
}

func TestMACDRisingSeriesPositive(t *testing.T) {
	// A steadily rising series: fast EMA above slow EMA => MACD line positive at the end.
	closes := make([]float64, 100)
	for i := range closes {
		closes[i] = float64(i)
	}
	m, s := MACD(closes, 12, 26, 9)
	if len(m) != len(closes) || len(s) != len(closes) {
		t.Fatalf("lengths: macd=%d signal=%d want %d", len(m), len(s), len(closes))
	}
	if m[len(m)-1] <= 0 {
		t.Fatalf("rising series: macd[last]=%f want >0", m[len(m)-1])
	}
}

func TestMACDLineEqualsFastMinusSlowEMA(t *testing.T) {
	closes := make([]float64, 60)
	for i := range closes {
		closes[i] = math.Sin(float64(i)/5) * 10
	}
	m, _ := MACD(closes, 12, 26, 9)
	fast := ema(closes, 12)
	slow := ema(closes, 26)
	last := len(closes) - 1
	if !approx(m[last], fast[last]-slow[last]) {
		t.Fatalf("macd[last]=%f want fast-slow=%f", m[last], fast[last]-slow[last])
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./pkg/indicators/ -run TestMACD -v`
Expected: FAIL — `undefined: MACD`, `undefined: ema`.

- [ ] **Step 3: Write minimal implementation**

```go
package indicators

// MACD returns the MACD line and signal line over closes (oldest-first, aligned
// to closes). macdLine = EMA(closes, fast) - EMA(closes, slow);
// signalLine = EMA(macdLine, signalPeriod).
//
// Returns nil, nil when any period is non-positive, when fast >= slow, or when
// there is not enough history (len(closes) < slow) — the insufficient-history
// rule is silent, mirroring ATR/VolumeConfirmed. The lines are only meaningful
// once there is ample history (callers size their lookback to slow+signalPeriod
// plus margin); early positions carry seeding bias and should not be inspected.
func MACD(closes []float64, fast, slow, signalPeriod int) (macdLine, signalLine []float64) {
	if fast <= 0 || slow <= 0 || signalPeriod <= 0 || fast >= slow || len(closes) < slow {
		return nil, nil
	}
	emaFast := ema(closes, fast)
	emaSlow := ema(closes, slow)
	macdLine = make([]float64, len(closes))
	for i := range closes {
		macdLine[i] = emaFast[i] - emaSlow[i]
	}
	signalLine = ema(macdLine, signalPeriod)
	return macdLine, signalLine
}

// ema is a sliding EMA seeded by the SMA of the first period values; positions
// before period-1 are 0. Local to keep pkg/indicators free of internal deps
// (mirrors internal/domain/ema.Compute).
func ema(values []float64, period int) []float64 {
	out := make([]float64, len(values))
	if period <= 0 || len(values) < period {
		return out
	}
	var sum float64
	for i := 0; i < period; i++ {
		sum += values[i]
	}
	out[period-1] = sum / float64(period)
	k := 2.0 / float64(period+1)
	for i := period; i < len(values); i++ {
		out[i] = (values[i]-out[i-1])*k + out[i-1]
	}
	return out
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./pkg/indicators/ -run TestMACD -v`
Expected: PASS (all 4).

- [ ] **Step 5: Commit**

```bash
gofmt -w pkg/indicators/macd.go pkg/indicators/macd_test.go
git add pkg/indicators/macd.go pkg/indicators/macd_test.go
git commit -m "feat(indicators): add reusable MACD with local EMA"
```

---

## Task 2: EntryReason plumbing (Signal → Trade → report)

**Files:**
- Modify: `internal/service/trading_strategy/scalping/model/signal.go`
- Modify: `internal/domain/backtest/types.go:26-39`
- Modify: `internal/domain/backtest/portfolio.go:32-58` and `:72-104`
- Modify: `internal/domain/backtest/engine.go:67`
- Modify: `internal/domain/backtest/report.go`
- Test: `internal/domain/backtest/report_test.go` (create if absent) or add to engine_test.go

- [ ] **Step 1: Write the failing test**

Create `internal/domain/backtest/report_entryreason_test.go`:

```go
package backtest

import (
	"strings"
	"testing"
	"time"
)

func TestRenderMarkdownShowsEntryReason(t *testing.T) {
	trades := []Trade{{
		EntryTime:   time.Date(2026, 1, 2, 10, 0, 0, 0, time.UTC),
		EntryPrice:  36.2,
		ExitTime:    time.Date(2026, 1, 2, 15, 0, 0, 0, time.UTC),
		ExitPrice:   38.0,
		Reason:      "TP",
		EntryReason: "Тренд↑; MACD кросс; объём 1.8×",
	}}
	out := RenderMarkdown(Meta{Ticker: "RUAL", Interval: "Hour1"}, Metrics{}, trades, nil)
	if !strings.Contains(out, "Причина входа") {
		t.Fatal("markdown header missing 'Причина входа' column")
	}
	if !strings.Contains(out, "MACD кросс") {
		t.Fatal("markdown missing the trade EntryReason text")
	}
}

func TestRenderTradesCSVShowsEntryReason(t *testing.T) {
	trades := []Trade{{Reason: "SL", EntryReason: "пробой; объём 2×"}}
	out := RenderTradesCSV(trades)
	if !strings.Contains(out, "entry_reason") {
		t.Fatal("csv header missing entry_reason")
	}
	if !strings.Contains(out, "пробой; объём 2×") {
		t.Fatal("csv missing EntryReason value")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/domain/backtest/ -run EntryReason -v`
Expected: FAIL — `unknown field EntryReason in struct literal` (compile error).

- [ ] **Step 3: Add the field to model.Signal**

In `internal/service/trading_strategy/scalping/model/signal.go`, add to the `Signal` struct (after `Reason`):

```go
	Reason         string  // exit reason: "TP"/"SL"/"TRAIL"; ignored for entries
	EntryReason    string  // human-readable entry rationale (set on Buy); empty for sells
```

- [ ] **Step 4: Add the field to Trade**

In `internal/domain/backtest/types.go`, add to the `Trade` struct (after `ATR`):

```go
	ATR             float64 // ATR at entry; 0 when n/a
	EntryReason     string  // human-readable entry rationale captured at entry
```

- [ ] **Step 5: Thread it through the portfolio**

In `internal/domain/backtest/portfolio.go`:

Add a field to the `portfolio` struct (after `entryStop`):

```go
	entryStop    float64 // hard stop frozen at entry
	entryReason  string  // human-readable entry rationale captured at entry
```

Change `open`'s signature and body — replace the signature line and add the assignment:

```go
func (p *portfolio) open(price float64, t time.Time, level, target, atr, stop float64, entryReason string) {
```

and after `p.entryStop = stop` add:

```go
	p.entryStop = stop
	p.entryReason = entryReason
```

In `close`, set the trade field and clear it on exit — add `EntryReason: p.entryReason,` to the `Trade{...}` literal (after `ATR: p.entryATR,`) and add `p.entryReason = ""` to the reset block (after `p.entryStop = 0`).

- [ ] **Step 6: Update the engine call site**

In `internal/domain/backtest/engine.go`, change the `open` call (line ~67):

```go
				p.open(c.Close, c.Time, sig.Level, sig.TakeProfit, sig.ATR, sig.StopLoss, sig.EntryReason)
```

- [ ] **Step 7: Add the report column (markdown + csv)**

In `internal/domain/backtest/report.go`, update the trade-journal header and row in `RenderMarkdown`:

Header line — append `Причина входа |`:

```go
	b.WriteString("\n## Журнал сделок\n\n| № | Вход | Цена входа | Выход | Цена выхода | Причина | Баров | PnL | PnL %% | Support | Resist | ATR | Причина входа |\n|---|---|---|---|---|---|---|---|---|---|---|---|---|\n")
```

Row `Fprintf` — append ` %s |` and `t.EntryReason`:

```go
		fmt.Fprintf(&b, "| %d | %s | %.4f | %s | %.4f | %s | %d | %.2f | %.2f%% | %.4f | %.4f | %.4f | %s |\n",
			i+1, t.EntryTime.Format(tsLayout), t.EntryPrice, t.ExitTime.Format(tsLayout),
			t.ExitPrice, t.Reason, t.BarsHeld, t.PnL, t.PnLPct*100,
			t.SupportLevel, t.ResistanceLevel, t.ATR, t.EntryReason)
```

In `RenderTradesCSV`, append the column. Header:

```go
	b.WriteString("idx,entry_time,entry_price,exit_time,exit_price,qty,reason,pnl,pnl_pct,bars_held,support_level,resistance_level,atr,entry_reason\n")
```

Row — wrap EntryReason in quotes (it contains commas/semicolons):

```go
		fmt.Fprintf(&b, "%d,%s,%.6f,%s,%.6f,%d,%s,%.6f,%.6f,%d,%.6f,%.6f,%.6f,%q\n",
			i+1, t.EntryTime.UTC().Format(time.RFC3339), t.EntryPrice,
			t.ExitTime.UTC().Format(time.RFC3339), t.ExitPrice, t.Quantity,
			t.Reason, t.PnL, t.PnLPct, t.BarsHeld,
			t.SupportLevel, t.ResistanceLevel, t.ATR, t.EntryReason)
```

- [ ] **Step 8: Run tests to verify they pass**

Run: `go build ./... && go test ./internal/domain/backtest/ ./internal/service/... -v 2>&1 | tail -30`
Expected: PASS, including the two new EntryReason tests and all existing engine/levels/scalping tests (the new `open` param and struct fields are additive; existing strategies pass empty EntryReason).

- [ ] **Step 9: Commit**

```bash
gofmt -w internal/domain/backtest/ internal/service/trading_strategy/scalping/model/
git add internal/domain/backtest/ internal/service/trading_strategy/scalping/model/signal.go
git commit -m "feat(backtest): carry human-readable EntryReason into trade journal"
```

---

## Task 3: Engine fills TP exits at the target

**Files:**
- Modify: `internal/domain/backtest/engine.go:74-77`
- Test: `internal/domain/backtest/engine_test.go`

- [ ] **Step 1: Write the failing test**

Add to `internal/domain/backtest/engine_test.go`:

```go
func TestEngineFillsTPAtTarget(t *testing.T) {
	// Bar 2 buys at 100; bar 3 has a high of 130 reaching the TP=120, closing at 110.
	// The TP exit must fill at the target (120), not the close (110).
	candles := []Candle{
		{Time: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC), Open: 10, High: 10, Low: 10, Close: 10, Volume: 1},
		{Time: time.Date(2026, 1, 1, 1, 0, 0, 0, time.UTC), Open: 100, High: 100, Low: 100, Close: 100, Volume: 1},
		{Time: time.Date(2026, 1, 1, 2, 0, 0, 0, time.UTC), Open: 105, High: 130, Low: 104, Close: 110, Volume: 1},
	}
	s := scriptedStrategy{lookback: 1, decide: func(md strategy.MarketData) model.Signal {
		if md.Position == nil && md.Price == 100 {
			return model.Signal{Kind: model.SignalBuy, TakeProfit: 120, StopLoss: 90}
		}
		if md.Position != nil {
			return model.Signal{Kind: model.SignalSell, Reason: "TP", TakeProfit: 120}
		}
		return model.Signal{Kind: model.SignalNone}
	}}
	res := Run(s, candles, nil, Config{InitialCash: 100000, Fraction: 1.0, Commission: 0, Lot: 1})
	if len(res.Trades) != 1 {
		t.Fatalf("trades=%d want 1", len(res.Trades))
	}
	if res.Trades[0].ExitPrice != 120 {
		t.Fatalf("TP exit price=%f want 120 (filled at target)", res.Trades[0].ExitPrice)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/domain/backtest/ -run TestEngineFillsTPAtTarget -v`
Expected: FAIL — exit price 110 (close) want 120.

- [ ] **Step 3: Implement the TP fill**

In `internal/domain/backtest/engine.go`, replace the stop-fill block inside the `SignalSell` case:

```go
				exitPrice := c.Close
				// Stop exits fill at the stop level, adjusted for a gap-down open:
				// min(level, open) lands inside the bar and charges real gap risk.
				// TP exits fill at the target, adjusted for a gap-up open: max(target,
				// open) is the limit fill and rewards a gap through the target.
				switch sig.Reason {
				case "SL", "TRAIL":
					exitPrice = min(sig.StopLoss, c.Open)
				case "TP":
					exitPrice = max(sig.TakeProfit, c.Open)
				}
				res.Trades = append(res.Trades, p.close(exitPrice, c.Time, sig.Reason))
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/domain/backtest/ -v 2>&1 | tail -20`
Expected: PASS, including the existing `TestEngineBuysFlatSellsInPosition` (its TP has TakeProfit=0; `max(0, open=110)=110` unchanged).

- [ ] **Step 5: Commit**

```bash
gofmt -w internal/domain/backtest/engine.go internal/domain/backtest/engine_test.go
git add internal/domain/backtest/engine.go internal/domain/backtest/engine_test.go
git commit -m "feat(backtest): fill TP exits at the target with gap-up handling"
```

---

## Task 4: Engine supplies daily highs/lows and today's intraday extent

**Files:**
- Modify: `internal/service/trading_strategy/scalping/strategy/strategy.go:22-33`
- Modify: `internal/domain/backtest/engine.go` (add helpers, set fields in `Run`)
- Test: `internal/domain/backtest/engine_test.go`

- [ ] **Step 1: Add the MarketData fields**

In `internal/service/trading_strategy/scalping/strategy/strategy.go`, add to `MarketData` (after `DailyCloses`):

```go
	DailyCloses []float64
	// DailyHighs / DailyLows are oldest-first highs/lows of the same COMPLETED daily
	// candles as DailyCloses (aligned index-for-index). Empty when no daily data.
	DailyHighs []float64
	DailyLows  []float64
	// TodayHigh / TodayLow are the high/low across all bars of the current MSK
	// calendar day up to and including the current bar (no lookahead). Zero when n/a.
	TodayHigh float64
	TodayLow  float64
	Position  *Position // nil when flat
```

Remove the old `Position *Position` line that was after `DailyCloses` (it moves to the end of the block above).

- [ ] **Step 2: Write the failing test**

Add to `internal/domain/backtest/engine_test.go`:

```go
func TestEngineSuppliesDailyHighsLowsAndTodayExtent(t *testing.T) {
	msk := mskLoc
	// Two MSK days of hourly bars. Day 1: 2026-01-02, Day 2: 2026-01-03.
	mk := func(y, mo, d, h int, o, hi, lo, c float64) Candle {
		return Candle{Time: time.Date(y, time.Month(mo), d, h, 0, 0, 0, msk), Open: o, High: hi, Low: lo, Close: c, Volume: 1}
	}
	candles := []Candle{
		mk(2026, 1, 2, 10, 10, 12, 9, 11),
		mk(2026, 1, 2, 11, 11, 15, 8, 14), // day1 running extent now H=15 L=8
		mk(2026, 1, 3, 10, 14, 16, 13, 15), // new day: extent resets to this bar
	}
	daily := []Candle{
		mk(2026, 1, 1, 0, 100, 110, 90, 105),
		mk(2026, 1, 2, 0, 105, 120, 100, 118),
	}
	var seenHi, seenLo []float64
	var seenDailyHighs [][]float64
	s := scriptedStrategy{lookback: 1, decide: func(md strategy.MarketData) model.Signal {
		seenHi = append(seenHi, md.TodayHigh)
		seenLo = append(seenLo, md.TodayLow)
		seenDailyHighs = append(seenDailyHighs, append([]float64(nil), md.DailyHighs...))
		return model.Signal{Kind: model.SignalNone}
	}}
	Run(s, candles, daily, Config{InitialCash: 1000, Fraction: 1, Commission: 0, Lot: 1})

	// Bar 0 (day1, 10:00): extent = first bar only -> H=12 L=9.
	if seenHi[0] != 12 || seenLo[0] != 9 {
		t.Fatalf("bar0 extent H=%v L=%v want 12/9", seenHi[0], seenLo[0])
	}
	// Bar 1 (day1, 11:00): extent over both day1 bars -> H=15 L=8.
	if seenHi[1] != 15 || seenLo[1] != 8 {
		t.Fatalf("bar1 extent H=%v L=%v want 15/8", seenHi[1], seenLo[1])
	}
	// Bar 2 (day2, 10:00): extent resets to that bar -> H=16 L=13.
	if seenHi[2] != 16 || seenLo[2] != 13 {
		t.Fatalf("bar2 extent H=%v L=%v want 16/13", seenHi[2], seenLo[2])
	}
	// On day1 bars only 2026-01-01 has completed -> 1 daily high (110). On day2 bar,
	// 2026-01-01 and 2026-01-02 completed -> 2 daily highs.
	if len(seenDailyHighs[0]) != 1 || seenDailyHighs[0][0] != 110 {
		t.Fatalf("bar0 daily highs=%v want [110]", seenDailyHighs[0])
	}
	if len(seenDailyHighs[2]) != 2 {
		t.Fatalf("bar2 daily highs len=%d want 2", len(seenDailyHighs[2]))
	}
}
```

- [ ] **Step 3: Run test to verify it fails**

Run: `go test ./internal/domain/backtest/ -run TodayExtent -v`
Expected: FAIL — fields zero / undefined helpers.

- [ ] **Step 4: Add the helpers**

In `internal/domain/backtest/engine.go` (after `visibleDailyCloses`):

```go
// visibleDailyHighsLows returns highs and lows of daily candles whose calendar day
// (in loc) is strictly before t's calendar day — the same completed days as
// visibleDailyCloses, so the three series are index-aligned.
func visibleDailyHighsLows(daily []Candle, t time.Time, loc *time.Location) (highs, lows []float64) {
	bound := startOfDay(t, loc)
	for _, c := range daily {
		if c.Time.Before(bound) {
			highs = append(highs, c.High)
			lows = append(lows, c.Low)
		}
	}
	return highs, lows
}

// todayExtent returns the high and low across all bars sharing candles[i]'s MSK
// calendar day, scanning back from i only (no lookahead). Returns (0,0) when i is
// out of range.
func todayExtent(candles []Candle, i int, loc *time.Location) (high, low float64) {
	if i < 0 || i >= len(candles) {
		return 0, 0
	}
	day := startOfDay(candles[i].Time, loc)
	high, low = candles[i].High, candles[i].Low
	for j := i - 1; j >= 0; j-- {
		if startOfDay(candles[j].Time, loc).Before(day) {
			break
		}
		if candles[j].High > high {
			high = candles[j].High
		}
		if candles[j].Low < low {
			low = candles[j].Low
		}
	}
	return high, low
}
```

- [ ] **Step 5: Set the fields in Run**

In `internal/domain/backtest/engine.go`, just after the existing `md.DailyCloses = ...` line in `Run`:

```go
		md.DailyCloses = visibleDailyCloses(dailyCandles, candles[i].Time, mskLoc)
		md.DailyHighs, md.DailyLows = visibleDailyHighsLows(dailyCandles, candles[i].Time, mskLoc)
		md.TodayHigh, md.TodayLow = todayExtent(candles, i, mskLoc)
```

- [ ] **Step 6: Run tests to verify they pass**

Run: `go build ./... && go test ./internal/domain/backtest/ -v 2>&1 | tail -20`
Expected: PASS, including existing engine/levels/scalping tests.

- [ ] **Step 7: Commit**

```bash
gofmt -w internal/domain/backtest/engine.go internal/domain/backtest/engine_test.go internal/service/trading_strategy/scalping/strategy/strategy.go
git add internal/domain/backtest/engine.go internal/domain/backtest/engine_test.go internal/service/trading_strategy/scalping/strategy/strategy.go
git commit -m "feat(backtest): supply daily highs/lows and intraday day extent to strategies"
```

---

## Task 5: Momentum core strategy (pure Decide)

**Files:**
- Create: `internal/service/trading_strategy/momentum/strategy/core/core.go`
- Test: `internal/service/trading_strategy/momentum/strategy/core/core_test.go`

This task builds the whole pure strategy with internal TDD steps. The `Strategy` implements
`scalping/strategy.Strategy` and returns `scalping/model.Signal`, exactly like `levels/strategy/core`.

- [ ] **Step 1: Write the Params + skeleton (compiles, no logic yet)**

Create `core.go`:

```go
// Package core implements a long-only hourly trend-momentum strategy: enter when
// price is above a long EMA (uptrend), MACD just crossed up (optionally below
// zero), volume is above its recent average, and the day still has room left
// within its typical daily-ATR range. Exits on a frozen structural ATR stop, a
// fixed reward-multiple take-profit, or an optional chandelier trail. The decision
// logic is pure and ticker-agnostic; per-share packages supply ticker + Params.
package core

import (
	"fmt"

	"tinvest/internal/domain/ema"
	"tinvest/internal/service/trading_strategy/scalping/model"
	"tinvest/internal/service/trading_strategy/scalping/strategy"
	"tinvest/pkg/indicators"
)

// Params holds every tunable. All fields are int or float64 (flags as int 0/1)
// so reflection grid calibration can sweep them.
type Params struct {
	EMAPeriod         int     // long EMA trend filter (hourly)
	MACDFast          int     // MACD fast EMA period
	MACDSlow          int     // MACD slow EMA period
	MACDSignal        int     // MACD signal EMA period
	MACDBelowZeroOnly int     // 1 = require macd<0 at the cross; 0 = any bullish cross
	VolLookback       int     // SMA window for the volume baseline
	VolMultiplier     float64 // last volume must exceed VolMultiplier*SMA(volume)
	DailyATRPeriod    int     // ATR period over completed daily candles
	MaxDailyATRUsed   float64 // block entry if today's range >= MaxDailyATRUsed*dailyATR
	ATRPeriod         int     // hourly ATR period (stops, anti-churn)
	SwingLowWindow    int     // bars scanned for the structural low anchoring the stop
	SLMult            float64 // stop = swingLow - SLMult*ATR
	TakeProfitRR      float64 // TP = entry + TakeProfitRR*(entry-stop)
	MinRR             float64 // reject entry if (TP-price) < MinRR*risk; <=0 disables
	MinATRFrac        float64 // reject entry if ATR < MinATRFrac*price; <=0 disables
	UseTrail          int     // 1 = trail instead of fixed TP
	TrailMult         float64 // chandelier = recentHigh(ChandelierWindow) - TrailMult*ATR
	ChandelierWindow  int     // window for the chandelier high
	TrailArmATR       float64 // trail arms after MaxFavorable >= entry + TrailArmATR*EntryATR
	CooldownBars      int     // reserved; not yet enforced
	DailyTrendPeriod  int     // reserved; not yet enforced
}

// Strategy trades a single instrument with the momentum rules. Ticker-agnostic.
type Strategy struct {
	ticker string
	p      Params
}

// NewWithParams returns the momentum strategy for a ticker with explicit params.
func NewWithParams(ticker string, p Params) *Strategy { return &Strategy{ticker: ticker, p: p} }

func (s *Strategy) Ticker() string { return s.ticker }

// Lookback sizes the candle window to feed the hungriest consumer.
func (s *Strategy) Lookback() int {
	m := s.p.EMAPeriod
	for _, c := range []int{
		s.p.MACDSlow + s.p.MACDSignal,
		s.p.VolLookback + 1,
		s.p.ATRPeriod + 1,
		s.p.SwingLowWindow,
		s.p.ChandelierWindow,
	} {
		if c > m {
			m = c
		}
	}
	return m + 5
}
```

Run: `go build ./internal/service/trading_strategy/momentum/...`
Expected: build fails only because `fmt`/`ema`/`indicators` imports are unused yet — that is acceptable mid-task; the next steps use them. If you prefer a green build here, defer the imports to Step 3. (Reviewer: do not commit until Step 8.)

- [ ] **Step 2: Write the entry tests (table-driven)**

Create `core_test.go`. The helper builds a `MarketData` that passes every gate, and each
case knocks out exactly one condition to prove it blocks. A rising series satisfies EMA + MACD.

```go
package core

import (
	"strings"
	"testing"

	"tinvest/internal/service/trading_strategy/scalping/model"
	"tinvest/internal/service/trading_strategy/scalping/strategy"
)

func defaultParams() Params {
	return Params{
		EMAPeriod: 200, MACDFast: 12, MACDSlow: 26, MACDSignal: 9, MACDBelowZeroOnly: 0,
		VolLookback: 20, VolMultiplier: 1.2, DailyATRPeriod: 14, MaxDailyATRUsed: 0.6,
		ATRPeriod: 14, SwingLowWindow: 10, SLMult: 0.5, TakeProfitRR: 2.0, MinRR: 1.5,
		MinATRFrac: 0.0, UseTrail: 0, TrailMult: 2.5, ChandelierWindow: 20, TrailArmATR: 1.0,
	}
}

// buildEntryMD constructs a snapshot engineered to pass all entry gates:
//   - 260 rising-then-dipping closes so close>EMA200 and a fresh bullish MACD cross,
//   - last bar volume well above the VolLookback average,
//   - daily series giving a positive ATR with plenty of remaining room.
func buildEntryMD() strategy.MarketData {
	n := 260
	closes := make([]float64, n)
	highs := make([]float64, n)
	lows := make([]float64, n)
	vols := make([]int64, n)
	for i := 0; i < n; i++ {
		base := 100.0 + float64(i)*0.5 // strong uptrend keeps close>EMA200
		closes[i] = base
		highs[i] = base + 0.3
		lows[i] = base - 0.3
		vols[i] = 1000
	}
	// Engineer a fresh MACD bullish cross at the last bar: dip for a few bars then pop.
	closes[n-4], closes[n-3], closes[n-2] = closes[n-5]-1, closes[n-5]-2, closes[n-5]-2.5
	highs[n-4], highs[n-3], highs[n-2] = closes[n-4]+0.3, closes[n-3]+0.3, closes[n-2]+0.3
	lows[n-4], lows[n-3], lows[n-2] = closes[n-4]-0.3, closes[n-3]-0.3, closes[n-2]-0.3
	closes[n-1] = closes[n-5] + 2 // strong pop -> MACD crosses up
	highs[n-1] = closes[n-1] + 0.5
	lows[n-1] = closes[n-1] - 0.5
	vols[n-1] = 5000 // above 1.2x average

	dailyH := []float64{105, 106, 107, 108, 109, 110, 111, 112, 113, 114, 115, 116, 117, 118, 119, 120}
	dailyL := []float64{100, 101, 102, 103, 104, 105, 106, 107, 108, 109, 110, 111, 112, 113, 114, 115}
	dailyC := []float64{104, 105, 106, 107, 108, 109, 110, 111, 112, 113, 114, 115, 116, 117, 118, 119}

	return strategy.MarketData{
		Price: closes[n-1], Highs: highs, Lows: lows, Closes: closes, Volumes: vols,
		DailyHighs: dailyH, DailyLows: dailyL, DailyCloses: dailyC,
		TodayHigh: closes[n-1] + 0.5, TodayLow: closes[n-1] - 0.5, // tiny consumed range -> room OK
	}
}

func TestEntryFiresWhenAllGatesPass(t *testing.T) {
	s := NewWithParams("TEST", defaultParams())
	sig := s.Decide(buildEntryMD())
	if sig.Kind != model.SignalBuy {
		t.Fatalf("kind=%v want Buy", sig.Kind)
	}
	if sig.StopLoss <= 0 || sig.TakeProfit <= sig.Price {
		t.Fatalf("SL=%f TP=%f price=%f want SL>0 and TP>price", sig.StopLoss, sig.TakeProfit, sig.Price)
	}
	if sig.ATR <= 0 {
		t.Fatalf("ATR=%f want >0", sig.ATR)
	}
	if sig.Ticker != "TEST" {
		t.Fatalf("ticker=%q want TEST", sig.Ticker)
	}
	if !strings.Contains(sig.EntryReason, "MACD") || !strings.Contains(sig.EntryReason, "ATR") {
		t.Fatalf("EntryReason missing detail: %q", sig.EntryReason)
	}
}

func TestEntryBlockedByTrendFilter(t *testing.T) {
	md := buildEntryMD()
	for i := range md.Closes { // flat-low series -> price below EMA200
		md.Closes[i] = 50
		md.Highs[i] = 50.3
		md.Lows[i] = 49.7
	}
	md.Price = 50
	s := NewWithParams("TEST", defaultParams())
	if sig := s.Decide(md); sig.Kind == model.SignalBuy {
		t.Fatal("entry should be blocked when close < EMA200")
	}
}

func TestEntryBlockedByVolume(t *testing.T) {
	md := buildEntryMD()
	md.Volumes[len(md.Volumes)-1] = 100 // below average
	s := NewWithParams("TEST", defaultParams())
	if sig := s.Decide(md); sig.Kind == model.SignalBuy {
		t.Fatal("entry should be blocked on weak volume")
	}
}

func TestEntryBlockedByDailyATRRoom(t *testing.T) {
	md := buildEntryMD()
	// Make today's consumed range exceed MaxDailyATRUsed*dailyATR.
	md.TodayHigh = md.Price + 50
	md.TodayLow = md.Price - 50
	s := NewWithParams("TEST", defaultParams())
	if sig := s.Decide(md); sig.Kind == model.SignalBuy {
		t.Fatal("entry should be blocked when daily ATR room is used up")
	}
}

func TestEntryBlockedByMACDBelowZeroFlag(t *testing.T) {
	md := buildEntryMD()
	p := defaultParams()
	p.MACDBelowZeroOnly = 1 // a strong uptrend has MACD>0, so the cross is above zero -> blocked
	s := NewWithParams("TEST", p)
	if sig := s.Decide(md); sig.Kind == model.SignalBuy {
		t.Fatal("entry should be blocked when MACDBelowZeroOnly=1 and macd>0")
	}
}
```

- [ ] **Step 3: Run entry tests to verify they fail**

Run: `go test ./internal/service/trading_strategy/momentum/strategy/core/ -run TestEntry -v`
Expected: FAIL — `Decide` undefined.

- [ ] **Step 4: Implement Decide + the pure entry core**

Append to `core.go`:

```go
// decideInput carries already-computed indicator values into the pure core.
type decideInput struct {
	price       float64
	atr         float64
	dailyATR    float64
	emaTrend    float64
	macdNow     float64
	crossUp     bool
	volumeOK    bool
	todayRange  float64
	barHigh     float64
	barLow      float64
	recentLow   float64
	recentHigh  float64
	pos         *strategy.Position
}

// Decide computes every indicator from md, packs them, and delegates to the pure core.
func (s *Strategy) Decide(md strategy.MarketData) model.Signal {
	atr := indicators.ATR(md.Highs, md.Lows, md.Closes, s.p.ATRPeriod)
	dailyATR := indicators.ATR(md.DailyHighs, md.DailyLows, md.DailyCloses, s.p.DailyATRPeriod)

	emaTrend := 0.0
	if e := ema.Compute(md.Closes, s.p.EMAPeriod); len(e) > 0 {
		emaTrend = e[len(e)-1]
	}

	macdNow, crossUp := 0.0, false
	if m, sg := indicators.MACD(md.Closes, s.p.MACDFast, s.p.MACDSlow, s.p.MACDSignal); len(m) >= 2 {
		prevDiff := m[len(m)-2] - sg[len(sg)-2]
		currDiff := m[len(m)-1] - sg[len(sg)-1]
		macdNow = m[len(m)-1]
		crossUp = prevDiff <= 0 && currDiff > 0
	}

	var barHigh, barLow float64
	if n := len(md.Highs); n > 0 {
		barHigh = md.Highs[n-1]
	}
	if n := len(md.Lows); n > 0 {
		barLow = md.Lows[n-1]
	}

	in := decideInput{
		price:      md.Price,
		atr:        atr,
		dailyATR:   dailyATR,
		emaTrend:   emaTrend,
		macdNow:    macdNow,
		crossUp:    crossUp,
		volumeOK:   indicators.VolumeConfirmed(md.Volumes, s.p.VolLookback, s.p.VolMultiplier),
		todayRange: md.TodayHigh - md.TodayLow,
		barHigh:    barHigh,
		barLow:     barLow,
		recentLow:  recentLow(md.Lows, s.p.SwingLowWindow),
		recentHigh: recentHigh(md.Highs, s.p.ChandelierWindow),
		pos:        md.Position,
	}

	sig := s.decide(in)
	sig.Ticker = s.ticker
	return sig
}

// decide is the pure decision core over already-computed indicator values.
func (s *Strategy) decide(in decideInput) model.Signal {
	sig := model.Signal{Price: in.price}

	if in.pos != nil {
		return s.manage(in, sig)
	}

	// Entry gates (all must pass).
	if !(in.emaTrend > 0 && in.price > in.emaTrend) {
		return sig // not an uptrend
	}
	if !in.crossUp {
		return sig
	}
	if s.p.MACDBelowZeroOnly == 1 && in.macdNow >= 0 {
		return sig
	}
	if !in.volumeOK {
		return sig
	}
	// Daily-ATR room: pass when daily data is absent (dailyATR<=0), else require room.
	if in.dailyATR > 0 && in.todayRange >= s.p.MaxDailyATRUsed*in.dailyATR {
		return sig
	}
	if s.p.MinATRFrac > 0 && in.atr < s.p.MinATRFrac*in.price {
		return sig
	}

	stop := in.recentLow - s.p.SLMult*in.atr
	risk := in.price - stop
	if risk <= 0 {
		return sig
	}
	target := in.price + s.p.TakeProfitRR*risk
	if s.p.MinRR > 0 && (target-in.price) < s.p.MinRR*risk {
		return sig
	}

	sig.Kind = model.SignalBuy
	sig.StopLoss = stop
	sig.TakeProfit = target
	sig.ATR = in.atr
	sig.EntryReason = s.entryReason(in, stop, target, risk)
	return sig
}

// entryReason renders the human-readable rationale shown in the trade journal.
func (s *Strategy) entryReason(in decideInput, stop, target, risk float64) string {
	zero := "над нулём"
	if in.macdNow < 0 {
		zero = "под нулём"
	}
	roomPct := 0.0
	if in.dailyATR > 0 {
		roomPct = (1 - in.todayRange/in.dailyATR) * 100
	}
	return fmt.Sprintf(
		"Тренд↑ (close %.4f > EMA%d %.4f); MACD бычий кросс %s (%.4f); объём > %.2g×ср(%d); дневной ATR-запас %.0f%% (прошло %.4f из %.4f); ATR(ч)=%.4f, ATR(д)=%.4f; SL=%.4f (-%.4f); TP=%.4f (+%.4f, %.2gR)",
		in.price, s.p.EMAPeriod, in.emaTrend, zero, in.macdNow,
		s.p.VolMultiplier, s.p.VolLookback,
		roomPct, in.todayRange, in.dailyATR,
		in.atr, in.dailyATR,
		stop, risk, target, target-in.price, s.p.TakeProfitRR,
	)
}
```

- [ ] **Step 5: Add the management (exit) helpers and small utilities**

Append to `core.go`:

```go
// manage handles an open long: frozen hard stop, fixed take-profit (reconstructed
// from the frozen entry stop), or an optional armed chandelier trail.
func (s *Strategy) manage(in decideInput, sig model.Signal) model.Signal {
	entry := in.pos.PurchasePrice
	hardSL := in.pos.StopLoss
	// TP is reconstructed from the frozen entry stop (Position carries no target):
	// risk and stop are both fixed at entry, so this is deterministic.
	risk := entry - hardSL
	tp := entry + s.p.TakeProfitRR*risk

	sig.StopLoss = hardSL
	sig.TakeProfit = tp

	switch {
	case in.barLow <= hardSL:
		sig.Kind, sig.Reason = model.SignalSell, "SL"
	case s.p.UseTrail == 0 && in.barHigh >= tp:
		sig.Kind, sig.Reason = model.SignalSell, "TP"
	case s.p.UseTrail == 1:
		chandelier := in.recentHigh - s.p.TrailMult*in.atr
		armed := s.p.TrailArmATR <= 0 || in.pos.MaxFavorablePrice >= entry+s.p.TrailArmATR*in.pos.EntryATR
		if armed && in.barLow <= chandelier {
			sig.Kind, sig.Reason, sig.StopLoss = model.SignalSell, "TRAIL", chandelier
		}
	}
	return sig
}

// recentLow returns the lowest low over the last window bars (all if fewer);
// a non-positive window is clamped to the last bar.
func recentLow(lows []float64, window int) float64 {
	n := len(lows)
	if n == 0 {
		return 0
	}
	start := n - window
	if start < 0 {
		start = 0
	}
	if start > n-1 {
		start = n - 1
	}
	l := lows[start]
	for i := start + 1; i < n; i++ {
		if lows[i] < l {
			l = lows[i]
		}
	}
	return l
}

// recentHigh returns the highest high over the last window bars (all if fewer);
// a non-positive window is clamped to the last bar.
func recentHigh(highs []float64, window int) float64 {
	n := len(highs)
	if n == 0 {
		return 0
	}
	start := n - window
	if start < 0 {
		start = 0
	}
	if start > n-1 {
		start = n - 1
	}
	h := highs[start]
	for i := start + 1; i < n; i++ {
		if highs[i] > h {
			h = highs[i]
		}
	}
	return h
}
```

- [ ] **Step 6: Write the exit tests**

Add to `core_test.go`:

```go
func inPositionMD(barLow, barHigh, recentHigh float64, pos *strategy.Position) strategy.MarketData {
	// 30 flat bars so ATR is well-defined; override the last bar's high/low.
	n := 30
	highs := make([]float64, n)
	lows := make([]float64, n)
	closes := make([]float64, n)
	vols := make([]int64, n)
	for i := 0; i < n; i++ {
		closes[i], highs[i], lows[i], vols[i] = 100, recentHigh, 99, 1000
	}
	highs[n-1], lows[n-1] = barHigh, barLow
	return strategy.MarketData{
		Price: 100, Highs: highs, Lows: lows, Closes: closes, Volumes: vols, Position: pos,
	}
}

func TestExitStopLoss(t *testing.T) {
	pos := &strategy.Position{PurchasePrice: 100, StopLoss: 95, EntryATR: 1, MaxFavorablePrice: 100}
	s := NewWithParams("TEST", defaultParams())
	sig := s.Decide(inPositionMD(94, 101, 100, pos)) // barLow 94 <= SL 95
	if sig.Kind != model.SignalSell || sig.Reason != "SL" {
		t.Fatalf("kind=%v reason=%q want Sell/SL", sig.Kind, sig.Reason)
	}
}

func TestExitTakeProfit(t *testing.T) {
	// TP = entry + RR*(entry-stop) = 100 + 2*5 = 110. barHigh 111 >= 110.
	pos := &strategy.Position{PurchasePrice: 100, StopLoss: 95, EntryATR: 1, MaxFavorablePrice: 100}
	s := NewWithParams("TEST", defaultParams())
	sig := s.Decide(inPositionMD(100, 111, 100, pos))
	if sig.Kind != model.SignalSell || sig.Reason != "TP" {
		t.Fatalf("kind=%v reason=%q want Sell/TP", sig.Kind, sig.Reason)
	}
	if sig.TakeProfit != 110 {
		t.Fatalf("TP=%f want 110", sig.TakeProfit)
	}
}

func TestExitStopLossWinsOverTP(t *testing.T) {
	// Both SL (95) and TP (110) touched in the same bar: SL has priority.
	pos := &strategy.Position{PurchasePrice: 100, StopLoss: 95, EntryATR: 1, MaxFavorablePrice: 100}
	s := NewWithParams("TEST", defaultParams())
	sig := s.Decide(inPositionMD(94, 111, 100, pos))
	if sig.Reason != "SL" {
		t.Fatalf("reason=%q want SL (priority over TP)", sig.Reason)
	}
}

func TestExitHoldsWhenNeitherHit(t *testing.T) {
	pos := &strategy.Position{PurchasePrice: 100, StopLoss: 95, EntryATR: 1, MaxFavorablePrice: 100}
	s := NewWithParams("TEST", defaultParams())
	sig := s.Decide(inPositionMD(100, 109, 100, pos)) // below TP 110, above SL 95
	if sig.Kind != model.SignalNone {
		t.Fatalf("kind=%v want None (hold)", sig.Kind)
	}
}

func TestExitTrailWhenEnabled(t *testing.T) {
	p := defaultParams()
	p.UseTrail = 1
	p.TrailArmATR = 0 // arm immediately
	pos := &strategy.Position{PurchasePrice: 100, StopLoss: 95, EntryATR: 1, MaxFavorablePrice: 120}
	s := NewWithParams("TEST", p)
	// recentHigh 120, ATR≈1, TrailMult 2.5 -> chandelier≈117.5; barLow 117 <= chandelier.
	sig := s.Decide(inPositionMD(117, 121, 120, pos))
	if sig.Kind != model.SignalSell || sig.Reason != "TRAIL" {
		t.Fatalf("kind=%v reason=%q want Sell/TRAIL", sig.Kind, sig.Reason)
	}
}
```

- [ ] **Step 7: Run all core tests to verify they pass**

Run: `go test ./internal/service/trading_strategy/momentum/strategy/core/ -v`
Expected: PASS (entry + exit). If `TestExitTrailWhenEnabled` is borderline on ATR, adjust `barLow` in the test, not the implementation.

- [ ] **Step 8: Commit**

```bash
gofmt -w internal/service/trading_strategy/momentum/strategy/core/
go build ./...
git add internal/service/trading_strategy/momentum/strategy/core/
git commit -m "feat(momentum): pure trend-momentum core with entry reason and SL/TP/trail exits"
```

---

## Task 6: RUAL per-ticker package, registry, and generic fallback

**Files:**
- Create: `internal/service/trading_strategy/momentum/strategy/rusal/rusal.go`
- Create: `internal/service/backtest/momentum_registry.go`
- Test: `internal/service/backtest/momentum_registry_test.go`

- [ ] **Step 1: Create the RUAL package**

Create `rusal.go`:

```go
// Package rusal supplies the ticker and calibrated momentum Params for RUAL.
// Starting values mirror the generic defaults; calibrate with -calibrate and then
// hardcode the winning combination here (MACD periods are tuned per ticker).
package rusal

import "tinvest/internal/service/trading_strategy/momentum/strategy/core"

// Ticker is the instrument this package configures.
const Ticker = "RUAL"

// DefaultParams returns RUAL's starting momentum parameters (uncalibrated).
func DefaultParams() core.Params {
	return core.Params{
		EMAPeriod: 200, MACDFast: 12, MACDSlow: 26, MACDSignal: 9, MACDBelowZeroOnly: 1,
		VolLookback: 20, VolMultiplier: 1.2, DailyATRPeriod: 14, MaxDailyATRUsed: 0.6,
		ATRPeriod: 14, SwingLowWindow: 10, SLMult: 0.5, TakeProfitRR: 2.0, MinRR: 1.5,
		MinATRFrac: 0.003, UseTrail: 0, TrailMult: 2.5, ChandelierWindow: 20, TrailArmATR: 1.0,
		CooldownBars: 0, DailyTrendPeriod: 0,
	}
}
```

- [ ] **Step 2: Create the registry**

Create `momentum_registry.go`:

```go
package backtest

import (
	"encoding/json"
	"fmt"

	"tinvest/internal/service/trading_strategy/momentum/strategy/core"
	momentumrusal "tinvest/internal/service/trading_strategy/momentum/strategy/rusal"
	"tinvest/internal/service/trading_strategy/scalping/strategy"
)

// momentumBindingFor builds a Binding for a ticker whose defaults come from defaults().
// All momentum tickers share the core engine; only ticker + defaults differ.
func momentumBindingFor(ticker string, defaults func() core.Params) Binding {
	return Binding{
		DefaultParams: func() any { return defaults() },
		Build: func(params any) strategy.Strategy {
			return core.NewWithParams(ticker, params.(core.Params))
		},
		ParseParams: func(raw []byte) (any, error) {
			p := defaults() // start from defaults so partial JSON overrides
			if err := json.Unmarshal(raw, &p); err != nil {
				return nil, fmt.Errorf("backtest: parse params: %w", err)
			}
			return p, nil
		},
	}
}

var momentumRegistry = map[string]Binding{
	momentumrusal.Ticker: momentumBindingFor(momentumrusal.Ticker, momentumrusal.DefaultParams),
}

// genericMomentumDefaults are neutral baseline params for tickers without a dedicated
// momentum config. Intentionally independent of rusal.DefaultParams so calibrating RUAL
// never drifts the generic baseline — mirroring genericLevelsDefaults.
func genericMomentumDefaults() core.Params {
	return core.Params{
		EMAPeriod: 200, MACDFast: 12, MACDSlow: 26, MACDSignal: 9, MACDBelowZeroOnly: 1,
		VolLookback: 20, VolMultiplier: 1.2, DailyATRPeriod: 14, MaxDailyATRUsed: 0.6,
		ATRPeriod: 14, SwingLowWindow: 10, SLMult: 0.5, TakeProfitRR: 2.0, MinRR: 1.5,
		MinATRFrac: 0.003, UseTrail: 0, TrailMult: 2.5, ChandelierWindow: 20, TrailArmATR: 1.0,
		CooldownBars: 0, DailyTrendPeriod: 0,
	}
}

// MomentumLookupOrGeneric returns the registered momentum binding for a ticker, or a
// generic binding bound to that ticker (with genericMomentumDefaults) when none is
// registered. This lets the backtest command validate the momentum strategy on any
// ticker without a dedicated package.
func MomentumLookupOrGeneric(ticker string) Binding {
	if b, ok := momentumRegistry[ticker]; ok {
		return b
	}
	return momentumBindingFor(ticker, genericMomentumDefaults)
}
```

- [ ] **Step 3: Write the registry test**

Create `momentum_registry_test.go`:

```go
package backtest

import (
	"testing"

	"tinvest/internal/service/trading_strategy/momentum/strategy/core"
)

func TestMomentumLookupRegisteredRUAL(t *testing.T) {
	b := MomentumLookupOrGeneric("RUAL")
	p, ok := b.DefaultParams().(core.Params)
	if !ok {
		t.Fatalf("DefaultParams type = %T want core.Params", b.DefaultParams())
	}
	if p.MACDBelowZeroOnly != 1 {
		t.Fatalf("RUAL MACDBelowZeroOnly=%d want 1", p.MACDBelowZeroOnly)
	}
	if s := b.Build(p); s.Ticker() != "RUAL" {
		t.Fatalf("ticker=%q want RUAL", s.Ticker())
	}
}

func TestMomentumLookupGenericFallback(t *testing.T) {
	b := MomentumLookupOrGeneric("UNKNOWN")
	if s := b.Build(b.DefaultParams()); s.Ticker() != "UNKNOWN" {
		t.Fatalf("ticker=%q want UNKNOWN", s.Ticker())
	}
}

func TestMomentumParseParamsPartialOverride(t *testing.T) {
	b := MomentumLookupOrGeneric("RUAL")
	got, err := b.ParseParams([]byte(`{"TakeProfitRR": 3.0}`))
	if err != nil {
		t.Fatalf("ParseParams: %v", err)
	}
	p := got.(core.Params)
	if p.TakeProfitRR != 3.0 {
		t.Fatalf("TakeProfitRR=%f want 3.0 (override)", p.TakeProfitRR)
	}
	if p.MACDSlow != 26 {
		t.Fatalf("MACDSlow=%d want 26 (default kept)", p.MACDSlow)
	}
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/service/backtest/ -run Momentum -v`
Expected: PASS (3).

- [ ] **Step 5: Commit**

```bash
gofmt -w internal/service/trading_strategy/momentum/strategy/rusal/ internal/service/backtest/momentum_registry.go internal/service/backtest/momentum_registry_test.go
go build ./...
git add internal/service/trading_strategy/momentum/strategy/rusal/ internal/service/backtest/momentum_registry.go internal/service/backtest/momentum_registry_test.go
git commit -m "feat(momentum): RUAL params, registry binding, and generic fallback"
```

---

## Task 7: Wire `-strategy momentum` into cmd/backtest

**Files:**
- Modify: `cmd/backtest/main.go:35` (flag help) and `:103-110` (strategy switch)

- [ ] **Step 1: Update the strategy switch**

In `cmd/backtest/main.go`, add a `momentum` case to the `switch strategyName` block:

```go
	switch strategyName {
	case "levels":
		binding = svc.LevelsLookupOrGeneric(ticker)
	case "momentum":
		binding = svc.MomentumLookupOrGeneric(ticker)
	case "scalping":
		binding = svc.LookupOrGeneric(ticker)
	default:
		return fmt.Errorf("unknown strategy %q (want scalping|levels|momentum)", strategyName)
	}
```

- [ ] **Step 2: Update the flag help text**

Change the `strategyName` flag default help to list momentum:

```go
		strategyName = flag.String("strategy", "scalping", "strategy engine: scalping|levels|momentum")
```

- [ ] **Step 3: Build and run a smoke backtest**

Run:
```bash
go build ./... && go vet ./...
go run ./cmd/backtest -ticker RUAL -strategy momentum -interval Hour1 -months 12
```
Expected: prints `report: reports/RUAL_momentum_Hour1_<stamp>.md (trades=N, net=..., PF=...)`. A low/zero trade count is acceptable (RUAL downtrend); the point is that it runs end-to-end and the report's trade journal shows the "Причина входа" column with SL/TP/ATR text. (Requires `T_BANK` token in `env/local.env`; if absent the command errors on auth — that is environment, not a code defect.)

- [ ] **Step 4: Commit**

```bash
gofmt -w cmd/backtest/main.go
git add cmd/backtest/main.go
git commit -m "feat(backtest): add momentum strategy to the backtest command"
```

---

## Task 8: Calibration grid + final verification

**Files:**
- Create: `data/params/rusal/momentum_grid.json`

- [ ] **Step 1: Create the calibration grid**

Create `data/params/rusal/momentum_grid.json` (sweep the per-ticker MACD periods + the
risk knobs the user will tune first; keep it modest so the cartesian product stays small):

```json
{
  "MACDFast": [8, 12],
  "MACDSlow": [21, 26],
  "MACDSignal": [9],
  "MACDBelowZeroOnly": [0, 1],
  "SLMult": [0.5, 1.0],
  "TakeProfitRR": [1.5, 2.0, 3.0],
  "VolMultiplier": [1.0, 1.2, 1.5],
  "MaxDailyATRUsed": [0.5, 0.6, 0.7]
}
```

- [ ] **Step 2: Smoke-run the calibration (optional, needs token)**

Run:
```bash
go run ./cmd/backtest -ticker RUAL -strategy momentum -calibrate data/params/rusal/momentum_grid.json
```
Expected: writes `..._calibration.md` and `..._best.md`. (Skip if no token; the grid is still validated for JSON shape by the run command's `Unmarshal`.)

- [ ] **Step 3: Full test + build sweep**

Run:
```bash
gofmt -l . ; go build ./... && go vet ./... && go test ./... 2>&1 | tail -30
```
Expected: no `gofmt` diffs, build OK, vet clean, all packages PASS.

- [ ] **Step 4: Sync the strategy explainer if anything drifted**

Open `docs/momentum/strategy.md` and confirm the parameter table, entry/exit rules, and the
example reason string match the implemented `core.Params` and `entryReason` output. Fix any
drift (e.g. wording of the reason string) so the doc stays the human description of the code.

- [ ] **Step 5: Commit**

```bash
git add data/params/rusal/momentum_grid.json docs/momentum/strategy.md
git commit -m "feat(momentum): calibration grid for RUAL and explainer sync"
```

---

## Self-Review Notes (author)

- **Spec coverage:** §1 architecture → Tasks 5–7; §2 entry gates → Task 5 (Decide); §3 MACD indicator → Task 1; §4 SL/TP/trail → Task 5 (manage) + Task 3 (TP fill); §5 reserved ideas (CooldownBars/DailyTrendPeriod) → Task 5 Params (fields present, unused, documented); §6 EntryReason + report → Task 2 + Task 5; §7 engine changes → Tasks 2–4; §8 Params list → Task 5; §9 RUAL + registry → Task 6, cmd → Task 7, grid → Task 8; §10 tests → each task TDD.
- **Type consistency:** `core.Params`, `core.NewWithParams`, `Binding{DefaultParams,Build,ParseParams}`, `model.Signal.EntryReason`, `Trade.EntryReason`, `portfolio.open(...,entryReason)`, `MarketData.{DailyHighs,DailyLows,TodayHigh,TodayLow}`, helpers `visibleDailyHighsLows`/`todayExtent`, indicator `MACD`/`ema` — names used consistently across tasks.
- **Reserved params:** `CooldownBars`/`DailyTrendPeriod` are intentionally inert (spec §5 future toggles); documented in both code comments and the explainer, not dead-code accidents.
- **Layering:** `pkg/indicators/macd.go` uses a local `ema`, not `internal/domain/ema`, to keep `pkg` free of `internal` imports. The strategy `core` (in `internal`) does use `internal/domain/ema` for the EMA200 trend filter.
