// Package phor provides parameters for PHOR in the FVG imbalance backtest.
package phor

import "tinvest/internal/service/trading_strategy/fvg_imbalance/strategy/core"

// Params returns the calibrated PHOR parameters.
// Selection 2023-11-12–2025-11-12: 15 and 20 closed trades by entry year;
// final check 2025-11-12–2026-11-12: 19 trades. Stochastic exit; selection
// expectancy 575.11 RUB. Gates: net PnL > 0, PF > 1, drawdown <= 10%,
// at least 10 closed trades per year. MaxWaitBars sensitivity is material.
func Params() core.Params {
	return core.Params{
		ATRPeriod:                 14,
		MinGapATR:                 0.25,
		MaxWaitBars:               30,
		StochPeriod:               14,
		UseStochExit:              1,
		StopATR:                   0.5,
		TakeProfitATR:             1.5,
		HTFTrendEMA:               0,
		RequireEntryAboveFVGLower: 0,
	}
}
