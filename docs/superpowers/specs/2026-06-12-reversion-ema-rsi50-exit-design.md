# Reversion: EMA-cross / RSI-50 exit, no stop-loss

**Date:** 2026-06-12
**Branch:** feat/reversion-rsi-dip
**Status:** approved design, pending implementation

## Problem

The reversion strategy currently exits an open long by (1) a hard ATR stop frozen
at entry, checked first, then (2) a dual-overbought XOVER (RSI crosses up into the
overbought zone while Stochastic %D is already overbought, or vice versa). The owner
wants a simpler, momentum-fade exit and no protective stop at all.

## New exit behavior

An open long exits on **either** condition (OR), both filled at the bar close:

1. **RSI50** — RSI crosses the 50 line from above: `rsiPrev >= 50 && rsiNow < 50`.
   The primary, fast exit: the reversion bounce has lost momentum.
2. **EMAX** — bearish EMA cross: `emaFastPrev >= emaSlowPrev && emaFast < emaSlow`,
   i.e. FastEMA drops below SlowEMA. A slow regime-break backstop on the Day1
   timeframe; reuses the existing `FastEMA`/`SlowEMA` params.

If both fire on the same bar, RSI50 is reported (it is the primary intent). The
fill price is identical (close) either way, so precedence is cosmetic.

The hard ATR stop is **removed entirely**. There is no protective stop.

## Decisions

- **RSI exit level is a hard constant `50`**, not a tunable param. (Owner choice.)
- **EMA cross reuses the existing `FastEMA`/`SlowEMA` Params**, the same pair the
  optional trend filter uses. On Day1 with e.g. 50/200 the death cross is rare, so
  RSI50 carries the everyday exit and EMAX is a backstop. (Owner named these EMAs.)
- **Entry no longer requires an ATR stop.** The old entry gate rejected a setup
  when `ATRMult <= 0 || atr <= 0` and froze a stop. With no stop, that gate is
  spurious and is removed. Entry is unchanged otherwise: optional trend filter +
  dual oversold confirmation (RSI + Stochastic %D).
- **Dead params removed from `core.Params`:** `ATRPeriod`, `ATRMult`,
  `RSIOverbought`, `StochOverbought` — none are referenced after this change.

## Params after the change

Kept: `UseTrend`, `FastEMA`, `SlowEMA`, `RSIPeriod`, `RSIOversold`,
`StochKPeriod`, `StochDSmooth`, `StochOversold`.

Removed: `ATRPeriod`, `ATRMult`, `RSIOverbought`, `StochOverbought`.

## Core changes (`reversion/strategy/core/core.go`)

- `Params`: drop the four removed fields.
- `Lookback()`: drop the `ATRPeriod+1` term.
- `decideInput`: drop `atr` and `barLow`; add `emaFastPrev`, `emaSlowPrev`. Keep
  `rsiNow/rsiPrev/rsiOK`, `stochNow/stochPrev/stochOK`, `emaFast/emaSlow`.
- `buildInput`: stop computing ATR; compute the EMA series and take the last two
  values for fast and slow (need `len(e) >= 2`, else prev = now so no false cross).
- `decide` (entry): remove the ATR-stop block; on a buy, set `sig.RSI` and
  `sig.EntryReason` only (no `sig.StopLoss`, no `sig.ATR`).
- `manage` (exit): replace the SL-first / XOVER switch with the two new conditions:
  - RSI test guarded by `in.rsiOK`: `rsiPrev >= 50 && rsiNow < 50` → `SignalSell`,
    `Reason = "RSI50"`. The warm-up sentinel (rsiOK false, values 0) cannot pass
    `0 >= 50`, but gate on `rsiOK` explicitly for clarity.
  - else bearish EMA cross → `SignalSell`, `Reason = "EMAX"`. The `emaPrev == emaNow`
    warm-up state (series shorter than 2) yields no cross, so no false trigger.
  - Do not set `sig.StopLoss`.
- `entryReason`: drop the SL clause.
- `exitFired`, `crossDown`-on-overbought, the overbought helpers: remove what is now
  unused. `crossDown`/`crossUp` stay where still used (entry still uses `crossDown`
  for oversold; `crossUp` may be deleted if unused).
- `Explain`: drop the ATR-stop gate; entry explanation otherwise unchanged.
- `indicatorsReady`: still used by `entryFired` (entry needs both oscillators). The
  exit does **not** call it — exit gates on `rsiOK` alone, since Stochastic is no
  longer part of the exit.

## Engine / model — unchanged

The backtest engine fills a `SignalSell` at close for any `Reason` other than
`SL`/`TRAIL`/`TP` (engine.go:117), so `RSI50`/`EMAX` need no engine change. The
shared `model.Signal` and `strategy.Position` keep their `StopLoss`/`ATR` fields;
reversion simply stops populating them (`StopLoss` defaults to 0, which the engine
treats as "no stop", and reversion never auto-stops).

## Per-ticker defaults & registry

Edit all 8 per-ticker `DefaultParams()` and `genericReversionDefaults()` in
`reversion_registry.go` to drop `ATRPeriod`, `ATRMult`, `RSIOverbought`,
`StochOverbought`. Keep the remaining fields at their current values.

## Calibration grids

Rewrite the 8 `data/params/<ticker>/reversion_grid.json` so no phase sweeps a
removed field. The `exit` phase no longer has tunables specific to the exit
(RSI-50 is constant, EMA cross reuses entry EMAs), so the grids reduce to entry/regime
params only: `FastEMA`, `SlowEMA`, `RSIPeriod`, `RSIOversold`, `StochKPeriod`,
`StochDSmooth`, `StochOversold`, `UseTrend`. Remove `RSIOverbought`,
`StochOverbought`, `ATRMult` entries.

## Tests (`core/core_test.go`)

- Remove the hard-SL exit test(s) and the dual-overbought XOVER test(s).
- Add exit tests:
  - RSI crosses 50 downward → sell with `Reason == "RSI50"`.
  - FastEMA crosses below SlowEMA → sell with `Reason == "EMAX"`.
  - No exit when RSI stays above 50 and EMAs hold (fast still above slow).
  - Both fire same bar → `Reason == "RSI50"` (precedence).
- Update entry tests that asserted `sig.StopLoss`/ATR gating to the new no-stop entry.
- Keep/adjust the registry test for the trimmed `Params`.

## Out of scope

- Re-calibrating the grids (owner runs `-calibrate` manually afterwards).
- Trailing stops, take-profit, or any new exit beyond the two specified.
- Live-trading wiring changes.
