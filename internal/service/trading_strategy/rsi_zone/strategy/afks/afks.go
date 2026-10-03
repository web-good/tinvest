// Package afks supplies the ticker and calibrated rsi_zone Params for AFKS (АФК «Система»).
//
// Откалиброван 2026-09-28. Гриды и итоги каждого прогона — в _comment файлов
// data/params/rsi_zone/afks/.
//
// # Протокол и результат
//
// Штатный walk-forward -months 36 -train-months 12 -test-months 6 -min-trades 20 -metric
// profit_factor, 4 фолда; темы свипали оси поверх core.DefaultParams(). Pooled OOS PF: trend
// 1.557/133 (фолды 1.065 / 2.367 / 2.709 / 1.540, EMAPeriod 600/50/50/50), phased 1.547/110
// (формально выше планки, но гуляли все пять полей), trend→risk 1.437/135, entry 1.112,
// risk 1.079. Дефолты ядра за 36 месяцев: PF 0.996 на 267 сделках.
//
// Планку 1.5 честно взяла одна тема — trend, со всеми четырьмя фолдами выше единицы.
//
// # Литерал
//
// Дефолты ядра, изменено одно поле: EMAPeriod 200 → 50.
//
//   - StopDailyATR 1.0 — пик по обоим срезам: на полной истории 0.5 → 1.155, 0.7 → 1.453,
//     1.0 → 1.503, 1.5 → 1.326, 2.0 → 1.267. Капкана широкого стопа нет: 11 SL-выходов из 154.
//
// Замеры литерала. Прогон по окнам фолдов сеткой из одной точки: pooled 2.328, фолды 6.583 /
// 2.367 / 2.709 / 1.540; при удвоенной комиссии 0.001 — 1.942. Полные 36 месяцев: 154 сделки,
// PF 1.503, +34.3%, просадка 19.2%; при удвоенной комиссии PF 1.226.
//
// Риски: весь результат сделан с 2025 года — полугодия 2023H2–2024H2 убыточны (PF 0.05 / 0.79 /
// 0.86), 2026H2 в нуле. Худшая сделка — стоп −6.67% 2026-07-06: дневной ATR у AFKS широкий.
//
// # Выход PROFIT (тема profit, 2026-10-02/03)
//
// Выключен: ProfitExitBars 0, ProfitExitPct 0 — записаны числами, чтобы решение было видно в
// литерале. Тема profit (cal_profit.json, оси N 0–12 × 0–2%, литерал прибит) — pooled 2.591/114
// против 2.324/111 без выхода, но доходность ниже: +55.0% против +65.0%. Голоса фолдов: N 2 / 0 /
// 3 / 3, порог 1 / 1 / 0.3 / 0.3 — большинства нет. Общий зонд N 3 · 0.5%: 2.164/116, +45.2%.
// PF темы растёт оттого, что выход режет прибыльные сделки раньше RSI-выхода, а денег меньше.
package afks

import "tinvest/internal/service/trading_strategy/rsi_zone/strategy/core"

// Ticker is the instrument this package configures.
const Ticker = "AFKS"

// DefaultParams returns AFKS's calibrated rsi_zone parameters.
func DefaultParams() core.Params {
	return core.Params{
		RSIPeriod:      4,
		RSILower:       25,
		RSIUpper:       75,
		EMAPeriod:      50,
		DailyATRPeriod: 14,
		StopDailyATR:   1.0,
		ProfitExitBars: 0,
		ProfitExitPct:  0,
	}
}
