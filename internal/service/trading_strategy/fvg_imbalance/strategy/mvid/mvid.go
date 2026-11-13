// Package mvid provides default parameters for MVID in the FVG imbalance backtest.
package mvid

import "tinvest/internal/service/trading_strategy/fvg_imbalance/strategy/core"

// Params returns the uncalibrated default parameters for MVID.
func Params() core.Params { return core.DefaultParams() }
