# Добавление тикера AFKS и извлечение общего adaptive-ядра

- Дата: 2026-06-06
- Статус: утверждён к планированию
- Область: `internal/service/trading_strategy/scalping`, `internal/service/backtest`, `data/`

## Контекст и проблема

Скальпинговая стратегия переработана; параметры под RUAL (`data/params/rusal/*`)
подбирались ранее. Свежие бэктесты за 24 мес (2024-06 → 2026-06) дают слабые
метрики:

| Набор | Сделок | Win rate | PF | Net | CAGR | Exposure | Max DD |
|---|---|---|---|---|---|---|---|
| `trend` | 64 | 35.9% | 1.381 | +10.8% | 5.26% | 4.93% | 9.7% |
| `robust` | 30 | 43.3% | 1.256 | +2.1% | 1.05% | 2.15% | 8.1% |
| `range` | 12 | 41.7% | 0.611 | −2.3% | −1.14% | 0.47% | 4.2% |
| `grid best` | 26 | 46.2% | 1.629 | +6.4% | 3.13% | 1.50% | 5.7% |

Диагноз: метрики слабые в первую очередь из-за **очень низкого exposure
(0.5–5%)** — капитал почти всё время вне рынка. Открытый вопрос: это дефект
стратегии или плохая пригодность RUAL (боковик/нисходящий тренд)?

**Цель этой итерации** — добавить AFK Система (AFKS) как полноценно
поддерживаемый тикер и провести честное сравнение RUAL vs AFKS, чтобы ответить
«дело в стратегии или в бумаге». AFKS — контраст к RUAL: выше волатильность,
более выраженные тренды.

## Решение (Вариант A): извлечь общее adaptive-ядро

Решающая логика в пакете `rusal` полностью обобщённая (адаптивные ADX-режимы:
mean-reversion в range, momentum в trend). RUAL-специфичны только
`const ticker = "RUAL"` и `DefaultParams()`. Поэтому:

- Выносим обобщённое ядро в новый пакет `strategy/adaptive`.
- Пакеты `rusal` и новый `afks` становятся тонкими конфигами: только `Ticker` +
  `DefaultParams()`.

Это behaviour-preserving рефактор: существующие тесты RUAL стерегут поведение.

### Целевая структура пакетов

```
internal/service/trading_strategy/scalping/strategy/
  strategy.go            # интерфейс Strategy, MarketData, Position (без изменений)
  adaptive/
    adaptive.go          # Params, Strategy, NewWithParams(ticker, p), Decide, decide,
                         #   regimeOf, emaTouched, recentHigh, Lookback
    adaptive_test.go     # перенос rusal_test.go (decision-ядро, ticker-агностично)
  rusal/
    rusal.go             # const Ticker="RUAL"; DefaultParams() adaptive.Params;
                         #   New() = adaptive.NewWithParams(Ticker, DefaultParams())
  afks/
    afks.go              # const Ticker="AFKS"; DefaultParams() adaptive.Params;
                         #   New() = adaptive.NewWithParams(Ticker, DefaultParams())
```

### Контракт `adaptive`

```go
// adaptive.Params — все 16 настраиваемых полей (как сейчас в rusal.Params).
type Params struct { /* EMAPeriod ... TrendFilterPeriod, без изменений */ }

// NewWithParams строит стратегию для конкретного тикера и параметров.
func NewWithParams(ticker string, p Params) *Strategy

func (s *Strategy) Ticker() string      // возвращает переданный ticker
func (s *Strategy) Lookback() int        // как сейчас
func (s *Strategy) Decide(md strategy.MarketData) model.Signal
```

- `Strategy` хранит `ticker string` и `p Params`.
- `Decide` проставляет `sig.Ticker = s.ticker` (вместо хардкода `"RUAL"`).
- Вся остальная логика (`decide`, `regimeOf`, `emaTouched`, `recentHigh`,
  индикаторы) переносится дословно.

### Контракт per-share конфигов

```go
// rusal
package rusal
const Ticker = "RUAL"
func DefaultParams() adaptive.Params { /* текущие значения rusal.DefaultParams */ }
func New() *adaptive.Strategy { return adaptive.NewWithParams(Ticker, DefaultParams()) }

// afks
package afks
const Ticker = "AFKS"
func DefaultParams() adaptive.Params { /* базовые НЕкалиброванные значения */ }
func New() *adaptive.Strategy { return adaptive.NewWithParams(Ticker, DefaultParams()) }
```

`DefaultParams` AFKS стартует с обобщённого baseline (как нынешние
«NOT-yet-calibrated» значения rusal), **но без** унаследованного
RUAL-калиброванного `TrendFilterPeriod=100` — для AFKS этот фильтр подбирается
заново через grid. Базовое значение `TrendFilterPeriod` для AFKS: `0` (фильтр
выключен по умолчанию; калибровка решит, включать ли).

