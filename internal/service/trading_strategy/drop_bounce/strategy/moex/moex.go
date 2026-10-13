// Package moex supplies the ticker and the drop_bounce Params for MOEX (ПАО «Московская Биржа»),
// one literal per timeframe: ParamsMinutes5, ParamsMinutes15, ParamsMinutes30, ParamsHour1.
//
// Калибровка 2026-10-13 скиллом drop-bounce-calibrate на 15m: окно 36 мес, walk-forward 12/6,
// сетка data/params/drop_bounce/moex/calstop_minutes15.json (entry → risk → exits, keepTop 3).
// Baseline на дефолтах почти не торгует (WF 1 сделка за окно): порог 0.4 дня одной свечой у MOEX
// недостижим.
//
// Фазовый WF 15m: pooled OOS PF 1.234 на 53, compounded +4.83% (фолды 3.889/5, 0.161/4, 2.739/8,
// 1.069/36). Планка по пулу взята: A 1.234/53, B +1.19%, C +2.82%, D 1.069 у единственного фолда
// с ≥ 10 сделками. Запас тонкий: большой фолд почти в ноль, пул держат три маленьких.
//
// Точка — консенсус фолдов: порог 0.19 дня, среднее 10, выход RSI(7)@68, стоп 1.62 ATR(7);
// in-sample PF 5.155 на 53. Плато по стопу есть (1.5 → 3.482, 2.0 → 5.155), по входу — острый
// пик: порог 0.15 дня даёт PF 1.293–1.632 на ~80 сделках, медиана соседей 2.237 при пороге 3.608.
// Под удвоенной комиссией 0.1% PF 3.701, net +25115 ₽ — издержки выдержаны, но решает плато.
// Решение: не торговать — in-sample край держится на узком пороге объёма.
//
// 5m/30m/1h не калибруются (решение владельца), их литералы — явно выписанные дефолты ядра.
package moex

import "tinvest/internal/service/trading_strategy/drop_bounce/strategy/core"

const Ticker = "MOEX"

// ParamsMinutes5 returns the MOEX literal for Minutes5.
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

// ParamsMinutes15 returns the MOEX literal for Minutes15.
func ParamsMinutes15() core.Params {
	return core.Params{
		VolDailyMult:   0.19,
		VolAvgPeriod:   10,
		RSIPeriod:      7,
		RSIExit:        68,
		StopDailyATR:   1.62,
		DailyATRPeriod: 7,
		StartHour:      7,
	}
}

// ParamsMinutes30 returns the MOEX literal for Minutes30.
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

// ParamsHour1 returns the MOEX literal for Hour1.
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
