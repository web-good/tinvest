// Package posi holds the calibrated POSI parameters for the FVG imbalance backtest.
package posi

import "tinvest/internal/service/trading_strategy/fvg_imbalance/strategy/core"

// Params returns the selected POSI calibration candidate.
// It was selected on 2023-11-12—2025-11-12 (MSK), using the stochastic-exit
// branch and expectancy. Training produced 27 closed trades, 12 in the first
// year and 15 in the second; Net PnL +28,365.88 RUB, PF 12.056, Max DD 2.78%.
// The reserved 2025-11-12—2026-11-12 check produced 17 closed trades,
// +19,843.99 RUB Net PnL, PF 7.826, and Max DD 3.39%. Both periods met the
// pre-set gates: at least 10 trades in each training year and in holdout,
// positive after-cost Net PnL, PF > 1, and Max DD <= 10%.
func Params() core.Params {
	return core.Params{
		ATRPeriod:                 14,
		MinGapATR:                 0.3,
		MaxWaitBars:               45,
		StochPeriod:               9,
		UseStochExit:              1,
		StopATR:                   1,
		TakeProfitATR:             2,
		HTFTrendEMA:               0,
		RequireEntryAboveFVGLower: 0,
	}
}
