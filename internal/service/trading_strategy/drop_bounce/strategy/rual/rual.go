// Package rual supplies the ticker and the drop_bounce Params for RUAL (МКПАО «ОК РУСАЛ»), one
// literal per timeframe: ParamsMinutes5, ParamsMinutes15, ParamsMinutes30, ParamsHour1.
//
// Калибровка 2026-10-13 скиллом drop-bounce-calibrate на ядре без выхода GREENVOL: только 15m,
// окно 36 мес, walk-forward 12/6, сетка data/params/drop_bounce/rual/calstop_minutes15.json
// (entry → risk → exits, keepTop 3). Baseline на дефолтах почти не торгует: WF 3 сделки за окно
// (два фолда пустые) — порог 0.4 дня одной свечой у RUAL недостижим.
//
// Фазовый WF: pooled OOS PF 0.392 на 17 сделках, compounded −10.72%; фолды 3.200/2, 4.266/3,
// 0.757/10, 0.000/2 (последний — −10.97% на двух стопах). Не выполнено ни одно условие планки:
// A (пул ниже 1.2 и меньше 30 сделок), B (без лучшего фолда −11.57%), C (последние 12 месяцев
// −12.41%), D (фолд на 10 сделках 0.757 < 0.8). Сделок мало при любом пороге — всплесков объёма у
// RUAL на 15m почти нет.
//
// Литерал 15m — консенсус фолдов (взвешенное среднее выборов, значение k фолдов весит k²): порог
// 0.23 дня при среднем 14, стоп 0.97 ATR(9), выход RSI(6) вверх через 58. In-sample замер 36 мес —
// PF 1.874 на 40 сделках, net +11.98%, просадка 8.10% (не судья).
//
// Решение: планка не взята — drop_bounce на RUAL не торговать; плато и издержки не мерились.
// Литералы 5m/30m/1h — явно выписанные дефолты ядра (не калибруются).
package rual

import "tinvest/internal/service/trading_strategy/drop_bounce/strategy/core"

const Ticker = "RUAL"

// ParamsMinutes5 returns the RUAL literal for Minutes5.
func ParamsMinutes5() core.Params {
	return core.Params{
		VolDailyMult:   0.4,
		VolAvgPeriod:   14,
		RSIPeriod:      6,
		RSIExit:        70,
		StopDailyATR:   1,
		DailyATRPeriod: 14,
		StartHour:      7,
	}
}

// ParamsMinutes15 returns the RUAL literal for Minutes15.
func ParamsMinutes15() core.Params {
	return core.Params{
		VolDailyMult:   0.23,
		VolAvgPeriod:   14,
		RSIPeriod:      6,
		RSIExit:        58,
		StopDailyATR:   0.97,
		DailyATRPeriod: 9,
		StartHour:      7,
	}
}

// ParamsMinutes30 returns the RUAL literal for Minutes30.
func ParamsMinutes30() core.Params {
	return core.Params{
		VolDailyMult:   0.4,
		VolAvgPeriod:   14,
		RSIPeriod:      6,
		RSIExit:        70,
		StopDailyATR:   1,
		DailyATRPeriod: 14,
		StartHour:      7,
	}
}

// ParamsHour1 returns the RUAL literal for Hour1.
func ParamsHour1() core.Params {
	return core.Params{
		VolDailyMult:   0.4,
		VolAvgPeriod:   14,
		RSIPeriod:      6,
		RSIExit:        70,
		StopDailyATR:   1,
		DailyATRPeriod: 14,
		StartHour:      7,
	}
}
