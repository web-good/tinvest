# Locked candidate before final-year inspection

Locked 2026-11-11 after stages 1–3, before running any report for the reserved final year.

- Candidate: `final-candidate.json` (all nine fields).
- Branch: `UseStochExit=0` (stochastic exit disabled; stop and take-profit remain active).
- Selection: stage-2 candidate 8 no-stochastic branch led the eligible stage-2 results (expectancy 1242 vs 400 in the comparable stochastic branch; >5% difference). Stage 3 MaxWait sensitivity found a better eligible 45-bar variant, so one bounded joint rerun was performed. The exact 45-bar result remained best in that rerun.
- Training: `[2023-11-11, 2025-11-11)`, 26 closed trades; first year 13 trades / +3,445.80 PnL, second year 13 / +31,259.31. Overall Net PnL +34,705.11, PF 2.961, expectancy 1,334.81, Max DD 7.11%.
- Constraints fixed in the plan: expectancy ranking, positive Net PnL, PF > 1, Max DD <= 10%; at least 20 training trades and at least 10 in each annual block.
- Final-year window reserved: `[2025-11-11, 2026-11-11)`; cached M30 candles currently end 2026-11-10 22:30 MSK. Compare only this candidate and the original literal. Do not replace the candidate based on the outcome.
