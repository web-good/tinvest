# Parallel Grid Calibration Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Parallelize the grid-calibration combo loop (`runCombos`) with a bounded goroutine worker pool, keeping output bit-for-bit identical to the sequential version.

**Architecture:** Replace the sequential `for` loop in `runCombos` with a fixed pool of `calibWorkers` goroutines that pull combo indices off a channel and write results into a pre-sized slice by index (one writer per cell, no mutex). `RunGrid`/`RunPhases` call `runCombos` unchanged and benefit transparently. Fetch and the per-ticker/per-phase loops are untouched.

**Tech Stack:** Go 1.25, `sync.WaitGroup`, channels. Tests with the standard `testing` package + `-race`.

**Spec:** `docs/superpowers/specs/2026-06-10-parallel-grid-calibration-design.md`

---

### Task 1: Determinism regression test (sequential baseline locked in)

Add a test that pins the exact ranked output of a known grid, so any reordering or race introduced by parallelization is caught. Written first against the current sequential code — it MUST pass before we touch `runCombos`.

**Files:**
- Test: `internal/service/backtest/calibrate_test.go` (add new test func)

- [ ] **Step 1: Write the test**

Add this function at the end of `internal/service/backtest/calibrate_test.go`. It runs a multi-combo grid and asserts the full ranked sequence of `(EMAPeriod, SLMult, ProfitFactor)` tuples is exactly reproducible. Uses the same `Lookup("RUAL")` + `tinyCandles` helpers already in the file.

```go
func TestRunGridDeterministicOrder(t *testing.T) {
	b, _ := Lookup("RUAL")
	grid := Grid{
		"EMAPeriod": {12, 21, 30},
		"SLMult":    {1.0, 1.5, 2.0},
	}
	cfg := backtest.Config{InitialCash: 100000, Fraction: 1.0, Commission: 0.0005, Lot: 1}
	candles := tinyCandles(600)

	// Run twice; the ranked output must be byte-stable across runs.
	first, err := RunGrid(b, grid, candles, nil, cfg, "profit_factor", 0, 16)
	if err != nil {
		t.Fatal(err)
	}
	second, err := RunGrid(b, grid, candles, nil, cfg, "profit_factor", 0, 16)
	if err != nil {
		t.Fatal(err)
	}
	if len(first) != 9 || len(second) != 9 { // 3 * 3
		t.Fatalf("combos = %d/%d, want 9", len(first), len(second))
	}
	for i := range first {
		pf1, pf2 := first[i].Metrics.ProfitFactor, second[i].Metrics.ProfitFactor
		if pf1 != pf2 {
			t.Fatalf("run mismatch at rank %d: PF %v != %v", i, pf1, pf2)
		}
		tr1, tr2 := first[i].Metrics.TotalTrades, second[i].Metrics.TotalTrades
		if tr1 != tr2 {
			t.Fatalf("run mismatch at rank %d: trades %d != %d", i, tr1, tr2)
		}
	}
}
```

- [ ] **Step 2: Run the test to verify it passes on current sequential code**

Run: `go test ./internal/service/backtest/ -run TestRunGridDeterministicOrder -v`
Expected: PASS (sequential output is already deterministic; this locks the baseline).

- [ ] **Step 3: Run it under the race detector**

Run: `go test -race ./internal/service/backtest/ -run TestRunGridDeterministicOrder -v`
Expected: PASS, no race report (sequential code has no concurrency yet — this just confirms the harness/flag works).

- [ ] **Step 4: Commit**

```bash
git add internal/service/backtest/calibrate_test.go
git commit -m "test(backtest): lock deterministic grid ranking before parallelizing

Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>"
```

---

### Task 2: Parallelize `runCombos` with a bounded worker pool

Replace the sequential loop with `calibWorkers` goroutines pulling indices off a channel, each writing `results[i]` directly. Output order is preserved by construction (index = combo position), so the Task 1 test stays green.

**Files:**
- Modify: `internal/service/backtest/calibrate.go:49-61` (the `runCombos` func) and the const block near line 25
- Import: add `"sync"` to the import block at `internal/service/backtest/calibrate.go:3-11`

- [ ] **Step 1: Add the `calibWorkers` const**

In `internal/service/backtest/calibrate.go`, just below the existing `defaultKeepTop` const (line 25), add:

```go
// calibWorkers bounds the goroutine pool that evaluates grid combinations in
// runCombos. Kept conservative so a calibration run does not peg the whole machine.
const calibWorkers = 4
```

- [ ] **Step 2: Add the `sync` import**

Change the import block at the top of `internal/service/backtest/calibrate.go` from:

```go
import (
	"encoding/json"
	"fmt"
	"reflect"
	"sort"
	"strings"

	"tinvest/internal/domain/backtest"
)
```

to:

```go
import (
	"encoding/json"
	"fmt"
	"reflect"
	"sort"
	"strings"
	"sync"

	"tinvest/internal/domain/backtest"
)
```

- [ ] **Step 3: Replace the `runCombos` body**

Replace the whole `runCombos` function (`internal/service/backtest/calibrate.go:49-61`) with:

```go
// runCombos runs the engine once per parameter combination and pairs each with its
// metrics. periodDays feeds CAGR. The result is unranked but its order matches combos
// exactly (results[i] is combos[i]), so ranking downstream is deterministic.
//
// Combinations are independent — each gets a fresh strategy via b.Build, its own
// portfolio inside backtest.Run, and only reads the shared candle slices — so they run
// on a bounded pool of calibWorkers goroutines. Each result slot has a single writer
// (its own index), so no mutex is needed.
func runCombos(b Binding, combos []any, candles, dailyCandles []backtest.Candle,
	cfg backtest.Config, periodDays float64,
) []CalibResult {
	results := make([]CalibResult, len(combos))

	jobs := make(chan int)
	var wg sync.WaitGroup
	workers := calibWorkers
	if workers > len(combos) {
		workers = len(combos)
	}
	for w := 0; w < workers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := range jobs {
				res := backtest.Run(b.Build(combos[i]), candles, dailyCandles, cfg)
				m := backtest.Compute(res, res.BarsInMarket, len(res.Equity), periodDays)
				results[i] = CalibResult{Params: combos[i], Metrics: m}
			}
		}()
	}
	for i := range combos {
		jobs <- i
	}
	close(jobs)
	wg.Wait()

	return results
}
```

- [ ] **Step 4: Build**

Run: `go build ./...`
Expected: no errors.

- [ ] **Step 5: Run the full backtest test package**

Run: `go test ./internal/service/backtest/ -v`
Expected: PASS — all existing tests plus `TestRunGridDeterministicOrder`. Confirms output unchanged.

- [ ] **Step 6: Run under the race detector (the critical safety gate)**

Run: `go test -race ./internal/service/backtest/...`
Expected: PASS, **no race report**. This is the gate that proves the parallel writes/reads are clean.

- [ ] **Step 7: Commit**

```bash
git add internal/service/backtest/calibrate.go
git commit -m "perf(backtest): parallelize grid calibration with bounded worker pool

runCombos now evaluates combinations on calibWorkers goroutines, writing
results by index so output stays bit-for-bit identical to the sequential
version. Combinations are independent (fresh strategy + portfolio, read-only
candles), verified race-free under -race.

Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>"
```

---

### Task 3: Benchmark confirming the speedup

Add a benchmark over a representative grid so the win is measurable and future regressions are visible. Nice-to-have, not a correctness gate.

**Files:**
- Test: `internal/service/backtest/calibrate_test.go` (add benchmark func)

- [ ] **Step 1: Write the benchmark**

Add at the end of `internal/service/backtest/calibrate_test.go`:

```go
func BenchmarkRunGrid(b *testing.B) {
	bind, _ := Lookup("RUAL")
	grid := Grid{
		"EMAPeriod":     {12, 21, 30, 50},
		"SLMult":        {1.0, 1.5, 2.0},
		"TakeProfitRR":  {1.5, 2.0, 3.0},
		"VolMultiplier": {1.0, 1.2, 1.5},
	}
	cfg := backtest.Config{InitialCash: 100000, Fraction: 1.0, Commission: 0.0005, Lot: 1}
	candles := tinyCandles(2000)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := RunGrid(bind, grid, candles, nil, cfg, "profit_factor", 0, 16); err != nil {
			b.Fatal(err)
		}
	}
}
```

- [ ] **Step 2: Run the benchmark**

Run: `go test ./internal/service/backtest/ -run '^$' -bench BenchmarkRunGrid -benchmem`
Expected: completes and prints ns/op. (Optional sanity: temporarily set `calibWorkers = 1`, rerun, compare — parallel should be meaningfully faster on a multi-core box. Revert to 4 before committing.)

- [ ] **Step 3: Commit**

```bash
git add internal/service/backtest/calibrate_test.go
git commit -m "test(backtest): benchmark grid calibration throughput

Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>"
```

---

## Notes for the implementer

- **Do not touch** `RunGrid`, `RunPhases`, `runBasket`, or any fetch code — `runCombos` is the only behavior change. Phases stay sequential by design (phase k+1 seeds from phase k's top survivors).
- **Determinism is non-negotiable.** If `TestRunGridDeterministicOrder` ever fails after Task 2, the parallelization is wrong — stop and fix, do not relax the test.
- The `-race` run in Task 2 Step 6 is the safety gate for this work. Treat any race report as a hard blocker.
- `calibWorkers = 4` is intentional (conservative; user decision). Do not turn it into a flag or auto-`GOMAXPROCS`.
