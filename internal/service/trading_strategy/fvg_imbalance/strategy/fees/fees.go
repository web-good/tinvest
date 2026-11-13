// Package fees provides parameters for FEES in the FVG imbalance backtest.
package fees

import "tinvest/internal/service/trading_strategy/fvg_imbalance/strategy/core"

// Params returns the FEES research candidate selected by expectancy without a
// stochastic exit. On 2023-11-12—2025-11-12 it produced 11 and 11 closed
// trades by year, net +15,620.21 RUB, PF 1.524 and max drawdown 9.65%.
// Nearby parameters were unstable; the reserved final year was not tested.
func Params() core.Params {
	return core.Params{
		ATRPeriod:                 14,
		MinGapATR:                 0.25,
		MaxWaitBars:               30,
		StochPeriod:               10,
		UseStochExit:              0,
		StopATR:                   1,
		TakeProfitATR:             1,
		HTFTrendEMA:               0,
		RequireEntryAboveFVGLower: 1,
	}
}
