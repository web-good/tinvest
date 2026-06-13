# Reversion optional vertical-volume entry filter — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add an optional entry gate to the reversion strategy that blocks a buy when the entry bar's volume is below the average volume of the preceding bars, excluding weekend (Sat/Sun) bars from that average.

**Architecture:** Three new int/float `core.Params` (`UseVolume`, `VolAvgPeriod`, `VolMult`) follow the existing `UseTrend`/`UseATRStop` optional-filter pattern. Weekend detection needs per-bar timestamps, so `strategy.MarketData` gains an additive `Times []time.Time` field populated by the backtest engine. The core computes the average-volume baseline in `buildInput` and applies the gate in `decide()`; the filter degrades gracefully (never blocks) when timestamps or volumes are missing.

**Tech Stack:** Go 1.25; existing reversion `core` package; backtest engine; reflection-based grid calibration.

**Spec:** `docs/superpowers/specs/2026-06-13-reversion-optional-volume-filter-design.md`

---

### Task 1: Add `Times` to MarketData and populate it in the engine

**Files:**
- Modify: `internal/service/trading_strategy/scalping/strategy/strategy.go`
- Modify: `internal/domain/backtest/engine.go:216-233` (`buildMarketData`)
- Test: `internal/domain/backtest/engine_test.go`

- [ ] **Step 1: Write the failing test**

Add to `internal/domain/backtest/engine_test.go`:

```go
func TestBuildMarketDataCopiesTimes(t *testing.T) {
	base := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	window := []Candle{
		{Time: base, Close: 10, High: 10, Low: 10, Volume: 1},
		{Time: base.Add(time.Hour), Close: 11, High: 11, Low: 11, Volume: 2},
		{Time: base.Add(2 * time.Hour), Close: 12, High: 12, Low: 12, Volume: 3},
	}
	md := buildMarketData(window)
	if len(md.Times) != len(window) {
		t.Fatalf("Times length: want %d, got %d", len(window), len(md.Times))
	}
	for i := range window {
		if !md.Times[i].Equal(window[i].Time) {
			t.Fatalf("Times[%d]: want %v, got %v", i, window[i].Time, md.Times[i])
		}
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/domain/backtest/ -run TestBuildMarketDataCopiesTimes -v`
Expected: FAIL — `md.Times` is empty (field does not exist / not populated).

- [ ] **Step 3: Add the field to MarketData**

In `internal/service/trading_strategy/scalping/strategy/strategy.go`, add an import for `time` and a `Times` field to `MarketData`. The file currently imports only the model package; change the import block to:

```go
import (
	"time"

	"tinvest/internal/service/trading_strategy/scalping/model"
)
```

Add the field right after `Volumes []int64` in the `MarketData` struct:

```go
	Volumes []int64
	// Times are oldest-first bar open-times, index-aligned to Closes/Volumes. Empty when
	// the runner does not supply them (e.g. live trading); consumers must degrade
	// gracefully (the reversion volume filter skips weekend exclusion when Times is empty).
	Times []time.Time
```

- [ ] **Step 4: Populate it in buildMarketData**

In `internal/domain/backtest/engine.go`, update `buildMarketData` to allocate and fill `Times`:

```go
func buildMarketData(window []Candle) strategy.MarketData {
	md := strategy.MarketData{
		Highs:   make([]float64, len(window)),
		Lows:    make([]float64, len(window)),
		Closes:  make([]float64, len(window)),
		Volumes: make([]int64, len(window)),
		Times:   make([]time.Time, len(window)),
	}
	for i, c := range window {
		md.Highs[i] = c.High
		md.Lows[i] = c.Low
		md.Closes[i] = c.Close
		md.Volumes[i] = c.Volume
		md.Times[i] = c.Time
	}
	if n := len(window); n > 0 {
		md.Price = window[n-1].Close
	}
	return md
}
```

(`time` is already imported in `engine.go`.)

