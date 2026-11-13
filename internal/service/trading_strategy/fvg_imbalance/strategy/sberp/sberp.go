// Package sberp provides parameters for SBERP in the FVG imbalance backtest.
package sberp

import "tinvest/internal/service/trading_strategy/fvg_imbalance/strategy/core"

// Params returns the research candidate selected on 2023-11-12..2025-11-12
// and checked on 2025-11-12..2026-11-12. The stochastic-exit branch used
// expectancy ranking; closed trades were 15 and 25 in the training years and
// 33 in validation. Limits: positive net PnL, PF > 1, drawdown <= 10%, and
// at least 10 closed trades per year. Validation: net RUB 4190.13, PF 1.960,
// drawdown 3.57%. The no-stochastic branch was not fully screened for percentage
// drawdown before validation; see data/params/fvg_imbalance/sberp/research_2026-11/outcome.md.
func Params() core.Params {
	return core.Params{
		ATRPeriod:                 14,
		MinGapATR:                 0.1,
		MaxWaitBars:               15,
		StochPeriod:               21,
		UseStochExit:              1,
		StopATR:                   1.5,
		TakeProfitATR:             1,
		HTFTrendEMA:               200,
		RequireEntryAboveFVGLower: 0,
	}
}
