// Package mgnt provides calibrated research parameters for MGNT in the FVG imbalance backtest.
package mgnt

import "tinvest/internal/service/trading_strategy/fvg_imbalance/strategy/core"

// Params returns the best training candidate from 2026-11-13.
// Training: 2023-11-13–2025-11-13, no-stochastic-exit branch, expectancy ranking;
// 21/11 closed trades in the two training years. Final year: 2025-11-13–2026-11-13,
// 20 closed trades, Net PnL +8,397.79 RUB, PF 1.308, Max DD 11.30% (limit 10%);
// one position remained open at the end. The candidate failed the pre-set drawdown
// gate and is recorded here at the user's request, not as a validated calibration.
func Params() core.Params {
	return core.Params{
		ATRPeriod:                 14,
		MinGapATR:                 0.3,
		MaxWaitBars:               40,
		StochPeriod:               9,
		UseStochExit:              0,
		StopATR:                   0.5,
		TakeProfitATR:             1.5,
		HTFTrendEMA:               0,
		RequireEntryAboveFVGLower: 1,
	}
}
