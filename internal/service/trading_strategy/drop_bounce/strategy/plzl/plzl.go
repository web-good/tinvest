// Package plzl supplies the ticker and the drop_bounce Params for PLZL (ПАО «Полюс»), one literal
// per timeframe: ParamsMinutes5, ParamsMinutes15, ParamsMinutes30, ParamsHour1.
//
// Калибровка 2026-10-13 скиллом drop-bounce-calibrate (первый тикер из шортлиста cmd/dropscreen
// на 15m): окно 36 мес, walk-forward 12/6, сетки data/params/drop_bounce/plzl/calstop_<таймфрейм>.json
// (entry → risk → exits, keepTop 3). Baseline на дефолтах (WF): 15m PF 0.396 на 6 сделках,
// 30m 0.306 на 8, 1h 0.612 на 24 — порог 0.4 дня одной свечой у PLZL почти недостижим.
//
// Фазовый WF, pooled OOS PF:
//   - 15m 0.830 на 31, compounded −4.05% (фолды 0.815/6, 2.945/6, 0.360/3, 0.712/16 — выше 1 один
//     из трёх живых).
//   - 30m 0.597 на 50, −21.03% (1.371/11, 0.643/5, 0.377/17, 0.559/17 — выше 1 один из четырёх).
//   - 1h 0.615 на 57, −27.81% (1.266/26, 7.689/3, 1.096/8, 0.185/20 — последний фолд обвалил пул).
//
// Планку (PF ≥ 1.2, ≥ 20 сделок, ≥ 3 живых фолда, ниже 1 не больше одного) не взял ни один
// таймфрейм — drop_bounce на PLZL не торговать; плато и издержки не мерились. Высокое место в
// скринере (max PF×2 1.25 на 15m) край в walk-forward не подтвердило.
//
// Литералы 15m/30m/1h — консенсус фолдов (взвешенное среднее выборов, значение k фолдов весит k²),
// in-sample замер 36 мес, не судья:
//   - 15m: порог 0.17 дня, среднее 6, RSI(6)@69, стоп 0.7 ATR(8), GreenVolExit 0.74 — PF 0.969 на 55.
//   - 30m: порог 0.2, среднее 10, RSI(6)@64, стоп 1.12 ATR(8), GreenVolExit 0.17 — PF 1.130 на 84.
//   - 1h: порог 0.34, среднее 13, RSI(6)@70, стоп 0.97 ATR(9), GreenVolExit 0.75 — PF 0.609 на 51.
//
// 5m не калибруется (решение владельца), его литерал — явно выписанные дефолты ядра.
//
// 2026-10-13 выход GREENVOL удалён из ядра по решению владельца, поле GreenVolExit из литералов
// убрано. Цифры калибровки выше мерились вместе с этим выходом и на текущем ядре не
// воспроизводятся: тикер нужно перекалибровать.
package plzl

import "tinvest/internal/service/trading_strategy/drop_bounce/strategy/core"

const Ticker = "PLZL"

// ParamsMinutes5 returns the PLZL literal for Minutes5.
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

// ParamsMinutes15 returns the PLZL literal for Minutes15.
func ParamsMinutes15() core.Params {
	return core.Params{
		VolDailyMult:   0.17,
		VolAvgPeriod:   6,
		RSIPeriod:      6,
		RSIExit:        69,
		StopDailyATR:   0.7,
		DailyATRPeriod: 8,
		StartHour:      7,
	}
}

// ParamsMinutes30 returns the PLZL literal for Minutes30.
func ParamsMinutes30() core.Params {
	return core.Params{
		VolDailyMult:   0.2,
		VolAvgPeriod:   10,
		RSIPeriod:      6,
		RSIExit:        64,
		StopDailyATR:   1.12,
		DailyATRPeriod: 8,
		StartHour:      7,
	}
}

// ParamsHour1 returns the PLZL literal for Hour1.
func ParamsHour1() core.Params {
	return core.Params{
		VolDailyMult:   0.34,
		VolAvgPeriod:   13,
		RSIPeriod:      6,
		RSIExit:        70,
		StopDailyATR:   0.97,
		DailyATRPeriod: 9,
		StartHour:      7,
	}
}
