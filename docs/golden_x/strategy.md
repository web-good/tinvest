# Алгоритм Golden X

Ядро стратегии — чистая функция `Detect()` в `internal/service/trading_strategy/golden_x/detector.go:19`. Никакого I/O, никакого времени, никакой телеметрии: на вход — массив недельных свечей (закрытые + текущая формирующаяся), на выходе — `dto.Signal`. Всю обвязку (загрузку данных, отправку в Telegram) делает обёртка `service.Trade`.

## Жизненный цикл сигнала на одну акцию

```
[свечи из Tinkoff]
       ↓
[удаление nil-свечей]        ← compactCandles()
       ↓
[адаптивный RSI ряд]         ← adaptiveRSIForShare()
       ↓
[пороги P5/P15 и P80/P90/P95]← adaptiveThresholds + adaptiveSellThresholds
       ↓
[tier покупки 🟢/🟡 или нет] ← tierFromAdaptive()
       ↓
[если есть tier покупки:]
   ├── фильтр тренда EMA200 ← trendStatusFromClosed()
   ├── бычья дивергенция    ← bullishDivergence()
   ├── подтверждение объёма ← indicators.VolumeConfirmed()
   └── стоп ATR             ← stopFromATR()
       ↓
[tier продажи 🟠/🔴/🚨 или нет] ← sellTierFromAdaptive()
       ↓
[dto.Signal] → service.Trade
                ↓
        [форматирование notification.Trade()]
                ↓
        [Telegram Bot API]
```

### Шаг 1. Загрузка данных

`fetchWeeklyCandles()` (`trade.go:138`) тянет ~260 недельных свечей через gRPC Tinkoff Invest. Tinkoff возвращает в том числе текущую формирующуюся неделю (помечена `IsComplete=false`). `compactCandles()` (`trend_filter.go:63`) убирает только nil-элементы — формирующаяся свеча сохраняется и попадает в `Detect`, поэтому RSI/тренд/стоп реагируют на движение цены внутри текущей недели, не дожидаясь её закрытия в понедельник 00:00 MSK.

### Шаг 2. Адаптивный RSI

`adaptiveRSIForShare()` (`trade.go:156`):

1. Считает полный ряд RSI методом Wilder через `computeRSISeries()` (`rsi.go:14`). Период — индивидуальный для каждой акции (`RSILength` в `shares.go`, обычно 7..11).
2. Отбрасывает первые `rsiPeriod` значений (warmup).
3. Проверяет, что осталось хотя бы `AdaptiveWindowMin` (по умолчанию 100) значений — иначе `ErrAdaptiveInsufficientHistory`.
4. Обрезает ряд до `AdaptiveWindowMax` (по умолчанию 200) последних значений — чтобы пороги отражали недавнее поведение, а не глубокую историю.

### Шаг 3. Адаптивные пороги

`adaptiveThresholds()` и `adaptiveSellThresholds()` (`percentile.go:49, 62`) — это просто перцентили (метод R-7, как NumPy/Excel) на обрезанном RSI-ряду:

- Покупка: `P5 = BuyGreen`, `P15 = BuyYellow`.
- Продажа: `P80 = SellYellow`, `P90 = SellOrange`, `P95 = SellRed`.

Идея: если RSI акции исторически почти никогда не падал ниже 35, то «классический» уровень 30 для неё бесполезен. Адаптивные пороги ловят именно её локальные экстремумы.

### Шаг 4. Tier покупки

`tierFromAdaptive()` (`percentile.go:34`):

| Условие | Tier | Emoji |
|---|---|---|
| `RSI < P5` | `tierGreen` | 🟢 |
| `P5 ≤ RSI < P15` | `tierYellow` | 🟡 |
| иначе | `tierNone` | — |

Сравнения **строгие** (`<`), не `≤`.

### Шаг 5. Подтверждения покупки (только если tier ≠ none)

#### 5.1 Фильтр тренда (`trend_filter.go:42`)

Включается через `useTrendFilter` (в проде включён для обеих стратегий). Считает EMA200 (по `TrendEMAPeriod`) и сравнивает последний `Close` с последним EMA:

| Условие | TrendStatus | Mark |
|---|---|---|
| `Close > EMA200_W` | `TrendWith` | ✅ |
| `Close ≤ EMA200_W` | `TrendAgainst` | 🚫 |

Если истории недостаточно для EMA — `ErrInsufficientHistory`. Если `useTrendFilter` не включён, поле остаётся `TrendUnknown`, в сообщении ничего не печатается.

#### 5.2 Бычья дивергенция (`divergence.go:40`)

Ищется по фракталу Williams с `k=2` (классическое значение, константа `divergenceFractalK` в `detector.go`). Логика:

1. Найти самый свежий фрактальный минимум в окне `DivergenceLookbackWeeks` (по умолчанию 52 недели).
2. Проверить: `low[последняя] < low[pivot]` И `RSI[последняя] > RSI[pivot]`.
3. Если оба условия выполнены — бычья дивергенция, флаг `DivergenceOK = true`, в сообщении появляется 📈.

#### 5.3 Подтверждение объёма (`detector.go:64`)

`indicators.VolumeConfirmed()`: `volume[last] > VolumeMultiplier × SMA(volume, VolumeSMALookback)`. По дефолту: `>1.5 × SMA20`. Флаг `VolumeOK = true` → 🔊 в сообщении.

#### 5.4 Стоп ATR (`stop.go:19`)

`stopFromATR(close, atr, K)`:

```
StopPrice    = Close − K × ATR14
DistancePct  = (Close − Stop) / Close × 100
```

`K` берётся через `kForKind()`:

- Growth → `ATRMultiplierGrowth` (1.5)
- Dividend (и unknown) → `ATRMultiplierDividend` (2.0)

