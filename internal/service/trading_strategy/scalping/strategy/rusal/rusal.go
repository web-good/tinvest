package rusal

import (
	"tinvest/internal/service/trading_strategy/scalping/strategy/adaptive"
)

// Ticker is the instrument this config targets.
const Ticker = "RUAL"

// DefaultParams returns RUAL-calibrated core params for the adaptive strategy;
// the quality knobs (TrailArmATR, ADXMargin, MinRR, MinATRFrac) are starting
// values pending calibration against RUAL history.
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
		TrendFilterPeriod: 100,   // calibrated: beats 0/50/200 across 6/12/18/24mo windows
		TrailArmATR:       1.0,   // arm trail after +1 ATR profit (kills bar-1 stop-outs)
		ADXMargin:         2.0,   // require ADX to clear its regime threshold by 2
		MinRR:             1.5,   // skip setups whose target is < 1.5x the risk
		MinATRFrac:        0.003, // skip sub-0.3%-ATR setups (anti-churn)
	}
}

// New returns the adaptive strategy bound to RUAL with its calibrated defaults.
func New() *adaptive.Strategy { return adaptive.NewWithParams(Ticker, DefaultParams()) }
