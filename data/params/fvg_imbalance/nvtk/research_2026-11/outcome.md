# NVTK FVG imbalance calibration outcome

## Selection

- Training window reported by CLI: 2023-11-12 16:11 to 2025-11-12 16:11; annual split 2024-11-12. Stage-1 admissible rank 7 is the existing entry setup. Stage-2 and stage-3 retained this candidate, with the stop=1.2 and target=1.5 neighbors also admissible. Selected and locked branch: `UseStochExit=1`, based on the then-confirmed admissible candidate and stage-3 stability. The stage-2 grids are fully preserved, but the annual audit did not exhaust all lower-ranked no-stochastic results; no claim is made that this branch had no admissible candidate.
- Frozen complete candidate: see `final-candidate.json`. It matches the prior NVTK/core defaults, so this is verification of current settings rather than a parameter change.
- Final window reported by CLI: 2025-11-12 16:17 to 2026-11-12 16:17. Candidate was locked before this run. Original settings were then replayed once on the refreshed local cache; reports are identical.

## Results

| Period | Closed trades by entry date | Open position | Net PnL | PF | Max DD | Expectancy |
|---|---:|---|---:|---:|---:|---:|
| Full training, two continuous years | 25 (10 / 15 by year) | None | +9,039.93 RUB | 1.737 | 8.43% | +361.60 RUB |
| Final year | 25 | One, marked to market | +1,324.86 RUB | 1.102 | 9.76% | +53.58 RUB |

The final report's Net PnL includes mark-to-market value of the open position. The 25 closed trades meet the minimum independently of that position. Candidate and original result: identical, as the parameter sets are identical. Final window meets >=10 closed trades, positive net, PF >1 and Max DD <=10%, so the existing NVTK parameter set was revalidated and recorded explicitly. It is unchanged from the prior core defaults. The incomplete lower-rank branch audit limits any claim of optimality.

## Reproduction

Training coverage: 23,857 completed M30 candles from 2023-11-12 to 2025-11-12; the replay used 814 daily candles and 4,169 H4 candles with the engine lead-in. Final data was refreshed from Tinkoff before candidate evaluation; the original run reused that same cache. The initial refresh attempt timed out; the final-year refresh completed. The refresh replaced the local training cache with the final-year window, so lower-ranked stage-2 yearly audits could not be resumed afterward without re-fetching training data. The locked candidate was not changed after opening the final year.

Costs and sizing for all runs: RUB 100,000 initial cash, fraction=1, risk-pct=0, commission=0.0005, default NVTK lot. Stage 1 used 350 combinations; all ranking rows are preserved in its report. Selection required >=20 closed trades and >=10 per training year, positive net PnL, PF>1, Max DD<=10%. Stages 2 and 3 used the corresponding saved grids and separate reports. All calibration commands used `-combos 0` and identical training boundaries. M30/daily caches were refreshed before the final candidate run; the original run reused that cache. Total grid runs, fixed-parameter qualification replays, sensitivities and final runs stayed within the precommitted budget of 1,000 evaluations. The engine warmed daily ATR/EMA from its one-year lead-in; ATRPeriod=14 uses completed daily candles only.
