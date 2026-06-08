# Дизайн: momentum-стратегия (EMA200 + MACD + объём + дневной ATR-запас)

Дата: 2026-06-08
Ветка: `feat/momentum-macd-strategy` (от `feat/levels-volume-profile-strategy`)
Статус: утверждён, готов к плану реализации

## Цель

Новая long-only стратегия на часовом таймфрейме поверх существующего движка
бэктестов (`internal/domain/backtest`). Ticker-agnostic ядро + per-ticker
калиброванные параметры, чтобы легко добавлять акции. Стратегия `levels` и
`scalping` остаются нетронутыми.

Заходим в лонг только при одновременном выполнении входных условий; SL/TP и
доп. идеи спроектированы как трейдерские дефолты. Отчёт по каждой сделке должен
показывать выставленные SL и TP, посчитанные ATR (часовой и дневной) и причину
входа человекочитаемой строкой. Условия входа должны легко расширяться новыми
параметрами без переписывания ядра.

## Решения, принятые при брейншторме

- **«Запас хода по дневному ATR» = «дневной ATR ещё не выбран»**: блокируем вход,
  если внутридневной ход уже съел бóльшую часть типичного дневного диапазона (ATR).
- **MACD-кросс «под нулём» — настраиваемый флаг на акцию** (`MACDBelowZeroOnly`,
  int 0/1, дефолт 1), калибруется вместе с периодами MACD.
- **SL/TP отданы на усмотрение (см. §4)**: структурный ATR-стоп + фикс. R-мультипликатор TP.

## 1. Архитектура

Зеркалит устройство пакета `levels`:

```
pkg/indicators/macd.go                               ← новый переиспользуемый MACD
internal/service/trading_strategy/momentum/
  strategy/core/core.go                              ← чистый Decide (ticker-agnostic), вся логика
  strategy/core/core_test.go
  strategy/<ticker>/<ticker>.go                      ← per-ticker Ticker + калиброванные DefaultParams
internal/service/backtest/momentum_registry.go       ← MomentumLookupOrGeneric + genericMomentumDefaults
internal/service/backtest/momentum_registry_test.go
cmd/backtest/main.go                                 ← +"momentum" в флаг -strategy
```

- Ядро реализует интерфейс `strategy.Strategy` (`Ticker() / Lookback() / Decide(MarketData)`),
  как `levels/strategy/core`. `Decide` чистая: считает свои индикаторы из `MarketData`, без I/O.
- `Params` — только поля `int`/`float64` (включая флаги как `int` 0/1), чтобы их свипила
  reflection-калибровка (`svc.RunGrid` / `ParamRows`).
- Реестр через `MomentumLookupOrGeneric(ticker)`: любой `-ticker` запускается сразу с
  generic-дефолтами; под конкретную акцию добавляется per-ticker пакет с калиброванными Params.
- Биндинг повторяет `levelsBindingFor`: `DefaultParams` / `Build` (`core.NewWithParams`) / `ParseParams`.

**Почему не складывать в `levels`:** это иная стратегия (моментум, а не отбой от уровней);
связывать их вредно и противоречит требованию оставить `levels` как есть.

## 2. Условия входа (flat → long, все обязательны)

Оцениваются на закрытой часовой свече. Вход — `model.SignalBuy`, только когда позиции нет
и выполнены ВСЕ условия:

| # | Условие | Параметры |
|---|---|---|
| 1 | Часовой `close > EMA(EMAPeriod)` по часовым закрытиям | `EMAPeriod=200` |
| 2 | Бычий кросс MACD на последней свече: `prev: macd ≤ signal`, `now: macd > signal`; если `MACDBelowZeroOnly=1` — дополнительно `macd < 0` в момент кросса | `MACDFast=12, MACDSlow=26, MACDSignal=9, MACDBelowZeroOnly=1` |
| 3 | Объём последней свечи `> VolMultiplier × SMA(объём, VolLookback)` (reuse `indicators.VolumeConfirmed`) | `VolLookback=20, VolMultiplier=1.2` |
| 4 | Остался запас дневного ATR: `(TodayHigh − TodayLow) < MaxDailyATRUsed × DailyATR` | `DailyATRPeriod=14, MaxDailyATRUsed=0.6` |
| — | Анти-черн: пропуск, если `ATR(час) < MinATRFrac × price` (`<=0` отключает) | `MinATRFrac=0.003` |

