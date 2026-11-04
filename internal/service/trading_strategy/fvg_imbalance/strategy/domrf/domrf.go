// Package domrf holds the uncalibrated DOMRF baseline for the backtest-only M30 FVG strategy.
package domrf

import "tinvest/internal/service/trading_strategy/fvg_imbalance/strategy/core"

// Params returns an explicit starting point for DOMRF calibration.
func Params() core.Params {
	return core.Params{
		ATRPeriod:             14,
		MinGapATR:             0.25,
		MaxRetestBars:         8,
		MinRetestDepthPct:     50,
		TakeProfitR:           1.5,
		RequireFirstRetest:    0,
		RequireBullishBOS:     0,
		RequireLiquiditySweep: 0,
		SwingK:                2,
		StructureLookbackBars: 48,
		MaxSweepToFVGBars:     8,
		HTFTrendEMA:           0,
		MaxEntryDistanceATR:   0,
		MaxHoldBars:           0,
		StopBufferATR:         0,
	}
}
