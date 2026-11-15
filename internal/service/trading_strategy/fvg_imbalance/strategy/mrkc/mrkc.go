// Package mrkc holds MRKC parameters for the backtest-only M30 FVG strategy.
package mrkc

import "tinvest/internal/service/trading_strategy/fvg_imbalance/strategy/core"

// Params returns the research candidate selected at the user's request.
// Training 2023-11-15–2025-11-15: stochastic exit, expectancy 698.46 RUB,
// 10/19 closed trades by year, net +20255.24 RUB, PF 5.255, max drawdown 4.04%.
// The 2025-11-15–2026-11-15 validation failed: 10 trades, net -2193.36 RUB,
// PF 0.607, max drawdown 5.90%; net must be positive and PF > 1, with at least
// 10 trades and max drawdown <= 10%. Results are sensitive to take profit.
func Params() core.Params {
	return core.Params{
		ATRPeriod:                 14,
		MinGapATR:                 0.3,
		MaxWaitBars:               15,
		StochPeriod:               9,
		UseStochExit:              0,
		StopATR:                   0.5,
		TakeProfitATR:             1.5,
		HTFTrendEMA:               0,
		RequireEntryAboveFVGLower: 1,
	}
}
