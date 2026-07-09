// Package sfin supplies the ticker and reversion Params for SFIN (ЭсЭфАй / SFI holding).
package sfin

import "tinvest/internal/service/trading_strategy/reversion/strategy/core"

// Ticker is the instrument this package configures.
const Ticker = "SFIN"

// DefaultParams returns SFIN's reversion parameters.
//
// ⚠️ 2026-06-19: CALIBRATION FAILED — DO NOT TRADE THIS TICKER LIVE on the reversion
// strategy. A full staged walk-forward (Hour1, 36 months, train 12 / OOS 6, 4 folds)
// found no out-of-sample-profitable configuration:
//   - The bare entry core is OOS-negative (pooled OOS PF 0.43, compounded -60.7%) while
//     in-sample PF is 1.4–2.4 every fold and every entry param is unstable across folds —
//     textbook overfitting. SFIN trends down hard on Hour1 and the oversold bounce the
//     entry bets on does not arrive (worst trade -21.6k, fold-4 OOS drawdown 42.7%).
//   - No single filter rescues it (best is the catastrophic stop at OOS PF 0.957, i.e.
//     ~breakeven before costs); the catstop+overbought+trail combo only reaches PF 0.942.
//
// SFIN is kept REGISTERED with this stripped baseline so the result is reproducible, but
// it must stay out of any live universe until a profitable config is found (e.g. on a
// slower timeframe). Walk-forward reports under reports/SFIN/.
//
// The baseline is the bare core: the dual RSI+Stoch oversold entry (entry params set to
// the modal cal_entry values) plus the always-on exits RSIOS, EMAX and the RSI-50
// momentum-fade (UseRSI50 ON). Every optional filter — trend, HTF trend, regime, volume,
// overbought take-profit, breakeven, ATR trail, ATR stop, catastrophic stop — is OFF; the
// disabled filters' thresholds are inert starting points carried for grid inheritance.
func DefaultParams() core.Params {
	return core.Params{
		// Ядро входа + линии всегда-включённого выхода EMAX.
		// Модальные значения из cal_entry walk-forward (параметры нестабильны
		// между фолдами; голое ядро OOS-убыточно — несущими должны стать фильтры).
		UseTrend: 0, FastEMA: 10, SlowEMA: 200,
		RSIPeriod: 14, RSIOversold: 25,
		StochKPeriod: 14, StochDSmooth: 3, StochOversold: 15,
		// RSI50 momentum-fade выход — ВКЛ (всегда-включённый, гридами не перебирается).
		UseRSI50: 1,
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
