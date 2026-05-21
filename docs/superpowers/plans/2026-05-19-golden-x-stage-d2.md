# Golden X Stage D2 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Extract all Golden X strategy knobs (buy/sell percentiles, ATR stop params, volume confirmation, history windows) out of unexported package-level `const`s and into a single `dto.Settings` value-type, threaded through `Detect` as a 5th argument. No behavioral change.

**Architecture:** New `dto.Settings` (14 fields, flat) acts as a value-type bag. `golden_x.DefaultSettings()` returns today's hardcoded values. Production wiring (`service.Trade`) and `cmd/backtest` both call `DefaultSettings()` once and pass it through. `Detect`'s body and helpers consume settings instead of consts; the consts themselves are deleted at the end. Byte-parity with pre-D2 is the explicit contract.

**Tech Stack:** Go 1.25, standard library only. Spec at `docs/superpowers/specs/2026-05-19-golden-x-stage-d2-design.md`. Working branch: `feature/golden-x-stage-a`.

---

## File Structure

**New files:**
- `internal/service/trading_strategy/golden_x/dto/settings.go` — `Settings` struct (14 fields)
- `internal/service/trading_strategy/golden_x/settings.go` — `DefaultSettings()` constructor
- `internal/service/trading_strategy/golden_x/settings_test.go` — `TestDefaultSettings` regression guard

**Modified files:**
- `internal/service/trading_strategy/golden_x/detector.go` — `Detect` gains a 5th `settings` arg, body consumes `settings.*` everywhere
- `internal/service/trading_strategy/golden_x/percentile.go` — `adaptiveThresholds` / `adaptiveSellThresholds` take percentile args explicitly
- `internal/service/trading_strategy/golden_x/percentile_test.go` — update existing test calls to pass percentile args
- `internal/service/trading_strategy/golden_x/stop.go` — `kForKind` takes `Settings`
- `internal/service/trading_strategy/golden_x/stop_test.go` — update `TestKForKind` to pass `DefaultSettings()`
- `internal/service/trading_strategy/golden_x/trade.go` — `adaptiveRSIForShare` no longer returns `Thresholds`, takes window-bound args; delete 8 strategy-knob `const`s; `Trade()` builds `settings := DefaultSettings()` and threads into `Detect`
- `internal/service/trading_strategy/golden_x/trend_filter.go` — delete `trendEMAPeriod` const
- `internal/service/trading_strategy/golden_x/detector_test.go` — update Detect calls to pass `DefaultSettings()` as 5th arg
- `internal/service/trading_strategy/golden_x/backtest/replay.go` — widen `DetectFunc` signature, add `Settings` field to `ReplayConfig`, thread `cfg.Settings` into `detect()` call
- `internal/service/trading_strategy/golden_x/backtest/replay_test.go` — fake `DetectFunc`s take 5th arg
- `cmd/backtest/main.go` — build `golden_x.DefaultSettings()` once, set `ReplayConfig.Settings`, widen the closure passed to `backtest.Replay`

**Untouched** (intentionally): `cache.go`, `report.go`, `position.go`, `divergence.go`, `rsi.go`, `candle.go`, `dedup.go`, `notification/`, `shares/`. `candleLookbackWeeks` and `divergenceFractalK` consts stay in `trade.go` per spec (out of scope for D2).

---

## Task ordering rationale

Each task ends in a clean build + green tests. The order is chosen so the build stays green at every checkpoint:

1. Add `dto.Settings` (new file, zero callers) → green
2. Add `DefaultSettings()` + regression test → green
3. Add 5th `settings` arg to `Detect` and cascade signatures (`DetectFunc`, `ReplayConfig`, all callers, all tests). Body of `Detect` ignores `settings` for now → green, behavior unchanged
4. Wire `settings.*` into `percentile.go` helpers and `adaptiveRSIForShare` → green
5. Wire `settings.*` into `kForKind` → green
6. Wire `settings.*` into remaining `Detect` internals; delete 9 unused consts (`adaptiveWindowMin/Max`, `divergenceLookbackWeeks`, `volumeSMALookback`, `volumeMultiplier`, `atrPeriod`, `atrMultiplierDividend/Growth` in `trade.go`; `trendEMAPeriod` in `trend_filter.go`) → green, byte-parity contract met

---

## Task 1: Add `dto.Settings` struct

**Files:**
- Create: `internal/service/trading_strategy/golden_x/dto/settings.go`

- [ ] **Step 1: Write the file**

```go
package dto

// Settings carries all tunable strategy knobs for Golden X. Each field has a
// well-defined default in golden_x.DefaultSettings(). Names use role-based
// terminology, not literal-value terminology, so the meaning survives retuning.
type Settings struct {
	// Buy-tier percentiles (RSI < BuyGreen → green; < BuyYellow → yellow).
	BuyGreen  float64 // default 5
	BuyYellow float64 // default 15

	// Sell-tier percentiles. Growth uses only SellOrange; Dividend uses all three.
	SellYellow float64 // default 80 (Dividend only)
	SellOrange float64 // default 90 (both kinds)
	SellRed    float64 // default 95 (Dividend only)

	// ATR-based stop.
	ATRPeriod             int     // default 14
	ATRMultiplierDividend float64 // default 2.0
	ATRMultiplierGrowth   float64 // default 1.5

	// Volume-confirmation indicator (last weekly volume > Multiplier × SMA of preceding Lookback weeks).
	VolumeSMALookback int     // default 20
	VolumeMultiplier  float64 // default 1.5

	// History windows.
	TrendEMAPeriod          int // default 200 (EMA200 W trend filter for Growth)
	AdaptiveWindowMin       int // default 100 (minimum closed-week RSI samples for adaptive tiers)
	AdaptiveWindowMax       int // default 200 (cap on closed-week RSI samples kept for percentiles)
	DivergenceLookbackWeeks int // default 52  (bullish-divergence pivot search horizon)
}
```

