# RNFT FVG imbalance calibration plan

- Planned run date: 2026-11-12 (CLI `time.Now`; save each report's actual candle boundaries and reject any mismatched window).
- Training window: 2023-11-12 through 2025-11-12, half-open; two training blocks 2023-11-12–2024-11-12 and 2024-11-12–2025-11-12.
- Reserved final window: 2025-11-12 through 2026-11-12, half-open. Do not inspect until one stage-3 candidate is locked.
- Source: Tinkoff Invest M30 and Day1 candles via backtest CandleProvider/cache; completed candles only. Strategy reads completed M30 bars and prior completed weekdays for daily ATR (Wilder ATRPeriod 14, requiring 28 daily bars); daily EMA gets one year lead-in from CLI. Confirm coverage and boundary timestamps before proceeding.
- Execution assumptions: initial cash 100,000 RUB, fraction 1.0, commission 0.0005 of turnover, risk-pct 0, next M30 open entry, instrument lot from API, identical throughout.
- Baseline: skill `referens/baseline.json`; ticker literal remains unchanged during calibration. Save baseline full params and control run.
- Primary metric: expectancy. Gates, fixed before results: positive Net PnL after costs, PF > 1, Max DD <= 10% (verify DD unit in engine), >=20 closed trades over training and >=10 by entry date in each fixed training year; final >=10 closed trades and same quality gates.
- Search: stage 1 entry template plus ticker's original values; stage 2 independent StopATR × TakeProfitATR grids for both exit branches and each admitted entry candidate; stage 3 local interaction grid and separate one-field sensitivities per refinement instructions. All grids explicitly contain nine fields; combos=0.
- Grid-count note: the supplied stage-1 template has 7×5×5×2 = 350 combinations, not 120 as stated in the skill prose. Actual JSON is authoritative. Baseline control is additional.
- Budget: at most 350 stage-1 combinations + baseline control; up to four stage-2 candidates × 24 combinations × two branches = 192 combinations; stage 3 up to 54 joint combinations plus separate sensitivities (StochPeriod neighbors, HTFTrendEMA alternative, MaxWaitBars 15/30/45), and exactly two final-year runs (candidate and baseline). No expanding search beyond these caps without recording a revised pre-result plan.
