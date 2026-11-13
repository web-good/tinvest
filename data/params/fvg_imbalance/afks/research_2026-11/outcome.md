# AFKS FVG imbalance calibration outcome

## Decision

**Do not update the AFKS literal.** The candidate was locked before the final year, but failed the pre-registered final quality limits (positive net PnL, PF > 1, Max DD <=10%). The final year was run once for the locked candidate and once for the original parameters; no alternate candidate was tried on it.

## Data and execution

- Source: Tinkoff Invest gRPC through `cmd/backtest` and `CandleProvider`; completed candles only.
- Training reports use `[2023-11-12 15:41..15:47 MSK, 2025-11-12 15:41..15:47 MSK)` due per-command wall-clock time. Training M30 and daily data are the same cached candle series in all runs. Daily warm-up requested from 12 months before each window; FVG ATR(14) requires 28 daily bars; EMA filters were off in selected candidate.
- Final candidate report window: `2025-11-12 15:48 — 2026-11-12 15:48 MSK`; original report window: `2025-11-12 15:49 — 2026-11-12 15:49 MSK`. Both include the same completed M30 bars (cache after refresh: 2025-11-12 15:30 through 2026-11-12 15:00 MSK; 13,466 bars) and completed Day1 bars through 2026-11-11 (644 bars). The API refresh filled the stale cache before the candidate run; original run reused it. Daily history begins 2024-11-12 for final evaluation, providing >28 bars of ATR warm-up.
- Execution/costs: next M30 bar open; initial cash 100,000 RUB, fraction 1.0, risk-pct 0, commission 0.0005, share lot from API.
- Drawdown in quality criteria is percentage of equity curve peak-to-trough; engine reports both RUB and percent.

## Locked candidate and training

Full params are in `final-candidate.json`:

`ATRPeriod=14, MinGapATR=0.3, MaxWaitBars=30, StochPeriod=10, UseStochExit=0, StopATR=0.5, TakeProfitATR=1, HTFTrendEMA=0, RequireEntryAboveFVGLower=1`.

Branch: no stochastic exit, with 0.5 daily ATR stop and 1 daily ATR target. The eligible stage-2 no-stoch result improved the best eligible stoch result by >5% on expectancy, so both branches were not carried into stage 3. The first stage-3 leader (MinGapATR 0.25) exceeded the DD cap at 10.83%. The bounded rerun produced the same locked value at rank 2; rank 1 again exceeded the DD cap (10.93%).

For the locked candidate on the full 24-month training window:

- First year: 12 completed trades.
- Second year: 10 completed trades.
- Total: 22 closed trades; no open position at training end.
- Net PnL +33,899.67 RUB; PF 3.138; Max DD 9,901.67 RUB (9.20%); expectancy 1,541.

These training results pass all pre-registered thresholds, but are not evidence of the final year.

## Reserved final-year comparison

| Set | Closed trades | Open at end | Net PnL | PF | Max DD | Result |
|---|---:|---:|---:|---:|---:|---|
| Locked candidate | 19 | 1 | -11,550.37 RUB (-11.55%) | 0.545 | 11,950.71 RUB (11.90%) | Fail: fewer than 10 closed trades? No, trade-count threshold passes; PnL, PF and DD limits fail |
| Original AFKS set | 205 | 1 | -1,815.66 RUB (-1.82%) | 0.989 | 27,378.22 RUB (23.71%) | Fail: PnL, PF and DD limits fail |

The candidate lost more net value than the original on this period, while its drawdown was lower. Its one open end-of-window position is marked to market in the report and does not count as a completed trade. The engine report does not give an open-position duration or independent exit reason for that unclosed position.

## Files

- Plan and thresholds: `plan.md`
- Baseline template / original literal snapshot: `baseline.json`, `original.json`
- Entry grid and results: `stage-1-entry.json`, `../../../../../reports/fvg_imbalance/afks/research_2026-11/stage-1/`
- Stage 2 grids and branch reports: `stage-2-candidate-*.json`, matching report directories under `reports/fvg_imbalance/afks/research_2026-11/`
- Local refinement and sensitivity: `stage-3-refinement.json`, `stage-3-rerun-stoch10.json`, `sensitivity-*.json`, matching report directories
- Locked full params: `final-candidate.json`
- Final comparison reports and trade/equity CSVs: `reports/fvg_imbalance/afks/research_2026-11/final-candidate/`, `reports/fvg_imbalance/afks/research_2026-11/final-original/`

AFKS strategy literal remains the original `core.DefaultParams()` value. No Go source or literal was changed for this calibration.
