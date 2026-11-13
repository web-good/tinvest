// Package sfin provides parameters for SFIN in the FVG imbalance backtest.
package sfin

import "tinvest/internal/service/trading_strategy/fvg_imbalance/strategy/core"

// Params returns the best SFIN parameters on the 2023-11-12 to 2025-11-12
// training window, with 15 and 15 closed trades in its two years. The stochastic
// exit candidate had expectancy 1229.26 and max drawdown 6.71% on training.
// It failed the 2025-11-12 to 2026-11-12 final test: 13 trades, net PnL
// -15496.50, PF 0.434, max drawdown 25.02% (limits: net PnL > 0, PF > 1,
// drawdown <= 10%). It is installed at the user's explicit request.
func Params() core.Params {
	return core.Params{
		ATRPeriod: 14, MinGapATR: 0.3, MaxWaitBars: 30, StochPeriod: 10,
		UseStochExit: 1, StopATR: 1, TakeProfitATR: 0, HTFTrendEMA: 0,
		RequireEntryAboveFVGLower: 0,
	}
}
