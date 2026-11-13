# AFKS Live Monitoring Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Include AFKS in FVG Imbalance live monitoring while leaving order execution disabled.

**Architecture:** Extend the live adapter's existing ticker-to-parameter switch to resolve AFKS through `afks.Params()`, and append AFKS to the configured FVG ticker list. The existing shared `TradeEnabled` setting remains false, so this does not enable order placement.

**Tech Stack:** Go; environment-backed config in `env/prod.env`.

**Spec:** `docs/superpowers/specs/2026-11-12-afks-live-monitoring-design.md`

## Global Constraints

- Preserve `FVG_IMBALANCE_TRADE_ENABLED=false`.
- Use the current AFKS literal, which returns `core.DefaultParams()`; do not install the failed calibration candidate.
- Do not change strategy mechanics, sizing, or other ticker settings.

## Review Focus

- AFKS ticker lookup returns a decider and uses the AFKS parameter package — pin this in `TestStrategyRegistryAndProtection`.
- Unknown ticker behavior remains false/nil — existing assertion remains.
- The shared trade flag remains false in `env/prod.env` — inspect the final diff.
- Existing ticker list entries remain intact — inspect the final diff.
- AFKS stop calculation uses the same registered params as its decider — assert its positive stop in the existing registry test loop.

---

### Task 1: Register AFKS for FVG live monitoring

**Files:**
- Modify: `internal/service/trading_strategy/fvg_imbalance/live/strategy.go`
- Modify: `internal/service/trading_strategy/fvg_imbalance/live/strategy_test.go`
- Modify: `env/prod.env`

**Interfaces:**
- Consumes: `afks.Params() core.Params` from `internal/service/trading_strategy/fvg_imbalance/strategy/afks`.
- Produces: `params("AFKS")` resolves AFKS params; `Decider("AFKS")` succeeds; AFKS appears in `FVG_IMBALANCE_TICKERS`.

- [x] Add `AFKS` to the ticker table in `TestStrategyRegistryAndProtection`, asserting its decider and positive stop through the existing loop.
- [x] Import `strategy/afks` and add a `case "AFKS": return afks.Params(), true` branch in `params`.
- [x] Append `AFKS` to `FVG_IMBALANCE_TICKERS` while preserving `FVG_IMBALANCE_TRADE_ENABLED=false` and all existing values.
- [x] Run `gofmt` on the modified Go files and inspect the final diff for only the specified live registration/config changes.
