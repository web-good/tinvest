# TATN calibration outcome — 2026-11-12

No ticker literal update. The selected candidate passed training but its planned final-year run was invoked with a different parameter file than the locked candidate. Since the reserved year has now been opened, it cannot be reused to validate the locked candidate; treat the candidate as unverified and retain the existing TATN literal (`core.DefaultParams()`).

## Locked candidate (training validated, final check unavailable)

`final-candidate.json`: `ATRPeriod=14, MinGapATR=0.3, MaxWaitBars=30, StochPeriod=11, UseStochExit=0, StopATR=0.5, TakeProfitATR=1, HTFTrendEMA=0, RequireEntryAboveFVGLower=0`.

Selected branch: no stochastic exit. Training `[2023-11-12,2025-11-12)` produced 28 closed trades (17 in year 1, 11 in year 2), Net PnL +28,745.38 RUB, PF 2.553, max DD 5.64%, no open position. It passed the frozen training quality limits (expectancy primary metric, positive Net PnL, PF > 1, max DD <= 10%).

## Holdout handling

The intended locked-candidate run on `[2025-11-12,2026-11-12)` mistakenly passed `candidate-1.json`, which contains `UseStochExit=1, StopATR=1, TakeProfitATR=2`. That is not the locked candidate. Its report shows 18 trades, Net PnL -3,663.11 RUB, PF 0.781, max DD 14.97%, no open position. This result must not be attributed to `final-candidate.json`.

The baseline was run on the same reserved period and did use all nine baseline values: 221 trades, Net PnL +33,358.29 RUB, PF 1.284, max DD 24.45%, no open position. This comparison cannot validate or reject the locked candidate because its holdout run was misconfigured. The holdout is consumed; no alternate candidate was tested on it.

The literal remains unchanged because there is no valid final-year result for the selected candidate. No strategy mechanics or other ticker files were changed for this calibration.
