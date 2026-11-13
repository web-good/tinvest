# PHOR FVG M30 calibration, 2026-11-12

- Selection [2023-11-12, 2025-11-12), with annual blocks split 2024-11-12. Reserve [2025-11-12, 2026-11-12) for one final test after candidate lock. CLI uses current wall time; verify actual candle boundaries and equality of data snapshots.
- Data: Tinkoff Invest API and local candle cache; completed M30 and daily candles. Record actual coverage and counts. Daily feed includes one year of prehistory for EMA, ATR(14) uses 28 daily bars; M30 lookback depends on stochastic and waiting. Check source coverage and do not assert independent validation if incomplete.
- Execution: initial capital 100000 RUB, fraction 1, risk-pct 0, commission 0.0005 per turnover, instrument lot from API, next-bar execution. Same for all runs.
- Research baseline and original complete literal saved separately. Stage 1 grid has 350 combinations, includes original entry values. All nine fields explicit. Save all combinations.
- Metric expectancy; require net PnL > 0, PF > 1 and MaxDrawdownPct <= 0.10 (engine stores fractional percent). Every stage requires >=20 closed trades total and >=10 by closed-trade entry date in each annual block. Final year >=10 closed trades and same quality gates. No open trade counted.
- Stage 2: at most four eligible diverse entry candidates; joint stop/take grids on both stochastic and no-stochastic exit branches, separately. Stage 3: local joint refinement and sensitivity per skill template; select one candidate before final year.
- Budget: control and stage 1; up to eight stage 2 grids; stage 3 up to 54 combinations, three sensitivity grids and one bounded rerun; two final single runs. Save grids and reports; do not change criteria or mechanics after results.
