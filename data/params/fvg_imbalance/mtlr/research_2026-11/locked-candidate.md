# MTLR candidate locked before final holdout

- Locked: 2026-11-12 Europe/Moscow, after the one bounded stage-3 joint rerun and before opening the reserved final year.
- Candidate: `final-candidate.json`.
- Training window from the final calibration metadata: 2023-11-12 22:33 — 2025-11-12 22:33 MSK; training split 2024-11-12.
- Selected branch: stochastic exit enabled (`UseStochExit=1`). Stage 2's no-stochastic branch failed the predeclared Max DD <=10% constraint in its best ranked candidate families. Stochastic candidates passed; candidate 1 had the strongest expectancy and lower drawdown.
- Stage-3 sensitivity found `MaxWaitBars=45` admissible (14/19 yearly deals) with improved expectancy. The mandatory single bounded joint rerun returned the same joint candidate, with 33 closed trades (14/19), expectancy 735.2, Net PnL 24,261.14 RUB, PF 3.756, Max DD 4,916.12 RUB / 4.61%, no open position.
- Quality gates frozen in `plan.md`: >=20 overall, >=10 each training year, positive net PnL, PF >1, percentage Max DD <=10%; final window needs >=10 deals and same quality limits.
- The final holdout has not been inspected as of this lock.
