// Package upro provides exploratory parameters for UPRO in the FVG imbalance backtest.
package upro

import "tinvest/internal/service/trading_strategy/fvg_imbalance/strategy/core"

// Params returns the exploratory UPRO M30 parameters selected by expectancy.
// Training 2023-11-15–2025-11-15: 14 and 12 closed trades in the two years,
// net PnL +16635.66 RUB, PF 7.845, max drawdown 3.48%. The stochastic-exit
// candidate failed the 2025-11-15–2026-11-15 validation: 16 trades, net PnL
// -4011.39 RUB, PF 0.503, max drawdown 7.37%; the preset is not validated.
// The run required positive net PnL, PF > 1 and max drawdown <= 10%.
// Reproduction: data/params/fvg_imbalance/upro/research_2026-11/.
func Params() core.Params {
	return core.Params{
		ATRPeriod:                 14,
		MinGapATR:                 0.4,
		MaxWaitBars:               30,
		StochPeriod:               10,
		UseStochExit:              0,
		StopATR:                   0.5,
		TakeProfitATR:             1.5,
		HTFTrendEMA:               0,
		RequireEntryAboveFVGLower: 1,
	}
}
