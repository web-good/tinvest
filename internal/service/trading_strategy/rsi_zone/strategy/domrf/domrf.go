// Package domrf supplies the ticker and calibrated rsi_zone Params for DOMRF (ДОМ.РФ).
//
// Откалиброван 2026-09-28. Гриды и итоги каждого прогона — в _comment файлов
// data/params/rsi_zone/domrf/.
//
// # Протокол
//
// История 30-минуток начинается в декабре 2025 (~10 месяцев), штатные -months 36
// -train-months 12 -test-months 6 не дают ни одного фолда. Прогоны шли по -months 10
// -train-months 3 -test-months 2 -min-trades 15 -metric profit_factor — 3 фолда с OOS-окнами
// 2026-02-28—04-28, 04-28—06-28, 06-28—08-28. Все темы свипали оси поверх core.DefaultParams(),
// а не поверх этого литерала.
//
// # Результат: планку 1.5 взяла одна тема из пяти
//
// Pooled OOS PF: trend 4.044/33, trend→risk 1.762/46, risk 1.123/54, entry 1.094/32, phased
// 1.026/30. Тема trend выбрала EMAPeriod 50 во всех трёх фолдах. Остальные темы гуляли по всем
// полям, и чем раньше в них шла фаза входа, тем хуже: entry переобучается (in-sample PF 60.8 и
// 10500 в первом фолде).
//
// # Литерал
//
// Дефолты ядра, изменено одно поле: EMAPeriod 200 → 50.
//
//   - StopDailyATR 1.0 — самый широкий стоп, который на этой истории ещё срабатывает (2 SL-выхода
//     из 56). При 1.5, 2.0 и 3.0 SL-выходов ноль и PF одинаковый 5.139 — капкан широкого стопа,
//     отвергнут. 0.7 даёт PF 1.883 при 6 SL-выходах.
//
// Замеры литерала. Walk-forward сеткой из одной точки: pooled 4.044 на 33 сделках, фолды
// 0.833 / 3.447 / 16.859. Полная история (12 мес): 56 сделок, PF 2.693, +17.2%, просадка 4.0%.
// Удвоенная комиссия 0.001: pooled 2.500, полная история 1.968.
//
// Риски: первый фолд (март–апрель 2026) убыточен; третий (PF 16.9) делает половину результата;
// три фолда на 10 месяцах. Сентябрь 2026 не попал ни в одно окно выбора и убыточен (−3.9%, из
// них −3.0% — один стоп).
package domrf

import "tinvest/internal/service/trading_strategy/rsi_zone/strategy/core"

// Ticker is the instrument this package configures.
const Ticker = "DOMRF"

// DefaultParams returns DOMRF's calibrated rsi_zone parameters.
func DefaultParams() core.Params {
	return core.Params{
		RSIPeriod:      4,
		RSILower:       25,
		RSIUpper:       75,
		EMAPeriod:      50,
		DailyATRPeriod: 14,
		StopDailyATR:   1.0,
	}
}
