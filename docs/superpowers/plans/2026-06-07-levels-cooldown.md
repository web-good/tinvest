# Levels re-entry cooldown — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add a level-aware re-entry cooldown to the levels strategy so it stops re-buying the same support level that just stopped it out, without killing entries on other levels.

**Architecture:** Pure, stateless. The cooldown is computed from the candle window inside `core.Decide` (mirroring the existing `recentlyBelow` flag) and passed into the pure `decide` core as a boolean. A new `CooldownBars` param gates it (`0` = disabled). No changes to `MarketData`, the backtest engine, or the live runner — the check is a pure function of the candles both already supply.

**Tech Stack:** Go 1.25, standard `testing`. Files live under `internal/service/trading_strategy/levels/strategy/core` and `.../rusal`.

**Spec:** `docs/superpowers/specs/2026-06-07-levels-cooldown-design.md`

---

## File Structure

- **Modify** `internal/service/trading_strategy/levels/strategy/core/core.go`
  - Add `CooldownBars int` to `Params`.
  - Add `recentBounce bool` to `decideInput`.
  - Add pure helper `recentBounceOff(...)`.
  - Compute `recentBounce` in `Decide` (next to `recentlyBelow`) and pass it in.
  - Add the cooldown guard in `decide` (flat entry path).
- **Modify** `internal/service/trading_strategy/levels/strategy/core/core_test.go`
  - Unit tests for `recentBounceOff` and the `decide` cooldown guard; one integration test through `Decide`.
- **Modify** `internal/service/trading_strategy/levels/strategy/rusal/rusal.go`
  - Set `CooldownBars: 10` starting value.

Why this shape: the codebase already separates the impure indicator wrapper (`Decide`) from the pure decision core (`decide`) and passes precomputed flags between them (`recentlyBelow`). The cooldown follows that exact pattern, so it is unit-testable through `decide(decideInput)` and integration-testable through `Decide(MarketData)`.

---

## Task 1: Add the `recentBounceOff` pure helper

**Files:**
- Modify: `internal/service/trading_strategy/levels/strategy/core/core.go`
- Test: `internal/service/trading_strategy/levels/strategy/core/core_test.go`

- [ ] **Step 1: Write the failing test**

Append to `core_test.go`:

```go
func TestRecentBounceOff(t *testing.T) {
	// level 100, tol 0.5 (absolute), confirmFrac 0.5.
	// Bars: [0] bounce off 100 (low 99.6, bullish close 101 above level),
	//       [1] no touch (low 101), [2] current bar (excluded from the scan).
	highs := []float64{101.5, 101.5, 101.5}
	lows := []float64{99.6, 101.0, 99.6}
	closes := []float64{101.0, 101.0, 101.0}

	tests := []struct {
		name     string
		lookback int
		want     bool
	}{
		{"sees the bounce two bars back", 5, true},
		{"lookback 1 only sees bar[1] (no touch)", 1, false},
		{"lookback 0 disabled", 0, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := recentBounceOff(highs, lows, closes, 100, 0.5, 0.5, tt.lookback)
			if got != tt.want {
				t.Fatalf("recentBounceOff = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRecentBounceOffExcludesCurrentBar(t *testing.T) {
	// Only the current (last) bar is a bounce; it must NOT count.
	highs := []float64{100.5, 101.5}
	lows := []float64{101.0, 99.6} // bar[0] no touch, bar[1]=current bounces
	closes := []float64{100.2, 101.0}
	if recentBounceOff(highs, lows, closes, 100, 0.5, 0.5, 5) {
		t.Fatal("current bar must be excluded from the cooldown scan")
	}
}

func TestRecentBounceOffOtherLevel(t *testing.T) {
	// A bounce off 100 must not trigger cooldown for a candidate level 110.
	highs := []float64{101.5, 110.0}
	lows := []float64{99.6, 109.0}
	closes := []float64{101.0, 109.5}
	if recentBounceOff(highs, lows, closes, 110, 0.5, 0.5, 5) {
		t.Fatal("bounce off a different level must not count")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/service/trading_strategy/levels/strategy/core/ -run TestRecentBounceOff -v`
