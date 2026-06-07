# Levels Swing-Low Hard Stop — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Anchor the levels strategy's hard `SL` to a structural swing-low (`recentLow(SwingLowWindow) − SLMult×ATR`) instead of a fixed distance from entry, using the same anchor for the entry RR check and the live stop.

**Architecture:** All decision logic lives in the pure `core` package (`internal/service/trading_strategy/levels/strategy/core/core.go`). We add a `recentLow` helper (mirror of `recentHigh`), a `SwingLowWindow` param, plumb `recentLow` through `decideInput`/`Build`/`Lookback`, and switch both stop computations to the structural anchor. RUAL + generic defaults get the new param; the calibration grid gains a `SwingLowWindow` axis.

**Tech Stack:** Go 1.25, standard `testing` package, table-driven tests. Backtest CLI: `go run ./cmd/backtest`.

**Spec:** `docs/superpowers/specs/2026-06-07-levels-swing-low-stop-design.md`

---

## File Structure

- `internal/service/trading_strategy/levels/strategy/core/core.go` — add `recentLow` helper, `SwingLowWindow` field, `decideInput.recentLow`, populate in `Build`, extend `Lookback`, switch stop anchor in `decide()`.
- `internal/service/trading_strategy/levels/strategy/core/core_test.go` — new tests for `recentLow`, `Lookback`, the structural stop anchor, and entry↔management consistency; update existing fixtures/assertions that hard-coded the old entry-anchored stop.
- `internal/service/trading_strategy/levels/strategy/rusal/rusal.go` — set `SwingLowWindow: 10` default.
- `internal/service/backtest/levels_registry.go` — set `SwingLowWindow: 10` in `genericLevelsDefaults` (kept equal to RUAL by the registry test).
- `data/params/rusal/levels_grid.json` — add `SwingLowWindow` axis.

---

### Task 1: `recentLow` helper

**Files:**
- Modify: `internal/service/trading_strategy/levels/strategy/core/core.go` (add helper near `recentHigh`, ~line 281)
- Test: `internal/service/trading_strategy/levels/strategy/core/core_test.go`

- [ ] **Step 1: Write the failing test**

Append to `core_test.go`:

```go
func TestRecentLow(t *testing.T) {
	cases := []struct {
		name   string
		lows   []float64
		window int
		want   float64
	}{
		{"window smaller than series", []float64{5, 4, 6, 3, 7}, 3, 3},
		{"window larger than series", []float64{5, 4, 6}, 10, 4},
		{"full series", []float64{9, 2, 8}, 3, 2},
		{"non-positive window clamps to last bar", []float64{9, 2, 8}, 0, 8},
		{"empty series", nil, 5, 0},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := recentLow(c.lows, c.window); got != c.want {
				t.Fatalf("recentLow(%v, %d) = %v, want %v", c.lows, c.window, got, c.want)
			}
		})
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/service/trading_strategy/levels/strategy/core/ -run TestRecentLow -v`
Expected: FAIL — `undefined: recentLow`.

- [ ] **Step 3: Write minimal implementation**

In `core.go`, immediately after the `recentHigh` function (ends ~line 281), add:

```go
// recentLow returns the lowest low over the last `window` bars (all bars if fewer).
// A non-positive window is clamped to the last bar so it can never index out of range.
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
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/service/trading_strategy/levels/strategy/core/ -run TestRecentLow -v`
Expected: PASS (all 5 subtests).

- [ ] **Step 5: Commit**

```bash
git add internal/service/trading_strategy/levels/strategy/core/core.go internal/service/trading_strategy/levels/strategy/core/core_test.go
git commit -m "feat(levels): add recentLow helper (mirror of recentHigh)

Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>"
```

---

### Task 2: Plumb `SwingLowWindow` param, `decideInput.recentLow`, and `Lookback`

This task adds the parameter and wiring **without** changing the stop formula yet, so existing behavior and tests stay green. Behavior switch is Task 3.

