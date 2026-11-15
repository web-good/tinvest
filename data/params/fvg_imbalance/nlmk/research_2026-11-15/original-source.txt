// Package nlmk provides default parameters for NLMK in the FVG imbalance backtest.
package nlmk

import "tinvest/internal/service/trading_strategy/fvg_imbalance/strategy/core"

// Params returns the uncalibrated default parameters for NLMK.
func Params() core.Params { return core.DefaultParams() }
