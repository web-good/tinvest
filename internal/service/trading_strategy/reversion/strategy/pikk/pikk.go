// Package pikk supplies the ticker and reversion Params for PIKK (ПИК-Специализированный застройщик / PIK Group).
package pikk

import "tinvest/internal/service/trading_strategy/reversion/strategy/core"

// Ticker is the instrument this package configures.
const Ticker = "PIKK"

// DefaultParams returns PIKK's reversion parameters.
//
// 2026-06-23: BASELINE — neutral params, registered for manual/staged tuning.
// PIKK topped the reworked Hour1 pullback-in-trend screener (#1, recovery 77.8%
// on the RSI14/recover24 config, 98.9% with RSI6, ~89 events, full 12-month Hour1
// window). IMPORTANT: that screen rank is NOT a profit prediction — the composite
// screener Score does not separate walk-forward winners from losers (see
// docs/reversion/screener.md); PIKK is a candidate to validate, not a confirmed
// edge. It must clear its own per-ticker walk-forward before being trusted.
//
// History caveat: the PIKK Hour1 candle cache spans only ~1 year (2025-06-23 →
// 2026-06-23, like RUAL — the API's hourly depth is limited), so walk-forward
// must use a modest window (e.g. train 6 / OOS 3) or a single holdout; a wide
// train window would precede available history and the calibration guard would
// reject it. Fetch more Hour1 history for a robust walk-forward when possible.
//
// These are bare entry-core params (dual RSI/Stoch oversold + always-on EMAX exit)
// with every optional gate OFF — a clean starting point for hand-picking
// parameters via the staged per-gate grids in data/params/pikk/.
func DefaultParams() core.Params {
	return core.Params{
		// Ядро входа + всегда-включённый выход EMAX.
		UseTrend: 0, FastEMA: 10, SlowEMA: 150,
		RSIPeriod: 12, RSIOversold: 25,
		StochKPeriod: 14, StochDSmooth: 3, StochOversold: 20,
		// RSI50 momentum-fade выход — ВЫКЛ (всегда-включённый, гридами не перебирается).
		UseRSI50: 0,
		// Опциональные гейты входа — ВЫКЛ.
		HTFTrendEMA: 0,
		UseRegime:   1, ADXPeriod: 20, ADXMax: 35,
		UseVolume: 1, VolAvgPeriod: 10, VolMult: 1.5,
		// Опциональные выходы/стопы — ВЫКЛ.
		UseOverbought: 0, RSIOverbought: 70, StochOverbought: 80,
		UseBreakeven: 0, BreakevenArmATR: 0.5,
		UseTrail: 1, TrailATRMult: 2.0,
		UseATRStop: 0, ATRPeriod: 14, StopATRMult: 1.0,
		CatStopATRMult: 2.0,
	}
}
