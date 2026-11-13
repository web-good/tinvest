// Package ozph provides default parameters for OZPH in the FVG imbalance backtest.
package ozph

import "tinvest/internal/service/trading_strategy/fvg_imbalance/strategy/core"

// Params returns the uncalibrated default parameters for OZPH.
func Params() core.Params { return core.DefaultParams() }
