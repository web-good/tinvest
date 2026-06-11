// Package ydex supplies the ticker and momentum Params for YDEX (Яндекс).
// Values currently mirror the generic baseline — UNCALIBRATED. Run the backtest
// with -calibrate and hardcode the winning combination here (MACD periods are
// tuned per ticker).
//
// Data caveat: YDEX is the post-relisting ticker for Yandex (trading resumed
// ~Aug 2024 after the MOEX symbol change from YNDX). Calibrate only on a window
// that STARTS after the relisting — otherwise the price gap is counted as a real
// move and corrupts the metrics. Constrain the window via the backtest -months flag.
package ydex

import "tinvest/internal/service/trading_strategy/momentum/strategy/core"

// Ticker is the instrument this package configures.
const Ticker = "YDEX"

// DefaultParams returns YDEX's momentum parameters (uncalibrated generic baseline).
func DefaultParams() core.Params {
	return core.Params{
		EMAPeriod: 200, MACDFast: 12, MACDSlow: 26, MACDSignal: 9, MACDBelowZeroOnly: 1,
		VolLookback: 20, VolMultiplier: 1.2, DailyATRPeriod: 14, MaxDailyATRUsed: 0.6,
		ATRPeriod: 14, SwingLowWindow: 10, SLMult: 0.5, TakeProfitRR: 2.0, MinRR: 1.5,
		MinATRFrac: 0.003, UseTrail: 0, TrailMult: 2.5, ChandelierWindow: 20, TrailArmATR: 1.0,
		CooldownBars: 0, DailyTrendPeriod: 0,
		UseMACDExit: 0, RSIPeriod: 0, RSIOverbought: 70,
	}
}
