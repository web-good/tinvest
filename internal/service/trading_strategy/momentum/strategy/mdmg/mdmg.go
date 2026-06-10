// Package mdmg supplies the ticker and calibrated momentum Params for MDMG
// (MD Medical Group / "Мать и дитя"). Trades on MOEX since ~Oct-2024 after
// redomiciliation, so its candle history is shorter than the other tickers.
// Starting values mirror the generic defaults; calibrate with -calibrate and then
// hardcode the winning combination here (MACD periods are tuned per ticker).
package mdmg

import "tinvest/internal/service/trading_strategy/momentum/strategy/core"

// Ticker is the instrument this package configures.
const Ticker = "MDMG"

// DefaultParams returns MDMG's momentum parameters (uncalibrated baseline).
func DefaultParams() core.Params {
	return core.Params{
		EMAPeriod: 200, MACDFast: 12, MACDSlow: 26, MACDSignal: 9, MACDBelowZeroOnly: 1,
		VolLookback: 20, VolMultiplier: 1.2, DailyATRPeriod: 14, MaxDailyATRUsed: 0.6,
		ATRPeriod: 14, SwingLowWindow: 10, SLMult: 0.5, TakeProfitRR: 2.0, MinRR: 1.5,
		MinATRFrac: 0.003, UseTrail: 0, TrailMult: 2.5, ChandelierWindow: 20, TrailArmATR: 1.0,
		CooldownBars: 0, DailyTrendPeriod: 0,
	}
}
