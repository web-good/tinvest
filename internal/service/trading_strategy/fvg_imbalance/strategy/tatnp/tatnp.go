// Package tatnp provides default parameters for TATNP in the FVG imbalance backtest.
package tatnp

import "tinvest/internal/service/trading_strategy/fvg_imbalance/strategy/core"

// Params returns the uncalibrated default parameters for TATNP.
func Params() core.Params { return core.DefaultParams() }
