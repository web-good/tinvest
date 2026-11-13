# ALRS calibration outcome

## Decision

The locked candidate was written to `strategy/alrs/alrs.go` after the user's follow-up request. The reserved-year reports cannot validate it because the M30 source snapshot has a 14-day hole from 2026-10-14 through 2026-10-26. The API request for the 2026-10-13 .. 2026-10-27 chunk failed with `DeadlineExceeded`; `CandleProvider` still returned the rest of the cache and the report did not flag the internal gap. Treat the literal as the user's requested research selection; out-of-sample validation remains unestablished.

The locked candidate and all training evidence remain available for a proper future validation after the missing candles are restored. The code comment beside the literal records the validation limitation.

## Locked candidate

`final-candidate.json`:

```json
{
  "ATRPeriod": 14,
  "MinGapATR": 0.25,
  "MaxWaitBars": 45,
  "StochPeriod": 14,
  "UseStochExit": 1,
  "StopATR": 2,
  "TakeProfitATR": 0,
  "HTFTrendEMA": 0,
  "RequireEntryAboveFVGLower": 1
}
```

Selected branch: stochastic exit enabled. Aligned training result: 33 completed trades (20 / 13 by year), Net PnL 20,699.07 RUB, PF 3.252, Max DD 4.86%, no open position. It passed the precommitted training gates.

## Reserved-year attempt (invalid for acceptance)

| Run | Trades | Net PnL | PF | Max DD | Open position |
|---|---:|---:|---:|---:|---|
| Locked candidate | 17 | 20,988.21 RUB | 4.199 | 7.77% | No |
| Original parameters | 17 | 20,929.86 RUB | 4.198 | 6.69% | No |

These are reported diagnostics only; the 14-day data gap makes the annual comparison incomplete. No final-year pass/fail is claimed.

## Reports

- Aligned training, controls, grids, and sensitivity reports: `reports/fvg_imbalance/alrs/research_2026-11/aligned/`
- Initial runs from before the common 13:30 M30 snapshot: `reports/fvg_imbalance/alrs/research_2026-11/` (not used for selection)
- Locked candidate: `final-candidate.json`; decision record: `locked-candidate.md`
- Verification: `go test ./internal/service/trading_strategy/fvg_imbalance/strategy/... ./internal/service/backtest ./cmd/backtest` passed. The ALRS registry already maps `ALRS` to `alrs.Params`, so the default run now resolves to these values without `-params`.
