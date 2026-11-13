# Locked ALRS candidate — before final-year run

Locked on 2026-11-12 at 13:39 MSK, after aligned training reports and before opening the reserved final year.

## Full parameters

See `final-candidate.json` (all nine numeric fields). Selected values:

| Parameter | Value |
|---|---:|
| ATRPeriod | 14 |
| MinGapATR | 0.25 |
| MaxWaitBars | 45 |
| StochPeriod | 14 |
| UseStochExit | 1 |
| StopATR | 2 |
| TakeProfitATR | 0 |
| HTFTrendEMA | 0 |
| RequireEntryAboveFVGLower | 1 |

## Selection basis

- Aligned 24-month training snapshot: reports run 2026-11-12 13:37–13:38 MSK, window approximately 2023-11-12 13:37 through 2025-11-12 13:37; all runs shared the same final completed M30 candle (2025-11-12 13:30 MSK). Entry-date trade blocks are 2023-11-12..2024-11-12 and 2024-11-12..2025-11-12.
- At the original entry candidate, best admissible `UseStochExit=1` exit was `StopATR=2, TakeProfitATR=0`: expectancy 623.8, 32 trades (22/10), Net PnL 19,960.26 RUB, PF 3.310, Max DD 5.02%. Best admissible `UseStochExit=0` exit was `StopATR=0.5, TakeProfitATR=1`: expectancy 376.6, 33 trades (23/10), Net PnL 12,429.34 RUB, PF 1.467, Max DD 6.85%. The primary-metric difference exceeded 5%, so the stochastic-exit branch was selected.
- In the first stage-3 refinement, the highest-ranked three combinations failed the second-year 10-trade gate. The next admissible base candidate (`MaxWaitBars=30`, `RequireEntryAboveFVGLower=0`) had expectancy 623.8, 32 trades (22/10), and remained profitable at adjacent TP values.
- Sensitivity found an admissible `MaxWaitBars=45` variant with expectancy 602.0 (within 5%), 38 trades (24/14), Net PnL 22,877.82 RUB, PF 3.394, Max DD 5.03%. This triggered the single permitted joint rerun around that variant.
- The 36-combination rerun's top admissible result is the locked candidate above: expectancy 627.2, 33 trades (20/13), Net PnL 20,699.07 RUB, PF 3.252, Max DD 4.86%, no open position. It passes the precommitted training gates: >=20 total trades, >=10 in each year, positive net PnL, PF >1, and Max DD <=10%.

At lock time, the final-year result had not been inspected. After locking, one candidate run and one original-parameter run were made as prescribed. Both reports cover the requested dates, but the source M30 cache has no bars from 2026-10-14 through 2026-10-26 after an API `DeadlineExceeded`; the runs are therefore invalid for final acceptance. The candidate was subsequently written to the ALRS literal at the user's request, with the validation limitation recorded beside it. See `outcome.md`.
