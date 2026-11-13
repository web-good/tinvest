# SPBE FVG imbalance calibration outcome (2026-11-12)

## Decision
No candidate cleared the frozen Max DD <= 10% gate, and the cached M30 history did not cover the complete reserved final year through 2026-11-12. At the user’s explicit request on 2026-11-12, the best Stage 3 research candidate was then copied into the SPBE literal despite these failed verification gates. This is a research setting, not a verified calibration. No final-year candidate or control run was made; that period was not used to select parameters.

## Data and common run settings
- CLI training window: 2023-11-12 22:59/23:00 through 2025-11-12 22:59/23:04 MSK, depending on command start time. This is a 24-month train window (`-months 36 -holdout-months 12`); the annual split is 2023-11-12 to 2024-11-12 and 2024-11-12 to 2025-11-12.
- The common window timestamps differ by a few minutes because CLI `now` is read separately on each command. All use the same underlying candle snapshot. The training boundaries therefore match by date, but are not a single frozen timestamp.
- Cached completed candles at inspection: M30 2023-09-04 21:30 to 2026-10-13 21:30 MSK (30,358); Day1 2022-10-28 to 2026-10-27 MSK (1,199). The reserved test ends 2026-11-12 and is not fully present in M30. Daily lead-in is one year before each window; M30 stochastic and daily ATR/EMA use engine/core warm-up behavior.
- Tinkoff Invest MarketData API/CandleProvider cache; cash RUB 100,000; fraction 1.0; commission 0.0005 per turnover; risk-pct 0; ordinary engine fills. Completed candles only.
- Expectancy was the ranking metric; gates frozen before results: positive Net PnL after costs, PF > 1, Max DD <= 10% (percentage equity drawdown, not absolute RUB), at least 20 closed trades over training and at least 10 in each annual block. Final test gate was at least 10 closed trades plus the same quality gates.

## Results and rationale
- Original literal: `original.json`; 27 closed trades (16 / 11 by training year), Net PnL +18,723.02 RUB, PF 1.761, Max DD 11.66%, no open position reported. It fails the drawdown gate.
- Stage 1 evaluated all 350 combinations in the copied template (the skill's prose says 120; its actual lists multiply to 350). Highest-ranked candidates #1–#5 were individually replayed. Only #3 and #4 passed the annual trade-count floor; both had Max DD about 14%. Candidate rank #36 was the first examined candidate that passed yearly counts and the quality gates: 48 trades (25 / 23), +31,746.19 RUB, PF 1.499, Max DD 9.24%. Its full parameters are in `stage-1-rank36.json`.
- Stage 2 stoch-exit branch kept adequate yearly counts and positive expectancy. The best local starting point was MinGapATR 0.1, StochPeriod 21, StopATR 1.7, TakeProfitATR 1, HTFTrendEMA 100, RequireEntryAboveFVGLower 0, UseStochExit 1. The no-stoch branch had negative expectancy for every grid result; best expectancy was -564.9 RUB/trade, so it failed quality.
- Stage 3 leader (`stage-3-candidate.json`): 49 closed trades (25 / 24), Net PnL +58,203.70 RUB, PF 1.905, Max DD 12.62%, no open position reported. It fails the frozen drawdown gate. The low-entry-frequency local candidate had 24 trades (12 / 12), +8,783.88 RUB, PF 1.319, Max DD 10.39%, also fails. The MaxWaitBars=45 sensitivity had 53 trades (26 / 27), +27,487.15 RUB, PF 1.411, Max DD 11.41%, also fails. One-factor StochPeriod=14 and HTFTrendEMA=120 did not improve expectancy over the leader; MaxWaitBars=15 failed yearly trade minimum.
- No Stage 3 rerun was warranted by sensitivity, and no candidate qualified for locking. Final-year results, final-year trades, and final-year open positions are therefore not available. The reports contain no ending open position for the training runs noted above.

## Reproduction artifacts
- `original.json`, `baseline.json`, `plan.md`
- `stage-1-entry-template.json`, `stage-1/`, `stage-1-candidate-*.json`, `stage-1-candidate-*/`, `stage-1-rank36.json`, `stage-1-rank36/`
- `stage-2-stoch.json`, `stage-2-stoch/`, `stage-2-no-stoch.json`, `stage-2-no-stoch/`
- `stage-3-refinement.json`, `stage-3/`, `stage-3-candidate.json`, `stage-3-candidate/`, `stage-3-candidate-lowtrade.json`, `stage-3-candidate-lowtrade/`
- `sensitivity-stochperiod.json`, `sensitivity-stochperiod/`; `sensitivity-ema.json`, `sensitivity-ema/`; `sensitivity-maxwaitbars.json`, `sensitivity-maxwaitbars/`; `stage-3-sensitivity-maxwait45.json`, `stage-3-sensitivity-maxwait45/`


## Literal update after calibration
The user requested applying the research leader despite the failed drawdown gate and missing independent final test. `internal/service/trading_strategy/fvg_imbalance/strategy/spbe/spbe.go` now returns the Stage 3 leader listed above, and its doc comment records the failed gate and missing final-year check. This change does not alter the calibration outcome or imply validation.
