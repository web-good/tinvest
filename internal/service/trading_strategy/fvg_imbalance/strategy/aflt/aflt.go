// Package aflt provides AFLT parameters for the FVG imbalance backtest.
package aflt

import "tinvest/internal/service/trading_strategy/fvg_imbalance/strategy/core"

// Params returns the AFLT parameters used by backtest and live trading.
// An earlier candidate with StopATR 1.5 and TakeProfitATR 1 failed the
// 2025-11-15–2026-11-15 holdout trade minimum (8 instead of 10). Those results
// do not validate the current stop and target values; see the research report.
func Params() core.Params {
	return core.Params{
		ATRPeriod:                 14,
		MinGapATR:                 0.25,
		MaxWaitBars:               30,
		StochPeriod:               14,
		UseStochExit:              1,
		StopATR:                   0.5,
		TakeProfitATR:             1.5,
		HTFTrendEMA:               120,
		RequireEntryAboveFVGLower: 0,
	}
}
