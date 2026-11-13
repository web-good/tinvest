# UGLD FVG imbalance calibration outcome — 2026-11-12

## Decision
No candidate was admitted from stage 1. The UGLD literal was not changed. Stages 2 and 3 and the final holdout were not run because no stage-1 candidate met the precommitted quality gates while also meeting the 20-closed-trade floor. No final-year result was inspected for parameter choice.

## Data and execution
- Provider/cache: Tinkoff Invest candles through `cmd/backtest`; M30 and daily candles were refreshed for the control and stage-1 training window.
- Requested calibration window: 2023-11-12 23:02 to 2025-11-12 23:02 MSK (CLI's actual `now` timestamp); year blocks split at 2024-11-12 23:02.
- Available requested M30 window began only at 2023-11-22 16:00 MSK. The CLI emitted its missing-history warning. It ended at 2025-11-12 23:00 MSK for these runs. Daily data included the CLI's one-year lead-in; strategy ATRPeriod=14 requires 28 completed daily candles. All decisions use completed M30 bars, with execution at the next M30 open.
- Run costs/sizing: 100,000 RUB cash, fraction 1, risk-pct 0, commission 0.0005, default instrument lot. Max drawdown quality gate is `MaxDrawdownPct <= 0.10` (fraction of running peak); calibration table's Max DD column is absolute RUB, while the single-run report gives both RUB and percent.
- Full eligible calibration span is short by the first ten days because UGLD cache/history starts on Nov 22. Treat year-block counts as descriptive for the loaded history, not evidence that missing bars were covered.

## Original control
Source params are in `original.json`; baseline reference is in `baseline.json`. Control report: `control-train/UGLD_fvg_imbalance_Minutes30_20261112_230217.md`, with trades/equity CSV alongside it.

| Window | Closed trades by entry date | Net PnL | PF | Max DD |
|---|---:|---:|---:|---:|
| Requested 24-month training window | 32 | -25,305.69 RUB | 0.401 | 29.48% |
| First year [2023-11-12, 2024-11-12) | 15 | — | — | — |
| Second year [2024-11-12, 2025-11-12) | 17 | — | — | — |

The control meets the two descriptive yearly count thresholds, but fails all locked quality gates.

## Stage 1
Grid copy: `stage-1-entry.json`. Complete ranking: `stage-1-entry/UGLD_fvg_imbalance_Minutes30_20261112_230300_calibration.md`; best single run: `stage-1-entry/UGLD_fvg_imbalance_Minutes30_20261112_230300_best.md`.

The actual template has 350 combinations (7 MinGapATR × 5 StochPeriod × 5 HTFTrendEMA × 2 entry-floor settings), although the skill description states 120. All 350 were saved using `-combos 0`.

- 98 combinations had at least 20 closed trades. None had positive expectancy, positive net PnL, or PF > 1. Thus none met the locked quality gate, before applying the additional yearly-count rule.
- The highest expectancy across the grid was +374.1 RUB, but that set had only 7 closed trades total (3 `RecentTrades`), so it was ineligible.
- Because every trade-qualified set already failed quality, no combination was forwarded; no claim is made that either exit branch or local refinements were searched.

The literal remains at the original full parameter set in `internal/service/trading_strategy/fvg_imbalance/strategy/ugld/ugld.go`. No final holdout candidate was locked, so the final year remains untested for selection purposes.
