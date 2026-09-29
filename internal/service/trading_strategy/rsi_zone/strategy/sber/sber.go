// Package sber supplies the ticker and rsi_zone Params for SBER (Сбербанк).
//
// Откалиброван 2026-09-28, но ПЛАНКУ НЕ ВЗЯЛ: литерал — лучшая точка in-sample, а не
// подтверждённый edge. Гриды и итоги каждого прогона — в _comment файлов
// data/params/rsi_zone/sber/.
//
// # Протокол и результат
//
// Штатный walk-forward -months 36 -train-months 12 -test-months 6 -min-trades 20 -metric
// profit_factor, 4 фолда; темы свипали оси поверх core.DefaultParams(). Pooled OOS PF: phased
// 1.360/94 (фолды 2.521 / 0.719 / 0.828 / 1.294) — единственная честная цифра; entry
// 1.135/94, trend 1.080/147, risk 0.794/209. Совместная сетка cal_joint дала 1.732/87, но
// строилась после просмотра тем, поэтому завышена. Дефолты ядра за 36 месяцев: PF 0.782 на 299
// сделках.
//
// # Литерал
//
// Взят из in-sample плато cal_joint за полные 36 месяцев (кластер RSI(4)@15 и EMA 100):
// RSIPeriod 4, RSILower 15, RSIUpper 75, EMAPeriod 100, StopDailyATR 2.0.
//
//   - StopDailyATR 2.0 — самый широкий стоп, который ещё срабатывает (2 SL-выхода из 93). При 3.0
//     PF растёт до 1.980, но SL-выходов ноль — капкан широкого стопа, отвергнут.
//
// Замеры литерала (не OOS: точка видела эти месяцы). Полные 36 месяцев: 93 сделки, PF 1.575,
// +12.3%, просадка 6.0%. Прогон по окнам фолдов: pooled 2.038 на 60 сделках, фолды 7.530 /
// 1.847 / 0.672 / 2.508; при удвоенной комиссии 0.001 — 1.441. Третий фолд
// (2025-09 — 2026-03) убыточен у всех кандидатов.
package sber

import "tinvest/internal/service/trading_strategy/rsi_zone/strategy/core"

// Ticker is the instrument this package configures.
const Ticker = "SBER"

// DefaultParams returns SBER's rsi_zone parameters (best in-sample point; the bar is not met).
func DefaultParams() core.Params {
	return core.Params{
		RSIPeriod:      4,
		RSILower:       15,
		RSIUpper:       75,
		EMAPeriod:      100,
		DailyATRPeriod: 14,
		StopDailyATR:   2.0,
	}
}
