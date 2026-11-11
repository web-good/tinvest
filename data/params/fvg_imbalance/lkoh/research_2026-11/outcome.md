# LKOH FVG calibration outcome (2026-11-11)

## Decision

The candidate failed the pre-registered quality gates on the reserved year: Net PnL was negative, PF was below 1, and Max DD exceeded 10%. It was subsequently applied to the LKOH strategy literal at the user's explicit request, with the failed validation recorded in the source comment. It must not be described as validated.

The only candidate tested in the reserved year was the locked candidate recorded in `locked-candidate.md`; no alternate candidate was substituted after seeing the result. Correction: the planned comparator run used `data/params/fvg_imbalance/lkoh/research_2026-11/original.json` (the LKOH research data baseline), not the Go ticker literal that existed before this user's update. That previous Go literal was ATRPeriod 14, MinGapATR 0.25, MaxWaitBars 30, StochPeriod 10, UseStochExit 1, StopATR 0.4, TakeProfitATR 1.5, HTFTrendEMA 0, RequireEntryAboveFVGLower 0. The reported +4,561.39 comparator result therefore must not be attributed to the previous Go literal.

## Parameters

Locked candidate: ATRPeriod 14, MinGapATR 0.25, MaxWaitBars 45, StochPeriod 9, UseStochExit 0, StopATR 0.5, TakeProfitATR 2, HTFTrendEMA 0, RequireEntryAboveFVGLower 1. Branch: no stochastic exit; stop and take-profit active.

Original literal: ATRPeriod 14, MinGapATR 0.25, MaxWaitBars 30, StochPeriod 11, UseStochExit 1, StopATR 0.4, TakeProfitATR 1.9, HTFTrendEMA 0, RequireEntryAboveFVGLower 0.

## Results

| Run | Window in report | Closed trades | Training annual trades / net PnL | Net PnL | PF | Max DD | Open at end |
|---|---|---:|---|---:|---:|---:|---|
| Candidate | 2025-11-11 12:54 — 2026-11-11 12:54 | 21 | training: 13 / 13; +3,445.80 / +31,259.31 | -2,088.20 | 0.921 | 20.09% | No |
| Research baseline comparator (not prior Go literal) | 2025-11-11 12:54 — 2026-11-11 12:54 | 23 | training: 14 / 18; -1,334.24 / +4,476.46 | +4,561.39 | 1.495 | 5.34% | No |

Training window was 2023-11-11 12:50–12:53 through 2025-11-11, depending on run start time. Candidate training result: 26 trades (13 in each fixed annual block), Net PnL +34,705.11, PF 2.961, expectancy 1,334.81, Max DD 7.11%. Both annual block PnLs were positive, +3,445.80 and +31,259.31.

## Reproduction artifacts

- Plan and frozen criteria: `plan.md`
- Stage 1 grid and full 420-combination report: `stage-1-entry.json`, `../../../../../reports/fvg_imbalance/lkoh/research_2026-11/stage-1/`
- Stage 2 branch grids and reports: `stage-2-candidate-*.json`, `../../../../../reports/fvg_imbalance/lkoh/research_2026-11/stage-2-candidate-*/`
- Stage 3 refinement, sensitivities, and bounded rerun: `stage-3-*.json`, `sensitivity-*.json`, `../../../../../reports/fvg_imbalance/lkoh/research_2026-11/stage-3-*/`, `../../../../../reports/fvg_imbalance/lkoh/research_2026-11/sensitivity-*/`
- Locked full parameter set: `final-candidate.json`
- Final-year candidate and original reports: `../../../../../reports/fvg_imbalance/lkoh/research_2026-11/final-candidate/`, `../../../../../reports/fvg_imbalance/lkoh/research_2026-11/final-original/`

All runs used the same cached LKOH completed M30/daily data, next-bar-open execution, 100,000 RUB initial cash, fraction 1, zero risk sizing, and 0.0005 commission per side. The daily indicator lead-in and strategy ATR warm-up were provided from the candle cache.