**Files:**
- Modify: `internal/service/trading_strategy/levels/strategy/core/core.go` (Params ~line 27, Lookback ~line 54, decideInput ~line 76, Build ~line 131)
- Test: `internal/service/trading_strategy/levels/strategy/core/core_test.go`

- [ ] **Step 1: Write the failing Lookback test**

Append to `core_test.go`:

```go
func TestLookbackIncludesSwingLowWindow(t *testing.T) {
	p := testParams()
	p.ProfileWindow = 30
	p.ChandelierWindow = 20
	p.ATRPeriod = 14
	p.BreakoutLookback = 10
	p.SwingLowWindow = 100 // now the hungriest consumer
	s := NewWithParams("TEST", p)
	if got := s.Lookback(); got != 105 {
		t.Fatalf("Lookback = %d, want 105 (SwingLowWindow 100 + margin 5)", got)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/service/trading_strategy/levels/strategy/core/ -run TestLookbackIncludesSwingLowWindow -v`
Expected: FAIL — `p.SwingLowWindow undefined` (compile error).

- [ ] **Step 3: Add the param field**

In `core.go`, in the `Params` struct, add after the `SLMult` line (line 27):

```go
	SwingLowWindow    int     // bars scanned for the structural low anchoring the hard stop
```

- [ ] **Step 4: Extend Lookback**

In `core.go` `Lookback()`, add this block before the final `return m + 5` (after the `BreakoutLookback` block, ~line 64):

```go
	if s.p.SwingLowWindow > m {
		m = s.p.SwingLowWindow
	}
```

- [ ] **Step 5: Add the decideInput field and populate it in Build**

In `core.go`, in `decideInput`, add after the `recentHigh` field (line 76):

```go
	recentLow     float64 // structural low over SwingLowWindow
```

In `Build` (the `Decide` method), in the `decideInput{...}` literal, add after the `recentHigh:` line (line 131):

```go
		recentLow:     recentLow(md.Lows, s.p.SwingLowWindow),
```

- [ ] **Step 6: Run test to verify it passes**

Run: `go test ./internal/service/trading_strategy/levels/strategy/core/ -run TestLookbackIncludesSwingLowWindow -v`
Expected: PASS.

- [ ] **Step 7: Verify the whole core package still compiles and passes**

Run: `go test ./internal/service/trading_strategy/levels/strategy/core/`
Expected: PASS (no behavior change yet; `recentLow` field is populated but unused by `decide`).

- [ ] **Step 8: Commit**

```bash
git add internal/service/trading_strategy/levels/strategy/core/core.go internal/service/trading_strategy/levels/strategy/core/core_test.go
git commit -m "feat(levels): plumb SwingLowWindow param and recentLow input

Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>"
```

---

### Task 3: Switch hard-stop anchor to the structural swing-low

Switch both the entry RR-check stop and the live management stop from entry/support anchors to `recentLow − SLMult×ATR`. Update the shared `bounceInput` fixture and the three assertions that hard-coded the old entry-anchored stop.

**Files:**
- Modify: `internal/service/trading_strategy/levels/strategy/core/core.go` (`decide`, lines ~149 and ~196)
- Test: `internal/service/trading_strategy/levels/strategy/core/core_test.go`

- [ ] **Step 1: Add `recentLow` to the shared fixture**

In `core_test.go`, in `bounceInput()`, add after the `recentHigh: 102,` line (line 50):

```go
		recentLow:     99.5, // structural low below the bounce (<= barLow 99.8)
```

- [ ] **Step 2: Write the new failing tests**

Append to `core_test.go`:

