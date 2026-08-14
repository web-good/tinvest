// Package wush supplies the ticker and starting rsi_pullback Params for WUSH (Whoosh).
//
// Calibration has NOT been run for this ticker: the body returns core.DefaultParams()
// unchanged rather than copying its fields, so a change to the baseline still reaches this
// ticker instead of silently drifting away from it. Once the calibration runs clear the bar
// declared in the spec, replace the body with an explicit literal — from that point the ticker
// must stop tracking the baseline, and TestRSIPullbackWUSHTracksBaseline must be replaced with
// a literal snapshot.
//
// Пакет заведён 2026-08-13 ДО калибровки — как место, куда ляжет литерал, и как носитель
// замеров инструмента. Спека: docs/superpowers/specs/2026-08-13-wush-rsi-pullback-prep-design.md,
// сетки: data/params/rsi_pullback/wush/.
//
// Что известно об инструменте (замеры по кэшу WUSH_Minutes30.json и WUSH_Day1.json,
// правилами ядра: будние дни MSK как weekdayDaily, сглаживание Уайлдера как в pkg/indicators):
//
//   - Истории достаточно: 33 823 30-минутных бара (23 901 будний) за 36.0 месяца с 2023-08-04.
//     Штатный протокол walk-forward §8 docs/rsi_pullback/strategy.md исполним целиком — как у
//     reni и fesh и в отличие от domrf.
//   - ПАДАЮТ ОБА ОКНА ПРОТОКОЛА, и это главное отличие WUSH от предшественников: обучающее окно
//     2023-08-04—2026-02-03 — падение 226.50 → 92.99 (−58.9%), holdout 2026-02-04—2026-08-03 —
//     падение 93.41 → 48.61 (−48.0%). Пик 337.48 (2024-03-25) → минимум 30.30 (2026-07-17), то
//     есть −91.0%; пять полугодий из семи отрицательные. У fesh holdout РОС на 13.1% и завышал
//     long-only механически, поэтому подтверждением служить не мог; здесь завысить результат
//     нечем — это самая честная проверка, какую ставил репозиторий. Обратная сторона: трендовый
//     фильтр в таком режиме держит EMAFast > EMASlow лишь 41–43% времени, вход закрыт большую
//     часть истории, и СЧЁТ СДЕЛОК надо читать раньше profit factor — красивый PF на шести
//     сделках здесь самый вероятный способ обмануться.
//   - Дневной ATR(14) идёт медианой 4.25% цены (среднее 4.52% — колонка скринера печатает
//     именно среднее, распределение скошено вправо хвостом до 13.41%). По ширине рядом с fesh
//     (4.42%) и ugld (4.28%), заметно выше reni (3.36%) и domrf (1.94%). Поэтому сетки в
//     data/params/rsi_pullback/wush/ пересчитаны от fesh/, а сужения domrf в них намеренно НЕ
//     перенесены.
//   - Круг издержек 0.024 дневного ATR; он лицензирует узкую строку стопа 0.3 ATR. Но тот же
//     стоп переживает целиком лишь 2.4% дней, то есть сидит внутри обычного внутридневного шума.
//   - День раскрывается БЫСТРЕЕ, чем у fesh: медиана доли ATR ко второму бару 0.33 против 0.28.
//     Отсюда единственное расхождение сеток с fesh/ — ось FreshDayATR [0, 0.3, 0.4] вместо
//     [0, 0.25, 0.35].
//   - Оборот 356 млн ₽/день при лоте 1 — вторая ликвидность из заведённых после domrf и
//     единственный тикер с лотом 1, то есть самая тонкая гранулярность позиции. На размер
//     позиции не давит.
//   - Слабое место, заявленное до прогонов: в отчёте скринера у WUSH Plateau 58% и Capped 4/24
//     против 83% и 2/24 у fesh. Рабочая зона узкая, и это прямо повышает шанс, что калибровка
//     сядет на случайный пик. Планка объявлена в спеке ДО первого прогона именно поэтому.
package wush

import "tinvest/internal/service/trading_strategy/rsi_pullback/strategy/core"

// Ticker is the instrument this package configures.
const Ticker = "WUSH"

// DefaultParams returns WUSH's starting rsi_pullback parameters (pre-calibration).
func DefaultParams() core.Params {
	return core.DefaultParams()
}
