// Package ugld supplies the ticker and calibrated reversion Params for UGLD (Yuzhuralzoloto / ЮГК).
package ugld

import "tinvest/internal/service/trading_strategy/reversion/strategy/core"

// Ticker is the instrument this package configures.
const Ticker = "UGLD"

// DefaultParams returns UGLD's reversion parameters.
//
// 2026-06-19: calibrated configuration, validated by rolling walk-forward (train 12 /
// OOS 6 over 36 months: pooled OOS PF 3.39, +26.4%, 25 trades, win 32%, max DD 2.9%).
// All three OOS folds were profitable (PF 3.81 / 6.88 / 2.05). UGLD topped the daily-ATR
// volatility screen (mean ATR% ~4.4 over 6 months, deep liquidity), which makes it a
// natural mean-reversion candidate, so it is registered as a first-class reversion
// ticker.
//
// This is a low-frequency, payoff-asymmetric profile: the edge comes from ~6 OB
// take-profit exits over 16 months (avg win/loss ratio ~7:1), while Breakeven and the
// ATR trail cap the many failed entries at near-zero. Exposure is only ~3% of bars.
// Because the profit rests on a handful of OB exits, size UGLD as one ticker in a
// basket, not a large solo position.
//
// Entry gates Range-regime (ADX) + Volume are ON; the trend (UseTrend) and 4H-HTF trend
// (HTFTrendEMA) gates are OFF — the cal_htf walk-forward picked HTFTrendEMA=0 in every
// fold, i.e. dropping the HTF gate beat every period. Exits OB take-profit, Breakeven
// and ATR trail are ON; ATR stop (UseATRStop) is OFF, so the always-on RSIOS
// (failed-bounce breakdown) and EMAX (bearish EMA cross) exits stay live. UseRSI50 is
// OFF. CatStopATRMult=0 → fixed -fraction sizing (no catastrophic-stop / risk sizing).
func DefaultParams() core.Params {
	return core.Params{
		// Ядро входа + линии всегда-включённого выхода EMAX — калиброваны.
		UseTrend: 0, FastEMA: 10, SlowEMA: 150,
		RSIPeriod: 7, RSIOversold: 20,
		StochKPeriod: 14, StochDSmooth: 3, StochOversold: 20,
		// RSI50 momentum-fade выход — ВЫКЛ.
		UseRSI50: 0,
		// Гейты входа: режим (ADX) + объём — ВКЛ. Тренд и 4H-HTF тренд — ВЫКЛ
		// (cal_htf во всех фолдах выбрал HTFTrendEMA=0).
		HTFTrendEMA: 0,
		UseRegime:   1, ADXPeriod: 14, ADXMax: 30,
		UseVolume: 1, VolAvgPeriod: 30, VolMult: 1.8,
		// Выходы: OB-тейк, безубыток и ATR-трейл — ВКЛ; ATR-стоп — ВЫКЛ.
		UseOverbought: 1, RSIOverbought: 70, StochOverbought: 80,
		UseBreakeven: 1, BreakevenArmATR: 0.5,
		UseTrail: 1, TrailATRMult: 1.5,
		UseATRStop: 0, ATRPeriod: 14, StopATRMult: 1.0,
		CatStopATRMult: 0,
	}
}
