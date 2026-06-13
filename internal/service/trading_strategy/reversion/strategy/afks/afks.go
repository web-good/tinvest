// Package afks supplies the ticker and starting reversion Params for AFKS (AFK Sistema).
// Starting values mirror the generic defaults; calibrate with -calibrate and then
// hardcode the winning combination here.
package afks

import "tinvest/internal/service/trading_strategy/reversion/strategy/core"

// Ticker is the instrument this package configures.
const Ticker = "AFKS"

// DefaultParams returns AFKS's starting reversion parameters (pre-calibration).
func DefaultParams() core.Params {
	return core.Params{
		UseTrend: 1, FastEMA: 10, SlowEMA: 200,
		RSIPeriod: 5, RSIOversold: 30,
		StochKPeriod: 14, StochDSmooth: 1, StochOversold: 20,
		UseATRStop: 0, ATRPeriod: 14, StopATRMult: 1.0,
		UseVolume: 0, VolAvgPeriod: 20, VolMult: 1.0,
		UseOverbought: 1, RSIOverbought: 70, StochOverbought: 80,
	}
}
