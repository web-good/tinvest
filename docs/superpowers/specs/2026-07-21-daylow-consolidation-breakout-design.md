# Day-Low Consolidation Breakout (`daylow`) — Design

**Дата:** 2026-07-21
**Статус:** утверждён к реализации (brainstorming)
**Область:** новая backtest-стратегия `daylow` в существующем движке `internal/domain/backtest`

## 1. Идея

Лонг-онли внутридневная стратегия на 5-минутном таймфрейме со старшим часовым как
фильтром тренда. Ловит отбой от вчерашнего дневного минимума через пробой зоны
консолидации:

1. цена спускается к вчерашнему дневному минимуму (поддержка);
2. у этого уровня формируется зона консолидации — мелкие свечи, тесный диапазон;
3. появляется зелёная импульсная свеча, пробивающая зону вверх на возросшем объёме;
4. вход по закрытию импульсной свечи.

Стоп ставится за зону мелких свечей, тейк-профит — кратно риску (RR, дефолт 2.0;
калибруем {1.5, 2.0, 2.5}). Все пороги и оба таймфрейма — настраиваемые параметры
бэктеста.

Стратегия торгует **только в активные часы МосБиржи** (окно входа настраивается,
дефолт 09:00–14:00 МСК). **Выход (SL/TP) — в любое время суток**, без оконного
ограничения.

## 2. Как встаёт в существующую архитектуру

Движок `internal/domain/backtest` уже даёт всё нужное:

- чистый `Strategy.Decide(md strategy.MarketData) model.Signal` — вызывается на каждом
  баре; получает окно свечей (`Highs/Lows/Closes/Volumes/Times`), завершённые дневные
  (`DailyLows/DailyHighs/DailyCloses`) и HTF-свечи (`HTFCloses/...`), а также `Position`;
- движок замораживает стоп при входе (`Position.StopLoss`) и моделирует intrabar-филлы:
  стоп-причины (`model.IsStopReason`) филлятся `min(StopLoss, open)`, `"TP"` — `max(TakeProfit, open)`;
- грид-калибровка (`RunPhases`), walk-forward (`RunWalkForward`), отчёты — общие для всех стратегий;
- реестр `Binding` (scalping/reversion) подключает движки в `cmd/backtest`.

Стратегия — третий движок рядом со `scalping` и `reversion`. Никаких изменений в
логике портфеля/метрик/отчётов.

### Требуемые доработки платформы

1. **5-минутный интервал.** `internal/enum` не знает Minutes5. Добавить
   `Minutes5 Interval = 2` (Tinkoff `CANDLE_INTERVAL_5_MIN = 2`) в константы,
   `intervalNames`, `ToTimeDuration` (`5*time.Minute`), `ToNumberInvestAPI` (→2).
   Добавить `"Minutes5"` в `parseInterval` (`cmd/backtest/main.go`).

2. **Настраиваемый HTF-интервал.** Сейчас `engine.go` жёстко использует
   `htfInterval = 4 * time.Hour` в `Run`/`Trace` (через `AssembleMarketData` →
   `visibleCompletedHTF`). Ввести `Config.HTFInterval time.Duration`; `Run`/`Trace`
   используют `cfg.HTFInterval` вместо константы, с фолбэком на 4h когда поле нулевое
   (сохраняет поведение reversion). `cmd/backtest` заполняет поле: для `daylow` — из
   нового флага `-htf-interval` (дефолт `Hour1`), для `reversion` — `Hour4`.

3. **Загрузка HTF-свечей для `daylow`.** В `cmd/backtest` `htfCandles` грузятся только
   для reversion (Hour4). Расширить: для `daylow` грузить HTF по `-htf-interval` с тем
   же годовым lead-in, что и дневные (прогрев HTF-EMA).

## 3. Пакет и файлы

```
internal/service/trading_strategy/daylow/strategy/core/
    core.go        # Params + Strategy (реализует strategy.Strategy), чистая логика
    core_test.go   # table-driven тесты фаз/уровней + интеграция через движок
internal/service/backtest/daylow_registry.go   # Binding + DayLowLookupOrGeneric
```

`daylow` — ticker-agnostic, per-тикерных пакетов на старте нет (как generic-binding
у scalping/reversion). Пер-тикерные конфиги добавляются позже при калибровке.

## 4. Параметры (`core.Params`)

Все поля `int`/`float64` (флаги как `int` 0/1), чтобы reflection-грид мог их свипать.

