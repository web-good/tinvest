// Package ugld supplies the ticker and starting rsi_pullback Params for UGLD (Yuzhuralzoloto / ЮГК).
//
// Calibration has NOT been run for this ticker: the body returns core.DefaultParams()
// unchanged rather than copying its fields, so a change to the baseline still reaches every
// uncalibrated ticker instead of silently drifting away from it. Once -calibrate picks a
// winning combination for UGLD, replace the body with an explicit literal — from that point
// the ticker must stop tracking the baseline.
package ugld

import "tinvest/internal/service/trading_strategy/rsi_pullback/strategy/core"

// Ticker is the instrument this package configures.
const Ticker = "UGLD"

// DefaultParams returns UGLD's starting rsi_pullback parameters (pre-calibration).
func DefaultParams() core.Params {
	return core.Params{
		RSIPeriod:       6,
		RSILower:        15,
		RSIUpper:        60,
		EMAFast:         20,
		EMASlow:         150,
		DailyATRPeriod:  14,
		UseDayATRGate:   1,
		FreshDayATR:     0.3,
		SpentDayATR:     0.8,
		StopDailyATR:    0.5,
		TPDailyATR:      1,
		UseVolume:       1,
		VolBaseDays:     5,
		VolLookbackBars: 3,
		VolMult:         1.2,
		UseRSIExit:      1,
		UseTrail:        1,
		TrailDailyATR:   0.5,
	}
}
