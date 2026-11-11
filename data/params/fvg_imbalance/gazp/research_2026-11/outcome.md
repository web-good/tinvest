# GAZP FVG imbalance calibration outcome — research only

## Data limit

The available M30 cache starts 2024-11-05 and ends 2026-11-10; daily history starts 2023-11-06 and ends 2026-11-09. The usable selection window was 2024-11-11 to 2026-11-11, split into 2024-11-11–2025-11-11 and 2025-11-11–2026-11-11. These two years were used for training and selection. There is no separate following 12-month period in the available data, so this result is not independent validation. No final-year comparison was run, and the GAZP literal was not changed.

Completed candles only; Tinkoff Invest API cache in `data/candles`; about one year of daily lead-in loaded for ATR/EMA warm-up, with ATRPeriod=14 (28 daily bars for Wilder seeding). Engine execution rules, next-M30-open fills and gap handling were unchanged. Every run used cash 100,000, fraction 1, risk-pct 0, commission 0.0005, and the instrument's default lot.

## Controls

- Existing GAZP literal (`original.json`): 27 closed trades, split 9/18, Net PnL +31,488.07, PF 2.041, Max DD 10.81%, no open position.
- Research baseline (`baseline.json`): 394 closed trades, Net PnL -51,514.53, PF 0.702, Max DD 52.62%; one open position at report end.

## Entry and exits

Stage 1 yielded four transferable seeds from the planned EMA set. Exact full-window closed-trade counts were 12/27, 13/30, 20/35, and 24/55. Their Net PnL/PF/MaxDD% were respectively +6,197.73/1.413/5.44%, +6,546.44/1.357/6.26%, +7,530.40/1.379/6.07%, and +8,860.60/1.314/9.42%.

Stage 2 selected the no-stochastic-exit branch on seed A. The highest expectancy rows ahead of its accepted row failed the 10% drawdown cap; the first acceptable row was StopATR=0.5, TakeProfitATR=2.0. This complete parameter set had 32 closed trades split 12/20, Net PnL +71,723.06, PF 2.508, Max DD 9.59%, and no open position. The best accepted stoch-exit alternative for seed A had expectancy 307.9 versus 2,241 for the selected no-stoch-exit row; it had 39 trades split 12/27, Net PnL +12,006.50, PF 2.073, Max DD 5.84%, and no open position.

Stage 3's full 36-combination grid had higher raw expectancy rows, but fixed runs for ranks 2–4 exceeded the drawdown cap (15.89%, 18.03%, 14.83%). Rank 1 had only 9 first-year trades; rank 5 had only 9 first-year trades. Rank 6 was the best fully checked eligible row and reproduced the Stage 2 candidate: 12/20 annual trades, Net PnL +71,723.06, PF 2.508, Max DD 9.59%, no open position.

Sensitivity checks did not yield an admissible replacement. StochPeriod=9 raised expectancy to 2,457 but failed the DD cap at 11.24%; StochPeriod=14 lowered expectancy to 1,919. HTFTrendEMA alternatives and MaxWaitBars 15/45 had lower expectancy than the selected value.

## Candidate (not verified out of sample)

`research-candidate.json` contains all nine fields:

```json
{
  "ATRPeriod": 14,
  "MinGapATR": 0.25,
  "MaxWaitBars": 30,
  "StochPeriod": 12,
  "UseStochExit": 0,
  "StopATR": 0.5,
  "TakeProfitATR": 2,
  "HTFTrendEMA": 0,
  "RequireEntryAboveFVGLower": 1
}
```

The ticker literal remains unchanged because only 24 months are available. The following 12-month independent test and a repeatable verified result are still required before updating it.

## Grid and report audit

The Stage 1 copied grid unexpectedly contained `HTFTrendEMA=120`, creating 240 combinations rather than the planned 192. Selection was restricted to the planned values `[0,50,100,200]`; all four transferred seeds are from that subset. The extra rows are retained in the report and were not used to select candidates.

Grid copies are in this directory. Full reports are under `reports/fvg_imbalance/GAZP/research_2026-11/`, separated by control, stage, candidate and sensitivity.
