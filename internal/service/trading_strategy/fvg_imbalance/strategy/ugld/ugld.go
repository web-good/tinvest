// Package ugld provides default parameters for UGLD in the FVG imbalance backtest.
package ugld

import "tinvest/internal/service/trading_strategy/fvg_imbalance/strategy/core"

// Params returns the uncalibrated default parameters for UGLD.
func Params() core.Params { return core.DefaultParams() }
