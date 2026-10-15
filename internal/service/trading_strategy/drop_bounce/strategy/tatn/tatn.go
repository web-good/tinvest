// Package tatn supplies the drop_bounce Params for TATN (Татнефть) on Minutes30.
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
package tatn

import "tinvest/internal/service/trading_strategy/drop_bounce/strategy/core"

// Ticker is the instrument this package configures.
const Ticker = "TATN"

// Params returns the TATN literal for Minutes30.
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
		RSIExitPeriod:   10,
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
