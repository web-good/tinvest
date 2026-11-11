// Package bspb holds the BSPB parameters for the backtest-only M30 FVG strategy.
package bspb

import "tinvest/internal/service/trading_strategy/fvg_imbalance/strategy/core"

// Params returns the BSPB calibration verified on 2025-11-11 through 2026-11-11.
// The 2023-11-11 through 2025-11-11 learning window used the stochastic exit
// branch and expectancy ranking; its two annual blocks had 19 and 14 closed
// trades. The locked candidate passed the final year with 15 trades, positive
// net PnL, PF > 1, and 2.97% maximum drawdown. See data/params/fvg_imbalance/bspb/research_2026-11.
func Params() core.Params {
	return core.Params{
		ATRPeriod:                 14,
		MinGapATR:                 0.25,
		MaxWaitBars:               30,
		StochPeriod:               10,
		UseStochExit:              1,
		StopATR:                   1.5,
		TakeProfitATR:             1,
		HTFTrendEMA:               0,
		RequireEntryAboveFVGLower: 1,
	}
}
