# FVG Imbalance Optional Stochastic Exit Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make the existing FVG stochastic position exit configurable while retaining stop-loss and take-profit exits.

**Architecture:** Add an integer `UseStochExit` parameter to the stateless core and guard only the stochastic sell branch. Set the enabled default for all ticker presets, expose both values in every FVG calibration grid, and update mechanics documentation.

**Tech Stack:** Go, JSON phased calibration grids, Markdown.

**Spec:** `docs/superpowers/specs/2026-11-09-fvg-imbalance-optional-stochastic-exit-design.md`

## Global Constraints

- `UseStochExit` accepts only `0` or `1` and defaults to `1`.
- `0` disables only the %K upward crossing of 80 as an exit.
- Stop-loss and take-profit ordering and behavior remain unchanged.
- Stochastic entry behavior remains unchanged.

## Review Focus

- `UseStochExit=0` must still allow stop and take-profit exits.
- `UseStochExit=1` must preserve the existing stochastic exit.
- Invalid values must fail parameter validation.
- Partial JSON parameter overlays must inherit the enabled default.
- Every current FVG calibration grid must offer both switch values.

---

### Task 1: Add the core exit switch

**Files:**
- Modify: `internal/service/trading_strategy/fvg_imbalance/strategy/core/core.go`
- Modify: `internal/service/trading_strategy/fvg_imbalance/strategy/core/core_test.go` (extend existing default/validation assertions if present)
- Modify: `internal/service/trading_strategy/fvg_imbalance/strategy/core/stochastic_new_test.go` (extend existing position exit cases if present)

**Interfaces:**
- Produces: `Params.UseStochExit int`, default `1`, valid values `0` and `1`.

- [ ] Add `UseStochExit int` to `Params`, initialize it to `1` in `DefaultParams`, and reject any value other than `0` or `1` in `ValidateParams`.
- [ ] Guard the current stochastic sell branch with `s.p.UseStochExit == 1`; leave the preceding stop-loss and take-profit branches unchanged.
- [ ] Extend existing tests for default and validation behavior, and confirm stochastic exit is suppressed when disabled while stop and take-profit behavior remains available.

### Task 2: Propagate defaults and calibration choices

**Files:**
- Modify: `internal/service/trading_strategy/fvg_imbalance/strategy/bspb/bspb.go`
- Modify: `internal/service/trading_strategy/fvg_imbalance/strategy/domrf/domrf.go`
- Modify: `internal/service/trading_strategy/fvg_imbalance/strategy/gazp/gazp.go`
- Modify: `internal/service/trading_strategy/fvg_imbalance/strategy/lkoh/lkoh.go`
- Modify: `internal/service/trading_strategy/fvg_imbalance/strategy/sber/sber.go`
- Modify: `internal/service/trading_strategy/fvg_imbalance/strategy/sngs/sngs.go`
- Modify: `internal/service/trading_strategy/fvg_imbalance/strategy/t/t.go`
- Modify: `data/params/fvg_imbalance/gazp/grid.json`
- Modify: `data/params/fvg_imbalance/gazp/ordered_grid.json`
- Modify: `data/params/fvg_imbalance/domrf/grid.json`
- Modify: `internal/service/backtest/fvg_imbalance_grid_test.go` (extend existing expected grid axes)

- [ ] Set `UseStochExit: 1` explicitly in each ticker preset so current live/backtest defaults remain the same.
- [ ] Add a `stochastic_exit` phase with `UseStochExit: [0, 1]` to each grid, preserving all existing phases and axes.
- [ ] Add `UseStochExit` to the existing expected axis list in the grid test.

### Task 3: Document parameter behavior

**Files:**
- Modify: `docs/fvg_imbalance/strategy.md`

- [ ] Document the `UseStochExit` values and default, and clarify that disabling it leaves stop-loss and take-profit exits active.
