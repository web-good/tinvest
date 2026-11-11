# FVG Imbalance Parameter Removal Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Remove eight obsolete `fvg_imbalance` parameters and require the third pattern candle's high to be strictly above the second candle's high.

**Architecture:** Keep the existing stateless M30 strategy and reflection-based parameter parsing, but shrink `core.Params` and delete every behavior branch owned by the removed fields. Formation remains centralized in `Strategy.discover`; ticker literals, calibration grids, registry tests, mechanics documentation, and the local calibration skill are aligned with the smaller API.

**Tech Stack:** Go 1.25, standard `testing`, JSON phased grids, Mage CI.

**Spec:** `docs/superpowers/specs/2026-11-05-fvg-imbalance-parameter-removal-design.md`

## Global Constraints

- Preserve the user's current `GAZP TakeProfitR = 1.5` change.
- Do not modify or restore unrelated dirty-worktree changes, especially the `pdl_rsi_m30` removal and `cmd/backtest/main.go` edits.
- Remove exactly `RequireFirstRetest`, `RequireBullishBOS`, `RequireLiquiditySweep`, `SwingK`, `StructureLookbackBars`, `MaxSweepToFVGBars`, `MaxEntryDistanceATR`, and `MaxHoldBars`.
- Keep historical design documents unchanged; update current mechanics docs, active parameter grids, and `.agents/skills/fvg-imbalance-calibrate/SKILL.md`.
- Do not commit unless the user explicitly requests it; the shared worktree already contains user-owned changes.

## Review Focus

- `high[c] == high[b]` must be rejected, not treated as a valid boundary case; Task 1 tests equality explicitly.
- `high[c] < high[b]` must also be rejected while the existing valid formation remains accepted; Task 1 covers all three cases.
- A second protected retest must remain eligible after removal of `RequireFirstRetest`; Task 2 retains a focused behavior test.
- JSON containing any removed field must fail as an unknown parameter rather than being silently ignored; Task 3 tests every removed name.
- Lookback must still retain formation ATR history through `MaxRetestBars` without structural padding; Task 2 pins the exact calculation and sliding-window behavior.

---

### Task 1: Enforce the third-candle high rule

**Files:**
- Modify: `internal/service/trading_strategy/fvg_imbalance/strategy/core/core_test.go`
- Modify: `internal/service/trading_strategy/fvg_imbalance/strategy/core/core.go`

**Interfaces:**
- Consumes: existing `(*Strategy).discover(strategy.MarketData) []zone`.
- Produces: formation qualification with the additional strict predicate `md.Highs[c] > md.Highs[b]`.

- [ ] **Step 1: Add failing formation cases**

Extend `TestFVGFormation` with `third high equals second high` and `third high below second high`, both expecting zero zones; retain `green body` expecting one zone.

- [ ] **Step 2: Verify RED**

Run: `go test ./internal/service/trading_strategy/fvg_imbalance/strategy/core -run TestFVGFormation -count=1`

Expected: FAIL because at least the equality case still discovers a zone.

- [ ] **Step 3: Add the minimal formation predicate**

In `(*Strategy).discover`, reject a candidate unless `md.Highs[c] > md.Highs[b]`; do not alter gap, body-cross, continuity, or ATR rules.

- [ ] **Step 4: Verify GREEN**

Run: `go test ./internal/service/trading_strategy/fvg_imbalance/strategy/core -run 'TestFVGFormation|TestFVGFormationRealGAZP20261015' -count=1`

Expected: PASS.

### Task 2: Remove the parameters and their core behavior

**Files:**
- Modify: `internal/service/trading_strategy/fvg_imbalance/strategy/core/core_test.go`
- Modify: `internal/service/trading_strategy/fvg_imbalance/strategy/core/engine_test.go`
- Modify: `internal/service/trading_strategy/fvg_imbalance/strategy/core/core.go`

**Interfaces:**
- Consumes: unchanged `Params`, `DefaultParams`, `ValidateParams`, `Strategy.Lookback`, and `Strategy.Decide` API names.
- Produces: `Params` containing only `ATRPeriod`, `MinGapATR`, `MaxRetestBars`, `MinRetestDepthPct`, `TakeProfitR`, `HTFTrendEMA`, and `StopBufferATR`; removed fields are absent from the Go type.

- [ ] **Step 1: Add a failing reflection test for the public API**

Add `TestParamsRemovedFieldsAbsent`, iterate the eight exact field names with `reflect.TypeOf(Params{}).FieldByName`, and fail if any field exists.

- [ ] **Step 2: Verify RED**

Run: `go test ./internal/service/trading_strategy/fvg_imbalance/strategy/core -run TestParamsRemovedFieldsAbsent -count=1`

Expected: FAIL naming `RequireFirstRetest` first.

- [ ] **Step 3: Shrink core parameters and behavior**

Remove the eight fields and their validation/defaults. Simplify `Lookback` to `max(3, s.atrBars()) + s.p.MaxRetestBars`; delete swing helpers, `structurePasses`, time exit, first-retest consumption, maximum-entry-price assignment, and related reason text. A protected earlier touch must no longer invalidate a zone.

- [ ] **Step 4: Align core tests with the reduced strategy**

Delete tests dedicated only to swing/structure gates, first-retest consumption, entry-distance limits, and time exits. Retain one focused test showing a later protected retest can enter, update `TestDefaultParams`/`TestValidateParams`, make `TestLookback` assert `12` for `ATRPeriod=2` and `MaxRetestBars=8`, and remove `TestEngineTimeExitAtExactBarCountUsesClose`.

