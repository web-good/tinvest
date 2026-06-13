# Reversion strategy (RSI + Stochastic dual confirmation)

Long-only, **daily timeframe** (`-interval Day1`). A mean-reversion core driven by the
agreement of two oscillators — RSI and the Stochastic %D line. It buys when one oscillator
is already inside its oversold zone and the other crosses into it. It exits on three signals
— RSI crossing 50 downward, a flag-selected middle exit (RSIOS or ATR stop), and a bearish
FastEMA/SlowEMA cross. An optional trend filter restricts buys to a confirmed uptrend.

The Stochastic working line is **%D** (SMA of %K over `StochDSmooth`; `StochDSmooth=1`
gives the raw %K). Volume gating and the time-stop from earlier versions are gone.

## Entry (gates in short-circuit order)

1. **Trend filter (optional, `UseTrend`):** `1` (default) requires
   `EMA(FastEMA) > EMA(SlowEMA)` and `close > EMA(SlowEMA)` (defaults 50/200); `0` ignores
   trend.
2. **Dual oversold confirmation** — at least one of:
   - RSI(`RSIPeriod`) crosses **down** through `RSIOversold` **and** Stoch %D is already
     `< StochOversold`;
   - Stoch %D crosses **down** through `StochOversold` **and** RSI is already
     `< RSIOversold`.
   Both crossing into the zone on the same bar also fires.

## Exit (first trigger wins)

The middle exit is selected by the `UseATRStop` flag. An open long exits on one of three
signals, filled at the bar close, in this precedence order:

1. **RSI50:** RSI crosses the 50 line from above (`prev ≥ 50`, `now < 50`) — the primary
   momentum-fade exit.
2. **Middle exit (flag-selected):**
   - `UseATRStop = 0` → **RSIOS:** RSI crosses `RSIOversold` from above
     (`prev ≥ RSIOversold`, `now < RSIOversold`) — the failed-bounce exit. It cannot fire
     on the bar right after entry, where RSI is already below the zone.
   - `UseATRStop = 1` → **ATRSL:** price falls to/below `PurchasePrice − StopATRMult ×
     EntryATR`, where `EntryATR` is the daily ATR (length `ATRPeriod`) frozen at entry.
     Guarded by `EntryATR > 0` and `StopATRMult > 0`, so it stays inert in live trading
     (entry ATR is not persisted) and on a zero multiplier.
3. **EMAX:** bearish EMA cross — `EMA(FastEMA)` drops below `EMA(SlowEMA)`. A slow
   regime-break backstop; reuses the same EMAs as the trend filter.

If several fire on the same bar the earliest in this order is reported; the fill (close)
is identical either way.

## Run

```bash
# single run (daily timeframe)
go run ./cmd/backtest -ticker SBER -strategy reversion -interval Day1 \
  -months 12 -out ./reports/SBER

# grid calibration with walk-forward OOS (Stochastic zones/periods are swept)
go run ./cmd/backtest -ticker SBER -strategy reversion -interval Day1 \
  -calibrate data/params/sber/reversion_grid.json -out ./reports/SBER \
  -months 24 -test-months 6 -min-trades 20 -metric profit_factor

# diagnose one bar
go run ./cmd/backtest -ticker SBER -strategy reversion -interval Day1 \
  -explain '2026-03-14 12:00' -months 12
```

## Params

`UseTrend, FastEMA, SlowEMA, RSIPeriod, RSIOversold, StochKPeriod, StochDSmooth,
StochOversold, UseATRStop, ATRPeriod, StopATRMult`. Flags (`UseTrend`, `UseATRStop`) are
int `0/1`; the rest are int/float64 so the grid calibrator can sweep them. The RSI-50
exit level is a fixed constant, not a param.

- `UseATRStop` — `0` (default): use RSIOS as the middle exit; `1`: use the ATR-based hard
  stop instead.
- `ATRPeriod` — daily ATR lookback length; consulted only when `UseATRStop=1`.
- `StopATRMult` — stop distance multiplier: stop placed at `PurchasePrice − StopATRMult ×
  EntryATR`; default `1.0`; `0` disables the stop even when `UseATRStop=1`.

## Not yet supported

`-basket` walk-forward (the basket runner is currently momentum-only).
