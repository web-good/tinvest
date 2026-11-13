// Package rnft provides RNFT parameters for the FVG imbalance backtest.
package rnft

import "tinvest/internal/service/trading_strategy/fvg_imbalance/strategy/core"

// Params returns the best training-period parameters for RNFT.
// Training (2023-11-12–2025-11-12): 37 closed trades (21/16 by year), net
// +7,437.07 RUB, PF 1.509, Max DD 5.98%. The reserved final-year check
// (2025-11-12–2026-11-12) failed the quality gates: 24 trades, net -1,882.39
// RUB, PF 0.896, Max DD 10.04%. These values are recorded for research and
// have not passed out-of-sample validation. Details: data/params/fvg_imbalance/rnft/research_2026-11/.
func Params() core.Params {
	return core.Params{
		ATRPeriod:                 14,
		MinGapATR:                 0.3,
		MaxWaitBars:               30,
		StochPeriod:               9,
		UseStochExit:              1,
		StopATR:                   0.3,
		TakeProfitATR:             1.7,
		HTFTrendEMA:               0,
		RequireEntryAboveFVGLower: 0,
	}
}
