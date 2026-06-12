# Reversion strategy (RSI + Stochastic dual confirmation)

Long-only, **daily timeframe** (`-interval Day1`). A mean-reversion core driven by the
agreement of two oscillators — RSI and the Stochastic %D line. It buys when one oscillator
is already inside its oversold zone and the other crosses into it, and exits when one is
already overbought and the other crosses up into the overbought zone. An optional trend
filter restricts buys to a confirmed uptrend. The only protective stop is a daily-ATR stop
frozen at entry.

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
3. **Protective stop:** `ATRMult > 0` and `ATR > 0` required; stop =
   `entry − ATRMult × ATR(ATRPeriod)`, frozen at entry.

## Exit (first trigger wins; protective first)

1. **SL:** bar low ≤ the frozen ATR stop.
2. **XOVER** — at least one of:
   - RSI crosses **up** through `RSIOverbought` **and** Stoch %D is already
     `> StochOverbought`;
   - Stoch %D crosses **up** through `StochOverbought` **and** RSI is already
     `> RSIOverbought`.

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

`UseTrend, FastEMA, SlowEMA, RSIPeriod, RSIOversold, RSIOverbought, StochKPeriod,
StochDSmooth, StochOversold, StochOverbought, ATRPeriod, ATRMult`. Flags (`UseTrend`) are
int `0/1`; the rest are int/float64 so the grid calibrator can sweep them — including the
Stochastic zones and periods.

## Not yet supported

`-basket` walk-forward (the basket runner is currently momentum-only).
