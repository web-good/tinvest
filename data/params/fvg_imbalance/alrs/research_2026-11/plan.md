# ALRS FVG imbalance calibration plan

## Window and data snapshot

- Started: 2026-11-12 13:27 MSK. An initial run batch crossed the 13:30 M30 boundary; those reports are retained but excluded from selection. Repeated all controls and grids in a common candle snapshot during 13:37–13:38 MSK. The fit reports cover `[2023-11-12 13:37, 2025-11-12 13:37]`; all aligned runs include the same final completed M30 candle at 2025-11-12 13:30 MSK. The reserved final-year run will use `-months 12 -holdout-months 0` and record its actual boundaries.
- Reserved-year attempt: candidate report window 2025-11-12 14:09 .. 2026-11-12 14:09; original report 2025-11-12 14:11 .. 2026-11-12 14:11. The M30 cache ends at 2026-11-12 13:30 but has an internal hole from 2026-10-14 through 2026-10-26 after an API `DeadlineExceeded` for the 2026-10-13 .. 2026-10-27 chunk. Both reports are retained as invalid diagnostics; neither qualifies as an independent complete final year.
- Candle source: Tinkoff Invest API through `CandleProvider`, using the repository cache `data/candles/ALRS_*`; cache inspected before run: M30 2023-08-04 17:30 .. 2026-10-13 13:00 MSK; Day1 2022-10-28 .. 2026-10-27. Provider may top up missing tail during run. Use completed candles only; record report boundaries and last returned candles after fetch.
- Daily warm-up: CLI loads one extra calendar year before window for daily EMA; strategy uses completed weekdays only, requires 28 completed daily candles for ATR(14), and HTF EMA only after at least the selected daily period. M30 lookback includes max(3+MaxWaitBars, StochPeriod+1).
- Execution assumptions: initial cash 100,000 RUB; fraction 1.0; commission 0.0005 per turnover; fixed-fraction sizing (`risk-pct=0`); lot from ALRS instrument metadata; entries at next M30 open, exits per engine; identical settings for all runs.

## Frozen decision rules

- Primary metric: expectancy per completed trade. Quality gate: positive Net PnL after commission, PF > 1, Max DD <= 10% of running peak (use `MaxDrawdownPct`; absolute `MaxDrawdown` is in RUB). Candidate selection additionally requires >=20 completed trades over fit and >=10 by entry date in each sequential 12-month block. Open positions do not count.
- Reserved final-year gate: >=10 completed trades, positive Net PnL, PF > 1, Max DD <= 10%. Compare to original parameters; do not substitute another candidate after opening this window.
- Do not inspect reserved-year results until one full candidate is locked in writing.
- Budget: stage 1 full template grid plus separate original control and baseline control (352 total runs); stage 2 four stage-1 candidates, both exit branches, full template grids (160 runs); stage 3 one 36-combination joint grid, one-field sensitivity grids and one 36-combination bounded joint rerun. Retain all combinations (`-combos 0`), CPU=2.

## Planned grids

- Stage 1: copy `referens/stage-1-entry.json`; preserve its full supplied values and add ALRS original `MinGapATR=0.25` if absent. The arrays as supplied multiply to 350 combinations (despite the skill prose saying 120); run the actual copied grid and report this discrepancy.
- Stage 2: independent `StopATR × TakeProfitATR` grids for `UseStochExit=1` and `0`, fixing the other seven fields to each admitted stage-1 candidate.
- Stage 3: use `referens/stage-3-refinement.md`; select neighbors from actually run grids and keep all fields explicit.

## Artifacts

- `original.json`: full current ticker parameters before any edit.
- `baseline.json`: research baseline from skill reference, not applied to source defaults.
- Stage grids, dated reports, candidate lock and outcome will remain under this directory.
