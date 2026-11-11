# FVG Daily Trend Filter Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make `fvg_imbalance` evaluate its optional EMA trend filter on completed daily closes, keeping the configured period count unchanged.

**Architecture:** Reuse `MarketData.DailyCloses`, which the backtest engine supplies as completed daily candles. Preserve `HTFTrendEMA` and the existing close-above-EMA condition; only its input timeframe changes from H4 to daily.

**Tech Stack:** Go 1.25, existing `ema.Compute`, Go unit tests.

**Spec:** `docs/superpowers/specs/2026-11-09-fvg-daily-trend-design.md`

## Global Constraints

- Periods are counted in daily candles without converting existing values.
- EMA period 0 disables the filter.
- Incomplete or invalid daily history fails closed when the filter is enabled.
- Do not change entry, exit, or grid parameter values.

## Review Focus

- Insufficient daily history must block an entry; test the shorter-than-period input.
- H4 data must not affect the decision; test contradictory daily and H4 trends.
- The latest completed daily close must be above its daily EMA; test both above and below cases.

---

### Task 1: Move the FVG trend gate to daily data

**Files:**
- Modify: `internal/service/trading_strategy/fvg_imbalance/strategy/core/core.go`
- Test: `internal/service/trading_strategy/fvg_imbalance/strategy/core/stochastic_new_test.go`
- Modify/test: `cmd/backtest/main.go`, `cmd/backtest/main_test.go`
- Test: `internal/service/backtest/fvg_imbalance_registry_test.go`, `internal/service/trading_strategy/fvg_imbalance/strategy/gazp/gazp_test.go`
- Modify: `internal/service/trading_strategy/fvg_imbalance/strategy/{bspb,domrf,gazp,lkoh,sber,sngs,t}/*.go`
- Modify: `docs/fvg_imbalance/strategy.md`
- Modify: `.agents/skills/fvg-imbalance-calibrate/SKILL.md`

**Interfaces:**
- Consumes `strategy.MarketData.DailyCloses` and existing `ema.Compute`.
- Keeps the `HTFTrendEMA` parameter and `htfTrendOK` helper interface; the helper reads daily closes.

- [ ] **Step 1: Write the failing tests**

Update `TestStochasticHTFFilter` to supply daily closes and cover: insufficient daily history, daily close below EMA, daily close above EMA, and contradictory H4/daily data where only the daily result controls the signal.

- [ ] **Step 2: Run the focused test to verify it fails**

Run: `go test ./internal/service/trading_strategy/fvg_imbalance/strategy/core -run '^TestStochasticHTFFilter$' -count=1`

Expected: cases using daily closes fail because the filter currently reads H4 closes.

- [ ] **Step 3: Implement the daily EMA gate**

Change `htfTrendOK` to validate and compute EMA from `md.DailyCloses`, keeping its period and close-above-EMA semantics. Update ticker parameter comments and the mechanics documentation to say the EMA period is measured in daily candles.

- [ ] **Step 4: Run focused and package tests**

Run: `go test ./internal/service/trading_strategy/fvg_imbalance/strategy/... ./internal/service/backtest ./cmd/backtest`

Expected: all packages pass.
