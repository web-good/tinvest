# Day-Low Consolidation Breakout (`daylow`) Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add a long-only 5-minute "consolidation breakout at the prior-day low" backtest strategy (`daylow`) with an optional hourly trend filter and a MOEX active-hours entry window, fully calibratable through the existing backtest engine.

**Architecture:** A new pure `Strategy` (`daylow/strategy/core`) plugs into the existing `internal/domain/backtest` engine exactly like `scalping`/`reversion`: per bar the engine hands `Decide(md)` a candle window plus completed daily and HTF series; the strategy detects the setup statelessly and emits `SignalBuy` with a frozen stop, or manages the open position with SL/TP/EOD exits. Three small platform additions make it work: a `Minutes5` interval, a configurable HTF interval on `Config`, and an `Opens` series on `MarketData`.

**Tech Stack:** Go 1.25, existing `pkg/indicators` (ATR) and `internal/domain/ema`, table-driven tests (`go test -race`), mage CI.

## Global Constraints

- Go 1.25; run quality gate with `./bin/mage ci` (lint + `go test -race ./...` + mock-drift). Build check: `go build ./internal/... ./pkg/... ./cmd/...` (never `go build ./...` — `magefiles` has no `main`).
- Strategy `Decide` MUST be pure (no I/O), a function of `md` only, mirroring `reversion/strategy/core`.
- All `Params` fields are `int`/`float64` (flags as `int` 0/1) so reflection grid calibration can sweep them.
- Exit reason codes drive engine fill pricing: stop-style reasons (`model.IsStopReason`) fill `min(StopLoss, open)`; `"TP"` fills `max(TakeProfit, open)`; any other reason fills at close. `daylow` uses `"SL"` (already a stop reason), `"TP"`, and `"EOD"` (close fill) — no change to `IsStopReason` needed.
- MSK timezone anchor is `Europe/Moscow` with UTC fallback (same pattern as `engine.go`/`reversion core`).
- Comments/log/report strings in Russian where user-facing (matches repo convention).

---

### Task 1: Add `Minutes5` interval

**Files:**
- Modify: `internal/enum/enum.go`
- Modify: `cmd/backtest/main.go:86-103` (`parseInterval`)
- Test: `internal/enum/enum_test.go` (create if absent)

**Interfaces:**
- Produces: `enum.Minutes5 Interval = 2`; `parseInterval("Minutes5") -> enum.Minutes5, nil`.

- [ ] **Step 1: Write the failing test**

Create/append `internal/enum/enum_test.go`:

```go
package enum

import (
	"testing"
	"time"
)

func TestMinutes5(t *testing.T) {
	if Minutes5 != 2 {
		t.Fatalf("Minutes5 = %d, want 2", Minutes5)
	}
	if got := Minutes5.String(); got != "Minutes5" {
		t.Fatalf("String() = %q, want Minutes5", got)
	}
	if got := Minutes5.ToTimeDuration(); got != 5*time.Minute {
		t.Fatalf("ToTimeDuration() = %v, want 5m", got)
	}
	if got := Minutes5.ToNumberInvestAPI(); got != 2 {
		t.Fatalf("ToNumberInvestAPI() = %d, want 2", got)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/enum/ -run TestMinutes5 -v`
Expected: FAIL (undefined: `Minutes5`).

- [ ] **Step 3: Implement in `internal/enum/enum.go`**

Add to the const block:

```go
	Minutes5  Interval = 2
```

Add to `intervalNames`:

```go
	2:  "Minutes5",
```

Add a case in `ToTimeDuration`'s switch:

```go
	case Minutes5:
		interval = time.Minute * 5
```

Add a branch in `ToNumberInvestAPI` (before the final `return 0`):

```go
	if i == Minutes5 {
		return 2
	}
```

- [ ] **Step 4: Wire `parseInterval` in `cmd/backtest/main.go`**

In `parseInterval`, add a case (and update the flag help + error text to mention `Minutes5`):

```go
	case "Minutes5":
		return enum.Minutes5, nil
```

Update the `-interval` flag usage string (line ~40) to include `Minutes5`:

```go
	intervalS = flag.String("interval", "Hour1", "candle timeframe: Minutes5|Minutes15|Minutes30|Hour1|Hour4|Day1|Week1")
```

- [ ] **Step 5: Run tests + build**

Run: `go test ./internal/enum/ -run TestMinutes5 -v && go build ./cmd/backtest`
Expected: PASS, build OK.

- [ ] **Step 6: Commit**

```bash
git add internal/enum/enum.go internal/enum/enum_test.go cmd/backtest/main.go
git commit -m "feat(enum): add Minutes5 interval and backtest flag support"
```

---

### Task 2: Configurable HTF interval on `Config`

**Files:**
- Modify: `internal/domain/backtest/types.go:17-24` (`Config`)
- Modify: `internal/domain/backtest/engine.go` (`AssembleMarketData`, `Run`, `Trace`, remove reliance on the `htfInterval` const)
- Test: `internal/domain/backtest/engine_test.go` (append)

**Interfaces:**
- Consumes: nothing new.
- Produces: `Config.HTFInterval time.Duration`; `AssembleMarketData(window, daily, htf []Candle, cur time.Time, htfInterval time.Duration) strategy.MarketData` (new trailing param). `Run`/`Trace` pass `htfIntervalOrDefault(cfg)` which returns `cfg.HTFInterval`, or `4*time.Hour` when zero.

- [ ] **Step 1: Write the failing test**

Append to `internal/domain/backtest/engine_test.go`:

```go
func TestAssembleMarketDataRespectsHTFInterval(t *testing.T) {
	base := time.Date(2026, 1, 5, 10, 0, 0, 0, time.UTC)
	htf := []Candle{
		{Time: base, High: 10, Low: 9, Close: 9.5},                 // closes at 11:00 for 1h
		{Time: base.Add(time.Hour), High: 11, Low: 10, Close: 10.5}, // closes at 12:00 for 1h
	}
	cur := base.Add(90 * time.Minute) // 11:30 — first 1h bar closed, second not
	md := AssembleMarketData(nil, nil, htf, cur, time.Hour)
	if len(md.HTFCloses) != 1 {
		t.Fatalf("1h interval: HTFCloses=%d, want 1 (only the 10:00 bar closed by 11:30)", len(md.HTFCloses))
	}
	// With a 4h interval neither bar is closed by 11:30.
	md4 := AssembleMarketData(nil, nil, htf, cur, 4*time.Hour)
	if len(md4.HTFCloses) != 0 {
		t.Fatalf("4h interval: HTFCloses=%d, want 0", len(md4.HTFCloses))
	}
}

func TestHTFIntervalOrDefault(t *testing.T) {
	if got := htfIntervalOrDefault(Config{}); got != 4*time.Hour {
		t.Fatalf("zero -> %v, want 4h", got)
	}
	if got := htfIntervalOrDefault(Config{HTFInterval: time.Hour}); got != time.Hour {
		t.Fatalf("1h -> %v, want 1h", got)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/domain/backtest/ -run 'TestAssembleMarketDataRespectsHTFInterval|TestHTFIntervalOrDefault' -v`
