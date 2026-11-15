// Package rtkm holds RTKM parameters for the backtest-only M30 FVG strategy.
package rtkm

import "tinvest/internal/service/trading_strategy/fvg_imbalance/strategy/core"

// Params returns the RTKM research candidate selected at the user's request.
// Training 2023-11-15–2025-11-15: stochastic exit, expectancy 1039.81 RUB,
// 10/10 closed trades by year, net +20796.15 RUB, PF 4.379, max drawdown 6.34%.
// The reserved 2025-11-15–2026-11-15 run had 7 closed trades, but an API timeout
// left a gap in its M30 candles; it does not establish the >=10-trade, positive
// net, PF > 1 and max drawdown <=10% validation gates. See the RTKM outcome report.
func Params() core.Params {
	return core.Params{
		ATRPeriod:                 14,
		MinGapATR:                 0.3,
		MaxWaitBars:               30,
		StochPeriod:               21,
		UseStochExit:              1,
		StopATR:                   0.5,
		TakeProfitATR:             2,
		HTFTrendEMA:               0,
		RequireEntryAboveFVGLower: 0,
	}
}
