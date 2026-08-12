// Package fesh supplies the ticker and starting rsi_pullback Params for FESH (ДВМП).
//
// Calibration has NOT been run for this ticker: the body returns core.DefaultParams()
// unchanged rather than copying its fields, so a change to the baseline still reaches this
// ticker instead of silently drifting away from it. Once -calibrate picks a winning
// combination for FESH, replace the body with an explicit literal — from that point the
// ticker must stop tracking the baseline.
//
// Что известно об инструменте (замеры 2026-08-12, спека
// docs/superpowers/specs/2026-08-12-fesh-rsi-pullback-prep-design.md):
//
//   - Истории достаточно: 35 062 30-минутных бара (25 321 будний) за 36.0 месяца с
//     2023-08-04. Штатный протокол walk-forward §8 docs/rsi_pullback/strategy.md исполним
//     целиком — как у reni и в отличие от domrf.
//   - Режимы разложены ЗЕРКАЛЬНО протоколу: обучающее окно 2023-08-04—2026-02-03 это падение
//     112.40 → 55.06 (−51.0%), а holdout 2026-02-04—2026-08-04 — рост 55.00 → 62.23 (+13.1%).
//     Для long-only покупки отката это значит, что in-sample PF занижен режимом, а holdout PF
//     завышен механически: подтверждением edge holdout здесь служить не может, его даёт только
//     rolling walk-forward, чьи фолды покрывают оба режима.
//   - Дневной ATR(14) идёт медианой 4.42% цены — самый широкий инструмент из заведённых
//     (ugld 4.28%, reni 3.36%, domrf 1.94%). Поэтому сетки в data/params/rsi_pullback/fesh/
//     пересчитаны от ugld/, а сужения, сделанные для domrf, в них намеренно НЕ перенесены.
//   - Круг издержек 0.023 дневного ATR — самый дешёвый из четырёх тикеров; он лицензирует
//     узкую строку стопа 0.3 ATR. Но тот же стоп переживает целиком лишь 1.0% дней, то есть
//     сидит внутри обычного внутридневного шума.
//   - Оборот 287 млн ₽/день при лоте 10 — втрое выше reni. На размер позиции не давит.
//
// Пакет заведён 2026-08-12 ДО калибровки — как место, куда ляжет литерал, и как носитель
// замеров выше. Попадание такого тикера в боевую вселенную ловит
// TestBaselineTrackingTickersStayOutOfTheDefaultUniverse в live/registry_test.go.
package fesh

import "tinvest/internal/service/trading_strategy/rsi_pullback/strategy/core"

// Ticker is the instrument this package configures.
const Ticker = "FESH"

// DefaultParams returns FESH's starting rsi_pullback parameters (pre-calibration).
func DefaultParams() core.Params {
	return core.DefaultParams()
}
