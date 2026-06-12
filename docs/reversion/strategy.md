# Reversion strategy (daily RSI mean-reversion)

Long-only, **daily timeframe** (`-interval Day1`). A pure RSI mean-reversion core: it
buys around the oversold zone and exits around the overbought zone. Both the entry and
the exit fire on an RSI zone crossing, and for each side you choose *when* the crossing
counts — entering the zone or exiting it. An optional trend filter restricts buys to a
confirmed uptrend. The only protective stop is a daily-ATR stop frozen at entry.

Volume gating, the Stochastic gate and the time-stop from the earlier hourly version are
**gone**. The Stochastic oscillator stays in `pkg/indicators` but is no longer used here.

## Entry (gates in short-circuit order)

1. **Trend filter (optional, `UseTrend`):**
   - `1` (default): require `EMA(FastEMA) > EMA(SlowEMA)` **and** `close > EMA(SlowEMA)`
     (defaults 50/200).
   - `0`: trend ignored — buy on the RSI trigger alone.
2. **RSI trigger (`EntryMode`, zone = `RSIOversold`):**
   - `0` enter zone: RSI(`RSIPeriod`) crosses **down** through `RSIOversold` (the knife).
   - `1` exit zone (default): RSI crosses **up** through `RSIOversold` (the bounce has
     started).
3. **Protective stop:** `ATRMult > 0` and `ATR > 0` required; the stop must size a
   positive risk. Stop = `entry − ATRMult × ATR(ATRPeriod)`, frozen at entry.

## Exit (first trigger wins; protective first)

1. **SL:** bar low ≤ the frozen ATR stop.
2. **RSI (`ExitMode`, zone = `RSIOverbought`):**
   - `0` enter zone (default): RSI crosses **up** through `RSIOverbought`.
   - `1` exit zone: RSI crosses **down** through `RSIOverbought`.

The protective stop is checked before the RSI exit, so on a tie the worst case for the
position wins.

## Run

```bash
# single run (daily timeframe)
go run ./cmd/backtest -ticker SBER -strategy reversion -interval Day1 \
  -months 12 -out ./reports/SBER

# grid calibration with walk-forward OOS
go run ./cmd/backtest -ticker SBER -strategy reversion -interval Day1 \
  -calibrate data/params/sber/reversion_grid.json -out ./reports/SBER \
  -months 24 -test-months 6 -min-trades 20 -metric profit_factor

# diagnose one bar
go run ./cmd/backtest -ticker SBER -strategy reversion -interval Day1 \
  -explain '2026-03-14 12:00' -months 12
```

## Params

`UseTrend, FastEMA, SlowEMA, RSIPeriod, RSIOversold, RSIOverbought, EntryMode, ExitMode,
ATRPeriod, ATRMult`. Flags (`UseTrend`, `EntryMode`, `ExitMode`) are int `0/1`; the rest
are int/float64 so the grid calibrator can sweep them. The zone semantics — *enter* vs
*exit* the zone — are shared between `EntryMode` (oversold) and `ExitMode` (overbought).

## Not yet supported

`-basket` walk-forward (the basket runner is currently momentum-only).
