// Package enpg provides default parameters for ENPG in the FVG imbalance backtest.
package enpg

import "tinvest/internal/service/trading_strategy/fvg_imbalance/strategy/core"

// Params returns the uncalibrated default parameters for ENPG.
func Params() core.Params { return core.DefaultParams() }
