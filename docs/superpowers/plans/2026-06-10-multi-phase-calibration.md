# Multi-phase grid calibration Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Let the backtest calibrator run a staged (multi-phase) grid search where each phase sweeps its sub-grid over the top-N survivors of the previous phase, with an arbitrary number of phases.

**Architecture:** Add `Phase`/`PhasedGrid`/`PhaseProgress` types and a `RunPhases` orchestrator to `internal/service/backtest/calibrate.go`, reusing the existing `expandGrid` (now invoked per seed) and `rankResults`. Extract a small `runCombos` helper so `RunGrid` and `RunPhases` share the run loop. Add a format-detecting `ParsePhases` so the CLI accepts both the new phased JSON and the legacy flat grid. Wire `cmd/backtest/main.go`'s `runCalibration` to parse + run phases and print per-phase progress.

**Tech Stack:** Go 1.25, standard `encoding/json`, existing pure backtest engine (`internal/domain/backtest`).

---

## Spec

See `docs/superpowers/specs/2026-06-10-multi-phase-calibration-design.md`.

## File Structure

- `internal/service/backtest/calibrate.go` — **modify**: add `Phase`, `PhasedGrid`, `PhaseProgress` types, `defaultKeepTop` const, `runCombos` helper, `RunPhases`, `ParsePhases`; add `encoding/json` import.
- `internal/service/backtest/calibrate_test.go` — **modify**: add tests for `RunPhases` and `ParsePhases`.
- `cmd/backtest/main.go` — **modify**: `runCalibration` switches from `RunGrid` on a flat grid to `ParsePhases` + `RunPhases` with an `onProgress` printer; remove the now-unused direct `encoding/json` usage and import.

---

## Task 1: Extract `runCombos`, add phase types and `RunPhases`

**Files:**
- Modify: `internal/service/backtest/calibrate.go`
- Test: `internal/service/backtest/calibrate_test.go`

- [ ] **Step 1: Extract the run loop into `runCombos` and reuse it in `RunGrid`**

In `internal/service/backtest/calibrate.go`, add this helper (place it directly above `RunGrid`):

```go
// runCombos runs the engine once per parameter combination and pairs each with its
// metrics. periodDays feeds CAGR. The result is unranked.
func runCombos(b Binding, combos []any, candles, dailyCandles []backtest.Candle,
	cfg backtest.Config, periodDays float64,
) []CalibResult {
	results := make([]CalibResult, 0, len(combos))
	for _, params := range combos {
		res := backtest.Run(b.Build(params), candles, dailyCandles, cfg)
		m := backtest.Compute(res, res.BarsInMarket, len(res.Equity), periodDays)
		results = append(results, CalibResult{Params: params, Metrics: m})
	}
	return results
}
```

Then replace the body of `RunGrid`'s run loop. Change this block:

```go
	results := make([]CalibResult, 0, len(combos))
	for _, params := range combos {
		res := backtest.Run(b.Build(params), candles, dailyCandles, cfg)
		m := backtest.Compute(res, res.BarsInMarket, len(res.Equity), periodDays)
		results = append(results, CalibResult{Params: params, Metrics: m})
	}
	return rankResults(results, metric, minTrades), nil
```

to:

```go
	results := runCombos(b, combos, candles, dailyCandles, cfg, periodDays)
	return rankResults(results, metric, minTrades), nil
```

- [ ] **Step 2: Run existing tests to confirm the refactor is green**

Run: `go test ./internal/service/backtest/ -run 'RunGrid|ApplyField|RankResults'`
Expected: PASS (refactor preserves behavior).

- [ ] **Step 3: Write the failing test for `RunPhases` (run-count + seed carry-forward)**

Add to `internal/service/backtest/calibrate_test.go`:

