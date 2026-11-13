# TATN FVG imbalance calibration plan

## Frozen criteria (before results)

- Run date: 2026-11-12 Europe/Moscow.
- Intended calibration window: `[2023-11-12, 2025-11-12)` (24 months).
- Reserved final check: `[2025-11-12, 2026-11-12)` (12 months). Do not inspect for candidate selection.
- Source: Tinkoff Invest candles via `cmd/backtest`, cache at `data/candles`; compare on one frozen candle snapshot. Only completed M30 bars and completed weekday daily bars are usable. M30 decisions are at close and entries fill at next open. Daily EMA lead-in requested by CLI is 12 months; ATR is Wilder period 14 and core needs 28 daily candles. Stochastic warmup is each grid's period + 1 bars.
- Existing cache bounds observed before runs: TATN M30 `2023-10-06 22:30 +03` through `2026-10-15 13:00 +03`; Day1 `2021-08-12` through `2026-10-27`. This does not establish availability through the reserved end; verify actual loaded bounds before any candidate decision.
- Baseline: `referens/baseline.json`; literal currently resolves to `core.DefaultParams()`: `{ATRPeriod:14, MinGapATR:0, MaxWaitBars:30, StochPeriod:14, UseStochExit:1, StopATR:1, TakeProfitATR:2, HTFTrendEMA:0, RequireEntryAboveFVGLower:0}`.
- Execution costs and sizing: commission `0.0005` per turnover, initial cash `100000 RUB`, fraction `1`, risk-pct `0`; no leverage. Same settings throughout.
- Primary metric: expectancy. Fixed quality limits: net PnL after commission > 0, PF > 1, max drawdown <= 10% (use `MaxDrawdownPct`, a fraction in engine, report as percent).
- Each training stage and exit branch: >=20 closed trades total, >=10 by entry date in each exact 12-month block. Final year: >=10 closed trades. Open positions never count. Do not use RecentTrades as a substitute unless its boundary matches exactly.
- Stage 1: the provided 120-entry grid plus baseline control. Stage 2: independent StopATR × TakeProfitATR grids for both exit branches for up to four eligible stable entry candidates. Stage 3: local joint refinement and separate one-field sensitivity runs per the skill reference. All grids list all nine fields; `-combos 0`; separate reports.
- Budget: run full stage 1 (120 combos) and baseline; stage 2 only for stage-1 candidates passing all fixed criteria and year counts; stage 3 only for branch finalists. Stop without claiming exhaustive absence if report/budget limits prevent reviewing all candidates. Final test only after locking one candidate before reading the reserved year.

## Run log

At plan creation, no calibration results had been viewed. CLI windows matched the intended 24-month training and 12-month final intervals. After loading, cached candles covered M30 through `2026-11-12 22:00 +03` and Day1 through `2026-11-11 03:00 +03`; runs used completed candles only. Reports were dated 2026-11-12 22:29–22:32 MSK. Stage 1 produced 350 combinations from the provided grid's actual cartesian product. The reserved final year was opened after locking one candidate. The locked-candidate command mistakenly used a different JSON file; see `outcome.md`. Therefore no valid final-year candidate result exists and this year is no longer an unseen holdout.
