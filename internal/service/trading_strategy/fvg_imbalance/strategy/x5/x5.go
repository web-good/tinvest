// Package x5 provides parameters for X5 in the FVG imbalance backtest.
package x5

import "tinvest/internal/service/trading_strategy/fvg_imbalance/strategy/core"

// Params returns the exploratory stage-one leader by expectancy. The 2023-11-12
// to 2025-11-12 history joins pre-suspension depositary receipts to shares after
// redomiciliation, and no independent validation or exit calibration was done.
func Params() core.Params {
	return core.Params{
		ATRPeriod:                 14,
		MinGapATR:                 0.13,
		MaxWaitBars:               30,
		StochPeriod:               14,
		UseStochExit:              1,
		StopATR:                   0.8,
		TakeProfitATR:             2,
		HTFTrendEMA:               0,
		RequireEntryAboveFVGLower: 1,
	}
}