- [ ] **Step 5: Run test to verify it passes**

Run: `go test ./internal/domain/backtest/ -run TestBuildMarketDataCopiesTimes -v`
Expected: PASS.

- [ ] **Step 6: Verify nothing else broke**

Run: `go build ./... && go test ./internal/domain/backtest/ ./internal/service/trading_strategy/...`
Expected: build clean, all tests PASS.

- [ ] **Step 7: Commit**

```bash
git add internal/service/trading_strategy/scalping/strategy/strategy.go internal/domain/backtest/engine.go internal/domain/backtest/engine_test.go
git commit -m "feat(backtest): carry per-bar Times into MarketData"
```

---

### Task 2: Add Params fields, weekday helper, and the average-volume baseline

**Files:**
- Modify: `internal/service/trading_strategy/reversion/strategy/core/core.go`
- Test: `internal/service/trading_strategy/reversion/strategy/core/core_test.go`

- [ ] **Step 1: Write the failing tests**

Add to `internal/service/trading_strategy/reversion/strategy/core/core_test.go`. (The `time` package must be imported in the test file; add it to the import block.)

```go
// mskDay builds noon of a specific MSK calendar date for weekend-aware tests.
func mskDay(y int, m time.Month, d int) time.Time {
	return time.Date(y, m, d, 12, 0, 0, 0, mskLoc)
}

func TestAverageVolumeExcludesWeekends(t *testing.T) {
	// Jun 13/14 2026 are Sat/Sun (MSK); Jun 15 is the entry bar (excluded from its avg).
	vols := []int64{100, 200, 300, 9999, 9999, 50}
	times := []time.Time{
		mskDay(2026, 6, 10), // Wed
		mskDay(2026, 6, 11), // Thu
		mskDay(2026, 6, 12), // Fri
		mskDay(2026, 6, 13), // Sat (excluded)
		mskDay(2026, 6, 14), // Sun (excluded)
		mskDay(2026, 6, 15), // Mon (entry bar, excluded from its own average)
	}
	avg, ok := averageVolumeExcludingWeekends(vols, times, 5)
	if !ok {
		t.Fatalf("want ok=true")
	}
	if avg != 200 { // mean(100,200,300)
		t.Fatalf("want avg 200 (weekends + entry bar excluded), got %v", avg)
	}
}

func TestAverageVolumeNoTimesKeepsAllBars(t *testing.T) {
	vols := []int64{100, 200, 300, 9999, 9999, 50}
	avg, ok := averageVolumeExcludingWeekends(vols, nil, 5)
	if !ok {
		t.Fatalf("want ok=true")
	}
	want := float64(100+200+300+9999+9999) / 5 // entry bar still excluded; no weekend drop
	if avg != want {
		t.Fatalf("no-times: want %v, got %v", want, avg)
	}
}

func TestAverageVolumeNoSamplesNotOK(t *testing.T) {
	// Window is entirely weekend bars -> no surviving sample.
	vols := []int64{9999, 9999, 50}
	times := []time.Time{
		mskDay(2026, 6, 13), // Sat
		mskDay(2026, 6, 14), // Sun
		mskDay(2026, 6, 15), // Mon (entry)
	}
	if _, ok := averageVolumeExcludingWeekends(vols, times, 2); ok {
		t.Fatalf("all-weekend window: want ok=false")
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/service/trading_strategy/reversion/strategy/core/ -run TestAverageVolume -v`
Expected: FAIL — `mskLoc` and `averageVolumeExcludingWeekends` are undefined.

- [ ] **Step 3: Add the Params fields**

In `core.go`, append three fields to the `Params` struct after `StopATRMult`:

```go
	StopATRMult   float64 // ATRSL distance: stop = PurchasePrice - StopATRMult*EntryATR (default 1.0)
	UseVolume     int     // 0 = no volume filter; 1 = block entries below the average bar volume
	VolAvgPeriod  int     // preceding-bar window for the average-volume baseline; consulted only when UseVolume=1
	VolMult       float64 // entry requires entryVolume >= avg*VolMult (default 1.0)
```

