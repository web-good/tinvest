# VTBR FVG imbalance calibration outcome

Run: 2026-11-12 MSK. Cached Tinkoff data was refreshed before final validation. Final M30 cache: 2025-11-12 15:00 through 2026-11-12 14:30 MSK; daily cache through 2026-11-11. Training runs used completed M30/daily candles on [2023-11-12, 2025-11-12); final validation used [2025-11-12, 2026-11-12). CLI timestamps differ by seconds per run, but the same completed candle dates were used. One-year daily warmup was present; ATRPeriod=14 requires 28 daily bars, EMA50 warmup is covered by the CLI lead-in. Execution/costs fixed at 100,000 RUB, fraction=1, risk-pct=0, 0.0005 commission, regular lot sizing, entry at next M30 open. MaxDrawdownPct is a fraction of running peak equity (0.10=10%).

The initial stage-1 template contains 350 combinations (7 MinGapATR × 5 StochPeriod × 5 HTFTrendEMA × 2 entry-filter values); this differs from the skill text's stated 120. All 350 results were preserved. Original literal equals core.DefaultParams() and was separately measured as a control.

Selected parameters: `final-candidate.json` (ATRPeriod=14, MinGapATR=0.1, MaxWaitBars=30, StochPeriod=9, UseStochExit=1, StopATR=1, TakeProfitATR=2, HTFTrendEMA=50, RequireEntryAboveFVGLower=1). Branch: stochastic exit. Stage-3 result: 32 trades, 10 in training year 1 and 22 in training year 2; expectancy 400.5 RUB/trade, Net PnL +12,815.66 RUB, PF 2.445, Max DD 7,029.61 RUB (7.03%). Stage 3 joint neighborhood did not improve the eligible candidate. StochPeriod=11 and MaxWaitBars=15 sensitivity leaders had only 9 and 8 entries, respectively, in training year 1 and were ineligible.

Both no-stochastic branches failed preset quality gates: rank2 best had expectancy 23.06, Net PnL +461.13, PF 1.011, Max DD 13.99%; rank6 best had expectancy 285, Net PnL +5,700.04, PF 1.108, Max DD 16.47%. Stage 2 stoch rank2 had higher expectancy than rank6 (400.5 vs 338.7), with both passing training quality gates. Fixed-parameter full journals confirmed both annual floors: rank2 10/22 trades; rank6 13/30.

Final candidate: 43 closed trades, no open position, Net PnL +2,668.13 RUB, PF 1.191, Max DD 4,773.63 RUB (4.77%), expectancy 62.05 RUB/trade. It passed >=10 trades and all pre-set quality gates. Original control on final year: 196 trades, no open position, Net PnL -14,369.74 RUB, PF 0.831, Max DD 35,358.77 RUB (31.36%).

The VTBR literal was updated after this final check. Ordinary no-`-params` execution was checked separately; report link is below. No average R is calculated by the engine.

## Artifacts

- Plan and locked criteria: `plan.md`, `locked-candidate.md`.
- Full selected params: `final-candidate.json`.
- Stage 1 grid: `stage-1-entry.json`; report: `reports/fvg_imbalance/vtbr/research_2026-11/stage-1/`.
- Candidate control journals: `reports/fvg_imbalance/vtbr/research_2026-11/stage-1-rank2-full/` (10/22 trades) and `stage-1-rank6-full/` (13/30 trades); original training control: `.../control/`.
- Stage 2 grids: `stage-2-rank{2,6}-{stoch,no-stoch}.json`; reports: `reports/fvg_imbalance/vtbr/research_2026-11/stage-2/`.
- Stage 3 grid: `stage-3-refinement.json`; report: `reports/fvg_imbalance/vtbr/research_2026-11/stage-3/`.
- Sensitivity grids/reports: `sensitivity-*.json`, `reports/fvg_imbalance/vtbr/research_2026-11/sensitivity-*/`.
- Final candidate and original reports: `reports/fvg_imbalance/vtbr/research_2026-11/final-candidate/`, `.../final-original/`. Default literal check: `.../default-literal-check/` (same 43 trades and metrics as final candidate).
