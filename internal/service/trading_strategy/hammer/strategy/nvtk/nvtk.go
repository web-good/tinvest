// Package nvtk supplies the hammer Params for NVTK (Новатэк) on Hour1.
//
// # 2026-10-21 — литерал владельца до калибровки
//
// Правила владельца как есть (core.DefaultParams): RSI(5) < 30 на свече-молоте, close молота над
// EMA(200) часовых закрытий, объём молота выше 1.5 × среднего 20 свечей основной сессии, стоп 0.5 дневного ATR, цель — половина вчерашнего дневного диапазона,
// выход по RSI(5) > 70. Вход под EMA (BelowEMA) выключен. Записаны литералом, чтобы правка дефолтов ядра не меняла тикер молча.
package nvtk

import "tinvest/internal/service/trading_strategy/hammer/strategy/core"

// Params returns the NVTK literal.
func Params() core.Params {
	return core.Params{
		RSIPeriod:       5,
		RSIZone:         30,
		RSIExit:         70,
		TrendEMA:        200,
		VolumeMult:      1.5,
		VolBars:         20,
		BelowEMA:        0,
		BelowRSIPeriod:  6,
		BelowRSIZone:    30,
		BelowRSIExit:    50,
		StopDailyATR:    0.5,
		TargetRangeMult: 0.5,
		DailyATRPeriod:  14,
	}
}