- [ ] **Step 4: Add mskLoc, isWeekend, and the averaging helper**

Add a `time` import to `core.go`'s import block. Then add, near the top-level helpers (e.g. after `crossDown`):

```go
// mskLoc anchors weekend detection to the Moscow trading calendar (UTC fallback if the
// tz DB is absent), mirroring the backtest engine. Weekend trading sessions (Sat/Sun) on
// MOEX trade at much lower volume and are excluded from the average-volume baseline.
var mskLoc = func() *time.Location {
	loc, err := time.LoadLocation("Europe/Moscow")
	if err != nil {
		return time.UTC
	}
	return loc
}()

// isWeekend reports whether t falls on Saturday or Sunday in mskLoc.
func isWeekend(t time.Time) bool {
	wd := t.In(mskLoc).Weekday()
	return wd == time.Saturday || wd == time.Sunday
}

// averageVolumeExcludingWeekends averages the volumes of the `period` bars that PRECEDE
// the final (entry) bar of vols. The entry bar is never part of its own average. When
// times is supplied and index-aligned to vols, weekend bars (Sat/Sun MSK) are dropped;
// when times is empty or misaligned, weekend exclusion is skipped (all preceding bars
// count). Non-positive volumes are ignored. ok is false when no sample survives — the
// caller must then skip the gate (never block an entry on missing data).
func averageVolumeExcludingWeekends(vols []int64, times []time.Time, period int) (avg float64, ok bool) {
	n := len(vols)
	if n < 2 || period <= 0 {
		return 0, false
	}
	lo := n - 1 - period // window = the `period` bars before the entry bar: [lo, n-1)
	if lo < 0 {
		lo = 0
	}
	haveTimes := len(times) == n
	var sum float64
	var count int
	for j := lo; j < n-1; j++ {
		if haveTimes && isWeekend(times[j]) {
			continue
		}
		if vols[j] <= 0 {
			continue
		}
		sum += float64(vols[j])
		count++
	}
	if count == 0 {
		return 0, false
	}
	return sum / float64(count), true
}
```

- [ ] **Step 5: Run tests to verify they pass**

Run: `go test ./internal/service/trading_strategy/reversion/strategy/core/ -run TestAverageVolume -v`
Expected: PASS.

- [ ] **Step 6: Run the package suite**

Run: `go test ./internal/service/trading_strategy/reversion/...`
Expected: PASS (existing tests still green; new Params fields default to zero and are unused yet).

- [ ] **Step 7: Commit**

```bash
git add internal/service/trading_strategy/reversion/strategy/core/core.go internal/service/trading_strategy/reversion/strategy/core/core_test.go
git commit -m "feat(reversion): add volume-filter params and weekend-aware average"
```

---

### Task 3: Wire the gate inputs into decideInput / buildInput and Lookback

**Files:**
- Modify: `internal/service/trading_strategy/reversion/strategy/core/core.go`
- Test: `internal/service/trading_strategy/reversion/strategy/core/core_test.go`

- [ ] **Step 1: Write the failing tests**

Add to `core_test.go`:

