// Package uwgn provides default parameters for UWGN in the FVG imbalance backtest.
package uwgn

import "tinvest/internal/service/trading_strategy/fvg_imbalance/strategy/core"

// Params returns the uncalibrated default parameters for UWGN.
func Params() core.Params { return core.DefaultParams() }
