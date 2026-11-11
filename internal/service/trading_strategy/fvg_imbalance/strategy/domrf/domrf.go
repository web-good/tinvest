// Package domrf holds the uncalibrated DOMRF baseline for the backtest-only M30 FVG strategy.
package domrf

import "tinvest/internal/service/trading_strategy/fvg_imbalance/strategy/core"

// Params returns an explicit starting point for DOMRF calibration.
func Params() core.Params {
	return core.Params{
		ATRPeriod:     14,   // Период дневного ATR.
		MinGapATR:     0.25, // Минимальный размер FVG в дневных ATR.
		MaxWaitBars:   35,   // Сколько M30-баров ждать сигнал после FVG.
		StochPeriod:   14,   // Период стохастика %K.
		UseStochExit:  1,    // Выход по пересечению %K уровня 80; 0 отключает.
		StopATR:       1,    // Расстояние стопа от цены входа в ATR.
		TakeProfitATR: 2,    // Расстояние цели в ATR; 0 отключает ценовую цель.
		HTFTrendEMA:   0,    // Период дневной EMA; 0 отключает фильтр тренда.
	}
}
