// Package gazp provides walk-forward parameters for GAZP.
package gazp

import "tinvest/internal/service/trading_strategy/fvg_imbalance/strategy/core"

// Params returns the research candidate selected for GAZP.
// Calibrated on 2024-11-11 through 2026-11-11 (24 months); annual trade counts
// were 12/20, with expectancy 2241, net PnL 71723.06, PF 2.508, and max DD 9.59%.
// This candidate has not had an independent following-year validation.
func Params() core.Params {
	return core.Params{
		ATRPeriod:                 14,   // Период дневного ATR.
		MinGapATR:                 0.25, // Минимальный размер FVG в дневных ATR.
		MaxWaitBars:               30,   // Сколько M30-баров ждать сигнал после FVG.
		StochPeriod:               12,   // Период стохастика %K.
		UseStochExit:              0,    // Выход по пересечению %K уровня 80; 0 отключает.
		StopATR:                   0.5,  // Walk-forward: стоп выбран по фолду 1.
		TakeProfitATR:             2,    // Цель в дневных ATR.
		HTFTrendEMA:               0,    // Период дневной EMA; 0 отключает фильтр тренда.
		RequireEntryAboveFVGLower: 1,    // Закрытие и исполнение не ниже нижней границы FVG.
	}
}
