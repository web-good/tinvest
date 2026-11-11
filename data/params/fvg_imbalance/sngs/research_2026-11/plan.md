# SNGS FVG calibration plan (2026-11-11)

## Locked criteria and windows
- Train window: `[2023-11-11, 2025-11-11)`; annual blocks `[2023-11-11, 2024-11-11)` and `[2024-11-11, 2025-11-11)`.
- Reserved final window: `[2025-11-11, 2026-11-11)`; do not inspect until one full candidate is locked after stage 3.
- Source: Tinkoff Invest candles via `data/candles`; use completed M30, Day1 and H4 candles only. At plan time local SNGS cache covers M30 2023-08-04..2026-10-13, Day1 2021-08-12..2026-10-27, H4 2023-10-13..2026-10-28; CLI may complete cache through the requested end. Actual loaded bounds to be recorded from reports.
- Daily ATR is Wilder, period 14, with 28 daily bars for seed/smoothing; daily lead-in is one year. H4 EMA lead-in is one year and HTF periods are swept only as specified.
- Execution follows existing engine: completed M30 signal, next M30 open fill; daily ATR stop/target from actual fill; one position, no mechanics changes.
- Fixed costs and sizing: commission 0.0005 per turnover, initial cash 100000 RUB, fraction 1.0, risk-pct 0; ticker lot from instrument metadata.
- Ranking metric: expectancy. Candidate constraints: positive net PnL after commission, PF > 1, MaxDrawdownPct <= 0.10. Engine MaxDrawdownPct is a fraction of equity (report displays percent); MaxDrawdown absolute is currency.
- Eligibility at all three training stages and both exit branches: >=20 completed closed trades total and >=10 by entry date in each annual block. Open position is not a completed trade. For final year require >=10 completed trades and same quality constraints.

## Planned stages and budget
1. Entry grid: 120 combinations in `stage-1-entry.json`, with SNGS literal entry MinGapATR=0.25 added as required. Baseline control and original-literal control run separately. Preserve full ranking (`-combos 0`).
2. For up to four eligible, stable entry candidates: independent full StopATR x TakeProfitATR grids for UseStochExit=1 and 0, all other fields explicitly fixed to the entry candidate. Inspect ranked results and candidate trade tables for both annual blocks. Budget up to 8 branch grids.
3. For selected branch/candidate: local joint refinement per skill; one-field sensitivity sweeps for StochPeriod, HTFTrendEMA, MaxWaitBars only if stage-3 evidence warrants. Full joint rerun if sensitivity changes finalist. Preserve every grid/report.
4. Lock one candidate before running the reserved final year. Run original and locked candidate once each on same final window.

No results have been viewed as of this plan. If candle coverage or API access cannot support these exact windows, stop with saved evidence and do not label a partial period as 24+12 validation. If compute budget limits exhaustive eligible-candidate review, state that limitation.
