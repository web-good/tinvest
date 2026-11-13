# POSI calibration outcome

## Frozen setup

- Source: refreshed Tinkoff Invest candle cache. Completed POSI M30 candles: 2023-11-12 16:00—2026-11-12 15:30 MSK (37,082); daily candles: 2022-11-14—2026-11-11 (1,153). Daily history supplies completed Wilder ATR(14), with one year of EMA/ATR lead-in. The executable POSI setup does not request H4 data; its optional trend EMA is calculated from daily closes.
- Training range: 2023-11-12 16:17—2025-11-12 16:17 MSK. Final range: 2025-11-12 16:28—2026-11-12 16:28 MSK. CLI boundary timestamps varied by minutes between runs, but each boundary stayed between the same adjacent M30 candle timestamps, so the corresponding runs used the same completed bars. The final-year M30 data ended at the latest completed candle, not an unfinished bar.
- Cash 100,000 RUB, fraction 1.0, risk-pct 0, instrument lot, commission 0.0005 per turnover side, next M30 open execution. The same local candle cache was reused throughout calibration and final runs.
- Frozen gates: 24 training months, minimum 20 closed trades overall and 10 by entry date in each training year; final year minimum 10 closed trades; positive after-cost Net PnL, PF > 1, Max DD <= 10%. `MaxDrawdownPct` is a fraction of running peak equity; the report displays it as a percent.

## Search and selection

- Stage 1 searched all 350 combinations in `stage-1-entry.json`. Top entry: MinGapATR=0.3, StochPeriod=9, EMA=0, lower-FVG filter off; 25 closed trades, 10/15 by year, PF 11.327, Net +26,202.32 RUB, absolute DD 3,147.60 RUB. A nearby MinGapATR=0.25 input produced 45 trades and 22/23 by year, and was also carried to stage 2.
- Stage 2 compared both exit branches for both inputs. With stochastic exit enabled, all stop/target combinations for the leading MinGapATR=0.3 entry tied on this sample because closed trades ended at the stochastic exit; expectancy was 1,048 RUB, with 10/15 trades, PF 11.327, Net +26,202.32 RUB, absolute DD 3,147.60 RUB. The best year-qualified no-stochastic configuration was StopATR=0.5, TakeProfitATR=1: expectancy 677.8 RUB, 10/13 trades, PF 1.653, Net +15,589.32 RUB, absolute DD 7,528.52 RUB. Its absolute DD below 10,000 RUB implies percentage DD below 10% because running peak equity never fell below initial cash. The stochastic branch was selected; its score exceeded the no-stochastic branch by more than the 5% tie band.
- Stage 3 refined the selected entry and checked StochPeriod, daily EMA, and MaxWaitBars separately. MaxWaitBars=45 modestly improved expectancy to 1,050.59 with 12/15 trades. StochPeriod 10/11 and nonzero EMA settings failed the annual trade threshold. One local combined rerun at MaxWaitBars=45 confirmed the complete selected set: MinGapATR=0.3, StochPeriod=9, StopATR=1, TakeProfitATR=2, filter off. Other tied exit values were retained at the original 1/2 pair.
- Candidate locked in `locked-candidate.md` before the final range was run. Final test was run once for the candidate and once for the original literal; no post-test retuning was performed.

## Final-year decision

| Set | Closed trades | Net PnL after costs | PF | Max DD | Open position | Result |
|---|---:|---:|---:|---:|---|---|
| Candidate | 17 | +19,843.99 RUB | 7.826 | 3.39% | No | Passed all gates |
| Original literal | 22 | +6,265.18 RUB | 1.376 | 7.87% | No | Passed all gates |

The candidate remains eligible and outperformed the original on this reserved year. Its complete nine-field object is `final-candidate.json` and has been installed in `posi.Params()`.

## Reports

- Control original on training: `reports/fvg_imbalance/posi/research_2026-11/control-train/`
- Stage 1: `reports/fvg_imbalance/posi/research_2026-11/stage-1/`
- Stage 2, both candidate inputs and both exit branches: `reports/fvg_imbalance/posi/research_2026-11/stage-2/`
- Stage 3, local rerun and sensitivity: `reports/fvg_imbalance/posi/research_2026-11/stage-3/`
- Final candidate and original: `reports/fvg_imbalance/posi/research_2026-11/final/`
