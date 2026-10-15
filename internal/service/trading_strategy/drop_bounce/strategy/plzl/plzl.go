// Package plzl supplies the drop_bounce Params for PLZL (Полюс) on Minutes30.
//
// # Перекалибровка 2026-10-14 с -risk-pct 1
//
// Та же схема (перебор 45 точек на 30 из 36 мес., центр плато, стресс, hold-out), но размер позиции —
// 1% капитала на стоп (-risk-pct 1), PrevCloseGate 0. Литерал — новый центр (point_minutes30.json);
// разделы ниже — прежние замеры при фиксированной доле.
//
// Центр плато: RSI входа 4, RSI выхода 12, стоп 0.5 дневного ATR — PF 1.238 на 53 сделках (лучшая
// точка перебора PF 1.238); соседи: RSI выхода 11 → 1.183/54, стоп 0.7 → 0.966/52, RSI входа 5 →
// 0.624/45 — ОСТРЫЙ ПИК. Стресс, издержки и hold-out не запускались. Решение: не торговать.
//
// # Калибровка 2026-10-14 (30m: сигнал на 1h, подтверждение MACD на 30m)
//
// Схема hold-out (скилл drop-bounce-calibrate): перебор RSIPeriod (4, 5, 6) × RSIExitPeriod (8–12) ×
// StopDailyATR (0.5, 0.7, 1.0), 45 точек, на первых 30 из 36 месяцев (sweep_minutes30.json),
// остальное — дефолты ядра (ConfirmBars 3, безубыток 0.5 ATR, пятница до 11:00). Отчёты —
// reports/PLZL_drop_bounce/{sweep,dev,cost,full}/Minutes30/ (вне git). Тикер взят в волне из семи
// голубых фишек, чтобы выводы о стратегии опирались на широкую вселенную.
//
//   - Перебор: PF 0.583–1.174, выше 1 только 4 точки из 45, 39–55 сделок на 30 мес..
//   - Центр плато: RSI входа 4, RSI выхода 11, стоп 0.5 дневного ATR — PF 1.138 на 54 сделках, медиана
//     окрестности 1.042; соседи RSI выхода 12 → 1.174, RSI выхода 10 → 1.042, стоп 0.7 → 0.858, RSI
//     входа 5 → 0.643. ОСТРЫЙ ПИК.
//   - Стресс, издержки и hold-out не гонялись: плато не пройдено.
//
// Решение: не торговать. Литерал — центр перебора (point_minutes30.json).
package plzl

import "tinvest/internal/service/trading_strategy/drop_bounce/strategy/core"

// Ticker is the instrument this package configures.
const Ticker = "PLZL"

// Params returns the PLZL literal for Minutes30.
func Params() core.Params {
	return core.Params{
		RSIPeriod:      4,
		RSIZone:        30,
		RSICap:         50,
		ZoneEntry:      1,
		EMAPeriod:      200,
		StopDailyATR:   0.5,
		BreakevenATR:   0.5,
		TrailATR:       0,
		DailyATRPeriod: 14,
		RSIExitPeriod:  12,
		RSIExit:        70,
		MACDFast:       12,
		MACDSlow:       26,
		MACDSignal:     9,
		ConfirmBars:    3,
		FridayTo:       11,
		StaleBars:      0,
		StaleATR:       0.3,
		PrevCloseGate:  0,
		PrevCloseATR:   0,
		RSIFloor:       0,
		SkipMonday:     0,
		EMAGapATR:      0,
		MACDGreenOnly:  0,
		BreakevenPct:   0.1,
	}
}
