// Package flot provides default parameters for FLOT in the FVG imbalance backtest.
package flot

import "tinvest/internal/service/trading_strategy/fvg_imbalance/strategy/core"

// Params returns the uncalibrated default parameters for FLOT.
func Params() core.Params { return core.DefaultParams() }