```go
func TestBuildInputVolumeGate(t *testing.T) {
	n := 60
	highs := make([]float64, n)
	lows := make([]float64, n)
	closes := make([]float64, n)
	vols := make([]int64, n)
	times := make([]time.Time, n)
	base := mskDay(2026, 1, 1)
	for i := 0; i < n; i++ {
		closes[i] = 100 + float64(i)
		highs[i] = closes[i] + 1
		lows[i] = closes[i] - 1
		vols[i] = 1000
		times[i] = base.AddDate(0, 0, i)
	}
	md := strategy.MarketData{Price: closes[n-1], Highs: highs, Lows: lows, Closes: closes, Volumes: vols, Times: times}

	p := defaultParams()
	p.VolAvgPeriod = 20

	p.UseVolume = 1
	if in := NewWithParams("T", p).buildInput(md); !in.volOK || in.avgVol <= 0 || in.entryVol != 1000 {
		t.Fatalf("UseVolume=1: want volOK, avgVol>0, entryVol=1000; got volOK=%v avg=%v entry=%v", in.volOK, in.avgVol, in.entryVol)
	}

	p.UseVolume = 0
	if in := NewWithParams("T", p).buildInput(md); in.volOK || in.avgVol != 0 || in.entryVol != 0 {
		t.Fatalf("UseVolume=0: volume inputs must stay zero, got volOK=%v avg=%v entry=%v", in.volOK, in.avgVol, in.entryVol)
	}
}

func TestLookbackIncludesVolumeWindow(t *testing.T) {
	p := defaultParams()
	p.FastEMA, p.SlowEMA = 50, 200
	p.RSIPeriod, p.StochKPeriod, p.StochDSmooth = 14, 14, 3
	p.VolAvgPeriod = 300 // dominates SlowEMA when the filter is on

	p.UseVolume = 1
	if got := NewWithParams("T", p).Lookback(); got != 306 {
		t.Fatalf("VolAvgPeriod=300 dominates: want 306, got %d", got)
	}

	p.UseVolume = 0
	if got := NewWithParams("T", p).Lookback(); got != 205 {
		t.Fatalf("UseVolume=0: VolAvgPeriod ignored, want SlowEMA+5=205, got %d", got)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/service/trading_strategy/reversion/strategy/core/ -run 'TestBuildInputVolumeGate|TestLookbackIncludesVolumeWindow' -v`
Expected: FAIL — `decideInput` has no `entryVol/avgVol/volOK`; Lookback ignores `VolAvgPeriod`.

- [ ] **Step 3: Add fields to decideInput**

In `core.go`, add to the `decideInput` struct after `atr`:

```go
	atr         float64 // daily ATR over the window (0 unless UseATRStop=1 and ATRPeriod>0); stamped onto sig.ATR at entry to freeze EntryATR
	entryVol    float64 // entry (latest) bar's volume; 0 unless UseVolume=1 and a baseline was computed
	avgVol      float64 // average volume of the preceding VolAvgPeriod bars (weekends excluded); 0 unless gate active
	volOK       bool    // true when the volume baseline could be computed; false -> gate is skipped
	pos         *strategy.Position
```

- [ ] **Step 4: Compute them in buildInput**

In `buildInput`, after the `atr` block and before the `return decideInput{...}`, add:

```go
	var entryVol, avgVol float64
	volOK := false
	if s.p.UseVolume == 1 && s.p.VolAvgPeriod > 0 {
		if n := len(md.Volumes); n > 0 && md.Volumes[n-1] > 0 {
			if a, ok := averageVolumeExcludingWeekends(md.Volumes, md.Times, s.p.VolAvgPeriod); ok {
				entryVol = float64(md.Volumes[n-1])
				avgVol = a
				volOK = true
			}
		}
	}
```

Then add the three fields to the returned `decideInput` literal (after `atr: atr,`):

```go
		atr:         atr,
		entryVol:    entryVol,
		avgVol:      avgVol,
		volOK:       volOK,
		pos:         md.Position,
```

- [ ] **Step 5: Extend Lookback**

In `Lookback`, add a candidate under the same gate style as the ATR branch:

```go
	if s.p.UseATRStop == 1 && s.p.ATRPeriod > 0 {
		cands = append(cands, s.p.ATRPeriod+1)
	}
	if s.p.UseVolume == 1 && s.p.VolAvgPeriod > 0 {
		cands = append(cands, s.p.VolAvgPeriod+1)
	}
```

- [ ] **Step 6: Run tests to verify they pass**

Run: `go test ./internal/service/trading_strategy/reversion/strategy/core/ -run 'TestBuildInputVolumeGate|TestLookbackIncludesVolumeWindow' -v`
Expected: PASS.

