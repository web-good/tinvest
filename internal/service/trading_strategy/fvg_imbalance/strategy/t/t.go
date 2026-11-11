// Package t holds walk-forward parameters for the backtest-only M30 FVG strategy.
package t

import "tinvest/internal/service/trading_strategy/fvg_imbalance/strategy/core"

// Params returns the selected walk-forward parameters for T.
func Params() core.Params {
	return core.Params{
		ATRPeriod:                 14,   // Период дневного ATR.
		MinGapATR:                 0.35, // Минимальный размер FVG в дневных ATR.
		MaxWaitBars:               40,   // Выбрано в фолде 1.
		StochPeriod:               12,   // Выбрано в фолде 1.
		UseStochExit:              0,    // Выбрано в фолде 1.
		StopATR:                   0.6,  // Выбрано в фолде 1.
		TakeProfitATR:             2,    // Выбрано в фолде 1.
		HTFTrendEMA:               0,    // Период дневной EMA; 0 отключает фильтр тренда.
		RequireEntryAboveFVGLower: 1,    // Выбрано в фолде 1.
	}
}
