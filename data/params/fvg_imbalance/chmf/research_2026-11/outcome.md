# CHMF FVG imbalance calibration outcome

## Decision

The Stage 3 candidate was locked before inspecting the final period. It met training criteria, but failed the reserved final-year quality limits. Do not change the CHMF strategy literal; retain the current default parameters. Do not select another candidate against the already viewed final period.

## Training selection

- Training window: M30 candles in `[2023-11-12 15:40 MSK, 2025-11-12 15:40 MSK)`; CLI report metadata timestamps vary by seconds/minutes between calls, but included M30 candle boundaries match. Reserved final window: `[2025-11-12 15:45 MSK, 2026-11-12 15:45 MSK)`.
- Highest qualifying Stage 2/3 no-stochastic branch result: `MinGapATR=0.3`, `StochPeriod=10`, `UseStochExit=0`, `StopATR=0.5`, `TakeProfitATR=1`, other fields as in `locked-candidate.json`.
- Training counts by entry timestamp: 10 in `[2023-11-12, 2024-11-12)`, 11 in `[2024-11-12, 2025-11-12)`, 21 total. Net PnL +17,546.98 RUB after commission; PF 2.033; max drawdown 7,072.34 RUB (6.01%); no open position.
- Best qualifying stochastic-exit comparison: `MinGapATR=0.3`, `StochPeriod=9`, `UseStochExit=1`, `StopATR=1`, `TakeProfitATR=2`; 10/12/22 entries, +12,983.26 RUB, PF 9.895, max DD 3,030.86 RUB. The selected branch ranked higher on expectancy and net PnL. No-stochastic top-ranked sets failing the first-year trade floor were skipped.

## Reserved final-year result

Actual M30 coverage in the refreshed cache: 2025-11-12 15:30 MSK through 2026-11-12 15:00 MSK; the last M30 candle had closed by the report cutoff. Daily coverage ended 2026-11-11, the last completed daily candle before the cutoff.

- Candidate: 14 closed trades (>=10 required); Net PnL -8,586.50 RUB; PF 0.536; max DD 14,463.63 RUB (13.98%); no open position. It fails positive Net PnL, PF > 1, and max DD <=10%.
- Original current-default set: 211 closed trades; Net PnL -1,755.67 RUB; PF 0.991; max DD 30,316.14 RUB (27.32%); one open position at end, mark-to-market. It also fails quality limits.
- The candidate performed worse than the original by 6,830.83 RUB in net PnL. Its PF and max drawdown were also worse, despite fewer trades. Thus the training improvement did not carry to the held-out period.

The final reports are in `reports/fvg_imbalance/chmf/research_2026-11/final-candidate/` and `final-original/`. The candidate and all grids/reports are retained for research. The ticker literal remains unchanged and the candidate is not verified for deployment.
