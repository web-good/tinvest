# SBERP FVG imbalance calibration outcome

## Status: research candidate installed at user request

The research candidate passed the reserved final-year thresholds, but the branch comparison was not fully screened against the frozen Max DD percentage limit before that year was opened. The user subsequently requested that the research candidate be installed despite this limitation. The reserved year has now been used and must not be reused to select a replacement.

Research candidate in `final-candidate.json`:

```json
{
  "ATRPeriod": 14,
  "MinGapATR": 0.1,
  "MaxWaitBars": 15,
  "StochPeriod": 21,
  "UseStochExit": 1,
  "StopATR": 1.5,
  "TakeProfitATR": 1,
  "HTFTrendEMA": 200,
  "RequireEntryAboveFVGLower": 0
}
```

`internal/service/trading_strategy/fvg_imbalance/strategy/sberp/sberp.go` now returns this research candidate.

## Criteria and execution

- Primary ranking metric: expectancy.
- Frozen quality limits: positive Net PnL after commission, PF > 1, Max DD <= 10% of running peak; at least 10 closed trades in each training year and 20 total; at least 10 closed trades in the final year.
- Training window: 2023-11-12 through 2025-11-12; annual blocks `[2023-11-12, 2024-11-12)` and `[2024-11-12, 2025-11-12)`.
- Reserved validation: 2025-11-12 through 2026-11-12. The candidate command labels its range 16:17 MSK and original 16:19 MSK, but both selected the same 13,728 completed M30 bars, from 2025-11-12 16:30 through 2026-11-12 15:30 MSK. Their completed daily series were also identical.
- Source: Tinkoff Invest API through `CandleProvider`, cached at `data/candles/SBERP_Minutes30.json` and `data/candles/SBERP_Day1.json`. Latest completed M30 bar: 2026-11-12 15:30 MSK; latest daily candle: 2026-11-11. Daily lead-in: 12 months; ATR warm-up: 28 completed daily bars; EMA uses completed daily closes.
- All runs used RUB 100,000 initial cash, fraction 1, risk-pct 0, commission 0.0005, API-resolved lot, M30 close signal, and next-bar-open execution. No trading mechanics changed.
- `MaxDrawdownPct` is the running-peak drawdown fraction; reports show it as a percentage. `MaxDrawdown` is currency (RUB).

## Training results

- Original literal/defaults: 27 closed trades, 13 / 14 by year; Net PnL RUB 368.55, PF 1.048, Max DD RUB 3,554.19 (3.52%).
- Candidate after stage-3 joint rerun: 40 closed trades, 15 / 25 by year; expectancy RUB 280.92, Net PnL RUB 11,236.74, PF 3.359, Max DD RUB 2,771.50 (2.76%). No open position at the end.
- The strongest raw no-stochastic result inspected in detail was stage-2 input candidate 3 rank 1: expectancy RUB 568.8, PF 1.587, 42 trades (20 / 22), but Max DD was 12.28% and failed the limit.
- Screening gap: other no-stochastic rows had enough trades and attractive metrics, including candidate 3 rank 9 (expectancy 239.8, PF 1.350, 47 trades split 23 / 24, absolute DD RUB 10,453.30). Calibration tables report absolute DD only; I did not run its single-parameter report to check Max DD percentage before locking the candidate. Thus I cannot claim that the chosen branch was the best quality-qualified branch. The other final-year candidate cannot be substituted after opening the reserved year.

## Reserved final-year comparison (same candles)

| Set | Closed trades | Open at end | Net PnL | PF | Max DD |
|---|---:|---|---:|---:|---:|
| Research candidate | 33 | No | RUB 4,190.13 | 1.960 | RUB 3,820.73 (3.57%) |
| Original literal | 38 | Yes, mark-to-market | RUB 4,149.20 | 1.700 | RUB 4,563.38 (4.35%) |

The candidate passed the final-year numeric limits and modestly exceeded the original on this window. Because branch selection evidence was incomplete before final validation, these results are retained as research evidence only; the candidate was installed at the user’s explicit request.

## Reproduction artifacts

- Frozen plan: `plan.md`.
- Copied research baseline and grids: `baseline.json`, stage-1, both stage-2 templates, stage-3 refinement, candidate-specific stage-2 grids, and stage-3 local/sensitivity/rerun grids.
- Full original and research candidate objects: `original.json`, `final-candidate.json`.
- Pre-validation lock and post-validation qualification note: `locked-before-final.md`.
- Reports: `reports/fvg_imbalance/SBERP_2026-11/` (control, stage-1, stage-2, stage-3, final).
