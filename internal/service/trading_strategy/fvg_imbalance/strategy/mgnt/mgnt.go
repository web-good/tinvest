// Package mgnt provides default parameters for MGNT in the FVG imbalance backtest.
package mgnt

import "tinvest/internal/service/trading_strategy/fvg_imbalance/strategy/core"

// Params returns the uncalibrated default parameters for MGNT.
func Params() core.Params { return core.DefaultParams() }
