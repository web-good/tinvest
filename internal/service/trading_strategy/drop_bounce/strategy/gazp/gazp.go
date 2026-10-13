// Package gazp supplies the ticker and the drop_bounce Params for GAZP (ПАО «Газпром»), one
// literal per timeframe: ParamsMinutes5, ParamsMinutes15, ParamsMinutes30, ParamsHour1.
//
// Калибровка 2026-10-12 скиллом drop-bounce-calibrate на правилах всплеска объёма: окно 36 мес,
// walk-forward 12/6, сетки data/params/drop_bounce/gazp/calstop_<таймфрейм>.json (entry → risk →
// exits, keepTop 3). Baseline на дефолтах почти не торгует: порог 0.4 дня одной свечой даёт 0–10
// сделок, поэтому судья — фазовый WF.
//
// Первый круг, без выхода GREENVOL: pooled OOS PF 15m 0.692 на 16, 5m 0.882 на 61 — планку не
// взял никто.
//
// Второй круг, с выходом GREENVOL (GreenVolExit 1.0 прибит в сетках), 30m по просьбе владельца.
// Pooled OOS PF: 15m 1.881 на 15 (фолды 3.425/4, 0/0, 1.633/8, 0.069/3 — живой один), 30m 1.032 на
// 42 (0.864/23, 1.233/6, 3.165/9, 0.503/4), 5m 1.129 на 148 (1.883/42, 2.545/32, 0.762/38,
// 0.125/36). Планку (PF ≥ 1.2, ≥ 20 сделок, ≥ 3 живых фолда, ниже 1 не больше одного) не взял ни
// один таймфрейм — drop_bounce на GAZP не торговать; плато и издержки не мерились.
//
// Третий круг 2026-10-12 — широкая сетка владельца (VolAvgPeriod 5/10/14, DailyATRPeriod 7/10/14,
// RSIPeriod 4/5/6, стоп 0.3–3, RSIExit 50–90, GreenVolExit 0–3; StartHour всегда 7). Pooled OOS PF:
// 15m 1.756 на 12 (фолды 4.068/4, 0/0, 1.201/8, 0/0 — живой один), 30m 1.644 на 30 (1.202/8,
// 0.953/9, 31.321/9, 0.500/4 — живых три, выше 1 два). Планку (PF ≥ 1.2, ≥ 20 сделок, ≥ 3 живых
// фолда, ниже 1 не больше одного) не взял ни один: 15m — мало сделок, 30m — фолд 0.953.
// Решение — drop_bounce на GAZP не торговать; плато и издержки не мерились.
//
// Литералы 15m и 30m — консенсус фолдов третьего круга (правило владельца: взвешенное среднее
// выборов фолдов, значение, выбранное k фолдами, весит k²; литерал пишется в любом случае). 15m:
// порог 0.27 дня, среднее 12, RSI(6)@75, стоп 0.97 ATR(10), GreenVolExit 2.17 — бэктест 36 мес
// PF 1.617 на 22 сделках. 30m: порог 0.28, среднее 14, RSI(6)@67, стоп 0.97 ATR(10), GreenVolExit
// 0.29 — PF 1.183 на 49. Это in-sample замер точки, не судья.
//
// 1h 2026-10-12 по просьбе владельца, та же широкая сетка, но уже с RSIPeriod 4–7 и GreenVolExit
// 0–1.7 (calstop_hour1.json). Baseline на дефолтах: pooled 1.678 на 17 (живых фолдов два). Фазовый
// WF: pooled 0.601 на 37 (2.701/4, 0/0, 0.553/18, 0.389/15) — живые фолды оба ниже 1, ПЛАНКА НЕ
// ВЗЯТА; калибровка на 1h хуже дефолтов. Литерал 1h — консенсус фолдов: порог 0.36 дня, среднее 8,
// RSI(5)@71, стоп 0.72 ATR(10), GreenVolExit 0.33 — бэктест 36 мес PF 1.386 на 47 (in-sample).
// 5m больше не калибруется (решение владельца), его литерал — явно выписанные дефолты ядра.
//
// 2026-10-13 выход GREENVOL удалён из ядра по решению владельца, поле GreenVolExit из литералов
// убрано. Цифры калибровки выше мерились вместе с этим выходом и на текущем ядре не
// воспроизводятся: тикер нужно перекалибровать.
package gazp

import "tinvest/internal/service/trading_strategy/drop_bounce/strategy/core"

const Ticker = "GAZP"

// ParamsMinutes5 returns the GAZP literal for Minutes5.
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

// ParamsMinutes15 returns the GAZP literal for Minutes15.
func ParamsMinutes15() core.Params {
	return core.Params{
		VolDailyMult:   0.27,
		VolAvgPeriod:   12,
		RSIPeriod:      6,
		RSIExit:        75,
		StopDailyATR:   0.97,
		DailyATRPeriod: 10,
		StartHour:      7,
	}
}

// ParamsMinutes30 returns the GAZP literal for Minutes30.
func ParamsMinutes30() core.Params {
	return core.Params{
		VolDailyMult:   0.28,
		VolAvgPeriod:   14,
		RSIPeriod:      6,
		RSIExit:        67,
		StopDailyATR:   0.97,
		DailyATRPeriod: 10,
		StartHour:      7,
	}
}

// ParamsHour1 returns the GAZP literal for Hour1.
func ParamsHour1() core.Params {
	return core.Params{
		VolDailyMult:   0.36,
		VolAvgPeriod:   8,
		RSIPeriod:      5,
		RSIExit:        71,
		StopDailyATR:   0.72,
		DailyATRPeriod: 10,
		StartHour:      7,
	}
}
