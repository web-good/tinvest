# BSPB candidate locked before final holdout

Locked at 2026-11-11 14:43 MSK, before opening the reserved 12-month window. Full parameters are in `final-candidate.json`: ATRPeriod=14, MinGapATR=0.25, MaxWaitBars=30, StochPeriod=10, UseStochExit=1, StopATR=1.5, TakeProfitATR=1, HTFTrendEMA=0, RequireEntryAboveFVGLower=1.

Selection: Stage 3 rank 13 (rank 14 is the identical TakeProfit neighbor) had expectancy 620.1, Net PnL 20,461.86 RUB, PF 4.933, Max DD 3,616 RUB; 33 closed trades, 19 in the first learning block and 14 in the second. It satisfies >=10 trades in each block, positive net PnL, PF>1, and <=10% MaxDrawdownPct. The nearby MinGapATR/StopATR/RequireEntry variants form a positive stable area; TakeProfit 0 and 1 produced identical results because the stochastic exit closed trades first. The stoch branch beat no-stoch branch; no-stoch top results failed quality gates. Sensitivity runs retained StochPeriod=10, HTFTrendEMA=0, MaxWaitBars=30 as best by expectancy. Stage 3 full joint run is the evidence for the actual combined parameter set.

Do not change this candidate after viewing the holdout. Final run is the reserved [2025-11-11, 2026-11-11) period using the most recent completed cached candles, with a clean position and one year of daily warmup. Compare this candidate once with the original BSPB literal on the same cache snapshot.
