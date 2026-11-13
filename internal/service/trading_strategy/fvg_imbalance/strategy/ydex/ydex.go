// Package ydex provides default parameters for YDEX in the FVG imbalance backtest.
package ydex

import "tinvest/internal/service/trading_strategy/fvg_imbalance/strategy/core"

// Params returns the best training-window parameters for YDEX. The candidate
// ranked first by expectancy during 2023-11-12—2025-11-12, but did not pass the
// reserved final-year minimum-trade check (7 trades; required 10).
func Params() core.Params {
	return core.Params{
		ATRPeriod:                 14,
		MinGapATR:                 0.3,
		MaxWaitBars:               30,
		StochPeriod:               14,
		UseStochExit:              1,
		StopATR:                   1.2,
		TakeProfitATR:             1.5,
		HTFTrendEMA:               0,
		RequireEntryAboveFVGLower: 1,
	}
}
