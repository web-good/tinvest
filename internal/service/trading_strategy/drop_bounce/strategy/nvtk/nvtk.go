// Package nvtk supplies the drop_bounce Params for NVTK (Новатэк) on Minutes15.
//
// # Перекалибровка 2026-10-15
//
// Перекалибровка 2026-10-15 по просьбе владельца (схема hold-out, 45 точек, -risk-pct 1, BreakevenPct
// 0.1, выбор по свежим месяцам при ничьей; отчёты reports/NVTK_drop_bounce/r1015/). Центр: #1 —
// RSIPeriod 4, StopDailyATR 0.5, RSIExitPeriod 8, PF 1.250 на 43 сделках; соседи: StopDailyATR 0.7: PF
// 1.045/41; RSIExitPeriod 9: PF 0.928/43; RSIPeriod 5: PF 0.663/38; ОСТРЫЙ ПИК. Точка: RSI входа 4,
// RSI выхода 8, стоп 0.5 ATR. Разработка: 43 сделок, PF 1.250, net +4442 ₽. Стресс: S1 PF 1.250 ≥ 1.3
// и 43 ≥ 25 сделок: нет; S2 без лучшей сделки (+1970 ₽) PF 1.139 ≥ 1.15: нет; S3 без трёх лучших PF
// 0.940 ≥ 1.0: нет; S4 полугодий в плюсе 3 из 5 (нужно ≥ 4): нет. Издержки 0.1%: PF 1.076, net 1427 ₽
// на 43 сделках. hold-out не открывался. Решение: не торговать (острый пик).
//
// # Перекалибровка 2026-10-14 с -risk-pct 1
//
// Та же схема (перебор 45 точек на 30 из 36 мес., центр плато, стресс, hold-out), но размер позиции —
// 1% капитала на стоп (-risk-pct 1), PrevCloseGate 0. Литерал — новый центр (point_minutes30.json);
// разделы ниже — прежние замеры при фиксированной доле.
//
// Центр плато: RSI входа 4, RSI выхода 9, стоп 0.5 дневного ATR — PF 1.120 на 43 сделках (лучшая
// точка перебора PF 1.298); соседи: RSI выхода 8 → 1.298/43, RSI выхода 10 → 1.145/43, стоп 0.7 →
// 0.927/41, RSI входа 5 → 0.597/38 — ОСТРЫЙ ПИК. Стресс, издержки и hold-out не запускались.
// Решение: не торговать.
//
// # Калибровка 2026-10-14 (30m: сигнал на 1h, подтверждение MACD на 30m)
//
// Схема hold-out (скилл drop-bounce-calibrate): перебор RSIPeriod (4, 5, 6) × RSIExitPeriod (8–12) ×
// StopDailyATR (0.5, 0.7, 1.0), 45 точек, на первых 30 из 36 месяцев (sweep_minutes30.json),
// остальное — дефолты ядра (ConfirmBars 3, безубыток 0.5 ATR, пятница до 11:00). Отчёты —
// reports/NVTK_drop_bounce/{sweep,dev,cost,full}/Minutes30/ (вне git). Тикер взят в волне из семи
// голубых фишек, чтобы выводы о стратегии опирались на широкую вселенную.
//
//   - Перебор: PF 0.362–1.093, выше 1 только 2 точки из 45, 31–43 сделки на 30 мес..
//   - Центр плато: RSI входа 4, RSI выхода 9, стоп 0.5 дневного ATR — PF 0.988 на 43 сделках, медиана
//     окрестности 0.988; соседи RSI выхода 8 → 1.093, RSI выхода 10 → 1.025, стоп 0.7 → 0.817, RSI
//     входа 5 → 0.540. ОСТРЫЙ ПИК.
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
package nvtk

import "tinvest/internal/service/trading_strategy/drop_bounce/strategy/core"

// Ticker is the instrument this package configures.
const Ticker = "NVTK"

// Params returns the NVTK literal for Minutes15.
func Params() core.Params {
	return core.Params{
		RSIPeriod:      4,
		RSIZone:        30,
		RSICap:         50,
		ZoneEntry:      1,
		EMAPeriod:      200,
		StopDailyATR:   0.5,
		DailyATRPeriod: 14,
		RSIExitPeriod:  8,
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