| Параметр | Тип | Смысл | Дефолт |
|---|---|---|---|
| `LowProximityPct` | float64 | макс. отклонение `zoneLow` от вчерашнего минимума, доля (0.005 = 0.5%) | 0.005 |
| `ConsolBars` | int | сколько свечей перед импульсной образуют зону | 6 |
| `ConsolRangeATR` | float64 | зона «тесная», если её высота ≤ `ConsolRangeATR × ATR` | 1.0 |
| `ATRPeriod` | int | период ATR (единица измерения зоны/стопа/тела) | 14 |
| `ImpulseBodyATR` | float64 | тело импульсной свечи ≥ `ImpulseBodyATR × ATR` | 0.8 |
| `VolMult` | float64 | объём импульса ≥ `VolMult × средний объём зоны` | 1.5 |
| `StopBufferATR` | float64 | стоп = `zoneLow − StopBufferATR × ATR` | 0.25 |
| `RR` | float64 | тейк = `entry + RR × (entry − stop)` | 2.0 |
| `HTFTrendEMA` | int | период EMA на HTF-часовых; 0 = фильтр тренда выкл. | 50 |
| `EntrySessionStartMin` | int | начало окна входа, минут от полуночи МСК | 540 (09:00) |
| `EntrySessionEndMin` | int | конец окна входа, минут от полуночи МСК | 840 (14:00) |
| `CloseAtDayEnd` | int | 1 = закрывать позицию к концу торгового дня; 0 = держать | 0 |
| `DayEndMin` | int | порог закрытия для `CloseAtDayEnd`, минут от полуночи МСК | 1120 (18:40) |

## 5. Логика `Decide`

Стратегия **без внутреннего состояния** — сетап пересчитывается из окна каждый бар
(как reversion пересчитывает индикаторы). `Decide` чистая, без I/O.

### 5.1. Когда флэт (поиск входа)

На баре `i` (последний в окне) последовательно проверяются гейты; любой невыполненный →
`SignalNone`:

1. **Сессионное окно.** `t := md.Times[last]` в `Europe/Moscow`; минута дня
   `min := t.Hour()*60 + t.Minute()` должна лежать в `[EntrySessionStartMin, EntrySessionEndMin)`.
   Если `md.Times` пуст — гейт пропускается (деградация как в reversion volume-фильтре).
2. **ATR** посчитан по окну; при прогреве (нет валидного ATR) — выход.
3. **Зона консолидации.** Окно `[i-ConsolBars .. i-1]` (свечи перед импульсной):
   `zoneHigh = max(High)`, `zoneLow = min(Low)`. Гейт: `zoneHigh − zoneLow ≤ ConsolRangeATR × ATR`.
4. **Близость к вчерашнему минимуму.** `prevDayLow = md.DailyLows[last]` (последний
   завершённый день). Гейт: `|zoneLow − prevDayLow| ≤ LowProximityPct × prevDayLow`.
   Если `DailyLows` пуст — выход (сетап неопределён).
5. **Импульсная свеча** = текущий бар `i`: `close > open` (зелёная);
   `body = close − open ≥ ImpulseBodyATR × ATR`; `close > zoneHigh` (пробой зоны вверх).
6. **Рост объёма.** `Volumes[i] ≥ VolMult × avg(Volumes[i-ConsolBars .. i-1])`.
7. **Фильтр тренда H1** (если `HTFTrendEMA > 0` и EMA прогрета): последний
   `HTFCloses[last] ≥ EMA(HTFCloses, HTFTrendEMA)`. Если HTF-данных мало (не прогрето) —
   тренд не подтверждён → выход.

Все гейты пройдены → `SignalBuy`:
- `Price = close` (движок филлит вход по close бара);
- `StopLoss = zoneLow − StopBufferATR × ATR`;
- `TakeProfit = close + RR × (close − StopLoss)`;
- `ATR = <ATR бара>` (замораживается в `Position.EntryATR`);
- `Level = prevDayLow`; `EntryReason` — человекочитаемое описание сетапа.

### 5.2. Когда в позиции (управление)

TP **не хранится** в `Position` и не нужен там: реконструируется из замороженных
`PurchasePrice` и `StopLoss`:

```
R  := pos.PurchasePrice - pos.StopLoss
tp := pos.PurchasePrice + RR*R
```

Приоритет выходов (консервативно — при одновременном касании считаем стоп):

