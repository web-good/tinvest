// Package fesh provides default parameters for FESH in the FVG imbalance backtest.
package fesh

import "tinvest/internal/service/trading_strategy/fvg_imbalance/strategy/core"

// Params returns the uncalibrated default parameters for FESH.
func Params() core.Params { return core.DefaultParams() }
