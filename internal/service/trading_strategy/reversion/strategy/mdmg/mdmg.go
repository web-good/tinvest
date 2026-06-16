// Package mdmg supplies the ticker and calibrated reversion Params for MDMG (MD Medical).
package mdmg

import "tinvest/internal/service/trading_strategy/reversion/strategy/core"

// Ticker is the instrument this package configures.
const Ticker = "MDMG"

// DefaultParams returns MDMG's calibrated reversion parameters.
//
// UseBreakeven is ON: a loss analysis (reports/_analysis/reversion_loss_analysis_2026-06-16.md)
// showed 7 of 12 losing trades ran >+1% in favor then reversed into the ATR stop (give-back).
// The breakeven floor (arm at 1.0×EntryATR) cuts those back to ~0. Other values are the
// 2026-06-16 calibration winner; the ATR stop is KEPT (removing it made MDMG worse).
func DefaultParams() core.Params {
	return core.Params{
		UseTrend: 1, FastEMA: 5, SlowEMA: 100,
		RSIPeriod:    6,
		RSIOversold:  30,
		StochKPeriod: 14, StochDSmooth: 1, StochOversold: 20,
		UseATRStop: 1, ATRPeriod: 14, StopATRMult: 0.8,
		UseVolume: 1, VolAvgPeriod: 20, VolMult: 1.2,
		UseOverbought: 1, RSIOverbought: 70, StochOverbought: 80,
		HTFTrendEMA:     10,
		UseBreakeven:    1,
		BreakevenArmATR: 1.0,
	}
}
