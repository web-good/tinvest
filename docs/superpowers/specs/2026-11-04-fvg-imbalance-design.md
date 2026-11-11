# FVG imbalance — GAZP M30 backtest design

## Intent

Add a separate, long-only `fvg_imbalance` backtest strategy for GAZP. It trades a bullish
three-candle fair-value gap (FVG) on completed 30-minute candles, and lets calibration compare a
confirmed retest with a more aggressive retest. This is deliberately not a live-trading change.

## Goals and boundaries

- Run with `cmd/backtest -strategy fvg_imbalance -interval Minutes30`.
- Supply an explicit GAZP literal and a GAZP calibration grid.
- Keep the strategy pure and compatible with the existing scalping strategy/backtest interfaces.
- Do not modify the deleted `pdl_rsi_m30` work or existing GAZP strategies.
- No live adapter, order placement, scanner, short trades, trailing stop, or extra trend filters.

## Approaches considered

1. **Separate backtest strategy (chosen).** A dedicated core, registry binding, GAZP literal, and
   grid keep FVG behaviour and its calibration independent from other entry models.
2. Extend `rsi_zone`. Rejected: its RSI critical-zone signal and exit rules are unrelated to an
   FVG retest, and its parameter surface would become ambiguous.
3. Build live execution now. Rejected for this slice: live state must safely persist active zones,
   position ownership, and idempotent orders; it should follow validated backtest results.

## Signal rules

All conditions use only completed M30 candles. Let candles `a`, `b`, and `c` be consecutive,
with `c` being the just-closed candle.

1. A bullish FVG forms when `b` is green (`close_b > open_b`), `high_a < low_c`, and the green
   body of `b` crosses the gap from below to above: `open_b <= high_a` and
   `close_b >= low_c`. This requires the middle candle to be the upward impulse that creates
   the imbalance, rather than merely a body contained inside a pre-existing gap.
2. The zone is `[lower, upper] = [high_a, low_c]`. It is valid only when
   `upper - lower >= MinGapATR × ATR(ATRPeriod)`; ATR is Wilder ATR over the M30 bars, evaluated
   and frozen at the formation candle `c`, with no look-ahead. A later retest does not requalify
   a zone using a newer ATR.
3. A zone becomes eligible only after `c` has closed. It lasts for at most `MaxRetestBars` later
   completed candles; the initial value is 8 (four hours). A later candle with `low <= lower`
   invalidates it permanently.
4. A retest candle has `low <= upper` and `low > lower`; it therefore reaches the zone without
   piercing its lower boundary. A close exactly on the lower boundary is treated as a pierce.
5. When `RequireCloseAboveZone = 1` (the default), that same candle must close strictly above
   `upper`. When it is `0`, the protected touch alone confirms the signal.
6. At most one position may be open. With several valid zones on a retest, use the newest one;
   this is the most recent impulse and gives deterministic, stateless reconstruction from the
   candle history.
7. The buy signal is emitted only after the retest candle closes. The existing engine's
   `EntryAtNextOpen` fills it at the following M30 open. `FillBy` is set to that exact next M30
   open, so a missing slot (night, weekend, or data gap) drops the signal rather than entering at
   a later price.

## Risk and exits

- Stop-loss is the selected zone's `lower` boundary, frozen at entry.
- Take-profit is frozen from the actual fill: `fill + TakeProfitR × (fill - stop)`. The core
  derives this level from `Position.PurchasePrice` and its frozen zone stop on every management
  decision, rather than trusting the engine's signal-time `Position.TakeProfit`; the signal still
  carries its close-based target for report/risk-sizing compatibility.
- `TakeProfitR` defaults to `1.5`; `0` disables the target.
- On each position candle the standard engine priority applies: stop first, then target. If the
  next-open fill is at or below the lower boundary, the fill bar immediately produces `SL`; the
  engine closes at its open, which is the available gapped price. No other exit and no end-of-day
  close are introduced.
- A buy signal carries stop/target calculated from the signal close for risk sizing; the engine
  recalculates the frozen levels from the actual next-open fill, as the existing fixed-R
  strategies do.

## Tunables and GAZP defaults

All tunables are numeric so they work with the reflection-based grid expander.

| Field | Default | Meaning |
|---|---:|---|
| `ATRPeriod` | 14 | Wilder ATR period on M30 candles |
| `MinGapATR` | 0.25 | minimum FVG height in M30 ATRs |
| `MaxRetestBars` | 8 | number of later M30 candles in which the zone may be retested |
| `RequireCloseAboveZone` | 1 | `1` requires close above the zone; `0` permits any protected touch |
| `TakeProfitR` | 1.5 | target in risks above the actual fill; `0` disables it |

Invalid parameters fail parsing: `ATRPeriod` and `MaxRetestBars` must be positive integers,
`MinGapATR` and `TakeProfitR` must be non-negative, and `RequireCloseAboveZone` must be `0` or
`1`.

`internal/service/trading_strategy/fvg_imbalance/strategy/gazp.Params()` repeats these defaults
so a future change to core defaults cannot silently alter GAZP.

The initial `data/params/fvg_imbalance/gazp/grid.json` sweeps `MinGapATR`,
`MaxRetestBars`, `RequireCloseAboveZone`, and `TakeProfitR`, including the defaults above.

## Code shape and verification

- `strategy/core`: parameter validation, FVG discovery/retest selection, buy/exit decisions, and
  focused unit tests for formation, formation-time ATR, expiry, invalidation, confirmed vs
  touch-only entry, missing-next-slot expiry, actual-fill target, gap-through-stop, and stop
  priority.
- `strategy/gazp`: GAZP literal and literal test.
- `internal/service/backtest/fvg_imbalance_registry.go`: M30-only binding with partial JSON
  overrides; registry tests cover interval rejection and GAZP/default lookup.
- `cmd/backtest/main.go`: route `-strategy fvg_imbalance` to the new binding.
- `docs/fvg_imbalance/strategy.md`: mechanics and invocation; it will not contain calibration
  results.

Tests are written first and run red before production implementation, followed by the focused
packages and the repository's applicable Go test suite.
