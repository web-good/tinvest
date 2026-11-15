# SVCB FVG imbalance calibration plan

## Status and available history

This is a research-only run because the available SVCB candle history is shorter than the protocol's required 36 months (24-month training plus the following independent 12-month test). After the CLI could not fetch its uncached Nov 3-13 tail (gRPC deadline exceeded), all runs are pinned to the cached window ending 2026-10-13 by `-months 25 -holdout-months 1`. At planning time the local cache has M30 candles from 2023-12-15 17:00 MSK through 2026-11-03 22:30 MSK (34,933 candles), and daily candles from 2023-12-15 through 2026-11-02 (866 candles). The intended protocol window would begin 2023-11-13, before the first available candle. Do not call any previously observed interval an untouched OOS period and the initial plan was not to update the ticker literal from these results; the user later explicitly requested recording the research candidate with its unverified status documented.

## Parameters and source

Research baseline (from the skill's `referens/baseline.json`): `ATRPeriod=14`, `MinGapATR=0`, `MaxWaitBars=30`, `StochPeriod=14`, `UseStochExit=1`, `StopATR=1`, `TakeProfitATR=2`, `HTFTrendEMA=0`, `RequireEntryAboveFVGLower=0`. Current SVCB literal delegates to `core.DefaultParams()`, whose `MinGapATR` is 0.25; preserve that full original set separately and run it as control. Do not modify strategy mechanics.

## Window and data rules

Potential exploratory training window is the exact 24-month intersection 2024-10-13 through 2026-10-13 using the common cached M30 and daily history. Final test is unavailable: the data does not contain a complete 12 months after this training window, and there is no earlier history to shift both windows back. Only completed candles count. The engine loads approximately one year of daily lead-in for EMA/ATR warmup; strategy ATR requires 28 completed weekday daily bars, HTF EMA is disabled in baseline, and M30 stochastic requires its configured lookback. Record actual report boundaries and candle counts for each run.

## Execution and cost assumptions

Use source Tinkoff Invest candles from the local provider cache, ending the CLI request 2026-10-13 to avoid its uncached tail, M30 plus daily candles, starting cash RUB 100,000, fraction 1.0, commission 0.0005 per turnover, fixed-fraction sizing (`risk-pct=0`), instrument lot from the API, next-M30-open entries, and the engine's standard stop/target execution. No artificial exits or data/mechanics changes. Max DD constraint is 10% of equity (`MaxDrawdownPct`; engine reports this as a fraction and Markdown renders percent).

## Research criteria and admission

Primary metric: expectancy. Candidate quality constraints: positive Net PnL after commission, PF > 1, Max DD <= 10%. Admission for any 24-month training candidate requires >=20 completed trades overall and >=10 closed trades by entry date in each consecutive 12-month block of that same run. Open positions do not count. `RecentTrades` is not a substitute unless its trailing-year boundaries match the planned second block. These criteria are fixed before results and will not be relaxed. Because the independent final year is unavailable, no candidate may be labeled verified or written to the SVCB literal.

## Grids and budget

Stage 1 uses the unmodified 350-combination `stage-1-entry.json` grid (7×5×5×2) plus an explicit run of the original full SVCB set as control. Stage-1 fixed exit settings use the research baseline. If research proceeds, later grids must explicitly carry all nine values and be saved separately by seed and exit branch; do not run those stages unless a candidate first passes both annual trade-count gates. Keep all combinations (`-combos 0`). Initial budget note was too restrictive for the user-requested staged calibration. Before launching stages 2 and 3, candidates #3, #4, and #6 passed the stage-1 annual trade gate and all quality constraints. The staged research budget was extended to stage 2 (3 seeds × both branches; 120 combinations), one stage-3 joint grid (48), and 9 one-field sensitivity combinations. This budget extension does not change the fixed metric, quality limits, or trade-count gates. A final holdout remains unavailable and no candidate is considered verified.
