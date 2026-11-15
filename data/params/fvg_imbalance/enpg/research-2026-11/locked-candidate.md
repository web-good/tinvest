# Locked ENPG candidate before final-year inspection

Locked at 2026-11-13 23:23 Europe/Moscow, before running any report with data after the training cutoff.

- Training window: 2023-11-13 23:22 to 2025-11-13 23:22 MSK (24 months), split at 2024-11-13 23:22.
- Reserved final window: 2025-11-13 23:23 to 2026-11-13 23:23 MSK (the CLI's `-months 12 -holdout-months 0` window).
- Locked full parameter object: `stage-3-candidate.json`.
- Chosen exit branch: `UseStochExit=1`, stochastic exit enabled; StopATR=1 and TakeProfitATR=1, both measured in daily ATR.
- Stage-3 complete-grid row: expectancy 832.2, 28 completed trades, Net PnL +23,302.80 RUB, PF 4.698, max drawdown 5,587.40 RUB / 4.43%.
- Completed trades by entry date in the two training blocks: 13 / 15. Sum of reported trade PnL by entry date: +4,132.51 / +19,170.28 RUB.
- Quality gates fixed in the plan: positive Net PnL, PF > 1, MaxDrawdownPct <= 10%, at least 20 trades overall and 10 in each training year. Stage-3 candidate meets them.
- The top audited no-stochastic candidates for the two strongest entry variants failed the fixed 10% drawdown gate (30.29% and 30.66%). Stage-3 sensitivity alternatives were lower in expectancy or below the 20-trade floor: EMA 50/100 had 15/13 trades; MaxWait 45 had 17 trades; StochPeriod 11 scored 789.4 on the corresponding top complete run. No higher eligible alternative was selected.
- Selection is final for this calibration. Do not replace it based on the reserved-year results.
