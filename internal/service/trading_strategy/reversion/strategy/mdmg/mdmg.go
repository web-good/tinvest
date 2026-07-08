// Package mdmg supplies the ticker and calibrated reversion Params for MDMG (MD Medical).
package mdmg

import "tinvest/internal/service/trading_strategy/reversion/strategy/core"

// Ticker is the instrument this package configures.
const Ticker = "MDMG"

// DefaultParams returns MDMG's calibrated reversion parameters.
//
// UseBreakeven is ON with a TIGHT arm (0.5×EntryATR): a loss analysis
// (reports/_analysis/reversion_loss_analysis_2026-06-16.md) showed losing trades ran into
// profit then reversed into the ATR stop (give-back). An A/B on the OOS holdout improved
// MDMG from -985 (PF 0.87) to -130 (PF 0.98) with arm 0.5; arm >= 1.0 never armed (no
// effect). Other values are the 2026-06-16 calibration winner; the ATR stop is KEPT
// (removing it made MDMG worse).
func DefaultParams() core.Params {
	return core.Params{
		UseTrend: 1, FastEMA: 5, SlowEMA: 100,
		RSIPeriod:    6,
		RSIOversold:  30,
		StochKPeriod: 14, StochDSmooth: 1, StochOversold: 20,
		UseATRStop: 1, ATRPeriod: 14, StopATRMult: 0.8,
		UseVolume: 1, VolAvgPeriod: 20, VolMult: 1.2,
		UseOverbought: 1, RSIOverbought: 70, StochOverbought: 80,
		UseRSI50:        1,
		HTFTrendEMA:     10,
		UseBreakeven:    1,
		BreakevenArmATR: 0.5,
	}
}
