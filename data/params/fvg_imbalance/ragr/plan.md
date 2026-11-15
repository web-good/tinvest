# RAGR FVG imbalance calibration plan

- Snapshot/start time: 2026-11-13 20:10 UTC (CLI uses `time.Now()` independently; capture actual report boundaries and keep one cached candle snapshot).
- Training: nominal `[2023-11-13 20:10 UTC, 2025-11-13 20:10 UTC)` (36-month CLI window with 12-month holdout); inspect sequential years `[2023-11-13, 2024-11-13)` and `[2024-11-13, 2025-11-13)` from the same continuous run.
- Reserved final test: nominal `[2025-11-13 20:10 UTC, 2026-11-13 20:10 UTC)`. Do not inspect until a single candidate is recorded below.
- Source: Tinkoff Invest candle API via `cmd/backtest`, using the repository candle cache. An initial forced refresh was attempted; request timeouts caused it to finish after the training and final runs, replacing the cache with incomplete historical data. Stage reports retain their measured results from the pre-refresh cache; see `outcome.md` for this deviation. Require completed M30 and daily candles. Core daily ATR uses 28 completed weekday daily bars as warmup; EMA filters use their configured daily period. M30 lookback follows `max(3+MaxWaitBars, StochPeriod+1)`. Engine additionally loads one year of daily lead-in.
- Execution/costs: mock execution, entry at next M30 open, gap-aware stop, exact target touch, close-based stochastic exit; initial cash RUB 100,000, fraction 1.0, commission 0.0005 per turnover, risk-pct 0, ticker lot from instrument metadata. No mechanic changes.
- Quality rules fixed before results: rank on expectancy; candidate must have positive net PnL after fees, PF > 1, Max DD <= 10% (`MaxDrawdownPct`, fraction of running equity peak), >=20 closed trades total and >=10 by entry date in each fixed training year. Final year requires >=10 closed trades and the same quality limits.
- Stage 1: copied entry grid (the supplied lists produce 350 combinations: 7 MinGapATR x 5 StochPeriod x 5 HTFTrendEMA x 2 entry-filter values); add RAGR literal values before viewing outcomes; run full baseline control and original-literal control separately. Save every combination (`-combos 0`).
- Stage 2: up to 4 eligible entry seeds; separate joint StopATR x TakeProfitATR grids for each stochastic exit branch; require the same training and annual trade gates.
- Stage 3: local joint refinement per `referens/stage-3-refinement.md` plus separate sensitivity grids; one bounded joint repeat only if sensitivity yields a better eligible candidate.
- Budget: up to 350 stage-1 combos + 2 controls; up to 4 seeds x 24 combos x 2 branches = 192 stage-2 combos; stage 3 capped at 54 joint combos + 3 one-field sensitivity runs + one 54-combo repeat. Stop early if no eligible candidates or required source history is unavailable. Budget limits will be reported as search incompleteness, not proof that no grid candidate exists.

## Candidate freeze before final test

Not selected yet. Record one full nine-field set, branch, training metrics, annual closed trade counts, quality checks, and reason before running the reserved year.

## Candidate frozen before reserved-window access

- Freeze time: 2026-11-13 23:15 MSK.
- Candidate: `research-candidate.json` (ATRPeriod=14, MinGapATR=0.3, MaxWaitBars=45, StochPeriod=14, UseStochExit=0, StopATR=1, TakeProfitATR=1, HTFTrendEMA=0, RequireEntryAboveFVGLower=0).
- Branch: no stochastic exit. The best eligible stage-2 stochastic result had expectancy 143.1; best eligible no-stochastic result had 621.7 (>5% separation), so stage 3 refined only no-stochastic branch.
- Final stage-3 repeat candidate training: expectancy 1,396.99; positive Net PnL +26,645.23; PF 1.905; Max DD 8.08%; 20 closed trades, 10 in each fixed year. One position remained open and is mark-to-market only; it is not counted in the closed-trade gate.
- Stage-3 joint repeat rank 1; fixed-parameter report confirms all quality thresholds. No further candidate changes after opening the reserved period.

## Outcome

- Final-year test was run once for the frozen candidate and once for the original RAGR literal. The final comparison has imperfect candle alignment: report ends match at the same completed bar, but first available equity candles differ by 10 hours. The delayed forced refresh also replaced the candle cache with a truncated historical window after these reports were written. The candidate failed predeclared final quality rules (PF <= 1, Net PnL <= 0, Max DD > 10%); the candidate did not qualify for automatic calibration writeback.
- Candidate report boundaries: 2025-11-13 23:15 to 2026-11-13 23:15 MSK. Original report boundaries: 2025-11-13 23:16 to 2026-11-13 23:16 MSK. Both equity series end on the same completed M30 bar (2026-11-13 19:30 UTC); first equity timestamps differ (2025-11-15 05:00 UTC candidate, 2025-11-14 19:00 UTC original), so compare as near-identical calendar windows, not perfectly identical candle spans.
- Final reports, parameter files, and all correct stage grids are retained separately under `reports/fvg_imbalance/ragr/` and `data/params/fvg_imbalance/ragr/`.


## User-directed writeback

After the candidate failed final quality checks, the user explicitly requested installing the best training candidate. `ragr.Params()` now returns that set, and the source comment states the failed final metrics. This is a user-directed override of the skill's automatic writeback gate; the calibration result itself remains failed OOS.
