// Package ragr provides default parameters for RAGR in the FVG imbalance backtest.
package ragr

import "tinvest/internal/service/trading_strategy/fvg_imbalance/strategy/core"

// Params returns the selected RAGR research parameters.
// The candidate passed the training gates but failed the reserved final-year
// quality checks (Net PnL -1,164.94 RUB, PF 0.956, Max DD 13.39%); it is
// installed at the user's request and should not be treated as OOS-validated.
func Params() core.Params {
	return core.Params{
		ATRPeriod:                 14,
		MinGapATR:                 0.3,
		MaxWaitBars:               45,
		StochPeriod:               14,
		UseStochExit:              0,
		StopATR:                   1,
		TakeProfitATR:             1,
		HTFTrendEMA:               0,
		RequireEntryAboveFVGLower: 0,
	}
}
