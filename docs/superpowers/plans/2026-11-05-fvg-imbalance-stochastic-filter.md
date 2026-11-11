# FVG Imbalance Stochastic Filter Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Add an optional `%K < 20` stochastic confirmation that waits after FVG retest depth is reached and remains active until zone invalidation or expiry.

**Architecture:** Keep the strategy stateless. Reconstruct each eligible zone, find whether its depth was reached on any valid candle in its lifetime, and evaluate the latest completed M30 stochastic value before emitting the next-open entry. Add one reflection-friendly integer switch and preserve existing behavior when it is disabled.

**Tech Stack:** Go 1.25, existing `pkg/indicators.StochasticSeries`, table-driven tests, JSON phased grids, Markdown mechanics documentation.

**Spec:** `docs/superpowers/specs/2026-11-05-fvg-imbalance-stochastic-filter-design.md`

## Global Constraints

- `UseStochFilter` accepts only `0` or `1`.
- The stochastic uses slow `%K` with `K=14`, `D=3`; entry requires `%K < 20`, while `%D` is informational only and is not an entry condition.
- Only completed M30 candles are inputs; the signal fills on the next M30 open.
- A zone remains subject to existing `MaxRetestBars` expiry and `low < lower` invalidation.
- Existing behavior must remain unchanged when `UseStochFilter = 0`.
- Preserve unrelated user changes already present in the dirty worktree.

## Review Focus

- Depth reached before stochastic: test that a later `%K < 20` can trigger without requiring the latest candle to touch the FVG.
- Stochastic already below 20 on the depth candle: test same-candle confirmation.
- Zone invalidation after depth: test that later overshoot never produces an entry.
- Insufficient stochastic history: test fail-closed behavior when the filter is enabled.
- Grid/reflection compatibility: test validation, default preservation, and both grid values.

### Task 1: Core parameter and stochastic entry behavior

**Files:**
- Modify: `internal/service/trading_strategy/fvg_imbalance/strategy/core/core.go`
- Test: `internal/service/trading_strategy/fvg_imbalance/strategy/core/core_test.go`

**Interfaces:**
- Consumes: existing `core.Params`, `strategy.MarketData`, and `indicators.StochasticSeries`.
- Produces: `Params.UseStochFilter`, default `0`, validation for `0/1`, and unchanged `Strategy.Decide` signal interface.

- [ ] **Step 1: Write failing tests** for default/validation and the five review-focus behaviors. Use deterministic M30 bars with a 14-bar stochastic warm-up and assert `SignalNone` while waiting, `SignalBuy` once `%K < 20`, no entry after `low < lower`, and no entry with insufficient history.
- [ ] **Step 2: Run the focused tests to verify they fail** because `UseStochFilter` does not exist and delayed confirmation is not implemented.

Run: `go test ./internal/service/trading_strategy/fvg_imbalance/strategy/core -run 'Test(Stochastic|DefaultParams|ValidateParams)' -count=1`

Expected: FAIL with compile/behavior failures caused by the missing parameter and filter.

- [ ] **Step 3: Add `UseStochFilter int` to `Params`, default it to `0`, validate it as `0/1`, and include `14` stochastic warm-up in `Lookback`.
- [ ] **Step 4: Implement the minimal state-free decision logic**: scan candles from `z.formed+1` through `k` for the first valid depth-reaching candle; retain existing invalidation checks; when enabled, calculate the latest `%K` with `indicators.StochasticSeries(md.Highs, md.Lows, md.Closes, 14, 3)` and continue unless `%K < 20`; when disabled, preserve immediate retest entry.
- [ ] **Step 5: Run the focused tests to verify they pass.**

Run: `go test ./internal/service/trading_strategy/fvg_imbalance/strategy/core -run 'Test(Stochastic|DefaultParams|ValidateParams)' -count=1`

Expected: PASS.

- [ ] **Step 6: Run all core tests and format changed Go files.**

Run: `gofmt -w internal/service/trading_strategy/fvg_imbalance/strategy/core/core.go internal/service/trading_strategy/fvg_imbalance/strategy/core/core_test.go && go test ./internal/service/trading_strategy/fvg_imbalance/strategy/core -count=1`

Expected: PASS with no regressions.

### Task 2: Expose the switch in ticker grids and grid tests

**Files:**
- Modify: `data/params/fvg_imbalance/gazp/grid.json`
- Modify: `data/params/fvg_imbalance/gazp/ordered_grid.json`
- Modify: `data/params/fvg_imbalance/domrf/grid.json`
- Modify: `internal/service/backtest/fvg_imbalance_grid_test.go`

**Interfaces:**
- Consumes: `core.Params.UseStochFilter` from Task 1.
- Produces: a `stochastic` phase with `UseStochFilter: [0, 1]` in each calibration grid and matching expansion assertions.

- [ ] **Step 1: Update the grid test first** to expect the new phase and its two combinations; run it and verify the test fails against the current JSON files.
- [ ] **Step 2: Add the `stochastic` phase** to all three grids, keeping existing phase order and values intact.
- [ ] **Step 3: Run the grid and registry tests.**

Run: `go test ./internal/service/backtest -run 'TestFVGImbalance(Grid|Registry)' -count=1`

Expected: PASS.

### Task 3: Document mechanics and verify the full affected surface

**Files:**
- Modify: `docs/fvg_imbalance/strategy.md`

**Interfaces:**
- Consumes: final behavior and parameter names from Tasks 1–2.
- Produces: mechanics-only documentation covering `%K < 20`, same/later-candle waiting, invalidation/expiry, and `UseStochFilter`.

- [ ] **Step 1: Update the strategy overview, retest rules, parameter table, and calibration description** without adding ticker-specific calibration results.
- [ ] **Step 2: Run the affected strategy, backtest, and build checks.**

Run: `go test ./internal/service/trading_strategy/fvg_imbalance/... ./internal/service/backtest -count=1`

Run: `go build ./internal/... ./pkg/... ./cmd/...`

Run: `git diff --check`

Expected: all commands exit 0 and the diff has no whitespace errors.

