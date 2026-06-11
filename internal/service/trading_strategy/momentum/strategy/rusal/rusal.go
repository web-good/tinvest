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
// Entry fires on a MACD↔RSI confluence (RSIPeriod=9, RSICrossLevel=50, window 4).
func DefaultParams() core.Params {
	return core.Params{
		EMAPeriod: 200, MACDFast: 12, MACDSlow: 26, MACDSignal: 9, MACDBelowZeroOnly: 1,
		VolLookback: 20, VolMultiplier: 1.0,
		ATRPeriod: 14, SwingLowWindow: 10, SLMult: 1.0, TakeProfitRR: 2.0, MinRR: 1.5,
		UseTrail: 1, TrailMult: 2.5, ChandelierWindow: 20, TrailArmATR: 1.0,
		UseMACDExit: 1, RSIPeriod: 9, RSICrossLevel: 50, RSIOverbought: 70,
		SignalValidBars: 4,
	}
}
