// Package cbom provides default parameters for CBOM in the FVG imbalance backtest.
package cbom

import "tinvest/internal/service/trading_strategy/fvg_imbalance/strategy/core"

// Params returns the uncalibrated default parameters for CBOM.
func Params() core.Params { return core.DefaultParams() }
