// Package msng provides default parameters for MSNG in the FVG imbalance backtest.
package msng

import "tinvest/internal/service/trading_strategy/fvg_imbalance/strategy/core"

// Params returns the uncalibrated default parameters for MSNG.
func Params() core.Params { return core.DefaultParams() }
