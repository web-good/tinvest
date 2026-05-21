# Бэктест Golden X

Полная инфраструктура замера стратегии на исторических свечах. Чистый, без сетевых I/O в процессе симуляции (свечи берутся из кэша или подтягиваются разово в начале).

## Запуск

```bash
go run ./cmd/backtest/main.go [флаги]
```

CLI лежит в `cmd/backtest/main.go`. Использует тот же `Detect()`, что и live (`internal/service/trading_strategy/golden_x/detector.go`), плюс отдельный `Replay`-движок в `internal/service/trading_strategy/golden_x/backtest/`.

## Что нужно для запуска

1. **Токен Tinkoff** (для первого прогона любой акции, потом — кэш).
   - Берётся из env: `T_BANK`.
   - Загрузка через `godotenv`: `env/local.env`, затем `env/token.env` (см. `cmd/backtest/main.go:62-64`).
   - Если все нужные акции уже в кэше — токен не нужен.

2. **Доступ к gRPC `invest-public-api.tinkoff.ru:443`** (только при cache miss / `--refresh`).

3. **Свободное место под кэш** — JSON-файлы `cache/candles/<shareID>_W.json`.

## Все флаги

| Флаг | Default | Назначение |
|---|---|---|
| `--kind` | `Dividend` | Тип стратегии: `Dividend` (она же `Gold`) или `Growth`. Парсинг case-insensitive (`cmd/backtest/main.go:127`). |
| `--from` | `2022-01-01` | Начало диапазона включительно. Формат `YYYY-MM-DD`, парсится как MSK. |
| `--to` | сегодня | Конец диапазона. Если не указан — `time.Now()` в MSK. |
| `--shares` | `""` (пусто) | Список UUIDs Tinkoff через запятую. Пусто — все из соответствующего списка (`shares.Dividend()` / `shares.Growth()`). Тикеры, не из этого списка, игнорируются. |
| `--refresh` | `false` | `true` — принудительно переподтягивать свечи из API (даже если кэш есть). |
| `--cache-dir` | `cache/candles` | Где лежат JSON-кэши свечей. |
| `--out-dir` | `cache/backtests` | Куда сохранять Markdown-отчёт. |

## Примеры

### Стандартный прогон Dividend за всё время

```bash
go run ./cmd/backtest/main.go --kind Dividend
```

Запустит на дефолтном диапазоне (2022-01-01 → сегодня) для всех 11 Dividend-тикеров.

### Прогон Growth за конкретный период

```bash
go run ./cmd/backtest/main.go \
  --kind Growth \
  --from 2023-01-01 \
  --to 2024-12-31
```

### Один тикер из списка

```bash
go run ./cmd/backtest/main.go \
  --kind Growth \
  --shares "962e2a95-02a9-4171-abd7-aa198dbe643a"   # Газпром
```

### Несколько тикеров

```bash
go run ./cmd/backtest/main.go \
  --kind Growth \
  --shares "962e2a95-02a9-4171-abd7-aa198dbe643a,7de75794-a27f-4d81-a39b-492345813822"
  # Газпром + Яндекс
```

### Перезалить свечи из API

```bash
go run ./cmd/backtest/main.go --kind Dividend --refresh
```

Очистит кэш и подтянет всё заново. Нужен `T_BANK`.

## Что происходит внутри

Алгоритм `cmd/backtest/main.go:77-111` и `backtest/replay.go`:

1. **Сбор тикеров** — `selectShareList(kind, --shares)`. Если `--shares=""`, берётся весь список из `shares.Dividend()` или `shares.Growth()`.

2. **Кэш свечей** — `backtest.Cache.Get(shareID)`. Логика:
   - Прочитать `<cache-dir>/<shareID>_W.json` если есть и `--refresh=false`.
   - Иначе вызвать `grpcFetcher.Fetch()` — 300 недельных свечей через Tinkoff (≈ 5.7 лет).
   - Сохранить в кэш на следующий прогон.
   - При cache miss и `--refresh=false` — печатает `WARN` и **пропускает** эту акцию (не падает).

3. **Фильтрация закрытых недель** — `filterClosed()` (`cmd/backtest/main.go:197`). Понедельник 00:00 MSK как cutoff, так же как в live.

4. **Обрезка по диапазону** — `trimByDateRange()` отбрасывает свечи вне `[--from, --to]`.

5. **Прогрев** — `chooseStartIdx(RSILength)` (`cmd/backtest/main.go:225`) пропускает первые `max(200, RSILength + 100)` свечей. Это даёт время «прогреться» EMA200 и накопить минимум 100 значений для адаптивных перцентилей.

6. **Replay по неделям** — `backtest.Replay` (см. `backtest/replay.go`). На каждой свече:
   - Вызвать `golden_x.Detect()` с текущей историей.
   - Если позиция уже открыта — проверить выходы **строго в этом порядке**:
     1. **Стоп** — если `Low ≤ StopPrice` за неделю → `ExitReasonStop`.
     2. **Sell-tier** — если RSI пересёк P80/P90/P95 → частичный (Gold) или полный (Growth) выход.
     3. **Timeout** — если позиция держится больше 52 недель → `ExitReasonTimeout`.
   - Если позиции нет и `Detect()` вернул сигнал покупки (зелёный/жёлтый) — открыть позицию с ATR-стопом.

7. **End-of-history** — открытые позиции отмечаются `ExitReasonOpen` по последней цене (mark-to-market), чтобы попасть в статистику.

8. **Агрегация** — `backtest.AggregateStats()` считает метрики per-share и общие.

9. **Запись отчёта** — `RenderMarkdown()` пишет в `<out-dir>/YYYY-MM-DD_HHMM_{Dividend|Growth}.md`.