Expected: FAIL (too few args to `AssembleMarketData`; undefined `htfIntervalOrDefault`).

- [ ] **Step 3: Add the field + helper**

In `types.go` (which already imports `time` for `Candle`), add to `Config`:

```go
	HTFInterval time.Duration // higher-timeframe bar span for completeness; 0 = default 4h (reversion)
```

In `engine.go`, add near the top (after the `htfInterval` const, which stays as the default source):

```go
// htfIntervalOrDefault returns cfg.HTFInterval, or the legacy 4h span when unset,
// so reversion (which never sets the field) keeps its Hour4 completeness rule.
func htfIntervalOrDefault(cfg Config) time.Duration {
	if cfg.HTFInterval > 0 {
		return cfg.HTFInterval
	}
	return htfInterval
}
```

- [ ] **Step 4: Thread the interval through `AssembleMarketData` and its callers**

Change `AssembleMarketData`'s signature and body (`engine.go:263-269`):

```go
func AssembleMarketData(window, daily, htf []Candle, cur time.Time, htfSpan time.Duration) strategy.MarketData {
	md := buildMarketData(window)
	md.DailyCloses = visibleDailyCloses(daily, cur, mskLoc)
	md.DailyHighs, md.DailyLows = visibleDailyHighsLows(daily, cur, mskLoc)
	md.HTFCloses, md.HTFHighs, md.HTFLows = visibleCompletedHTF(htf, cur, htfSpan)
	return md
}
```

In `Run` (line ~114) and `Trace` (line ~178), replace both `AssembleMarketData(...)` calls with the interval passed:

```go
		md := AssembleMarketData(candles[i-l+1:i+1], dailyCandles, htfCandles, candles[i].Time, htfIntervalOrDefault(cfg))
```

(In `Trace` the window slice is identical; make the same edit.) Search for any other `AssembleMarketData(` callers and update them:

Run: `grep -rn "AssembleMarketData(" --include=*.go`
Update each non-test caller to pass a 5th arg (`4*time.Hour` preserves old behavior); update test callers to pass `4*time.Hour`.

- [ ] **Step 5: Run tests + build**

Run: `go test ./internal/domain/backtest/ -run 'TestAssembleMarketDataRespectsHTFInterval|TestHTFIntervalOrDefault' -v && go build ./internal/... ./pkg/... ./cmd/...`
Expected: PASS, build OK.

- [ ] **Step 6: Run the full backtest package tests (no regression)**

Run: `go test ./internal/domain/backtest/ ./internal/service/backtest/`
Expected: PASS (reversion path unchanged — it never sets `HTFInterval`, so `htfIntervalOrDefault` returns 4h).

- [ ] **Step 7: Commit**

```bash
git add internal/domain/backtest/types.go internal/domain/backtest/engine.go internal/domain/backtest/engine_test.go
git commit -m "feat(backtest): make HTF interval configurable via Config.HTFInterval"
```

---

### Task 3: Add `Opens` series to `MarketData`

**Files:**
- Modify: `internal/service/trading_strategy/scalping/strategy/strategy.go:33-66` (`MarketData`)
- Modify: `internal/domain/backtest/engine.go:237-256` (`buildMarketData`)
- Test: `internal/domain/backtest/engine_test.go` (append)

