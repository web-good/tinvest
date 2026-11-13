# ROSN FVG imbalance calibration plan (2026-11)

## Locked protocol
- Run date: 2026-11-12 (CLI uses its wall clock at each run; record actual candle boundaries from reports; same local candle cache/source throughout).
- Training window: 24 months, intended `[2023-11-12, 2025-11-12)`; evaluate its consecutive year blocks separately. The following 12 months, intended `[2025-11-12, 2026-11-12)`, are reserved and must not be opened until one full candidate is locked after stage 3.
- Source: Tinkoff Invest API via `cmd/backtest`, cache `data/candles`; completed Minutes30 and Day1 candles. Indicators are warmed from the engine's extra one-year daily lead-in; ATRPeriod=14 needs 28 completed daily bars and EMA uses its configured period. Record actual counts/boundaries from run reports/output.
- Execution: initial cash RUB 100,000; fraction=1.0; risk-pct=0 (fixed cash sizing); commission=0.0005 per turnover; same settings all runs. CLI uses next M30 open for entry; stop/target use daily ATR; no mechanics changes.
- Primary ranking metric: expectancy. Admission is fixed: >=20 closed trades in the 24-month training window, >=10 by entry date in each consecutive 12-month block, Net PnL > 0 after commission, PF > 1, MaxDD <= 10% of running equity peak. MaxDD gate is percentage, not absolute RUB.
- Every stage and both exit branches use the same admission rules. Open positions do not count as closed trades. If candidate grids have no qualifying rows, keep original ticker literal unchanged.
- Stage 1: copied template; 120 reference combinations plus explicit current ticker values if not present. Preserve all combinations with `-combos 0`; compare full-run trades by entry year, neighboring values, expectancy, DD.
- Stage 2: separate StopATR x TakeProfitATR grids for UseStochExit=1 and 0, all fields pinned to each admitted stage-1 input. Compare both on identical training window; no final-year inspection.
- Stage 3: local joint refinement and separate one-field sensitivity grids per skill reference; fixed candidate must pass both year gates and quality criteria before holdout.
- Final: once only, compare locked candidate and original full params on reserved 12 months with clean positions and warmup. Require >=10 closed trades plus the same quality gates. Update literal only on pass.
- Budget: max 4 admitted stage-1 inputs; for each, both stage-2 branches; stage-3 on chosen branch unless branch metrics are within 5%, then refine both. Preserve all run grids/reports. No widening search after viewing holdout.
