// Package ivat provides exploratory parameters for IVAT in the FVG imbalance strategy.
package ivat

import "tinvest/internal/service/trading_strategy/fvg_imbalance/strategy/core"

// Params returns the exploratory IVAT M30 FVG imbalance parameters.
// Trained on the available 2024-06-04–2025-11-13 history (11 and 10 closed trades
// in the unequal descriptive blocks); training Net PnL was +RUB 38,842.28,
// PF 2.369, Max DD 7.84%. On the approximate 2025-11-14–2026-11-14 validation
// window it had 11 closed trades, Net PnL +RUB 25,426.57, PF 2.557, but Max DD
// was 12.50%, above the 10% limit. It is recorded for research, not as a verified
// calibration; the available history does not provide two full training years.
// Reproduction inputs: data/params/fvg_imbalance/ivat/research_2026-11/.
func Params() core.Params {
	return core.Params{
		ATRPeriod:                 14,
		MinGapATR:                 0.3,
		MaxWaitBars:               30,
		StochPeriod:               14,
		UseStochExit:              0,
		StopATR:                   1,
		TakeProfitATR:             1,
		HTFTrendEMA:               0,
		RequireEntryAboveFVGLower: 1,
	}
}