```go
func TestRunPhasesNarrowsAndCarriesSeeds(t *testing.T) {
	b, _ := Lookup("RUAL")
	cfg := backtest.Config{InitialCash: 100000, Fraction: 1.0, Commission: 0.0005, Lot: 1}
	phases := []Phase{
		{Name: "core", KeepTop: 2, Grid: Grid{"EMAPeriod": {44, 55}}},      // |G1| = 2
		{Name: "gates", Grid: Grid{"SLMult": {1.0, 1.5, 2.0}}},             // |G2| = 3
	}
	var prog []PhaseProgress
	results, err := RunPhases(b, phases, tinyCandles(400), nil, cfg, "profit_factor", 0, 16,
		func(p PhaseProgress) { prog = append(prog, p) })
	if err != nil {
		t.Fatal(err)
	}
	// Additive cost: phase 1 runs |G1|; phase 2 runs keepTop*|G2|.
	if len(prog) != 2 {
		t.Fatalf("want 2 phase callbacks, got %d", len(prog))
	}
	if prog[0].Combos != 2 {
		t.Fatalf("phase 1 combos = %d, want 2", prog[0].Combos)
	}
	if prog[0].Kept != 2 {
		t.Fatalf("phase 1 kept = %d, want 2", prog[0].Kept)
	}
	if prog[1].Combos != 6 { // keepTop(2) * |G2|(3)
		t.Fatalf("phase 2 combos = %d, want 6", prog[1].Combos)
	}
	if len(results) != 6 {
		t.Fatalf("final results = %d, want 6", len(results))
	}
	// Seed carry-forward: every final combo keeps a phase-1 EMAPeriod value (44 or 55),
	// proving phase 2 layered on the survivors instead of resetting to the default.
	for i, r := range results {
		ema := r.Params.(adaptive.Params).EMAPeriod
		if ema != 44 && ema != 55 {
			t.Fatalf("result %d EMAPeriod = %d, want carried 44 or 55", i, ema)
		}
	}
}
```

- [ ] **Step 4: Run the test to verify it fails**

Run: `go test ./internal/service/backtest/ -run TestRunPhasesNarrowsAndCarriesSeeds`
Expected: FAIL — `undefined: Phase` / `undefined: RunPhases` / `undefined: PhaseProgress` (compile error).

- [ ] **Step 5: Add the types and `RunPhases`**

In `internal/service/backtest/calibrate.go`, add after the `CalibResult` type:

```go
// defaultKeepTop is how many top-ranked parameter sets carry into the next phase
// when a phase does not set KeepTop.
const defaultKeepTop = 5

// Phase is one stage of a staged calibration: a sub-grid plus how many top-ranked
// parameter sets survive into the next phase.
type Phase struct {
	Name    string `json:"name"`
	KeepTop int    `json:"keepTop"`
	Grid    Grid   `json:"grid"`
}

// PhasedGrid is the ordered list of calibration phases.
type PhasedGrid struct {
	Phases []Phase `json:"phases"`
}

// PhaseProgress reports one phase's outcome to a RunPhases caller (e.g. for stdout).
type PhaseProgress struct {
	Index      int     // 1-based phase index
	Name       string  // phase name (defaults to phase-<index>)
	Combos     int     // combinations run in this phase
	Kept       int     // survivors carried forward (clamped to results; full ranking on the last phase)
	BestMetric float64 // best metric value after this phase
}
```

Then add `RunPhases` after `RunGrid`:

