# Reversion → дневной RSI: переработка стратегии

Дата: 2026-06-12. Ветка: `feat/reversion-rsi-dip`.

## Цель

Переработать стратегию `reversion` в чистый RSI mean-reversion на **дневном**
таймфрейме. Покупка и продажа управляются исключительно RSI; всё калибруется
через grid. Убрать из логики volume-gate, Stochastic-гейт и time-stop. Стоп —
дневной ATR с калибруемым множителем. Фильтр тренда — опциональный тумблер.

## Что меняется по сравнению с текущей версией

| Аспект            | Было                                   | Стало                                   |
|-------------------|----------------------------------------|-----------------------------------------|
| Таймфрейм         | hourly (`-interval Hour1`)             | daily (`-interval Day1`)                |
| Триггер выхода    | только RSI-кросс вверх через overbought| `ExitMode`: вход/выход из зоны          |
| Триггер входа     | `EntryMode` (confirmed/knife)          | `EntryMode`: вход/выход из зоны (та же ось)|
| Фильтр тренда     | обязательный                           | опциональный (`UseTrend`)               |
| Стоп              | фикс % `entry×(1−StopLossPct)`         | ATR `entry − ATRMult×ATR`               |
| Volume-gate       | обязательный                           | удалён из логики                        |
| Stochastic-gate   | опциональный                           | удалён из логики (пакет остаётся в репо)|
| Time-stop         | `MaxHoldBars`                          | удалён                                  |

Плумбинг данных НЕ меняется: движок бэктеста уже отдаёт дневные свечи как
основные при `-interval Day1`, поэтому RSI/EMA/ATR считаются на основном
(дневном) ряду `Closes/Highs/Lows`. Пакет `pkg/indicators/Stochastic` и его тесты
сохраняются — лишь перестают использоваться этой стратегией.

## Параметры (`core.Params`)

```go
type Params struct {
    UseTrend      int     // 0 = без фильтра тренда, 1 = только в аптренде
    FastEMA       int     // быстрая EMA режима (дефолт 50)
    SlowEMA       int     // медленная EMA + пол по цене (дефолт 200)
    RSIPeriod     int     // длина RSI; обязателен (>0)
    RSIOversold   float64 // граница зоны перепроданности
    RSIOverbought float64 // граница зоны перекупленности
    EntryMode     int     // 0 = вход в зону, 1 = выход из зоны (по oversold)
    ExitMode      int     // 0 = вход в зону, 1 = выход из зоны (по overbought)
    ATRPeriod     int     // длина ATR (дефолт 14); теперь гейтит стоп
    ATRMult       float64 // стоп = entry − ATRMult×ATR; обязателен (>0)
}
```

Удаляются поля: `VolLookback`, `VolMultiplier`, `UseStoch`, `StochPeriod`,
`StochSmooth`, `StochOversold`, `StopLossPct`, `MaxHoldBars`.

### Семантика триггеров (единая ось «вход/выход из зоны»)

RSI считается одним рядом длины `RSIPeriod`; используются текущее и предыдущее
значения (`rsiNow`, `rsiPrev`).

**Вход** (`EntryMode`, зона = перепроданность `RSIOversold`):
- `0` — вход в зону: `rsiPrev ≥ oversold && rsiNow < oversold` (кросс вниз);
- `1` — выход из зоны: `rsiPrev ≤ oversold && rsiNow > oversold` (кросс вверх).

**Выход** (`ExitMode`, зона = перекупленность `RSIOverbought`):
- `0` — вход в зону: `rsiPrev ≤ overbought && rsiNow > overbought` (кросс вверх);
- `1` — выход из зоны: `rsiPrev ≥ overbought && rsiNow < overbought` (кросс вниз).

Семантика согласована: `0` = «момент входа цены в зону», `1` = «момент выхода».

### Определение тренда

`UseTrend=1` ⇒ требуется аптренд: `EMA_fast > EMA_slow` **И** `close > EMA_slow`.
`UseTrend=0` ⇒ фильтр отключён, покупаем перепроданность в любом режиме.
`FastEMA/SlowEMA` остаются параметрами (50/200), грид их не свипит в v1.

## Логика решения

`decide()` — чистое ядро над уже посчитанными индикаторами.

