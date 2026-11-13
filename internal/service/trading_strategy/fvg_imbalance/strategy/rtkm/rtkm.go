// Package rtkm provides default parameters for RTKM in the FVG imbalance backtest.
package rtkm

import "tinvest/internal/service/trading_strategy/fvg_imbalance/strategy/core"

// Params returns the uncalibrated default parameters for RTKM.
func Params() core.Params { return core.DefaultParams() }
