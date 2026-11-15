# LSRG FVG imbalance calibration plan (2026-11-13)

## Frozen criteria
- Training window: 2023-11-13 through 2025-11-13 (24 months), split into [2023-11-13, 2024-11-13) and [2024-11-13, 2025-11-13). Reserve 2025-11-13 through 2026-11-13 for a single final evaluation; do not inspect it for selection.
- Cached M30 source presently spans 2023-08-07 through 2026-10-13. The requested OOS window is therefore not yet fully represented. Refresh/API retrieval will be attempted; if complete daily/M30 history cannot be obtained, do not claim a final validation or alter the ticker literal.
- Stage 1: copied common 120-combination entry grid plus explicit original candidate; base exits from referens/baseline.json. Stage 2: separate stop/target grids for each admitted entry, both stochastic exit branches. Stage 3: local joint refinement and separate one-field sensitivity grids per referens/stage-3-refinement.md.
- Primary metric: expectancy. Admission: at least 20 closed trades total, at least 10 by entry date in each training year; positive net PnL after commission; PF > 1; max drawdown <= 10% (equity peak-to-trough percentage, not currency amount). Apply at each stage and both exit branches. Open positions do not count.
- Budget: run full stage-1 grid with -combos 0; inspect/rerun ranked eligible candidates only as needed, preserve all reports, and avoid broad exploratory expansions beyond template. Fixed execution for all runs: M30, $100,000 starting capital, fraction 1.0, commission 0.0005, risk-pct 0, CPU 2; next-bar-open execution, completed candles only. Daily ATR period 14; one-year daily lead-in in CLI, plus engine ATR warmup 28 daily candles; H4 lead-in for nonzero HTF EMA.
- The CLI computes time.Now per invocation and `-months 36 -holdout-months 12` means 24-month training ending 12 months before now. Record report candle bounds and reject comparisons if they differ. Final test requires complete reserved year and >=10 closed trades.

## Baseline
- Current LSRG Params() delegates to core.DefaultParams(): {"ATRPeriod":14,"MinGapATR":0.25,"MaxWaitBars":30,"StochPeriod":14,"UseStochExit":1,"StopATR":1,"TakeProfitATR":2,"HTFTrendEMA":0,"RequireEntryAboveFVGLower":0}.
- Save independent control run and full trade/equity tables before inspecting grid results.
