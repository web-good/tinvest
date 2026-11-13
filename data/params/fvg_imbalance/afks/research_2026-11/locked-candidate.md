# AFKS locked candidate before reserved final-year test

Locked at 2026-11-12 Europe/Moscow, before running any result on the reserved period.

- Training window: `[2023-11-12 15:47 MSK, 2025-11-12 15:47 MSK)` as shown by the completed final training run. Year blocks split at `2024-11-12 15:47 MSK`.
- Reserved test: last 12 months as computed by CLI at final-run time (expected `[2025-11-12, 2026-11-12]`); must verify report bounds and candle coverage.
- Branch: `UseStochExit=0`, selected because best eligible stage-2 no-stoch expectancy exceeded best eligible stoch by >5%; comparison was performed before opening the final year.
- Locked params: `ATRPeriod=14, MinGapATR=0.3, MaxWaitBars=30, StochPeriod=10, UseStochExit=0, StopATR=0.5, TakeProfitATR=1, HTFTrendEMA=0, RequireEntryAboveFVGLower=1`.
- Evidence: stage-3 bounded rerun rank 2 is the locked combination (rank 1 `MinGapATR=0.25` had 10.93% DD and fails the 10% cap). One-field sensitivity had the selected `StochPeriod=10` combination at expectancy 1541, total 22 trades, last-year 10. Fixed training run confirms year counts 12/10, net PnL +33,899.67 RUB, PF 3.138, Max DD 9.20%.
- Quality/eligibility limits remain those in `plan.md`: net PnL >0, PF >1, Max DD <=10%, >=20 total closed trades, >=10 in each training year; final test >=10 closed trades.
- Only this candidate and the original full set will be run on the reserved year. If it fails, leave the AFKS literal unchanged; do not try another candidate on this same year.
