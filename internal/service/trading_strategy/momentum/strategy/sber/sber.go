// Package sber supplies the ticker and calibrated momentum Params for SBER (Sberbank).
// Starting values mirror the generic defaults; calibrate with -calibrate and then
// hardcode the winning combination here (MACD periods are tuned per ticker).
package sber

import "tinvest/internal/service/trading_strategy/momentum/strategy/core"

// Ticker is the instrument this package configures.
const Ticker = "SBER"

// DefaultParams returns SBER's momentum parameters (uncalibrated baseline).
// Entry fires on a MACD↔RSI confluence. EMAPeriod=150 and MACDSlow=18 are
// SBER-specific; all other fields use the generic baseline.
func DefaultParams() core.Params {
	return core.Params{
		EMAPeriod: 150, MACDFast: 12, MACDSlow: 18, MACDSignal: 9, MACDBelowZeroOnly: 1,
		VolLookback: 20, VolMultiplier: 1.2,
		ATRPeriod: 14, SwingLowWindow: 10, SLMult: 0.5, TakeProfitRR: 2.0, MinRR: 1.5,
		UseTrail: 0, TrailMult: 2.5, ChandelierWindow: 20, TrailArmATR: 1.0,
		UseMACDExit: 0, RSIPeriod: 14, RSICrossLevel: 50, RSIOverbought: 70,
		SignalValidBars: 0,
	}
}
