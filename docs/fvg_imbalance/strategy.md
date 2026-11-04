# FVG imbalance

`fvg_imbalance` is a backtest-only, long-only strategy on completed M30 candles.
It reconstructs zones and structure from the supplied history on each decision.
GAZP has an explicit parameter literal; unregistered tickers use core defaults.

## Formation and structure

For three consecutive M30 candles `a`, `b`, `c`, a bullish fair-value gap forms when:

- `close_b > open_b`;
- `high_a < low_c`;
- the green body of `b` crosses the gap from below to above:
  `open_b <= high_a` and `close_b >= low_c`.

The zone is `[lower, upper] = [high_a, low_c]`. Consecutive means exact 30-minute
timestamps, so a formation cannot cross a missing candle or overnight gap.
Its height must be at least `MinGapATR * ATR(ATRPeriod)`. Wilder ATR is computed
from exactly `2 * ATRPeriod` candles ending at `c`, and remains frozen at formation
for every subsequent retest, stop and entry-distance calculation.

Optional structure gates use strict fractal pivots: a high or low must exceed or
be below all `SwingK` bars on each side, and is usable only after the right-hand
bars have completed. Equal values do not form pivots. Searches are bounded by
`StructureLookbackBars` before `c` and use the most recent eligible pivot.

A bullish liquidity sweep is a bar `s` whose low is strictly below the most recent
swing low confirmed before `s` and whose close is strictly above it. It must occur
by `b`, with `1 <= c-s <= MaxSweepToFVGBars`; the newest qualifying sweep is used.
A bullish break of structure (BOS) requires a close strictly above a confirmed
swing high. With a sweep required, the high is the most recent one usable at `s`
and the break can occur from `s` through `c`. Without a sweep, the high is the
most recent one usable at `a` and `b` or `c` must break it. With both gates on,
the order is sweep, BOS, completed FVG; the sweep and BOS may share a bar.
Either gate can also be used independently.

The retained lookback is `max(3, 2 * ATRPeriod) + MaxRetestBars`; when a
structure gate is enabled, the maximum also includes
`StructureLookbackBars + 2 * SwingK + 3`. Missing or malformed required history
fails closed.

## Retest and execution

A zone can signal on the first through `MaxRetestBars`-th completed candle after
formation, inclusive. A later low at or below `lower` invalidates the zone,
including on the retest candle itself. A protected retest has
`lower < low <= upper`. With `RequireCloseAboveZone = 1`, the same candle must
also close strictly above `upper`; `0` accepts a protected touch without this
close confirmation. With `RequireFirstRetest = 1`, every earlier post-formation
bar must have `low > upper`: any earlier touch consumes the zone even if it
failed close confirmation or another gate, or its entry could not fill. When
several zones qualify, the newest qualifying zone wins. The strategy opens at
most one position at a time.

At the retest decision, `HTFTrendEMA > 0` requires the last completed H4 close
to be strictly above its EMA. At least that many finite H4 closes and a finite EMA
are required; missing, insufficient or non-finite H4 data blocks entry. The
backtest loads H4 with a one-year lead-in when the selected parameters or any
calibration phase enable the gate. It exposes H4 candles completed at or before
the M30 decision close; later-closing candles remain hidden. `0` disables the
gate, and position exits do not depend on H4 data.

The signal is emitted after the retest candle closes. `EntryAtNextOpen` schedules
the buy for the following M30 open, and `FillBy` is that exact timestamp
(`retest candle timestamp + 30 minutes`). If that slot is absent, the engine
drops the pending buy instead of carrying it across a night, weekend, or data gap.
A new entry whose exact `FillBy` falls on Saturday or Sunday is discarded by the
strategy. When `MaxEntryDistanceATR > 0`, the engine also cancels a buy if the
actual next open is strictly above `upper + MaxEntryDistanceATR * formationATR`;
equality is allowed. This limit does not reject a gap down.

## Stop, target and time exit

