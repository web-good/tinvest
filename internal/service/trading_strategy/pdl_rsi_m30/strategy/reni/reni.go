// Package reni supplies the independent pdl_rsi_m30 baseline Params for RENI
// (Ренессанс Страхование) on Minutes30.
//
// The numbers are a mechanical starting point from RENI's Hour1 literal. They are not an M30
// calibration result and must be replaced only after the dedicated rolling walk-forward procedure.
package reni

import "tinvest/internal/service/trading_strategy/pdl_rsi_m30/strategy/core"

// Params returns the RENI M30 baseline literal.
func Params() core.Params {
	return core.Params{
		RSIPeriod:      6,   // Период Wilder RSI по закрытиям M30.
		RSIArm:         25,  // Верхняя граница нижней RSI-зоны: на сигнале RSI должен быть ниже 25.
		RSIFloor:       0,   // Нижняя допустимая граница RSI; 0 отключает фильтр слишком глубокого падения.
		RSIExit:        70,  // Выход по close при пересечении RSI уровня 70 снизу вверх.
		StochK:         13,  // Число M30-баров для расчёта сырого stochastic %K; 0 отключает stochastic.
		StochSmooth:    3,   // Число значений сырого %K для сглаживания итогового %K.
		StochD:         3,   // Период линии %D; она используется только в тексте причины входа.
		VolumeMult:     0,   // Минимальный объём сигнала в долях среднего; 0 отключает фильтр объёма.
		VolBars:        20,  // Число предыдущих M30-баров основной сессии в базе среднего объёма.
		MinRoomATR:     0,   // Минимальный оставшийся дневной ход в ATR; 0 отключает фильтр запаса хода.
		StopATR:        0.3, // Расстояние стопа от цены входа в дневных ATR.
		MaxStopPct:     0,   // Максимальное расстояние стопа в процентах цены; 0 отключает ограничение.
		TargetR:        3,   // Цель выше входа в расстояниях риска R; 0 отключает цель.
		TrendEMA:       200, // Период EMA по M30-close; сигнал разрешён только строго выше EMA.
		DailyATRPeriod: 14,  // Период Wilder ATR по завершённым будним дневным свечам.
	}
}
