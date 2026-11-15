// Package vkco provides default parameters for VKCO in the FVG imbalance backtest.
package vkco

import "tinvest/internal/service/trading_strategy/fvg_imbalance/strategy/core"

// Params returns the best exploratory VKCO calibration candidate from the
// 2023-11-13—2025-11-13 training window (stochastic exit, expectancy).
// It produced 18 and 16 closed trades in the two training years, Net PnL
// 14916.56 RUB, PF 3.132, and Max DD 3.32%. The reserved final year was not
// tested, and results were sensitive to StochPeriod; treat this as unverified.
func Params() core.Params {
	return core.Params{
		ATRPeriod:                 14,
		MinGapATR:                 0.25,
		MaxWaitBars:               45,
		StochPeriod:               9,
		UseStochExit:              1,
		StopATR:                   1,
		TakeProfitATR:             2,
		HTFTrendEMA:               0,
		RequireEntryAboveFVGLower: 1,
	}
}