- [ ] **Step 7: Run the package suite**

Run: `go test ./internal/service/trading_strategy/reversion/...`
Expected: PASS.

- [ ] **Step 8: Commit**

```bash
git add internal/service/trading_strategy/reversion/strategy/core/core.go internal/service/trading_strategy/reversion/strategy/core/core_test.go
git commit -m "feat(reversion): plumb volume-gate inputs and Lookback"
```

---

### Task 4: Apply the gate in decide() and report it in Explain()

**Files:**
- Modify: `internal/service/trading_strategy/reversion/strategy/core/core.go`
- Test: `internal/service/trading_strategy/reversion/strategy/core/core_test.go`

- [ ] **Step 1: Write the failing tests**

Add to `core_test.go`. (`passingInput()` clears every oscillator/trend gate; here we set the volume fields directly to exercise the new gate.)

```go
func TestVolumeGateBlocksBelowAverage(t *testing.T) {
	p := defaultParams()
	p.UseVolume = 1
	p.VolMult = 1.0
	s := NewWithParams("T", p)

	in := passingInput()
	in.volOK = true
	in.avgVol = 1000
	in.entryVol = 800 // below average -> blocked
	if sig := s.decide(in); sig.Kind == model.SignalBuy {
		t.Fatalf("entry volume below average: want no Buy")
	}
}

func TestVolumeGateAllowsAtOrAboveAverage(t *testing.T) {
	p := defaultParams()
	p.UseVolume = 1
	p.VolMult = 1.0
	s := NewWithParams("T", p)

	in := passingInput()
	in.volOK = true
	in.avgVol = 1000
	in.entryVol = 1000 // exactly at average -> allowed
	if sig := s.decide(in); sig.Kind != model.SignalBuy {
		t.Fatalf("entry volume at average: want Buy, got %v", sig.Kind)
	}
}

func TestVolumeGateMultiplierRaisesBar(t *testing.T) {
	p := defaultParams()
	p.UseVolume = 1
	p.VolMult = 1.5
	s := NewWithParams("T", p)

	in := passingInput()
	in.volOK = true
	in.avgVol = 1000
	in.entryVol = 1200 // passes at 1.0, fails at 1.5 (threshold 1500)
	if sig := s.decide(in); sig.Kind == model.SignalBuy {
		t.Fatalf("VolMult=1.5: 1200 < 1500 threshold, want no Buy")
	}
}

func TestVolumeGateOffIgnoresVolume(t *testing.T) {
	p := defaultParams() // UseVolume defaults to 0
	s := NewWithParams("T", p)

	in := passingInput()
	in.volOK = true
	in.avgVol = 1000
	in.entryVol = 1 // would be blocked if the gate were on
	if sig := s.decide(in); sig.Kind != model.SignalBuy {
		t.Fatalf("UseVolume=0: volume must be ignored, want Buy, got %v", sig.Kind)
	}
}

func TestVolumeGateSkippedWhenNotOK(t *testing.T) {
	p := defaultParams()
	p.UseVolume = 1
	s := NewWithParams("T", p)

	in := passingInput()
	in.volOK = false // baseline unavailable -> gate must not block
	in.avgVol = 0
	in.entryVol = 0
	if sig := s.decide(in); sig.Kind != model.SignalBuy {
		t.Fatalf("volOK=false: gate must be skipped, want Buy, got %v", sig.Kind)
	}
}

func TestExplainBlocksOnVolume(t *testing.T) {
	p := defaultParams()
	p.UseVolume = 1
	p.VolMult = 1.0
	s := NewWithParams("T", p)

	in := passingInput()
	in.volOK = true
	in.avgVol = 1000
	in.entryVol = 800
	out := s.explainFrom(in)
	if !strings.Contains(out, "Объём") || !strings.Contains(out, "ВХОДА НЕТ") {
		t.Fatalf("Explain should block on volume, got: %q", out)
	}
}
```

