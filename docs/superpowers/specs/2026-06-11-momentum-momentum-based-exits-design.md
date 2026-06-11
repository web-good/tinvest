# Momentum strategy: momentum-based exit conditions

## Problem

The momentum strategy (`internal/service/trading_strategy/momentum/strategy/core`)
exits on one of two mutually-exclusive modes, selected by the `UseTrail` flag:

- `UseTrail=0`: a frozen structural hard stop **or** a fixed reward-multiple
  take-profit (`TakeProfitRR`).
- `UseTrail=1`: the hard stop **or** a chandelier trailing stop (the fixed TP is
  dropped).

The fixed-RR take-profit caps winners at a predetermined target, which cuts
trends short — a common failure mode for momentum systems in a trending,
volatile name like RUAL. We want exits that let a trade ride while momentum
persists and close it when momentum fades, while keeping a catastrophic stop.

## Goal

Replace the either/or exit mode with **independently toggleable** exit triggers.
A position closes on the **first** of these to occur on a bar:

1. **Hard stop** — structural, frozen at entry. Always active (protection floor).
2. **Trailing stop** — chandelier (`recentHigh − TrailMult×ATR`), gated by `UseTrail`.
3. **Fixed take-profit** — `entry + TakeProfitRR×risk`, gated by `TakeProfitRR > 0`.
4. **MACD bearish cross** — MACD line crosses **below** its signal line
   (mirror of the bullish entry cross), gated by `UseMACDExit`.
5. **RSI overbought cross-down** — RSI crosses the overbought line from above,
   gated by `RSIPeriod > 0`.

Default parameters leave the two new exits (MACD, RSI) **disabled**, so current
behavior is unchanged and the new exits are opt-in via calibration.

## Non-goals

- No change to entry logic (trend/MACD-cross/volume/daily-ATR/daily-trend gates).
- No change to the backtest engine's fill model.
- No lower-timeframe / intrabar data stream.
- Not introducing live trading wiring beyond what the strategy already exposes.

## Parameters (`core.Params`)

Follow the existing in-codebase idiom: `<=0 disables` where a numeric knob has a
natural zero, an explicit `int` 0/1 flag otherwise.

| Param | Type | Change | Meaning |
|---|---|---|---|
| `TakeProfitRR` | float64 | **semantics change** | `>0` enables the fixed TP; `<=0` disables it (was always on when `UseTrail=0`). |
| `UseTrail` | int | **semantics change** | `1` enables the trailing stop (was "trail *instead of* TP"; now an independent toggle). |
| `UseMACDExit` | int | new | `1` enables exit on a bearish MACD cross of the signal line. |
| `RSIPeriod` | int | new | RSI length; `0` disables the RSI exit. |
| `RSIOverbought` | float64 | new | Overbought threshold (default `70`); only meaningful when `RSIPeriod > 0`. |

The hard stop stays always-on, driven by the existing `SLMult` / `SwingLowWindow`.

## Exit logic (`manage`)

Evaluation order is deterministic; protective and intrabar stops are checked
first so the worst case for the position wins ties on a bar:

1. `barLow ≤ hardSL` → reason `"SL"` (intrabar, level frozen at entry).
2. `UseTrail==1` && armed && `barLow ≤ chandelier` → reason `"TRAIL"` (intrabar).
   Arming logic unchanged (`TrailArmATR`, `MaxFavorablePrice`, `EntryATR`).
3. `TakeProfitRR > 0` && `barHigh ≥ tp` → reason `"TP"` (intrabar).
4. `UseMACDExit==1` && MACD bearish cross on this bar → reason `"MACD"` (at close).
5. `RSIPeriod > 0` && RSI crossed `RSIOverbought` top-to-bottom this bar →
   reason `"RSI"` (at close).

Fills: the engine special-cases only `"SL"` and `"TP"`, filling them at their
levels with gap adjustment; every other reason fills at the bar close. So
`"TRAIL"` already fills at close today, and the new `"MACD"` / `"RSI"` exits —
confirmed only at bar close anyway — fill at close too. **No engine change is
required** (`internal/domain/backtest/engine.go`).

