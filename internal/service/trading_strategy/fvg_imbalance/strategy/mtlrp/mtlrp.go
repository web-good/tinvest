// Package mtlrp provides default parameters for MTLRP in the FVG imbalance backtest.
package mtlrp

import "tinvest/internal/service/trading_strategy/fvg_imbalance/strategy/core"

// Params returns the uncalibrated default parameters for MTLRP.
func Params() core.Params { return core.DefaultParams() }
