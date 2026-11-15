# MGNT FVG imbalance calibration outcome — 2026-11-13

## Decision

The candidate was not written to the ticker literal. The reserved holdout failed the precommitted MaxDrawdownPct <= 10% rule: the candidate's report shows 11.30%. It did meet the holdout closed-trade minimum and positive Net PnL / PF rules, but all criteria were required. Do not substitute another candidate using this already viewed final year.

## Locked candidate and training

`final-candidate.json`, locked in `locked-candidate.md` before the final-year run:

- ATRPeriod=14, MinGapATR=0.25, MaxWaitBars=45, StochPeriod=9, UseStochExit=0, StopATR=0.5, TakeProfitATR=1, HTFTrendEMA=0, RequireEntryAboveFVGLower=1.
- Training `[2023-11-13 23:21, 2025-11-13 23:21)`: 32 closed trades; 21 in year one (Net PnL +2,969.59 RUB), 11 in year two (+16,560.58 RUB); Net PnL +19,530.17 RUB, PF 1.828, Max DD 9.19%, expectancy 610.32 RUB. No open position is reported in this training run.

## Final holdout comparison

- Candidate report window `[2025-11-13 23:22, 2026-11-13 23:22]`: 20 closed trades, Net PnL +8,397.79 RUB after commission, PF 1.308, Max DD 12,069.93 RUB / 11.30%, expectancy 313.85 RUB. One position remained open and was marked to market at report end; it is not included among the 20 closed trades. Result fails the 10% drawdown cap.
- Original report window `[2025-11-13 23:23, 2026-11-13 23:23]`: 20 closed trades, Net PnL -2,877.82 RUB, PF 0.812, Max DD 11,064.71 RUB / 10.76%, expectancy -143.89 RUB. No open-position warning appears.
- The runs share the same calendar holdout dates and cached M30 snapshot; report endpoints differ by one minute due the CLI calling `time.Now()` for each run. Candle source cache after fetch reached MGNT M30 `2026-11-13T22:30:00+03:00` and daily `2026-11-12T03:00:00+03:00`. Both are after their last relevant entry/exit events; minute-level CLI endpoints are recorded rather than described as identical timestamps.
- Starting cash RUB 100,000; fraction 1.0; commission 0.0005; risk-pct 0; API lot; next M30 open execution. Daily ATRPeriod=14 with 28 completed weekday bars for Wilder seed/smoothing; CLI requests 250 calendar days lead-in for daily EMA. Selected candidate has HTFTrendEMA=0.

## Artifacts

All grids and reports are retained under `data/params/fvg_imbalance/mgnt/research_2026-11/` and `reports/MGNT_fvg_imbalance/research_2026-11/`. The executable MGNT `Params()` still delegates to `core.DefaultParams()` because the locked candidate failed final validation.
