// Package lsrg provides ticker-specific parameters for the FVG imbalance backtest.
package lsrg

import "tinvest/internal/service/trading_strategy/fvg_imbalance/strategy/core"

// Research candidate locked 2026-11-13 from training [2023-11-13, 2025-11-13),
// stochastic exit branch, ranked by expectancy. The 11/10 trades in the two
// training years met the run's minimums; training PF was 2.348, net PnL +11386.21,
// and max drawdown 6.23% (limits: positive net PnL, PF > 1, max DD <= 10%).
// The reserved final year could not be validated because candle history/API access
// was incomplete; these values are an unvalidated research candidate.
func Params() core.Params {
	return core.Params{
		ATRPeriod:                 14,
		MinGapATR:                 0.3,
		MaxWaitBars:               30,
		StochPeriod:               9,
		UseStochExit:              1,
		StopATR:                   2,
		TakeProfitATR:             1,
		HTFTrendEMA:               0,
		RequireEntryAboveFVGLower: 1,
	}
}
