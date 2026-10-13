// Package rual supplies the ticker and the drop_bounce Params for MAGN (ПАО «ММК»), one
// literal per timeframe: ParamsMinutes5, ParamsMinutes15, ParamsMinutes30, ParamsHour1.
//
// Калибровка 2026-10-13 скиллом drop-bounce-calibrate на ядре без выхода GREENVOL: только 15m,
// окно 36 мес, walk-forward 12/6, сетка data/params/drop_bounce/magn/calstop_minutes15.json
// (entry → risk → exits, keepTop 3). Baseline на дефолтах почти не торгует: WF 7 сделок за окно
// (последний фолд пустой) — порог 0.4 дня одной свечой у MAGN почти недостижим.
//
// Фазовый WF: pooled OOS PF 1.095 на 24 сделках, compounded +1.42%; фолды 1.252/13 (+2.53%),
// 0.397/2 (−0.70%), 0.206/5 (−3.86%), 6.889/4 (+3.61%). Планка (по пулу сделок) не взята: A — пул
// ниже 1.2 и меньше 30 сделок, B — без лучшего фолда −2.03%, C — последние 12 месяцев −0.25%;
// D выполнено. Фолды выбирали стоп от 0.3 до 1.5 ATR — оптимум по риску не держится.
//
// Литерал 15m — консенсус фолдов (взвешенное среднее выборов, значение k фолдов весит k²): порог
// 0.17 дня при среднем 13, стоп 0.89 ATR(7), выход RSI(7) вверх через 68. In-sample замер 36 мес —
// PF 2.298 на 42 сделках, net +18.67%, просадка 6.21% (не судья).
//
// Решение: планка не взята — drop_bounce на MAGN не торговать; плато и издержки не мерились.
// Литералы 5m/30m/1h — явно выписанные дефолты ядра (не калибруются).
package magn

import "tinvest/internal/service/trading_strategy/drop_bounce/strategy/core"

const Ticker = "MAGN"

// ParamsMinutes5 returns the MAGN literal for Minutes5.
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

// ParamsMinutes15 returns the MAGN literal for Minutes15.
func ParamsMinutes15() core.Params {
	return core.Params{
		VolDailyMult:   0.17,
		VolAvgPeriod:   13,
		RSIPeriod:      7,
		RSIExit:        68,
		StopDailyATR:   0.89,
		DailyATRPeriod: 7,
		StartHour:      7,
	}
}

// ParamsMinutes30 returns the MAGN literal for Minutes30.
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

// ParamsHour1 returns the MAGN literal for Hour1.
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
