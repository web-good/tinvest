// Package enpg provides default parameters for ENPG in the FVG imbalance backtest.
package enpg

import "tinvest/internal/service/trading_strategy/fvg_imbalance/strategy/core"

// Params returns the best ENPG training candidate, written at explicit request
// despite failing the reserved-year quality checks.
// Training: 2023-11-13–2025-11-13; stochastic exit; expectancy metric;
// completed trades by training year: 13/15; Net PnL +23,302.80 RUB;
// PF 4.698; Max DD 4.43%. Reserved year: 2025-11-13–2026-11-13;
// 16 trades; Net PnL -7,146.66 RUB; PF 0.397; Max DD 9.93%.
// The reserved-year result failed the positive Net PnL and PF > 1 gates.
func Params() core.Params {
	return core.Params{
		ATRPeriod:                 14,
		MinGapATR:                 0.3,
		MaxWaitBars:               30,
		StochPeriod:               10,
		UseStochExit:              1,
		StopATR:                   0.5,
		TakeProfitATR:             1.5,
		HTFTrendEMA:               0,
		RequireEntryAboveFVGLower: 0,
	}
}
