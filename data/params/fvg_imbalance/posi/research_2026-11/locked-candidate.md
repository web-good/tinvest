# Locked POSI candidate

Locked before examining the reserved final-year results.

- Training range shown by the fixed candidate report: 2023-11-12 16:27—2025-11-12 16:27 MSK. It contains the same completed M30 candles as the stage reports (their CLI boundary times differ by minutes, but lie between the same candle timestamps).
- Candidate: `final-candidate.json`; `UseStochExit=1`.
- Stage 3 rerun: expectancy 1,050.59 RUB, 27 closed trades, 12 in the first training year and 15 in the second; Net PnL +28,365.88 RUB; PF 12.056; Max DD 3,184.20 RUB (2.78%); no open position at end.
- Reason locked: it ranked first in the one allowed joint rerun after MaxWait sensitivity showed a small eligible improvement; the final-year test will be used only for acceptance/rejection, with the original literal run on the same reserved window.
- Acceptance gates frozen in `plan.md`: at least 10 closed trades, positive after-cost Net PnL, PF > 1, Max DD <= 10%.
