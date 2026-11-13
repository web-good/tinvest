// Package ozon provides default parameters for OZON in the FVG imbalance backtest.
package ozon

import "tinvest/internal/service/trading_strategy/fvg_imbalance/strategy/core"

// Calibration note: trained on 2023-11-12–2025-11-12 and checked on
// 2025-11-12–2026-11-12. Stochastic exit branch; training expectancy 915.56,
// PF 7.333, net PnL 21,883.86 RUB, max drawdown 4.99%, with 10 and 15
// completed trades in the two training years. The final year had 11 trades,
// net PnL -1,049.94 RUB, PF 0.898, and max drawdown 8.76%; it failed the
// pre-set positive-PnL and PF > 1 gates. Parameters are installed at the
// user's request and are not validated for live use.
// Params returns the selected research parameters for OZON.
func Params() core.Params {
	return core.Params{
		ATRPeriod:                 14,
		MinGapATR:                 0.3,
		MaxWaitBars:               45,
		StochPeriod:               14,
		UseStochExit:              1,
		StopATR:                   1.5,
		TakeProfitATR:             1.5,
		HTFTrendEMA:               0,
		RequireEntryAboveFVGLower: 1,
	}
}