NOTE: `explainFrom` is a small testable seam introduced in Step 3 so the volume gate (which reads `decideInput` fields, not raw `md`) can be exercised without constructing a full candle series. If the implementer prefers, the existing `Explain(md)` may instead be tested by building an `md` whose volumes force a block — but the seam keeps the test focused and deterministic.

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/service/trading_strategy/reversion/strategy/core/ -run 'TestVolumeGate|TestExplainBlocksOnVolume' -v`
Expected: FAIL — gate not applied; `explainFrom` undefined.

- [ ] **Step 3: Apply the gate in decide() and refactor Explain to take decideInput**

In `decide()`, add the volume gate in the entry path, after the dual-confirmation check and before `sig.Kind = model.SignalBuy`:

```go
	// 2. Dual oversold confirmation.
	if !s.entryFired(in) {
		return sig
	}
	// 3. Optional volume filter: block a buy on a below-average-volume bar.
	if s.p.UseVolume == 1 && in.volOK && in.entryVol < in.avgVol*s.p.VolMult {
		return sig
	}

	sig.Kind = model.SignalBuy
```

Refactor `Explain` to delegate to a new `explainFrom(in decideInput)` so the gate logic is testable from a `decideInput`. Replace the current `Explain` body's `in := s.buildInput(md)` + logic with:

```go
// Explain re-runs the entry gates over md and reports each gate's value and verdict
// (✓ pass / ✗ block) in entry order, stopping at the first blocker. Diagnostic only.
func (s *Strategy) Explain(md strategy.MarketData) string {
	return s.explainFrom(s.buildInput(md))
}

