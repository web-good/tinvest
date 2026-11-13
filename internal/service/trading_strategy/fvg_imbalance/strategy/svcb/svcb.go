// Package svcb provides default parameters for SVCB in the FVG imbalance backtest.
package svcb

import "tinvest/internal/service/trading_strategy/fvg_imbalance/strategy/core"

// Params returns the uncalibrated default parameters for SVCB.
func Params() core.Params { return core.DefaultParams() }
