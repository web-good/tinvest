// Package banep provides default parameters for BANEP in the FVG imbalance backtest.
package banep

import "tinvest/internal/service/trading_strategy/fvg_imbalance/strategy/core"

// Params returns the uncalibrated default parameters for BANEP.
func Params() core.Params { return core.DefaultParams() }