**Когда flat (вход):**
1. Если `UseTrend=1` и не аптренд → нет сигнала.
2. RSI-триггер по `EntryMode` не сработал → нет сигнала.
3. `ATRMult ≤ 0` или `atr ≤ 0` → нет сигнала (стоп обязателен).
4. `stop = price − ATRMult×atr`; `risk = price − stop`; если `risk ≤ 0` → нет сигнала.
5. Иначе `SignalBuy`, `StopLoss = stop`, заполняем `ATR/RSI/EntryReason`.

**Когда в позиции (`manage`), приоритет защиты:**
1. `barLow ≤ pos.StopLoss` → `SignalSell`, причина `SL`.
2. RSI-триггер по `ExitMode` сработал → `SignalSell`, причина `RSI`.

Стоп замораживается на входе (`pos.StopLoss`), как сейчас. Time-stop отсутствует.

`Explain()` переписывается под новую цепочку гейтов в порядке `decide`:
тренд (если включён) → RSI-вход → стоп. `Lookback()` = `max(SlowEMA, FastEMA,
RSIPeriod+1, ATRPeriod+1) + буфер` (Vol/Stoch-слагаемые убираются).

## Сетка калибровки (`reversion_grid.json`, по фазам)

```jsonc
{
  "phases": [
    {
      "name": "entry", "keepTop": 5,
      "grid": {
        "UseTrend":    [0, 1],
        "FastEMA":     [50],
        "SlowEMA":     [200],
        "RSIPeriod":   [4, 5, 10, 12, 14],
        "RSIOversold": [15, 20, 35],
        "EntryMode":   [0, 1]
      }
    },
    {
      "name": "exit", "keepTop": 5,
      "grid": {
        "RSIOverbought": [65, 70, 85],
        "ExitMode":      [0, 1],
        "ATRMult":       [1.0, 1.5, 2.0],
        "ATRPeriod":     [14]
      }
    }
  ]
}
```

Одна и та же сетка кладётся во все 8 `data/params/<ticker>/reversion_grid.json`.

## Дефолты

Generic-дефолты (в `reversion_registry.go`) и все 8 пер-тикерных `DefaultParams`
приводятся к новой форме и идентичны до калибровки:

```go
core.Params{
    UseTrend: 1, FastEMA: 50, SlowEMA: 200,
    RSIPeriod: 14, RSIOversold: 20, RSIOverbought: 70,
    EntryMode: 1, ExitMode: 0,
    ATRPeriod: 14, ATRMult: 1.0,
}
```

`EntryMode=1` (покупка на выходе из перепроданности — подтверждённый отскок),
`ExitMode=0` (продажа на входе в перекупленность). После калибровки победители
хардкодятся в пер-тикерные пакеты, как у momentum.

## Затрагиваемые файлы

- `internal/service/trading_strategy/reversion/strategy/core/core.go` — ядро.
- `internal/service/trading_strategy/reversion/strategy/core/core_test.go` — тесты.
- `internal/service/trading_strategy/reversion/strategy/<ticker>/<ticker>.go` ×8 — DefaultParams.
- `internal/service/backtest/reversion_registry.go` — `genericReversionDefaults`.
- `internal/service/backtest/reversion_registry_test.go` — при необходимости.
- `data/params/<ticker>/reversion_grid.json` ×8 — новые сетки.
- `docs/reversion/strategy.md` — обновить объяснение стратегии.

## Тестирование (TDD)

Табличные тесты `core_test.go` покрывают:
- RSI-вход по обоим `EntryMode` (кросс вниз / кросс вверх через oversold);
- RSI-выход по обоим `ExitMode` (кросс вверх / кросс вниз через overbought);
- `UseTrend=0` пропускает вход вне тренда; `UseTrend=1` блокирует вне аптренда;
- ATR-стоп: верный уровень `entry − ATRMult×atr`; срабатывание `barLow ≤ stop`;
- приоритет защиты: стоп бьёт RSI-выход на одном баре;
- санити: `ATRMult ≤ 0` или `atr ≤ 0` → нет входа.

Прогон: `go build ./... && go vet ./... && go test ./...`.
Ручной запуск: `go run ./cmd/backtest -ticker SBER -strategy reversion -interval Day1 -months 24 -out ./reports/SBER`.

## Вне области (YAGNI)

- `-basket` для reversion (раннер momentum-hardcoded — как и в v1).
- Свип `FastEMA/SlowEMA` в сетке.
- Трейлинг-стоп, частичные выходы, ADX/режимные фильтры.
