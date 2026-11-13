// Package mdmg provides default parameters for MDMG in the FVG imbalance backtest.
package mdmg

import "tinvest/internal/service/trading_strategy/fvg_imbalance/strategy/core"

// Params returns the uncalibrated default parameters for MDMG.
func Params() core.Params { return core.DefaultParams() }
