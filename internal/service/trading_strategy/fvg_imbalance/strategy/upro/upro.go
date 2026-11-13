// Package upro provides default parameters for UPRO in the FVG imbalance backtest.
package upro

import "tinvest/internal/service/trading_strategy/fvg_imbalance/strategy/core"

// Params returns the uncalibrated default parameters for UPRO.
func Params() core.Params { return core.DefaultParams() }
