# SNGS repeated FVG calibration, 2026-11-15

**Outcome: preserve the existing literal.** The research winner exactly matches the SNGS literal. It passes training criteria but no new independent final validation is claimed: the next year was already exposed by the November 11 calibration. Local cache also ends before the nominal November 15 final endpoint. No live strategy or common defaults changed.

Full chosen numeric parameters: [research-candidate.json](research-candidate.json). Entry: ATRPeriod=14, MinGapATR=0.1, MaxWaitBars=30, StochPeriod=21, HTFTrendEMA=120, RequireEntryAboveFVGLower=1. Exit: UseStochExit=1, StopATR=1.7, TakeProfitATR=0. Ranking: expectancy. Limits: net>0, PF>1, engine MaxDrawdownPct<=0.10, 10 completed trades per fixed annual training block / 20 total. Cash 100000 RUB, fraction1, lot100, risk0, commission0.0005 each turnover, no modeled additional slippage. Existing engine unchanged.

## Fixed training replay

Window [2023-11-15T00:00:00+03:00,2025-11-15T00:00:00+03:00), split 2024-11-15T00:00:00+03:00. 23658 M30 bars, actual first 2023-11-15 09:30+03, last 2025-11-14 23:30+03. Daily lead-in [2022-11-15,2025-11-15): 820 bars; last daily 2025-11-14 03:00+03. ATR14 requires 28 completed weekday bars; daily EMA uses completed daily closes. No H4 input is consumed by the current FVG core. M30 startup lookback follows CLI; no extra pre-window M30 candles.

| Set | Closed trades year1/year2 | Net PnL RUB | PF | Max DD | Open at end |
|---|---|---:|---:|---:|---|
| Incumbent / research winner | 11 / 12 | +9455.99 | 2.5013 | 3.318% | No |
| Research baseline | 167 / 213 | -36228.86 | 0.7720 | 39.606% | No |

Winner expectancy 411.1301 RUB per closed trade. Annual entry-cohort closed PnL: +924.99 / +8531.00 RUB; PF 1.2526 / 4.2348. These are cohorts of one continuous training replay, not independent reset annual runs. No training improvement over current literal: all parameters, metrics and trades coincide. Ordinary registry replay without params exactly matches the explicit original parameter replay.

## Search and sensitivity

350 stage-1 combinations: 3 eligible; retained 2 adjacent EMA filter regimes, third near-zero-expectancy isolated eligible point rejected with neighboring evidence. See [stage-1-selection.md](stage-1-selection.md).

80 stage-2 combinations: two inputs, each 24 stochastic and 16 no-stochastic exits. Both no-stochastic grids have zero eligible rows; stochastic EMA120 wins. See [stage-2-selection.md](stage-2-selection.md).

36 stage-3 joint combinations: 6 eligible. Stop1.7 and TP0 retain the incumbent; target1.5/2 had tied stage-2 results. Neighbor Stop1.5 remains eligible with expectancy160.03; Stop2 ties the incumbent expectancy411.13. The tie retains the existing Stop1.7. Gap0 and .25 fail eligibility in the local grid, so the result remains specific to gap.1, with limited trade counts. Sensitivity: Stoch14 expectancy77.02 versus Stoch21 411.13; EMA100 347.24 versus EMA120 411.13; EMA200 411.25 fails counts (15/8). MaxWait15 has 0/1 trades, ineligible; MaxWait45 has 14/17 trades and positive PnL but DD12.58%, ineligible. No better eligible sensitivity set, so no second joint grid. This does not demonstrate a global optimum or broad stability.

474 grid combinations + 2 controls + 1 registry replay = 477 full training replays, below budget750. Every row saved, no keepTop cutoff. All 476 non-registry replay trade CSVs independently recounted for annual counts, total count and closed PnL; registry result equals original exactly. Snapshot SHA256 unchanged.

## Stop-only branch (UseStochExit0, TakeProfitATR0)

No forced liquidation. Open PnL is marked to last close and excluded from closed trade counts and expectancy; no positions remained open in these eight rows. Maximum calendar holding duration is calculated from actual closed trade timestamps (hours):

| Grid | StopATR | Closed trades | Open at end | Longest holding hours |
|---|---:|---:|---|---:|
| stage-2-entry-1-no-stoch | 0.5 | 18 | No | 1277.5 |
| stage-2-entry-1-no-stoch | 1 | 13 | No | 1321.5 |
| stage-2-entry-1-no-stoch | 1.5 | 9 | No | 1337.5 |
| stage-2-entry-1-no-stoch | 2 | 9 | No | 1347.5 |
| stage-2-entry-2-no-stoch | 0.5 | 18 | No | 1277.5 |
| stage-2-entry-2-no-stoch | 1 | 12 | No | 1321.5 |
| stage-2-entry-2-no-stoch | 1.5 | 9 | No | 1337.5 |
| stage-2-entry-2-no-stoch | 2 | 8 | No | 1347.5 |

## Previously observed final year — historical audit only

These values come from the **November 11 saved reports**, not a new independent final test. Report period 2025-11-11 14:56 to 2026-11-11 14:56, distinct from this run's nominal final window. CSV closed counts and net sums rechecked.

| Historical set | Closed trades | Net PnL RUB | PF | Max DD | Open at end |
|---|---:|---:|---:|---:|---|
| Current literal (then locked candidate) | 13 | +8053.51 | 4.425 | 2.54% | No |
| Pre-November-11 literal (not today's original) | 24 | +2806.12 | 1.284 | 6.93% | No |

The old selected set traded less (13 vs24). Historical metrics do not establish improvement from the current repeated selection, which leaves the parameters unchanged. No fresh 12-month test count is available. Average R is not supplied by the engine and is omitted.

## Artifacts and reproduction

- [plan.md](plan.md), [snapshot-meta.json](snapshot-meta.json), all stage/sensitivity grids in this directory.
- [all-results.csv](all-results.csv) preserves all non-registry metrics, full parameters, fixed annual counts and eligibility in versioned data.
- [locked-candidate.md](locked-candidate.md), [research-candidate.json](research-candidate.json).
- Full replay results, trade CSVs, controls and snapshot files: [reports](../../../../../reports/fvg_imbalance/sngs/research_2026-11-15/). Reports and candle snapshots follow repository ignored-artifact convention; metadata/hash are tracked with this research directory.
- [Historical final candidate report](../../../../../reports/fvg_imbalance/sngs/research_2026-11/final-candidate/SNGS_fvg_imbalance_Minutes30_20261111_145641.md).

From repository root, run the production-engine harness against saved snapshots:

```sh
go run data/params/fvg_imbalance/sngs/research_2026-11-15/replay.go -grid data/params/fvg_imbalance/sngs/research_2026-11-15/stage-1-entry.json -out reports/fvg_imbalance/sngs/research_2026-11-15/reproduced-stage-1
```

For fixed original, use -params with original.json; omitting -params selects the registry. Harness uses production domain.Run/Compute and registry Binding; it directly enumerates full grids to preserve annual counts and open position metadata, rather than relying on CLI RecentTrades or moving time.Now. Completed bars are inherited from CandleProvider's completed-only cache; snapshots retain provider provenance but lack an independent IsComplete field. No new market-data fetch was performed.

Validation: `go test ./internal/service/trading_strategy/fvg_imbalance/strategy/... ./internal/service/backtest ./cmd/backtest` passed. Research harness gofmt and build passed. No production files modified.
