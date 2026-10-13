// Package enpg supplies the drop_bounce Params for ENPG (ЭН+ ГРУП) on Hour1.
//
// Откалиброван 2026-10-13 на правилах 1h RSI-зоны с выходом RSI+MACD (скилл
// drop-bounce-calibrate). Гриды и итоги каждого прогона — в _comment файлов
// data/params/drop_bounce/enpg/, отчёты — reports/ENPG_drop_bounce/{baseline,wf,point}/Hour1/
// (вне git).
//
// # Протокол
//
// 36 месяцев, walk-forward 12/6 (4 фолда), -min-trades 10, -metric profit_factor. Сетка v2
// (calstop_hour1.json): вход RSI на дефолтах 6/30/50, filter EMAPeriod (100, 200) ×
// DaySpentATR (0.5, 0.7, 1.0) → risk StopDailyATR (0.7, 1.0, 1.5) → exit RSIExit (65, 70, 75),
// keepTop 3, стоп в каждой точке.
//
// # Замеры
//
//   - Baseline (дефолты ядра): pooled OOS PF 1.055 на 40 сделках, compounded +2.30%, фолды
//     1.607(14) / 1.047(13) / 0.127(4) / 0.987(9).
//   - Сетка v2: pooled OOS PF 1.194 на 37 сделках, compounded +7.43%, фолды 2.633(15) /
//     0.600(9) / 0.959(2) / 1.009(11). Выборы фолдов: EMA 200 во всех, день 0.7/0.5/0.5/0.7,
//     стоп 1/1.5/1/0.7, выход 65/75/65/65.
//   - report.py pick: ЛИТЕРАЛ: КОНСЕНСУС — v2 обошёл baseline и по PF, и по доходности.
//   - Точка (консенсус v2: EMA 200, день 0.6 дневного ATR, стоп 1.03 дневного ATR, выход RSI 66;
//     in-sample, не судья): за 36 мес 46 сделок, PF 1.267, +11 708 ₽, win 63%.
//
// # Решение
//
// Не торговать: планка не взята — A нет (1.194 < 1.2 на 37 сделках), B нет (без лучшего фолда
// −6.83%: весь плюс пула — первый фолд 2.633), C да (+0.09%, последние 12 месяцев в ноль),
// D да (фолды с ≥ 10 сделками: 2.633, 1.009). Плато и издержки не гонялись: они нужны только при
// взятой планке. Тикер не входит в DROP_BOUNCE_TICKERS_H1.
package enpg

import "tinvest/internal/service/trading_strategy/drop_bounce/strategy/core"

// Ticker is the instrument this package configures.
const Ticker = "ENPG"

// Params returns the ENPG literal for Hour1.
func Params() core.Params {
	return core.Params{
		RSIPeriod:      6,
		RSIZone:        30,
		RSICap:         50,
		EMAPeriod:      200,
		DaySpentATR:    0.6,
		StopDailyATR:   1.03,
		DailyATRPeriod: 14,
		RSIExit:        66,
		StartHour:      7,
		MACDFast:       12,
		MACDSlow:       26,
		MACDSignal:     9,
	}
}
