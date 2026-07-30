// Package tbank supplies the ticker and starting rsi_pullback Params for T (T-Bank).
//
// Calibration has NOT been run for this ticker. The values below are a COPY of GAZP's
// post-grid literal, taken as an explicit hypothesis that parameters transfer between liquid
// names — they are not a claim that T is tuned. Two consequences follow. First, this package
// does NOT track core.DefaultParams(): a change to the baseline must not reach it silently,
// because these values came from a different instrument's calibration. Second, once -calibrate
// picks a winning combination for T, this literal must be REPLACED — leaving GAZP's numbers in
// place after a run of T's own would ship a config the data never endorsed.
//
// The package is named tbank rather than t: a one-letter package reads as t.DefaultParams() at
// the call site and collides with the t *testing.T idiom. The exchange ticker stays "T".
package tbank

import "tinvest/internal/service/trading_strategy/rsi_pullback/strategy/core"

// Ticker is the instrument this package configures.
const Ticker = "T"

// DefaultParams returns T's starting rsi_pullback parameters: GAZP's calibrated config,
// unverified on T.
func DefaultParams() core.Params {
	return core.Params{
		RSIPeriod:       4,
		RSILower:        25,
		RSIUpper:        70,
		EMAFast:         10,
		EMASlow:         70,
		DailyATRPeriod:  14,
		UseDayATRGate:   1,
		FreshDayATR:     0,
		SpentDayATR:     0.9,
		StopDailyATR:    0.5,
		TPDailyATR:      0.7,
		UseVolume:       0,
		VolBaseDays:     7,
		VolLookbackBars: 2,
		VolMult:         1,
	}
}
