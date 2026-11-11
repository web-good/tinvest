# Locked SNGS candidate before final-year inspection

Locked at 2026-11-11 after stage 3, before running any backtest in the reserved final window `[2025-11-11, 2026-11-11)`.

```json
{"ATRPeriod":14,"MinGapATR":0.1,"MaxWaitBars":30,"StochPeriod":21,"UseStochExit":1,"StopATR":1.7,"TakeProfitATR":0,"HTFTrendEMA":120,"RequireEntryAboveFVGLower":1}
```

Selection basis: stage 1 entry candidate #3 met the 20-total/10-per-year rule (11 trades in `[2023-11-11, 2024-11-11)`, 12 in `[2024-11-11, 2025-11-11)`). Stage 2 `UseStochExit=1` top result had 23 trades (11/12), positive net PnL and PF > 1; no-stoch branch had no positive eligible result. Stage 3 joint refinement reproduced the same complete combination as top rank (11/12 trades). Stage-3 metric expectancy, Net PnL +9455.99, PF 2.501, Max DD 3.32% against limit 10%. EMA sensitivity: EMA200 failed annual eligibility (8 recent-year trades); EMA120 selected. StochPeriod 14 and MaxWaitBars 15 alternatives ranked lower; MaxWaitBars45 produced one trade and is ineligible.

Final test protocol, unchanged from plan: run original and this single locked candidate once each on `-months 12 -holdout-months 0`, using the same costs and clean position. Do not change candidate based on that period.
