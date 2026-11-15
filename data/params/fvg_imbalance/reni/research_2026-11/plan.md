# RENI FVG imbalance calibration plan (2026-11)

## Data and windows
- Instrument/timeframe: RENI, completed M30 bars; daily bars supply ATR/EMA and must be completed before each M30 formation/decision as enforced by `core`.
- Source snapshot: local `data/candles/RENI_Minutes30.json` (34,009 bars, 2023-10-06 11:30 MSK through 2026-11-03 22:00 MSK) and `data/candles/RENI_Day1.json` (1,387 bars, 2021-10-21 through 2026-11-02). Re-check coverage after each run; do not refresh midway.
- Intended calibration: [2023-11-13, 2025-11-13), divided into [2023-11-13, 2024-11-13) and [2024-11-13, 2025-11-13). Final reserve: [2025-11-13, 2026-11-13). The current cached M30 history ends 2026-11-03, so the full final window is not yet available; no candidate can be marked verified or written into `reni.Params()` during this run.
- Indicator warmup: daily ATR period fixed at 14, implementation requires 2*ATRPeriod daily bars; EMA history uses completed daily closes (period by parameter). M30 stochastic period as in grid, with strategy lookback including `StochPeriod+1`; all runs use identical cache/history.
- Execution/economics: initial cash 100,000 RUB, fraction 1.0, commission 0.0005 per turnover, risk-pct 0 (fixed fraction); same engine execution rules for every run.

## Criteria (locked before results)
- Main metric: expectancy. Quality gates: Net PnL > 0 after commission, PF > 1, Max DD <= 10%; the engine reports MaxDrawdownPct as a fraction and the markdown renders it as percent. Max DD gate uses percentage.
- Every stage/exit branch requires >=20 closed trades over the full training period and >=10 closed trades by entry date in each of the two calendar training blocks. Open positions do not count. Stage-grid `RecentTrades` is only a preliminary second-year proxy; verify both blocks from fixed-parameter full-run trade CSVs.
- Final validation requires a full reserved 12-month window, >=10 closed trades, and the same quality gates. Since the current cache ends 10 days early, final validation is deferred; no literate update is permitted.

## Grids and search budget
- Stage 1: copied `stage-1-entry.json`, 120 combinations; add original ticker settings if absent (already represented), plus a separate unmodified `original.json` control. Evaluate all combos (`-combos 0`), then inspect and full-run up to four eligible entry candidates across neighborhoods.
- Stage 2: copied stoch/no-stoch templates, one independent complete 9-field grid per selected entry and branch; up to four entries × two branches × 24 combos. Compare on identical 24-month training history; inspect both training years and neighboring values.
- Stage 3: use the selected branch, jointly refine StopATR × TakeProfitATR × MinGapATR × RequireEntryAboveFVGLower (<=54 combinations), with separate one-field sensitivity grids for StochPeriod, HTFTrendEMA and MaxWaitBars [15,30,45]. One limited joint repeat only if sensitivity points to a better eligible candidate.
- Budget: 120 stage-1 combinations; at most 192 stage-2 combinations; at most 54 joint refinement combinations plus three small sensitivity runs; fixed-candidate checks only for candidates promoted after eligibility review. Preserve all calibration reports and per-run trades/equity.

## Baseline distinction
- Research baseline is `baseline.json` from the skill reference (MinGapATR=0 and otherwise core baseline); it does not change the RENI literal. The actual current RENI literal is core defaults (`MinGapATR=0.25`, other values in `original.json`) and is evaluated as a separate control.

## Execution note
- The supplied stage-1 JSON contains 350, not 120, combinations (7 MinGapATR × 5 StochPeriod × 5 HTFTrendEMA × 2 entry-floor values). The whole grid was retained and run with `-combos 0`; this supersedes the template's stated count and plan's initial arithmetic, without changing any grid values or gates.

## Actual search accounting and stop
- Actual evaluated combinations: stage 1 = 350; stage 2 = 2 eligible entry starts × (24 stoch + 16 no-stoch) = 80; stage 3 = 54 + 3 sensitivity grids (3 each) + one 54-combination repeat = 117; total calibration combinations = 547, plus fixed-parameter controls/candidate checks.
- Training report boundaries were 2023-11-13 23:17–23:20 MSK through the same timestamp on 2025-11-13; year split is start + 12 calendar months. Daily and M30 cache were not refreshed between runs.
- The full final holdout was not available: M30 cache ended at 2026-11-03 22:00 MSK. No final-year run was made and no literal update is authorized by the calibration acceptance criteria until a full reserved 12-month period is available.
