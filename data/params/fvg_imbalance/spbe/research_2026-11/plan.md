# SPBE FVG imbalance calibration plan (2026-11-12)

## Frozen criteria
- Requested ticker: SPBE; strategy: fvg_imbalance, completed M30 decisions, daily ATR and optional daily EMA as implemented in `core`.
- Required training window: `[2023-11-12, 2025-11-12)`; reserved final test: `[2025-11-12, 2026-11-12)`. Actual CLI bounds and last completed candle will be recorded for each run. The cached M30 currently spans 2023-09-04 through 2026-10-13; the daily cache spans 2022-10-28 through 2026-10-27. If refresh cannot supply the reserved test window, the candidate remains research-only and the literal will not be changed.
- Data source: Tinkoff Invest MarketData API through `cmd/backtest` CandleProvider, using its local `data/candles` cache. M30 data and daily data use the same share and API source.
- Warm-up: backtest requests daily candles from one year before the training start; FVG core uses 28 completed weekdays minimum for ATRPeriod=14 and daily EMA uses requested history. M30 warm-up is the engine's configured strategy lookback. Only completed candles are used.
- Execution/costs: cash 100000 RUB, fraction 1.0, commission 0.0005 of turnover per side, risk-pct 0, same engine fill rules for every run. No mechanics changes.
- Primary metric: expectancy. Frozen quality gates: Net PnL after commission > 0, PF > 1, Max DD <= 10% (engine percentage, not absolute currency).
- Every training-stage candidate/branch requires >=20 completed trades over the full 24 months and >=10 closed trades by entry date in each consecutive 12-month block. Do not infer the first block from total and RecentTrades. Final test requires >=10 completed trades and all quality gates.
- If no candidate clears all gates or the full independent final year is unavailable, preserve reports and retain the current SPBE literal.

## Search
- Stage 1: copied `stage-1-entry-template.json`, joint MinGapATR × RequireEntryAboveFVGLower × HTFTrendEMA × StochPeriod search (the copied template is actually 350 combos; preserve all, rather than trusting the inconsistent 120-combo description), baseline exits. Add original literal values if absent before first run. Assess original literal separately as control. Keep at most four candidates, only after annual trade-count checks, DD/quality checks, and neighborhood review.
- Stage 2: independent StopATR × TakeProfitATR grids for UseStochExit=1 and =0 per retained stage-1 input candidate, with all fixed fields explicitly set. No implicit carry-forward. KeepTop is not used to replace annual gates.
- Stage 3: local joint refinement per `stage-3-refinement.md`, sensitivity checks for StochPeriod, HTFTrendEMA and MaxWaitBars, and one bounded rerun only if sensitivity warrants it.
- Budget: complete all 120 stage-1 combinations and baseline control; no candidate may proceed without full-trade-table checks. Stage 2/3 only for candidates passing frozen gates. If available compute/time budget prevents complete checks, report search as incomplete and do not claim no qualifying set exists.
- Final-year window remains unopened for selection; after locking one candidate, run it and original once each with `-params` and identical execution settings.
