// Package moex provides calibrated parameters for MOEX in the FVG imbalance backtest.
package moex

import "tinvest/internal/service/trading_strategy/fvg_imbalance/strategy/core"

// Params returns the MOEX parameters. Calibration: 2023-11-12–2025-11-12;
// validation: 2025-11-12–2026-11-12. The stochastic-exit branch was ranked by
// expectancy; closed trades were 15 and 18 in the two calibration years and
// 24 in validation. Admission required positive net PnL, PF > 1 and max
// drawdown <= 10%, plus at least 10 closed trades per year.
func Params() core.Params {
	return core.Params{
		ATRPeriod:                 14,
		MinGapATR:                 0.25,
		MaxWaitBars:               30,
		StochPeriod:               9,
		UseStochExit:              1,
		StopATR:                   1.2,
		TakeProfitATR:             1,
		HTFTrendEMA:               0,
		RequireEntryAboveFVGLower: 0,
	}
}
