// Package head provides default parameters for HEAD in the FVG imbalance backtest.
package head

import "tinvest/internal/service/trading_strategy/fvg_imbalance/strategy/core"

// Params returns the uncalibrated default parameters for HEAD.
func Params() core.Params { return core.DefaultParams() }
