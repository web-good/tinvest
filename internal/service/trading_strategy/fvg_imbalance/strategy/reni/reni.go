// Package reni provides default parameters for RENI in the FVG imbalance backtest.
package reni

import "tinvest/internal/service/trading_strategy/fvg_imbalance/strategy/core"

// Params returns the uncalibrated default parameters for RENI.
func Params() core.Params { return core.DefaultParams() }