```go
// RunPhases runs a staged calibration: phase k+1 sweeps its grid over the top-KeepTop
// survivors of phase k. It returns the final phase's full ranking (best first).
// onProgress, when non-nil, is called once per phase. metric and minTrades are global
// across phases; minTrades floors every phase's ranking so a low-trade fluke cannot
// survive forward.
func RunPhases(b Binding, phases []Phase, candles, dailyCandles []backtest.Candle,
	cfg backtest.Config, metric string, minTrades int, periodDays float64,
	onProgress func(PhaseProgress),
) ([]CalibResult, error) {
	if err := validateMetric(metric); err != nil {
		return nil, err
	}
	if len(phases) == 0 {
		return nil, fmt.Errorf("backtest: phased grid has no phases")
	}
	seeds := []any{b.DefaultParams()}
	var results []CalibResult
	for i, ph := range phases {
		combos := make([]any, 0, len(seeds))
		for _, seed := range seeds {
			expanded, err := expandGrid(seed, ph.Grid)
			if err != nil {
				return nil, err
			}
			combos = append(combos, expanded...)
		}
		results = rankResults(runCombos(b, combos, candles, dailyCandles, cfg, periodDays), metric, minTrades)

		keep := ph.KeepTop
		if keep <= 0 {
			keep = defaultKeepTop
		}
		if keep > len(results) {
			keep = len(results)
		}
		if onProgress != nil {
			name := ph.Name
			if name == "" {
				name = fmt.Sprintf("phase-%d", i+1)
			}
			var best float64
			if len(results) > 0 {
				best = metricValue(results[0].Metrics, metric)
			}
			onProgress(PhaseProgress{Index: i + 1, Name: name, Combos: len(combos), Kept: keep, BestMetric: best})
		}
		if i < len(phases)-1 { // carry survivors into the next phase
			seeds = make([]any, 0, keep)
			for _, r := range results[:keep] {
				seeds = append(seeds, r.Params)
			}
		}
	}
	return results, nil
}
```

- [ ] **Step 6: Run the test to verify it passes**

Run: `go test ./internal/service/backtest/ -run TestRunPhasesNarrowsAndCarriesSeeds`
Expected: PASS.

- [ ] **Step 7: Add regression-guard tests for keepTop default/clamp, equivalence, and empty phases**

Add to `internal/service/backtest/calibrate_test.go`:

```go
func TestRunPhasesKeepTopDefaultsAndClamps(t *testing.T) {
	b, _ := Lookup("RUAL")
	cfg := backtest.Config{InitialCash: 100000, Fraction: 1.0, Commission: 0.0005, Lot: 1}
	// Phase 1 sweeps 2 combos. KeepTop omitted on phase 1 -> defaults to 5 but clamps
	// to the 2 available results. Phase 2 then runs keptTop(2) * |G2|(2) = 4.
	phases := []Phase{
		{Grid: Grid{"EMAPeriod": {44, 55}}},
		{Grid: Grid{"SLMult": {1.0, 1.5}}},
	}
	var prog []PhaseProgress
	results, err := RunPhases(b, phases, tinyCandles(400), nil, cfg, "profit_factor", 0, 16,
		func(p PhaseProgress) { prog = append(prog, p) })
	if err != nil {
		t.Fatal(err)
	}
	if prog[0].Kept != 2 {
		t.Fatalf("phase 1 kept = %d, want 2 (default 5 clamped to 2 results)", prog[0].Kept)
	}
	if prog[1].Combos != 4 {
		t.Fatalf("phase 2 combos = %d, want 4", prog[1].Combos)
	}
	if len(results) != 4 {
		t.Fatalf("final results = %d, want 4", len(results))
	}
	if prog[0].Name != "phase-1" {
		t.Fatalf("phase 1 default name = %q, want phase-1", prog[0].Name)
	}
}

func TestRunPhasesSinglePhaseMatchesRunGrid(t *testing.T) {
	b, _ := Lookup("RUAL")
	cfg := backtest.Config{InitialCash: 100000, Fraction: 1.0, Commission: 0.0005, Lot: 1}
	grid := Grid{"EMAPeriod": {44, 55}, "SLMult": {1.0, 1.5}}
	want, err := RunGrid(b, grid, tinyCandles(400), nil, cfg, "profit_factor", 0, 16)
	if err != nil {
		t.Fatal(err)
	}
	got, err := RunPhases(b, []Phase{{Grid: grid}}, tinyCandles(400), nil, cfg, "profit_factor", 0, 16, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != len(want) {
		t.Fatalf("len(got) = %d, want %d", len(got), len(want))
	}
	if !reflect.DeepEqual(got[0].Params, want[0].Params) {
		t.Fatalf("top params differ:\ngot  %+v\nwant %+v", got[0].Params, want[0].Params)
	}
}

func TestRunPhasesEmptyErrors(t *testing.T) {
	b, _ := Lookup("RUAL")
	cfg := backtest.Config{InitialCash: 100000, Fraction: 1.0, Lot: 1}
	if _, err := RunPhases(b, nil, tinyCandles(400), nil, cfg, "profit_factor", 0, 16, nil); err == nil {
		t.Fatal("expected error for empty phases")
	}
}
```

