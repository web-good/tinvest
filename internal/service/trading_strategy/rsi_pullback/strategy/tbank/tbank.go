// Package tbank supplies the ticker and starting rsi_pullback Params for T (T-Bank).
//
// The transferability hypothesis this package started from — seed T with GAZP's post-grid
// literal and see whether it holds up — is retired. T has since been calibrated on its own
// data (report reports/T/T_rsi_pullback_Minutes30_20260731_134407.md, 67 trades, in-sample
// profit factor 1.312), and the literal below is that run's result, not a copy of GAZP's (which
// is 4/25/70/10/70/TP 0.7 vs T's 5/20/65/20/100/TP 1.5). Two consequences follow. First, this
// package does NOT track core.DefaultParams(): a change to the baseline must not reach it
// silently, because these values came from T's own calibration, not the shared default. Second,
// in-sample PF is not an edge claim — this literal has NOT been validated by walk-forward OOS,
// so treat it as a calibration starting point, not a trade-ready configuration.
//
// The package is named tbank rather than t: a one-letter package reads as t.DefaultParams() at
// the call site and collides with the t *testing.T idiom. The exchange ticker stays "T".
package tbank

import "tinvest/internal/service/trading_strategy/rsi_pullback/strategy/core"

// Ticker is the instrument this package configures.
const Ticker = "T"

// DefaultParams returns T's starting rsi_pullback parameters: T's own calibrated literal
// (in-sample only, not yet validated by walk-forward OOS).
func DefaultParams() core.Params {
	return core.Params{
		RSIPeriod:       5,
		RSILower:        20,
		RSIUpper:        65,
		EMAFast:         20,
		EMASlow:         100,
		DailyATRPeriod:  14,
		UseDayATRGate:   1,
		FreshDayATR:     0,
		SpentDayATR:     0.9,
		StopDailyATR:    0.5,
		TPDailyATR:      1.5,
		UseVolume:       0,
		VolBaseDays:     5,
		VolLookbackBars: 3,
		VolMult:         1.2,
		UseRSIExit:      1,
		UseTrail:        0,
		TrailDailyATR:   0.5,
	}
}
