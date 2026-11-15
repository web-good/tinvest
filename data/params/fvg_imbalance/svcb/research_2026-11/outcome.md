# SVCB FVG imbalance calibration outcome

## Decision

Research candidate saved to `candidate-research.json`. **Not independently verified; per the user’s follow-up request, the research candidate is now recorded in `strategy/svcb/svcb.go` with its validation limitation documented.** A valid independent final year is unavailable. The default split at the 2026-11-13 run date requires training from 2023-11-13, before the local M30 cache starts on 2023-12-15; it also requires final candles through 2026-11-13, while the cache ends 2026-11-03. The chosen research candidate was fit through 2026-10-13, so the desired final window 2025-11-13 through 2026-11-13 overlaps its training data. The remaining post-training tail is only about three weeks. No final-year test was run, and no previously viewed period is called OOS.

## Data and execution

- Source: Tinkoff Invest candles cached at `data/candles/SVCB_Minutes30.json` and `data/candles/SVCB_Day1.json`; the cache covers 2023-12-15 through M30 2026-11-03 / daily 2026-11-02.
- Train runs used the common CLI window 2024-10-13 22:59 MSK to 2026-10-13 22:59 MSK (24 months; reports vary by a few minutes because CLI computes `now` per run; the selected M30 candle set and end date are the same).
- The calibration reports count `RecentTrades` from the final 12 months. For these runs that matches the second annual block; trade journals were also checked by entry date for the annual counts shown below.
- Some CLI runs logged failed gRPC requests while trying to extend the cached daily lead-in back to 2023-10; the requested M30 backtests completed from available cached candles. Daily ATR has more than its required 28 completed weekday bars before the first training-period setup. HTF EMA is disabled in the selected candidate.
- Parameters and execution: M30 completed candles; daily ATR(14), Wilder calculation; stochastic %K period 9; starting cash RUB 100,000; fraction 1; commission 0.0005 per turnover; fixed-fraction sizing (`risk-pct=0`); instrument lot from API; entry at next M30 open; no mechanical changes. Max DD limit was 10% of equity (`MaxDrawdownPct`, a fraction in the engine; reports render percent).
- Fixed selection rules: expectancy; positive net PnL after commission; PF > 1; Max DD <= 10%; at least 20 closed trades overall and at least 10 in each contiguous training year; at least 10 closed trades in final test. Open positions are excluded from closed-trade counts.

## Baseline and original control

| Run | Closed trades (year 1 / year 2) | Net PnL | PF | Max DD | Outcome |
|---|---:|---:|---:|---:|---|
| Skill baseline (`MinGapATR=0`) | 197 / 193 (390 total) | -RUB 23,074.14 | 0.889 | 38.10% | Fails quality |
| Original SVCB literal (`MinGapATR=0.25`, default remaining values) | 16 / 13 (29 total) | -RUB 1,028.55 | 0.948 | 9.34% | Fails PnL/PF |

Reports: `baseline-train/` and `control-train/`.

## Stage 1: entry search

The reference grid expands to 350 combinations (7 MinGapATR × 5 StochPeriod × 5 HTFTrendEMA × 2 entry-floor flags). It ran with the baseline exit values. Ranks 1 and 2 did not pass the second-year minimum (8 trades each). Ranks 3, 4, and 6 passed both annual blocks and the fixed quality constraints; annual counts were checked from complete trade journals.

| Stage-1 rank | MinGapATR | StochPeriod | Entry floor | Trades Y1 / Y2 | Net PnL | PF | Max DD | Expectancy |
|---|---:|---:|---:|---:|---:|---:|---:|---:|
| 3 | 0.25 | 9 | 0 | 22 / 14 | RUB 12,881.06 | 1.830 | 5.67% | RUB 357.8 |
| 4 | 0.25 | 10 | 0 | 21 / 14 | RUB 10,831.76 | 1.643 | 6.55% | RUB 309.5 |
| 6 | 0.25 | 9 | 1 | 15 / 10 | RUB 4,685.69 | 1.463 | 4.86% | RUB 187.4 |

All other fields remained at baseline values. Grids and full reports: `stage-1-entry.json`, `stage-1-entry/`, `stage-1-rank3.json` through `stage-1-rank6-train/`.

## Stage 2: exits

Separate full grids were run for both exit branches for each accepted stage-1 candidate. On the best allowable stoch-exit result (rank-3 entry, `StopATR=1.5`, `TakeProfitATR=0`), the two training years had 22 and 14 closed trades; Net PnL RUB 14,497.14, PF 2.035, Max DD 5.27%, expectancy RUB 402.7. The branch's nearby stop/target variants produced the same trades/metrics where the stochastic exit closed positions first. The adjacent StochPeriod=10 candidate was also positive and passed both annual gates (21/14, Net PnL RUB 12,504.98, PF 1.819, DD 5.95%).

The no-stochastic-exit branch did not yield a candidate satisfying both the annual gate and quality constraints. For the rank-3 entry its best expectancy configuration had only 8 second-year trades and ended with an open position; another result with 10 recent trades had PF 1.002, Max DD 26.14%, and an open position at the end, so it fails the 10% drawdown limit. The open position is not treated as a completed trade; see `no-stoch-rank2-train/`.

Reports and grids are in `stage-2-rank{3,4,6}-{stoch,no-stoch}/` with matching `.json` files. All combinations were retained.

## Stage 3 and sensitivity

The 48-combination local grid used MinGapATR `[0.1, 0.25, 0.3]`, StopATR `[1, 1.2, 1.5, 1.7]`, TakeProfitATR `[0, 1]`, and entry floor `[0, 1]`, keeping the selected branch and other fields fixed. The numerical leader (`MinGapATR=0.3`) had only 8 trades in the second year, so it was rejected. The best candidate passing both year gates was the already-seen `MinGapATR=0.25`, `StopATR=1.5`, `TakeProfitATR=0`, entry floor off; total 36 trades (22/14), expectancy RUB 402.7, Net PnL RUB 14,497.14, PF 2.035, DD 5.27%.

One-field checks:

- `StochPeriod=10`: 35 trades, 21/14, expectancy RUB 357.3; positive quality and the year gate passed. `StochPeriod=11`: negative expectancy and Net PnL.
- `HTFTrendEMA=50,100,120,200`: 10–13 trades total and at most 1 in the recent year; none passed admission.
- `MaxWaitBars=15`: expectancy RUB 661.5 but only 9 recent-year trades, so rejected. `MaxWaitBars=45`: 39 trades (24/15), expectancy RUB 368.8, less than the selected candidate.

No sensitivity result both improved on the selected eligible candidate and justified a repeat local grid. Detailed grids/reports are under `stage-3-refinement/` and `sensitivity-*/`.

## Research candidate (not a verified deployment value)

```json
{
  "ATRPeriod": 14,
  "MinGapATR": 0.25,
  "MaxWaitBars": 30,
  "StochPeriod": 9,
  "UseStochExit": 1,
  "StopATR": 1.5,
  "TakeProfitATR": 0,
  "HTFTrendEMA": 0,
  "RequireEntryAboveFVGLower": 0
}
```

Selected exit branch: stochastic exit enabled. The full 24-month research trade log had 36 closed trades (22 in `[2024-10-13, 2025-10-13)`, 14 in `[2025-10-13, 2026-10-13)`), no open position at the end, Net PnL RUB 14,497.14 after the configured commission, PF 2.035, Max DD 5.27%, and expectancy RUB 402.7. These are training-period observations only. No `_comment` calibration record was added to mechanics documentation. The user explicitly requested recording these values in the ticker despite the missing independent test; the Go comment marks the candidate as unverified.