10. **Консоль** — короткая сводка плюс путь к файлу.

## Структура отчёта

Каждый прогон создаёт один Markdown-файл с четырьмя секциями.

### 1. Overall

Сводная таблица по всему прогону:

| Колонка | Что значит |
|---|---|
| `Count` | Всего сделок (включая `Open`) |
| `Wins` | `ReturnPct > 0` (исключая `Open`) |
| `Losses` | `ReturnPct < 0` (исключая `Open`) |
| `Open` | Открытые позиции на конец прогона (mark-to-market) |
| `WinRate` | `Wins / (Wins + Losses)` × 100% |
| `AvgReturn%` | Средний `ReturnPct` на сделку (без взвешивания по Units) |
| `Median%` | Медиана `ReturnPct` |
| `Cumulative%` | Сумма `ReturnPct × Units` — реальный накопленный результат с учётом частичных выходов |
| `MaxDD%` | Максимальная просадка от пика по equity-кривой (`peak − equity`) |
| `AvgWeeks` | Среднее число недель в позиции |

### 2. Exit reasons

Распределение причин выхода с средним результатом:

```
| Reason     | Count | AvgReturn% |
| sell_p80   | 7     | 18.2       |
| sell_p90   | 5     | 25.7       |
| sell_p95   | 3     | 31.4       |
| stop       | 9     | -8.3       |
| timeout    | 2     | 4.1        |
| open       | 1     | 12.0       |
```

| Reason | Что значит |
|---|---|
| `sell_p80` / `sell_p90` / `sell_p95` | Выход по соответствующему перцентилю RSI |
| `stop` | Цена пересекла ATR-стоп вниз |
| `timeout` | 52 недели в позиции без выхода |
| `open` | Позиция всё ещё открыта на конец диапазона |

### 3. Per share

Метрики отдельно по каждой акции (UUID, Count, WinRate, Cumulative%, MaxDD%).

### 4. Trades (chronological)

Хронологический список всех сделок:

| Колонка | Описание |
|---|---|
| `Share` | UUID акции |
| `Entry` / `Exit` | Даты входа и выхода |
| `EntryPx` / `ExitPx` | Цена входа и выхода |
| `Units` | Доля позиции: `1.0` для полного выхода, `~0.333` для каждого частичного (Gold) |
| `Reason` | Из таблицы Exit reasons |
| `Return%` | `(ExitPrice − EntryPrice) / EntryPrice × 100` |
| `Weeks` | Сколько недель держалась позиция |

## Файлы пакета `backtest`

| Файл | Назначение |
|---|---|
| `replay.go` | Главный движок симуляции: итерация по свечам, вызовы `Detect`, управление позицией |
| `position.go` | `Position` + `OpenPosition` + `CloseAll` + `EvaluateSellExits`; для Gold — три частичных, для Growth — один полный |
| `cache.go` | Дисковый JSON-кэш недельных свечей |
| `report.go` | `Stats`, `Report`, `AggregateStats`, `RenderMarkdown` |
| `*_test.go` | Unit-тесты на каждый из выше |

## Где менять параметры стратегии

CLI **не** даёт прямо подкрутить `Settings`. Сейчас всё через код:

### Вариант 1. Изменить дефолты глобально

`internal/service/trading_strategy/golden_x/settings.go`:

```go
func DefaultSettings() dto.Settings {
    return dto.Settings{
        BuyGreen:  3,        // было 5 — теперь зелёная зона уже
        SellOrange: 85,      // было 90 — продаём раньше
        // ... остальные
    }
}
```

После этого живой код **тоже** будет работать по-новому — будьте внимательны.

### Вариант 2. Альтернативный набор для бэктеста

Добавить рядом с `DefaultSettings()`:

```go
func AggressiveSettings() dto.Settings {
    s := DefaultSettings()
    s.BuyYellow = 25
    return s
}
```

И в `cmd/backtest/main.go:83` заменить:

```go
settings := golden_x.AggressiveSettings()
```

Live при этом не затрагивается.

### Вариант 3. Поменять период RSI у тикера

`shares/shares.go` — поправить `RSILength` у нужной строки. Затронет и live, и бэктест.

### Вариант 4. Добавить новый тикер

`shares/shares.go`, в `Dividend()` или `Growth()`:

```go
c.Add(collection.Instrument{
    ID:        "uuid-from-tinkoff",
    RSILength: 9,
    Name:      "Название",
})
```

UUID берётся из справочника Tinkoff Invest API.

## Запуск через интерфейс / Makefile?

На момент написания документации — нет. Запуск только напрямую через `go run`. Если нужен Makefile-таргет, добавить в `Makefile` строки вида:

```makefile
backtest-dividend:
	go run ./cmd/backtest/main.go --kind Dividend

backtest-growth:
	go run ./cmd/backtest/main.go --kind Growth
```

## Типичные проблемы

**`T_BANK env var required (no cache and not --refresh)`** — нет кэша на одной из акций. Либо запустить с `--refresh`, либо положить токен в `env/token.env` и повторить.

**`WARN <имя>: ... (skipping; rerun with --refresh to retry)`** — отдельная акция не нашлась в кэше; прогон продолжается без неё. Если важна — `--refresh`.

**`no shares selected`** — флаг `--shares` указал на UUID, которых нет в списке этого `--kind`. Проверить, что UUID есть в нужной функции `shares/shares.go`.

**Отчёт пустой / нет сделок** — диапазон `--from .. --to` слишком короткий или весь попадает в warmup. `chooseStartIdx` пропускает ~200 свечей в начале (≈4 года). При диапазоне меньше — стратегия не запустится.
