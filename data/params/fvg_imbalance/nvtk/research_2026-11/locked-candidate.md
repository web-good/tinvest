# NVTK locked candidate before final-year check

Locked 2026-11-12 before reading any final-window result.

- Candidate: `final-candidate.json`; `UseStochExit=1` branch.
- Full parameters: ATRPeriod=14, MinGapATR=0.25, MaxWaitBars=30, StochPeriod=14, UseStochExit=1, StopATR=1, TakeProfitATR=2, HTFTrendEMA=0, RequireEntryAboveFVGLower=0.
- Training period: 2023-11-12 through 2025-11-12. Both year blocks pass: 10 and 15 closed trades. Net PnL +9,039.93 RUB, PF 1.737, Max DD 8.43%, expectancy 361.60 RUB.
- Stage 2: no-stochastic branch had no candidate meeting both yearly gates and quality constraints. In the stochastic branch the selected candidate has the strongest expectancy among admissible stage-2 candidates and is >5% from the next alternatives. Stage 3 confirms a stable local area: stop=1.2 gives expectancy 320.83 with the same 10/15 trades, PF 1.607, DD 9.18%; take profit=1.5 gives expectancy 293.27, PF 1.604, DD 8.42%; both pass.
- Candidate equals the existing NVTK/core defaults. This selection is an out-of-sample check of the current settings, not an asserted parameter improvement.
- Final check reserved: 2025-11-12 through 2026-11-12. Require a complete 12-month candle window, >=10 closed trades, positive net PnL, PF>1, Max DD<=10%. Only after this lock are final-year results examined.
