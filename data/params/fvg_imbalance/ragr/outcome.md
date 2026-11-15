# RAGR FVG imbalance calibration outcome

Calibration source: cached Tinkoff Invest candles. Stage 1–3 reports were produced from the original cache (M30 2023-10-07 through 2026-10-13; daily 2021-08-12 through 2026-10-27), which covered the full training windows. A forced refresh was started before the first run but API requests timed out; it completed later and replaced the cache with a truncated series (M30 starts 2023-11-27, missing the first two weeks of the requested range, and ends 2025-11-13). The refresh control report is therefore not a valid full-window calibration baseline. Final-year runs used the original cached history plus fetched tail through 2026-11-13; their actual first available equity candles differed by 10 hours, so final candidate/original comparison is not perfectly candle-aligned. Completed daily candles were used; the engine warmup includes one year of daily lead-in, 28 completed weekday dailies for Wilder ATR(14), and the strategy's M30 lookback. Runs used next M30 open entry, RUB 100,000 initial cash, fraction 1, 0.05% commission per turnover, risk-pct 0, lot from instrument metadata. Max DD quality gate was `MaxDrawdownPct <= 10%`, not absolute RUB.

## Training

Common nominal training window: 2023-11-13 through 2025-11-13, with fixed annual blocks `[2023-11-13, 2024-11-13)` and `[2024-11-13, 2025-11-13)`. Reports were run against the same cached candle snapshot; CLI timestamps vary by a few minutes. All trade counts below count closed round trips once by entry date; an open position is excluded.

The original RAGR literal is `ATRPeriod=14, MinGapATR=0.25, MaxWaitBars=30, StochPeriod=14, UseStochExit=1, StopATR=1, TakeProfitATR=2, HTFTrendEMA=0, RequireEntryAboveFVGLower=0`. Training control: 28 closed trades, 14/14 by year, Net PnL -5,891.73 RUB, PF 0.750, Max DD 13.83%.

The selected research candidate is `ATRPeriod=14, MinGapATR=0.3, MaxWaitBars=45, StochPeriod=14, UseStochExit=0, StopATR=1, TakeProfitATR=1, HTFTrendEMA=0, RequireEntryAboveFVGLower=0`. It closed 20 trades, 10/10 by year; Net PnL +26,645.23 RUB, PF 1.905, Max DD 8.08%, expectancy 1,396.99. One position remained open at the training boundary and was mark-to-market in equity, but not counted as a completed trade. It met the training gates.

Stage 1 covered all 350 actual combinations in the supplied template (the template's prose says 120, but its lists multiply to 350). Stage 2 tested both exit branches with explicit inherited entry fields. Best eligible stochastic-exit result had expectancy 143.1; best eligible no-stochastic-exit result had expectancy 621.7. The difference exceeded the 5% closeness threshold. Stage 3 refined only the no-stochastic branch. A separate MaxWaitBars sensitivity showed 45 as a better eligible value, followed by the one allowed joint repeat. The frozen candidate was rank 1 in that repeat. Search scope and exact grids are retained in the JSON files.

## Reserved final year

The selected candidate was frozen before accessing the reserved window and tested once. It closed 13 trades (>=10), with no open position at report end. It failed quality rules: Net PnL -1,164.94 RUB, PF 0.956, Max DD 13.39%. Do not install it in the ticker literal.

Original-literal control on the final year: 17 closed trades, no open position, Net PnL -1,278.57 RUB, PF 0.900, Max DD 6.98%. The report boundary timestamps differ by one minute; equity series end at the same M30 close, while their first available equity candles differ by 10 hours. Treat this as a near-identical calendar-window comparison, not exact candle-for-candle equality. Both candidates lost money and failed the predeclared positive-PnL/PF rules.

After the final failure, the user explicitly requested installing the best training candidate. The RAGR ticker literal now uses these values despite the failed OOS checks; they remain unvalidated for production and the failed metrics are documented beside the literal. `research-candidate.json` and `selected.json` preserve the exact set. Initial one-combination stage-2 reports were superseded by the corrected 24- and 16-combination grids and are not calibration evidence.

Closed-trade PnL attributed to training entry date (not a separate replay or annual equity curve): candidate year 1 +13,864.10 RUB, PF 2.186; year 2 +14,075.68 RUB, PF 1.734. Original literal year 1 -10,943.93 RUB, PF 0.316; year 2 +5,052.20 RUB, PF 1.666. The candidate's pooled report Net PnL is lower than the sum of its closed-trade PnL because the report also marks its open boundary position to market.
