// Package sngsp provides calibrated FVG imbalance parameters for SNGSP.
package sngsp

import "tinvest/internal/service/trading_strategy/fvg_imbalance/strategy/core"

// Params returns SNGSP parameters calibrated on 2023-11-12–2025-11-12 and
// validated on 2025-11-12–2026-11-12. Stochastic exit; expectancy ranking;
// closed trades: 10 and 14 in the training years, 17 in validation.
// Commission 0.05% per side; minimum 10 trades per year, PF > 1,
// positive net PnL, and maximum drawdown <= 10%.
func Params() core.Params {
	return core.Params{
		ATRPeriod:                 14,
		MinGapATR:                 0.25,
		MaxWaitBars:               30,
		StochPeriod:               14,
		UseStochExit:              1,
		StopATR:                   2,
		TakeProfitATR:             2,
		HTFTrendEMA:               0,
		RequireEntryAboveFVGLower: 0,
	}
}