1. `md.Lows[last] ≤ pos.StopLoss` → `SignalSell`, `Reason="SL"`, `StopLoss=pos.StopLoss`
   (движок филлит `min(SL, open)`).
2. иначе `md.Highs[last] ≥ tp` → `SignalSell`, `Reason="TP"`, `TakeProfit=tp`
   (движок филлит `max(tp, open)`).
3. иначе, если `CloseAtDayEnd=1` и минута бара по МСК `≥ DayEndMin` → `SignalSell`,
   `Reason="EOD"` (филл по close).
4. иначе `SignalNone` (держим).

Коды `"SL"`/`"TP"` движок уже знает (`IsStopReason`/TP-ветка). `"EOD"` — не стоп-причина,
филлится по close, отдельной регистрации не требует.

**Почему порог по времени, а не «последний бар дня»:** «последний бар дня» без
look-ahead определим только на границе серии — внутри серии он требует знания
следующего бара, которого в чистом `Decide` нет. Поэтому EOD-закрытие делаем по
фиксированному порогу минуты дня `DayEndMin` (дефолт 1120 = 18:40 МСК, конец основной
сессии МосБиржи): при `CloseAtDayEnd=1` первый бар с минутой `≥ DayEndMin` закрывает
позицию. Без look-ahead и без знания следующего бара. `DayEndMin` — параметр (см. §4).

### 5.3. `Lookback`

```
max(ConsolBars + 1, ATRPeriod + 1) + margin(5)
```

(HTF-EMA прогревается по отдельной `HTFCloses`-серии — на длину основного окна не влияет;
`htfOK` гейтит фильтр при недостатке HTF-данных.)

## 6. Подключение в `cmd/backtest`

- `-strategy daylow` → `svc.DayLowLookupOrGeneric(ticker)`.
- Новый флаг `-htf-interval` (дефолт `Hour1`), парсится тем же `parseInterval`.
- Для `daylow`: грузить `htfCandles` по `-htf-interval` (год lead-in), `cfg.HTFInterval`
  = длительность этого интервала. Дневные грузятся как обычно.
- `Trace`/`-explain`: стратегия реализует опциональный `Explain(md)` — по-гейтовый
  вердикт (почему не вошли на баре), как у прочих стратегий.

## 7. Тестирование (TDD)

Table-driven на чистую логику:

- сессионное окно: до/внутри/после, пустой `Times`;
- зона консолидации: тесная/широкая относительно ATR;
- близость к дневному минимуму: в допуске/вне;
- импульс: зелёная/красная, тело больше/меньше порога, пробой/не-пробой зоны;
- объём: выше/ниже `VolMult × avg`;
- фильтр H1: выше/ниже EMA, не прогрет;
- уровни: `StopLoss`/`TakeProfit` считаются точно; приоритет SL над TP в одном баре;
- `CloseAtDayEnd`: закрытие после 18:40, отсутствие закрытия при 0.

Интеграция: синтетическая серия свечей с заложенным сетапом прогоняется через
`backtest.Run` — проверяется один round-trip с ожидаемыми ценами входа/выхода.

## 8. Валидация (после реализации)

Реальный прогон с калибровкой и **честным OOS**:

```
go run ./cmd/backtest -ticker <T> -strategy daylow -interval Minutes5 -htf-interval Hour1 \
  -calibrate data/params/<t>/daylow_grid.json -out ./reports/<T> \
  -months 12 -test-months 3 -min-trades 20 -metric profit_factor
```

Критерий как у прочих: **если pooled OOS PF не держит планку — в live не тащим.**
Несколько backtest-стратегий (smc, momentum, levels, ORB) уже провалили walk-forward;
эта проходит тот же барьер честности. Ограничение по данным: глубина истории 5-мин
свечей у Tinkoff ограничена — провайдер чанкует по дням, прогон медленнее; фактическую
доступную глубину проверяем на первом реальном тикере.

## 9. Осознанно отложено (YAGNI)

- Пер-тикерные конфиг-пакеты `daylow/strategy/<ticker>` — до успешной калибровки.
- Live-слой (runner/scheduler/notification) — только после подтверждённого OOS-edge.
- Cooldown после стоп-аута / запрет повторного входа в ту же зону — добавим, если
  калибровка покажет частые немедленные ре-входы.
- Двойное окно активности (утро + предзакрытие) — пока одно непрерывное окно; расширим при необходимости.
```
