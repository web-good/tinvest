# FVG Bullish Body-Cross Correction Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Recognize only bullish M30 FVGs whose middle green candle creates the gap by crossing it from below to above.

**Architecture:** Correct the formation predicate in the pure FVG core. Pin the change with two real GAZP M30 patterns from 2026-10-15: reject the false 07:00 formation and accept the bullish 15:00 impulse.

**Tech Stack:** Go 1.25, existing core strategy tests.

**Spec:** `docs/superpowers/specs/2026-11-04-fvg-imbalance-design.md`

## Global Constraints

- A bullish formation requires `close_b > open_b`, `high_a < low_c`, `open_b <= high_a`, and `close_b >= low_c`.
- Preserve all retest, ATR, next-open entry, stop, and target behaviour.
- Keep the GAZP default literal at `ATRPeriod=14`, `MinGapATR=0.25`, `MaxRetestBars=8`, `RequireCloseAboveZone=1`, `TakeProfitR=1.5`.

## Review Focus

- The 07:00 MSK 2026-10-15 contained-body pattern must not form a bullish FVG.
- The 15:00 MSK 2026-10-15 impulse must form a bullish FVG at the third candle's close.
- Equal boundary values (`open_b == high_a`, `close_b == low_c`) remain valid crossings.
- A red middle candle remains invalid even when its wick geometry matches.
- Existing retest selection sees the corrected zone boundaries unchanged.

### Task 1: Correct bullish formation qualification

**Files:**
- Modify: `internal/service/trading_strategy/fvg_imbalance/strategy/core/core_test.go`
- Modify: `internal/service/trading_strategy/fvg_imbalance/strategy/core/core.go`
- Modify: `docs/fvg_imbalance/strategy.md`

**Interfaces:**
- Consumes: `Strategy.discover(strategy.MarketData) []zone`.
- Produces: bullish zones only when the middle green body crosses the outer-wick gap.

- [ ] **Step 1: Write failing regression tests**

Add a table test using the exact 2026-10-15 GAZP OHLC triples. Assert that 06:30/07:00/07:30 produces no zone and 14:30/15:00/15:30 produces one zone with `[104.61, 105.35]`.

- [ ] **Step 2: Run the regression test to verify it fails**

Run: `go test ./internal/service/trading_strategy/fvg_imbalance/strategy/core -run TestFVGFormationRealGAZP20261015 -count=1`

Expected: FAIL because the current predicate admits the contained-body triple and rejects the impulse triple.

- [ ] **Step 3: Correct `(*Strategy).discover`**

Replace the contained-body comparisons with the bullish body-cross comparisons: `open_b <= high_a` and `close_b >= low_c`. Keep the other formation, ATR, and zone calculations unchanged.

- [ ] **Step 4: Run focused tests**

Run: `go test ./internal/service/trading_strategy/fvg_imbalance/strategy/core -count=1`

Expected: PASS.

- [ ] **Step 5: Update mechanics documentation**

Describe that a bullish FVG requires the middle green body to cross the gap from below to above, not be contained in it.

- [ ] **Step 6: Run relevant verification**

Run: `go test ./internal/service/trading_strategy/fvg_imbalance/... ./internal/service/backtest ./cmd/backtest -count=1 && go build ./internal/... ./pkg/... ./cmd/... && git diff --check`

Expected: exit 0.

- [ ] **Step 7: Commit the correction**

```bash
git add docs/superpowers/specs/2026-11-04-fvg-imbalance-design.md docs/superpowers/plans/2026-11-04-fvg-imbalance-bullish-body-cross.md docs/fvg_imbalance/strategy.md internal/service/trading_strategy/fvg_imbalance/strategy/core/core.go internal/service/trading_strategy/fvg_imbalance/strategy/core/core_test.go
git commit -m "fix: require bullish FVG impulse body cross"
```
