# LKOH FVG calibration plan (2026-11-11)

## History and execution

- Ticker / interval: LKOH / completed M30 candles; strategy uses completed daily candles for Wilder ATR(14) and EMA filters. Candle cache currently spans M30 2023-08-04 22:00 MSK through 2026-11-10 22:30 MSK and daily 2021-08-12 through 2026-11-09. The final observed M30 candle is complete.
- Requested training window: `[2023-11-11, 2025-11-11)` (24 months), split `[2023-11-11, 2024-11-11)` and `[2024-11-11, 2025-11-11)`.
- Reserved validation window: `[2025-11-11, 2026-11-11)` (12 months). Do not inspect until a single candidate is locked after stage 3.
- Data source: repository cache `data/candles/LKOH_Minutes30.json` and `LKOH_Day1.json`; no refresh during runs. Daily warm-up supplied by CLI begins one calendar year before window start; strategy's ATR requires 28 weekday daily bars and EMA uses requested completed daily closes.
- Execution: next M30 open; initial cash 100,000 RUB; fraction 1.0; risk sizing 0; commission 0.0005 per side; instrument lot as resolved by Tinkoff API. Stop/target in daily ATR from actual fill. Same settings for all runs.
- Current literal saved as `original.json`: ATRPeriod 14, MinGapATR 0.25, MaxWaitBars 30, StochPeriod 11, UseStochExit 1, StopATR 0.4, TakeProfitATR 1.9, HTFTrendEMA 0, RequireEntryAboveFVGLower 0. Baseline for research is `baseline.json` from the calibration reference (not applied to the literal).

## Selection rules (fixed before results)

- Rank by expectancy. On training window, require >=20 closed trades overall and >=10 closed trades by entry date in each of the two fixed annual blocks. Open positions do not count.
- Quality gates: Net PnL after commission > 0, Profit Factor > 1, MaxDrawdownPct <= 0.10. The engine's `MaxDrawdownPct` is a fraction of running equity peak; `MaxDrawdown` itself is absolute currency.
- Stage 1: reference 420-combination entry grid (350 reference combinations plus the added LKOH StochPeriod=11), plus original literal and baseline controls; up to four eligible input candidates carried forward based on annual trade counts, both annual results, DD and neighboring values.
- Stage 2: independent StopATR × TakeProfitATR grids for both exit branches per carried input candidate; all nine fields explicit, `-combos 0`. Compare only eligible candidates; if branch expectancy difference is <=5% of larger absolute expectancy, refine both branches.
- Stage 3: bounded local joint refinement per reference instructions, then separate one-field sensitivity grids for StochPeriod, HTFTrendEMA and MaxWaitBars. One real joint rerun is allowed if sensitivity produces a better eligible candidate.
- Before final test, lock exactly one full parameter set. Final test is the reserved trailing 12 months, fresh position and indicator warm-up; require >=10 closed trades and the same quality gates. Compare only locked candidate and original literal.
- If no eligible candidate or insufficient cached history, preserve reports and leave the ticker literal unchanged.

## Run budget

- Stage 1: 420 grid combinations + two controls.
- Stage 2: at most four entry candidates × two branches × (24 stochastic-exit + 16 no-stochastic-exit combinations), 160 total.
- Stage 3: at most 54 joint combinations per refined branch, three independent sensitivity sweeps, and at most one 54-combination joint rerun per branch.
- Final: two single-parameter-set reports, only after candidate lock. All grids use `-combos 0`, metric `expectancy`, `-min-trades 20`, `-cpu 2`.
