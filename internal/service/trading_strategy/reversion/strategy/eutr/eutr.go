// Package eutr supplies the ticker and reversion Params for EUTR (ЕвроТранс / EuroTrans).
package eutr

import "tinvest/internal/service/trading_strategy/reversion/strategy/core"

// Ticker is the instrument this package configures.
const Ticker = "EUTR"

// DefaultParams returns EUTR's reversion parameters.
//
// 2026-06-19: BASELINE — params reset for manual tuning. EUTR topped the
// reversion-fitness screen (score 0.869, mean ATR% 3.79, autocorr -0.112, classified
// mean-reverting), so it is registered as a reversion candidate. Hour1 history starts
// 2023-11-21 (IPO), giving ~31 months — shorter than the 36-month window used for the
// other tickers, so any walk-forward runs train 12 / OOS 6 over -months 31 (~3 folds).
//
// These are neutral baseline params (bare entry core + always-on RSIOS/EMAX exits;
// every optional gate OFF) — a starting point for hand-picking parameters. (A prior
// staged walk-forward reached pooled OOS PF 1.953 but on only 13 OOS trades with a
// dead fold2, so its config was set aside in favour of manual tuning.)
func DefaultParams() core.Params {
	return core.Params{
		// Ядро входа + всегда-включённый выход EMAX.
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
