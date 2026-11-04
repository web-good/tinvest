# FVG imbalance

`fvg_imbalance` is a backtest-only, long-only strategy on completed M30 candles.
It uses no live adapter or higher-timeframe filter. GAZP has an explicit parameter
literal; unregistered tickers use the core defaults.

## Formation and qualification

For three consecutive M30 candles `a`, `b`, `c`, a bullish fair-value gap forms when:

- `close_b > open_b`;
- `high_a < low_c`;
- the green body of `b` fits inside the gap: `open_b >= high_a` and `close_b <= low_c`.

The zone is `[lower, upper] = [high_a, low_c]`. Consecutive means exact 30-minute
timestamps, so a formation cannot cross a missing candle or overnight gap.
Its height must be at least `MinGapATR * ATR(ATRPeriod)`. ATR is Wilder ATR,
calculated at formation from a finite window of exactly `2 * ATRPeriod` candles
ending at `c`. This supplies the initial seed plus `ATRPeriod - 1` smoothing
steps. It does not use all available history. The formation must have that
warmup and all three formation candles available.

The ATR window is anchored to `c`: both gap qualification and signal ATR stay
fixed when the zone is reconstructed at later retests. The retained lookback is
`max(3, 2 * ATRPeriod) + MaxRetestBars` candles.

## Retest and execution

A zone can signal on the first through `MaxRetestBars`-th completed candle after
formation, inclusive. A later low at or below `lower` invalidates the zone,
including on the retest candle itself. A protected retest has
`lower < low <= upper`. With `RequireCloseAboveZone = 1`, the same candle must
also close strictly above `upper`; `0` accepts a protected touch without this
close confirmation. When several zones qualify, the newest qualifying zone wins.
The strategy opens at most one position at a time.

The signal is emitted after the retest candle closes. `EntryAtNextOpen` schedules
the buy for the following M30 open, and `FillBy` is that exact timestamp
(`retest candle timestamp + 30 minutes`). If that slot is absent, the engine
drops the pending buy instead of carrying it across a night, weekend, or data gap.

## Stop and target

The stop is the selected zone's frozen `lower` boundary. For an open position,
the target is `actual fill + TakeProfitR * (actual fill - stop)`, derived from
the position purchase price. The signal also carries a provisional target based
on the retest close; position management uses the actual fill instead.
`TakeProfitR = 0` disables target exits.

The stop takes priority if both stop and target are reached in one candle. If
the buy fills at or below the stop, the fill candle immediately signals a stop
exit; the engine uses the available opening price for a gap through the stop.
There is no trailing stop, time exit, or forced end-of-day close.

## Parameters

| Parameter | Default | Validation and meaning |
|---|---:|---|
| `ATRPeriod` | 14 | Positive integer; M30 Wilder ATR period |
| `MinGapATR` | 0.25 | Finite, non-negative minimum zone height in formation ATRs |
| `MaxRetestBars` | 8 | Positive integer; maximum later candles eligible for retest |
| `RequireCloseAboveZone` | 1 | Exactly `0` or `1`; require a close strictly above the zone |
| `TakeProfitR` | 1.5 | Finite, non-negative target multiple; `0` disables target |

GAZP repeats these defaults in its ticker literal. Partial parameter JSON overlays
the selected ticker defaults and is validated by the backtest binding. Other
intervals are rejected; invalid parameters or a non-M30 step make the core fail
closed.

## Backtest and calibration

Run with the M30 interval explicitly:

```bash
go run ./cmd/backtest -ticker GAZP -strategy fvg_imbalance -interval Minutes30 -months 24 -out ./reports/GAZP/fvg_imbalance
```

The single-phase grid varies gap qualification, retest lifetime, close confirmation,
and target multiple while retaining the default ATR period:

```bash
go run ./cmd/backtest -ticker GAZP -strategy fvg_imbalance -interval Minutes30 -calibrate data/params/fvg_imbalance/gazp/grid.json -months 24 -min-trades 20 -test-months 6 -metric profit_factor -out ./reports/GAZP/fvg_imbalance
```
