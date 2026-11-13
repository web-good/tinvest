// Package vtbr provides calibrated parameters for VTBR in the FVG imbalance backtest.
package vtbr

import "tinvest/internal/service/trading_strategy/fvg_imbalance/strategy/core"

// Params returns the VTBR calibration validated on 2026-11-12.
// Training: 2023-11-12–2025-11-12, stochastic exit branch, expectancy ranking;
// 10/22 closed trades in the two training years. Final year: 2025-11-12–2026-11-12,
// 43 closed trades, Net PnL +2,668.13 RUB after commission, PF 1.191, Max DD 4.77%.
// Gates: >=10 trades per training year, >=20 total; positive Net PnL, PF >1,
// Max DD <=10%; final-year minimum 10 closed trades with the same quality gates.
func Params() core.Params {
	return core.Params{
		ATRPeriod:                 14,
		MinGapATR:                 0.1,
		MaxWaitBars:               30,
		StochPeriod:               9,
		UseStochExit:              1,
		StopATR:                   1,
		TakeProfitATR:             2,
		HTFTrendEMA:               50,
		RequireEntryAboveFVGLower: 1,
	}
}
