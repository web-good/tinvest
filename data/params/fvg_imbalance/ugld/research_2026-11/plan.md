# UGLD FVG imbalance calibration plan (2026-11-12)

## Locked protocol
- Desired calibration window: 2023-11-12 through 2025-11-12 (24 months), split into [2023-11-12, 2024-11-12) and [2024-11-12, 2025-11-12). Final holdout: [2025-11-12, 2026-11-12).
- Before running grids, verify completed M30 coverage and daily coverage including warmup; if exact full windows cannot be loaded, stop and report insufficient data. Do not substitute partial windows or use the holdout to choose parameters.
- Source: local `data/candles/UGLD_Minutes30.json` and `UGLD_Day1.json` cache; attempt standard provider refresh only if credentials/config allow. Preserve actual candle timestamps in reports.
- Warmup: daily history loaded from one year before the backtest start by CLI; strategy daily ATR period 14 requires 28 completed weekday daily candles; daily EMA filter uses completed daily closes. M30 indicators use completed candles, and entries execute at next M30 open.
- Execution: cash RUB 100,000; fraction 1.0; risk-pct 0 (all available cash sizing); commission 0.0005 of turnover; same settings across runs. No mechanics changes.
- Primary metric: expectancy. Gates fixed before results: positive net PnL after commission, PF > 1, Max DD <= 10% of running peak equity. For every stage/exit branch require >=20 closed trades overall and >=10 closed trades by entry date in each consecutive training year. Final holdout requires >=10 closed trades plus the same quality gates. Open positions do not count as closed trades.
- Max DD is fraction in metrics (`MaxDrawdownPct`), displayed as percent; enforce <=0.10. `NetPnL` is final equity minus initial cash, includes mark-to-market of an open position; PF and closed trade counts use closed trades only.

## Search budget
- Stage 1: the copied 120-combination grid plus separate original control; inspect all ranked results with `-combos 0`, verify year counts from trade CSVs.
- Stage 2: at most four eligible entry candidates; each gets both supplied exit grids, one independent 24-month run per branch.
- Stage 3: one local joint grid (<=54 combinations) on selected branch, plus one-field sensitivity checks for stochastic period, EMA, and MaxWaitBars; at most one bounded rerun if warranted.
- Final: exactly one locked candidate and the original control on the reserved year, once each. Stop without calibration if coverage cannot supply the locked windows. Never claim a full-grid negative finding if budget/search was incomplete.

## Baseline
- Research base: skill `referens/baseline.json` copied here. This does not modify ticker Params or core defaults.
- Current UGLD ticker Params copied to `original.json`: ATRPeriod=14, MinGapATR=0.25, MaxWaitBars=30, StochPeriod=14, UseStochExit=1, StopATR=1, TakeProfitATR=2, HTFTrendEMA=0, RequireEntryAboveFVGLower=0.

## Execution note
The copied reference template actually contains 7 × 5 × 5 × 2 = 350 combinations, despite the skill text saying 120. The full 350-combination grid was run with `-combos 0`; this is recorded as a protocol/budget discrepancy, and no additional stage-1 search was run after seeing results.
