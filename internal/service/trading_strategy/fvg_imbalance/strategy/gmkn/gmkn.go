// Package gmkn provides default parameters for GMKN in the FVG imbalance backtest.
package gmkn

import "tinvest/internal/service/trading_strategy/fvg_imbalance/strategy/core"

// Params returns the GMKN research candidate selected by expectancy with the
// stochastic exit. The 2023-11-12 to 2025-11-12 training window had 13 and 46
// closed trades in its annual blocks. The 2025-11-12 to 2026-11-12 test had 43
// closed trades, net PnL -2653.07 RUB, PF 0.906 and maximum drawdown 6.91%.
// It failed the positive net PnL and PF > 1 criteria and is retained by request.
func Params() core.Params {
	return core.Params{
		ATRPeriod:                 14,
		MinGapATR:                 0.2,
		MaxWaitBars:               30,
		StochPeriod:               14,
		UseStochExit:              1,
		StopATR:                   0.5,
		TakeProfitATR:             1.5,
		HTFTrendEMA:               100,
		RequireEntryAboveFVGLower: 0,
	}
}
