# POSI FVG imbalance calibration plan

## Frozen criteria and execution

- Run date: 2026-11-12 (CLI resolves `now` independently on each run; record actual report dates and reject runs whose windows differ).
- Instrument/source: POSI candles from the Tinkoff Invest Market Data API, using the repository candle cache. Use completed M30, daily, and H4 candles only; 1-year daily/H4 lead-in warms EMA/ATR. The strategy itself uses completed daily ATR(14), M30 stochastic, and completed H4 data supplied by the backtest engine.
- Training: 24 months `[T-36 months, T-12 months)`; holdout: next 12 months `[T-12 months, T)`. If the source does not contain a complete holdout or training window, stop before viewing holdout results and leave the ticker literal unchanged.
- Execution: cash 100,000 RUB, fixed fraction 1.0, risk-pct 0, lot from instrument metadata, commission 0.0005 per turnover side, next M30 open execution. Same candle snapshot and costs for every grid.
- Candidate score: expectancy. Eligibility at each stage and each exit branch: at least 20 closed trades over training and at least 10 by entry date in each consecutive training year. Quality gates: positive Net PnL after commission, PF > 1, Max DD <= 10% of equity. Engine `MaxDrawdown` is absolute RUB; compare `MaxDrawdownPct` to the 10% gate.
- Final holdout gate: at least 10 closed trades and the same quality gates. Count open positions separately; do not treat unrealized PnL as a closed trade.
- Budget: stage 1 all 350 grid combinations including both baseline and literal input values, plus a separate original-literal control; at most four eligible input candidates. Stage 2: two 24-combination exit grids per candidate. Stage 3: up to 54 combined refinement combinations and three one-field sensitivity grids; at most one local rerun if sensitivity warrants it. No search after opening the final holdout.

## Grids and sequence

1. `stage-1-entry.json`: research template plus POSI literal values if missing; fixed baseline exits. Include baseline as an explicit grid point. The supplied template expands to 350 combinations (7 × 5 × 2 × 5), despite the skill prose describing 120. Verify all results against the annual entry-date counts and neighboring grid values.
2. For each carried candidate, run independent `stage-2-*-<candidate>.json` grids copied from the two templates, explicitly fixing all nine fields. Compare stochastic-exit and no-stochastic-exit branches without opening holdout.
3. Build `stage-3-refinement.json` and individual sensitivity grids from actual selected/report values, then lock one complete candidate in `locked-candidate.md` before the holdout run.
4. Run locked candidate and original literal once each on the reserved final year from the same frozen candle snapshot, with warmup and clean positions. Install a candidate in `posi.Params()` only if the predeclared holdout gates pass.

## Snapshot note

After refresh, the POSI M30 snapshot spans 2023-11-12 16:00—2026-11-12 15:30 MSK (37,082 bars); daily candles span 2022-11-14—2026-11-11 (1,153 bars). The last M30 bar is completed before the run boundary; daily ATR/EMA have a 1-year lead-in. HTFTrendEMA is a daily EMA in executable core code; no H4 candles are required for this strategy's current parameter set.
