# RSI+EMA fresh-entry filter — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add a "freshness" entry gate to the `rsi_ema` strategy so the RSI↑50 cross does not fire on choppy re-entries where RSI has recently sat above the mid line.

**Architecture:** New pure, stateless gate inside `enter()`: within the last `EntryLookbackBars` bars before the cross bar, count bars with `RSI > RSIMid`; reject when that count `≥ EntryAboveMidLimit`. Two new grid-tunable int params. No new cross-bar state; timeframe-agnostic (window is in bars).

**Tech Stack:** Go 1.25; existing `internal/service/trading_strategy/rsi_ema/strategy/core`; reflection-based phased grid calibration (`internal/service/backtest`).

## Global Constraints

- Defaults: `EntryLookbackBars = 5`, `EntryAboveMidLimit = 3`.
- Window = the `EntryLookbackBars` bars immediately before the cross bar `i`: indices `i-1 … i-EntryLookbackBars`, clamped at index 0. Count bars with `rsi[j] > RSIMid` (strict).
- Reject the entry when the count `≥ EntryAboveMidLimit`.
- Filter DISABLED (never rejects) when `EntryAboveMidLimit <= 0` OR `EntryLookbackBars <= 0`.
- Core stays pure, stateless between bars, ticker-agnostic. Only entry logic changes; exits (`SL`/`EMAX`/`RSI70`/`RSI50`/`EOD`) are untouched.
- Design source of truth: `docs/superpowers/specs/2026-07-24-rsi-ema-fresh-entry-filter-design.md`.

---

### Task 1: Fresh-entry filter in the core

**Files:**
- Modify: `internal/service/trading_strategy/rsi_ema/strategy/core/core.go`
- Test: `internal/service/trading_strategy/rsi_ema/strategy/core/core_test.go`

**Interfaces:**
- Consumes: `indicators.RSISeries`, existing `Params`, `crossedUp`, `enter()`.
- Produces: `Params.EntryLookbackBars int`, `Params.EntryAboveMidLimit int`; methods `(*Strategy).barsAboveMid(rsi []float64, i int) int` and `(*Strategy).freshEntry(rsi []float64, i int) bool`.

- [ ] **Step 1: Write failing tests for defaults and the helper**

Add to `core_test.go`:

```go
func TestDefaultParamsFreshEntryFilter(t *testing.T) {
	p := DefaultParams()
	if p.EntryLookbackBars != 5 || p.EntryAboveMidLimit != 3 {
		t.Fatalf("fresh-entry defaults wrong: %+v", p)
	}
}

func TestFreshEntryFilter(t *testing.T) {
	s := NewWithParams("TEST", DefaultParams()) // window 5, limit 3
	cases := []struct {
		name string
		rsi  []float64 // indices 0..i-1 form the window when i = len(rsi)
		want bool      // true = entry allowed
	}{
		// 5-bar window ending just before the cross bar (i = len(rsi)).
		{"chop: 4 of 5 above -> reject", []float64{55, 55, 55, 55, 47}, false},
		{"fresh: 1 of 5 above -> allow", []float64{55, 47, 47, 47, 47}, true},
		{"boundary: exactly limit-1 (2) above -> allow", []float64{55, 55, 47, 47, 47}, true},
		{"boundary: exactly limit (3) above -> reject", []float64{55, 55, 55, 47, 47}, false},
		{"short history truncates, no panic", []float64{55, 47}, true}, // only 2 bars, 1 above < 3
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := s.freshEntry(tc.rsi, len(tc.rsi)); got != tc.want {
				t.Fatalf("freshEntry=%v want %v (above=%d)", got, tc.want, s.barsAboveMid(tc.rsi, len(tc.rsi)))
			}
		})
	}
}

func TestFreshEntryFilterDisabled(t *testing.T) {
	p := DefaultParams()
	p.EntryAboveMidLimit = 0 // off
	s := NewWithParams("TEST", p)
	if !s.freshEntry([]float64{55, 55, 55, 55, 47}, 5) {
		t.Fatalf("EntryAboveMidLimit<=0 must disable the filter")
	}
	p2 := DefaultParams()
	p2.EntryLookbackBars = 0 // off
	s2 := NewWithParams("TEST", p2)
	if !s2.freshEntry([]float64{55, 55, 55, 55, 47}, 5) {
		t.Fatalf("EntryLookbackBars<=0 must disable the filter")
	}
}
```

- [ ] **Step 2: Run the tests, verify they fail to compile / fail**

Run: `go test ./internal/service/trading_strategy/rsi_ema/strategy/core/ -run 'FreshEntry|DefaultParamsFreshEntry' -v`
Expected: FAIL — `EntryLookbackBars`/`EntryAboveMidLimit`/`freshEntry`/`barsAboveMid` undefined.

