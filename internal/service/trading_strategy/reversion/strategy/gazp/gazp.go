// Package gazp supplies the ticker and starting reversion Params for GAZP (Gazprom).
// Starting values mirror the generic defaults; calibrate with -calibrate and then
// hardcode the winning combination here.
package gazp

import "tinvest/internal/service/trading_strategy/reversion/strategy/core"

// Ticker is the instrument this package configures.
const Ticker = "GAZP"

// DefaultParams returns GAZP's starting reversion parameters (pre-calibration).
func DefaultParams() core.Params {
	return core.Params{
		UseTrend: 1, FastEMA: 50, SlowEMA: 200,
		RSIPeriod: 14, RSIOversold: 20,
		StochKPeriod: 14, StochDSmooth: 3, StochOversold: 20,
		UseATRStop: 0, ATRPeriod: 14, StopATRMult: 1.0,
		UseVolume: 0, VolAvgPeriod: 20, VolMult: 1.0,
		UseOverbought: 1, RSIOverbought: 70, StochOverbought: 80,
		UseBreakeven: 0, BreakevenArmATR: 1.0,
	}
}