## Регистрация AFKS

### Backtest-реестр

`internal/service/backtest/registry.go` — добавить запись `"AFKS"`. Биндинг
параметризуется тикером через `adaptive`:

```go
"AFKS": {
    DefaultParams: func() any { return afks.DefaultParams() },
    Build:         func(p any) strategy.Strategy { return adaptive.NewWithParams(afks.Ticker, p.(adaptive.Params)) },
    ParseParams:   /* как у RUAL: старт от afks.DefaultParams(), json.Unmarshal поверх */,
}
```

`rusal`-биндинг аналогично переключается на `adaptive.Params` /
`adaptive.NewWithParams(rusal.Ticker, ...)`.

### Живой раннер

`scalping/registry.go` — `defaultStrategies()` возвращает обе:

```go
return []strategy.Strategy{ rusal.New(), afks.New() }
```

Раннер `trade.go` уже мультитикерный (итерирует `s.strategies`, тянет свечи по
`st.Ticker()`), дополнительных изменений не требует. Scalping шлёт только
Telegram-уведомления (ордера не выставляет) — добавление AFKS в живой раннер
безопасно.

## Данные

### Параметры

Создать `data/params/afks/`:
- `scalp.json` — стартовый одиночный набор (зеркало `rusal/scalp.json`, как
  отправная точка для ручного прогона).
- `grid.json` — сетка для калибровки (зеркало `rusal/grid.json`).

Прочие профили RUAL (`robust`/`trend`/`range`/…) для AFKS на этой итерации не
заводим — добавим по результатам калибровки, если понадобятся.

### Свечи

Дотянуть в `data/candles/`:
- `AFKS_Hour1.json`
- `AFKS_Day1.json`

через `cmd/backtest` (`svc.NewCandleProvider`, API-токен `T_BANK` в
`env/local.env`). Период — те же 24 мес + год лид-ина для дневной EMA, как для
RUAL.

## Сравнительный прогон (приёмка)

Запустить `cmd/backtest`:

1. RUAL, `DefaultParams` (без `-params`), 24 мес → отчёт.
2. AFKS, `DefaultParams`, 24 мес → отчёт.
3. AFKS, `-calibrate data/params/afks/grid.json` → калибровка + best.

Сравнить RUAL vs AFKS на **одинаковых `DefaultParams`** (изолирует
«стратегия vs бумага»), затем посмотреть, что даёт калибровка AFKS.

**Вывод**, который должен дать прогон: нормальные ли метрики AFKS на той же
логике. Если да — логика рабочая, RUAL плохо подходит. Если на обеих плохо —
проблема структурная (тема следующей итерации).

## Тестирование

- `adaptive_test.go`: перенос `rusal_test.go`; тесты decision-ядра не зависят от
  тикера — вызвать `adaptive.NewWithParams("RUAL", …)` (или табличный тикер).
- `rusal_test.go`: оставить тонкий тест, что `rusal.New().Ticker() == "RUAL"` и
  `DefaultParams()` возвращает ожидаемые ключевые значения (в т.ч.
  `TrendFilterPeriod == 100`).
- `afks_test.go`: `afks.New().Ticker() == "AFKS"`, `DefaultParams()` возвращает
  baseline (в т.ч. `TrendFilterPeriod == 0`).
- `internal/service/backtest/registry_test.go`: расширить — `Lookup("AFKS")`
  существует, `Build`+`ParseParams` дают рабочую стратегию с `Ticker()=="AFKS"`.
- Весь существующий backtest/engine/metrics-тестинг должен остаться зелёным.
- `go build ./...` и `go test ./...` зелёные.

## Вне scope (follow-up)

- Доработка метрик: Sharpe / Sortino / Calmar (return-over-MaxDD),
  exposure-aware цель калибровки вместо чистого `profit_factor`. Для текущего
  диагностического сравнения хватает PF / exposure / CAGR / Max DD.
- Структурная переработка логики входов/выходов (низкий exposure, микро-стопы по
  TRAIL) — рассматривается только если AFKS тоже покажет слабые метрики.
- Калибровка финальных боевых параметров AFKS и заведение профилей
  `robust/trend/range`.

## Риски

- **Перенос ломает поведение RUAL.** Митигировано тем, что `rusal_test.go`
  переезжает в `adaptive_test.go` как есть и должен остаться зелёным; отдельный
  тонкий тест фиксирует RUAL-дефолты.
- **Нет токена / нет данных AFKS.** Прогон свечей требует `T_BANK`; если токена
  нет — шаг 5 выполняется пользователем командой с `!`, остальное (код, тесты)
  не блокируется.
- **AFKS-дефолты «из головы».** Намеренно стартуем от обобщённого baseline и
  калибруем; не претендуем на боевые значения до калибровки.
```