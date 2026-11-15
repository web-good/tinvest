# LSRG calibration outcome (2026-11-13)

## Status: candidate retained for research; ticker literal unchanged

A candidate passed the training rules and is locked in `locked-candidate.json`. It is not written to `Params()` because the reserved final 12-month validation could not be run: the M30 cache ends at 2026-10-13 13:00 MSK, before the planned 2026-11-13 end, and the API refresh failed repeatedly with `DeadlineExceeded` for M30 candle chunks (first logged chunk: 2025-11-13 to 2025-11-27). The refresh command was interrupted after repeated timeouts; no report was produced. No final-period result is claimed. Baseline final test was not run because the fixed market-data snapshot could not be obtained.

## Frozen training and candidate evidence

- Training window in reports: 2023-11-13 22:59/23:02 through 2025-11-13 22:59/23:02 MSK (small CLI start-time drift; all compared training grids had the same requested months and holdout and the same endpoint date).
- Original control: 34 closed trades (21 in training year 1, 13 in year 2), Net PnL -7,653.28 after commission, PF 0.705, max drawdown 9.91%. Trades are in `control-train/`.
- Locked candidate: `{ATRPeriod:14, MinGapATR:0.3, MaxWaitBars:30, StochPeriod:9, UseStochExit:1, StopATR:2, TakeProfitATR:1, HTFTrendEMA:0, RequireEntryAboveFVGLower:1}`.
- Candidate training: 21 closed trades, 11 entered in training year 1 and 10 entered in year 2; Net PnL +11,386.21, PF 2.348, expectancy 542.20, max DD 6.23%, no open position at report end. This meets training rules only; it does not establish OOS performance.
- Selection was fixed before attempting final validation in `locked-candidate.md`. No final-window result was used to select the candidate.
- Sensitivity grids for stochastic period and HTF EMA did not meet the 20-trade total floor. MaxWaitBars sensitivity did not improve on the locked 30-bar choice.

## Preserved reports

- Entry grid and control: `stage-1-entry.json`, `stage-1-entry/`, `control-train/`.
- Separate stage 2 candidate/branch grids and reports: `stage-2-candidate-*/` (the first one-value template probe is retained in each `report/`; the valid full exit grids and reports are in `corrected-report/`).
- Stage 3 joint grid and report: `stage-3-candidate-1/`; the valid grid holds StochPeriod=9 and evaluates 48 combinations. The initial 96-combination probe is retained under `stage-3-candidate-1/report/` and was superseded by `corrected-report/`.
- Stage 3 sensitivity grids and reports: `stage-3-sensitivity/`.
- Fixed final candidate: `locked-candidate.json` and `final-candidate.json`.
