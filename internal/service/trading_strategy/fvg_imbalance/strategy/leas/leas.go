// Package leas provides default parameters for LEAS in the FVG imbalance backtest.
package leas

import "tinvest/internal/service/trading_strategy/fvg_imbalance/strategy/core"

// Params returns the uncalibrated default parameters for LEAS.
func Params() core.Params { return core.DefaultParams() }
