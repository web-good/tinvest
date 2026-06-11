// Package afks supplies the ticker and calibrated momentum Params for AFKS (AFK Sistema).
// Starting values mirror the generic defaults; calibrate with -calibrate and then
// hardcode the winning combination here (MACD periods are tuned per ticker).
package afks

import "tinvest/internal/service/trading_strategy/momentum/strategy/core"

// Ticker is the instrument this package configures.
const Ticker = "AFKS"

// DefaultParams returns AFKS's calibrated momentum parameters.
// Grid winner from 2026-06-09 (11664 combos, ranked by profit_factor):
// PF 2.12, net +13691, win rate 52.9%, max DD 4665, 17 trades over 12 months.
// The grid left DailyTrendPeriod=0 (AFKS had no Sep–Dec chop cluster to filter)
// and CooldownBars=0. Re-run -calibrate and update if the instrument's regime shifts.
func DefaultParams() core.Params {
	return core.Params{
		EMAPeriod: 200, MACDFast: 10, MACDSlow: 26, MACDSignal: 9, MACDBelowZeroOnly: 0,
		VolLookback: 20, VolMultiplier: 1.0, DailyATRPeriod: 14, MaxDailyATRUsed: 0.6,
		ATRPeriod: 14, SwingLowWindow: 10, SLMult: 1.0, TakeProfitRR: 2.0, MinRR: 1.5,
		MinATRFrac: 0.003, UseTrail: 0, TrailMult: 2.5, ChandelierWindow: 20, TrailArmATR: 1.0,
		CooldownBars: 0, DailyTrendPeriod: 0,
		UseMACDExit: 0, RSIPeriod: 0, RSIOverbought: 70,
	}
}
