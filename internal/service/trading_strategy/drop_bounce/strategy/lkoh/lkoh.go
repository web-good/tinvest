// Package lkoh supplies the drop_bounce Params for LKOH (Лукойл) on Minutes30.
//
// # Перекалибровка 2026-10-14 с -risk-pct 1
//
// Та же схема (перебор 45 точек на 30 из 36 мес., центр плато, стресс, hold-out), но размер позиции —
// 1% капитала на стоп (-risk-pct 1), PrevCloseGate 0. Литерал — новый центр (point_minutes30.json);
// разделы ниже — прежние замеры при фиксированной доле.
//
// Центр плато: RSI входа 5, RSI выхода 8, стоп 1 дневного ATR — PF 0.870 на 36 сделках (лучшая точка
// перебора PF 0.9211); соседи: стоп 0.7 → 0.921/39, RSI выхода 9 → 0.908/36, RSI входа 4 →
// 0.739/47, RSI входа 6 → 0.674/30 — ОСТРЫЙ ПИК. Стресс, издержки и hold-out не запускались.
// Решение: не торговать.
//
// # Калибровка 2026-10-14 (30m: сигнал на 1h, подтверждение MACD на 30m)
//
// Схема hold-out (скилл drop-bounce-calibrate): перебор RSIPeriod (4, 5, 6) × RSIExitPeriod (8–12) ×
// StopDailyATR (0.5, 0.7, 1.0), 45 точек, на первых 30 из 36 месяцев (sweep_minutes30.json),
// остальное — дефолты ядра (ConfirmBars 3, безубыток 0.5 ATR, пятница до 11:00). Отчёты —
// reports/LKOH_drop_bounce/{sweep,dev,cost,full}/Minutes30/ (вне git). Тикер взят в волне из семи
// голубых фишек, чтобы выводы о стратегии опирались на широкую вселенную.
//
//   - Перебор: все 45 точек PF < 1 (0.375–0.823, 29–51 сделка на 30 мес.).
//   - Центр плато: RSI входа 5, RSI выхода 8, стоп 0.7 дневного ATR — PF 0.823 на 39 сделках, медиана
//     окрестности 0.752; соседи RSI выхода 9 → 0.815, RSI входа 4 → 0.754, стоп 1.0 → 0.749, стоп 0.5
//     → 0.713, RSI входа 6 → 0.590. ОСТРЫЙ ПИК.
//   - Стресс, издержки и hold-out не гонялись: плато не пройдено.
//
// Решение: не торговать. Литерал — центр перебора (point_minutes30.json).
package lkoh

import "tinvest/internal/service/trading_strategy/drop_bounce/strategy/core"

// Ticker is the instrument this package configures.
const Ticker = "LKOH"

// Params returns the LKOH literal for Minutes30.
func Params() core.Params {
	return core.Params{
		RSIPeriod:      5,
		RSIZone:        30,
		RSICap:         50,
		ZoneEntry:      1,
		EMAPeriod:      200,
		StopDailyATR:   1,
		BreakevenATR:   0.5,
		TrailATR:       0,
		DailyATRPeriod: 14,
		RSIExitPeriod:  8,
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
	}
}
