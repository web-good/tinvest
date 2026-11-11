// Package sngs holds the SNGS baseline for the backtest-only M30 FVG strategy.
package sngs

import "tinvest/internal/service/trading_strategy/fvg_imbalance/strategy/core"

// Params returns the SNGS parameters validated on the reserved final year.
// Calibration: 2023-11-11..2025-11-11; final check: 2025-11-11..2026-11-11.
// UseStochExit=1; expectancy-ranked. Training trades: 11/12 by annual block;
// final: 13 trades, Net PnL +8053.51, PF 4.425, Max DD 2.54%; limits:
// positive Net PnL, PF > 1, Max DD <= 10%, and >=10 final trades.
func Params() core.Params {
	return core.Params{
		ATRPeriod:                 14,  // Период дневного ATR.
		MinGapATR:                 0.1, // Минимальный размер FVG в дневных ATR.
		MaxWaitBars:               30,  // Сколько M30-баров ждать сигнал после FVG.
		StochPeriod:               21,  // Период стохастика %K.
		UseStochExit:              1,   // Выход по пересечению %K уровня 80; 0 отключает.
		StopATR:                   1.7, // Расстояние стопа от цены входа в ATR.
		TakeProfitATR:             0,   // Целевая цена отключена; действует стоп и стохастический выход.
		HTFTrendEMA:               120, // Период дневной EMA; 0 отключает фильтр тренда.
		RequireEntryAboveFVGLower: 1,   // Сигнал и фактический вход не ниже нижней границы FVG.
	}
}