- [ ] **Step 2: Verify build is still clean**

Run: `cd /home/oleg/GolandProjects/tinvest && go build ./...`
Expected: exit code 0 (no callers reference `Settings` yet)

- [ ] **Step 3: Commit**

```bash
git add internal/service/trading_strategy/golden_x/dto/settings.go
git commit -m "feat(golden_x/dto): add Settings struct for strategy knobs"
```

---

## Task 2: Add `DefaultSettings()` + regression test

**Files:**
- Create: `internal/service/trading_strategy/golden_x/settings.go`
- Create: `internal/service/trading_strategy/golden_x/settings_test.go`

- [ ] **Step 1: Write the failing test**

Create `internal/service/trading_strategy/golden_x/settings_test.go`:

```go
package golden_x

import (
	"testing"

	"tinvest/internal/service/trading_strategy/golden_x/dto"
)

func TestDefaultSettings(t *testing.T) {
	s := DefaultSettings()
	want := dto.Settings{
		BuyGreen: 5, BuyYellow: 15,
		SellYellow: 80, SellOrange: 90, SellRed: 95,
		ATRPeriod: 14, ATRMultiplierDividend: 2.0, ATRMultiplierGrowth: 1.5,
		VolumeSMALookback: 20, VolumeMultiplier: 1.5,
		TrendEMAPeriod: 200, AdaptiveWindowMin: 100, AdaptiveWindowMax: 200,
		DivergenceLookbackWeeks: 52,
	}
	if s != want {
		t.Fatalf("DefaultSettings drift:\n got %+v\nwant %+v", s, want)
	}
}
```

- [ ] **Step 2: Run test to verify it fails (function not defined yet)**

Run: `cd /home/oleg/GolandProjects/tinvest && go test ./internal/service/trading_strategy/golden_x -run TestDefaultSettings`
Expected: build error `undefined: DefaultSettings`

- [ ] **Step 3: Write the implementation**

Create `internal/service/trading_strategy/golden_x/settings.go`:

