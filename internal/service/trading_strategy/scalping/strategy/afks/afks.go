package afks

import (
	"tinvest/internal/service/trading_strategy/scalping/strategy/adaptive"
)

// Ticker is the instrument this config targets.
const Ticker = "AFKS"

// DefaultParams returns generic, NOT-yet-calibrated starting values for AFKS.
// The HTF daily trend filter is disabled (0) until calibration justifies it.
func DefaultParams() adaptive.Params {
	return adaptive.Params{
		EMAPeriod:         21,
		ADXPeriod:         14,
		ADXTrendLevel:     25,
		ADXRangeLevel:     20,
		RSIPeriod:         14,
		RSITrendLevel:     45,
		RSIRangeLevel:     35,
		PullbackWindow:    5,
		DonchianPeriod:    20,
		ATRPeriod:         14,
		SLMult:            1.0,
		TrailMult:         2.5,
		ChandelierWindow:  20,
		EMATouchTol:       0.002,
		BandTol:           0.003,
		TrendFilterPeriod: 0,
	}
}

// New returns the adaptive strategy bound to AFKS with its baseline defaults.
func New() *adaptive.Strategy { return adaptive.NewWithParams(Ticker, DefaultParams()) }
