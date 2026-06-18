// Package nvtk supplies the ticker and calibrated reversion Params for NVTK (Novatek).
package nvtk

import "tinvest/internal/service/trading_strategy/reversion/strategy/core"

// Ticker is the instrument this package configures.
const Ticker = "NVTK"

// DefaultParams returns NVTK's reversion parameters.
//
// 2026-06-18: calibrated configuration, validated by rolling walk-forward (train 12 /
// OOS 6 over 36 months: pooled OOS PF 4.37, +31.6%, 39 trades, win 51%). The OOS folds
// cover the weak 2024-H2 regime too — there the edge only bled a contained -1.18%
// (max DD 2.25%), then made the bulk of its money in the 2025 dip-rebound regime.
// Entry gates Volume + HTF(4H) trend are ON; exits OB take-profit, Breakeven, and ATR
// trail are ON. UseRSI50 is OFF (the RSI-50 momentum-fade exit hurt NVTK). UseATRStop
// stays OFF: a 2026-06-16 loss analysis found the ATR stop net-negative on NVTK (knocked
// out by noise on trades that later recovered). UseRegime is OFF (no ADX filter), so
// ADXPeriod/ADXMax are left at their zero value. CatStopATRMult=0 → fixed -fraction
// sizing (no catastrophic-stop / risk sizing).
func DefaultParams() core.Params {
	return core.Params{
		// Ядро входа + линии всегда-включённого выхода EMAX — калиброваны.
		UseTrend: 1, FastEMA: 10, SlowEMA: 150,
		RSIPeriod: 7, RSIOversold: 30, RSIOverbought: 70,
		StochKPeriod: 14, StochDSmooth: 1, StochOversold: 20, StochOverbought: 80,
		ATRPeriod: 14,
		// RSI50-выход momentum-fade — ВЫКЛ (на NVTK ухудшал результат).
		UseRSI50: 0,
		// Гейты входа: 4H-HTF тренд + объём — ВКЛ. Режим (ADX) — ВЫКЛ,
		// поэтому ADXPeriod/ADXMax оставлены в нуле.
		HTFTrendEMA: 150,
		UseRegime:   0,
		UseVolume:   1, VolAvgPeriod: 10, VolMult: 1.5,
		// Выходы: OB-тейк, безубыток и ATR-трейл — ВКЛ; ATR-стоп — ВЫКЛ.
		UseOverbought: 1,
		UseBreakeven:  1, BreakevenArmATR: 0.5,
		UseTrail: 1, TrailATRMult: 1.5,
		UseATRStop: 0, StopATRMult: 1.0,
		CatStopATRMult: 0,
	}
}
