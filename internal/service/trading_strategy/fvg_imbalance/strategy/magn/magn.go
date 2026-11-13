// Package magn provides calibrated parameters for MAGN in the FVG imbalance backtest.
package magn

import "tinvest/internal/service/trading_strategy/fvg_imbalance/strategy/core"

// Params returns the MAGN parameters. The stochastic-exit branch was selected by
// expectancy on 2023-11-12–2025-11-12 (13 and 13 closed trades in the two years)
// and checked on 2025-11-12–2026-11-12 (24 trades). Each period met the run's
// positive net PnL, PF > 1 and max drawdown <= 10% limits after 0.05% per side.
func Params() core.Params {
	return core.Params{
		ATRPeriod:                 14,
		MinGapATR:                 0.25,
		MaxWaitBars:               45,
		StochPeriod:               11,
		UseStochExit:              1,
		StopATR:                   0.5,
		TakeProfitATR:             0,
		HTFTrendEMA:               0,
		RequireEntryAboveFVGLower: 1,
	}
}
