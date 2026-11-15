// Package svcb provides default parameters for SVCB in the FVG imbalance backtest.
package svcb

import "tinvest/internal/service/trading_strategy/fvg_imbalance/strategy/core"

// Params returns the best SVCB research candidate from the 2026-11 calibration.
// On 2024-10-13—2026-10-13 training data, the stochastic-exit branch had 36
// closed trades (22/14 by year), expectancy RUB 402.7, PF 2.035, and Max DD
// 5.27%, within the fixed gates of positive Net PnL, PF > 1, and DD <= 10%.
// It has no independent final-year validation: available history is too short.
func Params() core.Params {
	return core.Params{
		ATRPeriod:                 14,
		MinGapATR:                 0.25,
		MaxWaitBars:               30,
		StochPeriod:               9,
		UseStochExit:              0,
		StopATR:                   0.5,
		TakeProfitATR:             1.5,
		HTFTrendEMA:               0,
		RequireEntryAboveFVGLower: 1,
	}
}
