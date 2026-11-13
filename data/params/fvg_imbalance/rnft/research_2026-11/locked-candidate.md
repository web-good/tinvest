# RNFT locked candidate, before final-year inspection

Locked on 2026-11-12 after the 24-month training period 2023-11-12 22:32–2025-11-12 22:32. The reserved final window is 2025-11-12–2026-11-12 and has not been inspected at lock time.

Full candidate: `final-candidate.json` (same as `stage-3-selected.json`): ATRPeriod 14, MinGapATR 0.3, MaxWaitBars 30, StochPeriod 9, UseStochExit 1, StopATR 1.5, TakeProfitATR 1, HTFTrendEMA 0, RequireEntryAboveFVGLower 0.

Selected branch: stochastic exit enabled. The closest tied stage-3 profiles had identical reports across TakeProfitATR 0/1/1.5 at StopATR 1.5 and 1.7; chose finite 1 ATR target. Stage-2 no-stochastic leader failed the pre-set drawdown gate (24.52%). Stage-3 sensitivities did not improve the locked profile: StochPeriod 10 expectancy -46.22, 11 +38.71; HTFTrendEMA 50 had 16 total/5 recent trades; MaxWaitBars 15 expectancy 185 and 45 expectancy 64.47 versus locked 201.

Training report: `reports/fvg_imbalance/RNFT/2026-11/stage-3-finalist-training/RNFT_fvg_imbalance_Minutes30_20261112_223250.md`. Metrics: 37 closed trades, 21 in first training year and 16 in second; net +7,437.07 RUB, PF 1.509, Max DD 5.98%, no open position. Gates fixed before results: positive Net PnL, PF >1, Max DD <=10%, >=10 closed trades per training year; final >=10 and same quality limits.

Execution assumptions: M30 and completed Day1 candles, Tinkoff Invest CandleProvider, daily ATR Wilder(14) with 28 completed weekday candles needed, one-year daily EMA warmup, cash 100,000 RUB, fraction 1, commission 0.0005, risk-pct 0, API lot size, entry at next M30 open. No changes to trade mechanics.
