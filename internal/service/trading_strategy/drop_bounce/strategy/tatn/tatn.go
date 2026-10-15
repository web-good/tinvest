// Package tatn supplies the drop_bounce Params for TATN (Татнефть) on Minutes15.
//
// # Перекалибровка 2026-10-15
//
// Перекалибровка 2026-10-15 по просьбе владельца (схема hold-out, 45 точек, -risk-pct 1, BreakevenPct
// 0.1, выбор по свежим месяцам при ничьей; отчёты reports/TATN_drop_bounce/r1015/). Центр: #5 —
// RSIPeriod 4, StopDailyATR 0.5, RSIExitPeriod 10, PF 1.170 на 44 сделках; соседи: RSIExitPeriod 11:
// PF 1.155/43; RSIExitPeriod 9: PF 1.149/44; RSIPeriod 5: PF 1.078/30; StopDailyATR 0.7: PF 0.914/43;
// ОСТРЫЙ ПИК. Точка: RSI входа 4, RSI выхода 10, стоп 0.5 ATR. Разработка: 44 сделок, PF 1.170, net
// +3416 ₽. Стресс: S1 PF 1.170 ≥ 1.3 и 44 ≥ 25 сделок: нет; S2 без лучшей сделки (+4813 ₽) PF 0.930 ≥
// 1.15: нет; S3 без трёх лучших PF 0.746 ≥ 1.0: нет; S4 полугодий в плюсе 3 из 5 (нужно ≥ 4): нет.
// Издержки 0.1%: PF 1.011, net 241 ₽ на 44 сделках. hold-out не открывался. Решение: не торговать
// (острый пик).
//
// # Перекалибровка 2026-10-14 с -risk-pct 1
//
// Та же схема (перебор 45 точек на 30 из 36 мес., центр плато, стресс, hold-out), но размер позиции —
// 1% капитала на стоп (-risk-pct 1), PrevCloseGate 0. Литерал — новый центр (point_minutes30.json);
// разделы ниже — прежние замеры при фиксированной доле.
//
// Центр плато: RSI входа 4, RSI выхода 10, стоп 0.5 дневного ATR — PF 1.170 на 44 сделках (лучшая
// точка перебора PF 1.317); соседи: RSI выхода 11 → 1.215/43, RSI выхода 9 → 1.149/44, RSI входа 5
// → 1.078/30, стоп 0.7 → 0.914/43 — ОСТРЫЙ ПИК. Стресс, издержки и hold-out не запускались.
// Решение: не торговать.
//
// # Калибровка 2026-10-14 (30m: сигнал на 1h, подтверждение MACD на 30m)
//
// Схема hold-out (скилл drop-bounce-calibrate): перебор RSIPeriod (4, 5, 6) × RSIExitPeriod (8–12) ×
// StopDailyATR (0.5, 0.7, 1.0), 45 точек, на первых 30 из 36 месяцев (sweep_minutes30.json),
// остальное — дефолты ядра (ConfirmBars 3, безубыток 0.5 ATR, пятница до 11:00). Отчёты —
// reports/TATN_drop_bounce/{sweep,dev,cost,full}/Minutes30/ (вне git). Тикер взят в волне из семи
// голубых фишек, чтобы выводы о стратегии опирались на широкую вселенную.
//
//   - Перебор: PF 0.476–1.084, выше 1 только 5 точек из 45, 26–44 сделки на 30 мес..
//   - Центр плато: RSI входа 4, RSI выхода 10, стоп 0.5 дневного ATR — PF 0.991 на 44 сделках, медиана
//     окрестности 0.978; соседи RSI выхода 11 → 1.084, RSI выхода 9 → 0.978, RSI входа 5 → 0.829, стоп
//     0.7 → 0.755. ОСТРЫЙ ПИК.
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
package tatn

import "tinvest/internal/service/trading_strategy/drop_bounce/strategy/core"

// Ticker is the instrument this package configures.
const Ticker = "TATN"

// Params returns the TATN literal for Minutes15.
func Params() core.Params {
	return core.Params{
		RSIPeriod:      4,
		RSIZone:        30,
		RSICap:         50,
		ZoneEntry:      1,
		EMAPeriod:      200,
		StopDailyATR:   0.5,
		DailyATRPeriod: 14,
		RSIExitPeriod:  10,
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
