# Locked IRAO candidate before final-year check

Locked at 2026-11-12 16:22 MSK, before loading the reserved final-year window.

- Candidate file: `final-candidate.json` (all nine fields).
- Params: ATRPeriod=14, MinGapATR=0.25, MaxWaitBars=45, StochPeriod=14, UseStochExit=0, StopATR=1, TakeProfitATR=1.5, HTFTrendEMA=0, RequireEntryAboveFVGLower=1.
- Branch: no stochastic exit; StopATR and TakeProfitATR active.
- Stage 3 selection: best expectancy in the 54-combination joint rerun around sensitivity-selected MaxWaitBars=45. Fixed full run matched the joint-grid result.
- Training interval reported by CLI: 2023-11-12 16:21 — 2025-11-12 16:21 MSK (the report windows in this run series vary by several minutes because CLI computes `now` per invocation; all use the same completed M30 candle boundaries and no new historical candles crossed the slice boundaries).
- Training metrics: 24 closed trades, split by entry time 11 / 13 across the two consecutive 12-month blocks; Net PnL +43,236.77 after 0.0005 commission; PF 3.086; expectancy 1,801.53; Max DD 9.55%; no open position.
- Precommitted acceptance for final year remains >=10 closed trades, positive Net PnL after costs, PF > 1, Max DD <=10%. No further candidate substitution is allowed after the final-year check.

Branch comparison on admissible Stage 2 results: best no-stochastic result was rank-2 entry / combo 1, expectancy 1,581.38, 22 trades (10 / 12), PF 2.728, Net PnL +34,790.46, Max DD 9.54%. Best stochastic-exit result was rank-5 entry / combo 1, expectancy 697.59, 24 trades (13 / 11), PF 5.641, Net PnL +16,742.19, Max DD 3.58%. Expectancy difference exceeds the precommitted 5% closeness band, so the no-stochastic branch was selected. Stage 3 then improved expectancy with MaxWaitBars=45 and the joint rerun retained that exact combination.
