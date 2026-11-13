// Package plzl provides default parameters for PLZL in the FVG imbalance backtest.
package plzl

import "tinvest/internal/service/trading_strategy/fvg_imbalance/strategy/core"

// Params returns the uncalibrated default parameters for PLZL.
func Params() core.Params { return core.DefaultParams() }
