// Package rosn provides default parameters for ROSN in the FVG imbalance backtest.
package rosn

import "tinvest/internal/service/trading_strategy/fvg_imbalance/strategy/core"

// Params returns ROSN's calibrated M30 FVG imbalance parameters.
// Trained on 2023-11-12–2025-11-12 (21 / 19 closed trades by year), then
// checked on 2025-11-12–2026-11-12 (20 closed trades). The stochastic-exit
// branch passed the locked gates: expectancy 445.92 RUB, training net +17,836.71,
// PF 4.565, Max DD 3.06%; holdout net +4,233.02, PF 2.372, Max DD 4.47%.
// Gates: >=10 trades per training year, >=10 holdout trades, positive net,
// PF > 1, Max DD <=10%. Reproduction inputs: data/params/fvg_imbalance/rosn/research_2026-11/.
func Params() core.Params {
	return core.Params{
		ATRPeriod:                 14,
		MinGapATR:                 0.1,
		MaxWaitBars:               30,
		StochPeriod:               10,
		UseStochExit:              1,
		StopATR:                   1,
		TakeProfitATR:             0,
		HTFTrendEMA:               50,
		RequireEntryAboveFVGLower: 1,
	}
}
