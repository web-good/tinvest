// Package enpg supplies the ticker and the drop_bounce Params for ENPG (МКПАО «ЭН+ ГРУП»), one
// literal per timeframe: ParamsMinutes5, ParamsMinutes15, ParamsMinutes30, ParamsHour1.
//
// Перекалибровка 15m 2026-10-13 скиллом drop-bounce-calibrate на ядре без выхода GREENVOL:
// окно 36 мес, walk-forward 12/6, сетка data/params/drop_bounce/enpg/calstop_minutes15.json
// (entry → risk → exits, keepTop 3, перегенерирована без GreenVolExit). Baseline на дефолтах почти
// не торгует: WF 5 сделок, все в первом фолде, — порог 0.4 дня одной свечой у ENPG недостижим.
//
// Фазовый WF: pooled OOS PF 2.842 на 25 сделках, compounded +27.63%; фолды 7.349/8 (+30.70%),
// 0/0, 0/2 (−5.62%), 1.688/15 (+3.47%). Планка (по пулу сделок) не взята: A — 25 сделок меньше 30,
// B — без лучшего фолда −2.15%, C — последние 12 месяцев −2.15%; D выполнено. Весь результат —
// полугодие 2024-10…2025-04 на 8 сделках, второй фолд без сделок: всплесков объёма у ENPG мало.
//
// Литерал 15m — консенсус фолдов (взвешенное среднее выборов, значение k фолдов весит k²): порог
// 0.21 дня при среднем 12, стоп 0.88 ATR(8), выход RSI(6) вверх через 78. In-sample замер 36 мес —
// PF 2.046 на 48 сделках, net +35.22%, просадка 7.36% (не судья).
//
// Решение: drop_bounce на ENPG не торговать; плато и издержки не мерились.
//
// Литералы 30m/1h — консенсус калибровки 2026-10-13 на прежнем ядре с GREENVOL (поле убрано из
// литералов, цифры на текущем ядре не воспроизводятся; таймфреймы не калибруются с решения
// владельца торговать только 15m): 30m — pooled 6.215 на 13, порог 0.36, RSI(5)@78, стоп 0.79
// ATR(8); 1h — pooled 2.589 на 20, порог 0.48, RSI(6)@72, стоп 0.63 ATR(9). 5m не калибруется,
// его литерал — явно выписанные дефолты ядра.
package enpg

import "tinvest/internal/service/trading_strategy/drop_bounce/strategy/core"

const Ticker = "ENPG"

// ParamsMinutes5 returns the ENPG literal for Minutes5.
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

// ParamsMinutes15 returns the ENPG literal for Minutes15.
func ParamsMinutes15() core.Params {
	return core.Params{
		VolDailyMult:   0.21,
		VolAvgPeriod:   12,
		RSIPeriod:      6,
		RSIExit:        78,
		StopDailyATR:   0.88,
		DailyATRPeriod: 8,
		StartHour:      7,
	}
}

// ParamsMinutes30 returns the ENPG literal for Minutes30.
func ParamsMinutes30() core.Params {
	return core.Params{
		VolDailyMult:   0.36,
		VolAvgPeriod:   7,
		RSIPeriod:      5,
		RSIExit:        78,
		StopDailyATR:   0.79,
		DailyATRPeriod: 8,
		StartHour:      7,
	}
}

// ParamsHour1 returns the ENPG literal for Hour1.
func ParamsHour1() core.Params {
	return core.Params{
		VolDailyMult:   0.48,
		VolAvgPeriod:   14,
		RSIPeriod:      6,
		RSIExit:        72,
		StopDailyATR:   0.63,
		DailyATRPeriod: 9,
		StartHour:      7,
	}
}
