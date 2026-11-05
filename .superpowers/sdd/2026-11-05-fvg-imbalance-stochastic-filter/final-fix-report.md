# Final fix wave — 2026-11-05

Base: `8b66d0c0` (`docs(fvg_imbalance): describe stochastic entry filter`). All four requested findings are addressed in one fix commit. Work was performed in this session without subagents. The plan, spec, ledger, Task 1 report, review diff, actual current sources, and pre-existing worktree changes were inspected before implementation.

## Findings and fixes

### 1. Commit the complete dependency migration without restoring removed behavior

A pristine archive of HEAD reproduced the compile failure independently of the working tree:

```text
$ go test ./internal/service/trading_strategy/fvg_imbalance/... ./internal/service/backtest -run '^$'
unknown field RequireFirstRetest in struct literal of type core.Params
unknown field RequireBullishBOS in struct literal of type core.Params
unknown field RequireLiquiditySweep in struct literal of type core.Params
unknown field SwingK in struct literal of type core.Params
unknown field StructureLookbackBars in struct literal of type core.Params
unknown field MaxSweepToFVGBars in struct literal of type core.Params
unknown field MaxEntryDistanceATR in struct literal of type core.Params
unknown field MaxHoldBars in struct literal of type core.Params
core/engine_test.go:90:4: p.MaxHoldBars undefined
FAIL [build failed]
```

The starting worktree already removed these consumers in the two ticker literals and removed the obsolete time-exit test. Those necessary changes are included in the fix. Registry tests now reject the removed JSON keys instead of setting removed fields. No removed parameter, structure filter, entry-distance limit, first-retest restriction, or time exit was restored.

The actual starting literals for **both GAZP and DOMRF** are preserved: ATR 14, minimum gap .6, retest lifetime 10, depth 80%, target 2R, optional third higher high disabled, wick depth, H4 EMA 200, stop buffer 1. Each explicitly sets `UseStochFilter: 0`. The old test/report values were stale; ticker and registry expectations now match these literals. The CLI H4-selection test again expects H4 for the current default and partial overlay. That correction makes `cmd/backtest/main_test.go` identical to HEAD, so it is not in the commit.

Initial worktree verification reproduced `TestGAZPParams`, `TestFVGImbalanceDefaults/GAZP`, `TestFVGImbalanceDefaults/DOMRF`, and `TestFVGSelectedInputsRequireHTF/{default,partial_params_inherit_disabled_H4}` failures. They are resolved by updating expectations, without changing the intended ticker values.

### 2. Use fully smoothed slow K, with correct alignment

`pkg/indicators/StochasticSeries` returns raw K followed by its `dSmooth`-value simple moving average. With arguments `(14,3)`, the second series is the requested slow K. Its element zero belongs to input candle index **15**, because it averages raw K at indices 13, 14, and 15. No indicator/API implementation was changed, and no additional slow-D entry condition was added.

Core now uses that second series, requires strict `<20`, fails closed with fewer than 16 candles, and retains at least 16 warm-up candles plus `MaxRetestBars`. Disabled-filter lookback and entry behavior remain unchanged.

TDD evidence, written and run before the production change:

```text
$ go test ./internal/service/trading_strategy/fvg_imbalance/strategy/core -run TestStochastic -count=1
FAIL TestStochasticRejectsRawKBelow20WhenSlowKAbove
FAIL TestStochasticInsufficientSmoothingHistory/14
FAIL TestStochasticInsufficientSmoothingHistory/15
FAIL TestStochasticSlowKStrictBoundary/equal
FAIL TestStochasticSlowKStrictBoundary/above
FAIL TestStochasticLookback
```

The failures were incorrect buys (or the old lookback), not compilation errors. After switching series/alignment, the focused tests passed. The earlier delayed-confirmation fixture initially failed because its three-value average was already below 20; it was corrected to wait through genuinely high slow-K values. Existing stochastic fixtures now provide smoothing history. The same-candle test uses exactly 16 candles, proving entry at the first fully smoothed value. Invalidation and expiry tests have positive controls: removing only the breach permits entry; extending expiry to the confirmation candle permits entry at the inclusive `MaxRetestBars` boundary.

### 3. H4 rejection does not consume an opportunity

The first confirmation now requires both slow K below 20 and a passing H4 gate at that candle's completed M30 decision time. A failed H4 gate leaves the zone waiting. Once both gates have passed, the exact next-open opportunity remains one-shot, including weekend rejection and a missing execution slot.

Previously, MarketData carried completed H4 prices but no timestamps. Evaluating an earlier confirmation against the latest H4 prices would introduce lookahead and falsely consume the opportunity. `MarketData.HTFClosedAt` now supplies aligned completion timestamps. Both engine cursors (`Run` and `Trace`) and the direct assembler populate them using the configured HTF interval. Historical checks use only the prefix completed by the earlier M30 close. Existing cursor alignment/no-lookahead tests also cover the timestamps; prefixes retain capacity limits.

`TestEngineStochasticWaitsForH4` was added before implementing this change. It failed in both Run trade assertions and Trace position assertions for:

- H4 rejection followed by a pass on the next candle;
- a missing M30 slot before H4 passes;
- a weekend before H4 passes.

It also checks that a missing slot or weekend **after H4 has passed** does not carry an execution opportunity. All five cases pass after the fix, alongside the original no-carry and H4-decision-close tests. The strategy remains stateless.

### 4. Cover all three grids

