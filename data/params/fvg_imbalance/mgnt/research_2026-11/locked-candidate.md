# MGNT locked candidate — 2026-11-13

Locked before any final-year report is opened.

- Candidate: `final-candidate.json`.
- Parameters: ATRPeriod=14, MinGapATR=0.25, MaxWaitBars=45, StochPeriod=9, UseStochExit=0, StopATR=0.5, TakeProfitATR=1, HTFTrendEMA=0, RequireEntryAboveFVGLower=1.
- Branch: stochastic exit disabled; stop and take-profit remain active.
- Selection basis: expectancy, run-specific filters Net PnL > 0, PF > 1, MaxDrawdownPct <= 10%; per-stage eligibility requires >=20 total closed trades and >=10 in each train year.
- Training window: `[2023-11-13, 2025-11-13)` using the 2026-11-13 run snapshot; holdout reserved: `[2025-11-13, 2026-11-13)`.
- Stage 1 entry rank 6 (year counts 20/10), stable qualified entry neighbors included rank 3 (28/18), rank 10 (47/37), rank 19 (26/14).
- Stage 2 rank 6 no-stoch best qualified result: expectancy 588.6, 30 closed trades (20/10); the stoch branch had expectancy 279.7 (21/10). Entry rank 3 branch results were 318.0 (28/18) with stochastic exit and 244.3 (23/17) without; rank 10 209.6 (47/37) vs 165.2 (43/31); rank 19 183.8 (26/15) vs 208.8 (24/15). These branch comparisons used matching training windows and costs.
- Stage 3 local refinement rank 4, MinGapATR=0.25/StopATR=0.5/TP=1/MaxWaitBars=30: expectancy 588.6, 30 trades (20/10). Adjacent eligible stop values 1 and 1.5 produced expectancy 467.8 and 354.2. Sensitivity showed MaxWaitBars=45 improved expectancy to 610.3 with 32 trades (21/11); MaxWaitBars=15 was rejected (7 second-year trades). Separate StochPeriod and HTFTrendEMA sensitivities did not produce a stronger eligible candidate.
- One bounded repeat joint grid at MaxWaitBars=45 selected this exact full parameter set, rank 3: expectancy 610.3, Net PnL 19,530.17 RUB, PF 1.828, absolute Max DD 9,713.30 RUB, 32 closed trades. Its most relevant neighboring MinGapATR=0.3 result had the same expectancy and trades. Year counts are 21/11. Percentage drawdown and all trade-table-derived figures will be checked in the locked-parameter training report.
- Run conditions: RUB 100,000, fraction 1.0, commission 0.0005, risk-pct 0, API lot, next M30 open execution. M30 and daily candle snapshot boundaries and warmup must be confirmed from final reports before interpreting the reserved holdout.

## Final result (opened after lock)

- Candidate final-year report: `[2025-11-13 23:22, 2026-11-13 23:22]`, 20 closed trades, +8,397.79 RUB Net PnL, PF 1.308, Max DD 11.30%, expectancy 313.85 RUB, one open position marked to market. It fails the precommitted Max DD <=10% rule, so this candidate is retained as research only and is not eligible for the MGNT ticker literal.
- Original full set: 20 closed trades, -2,877.82 RUB Net PnL, PF 0.812, Max DD 10.76%, no open-position warning. Its CLI window ended one minute later; cached M30 and daily bars were the same snapshot.
- Full comparison and candle coverage notes: `outcome.md`.
