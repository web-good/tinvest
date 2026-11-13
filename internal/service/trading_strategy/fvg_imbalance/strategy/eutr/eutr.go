// Package eutr provides default parameters for EUTR in the FVG imbalance backtest.
package eutr

import "tinvest/internal/service/trading_strategy/fvg_imbalance/strategy/core"

// Params returns the uncalibrated default parameters for EUTR.
func Params() core.Params { return core.DefaultParams() }
