# Golden X — стратегия торговли

Документация по реализации в `internal/service/trading_strategy/golden_x/`.

## Что такое Golden X

Golden X — стратегия покупки акций в локальных RSI-минимумах и продажи в локальных RSI-максимумах на **недельном** таймфрейме. От «классического RSI 30/70» отличается тремя вещами:

1. **Адаптивные пороги.** Зоны перепроданности/перекупленности считаются как перцентили (P5, P15, P80, P90, P95) на исторических значениях RSI этой же акции (окно 100–200 недель, включая текущую формирующуюся). У каждой бумаги — свои пороги.
2. **Подтверждения сигнала.** Покупочный tier (🟢 / 🟡) дополняется опциональным фильтром тренда (EMA200), бычьей дивергенцией RSI (фрактал Williams k=2) и подтверждением объёма (>1.5 × SMA20).
3. **Стоп-лосс на ATR.** Для каждой покупки сразу считается уровень стопа: `Close − K × ATR14`, где K зависит от типа стратегии.

## Два режима

| Режим | Что используется | Список бумаг |
|---|---|---|
| 🥇 **Dividend (Gold)** — дивидендные «голубые фишки» | Полные тиры продажи (P80 🟠, P90 🔴, P95 🚨), частичные выходы по 1/3, ATR×2.0, без фильтра тренда | `shares.Dividend()` — 11 тикеров |
| 🥈 **Growth** — растущие акции | Единственный выход на P90 🔴 (полный), ATR×1.5, EMA200-фильтр тренда ✅/🚫 | `shares.Growth()` — 6 тикеров |

## Точки входа

- Чистая функция расчёта сигнала: `Detect()` — `internal/service/trading_strategy/golden_x/detector.go:18`.
- Шедулер для prod: `internal/service/trading_strategy/golden_x/scheduler/trade.go`.
- Бэктест: `cmd/backtest/main.go`.
- Списки тикеров: `internal/service/trading_strategy/golden_x/shares/shares.go`.
- Дефолтные параметры: `internal/service/trading_strategy/golden_x/settings.go` (`DefaultSettings()`).

## Навигация

| Документ | О чём |
|---|---|
| [strategy.md](strategy.md) | Алгоритм пошагово: жизненный цикл сигнала от свечей до Telegram-алерта; разница Dividend vs Growth; дедупликация |
| [settings.md](settings.md) | Все 14 параметров `Settings` с дефолтами; per-share `RSILength`; рецепты тюнинга |
| [alerts.md](alerts.md) | Расшифровка 12 emoji; шаблон сообщения; формат полей; два типа уведомлений |
| [backtest.md](backtest.md) | Запуск `cmd/backtest`, флаги, метрики отчёта, как менять параметры |
| [indicators.md](indicators.md) | Математика индикаторов: RSI Wilder, EMA, percentile R-7, фрактал Williams, ATR |

## Быстрый старт

Прочитать алерт в Telegram → [alerts.md](alerts.md#каталог-иконок).

Запустить бэктест → [backtest.md](backtest.md#запуск).

Поменять чувствительность сигналов → [settings.md](settings.md#рецепты-тюнинга).

Разобраться в формулах → [indicators.md](indicators.md).
