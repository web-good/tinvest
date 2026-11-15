# NLMK calibration — locked before first replay

Anchor T = 2026-11-15 00:00:00 Europe/Moscow. Train [2023-11-15, 2025-11-15), annual entry-date blocks split 2024-11-15. Reserved final [2025-11-15, 2026-11-15). Fixed completed-candle snapshot required; CLI time drift must be disclosed or a fixed-window replay of the same engine used.

Source: Tinkoff Invest CandleProvider/cache; only completed bars (provider drops IsComplete=false). Initial M30 cache: 2023-08-04 to 2026-10-15; daily: 2021-08-12 to 2026-10-27. Top up missing history before final evaluation; never accept a shortened final window as complete. Preserve hashes/bounds of snapshot. Daily lead-in starts 2022-11-15 (one year per executable CLI); ATR requires 28 completed weekday daily bars, EMA up to 200 daily bars. M30 lookback max(3+MaxWaitBars, StochPeriod+1); no trades during initial lookback. Daily cursor excludes current day. H4 unused by executable FVG.

Cash RUB 100000, fraction=1, risk-pct=0, commission=0.0005 of turnover each side, instrument lot from cached/API instrument metadata. Next M30 open entries, stop gap-aware, exact target fills, stochastic close exits. No slippage beyond engine model. Open position marked to market, never forced closed or counted as closed trade.

Primary metric expectancy. Quality: Net PnL > 0 after costs, PF > 1, MaxDrawdownPct <= 0.10 (fraction, NOT absolute RUB MaxDrawdown). Require >=20 closed trades total AND >=10 by entry date in EACH fixed train year at every stage in both branches. Report annual PnL/PF from entry-date closed cohorts, not reset-position annual replays. Final >=10 closed trades plus same quality. Criteria immutable after results.

Stage 1: complete actual template 350 combinations (text says 120, but actual JSON 7*5*5*2=350), plus original control and baseline. Original entry values already included. Keep up to four qualified stable entry candidates with differing filters where feasible.
Stage 2: each selected entry, both full template StopATR x TakeProfitATR grids, all nine fields explicit; no keepTop pruning. Save all rows.
Stage 3: local joint neighbors StopATR/TakeProfitATR/MinGapATR and both lower-bound flags, <=54 per branch; refine both if metric difference <=5% of greater absolute value. Separate StochPeriod neighbor, HTFTrendEMA alternatives, MaxWaitBars [15,30,45] sensitivity; at most one <=54-combination repeat if warranted. At boundary allow one spacing-sized outward step only, constrained positive stop and nonnegative target/gap.
Budget: full stage 1, <=4 entry candidates x2 full exit templates, <=2 initial refinement grids, three sensitivity grids per finalist and at most one repeat per refined branch. Two CPU execution limit, up to 2 hours wall time. If blocked on data or no admissible stage-1 candidate, preserve results, do not update literal or open final year. If budget prevents exhaustion disclose incompleteness.

Lock one full candidate and rationale before final year. Evaluate it and original once on reserved final year with clean positions and daily warmup, ordinary full params. No alternate candidate after final failure. Only passing final candidate can update NLMK literal/its tests; no common defaults, other tickers or live runner changes.

## Execution notes (criteria unchanged)

The ordinary CLI control failed while refreshing instrument metadata: `load shares: failed to request shares: rpc error: code = Internal desc = unexpected EOF` (2026-11-15 22:33:47 MSK). To preserve exact dates and continue research, `replay.go` directly calls the SAME production binding, `domain.Run`, `domain.Compute`, and report renderers, using the fixed cached snapshot and cached NLMK lot=10/UID. This is a research execution adapter, not a strategy change. Every grid combination is saved with its complete closed-trade table and fixed-entry-date annual cohorts; no RecentTrades inference or keepTop. Final history still requires a successful API top-up. The CLI attempt did not reach a strategy replay.

Actual stage-2 templates also differ: stoch template StopATR has six values and TakeProfitATR four (24 exit combinations once StochPeriod is fixed from each stage-1 candidate); no-stoch has four stop and four target values (16 combinations). Its template StochPeriod ranges are explicitly replaced by the selected candidate, as the skill requires. Stage 2 budget is therefore <=160 combinations, not 192.

Drawdown quality uses engine `MaxDrawdownPct`; the engine records the percentage at the largest absolute equity drawdown. No modification to this metric definition is made for this research.


## Final execution record

After candidate lock, direct CandleProvider top-up using cached UID succeeded. Final reserved-year history is complete through last completed M30 November14 23:30 MSK; frozen separately before evaluating the candidate and original once each with full -params in the fixed-window replay adapter. Candidate fails three locked quality conditions; literal retained. See outcome.md, rejected-candidate.json and reports.
