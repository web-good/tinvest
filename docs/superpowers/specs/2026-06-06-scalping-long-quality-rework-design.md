# Long-Only Scalping Quality Rework — Design

**Date:** 2026-06-06
**Branch:** `feat/scalping-long-quality-rework`
**Status:** Approved (design), pending implementation plan

## Problem

The adaptive scalping strategy (`internal/service/trading_strategy/scalping/strategy/adaptive`)
is net-negative on both RUAL (−1.9%) and AFKS (−14.8%) over 24 months (Hour1).
A trade-level read of the backtest journals shows the loss is **not** from the
dual-regime architecture but from a handful of concrete defects that turn most
trades into small losers.

### Evidence (from `reports/cmp/`)

- **AFKS journal (50 trades):** asset fell −55% buy-and-hold; the strategy's low
  exposure actually *beat* buy-and-hold (−14.8%). It is a "lose less" engine, not
  yet a "make money" engine.
- **`TP`-labeled exits that are losses:** AFKS trades 6/15/20/21/29/31/40/44 are
  marked `TP` but closed in the red (−303, −31, −115, −187, −128, −278, −11, −168).
  Cause: range exit uses `TakeProfit = (donUpper+donLower)/2` (Donchian mid) and
  sells on `price >= mid`. As the channel slides down, `mid` drops below the entry
  price, so the position dumps at a loss the moment price ≥ a mid that is already
  under water. (`adaptive.go:162-170`, `adaptive.go:190`)
- **Bar-1 TRAIL stop-outs:** AFKS trades 2/4/8/9/36/47/48 exit via `TRAIL` after a
  single bar (−589, −633, −171, −887, …). Cause: `chandelier = recentHigh(window)
  − TrailMult·ATR`; entering near a local high places the trail just under entry,
  so the first down-tick triggers it. (`adaptive.go:149`, `adaptive.go:157`)
- **Commission churn:** ~50 round-trips × all-in × 0.0005×2 ≈ **−5% of capital**
  on AFKS — roughly a third of the total loss — much of it on sub-0.2% "trades."
- **Perverse calibration:** ranking by `profit_factor` (`calibrate.go:108`) is blind
  to trade count and exposure; the top-20 combos are all losing and the "best" combo
  sets `TrendFilterPeriod=0`, disabling the very HTF filter that would keep longs out
  of a downtrend.

## Constraints (from the user)

- **Long-only. No shorts, no margin.** Phase-2 short symmetry is explicitly out of scope.
- **Few trades are fine.** Low exposure is acceptable; quality per trade is the goal.
  Do **not** add an exposure penalty to the calibration objective.

## Goal

Raise the **quality** of the few trades the strategy takes — eliminate the
parasite losers (TP bug, bar-1 trail, commission churn), keep only high-confluence
A+ entries, and let winners run. Success = expectancy / profit factor / win-rate up,
with the same or fewer trades. The dual-regime (trend/range) architecture is
**kept as-is** — it is not the source of the loss.

## Design — four blocks

### Block 1 — Fix the TP-below-entry leak

In the range/dead-zone management branch (`adaptive.go:162-170`), the Donchian mid
must act as a take-profit **only when it is above the entry price**. When `mid <=
entryPrice`, the position is not exited on the mid; it is managed by the stop only.

