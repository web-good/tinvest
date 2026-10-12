// Package ivat supplies the ticker and the drop_bounce Params for IVAT (ПАО «ИВА»), one
// literal per timeframe: ParamsMinutes5, ParamsMinutes15, ParamsMinutes30, ParamsHour1.
//
// Калибровка 2026-10-11/12, окно 36 мес, сетки data/params/drop_bounce/ivat/cal_<таймфрейм>.json (та
// же сетка, что у GAZP/DOMRF). Литералы — лучшая in-sample точка со стопом: у 30m и 1h это точка #1
// (стоп 0.7, 2.163/41 и 3.623/34); у 5m и 15m весь топ-20 был без стопа, стоп подобран сеткой
// stop_<таймфрейм>.json поверх точки #1 и съел край: 5m 9.488 → 2.353/31 (стоп 1%), 15m 3.343 →
// 1.195/48 (стоп 1.5%) — точка 15m живёт пересиживанием убытков. Walk-forward 36/12/6, pooled OOS
// PF: 5m 1.497 на 185 сделках (фолды 0.725/60, 1.572/119 с просадкой 28%, 2 и 4 сделки; худшая
// сделка −36.7 тыс. ₽), 15m 1.220 на 49 (два последних фолда в минусе), 30m 1.779 на 85 (три из
// четырёх), 1h 1.165 на 81 (последний фолд −17%, просадка 48%). Параметры между фолдами гуляли.
// Числа выписаны явно, чтобы бэктест IVAT не уезжал вместе с правкой core.DefaultParams.
//
// 2026-10-12 стоп переведён с процента на дневной ATR: StopPct (5m/15m/30m/1h 1/1.5/0.7/0.7 %) пересчитан
// в StopDailyATR 0.25/0.4/0.2/0.2 через медианный дневной ATR тикера за историю с 2024-06 (3.88%, Wilder 14 по
// будним дневкам), шаг 0.05. Это пересчёт, не калибровка: числа walk-forward выше получены на
// процентном стопе. Фильтр размаха дня MaxDayRangeATR выключен.
//
// 2026-10-12 из ядра убраны фильтр тренда EMA и выход MACD. Литералы потеряли поля
// (5m/15m/30m/1h) MACDExit 0/0/1/1, EMAPeriod 100/100/0/0; все числа PF выше получены с ними и к текущему поведению не относятся.
//
// 15m перекалиброван 2026-10-12 скиллом drop-bounce-calibrate на текущем ядре (стоп в дневных
// ATR, фильтры размаха дня и объёма, ExitInProfit), сетка calstop_minutes15.json, окно 36 мес
// (дневки с 2024-06), walk-forward 12/6: baseline дефолтов pooled OOS PF 0.928 на 336 (три фолда
// из четырёх выше 1, последний 0.744 с просадкой 39%), фазовый WF 0.973 на 73 (3.468 / 1.360 /
// вырожденный / 0.237 с просадкой 42%). Планка не взята — литерал 15m — дефолты ядра; ручная
// правка 15m (RSIPeriod 6, стоп 0.5) и числа 15m выше этим заменены.
package ivat

import "tinvest/internal/service/trading_strategy/drop_bounce/strategy/core"

const Ticker = "IVAT"

// ParamsMinutes5 returns the IVAT literal for Minutes5.
func ParamsMinutes5() core.Params {
	return core.Params{
		MinDropPct:     0.3,
		RSIPeriod:      6,
		RSIEntry:       20,
		RSIExit:        70,
		ExitInProfit:   0,
		StochPeriod:    14,
		StochSmooth:    3,
		StochExit:      0,
		ProfitPct:      1.5,
		StopDailyATR:   0.25,
		DailyATRPeriod: 14,
		MaxDayRangeATR: 0,
		VolMult:        0,
		VolBars:        20,
		StartHour:      8,
	}
}

// ParamsMinutes15 returns the IVAT literal for Minutes15.
func ParamsMinutes15() core.Params {
	return core.Params{
		MinDropPct:     0.6,
		RSIPeriod:      6,
		RSIEntry:       30,
		RSIExit:        70,
		ExitInProfit:   1,
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

// ParamsMinutes30 returns the IVAT literal for Minutes30.
func ParamsMinutes30() core.Params {
	return core.Params{
		MinDropPct:     0.8,
		RSIPeriod:      9,
		RSIEntry:       20,
		RSIExit:        0,
		ExitInProfit:   0,
		StochPeriod:    14,
		StochSmooth:    3,
		StochExit:      80,
		ProfitPct:      0,
		StopDailyATR:   0.2,
		DailyATRPeriod: 14,
		MaxDayRangeATR: 0,
		VolMult:        0,
		VolBars:        20,
		StartHour:      8,
	}
}

// ParamsHour1 returns the IVAT literal for Hour1.
func ParamsHour1() core.Params {
	return core.Params{
		MinDropPct:     1.4,
		RSIPeriod:      9,
		RSIEntry:       20,
		RSIExit:        75,
		ExitInProfit:   0,
		StochPeriod:    14,
		StochSmooth:    3,
		StochExit:      0,
		ProfitPct:      0.5,
		StopDailyATR:   0.2,
		DailyATRPeriod: 14,
		MaxDayRangeATR: 0,
		VolMult:        0,
		VolBars:        20,
		StartHour:      8,
	}
}
