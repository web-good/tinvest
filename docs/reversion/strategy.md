# Reversion strategy (RSI buy-the-dip)

Long-only, hourly. Buys sharp RSI drawdowns inside a confirmed uptrend and sells the
bounce. The conceptual mirror of the momentum strategy.

## Entry (all gates must pass, short-circuit order)

1. **Regime:** `EMA(FastEMA) > EMA(SlowEMA)` and `close > EMA(SlowEMA)` (defaults 50/200).
2. **Dip trigger** (`EntryMode`):
   - `0` confirmed (default): RSI(`RSIPeriod`, default 6) crosses **up** through
     `RSIOversold` (40) — the bounce has started.
   - `1` knife: RSI crosses **down** through `RSIOversold`.
3. **Volume:** last volume > `VolMultiplier × SMA(VolLookback)`.
4. **Stochastic (optional, `UseStoch=1`):** %K < `StochOversold` (20). During the
   indicator's warm-up window (fewer than `StochPeriod` bars) the gate blocks rather
   than silently passing.
5. **Protective stop:** `StopLossPct > 0` required; stop = `entry × (1 − StopLossPct)`.

## Exit (first trigger wins; protective first)

1. **SL:** bar low ≤ frozen `entry × (1 − StopLossPct)`.
2. **TIME:** `MaxHoldBars` bars elapsed without a bounce.
3. **RSI:** RSI crosses **up** through `RSIOverbought` (70).

## Run

```bash
# single run
go run ./cmd/backtest -ticker SBER -strategy reversion -months 12 -out ./reports/SBER

# grid calibration with walk-forward OOS
go run ./cmd/backtest -ticker SBER -strategy reversion \
  -calibrate data/params/sber/reversion_grid.json -out ./reports/SBER \
  -months 24 -test-months 6 -min-trades 20 -metric profit_factor

# diagnose one bar
go run ./cmd/backtest -ticker SBER -strategy reversion \
  -explain '2026-03-14 12:00' -months 12
```

## Params

`FastEMA, SlowEMA, RSIPeriod, RSIOversold, RSIOverbought, EntryMode, VolLookback,
VolMultiplier, UseStoch, StochPeriod, StochSmooth, StochOversold, StopLossPct,
MaxHoldBars, ATRPeriod` (ATR is display-only). All are int/float64 so the grid
calibrator can sweep them.

## Not yet supported

`-basket` walk-forward (the basket runner is currently momentum-only).
