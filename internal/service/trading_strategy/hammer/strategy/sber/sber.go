// Package sber supplies the hammer Params for SBER (Сбербанк) on Hour1.
//
// # 2026-10-21 — литерал владельца до калибровки
//
// Правила владельца как есть (core.DefaultParams): RSI(5) < 30 на свече-молоте, close молота над
// EMA(200) часовых закрытий, стоп 0.5 дневного ATR, цель — половина вчерашнего дневного диапазона,
// выход по RSI(5) > 70. Записаны литералом, чтобы правка дефолтов ядра не меняла тикер молча.
package sber

import "tinvest/internal/service/trading_strategy/hammer/strategy/core"

// Params returns the SBER literal.
func Params() core.Params {
	return core.Params{
		RSIPeriod:       5,
		RSIZone:         30,
		RSIExit:         70,
		TrendEMA:        200,
		StopDailyATR:    0.5,
		TargetRangeMult: 0.5,
		DailyATRPeriod:  14,
	}
}