```go
func TestDecideHardStopUsesSwingLow(t *testing.T) {
	s := newCore()
	in := bounceInput()
	in.pos = &strategy.Position{PurchasePrice: 100, Quantity: 1}
	in.recentLow = 97 // deep structural low -> stop 97 - 1*1 = 96, wider than entry-1ATR
	in.price = 95.9
	in.barLow = 95.9 // pierces the structural stop 96
	sig := s.decide(in)
	if sig.Kind != model.SignalSell || sig.Reason != "SL" {
		t.Fatalf("want Sell/SL, got %v/%q", sig.Kind, sig.Reason)
	}
	if sig.StopLoss != 96 {
		t.Errorf("stop = %v, want 96 (recentLow 97 - SLMult 1 * atr 1)", sig.StopLoss)
	}
}

func TestDecideHardStopWiderThanEntryAnchor(t *testing.T) {
	s := newCore()
	in := bounceInput()
	in.pos = &strategy.Position{PurchasePrice: 100, Quantity: 1}
	in.recentLow = 97 // structural stop 96; the old entry-anchored stop would be 99
	in.price = 98     // below old stop 99 but above new structural stop 96
	in.barLow = 98
	sig := s.decide(in)
	if sig.Kind != model.SignalNone {
		t.Fatalf("structural stop must hold where entry-anchored stop would have fired, got %v/%q", sig.Kind, sig.Reason)
	}
}

func TestDecideEntryAndLiveStopShareAnchor(t *testing.T) {
	s := newCore()
	in := bounceInput() // recentLow 99.5, atr 1, SLMult 1 -> stop 98.5
	entrySig := s.decide(in)
	if entrySig.Kind != model.SignalBuy {
		t.Fatalf("want Buy entry, got %v", entrySig.Kind)
	}
	// Same bar, now holding: the live management stop must use the same anchor.
	in.pos = &strategy.Position{PurchasePrice: in.price, Quantity: 1}
	liveSig := s.decide(in)
	if liveSig.StopLoss != entrySig.StopLoss {
		t.Fatalf("live stop %v != entry stop %v (anchors diverged)", liveSig.StopLoss, entrySig.StopLoss)
	}
	if entrySig.StopLoss != 98.5 {
		t.Errorf("stop = %v, want 98.5 (recentLow 99.5 - SLMult 1 * atr 1)", entrySig.StopLoss)
	}
}
```

- [ ] **Step 3: Run new tests to verify they fail**

Run: `go test ./internal/service/trading_strategy/levels/strategy/core/ -run 'TestDecideHardStopUsesSwingLow|TestDecideHardStopWiderThanEntryAnchor|TestDecideEntryAndLiveStopShareAnchor' -v`
Expected: FAIL — stops still computed from entry (99) / support (99), not from `recentLow`.

- [ ] **Step 4: Switch the live (management) stop anchor**

In `core.go` `decide()`, the position-management branch. Replace:

```go
		entry := in.pos.PurchasePrice
		hardSL := entry - s.p.SLMult*in.atr
```

with:

```go
		entry := in.pos.PurchasePrice
		hardSL := in.recentLow - s.p.SLMult*in.atr
```

(`entry` stays — it is still used by the `armed` trail-arming check below.)

- [ ] **Step 5: Switch the entry RR-check stop anchor**

In `core.go` `decide()`, the entry branch. Replace:

```go
		stop := support.Price - s.p.SLMult*in.atr
```

with:

```go
		stop := in.recentLow - s.p.SLMult*in.atr
```

- [ ] **Step 6: Update the three existing assertions broken by the new anchor**

In `core_test.go`:

(a) `TestDecideBounceBuy` — replace the stop assertion block:

```go
	if sig.StopLoss != 100-1.0*1.0 {
		t.Errorf("stop = %v, want %v", sig.StopLoss, 99.0)
	}
```

with:

```go
	if sig.StopLoss != 99.5-1.0*1.0 {
		t.Errorf("stop = %v, want %v (recentLow 99.5 - SLMult*atr)", sig.StopLoss, 98.5)
	}
```

(b) `TestDecideHardStopExit` — replace its body (after the `in.pos` line) so it uses the structural stop 98.5:

