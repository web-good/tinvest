# MTLR FVG Imbalance Live Support Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make MTLR available to the FVG imbalance live adapter while production order submission remains disabled.

**Architecture:** Add MTLR's existing `Params()` provider to the live adapter's ticker dispatch, and list MTLR in the production FVG ticker configuration. Keep all order-enable and sizing settings unchanged.

**Tech Stack:** Go, existing FVG live adapter/configuration, Go tests.

**Spec:** `docs/superpowers/specs/2026-11-12-mtlr-fvg-live-design.md`

## Global Constraints

- Preserve `FVG_IMBALANCE_TRADE_ENABLED=false`.
- Use `mtlr.Params()`; do not duplicate parameter values in the adapter.
- Keep the present MTLR literal unchanged by this live-registration change.

## Review Focus

- MTLR returns a live decider: test `Decider("MTLR")` succeeds.
- Stop protection follows the MTLR literal: test `DesiredStop` equals `EntryPrice - StopATR*EntryATR`.
- Unsupported ticker remains rejected: retain the existing `UNKNOWN` assertion.
- Production config includes MTLR without enabling orders: inspect the edited env lines and assert `FVG_IMBALANCE_TRADE_ENABLED=false` remains.
- Existing ticker dispatch stays intact: run the live package tests covering the whole ticker list.

---

### Task 1: Register MTLR in the live adapter and production ticker list

**Files:**
- Modify: `internal/service/trading_strategy/fvg_imbalance/live/strategy.go`
- Modify: `internal/service/trading_strategy/fvg_imbalance/live/strategy_test.go`
- Modify: `env/prod.env`

**Interfaces:**
- Consumes: `mtlr.Params() core.Params` and existing `Strategy.Decider` / `Strategy.DesiredStop` methods.
- Produces: `params("MTLR")` returns the MTLR params and `Strategy.Decider("MTLR")` returns a decider; production ticker config lists MTLR while trade enable stays false.

- [x] Add MTLR to the live adapter imports and `params` switch using `mtlr.Params()`.
- [x] Extend `TestStrategyRegistryAndProtection` to include MTLR and assert its provider and derived stop use `mtlr.Params()`.
- [x] Add `MTLR` to `FVG_IMBALANCE_TICKERS` in `env/prod.env`; leave `FVG_IMBALANCE_TRADE_ENABLED=false` unchanged.
- [x] Run `gofmt` on the changed Go files.
- [x] Run `go test ./internal/service/trading_strategy/fvg_imbalance/live ./internal/config` and confirm it passes.
