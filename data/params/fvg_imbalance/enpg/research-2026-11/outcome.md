# ENPG FVG imbalance calibration outcome

## Decision

The locked candidate failed the one reserved-year quality check. After that result was reported, the user explicitly requested writing the best training candidate into ENPG; `internal/service/trading_strategy/fvg_imbalance/strategy/enpg/enpg.go` now returns that candidate. The source comment records the failed holdout metrics. Treat it as a training-selected parameter set, not as a verified calibration.

Locked candidate, preserved in `stage-3-candidate.json`:

```json
{
  "ATRPeriod": 14,
  "MinGapATR": 0.3,
  "MaxWaitBars": 30,
  "StochPeriod": 10,
  "UseStochExit": 1,
  "StopATR": 1,
  "TakeProfitATR": 1,
  "HTFTrendEMA": 0,
  "RequireEntryAboveFVGLower": 1
}
```

The candidate passed training gates, but failed the fixed final quality gates: its reserved-year Net PnL was negative and PF was below 1. No alternate candidate was tried on the same final year.

## Windows, data, and execution

- Training: 2023-11-13 23:18/23:22 to 2025-11-13 23:18/23:22 MSK, from CLI runs at their recorded start times. The fixed year split was 2024-11-13 at the same time of day.
- Reserved check: 2025-11-13 23:23 to 2026-11-13 23:23 MSK. Candidate and original control reports have identical boundaries.
- Candle source: Tinkoff Invest gRPC, cached under `data/candles`; cached M30 was topped up to 2026-11-13 22:30 MSK and daily candles to 2026-11-12. The backtest uses completed candles. It warms daily data from one year before the requested range; 14-period Wilder ATR needs 28 completed daily bars, and daily EMA warmup is adequate.
- Execution held fixed: 100,000 RUB initial cash, 100% fraction, 0.0005 commission, risk fraction 0, instrument lot from metadata, entry at the next M30 open. Candidate and control used the same run settings.
- Engine drawdown output is both absolute RUB and a percentage of peak equity; the fixed gate uses `MaxDrawdownPct <= 10%`.

## Calibration evidence

- Stage 1 actually contained 350 combinations (the supplied template is 7 x 5 x 5 x 2), despite the skill text saying 120. Top eligible candidates included four close `MinGapATR=0.3`, `StochPeriod=10/11` combinations. The winner and runner-up's full trade reports showed 14/16 and 13/15 trades by training year respectively.
- Stage 2 evaluated both exit branches independently for each of four entry candidates. The strongest audited stochastic candidate scored expectancy 832.2, PF 4.698, Net PnL +23,302.80 RUB, DD 4.43%, with 13/15 year trades. Strong no-stoch candidates had high expectancy (including 914.8) but failed the fixed DD gate: the best audited example had DD 30.66% (44,254.06 RUB); another had 30.29% (41,709.04 RUB). The stoch branch was retained.
- Stage 3's 54-combination joint grid ranked the locked floor-enabled candidate first. The floor-disabled near-tie was lower in expectancy (829.6 vs 832.2). The locked candidate's training trade PnL by entry year was +4,132.51 / +19,170.28 RUB.
- Separate sensitivity grids showed StochPeriod 11 below the locked candidate; EMA 50/100 had 15/13 total trades, and MaxWaitBars 45 had 17 total trades. They did not produce a stronger eligible full candidate, so no extra joint rerun was triggered.

## Final reserved-year comparison

| Set | Closed trades | Net PnL | PF | Max DD | Open position |
|---|---:|---:|---:|---:|---|
| Locked candidate | 16 | -7,146.66 RUB | 0.397 | 10,236.74 RUB (9.93%) | None |
| Original ENPG params | 20 | -4,212.19 RUB | 0.739 | 11,056.47 RUB (10.55%) | None |

The candidate met the final minimum of 10 closed trades and stayed within the drawdown limit, but failed both positive Net PnL and PF > 1. The original also failed the fixed quality gates. The candidate remains rejected by the predeclared final quality gates despite being written to the ticker at the user’s explicit request.

## Reproduction artifacts

All stage grids, full reports, trade CSVs, equity CSVs, and parameter JSONs are retained in this directory. The principal files are `plan.md`, `original.json`, `stage-1-entry.json`, `stage-3-refinement.json`, `stage-3-candidate.json`, `locked-candidate.md`, and the `stage-1`, `stage-2-*`, `stage-3`, `sensitivity-*`, `final-candidate`, and `final-original` report directories.
