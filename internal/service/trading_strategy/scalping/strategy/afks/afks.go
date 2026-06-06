package afks

import (
	"tinvest/internal/service/trading_strategy/scalping/strategy/adaptive"
)

// Ticker is the instrument this config targets.
const Ticker = "AFKS"

// DefaultParams returns generic, NOT-yet-calibrated starting values for AFKS with the
// quality knobs opted in and the HTF daily trend filter on. Calibration refines these.
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
		TrendFilterPeriod: 100,
		TrailArmATR:       1.0,
		ADXMargin:         2.0,
		MinRR:             1.5,
		MinATRFrac:        0.003,
	}
}

// New returns the adaptive strategy bound to AFKS with its baseline defaults.
func New() *adaptive.Strategy { return adaptive.NewWithParams(Ticker, DefaultParams()) }
