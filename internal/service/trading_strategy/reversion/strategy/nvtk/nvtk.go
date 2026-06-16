// Package nvtk supplies the ticker and calibrated reversion Params for NVTK (Novatek).
package nvtk

import "tinvest/internal/service/trading_strategy/reversion/strategy/core"

// Ticker is the instrument this package configures.
const Ticker = "NVTK"

// DefaultParams returns NVTK's calibrated reversion parameters.
//
// Calibrated 2026-06-16 (50mo load, last 12mo OOS holdout). UseATRStop is OFF: a loss
// analysis (reports/_analysis/reversion_loss_analysis_2026-06-16.md) showed the ATR stop
// was net-negative here — it was knocked out by noise and booked losses on trades that
// later recovered. Dropping it (middle exit reverts to RSIOS) flipped NVTK's OOS result
// from -4649 (PF 0.60) to +5249 (PF 1.70, win rate 45%, ~2:1 payoff).
func DefaultParams() core.Params {
	return core.Params{
		UseTrend: 1, FastEMA: 5, SlowEMA: 150,
		RSIPeriod:    6,
		RSIOversold:  30,
		StochKPeriod: 14, StochDSmooth: 1, StochOversold: 20,
		UseATRStop: 0, ATRPeriod: 14, StopATRMult: 0.9,
		UseVolume: 1, VolAvgPeriod: 20, VolMult: 1.2,
		UseOverbought: 1, RSIOverbought: 70, StochOverbought: 80,
		HTFTrendEMA:  100,
		UseBreakeven: 0, BreakevenArmATR: 1.0,
	}
}
