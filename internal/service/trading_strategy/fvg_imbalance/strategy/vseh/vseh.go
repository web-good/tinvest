// Package vseh provides default parameters for VSEH in the FVG imbalance backtest.
package vseh

import "tinvest/internal/service/trading_strategy/fvg_imbalance/strategy/core"

// Params returns the uncalibrated default parameters for VSEH.
func Params() core.Params { return core.DefaultParams() }
