// Package alrs provides default parameters for ALRS in the FVG imbalance backtest.
package alrs

import "tinvest/internal/service/trading_strategy/fvg_imbalance/strategy/core"

// Params returns the selected ALRS parameters. Training window: 2023-11-12 to
// 2025-11-12; stochastic exit enabled; ranked by expectancy. The selected
// candidate had 33 closed trades (20/13 by year), Net PnL 20,699.07 RUB,
// PF 3.252, and Max DD 4.86%, meeting the training gates. The reserved
// 2025-11-12 to 2026-11-12 check was incomplete: M30 data were missing for
// 2026-10-14 through 2026-10-26, so out-of-sample validation is not established.
func Params() core.Params {
	return core.Params{
		ATRPeriod:                 14,
		MinGapATR:                 0.25,
		MaxWaitBars:               45,
		StochPeriod:               14,
		UseStochExit:              1,
		StopATR:                   2,
		TakeProfitATR:             0,
		HTFTrendEMA:               0,
		RequireEntryAboveFVGLower: 1,
	}
}
