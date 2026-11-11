# FVG imbalance RSI exit design

## Intent
Add configurable RSI length and an optional RSI exit to the M30 `fvg_imbalance` backtest strategy, and remove the stochastic entry filter.

## Behavior
- Add `RSIPeriod` and binary `UseRSIExit` parameters.
- While a position is open, RSI exit can trigger only after a completed in-position candle has RSI below 70. A later completed candle crossing RSI from below 80 to at least 80 exits at that candle's close.
- RSI already above 70 at entry does not block entry or arm the exit. The exit arms only after RSI falls below 70 while the position is open.
- Existing emergency stop, close-confirmed stop, and take-profit behavior remain. Stops retain priority; RSI and take-profit exits are evaluated at completed candle close.
- Remove `UseStochFilter` and all stochastic entry logic.

## Approaches
1. Persist arm state in `Strategy`: rejected because the strategy reconstructs decisions from snapshots and is intentionally stateless.
2. Reconstruct the in-position RSI history from completed candles on every decision: selected; preserves stateless behavior and deterministic backtests.

## Scope and constraints
Update core parameters/validation/decision logic, tests, parameter grids/literals as appropriate, and the FVG strategy documentation. Use the existing RSI indicator implementation. No live runner is in scope.

## Builder decisions
Use RSI below 70 as the arm condition and a crossing from below 80 to 80 or above as the trigger. RSI exit is checked after protective stops and before TP. Default `UseRSIExit` is disabled; choose the strategy's existing conventional RSI period default of 14.
