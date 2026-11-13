// Package lent provides default parameters for LENT in the FVG imbalance backtest.
package lent

import "tinvest/internal/service/trading_strategy/fvg_imbalance/strategy/core"

// Params returns the uncalibrated default parameters for LENT.
func Params() core.Params { return core.DefaultParams() }
