# SBERP FVG imbalance calibration plan

## Frozen criteria and budget (written before first backtest)
- Run date anchor: 2026-11-12; requested window is calibration 2023-11-12 through 2025-11-12, split into [2023-11-12, 2024-11-12) and [2024-11-12, 2025-11-12); reserved final validation 2025-11-12 through 2026-11-12.
- One exact candle snapshot is required across all commands. The local cache currently contains SBERP M30 2023-08-04..2026-10-13 and daily 2021-08-12..2026-10-27; refresh/API must extend M30 through the reserved end before calibration. If source coverage cannot provide the entire window, stop before tuning and report insufficiency.
- Source: Tinkoff Invest API via the backtest candle provider (`data/candles` cache); completed M30 and daily candles only. Daily lead-in: 12 months as requested by CLI. Strategy ATR warm-up: 28 completed daily bars (2*ATRPeriod); EMA candidate periods use completed daily closes, warmed by the 12-month daily lead-in. The engine uses completed M30 candles and next-bar-open execution.
- Execution held constant: initial cash RUB 100,000; fraction 1.0; risk-pct 0; commission 0.0005 per turnover; SBERP instrument lot as resolved by API; no parameter-specific changes to execution.
- Primary ranking metric: expectancy. Acceptance: positive Net PnL after commission, Profit Factor > 1, Max DD <= 10% of running peak (use MaxDrawdownPct, not absolute currency drawdown), and at least 20 completed round trips over 24 months with at least 10 entered in each exact 12-month block. Final year requires at least 10 completed trades and the same quality limits. Open trades do not count.
- Stage 1: 350 combinations (copied reference grid plus literal values MinGapATR=.25 and StochPeriod=14); compare full literal control separately. Stage 2: up to four stage-1 candidates, each independently tested in both exit branches using the complete copied templates (96 combinations/branch). Stage 3: one local joint refinement (<=54 combinations) on selected branch, plus separate one-field sensitivity grids (StochPeriod, HTFTrendEMA, MaxWaitBars). At most one joint rerun (<=54) only if sensitivity justifies it. No candidate selection from final-year results.
- Budget cap: 1,300 grid-combination evaluations plus the original/control and final paired runs. All combinations retained (`-combos 0`); annual trade eligibility checked from full-run trade CSV by entry date, never inferred from RecentTrades alone. If cap prevents exhaustively checking the required ranked candidates, report incomplete search.
- Preserve all grids, command output, reports, candidate JSON, and a final outcome note here. Do not edit the SBERP literal unless the locked candidate passes the single reserved final-year comparison against the original.

## Required pre-run checks
- Verify actual candle window endpoints after refresh, especially the calibration split and final-year endpoint.
- Confirm backtest MaxDrawdownPct is a fraction (multiply by 100 for reporting); MaxDrawdown is currency.
- Confirm full parameter grids explicitly include all nine fields; ATRPeriod=14 and MaxWaitBars=30 except the specified sensitivity sweep.

## Stage 1 carry-forward choices (locked before stage 2)

Candidates are ranks 1–4 from the full 350-row report. Each meets both annual trade floors based on total trades and RecentTrades (the latter cutoff matches the second annual block within the candle-bar boundary); their metrics all satisfy the frozen quality limits. Ranks 3 and 4 represent nearby HTF EMA values within the same active entry regime; ranks 1 and 2 represent the strongest distinct entry regimes.
