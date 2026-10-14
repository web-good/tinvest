// Package ivat supplies the drop_bounce Params for IVAT (ИВА Технолоджис) on Minutes30.
//
// # Правила 2026-10-14: сигнал на 1h, подтверждение MACD на 30m
//
// Ядро переведено на 30m: часовой сигнал RSI над EMA собирается из 30-минутных свечей, вход — по
// первой зелёной гистограмме MACD(12,26,9) на 30m за ConfirmBars баров, только будни (в пятницу покупка до 11:00); выход — стоп
// в дневных ATR или часовой RSI(RSIExitPeriod) крест 70 вверх. Литерал сброшен на дефолты ядра до
// калибровки на этих правилах; все замеры ниже сняты на прежних правилах (1h) и к ним не относятся.
//
// # Правила 2026-10-14
//
// Ядро переписано (окна покупки 07–11 и 17–24 MSK, выход по RSI(RSIExitPeriod), без гейта хода
// дня, MACD-выхода и пятничных правил). Все замеры ниже сняты на прежних правилах и к новым не
// относятся; литерал сброшен на дефолты ядра до калибровки на новых правилах (скилл
// drop-bounce-calibrate, перебор RSIPeriod × RSIExitPeriod × StopDailyATR).
//
// Откалиброван 2026-10-13 на правилах 1h RSI-зоны с выходом RSI+MACD (скилл
// drop-bounce-calibrate), дважды: без выхода по времени (сетка v2) и с выходом STALE 10 баров /
// 0.3 ATR в дефолтах ядра (сетка v4). Гриды и итоги каждого прогона — в _comment файлов
// data/params/drop_bounce/ivat/, отчёты — reports/IVAT_drop_bounce/{baseline,wf,point}/Hour1/
// (вне git; прогоны без STALE — в *_nostale/).
//
// # Протокол
//
// 36 месяцев, walk-forward 12/6 (4 фолда), -min-trades 10, -metric profit_factor. Сетка v4
// (calstop_hour1.json): вход RSI на дефолтах 6/30/50, filter EMAPeriod (100, 200) ×
// DaySpentATR (0.5, 0.7, 1.0) → risk StopDailyATR (0.7, 1.0, 1.5) → exit RSIExit (65, 70, 75),
// keepTop 3, стоп в каждой точке, STALE 10/0.3 прибит.
//
// # Замеры
//
//   - Без STALE (сетка v2, calstop_hour1_v2.json): baseline 0.806/19, подбор 1.166/28, литерал
//     был консенсусом (EMA 200, день 0.97, стоп 0.88, выход 66), планка не взята (A, B нет).
//   - Baseline v4 (дефолты ядра со STALE): pooled OOS PF 1.226 на 21 сделке, compounded +4.07%,
//     фолды 0.000(3) / 1.755(6) / 0.048(5) / 3.761(7).
//   - Сетка v4: pooled OOS PF 1.140 на 29 сделках, compounded +3.30%, фолды 0.392(6) /
//     1.327(9) / 0.239(7) / 3.761(7). Выборы фолдов: EMA 200 во всех, день 1/1/1/0.7, стоп
//     0.7/1/0.7/1.5, выход 75/65/65/65.
//   - report.py pick: ЛИТЕРАЛ: ДЕФОЛТЫ — подбор не обошёл baseline.
//   - Точка (дефолты, in-sample, не судья): за 36 мес 21 сделка, PF 1.205, +4 731 ₽, win 33%.
//
// # Решение
//
// Не торговать: планка не взята — A нет (21 < 30 сделок), B нет (без лучшего фолда −8.07%: весь
// плюс — последний фолд 3.761 на 7 сделках), C да (+6.52%), D да (фолдов с ≥ 10 сделками нет).
// Сигналов мало — около 7 сделок в год. Плато и издержки не гонялись: они нужны только при
// взятой планке. Тикер не входит в DROP_BOUNCE_TICKERS_H1.
package ivat

import "tinvest/internal/service/trading_strategy/drop_bounce/strategy/core"

// Ticker is the instrument this package configures.
const Ticker = "IVAT"

// Params returns the IVAT literal for Minutes30.
func Params() core.Params {
	return core.Params{
		RSIPeriod:      6,
		RSIZone:        30,
		RSICap:         50,
		ZoneEntry:      1,
		EMAPeriod:      200,
		StopDailyATR:   1,
		BreakevenATR:   0.5,
		TrailATR:       0,
		DailyATRPeriod: 14,
		RSIExitPeriod:  10,
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
	}
}
