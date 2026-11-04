// Package gazp holds the GAZP baseline for the backtest-only M30 FVG strategy.
package gazp

import "tinvest/internal/service/trading_strategy/fvg_imbalance/strategy/core"

// Params returns the explicit ticker literal, independent of future core defaults.
func Params() core.Params {
	return core.Params{
		ATRPeriod: 14, MinGapATR: 0.25, MaxRetestBars: 8,
		RequireCloseAboveZone: 1, TakeProfitR: 1.5,
	}
}
