# YDEX locked FVG imbalance candidate

Locked before opening the reserved final window, 2026-11-12 13:00 Europe/Moscow.

- Training window reported by CLI: 2023-11-12 12:59 — 2025-11-12 12:59 MSK. The exact 24-month trade ledger has 25 closed trades: 10 entered before 2024-11-12 and 15 entered from 2024-11-12 onward. Both annual blocks meet the 10-trade rule.
- Candidate: `final-candidate.json`; selected from stage 3 rank 1 on primary expectancy. Branch `UseStochExit=1`.
- Full fields: ATRPeriod=14, MinGapATR=0.3, MaxWaitBars=30, StochPeriod=14, UseStochExit=1, StopATR=1.2, TakeProfitATR=1.5, HTFTrendEMA=0, RequireEntryAboveFVGLower=1.
- Stage 3 training metrics: expectancy 711.77 RUB/trade, Net PnL +17,794.31 RUB after commission, PF 3.264, Max DD 6,467.09 RUB / 6.08%, 25 closed trades, no open position. Quality gates pass: Net>0, PF>1, MaxDrawdownPct<=10%.
- Baseline research JSON (`baseline.json`) performed poorly and is not the original ticker literal. The actual original `core.DefaultParams()` literal is saved in `original.json`; its training control had 37 closed trades (18/19 by the same annual split), Net PnL +18,818.01, PF 2.409, Max DD 6,371.46 / 5.87%.
- Both exit branches were evaluated over the copied grids. The stoch branch had a training candidate that passed the sample and quality checks. Every no-stoch combination in the four tested seed grids had absolute Max DD >=23,952.91 RUB; the calibration summary reports absolute drawdown, so the exact `MaxDrawdownPct` for those grid rows was not tabulated separately. No no-stoch set was advanced to stage 3.
- Sensitivity: StochPeriod 11 reduced expectancy; HTFTrendEMA 200 reduced expectancy; MaxWaitBars 45 raised Net PnL and PF but reduced expectancy to 690.4. The declared primary metric therefore retains MaxWaitBars=30. Joint refinement was executed with all nine fields explicit.
- Final validation used 2025-11-12 through 2026-11-12. At lock time cached M30 coverage ended 2026-10-15; normal provider tail top-up completed the data before the one original and one candidate `-months 12 -holdout-months 0` run. The candidate failed the final minimum trade count; the literal remains unchanged. No replacement candidate was selected after opening the period.
