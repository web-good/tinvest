// Package nlmk provides parameters for NLMK in the FVG imbalance backtest.
package nlmk

import "tinvest/internal/service/trading_strategy/fvg_imbalance/strategy/core"

// Params returns the NLMK research candidate, installed at the user's request.
// Selection 2023-11-15–2025-11-15: no stochastic exit, expectancy 1162.13,
// 18/10 closed trades in the two years; Net PnL > 0, PF > 1, DD <= 10%.
// Final check 2025-11-15–2026-11-15 failed: 20 trades, Net PnL -1570.62,
// PF 0.93, DD 15.48%. This set has not passed independent validation.
func Params() core.Params {
	return core.Params{
		ATRPeriod:                 14,
		MinGapATR:                 0.3,
		MaxWaitBars:               45,
		StochPeriod:               11,
		UseStochExit:              1,
		StopATR:                   0.3,
		TakeProfitATR:             1.5,
		HTFTrendEMA:               0,
		RequireEntryAboveFVGLower: 1,
	}
}
