// Package irao provides parameters for IRAO in the FVG imbalance backtest.
package irao

import "tinvest/internal/service/trading_strategy/fvg_imbalance/strategy/core"

// Params returns the research candidate selected by expectancy on 2023-11-12–2025-11-12.
// Its closed trades by training year were 11 and 13; the 2025-11-12–2026-11-12
// check had 13 trades and failed the precommitted 10% drawdown limit (13.83%).
// The candidate uses a price target and stop without the stochastic exit.
func Params() core.Params {
	return core.Params{
		ATRPeriod:                 14,
		MinGapATR:                 0.25,
		MaxWaitBars:               30,
		StochPeriod:               14,
		UseStochExit:              0,
		StopATR:                   1,
		TakeProfitATR:             1.5,
		HTFTrendEMA:               0,
		RequireEntryAboveFVGLower: 1,
	}
}
