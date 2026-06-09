// Package rusal supplies the ticker and calibrated momentum Params for RUAL.
// Starting values mirror the generic defaults; calibrate with -calibrate and then
// hardcode the winning combination here (MACD periods are tuned per ticker).
package rusal

import "tinvest/internal/service/trading_strategy/momentum/strategy/core"

// Ticker is the instrument this package configures.
const Ticker = "RUAL"

// DefaultParams returns RUAL's calibrated momentum parameters.
// Grid winner from 2026-06-09 (11664 combos, ranked by profit_factor):
// PF 2.40, net +23419, win rate 58.8%, max DD 7438, 17 trades over 12 months.
// DailyTrendPeriod=20 (daily-EMA slope filter) removed the Sep–Dec 2025 loss
// cluster, roughly halving max drawdown; CooldownBars stays 0 (filter made it
// redundant). Re-run -calibrate and update if the instrument's regime shifts.
func DefaultParams() core.Params {
	return core.Params{
		EMAPeriod: 100, MACDFast: 12, MACDSlow: 26, MACDSignal: 9, MACDBelowZeroOnly: 0,
		VolLookback: 20, VolMultiplier: 1.0, DailyATRPeriod: 14, MaxDailyATRUsed: 0.7,
		ATRPeriod: 14, SwingLowWindow: 10, SLMult: 1.0, TakeProfitRR: 2.0, MinRR: 1.5,
		MinATRFrac: 0.003, UseTrail: 0, TrailMult: 2.5, ChandelierWindow: 20, TrailArmATR: 1.0,
		CooldownBars: 0, DailyTrendPeriod: 20,
	}
}