- [ ] **Step 3: Add the params and defaults**

In `core.go`, add to the `Params` struct (after `DayEndMin`):

```go
	EntryLookbackBars  int     // fresh-entry window: bars before the cross to inspect (grid; default 5)
	EntryAboveMidLimit int     // reject entry when >= this many bars in the window are above RSIMid; <=0 disables (grid; default 3)
```

In `DefaultParams()`, add (after `DayEndMin: 1380,`):

```go
		EntryLookbackBars:  5,
		EntryAboveMidLimit: 3,
```

- [ ] **Step 4: Add the helpers**

In `core.go`, add near `crossedUp` (before `enter`):

```go
// barsAboveMid counts, within the EntryLookbackBars bars immediately before bar i (indices
// i-EntryLookbackBars .. i-1, clamped at 0), how many had RSI strictly above RSIMid.
func (s *Strategy) barsAboveMid(rsi []float64, i int) int {
	start := i - s.p.EntryLookbackBars
	if start < 0 {
		start = 0
	}
	n := 0
	for j := start; j < i && j < len(rsi); j++ {
		if rsi[j] > s.p.RSIMid {
			n++
		}
	}
	return n
}

// freshEntry reports whether the RSI cross at bar i is "fresh": fewer than EntryAboveMidLimit
// of the preceding EntryLookbackBars bars sat above RSIMid. This rejects choppy re-entries where
// RSI only briefly dipped below the mid line after a sustained run above it. Disabled (always
// true) when EntryAboveMidLimit<=0 or EntryLookbackBars<=0.
func (s *Strategy) freshEntry(rsi []float64, i int) bool {
	if s.p.EntryAboveMidLimit <= 0 || s.p.EntryLookbackBars <= 0 {
		return true
	}
	return s.barsAboveMid(rsi, i) < s.p.EntryAboveMidLimit
}
```

- [ ] **Step 5: Run Step-1 tests, verify PASS**

Run: `go test ./internal/service/trading_strategy/rsi_ema/strategy/core/ -run 'FreshEntry|DefaultParamsFreshEntry' -v`
Expected: PASS.

- [ ] **Step 6: Wire the gate into `enter()`**

In `core.go` `enter()`, after the EMA trend check (the `if !ok || fast[i] <= slow[i]` block that returns `sig`) and before the optional ATR stop (`entry := md.Closes[i]`), insert:

```go
	// 3.5 freshness filter: reject re-entries where RSI recently sat above the mid line
	// (short dip after a sustained run above 50 — chop around the mid, not a fresh reset).
	if !s.freshEntry(rsi, i) {
		return sig
	}
```

- [ ] **Step 7: Write the integration test — a real chop re-entry is filtered**

Add to `core_test.go` (add `"tinvest/pkg/indicators"` to the test imports):

```go
// TestEnterFilterRejectsChopReentry finds a bar that the filter-OFF strategy buys but where the
// preceding window holds >= EntryAboveMidLimit bars above the mid, and asserts the default
// (filter-ON) strategy rejects that exact bar.
func TestEnterFilterRejectsChopReentry(t *testing.T) {
	off := DefaultParams()
	off.EntryAboveMidLimit = 0 // filter off
	sOff := NewWithParams("TEST", off)
	on := NewWithParams("TEST", DefaultParams()) // filter on (window 5, limit 3)
	closes, highs, lows := driftWalk(1500, 7)
	end := mskAt(2026, 7, 20, 12, 0)
	for k := 60; k < len(closes); k++ {
		md := mdEndingAt(closes, highs, lows, k, end, nil)
		if sOff.Decide(md).Kind != model.SignalBuy {
			continue
		}
		rsi := indicators.RSISeries(md.Closes, DefaultParams().RSIPeriod)
		if on.barsAboveMid(rsi, k) >= DefaultParams().EntryAboveMidLimit {
			if on.Decide(md).Kind == model.SignalBuy {
				t.Fatalf("chop re-entry at bar %d must be filtered out", k)
			}
			return // found and verified a chop candidate
		}
	}
	t.Fatalf("no chop re-entry candidate found in series (adjust the driftWalk seed)")
}
```

- [ ] **Step 8: Run the integration test**

Run: `go test ./internal/service/trading_strategy/rsi_ema/strategy/core/ -run TestEnterFilterRejectsChopReentry -v`
Expected: PASS. If it fails with "no chop re-entry candidate found", try seeds 7→3→9→11 in `driftWalk(1500, seed)` until a candidate is found; the assertion itself (filter rejects the candidate) must hold — if it does not, the gate wiring is wrong.

- [ ] **Step 9: Add the Explain line**