**Interfaces:**
- Produces: `MarketData.Opens []float64` — oldest-first bar opens, index-aligned to `Closes`; empty when the runner does not supply them (live paths that don't set it; consumers must guard on `len(Opens)`).

- [ ] **Step 1: Write the failing test**

Append to `internal/domain/backtest/engine_test.go`:

```go
func TestBuildMarketDataPopulatesOpens(t *testing.T) {
	window := []Candle{
		{Open: 1, High: 2, Low: 0.5, Close: 1.5, Volume: 100},
		{Open: 1.5, High: 2.5, Low: 1.0, Close: 2.0, Volume: 200},
	}
	md := AssembleMarketData(window, nil, nil, time.Now(), 4*time.Hour)
	if len(md.Opens) != 2 || md.Opens[0] != 1 || md.Opens[1] != 1.5 {
		t.Fatalf("Opens = %v, want [1 1.5]", md.Opens)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/domain/backtest/ -run TestBuildMarketDataPopulatesOpens -v`
Expected: FAIL (`md.Opens` undefined).

- [ ] **Step 3: Add the field**

In `strategy.go`, add to `MarketData` right after `Price`:

```go
	// Opens are oldest-first bar opens, index-aligned to Closes. Empty when the runner
	// does not supply them (live paths that don't set it); consumers must guard on len.
	Opens []float64
```

- [ ] **Step 4: Populate it in `buildMarketData`**

In `engine.go` `buildMarketData`, add `Opens` to the struct literal and the loop:

```go
	md := strategy.MarketData{
		Opens:   make([]float64, len(window)),
		Highs:   make([]float64, len(window)),
		Lows:    make([]float64, len(window)),
		Closes:  make([]float64, len(window)),
		Volumes: make([]int64, len(window)),
		Times:   make([]time.Time, len(window)),
	}
	for i, c := range window {
		md.Opens[i] = c.Open
		md.Highs[i] = c.High
		md.Lows[i] = c.Low
		md.Closes[i] = c.Close
		md.Volumes[i] = c.Volume
		md.Times[i] = c.Time
	}
```

- [ ] **Step 5: Run tests + build**

Run: `go test ./internal/domain/backtest/ -run TestBuildMarketDataPopulatesOpens -v && go build ./internal/... ./pkg/... ./cmd/...`
Expected: PASS, build OK.

- [ ] **Step 6: Commit**

```bash
git add internal/service/trading_strategy/scalping/strategy/strategy.go internal/domain/backtest/engine.go internal/domain/backtest/engine_test.go
git commit -m "feat(backtest): expose bar Opens on MarketData for candle-body strategies"
```

---

### Task 4: `daylow` core — Params, skeleton, Lookback, session helper

**Files:**
- Create: `internal/service/trading_strategy/daylow/strategy/core/core.go`
- Test: `internal/service/trading_strategy/daylow/strategy/core/core_test.go`

**Interfaces:**
- Consumes: `strategy.MarketData` (with `Opens`), `model.Signal`, `indicators.ATR`, `ema.Compute`.
- Produces: `core.Params`, `core.DefaultParams() Params`, `core.NewWithParams(ticker string, p Params) *Strategy`, `(*Strategy).Ticker() string`, `(*Strategy).Lookback() int`, `(*Strategy).Decide(md strategy.MarketData) model.Signal` (entry/manage added in Tasks 5-6; for now returns `SignalNone`). Unexported helpers `inSession(t time.Time) bool`, `barMinute(md strategy.MarketData) int`.

- [ ] **Step 1: Write the failing test**

Create `internal/service/trading_strategy/daylow/strategy/core/core_test.go`:

```go
package core

import (
	"testing"
	"time"
)

func msk(hour, minute int) time.Time {
	loc, _ := time.LoadLocation("Europe/Moscow")
	return time.Date(2026, 1, 5, hour, minute, 0, 0, loc)
}

func TestDefaultParams(t *testing.T) {
	p := DefaultParams()
	if p.ConsolBars != 6 || p.RR != 2.0 || p.EntrySessionStartMin != 540 || p.EntrySessionEndMin != 840 || p.DayEndMin != 1120 {
		t.Fatalf("unexpected defaults: %+v", p)
	}
}

func TestLookback(t *testing.T) {
	s := NewWithParams("TEST", DefaultParams()) // ConsolBars=6 -> 7; ATRPeriod=14 -> 15; +5
	if got := s.Lookback(); got != 20 {
		t.Fatalf("Lookback() = %d, want 20", got)
	}
}

func TestInSession(t *testing.T) {
	s := NewWithParams("TEST", DefaultParams()) // window [540,840) = 09:00..14:00
	cases := []struct {
		t    time.Time
		want bool
	}{
		{msk(8, 59), false},
		{msk(9, 0), true},
		{msk(13, 59), true},
		{msk(14, 0), false},
		{time.Time{}, true}, // zero time -> gate skipped (degrade like reversion)
	}
	for _, c := range cases {
		if got := s.inSession(c.t); got != c.want {
			t.Fatalf("inSession(%v) = %v, want %v", c.t, got, c.want)
		}
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/service/trading_strategy/daylow/strategy/core/ -v`
Expected: FAIL (package/symbols do not exist).

- [ ] **Step 3: Create `core.go` with Params + skeleton**

```go
// Package core implements a long-only 5-minute "consolidation breakout at the prior-day
// low" strategy. When flat it looks for a tight consolidation zone sitting at yesterday's
// daily low, then enters on a green impulse candle that breaks the zone high on rising
// volume — only inside a configurable MOEX active-hours window, and (optionally) only when
// the higher-timeframe (hourly) trend is up. The stop is frozen just below the zone; the
// take-profit is RR times the entry risk. Exits (SL/TP/EOD) fire at any time of day.
// The decision logic is pure and ticker-agnostic. Run with `-strategy daylow -interval Minutes5`.
package core

import (
	"fmt"
	"math"
	"time"

	"tinvest/internal/domain/ema"
	"tinvest/internal/service/trading_strategy/scalping/model"
	"tinvest/internal/service/trading_strategy/scalping/strategy"
	"tinvest/pkg/indicators"
)

// Params holds every tunable. All fields are int or float64 (flags as int 0/1) so
// reflection grid calibration can sweep them.
type Params struct {
	LowProximityPct      float64 // max |zoneLow - prevDayLow| / prevDayLow for the zone to count as "at" the low
	ConsolBars           int     // bars before the impulse candle that form the consolidation zone
	ConsolRangeATR       float64 // zone is "tight" when (zoneHigh-zoneLow) <= ConsolRangeATR*ATR
	ATRPeriod            int     // ATR length (unit for zone/stop/body thresholds)
	ImpulseBodyATR       float64 // impulse body (close-open) >= ImpulseBodyATR*ATR
	VolMult              float64 // impulse volume >= VolMult * mean zone volume
	StopBufferATR        float64 // stop = zoneLow - StopBufferATR*ATR
	RR                   float64 // take-profit = entry + RR*(entry-stop)
	HTFTrendEMA          int     // EMA period on the HTF (hourly) series; 0 = trend filter off
	EntrySessionStartMin int     // entry window start, minutes from MSK midnight (540 = 09:00)
	EntrySessionEndMin   int     // entry window end, minutes from MSK midnight (840 = 14:00)
	CloseAtDayEnd        int     // 1 = close the position at day end; 0 = hold
	DayEndMin            int     // day-end close threshold, minutes from MSK midnight (1120 = 18:40)
}

// DefaultParams returns neutral starting values; real values come from calibration.
func DefaultParams() Params {
	return Params{
		LowProximityPct:      0.005,
		ConsolBars:           6,
		ConsolRangeATR:       1.0,
		ATRPeriod:            14,
		ImpulseBodyATR:       0.8,
		VolMult:              1.5,
		StopBufferATR:        0.25,
		RR:                   2.0,
		HTFTrendEMA:          50,
		EntrySessionStartMin: 540,
		EntrySessionEndMin:   840,
		CloseAtDayEnd:        0,
		DayEndMin:            1120,
	}
}

// Strategy trades a single instrument with the breakout rules. Ticker-agnostic and pure.
type Strategy struct {
	ticker string
	p      Params
}

// NewWithParams returns the daylow strategy for a ticker with explicit params.
func NewWithParams(ticker string, p Params) *Strategy { return &Strategy{ticker: ticker, p: p} }

func (s *Strategy) Ticker() string { return s.ticker }

// Lookback sizes the candle window to feed the hungriest consumer (zone + ATR).
func (s *Strategy) Lookback() int {
	m := s.p.ConsolBars + 1
	if s.p.ATRPeriod+1 > m {
		m = s.p.ATRPeriod + 1
	}
	return m + 5
}

// mskLoc anchors the trading-session window to the Moscow calendar (UTC fallback).
var mskLoc = func() *time.Location {
	loc, err := time.LoadLocation("Europe/Moscow")
	if err != nil {
		return time.UTC
	}
	return loc
}()

// inSession reports whether bar-time t falls in the entry window [start,end) in MSK.
// A zero time (live paths without Times) skips the gate — never block on missing data.
func (s *Strategy) inSession(t time.Time) bool {
	if t.IsZero() {
		return true
	}
	tl := t.In(mskLoc)
	min := tl.Hour()*60 + tl.Minute()
	return min >= s.p.EntrySessionStartMin && min < s.p.EntrySessionEndMin
}

// barMinute returns the MSK minute-of-day of the latest bar, or -1 when Times is absent
// or misaligned (so the EOD gate degrades to a no-op).
func (s *Strategy) barMinute(md strategy.MarketData) int {
	n := len(md.Times)
	if n == 0 || len(md.Closes) != n {
		return -1
	}
	tl := md.Times[n-1].In(mskLoc)
	return tl.Hour()*60 + tl.Minute()
}

// Decide is filled out in Tasks 5-6; for now it never trades.
func (s *Strategy) Decide(md strategy.MarketData) model.Signal {
	return model.Signal{Ticker: s.ticker, Price: md.Price}
}

// Silence unused-import errors until Tasks 5-6 wire these in.
var (
	_ = fmt.Sprintf
	_ = math.Abs
	_ = ema.Compute
	_ = indicators.ATR
)
```

- [ ] **Step 4: Run tests + build**

Run: `go test ./internal/service/trading_strategy/daylow/strategy/core/ -v && go build ./internal/...`
Expected: PASS, build OK.

- [ ] **Step 5: Commit**

```bash
git add internal/service/trading_strategy/daylow/strategy/core/
git commit -m "feat(daylow): strategy scaffold, params, lookback, session helpers"
```

---

### Task 5: `daylow` entry logic (`enter`)

**Files:**
- Modify: `internal/service/trading_strategy/daylow/strategy/core/core.go`
- Test: `internal/service/trading_strategy/daylow/strategy/core/core_test.go` (append)

**Interfaces:**
- Consumes: helpers from Task 4.
- Produces: unexported `(*Strategy).enter(md strategy.MarketData, sig model.Signal) model.Signal`, called from `Decide` when `md.Position == nil`. On a valid setup returns `model.Signal{Kind: SignalBuy, Price: close, StopLoss: zoneLow - StopBufferATR*ATR, TakeProfit: close + RR*(close-stop), ATR: atr, Level: prevDayLow, EntryReason: <text>}`.

- [ ] **Step 1: Write the failing test**

Append to `core_test.go`. This builds a window with a 6-bar tight zone sitting at the prior-day low, followed by a green impulse bar breaking the zone on 3× volume:

```go
func buildEntryMD(p Params) strategy.MarketData {
	// 6 consolidation bars around 100.0 (tight, high-low <= ATR), then an impulse bar.
	loc, _ := time.LoadLocation("Europe/Moscow")
	start := time.Date(2026, 1, 5, 10, 0, 0, 0, loc) // inside 09:00-14:00
	var opens, highs, lows, closes []float64
	var vols []int64
	var times []time.Time
	// Warm-up filler bars so ATR(14) is comfortably defined: 15 flat bars before the zone.
	for i := 0; i < 15; i++ {
		opens = append(opens, 100)
		highs = append(highs, 100.4)
		lows = append(lows, 99.6)
		closes = append(closes, 100)
		vols = append(vols, 1000)
		times = append(times, start.Add(time.Duration(i-20)*5*time.Minute))
	}
	// 6 tight consolidation bars: zoneHigh=100.3, zoneLow=99.8 (range 0.5 < ATR).
	for i := 0; i < 6; i++ {
		opens = append(opens, 100.0)
		highs = append(highs, 100.3)
		lows = append(lows, 99.8)
		closes = append(closes, 100.0)
		vols = append(vols, 1000)
		times = append(times, start.Add(time.Duration(i)*5*time.Minute))
	}
	// Impulse bar: green, big body, breaks zoneHigh, 3x volume.
	opens = append(opens, 100.1)
	highs = append(highs, 101.6)
	lows = append(lows, 100.0)
	closes = append(closes, 101.5)
	vols = append(vols, 3000)
	times = append(times, start.Add(6*5*time.Minute))

	return strategy.MarketData{
		Price:     closes[len(closes)-1],
		Opens:     opens, Highs: highs, Lows: lows, Closes: closes, Volumes: vols, Times: times,
		DailyLows: []float64{99.8}, // prior-day low == zoneLow
		// HTF trend confirmed: closes above their EMA.
		HTFCloses: repeatUp(60),
	}
}

// repeatUp returns n rising closes so any EMA sits below the last close (trend up).
func repeatUp(n int) []float64 {
	out := make([]float64, n)
	for i := range out {
		out[i] = 90 + float64(i)*0.2
	}
	return out
}

func TestEnterFiresOnValidSetup(t *testing.T) {
	p := DefaultParams()
	s := NewWithParams("TEST", p)
	sig := s.Decide(buildEntryMD(p))
	if sig.Kind != model.SignalBuy {
		t.Fatalf("Kind = %v, want SignalBuy", sig.Kind)
	}
	wantStop := 99.8 - p.StopBufferATR*sig.ATR
	if math.Abs(sig.StopLoss-wantStop) > 1e-9 {
		t.Fatalf("StopLoss = %.6f, want %.6f", sig.StopLoss, wantStop)
	}
	wantTP := sig.Price + p.RR*(sig.Price-sig.StopLoss)
	if math.Abs(sig.TakeProfit-wantTP) > 1e-9 {
		t.Fatalf("TakeProfit = %.6f, want %.6f", sig.TakeProfit, wantTP)
	}
}

func TestEnterBlocked(t *testing.T) {
	base := DefaultParams()
	cases := map[string]func(*strategy.MarketData){
		"outside session":  func(m *strategy.MarketData) { for i := range m.Times { m.Times[i] = m.Times[i].Add(6 * time.Hour) } }, // shove to 16:00+
		"no daily low":     func(m *strategy.MarketData) { m.DailyLows = nil },
		"zone off the low":  func(m *strategy.MarketData) { m.DailyLows = []float64{80.0} },
		"red impulse":       func(m *strategy.MarketData) { n := len(m.Closes) - 1; m.Opens[n], m.Closes[n] = 101.5, 100.05 },
		"weak volume":       func(m *strategy.MarketData) { m.Volumes[len(m.Volumes)-1] = 1000 },
		"no breakout":       func(m *strategy.MarketData) { n := len(m.Closes) - 1; m.Closes[n], m.Highs[n] = 100.2, 100.25 },
		"htf downtrend":     func(m *strategy.MarketData) { for i := range m.HTFCloses { m.HTFCloses[i] = 200 - float64(i)*0.2 } },
	}
	for name, mutate := range cases {
		md := buildEntryMD(base)
		mutate(&md)
		if sig := NewWithParams("TEST", base).Decide(md); sig.Kind == model.SignalBuy {
			t.Fatalf("%s: expected no buy, got SignalBuy", name)
		}
	}
}

// import model at top of test file
var _ = model.SignalBuy
```

Add `"tinvest/internal/service/trading_strategy/scalping/model"` and `"tinvest/internal/service/trading_strategy/scalping/strategy"` and `"math"` to the test imports.

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/service/trading_strategy/daylow/strategy/core/ -run 'TestEnter' -v`
Expected: FAIL (`Decide` never buys — still the skeleton).

- [ ] **Step 3: Implement `enter` and call it from `Decide`**

Replace the placeholder `Decide` and the unused-import block with:

```go
// Decide routes to entry (flat) or position management (open).
func (s *Strategy) Decide(md strategy.MarketData) model.Signal {
	sig := model.Signal{Ticker: s.ticker, Price: md.Price}
	if md.Position != nil {
		return s.manage(md, sig) // added in Task 6
	}
	return s.enter(md, sig)
}

// enter runs the entry gates in order; any failing gate returns SignalNone.
func (s *Strategy) enter(md strategy.MarketData, sig model.Signal) model.Signal {
	n := len(md.Closes)
	if n < s.p.ConsolBars+1 || len(md.Opens) != n {
		return sig
	}
	// 1. active-hours window (skipped when Times absent).
	var barTime time.Time
	if len(md.Times) == n {
		barTime = md.Times[n-1]
	}
	if !s.inSession(barTime) {
		return sig
	}
	// 2. ATR unit.
	atr := indicators.ATR(md.Highs, md.Lows, md.Closes, s.p.ATRPeriod)
	if atr <= 0 {
		return sig
	}
	// 3. consolidation zone over the ConsolBars bars BEFORE the impulse bar.
	lo := n - 1 - s.p.ConsolBars
	if lo < 0 {
		return sig
	}
	zoneHigh, zoneLow := md.Highs[lo], md.Lows[lo]
	for j := lo; j <= n-2; j++ {
		if md.Highs[j] > zoneHigh {
			zoneHigh = md.Highs[j]
		}
		if md.Lows[j] < zoneLow {
			zoneLow = md.Lows[j]
		}
	}
	if zoneHigh-zoneLow > s.p.ConsolRangeATR*atr {
		return sig
	}
	// 4. zone sits at yesterday's daily low.
	if len(md.DailyLows) == 0 {
		return sig
	}
	prevDayLow := md.DailyLows[len(md.DailyLows)-1]
	if prevDayLow <= 0 || math.Abs(zoneLow-prevDayLow) > s.p.LowProximityPct*prevDayLow {
		return sig
	}
	// 5. impulse candle = current bar: green, big body, breaks the zone high.
	open, close := md.Opens[n-1], md.Closes[n-1]
	body := close - open
	if body <= 0 || body < s.p.ImpulseBodyATR*atr || close <= zoneHigh {
		return sig
	}
	// 6. rising volume vs the zone average.
	avgVol, ok := meanVolume(md.Volumes, lo, n-1)
	if !ok || float64(md.Volumes[n-1]) < s.p.VolMult*avgVol {
		return sig
	}
	// 7. optional HTF (hourly) trend filter: last completed HTF close above its EMA.
	if s.p.HTFTrendEMA > 0 && !s.htfUptrend(md) {
		return sig
	}

	stop := zoneLow - s.p.StopBufferATR*atr
	sig.Kind = model.SignalBuy
	sig.StopLoss = stop
	sig.TakeProfit = close + s.p.RR*(close-stop)
	sig.ATR = atr
	sig.Level = prevDayLow
	sig.EntryReason = s.entryReason(prevDayLow, zoneLow, zoneHigh, atr, close)
	return sig
}

// meanVolume averages volumes in [lo, hi) (the zone bars, excluding the impulse bar at hi).
// ok is false when no positive sample exists — the caller then skips the gate.
func meanVolume(vols []int64, lo, hi int) (float64, bool) {
	if lo < 0 || hi > len(vols) || lo >= hi {
		return 0, false
	}
	var sum float64
	var count int
	for j := lo; j < hi; j++ {
		if vols[j] > 0 {
			sum += float64(vols[j])
			count++
		}
	}
	if count == 0 {
		return 0, false
	}
	return sum / float64(count), true
}

// htfUptrend reports whether the last completed HTF close is above its EMA. Un-warmed
// HTF data (too few closes / zero-sentinel EMA) returns false: a protective filter must
// not trade when it cannot confirm the higher trend.
func (s *Strategy) htfUptrend(md strategy.MarketData) bool {
	if len(md.HTFCloses) < s.p.HTFTrendEMA {
		return false
	}
	e := ema.Compute(md.HTFCloses, s.p.HTFTrendEMA)
	if len(e) == 0 {
		return false
	}
	v := e[len(e)-1]
	if v == 0 { // warm-up sentinel (prices are positive, so a real EMA is never 0)
		return false
	}
	return md.HTFCloses[len(md.HTFCloses)-1] > v
}

// entryReason renders the human-readable rationale shown in the trade journal.
func (s *Strategy) entryReason(prevDayLow, zoneLow, zoneHigh, atr, close float64) string {
	htf := "выкл"
	if s.p.HTFTrendEMA > 0 {
		htf = fmt.Sprintf("EMA%d(HTF) вверх", s.p.HTFTrendEMA)
	}
	return fmt.Sprintf(
		"отбой от вчерашнего минимума %.4f: зона [%.4f..%.4f] (≤%.2f×ATR), импульс close %.4f > зоны, объём ≥%.2f×; HTF: %s",
		prevDayLow, zoneLow, zoneHigh, s.p.ConsolRangeATR, close, s.p.VolMult, htf,
	)
}
```

Remove the temporary `var ( _ = fmt.Sprintf ... )` block added in Task 4 (the symbols are now used).

- [ ] **Step 4: Add a temporary no-op `manage` so the package compiles before Task 6**

Add (will be replaced in Task 6):

```go
// manage is implemented in Task 6; temporary no-op keeps the package compiling.
func (s *Strategy) manage(md strategy.MarketData, sig model.Signal) model.Signal { return sig }
```

- [ ] **Step 5: Run tests + build**

Run: `go test ./internal/service/trading_strategy/daylow/strategy/core/ -run 'TestEnter' -v && go build ./internal/...`
Expected: PASS, build OK.

- [ ] **Step 6: Commit**

```bash
git add internal/service/trading_strategy/daylow/strategy/core/
git commit -m "feat(daylow): entry gates — zone/proximity/impulse/volume/HTF, frozen stop+TP"
```

---

### Task 6: `daylow` position management (`manage`)

**Files:**
- Modify: `internal/service/trading_strategy/daylow/strategy/core/core.go` (replace the no-op `manage`)
- Test: `internal/service/trading_strategy/daylow/strategy/core/core_test.go` (append)

**Interfaces:**
- Consumes: `strategy.Position` (`PurchasePrice`, `StopLoss`), `barMinute` helper.
- Produces: `(*Strategy).manage` emitting `SignalSell` with `Reason` in `{"SL","TP","EOD"}`. Precedence: SL (`low <= StopLoss`) → TP (`high >= PurchasePrice + RR*(PurchasePrice-StopLoss)`) → EOD (`CloseAtDayEnd==1 && barMinute >= DayEndMin`).

- [ ] **Step 1: Write the failing test**

Append to `core_test.go`:

```go
func positionMD(low, high float64, barMin int, pos *strategy.Position) strategy.MarketData {
	loc, _ := time.LoadLocation("Europe/Moscow")
	t := time.Date(2026, 1, 5, barMin/60, barMin%60, 0, 0, loc)
	return strategy.MarketData{
		Price: (low + high) / 2,
		Highs: []float64{high}, Lows: []float64{low}, Closes: []float64{(low + high) / 2},
		Times: []time.Time{t}, Position: pos,
	}
}

func TestManageExits(t *testing.T) {
	p := DefaultParams()
	p.CloseAtDayEnd = 1
	s := NewWithParams("TEST", p)
	pos := &strategy.Position{PurchasePrice: 100, StopLoss: 98} // R=2, TP=100+2*2=104
	// SL: low pierces the stop.
	if sig := s.manage(positionMD(97.9, 100, 660, pos), model.Signal{}); sig.Kind != model.SignalSell || sig.Reason != "SL" || sig.StopLoss != 98 {
		t.Fatalf("SL: got kind=%v reason=%q stop=%.2f", sig.Kind, sig.Reason, sig.StopLoss)
	}
	// TP: high reaches the target, low safe.
	if sig := s.manage(positionMD(100, 104.1, 660, pos), model.Signal{}); sig.Kind != model.SignalSell || sig.Reason != "TP" || sig.TakeProfit != 104 {
		t.Fatalf("TP: got kind=%v reason=%q tp=%.2f", sig.Kind, sig.Reason, sig.TakeProfit)
	}
	// SL precedence when both hit in one bar.
	if sig := s.manage(positionMD(97.9, 104.1, 660, pos), model.Signal{}); sig.Reason != "SL" {
		t.Fatalf("both-hit: reason=%q, want SL", sig.Reason)
	}
	// EOD: neither SL nor TP, but past DayEndMin (1120 = 18:40).
	if sig := s.manage(positionMD(100, 101, 1125, pos), model.Signal{}); sig.Kind != model.SignalSell || sig.Reason != "EOD" {
		t.Fatalf("EOD: got kind=%v reason=%q", sig.Kind, sig.Reason)
	}
	// Hold: nothing hit, before DayEndMin.
	if sig := s.manage(positionMD(100, 101, 660, pos), model.Signal{}); sig.Kind == model.SignalSell {
		t.Fatalf("hold: unexpected sell")
	}
	// CloseAtDayEnd off -> no EOD exit.
	sOff := NewWithParams("TEST", DefaultParams())
	if sig := sOff.manage(positionMD(100, 101, 1125, pos), model.Signal{}); sig.Kind == model.SignalSell {
		t.Fatalf("EOD off: unexpected sell")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/service/trading_strategy/daylow/strategy/core/ -run TestManageExits -v`
Expected: FAIL (no-op `manage` never sells).

- [ ] **Step 3: Replace the no-op `manage`**

```go
// manage handles an open long, exiting in precedence SL -> TP -> EOD. The take-profit
// is reconstructed each bar from the frozen entry (PurchasePrice, StopLoss) — no TP field
// is stored on the Position. SL fills intrabar at min(StopLoss, open); TP at
// max(TakeProfit, open); EOD at close. SL wins when both SL and TP touch the same bar.
func (s *Strategy) manage(md strategy.MarketData, sig model.Signal) model.Signal {
	pos := md.Position
	n := len(md.Lows)
	if pos == nil || n == 0 || len(md.Highs) != n {
		return sig
	}
	low, high := md.Lows[n-1], md.Highs[n-1]
	risk := pos.PurchasePrice - pos.StopLoss
	tp := pos.PurchasePrice + s.p.RR*risk

	switch {
	case pos.StopLoss > 0 && low <= pos.StopLoss:
		sig.Kind, sig.Reason = model.SignalSell, "SL"
		sig.StopLoss = pos.StopLoss
		sig.ExitReason = fmt.Sprintf("SL: low %.4f ≤ стоп %.4f (вход %.4f)", low, pos.StopLoss, pos.PurchasePrice)
	case risk > 0 && high >= tp:
		sig.Kind, sig.Reason = model.SignalSell, "TP"
		sig.TakeProfit = tp
		sig.ExitReason = fmt.Sprintf("TP: high %.4f ≥ тейк %.4f (вход %.4f, RR %.2f)", high, tp, pos.PurchasePrice, s.p.RR)
	case s.p.CloseAtDayEnd == 1:
		if m := s.barMinute(md); m >= 0 && m >= s.p.DayEndMin {
			sig.Kind, sig.Reason = model.SignalSell, "EOD"
			sig.ExitReason = fmt.Sprintf("EOD: закрытие по времени (мин %d ≥ %d)", m, s.p.DayEndMin)
		}
	}
	return sig
}
```

- [ ] **Step 4: Run tests + build**

Run: `go test ./internal/service/trading_strategy/daylow/strategy/core/ -v && go build ./internal/...`
Expected: PASS (all daylow tests), build OK.

- [ ] **Step 5: Commit**

```bash
git add internal/service/trading_strategy/daylow/strategy/core/
git commit -m "feat(daylow): position management — SL/TP/EOD exits with reconstructed TP"
```

---

### Task 7: `Explain` diagnostic for `-explain`

**Files:**
- Modify: `internal/service/trading_strategy/daylow/strategy/core/core.go`
- Test: `internal/service/trading_strategy/daylow/strategy/core/core_test.go` (append)

**Interfaces:**
- Produces: `(*Strategy).Explain(md strategy.MarketData) string` — the gate-by-gate verdict the engine's `Trace` prints for `-explain`.

- [ ] **Step 1: Write the failing test**

```go
func TestExplainReportsBlockingGate(t *testing.T) {
	p := DefaultParams()
	md := buildEntryMD(p)
	md.DailyLows = []float64{80.0} // zone off the low -> proximity gate blocks
	out := NewWithParams("TEST", p).Explain(md)
	if out == "" {
		t.Fatal("Explain returned empty string")
	}
	if !strings.Contains(out, "минимум") {
		t.Fatalf("Explain missing proximity gate line: %q", out)
	}
}
```

Add `"strings"` to the test imports.

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/service/trading_strategy/daylow/strategy/core/ -run TestExplainReportsBlockingGate -v`
Expected: FAIL (`Explain` undefined).

- [ ] **Step 3: Implement `Explain`**

```go
// Explain returns a gate-by-gate verdict for one bar, consumed by the engine's Trace
// (-explain). It recomputes the same values enter() does and reports pass/fail per gate.
func (s *Strategy) Explain(md strategy.MarketData) string {
	var sb strings.Builder
	n := len(md.Closes)
	if n < s.p.ConsolBars+1 || len(md.Opens) != n {
		return "недостаточно свечей/нет Opens\n"
	}
	var barTime time.Time
	if len(md.Times) == n {
		barTime = md.Times[n-1]
	}
	fmt.Fprintf(&sb, "сессия: %v\n", s.inSession(barTime))

	atr := indicators.ATR(md.Highs, md.Lows, md.Closes, s.p.ATRPeriod)
	fmt.Fprintf(&sb, "ATR(%d): %.4f\n", s.p.ATRPeriod, atr)

	lo := n - 1 - s.p.ConsolBars
	zoneHigh, zoneLow := md.Highs[lo], md.Lows[lo]
	for j := lo; j <= n-2; j++ {
		zoneHigh = math.Max(zoneHigh, md.Highs[j])
		zoneLow = math.Min(zoneLow, md.Lows[j])
	}
	fmt.Fprintf(&sb, "зона [%.4f..%.4f] высота %.4f ≤ %.4f? %v\n",
		zoneLow, zoneHigh, zoneHigh-zoneLow, s.p.ConsolRangeATR*atr, zoneHigh-zoneLow <= s.p.ConsolRangeATR*atr)

	if len(md.DailyLows) > 0 {
		prev := md.DailyLows[len(md.DailyLows)-1]
		fmt.Fprintf(&sb, "минимум прошлого дня %.4f, |zoneLow-low|=%.4f ≤ %.4f? %v\n",
			prev, math.Abs(zoneLow-prev), s.p.LowProximityPct*prev, math.Abs(zoneLow-prev) <= s.p.LowProximityPct*prev)
	} else {
		sb.WriteString("минимум прошлого дня: нет данных\n")
	}

	open, close := md.Opens[n-1], md.Closes[n-1]
	fmt.Fprintf(&sb, "импульс: тело %.4f ≥ %.4f и close>%.4f? %v\n",
		close-open, s.p.ImpulseBodyATR*atr, zoneHigh, close-open >= s.p.ImpulseBodyATR*atr && close > zoneHigh)

	if avg, ok := meanVolume(md.Volumes, lo, n-1); ok {
		fmt.Fprintf(&sb, "объём %d ≥ %.0f (=%.2f×%.0f)? %v\n",
			md.Volumes[n-1], s.p.VolMult*avg, s.p.VolMult, avg, float64(md.Volumes[n-1]) >= s.p.VolMult*avg)
	}
	if s.p.HTFTrendEMA > 0 {
		fmt.Fprintf(&sb, "HTF тренд вверх? %v\n", s.htfUptrend(md))
	}
	return sb.String()
}
```

Add `"strings"` to the `core.go` imports.

- [ ] **Step 4: Run tests + build**

Run: `go test ./internal/service/trading_strategy/daylow/strategy/core/ -v && go build ./internal/...`
Expected: PASS, build OK.

- [ ] **Step 5: Commit**

```bash
git add internal/service/trading_strategy/daylow/strategy/core/
git commit -m "feat(daylow): gate-by-gate Explain for -explain diagnostics"
```

---

### Task 8: Register `daylow` binding + wire `cmd/backtest`

**Files:**
- Create: `internal/service/backtest/daylow_registry.go`
- Test: `internal/service/backtest/daylow_registry_test.go`
- Modify: `cmd/backtest/main.go` (add `-htf-interval` flag; `daylow` in the strategy switch; load HTF candles + set `cfg.HTFInterval` for `daylow`)

**Interfaces:**
- Consumes: `core.DefaultParams`, `core.NewWithParams` (Task 4-6); `Binding` (existing `registry.go`).
- Produces: `svc.DayLowLookupOrGeneric(ticker string) Binding`; `-strategy daylow`; `-htf-interval` flag (default `Hour1`).

- [ ] **Step 1: Write the failing test**

Create `internal/service/backtest/daylow_registry_test.go`:

```go
package backtest

import (
	"testing"

	"tinvest/internal/service/trading_strategy/daylow/strategy/core"
)

func TestDayLowBinding(t *testing.T) {
	b := DayLowLookupOrGeneric("NVTK")
	p := b.DefaultParams()
	if _, ok := p.(core.Params); !ok {
		t.Fatalf("DefaultParams type = %T, want core.Params", p)
	}
	s := b.Build(p)
	if s.Ticker() != "NVTK" {
		t.Fatalf("Ticker() = %q, want NVTK", s.Ticker())
	}
	// Partial JSON overrides only the named field.
	parsed, err := b.ParseParams([]byte(`{"RR": 1.5}`))
	if err != nil {
		t.Fatalf("ParseParams: %v", err)
	}
	if parsed.(core.Params).RR != 1.5 || parsed.(core.Params).ConsolBars != 6 {
		t.Fatalf("ParseParams merge wrong: %+v", parsed)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/service/backtest/ -run TestDayLowBinding -v`
Expected: FAIL (`DayLowLookupOrGeneric` undefined).

- [ ] **Step 3: Create `daylow_registry.go`**

```go
package backtest

import (
	"encoding/json"
	"fmt"

	"tinvest/internal/service/trading_strategy/daylow/strategy/core"
	"tinvest/internal/service/trading_strategy/scalping/strategy"
)

// daylowBindingFor builds a Binding for a ticker on the daylow engine. daylow is
// ticker-agnostic; only the ticker label differs, so a single generic default suffices.
func daylowBindingFor(ticker string) Binding {
	return Binding{
		DefaultParams: func() any { return core.DefaultParams() },
		Build: func(params any) strategy.Strategy {
			return core.NewWithParams(ticker, params.(core.Params))
		},
		ParseParams: func(raw []byte) (any, error) {
			p := core.DefaultParams() // start from defaults so partial JSON overrides
			if err := json.Unmarshal(raw, &p); err != nil {
				return nil, fmt.Errorf("backtest: parse daylow params: %w", err)
			}
			return p, nil
		},
	}
}

// DayLowLookupOrGeneric returns a daylow binding bound to the given ticker. There are no
// per-ticker daylow packages yet (calibration pending), so every ticker gets the generic
// default params.
func DayLowLookupOrGeneric(ticker string) Binding {
	return daylowBindingFor(ticker)
}
```

- [ ] **Step 4: Run the binding test**

Run: `go test ./internal/service/backtest/ -run TestDayLowBinding -v`
Expected: PASS.

- [ ] **Step 5: Wire `cmd/backtest/main.go`**

(a) Add the flag in the `flag` block (near `intervalS`):

```go
	htfIntervalS = flag.String("htf-interval", "Hour1", "daylow HTF trend-filter timeframe: Minutes15|Minutes30|Hour1|Hour4")
```

(b) Add `*htfIntervalS` to the `run(...)` call and to `run`'s signature (as `htfIntervalS string`). Parse it near the top of `run`, after the primary interval parse in `main`:

In `main`, after `interval, err := parseInterval(*intervalS)`:

```go
	htfInterval, err := parseInterval(*htfIntervalS)
	if err != nil {
		log.Fatalf("backtest: -htf-interval: %v", err)
	}
```

Pass `htfInterval` into `run` (add param `htfInterval enum.Interval`).

(c) In `run`'s strategy switch, add the case:

```go
	case "daylow":
		binding = svc.DayLowLookupOrGeneric(ticker)
```

Update the default error text to `want scalping|reversion|daylow`.

(d) Load HTF candles and set `cfg.HTFInterval` for `daylow`. Extend the existing HTF-loading block (currently reversion-only, ~line 184-190):

```go
	var htfCandles []domain.Candle
	switch strategyName {
	case "reversion":
		htfCandles, err = provider.Load(ctx, ticker, share.ID, enum.Hour4, dailyFrom, to, refresh)
		if err != nil {
			return err
		}
	case "daylow":
		htfCandles, err = provider.Load(ctx, ticker, share.ID, htfInterval, dailyFrom, to, refresh)
		if err != nil {
			return err
		}
	}
```

Then set the interval on `cfg` (find the `cfg := domain.Config{...}` construction, ~line 192, and add the field):

```go
	cfg := domain.Config{InitialCash: cash, Fraction: fraction, Commission: commission, Lot: share.Lot, RiskFractionPct: riskPct}
	switch strategyName {
	case "reversion":
		cfg.HTFInterval = enum.Hour4.ToTimeDuration()
	case "daylow":
		cfg.HTFInterval = htfInterval.ToTimeDuration()
	}
```

Update the `-strategy` flag help string to include `daylow`.

- [ ] **Step 6: Build + run the full backtest tests**

Run: `go build ./cmd/backtest && go test ./internal/service/backtest/ ./internal/domain/backtest/`
Expected: build OK, PASS.

- [ ] **Step 7: Commit**

```bash
git add internal/service/backtest/daylow_registry.go internal/service/backtest/daylow_registry_test.go cmd/backtest/main.go
git commit -m "feat(backtest): register daylow strategy and -htf-interval wiring"
```

---

### Task 9: End-to-end engine integration test

**Files:**
- Test: `internal/service/trading_strategy/daylow/strategy/core/core_test.go` (append)

**Interfaces:**
- Consumes: `backtest.Run`, `backtest.Config`, `backtest.Candle` (domain engine).

- [ ] **Step 1: Write the failing test**

Append an integration test that drives `backtest.Run` over a synthetic 5-minute series with a warm-up, a tight zone at the prior-day low, a green impulse (entry), then a bar whose high hits the take-profit (exit). Import `domain "tinvest/internal/domain/backtest"` and `"tinvest/internal/enum"`.

```go
func TestDaylowRoundTripThroughEngine(t *testing.T) {
	loc, _ := time.LoadLocation("Europe/Moscow")
	day := time.Date(2026, 1, 5, 10, 0, 0, 0, loc)
	var candles []domain.Candle
	push := func(o, h, l, c float64, v int64, tm time.Time) {
		candles = append(candles, domain.Candle{Time: tm, Open: o, High: h, Low: l, Close: c, Volume: v})
	}
	// 20 warm-up bars (before the day's session) so ATR(14)+Lookback are satisfied.
	for i := 0; i < 20; i++ {
		push(100, 100.4, 99.6, 100, 1000, day.Add(time.Duration(i-25)*5*time.Minute))
	}
	// 6 tight consolidation bars at the prior-day low.
	for i := 0; i < 6; i++ {
		push(100, 100.3, 99.8, 100, 1000, day.Add(time.Duration(i)*5*time.Minute))
	}
	// Impulse bar -> entry at close 101.5; stop = 99.8 - 0.25*ATR.
	push(100.1, 101.6, 100.0, 101.5, 3000, day.Add(6*5*time.Minute))
	// Next bar: high spikes above the take-profit -> TP exit.
	push(101.5, 130.0, 101.4, 129.0, 1500, day.Add(7*5*time.Minute))
	// Trailing bar so the loop marks the exit bar.
	push(129, 129.5, 128.5, 129, 1000, day.Add(8*5*time.Minute))

	daily := []domain.Candle{{Time: day.AddDate(0, 0, -1), High: 101, Low: 99.8, Close: 100}}

	p := DefaultParams()
	p.HTFTrendEMA = 0 // isolate the entry setup from the HTF filter in this fixture
	binding := NewWithParams("TEST", p)
	cfg := domain.Config{InitialCash: 100000, Fraction: 1.0, Commission: 0, Lot: 1}
	res := domain.Run(binding, candles, daily, nil, cfg)

	if len(res.Trades) != 1 {
		t.Fatalf("trades = %d, want 1", len(res.Trades))
	}
	tr := res.Trades[0]
	if tr.Reason != "TP" {
		t.Fatalf("exit reason = %q, want TP", tr.Reason)
	}
	if tr.EntryPrice != 101.5 {
		t.Fatalf("entry = %.4f, want 101.5", tr.EntryPrice)
	}
	if tr.PnL <= 0 {
		t.Fatalf("PnL = %.4f, want > 0", tr.PnL)
	}
	_ = enum.Minutes5 // interval used at the CLI layer; asserted here only for import wiring
}
```

- [ ] **Step 2: Run test to verify it fails, then passes**

Run: `go test ./internal/service/trading_strategy/daylow/strategy/core/ -run TestDaylowRoundTripThroughEngine -v`
Expected: PASS (the strategy is already implemented; this test verifies engine integration). If it FAILS, debug with `superpowers:systematic-debugging` before proceeding — a failure here means entry/exit wiring disagrees with the engine's fill model.

- [ ] **Step 3: Commit**

```bash
git add internal/service/trading_strategy/daylow/strategy/core/core_test.go
git commit -m "test(daylow): end-to-end round-trip through the backtest engine"
```

---

### Task 10: Grid JSON, docs, and validation run

**Files:**
- Create: `data/params/daylow/daylow_grid.json`
- Create: `docs/daylow/strategy.md`
- (No unit test — this task produces the calibration artifact and the real backtest run.)

**Interfaces:**
- Consumes: everything above, through `cmd/backtest`.

- [ ] **Step 1: Create the calibration grid**

`data/params/daylow/daylow_grid.json` (phased grid; keep combos modest — 5m history is large):

```json
{
  "_comment": "daylow phased grid (2026-07-21). Phase 1 sweeps the setup geometry; phase 2 the risk/exit; phase 3 the session window and HTF filter. Run with rolling walk-forward (-train-months/-test-months) or a single -test-months holdout. Judge on pooled OOS PF, not a single calibration.",
  "phases": [
    {
      "name": "setup",
      "keepTop": 6,
      "grid": {
        "LowProximityPct": [0.003, 0.005, 0.008],
        "ConsolBars": [4, 6, 8],
        "ConsolRangeATR": [0.8, 1.0, 1.3],
        "ImpulseBodyATR": [0.6, 0.8, 1.0],
        "VolMult": [1.2, 1.5, 2.0]
      }
    },
    {
      "name": "risk",
      "keepTop": 6,
      "grid": {
        "StopBufferATR": [0.1, 0.25, 0.5],
        "RR": [1.5, 2.0, 2.5]
      }
    },
    {
      "name": "context",
      "grid": {
        "HTFTrendEMA": [0, 50],
        "EntrySessionStartMin": [540, 600],
        "EntrySessionEndMin": [780, 840, 1080],
        "CloseAtDayEnd": [0, 1]
      }
    }
  ]
}
```

- [ ] **Step 2: Write the strategy doc**

Create `docs/daylow/strategy.md` summarizing: the setup (prior-day-low consolidation breakout), every `Params` field with its default and meaning (copy the §4 table from the spec `docs/superpowers/specs/2026-07-21-daylow-consolidation-breakout-design.md`), the exit precedence (SL→TP→EOD), the session window semantics (entries gated, exits unrestricted), and the exact validation command below. Keep it concise (one screen).

- [ ] **Step 3: Full quality gate**

Run: `./bin/mage ci`
Expected: lint clean, `go test -race ./...` PASS, mock-drift clean. Fix anything that fails before proceeding.

- [ ] **Step 4: Real calibration + honest OOS run**

Pick one liquid ticker with intraday history (e.g. NVTK). Run:

```bash
go run ./cmd/backtest -ticker NVTK -strategy daylow -interval Minutes5 -htf-interval Hour1 \
  -calibrate data/params/daylow/daylow_grid.json -out ./reports/NVTK \
  -months 12 -test-months 3 -min-trades 20 -metric profit_factor -risk-pct 1
```

Read `reports/NVTK/..._best.md` and `..._calibration.md`. Record in the doc: pooled/OOS PF, trade count, whether OOS PF clears the same bar the prior failed strategies did **not** (smc/momentum/levels/ORB). If 5m history is too shallow (the run warns "not enough candles"), reduce `-months` or note the data limit.

- [ ] **Step 5: Commit the artifacts**

```bash
git add data/params/daylow/daylow_grid.json docs/daylow/strategy.md
git commit -m "docs(daylow): calibration grid, strategy reference, validation run"
```

- [ ] **Step 6: Report the validation verdict**

Summarize the OOS result to the user. If OOS PF fails the bar, STOP — do not add per-ticker configs or a live layer (both explicitly deferred in the spec §9). The go/no-go on live promotion is the user's call.

---

## Notes on deferred scope (from spec §9)

Not in this plan — revisit only after a confirmed OOS edge: per-ticker `daylow/strategy/<ticker>` configs; the live layer (runner/scheduler/notification); a stop-out cooldown / re-entry guard; a second (pre-close) active-hours window. `Opens` on `MarketData` is populated only by the backtest engine here; a future live `daylow` runner must populate it too.
