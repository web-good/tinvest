// Package astr provides default parameters for ASTR in the FVG imbalance backtest.
package astr

import "tinvest/internal/service/trading_strategy/fvg_imbalance/strategy/core"

// Params returns the uncalibrated default parameters for ASTR.
func Params() core.Params { return core.DefaultParams() }