In `core.go` `Explain()`, after the EMA block (the `if ok { ... } else { ... }` printing EMA) and before the `StopATR` block, insert:

```go
	if len(rsi) == n {
		fmt.Fprintf(&sb, "фильтр свежести: баров выше %.0f в окне %d = %d (лимит %d); прошёл? %v\n",
			s.p.RSIMid, s.p.EntryLookbackBars, s.barsAboveMid(rsi, i), s.p.EntryAboveMidLimit, s.freshEntry(rsi, i))
	}
```

- [ ] **Step 10: Extend the Explain test**

In `core_test.go`, in `TestExplainReportsGates`, add `"фильтр свежести"` to the `want` slice:

```go
	for _, want := range []string{"сессия", "RSI", "EMA", "фильтр свежести"} {
```

- [ ] **Step 11: Run the full package test suite with race**

Run: `go test -race ./internal/service/trading_strategy/rsi_ema/...`
Expected: PASS (all existing tests still green — the filter only moves which bar `firstBuyBar` lands on; it never removes all buys from an 800-bar `driftWalk`).

- [ ] **Step 12: Commit**

```bash
git add internal/service/trading_strategy/rsi_ema/strategy/core/core.go internal/service/trading_strategy/rsi_ema/strategy/core/core_test.go
git commit -m "feat(rsi_ema): фильтр свежести входа — отсекать перезаходы-чоп у линии 50"
```

---

### Task 2: Grid sweep + documentation

**Files:**
- Modify: `data/params/rsi_ema/grid.json`
- Modify: `docs/rsi_ema/strategy.md`
- Modify: `docs/superpowers/specs/2026-07-23-rsi-ema-design.md`

**Interfaces:**
- Consumes: `Params.EntryLookbackBars`, `Params.EntryAboveMidLimit` from Task 1 (reflection grid keys must match these field names exactly).

- [ ] **Step 1: Add the freshness phase to the grid**

In `data/params/rsi_ema/grid.json`, append a fourth phase after `risk` (make `risk` non-terminal). The new terminal phase:

```json
    {
      "name": "freshness",
      "grid": {
        "EntryLookbackBars": [4, 5, 6],
        "EntryAboveMidLimit": [0, 2, 3, 4]
      }
    }
```

Update the `_comment` to describe phase 4 (freshness filter; `EntryAboveMidLimit=0` = filter off as the control point) and the combo count: 27 (entry) + 6×12 (exits) + 6×4 (risk, keepTop defaults to 5) + 5×12 (freshness) = 183 combos.

- [ ] **Step 2: Validate the grid JSON parses**

Run: `python3 -c "import json;json.load(open('data/params/rsi_ema/grid.json'));print('ok')"`
Expected: `ok`.

- [ ] **Step 3: Smoke-test that calibration accepts the new keys**

Run: `go run ./cmd/backtest -ticker SBER -strategy rsi_ema -interval Minutes15 -calibrate data/params/rsi_ema/grid.json -out ./reports/SBER -months 6 -min-trades 5`
Expected: runs without an "unknown parameter" / reflection error and writes a report. (Result quality is not judged here — only that the new grid keys bind.)

- [ ] **Step 4: Document the filter in the strategy reference**

In `docs/rsi_ema/strategy.md`, under **Вход**, add a rule describing the freshness filter (window `EntryLookbackBars`=5 before the cross, reject when `≥ EntryAboveMidLimit`=3 bars above `RSIMid`, `≤0` disables), and add both params to the grid-phase description (`entry` / `exits` / `risk` / `freshness`) with the updated combo count (183).

- [ ] **Step 5: Cross-reference the design spec**

In `docs/superpowers/specs/2026-07-23-rsi-ema-design.md`, in the entry rules, add a short note that the RSI↑50 cross is additionally gated by the fresh-entry filter, linking `docs/superpowers/specs/2026-07-24-rsi-ema-fresh-entry-filter-design.md`.

- [ ] **Step 6: Commit**

```bash
git add data/params/rsi_ema/grid.json docs/rsi_ema/strategy.md docs/superpowers/specs/2026-07-23-rsi-ema-design.md
git commit -m "feat(rsi_ema): грид-фаза и доки для фильтра свежести входа"
```

---

## Notes for the executor

- The unrelated uncommitted WIP in `internal/service/trading_strategy/scalping_rsimacd/strategy/core/core.go` (StopATR/EnableStochExit default tweaks) MUST NOT be staged or committed by any task here. Stage only the files each step lists explicitly.
- `mage ci` currently fails ONLY because of that scalping_rsimacd WIP; verify this task's work with the scoped `go test -race ./internal/service/trading_strategy/rsi_ema/...` (and the Task 2 smoke run), not the full `mage ci`, unless the WIP has since been resolved.
