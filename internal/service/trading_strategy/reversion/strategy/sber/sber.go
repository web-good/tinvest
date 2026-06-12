// Package sber supplies the ticker and starting reversion Params for SBER (Sberbank).
// Starting values mirror the generic defaults; calibrate with -calibrate and then
// hardcode the winning combination here.
package sber

import "tinvest/internal/service/trading_strategy/reversion/strategy/core"

// Ticker is the instrument this package configures.
const Ticker = "SBER"

// DefaultParams returns SBER's starting reversion parameters (pre-calibration).
func DefaultParams() core.Params {
	return core.Params{
		FastEMA: 50, SlowEMA: 200,
		RSIPeriod: 6, RSIOversold: 40, RSIOverbought: 70,
		EntryMode:   0,
		VolLookback: 20, VolMultiplier: 1.2,
		UseStoch: 0, StochPeriod: 14, StochSmooth: 3, StochOversold: 20,
		StopLossPct: 0.03, MaxHoldBars: 24, ATRPeriod: 14,
	}
}
