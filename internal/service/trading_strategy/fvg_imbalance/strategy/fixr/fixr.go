// Package fixr provides calibrated parameters for FIXR in the FVG imbalance backtest.
package fixr

import "tinvest/internal/service/trading_strategy/fvg_imbalance/strategy/core"

// Params returns the FIXR parameters. The 2023-11-12–2025-11-12 training
// selected the stochastic-exit branch by expectancy: 14/14 closed trades in
// consecutive years, PF 2.273, net 17740.83 RUB and max drawdown 9.15%.
// The reserved 2025-11-12–2026-11-12 test had 14 trades, PF 10.340, net
// 9480.85 RUB and max drawdown 2.27%. Each period required positive net,
// PF > 1, drawdown <= 10%, and at least 10 closed trades per year.
func Params() core.Params {
	return core.Params{ATRPeriod: 14, MinGapATR: 0.3, MaxWaitBars: 30, StochPeriod: 10, UseStochExit: 1, StopATR: 1.7, TakeProfitATR: 2, HTFTrendEMA: 0, RequireEntryAboveFVGLower: 0}
}
