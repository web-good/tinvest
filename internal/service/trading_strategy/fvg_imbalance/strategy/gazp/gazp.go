// Package gazp holds the GAZP baseline for the backtest-only M30 FVG strategy.
package gazp

import "tinvest/internal/service/trading_strategy/fvg_imbalance/strategy/core"

// Params returns the explicit ticker literal, independent of future core defaults.
func Params() core.Params {
	return core.Params{
		ATRPeriod: 14, MinGapATR: 0.8, MaxRetestBars: 8,
		RequireCloseAboveZone: 0, TakeProfitR: 1.5,
		RequireFirstRetest: 0, RequireBullishBOS: 0, RequireLiquiditySweep: 0,
		SwingK: 2, StructureLookbackBars: 48, MaxSweepToFVGBars: 8,
		HTFTrendEMA: 0, MaxEntryDistanceATR: 0, MaxHoldBars: 0, StopBufferATR: 0,
	}
}
