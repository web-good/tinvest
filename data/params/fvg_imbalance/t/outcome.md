# T calibration outcome

Calibration stopped after Stage 1 because the complete entry grid contained no eligible candidate. No Stage 2/3 grids were run, the reserved final year was not opened, and the existing ticker literal was left unchanged.

- Actual training report range: 2023-11-11 12:09 to 2025-11-11 12:09 MSK. The report uses the latest cached candles not later than the CLI boundary.
- Full Stage 1 grid: 192/192 combinations retained (`-combos 0`); baseline values and the original literal's entry values were included. Highest expectancy was -70.33, with 84 closed trades, PF 0.820, Net PnL -5,907.76. Its absolute max drawdown was 18,151.34.
- Across all 192 results, 61 had positive Net PnL and PF > 1, but none also had at least 20 closed trades and at least 10 RecentTrades. Thus none could pass the predeclared quality and second-year trade-count gates; the first-year check cannot admit any of them.
- Original literal control: 17 trades, Net PnL -15,305.34, PF 0.364. Research baseline control: 354 trades, Net PnL -68,298.73, PF 0.460, Max DD 68.90%. Both were run only on the training period.
- No candidate was frozen, so no independent 2025-11-11 to 2026-11-11 comparison was made. The calibration did not establish a verified replacement for the existing literal.

Reports: `reports/fvg_imbalance/T/stage-1/` (complete grid and full combination details), `reports/fvg_imbalance/T/control/` (original literal), and `reports/fvg_imbalance/T/control-baseline/` (research baseline).
