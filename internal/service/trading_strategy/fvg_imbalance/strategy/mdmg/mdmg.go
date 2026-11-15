// Package mdmg provides parameters for MDMG in the FVG imbalance backtest.
package mdmg

import "tinvest/internal/service/trading_strategy/fvg_imbalance/strategy/core"

// Params returns the research candidate selected by expectancy on 2023-11-15–2025-11-15.
// It closed 13 and 10 trades in the respective training years. The stochastic-exit
// candidate is sensitive to StochPeriod and MaxWaitBars. On 2025-11-15–2026-11-15
// it closed 7 trades, with negative net PnL and PF 0.840, failing the precommitted
// requirements of 10 trades, positive net PnL, PF > 1, and drawdown <= 10%.
// It is retained at the user's request as an unverified research configuration.
func Params() core.Params {
	return core.Params{
		ATRPeriod:                 14,
		MinGapATR:                 0.3,
		MaxWaitBars:               30,
		StochPeriod:               21,
		UseStochExit:              0,
		StopATR:                   0.5,
		TakeProfitATR:             1.5,
		HTFTrendEMA:               50,
		RequireEntryAboveFVGLower: 1,
	}
}
