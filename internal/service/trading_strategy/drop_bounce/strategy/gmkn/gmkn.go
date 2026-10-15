// Package gmkn supplies the drop_bounce Params for GMKN (Норникель) on Minutes30.
//
// # Калибровка 2026-10-14 (30m: сигнал на 1h, подтверждение MACD на 30m)
//
// Схема hold-out (скилл drop-bounce-calibrate): перебор RSIPeriod (4, 5, 6) × RSIExitPeriod (8–12) ×
// StopDailyATR (0.5, 0.7, 1.0), 45 точек, на первых 30 из 36 месяцев (sweep_minutes30.json),
// остальное — дефолты ядра (ConfirmBars 3, безубыток 0.5 ATR, пятница до 11:00). Отчёты —
// reports/GMKN_drop_bounce/{sweep,dev,cost,full}/Minutes30/ (вне git). Тикер взят в волне из семи
// голубых фишек, чтобы выводы о стратегии опирались на широкую вселенную.
//
//   - Перебор: все 45 точек PF > 1 (1.178–2.271, 29–41 сделка на 30 мес.).
//   - Центр плато: RSI входа 4, RSI выхода 11, стоп 1.0 дневного ATR — PF 1.931 на 38 сделках, медиана
//     окрестности 1.931; соседи RSI входа 5 → 2.096, стоп 0.7 → 1.979, RSI выхода 10 → 1.842, RSI
//     выхода 12 → 1.812. ПЛАТО ЕСТЬ.
//   - Стресс на окне разработки 14.10.2023 — 14.04.2026 (38 сделок, PF 1.931, +23 676 ₽): S1 да;
//     S2 да — без лучшей сделки (+3 992 ₽) PF 1.774; S3 да — без трёх лучших 1.484; S4 да —
//     полугодия в плюсе 4 из 5 (2025-04 — 2025-10: 9 сделок, PF 0.900, −1 266 ₽).
//   - Издержки (комиссия 0.1%): PF 1.726, +19 007 ₽ — выдержаны.
//   - Hold-out с 14.04.2026 (открыт один раз): 12 сделок, PF 1.043, +898 ₽, лучшая +5 474 ₽,
//     худшая −6 338 ₽ — HOLD-OUT В ПЛЮСЕ, но край на нём почти нулевой.
//
// Решение: торговать на 30m (DROP_BOUNCE_TICKERS_M30).
package gmkn

import "tinvest/internal/service/trading_strategy/drop_bounce/strategy/core"

// Ticker is the instrument this package configures.
const Ticker = "GMKN"

// Params returns the GMKN literal for Minutes30.
func Params() core.Params {
	return core.Params{
		RSIPeriod:       4,
		RSIZone:         30,
		RSICap:          50,
		ZoneEntry:       1,
		EMAPeriod:       200,
		StopDailyATR:    1,
		BreakevenATR:    0.5,
		TrailATR:        0,
		DailyATRPeriod:  14,
		RSIExitPeriod:   11,
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
