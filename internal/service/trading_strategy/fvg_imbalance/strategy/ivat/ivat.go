// Package ivat provides default parameters for IVAT in the FVG imbalance backtest.
package ivat

import "tinvest/internal/service/trading_strategy/fvg_imbalance/strategy/core"

// Params returns the uncalibrated default parameters for IVAT.
func Params() core.Params { return core.DefaultParams() }