## Indicators for exit (`buildInput` → `decideInput`)

`buildInput` already computes MACD; extend it to also expose the bearish cross
and the RSI series:

- `macdCrossDown bool` — mirror of the existing `crossUp`:
  `prevDiff >= 0 && currDiff < 0`, where `diff = macdLine − signalLine`.
- `rsiNow, rsiPrev float64` — the last two values of
  `indicators.RSISeries(md.Closes, RSIPeriod)` when `RSIPeriod > 0`
  (zero/unset when disabled or history is insufficient).

RSI cross-down test: `rsiPrev > RSIOverbought && rsiNow <= RSIOverbought`.

`Lookback()` must account for `RSIPeriod` so enough candles are fed when the RSI
exit is enabled.

## MinRR decoupling (footgun)

The entry-side `MinRR` check is currently computed against the
`TakeProfitRR`-based target. With `TakeProfitRR = 0` (fixed TP disabled) the
target collapses to `price`, RR becomes 0, and `MinRR` would block **every**
entry — the same class of footgun fixed for `MaxDailyATRUsed`.

Fix: apply the `MinRR` entry check **only when `TakeProfitRR > 0`**. When the
fixed TP is disabled, the trade is managed by the trail / MACD / RSI exits and a
fixed reward-to-risk entry filter no longer applies. This affects both the
`decide` decision path and the `Explain` trace.

## Defaults and grid

- Add the new fields to **all 8** per-ticker `DefaultParams()` constructors
  (`afks, mdmg, sber, nvtk, ydex, plzl, gazp, rusal`) with:
  `UseMACDExit: 0`, `RSIPeriod: 0`, `RSIOverbought: 70`. Leave `UseTrail` and
  `TakeProfitRR` at their current values. This keeps default behavior identical
  (both new exits off; current SL/TP behavior preserved).
- Extend the **RUAL** calibration grid (`data/params/rual/momentum_grid.json`)
  with an exit-tuning phase sweeping the toggles, e.g.
  `UseTrail [0,1]`, `UseMACDExit [0,1]`, `RSIPeriod [0,14]`,
  `TakeProfitRR [0,2,3]`. Other tickers' grids are out of scope for this change.

## Report / Explain

The trade-journal "Причина" column already renders `sig.Reason`, so the new
`"MACD"` / `"RSI"` reason codes flow through automatically. If a reason legend
exists in the report, add the two new codes to it.

## Testing (`core_test.go`)

Table-driven / focused tests following the existing patterns:

- `UseMACDExit=1`: position exits with reason `"MACD"` on a bearish signal cross;
  with `UseMACDExit=0` the same bar does **not** exit.
- `RSIPeriod>0`: position exits `"RSI"` when RSI crosses `RSIOverbought`
  top-to-bottom; no exit when `RSIPeriod=0`; no exit when RSI stays above or
  below the line without crossing.
- Priority: when the hard stop and a MACD/RSI condition trigger on the same bar,
  the exit reason is `"SL"` (protective stop wins).
- MinRR decoupling: with `TakeProfitRR=0` and a qualifying setup, entry still
  fires (MinRR check skipped); with `TakeProfitRR>0` the existing MinRR block
  still applies.
- `TakeProfitRR=0`: no `"TP"` exit fires even when price runs up far.

All existing momentum core tests must continue to pass.

## Files touched

- `internal/service/trading_strategy/momentum/strategy/core/core.go`
  (Params, `Lookback`, `buildInput`, `decideInput`, `decide`, `manage`,
  `Explain`).
- `internal/service/trading_strategy/momentum/strategy/core/core_test.go`.
- 8 per-ticker `DefaultParams()` files.
- `data/params/rual/momentum_grid.json`.

## Out of scope / follow-ups

- Validating the new exits on OOS (walk-forward) — done after implementation via
  the existing `-calibrate` / `-test-months` flow. The strategy is known to be
  overfit; this change is a hypothesis to test, not a confirmed improvement.
- Tuning grids for the other 7 tickers.
