// Package nvtk provides calibrated parameters for NVTK in the FVG imbalance backtest.
package nvtk

import "tinvest/internal/service/trading_strategy/fvg_imbalance/strategy/core"

// Params returns NVTK's checked M30 FVG imbalance parameters.
// Trained on 2023-11-12–2025-11-12 (10 / 15 closed trades by year) and checked
// on 2025-11-12–2026-11-12 (25 closed trades plus one open position). The
// stochastic-exit branch met the locked gates: training expectancy 361.60 RUB,
// net +9,039.93 RUB, PF 1.737, Max DD 8.43%; final net +1,324.86 RUB, PF 1.102,
// Max DD 9.76%. The final open position is mark-to-market in Net PnL and excluded
// from closed-trade count and PF. Gates: >=10 trades per training year and final
// year, positive net, PF >1, Max DD <=10%. Reproduction: data/params/fvg_imbalance/nvtk/research_2026-11/.
func Params() core.Params {
	return core.Params{
		ATRPeriod:                 14,
		MinGapATR:                 0.25,
		MaxWaitBars:               30,
		StochPeriod:               14,
		UseStochExit:              1,
		StopATR:                   1,
		TakeProfitATR:             2,
		HTFTrendEMA:               0,
		RequireEntryAboveFVGLower: 0,
	}
}
