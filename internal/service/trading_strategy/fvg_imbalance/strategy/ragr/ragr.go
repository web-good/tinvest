// Package ragr provides default parameters for RAGR in the FVG imbalance backtest.
package ragr

import "tinvest/internal/service/trading_strategy/fvg_imbalance/strategy/core"

// Params returns the uncalibrated default parameters for RAGR.
func Params() core.Params { return core.DefaultParams() }
