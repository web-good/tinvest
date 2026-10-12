// Package gazp supplies the ticker and the drop_bounce Params for GAZP (ПАО «Газпром»), one literal
// per timeframe: ParamsMinutes5, ParamsMinutes15, ParamsMinutes30, ParamsHour1.
//
// Калибровка 2026-10-12 скиллом drop-bounce-calibrate на текущем ядре (стоп в дневных ATR, фильтры
// размаха дня и объёма, выход только в плюс ExitInProfit, без EMA и MACD); окно 36 мес,
// walk-forward 12/6, сетки data/params/drop_bounce/gazp/calstop_<таймфрейм>.json (entry → risk →
// day → volume → exits, keepTop 3). 15m, 30m и 1h — по просьбе владельца, 5m — прогоном того же
// дня до появления ExitInProfit.
//
// Baseline на дефолтах, pooled OOS PF: 15m 0.871 на 214, 30m 0.654 на 209, 1h 0.824 на 191 (5m
// ранее 1.200 на 191). Фазовый WF: 15m 1.152 на 63 (живых фолдов три, выше 1 один), 30m 0.733 на
// 58 (живых два, оба ниже 1), 1h 0.665 на 39 (живых три, выше 1 один), 5m ранее 1.347 на 46 (живых
// два). Параметры гуляли по всем осям, фильтры и ExitInProfit выбирались через фолд. Планку не взял
// ни один таймфрейм — drop_bounce на GAZP не торговать. Все четыре литерала — дефолты ядра (стоп
// 0.5 ATR, фильтры выключены, ExitInProfit 1; у 5m ExitInProfit 0 — он калибровался до флага).
// Ручной литерал 15m (MinDropPct 0.7) и старые литералы 30m/1h (калибровка 2026-10-11 на
// процентном стопе с EMA) этим заменены; прежние прогоны записаны в _comment сеток. Числа выписаны
// явно, чтобы бэктест GAZP не уезжал вместе с правкой core.DefaultParams.
//
// Ожидание первой зелёной свечи (GreenWaitBars) 2026-10-12, 15m, 36 мес, литерал 15m: при 0 —
// PF 1.491 на 39 сделках (под комиссией 0.1% — 1.298), при 3 — 1.159 на 38 (под 0.1% — 0.997):
// ожидание отдаёт часть отскока более поздним входом. GreenWaitBars 0 на всех таймфреймах.
package gazp

import "tinvest/internal/service/trading_strategy/drop_bounce/strategy/core"

const Ticker = "GAZP"

// ParamsMinutes5 returns the GAZP literal for Minutes5.
func ParamsMinutes5() core.Params {
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
		GreenWaitBars:  0,
	}
}

// ParamsMinutes15 returns the GAZP literal for Minutes15.
func ParamsMinutes15() core.Params {
	return core.Params{
		MinDropPct:     1.3,
		RSIPeriod:      6,
		RSIEntry:       30,
		RSIExit:        70,
		ExitInProfit:   0,
		StochPeriod:    14,
		StochSmooth:    3,
		StochExit:      80,
		ProfitPct:      1,
		StopDailyATR:   0.7,
		DailyATRPeriod: 14,
		MaxDayRangeATR: 0,
		VolMult:        1.4,
		VolBars:        20,
		StartHour:      8,
		GreenWaitBars:  0,
	}
}

// ParamsMinutes30 returns the GAZP literal for Minutes30.
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
		GreenWaitBars:  0,
	}
}

// ParamsHour1 returns the GAZP literal for Hour1.
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
		GreenWaitBars:  0,
	}
}
