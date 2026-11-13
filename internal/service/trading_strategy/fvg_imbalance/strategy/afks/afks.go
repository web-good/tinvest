// Package afks provides default parameters for AFKS in the FVG imbalance backtest.
package afks

import "tinvest/internal/service/trading_strategy/fvg_imbalance/strategy/core"

// Params returns the uncalibrated default parameters for AFKS.
func Params() core.Params { return core.DefaultParams() }
