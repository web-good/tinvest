// Package tbank supplies the ticker and the drop_bounce Params for T (МКПАО «Т-Технологии»), one
// literal per timeframe: ParamsMinutes5, ParamsMinutes15, ParamsMinutes30, ParamsHour1.
//
// Калибровка 15m, 30m и 1h 2026-10-12 скиллом drop-bounce-calibrate по просьбе владельца, на
// текущем ядре (стоп в дневных ATR, фильтры размаха дня и объёма, ExitInProfit, без EMA и MACD);
// окно 36 мес, walk-forward 12/6, сетки data/params/drop_bounce/t/calstop_<таймфрейм>.json (entry →
// risk → day → volume → exits, keepTop 3). Baseline на дефолтах, pooled OOS PF: 15m 0.565 на 129,
// 30m 0.800 на 150, 1h 0.903 на 142 — везде выше 1 только один фолд. Фазовый WF: 15m 1.977 на 29
// (живых фолдов два), 30m 1.220 на 26 (живых два, выше 1 один), 1h 1.613 на 22 (живой один);
// сделки сжимаются до 22–29, параметры гуляли по всем осям. Планку не взял ни один таймфрейм —
// литералы 15m, 30m и 1h — дефолты ядра (старые точки 2026-10-11 на процентном стопе с EMA/MACD
// этим заменены).
//
// 5m не перекалибровывался: литерал — in-sample точка 2026-10-11 (WF pooled 2.144 на 36 на
// процентном стопе с EMA(100)), стоп 3% пересчитан в StopDailyATR 1.15 через медианный дневной ATR
// (2.66%), EMA-фильтр потерян при удалении из ядра — к текущему поведению эти числа не относятся.
// Числа выписаны явно, чтобы бэктест T не уезжал вместе с правкой core.DefaultParams.
package tbank

import "tinvest/internal/service/trading_strategy/drop_bounce/strategy/core"

const Ticker = "T"

// ParamsMinutes5 returns the T literal for Minutes5.
func ParamsMinutes5() core.Params {
	return core.Params{
		MinDropPct:     0.5,
		RSIPeriod:      3,
		RSIEntry:       20,
		RSIExit:        70,
		ExitInProfit:   0,
		StochPeriod:    14,
		StochSmooth:    3,
		StochExit:      0,
		ProfitPct:      0.5,
		StopDailyATR:   1.15,
		DailyATRPeriod: 14,
		MaxDayRangeATR: 0,
		VolMult:        0,
		VolBars:        20,
		StartHour:      8,
	}
}

// ParamsMinutes15 returns the T literal for Minutes15.
func ParamsMinutes15() core.Params {
	return core.Params{
		MinDropPct:     0.5,
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
		GreenWaitBars:  3,
	}
}

// ParamsMinutes30 returns the T literal for Minutes30.
func ParamsMinutes30() core.Params {
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

// ParamsHour1 returns the T literal for Hour1.
func ParamsHour1() core.Params {
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
