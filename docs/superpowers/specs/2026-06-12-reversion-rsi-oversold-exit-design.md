# Reversion: add RSI-oversold breakdown exit

**Date:** 2026-06-12
**Branch:** feat/reversion-rsi-dip
**Status:** approved design, pending implementation

## Problem

The reversion strategy (daily, long-only) currently exits an open long on either
`RSI50` (RSI crosses 50 downward, primary momentum fade) or `EMAX` (bearish
FastEMA/SlowEMA cross, regime backstop). It carries no protective stop. The owner
wants a third exit that acts as a momentum-based stop replacement: if the bounce
fails and RSI breaks back down through the oversold zone, close the position.

## New exit behavior

Add a third exit branch to `manage()`:

- **RSIOS** — RSI crosses `RSIOversold` from above:
  `in.rsiOK && crossDown(in.rsiPrev, in.rsiNow, s.p.RSIOversold)`.
  Reason code `"RSIOS"`.

The exit threshold is the **existing `RSIOversold` Param** (no new field, no new
constant). The grid already sweeps `RSIOversold`, so calibration is unchanged.

### Precedence (cosmetic — all three fill at close)

`RSI50` → `RSIOS` → `EMAX`. When several fire on the same bar the earlier one is
reported; the fill price (bar close) is identical, so order is purely the label
shown in the trade journal.

### Why no extra guard is needed

On the entry bar RSI has just crossed `RSIOversold` downward (or is already below
it via the Stochastic-cross entry branch), so `rsiPrev < RSIOversold`. `crossDown`
requires `prev >= level`, so the RSIOS exit cannot fire on the bar right after
entry — it only triggers on a **fresh** breakdown after RSI has recovered above the
zone. The `in.rsiOK` guard (warm-up sentinel protection) is sufficient, mirroring
the existing RSI50 branch.

## Params — unchanged

No `Params` field added or removed. `RSIOversold` now drives both the entry zone
and the RSIOS exit. The RSI-50 exit level stays a hard constant.

## Core changes (`reversion/strategy/core/core.go`)

- `manage()`: insert a new `case` between the RSI50 case and the EMAX case:
  ```go
  case in.rsiOK && crossDown(in.rsiPrev, in.rsiNow, s.p.RSIOversold):
      sig.Kind, sig.Reason = model.SignalSell, "RSIOS"
      sig.ExitReason = fmt.Sprintf("RSIOS: RSI %.2f→%.2f пробил зону перепроданности %.0f сверху вниз",
          in.rsiPrev, in.rsiNow, s.p.RSIOversold)
  ```
- Update the `manage()` doc comment to list the three exits and the precedence.
- Update the package doc comment (top of file) exit description.

## Tests (`core/core_test.go`)

- Add a test: RSI crosses `RSIOversold` downward while holding → sell, `Reason == "RSIOS"`.
- Add a regression test: a setup matching the entry bar (RSI already below
  `RSIOversold`, `rsiPrev < RSIOversold`) does NOT fire RSIOS — i.e. no false exit
  immediately after entry.
- Keep the existing RSI50 / EMAX / precedence tests passing.

## Docs

Update `docs/reversion/strategy.md` Exit section to list the three triggers and the
RSIOS rationale (failed-bounce / stop replacement).

## Out of scope

- Re-calibrating grids (owner runs `-calibrate` manually).
- Any change to entry, EMAX, RSI50, or the engine.