`TestFVGImbalanceGrid` now covers GAZP's normal grid, GAZP's ordered grid, and DOMRF's grid. It checks complete ordered phases and values, keepTop settings, expansion counts, validation of every expanded Params value, and both stochastic alternatives while preserving other defaults.

Expected expansion counts are respectively `[108,2,2,6,4,2]`, `[4,3,3,2,2,6,5,3,2]`, and `[4,2,2,6,4,3,2]`. All pass. This is additional coverage of already-correct artifacts, not a new grid behavior, so no production-grid change or artificial red phase was needed. All three JSON files, including prior ordered-grid edits/comments, remain unchanged.

## Files in the fix

- `internal/service/trading_strategy/fvg_imbalance/strategy/core/core.go`: slow K, warm-up, historical joint H4 confirmation.
- `internal/service/trading_strategy/fvg_imbalance/strategy/core/core_test.go`: raw/slow distinction, smoothing availability, strict boundary, corrected waiting fixtures and positive invalidation/expiry controls.
- `internal/service/trading_strategy/fvg_imbalance/strategy/core/engine_test.go`: remove obsolete MaxHoldBars consumer; add combined Run/Trace regression cases.
- `internal/service/trading_strategy/fvg_imbalance/strategy/gazp/gazp.go` and `domrf/domrf.go`: preserve intended current literals, remove obsolete fields, explicitly disable stochastic by default.
- `internal/service/trading_strategy/fvg_imbalance/strategy/gazp/gazp_test.go`: current literal expectation.
- `internal/service/backtest/fvg_imbalance_registry_test.go`: current Params shape, defaults, and rejection of removed parameters.
- `internal/service/backtest/fvg_imbalance_grid_test.go`: all three grid variants.
- `internal/service/trading_strategy/scalping/strategy/strategy.go`: aligned HTF completion metadata.
- `internal/domain/backtest/engine.go` and `engine_test.go`: metadata assembly in all paths and alignment checks.
- `docs/fvg_imbalance/strategy.md`: slow-K API semantics, 16-candle minimum, joint H4 confirmation, no-carry rule, and parameter table matching preserved literals.
- This report is force-added at the requested normally ignored `.superpowers/sdd/...` path. The existing progress ledger was appended locally; existing report/review artifacts are retained.

## Verification

```text
$ go test ./internal/service/trading_strategy/fvg_imbalance/strategy/core -count=1
ok tinvest/internal/service/trading_strategy/fvg_imbalance/strategy/core

$ go test ./internal/service/trading_strategy/fvg_imbalance/... ./internal/service/backtest ./cmd/backtest ./internal/domain/backtest -count=1
ok tinvest/internal/service/trading_strategy/fvg_imbalance/strategy/core
?  tinvest/internal/service/trading_strategy/fvg_imbalance/strategy/domrf [no test files]
ok tinvest/internal/service/trading_strategy/fvg_imbalance/strategy/gazp
ok tinvest/internal/service/backtest
ok tinvest/cmd/backtest
ok tinvest/internal/domain/backtest

$ ./bin/mage ci
0 issues.
mock regeneration/drift check: passed
go test -race ./...: passed (267 packages with tests)
exit 0

$ go build ./internal/... ./pkg/... ./cmd/...
exit 0; no output

$ git diff --check
$ git diff --cached --check
exit 0; no output
```

The full CI run preceded the final strengthening of existing test controls; the final core run above covers those controls. A pristine archive of the final staged code/tests is also independently checked with the uncached full race suite and build, as recorded below. This prevents uncommitted user changes from masking another broken commit.

Evidence files alongside this report: `final-head-baseline.log`, `final-baseline.log`, `final-stochastic-red.log`, `final-stochastic-green.log`, `final-h4-red.log`, `final-h4-green.log`, `final-core-controls.log`, `final-affected.log`, `final-ci.log`, `final-build.log`, `final-staged-test.log`, `final-staged-build.log`, and `final-staged-tree.txt`.

## Preservation and concerns

- A comparison with the captured initial diff confirms **17 unrelated tracked-file diffs are byte-for-byte unchanged**, including the CLI CPU work and PDL-RSI-M30 removals. Existing untracked plans/specs/skills/docs remain in place. None are staged into the fix.
- Self-review only, as explicitly requested; no subagent or independent reviewer was dispatched. The four requested findings have no known remaining blocker.
- Historical H4 reconstruction requires aligned `HTFClosedAt`. All supported backtest assembly paths now provide it. A custom hand-built snapshot without aligned timing cannot prove an earlier H4 pass and therefore cannot consume a historical opportunity; its current H4 gate still works. This conservative fallback is documented in code.
- Existing ticker calibration-history comments are preserved as historical records; this wave performs no calibration or parameter selection. The actual current numeric literals are authoritative.
- Mockery emits existing v2 deprecation warnings; the drift check and CI gate pass. No mocks changed.

### Final staged-tree verification

The final code/tests tree `1a72ab465e831b7dae5d588d14e65bc2c6550ea3` was exported to `/tmp/tinvest-fvg-final-m03x8gbf`, including the strengthened controls. From that pristine archive:

```text
$ go test -race ./... -count=1
269 packages with tests passed; zero FAIL lines; exit 0
$ go build ./internal/... ./pkg/... ./cmd/...
no output; exit 0
```

The two extra tested packages versus the working-tree CI run are the pre-existing PDL-RSI-M30 packages deleted only in the user's worktree. Both pass in the proposed commit. Only this report is added after that source-tree verification; no code or tests changed afterward.