func (s *Strategy) explainFrom(in decideInput) string {
	if in.pos != nil {
		return "позиция уже открыта — вход не рассматривается"
	}

	var b strings.Builder
	pass := func(format string, args ...any) { fmt.Fprintf(&b, "✓ "+format+"\n", args...) }
	block := func(format string, args ...any) string {
		fmt.Fprintf(&b, "✗ "+format+"\n", args...)
		fmt.Fprintf(&b, "→ ВХОДА НЕТ: заблокировал этот фильтр")
		return b.String()
	}

	// 1. Optional trend filter.
	if s.p.UseTrend == 1 {
		if !uptrend(in) {
			return block("Тренд: нужно EMA%d > EMA%d и close > EMA%d (EMA%d=%.4f, EMA%d=%.4f, close=%.4f)",
				s.p.FastEMA, s.p.SlowEMA, s.p.SlowEMA, s.p.FastEMA, in.emaFast, s.p.SlowEMA, in.emaSlow, in.price)
		}
		pass("Тренд↑: EMA%d %.4f > EMA%d %.4f, close %.4f > EMA%d", s.p.FastEMA, in.emaFast, s.p.SlowEMA, in.emaSlow, in.price, s.p.SlowEMA)
	}

	// 2. Dual oversold confirmation.
	if !s.entryFired(in) {
		return block("Двойное подтверждение: нет (RSI(%d) %.2f→%.2f зона<%.0f; Stoch%%D %.2f→%.2f зона<%.0f) — нужен кросс одного в зону при другом уже в зоне",
			s.p.RSIPeriod, in.rsiPrev, in.rsiNow, s.p.RSIOversold, in.stochPrev, in.stochNow, s.p.StochOversold)
	}
	pass("Двойное подтверждение: RSI(%d) %.2f→%.2f + Stoch%%D %.2f→%.2f в зоне перепроданности",
		s.p.RSIPeriod, in.rsiPrev, in.rsiNow, in.stochPrev, in.stochNow)

	// 3. Optional volume filter.
	if s.p.UseVolume == 1 {
		switch {
		case in.volOK && in.entryVol < in.avgVol*s.p.VolMult:
			return block("Объём: бар входа %.0f < порога %.0f (среднее %.0f × %.2g, бары выходных исключены)",
				in.entryVol, in.avgVol*s.p.VolMult, in.avgVol, s.p.VolMult)
		case in.volOK:
			pass("Объём: бар входа %.0f ≥ порога %.0f (среднее %.0f × %.2g)",
				in.entryVol, in.avgVol*s.p.VolMult, in.avgVol, s.p.VolMult)
		default:
			pass("Объём: фильтр включён, но базу не посчитать (нет данных) — пропуск")
		}
	}

	fmt.Fprintf(&b, "→ ВХОД: все фильтры пройдены, должна быть покупка")
	return b.String()
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/service/trading_strategy/reversion/strategy/core/ -run 'TestVolumeGate|TestExplainBlocksOnVolume' -v`
Expected: PASS.

- [ ] **Step 5: Run the full reversion suite**

Run: `go test ./internal/service/trading_strategy/reversion/...`
Expected: PASS (including the existing `TestExplainBlocksOnTrend`, `TestExplainTrendOffSkipsGate`, `TestExplainPositionOpen`).

- [ ] **Step 6: Commit**

```bash
git add internal/service/trading_strategy/reversion/strategy/core/core.go internal/service/trading_strategy/reversion/strategy/core/core_test.go
git commit -m "feat(reversion): apply optional volume gate and report it in Explain"
```

---

### Task 5: Per-ticker defaults and calibration grids

**Files:**
- Modify: `internal/service/trading_strategy/reversion/strategy/{afks,rusal,gazp,ydex,mdmg,nvtk,plzl,sber}/*.go`
- Modify: `data/params/{afks,rual,gazp,ydex,mdmg,nvtk,plzl,sber}/reversion_grid.json`

- [ ] **Step 1: Add the three fields to all 8 DefaultParams**

For each per-ticker package file, append `UseVolume: 0, VolAvgPeriod: 20, VolMult: 1.0,` as a new line after the `UseATRStop: ...` line. Example for `afks/afks.go`:

```go
func DefaultParams() core.Params {
	return core.Params{
		UseTrend: 1, FastEMA: 10, SlowEMA: 200,
		RSIPeriod: 5, RSIOversold: 30,
		StochKPeriod: 14, StochDSmooth: 1, StochOversold: 20,
		UseATRStop: 0, ATRPeriod: 14, StopATRMult: 1.0,
		UseVolume: 0, VolAvgPeriod: 20, VolMult: 1.0,
	}
}
```

Apply the identical added line to: `rusal/rusal.go`, `gazp/gazp.go`, `ydex/ydex.go`, `mdmg/mdmg.go`, `nvtk/nvtk.go`, `plzl/plzl.go`, `sber/sber.go`. Only that one line is added — the existing values per ticker stay unchanged.

- [ ] **Step 2: Verify it builds**

Run: `go build ./...`
Expected: clean (the new struct fields exist from Task 2).

- [ ] **Step 3: Add the three keys to all 8 grids**

For each `data/params/<ticker>/reversion_grid.json`, add three keys to the `entry` phase `grid` object, after the `"StopATRMult"` line. Add a trailing comma to the current last key (`"StopATRMult"`) and append:

```json
        "StopATRMult": [1.0, 1.5, 2.0],
        "UseVolume": [0, 1],
        "VolAvgPeriod": [20],
        "VolMult": [1.0, 1.5]
```

Apply to all 8 folders: `afks`, `rual`, `gazp`, `ydex`, `mdmg`, `nvtk`, `plzl`, `sber`.

- [ ] **Step 4: Validate every grid is well-formed JSON**

Run:
```bash
for t in afks rual gazp ydex mdmg nvtk plzl sber; do
  python3 -m json.tool "data/params/$t/reversion_grid.json" >/dev/null && echo "$t ok" || echo "$t BAD"
done
```
Expected: all 8 print `<ticker> ok`.

- [ ] **Step 5: Smoke-check that calibration parses the new keys (one ticker)**

Run:
```bash
go run ./cmd/backtest -ticker SBER -strategy reversion -interval Day1 \
  -calibrate data/params/sber/reversion_grid.json -out ./reports/SBER \
  -months 24 -test-months 6 -min-trades 20 -metric profit_factor 2>&1 | tail -20
```
Expected: calibration runs and reports a best combination including `UseVolume/VolAvgPeriod/VolMult` (no "unknown field" / reflection error). It is fine if PF is unimpressive — this only confirms the grid is consumed.

- [ ] **Step 6: Commit**

```bash
git add internal/service/trading_strategy/reversion/strategy data/params
git commit -m "feat(reversion): default volume filter off and sweep it in grids"
```

---

### Task 6: Documentation

**Files:**
- Modify: `internal/service/trading_strategy/reversion/strategy/core/core.go` (package doc)
- Modify: `docs/reversion/strategy.md`

- [ ] **Step 1: Update the core package docstring**

In `core.go`, extend the entry-side description in the package comment. After the sentence describing the trend filter ("An optional trend filter restricts buys to a confirmed uptrend."), add:

```go
// An optional volume filter (UseVolume) additionally blocks a buy when the entry bar's
// volume is below the average of the preceding VolAvgPeriod bars (weekend Sat/Sun bars
// excluded), scaled by VolMult.
```

- [ ] **Step 2: Update the Entry section in docs/reversion/strategy.md**

In `docs/reversion/strategy.md`, add a third entry gate after the "Dual oversold confirmation" item (item 2), in the `## Entry (gates in short-circuit order)` list:

```markdown
3. **Volume filter (optional, `UseVolume`):** `0` (default) ignores volume; `1` blocks
   the buy when the entry bar's volume is below `avg × VolMult`, where `avg` is the mean
   volume of the preceding `VolAvgPeriod` bars with weekend (Sat/Sun MSK) bars excluded.
   The gate is skipped (entry allowed) when the baseline cannot be computed — no
   timestamps means weekends are not excluded; no valid samples means no block.
```

- [ ] **Step 3: Update the Params section in docs/reversion/strategy.md**

Change the params name list (the backtick block) to include the new fields, and update the flag note:

```markdown
`UseTrend, FastEMA, SlowEMA, RSIPeriod, RSIOversold, StochKPeriod, StochDSmooth,
StochOversold, UseATRStop, ATRPeriod, StopATRMult, UseVolume, VolAvgPeriod, VolMult`.
Flags (`UseTrend`, `UseATRStop`, `UseVolume`) are int `0/1`; the rest are int/float64 so
the grid calibrator can sweep them. The RSI-50 exit level is a fixed constant, not a param.
```

Then add three bullets after the `StopATRMult` bullet:

```markdown
- `UseVolume` — `0` (default): no volume filter; `1`: block entries on below-average
  volume bars.
- `VolAvgPeriod` — number of preceding bars averaged for the volume baseline (default
  `20`); weekend (Sat/Sun) bars are excluded from the average.
- `VolMult` — entry-volume threshold multiplier: a buy needs `entryVolume ≥ avg ×
  VolMult`; default `1.0` (strictly at/above average).
```

- [ ] **Step 4: Verify build + full repo tests**

Run: `go build ./... && go test ./...`
Expected: build clean, all tests PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/service/trading_strategy/reversion/strategy/core/core.go docs/reversion/strategy.md
git commit -m "docs(reversion): document optional vertical-volume entry filter"
```

---

## Final verification (after all tasks)

- [ ] `go build ./...` — clean
- [ ] `go vet ./...` — clean
- [ ] `go test ./...` — all green
- [ ] Spot-check: `go run ./cmd/backtest -ticker SBER -strategy reversion -interval Day1 -explain '<a recent bar ts>' -months 12` with a temporarily volume-on SBER (or rely on the calibration smoke from Task 5) shows the "Объём" line in the Explain output.
