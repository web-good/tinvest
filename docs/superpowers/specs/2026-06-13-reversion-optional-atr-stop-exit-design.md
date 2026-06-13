# Reversion: optional ATR-stop exit (toggle vs RSI-oversold breakdown)

Date: 2026-06-13
Branch: feat/reversion-rsi-dip
Package: `internal/service/trading_strategy/reversion/strategy/core`

## Problem

The reversion strategy's `manage()` exits an open long on one of three signals, in
precedence order, all filling at close:

1. **RSI50** — RSI crosses the 50 midline downward (primary momentum fade).
2. **RSIOS** — RSI breaks back down through the oversold zone from above
   (failed-bounce / RSI-oversold breakdown).
3. **EMAX** — bearish EMA cross (regime-break backstop).

We want the **middle** exit (RSIOS) to become one of two mutually exclusive
branches, selected by a flag:

- flag off → keep RSIOS (RSI crossing the oversold lower boundary downward), or
- flag on → exit when price has moved below the entry by the size of the daily ATR.

RSI50 and EMAX stay unchanged.

## Design

### Params (all int/float for grid calibration via reflection)

Add three fields to `core.Params`:

- `UseATRStop int` — `0` (default) selects the **RSIOS** branch (current behavior);
  `1` selects the **ATRSL** branch. Mutually exclusive. Naming mirrors the existing
  `UseTrend` flag.
- `ATRPeriod int` — daily ATR length; only consulted when `UseATRStop==1`.
- `StopATRMult float64` — stop distance multiplier; default `1.0` ("below the size of
  the daily ATR"). Tunable so calibration can sweep it.

The strategy runs on the daily interval (`-interval Day1`), so ATR computed on
`md.Highs/Lows/Closes` IS the daily ATR — no higher-timeframe series needed.

### Exit logic (`manage()`)

Precedence order is preserved; only the middle branch is flag-gated:

```go
switch {
case in.rsiOK && crossDown(in.rsiPrev, in.rsiNow, rsiExitLevel):
    // RSI50 — unchanged
case s.p.UseATRStop == 0 && in.rsiOK && crossDown(in.rsiPrev, in.rsiNow, s.p.RSIOversold):
    // RSIOS — RSI crosses the oversold lower boundary downward (current behavior)
case s.p.UseATRStop == 1 && in.pos.EntryATR > 0 &&
    in.price <= in.pos.PurchasePrice - s.p.StopATRMult*in.pos.EntryATR:
    // ATRSL — price dropped below entry by the daily ATR
case in.emaOK && crossDown(in.emaFastPrev-in.emaSlowPrev, in.emaFast-in.emaSlow, 0):
    // EMAX — unchanged
}
```

- `UseATRStop==0` → RSIOS active, ATRSL inert.
- `UseATRStop==1` → ATRSL active, RSIOS inert.

ATRSL emits `SignalSell` with `Reason="ATRSL"` and a human-readable `ExitReason`.

### ATR freeze at entry

The stop is anchored to the entry: threshold = `PurchasePrice − StopATRMult×EntryATR`,
where `EntryATR` is the daily ATR captured at the moment of entry (frozen, not live).
Rationale: the request anchors the distance to the entry point; a live ATR makes the
stop drift (widening exactly when volatility expands), giving a non-deterministic,
poorly-reproducible level. Freezing matches how the momentum strategy already handles
`EntryATR`.

Wiring:

- `buildInput` computes `atr := indicators.ATR(md.Highs, md.Lows, md.Closes, s.p.ATRPeriod)`
  when `ATRPeriod > 0` (else `0`), and carries it in `decideInput` (new field `atr`). No
  separate readiness flag is needed: the entry path only stamps a positive ATR, and the
  exit path gates on `in.pos.EntryATR > 0`.
- `decide()` on a Buy signal sets `sig.ATR = in.atr`. The backtest engine already
  persists `sig.ATR` into `Position.EntryATR` via `p.open(...)` (see
  `internal/domain/backtest/engine.go:108`).
- `manage()`'s ATRSL branch reads `in.pos.EntryATR`.

### Safety guard (live trading)

Live trading does NOT persist entry-locked fields — `Position.EntryATR` is `0` there
(see `scalping/trade.go`). Without a guard the stop would be `price − 0`, triggering
immediately on the first managed bar. The ATRSL branch therefore gates on
`in.pos.EntryATR > 0`; when zero it is simply skipped (same discipline as the
`rsiOK`/`emaOK` warm-up sentinels). The flag has no effect in live trading until
entry state is persisted, which is acceptable and explicit.

### Fill semantics

ATRSL is a close-fill exit like every other reversion exit (daily timeframe, single
close-fill philosophy). It uses a dedicated `Reason="ATRSL"`, so it does NOT engage the
engine's intrabar stop path (`min(StopLoss, c.Open)`), which is reserved for
`Reason=="SL"/"TRAIL"`. Trigger and fill are both at the bar close.

### Lookback / defaults / per-ticker / grid

- `Lookback()` accounts for `ATRPeriod + 1` when `UseATRStop == 1` (ATR needs
  period+1 candles).
- Per-ticker defaults (`afks`, `rusal`, and any others present): add
  `UseATRStop: 0, ATRPeriod: 14, StopATRMult: 1.0` — preserves current behavior.
- `data/params/rual/reversion_grid.json`: add `UseATRStop: [0, 1]`,
  `ATRPeriod: [14]`, `StopATRMult: [1.0, 1.5, 2.0]` to the entry phase grid.

### Tests (`core_test.go`)

- ATRSL fires when `price <= PurchasePrice − Mult×EntryATR` (UseATRStop=1).
- ATRSL does NOT fire just above the threshold (UseATRStop=1).
- ATRSL skipped when `EntryATR == 0` (live-trading guard).
- `UseATRStop=0` keeps RSIOS firing and ATRSL inert.
- Precedence: RSI50 wins over ATRSL; ATRSL wins over EMAX (UseATRStop=1).

### Docs

- Update the `core` package docstring (exit list now describes the toggle).
- Update `docs/` reversion exit documentation to describe `UseATRStop` and the ATRSL
  branch.

## Out of scope

- Live persistence of `EntryATR` (the guard makes the flag a no-op live until that
  exists; not changed here).
- Intrabar stop execution for ATRSL (close-fill only, consistent with reversion).
- Changes to RSI50 or EMAX exits.
