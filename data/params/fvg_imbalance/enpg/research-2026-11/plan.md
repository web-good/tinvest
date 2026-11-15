# ENPG FVG imbalance calibration plan

## Windows and data snapshot

- Run date reference: 2026-11-13 Europe/Moscow.
- Intended training window: 2023-11-13 through 2025-11-13 (24 months), split into those two consecutive 12-month blocks.
- Intended reserved validation window: 2025-11-13 through 2026-11-13 (12 months).
- Existing local cache at planning time: ENPG M30 2023-08-04 22:00 MSK through 2026-10-15 13:30 MSK; daily 2022-10-28 through 2026-10-27. A standard API refresh will be attempted. Actual bars and bounds must be recorded from reports. If API does not fill the gap, use an explicitly labeled shortened/earlier window; do not claim a current final-year check.
- Candle source: Tinkoff Invest gRPC via `cmd/backtest`, cache `data/candles`; only completed bars are used by the backtest provider/engine. Daily ATR uses prior completed weekday dailies; 14-period Wilder ATR needs 28 daily bars. The command loads 12 months of daily warmup, adequate for ATR and any EMA up to 200.

## Fixed execution settings

- Interval: M30; strategy: `fvg_imbalance`; ticker: ENPG.
- Initial cash: 100,000 RUB; fraction: 1.0; commission: 0.0005 per turnover; risk fraction: 0; lot from instrument metadata; entries at next M30 open.
- Same settings and candle snapshot for all candidates. Stops and targets use daily ATR at signal/actual fill per engine mechanics.

## Baseline and staged grids

- Baseline research params copied from skill reference: ATR 14, MinGapATR 0, MaxWaitBars 30, StochPeriod 14, stochastic exit on, StopATR 1, TakeProfitATR 2, HTF EMA 0, FVG floor filter off.
- Original ENPG literal saved in `original.json`: MinGapATR 0.25; all other fields currently equal baseline defaults.
- Stage 1: supplied 120-combination joint grid over MinGapATR, RequireEntryAboveFVGLower, HTFTrendEMA, StochPeriod, with baseline exits; original ticker values included as separate evaluated control where absent.
- Stage 2: separate StopATR x TakeProfitATR joint grids for both stochastic exit branches, inheriting each locked stage-1 entry candidate explicitly. Preserve all results (`-combos 0`).
- Stage 3: local joint refinement from nearest actually tested neighbors, plus separate sensitivity grids for StochPeriod, HTFTrendEMA, MaxWaitBars as described in `stage-3-refinement.md`.

## Selection rules (fixed before results)

- Primary metric: expectancy (net trade PnL divided by completed trades, after commission).
- Quality constraints: Net PnL > 0, Profit Factor > 1, MaxDrawdownPct <= 10% (the engine reports percent as a fraction; absolute MaxDrawdown is in RUB).
- Every calibration phase and both exit branches: >=20 completed trades across the full 24-month training window and >=10 completed trades by entry date in each fixed 12-month block. Open positions do not count. RecentTrades is not sufficient unless its boundary exactly matches the second block.
- Final reserved 12 months: >=10 completed trades and the same quality constraints. The final period is not inspected until exactly one full stage-3 candidate is locked.
- At most four stage-1 entry candidates. Choose only eligible candidates with stable neighboring results. No fallback candidate after final-year inspection.
- Budget: supplied stage-1 grid (120 combos), original control, up to 4 stage-2 branches (24 combos each from templates), one selected stage-3 joint grid (<=54 combos), and separate sensitivity runs only if needed; report every skipped candidate/check and any truncation.
