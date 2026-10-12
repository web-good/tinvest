// Package wush supplies the ticker and the drop_bounce Params for WUSH (ПАО «ВУШ Холдинг»), one
// literal per timeframe: ParamsMinutes5, ParamsMinutes15, ParamsMinutes30, ParamsHour1.
//
// Калибровка 2026-10-12 скиллом drop-bounce-calibrate, окно 36 мес, walk-forward 12/6, сетки
// data/params/drop_bounce/wush/calstop_<таймфрейм>.json (фазы entry → risk → exits, стоп без нуля).
// Baseline на дефолтах убыточен: 5m 0.949 на 670 сделках, 15m 0.862. Фазовый WF со стопом:
// 5m pooled OOS PF 1.082 на 29 сделках (последний фолд 0.170, просадка 11.7%), 15m 0.691 на 38
// (выше 1 один фолд из четырёх). Планку не взял ни один таймфрейм — drop_bounce на WUSH не
// торговать. Все четыре литерала — дефолты ядра, выписанные явно, чтобы бэктест WUSH не уезжал
// вместе с правкой core.DefaultParams.
//
// 2026-10-12 стоп переведён с процента на дневной ATR: StopPct (5m/15m/30m/1h 1/1/1/1 %) пересчитан
// в StopDailyATR 0.25/0.25/0.25/0.25 через медианный дневной ATR тикера за 36 мес (4.44%, Wilder 14 по
// будним дневкам), шаг 0.05. Это пересчёт, не калибровка: числа walk-forward выше получены на
// процентном стопе. Фильтр размаха дня MaxDayRangeATR выключен.
//
// 2026-10-12 из ядра убраны фильтр тренда EMA и выход MACD. Литералы потеряли поля
// (5m/15m/30m/1h) MACDExit 1/1/1/1, EMAPeriod 0/0/0/0; все числа PF выше получены с ними и к текущему поведению не относятся.
package wush

import "tinvest/internal/service/trading_strategy/drop_bounce/strategy/core"

const Ticker = "WUSH"

// ParamsMinutes5 returns the WUSH literal for Minutes5.
func ParamsMinutes5() core.Params {
	return core.Params{
		MinDropPct:     1.5,
		RSIPeriod:      6,
		RSIEntry:       30,
		RSIExit:        70,
		ExitInProfit:   0,
		StochPeriod:    14,
		StochSmooth:    3,
		StochExit:      80,
		ProfitPct:      1,
		StopDailyATR:   0.25,
		DailyATRPeriod: 14,
		MaxDayRangeATR: 0,
		VolMult:        0,
		VolBars:        20,
		StartHour:      8,
	}
}

// ParamsMinutes15 returns the WUSH literal for Minutes15.
func ParamsMinutes15() core.Params {
	return core.Params{
		MinDropPct:     1,
		RSIPeriod:      6,
		RSIEntry:       30,
		RSIExit:        70,
		ExitInProfit:   0,
		StochPeriod:    14,
		StochSmooth:    3,
		StochExit:      80,
		ProfitPct:      2,
		StopDailyATR:   0.25,
		DailyATRPeriod: 14,
		MaxDayRangeATR: 0.5,
		VolMult:        0,
		VolBars:        20,
		StartHour:      8,
	}
}

// ParamsMinutes30 returns the WUSH literal for Minutes30.
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
		StopDailyATR:   0.25,
		DailyATRPeriod: 14,
		MaxDayRangeATR: 0,
		VolMult:        0,
		VolBars:        20,
		StartHour:      8,
	}
}

// ParamsHour1 returns the WUSH literal for Hour1.
func ParamsHour1() core.Params {
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
		StopDailyATR:   0.25,
		DailyATRPeriod: 14,
		MaxDayRangeATR: 0,
		VolMult:        0,
		VolBars:        20,
		StartHour:      8,
	}
}
