// Package reni provides parameters for RENI in the FVG imbalance backtest.
package reni

import "tinvest/internal/service/trading_strategy/fvg_imbalance/strategy/core"

// Params returns the leading research candidate from the 2023-11-13–2025-11-13
// calibration (stochastic exit branch, expectancy ranking). It produced 13/20
// closed trades in the two training years, +23,994.06 net PnL, PF 11.319, and
// 4.11% max drawdown. It is not independently validated: the reserved
// 2025-11-13–2026-11-13 period was unavailable in the candle cache at calibration
// time. Reproduction reports: data/params/fvg_imbalance/reni/research_2026-11/.
func Params() core.Params {
	return core.Params{
		ATRPeriod:                 14,
		MinGapATR:                 0.25,
		MaxWaitBars:               45,
		StochPeriod:               14,
		UseStochExit:              1,
		StopATR:                   1,
		TakeProfitATR:             1,
		HTFTrendEMA:               0,
		RequireEntryAboveFVGLower: 1,
	}
}