```go
package golden_x

import "tinvest/internal/service/trading_strategy/golden_x/dto"

// DefaultSettings returns the strategy knobs in use across the codebase prior
// to D2. Behavioral parity is the explicit contract: calling Detect with
// DefaultSettings() must produce byte-identical output to the pre-D2 code.
func DefaultSettings() dto.Settings {
	return dto.Settings{
		BuyGreen:                5,
		BuyYellow:               15,
		SellYellow:              80,
		SellOrange:              90,
		SellRed:                 95,
		ATRPeriod:               14,
		ATRMultiplierDividend:   2.0,
		ATRMultiplierGrowth:     1.5,
		VolumeSMALookback:       20,
		VolumeMultiplier:        1.5,
		TrendEMAPeriod:          200,
		AdaptiveWindowMin:       100,
		AdaptiveWindowMax:       200,
		DivergenceLookbackWeeks: 52,
	}
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `cd /home/oleg/GolandProjects/tinvest && go test ./internal/service/trading_strategy/golden_x -run TestDefaultSettings -v`
Expected: `--- PASS: TestDefaultSettings`

- [ ] **Step 5: Full test sweep to confirm no regressions**

Run: `cd /home/oleg/GolandProjects/tinvest && go build ./... && go vet ./... && go test ./...`
Expected: all clean, no failures

- [ ] **Step 6: Commit**

```bash
git add internal/service/trading_strategy/golden_x/settings.go internal/service/trading_strategy/golden_x/settings_test.go
git commit -m "feat(golden_x): add DefaultSettings() with regression test"
```

---

## Task 3: Thread `settings` argument through `Detect` (no-op internally)

Widens `Detect`'s signature and cascades through `DetectFunc`, `ReplayConfig`, and every caller (production + tests). Body of `Detect` does not yet consume `settings` — that happens in Tasks 4–6. After this task, behavior is identical and all tests stay green.

**Files:**
- Modify: `internal/service/trading_strategy/golden_x/detector.go`
- Modify: `internal/service/trading_strategy/golden_x/trade.go`
- Modify: `internal/service/trading_strategy/golden_x/detector_test.go`
- Modify: `internal/service/trading_strategy/golden_x/backtest/replay.go`
- Modify: `internal/service/trading_strategy/golden_x/backtest/replay_test.go`
- Modify: `cmd/backtest/main.go`

- [ ] **Step 1: Widen `Detect` signature in `detector.go`**

In `internal/service/trading_strategy/golden_x/detector.go`, change the signature only (body untouched):

Old (lines 18-23):
```go
func Detect(
	closed []*model.CandleItemTechAnalyse,
	rsiPeriod int,
	kind dto.StrategyKind,
	useTrendFilter bool,
) (dto.Signal, error) {
```

New:
```go
func Detect(
	closed []*model.CandleItemTechAnalyse,
	rsiPeriod int,
	kind dto.StrategyKind,
	useTrendFilter bool,
	settings dto.Settings,
) (dto.Signal, error) {
	_ = settings // consumed in Tasks 4-6; explicit blank-assign documents intent and silences unused-param lint
```

Add the `_ = settings` line as the first line of the body. This is a deliberate placeholder removed in Task 6.

- [ ] **Step 2: Update `service.Trade` in `trade.go`**

In `internal/service/trading_strategy/golden_x/trade.go`, modify the `Trade` method body. Add a `settings := DefaultSettings()` declaration before the `for _, share := range in.ShareList.All()` loop (just after the existing `stops := make(map[string]dto.Stop)` line at line 88):

Old (line 88-90):
```go
	stops := make(map[string]dto.Stop)

	for _, share := range in.ShareList.All() {
```

New:
```go
	stops := make(map[string]dto.Stop)
	settings := DefaultSettings()

	for _, share := range in.ShareList.All() {
```

Then update the `Detect` call at line 98:

Old:
```go
		sig, detectErr := Detect(closed, share.RSILength, in.Kind, in.UseTrendFilter)
```

New:
```go
		sig, detectErr := Detect(closed, share.RSILength, in.Kind, in.UseTrendFilter, settings)
```

- [ ] **Step 3: Update `detector_test.go`**

In `internal/service/trading_strategy/golden_x/detector_test.go`, every `Detect(...)` call must pass `DefaultSettings()` as the 5th argument. There are 9 call sites at lines 64, 74, 102, 136, 156, 172, 185, 215, 253.

Search-and-replace pattern (verify each replacement matches exactly one of the 9 known call sites):

Old form `Detect(candles, 7, dto.StrategyKindDividend, false)` → `Detect(candles, 7, dto.StrategyKindDividend, false, DefaultSettings())`
Old form `Detect(candles, 7, dto.StrategyKindGrowth, true)` → `Detect(candles, 7, dto.StrategyKindGrowth, true, DefaultSettings())`

After the edits, every `Detect(...)` call in the file has the trailing `, DefaultSettings()`.

- [ ] **Step 4: Widen `DetectFunc` and add `Settings` to `ReplayConfig`**

In `internal/service/trading_strategy/golden_x/backtest/replay.go`:

Old (lines 9-19):
```go
// DetectFunc is the signature of golden_x.Detect; injected so the replay
// engine can be unit-tested with a fake detector.
type DetectFunc func(closed []*model.CandleItemTechAnalyse, rsiPeriod int, kind dto.StrategyKind, useTrendFilter bool) (dto.Signal, error)

type ReplayConfig struct {
	Kind           dto.StrategyKind
	RSIPeriod      int // per-share; pulled from collection.Instrument by caller
	StartIdx       int // first index at which we will evaluate a signal
	MaxWeeks       int // timeout cap (52 per spec §4.2)
	UseTrendFilter bool
}
```

New:
```go
// DetectFunc is the signature of golden_x.Detect; injected so the replay
// engine can be unit-tested with a fake detector.
type DetectFunc func(closed []*model.CandleItemTechAnalyse, rsiPeriod int, kind dto.StrategyKind, useTrendFilter bool, settings dto.Settings) (dto.Signal, error)

type ReplayConfig struct {
	Kind           dto.StrategyKind
	RSIPeriod      int // per-share; pulled from collection.Instrument by caller
	StartIdx       int // first index at which we will evaluate a signal
	MaxWeeks       int // timeout cap (52 per spec §4.2)
	UseTrendFilter bool
	Settings       dto.Settings // strategy knobs; built once by the caller and threaded into every detect() call
}
```

And update the single `detect(...)` call inside `Replay` (line 34):

Old:
```go
		sig, err := detect(closed, cfg.RSIPeriod, cfg.Kind, cfg.UseTrendFilter)
```

New:
```go
		sig, err := detect(closed, cfg.RSIPeriod, cfg.Kind, cfg.UseTrendFilter, cfg.Settings)
```

- [ ] **Step 5: Update `replay_test.go` fakes**

In `internal/service/trading_strategy/golden_x/backtest/replay_test.go`, each `fake := func(...)` literal declares the `DetectFunc` signature inline. There are four of them at lines 30, 61, 96, 122.

For each, add the 5th argument `_ dto.Settings` (ignored by the fake):

Old form:
```go
	fake := func(closed []*model.CandleItemTechAnalyse, _ int, _ dto.StrategyKind, _ bool) (dto.Signal, error) {
```

New form:
```go
	fake := func(closed []*model.CandleItemTechAnalyse, _ int, _ dto.StrategyKind, _ bool, _ dto.Settings) (dto.Signal, error) {
```

Apply this to all four declarations. The body of each fake is unchanged.

- [ ] **Step 6: Update `cmd/backtest/main.go`**

In `cmd/backtest/main.go`, two changes:

(a) Add `settings := golden_x.DefaultSettings()` before the `for _, instr := range list.All()` loop. Place it just after the `cache := backtest.NewCache(*cacheDir, fetcher, *refresh)` line (after line 82):

Old (lines 81-84):
```go
	fetcher := newGrpcFetcher(grpcClient, to)
	cache := backtest.NewCache(*cacheDir, fetcher, *refresh)

	for _, instr := range list.All() {
```

New:
```go
	fetcher := newGrpcFetcher(grpcClient, to)
	cache := backtest.NewCache(*cacheDir, fetcher, *refresh)
	settings := golden_x.DefaultSettings()

	for _, instr := range list.All() {
```

(b) Widen the closure passed to `backtest.Replay` and add the `Settings` field to `ReplayConfig`:

Old (lines 99-106):
```go
		trades := backtest.Replay(instr.ID, closed,
			func(c []*model.CandleItemTechAnalyse, period int, k dto.StrategyKind, uft bool) (dto.Signal, error) {
				return golden_x.Detect(c, period, k, uft)
			},
			backtest.ReplayConfig{
				Kind: kind, RSIPeriod: instr.RSILength, StartIdx: startIdx,
				MaxWeeks: 52, UseTrendFilter: useTrendFilter,
			})
```

New:
```go
		trades := backtest.Replay(instr.ID, closed,
			func(c []*model.CandleItemTechAnalyse, period int, k dto.StrategyKind, uft bool, s dto.Settings) (dto.Signal, error) {
				return golden_x.Detect(c, period, k, uft, s)
			},
			backtest.ReplayConfig{
				Kind: kind, RSIPeriod: instr.RSILength, StartIdx: startIdx,
				MaxWeeks: 52, UseTrendFilter: useTrendFilter, Settings: settings,
			})
```

- [ ] **Step 7: Build, vet, and run the whole test suite**

Run: `cd /home/oleg/GolandProjects/tinvest && go build ./... && go vet ./... && go test ./...`
Expected: all green. Behaviorally identical to pre-Task-3 because `Detect`'s body still consumes only the original consts.

- [ ] **Step 8: Commit**

```bash
git add internal/service/trading_strategy/golden_x/detector.go internal/service/trading_strategy/golden_x/trade.go internal/service/trading_strategy/golden_x/detector_test.go internal/service/trading_strategy/golden_x/backtest/replay.go internal/service/trading_strategy/golden_x/backtest/replay_test.go cmd/backtest/main.go
git commit -m "refactor(golden_x): thread settings arg through Detect (no-op body)"
```

---

## Task 4: Wire `settings.*` into percentile helpers and `adaptiveRSIForShare`

Refactors `adaptiveThresholds`, `adaptiveSellThresholds` to take percentile pivots as explicit args. Refactors `adaptiveRSIForShare` to drop the `Thresholds` return value and take window-bound args. Updates `Detect` to call `adaptiveThresholds` / `adaptiveSellThresholds` directly with `settings.*` fields.

**Files:**
- Modify: `internal/service/trading_strategy/golden_x/percentile.go`
- Modify: `internal/service/trading_strategy/golden_x/percentile_test.go`
- Modify: `internal/service/trading_strategy/golden_x/trade.go`
- Modify: `internal/service/trading_strategy/golden_x/detector.go`

- [ ] **Step 1: Update `percentile.go` helpers**

In `internal/service/trading_strategy/golden_x/percentile.go`, replace lines 45-67 with:

```go
// adaptiveThresholds computes the two buy-tier percentiles over an unordered
// slice of historical RSI values. pGreen and pYellow are percentile points
// in [0, 100] (typically 5 and 15). The input is not mutated; a sorted copy
// is taken internally.
func adaptiveThresholds(rsiSeries []float64, pGreen, pYellow float64) dto.Thresholds {
	sorted := append([]float64(nil), rsiSeries...)
	sort.Float64s(sorted)
	return dto.Thresholds{
		P5:  percentile(sorted, pGreen),
		P15: percentile(sorted, pYellow),
	}
}

// adaptiveSellThresholds computes the three sell-tier percentiles over an
// unordered slice of historical RSI values. pYellow/pOrange/pRed are
// percentile points in [0, 100] (typically 80/90/95). The input is not
// mutated; a sorted copy is taken internally.
func adaptiveSellThresholds(rsiSeries []float64, pYellow, pOrange, pRed float64) dto.SellThresholds {
	sorted := append([]float64(nil), rsiSeries...)
	sort.Float64s(sorted)
	return dto.SellThresholds{
		P80: percentile(sorted, pYellow),
		P90: percentile(sorted, pOrange),
		P95: percentile(sorted, pRed),
	}
}
```

The field names `P5`/`P15` on `dto.Thresholds` and `P80`/`P90`/`P95` on `dto.SellThresholds` are intentionally left as-is — renaming those is out of scope per the spec.

- [ ] **Step 2: Update tests in `percentile_test.go`**

In `internal/service/trading_strategy/golden_x/percentile_test.go`, update three call sites:

Line 103 — `TestAdaptiveThresholds`:
Old:
```go
	got := adaptiveThresholds(rsi)
```
New:
```go
	got := adaptiveThresholds(rsi, 5, 15)
```

Line 117 — `TestAdaptiveThresholds_DoesNotMutateInput`:
Old:
```go
	_ = adaptiveThresholds(in)
```
New:
```go
	_ = adaptiveThresholds(in, 5, 15)
```

Line 131 — `TestAdaptiveSellThresholds`:
Old:
```go
	got := adaptiveSellThresholds(rsi)
```
New:
```go
	got := adaptiveSellThresholds(rsi, 80, 90, 95)
```

Line 146 — `TestAdaptiveSellThresholds_DoesNotMutateInput`:
Old:
```go
	_ = adaptiveSellThresholds(in)
```
New:
```go
	_ = adaptiveSellThresholds(in, 80, 90, 95)
```

- [ ] **Step 3: Refactor `adaptiveRSIForShare` in `trade.go` to drop the Thresholds return and take window-bound args**

In `internal/service/trading_strategy/golden_x/trade.go`, replace the function at lines 194-218:

Old:
```go
// adaptiveRSIForShare computes the share's last-closed-week RSI, the trimmed
// historical RSI slice used for percentiles, and the adaptive p5/p15 buy
// thresholds. Returns ErrAdaptiveInsufficientHistory if fewer than
// adaptiveWindowMin RSI values are available. The returned slice may be used
// by the caller to derive additional percentiles (e.g. sell thresholds)
// without recomputing RSI.
func adaptiveRSIForShare(closedCandles []*model.CandleItemTechAnalyse, rsiPeriod int) (float64, []float64, dto.Thresholds, error) {
	closes := make([]float64, len(closedCandles))
	for i, c := range closedCandles {
		closes[i] = utils.CombinePrice(c.Close.Units, c.Close.Nano)
	}
	full := computeRSISeries(closes, rsiPeriod)
	if len(full) <= rsiPeriod {
		return 0, nil, dto.Thresholds{}, ErrAdaptiveInsufficientHistory
	}
	rsi := full[rsiPeriod:]
	if len(rsi) < adaptiveWindowMin {
		return 0, nil, dto.Thresholds{}, ErrAdaptiveInsufficientHistory
	}
	if len(rsi) > adaptiveWindowMax {
		rsi = rsi[len(rsi)-adaptiveWindowMax:]
	}
	lastRSI := rsi[len(rsi)-1]
	return lastRSI, rsi, adaptiveThresholds(rsi), nil
}
```

New:
```go
// adaptiveRSIForShare computes the share's last-closed-week RSI and the
// trimmed historical RSI slice used for percentile calculations. Returns
// ErrAdaptiveInsufficientHistory if fewer than minWin RSI values are
// available; trims the head to maxWin if longer. Threshold computation
// (adaptiveThresholds / adaptiveSellThresholds) is the caller's responsibility.
func adaptiveRSIForShare(closedCandles []*model.CandleItemTechAnalyse, rsiPeriod, minWin, maxWin int) (float64, []float64, error) {
	closes := make([]float64, len(closedCandles))
	for i, c := range closedCandles {
		closes[i] = utils.CombinePrice(c.Close.Units, c.Close.Nano)
	}
	full := computeRSISeries(closes, rsiPeriod)
	if len(full) <= rsiPeriod {
		return 0, nil, ErrAdaptiveInsufficientHistory
	}
	rsi := full[rsiPeriod:]
	if len(rsi) < minWin {
		return 0, nil, ErrAdaptiveInsufficientHistory
	}
	if len(rsi) > maxWin {
		rsi = rsi[len(rsi)-maxWin:]
	}
	return rsi[len(rsi)-1], rsi, nil
}
```

Note: `lowsAlignedToRSI` is intentionally unchanged. Its existing logic uses `len(rsiSeries)` to derive the trim, which is equivalent to (and slightly safer than) consuming `maxWin` directly — the spec's mention of a `maxWin` arg on this helper is informational; preserving the current signature keeps the helper's correctness guarantees intact.

- [ ] **Step 4: Update `Detect` body in `detector.go` to use the new helper signatures**

In `internal/service/trading_strategy/golden_x/detector.go`, replace lines 24-33 of `Detect`:

Old:
```go
	lastRSI, rsiSeries, thresholds, err := adaptiveRSIForShare(closed, rsiPeriod)
	if err != nil {
		return dto.Signal{}, err
	}

	sig := dto.Signal{
		RSI:            lastRSI,
		Thresholds:     thresholds,
		SellThresholds: adaptiveSellThresholds(rsiSeries),
	}
```

New:
```go
	lastRSI, rsiSeries, err := adaptiveRSIForShare(closed, rsiPeriod, settings.AdaptiveWindowMin, settings.AdaptiveWindowMax)
	if err != nil {
		return dto.Signal{}, err
	}

	sig := dto.Signal{
		RSI:            lastRSI,
		Thresholds:     adaptiveThresholds(rsiSeries, settings.BuyGreen, settings.BuyYellow),
		SellThresholds: adaptiveSellThresholds(rsiSeries, settings.SellYellow, settings.SellOrange, settings.SellRed),
	}
```

This is the first place `settings` is actually consumed. Remove the `_ = settings` placeholder added in Task 3 from line 24's predecessor area — search for the `_ = settings` line introduced in Task 3 and delete it.

- [ ] **Step 5: Build, vet, run the whole test suite**

Run: `cd /home/oleg/GolandProjects/tinvest && go build ./... && go vet ./... && go test ./...`
Expected: all green. Behaviorally identical because `DefaultSettings()` carries `BuyGreen=5, BuyYellow=15, SellYellow=80, SellOrange=90, SellRed=95, AdaptiveWindowMin=100, AdaptiveWindowMax=200` — exactly the literals/consts previously used.

- [ ] **Step 6: Commit**

```bash
git add internal/service/trading_strategy/golden_x/percentile.go internal/service/trading_strategy/golden_x/percentile_test.go internal/service/trading_strategy/golden_x/trade.go internal/service/trading_strategy/golden_x/detector.go
git commit -m "refactor(golden_x): percentile helpers + adaptiveRSIForShare consume settings"
```

---

## Task 5: Wire `settings.*` into `kForKind`

**Files:**
- Modify: `internal/service/trading_strategy/golden_x/stop.go`
- Modify: `internal/service/trading_strategy/golden_x/stop_test.go`
- Modify: `internal/service/trading_strategy/golden_x/detector.go`

- [ ] **Step 1: Update `kForKind` signature in `stop.go`**

In `internal/service/trading_strategy/golden_x/stop.go`, replace lines 5-14:

Old:
```go
// kForKind returns the ATR multiplier appropriate for the given strategy kind.
// Dividend holds longer and needs wider stops; Growth exits sooner and uses
// tighter stops. Unknown kinds fall back to Dividend — defensive only; the
// production code paths construct the enum at the call site.
func kForKind(kind dto.StrategyKind) float64 {
	if kind == dto.StrategyKindGrowth {
		return atrMultiplierGrowth
	}
	return atrMultiplierDividend
}
```

New:
```go
// kForKind returns the ATR multiplier appropriate for the given strategy kind.
// Dividend holds longer and needs wider stops; Growth exits sooner and uses
// tighter stops. Unknown kinds fall back to Dividend — defensive only; the
// production code paths construct the enum at the call site.
func kForKind(kind dto.StrategyKind, settings dto.Settings) float64 {
	if kind == dto.StrategyKindGrowth {
		return settings.ATRMultiplierGrowth
	}
	return settings.ATRMultiplierDividend
}
```

- [ ] **Step 2: Update `TestKForKind` in `stop_test.go`**

In `internal/service/trading_strategy/golden_x/stop_test.go`, replace lines 10-28:

Old:
```go
func TestKForKind(t *testing.T) {
	tests := []struct {
		name string
		kind dto.StrategyKind
		want float64
	}{
		{"Gold (Dividend) -> 2.0", dto.StrategyKindDividend, 2.0},
		{"Growth -> 1.5", dto.StrategyKindGrowth, 1.5},
		{"unknown zero-value defaults to Gold's multiplier", dto.StrategyKind(0), 2.0},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := kForKind(tc.kind)
			if math.Abs(got-tc.want) > 1e-9 {
				t.Fatalf("kForKind(%v) = %v, want %v", tc.kind, got, tc.want)
			}
		})
	}
}
```

New:
```go
func TestKForKind(t *testing.T) {
	settings := DefaultSettings()
	tests := []struct {
		name string
		kind dto.StrategyKind
		want float64
	}{
		{"Gold (Dividend) -> 2.0", dto.StrategyKindDividend, 2.0},
		{"Growth -> 1.5", dto.StrategyKindGrowth, 1.5},
		{"unknown zero-value defaults to Gold's multiplier", dto.StrategyKind(0), 2.0},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := kForKind(tc.kind, settings)
			if math.Abs(got-tc.want) > 1e-9 {
				t.Fatalf("kForKind(%v) = %v, want %v", tc.kind, got, tc.want)
			}
		})
	}
}
```

- [ ] **Step 3: Update the single call site in `detector.go`**

In `internal/service/trading_strategy/golden_x/detector.go`, line 75:

Old:
```go
			sig.Stop = stopFromATR(lastClose, atrValue, kForKind(kind))
```

New:
```go
			sig.Stop = stopFromATR(lastClose, atrValue, kForKind(kind, settings))
```

- [ ] **Step 4: Build, vet, test**

Run: `cd /home/oleg/GolandProjects/tinvest && go build ./... && go vet ./... && go test ./...`
Expected: all green.

- [ ] **Step 5: Commit**

```bash
git add internal/service/trading_strategy/golden_x/stop.go internal/service/trading_strategy/golden_x/stop_test.go internal/service/trading_strategy/golden_x/detector.go
git commit -m "refactor(golden_x): kForKind consumes settings"
```

---

## Task 6: Replace remaining const usages in `Detect`; delete unused consts

Final substitution pass. Replaces direct const references in `detector.go` with `settings.*` fields, then deletes the now-unused `const` declarations from `trade.go` and `trend_filter.go`. This closes the byte-parity contract.

**Files:**
- Modify: `internal/service/trading_strategy/golden_x/detector.go`
- Modify: `internal/service/trading_strategy/golden_x/trade.go`
- Modify: `internal/service/trading_strategy/golden_x/trend_filter.go`

- [ ] **Step 1: Replace const usages in `detector.go`**

In `internal/service/trading_strategy/golden_x/detector.go`:

(a) Line 36 — trend filter call:
Old:
```go
		status, terr := trendStatusFromClosed(closed, trendEMAPeriod)
```
New:
```go
		status, terr := trendStatusFromClosed(closed, settings.TrendEMAPeriod)
```

(b) Lines 49-55 — divergence window (two references to `divergenceLookbackWeeks`):
Old:
```go
		lows := lowsAlignedToRSI(closed, rsiPeriod, rsiSeries)
		if len(lows) > divergenceLookbackWeeks {
			lows = lows[len(lows)-divergenceLookbackWeeks:]
		}
		rsiTail := rsiSeries
		if len(rsiTail) > divergenceLookbackWeeks {
			rsiTail = rsiTail[len(rsiTail)-divergenceLookbackWeeks:]
		}
```
New:
```go
		lows := lowsAlignedToRSI(closed, rsiPeriod, rsiSeries)
		if len(lows) > settings.DivergenceLookbackWeeks {
			lows = lows[len(lows)-settings.DivergenceLookbackWeeks:]
		}
		rsiTail := rsiSeries
		if len(rsiTail) > settings.DivergenceLookbackWeeks {
			rsiTail = rsiTail[len(rsiTail)-settings.DivergenceLookbackWeeks:]
		}
```

(c) Line 62 — volume confirmation:
Old:
```go
		sig.VolumeOK = indicators.VolumeConfirmed(volumes, volumeSMALookback, volumeMultiplier)
```
New:
```go
		sig.VolumeOK = indicators.VolumeConfirmed(volumes, settings.VolumeSMALookback, settings.VolumeMultiplier)
```

(d) Line 72 — ATR period:
Old:
```go
		if atrValue := indicators.ATR(highs, lowsF, closes, atrPeriod); atrValue > 0 {
```
New:
```go
		if atrValue := indicators.ATR(highs, lowsF, closes, settings.ATRPeriod); atrValue > 0 {
```

(e) Lines 84-85 — the `_ = kind` blank-assign is now obsolete because `kForKind(kind, settings)` consumes `kind` directly when the buy branch is skipped (Go's compiler doesn't warn on unused function params, so the blank-assign was only documentation). Delete this line and its preceding comment block:

Old:
```go
	_ = kind // consumed by kForKind only when a buy tier fires; blank-assign silences the compiler when the buy branch is skipped
	return sig, nil
}
```
New:
```go
	return sig, nil
}
```

- [ ] **Step 2: Delete strategy-knob consts from `trade.go`**

In `internal/service/trading_strategy/golden_x/trade.go`, delete the eight `const` declarations that are now unreferenced. Specifically, replace lines 20-66 (covering all comment+const blocks from `adaptiveWindowMax` through `atrMultiplierGrowth` except `candleLookbackWeeks` and `divergenceFractalK` which are kept):

Old (lines 20-66):
```go
// adaptiveWindowMax is the maximum number of historical RSI values we keep for
// percentile computation. Matches the original ~200 from the design spec.
const adaptiveWindowMax = 200

// adaptiveWindowMin is the lower bound; shares with fewer closed-week RSI
// values than this are skipped (consistent with C1's insufficient-history rule).
const adaptiveWindowMin = 100

// candleLookbackWeeks is how many weekly candles we request per share per tick.
// Covers EMA200 warmup (Growth-only) and adaptive-tier RSI history (both
// instances) in one RPC, staying under the Tinkoff weekly cap (300 per call).
const candleLookbackWeeks = 260

// divergenceFractalK is the half-window for fractal pivot detection: a candle
// at index i is a confirmed pivot low iff its Low is strictly less than the
// Lows of its 2*k neighbors (k on each side). k=2 is the classical Williams
// fractal setting; on a weekly TF it gives a swing-low at least 2 weeks old.
const divergenceFractalK = 2

// divergenceLookbackWeeks bounds how far back we search for the prior pivot
// low. Older pivots are ignored — a year-old swing low is "stale" evidence
// for current week behavior on a weekly TF.
const divergenceLookbackWeeks = 52

// volumeSMALookback is the number of closed weekly candles preceding the
// last closed week, used as the SMA baseline for volume confirmation.
const volumeSMALookback = 20

// volumeMultiplier is the strictness factor: the last closed week's volume
// must be > volumeMultiplier × SMA of the previous volumeSMALookback weeks
// for the 🔊 badge to fire. 1.5× is the balance between "barely above
// average" (which would emit the badge for most shares and dilute meaning)
// and a 2× "rare spike" (which would almost never fire).
const volumeMultiplier = 1.5

// atrPeriod is Wilder's standard ATR period applied on the weekly TF closed-
// candle stream used for buy-side stop suggestions.
const atrPeriod = 14

// atrMultiplierDividend is the ATR stop multiplier for the Dividend (long-
// hold) strategy: wider stops survive deeper weekly noise.
const atrMultiplierDividend = 2.0

// atrMultiplierGrowth is the stop multiplier for Growth — tighter, since the
// strategy exits sooner on RSI overheats.
const atrMultiplierGrowth = 1.5
```

New (only the two kept consts remain, with their comments):
```go
// candleLookbackWeeks is how many weekly candles we request per share per tick.
// Covers EMA200 warmup (Growth-only) and adaptive-tier RSI history (both
// instances) in one RPC, staying under the Tinkoff weekly cap (300 per call).
// Out of scope for D2: this is a fetch-policy knob, not an algorithm knob.
const candleLookbackWeeks = 260

// divergenceFractalK is the half-window for fractal pivot detection: a candle
// at index i is a confirmed pivot low iff its Low is strictly less than the
// Lows of its 2*k neighbors (k on each side). k=2 is the classical Williams
// fractal setting; on a weekly TF it gives a swing-low at least 2 weeks old.
// Out of scope for D2: indicator-internal pivot width, not on the knob list.
const divergenceFractalK = 2
```

- [ ] **Step 3: Delete `trendEMAPeriod` from `trend_filter.go`**

In `internal/service/trading_strategy/golden_x/trend_filter.go`, delete lines 12-13:

Old:
```go
// trendEMAPeriod is the lookback for the higher-trend EMA filter (EMA200 W).
const trendEMAPeriod = 200
```

(Delete those two lines, leaving the preceding `import` block and the next comment (`// ErrInsufficientHistory ...`) adjacent. `trendStatusFromClosed` already takes `period` as an argument, so no signature change is needed.)

- [ ] **Step 4: Build, vet, run the whole test suite**

Run: `cd /home/oleg/GolandProjects/tinvest && go build ./... && go vet ./... && go test ./...`
Expected: all green. No `undefined: <constname>` errors — every former const reference now resolves to a field on the `settings` value plumbed through `Detect`.

- [ ] **Step 5: Manual byte-parity check (optional but recommended)**

If a recent backtest Markdown report exists from before this branch was rebuilt (or if the user has one cached), regenerate one with the same `--kind / --from / --to / --shares` args and diff it against the pre-D2 version:

```bash
go run ./cmd/backtest --kind=Dividend --from=2020-08-17 > /dev/null  # uses cached candles; no T_BANK needed
```

Then compare the latest `cache/backtests/<ts>_Dividend.md` against the prior one. Diff should be empty except for the `generated <timestamp>` line in the header. This is the byte-parity acceptance criterion from the spec (§Acceptance, item 8). If any other line differs, halt and investigate — a knob slipped.

This step is technically optional for plan execution because it requires a recent baseline; if no baseline exists, skip and rely on the test suite for parity confirmation.

- [ ] **Step 6: Commit**

```bash
git add internal/service/trading_strategy/golden_x/detector.go internal/service/trading_strategy/golden_x/trade.go internal/service/trading_strategy/golden_x/trend_filter.go
git commit -m "refactor(golden_x): strategy knobs through settings, drop unused consts"
```

---

## Acceptance check (run after Task 6)

Re-verify every acceptance item from the spec:

1. **`dto.Settings` exists with all 14 fields and the documented defaults.** Verified by Task 1 + Task 2's `TestDefaultSettings`.
2. **`golden_x.DefaultSettings()` returns those defaults.** Verified by `TestDefaultSettings`.
3. **`Detect` accepts a 5th `settings` argument; all in-package `const` knobs for strategy parameters are removed.** Verified by Task 3 (signature) and Task 6 (const deletion).
4. **`service.Trade` and `cmd/backtest/main.go` both call `DefaultSettings()` and pass it through.** Verified by Task 3 Step 2 + Step 6.
5. **`backtest.ReplayConfig` carries `Settings dto.Settings`; `DetectFunc` matches the new `Detect` signature.** Verified by Task 3 Step 4.
6. **`go build ./... && go vet ./... && go test ./...` all clean.** Final gate after Task 6.
7. **New `TestDefaultSettings` regression test passes.** Verified each task's test sweep.
8. **A backtest run against the cached Dividend full list produces byte-identical output (modulo timestamp).** Optional manual step (Task 6 Step 5); contract enforced indirectly by the unchanged-semantics test suite.
9. **Commit lands on `feature/golden-x-stage-a`** with messages in the D3-style. Each task's commit follows the convention.

---

## Self-Review Notes

- **Spec coverage:** every spec component is covered by a task: `dto/settings.go` (Task 1), `golden_x/settings.go` (Task 2), `detector.go` widening (Task 3), `percentile.go` + `adaptiveRSIForShare` (Task 4), `kForKind` (Task 5), remaining literals + const cleanup (Task 6), all callers (Tasks 3 + 4 + 5 + 6 distributed across), all tests (Tasks 2-5).
- **Spec deviation note:** `lowsAlignedToRSI` is intentionally left at its current signature. The spec line «`lowsAlignedToRSI(closed, rsiPeriod, rsiSeries, maxWin int)` accept window bounds explicitly» is informational; the helper's existing logic derives the trim from `len(rsiSeries)`, which after `adaptiveRSIForShare`'s trim is always `≤ maxWin` — using `len(rsiSeries)` is correct in every reachable case, and adding `maxWin` as a redundant arg adds no value. This deviation does not affect any acceptance criterion.
- **Type consistency:** field names `BuyGreen / BuyYellow / SellYellow / SellOrange / SellRed / ATRPeriod / ATRMultiplierDividend / ATRMultiplierGrowth / VolumeSMALookback / VolumeMultiplier / TrendEMAPeriod / AdaptiveWindowMin / AdaptiveWindowMax / DivergenceLookbackWeeks` are used consistently across Task 1 (struct def), Task 2 (regression test + constructor), Task 4 (consumed in `Detect`, `percentile.go`), Task 5 (consumed in `kForKind`), Task 6 (consumed for remaining literals).
- **No placeholders:** every step contains the exact code change or exact command. The optional Task 6 Step 5 (manual byte-parity check) is gated on the existence of a pre-D2 baseline report and is explicitly flagged as such.
