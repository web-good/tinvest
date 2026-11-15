# IVAT exploratory candidate locked before approximate validation (2026-11-14)

Candidate: `final-candidate.json`.

Selected branch: `UseStochExit=0`; expectancy 1,849.63 RUB, compared with 1,124.27 RUB for the best stochastic-exit combination, a difference greater than the precommitted 5% proximity threshold. Both had 21 closed training trades, positive net PnL, PF > 1, and Max DD <= 10%. The candidate center was the top result in the 54-combination stage-3 local grid. All nearest tested stop/target/gap neighbors ranked lower in expectancy; among combinations meeting >=20 closed trades and quality gates, only the center qualified. Sensitivities did not find a better eligible alternative: StochPeriod 11 had much lower expectancy and higher DD; EMA 50 and MaxWaitBars 45 had fewer than 20 closed trades.

Training report boundary: 2023-11-13 23:59–2025-11-13 23:59. Because the source begins 2024-06-04, this is only about 17 months of observations. Candidate had 11 closed entries in `[2024-06-04, 2025-06-04)` and 10 in `[2025-06-04, 2025-11-13]`; these blocks are unequal, so this does not meet the standard two-full-year training rule.

Stage-3 metrics: 21 trades, Net PnL +38,842.28 RUB after commission, PF 2.369, Max DD 11,048.40 RUB (7.84%), expectancy 1,849.63 RUB. Execution: cash RUB 100,000, fraction 1, risk-pct 0, commission 0.0005, next M30 open.

Approximate validation is locked to one run of this candidate and one baseline run, using the last available cached period. Precommitted descriptive pass gates: >=10 closed trades, positive Net PnL, PF > 1, Max DD <=10%. Cached M30 ends 2026-10-15, so the validation slice will be shorter than 12 months. No result from it can turn this exploratory candidate into a verified ticker literal; do not tune or replace the candidate based on that slice.
