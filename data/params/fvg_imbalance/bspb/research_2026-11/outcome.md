# BSPB FVG imbalance calibration outcome

## Data and execution

Run date: 2026-11-11 MSK. Cached source: `data/candles/BSPB_Minutes30.json` (2023-08-24 12:30 through 2026-11-10 23:30 MSK) and `data/candles/BSPB_Day1.json` (2021-08-12 through 2026-11-09). Completed bars only. CLI reports learning window 2023-11-11 14:40–2025-11-11 14:40 and holdout 2025-11-11 14:43–2026-11-11 14:43; those bounds use latest completed cached bars and the requested calendar windows. Daily lead-in is one year; it exceeds the 28 daily candles required for ATRPeriod=14 and warms EMA up to 200 days. Execution was identical throughout: 100,000 RUB, fraction 1, risk-pct 0, commission 0.0005, next M30 open and normal lot sizing. No strategy mechanics changed.

## Results and decision

Locked parameters: `final-candidate.json` — ATRPeriod=14, MinGapATR=0.25, MaxWaitBars=30, StochPeriod=10, UseStochExit=1, StopATR=1.5, TakeProfitATR=1, HTFTrendEMA=0, RequireEntryAboveFVGLower=1. Selected branch: stochastic exit. Stage 3 full joint run at rank 13/14 showed 33 closed trades (19 in 2023-11-11–2024-11-11; 14 in 2024-11-11–2025-11-11), expectancy 620.1 RUB, Net PnL +20,461.86 RUB, PF 4.933, max drawdown 3,616 RUB (3.52%). Nearby joint variants ranked 13–18 remained positive and passed both annual trade floors. The separate sensitivities favored StochPeriod=10, HTFTrendEMA=0, MaxWaitBars=30.

The best no-stoch results did not pass the preset quality gates (rank-5 and -6 input candidates had PF below 1 and negative Net PnL); the stochastic branch was retained. A fixed-parameter learning-period report was also generated after locking to confirm full-candidate statistics; it agrees with the Stage 3 result (33 trades, 19/14 annual split, 3.52% drawdown). The locked candidate was evaluated once on the reserved final year: 15 closed trades, no open position, Net PnL +715.64 RUB, PF 1.155, max drawdown 3,042.32 RUB (2.97%). It passes the >=10 closed trade and quality requirements. The original literal on the same cached period had 12 closed trades, no open position, Net PnL -7,056.88 RUB, PF 0.482, max drawdown 12,497.08 RUB (11.85%). Candidate has a modest positive holdout result; training expectancy was much stronger.

The original full-literal control on the learning window had 21 trades, Net PnL -33,085.32 RUB, PF 0.343, max drawdown 37.87%. The independent control uses the original literal values (MaxWaitBars=35, UseStochExit=0, StopATR=0.5, etc.).

## Artifacts

- Plan and locked criteria: `plan.md`; candidate lock: `locked-candidate.md`.
- Grids: `stage-1-entry.json`, `stage-2-rank{5,6,7,8}-{stoch,no-stoch}.json`, `stage-3-refinement.json`, and three `sensitivity-*.json` files.
- Learning reports: `reports/fvg_imbalance/bspb/research_2026-11/` (`control`, `stage-1`, `stage-2/rank*`, `stage-3`, `sensitivity-*`). Each calibration report preserves every combination (`-combos 0`) and its best-run report/ledger where emitted.
- Fixed-candidate learning report: `reports/fvg_imbalance/bspb/research_2026-11/locked-candidate-learning/`.
- Holdout reports: `reports/fvg_imbalance/bspb/research_2026-11/final-candidate/` and `final-original/`.

The CLI's grid parser rejects string `_comment` values, so annotations live in this outcome and plan rather than inside executable JSON grids.
