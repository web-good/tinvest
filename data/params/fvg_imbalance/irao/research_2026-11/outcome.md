# IRAO FVG imbalance calibration outcome

## Decision

The candidate failed the precommitted final-year Max DD limit (13.83% > 10%). It was subsequently written to the IRAO literal at the user’s explicit request; the other final-year results remain as recorded below.

## Locked candidate

`final-candidate.json`:

```json
{
  "ATRPeriod": 14,
  "MinGapATR": 0.25,
  "MaxWaitBars": 45,
  "StochPeriod": 14,
  "UseStochExit": 0,
  "StopATR": 1,
  "TakeProfitATR": 1.5,
  "HTFTrendEMA": 0,
  "RequireEntryAboveFVGLower": 1
}
```

Branch: no stochastic exit, with both daily-ATR stop and target enabled.

## Periods and setup

- Intended training: 2023-11-12 through 2025-11-12; CLI reports ran 16:16–16:21 MSK on those dates. The CLI computes `now` each run. Those cuts all fell between completed M30 bars and the input M30 slices were unchanged.
- Final check: 2025-11-12 16:23 — 2026-11-12 16:23 MSK. Latest included M30 candle opened 2026-11-12 15:30 MSK; latest daily candle was 2026-11-11 03:00 MSK. Market data came from Tinkoff Invest via CandleProvider.
- Starting capital RUB 100,000; fraction 1; commission 0.0005; fixed-fraction sizing; instrument lot from metadata; entries filled next M30 open. Daily ATRPeriod=14 with the core's 28-candle warmup; daily EMA/HTF lead-in was 12 months. No strategy-mechanics changes.
- Locked quality gates: positive net PnL after commission, PF > 1, Max DD <= 10%; training minimum 20 closed trades and >=10 in each sequential year; final minimum 10 closed trades. Max DD is equity-relative percent for the gate (`MaxDrawdownPct`), while the calibration table's `Max DD` column is absolute RUB.

## Results

| Run | Closed trades | Training year 1 / year 2 | Net PnL | PF | Max DD | Open at end |
|---|---:|---:|---:|---:|---:|---|
| Original, 24-month training | 33 | 18 / 15 | +4,039.22 | 1.250 | 10.50% | No |
| Locked candidate, 24-month training | 24 | 11 / 13 | +43,236.77 | 3.086 | 9.55% | No |
| Original, final 12 months | 16 | — | +2,588.78 | 1.628 | 4.26% | No |
| Locked candidate, final 12 months | 13 | — | +1,716.67 | 1.069 | 13.83% | No |

Final candidate acceptance: 13 closed trades meets the count floor; Net PnL and PF are positive/above one. Max DD fails by 3.83 percentage points. It also underperformed the original on final-year Net PnL by RUB 872.11, PF by 0.559, and had 9.57 percentage points more drawdown. No fallback candidate was tested on the same final year.

## Search summary

- Stage 1 evaluated all 350 supplied entry combinations (the template's actual product is 350, despite the skill prose saying 120) and independently ran the original control. The first four train-year-qualified quality candidates advanced: ranks 2, 3, 5, and 8. Rank 1 was excluded because its second year had 8 trades.
- Stage 2 separately evaluated both exit branches for each of those four candidates. Best admissible no-stochastic: rank-2 entry / combo 1, expectancy 1,581.38, 22 trades (10 / 12), PF 2.728, PnL +34,790.46, DD 9.54%. Best admissible stochastic exit: rank-5 entry / combo 1, expectancy 697.59, 24 trades (13 / 11), PF 5.641, PnL +16,742.19, DD 3.58%. Their expectancy difference exceeded the precommitted 5% close-result band; no-stochastic proceeded.
- Stage 3 joint grid's top candidate matched the stage-2 no-stochastic candidate. Sensitivity on MaxWaitBars [15,30,45] identified 45 as a better train-period result, so one joint 54-combination rerun was performed around it; rank 1 retained the locked full parameter set. Sensitivity checks did not produce a valid superior StochPeriod or HTFTrendEMA alternative.

## Reproduction artifacts

- Plan and frozen criteria: `plan.md`
- Full original baseline: `original.json`; research baseline: `baseline.json`; control run: `reports/IRAO_fvg_imbalance/research_2026-11/control/`
- Stage 1 grid and reports: `stage-1-entry.json`; `reports/IRAO_fvg_imbalance/research-2026-11/stage-1/`
- Stage 2 grids/reports: `stage-2-rank{2,3,5,8}-{stoch,no-stoch}.json`; `reports/IRAO_fvg_imbalance/research-2026-11/stage-2/`
- Stage 3 and sensitivity: `stage-3-refinement.json`, `sensitivity-{stochperiod,htftrendema,maxwaitbars}.json`, `stage-3-rerun-maxwait45.json`; matching report directories under `reports/IRAO_fvg_imbalance/research-2026-11/`
- Locked candidate training report and final comparison reports: `reports/IRAO_fvg_imbalance/research-2026-11/stage-3-rerun-maxwait45/final-candidate-training/` and `reports/IRAO_fvg_imbalance/research-2026-11/final12/`
