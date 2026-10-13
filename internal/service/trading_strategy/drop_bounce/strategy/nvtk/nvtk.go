// Package nvtk supplies the ticker and the drop_bounce Params for NVTK (ПАО «Новатэк»), one literal
// per timeframe: ParamsMinutes5, ParamsMinutes15, ParamsMinutes30, ParamsHour1.
//
// Калибровка 2026-10-13 скиллом drop-bounce-calibrate (тикер из шортлиста cmd/dropscreen на 15m):
// окно 36 мес, walk-forward 12/6, сетки data/params/drop_bounce/nvtk/calstop_<таймфрейм>.json
// (entry → risk → exits, keepTop 3). Baseline на дефолтах (WF): 15m 0 сделок OOS, 30m PF 6.062 на 5,
// 1h 1.448 на 15 — порог 0.4 дня одной свечой у NVTK почти недостижим.
//
// Фазовый WF, pooled OOS PF:
//   - 15m 3.002 на 36, compounded +19.76% (фолды 2.550/14, 1.369/4, 8.345/13, 3.092/5 — все три
//     живых выше 1) — ПЛАНКА ВЗЯТА, первый тикер drop_bounce, взявший её.
//   - 30m 4.622 на 31, +25.42% (6.544/15, 1.009/4, 15.501/10, 1178.6/2) — живых фолдов два, планка
//     не взята, таймфрейм отвергнут.
//   - 1h 1.379 на 63, +11.87% (4882.9/1, 2.188/2, 1.179/51, 0.614/9) — живых два, один выше 1,
//     отвергнут.
//
// Литералы 15m/30m/1h — консенсус фолдов (взвешенное среднее выборов, значение k фолдов весит k²);
// in-sample замер 36 мес, не судья:
//   - 15m: порог 0.15 дня, среднее 8, RSI(6)@56, стоп 0.88 ATR(7), GreenVolExit 0.85 — PF 2.022 на 81,
//     +18.82%, DD 3.33%. Фолды выбирали GreenVolExit 1.7 дважды и 0 (выкл) дважды: 0.85 — середина,
//     а не выбор фолда. Плато есть: вход (порог 0.1/0.15/0.2 × RSIExit 55/56/60) — медиана соседей
//     1.887, минимум 1.116 при пороге 0.1; стоп 0.75/0.88/1.0 — 1.890…2.022. Комиссия 0.1%: PF 1.466,
//     net +9.63% на 81 сделке — издержки выдержаны.
//   - 30m: порог 0.27, среднее 13, RSI(4)@57, стоп 0.88 ATR(7), GreenVolExit 1.17 — PF 1.974 на 42.
//   - 1h: порог 0.34, среднее 12, RSI(6)@57, стоп 1.2 ATR(9), GreenVolExit 1.43 — PF 1.245 на 54.
//
// Решение: торговать drop_bounce на NVTK только на 15m — единственный таймфрейм, прошедший WF,
// плато и издержки; совмещение не проверялось, второго прошедшего нет.
//
// 5m не калибруется (решение владельца), его литерал — явно выписанные дефолты ядра.
//
// 2026-10-13 выход GREENVOL удалён из ядра по решению владельца, поле GreenVolExit из литералов
// убрано. Цифры калибровки выше мерились вместе с этим выходом и на текущем ядре не
// воспроизводятся: тикер нужно перекалибровать.
package nvtk

import "tinvest/internal/service/trading_strategy/drop_bounce/strategy/core"

const Ticker = "NVTK"

// ParamsMinutes5 returns the NVTK literal for Minutes5.
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

// ParamsMinutes15 returns the NVTK literal for Minutes15.
func ParamsMinutes15() core.Params {
	return core.Params{
		VolDailyMult:   0.15,
		VolAvgPeriod:   8,
		RSIPeriod:      6,
		RSIExit:        56,
		StopDailyATR:   0.88,
		DailyATRPeriod: 7,
		StartHour:      7,
	}
}

// ParamsMinutes30 returns the NVTK literal for Minutes30.
func ParamsMinutes30() core.Params {
	return core.Params{
		VolDailyMult:   0.27,
		VolAvgPeriod:   13,
		RSIPeriod:      4,
		RSIExit:        57,
		StopDailyATR:   0.88,
		DailyATRPeriod: 7,
		StartHour:      7,
	}
}

// ParamsHour1 returns the NVTK literal for Hour1.
func ParamsHour1() core.Params {
	return core.Params{
		VolDailyMult:   0.34,
		VolAvgPeriod:   12,
		RSIPeriod:      6,
		RSIExit:        57,
		StopDailyATR:   1.2,
		DailyATRPeriod: 9,
		StartHour:      7,
	}
}
