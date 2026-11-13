// Package spbe provides SPBE parameters for the FVG imbalance backtest.
package spbe

import "tinvest/internal/service/trading_strategy/fvg_imbalance/strategy/core"

// Params returns the best research candidate from 2026-11 calibration.
// It is not a verified calibration: training Max DD was 12.62% (above the 10% gate),
// and no independent final-year check was available.
func Params() core.Params {
	return core.Params{
		ATRPeriod:                 14,
		MinGapATR:                 0.1,
		MaxWaitBars:               30,
		StochPeriod:               21,
		UseStochExit:              1,
		StopATR:                   1.7,
		TakeProfitATR:             1,
		HTFTrendEMA:               100,
		RequireEntryAboveFVGLower: 0,
	}
}
