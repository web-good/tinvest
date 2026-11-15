# IVAT FVG imbalance exploratory recalibration outcome (2026-11-14)

## Status
The exploratory candidate is preserved in `final-candidate.json`; the IVAT strategy literal was not created or changed. Training had only about 17 months of IVAT history and the subsequent validation slice had under 12 months. In addition, the candidate breached the precommitted validation Max DD limit, so it is not eligible for ticker use.

## Locked candidate
```json
{
  "ATRPeriod": 14,
  "MinGapATR": 0.3,
  "MaxWaitBars": 30,
  "StochPeriod": 14,
  "UseStochExit": 0,
  "StopATR": 1,
  "TakeProfitATR": 1,
  "HTFTrendEMA": 0,
  "RequireEntryAboveFVGLower": 1
}
```
- Selected exit branch: no stochastic exit. Best stage-2 expectancy was RUB 1,849.63 versus RUB 1,124.27 for the stochastic branch; difference was larger than the precommitted 5% proximity threshold.
- Stage 3 re-ran the full center and nearest local grid. The center remained top by expectancy. One-field sensitivities did not produce a stronger eligible setting: StochPeriod 11 had expectancy RUB 254.70, PF 1.120, DD RUB 29,606.37, 25 trades; HTFTrendEMA 50 had only 7 trades; MaxWaitBars 45 had expectancy RUB 3,772 but only 13 trades, below the 20-trade exploratory training floor.
- Effective training source began 2024-06-04; report boundary was 2023-11-13 23:59–2025-11-13 23:59. Candidate closed entries: 11 in `[2024-06-04, 2025-06-04)`, 10 in `[2025-06-04, 2025-11-13]`. These unequal blocks do not meet the standard two-full-year rule.
- Stage-3 candidate metrics: 21 closed trades; Net PnL +RUB 38,842.28 after commission; PF 2.369; Max DD RUB 11,048.40 (7.84%); expectancy RUB 1,849.63. No open position is listed in the report.

## Approximate validation vs baseline
Both reports cover 2025-11-14 00:02–2026-11-14 00:02. M30 history was updated through 2026-11-13 23:30 during the candidate run. This is still less than a complete 12-month set of completed M30 candles at the end boundary.

| Run | Closed trades | Open at end | Net PnL after commission | PF | Max DD | Expectancy |
|---|---:|---:|---:|---:|---:|---:|
| Candidate | 11 | 0 listed | +RUB 25,426.57 | 2.557 | RUB 13,431.59 (12.50%) | RUB 2,311.51 |
| Baseline | 168 | 1 | +RUB 7,610.15 including mark-to-market | 1.079 | RUB 37,274.56 (33.94%) | RUB 45.15 |

The candidate clears the approximate trade-count, net PnL, and PF checks but fails the fixed Max DD <=10% gate at 12.50%. The baseline report explicitly marks one open position; it is not counted as a completed trade. The candidate report lists no open position and all 11 entries are closed. Therefore the candidate is rejected for literal adoption. This validation slice is not a full independent 12-month test and cannot certify any ticker parameters.

## Reproduction artifacts
- `baseline.json`, `original` baseline control: `control/IVAT_fvg_imbalance_Minutes30_20261113_231949.md` and paired CSVs.
- Stage 1: `stage-1-entry.json`; reports under `stage-1/`.
- Stage 2: `stage-2-candidate-1-stoch.json`, `stage-2-candidate-1-no-stoch.json`; reports under their correspondingly named directories.
- Stage 3: `stage-3-refinement.json`; report under `stage-3/`.
- Sensitivities: `sensitivity-stochperiod.json`, `sensitivity-htftrendema.json`, `sensitivity-maxwaitbars.json`; reports under correspondingly named directories.
- Locked parameters and pre-validation decision: `final-candidate.json`, `locked-candidate.md`.
- Approximate validation reports and CSVs: `validation-candidate/`, `validation-baseline/`.
