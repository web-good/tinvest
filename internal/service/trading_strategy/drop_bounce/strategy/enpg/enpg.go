// Package enpg supplies the drop_bounce Params for ENPG (ЭН+ ГРУП) on Hour1.
//
// Откалиброван 2026-10-13 на правилах 1h RSI-зоны с выходом RSI+MACD (скилл
// drop-bounce-calibrate), дважды: без выхода по времени (сетка v2) и с выходом STALE 10 баров /
// 0.3 ATR в дефолтах ядра (сетка v4). Гриды и итоги каждого прогона — в _comment файлов
// data/params/drop_bounce/enpg/, отчёты — reports/ENPG_drop_bounce/{baseline,wf,point,
// plateau_entry,plateau_stop,cost}/Hour1/ (вне git; прогоны без STALE — в *_nostale/).
//
// # Протокол
//
// 36 месяцев, walk-forward 12/6 (4 фолда), -min-trades 10, -metric profit_factor. Сетка v4
// (calstop_hour1.json): вход RSI на дефолтах 6/30/50, filter EMAPeriod (100, 200) ×
// DaySpentATR (0.5, 0.7, 1.0) → risk StopDailyATR (0.7, 1.0, 1.5) → exit RSIExit (65, 70, 75),
// keepTop 3, стоп в каждой точке, STALE 10/0.3 прибит.
//
// # Замеры
//
//   - Без STALE (сетка v2, calstop_hour1_v2.json): baseline 1.055/40, подбор 1.194/37, литерал
//     был консенсусом (EMA 200, день 0.6, стоп 1.03, выход 66), планка не взята (A, B нет).
//   - Baseline v4 (дефолты ядра со STALE): pooled OOS PF 1.469 на 48 сделках, compounded
//     +17.41%, фолды 1.615(18) / 1.722(15) / 0.200(6) / 1.675(9).
//   - Сетка v4: pooled OOS PF 1.421 на 43 сделках, compounded +13.71%, фолды 1.246(14) /
//     1.955(15) / 0.200(6) / 1.683(8). Выборы фолдов: EMA 100/200/200/200, стоп 1/1.5/1.5/1,
//     выход 65/75/75/75.
//   - report.py pick: ЛИТЕРАЛ: ДЕФОЛТЫ — подбор не обошёл baseline. Планка (по baseline) взята:
//     A 1.469/48, B +6.96%, C +1.32%, D да.
//   - Точка (дефолты, in-sample, не судья): за 36 мес 65 сделок, PF 1.354, +16 630 ₽, win 46%.
//   - Плато по входу (RSIZone 25/30/35 × DaySpentATR 0.5/0.7/1.0): ОСТРЫЙ ПИК — медиана соседей
//     0.964 (порог 0.948), но четыре соседа ниже 1.0 (худший 0.775 — зона 25, день 1.0).
//   - Плато по стопу: ПЛАТО ЕСТЬ — 0.7 → 1.260, 1.5 → 1.609.
//   - Издержки (комиссия 0.1%): PF 1.189, +9 281 ₽ — ИЗДЕРЖКИ СЪЕДАЮТ КРАЙ (PF < 1.2).
//
// Оговорка: правило STALE 10/0.3 выведено из разбора сделок ENPG, поэтому baseline v4 на ENPG —
// не независимая проверка.
//
// # Решение
//
// Не торговать: walk-forward планку взял, но §4 провален — плато по входу острое (соседи по
// зоне RSI и ходу дня ниже 1.0), удвоенная комиссия опускает PF до 1.189. Тикер не входит в
// DROP_BOUNCE_TICKERS_H1.
package enpg

import "tinvest/internal/service/trading_strategy/drop_bounce/strategy/core"

// Ticker is the instrument this package configures.
const Ticker = "ENPG"

// Params returns the ENPG literal for Hour1.
func Params() core.Params {
	return core.Params{
		RSIPeriod:      6,
		RSIZone:        30,
		RSICap:         50,
		EMAPeriod:      200,
		DaySpentATR:    0.7,
		StopDailyATR:   1.0,
		DailyATRPeriod: 14,
		RSIExit:        70,
		StartHour:      7,
		MACDFast:       12,
		MACDSlow:       26,
		MACDSignal:     9,
		StaleBars:      10,
		StaleATR:       0.3,
	}
}
