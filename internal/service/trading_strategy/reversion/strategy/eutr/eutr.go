// Package eutr supplies the ticker and reversion Params for EUTR (ЕвроТранс / EuroTrans).
package eutr

import "tinvest/internal/service/trading_strategy/reversion/strategy/core"

// Ticker is the instrument this package configures.
const Ticker = "EUTR"

// DefaultParams returns EUTR's reversion parameters.
//
// 2026-06-19: calibrated configuration, validated by staged walk-forward (Hour1, 31
// months, train 12 / OOS 6, ~3 folds). Pooled OOS PF 1.953, compounded +2.35%, 13
// OOS trades, win rate 30.77%.
//
// STATISTICAL FRAGILITY — read before sizing:
//   - The trend filter (EMA5/200) cuts 159 raw entry signals down to 13 pooled OOS
//     trades over 31 months. The 1.953 PF rests on a handful of trades; treat it as
//     directional, not robust.
//   - fold2 is effectively dead: 3 trades, PF 0.000 (all losers, -0.38%). Unlike UGLD,
//     not all folds are profitable. Per-fold OOS PF: fold1 2.324 (7 trades, +2.45%),
//     fold2 0.000 (3 trades, -0.38%), fold3 2.181 (3 trades, +0.28%).
//   - The -min-trades 20 guard applies per-fold during calibration ranking, not to the
//     pooled aggregate, so the headline 1.953 is directional, not statistically robust.
//   - Win rate is low (30.77%); the edge is win-size-driven — mean-reversion bounces
//     captured by the trail/breakeven exit stack. It is sensitive to exit logic.
//
// Sizing: EUTR is registered as tradeable but must be sized SMALL — one minor basket
// position, never a solo or large position — pending confirmation on a larger sample.
// This is a stricter caveat than UGLD (which has 25 OOS trades and all-positive folds).
// Walk-forward reports under reports/EUTR_final/.
//
// ON gates: Trend (EMA5/200), HTF (4H EMA150), Overbought (RSI65 / Stoch75), Breakeven
// (0.5×ATR arm), Trail (1.5×ATR). OFF gates: Regime (ADX hurt PF), Volume (collapses
// edge), ATRStop (conflicts with trail), CatStop (0 won every fold). Always-on exits
// RSIOS, EMAX, and RSI50 momentum-fade (UseRSI50=1) remain live.
func DefaultParams() core.Params {
	return core.Params{
		// Ядро входа + всегда-включённый выход EMAX.
		UseTrend: 1, FastEMA: 5, SlowEMA: 200,
		RSIPeriod: 14, RSIOversold: 35,
		StochKPeriod: 14, StochDSmooth: 1, StochOversold: 20,
		// RSI50 momentum-fade выход — ВКЛ (всегда-включённый, гридами не перебирается).
		UseRSI50: 1,
		// Гейты входа: 4H-HTF тренд (EMA150) — ВКЛ; режим (ADX) и объём — ВЫКЛ
		// (cal_regime и cal_volume ухудшили PF во всех фолдах).
		HTFTrendEMA: 150,
		UseRegime:   0, ADXPeriod: 14, ADXMax: 30,
		UseVolume: 0, VolAvgPeriod: 30, VolMult: 1.8,
		// Выходы: OB-тейк, безубыток и ATR-трейл — ВКЛ; ATR-стоп и катстоп — ВЫКЛ.
		UseOverbought: 1, RSIOverbought: 65, StochOverbought: 75,
		UseBreakeven: 1, BreakevenArmATR: 0.5,
		UseTrail: 1, TrailATRMult: 1.5,
		UseATRStop: 0, ATRPeriod: 14, StopATRMult: 1.0,
		CatStopATRMult: 0,
	}
}
