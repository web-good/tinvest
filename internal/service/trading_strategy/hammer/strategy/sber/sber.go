// Package sber supplies the hammer Params for SBER (Сбербанк) on Hour1.
//
// # 2026-10-22 — литерал владельца до калибровки
//
// Правила владельца (core.DefaultParams): RSI(5) < 30 на свече-молоте, close молота над EMA(200)
// часовых закрытий, объём молота выше 1.5 × среднего 20 свечей основной сессии, стоп под low молота
// с запасом 0.05 дневного ATR, цель 2R от цены покупки, выход по RSI(5) > 70. Записаны литералом,
// чтобы правка дефолтов ядра не меняла тикер молча.
package sber

import "tinvest/internal/service/trading_strategy/hammer/strategy/core"

// Params returns the SBER literal.
func Params() core.Params {
	return core.Params{
		RSIPeriod:      5,
		RSIZone:        30,
		MACDFast:       11,
		MACDSlow:       15,
		MACDSignal:     9,
		RSIHold:        60,
		TrendEMA:       200,
		VolumeMult:     1.5,
		VolBars:        20,
		StopBufferATR:  0.05,
		TargetR:        2,
		DailyATRPeriod: 14,
	}
}
