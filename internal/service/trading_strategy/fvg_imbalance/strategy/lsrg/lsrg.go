// Package lsrg provides default parameters for LSRG in the FVG imbalance backtest.
package lsrg

import "tinvest/internal/service/trading_strategy/fvg_imbalance/strategy/core"

// Params returns the uncalibrated default parameters for LSRG.
func Params() core.Params { return core.DefaultParams() }
