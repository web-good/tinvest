// Package vsmo supplies the ticker and the drop_bounce Params for VSMO (ПАО «Корпорация ВСМПО-АВИСМА»),
// one literal per timeframe: ParamsMinutes5, ParamsMinutes15, ParamsMinutes30, ParamsHour1.
//
// Перекалибровка 2026-10-13 на ядре без выхода GREENVOL, скилл drop-bounce-calibrate: окно 36 мес,
// walk-forward 12/6, сетки data/params/drop_bounce/vsmo/calstop_<таймфрейм>.json
// (entry → risk → exits, keepTop 3). Первая калибровка того же дня (с GREENVOL) заменена целиком;
// её цифры — в РЕЗУЛЬТАТ сеток с пометкой «ядро с GREENVOL».
//
// Baseline на дефолтах (WF): 15m PF 2.732 на 20 сделках (живых фолдов 2), 30m 2.087 на 32 (живых
// 2, выше 1 один), 1h 2.164 на 40 (4 живых, все выше 1: 3.028/13, 2.763/7, 3.029/9, 1.009/11).
// Baseline не судья: формально 1h на дефолтах берёт планку, но фазовая сетка на тех же данных
// дефолты на train-окнах не выбрала, и её pooled OOS ниже 1.
//
// Фазовый WF, pooled OOS PF:
//   - 15m 1.432 на 53, compounded +17.77% (фолды 0.908/34, 2.727/5, 3.913/4 вырожден, 2.617/10 —
//     живых три, выше 1 два; первый фолд с 34 сделками ниже 1).
//   - 30m 1.534 на 39, +15.15% (5.037/16, 0.419/6, 2090/2 вырожден, 1.283/15 — живых три, выше 1 два).
//   - 1h 0.800 на 30, −10.04% (0.969/14, 1.193/3, 0.606/2, 0.684/11 — живых два, выше 1 ни одного).
//
// Прежнюю планку (≥ 3 живых фолда, выше 1 не меньше трёх) не взял ни один таймфрейм: у 15m и 30m
// выше 1 только два живых фолда из трёх.
//
// 2026-10-13 владелец заменил планку на судейство по пулу сделок (A: PF ≥ 1.2 и ≥ 30 сделок;
// B: net фолдов без лучшего > 0; C: net двух последних фолдов > 0; D: фолды с ≥ 10 сделками
// PF ≥ 0.8). 15m её взял: A 1.432/53, B +4.84%, C +16.00%, D 0.908/34 и 2.617/10. 30m — нет
// (B −4.86%), 1h — нет.
//
// 15m, точка 0.28/RSI@70/стоп 1.88: плато по входу и выходу есть (8 соседей 1.320–1.903, медиана
// 1.444 — точка на нижнем краю, соседи не хуже), по стопу есть (1.5 → 1.583, 2.0 → 1.324); под
// удвоенной комиссией 0.1% PF 1.219, net +9539 ₽ на 58 сделках — издержки выдержаны.
// Решение: торговать на 15m (DROP_BOUNCE_TICKERS_M15).
//
// Литералы 15m/30m/1h — консенсус фолдов (взвешенное среднее выборов, значение k фолдов весит k²),
// in-sample замер 36 мес, не судья:
//   - 15m: порог 0.28 дня, среднее 14, RSI(6)@70, стоп 1.88 ATR(7) — PF 1.357 на 58.
//   - 30m: порог 0.35, среднее 14, RSI(6)@66, стоп 1.43 ATR(7) — PF 1.559 на 64.
//   - 1h: порог 0.55, среднее 10, RSI(6)@68, стоп 1.59 ATR(10) — PF 1.577 на 38.
//
// 5m не калибруется (решение владельца), его литерал — явно выписанные дефолты ядра.
package vsmo

import "tinvest/internal/service/trading_strategy/drop_bounce/strategy/core"

const Ticker = "VSMO"

// ParamsMinutes5 returns the VSMO literal for Minutes5.
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

// ParamsMinutes15 returns the VSMO literal for Minutes15.
func ParamsMinutes15() core.Params {
	return core.Params{
		VolDailyMult:   0.28,
		VolAvgPeriod:   14,
		RSIPeriod:      6,
		RSIExit:        70,
		StopDailyATR:   1.88,
		DailyATRPeriod: 7,
		StartHour:      7,
	}
}

// ParamsMinutes30 returns the VSMO literal for Minutes30.
func ParamsMinutes30() core.Params {
	return core.Params{
		VolDailyMult:   0.35,
		VolAvgPeriod:   14,
		RSIPeriod:      6,
		RSIExit:        66,
		StopDailyATR:   1.43,
		DailyATRPeriod: 7,
		StartHour:      7,
	}
}

// ParamsHour1 returns the VSMO literal for Hour1.
func ParamsHour1() core.Params {
	return core.Params{
		VolDailyMult:   0.55,
		VolAvgPeriod:   10,
		RSIPeriod:      6,
		RSIExit:        68,
		StopDailyATR:   1.59,
		DailyATRPeriod: 10,
		StartHour:      7,
	}
}
