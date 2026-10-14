// Package gazp supplies the drop_bounce Params for GAZP (Газпром) on Hour1.
//
// # Правила 2026-10-14
//
// Ядро переписано (окна покупки 07–11 и 17–24 MSK, выход по RSI(RSIExitPeriod), без гейта хода
// дня, MACD-выхода и пятничных правил). Все замеры ниже сняты на прежних правилах и к новым не
// относятся; литерал сброшен на дефолты ядра до калибровки на новых правилах (скилл
// drop-bounce-calibrate, перебор RSIPeriod × RSIExitPeriod × StopDailyATR).
//
// Последняя калибровка 2026-10-14 по схеме hold-out (скилл drop-bounce-calibrate): перебор
// EMAPeriod (100, 120, 150, 200) × StaleBars (10, 15, 25, 35) × StopDailyATR (0.7, 1.0, 1.5) на
// первых 30 из 36 месяцев (sweep_hour1.json), точка — центр плато, затем стресс и hold-out.
// Гриды и итоги каждого прогона — в _comment файлов data/params/drop_bounce/gazp/, отчёты —
// reports/GAZP_drop_bounce/{sweep,dev,cost}/Hour1/ (вне git).
//
// # Схема hold-out (2026-10-14)
//
//   - Центр плато: EMA 100, стоп 0.7 дневного ATR, STALE 10 баров — PF 1.579 на 54 сделках за
//     14.10.2023 — 14.04.2026; соседи стоп 1.0 → 1.470, STALE 15 → 1.408, EMA 120 → 1.377 (плато
//     есть, но центр в углу сетки). Ручная точка владельца EMA 120 / STALE 25 / стоп 1.0 — 1.182/55,
//     дефолты — 0.959/71.
//   - Стресс: S1 да, S4 да (полугодия в плюсе 4 из 5), S2 нет — без лучшей сделки (10.02.2025 →
//     13.02.2025, +17 587 ₽, рывок на новостях) PF 1.016, S3 нет — без трёх лучших 0.760.
//   - Издержки (комиссия 0.1%): PF 1.362, +12 130 ₽ — выдержаны.
//   - Hold-out (последние 6 месяцев) не открывался: стресс не пройден.
//
// # Прежние замеры (walk-forward 12/6, 2026-10-13)
//
//   - Без STALE: baseline 0.999/58; v1 (calstop_hour1_v1.json) 0.482/35 — поля скакали, pick
//     оставил дефолты; v2 (calstop_hour1_v2.json) 1.321/51, литерал был консенсусом (EMA 110,
//     день 0.68, стоп 0.85, выход 66), планка не взята (D: фолд 0.584 на 18 сделках).
//   - Baseline v4 (дефолты ядра со STALE): pooled OOS PF 1.042 на 66 сделках, compounded
//     +1.14%, фолды 1.865(17) / 0.580(18) / 0.798(14) / 1.081(17).
//   - Сетка v4: pooled OOS PF 1.013 на 50 сделках, compounded −1.37%, фолды 3.193(13) /
//     0.366(17) / 1.022(11) / 0.729(9). Выборы фолдов: EMA 100 во всех, день 0.7/0.7/1/0.5,
//     стоп 1.5/1.5/0.7/0.7, выход 65/65/75/70.
//   - report.py pick: ЛИТЕРАЛ: ДЕФОЛТЫ — подбор не обошёл baseline.
//   - Точка (дефолты, in-sample, не судья): за 36 мес 88 сделок, PF 0.979, −1 221 ₽.
//
// # Решение
//
// Не торговать: весь край GAZP — одна сделка февраля 2025, без неё PF около 1.0 при любых
// параметрах (и на walk-forward, и в переборе). Литерал был центром плато (сброшен 2026-10-14, см. выше). Тикер не входит в
// DROP_BOUNCE_TICKERS_H1.
package gazp

import "tinvest/internal/service/trading_strategy/drop_bounce/strategy/core"

// Ticker is the instrument this package configures.
const Ticker = "GAZP"

// Params returns the GAZP literal for Hour1.
func Params() core.Params {
	return core.Params{
		RSIPeriod:      6,
		RSIZone:        30,
		RSICap:         50,
		EMAPeriod:      200,
		StopDailyATR:   1,
		DailyATRPeriod: 14,
		RSIExitPeriod:  10,
		RSIExit:        70,
		MorningFrom:    7,
		MorningTo:      11,
		EveningFrom:    17,
		EveningTo:      24,
		StaleBars:      0,
		StaleATR:       0.3,
	}
}
