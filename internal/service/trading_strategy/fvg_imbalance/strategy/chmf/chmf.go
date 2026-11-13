// Package chmf provides default parameters for CHMF in the FVG imbalance backtest.
package chmf

import "tinvest/internal/service/trading_strategy/fvg_imbalance/strategy/core"

// Params returns the best training candidate from the 2023-11-12—2025-11-12
// CHMF calibration (UseStochExit=0, expectancy; 10/11 closed trades by year,
// Net PnL +17,546.98 RUB, PF 2.033, Max DD 6.01%). It failed the reserved
// 2025-11-12—2026-11-12 check (14 trades, Net PnL -8,586.50 RUB, PF 0.536,
// Max DD 13.98%); this literal is unverified out of sample.
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
		RequireEntryAboveFVGLower: 1,
	}
}