- `entryPrice` is available as `in.pos.PurchasePrice`.
- Behavior change: `case in.price >= mid && mid > 0:` becomes
  `case in.price >= mid && mid > in.pos.PurchasePrice:` (mid still reported as
  `sig.TakeProfit` only when it is a valid, profitable target; otherwise leave
  `sig.TakeProfit` at 0 so reports don't show a phantom target below entry).

### Block 2 — Arm the trailing stop only after the trade is in profit

The chandelier trail must not be active until the trade has moved at least
`TrailArmATR · ATR` in favor (breakeven-arming). Until armed, only the initial
hard stop (`entry − SLMult·ATR`) protects the position.

- New `Params` field: `TrailArmATR float64` (e.g. 1.0). `0` = arm immediately
  (preserves current behavior, used to prove the refactor is behavior-preserving
  before tuning).
- Armed test in the trend branch (`adaptive.go:148-159`):
  `armed := in.price >= in.pos.PurchasePrice + s.p.TrailArmATR*in.atr`. The
  `in.price <= chandelier` TRAIL exit only fires when `armed`. The hard-SL exit is
  unchanged and always active.

### Block 3 — Entry-quality gate ("fewer but better")

Tighten entries (do **not** loosen them). A flat-state Buy fires only when the
existing regime conditions hold **and** the new quality gate passes:

- **HTF alignment on by default.** Default `TrendFilterPeriod` stays > 0 for every
  ticker (no ticker ships with the filter disabled). The calibration grid must not
  be allowed to set it to 0 (remove `0` from any `TrendFilterPeriod` sweep).
- **Regime-strength margin.** Require ADX to clear its threshold by a margin, not
  just touch it: trend entry needs `adx >= ADXTrendLevel + ADXMargin`; range entry
  needs `adx <= ADXRangeLevel − ADXMargin`. New `Params` field `ADXMargin float64`
  (e.g. 2.0); `0` preserves current behavior.
- **Minimum reward:risk.** Reject an entry whose target is too close to its stop.
  For a trend entry the target proxy is `price + TrailMult·ATR`; for a range entry
  it is the Donchian `mid`. Require `(target − price) >= MinRR · (price − stop)`.
  New `Params` field `MinRR float64` (e.g. 1.5); `0` disables the check.
- **Anti-churn min-move gate.** Skip an entry when ATR is a negligible fraction of
  price (no room to clear two-sided commission). Require `atr >= MinATRFrac · price`.
  New `Params` field `MinATRFrac float64` (e.g. 0.003); `0` disables the check.

All new fields default to `0` in `adaptive.Params` semantics (behavior-preserving),
and per-ticker configs (`rusal`, `afks`) opt in with calibrated non-zero values.

### Block 4 — Quality-focused calibration objective

Replace the exposure/count-blind ranking with a quality objective that guards
against overfitting to a few lucky trades. **No exposure penalty.**

- Add a `min_trades` floor: combos with fewer than a threshold of trades are ranked
  last (treated as statistically unreliable), regardless of their PF/expectancy.
- Add `expectancy` and a downside-risk-adjusted metric (**Sortino**: mean trade
  return over downside deviation of trade returns) as selectable ranking metrics,
  alongside the existing ones.
- Default calibration metric for this strategy becomes `expectancy` (or `sortino`),
  applied **after** the `min_trades` floor.

Implementation touches `internal/service/backtest/calibrate.go` (ranking, new metric
names, min-trades floor) and `internal/domain/backtest/metrics.go` (Sortino over the
per-trade PnL series; the trade list is already on `Result.Trades`).

## Validation

- **Basket walk-forward.** Validate on 5–10 liquid MOEX tickers (not just RUAL), with
  an out-of-sample split, so "quality" is not a RUAL overfit. Re-fetch candles per
  ticker via the existing `internal/service/backtest/candles.go` path.
- **Behavior-preservation gate.** With every new `Params` knob at `0`/default-off,
  the engine must reproduce the current RUAL/AFKS journals byte-for-byte before any
  tuning. This is the first test of the implementation.
- Acceptance target: on the basket, median per-trade expectancy > 0 and profit factor
  > 1.2 out-of-sample, with trade count unconstrained (few is fine).

## Out of scope

- Shorts / margin / Phase-2 symmetry.
- Rewriting the dual-regime architecture.
- Raising exposure or trade frequency as goals in themselves.
- Position sizing / volatility targeting (Fraction stays as configured).

## Affected files

- `internal/service/trading_strategy/scalping/strategy/adaptive/adaptive.go` — Blocks 1–3.
- `internal/service/trading_strategy/scalping/strategy/adaptive/adaptive_test.go` — tests.
- `internal/service/trading_strategy/scalping/strategy/rusal/rusal.go`,
  `.../afks/afks.go` — opt-in calibrated values for the new knobs; keep HTF filter on.
- `internal/domain/backtest/metrics.go` — Sortino metric.
- `internal/service/backtest/calibrate.go` — min-trades floor, new ranking metrics, default.
- `data/params/{rusal,afks}/*.json` — refreshed scalp/grid (no `TrendFilterPeriod:0`).
