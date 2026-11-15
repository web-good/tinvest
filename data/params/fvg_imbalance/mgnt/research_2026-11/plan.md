# MGNT FVG imbalance calibration plan

## Locked criteria (written before the first run)

- Snapshot date: 2026-11-13 (Europe/Moscow); strategy: M30 `fvg_imbalance`; ticker: MGNT.
- Train window: `[2023-11-13, 2025-11-13)` (24 calendar months). Final holdout: `[2025-11-13, 2026-11-13)` (12 months). CLI representation: `-months 36 -holdout-months 12`, evaluated at a single fixed `now` of 2026-11-13. If the provider cannot supply the full window, do not call shortened coverage equivalent; stop before viewing/selecting a candidate for the final year.
- Candle source: Tinkoff Invest API through `CandleProvider` with `data/candles` cache. At planning time the local M30 cache spans 2023-09-27 through 2026-10-13 and daily cache 2021-08-12 through 2026-10-26; API top-up availability and actual returned boundaries must be recorded before interpreting reports. Only completed M30 and daily bars count. Daily ATR uses 28 weekday bars for Wilder seed/smoothing (`2*ATRPeriod`, ATRPeriod=14); HTF EMA warmup is 250 calendar days per CLI. Record actual boundaries from downloaded candles/reports.
- Starting cash: RUB 100,000; fraction 1.0; risk-pct 0 (fixed fraction); commission 0.0005 per turnover; default instrument lot from API; next M30 open execution. Same settings for all runs.
- Main metric: expectancy (RUB per closed trade). Candidate quality: Net PnL after commission > 0, Profit Factor > 1, Max Drawdown <= 10% (use `MaxDrawdownPct`, a fraction of equity; absolute MaxDrawdown is RUB). These are run-specific research constraints.
- Candidate eligibility at every stage and both exit branches: >=20 closed trades in full 24-month train and >=10 entry-dated closed trades in each fixed train year `[2023-11-13, 2024-11-13)` and `[2024-11-13, 2025-11-13)`. Do not count open positions. Final year needs >=10 closed trades plus the same quality constraints.
- Stage 1: 120-combination entry grid from template; include ticker's original MinGapATR=0.25 explicitly; baseline controls and separate original full-parameter control. Keep up to four qualifying stable candidates.
- Stage 2: separate full StopATR x TakeProfitATR grids for UseStochExit=1 and 0 per eligible entry candidate, all other parameters fixed explicitly from that candidate. Preserve every combination (`-combos 0`).
- Stage 3: local joint refinement per the skill's `referens/stage-3-refinement.md`, plus separate one-field sensitivity grids for StochPeriod, HTFTrendEMA, MaxWaitBars; only one bounded repeat joint grid if sensitivity changes the winner.
- Ranking and selection: expectancy, with year-by-year closed-trade count, annual results, drawdown and neighboring grid stability. Treat branch metric difference <=5% of larger absolute value as close and refine both. Final candidate and branch must be recorded in `locked-candidate.md` before running any final-year comparison.
- Execution budget: full 120 stage-1 combos plus control; stage 2 at most four candidates x two 24-combo branches (192 combos); stage 3 one <=54-combo joint grid, up to three sensitivity grids, and one <=54-combo repeat only if warranted. If the budget or window prevents complete evaluation, state which grids/results were not exhausted and do not claim global absence/optimality.

## Parameter provenance

- `baseline.json` is copied from the skill's `referens/baseline.json`; it is the research baseline, not the current MGNT literal.
- `original.json` records the current executable MGNT parameters (`core.DefaultParams()`: MinGapATR=0.25; remaining fields as shown).
- Current executable ticker `Params()` delegates to `core.DefaultParams()`; no per-ticker calibration is present at plan time.
