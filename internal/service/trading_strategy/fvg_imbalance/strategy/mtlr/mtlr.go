// Package mtlr provides default parameters for MTLR in the FVG imbalance backtest.
package mtlr

import "tinvest/internal/service/trading_strategy/fvg_imbalance/strategy/core"

// Params returns the best in-sample MTLR parameters from the 2026-11 calibration.
// The candidate did not meet the frozen net-PnL and profit-factor gates on the
// reserved 2025-11-12–2026-11-12 holdout; see data/params/fvg_imbalance/mtlr/research_2026-11/outcome.md.
func Params() core.Params {
	return core.Params{
		ATRPeriod:                 14,
		MinGapATR:                 0.30,
		MaxWaitBars:               45,
		StochPeriod:               10,
		UseStochExit:              1,
		StopATR:                   1,
		TakeProfitATR:             1.5,
		HTFTrendEMA:               0,
		RequireEntryAboveFVGLower: 1,
	}
}
