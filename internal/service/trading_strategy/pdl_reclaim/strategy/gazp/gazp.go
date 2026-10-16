// Package gazp supplies the pdl_reclaim Params for GAZP (Газпром) on Minutes5.
//
// # Заведён 2026-10-16 без калибровки
//
// Пакет заведён по просьбе владельца до калибровки, чтобы у GAZP были свои переключатели
// таймфреймов и фильтры. Числа — дефолты ядра на эту дату, записанные литералом (правка дефолтов
// ядра GAZP не меняет), кроме выбора владельца:
//   - только утро на 5m (Entry30m 0); часовой RSI-гейт (HourRSIPeriod 6, HourRSIMax 30) записан,
//     но при Entry30m 0 не работает;
//   - только вчерашние уровни (LevelsOnly 1): отскок от PDL формацией разворота, low которой дошёл
//     до PDL с допуском 0.1 дневного ATR (LevelTolATR 0.1), или пробой PDH (EntryC 1);
//   - сила свечи паттерна: тело и отрыв от уровня больше 0.1 дневного ATR (BodyATR 0.1);
//   - фильтр объёма выключен (VolumeMult 0).
//
// Калибровка (окно -months 12) заменит числа и допишет сюда замеры.
package gazp

import "tinvest/internal/service/trading_strategy/pdl_reclaim/strategy/core"

// Params is the GAZP Minutes5 literal.
func Params() core.Params {
	return core.Params{
		EntryA:          1,
		EntryB:          1,
		StopDailyATR:    0.5,
		DailyATRPeriod:  14,
		TakeProfitPDH:   1,
		RSIPeriod:       6,
		RSIExit:         70,
		RSIEntryMax:     50,
		MACDFast:        9,
		MACDSlow:        26,
		MACDSignal:      8,
		EntryFromHour:   7,
		EntryFromMinute: 0,
		EntryToHour:     18,
		EntryToMinute:   0,
		ExitHour:        23,
		ExitMinute:      45,
		SwitchHour:      10,
		SwitchMinute:    0,
		GapOnly:         0,
		Confirm:         0,
		TargetMid:       0,
		Entry5m:         1,
		Entry30m:        0,
		VolumeMult:      0,
		VolDays:         20,
		HourRSIPeriod:   6,
		HourRSIMax:      30,
		EntryC:          1,
		BodyATR:         0.1,
		LevelsOnly:      1,
		LevelTolATR:     0.1,
	}
}