Expected: FAIL — `undefined: recentBounceOff`.

- [ ] **Step 3: Write minimal implementation**

In `core.go`, add this helper next to `recentlyBelowLevel`:

```go
// recentBounceOff reports whether any of the last `lookback` completed bars
// (excluding the current bar) was itself a bounce off `level`: its low touched the
// level within tol AND it closed bullishly back above the level. It cools down
// repeated entries on a level that just stopped the strategy out. tol is absolute
// (LevelTolATR*ATR); confirmFrac mirrors the entry's bullish-close test. A
// non-positive lookback disables the check.
func recentBounceOff(highs, lows, closes []float64, level, tol, confirmFrac float64, lookback int) bool {
	n := len(closes)
	if n < 2 || lookback <= 0 {
		return false
	}
	end := n - 1 // exclude the current bar
	start := end - lookback
	if start < 0 {
		start = 0
	}
	for i := start; i < end; i++ {
		touched := lows[i] <= level+tol
		bullish := bullishClose(highs[i], lows[i], closes[i], confirmFrac) && closes[i] > level
		if touched && bullish {
			return true
		}
	}
	return false
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/service/trading_strategy/levels/strategy/core/ -run TestRecentBounceOff -v`
Expected: PASS (all three subtests of `TestRecentBounceOff` plus the two sibling tests).

- [ ] **Step 5: Commit**

```bash
git add internal/service/trading_strategy/levels/strategy/core/core.go internal/service/trading_strategy/levels/strategy/core/core_test.go
git commit -m "feat(levels): add recentBounceOff pure helper for re-entry cooldown

Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>"
```

---

## Task 2: Wire the cooldown into the decision core

**Files:**
- Modify: `internal/service/trading_strategy/levels/strategy/core/core.go` (Params, decideInput, decide, Decide)
- Test: `internal/service/trading_strategy/levels/strategy/core/core_test.go`

- [ ] **Step 1: Add the `CooldownBars` param and `recentBounce` input field**

In `core.go`, add to the `Params` struct (after `MinRR`):

```go
	MinRR             float64 // reject entry if (target-price) < MinRR*(price-stop); <=0 disables
	CooldownBars      int     // after a bounce off a level, skip new entries on that level for N bars; <=0 disables
```

In `core.go`, add to the `decideInput` struct (after `recentlyBelow`):

```go
	recentlyBelow bool    // some close within BreakoutLookback was below the support level
	recentBounce  bool    // this same support already bounced within CooldownBars (cooldown)
```

This compiles immediately (zero-value `false`/`0` is inert). Run `go build ./...` to confirm.

- [ ] **Step 2: Write the failing decide-level tests**

Append to `core_test.go`:

```go
// coreWithCooldown returns a core whose params match testParams but with the
// cooldown enabled for N bars.
func coreWithCooldown(bars int) *Strategy {
	p := testParams()
	p.CooldownBars = bars
	return NewWithParams("TEST", p)
}

func TestDecideCooldownBlocksReentry(t *testing.T) {
	s := coreWithCooldown(5)
	in := bounceInput()
	in.recentBounce = true // same level already bounced within the window
	if sig := s.decide(in); sig.Kind != model.SignalNone {
		t.Fatalf("cooldown must block re-entry, got %v/%q", sig.Kind, sig.Reason)
	}
}

func TestDecideCooldownDisabledByZero(t *testing.T) {
	s := newCore() // testParams() has CooldownBars = 0
	in := bounceInput()
	in.recentBounce = true // flag set, but cooldown disabled -> must still buy
	if sig := s.decide(in); sig.Kind != model.SignalBuy {
		t.Fatalf("CooldownBars=0 must ignore recentBounce, got %v/%q", sig.Kind, sig.Reason)
	}
}

func TestDecideCooldownAllowsWhenNoRecentBounce(t *testing.T) {
	s := coreWithCooldown(5)
	in := bounceInput()
	in.recentBounce = false // no recent bounce on this level -> entry allowed
	if sig := s.decide(in); sig.Kind != model.SignalBuy {
		t.Fatalf("cooldown enabled but no recent bounce must still buy, got %v/%q", sig.Kind, sig.Reason)
	}
}
```

