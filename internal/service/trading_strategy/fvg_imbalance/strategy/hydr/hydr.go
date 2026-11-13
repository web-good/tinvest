// Package hydr provides default parameters for HYDR in the FVG imbalance backtest.
package hydr

import "tinvest/internal/service/trading_strategy/fvg_imbalance/strategy/core"

// Params returns the uncalibrated default parameters for HYDR.
func Params() core.Params { return core.DefaultParams() }
