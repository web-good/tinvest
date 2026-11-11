# FVG stochastic implementation plan

> Execute locally, task by task, using superpowers:executing-plans and test-driven development.

**Goal:** Replace FVG retest entries with stochastic crossings and ATR-based exits.
**Architecture:** Preserve FVG discovery and daily ATR, use completed M30 stochastic series, and resolve entry-relative stop/target at actual backtest fill. Update configuration and documentation together.
**Tech Stack:** Go, existing stochastic indicator and backtest engine.
**Spec:** `docs/superpowers/specs/2026-11-09-fvg-imbalance-stochastic-design.md`.

## Constraints and review focus

M30 only; daily ATR; no lookahead; preserve unrelated workspace changes. Verify insufficient history, overlapping zones, expiry boundary, stop/target collision, and next-open gaps.

### Task 1: Entry and exits

**Files:** `internal/service/trading_strategy/fvg_imbalance/strategy/core/`, ticker strategy packages.

- [ ] Add failing tests for crossings, zone expiry and no retest, insufficient history, stop/target precedence, stochastic exit and parameter validation.
- [ ] Implement the specified parameters, entry, and exits; replace ticker literals with uncalibrated defaults.
- [ ] Run core and ticker tests; verify all pass.

### Task 2: Fill-relative levels

**Files:** `internal/domain/backtest/engine.go`, `internal/domain/backtest/entry_next_open_test.go`, shared signal type if needed.

- [ ] Add failing integration test for a next-open gap with levels based on actual fill and frozen ATR.
- [ ] Implement fill-relative levels without changing other strategies; test gap and stop/target collision.

### Task 3: Calibration and verification

**Files:** `internal/service/backtest/fvg_imbalance_registry.go`, grid tests, `data/params/fvg_imbalance/`, `docs/fvg_imbalance/strategy.md`.

- [ ] Update registry, grids and mechanics documentation; remove obsolete calibrated claims from active settings.
- [ ] Run affected package tests and `go test ./...`; report any unrelated failures.
