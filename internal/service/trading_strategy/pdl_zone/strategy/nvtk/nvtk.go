// Package nvtk supplies the pdl_zone Params for NVTK (Новатэк) on Minutes5.
//
// # 2026-10-21 — литерал до калибровки
//
// Числа поставил владелец (2026-10-21): только пробой с возвратом над PDL (EntryZone 0,
// BreakReclaim 1), TargetR 1.7, MinRoomATR 0.2; остальное — core.DefaultParams. Записаны литералом,
// чтобы правка дефолтов ядра не меняла тикер молча.
// Калибровка — скользящий walk-forward, сетка data/params/pdl_zone/nvtk/wf_rolling.json.
package nvtk

import "tinvest/internal/service/trading_strategy/pdl_zone/strategy/core"

// Params returns the NVTK literal.
func Params() core.Params {
	return core.Params{
		// Взвод сетапа
		EntryZone:    0,   // 1 — взвод заходом в зону над PDL (low ≤ PDL + ZoneATR·ATR, close ≥ PDL); 0 — выключен
		EntryBreak:   1,   // 1 — взвод закрытием ниже PDL (пробой); 0 — выключен
		ZoneATR:      0.2, // высота зоны над PDL в дневных ATR (работает только при EntryZone 1)
		BreakReclaim: 1,   // 1 — после пробоя сигнал только после бара, закрывшегося обратно выше PDL; 0 — первый зелёный бар после пробоя

		// Сигнал: первый закрытый бар с зелёной гистограммой MACD (macd − signal > 0) от бара взвода
		MACDFast:   8,  // период быстрой EMA MACD на 5m
		MACDSlow:   21, // период медленной EMA MACD на 5m
		MACDSignal: 9,  // период сигнальной линии MACD

		// Фильтр объёма: средний объём баров основной сессии от взвода до сигнала >
		// VolumeMult × среднего объёма VolBars баров основной сессии перед взводом
		VolumeMult: 1.5, // во сколько раз объём окна должен превышать базу; 0 — фильтр выключен
		VolBars:    15,  // сколько баров основной сессии перед взводом берётся в базу

		// Стоп и цель
		StopBars:   5,   // стоп — минимальный low этого числа баров перед сигнальным
		MinStopATR: 0.2, // стоп не ближе этого числа дневных ATR к цене покупки (иначе опускается)
		TargetR:    1.7, // цель — столько расстояний до стопа выше цены покупки

		// Запас хода дня на сигнале: (лой дня + 1 ATR − close сигнала) / ATR ≥ MinRoomATR; 0 — выключен
		MinRoomATR: 0.5,

		DailyATRPeriod: 14, // период дневного ATR (Wilder) по завершённым будним дневкам
	}
}
