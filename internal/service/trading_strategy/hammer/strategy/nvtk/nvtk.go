// Package nvtk supplies the hammer Params for NVTK (Новатэк) on Hour1.
//
// # 2026-10-23 — калибровка скользящим walk-forward, планка не взята
//
// Сетка data/params/hammer/nvtk/wf_rolling.json (три фазы: вход → выходы → контекст, 5 лучших в
// следующую фазу), Hour1, 36 мес, -risk-pct 1, подбор 3 мес / проверка 1 мес, -min-trades 5,
// метрика profit_factor; MACD 11/15/9 и DailyATRPeriod 14 по просьбе владельца не калибровались.
// Пул OOS 33 фолдов: PF 0.444 на 105 сделках, win rate 24.8%, compounded −23.3% — края нет.
// Параметры по фолдам гуляли; литерал — значение, которое фолды выбирали чаще всего (весовой
// консенсус для выключателей RSIHold/PDLZoneATR/ConfirmBars даёт бессмысленные 1.85/0.2/0.02):
// RSI(6) < 50 (27 из 33), без RSIHold (28), EMA 100 (14), без объёма (29), без подтверждения (29),
// без зоны у PDL (18), стоп под молотом 0.05 ATR (13), цель 1.5R (17). На 36 мес PF 0.552 на 85
// сделках (−15.2%), последние 12 мес 0.559 на 34 (−6.3%); литерал владельца до калибровки
// (RSI 6/50, RSIHold 70, EMA 200, объём 0.5, подтверждение 3, цель 1.5R) — 0.366 на 28 и 0.361 на 13.
// Лучшая точка статического прогона той же сетки на 36 мес (in-sample, в литерал не идёт):
// RSIHold 60, EMA 200, подтверждение 1, PDLZoneATR 0.5, стоп 0–0.2 ATR — PF 1.427 на 27 сделках.
package nvtk

import "tinvest/internal/service/trading_strategy/hammer/strategy/core"

// Params returns the NVTK literal.
func Params() core.Params {
	return core.Params{
		RSIPeriod:      6,
		RSIZone:        50,
		MACDFast:       11,
		MACDSlow:       15,
		MACDSignal:     9,
		RSIHold:        0,
		TrendEMA:       100,
		VolumeMult:     0,
		VolBars:        20,
		ConfirmBars:    0,
		PDLZoneATR:     0,
		StopBufferATR:  0.05,
		TargetR:        1.5,
		DailyATRPeriod: 14,
	}
}
