# FVG Optional Formation and Retest Mode Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add independent `0/1` parameters for the third-candle higher-high filter and body-versus-wick retest-depth calculation while preserving current defaults.

**Architecture:** Extend the existing reflection-friendly `core.Params` value with two integer switches and keep all behavioral decisions in `core`. Ticker literals, registry overlays, calibration grids, and mechanics documentation expose the same fields without adding a new configuration layer.

**Tech Stack:** Go 1.25, table-driven Go tests, JSON parameter grids, Markdown documentation.

**Spec:** `docs/superpowers/specs/2026-11-05-fvg-optional-formation-and-retest-mode-design.md`

## Global Constraints

- `RequireThirdCandleHigherHigh` and `RetestDepthByBody` accept only `0` or `1`.
- Both default to `1` in core, GAZP, and DOMRF to preserve current behavior.
- `RetestDepthByBody=0` changes only the depth input to `low`; touch and invalidation rules remain unchanged.
- No unrelated strategy mechanics or historical reports are changed.

## Review Focus

- Invalid switch values (`-1`, `2`) must fail validation for both fields; covered by core validation tests.
- Default and ticker-specific params must retain both switches at `1`; covered by default, ticker, and registry tests.
- With the formation switch off, an otherwise valid FVG whose third high is not above the middle high must form; covered by a core decision test.
- Wick mode must accept a depth reached only by the wick, while body mode rejects it; covered by retest-depth tests.
- Wick mode must still reject `low < lower` and a candle that never touches the zone; covered by retest-depth boundary tests.

---

### Task 1: Add and expose both strategy switches

**Files:**
- Modify: `internal/service/trading_strategy/fvg_imbalance/strategy/core/core.go`
- Test: `internal/service/trading_strategy/fvg_imbalance/strategy/core/core_test.go`
- Modify: `internal/service/trading_strategy/fvg_imbalance/strategy/gazp/gazp.go`
- Test: `internal/service/trading_strategy/fvg_imbalance/strategy/gazp/gazp_test.go`
- Modify: `internal/service/trading_strategy/fvg_imbalance/strategy/domrf/domrf.go`
- Test: `internal/service/backtest/fvg_imbalance_registry_test.go`
- Modify: `data/params/fvg_imbalance/gazp/grid.json`
- Modify: `data/params/fvg_imbalance/gazp/ordered_grid.json`
- Modify: `data/params/fvg_imbalance/domrf/grid.json`
- Modify: `docs/fvg_imbalance/strategy.md`

**Interfaces:**
- Consumes: existing `core.Params`, `ValidateParams`, `discover`, and `Decide` behavior.
- Produces: `core.Params.RequireThirdCandleHigherHigh int` and `core.Params.RetestDepthByBody int`.

- [ ] **Step 1: Write failing parameter and behavior tests**

Extend `TestDefaultParams`, `TestValidateParams`, ticker/registry expectations, the formation tests, and `TestRetestDepthUsesBody` so they assert both `0/1` modes, preservation of touch/invalidation, and rejection of invalid switch values.

- [ ] **Step 2: Run focused tests and verify RED**

Run: `go test ./internal/service/trading_strategy/fvg_imbalance/strategy/core ./internal/service/trading_strategy/fvg_imbalance/strategy/gazp ./internal/service/backtest -run 'FVG|DefaultParams|ValidateParams|Retest|Formation|GAZPParams' -count=1`

Expected: FAIL because the two fields and their configurable behaviors do not exist.

- [ ] **Step 3: Implement the minimal core behavior**

Add the fields, default them to `1`, validate them as binary switches, guard `high[c] > high[b]` with `RequireThirdCandleHigherHigh`, and select `min(open, close)` versus `low` with `RetestDepthByBody` while retaining the existing touch and invalidation checks.

- [ ] **Step 4: Update literals, grids, and mechanics documentation**

Set both switches to `1` in GAZP and DOMRF literals; add independent `[0, 1]` phases/fields to all FVG grids; document both modes and defaults in the formation, retest, and parameter sections.

- [ ] **Step 5: Run focused tests and verify GREEN**

Run: `go test ./internal/service/trading_strategy/fvg_imbalance/... ./internal/service/backtest -count=1`

Expected: PASS.

- [ ] **Step 6: Run repository verification**

Run: `git diff --check && go test ./...`

Expected: PASS with no formatting errors or test failures.

- [ ] **Step 7: Commit if requested**

No commit is created unless the user explicitly asks for one.