Флаги-переключатели (`MACDBelowZeroOnly`) — `int 0/1`, чтобы участвовать в grid-калибровке.

## 3. Индикатор MACD (новый)

`pkg/indicators/macd.go`:

```go
// MACD возвращает линию MACD и сигнальную линию (oldest-first, выровнены с closes).
// macd = EMA(closes, fast) − EMA(closes, slow); signal = EMA(macd, signalPeriod).
func MACD(closes []float64, fast, slow, signal int) (macdLine, signalLine []float64)
```

- Использует `internal/domain/ema.Compute` (или локальный EMA, чтобы не плодить
  зависимость pkg→internal — решается в плане; вероятно локальный EMA в pkg).
- Возвращает `nil, nil` при недостатке истории (`len(closes) < slow+signal`).
- Кросс определяется по последним двум значениям обеих линий.
- Покрыт unit-тестами (восходящий/нисходящий кросс, кросс под/над нулём, нехватка истории).

## 4. Стоп-лосс и тейк-профит

- **SL — структурный + ATR-буфер:** `SL = swingLow(SwingLowWindow) − SLMult × ATR(час)`.
  Дефолты `SwingLowWindow=10, SLMult=0.5`. Замораживается на входе (движок уже несёт
  `entryStop` через `Position.StopLoss`).
- **TP — фикс. R-мультипликатор:** `risk = entry − SL`; `TP = entry + TakeProfitRR × risk`.
  Дефолт `TakeProfitRR=2.0` (2R). Конкретное видимое число в отчёте.
- **Опц. трейлинг (по умолчанию выкл):** chandelier `recentHigh(ChandelierWindow) − TrailMult×ATR`,
  армится после `TrailArmATR×EntryATR` прибыли. `UseTrail=0, TrailMult=2.5, ChandelierWindow=20,
  TrailArmATR=1.0`. Когда `UseTrail=1` — трейл заменяет фикс-TP (едем в тренде).

**Качество входа:** `risk > 0` обязательно; при `MinRR>0` отвергаем вход, если
`(TP − entry) < MinRR × risk` (`MinRR=1.5`, согласовано с `TakeProfitRR=2.0`).

### Логика выхода (в позиции), приоритет в баре

1. `barLow ≤ SL` → `SignalSell`, `Reason="SL"`. Налив `min(SL, open)` (учёт гэп-дауна).
2. иначе `barHigh ≥ TP` → `SignalSell`, `Reason="TP"`. Налив `max(TP, open)` (лимитный, гэп-ап лучше).
3. иначе при `UseTrail=1` и взведённом трейле `barLow ≤ chandelier` → `SignalSell`, `Reason="TRAIL"`,
   `StopLoss=chandelier`, налив `min(chandelier, open)`.

SL имеет приоритет над TP при касании обоих в одном баре (консервативно).

## 5. Доп. идеи (заложены как параметры, дефолты разумные)

- **Кулдаун после стопа** (опц., `CooldownBars=0` выкл): не входить N баров после SL —
  не лезть в падающий нож. Параметр на будущее.
- **Дневной тренд-фильтр** (опц., `DailyTrendPeriod=0` выкл): доп. конфлюэнс «close выше
  дневной EMA». Лёгкое расширение под «докинуть условие».

Оба — отдельные параметры; ядро не переписывается при добавлении новых условий.

## 6. Причина входа словами + ATR в отчёте

- `model.Signal` += `EntryReason string` (заполняется при `SignalBuy`).
- `backtest.Trade` += `EntryReason string`; `portfolio.open` принимает и сохраняет его;
  `engine.go` прокидывает `sig.EntryReason` в `open`.
- Отчёт (`report.go` Markdown + CSV) += колонка «Причина входа». Колонка ATR остаётся
  (= ATR(час) на входе). В строке причины — оба посчитанных ATR (часовой и дневной).

Пример строки причины:

