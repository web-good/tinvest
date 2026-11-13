# Locked CHMF candidate before final-year access

Locked 2026-11-12 15:44 MSK, before running or inspecting the reserved period `[2025-11-12, 2026-11-12)`. Candidate: `locked-candidate.json`.

Parameters: ATRPeriod=14, MinGapATR=0.3, MaxWaitBars=30, StochPeriod=10, UseStochExit=0, StopATR=0.5, TakeProfitATR=1, HTFTrendEMA=0, RequireEntryAboveFVGLower=0.

Selection basis: Stage 1 eligible input candidate #4, followed by both Stage 2 exit branches across all four eligible inputs. The highest Stage 2 no-stochastic result meeting both year-by-year trade floors and quality limits came from input candidate #4, StopATR=0.5, TakeProfitATR=1. Stage 3 joint refinement's higher-ranked parameter sets had only 9 entries in the first year; rank #4 reproduced the selected center, and the MaxWaitBars=45 sensitivity was identical. StochPeriod=9 sensitivity was lower on expectancy; HTFTrendEMA=50 had 11 trades total and 5 in the second year, so failed the trading-frequency rule. No sensitivity result replaced the locked center.

The best qualifying stochastic-exit comparison was MinGapATR=0.3, StochPeriod=9, StopATR=1, TakeProfitATR=2, UseStochExit=1. Both branches were evaluated on the same training candles with the same commission, cash, sizing, and rules. The selected no-stochastic candidate has the higher training expectancy. Stage 3 selection uses training only; the locked candidate is not yet established as tested or deployable.