- [ ] **Step 5: Verify the core package**

Run: `go test ./internal/service/trading_strategy/fvg_imbalance/strategy/core -count=1`

Expected: PASS.

### Task 3: Align literals, registry parsing, and active grids

**Files:**
- Modify: `internal/service/trading_strategy/fvg_imbalance/strategy/gazp/gazp.go`
- Modify: `internal/service/trading_strategy/fvg_imbalance/strategy/gazp/gazp_test.go`
- Modify: `internal/service/trading_strategy/fvg_imbalance/strategy/domrf/domrf.go`
- Modify: `internal/service/backtest/fvg_imbalance_registry_test.go`
- Modify: `internal/service/backtest/fvg_imbalance_grid_test.go`
- Modify: `data/params/fvg_imbalance/gazp/grid.json`
- Modify: `data/params/fvg_imbalance/gazp/ordered_grid.json`
- Modify: `data/params/fvg_imbalance/domrf/grid.json`

**Interfaces:**
- Consumes: reduced `core.Params` from Task 2 and existing strict reflection-based `Binding.ParseParams`.
- Produces: compilable ticker defaults and active grids containing only supported parameter names.

- [ ] **Step 1: Update literals and registry tests to compile against the reduced API**

Remove all eight fields from GAZP/DOMRF literals and expected structs. Preserve GAZP `TakeProfitR: 1.5`. In registry tests, require each removed JSON name (using value `1`) to be rejected as unknown, remove them from valid overlays, and retain valid overlays for `HTFTrendEMA` and `StopBufferATR`.

- [ ] **Step 2: Verify literals and strict parsing**

Run: `go test ./internal/service/trading_strategy/fvg_imbalance/... ./internal/service/backtest -run 'TestParams|TestFVGImbalanceDefaults|TestFVGImbalanceParseParams|TestFVGImbalancePartialQualityOverlays' -count=1`

Expected: PASS, including unknown-field rejection for all eight removed names.

- [ ] **Step 3: Make grid expectations describe the reduced surface**

Change `TestFVGImbalanceGrid` to expect GAZP phases `fvg` (108 combinations), `context` with only `HTFTrendEMA` (6), and `execution` with only `StopBufferATR` (4); remove the `holding` phase expectation.

- [ ] **Step 4: Verify the grid test is RED**

Run: `go test ./internal/service/backtest -run TestFVGImbalanceGrid -count=1`

Expected: FAIL because `gazp/grid.json` still contains the removed fields/phases.

- [ ] **Step 5: Update active grids**

Apply the expected GAZP phase shape; remove `first_retest`, `structure`, and `swing_k_sensitivity` from `ordered_grid.json`; remove `first_retest` and `structure` from DOMRF; rewrite comments so they do not mention `MaxHoldBars`.

- [ ] **Step 6: Verify strategy adapters and grids**

Run: `go test ./internal/service/trading_strategy/fvg_imbalance/... ./internal/service/backtest -run 'FVGImbalance|TestParams|TestFVGFormation|TestRetest|TestEngine' -count=1`

Expected: PASS.

### Task 4: Update current documentation and calibration guidance

**Files:**
- Modify: `docs/fvg_imbalance/strategy.md`
- Modify: `.agents/skills/fvg-imbalance-calibrate/SKILL.md`

**Interfaces:**
- Consumes: final behavior and parameter list from Tasks 1–3.
- Produces: mechanics and calibration instructions that expose no removed functionality.

- [ ] **Step 1: Rewrite mechanics documentation**

Document `high[c] > high[b]` in formation; remove swing/sweep/BOS, first-retest-only, entry-distance, and time-exit sections and table rows; adjust calibration text to the remaining knobs without adding ticker-specific results.

- [ ] **Step 2: Rewrite the local calibration skill**

Remove phases and instructions for all eight deleted fields, renumber the remaining procedure, and retain the existing out-of-sample stability rules.

- [ ] **Step 3: Verify stale active references are gone**

Run: `rg -n 'RequireFirstRetest|RequireBullishBOS|RequireLiquiditySweep|SwingK|StructureLookbackBars|MaxSweepToFVGBars|MaxEntryDistanceATR|MaxHoldBars' internal/service/trading_strategy/fvg_imbalance internal/service/backtest/fvg_imbalance* data/params/fvg_imbalance docs/fvg_imbalance .agents/skills/fvg-imbalance-calibrate`

Expected: no matches.

- [ ] **Step 4: Format and run focused verification**

Run: `gofmt -w internal/service/trading_strategy/fvg_imbalance/strategy/core/core.go internal/service/trading_strategy/fvg_imbalance/strategy/core/core_test.go internal/service/trading_strategy/fvg_imbalance/strategy/core/engine_test.go internal/service/trading_strategy/fvg_imbalance/strategy/gazp/gazp.go internal/service/trading_strategy/fvg_imbalance/strategy/gazp/gazp_test.go internal/service/trading_strategy/fvg_imbalance/strategy/domrf/domrf.go internal/service/backtest/fvg_imbalance_registry_test.go internal/service/backtest/fvg_imbalance_grid_test.go`

Run: `go test ./internal/service/trading_strategy/fvg_imbalance/... ./internal/service/backtest ./cmd/backtest -count=1`

Expected: PASS.

- [ ] **Step 5: Run the repository gate and report unrelated failures explicitly**

Run: `./bin/mage ci`

Expected: PASS. If the pre-existing dirty-tree deletions make the repository gate fail, report every failing command/test and separately show that all focused FVG checks passed.
