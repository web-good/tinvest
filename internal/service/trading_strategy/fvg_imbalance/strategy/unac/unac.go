// Package unac provides default parameters for UNAC in the FVG imbalance backtest.
package unac

import "tinvest/internal/service/trading_strategy/fvg_imbalance/strategy/core"

// Params returns the uncalibrated default parameters for UNAC.
func Params() core.Params { return core.DefaultParams() }
