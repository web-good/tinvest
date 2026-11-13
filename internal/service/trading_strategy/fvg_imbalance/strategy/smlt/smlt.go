// Package smlt provides research parameters for SMLT in the FVG imbalance backtest.
package smlt

import "tinvest/internal/service/trading_strategy/fvg_imbalance/strategy/core"

// Params returns SMLT's research M30 parameters without a stochastic exit.
// The 2023-11-12–2025-11-12 training period had 10 / 11 closed trades by
// year, expectancy 651.8 RUB, net 13687.48 RUB, PF 1.512 and Max DD 6.99%.
// Gates were >=10 trades per year, positive net, PF >1 and Max DD <=10%.
// The adjacent MinGapATR value was unprofitable, so stability is not confirmed;
// the reserved 2025-11-12–2026-11-12 final year has not been tested. These
// research parameters were installed at the user's request. Details:
// data/params/fvg_imbalance/smlt/research_2026-11/.
func Params() core.Params {
	return core.Params{
		ATRPeriod:                 14,
		MinGapATR:                 0.3,
		MaxWaitBars:               30,
		StochPeriod:               11,
		UseStochExit:              0,
		StopATR:                   1.5,
		TakeProfitATR:             1,
		HTFTrendEMA:               0,
		RequireEntryAboveFVGLower: 0,
	}
}
