# MTLR FVG imbalance live support

## Intent

Make MTLR available to the existing FVG imbalance live runner while keeping actual order submission disabled in production.

## Scope and constraints

- Add MTLR's parameter provider to the ticker dispatch in `internal/service/trading_strategy/fvg_imbalance/live/strategy.go`.
- Add `MTLR` to `FVG_IMBALANCE_TICKERS` in `env/prod.env`.
- Preserve `FVG_IMBALANCE_TRADE_ENABLED=false`; do not enable live orders or change sizing.
- Live decider and protection must use `mtlr.Params()`. The current literal has `MinGapATR=0.30`; the research candidate JSON has `0.25`, but this change will not modify or override the literal.
- Extend live registry tests to verify MTLR dispatch and the derived stop from its parameters.

## Approaches

1. Register MTLR in the existing live ticker switch and production ticker list. This is the selected approach: it matches the existing architecture and enables support while the global trade switch remains off.
2. Also set `FVG_IMBALANCE_TRADE_ENABLED=true`. This would permit orders for every configured ticker and exceeds the requested scope, so it is excluded.

## Acceptance

The live adapter returns a decider for MTLR, derives its desired stop from `mtlr.Params()`, production config includes MTLR, and the global trade switch remains false. Run focused live package tests and formatting; no deployment is part of this change.
