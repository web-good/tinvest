// Package rual supplies the ticker and the drop_bounce Params for NLMK (ПАО «НЛМК»), one
// literal per timeframe: ParamsMinutes5, ParamsMinutes15, ParamsMinutes30, ParamsHour1.
//
// Калибровка 2026-10-13 скиллом drop-bounce-calibrate на ядре без выхода GREENVOL: только 15m,
// окно 36 мес, walk-forward 12/6, сетка data/params/drop_bounce/nlmk/calstop_minutes15.json
// (entry → risk → exits, keepTop 3). Baseline на дефолтах не торгует вовсе: WF 0 сделок во всех
// фолдах — порог 0.4 дня одной свечой у NLMK недостижим.
//
// Фазовый WF: pooled OOS PF 0.812 на 102 сделках, compounded −14.18%; фолды 2034.452/1 (+2.03%),
// 0.318/5 (−1.32%), 0.551/9 (−1.75%), 0.811/87 (−13.25%, просадка 33.28% — последний фолд выбрал
// порог 0.05 дня и торговал почти каждый день). Планка не взята: A — пул ниже 1.0, B — без лучшего
// фолда −16.32%, C — последние 12 месяцев −15.00%; D формально выполнено (0.811). Выборы фолдов
// скачут: порог 0.05–0.25, стоп 0.3–1.5 ATR.
//
// Литерал 15m — консенсус фолдов (взвешенное среднее выборов, значение k фолдов весит k²): порог
// 0.15 дня при среднем 7, стоп 0.82 ATR(8), выход RSI(6) вверх через 63. In-sample замер 36 мес —
// PF 0.756 на 55 сделках, net −7.93%, просадка 11.53%: точка убыточна даже на истории.
//
// Решение: края нет — drop_bounce на NLMK не торговать; плато и издержки не мерились.
// Литералы 5m/30m/1h — явно выписанные дефолты ядра (не калибруются).
package nlmk

import "tinvest/internal/service/trading_strategy/drop_bounce/strategy/core"

const Ticker = "NLMK"

// ParamsMinutes5 returns the NLMK literal for Minutes5.
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

// ParamsMinutes15 returns the NLMK literal for Minutes15.
func ParamsMinutes15() core.Params {
	return core.Params{
		VolDailyMult:   0.15,
		VolAvgPeriod:   7,
		RSIPeriod:      6,
		RSIExit:        63,
		StopDailyATR:   0.82,
		DailyATRPeriod: 8,
		StartHour:      7,
	}
}

// ParamsMinutes30 returns the NLMK literal for Minutes30.
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

// ParamsHour1 returns the NLMK literal for Hour1.
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