Add `"reflect"` to the test file's import block (alongside `"strings"`, `"testing"`, `"time"`).

- [ ] **Step 8: Run the new tests to verify they pass**

Run: `go test ./internal/service/backtest/ -run 'TestRunPhases'`
Expected: PASS (all four `TestRunPhases*` tests).

- [ ] **Step 9: Commit**

```bash
git add internal/service/backtest/calibrate.go internal/service/backtest/calibrate_test.go
git commit -m "feat(backtest): staged multi-phase grid calibration

Add RunPhases: each phase sweeps its grid over the top-KeepTop survivors
of the previous phase, for an arbitrary number of phases. Extract
runCombos so RunGrid and RunPhases share the run loop.

Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>"
```

---

## Task 2: `ParsePhases` format detection

**Files:**
- Modify: `internal/service/backtest/calibrate.go` (add `encoding/json` import + `ParsePhases`)
- Test: `internal/service/backtest/calibrate_test.go`

- [ ] **Step 1: Write the failing test for `ParsePhases`**

Add to `internal/service/backtest/calibrate_test.go`:

```go
func TestParsePhasesPhasedFormat(t *testing.T) {
	raw := []byte(`{"phases":[{"name":"core","keepTop":3,"grid":{"EMAPeriod":[44,55]}},{"grid":{"SLMult":[1.0,1.5]}}]}`)
	phases, err := ParsePhases(raw)
	if err != nil {
		t.Fatal(err)
	}
	if len(phases) != 2 {
		t.Fatalf("phases = %d, want 2", len(phases))
	}
	if phases[0].Name != "core" || phases[0].KeepTop != 3 {
		t.Fatalf("phase 0 = %+v, want name=core keepTop=3", phases[0])
	}
	if got := phases[0].Grid["EMAPeriod"]; len(got) != 2 || got[0] != 44 {
		t.Fatalf("phase 0 grid EMAPeriod = %v, want [44 55]", got)
	}
}

func TestParsePhasesLegacyFlatFormat(t *testing.T) {
	raw := []byte(`{"EMAPeriod":[44,55],"SLMult":[1.0,1.5]}`)
	phases, err := ParsePhases(raw)
	if err != nil {
		t.Fatal(err)
	}
	if len(phases) != 1 {
		t.Fatalf("phases = %d, want 1 (flat wrapped as single phase)", len(phases))
	}
	if got := phases[0].Grid["EMAPeriod"]; len(got) != 2 {
		t.Fatalf("flat grid EMAPeriod = %v, want 2 values", got)
	}
}

func TestParsePhasesMalformedErrors(t *testing.T) {
	if _, err := ParsePhases([]byte(`{not json`)); err == nil {
		t.Fatal("expected error for malformed JSON")
	}
}
```

- [ ] **Step 2: Run the test to verify it fails**

Run: `go test ./internal/service/backtest/ -run TestParsePhases`
Expected: FAIL — `undefined: ParsePhases` (compile error).

- [ ] **Step 3: Add the `encoding/json` import and `ParsePhases`**

In `internal/service/backtest/calibrate.go`, add `"encoding/json"` to the import block (it currently imports `"fmt"`, `"reflect"`, `"sort"`, `"strings"`). Then add:

```go
// ParsePhases decodes a calibration grid file into ordered phases. It accepts the
// phased format ({"phases":[{grid, keepTop, name}, ...]}) and the legacy flat format
// ({"Field":[...], ...}), wrapping the latter as a single phase.
func ParsePhases(raw []byte) ([]Phase, error) {
	var pg PhasedGrid
	if err := json.Unmarshal(raw, &pg); err == nil && len(pg.Phases) > 0 {
		return pg.Phases, nil
	}
	var flat Grid
	if err := json.Unmarshal(raw, &flat); err != nil {
		return nil, fmt.Errorf("backtest: parse grid: %w", err)
	}
	return []Phase{{Grid: flat}}, nil
}
```

- [ ] **Step 4: Run the test to verify it passes**

Run: `go test ./internal/service/backtest/ -run TestParsePhases`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/service/backtest/calibrate.go internal/service/backtest/calibrate_test.go
git commit -m "feat(backtest): parse phased and legacy flat calibration grids

Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>"
```

---

## Task 3: Wire `runCalibration` to phased calibration

**Files:**
- Modify: `cmd/backtest/main.go`

- [ ] **Step 1: Replace flat-grid parse + RunGrid with ParsePhases + RunPhases**

In `cmd/backtest/main.go`, inside `runCalibration`, replace this block:

```go
	var grid svc.Grid
	if err := json.Unmarshal(raw, &grid); err != nil {
		return fmt.Errorf("parse grid: %w", err)
	}
```

with:

```go
	phases, err := svc.ParsePhases(raw)
	if err != nil {
		return err
	}
```

Then replace this line:

```go
	results, err := svc.RunGrid(b, grid, gridCandles, gridDaily, cfg, metric, minTrades, gridDays)
	if err != nil {
		return err
	}
```

with:

```go
	results, err := svc.RunPhases(b, phases, gridCandles, gridDaily, cfg, metric, minTrades, gridDays,
		func(p svc.PhaseProgress) {
			fmt.Printf("phase %s: %d combos -> kept %d (best %s=%.4g)\n", p.Name, p.Combos, p.Kept, metric, p.BestMetric)
		})
	if err != nil {
		return err
	}
```

- [ ] **Step 2: Remove the now-unused `encoding/json` import**

`encoding/json` was only used by the line deleted in Step 1. Remove `"encoding/json"` from the import block in `cmd/backtest/main.go`.

- [ ] **Step 3: Build the backtest command**

Run: `go build ./cmd/backtest/`
Expected: builds with no errors (no "imported and not used", no undefined symbols).

- [ ] **Step 4: Run the full backtest package test suite**

Run: `go test ./internal/service/backtest/ ./cmd/backtest/...`
Expected: PASS.

- [ ] **Step 5: Smoke-test the phased format end to end on cached candles**

Create a temporary phased grid and run the calibrator against the warm RUAL hourly cache (no network; uses `data/candles/RUAL_Hour1.json`).

Run:
```bash
cat > /tmp/phased_grid.json <<'JSON'
{
  "phases": [
    { "name": "core", "keepTop": 3, "grid": { "EMAPeriod": [200, 100], "SLMult": [0.5, 1.0] } },
    { "name": "gates", "grid": { "DailyTrendPeriod": [0, 20], "SignalValidBars": [0, 2] } }
  ]
}
JSON
go run ./cmd/backtest -strategy momentum -ticker RUAL -calibrate /tmp/phased_grid.json -metric profit_factor
```
Expected: stdout shows two `phase ...: N combos -> kept K (best profit_factor=...)` lines (phase `core` with 4 combos, phase `gates` with 3·4 = 12 combos), then a `calibration: ..._calibration.md (combos=12, ...)` line. Open the `_calibration.md` and confirm it ranks 12 combinations with per-combo params. (If the command needs env/config to construct the provider, run it the same way the existing single-grid calibration is normally run in this repo.)

- [ ] **Step 6: Commit**

```bash
git add cmd/backtest/main.go
git commit -m "feat(backtest): run phased calibration from the CLI

runCalibration now parses the grid via ParsePhases (phased or legacy flat)
and drives RunPhases, printing per-phase narrowing progress.

Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>"
```

---

## Task 4: Convert the momentum grids to the phased format

**Files:**
- Modify: `data/params/rusal/momentum_grid.json`
- Modify: `data/params/afks/momentum_grid.json`

- [ ] **Step 1: Rewrite both momentum grids as two phases**

Write this content to **both** `data/params/rusal/momentum_grid.json` and `data/params/afks/momentum_grid.json`:

```json
{
  "phases": [
    {
      "name": "core",
      "keepTop": 5,
      "grid": {
        "EMAPeriod": [200, 100],
        "SLMult": [0.5, 1.0, 1.5],
        "TakeProfitRR": [1.5, 2.0, 3.0],
        "VolMultiplier": [1.0, 1.2, 1.5],
        "MaxDailyATRUsed": [0.5, 0.7],
        "MACDBelowZeroOnly": [0, 1]
      }
    },
    {
      "name": "gates",
      "grid": {
        "MACDFast": [12, 8],
        "DailyTrendPeriod": [0, 10, 20],
        "SignalValidBars": [0, 2, 4],
        "MaxDriftATR": [0, 1.0]
      }
    }
  ]
}
```

Cost: phase `core` = 2·3·3·3·2·2 = 216 combos; phase `gates` = keepTop(5)·(2·3·3·2) = 5·36 = 180 combos; total 396 runs (~4.5 min) versus ~70 000 for the equivalent combined grid.

- [ ] **Step 2: Validate both files parse as phases**

Run:
```bash
go run ./cmd/backtest -strategy momentum -ticker RUAL -calibrate data/params/rusal/momentum_grid.json -metric profit_factor
```
Expected: two `phase ...` progress lines (`core: 216 combos`, `gates: 180 combos`) and a final `calibration:` line. (Same env caveat as Task 3 Step 5.)

- [ ] **Step 3: Commit**

```bash
git add data/params/rusal/momentum_grid.json data/params/afks/momentum_grid.json
git commit -m "chore(momentum): convert calibration grids to phased format

Two phases (core entry/exit, then gates) cut the sweep from ~70k combos
to ~396 by carrying the top-5 phase-1 survivors into phase 2.

Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>"
```

---

## Self-Review Notes

- **Spec coverage:** JSON schema (Task 4 + types in Task 1), run semantics/seed carry-forward (Task 1), additive cost (Task 1 run-count test), backward compat via `ParsePhases` (Task 2), CLI wiring + per-phase stdout + unchanged walk-forward/report path (Task 3), arbitrary phase count (loop in `RunPhases`), keepTop default 5 / clamp (Task 1), empty-phases error (Task 1), min-trades floor reused per phase (`rankResults` in the `RunPhases` loop). All covered.
- **Type consistency:** `Phase{Name, KeepTop, Grid}`, `PhasedGrid{Phases}`, `PhaseProgress{Index, Name, Combos, Kept, BestMetric}`, `RunPhases(b, phases, candles, dailyCandles, cfg, metric, minTrades, periodDays, onProgress)`, `ParsePhases(raw) ([]Phase, error)`, `runCombos(b, combos, candles, dailyCandles, cfg, periodDays)` — names and signatures match across Tasks 1–3.
- **Placeholders:** none — every code/command step is concrete.
- **min-trades floor test:** intentionally not a separate test; `rankResults`'s floor already has dedicated coverage (`TestRankResultsMinTradesFloor`) and is reused unchanged inside the `RunPhases` loop.
```
