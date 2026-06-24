// Package astr supplies the ticker and reversion Params for ASTR (Группа Астра / Astra Group).
package astr

import "tinvest/internal/service/trading_strategy/reversion/strategy/core"

// Ticker is the instrument this package configures.
const Ticker = "ASTR"

// DefaultParams returns ASTR's reversion parameters.
//
// 2026-06-22: BASELINE — neutral params, registered for manual/staged tuning.
// ASTR (Группа Астра) IPO'd on MOEX in October 2023, giving ~32 months of history —
// a short window like EUTR, so walk-forward runs train 12 / OOS 6 over -months 31
// (~3 folds). NOTE: only an ASTR_Day1.json candle cache exists; the reversion
// strategy runs on Hour1, so Hour1 candles must be fetched before calibration.
//
// These are bare entry-core params (dual RSI/Stoch oversold + always-on RSIOS/EMAX
// exits) with every optional gate OFF — a clean starting point for hand-picking
// parameters via the staged per-gate grids in data/params/astr/.
func DefaultParams() core.Params {
	return core.Params{
		// Ядро входа + всегда-включённый выход EMAX.
		UseTrend: 0, FastEMA: 10, SlowEMA: 200,
		RSIPeriod: 14, RSIOversold: 25,
		StochKPeriod: 14, StochDSmooth: 3, StochOversold: 20,
		// RSI50 momentum-fade выход — ВКЛ (всегда-включённый, гридами не перебирается).
		UseRSI50: 0,
		// Опциональные гейты входа — ВЫКЛ.
		HTFTrendEMA: 0,
		UseRegime:   0, ADXPeriod: 14, ADXMax: 30,
		UseVolume: 0, VolAvgPeriod: 30, VolMult: 1.8,
		// Опциональные выходы/стопы — ВЫКЛ.
		UseOverbought: 0, RSIOverbought: 70, StochOverbought: 80,
		UseBreakeven: 0, BreakevenArmATR: 0.5,
		UseTrail: 0, TrailATRMult: 1.5,
		UseATRStop: 0, ATRPeriod: 14, StopATRMult: 1.0,
		CatStopATRMult: 0,
	}
}
