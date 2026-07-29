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
	return core.DefaultParams()
}
