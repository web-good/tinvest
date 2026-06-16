// Package mdmg supplies the ticker and starting reversion Params for MDMG (MD Medical).
// Starting values mirror the generic defaults; calibrate with -calibrate and then
// hardcode the winning combination here.
package mdmg

import "tinvest/internal/service/trading_strategy/reversion/strategy/core"

// Ticker is the instrument this package configures.
const Ticker = "MDMG"

// DefaultParams returns MDMG's starting reversion parameters (pre-calibration).
func DefaultParams() core.Params {
	return core.Params{
		UseTrend: 1, FastEMA: 10, SlowEMA: 200,
		RSIOversold:  30,
		StochKPeriod: 14, StochDSmooth: 1, StochOversold: 20,
		UseATRStop: 1, ATRPeriod: 14, StopATRMult: 0.8,
		UseVolume: 1, VolAvgPeriod: 20, VolMult: 1.1,
		UseOverbought: 1, RSIOverbought: 70, StochOverbought: 80,

		RSIPeriod: 6,
	}
}
