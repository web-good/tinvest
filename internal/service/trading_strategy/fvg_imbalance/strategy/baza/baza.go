// Package baza provides default parameters for BAZA in the FVG imbalance backtest.
package baza

import "tinvest/internal/service/trading_strategy/fvg_imbalance/strategy/core"

// Params returns the uncalibrated default parameters for BAZA.
func Params() core.Params { return core.DefaultParams() }
