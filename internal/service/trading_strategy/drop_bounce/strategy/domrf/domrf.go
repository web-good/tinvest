// Package domrf supplies the ticker and the drop_bounce Params for DOMRF (ДОМ.РФ), one literal per
// timeframe: ParamsMinutes5, ParamsMinutes15, ParamsMinutes30, ParamsHour1.
//
// Калибровка 2026-10-11 на всей истории (торги с 2025-11-20, -months 11), сетки
// data/params/drop_bounce/domrf/cal_<таймфрейм>.json (фазы entry → risk → exits, keepTop 3).
// Литералы 5m, 15m и 1h — лучшая in-sample точка со стопом (точки без стопа не берутся):
// 5m 5.715/28, 15m 14.150/21 (окно с 07:00 — из литерала владельца, не свипалось), 1h 3.046/24.
// Walk-forward 11/6/2 (два фолда), pooled OOS PF: 5m 3.498 на 11 сделках, 15m 1.391 на 8, 1h 1.707
// на 10 — выборки слишком малы для вывода; in-sample PF двузначные на 15–24 сделках — подгонка
// вероятна. 30m провалил walk-forward (pooled 0.989 на 15) и остался на дефолтах ядра. Числа
// выписаны явно, чтобы бэктест DOMRF не уезжал вместе с правкой core.DefaultParams.
//
// 2026-10-12 стоп переведён с процента на дневной ATR: StopPct (5m/15m/30m/1h 1.5/2/1/3 %) пересчитан
// в StopDailyATR 0.7/0.95/0.5/1.45 через медианный дневной ATR тикера за всю историю с 2025-11 (2.10%, Wilder 14 по
// будним дневкам), шаг 0.05. Это пересчёт, не калибровка: числа walk-forward выше получены на
// процентном стопе. Фильтр размаха дня MaxDayRangeATR выключен.
//
// 2026-10-12 из ядра убраны фильтр тренда EMA и выход MACD. Литералы потеряли поля
// (5m/15m/30m/1h) MACDExit 1/0/1/1, EMAPeriod 0/0/0/200; все числа PF выше получены с ними и к текущему поведению не относятся.
package domrf

import "tinvest/internal/service/trading_strategy/drop_bounce/strategy/core"

const Ticker = "DOMRF"

// ParamsMinutes5 returns the DOMRF literal for Minutes5.
func ParamsMinutes5() core.Params {
	return core.Params{
		MinDropPct:     0.7,
		RSIPeriod:      3,
		RSIEntry:       30,
		RSIExit:        70,
		ExitInProfit:   0,
		StochPeriod:    14,
		StochSmooth:    3,
		StochExit:      0,
		ProfitPct:      0.5,
		StopDailyATR:   0.7,
		DailyATRPeriod: 14,
		MaxDayRangeATR: 0,
		VolMult:        0,
		VolBars:        20,
		StartHour:      8,
	}
}

// ParamsMinutes15 returns the DOMRF literal for Minutes15.
func ParamsMinutes15() core.Params {
	return core.Params{
		MinDropPct:     1,
		RSIPeriod:      3,
		RSIEntry:       30,
		RSIExit:        0,
		ExitInProfit:   0,
		StochPeriod:    14,
		StochSmooth:    3,
		StochExit:      80,
		ProfitPct:      0,
		StopDailyATR:   0.95,
		DailyATRPeriod: 14,
		MaxDayRangeATR: 0,
		VolMult:        0,
		VolBars:        20,
		StartHour:      7,
	}
}

// ParamsMinutes30 returns the DOMRF literal for Minutes30.
func ParamsMinutes30() core.Params {
	return core.Params{
		MinDropPct:     0.6,
		RSIPeriod:      6,
		RSIEntry:       30,
		RSIExit:        70,
		ExitInProfit:   0,
		StochPeriod:    14,
		StochSmooth:    3,
		StochExit:      80,
		ProfitPct:      1,
		StopDailyATR:   0.5,
		DailyATRPeriod: 14,
		MaxDayRangeATR: 0,
		VolMult:        0,
		VolBars:        20,
		StartHour:      8,
	}
}

// ParamsHour1 returns the DOMRF literal for Hour1.
func ParamsHour1() core.Params {
	return core.Params{
		MinDropPct:     0.6,
		RSIPeriod:      3,
		RSIEntry:       20,
		RSIExit:        75,
		ExitInProfit:   0,
		StochPeriod:    14,
		StochSmooth:    3,
		StochExit:      0,
		ProfitPct:      0.5,
		StopDailyATR:   1.45,
		DailyATRPeriod: 14,
		MaxDayRangeATR: 0,
		VolMult:        0,
		VolBars:        20,
		StartHour:      8,
	}
}