- [ ] **Step 3: Run tests to verify they fail**

Run: `go test ./internal/service/trading_strategy/levels/strategy/core/ -run TestDecideCooldown -v`
Expected: `TestDecideCooldownBlocksReentry` FAILS (returns Buy, cooldown not yet enforced). The other two PASS already (no guard yet means Buy in both, which is what they assert).

- [ ] **Step 4: Add the cooldown guard in `decide`**

In `core.go`, in the flat entry path of `decide`, add the guard immediately after the `if !okS { return sig }` line and before `dist := in.price - support.Price`:

```go
	if !okS {
		return sig
	}

	// Cooldown: skip a new entry if this same support already produced a bounce
	// within the last CooldownBars (anti-churn on a level that just stopped us out).
	if s.p.CooldownBars > 0 && in.recentBounce {
		return sig
	}

	dist := in.price - support.Price
```

- [ ] **Step 5: Run the decide tests to verify they pass**

Run: `go test ./internal/service/trading_strategy/levels/strategy/core/ -run TestDecideCooldown -v`
Expected: all three PASS.

- [ ] **Step 6: Compute `recentBounce` in `Decide`**

In `core.go`, replace the existing support-resolution block in `Decide`:

```go
	recentlyBelow := false
	if support, ok := levels.NearestSupportBelow(lvls, md.Price); ok {
		recentlyBelow = recentlyBelowLevel(md.Closes, support.Price, s.p.BreakoutLookback)
	}
```

with:

```go
	recentlyBelow := false
	recentBounce := false
	if support, ok := levels.NearestSupportBelow(lvls, md.Price); ok {
		recentlyBelow = recentlyBelowLevel(md.Closes, support.Price, s.p.BreakoutLookback)
		if s.p.CooldownBars > 0 {
			recentBounce = recentBounceOff(md.Highs, md.Lows, md.Closes,
				support.Price, s.p.LevelTolATR*atr, s.p.ConfirmCloseFrac, s.p.CooldownBars)
		}
	}
```

Then add `recentBounce` to the `decideInput` literal in `Decide` (after the `recentlyBelow:` line):

```go
		recentlyBelow: recentlyBelow,
		recentBounce:  recentBounce,
		pos:           md.Position,
```

- [ ] **Step 7: Run the full core package tests**

Run: `go test ./internal/service/trading_strategy/levels/strategy/core/ -v`
Expected: PASS — all existing tests (regression at `CooldownBars=0`) plus the new ones.

- [ ] **Step 8: Commit**

```bash
git add internal/service/trading_strategy/levels/strategy/core/core.go internal/service/trading_strategy/levels/strategy/core/core_test.go
git commit -m "feat(levels): enforce level-aware re-entry cooldown in decide

Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>"
```

---

## Task 3: Integration test through `Decide` and RUAL default

**Files:**
- Modify: `internal/service/trading_strategy/levels/strategy/rusal/rusal.go`
- Test: `internal/service/trading_strategy/levels/strategy/core/core_test.go`

- [ ] **Step 1: Write the failing integration test**

Append to `core_test.go`. This builds the same support/resistance clusters as `TestDecideIntegrationBounceBuy`, then puts a bounce bar off support immediately before the current bounce bar; with cooldown enabled, the current entry must be suppressed.

