// Package data provides default parameters for DATA in the FVG imbalance backtest.
package data

import "tinvest/internal/service/trading_strategy/fvg_imbalance/strategy/core"

// Params returns the uncalibrated default parameters for DATA.
func Params() core.Params { return core.DefaultParams() }
