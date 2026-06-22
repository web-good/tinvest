// Package eutr supplies the ticker and reversion Params for EUTR (ЕвроТранс / EuroTrans).
package eutr

import "tinvest/internal/service/trading_strategy/reversion/strategy/core"

// Ticker is the instrument this package configures.
const Ticker = "EUTR"

// DefaultParams returns EUTR's reversion parameters.
//
// EUTR topped the reversion-fitness screen (score 0.869, mean ATR% 3.79, autocorr
// -0.112, classified mean-reverting). Hour1 history starts 2023-11-21 (IPO), giving
// ~31 months — shorter than the 36-month window used for the other tickers, so
// walk-forward runs train 12 / OOS 6 over -months 31 (~3 folds).
//
// 2026-06-22: MANUALLY TUNED + walk-forward validated. Entry = trend (EMA5/100) +
// dual RSI(6)/Stoch(9) oversold; exits = overbought take-profit (RSI65/Stoch85),
// trailing stop (1.5×ATR), ATR stop (1.2×ATR), plus always-on EMAX. RSI50 OFF;
// optional gates regime/HTF/volume/breakeven/catstop all OFF (each rejected in
// isolated per-gate walk-forward sweeps — see reports/EUTR/<gate>/). The ATR stop
// (UseATRStop=1) does not disable the trailing stop; it swaps the soft RSIOS exit
// for a fixed entry−1.2×ATR floor while TRAIL keeps higher precedence.
//
// Walk-forward (degenerate fixed grid, no search; reports/EUTR/*_walkforward.md):
// pooled OOS PF 2.529, +18.70%, 47 trades, win 55%, Sortino 0.92 — all 3 folds
// positive (OOS PF 3.73/1.59/2.72), MaxDD <3.2%, OOS PF ≥ in-sample (no overfit
// signature). Clean accept. Full-history single run: PF 1.995, 69 trades.
func DefaultParams() core.Params {
	return core.Params{
		// Ядро входа + всегда-включённый выход EMAX.
		UseTrend: 1, FastEMA: 5, SlowEMA: 100,
		RSIPeriod: 6, RSIOversold: 30,
		StochKPeriod: 9, StochDSmooth: 1, StochOversold: 20,
		// RSI50 momentum-fade выход — ВКЛ (всегда-включённый, гридами не перебирается).
		UseRSI50: 0,
		// Опциональные гейты входа — ВЫКЛ.
		HTFTrendEMA: 0,
		UseRegime:   0, ADXPeriod: 14, ADXMax: 30,
		UseVolume: 0, VolAvgPeriod: 30, VolMult: 1.8,
		// Опциональные выходы/стопы — ВЫКЛ.
		UseOverbought: 1, RSIOverbought: 65, StochOverbought: 85,
		UseBreakeven: 0, BreakevenArmATR: 0.5,
		UseTrail: 1, TrailATRMult: 1.5,
		UseATRStop: 1, ATRPeriod: 14, StopATRMult: 1.2,
		CatStopATRMult: 0,
	}
}
