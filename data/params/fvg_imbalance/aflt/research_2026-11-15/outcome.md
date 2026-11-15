# AFLT calibration outcome

The locked candidate failed the reserved final-year minimum: 8 closed trades < 10. At the user's subsequent request, the literal was updated to this unverified training candidate; no other candidate was tested on the final year. Candidate and original were each run once with clean positions; their reports show 2025-11-15 22:39–2026-11-15 22:39 MSK. The training reports show 2023-11-15 22:36–2025-11-15 22:36/22:38; the small timestamp differences do not cross an M30 boundary. Loaded M30 and Day1 cache after final fetch ends 2026-11-15 22:00 and 2026-11-14 03:00 MSK respectively. Daily data is supplied under the engine's completed weekday candle rules. The same historical candle cache covered all training runs. Reported candle start times correspond to completed M30 bars by the final timestamp.

| Set/window | Closed trades (year 1 / year 2) | Net RUB after costs | PF | MaxDrawdownPct | Open at end |
|---|---:|---:|---:|---:|---|
| Candidate, training | 26 (16 / 10) | +20741.04 | 3.268 | 4.71% | no |
| Original, training | 44 (25 / 19) | +17109.41 | 1.655 | 9.51% | no |
| Candidate, final year | 8 | +4597.79 | 2.786 | 3.91% | no |
| Original, final year | 27 | +1769.90 | 1.144 | 7.43% | no |

Fixed gates, execution, budget, windows and baseline are in plan.md; candidate and rationale were fixed in locked-candidate.md before final-year runs. The source template actually contains 350 stage-1 combinations, despite the skill prose's 120. Stage 1 top had 15/9 and failed annual gate; retained seeds 2,4,6,9 had 16/10,18/12,25/19,16/13. Stage 2 best eligible stochastic branch expectancy 797.73; best eligible no-stochastic branch 756.67, DD 9.12%, difference 5.15% of larger absolute expectancy. Stochastic branch selected and joint stage 3 and isolated sensitivities checked; neighbors remained positive, as recorded in locked-candidate.md. Each grid contains all nine fields and saved reports retain every combination.

A stop-only example (entry 2, stop 1, take 0, stochastic exit disabled) had only two closed losing trades, an open position at the end, and 89.15% exposure. Equity begins moving after 2024-03-05 around 11:00 MSK after its last closed trade, implying an approximately 620-day open holding to the end; the report does not explicitly record the open entry time. End-of-run equity marks this position to market, without a forced exit; its unrealized PnL contributes to net equity but not closed trade count or PF. See stop-only-example report.

Training and final runs use RUB 100000 starting cash, fraction 1, commission 0.0005 per side, risk-pct 0 and instrument lot resolved by CLI; ATR daily warmup and next-bar execution described in plan.md. MaxDrawdownPct is computed at the engine's maximum absolute drawdown, divided by the corresponding equity peak, and is not necessarily the largest percentage drawdown at any other point.
