// Package nvtk supplies the drop_bounce Params for NVTK (Новатэк) on Minutes30.
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
package nvtk

import "tinvest/internal/service/trading_strategy/drop_bounce/strategy/core"

// Ticker is the instrument this package configures.
const Ticker = "NVTK"

// Params returns the NVTK literal for Minutes30.
func Params() core.Params {
	return core.Params{
		RSIPeriod:       4,
		RSIZone:         30,
		RSICap:          50,
		ZoneEntry:       1,
		EMAPeriod:       200,
		StopDailyATR:    0.5,
		BreakevenATR:    0.5,
		TrailATR:        0,
		DailyATRPeriod:  14,
		RSIExitPeriod:   9,
		RSIExit:         70,
		MACDFast:        12,
		MACDSlow:        26,
		MACDSignal:      9,
		ConfirmBars:     3,
		FridayTo:        11,
		StaleBars:       0,
		StaleATR:        0.3,
		PrevCloseGate:   0,
		PrevCloseATR:    0,
		RSIFloor:        0,
		SkipMonday:      0,
		EMAGapATR:       0,
		MACDGreenOnly:   0,
		BreakevenPct:    0.1,
		MACDConfirm:     1,
		ConfirmMoveBars: 0,
		ConfirmMoveATR:  0.1,
	}
}
