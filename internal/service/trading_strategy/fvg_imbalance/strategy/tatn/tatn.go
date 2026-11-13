// Package tatn provides default parameters for TATN in the FVG imbalance backtest.
package tatn

import "tinvest/internal/service/trading_strategy/fvg_imbalance/strategy/core"

// Params returns the TATN research candidate selected on 2023-11-12–2025-11-12.
// Training: no-stochastic-exit branch, expectancy 1,027, 17/11 closed trades by year,
// Net PnL +28,745.38 RUB, PF 2.553, Max DD 5.64%.
// The final-year run used a different parameter file, so this candidate is unverified OOS.
func Params() core.Params {
	return core.Params{
		ATRPeriod:                 14,
		MinGapATR:                 0.3,
		MaxWaitBars:               30,
		StochPeriod:               11,
		UseStochExit:              0,
		StopATR:                   0.5,
		TakeProfitATR:             1.5,
		HTFTrendEMA:               0,
		RequireEntryAboveFVGLower: 0,
	}
}
