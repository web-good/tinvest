# Multi-phase grid calibration — design

## Problem

Backtest calibration (`cmd/backtest -calibrate <grid.json>`) sweeps a single flat
grid (`map[string][]float64`) as one cartesian product. For the momentum strategy a
realistic grid reaches ~70 000 combinations × ~0.69 s/run ≈ **13.5 hours**
single-threaded, and a large fraction of those runs are wasted on parameter
interactions that cancel out (e.g. `TakeProfitRR` is inert whenever `UseTrail=1`;
`MaxDriftATR` is inert whenever `SignalValidBars=0`).

The user wants the backtest itself to run a **staged** calibration: sweep a small set
of high-impact parameters first, keep the N best full parameter sets, then sweep a
second set of parameters layered on top of those survivors, and so on for an arbitrary
number of phases — picking the best combination at the end.

## Goals

- A JSON grid format that declares an ordered list of phases, each with its own
  sub-grid and a "keep top N" count.
- The runner executes phases sequentially: phase _k+1_ sweeps its grid on top of each
  of phase _k_'s top-N survivors.
- Cost is additive across phases (`|G1| + N·|G2| + N·|G3| + …`) instead of the
  multiplicative blow-up of one combined grid.
- The existing flat grid format keeps working unchanged (backward compatible).
- Arbitrary number of phases (N=1 phase reduces to today's behaviour).

## Non-goals

- Parallelising the run loop (orthogonal; may come later).
- Per-phase `metric` / `min-trades` overrides (YAGNI; global CLI flags for now).
- Per-phase intermediate report files (only stdout progress + one final report).
- Enforcing that phases sweep disjoint fields (allowed to overlap; later phase wins).

## JSON schema

New phased format:

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

Field semantics:

- `phases` — ordered, non-empty array. Phases run in array order.
- `phase.name` — optional label, used only in stdout progress and the report. Defaults
  to `phase-<index>` (1-based) when omitted.
- `phase.keepTop` — how many top-ranked full parameter sets survive into the next
  phase. Optional; defaults to **5**. Ignored on the last phase (it emits the full
  final ranking). A value `<= 0` is treated as the default.
- `phase.grid` — the same `map[string][]float64` shape as today. May be empty (the
  phase then just re-ranks/narrows the incoming seeds without sweeping anything new).

**Legacy flat format** (`{"EMAPeriod": [...], "SLMult": [...]}`) is still accepted: the
parser detects the absence of a top-level `phases` key and treats the whole object as a
single phase's grid.

## Run semantics

```
seeds := [ b.DefaultParams() ]            // one seed to start
for each phase in phases:
    combos := []                          // expand the phase grid over every seed
    for each seed in seeds:
        combos += expandGrid(seed, phase.grid)
    results := run+rank(combos, metric, minTrades)
    if phase is last:
        return results                    // full ranking → reports
    seeds := top keepTop Params from results
```

Key points:

- A seed is a **full** `Params` struct, so it already carries the values chosen in
  earlier phases. `expandGrid(seed, phase.grid)` overrides only the phase's fields on a
  copy of that seed — earlier choices are preserved.
- Ranking each phase reuses the existing `rankResults` (metric + `min-trades` floor), so
  a low-trade fluke seed cannot survive into the next phase.
- `expandGrid` already builds the cartesian product over a single base; multi-seed is
  just a loop calling it per seed and concatenating. No change to `expandGrid` itself.
- De-duplication across seeds is **not** performed. Overlapping seeds producing
  identical combos is possible only if two survivors collapse to the same values under
  the next grid; the cost is negligible and dropping it keeps the code simple.

### Cost

`|G1| + keepTop·|G2| + keepTop·|G3| + …`. For the example above:
`216 + 5·36 = 396` runs ≈ **4.5 min**, versus ~70 000 runs ≈ 13.5 h for the equivalent
combined grid.

## Components

### `internal/service/backtest/calibrate.go`

New types:

```go
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
```

New function:

```go
// RunPhases runs a staged calibration: each phase sweeps its grid over the top-KeepTop
// survivors of the previous phase. It returns the final phase's full ranking
// (best first). onProgress, if non-nil, is called once per phase with a summary.
func RunPhases(b Binding, phases []Phase, candles, dailyCandles []backtest.Candle,
    cfg backtest.Config, metric string, minTrades int, periodDays float64,
    onProgress func(PhaseProgress),
) ([]CalibResult, error)
```

- Validates `metric` once and that `phases` is non-empty.
- Defaults `KeepTop` to 5 when `<= 0`.
- Reuses `expandGrid` (per seed) and `rankResults` (per phase).
- `PhaseProgress` carries `{Index int; Name string; Combos int; Kept int; BestMetric float64}`
  for stdout reporting; the runner formats it.

`RunGrid` stays as-is for direct/test use and is internally equivalent to a single-phase
`RunPhases`.

### `cmd/backtest/main.go` — `runCalibration`

- Read the grid file, then **detect shape**: unmarshal into a struct with an optional
  `Phases` field. If `Phases` is present and non-empty → phased path. Otherwise
  unmarshal again as a flat `Grid` and wrap it as a single phase (`KeepTop` irrelevant).
- Convert to `[]Phase` and call `svc.RunPhases(...)` with an `onProgress` callback that
  prints one line per phase, e.g. `phase core: 216 combos -> kept 5 (best PF=1.842)`.
- Everything downstream (`_calibration.md` via `RenderCalibrationMarkdown`, `_best.md`
  for `results[0]`, walk-forward split via `-test-months`) is unchanged: all phases
  calibrate on `gridCandles`; the final `best.md` is still computed on `bestCandles`.

Walk-forward note: the phased calibration runs entirely on the in-sample `gridCandles`
window; only the single best combination is re-evaluated on the out-of-sample
`bestCandles` window for `best.md`, exactly as today.

## Error handling

- Empty `phases` array → error (`backtest: phased grid has no phases`).
- Unknown grid field or unsupported metric → existing `expandGrid` / `validateMetric`
  errors, surfaced from the first phase that hits them.
- A phase that yields zero results (e.g. all combos error) → propagate the underlying
  error; do not silently continue with empty seeds.
- Malformed JSON (neither valid phased nor valid flat) → wrapped `parse grid` error.

## Testing

Table-driven Go tests in `calibrate_test.go`, all using real in-process runs on
`tinyCandles` (no mocks):

- **Two-phase narrowing:** a phased grid where phase 1 sweeps one field and phase 2
  another; assert the final winner's phase-1 field equals one of the surviving values
  and its phase-2 field is swept. Assert total runs = `|G1| + keepTop·|G2|` via a
  counting `Build`/`onProgress` hook.
- **Seed carry-forward:** assert a phase-1 field value chosen as a survivor is preserved
  in the final results (phase 2 does not reset it to the default).
- **`keepTop` default + clamp:** omitted/`<=0` `keepTop` defaults to 5; `keepTop`
  larger than the number of phase results keeps all.
- **`min-trades` floor across phases:** a high-metric but low-trade phase-1 combo must
  not survive into phase 2.
- **Single phase == `RunGrid`:** a one-phase `RunPhases` returns the same ranking as the
  equivalent `RunGrid` call.
- **Legacy flat format still parses** (CLI-level detection): a flat grid JSON routes to
  the single-phase path. (Covered by a small parse/detect unit test.)
- **Empty phases errors.**

Follow TDD: write each test, watch it fail, implement minimally.

## Open questions

None outstanding. Arbitrary phase count and top-N-seeds-per-phase semantics are
confirmed.
