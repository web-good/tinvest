// Package sber supplies the drop_bounce Params for SBER (Сбербанк) on Minutes15.
//
// # Перекалибровка 2026-10-15
//
// Перекалибровка 2026-10-15 по просьбе владельца (схема hold-out, 45 точек, -risk-pct 1, BreakevenPct
// 0.1, выбор по свежим месяцам при ничьей; отчёты reports/SBER_drop_bounce/r1015/). Центр: #1 —
// RSIPeriod 4, StopDailyATR 1, RSIExitPeriod 11, PF 0.657 на 45 сделках; соседи: RSIExitPeriod 12: PF
// 0.648/45; StopDailyATR 0.7: PF 0.618/46; RSIExitPeriod 10: PF 0.578/45; RSIPeriod 5: PF 0.487/41;
// ОСТРЫЙ ПИК. Точка: RSI входа 4, RSI выхода 11, стоп 1 ATR. Разработка: 45 сделок, PF 0.657, net
// -4094 ₽. Стресс: S1 PF 0.657 ≥ 1.3 и 45 ≥ 25 сделок: нет; S2 без лучшей сделки (+796 ₽) PF 0.591 ≥
// 1.15: нет; S3 без трёх лучших PF 0.471 ≥ 1.0: нет; S4 полугодий в плюсе 1 из 5 (нужно ≥ 4): нет.
// Издержки 0.1%: PF 0.476, net -6677 ₽ на 45 сделках. hold-out не открывался. Решение: не торговать
// (острый пик).
//
// # Перекалибровка 2026-10-14 с -risk-pct 1
//
// Та же схема (перебор 45 точек на 30 из 36 мес., центр плато, стресс, hold-out), но размер позиции —
// 1% капитала на стоп (-risk-pct 1), PrevCloseGate 0. Литерал — новый центр (point_minutes30.json);
// разделы ниже — прежние замеры при фиксированной доле.
//
// Центр плато: RSI входа 4, RSI выхода 11, стоп 1 дневного ATR — PF 0.675 на 45 сделках (лучшая
// точка перебора PF 0.6749); соседи: стоп 0.7 → 0.604/46, RSI выхода 12 → 0.594/45, RSI выхода 10 →
// 0.555/45, RSI входа 5 → 0.506/41 — ОСТРЫЙ ПИК. Стресс, издержки и hold-out не запускались.
// Решение: не торговать.
//
// # Калибровка 2026-10-14 (30m: сигнал на 1h, подтверждение MACD на 30m)
//
// Схема hold-out (скилл drop-bounce-calibrate): перебор RSIPeriod (4, 5, 6) × RSIExitPeriod (8–12) ×
// StopDailyATR (0.5, 0.7, 1.0), 45 точек, на первых 30 из 36 месяцев (sweep_minutes30.json),
// остальное — дефолты ядра (ConfirmBars 3, безубыток 0.5 ATR, пятница до 11:00). Отчёты —
// reports/SBER_drop_bounce/{sweep,dev,cost,full}/Minutes30/ (вне git). Тикер взят в волне из семи
// голубых фишек, чтобы выводы о стратегии опирались на широкую вселенную.
//
//   - Перебор: все 45 точек PF < 1 (0.171–0.583, 30–47 сделок на 30 мес.).
//   - Центр плато: RSI входа 4, RSI выхода 11, стоп 1.0 дневного ATR — PF 0.583 на 45 сделках, медиана
//     окрестности 0.564; соседи стоп 0.7 → 0.574, RSI выхода 12 → 0.564, RSI выхода 10 → 0.494, RSI
//     входа 5 → 0.409. ОСТРЫЙ ПИК.
//   - Стресс, издержки и hold-out не гонялись: плато не пройдено.
//
// Решение: не торговать. Литерал — центр перебора (point_minutes30.json).
//
// # Переход на 15m (2026-10-15)
//
// Владелец перевёл стратегию на 15m: сигнал на часе, затем зелёная гистограмма часового MACD(8,26,5)
// не позже 20 часов, затем MACD(12,26,9) на 15m за 6 баров; безубыток, трейлинг и выключенные фильтры
// удалены из ядра. Точка ниже (RSI входа, RSI выхода, стоп) снята на прежних правилах (30m, безубыток
// 0.5 ATR) и на новых не проверена: тикер ждёт перекалибровки скиллом drop-bounce-calibrate.
package sber

import "tinvest/internal/service/trading_strategy/drop_bounce/strategy/core"

// Ticker is the instrument this package configures.
const Ticker = "SBER"

// Params returns the SBER literal for Minutes15.
func Params() core.Params {
	return core.Params{
		RSIPeriod:      4,
		RSIZone:        30,
		RSICap:         50,
		ZoneEntry:      1,
		EMAPeriod:      200,
		StopDailyATR:   1,
		DailyATRPeriod: 14,
		RSIExitPeriod:  11,
		RSIExit:        70,
		HourMACDFast:   8,
		HourMACDSlow:   26,
		HourMACDSignal: 5,
		HourMACDHours:  20,
		MACDFast:       12,
		MACDSlow:       26,
		MACDSignal:     9,
		ConfirmBars:    6,
		FridayTo:       11,
	}
}
