// Package nvtk supplies the pdl_reclaim Params for NVTK (Новатэк) on Minutes15.
//
// # Калибровка 2026-10-19 — планка не взята, литерал заведён по решению владельца
//
// Окно -months 12 (2025-10-19 — 2026-10-19): разработка — первые 9 месяцев (-holdout-months 3),
// hold-out — последние 3 месяца. Сетки и точка — data/params/pdl_reclaim/nvtk/.
//
// Полная сетка sweep_full.json (1152 точки, оси как у GAZP/SBER): лучшая точка — обе ветки,
// HoldBars 3, стоп 0.7, 10:00–16:00, IndexTrendEMA 50 — PF 1.409/23 net +7612. Стоп 0.7 лёг на край
// сетки; расширение вниз (sweep_stop_low.json) показало, что 0.7 — пик внутри (0.6 — 1.329/24).
// MaxDayRangeATR 0 и 1.2 дают одни и те же сделки; взят 1.2, центр окрестности (медиана 1.312).
// Фильтр индекса 200, лучший у GAZP и SBER, здесь даёт PF около 1.1.
//
// Планка PF ≥ 1.3 на ≥ 25 сделках не взята, и вне разработки точка проигрывает: без лучшей сделки
// 1.253, без трёх лучших 0.967; hold-out 7 сделок PF 0.280 net −9188; весь год 30 сделок PF 0.950
// net −1576; год до окна (-months 24 -holdout-months 12) 43 сделки PF 0.581 net −20155.
//
// Скользящий walk-forward (wf_rolling.json: подбор 3 месяца, проверка 1 месяц, 24 месяца, 21 фолд,
// компактная сетка) — пул OOS 83 сделки PF 0.638, compounded −32.4%; та же сетка у SBER 1.298,
// у GAZP 0.973. Края на NVTK не найдено: литерал записан, чтобы тикер гонялся с лучшей точкой, а
// не с дефолтами ядра; в PDL_RECLAIM_TICKERS_M15 он не добавлен.
//
// # 2026-10-19 — окно входа удалено из ядра
//
// Параметры EntryFrom*/EntryTo* удалены по решению владельца: вход в любой будний бар, покупка в ту
// же дату. Окно 10:00–16:00 из замеров выше больше не действует; литерал без перекалибровки, 24 месяца:
// 115 сделок, PF 0.667, net −34092.
//
// # 2026-10-20 — молот у PDL
//
// Ядро получило PDLEngulf/PDLHammer. Сетка wf_patterns.json (4 точки), -months 12: молот PF
// 1.270/35, оба 1.098/36, без паттернов 0.794/54, поглощение 0.721/24; на 36 месяцах молот тоже
// лучший, но в минусе — 0.675/104 против 0.647/168. Walk-forward (3/1, 9 фолдов): выбор паттерна на
// подборе — пул OOS 1.065, фиксированный молот 1.312, без паттернов 0.796. Тикер по-прежнему не в
// PDL_RECLAIM_TICKERS_M15.
package nvtk

import "tinvest/internal/service/trading_strategy/pdl_reclaim/strategy/core"

// Params is the NVTK Minutes15 literal.
func Params() core.Params {
	return core.Params{
		EntryPDL:       1,
		EntryPDH:       1,
		HoldBars:       3,
		StopDailyATR:   0.7,
		DailyATRPeriod: 14,
		VolumeMult:     1.5,
		VolBars:        20,
		MaxDayRangeATR: 1.2,
		IndexTrendEMA:  50,
		PDLEngulf:      0,
		PDLHammer:      1,
	}
}
