// Package vkco provides default parameters for VKCO in the FVG imbalance backtest.
package vkco

import "tinvest/internal/service/trading_strategy/fvg_imbalance/strategy/core"

// Params returns the uncalibrated default parameters for VKCO.
func Params() core.Params { return core.DefaultParams() }
