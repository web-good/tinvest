# Locked TATN candidate before final-year inspection

Locked: 2026-11-12, Europe/Moscow. Final holdout not inspected yet.

Candidate: `candidate-1` entry `{MinGapATR:0.3, StochPeriod:11, HTFTrendEMA:0, RequireEntryAboveFVGLower:0}`, with `UseStochExit:0`, `StopATR:0.5`, `TakeProfitATR:1`, `ATRPeriod:14`, `MaxWaitBars:30`.

The fixed 24-month training run at `reports/TATN/fvg_imbalance/research_2026-11/stage-2-selected-no-stoch/` had 28 closed trades, 17 in `[2023-11-12,2024-11-12)` and 11 in `[2024-11-12,2025-11-12)`. Net PnL 28,745.38 RUB, PF 2.553, max DD 5.64% (fractional equity drawdown reported as percent). It meets the frozen limits. Stage 3 ranked higher combinations, but their max DD exceeded the 10% cap; candidates with lower DD but higher rank were not selected over this robust result. The one-field sensitivity runs found no higher candidate satisfying both the year trade floor and quality limits: MaxWaitBars 15 had only 6 recent-year entries; MaxWaitBars 45 failed the 10% DD limit. StochPeriod and HTFTrendEMA alternatives did not improve the metric while passing the same criteria.

Selection basis: highest-ranked eligible Stage 2 candidate on the no-stochastic branch, retained through Stage 3; branch leader on the stochastic branch had lower expectancy (854.2) and similar low drawdown (4.01%). Both had the same 29 total trades on training, but the no-stochastic branch was selected on the frozen primary metric after year counts were checked. The no-stochastic Stage 2 rank 4 result is 1,027 RUB expectancy; Stage 3's higher expectancy results violate the predeclared drawdown ceiling.

Baseline training control had 389 trades (180 / 209 by the same year blocks), Net PnL -10,508.82 RUB, PF 0.950, max DD 23.06%, and an open position at window end. Costs and sizing: 0.05% commission per turnover, 100,000 RUB cash, fraction 1, risk-pct 0. This decision is final before opening `[2025-11-12,2026-11-12)`.