При `ATR ≤ 0` или вырожденных значениях возвращается `Stop{}` — пустой, в сообщении ничего не выводится.

### Шаг 6. Tier продажи

`sellTierFromAdaptive()` (`percentile.go:82`):

| Тип | RSI > P95 | RSI > P90 | RSI > P80 |
|---|---|---|---|
| 🥇 Dividend | 🚨 | 🔴 | 🟠 |
| 🥈 Growth | — | 🔴 (полный выход) | — |

Сравнения строгие (`>`). Growth-стратегия игнорирует P80 и P95 — продажа только на P90.

### Шаг 7. Формирование уведомления

Все акции с tier ≠ `tierNone` попадают в уведомление — покупочные (`tierYellow`, `tierGreen`) в секцию «Сигналы на покупку», продажные (`tierSellYellow`, `tierSellOrange`, `tierSellRed`) в секцию «Сигналы на продажу». Дедупликации нет: каждый тик стратегии отправляет полный список всех акций, находящихся в зонах покупки или продажи.

### Шаг 8. Отправка в Telegram

`notification.Trade()` (`notification/notifications.go:12`) собирает HTML-сообщение со всеми покупками и продажами одной партией. Сообщение начинается с блока легенды (`legendBlock`), объясняющего все emoji. Доставка через Telegram Bot API на все `chatIds` из конфига (см. `internal/app/app.go`).

## Score сигнала

После `Detect()` акции классифицируются в `classifier.Classify()` (чистая функция, без I/O). Там же считается `Score` — композитная сила сигнала (`classifier/classifier.go:signalScore`):

| Компонент | Вклад |
|---|---|
| `TierGreen` | +3 |
| `TierYellow` | +1 |
| `TrendWith` (тренд за нас) | +2 |
| `DivergenceOK` (бычья дивергенция) | +2 |
| `VolumeOK` (объём подтверждён) | +1 |
| Фунд-бонус (абсолютные полосы композита дивидендного скринера) | +0..+3 |

Итоговый диапазон `Score`: **1..11** (было 1..8 до добавления фунд-бонуса). Список покупок в уведомлении сортируется по убыванию `Score`.

Фунд-бонус приходит как данные извне: `service.Trade` прокидывает `s.rankProvider.RankBonus(instrumentID)` в `detector.DetectAll()`, которая кладёт его в `DetectResult.FundamentalBonus` для каждой акции; `Classify()` лишь копирует это значение в `ShareResult.FundamentalBonus` и суммирует — сама остаётся чистой (никакого I/O, никакого времени). Источник ранга — `dividend.RankProvider` (скринер дивидендных бумаг, `internal/service/screener/dividend`), обновляется ~раз в 24 часа. Если провайдер не подключён (`golden_x.WithRankProvider` не передан), используется no-op-реализация — бонус всегда 0, поведение идентично коду до интеграции. В уведомлении бумаги с положительным бонусом дополнительно помечаются 🏆.

Как считается бонус (`bonusFromScore`, откалибровано 2026-07-21): композит скринера (0..100) отображается **абсолютными полосами** — ≥73 → +3, ≥67 → +2, ≥56 → +1, иначе 0. Полосы абсолютны (а не перцентильны), поэтому величина бонуса не зависит от состава сканируемой вселенной. Бонус получают только имена, прошедшие ворота скринера, в т.ч. фильтр ликвидности по капитализации (₽50 млрд): micro-cap и не платящие бонуса не получают. Полное описание gate, пилляров и калибровки — `docs/dividend/screener.md`.

## Разница Dividend vs Growth

| Аспект | 🥇 Dividend | 🥈 Growth |
|---|---|---|
| Фильтр тренда (EMA200) | включён ✅/🚫 | включён ✅/🚫 |
| Множитель ATR для стопа | 2.0 | 1.5 |
| Тиры продажи | P80 🟠, P90 🔴, P95 🚨 | P90 🔴 |
| Стиль выхода | Частичные по 1/3 | Полный выход на первом сигнале |
| Список тикеров | `shares.Dividend()` (11 шт) | `shares.Growth()` (6 шт) |
| Назначение | Дивидендные «голубые фишки», долгие удержания | Растущие акции, активный выход на росте |

«Частичные выходы» — это бэктестная семантика (`backtest/position.go`). В live-режиме стратегия лишь рассылает алерты, торговые решения принимает оператор.

## Семантика `dto.Signal`

```go
type Signal struct {
    RSI            float64       // последний RSI
    LastClose      float64       // последняя цена закрытия
    Stop           Stop          // ATR-based стоп (пустой, если истории мало)
    Thresholds     Thresholds    // {P5, P15} для покупки
    SellThresholds SellThresholds // {P80, P90, P95} для продажи
    TrendStatus    TrendStatus   // TrendWith/TrendAgainst/TrendUnknown
    GreenBuy       bool          // RSI < P5
    YellowBuy      bool          // P5 ≤ RSI < P15
    DivergenceOK   bool          // бычья RSI-дивергенция
    VolumeOK       bool          // объём подтверждён
}
```

Файл: `internal/service/trading_strategy/golden_x/dto/signal.go`.

## Где конфигурируется в проде

`internal/app/app.go` создаёт два экземпляра задачи (`goldenx.Trade`):

```go
goldenx.Trade{
    Kind:           goldenx.StrategyKindDividend,    // или StrategyKindGrowth
    Interval:       enum.Week1,
    Scheduler:      "0 */5 * * *",                   // cron-выражение
    ShareList:      *a.collection.GoldInstruments,   // или GrowthShare
    UseTrendFilter: true,
}
```

В `dev`-режиме шедулер крутит задачу как goroutine, в `prod` — через `pkg/scheduler` (cron-like). См. `internal/app/app.go` (`runDev` / `runProd`).