```go
	in.price = 98.4  // <= recentLow(99.5) - SLMult*atr = 98.5
	in.barLow = 98.4 // low тоже пробил hard SL
	sig := s.decide(in)
	if sig.Kind != model.SignalSell || sig.Reason != "SL" {
		t.Fatalf("want Sell/SL, got %v/%q", sig.Kind, sig.Reason)
	}
	if sig.StopLoss != 98.5 {
		t.Errorf("stop = %v, want 98.5", sig.StopLoss)
	}
```

(c) `TestDecideHardStopOnLowWhileCloseAbove` — replace its body (after the `in.pos` line):

```go
	in.price = 99.0  // close выше hard SL (98.5) — по close выхода бы не было
	in.barLow = 98.4 // но low пробил hard SL внутри бара
	sig := s.decide(in)
	if sig.Kind != model.SignalSell || sig.Reason != "SL" {
		t.Fatalf("low pierce of hard SL must sell SL, got %v/%q", sig.Kind, sig.Reason)
	}
	if sig.StopLoss != 98.5 {
		t.Errorf("stop = %v, want 98.5", sig.StopLoss)
	}
```

(d) `TestDecideBlockedByMinRR` — update the stale comment only (logic still blocks: risk is now `101 − 98.5 = 2.5`, reward `2`, RR `0.8 < 1.5`):

```go
	// Resistance close enough for room (2 ATR) but reward/risk under MinRR:
	// price 101, stop 98.5 (risk 2.5), target 103 (reward 2) -> RR 0.8 < 1.5.
```

- [ ] **Step 7: Run the full core package test suite**

Run: `go test ./internal/service/trading_strategy/levels/strategy/core/ -v`
Expected: PASS — new anchor tests pass, all previously-passing tests still pass.

- [ ] **Step 8: Commit**

```bash
git add internal/service/trading_strategy/levels/strategy/core/core.go internal/service/trading_strategy/levels/strategy/core/core_test.go
git commit -m "feat(levels): anchor hard stop to structural swing-low

Both the entry RR check and the live stop now use recentLow - SLMult*ATR,
fixing the prior entry-vs-support anchor mismatch.

Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>"
```

---

### Task 4: Set RUAL and generic defaults

**Files:**
- Modify: `internal/service/trading_strategy/levels/strategy/rusal/rusal.go:13`
- Modify: `internal/service/backtest/levels_registry.go:39`
- Test: `internal/service/backtest/levels_registry_test.go` (existing equality test must stay green)

- [ ] **Step 1: Set the RUAL default**

In `rusal.go` `DefaultParams()`, add after the `SLMult` line (line 21):

```go
		SwingLowWindow:    10,    // ~1.5 sessions of bars for the structural stop low
```

- [ ] **Step 2: Set the generic default to match**

In `levels_registry.go` `genericLevelsDefaults()`, add after the `SLMult` line (line 47):

```go
		SwingLowWindow:    10,
```

- [ ] **Step 3: Run the registry equality test**

Run: `go test ./internal/service/backtest/ -run TestLevels -v`
Expected: PASS — `genericLevelsDefaults() == levelsrusal.DefaultParams()` still holds (both set `SwingLowWindow: 10`).

- [ ] **Step 4: Commit**

```bash
git add internal/service/trading_strategy/levels/strategy/rusal/rusal.go internal/service/backtest/levels_registry.go
git commit -m "feat(levels): default SwingLowWindow=10 for RUAL and generic baseline

Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>"
```

---

### Task 5: Add `SwingLowWindow` axis to the calibration grid

Keep all existing axes (pruning should be data-driven after seeing results under the new stop, not guessed now). Grid grows 324 → 1296 combos.

**Files:**
- Modify: `data/params/rusal/levels_grid.json`

- [ ] **Step 1: Add the axis**

Replace the full contents of `data/params/rusal/levels_grid.json` with:

```json
{
  "SLMult": [1.0, 1.5, 2.0, 2.5],
  "TrailArmATR": [0.5, 1.0, 1.5],
  "TrailMult": [2.0, 2.5, 3.0],
  "MinRR": [1.2, 1.5, 2.0],
  "RoomATR": [1.5, 2.0, 2.5],
  "SwingLowWindow": [5, 10, 15, 20]
}
```

- [ ] **Step 2: Verify the JSON parses and the field is valid for the grid**

Run: `go vet ./... && python3 -m json.tool data/params/rusal/levels_grid.json`
Expected: valid JSON printed, no vet errors. (`SwingLowWindow` is a real `int` field on `core.Params`, so the reflection-based grid in `internal/service/backtest/calibrate.go` accepts it.)

- [ ] **Step 3: Commit**

```bash
git add data/params/rusal/levels_grid.json
git commit -m "feat(levels): sweep SwingLowWindow in the RUAL calibration grid

Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>"
```

---

### Task 6: Full verification and baseline comparison

**Files:** none (verification only)

- [ ] **Step 1: Run the full test suite**

Run: `go test ./...`
Expected: PASS across all packages.

- [ ] **Step 2: Run the RUAL levels backtest with the new default**

Run the project's usual levels backtest command for RUAL (single run, default params), writing into a fresh report dir, e.g.:

```bash
go run ./cmd/backtest -strategy levels -ticker RUAL -interval Hour1 -months 24 -out reports/level_swinglow
```

(`-months 24` matches the baseline's 2024-05 — 2026-06 span so the comparison is apples-to-apples. Verified flags in `cmd/backtest/main.go`: `-strategy`, `-ticker`, `-interval`, `-months`, `-out`.)
Expected: a `*_best.md` report is written under `reports/level_swinglow/`.

- [ ] **Step 3: Compare against baseline**

Open the new `*_best.md` and compare to baseline `reports/level_new/RUAL_levels_Hour1_20260607_224555_best.md` (PF 0.934, 35 trades, 22 `SL` exits). Record:
- number of `SL` exits (expect fewer early stop-outs),
- Profit factor (expect ≥ 0.934, ideally > 1.0),
- Net PnL and max drawdown.

- [ ] **Step 4: Report outcome**

Summarize the comparison for the user. If PF improved, propose running the calibration grid (`-calibrate data/params/rusal/levels_grid.json -metric expectancy`, optionally with `-test-months` for walk-forward). If PF did not improve, state it plainly as a negative result and recommend the walk-forward / instrument-change paths from the spec. Do not hard-code any calibrated winner without the user's go-ahead.

---

## Self-Review

**Spec coverage:**
- Structural swing-low anchor for entry + live stop → Task 3. ✓
- `recentLow` helper mirroring `recentHigh` → Task 1. ✓
- `SwingLowWindow` param + decideInput + Build + Lookback → Task 2. ✓
- RUAL default 10 → Task 4; generic baseline kept equal → Task 4. ✓
- Trail stop untouched → no task modifies the chandelier branch. ✓
- No `MinStopATR`/separate floor, no `Position` field (stateless recompute) → respected (Task 3 keeps per-bar recompute). ✓
- Grid axis `SwingLowWindow` → Task 5. ✓
- Tests: recentLow, live-stop anchor, entry↔live consistency, "not tighter than before" (TestDecideHardStopWiderThanEntryAnchor), Lookback → Tasks 1–3. ✓
- Success criterion vs baseline → Task 6. ✓

**Placeholder scan:** No TBD/TODO; every code step shows exact code. Task 6 Step 2 notes the implementer should reuse the prior backtest command if flag names differ — this is the one intentional environment-specific spot, with explicit fallback guidance.

**Type consistency:** `recentLow(lows []float64, window int) float64`, field `recentLow float64`, param `SwingLowWindow int` used consistently across Tasks 1–5. Stop formula `recentLow − SLMult×ATR` identical in entry and management branches (Task 3). Fixture `recentLow: 99.5` ⇒ stop `98.5` used consistently in all updated assertions.
