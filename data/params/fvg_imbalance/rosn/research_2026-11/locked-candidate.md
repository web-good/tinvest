# ROSN FVG imbalance candidate locked before final holdout

Training window: 2023-11-12 through 2025-11-12 MSK. The final stage-3 report window was 2023-11-12 13:12 through 2025-11-12 13:12; earlier stage reports vary by a few minutes because CLI derives `now` at invocation. Their completed M30 bar boundaries and candle cache were the same. Reserved holdout: 2025-11-12 through 2026-11-12; not inspected before this lock.

Locked full candidate: `final-candidate.json` — ATRPeriod=14, MinGapATR=0.1, MaxWaitBars=30, StochPeriod=10, UseStochExit=1, StopATR=1, TakeProfitATR=0, HTFTrendEMA=50, RequireEntryAboveFVGLower=1.

Reason: stage 2's best stochastic-exit candidate ranked above all other entry candidates and its no-stochastic counterpart. The latter failed training quality (PF 1.019, Net +526.26 RUB, Max DD 10.07%); the stochastic candidate passed both annual gates (21 / 19 closed trades) and quality (expectancy 445.92 RUB, Net +17,836.71 RUB, PF 4.565, Max DD 3.06%). No open position remained. Stage 3 reran this complete point and kept it at rank 1 (40 trades, expectancy 445.9 RUB); close StopATR=1.2 neighbor was rank 2 at 437.8. Single-field sensitivities did not produce a stronger point: StochPeriod best 418.6, HTFTrendEMA best 296.1, MaxWaitBars best remained 445.9 at 30.

Final gate was fixed before holdout: >=10 closed trades, Net PnL > 0 after commission, PF > 1, Max DD <= 10% of equity peak. If candidate fails, retain the existing ROSN literal unchanged.
