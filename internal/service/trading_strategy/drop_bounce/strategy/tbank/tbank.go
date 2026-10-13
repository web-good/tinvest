// Package tbank supplies the ticker and the drop_bounce Params for T (МКПАО «Т-Технологии»), one
// literal per timeframe: ParamsMinutes5, ParamsMinutes15, ParamsMinutes30, ParamsHour1.
//
// Калибровка 2026-10-12 скиллом drop-bounce-calibrate на правилах всплеска объёма: окно 36 мес,
// walk-forward 12/6, сетки data/params/drop_bounce/t/calstop_<таймфрейм>.json (entry → risk →
// exits, keepTop 3). Baseline на дефолтах почти не торгует (WF: 15m 1 сделка, 30m 3, 1h 8, живых
// фолдов ноль) — порог 0.4 дня одной свечой у T недостижим.
//
// Фазовый WF, pooled OOS PF:
//   - 15m 1.184 на 166 (фолды 1.015/121, 1.128/4, 11.718/12, 1.191/29 — живых три, все выше 1);
//     не хватило 0.016 до PF 1.2. Край тонкий: первый фолд — 121 сделка почти в ноль.
//   - 30m 0.793 на 36 (0/2, 0.408/5, 0.952/6, 1.382/23 — выше 1 один из трёх живых).
//   - 1h 0.641 на 46 (0.817/12, 0.513/6, 0/2, 0.737/26 — все живые ниже 1).
//
// Планку (PF ≥ 1.2, ≥ 20 сделок, ≥ 3 живых фолда, ниже 1 не больше одного) не взял ни один
// таймфрейм — drop_bounce на T не торговать; плато и издержки не мерились.
//
// Литералы 15m/30m/1h — консенсус фолдов (взвешенное среднее выборов, значение k фолдов весит k²),
// in-sample замер 36 мес, не судья:
//   - 15m: порог 0.1 дня, среднее 12, RSI(4)@51, стоп 1.02 ATR(8), GreenVolExit 1.63 — PF 1.512 на 167.
//   - 30m: порог 0.21, среднее 6, RSI(4)@64, стоп 1.0 ATR(7), GreenVolExit 1.25 — PF 1.148 на 62.
//   - 1h: порог 0.29, среднее 7, RSI(6)@77, стоп 0.79 ATR(7), GreenVolExit 0.99 — PF 1.420 на 58.
//
// 5m не калибруется (решение владельца), его литерал — явно выписанные дефолты ядра.
//
// 2026-10-13 выход GREENVOL удалён из ядра по решению владельца, поле GreenVolExit из литералов
// убрано. Цифры калибровки выше мерились вместе с этим выходом и на текущем ядре не
// воспроизводятся: тикер нужно перекалибровать.
package tbank

import "tinvest/internal/service/trading_strategy/drop_bounce/strategy/core"

const Ticker = "T"

// ParamsMinutes5 returns the T literal for Minutes5.
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

// ParamsMinutes15 returns the T literal for Minutes15.
func ParamsMinutes15() core.Params {
	return core.Params{
		VolDailyMult:   0.1,
		VolAvgPeriod:   12,
		RSIPeriod:      4,
		RSIExit:        51,
		StopDailyATR:   1.02,
		DailyATRPeriod: 8,
		StartHour:      7,
	}
}

// ParamsMinutes30 returns the T literal for Minutes30.
func ParamsMinutes30() core.Params {
	return core.Params{
		VolDailyMult:   0.21,
		VolAvgPeriod:   6,
		RSIPeriod:      4,
		RSIExit:        64,
		StopDailyATR:   1,
		DailyATRPeriod: 7,
		StartHour:      7,
	}
}

// ParamsHour1 returns the T literal for Hour1.
func ParamsHour1() core.Params {
	return core.Params{
		VolDailyMult:   0.29,
		VolAvgPeriod:   7,
		RSIPeriod:      6,
		RSIExit:        77,
		StopDailyATR:   0.79,
		DailyATRPeriod: 7,
		StartHour:      7,
	}
}
