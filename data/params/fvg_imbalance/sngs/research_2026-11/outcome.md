# SNGS FVG calibration outcome

Windows and criteria are locked in [plan.md](plan.md). Data source was `data/candles` via the configured Tinkoff candle provider; actual CLI report windows were exactly 2023-11-11 14:53/14:54 to 2025-11-11 14:53/14:54 for calibration and 2025-11-11 14:56 to 2026-11-11 14:56 for final. Reports show completed M30 bars; daily ATR period 14 (28-bar Wilder seed/smoothing), one-year daily lead-in, completed H4 context with one-year lead-in when enabled. Same engine, costs (0.0005), cash (100000), fraction (1), and risk-pct (0) were used throughout. Max DD limit is 10% equity; engine's `MaxDrawdownPct` is a fraction and reports render it as percent.

## Training controls and stage 1

- SNGS original literal, MinGapATR 0.25: 40 trades, Net PnL -5309.13, PF 0.789, Max DD 12.49%.
- Research baseline (`referens/baseline.json`, MinGapATR 0): 378 trades, Net PnL -40224.87, PF 0.744, Max DD 41.36%.
- Stage-1 grid has 350 combinations (7 MinGapATR x 5 StochPeriod x 5 HTFTrendEMA x 2 floor flags, including literal/baseline values). Top #1 failed first-year volume (9/14); #2 failed second-year volume (15/8). #3 passed with 11/12 trades, positive Net PnL, PF > 1, DD <10%; #4 passed with 10/13. Retained #3 and #4 as two filter regimes. No candidates were promoted based only on total trades.

## Stage 2 and 3

Both exit branches were run for both retained inputs. In each no-stochastic grid, the highest-ranked result was negative (candidate 3: 22 trades, PF 0.704, Net PnL -6961.39; candidate 4: 22, PF 0.860, Net PnL -3107.57), so neither meets quality limits. Stochastic exit candidate 3 ranked first with StopATR 1.7, TakeProfitATR 0: 23 trades (11/12), expectancy 411.13, Net PnL +9455.99, PF 2.501, Max DD 3.32%. Candidate 4's equivalent branch had lower expectancy (347.2), Net PnL +7986.45 and PF 2.125. Top stop-only-price-target combination has no open position at report end; trades were closed by stop or stochastic exit.

The 54-combination joint stage-3 grid reproduced the selected complete set and its metrics. EMA200 had slightly higher expectancy (411.2) but only 8 trades in the second year, so was ineligible. StochPeriod14 and MaxWaitBars15 were weaker; MaxWaitBars45 generated one trade and was ineligible. Stage-3 candidate stayed unchanged and was locked before final-year reports were opened. See [locked-candidate.md](locked-candidate.md).

## Reserved final year

| Set | Closed trades | Net PnL | PF | Max DD | Open at end |
|---|---:|---:|---:|---:|---|
| Original literal | 24 | +2806.12 | 1.284 | 6.93% | No |
| Locked candidate | 13 | +8053.51 | 4.425 | 2.54% | No |

Both met final-year criteria; candidate became the SNGS literal. All PnL values are net of the configured commission. The report engine does not provide average R, so none is reported.

## Reproduction artifacts

- Grids and full parameters: this directory (`baseline.json`, `original.json`, `stage-1-entry.json`, `stage-1-candidate-3.json`, `stage-1-candidate-4.json`, both `stage-2-candidate-*` grids, `stage-3-refinement.json`, sensitivity grids, and `final-candidate.json`).
- Reports: `reports/fvg_imbalance/sngs/research_2026-11/` with baseline/original controls, all stage-1 to stage-3 calibrations, sensitivity reports, and final original/candidate trade reports.
- Calibration report tables save all combinations (`-combos 0`). Yearly trade counts above are derived from each full run's closed trade log using entry dates; the calibration report's RecentTrades represents the 12-month tail and matches the second block boundary here.
