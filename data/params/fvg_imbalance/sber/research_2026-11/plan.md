# SBER FVG imbalance calibration plan

Recorded before any backtest run: 2026-11-11.

## Data and windows

- Source: local Tinkoff candle cache (`data/candles/SBER_Minutes30.json`, `SBER_Day1.json`); CLI refresh is off. Cached M30 spans 2023-08-04 22:00 MSK through 2026-11-10 23:30 MSK; daily spans 2021-08-12 through 2026-11-09. Runs use only completed candles. The backtest loader requests the exact CLI window and daily warmup from one year before its start.
- Planned selection/training: 2023-11-11 through 2025-11-11, split into `[2023-11-11, 2024-11-11)` and `[2024-11-11, 2025-11-11)`.
- Reserved final check: 2025-11-11 through 2026-11-11. Do not inspect these results before locking one candidate; check actual candle boundaries in each report.
- Indicator warmup: daily candles begin one calendar year before the M30 window; ATRPeriod=14 and EMA is warmed from the available daily series. No extra HTF series is used by FVG.

## Execution and costs

- Initial capital: RUB 100,000; fraction 1.0; risk-pct 0; commission 0.0005 per side/turnover as configured by engine; M30 signals fill at next bar open, exits per engine stop/target/stochastic logic.
- All runs use identical cached history, costs, sizing, and execution policy.
- In this engine MaxDrawdown is absolute RUB and MaxDrawdownPct is a fraction of peak equity. Acceptance uses MaxDrawdownPct <= 0.10.

## Selection rules

- Primary metric: expectancy. Candidate acceptance also requires positive Net PnL after commission, Profit Factor > 1, and MaxDrawdownPct <= 10%.
- Each stage/branch candidate needs >=20 closed trades over the 24-month selection interval and >=10 trades by entry date in each exact sequential 12-month block. Open positions do not count. Final check requires >=10 closed trades and the same quality limits.
- `RecentTrades` is supporting information only; verify annual counts from closed-trade entry dates at the exact split.
- Compare neighbors and both annual blocks. Keep up to four stage-1 candidates, then evaluate both exit branches independently. Refine only a locked training candidate. Do not use the reserved year to choose among candidates.

## Grid and run budget

- Stage 1: copied reference grid, 350 combos (the copied template expands to 7 x 5 x 5 x 2), plus separate original-parameter control. Budget 351 runs. The template text’s claimed count of 120 does not match its value lists; preserve and run the lists as supplied.
- Stage 2: per retained stage-1 candidate, copied independent stop x target grids for both exit branches; with input fields fixed, 24 combos in the stoch branch and 16 in the no-stoch branch per start, budget 320 combinations for four starts.
- Stage 3: one local joint grid (up to 54 combos) and separate one-field sensitivity grids (StochPeriod, HTFTrendEMA, MaxWaitBars), plus at most one bounded local rerun if sensitivity changes the finalist.
- Preserve all combos (`-combos 0`), all grids and reports under this directory. If budget or qualifying results prevent a stage, document exactly what was and was not searched. No thresholds will change after seeing results.

## Итог

Фактические обучающие отчёты: окна CLI отображены как 2023-11-11 14:12—2025-11-11 14:18 по мере прогонов; срез кэшированных M30 свечей был общим по годовым границам (даты входа считаются по таблице CSV). Зарезервированное окно было открыто после `locked-candidate.md`; оба финальных отчёта показывают 2025-11-11 14:19—2026-11-11 14:19. Кандидат провалил лимит MaxDrawdownPct 10% (факт 10.56%), поэтому литерал оставлен без изменений. Подробности сравнения — `outcome.md`.
