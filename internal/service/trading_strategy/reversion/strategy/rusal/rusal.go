// Package rusal supplies the ticker and starting reversion Params for RUAL (RUSAL).
// Starting values mirror the generic defaults; calibrate with -calibrate and then
// hardcode the winning combination here.
package rusal

import "tinvest/internal/service/trading_strategy/reversion/strategy/core"

// Ticker is the instrument this package configures.
const Ticker = "RUAL"

// DefaultParams returns RUAL's starting reversion parameters (pre-calibration).
func DefaultParams() core.Params {
	return core.Params{
		UseTrend: 1, FastEMA: 10, SlowEMA: 200,
		RSIOversold:  30,
		StochKPeriod: 14, StochDSmooth: 1, StochOversold: 20,
		UseATRStop: 1, ATRPeriod: 14, StopATRMult: 1.0,
		UseVolume: 1, VolAvgPeriod: 20, VolMult: 1.0,
		UseOverbought: 1, RSIOverbought: 70, StochOverbought: 80,

		RSIPeriod: 6,
	}
}
