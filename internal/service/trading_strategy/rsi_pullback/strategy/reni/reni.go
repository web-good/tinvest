// Package reni supplies the ticker and calibrated rsi_pullback Params for RENI
// (Ренессанс Страхование).
//
// Тикер откалиброван 2026-08-12: литерал ниже — параметры отчёта
// reports/RENI/RENI_rsi_pullback_Minutes30_20260812_005046.md (74 сделки, in-sample PF 1.885 за
// 24 месяца). Связь с baseline разорвана осознанно, литерал прибит снимком в reni_test.go.
//
// Walk-forward того же дня (RENI_rsi_pullback_Minutes30_20260812_003547_walkforward.md) дал
// pooled OOS PF 2.060 на 78 сделках при 4 фолдах, и это лучший результат среди тикеров после
// ugld. Две оговорки, которые обязаны ехать вместе с этим числом:
//
//   - Фолд 3 (OOS 2025-08-12—2026-02-12) провален: PF 0.337 на 7 сделках. Пул вытянут фолдами
//     1, 2 и 4 — то есть режим, в котором стратегия не работает, в окне существует.
//   - Стабильным во всех четырёх фолдах walk-forward назвал TPDailyATR=0.6, а в литерале стоит
//     1.5. Это единственное поле, где принятый конфиг расходится с победителем walk-forward, и
//     его OOS-подтверждение относится к 0.6, а не к 1.5.
//
// Из-за второй оговорки решение о боевой вселенной RSI_PULLBACK_TICKERS остаётся за владельцем:
// наличие литерала снимает техническую блокировку (TestBaselineTrackingTickersStayOutOfTheDefaultUniverse
// больше не валит сборку на RENI), но не заменяет подтверждение именно тех параметров, которыми
// пойдёт торговля.
//
// Что известно об инструменте (замеры 2026-08-10, спека
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
// Пакет был заведён 2026-08-10 до калибровки — как место, куда ляжет литерал, и как носитель
// замеров выше. С появлением литерала 2026-08-12 исключение закрыто: пакет несёт собственные
// значения параметров, а не тот же baseline, что вернула бы generic-ветка
// RSIPullbackLookupOrGeneric для незарегистрированного имени.
package reni

import "tinvest/internal/service/trading_strategy/rsi_pullback/strategy/core"

// Ticker is the instrument this package configures.
const Ticker = "RENI"

// DefaultParams returns RENI's calibrated rsi_pullback parameters.
func DefaultParams() core.Params {
	return core.Params{
		RSIPeriod:       4,
		RSILower:        20,
		RSIUpper:        70,
		EMAFast:         10,
		EMASlow:         100,
		DailyATRPeriod:  14,
		UseDayATRGate:   1,
		FreshDayATR:     0,
		SpentDayATR:     0.8,
		StopDailyATR:    0.5,
		TPDailyATR:      1.5,
		UseVolume:       0,
		VolBaseDays:     14,
		VolLookbackBars: 3,
		VolMult:         1.2,
		UseRSIExit:      1,
		UseTrail:        0,
		TrailDailyATR:   0,
	}
}
