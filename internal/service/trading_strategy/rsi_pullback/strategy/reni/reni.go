// Package reni supplies the ticker and starting rsi_pullback Params for RENI
// (Ренессанс Страхование).
//
// Calibration has NOT been run for this ticker: the body returns core.DefaultParams()
// unchanged rather than copying its fields, so a change to the baseline still reaches every
// uncalibrated ticker instead of silently drifting away from it. Once -calibrate picks a
// winning combination for RENI, replace the body with an explicit literal — from that point
// the ticker must stop tracking the baseline, and the literal must be pinned by a snapshot
// test the way ugld and domrf pin theirs.
//
// Что известно об инструменте до калибровки (замеры 2026-08-10, спека
// docs/superpowers/specs/2026-08-10-reni-rsi-pullback-prep-design.md):
//
//   - Истории достаточно: 31 658 30-минутных баров (23 071 будний) за 35.9 месяца с
//     2023-08-07. Штатный протокол walk-forward §8 docs/rsi_pullback/strategy.md исполним
//     целиком — в отличие от domrf, где он неисполним физически.
//   - В окне есть настоящий нисходящий режим: пик 141.06 (2025-03-20) → минимум 63.72
//     (2026-07-20), просадка 54.8%; полугодия 2025H2 −21.9% и 2026H1 −17.0%. Для long-only
//     покупки отката это редкая возможность проверить трендовый фильтр в режиме, где он
//     обязан выключать вход.
//   - Дневной ATR(14) идёт медианой 3.36% цены — инструмент одного класса с ugld (4.28%) и
//     вдвое шире domrf (1.94%). Поэтому сетки в data/params/rsi_pullback/reni/ пересчитаны от
//     ugld/, а сужения, сделанные для domrf, в них намеренно НЕ перенесены.
//   - Оборот 91 млн ₽/день — вчетверо ниже domrf. Калибровку это не ограничивает, но станет
//     ограничением на размер позиции, если тикер дойдёт до живой вселенной.
//
// Пакет заведён до калибровки по решению владельца — как место, куда ляжет литерал, и как
// носитель замеров выше. Пока тела нет, он не несёт ни одного значения параметра: тот же
// baseline вернула бы generic-ветка RSIPullbackLookupOrGeneric для незарегистрированного
// имени. Из этого следует главное ограничение: RENI не должен попадать в боевую вселенную
// RSI_PULLBACK_TICKERS, пока literal не появился.
package reni

import "tinvest/internal/service/trading_strategy/rsi_pullback/strategy/core"

// Ticker is the instrument this package configures.
const Ticker = "RENI"

// DefaultParams returns RENI's starting rsi_pullback parameters (pre-calibration).
func DefaultParams() core.Params {
	return core.DefaultParams()
}