> `Тренд↑ (close 36.20 > EMA200 35.10); MACD бычий кросс под нулём (−0.05→+0.02); объём 1.8×ср(20); дневной ATR-запас 58% (прошло 0.58 из 1.40); ATR(ч)=0.35, ATR(д)=1.40; SL=35.30 (−0.90); TP=38.00 (+1.80, 2R)`

## 7. Правки общего движка (аддитивные; levels/scalping не ломают)

- `strategy.MarketData` += `DailyATR float64`, `TodayHigh float64`, `TodayLow float64`.
  Движок считает:
  - `DailyATR` = `indicators.ATR` по завершённым дневным свечам (highs/lows/closes),
    без заглядывания вперёд (нужны не только closes — добавить `visibleDailyOHLC`-хелпер).
  - `TodayHigh/TodayLow` = экстремумы часовых свечей текущего MSK-календарного дня
    до текущего бара включительно (механика `startOfDay`/`mskLoc` уже есть в `engine.go`).
- `model.Signal` += `EntryReason`; `Trade` += `EntryReason`.
- Налив `"TP"` в `engine.go` (расширить текущую спец-обработку `SL`/`TRAIL`).
- Существующие стратегии (`levels`, `scalping`) новые поля игнорируют → обратная совместимость.

## 8. Параметры (итоговый список Params)

Все `int`/`float64` (флаги — `int` 0/1), grid-совместимы.

```
EMAPeriod          int     = 200    // часовой тренд-фильтр
MACDFast           int     = 12
MACDSlow           int     = 26
MACDSignal         int     = 9
MACDBelowZeroOnly  int     = 1      // 1 = кросс только при macd<0
VolLookback        int     = 20
VolMultiplier      float64 = 1.2
DailyATRPeriod     int     = 14
MaxDailyATRUsed    float64 = 0.6    // блок входа, если внутридневной ход >= доли дневного ATR
ATRPeriod          int     = 14     // часовой ATR для стопов/анти-черна
SwingLowWindow     int     = 10
SLMult             float64 = 0.5
TakeProfitRR       float64 = 2.0
MinRR              float64 = 1.5
MinATRFrac         float64 = 0.003
UseTrail           int     = 0
TrailMult          float64 = 2.5
ChandelierWindow   int     = 20
TrailArmATR        float64 = 1.0
CooldownBars       int     = 0      // доп. идея, выкл
DailyTrendPeriod   int     = 0      // доп. идея, выкл
```

`Lookback()` = максимум из потребителей (`EMAPeriod`, `MACDSlow+MACDSignal`, `VolLookback+1`,
`ATRPeriod+1`, `SwingLowWindow`, `ChandelierWindow`) + запас.

## 9. Первый тикер и запуск

- Реестр: `MomentumLookupOrGeneric` + `genericMomentumDefaults` (значения = дефолты выше),
  как `LevelsLookupOrGeneric`/`genericLevelsDefaults`.
- Любой тикер запускается сразу:
  `go run ./cmd/backtest -ticker RUAL -strategy momentum -interval Hour1 -months 12`
- Калибровка как у levels: `-calibrate data/params/<ticker>/momentum_grid.json`,
  затем зашить победителя в per-ticker пакет (MACD-периоды — индивидуально).

## 10. Тестирование (TDD)

- `pkg/indicators/macd_test.go` — MACD и детект кросса.
- `momentum/strategy/core/core_test.go` — таблично: каждое входное условие по отдельности
  блокирует/пропускает; SL/TP расчёт; приоритет SL над TP; трейл вкл/выкл; формирование
  `EntryReason`; `Lookback`.
- `backtest/momentum_registry_test.go` — биндинг, парсинг частичного JSON, generic-фоллбек.
- `internal/domain/backtest/engine_test.go` — `DailyATR`/`TodayHigh`/`TodayLow` без lookahead;
  налив `"TP"` по `max(TP, open)`.

## Вне рамок (YAGNI / future)

- Живая торговля (live wiring `EntryReason`/состояния входа) — отдельная итерация, как у levels.
- Частичные выходы / пирамидинг — движок all-in single-position, не поддерживает.
- Реальная калибровка под конкретные акции — отдельная задача после реализации.
