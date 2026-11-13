# Locked candidate before final-year access

Candidate fixed from stage 2: `ATRPeriod=14, MinGapATR=0.1, MaxWaitBars=30, StochPeriod=21, UseStochExit=1, StopATR=1.7, TakeProfitATR=1, HTFTrendEMA=200, RequireEntryAboveFVGLower=0`. Stage 2 best admissible candidate was rank 1 in input candidate 2, 78 closed trades (38 in year 1, 40 in year 2), expectancy 180.3, Net PnL RUB 14,065.64, PF 1.904, Max DD 4.76%. Nearby stop values 1.2/2 and target values 0/1.5/2 also qualified, with expectancy still positive and PF >1; the selected point has the best expectancy among the broad stable cluster.

The no-stochastic branch was compared before final-year access. Its strongest eligible result was input candidate 3, `StopATR=1.5, TakeProfitATR=1`, expectancy 568.8 and PF 1.587, but Max DD was 12.28%, above the frozen 10% limit. Candidate 1 no-stochastic had no combination with 20 total trades. Thus the no-stochastic branch does not have an admissible choice under the frozen criteria; the stochastic branch is fixed for stage 3. The final year remains unseen.

Sensitivity (still training window only) found `MaxWaitBars=15` at expectancy 271.1, 40 trades (15/25 by year), PF 3.124, Net PnL RUB 10,842.69, Max DD absolute RUB 3,118.12; this passed the frozen floors. One allowed joint rerun was therefore fixed around this seed with nearby MinGapATR/StopATR/TakeProfitATR and RequireEntryAboveFVGLower [0,1]. No final-year candles were examined.

## Final candidate locked before opening validation year

The one allowed rerun ranked three exit variants with identical outcomes at `StopATR=1.5`; `TakeProfitATR=1` is retained as the finite target. Locked full parameters are in `final-candidate.json`: `ATRPeriod=14, MinGapATR=0.1, MaxWaitBars=15, StochPeriod=21, UseStochExit=1, StopATR=1.5, TakeProfitATR=1, HTFTrendEMA=200, RequireEntryAboveFVGLower=0`. Stage 3 rerun result: 40 trades (15 in year 1, 25 in year 2), expectancy 280.9, Net PnL RUB 11,236.74, PF 3.359, absolute Max DD RUB 2,771.50. This candidate passes all frozen training criteria and is fixed for exactly one final-year run against `original.json`. No tuning after opening that year.


## Post-validation audit note

After final-year access, the stage-2 table was rechecked and revealed that no-stochastic candidate 3 rank 9 also warranted an exact percentage-DD single-run check. That check was not performed before locking; it is not being run now because the reserved year has already been opened. The final-year run remains evidence for the locked candidate only, and the SBERP literal is left unchanged.
