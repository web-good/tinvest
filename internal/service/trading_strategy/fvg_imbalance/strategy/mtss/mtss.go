// Package mtss provides parameters for MTSS in the FVG imbalance backtest.
package mtss

import "tinvest/internal/service/trading_strategy/fvg_imbalance/strategy/core"

// Params returns the exploratory MTSS parameters selected on 2023-11-12–2025-11-12.
// The no-stochastic-exit branch had 16 and 16 closed trades in the two years,
// expectancy 799.25, PF 2.040 and maximum drawdown 7.59% after commission.
// The result is sensitive to the target; the reserved final year was not tested.
func Params() core.Params {
	return core.Params{
		ATRPeriod:                 14,
		MinGapATR:                 0.3,
		MaxWaitBars:               45,
		StochPeriod:               14,
		UseStochExit:              0,
		StopATR:                   0.5,
		TakeProfitATR:             1.5,
		HTFTrendEMA:               0,
		RequireEntryAboveFVGLower: 1,
	}
}
