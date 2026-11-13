// Package aflt provides default parameters for AFLT in the FVG imbalance backtest.
package aflt

import "tinvest/internal/service/trading_strategy/fvg_imbalance/strategy/core"

// Params returns the uncalibrated default parameters for AFLT.
func Params() core.Params { return core.DefaultParams() }
