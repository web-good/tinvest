# YDEX M30 FVG imbalance research outcome

The reserved final window was opened after locking `final-candidate.json`. The candidate failed the pre-set minimum of 10 closed trades, so the YDEX literal remains `core.DefaultParams()` and was not modified.

## Fixed protocol and coverage

All runs used Tinkoff Invest candle data cached at `data/candles`, completed M30/daily candles, 100,000 RUB initial cash, fraction 1, risk-pct 0, 0.0005 commission, YDEX lot, and next M30 open execution. The provider topped up the cache during final runs. At completion, M30 cache coverage was 2023-10-07 17:30 through 2026-11-12 12:30 MSK; daily coverage was 2021-08-12 through 2026-11-11. ATRPeriod=14 needs 28 daily candles; CLI provides about a year of lead-in, sufficient for ATR and candidate EMA warm-up. Training report period was 2023-11-12 12:59 — 2025-11-12 12:59. Final reports used 2025-11-12 13:00 — 2026-11-12 13:00.

## Training and branch decision

The primary metric and quality limits were fixed in `plan.md`: expectancy, Net PnL > 0, PF > 1, MaxDrawdownPct <= 10%, plus >=20 training trades and >=10 in each exact 12-month block. Stage 1 evaluated 350 combinations (the actual template arrays multiply to 350; skill prose incorrectly states 120). Four representative seeds passed annual trade counts: ranks 1, 2, 4, 6 had respectively 10/15, 10/10, 12/18, and 18/19 closed trades by entry date.

The chosen full candidate is `final-candidate.json`: ATRPeriod=14, MinGapATR=0.3, MaxWaitBars=30, StochPeriod=14, UseStochExit=1, StopATR=1.2, TakeProfitATR=1.5, HTFTrendEMA=0, RequireEntryAboveFVGLower=1. Stage 3 full joint refinement and single-field sensitivities were run. The locked training report has 25 trades (10/15), Net PnL +17,794.31 RUB, PF 3.264, expectancy 711.77 RUB/trade, Max DD 6,467.09 RUB / 6.08%, no open position. This passes the training quality and sample gates.

Both exit branches were tested on all copied grids for all four seeds. All 64 no-stoch outcomes had absolute Max DD >=23,952.91 RUB. The grid summary reports absolute drawdown but not `MaxDrawdownPct`, so it does not establish the exact percent gate for every row; no no-stoch outcome was advanced to stage 3. The chosen stoch candidate passed the displayed 6.08% training DD gate. StochPeriod=11 and HTFTrendEMA=200 reduced expectancy. MaxWaitBars=45 improved Net PnL and PF but reduced the declared primary metric, so the locked candidate retained 30.

The actual original literal is `original.json` (core defaults), not `baseline.json`. Its training control had 37 trades (18/19), Net PnL +18,818.01 RUB, PF 2.409, Max DD 6,371.46 RUB / 5.87%, no open position. The separate research baseline `baseline.json` had 338 trades, Net PnL -26,132.95 RUB, PF 0.860; it was not used as the original-literal control.

## Reserved final check

The final-year comparison was run once per set after lock, with clean positions and indicator warm-up:

| Set | Closed trades | Net PnL | PF | Max DD | Open position | Result |
|---|---:|---:|---:|---:|---|---|
| Original literal | 21 | +6,823.96 RUB | 3.053 | 3,581.50 RUB / 3.36% | No | Meets sample and quality limits |
| Locked candidate | 7 | +496.83 RUB | 1.315 | 3,604.89 RUB / 3.55% | No | Fails >=10 trade sample limit |

The candidate also produced substantially lower final-year Net PnL and PF than the original. Because it failed the sample limit, the ticker literal remains unchanged. The final period was not reused to select another candidate.

## Artifacts

- Plan: `plan.md`; pre-final lock: `locked-candidate.md`.
- Full params: `baseline.json`, `original.json`, `final-candidate.json`.
- Stage grids and reports: `reports/YDEX/fvg_imbalance/research_2026-11/`.
- The CLI's flat grid parser rejects `_comment` keys, so explanatory notes are preserved in this outcome/plan and grid file names instead of adding invalid JSON keys.
