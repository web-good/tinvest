// Package etln provides default parameters for ETLN in the FVG imbalance backtest.
package etln

import "tinvest/internal/service/trading_strategy/fvg_imbalance/strategy/core"

// Params returns the uncalibrated default parameters for ETLN.
func Params() core.Params { return core.DefaultParams() }