```go
func TestDecideIntegrationCooldownBlocks(t *testing.T) {
	p := testParams()
	p.TrendFilterPeriod = 0 // no HTF data in this synthetic series
	p.CooldownBars = 3
	s := NewWithParams("TEST", p)

	var highs, lows, closes []float64
	var vols []int64

	appendBars(&highs, &lows, &closes, &vols, 30, 99.5, 100.5, 100, 1000)  // support cluster
	appendBars(&highs, &lows, &closes, &vols, 20, 104.0, 106.0, 105, 50)   // thin middle
	appendBars(&highs, &lows, &closes, &vols, 30, 109.5, 110.5, 110, 1000) // resistance cluster
	appendBars(&highs, &lows, &closes, &vols, 20, 104.0, 106.0, 105, 50)   // thin, drift back down

	// Prior bar: a bounce off support 100 (low 99.6, bullish close above 100).
	highs = append(highs, 101.5)
	lows = append(lows, 99.6)
	closes = append(closes, 101.0)
	vols = append(vols, 200)

	// Current bar: another bounce off the same support 100.
	highs = append(highs, 101.5)
	lows = append(lows, 99.6)
	closes = append(closes, 101.0)
	vols = append(vols, 200)

	md := strategy.MarketData{
		Price:   101.0,
		Highs:   highs,
		Lows:    lows,
		Closes:  closes,
		Volumes: vols,
	}
	if sig := s.Decide(md); sig.Kind != model.SignalNone {
		t.Fatalf("cooldown must suppress the second consecutive bounce, got %v/%q", sig.Kind, sig.Reason)
	}
}
```

- [ ] **Step 2: Run it to verify it passes (behavior already implemented in Task 2)**

Run: `go test ./internal/service/trading_strategy/levels/strategy/core/ -run TestDecideIntegrationCooldownBlocks -v`
Expected: PASS. (The decision logic from Task 2 already enforces this; this test proves the `Decide` wiring computes `recentBounce` correctly end-to-end.)

If it FAILS with Buy, the `recentBounce` computation in `Decide` (Task 2, Step 6) is wrong — fix there, not by weakening the test.

- [ ] **Step 3: Set the RUAL starting value**

In `internal/service/trading_strategy/levels/strategy/rusal/rusal.go`, add the field to `DefaultParams()` (after `MinRR`):

```go
		MinRR:             1.5,   // skip setups whose target is < 1.5x the risk
		CooldownBars:      10,    // after a bounce off a level, mute re-entries on it for 10 bars
```

- [ ] **Step 4: Run the whole levels + backtest packages**

Run: `go test ./internal/service/trading_strategy/levels/... ./internal/service/backtest/...`
Expected: PASS (no existing test asserts the absence of `CooldownBars`; the levels-registry generic-binding tests stay green).

- [ ] **Step 5: Commit**

```bash
git add internal/service/trading_strategy/levels/strategy/core/core_test.go internal/service/trading_strategy/levels/strategy/rusal/rusal.go
git commit -m "feat(levels): integration test + RUAL cooldown default

Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>"
```

---

## Task 4: Verify effect on the RUAL backtest

**Files:** none (verification only).

- [ ] **Step 1: Re-run the backtest with the new default**

Run:
```bash
go run ./cmd/backtest -ticker RUAL -strategy levels -interval Hour1 -months 18 -out reports/level
```
Expected: prints a `report: ...` line with `trades=`, `net=`, `PF=`.

- [ ] **Step 2: Compare against the baseline**

Baseline (no cooldown): trades=42, net=8766.52, PF=1.344, max DD=10.0%.
Open the new `reports/level/RUAL_levels_Hour1_*.md` and compare the metrics table. Confirm:
- The March–April 2026 same-level re-entry cluster shrank (fewer SL trades in that window in the trade journal).
- Trade #41-style winners on *other* levels are retained where applicable.

This step is observational — report the new metrics back to the user. Do not tune `CooldownBars` here; calibration is a separate follow-up.

- [ ] **Step 3: Final full test run**

Run: `go test ./...`
Expected: PASS.

---

## Notes for the implementer

- Keep `CooldownBars=0` behavior bit-for-bit identical to today — the guard and the `Decide` computation are both gated on `CooldownBars > 0`.
- Do not touch `MarketData`, `internal/domain/backtest/engine.go`, or `scalping/trade.go`. The cooldown is intentionally a pure function of the candle window; live-runner parity needs no extra wiring.
- `recentBounceOff` assumes `highs`, `lows`, `closes` are the same length and aligned (as `Decide` always builds them); this matches the existing `recentlyBelowLevel`/`recentHigh` helpers.
