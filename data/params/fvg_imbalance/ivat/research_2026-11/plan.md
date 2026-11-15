# IVAT FVG imbalance exploratory recalibration plan (2026-11-13)

## Fixed limits and protocol
- Source snapshot: existing Tinkoff candle cache at `data/candles`; IVAT Minutes30 2024-06-04 15:00 MSK through 2026-10-15 13:00 MSK (29,179 bars); Day1 2024-06-04 03:00 MSK through 2026-10-27 03:00 MSK (733 bars). No `-refresh`; inspect actual report boundaries after each command and stop if CLI changes/extends this snapshot.
- Approximate training window requested: `[2023-11-13 23:19, 2025-11-13 23:19)` via `-months 36 -holdout-months 12`. Because source starts 2024-06-04, effective training data is only about 17 months. Analyze descriptive blocks `[2024-06-04, 2025-06-04)` and `[2025-06-04, 2025-11-13)`; these are unequal and cannot satisfy the standard two-year gate.
- Approximate validation window: cached M30 from 2025-11-13 to 2026-10-15 (under 12 months). Do not open/report its strategy performance until one complete candidate is locked. This period is only an approximate validation and is not a full independent 12-month test.
- Engine uses completed candles, prior completed daily candles for ATR(14) with 28-bar warmup; record reports' actual loaded boundaries. EMA warmup is its daily period. No strategy-mechanics changes.
- All runs: initial cash RUB 100,000; fraction 1.0; risk-pct 0; commission 0.0005 per turnover; next M30 open; CPU 2; same cache and execution settings.
- Fixed exploratory ranking metric: expectancy. Selection gates, applied unchanged to every stage: >=20 closed training trades, positive net PnL after commission, PF >1, Max DD <=10% of running equity peak. Additionally inspect trade counts/PnL in the two unequal descriptive blocks, but do not claim the skill's two-full-year minimum was met. Candidate must be fully run on training before lock. If no candidate meets these gates or report boundaries drift, stop and leave the ticker literal unchanged. On the approximate validation window, the precommitted descriptive pass gates are >=10 closed trades, positive Net PnL, PF >1, Max DD <=10%; because the cache covers under 12 months, a pass still does not qualify the ticker literal.
- Standard criteria are not relaxed after seeing results. A selected candidate is saved separately and checked once on the approximate validation slice against baseline; validation cannot promote it to a verified ticker literal.

## Stages and budget
- Baseline is the skill research baseline (`baseline.json`); no IVAT ticker literal currently exists. Existing baseline control is preserved in the same directory.
- Stage 1 copies the skill template into `stage-1-entry.json`, sweeps 350 combinations (7 MinGapATR × 5 StochPeriod × 5 HTFTrendEMA × 2 entry-floor modes; the skill reference comment incorrectly says 120), all nine fields explicit, `-combos 0`.
- Preserve up to two distinct eligible entry candidates to bound compute. For each, run both stage-2 exit templates as separate flat grids, all nine fields explicit and pinned to that candidate except StopATR/TakeProfitATR and branch. Preserve all combinations.
- Stage 3: locally refine the selected exit branch per reference instructions; run sensitivity grids separately. Save candidate and decision before touching the approximate validation window.
- Approximate validation: run only locked candidate and baseline once each. Keep every grid/report. Never use this period for another candidate search.

## Execution addendum (recorded after the locked runs)
- Stage 1, stage 2, stage 3, and sensitivities used the training cutoff ending 2025-11-13; their report windows differed only by CLI start minute and used the cached history available before the reserved period was opened.
- The first locked-validation command extended the cached M30 tail from 2026-10-15 to the last completed M30 candle at 2026-11-13 23:30. The subsequent baseline report used the same requested date window and exact same last equity timestamp (2026-11-13 20:30Z). No further candidate search was run after this validation.
- Validation report boundary for both runs: 2025-11-14 00:02–2026-11-14 00:02. The candle series ends one M30 slot before the report end, so this remains a slightly incomplete 12-month slice.
- The candidate failed the precommitted Max DD <=10% validation gate (12.50%). No literal update is allowed by the locked protocol.