The stop is `lower - StopBufferATR * formationATR`; a non-positive or non-finite
stop rejects the entry. For an open position, the target is
`actual fill + TakeProfitR * (actual fill - stop)`, derived from the position
purchase price. The signal also carries a provisional target based on the retest
close; position management uses the actual fill instead. `TakeProfitR = 0`
disables target exits. The buffer therefore widens both the stop and R target.

Exit priority is stop loss, then take profit, then time exit. If the buy fills at
or below the stop, the fill candle immediately signals a stop exit; the engine
uses the available opening price for a gap through the stop. With `MaxHoldBars > 0`,
a position exits at the current M30 close once `BarsHeld >= MaxHoldBars`. Age is
the number of processed M30 bars since entry, with the fill bar counted as zero.
There is no trailing stop or forced end-of-day close.

## Parameters

| Parameter | Core default | GAZP baseline | Validation and meaning |
|---|---:|---:|---|
| `ATRPeriod` | 14 | 14 | Positive integer; M30 Wilder ATR period |
| `MinGapATR` | 0.25 | 0.8 | Finite, non-negative minimum zone height in formation ATRs |
| `MaxRetestBars` | 8 | 8 | Positive integer; maximum later candles eligible for retest |
| `RequireCloseAboveZone` | 1 | 0 | `0` or `1`; require a close strictly above the zone |
| `TakeProfitR` | 1.5 | 1.5 | Finite, non-negative target multiple; `0` disables target |
| `RequireFirstRetest` | 0 | 0 | `0` or `1`; accept only the first touch |
| `RequireBullishBOS` | 0 | 0 | `0` or `1`; require bullish BOS |
| `RequireLiquiditySweep` | 0 | 0 | `0` or `1`; require a preceding bullish sweep |
| `SwingK` | 2 | 2 | Positive integer; bars on each side of a pivot |
| `StructureLookbackBars` | 48 | 48 | Positive integer; structure search history before formation |
| `MaxSweepToFVGBars` | 8 | 8 | Positive integer; maximum sweep-to-formation distance |
| `HTFTrendEMA` | 0 | 0 | Non-negative H4 EMA period; `0` disables the gate |
| `MaxEntryDistanceATR` | 0 | 0 | Finite, non-negative next-open limit above the zone in formation ATRs; `0` disables |
| `MaxHoldBars` | 0 | 0 | Non-negative holding limit in processed bars; `0` disables |
| `StopBufferATR` | 0 | 0 | Finite, non-negative stop buffer below the zone in formation ATRs |

Structural helper parameters must stay positive even when their gates are off.
Partial parameter JSON overlays the selected ticker defaults and is validated by
the backtest binding. Other intervals are rejected; invalid parameters or a
non-M30 step make the core fail closed.

## Backtest and calibration

The refreshed full-window baseline uses:

```bash
go run ./cmd/backtest \
  -ticker GAZP -strategy fvg_imbalance -interval Minutes30 \
  -out reports/GAZP/fvg_imbalance/r20261104/baseline_full \
  -months 24 -risk-pct 1 -commission 0.0005 -refresh
```

The staged grid varies formation, context, execution and holding controls in that
order. The primary rolling walk-forward uses 12 months of training and 3 months
of testing inside a 24-month window:

```bash
go run ./cmd/backtest \
  -ticker GAZP -strategy fvg_imbalance -interval Minutes30 \
  -calibrate data/params/fvg_imbalance/gazp/grid.json \
  -out reports/GAZP/fvg_imbalance/r20261104/quality_wf \
  -months 24 -train-months 12 -test-months 3 \
  -min-trades 20 -metric profit_factor \
  -risk-pct 1 -commission 0.0005 -refresh
```

`-risk-pct 1` sizes positions by stop risk to normalize comparisons across stop
widths. The minimum trade count applies to each training ranking; inspect any
phase where all combinations fall below it. Walk-forward results inform a later
default-selection decision and do not automatically change the ticker literal.
