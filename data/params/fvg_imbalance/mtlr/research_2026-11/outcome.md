# MTLR FVG imbalance calibration outcome

The candidate was locked before the reserved holdout was opened. The reserved interval in both final report metadata is 2025-11-12 22:33 — 2026-11-12 22:33 MSK, matching the planned final year. No additional calibration was run after viewing it.

## Final comparison

| Run | Closed trades | Net PnL after commission | PF | Max DD | Open position |
|---|---:|---:|---:|---:|---|
| Candidate (`final-candidate.json`) | 16 | -1,259.07 RUB (-1.26%) | 0.869 | 7,716.85 RUB (7.40%) | no |
| Original (`original.json`) | 17 | -1,928.52 RUB (-1.93%) | 0.800 | 5,427.78 RUB (5.33%) | no |

The candidate clears the final minimum of 10 closed trades and is less negative with a higher PF, but fails the frozen quality gates of positive net PnL and PF >1. Its drawdown is also larger than the original. Following the user's subsequent instruction, these best in-sample values were added to the MTLR literal despite the failed holdout; this is a deliberate promotion of an unvalidated research candidate, not a passed out-of-sample calibration.

Candidate parameters: ATRPeriod=14, MinGapATR=0.25, MaxWaitBars=45, StochPeriod=10, UseStochExit=1, StopATR=1, TakeProfitATR=1.5, HTFTrendEMA=0, RequireEntryAboveFVGLower=1.
