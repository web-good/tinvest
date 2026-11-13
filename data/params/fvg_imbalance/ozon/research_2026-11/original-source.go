// Package ozon provides default parameters for OZON in the FVG imbalance backtest.
package ozon

import "tinvest/internal/service/trading_strategy/fvg_imbalance/strategy/core"

// Params returns the uncalibrated default parameters for OZON.
func Params() core.Params { return core.DefaultParams() }
