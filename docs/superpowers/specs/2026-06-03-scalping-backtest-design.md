# Бэктест и калибровка скальпинг-стратегий — дизайн

**Дата:** 2026-06-03
**Статус:** утверждён
**Ветка:** `feat/per-share-scalping-strategies`

## Цель

Отдельный исполняемый файл (`cmd/backtest`), который прогоняет по-акционную скальпинг-стратегию
(`internal/service/trading_strategy/scalping/strategy/...`) на исторических часовых свечах за
указанный период, симулирует торговлю на мок-портфеле и пишет отчёт с полным журналом сделок,
движением капитала и качественными метриками. Поддерживает ручной прогон одного набора `Params`
и автоматический перебор по сетке (grid search) для калибровки параметров.

## Принципы

- Движок реплея, портфель, метрики и рендер отчёта — **чистые** (без I/O), полностью тестируемые
  таблицей. Весь I/O (gRPC, файлы, флаги) изолирован в `cmd/backtest` и `internal/service/backtest`.
- Симулятор повторяет **ровно путь живого раннера** (`scalping/trade.go`): на каждом баре вызывает
  `Decide`, действует только по `Buy`/`Sell`, исполняет по `close` текущего бара. Никаких
  интрабар-стопов и допущений о порядке касаний — `SL/TRAIL/TP` срабатывают на закрытии бара, когда
  их вернёт `Decide`.
- Движок прогона общий для любой `strategy.Strategy`. Калибровка (перебор `Params`) пока
  специфична для RUSAL, через реестр `ticker → (DefaultParams, фабрика(Params))`.
- DRY / YAGNI: переиспользуем существующие `strategy.Strategy`, `strategy.MarketData`,
  `model.Signal`, `enum.Interval`, `utils.CombinePrice`, gRPC-клиент и конфиг.

## Расположение кода

| Слой | Путь | Ответственность | I/O |
|---|---|---|---|
| CLI | `cmd/backtest/main.go` | парсинг флагов, сборка config+gRPC, оркестрация | да |
| Источник свечей | `internal/service/backtest/candles.go` | кэш-файл, дозагрузка из Tinkoff кусками, merge, срез окна | да |
| Реестр/калибровка | `internal/service/backtest/registry.go`, `calibrate.go` | `ticker→фабрика(Params)`, перебор сетки | нет |
| Движок | `internal/domain/backtest/engine.go` | реплей бар-за-баром, вызов `Decide`, применение сигналов | нет |
| Портфель | `internal/domain/backtest/portfolio.go` | мок-кэш, позиция, исполнение с долей+комиссией+лотами | нет |
| Метрики | `internal/domain/backtest/metrics.go` | расчёт метрик из сделок и equity-кривой | нет |
| Отчёт | `internal/domain/backtest/report.go` | рендер Markdown + CSV сделок (возвращает строки) | нет |
| Типы | `internal/domain/backtest/types.go` | `Candle`, `Trade`, `Config`, `Result`, `Metrics` | нет |

Существующая заглушка `internal/domain/backtest/order_line.go` (`OrderLine struct{}`) **удаляется** —
её заменяют реальные типы.

Импорты: `internal/domain/backtest` импортирует `scalping/strategy` и `scalping/model` (как и живой
раннер) — цикла импортов нет.

## Доменные типы (`internal/domain/backtest/types.go`)

```go
// Candle is one OHLCV bar (oldest-first in series).
type Candle struct {
    Time   time.Time
    Open   float64
    High   float64
    Low    float64
    Close  float64
    Volume int64
}

// Config controls the mock portfolio and fills.
type Config struct {
    InitialCash float64 // starting mock cash
    Fraction    float64 // fraction of current cash deployed per Buy (1.0 = all-in)
    Commission  float64 // commission as a fraction of turnover (e.g. 0.0005)
    Lot         int32   // share lot size (orders are whole lots)
}

// Trade is one completed round-trip (entry -> exit).
type Trade struct {
    EntryTime  time.Time
    EntryPrice float64
    ExitTime   time.Time
    ExitPrice  float64
    Quantity   int64   // shares (lots * Lot)
    Reason     string  // exit reason: "SL" / "TRAIL" / "TP"
    PnL        float64 // net of commission, in currency
    PnLPct     float64 // PnL relative to entry cost
    BarsHeld   int
}

// EquityPoint is portfolio value at one bar.
type EquityPoint struct {
    Time   time.Time
    Equity float64 // cash + position * close
}

// Result is the raw outcome of a single backtest run.
type Result struct {
    Trades      []Trade
    Equity      []EquityPoint
    InitialCash float64
    FinalEquity float64
}
```

## Поток данных

```
flags (cmd/backtest)
   ↓ ticker → Shares() → ID, Lot
CandleProvider.Load(ticker, ID, interval, from, to)
   ├─ файл есть → читаем; дотягиваем хвост (last cached → now); merge+dedup; сохраняем
   └─ файла нет → тянем весь [from,to] кусками; сохраняем
   → срез []Candle за [from, to] (oldest-first)
   ↓
Engine.Run(strategy, candles, cfg) → Result
   ↓
Metrics.Compute(result) → Metrics
   ↓
Report.Render(...) → Markdown + trades CSV → reports/
```

В режиме `-calibrate`: тот же `Engine.Run` гоняется для каждой комбинации сетки;
результаты ранжируются по `-metric`; пишется сводный отчёт калибровки.

## Кэш свечей (`internal/service/backtest/candles.go`)

Один файл на пару (тикер, интервал): `data/candles/<TICKER>_<Interval>.json` — JSON-массив `Candle`,
отсортированный oldest-first.

Алгоритм `Load(ticker, instrumentID, interval, from, to)`:
1. Если файла нет — `fetchRange(from, to)` целиком, сохранить, вернуть срез `[from,to]`.
2. Если файл есть — прочитать. Если `last.Time < to` — `fetchRange(last.Time, to)` (дотяжка хвоста),
   смержить с дедупликацией по `Time`, отсортировать, сохранить. Вернуть срез `[from, to]`.
3. Флаг `-refresh` — игнорировать кэш, перетянуть весь `[from,to]`, перезаписать файл.

`fetchRange(from, to)` тянет данные кусками, т.к. Tinkoff `GetCandles` ограничивает диапазон за один
запрос для H1. Идём окнами по `chunk` (константа в пакете, напр. 30 дней) от `from` к `to`, для
каждого окна вызываем `marketDataClient.GetCandles(ctx, &id, interval.ToNumberInvestApi(),
timestamppb.New(winFrom), timestamppb.New(winTo), &limit, true)`, конвертируем
`[]*model.CandleItemTechAnalyse` → `[]Candle` через `utils.CombinePrice`, склеиваем, дедуп по `Time`.
Между запросами пауза (как `time.Sleep(300ms)` в раннере), чтобы не упереться в лимиты API.
Берём только свечи с `IsComplete == true` (незакрытый последний бар отбрасывается).

Каталог `data/candles/` создаётся при необходимости и добавляется в `.gitignore`.

## Движок реплея (`internal/domain/backtest/engine.go`)

```go
func Run(s strategy.Strategy, candles []Candle, cfg Config) Result
```

- `L := s.Lookback()`. Если `len(candles) < L` — вернуть пустой `Result` (нет данных для прогрева).
- Завести `portfolio` с `cfg.InitialCash`.
- Цикл `i := L-1 .. len(candles)-1`:
  - `window := candles[i-L+1 : i+1]`;
  - построить `strategy.MarketData{Highs, Lows, Closes, Volumes}` из окна (oldest-first),
    `Price = window[last].Close`, `Position = portfolio.StrategyPosition()` (nil, если плоско);
  - `sig := s.Decide(md)`;
  - `switch sig.Kind`:
    - `SignalBuy` и нет позиции → `portfolio.Open(candles[i].Close, candles[i].Time)`;
    - `SignalSell` и есть позиция → `trade := portfolio.Close(candles[i].Close, candles[i].Time, sig.Reason)`,
      `result.Trades = append(...)`;
    - иначе — игнор (Buy в позиции / Sell без позиции — как в раннере);
  - записать `EquityPoint{candles[i].Time, portfolio.Equity(candles[i].Close)}`.
- В конце: если осталась открытая позиция — **не** закрываем принудительно; считаем `FinalEquity`
  по последнему close (открытая позиция оценивается mark-to-market, отдельная пометка в отчёте).

Построение `MarketData` повторяет `buildMarketData` из `scalping/trade.go` (срезы Highs/Lows/Closes/Volumes).

## Портфель (`internal/domain/backtest/portfolio.go`)

Состояние: `cash float64`, `qty int64`, `entryPrice float64`, `entryTime time.Time`, `entryBar int`,
`cfg Config`, счётчик баров.

- `Open(price, time)`:
  - бюджет = `cfg.Fraction * cash`;
  - число лотов = `floor(бюджет / (price * cfg.Lot * (1 + cfg.Commission)))`; если `0` — вход не
    совершается (недостаточно кэша на лот);
  - `qty = лоты * cfg.Lot`; стоимость = `qty * price`; комиссия = `стоимость * cfg.Commission`;
  - `cash -= стоимость + комиссия`; запомнить `entryPrice/entryTime/entryBar`.
- `Close(price, time, reason) Trade`:
  - выручка = `qty * price`; комиссия = `выручка * cfg.Commission`; `cash += выручка - комиссия`;
  - `entryCost = qty*entryPrice * (1+commission)` (вход с комиссией);
  - `PnL = (выручка - комиссия) - entryCost`; `PnLPct = PnL / entryCost`;
  - `BarsHeld = текущийБар - entryBar`; сбросить позицию; вернуть `Trade`.
- `StrategyPosition() *strategy.Position` — `nil`, если `qty==0`, иначе `{PurchasePrice: entryPrice, Quantity: qty}`.
- `Equity(price) float64` — `cash + qty*price`.

Лот берётся из `Shares()` (`model.Share.Lot`) и кладётся в `cfg.Lot`.

## Метрики (`internal/domain/backtest/metrics.go`)

```go
type Metrics struct {
    TotalTrades   int
    Wins, Losses  int
    WinRate       float64 // Wins/TotalTrades
    LossRate      float64
    GrossProfit   float64 // сумма положительных PnL
    GrossLoss     float64 // сумма |отрицательных PnL|
    ProfitFactor  float64 // GrossProfit/GrossLoss (0, если GrossLoss==0 и нет прибыли; +Inf-защита: при GrossLoss==0 и GrossProfit>0 → GrossProfit)
    NetPnL        float64 // FinalEquity - InitialCash
    NetPnLPct     float64
    MaxDrawdown   float64 // абсолютная, по equity-кривой
    MaxDrawdownPct float64
    AvgWin        float64
    AvgLoss       float64
    Expectancy    float64 // средний PnL на сделку
    BestTrade     float64
    WorstTrade    float64
    ExposurePct   float64 // доля баров в рынке
    CAGR          float64 // годовая доходность от длительности прогона
}

func Compute(r Result, barsInMarket, totalBars int, periodDays float64) Metrics
```

`MaxDrawdown` — по equity-кривой: для каждой точки трекаем бегущий максимум, просадка = `peak-equity`;
берём максимум просадки (и её % от peak). `ProfitFactor` при нулевом `GrossLoss` и положительной
прибыли = `GrossProfit` (не делим на ноль). `CAGR = (FinalEquity/InitialCash)^(365/periodDays) - 1`
при `periodDays>0` и `InitialCash>0`, иначе `0`.

## Отчёт (`internal/domain/backtest/report.go`)

Чистые функции, возвращающие строки; запись на диск — в `cmd/backtest`.

- `RenderMarkdown(meta, metrics, trades, equity) string` — секции:
  - **Заголовок прогона**: тикер, интервал, период (from–to), стартовый кэш, fraction, commission,
    использованные `Params` (таблица поле→значение), флаг открытой позиции на конце.
  - **Сводка метрик**: таблица (всего сделок, % выигрышных/проигрышных, gross profit/loss,
    profit factor, чистый PnL ₽ и %, стартовый/финальный капитал, макс. просадка ₽ и %,
    средняя прибыль/убыток, expectancy, лучшая/худшая сделка, exposure %, CAGR).
  - **Журнал сделок**: таблица № | вход (время, цена) | выход (время, цена) | причина | баров | PnL ₽ | PnL %.
  - **Движение капитала**: компактная сводка equity (старт, мин, макс, финал) — полная кривая в CSV.
- `RenderTradesCSV(trades) string` — CSV-журнал сделок для дальнейшего анализа.
- `RenderEquityCSV(equity) string` — CSV equity-кривой (время, equity) для построения графика.

Файлы пишутся в `-out` (дефолт `reports/`):
`<TICKER>_<Interval>_<timestamp>.md`, `..._trades.csv`, `..._equity.csv`.
Каталог `reports/` добавляется в `.gitignore`.

## Реестр и калибровка (`internal/service/backtest/registry.go`, `calibrate.go`)

`registry.go`:
```go
type Binding struct {
    DefaultParams func() any                         // rusal.DefaultParams()
    Build         func(params any) strategy.Strategy // rusal.NewWithParams(p)
    ParseParams   func(raw []byte) (any, error)      // json -> rusal.Params
}
var registry = map[string]Binding{ "RUAL": {...} }
func Lookup(ticker string) (Binding, bool)
```

Реестр пока содержит только RUAL. Движок прогона (`Engine.Run`) при этом общий — принимает любую
`strategy.Strategy`.

Ручной прогон: `-params params.json` (JSON c полями `rusal.Params`) → `ParseParams` → `Build` →
`Engine.Run`. Без `-params` — `DefaultParams`.

`calibrate.go` (режим `-calibrate grid.json`):
```go
func RunGrid(b Binding, grid Grid, candles []Candle, cfg Config, metric string) []CalibResult
```
- `grid.json` — карта `имя поля Params → []значений`. Варьируются только перечисленные поля,
  остальные = `DefaultParams`.
- Строим декартово произведение комбинаций; для каждой — `Build(params)` → `Engine.Run` →
  `Metrics.Compute`; собираем `CalibResult{Params, Metrics}`.
- Ранжируем по `metric` (см. ниже), пишем сводный отчёт: таблица топ-N комбинаций с ключевыми
  метриками + полный отчёт лучшей комбинации.
- Применение `grid` к `Params` — рефлексия по именам полей (RUSAL-specific биндинг знает тип
  `rusal.Params`). Неизвестное имя поля → ошибка с понятным сообщением.

Файл калибровки: `reports/<TICKER>_calibration_<timestamp>.md`.

## CLI-флаги (`cmd/backtest/main.go`)

```
-ticker RUAL          (обязательный) тикер; ищется через Shares() → ID, Lot
-months 12            период прогона в месяцах (from = now-N мес, to = now)
-cash 100000          стартовый кэш мок-портфеля
-fraction 1.0         доля кэша на вход (1.0 = all-in)
-commission 0.0005    комиссия как доля оборота (дефолт ~0.05%)
-params params.json   путь к JSON c Params (по умолчанию DefaultParams)
-calibrate grid.json  режим перебора по сетке (взаимоисключающ с -params)
-metric profit_factor метрика ранжирования при калибровке
-out reports/         каталог отчётов
-refresh              форсировать перезагрузку свечей (игнор кэша)
```

Метрика ранжирования (`-metric`), поддерживаемые значения:
`profit_factor` (дефолт), `net_pnl`, `win_rate`, `max_drawdown` (меньше — лучше), `expectancy`.
Неизвестное значение → ошибка при старте.

Проводка зависимостей в `main.go`:
1. Загрузить env (`./env/local.env`; токен `T_BANK`) — повторяя подход `app.initConfig` (godotenv),
   но компактно для CLI; без полного `confita`-лоадера достаточно прочитать `T_BANK` и адрес.
2. `grpc.NewClientGrpc("invest-public-api.tinkoff.ru:443", token)`.
3. `Shares(ctx)` → найти `Share` по `Ticker == -ticker` → `ID`, `Lot`. Нет — ошибка.
4. `CandleProvider.Load(...)` → `[]Candle`.
5. Ветка: одиночный прогон или `-calibrate`.
6. Записать отчёты в `-out`.

## Обработка ошибок

- Нет `T_BANK` / адреса — фатально с понятным сообщением.
- Тикер не найден в `Shares()` или `Trading==false` — фатально.
- `len(candles) < Lookback()` — предупреждение + пустой отчёт (метрики нулевые, 0 сделок).
- Сбой одного chunk при загрузке — лог + продолжаем (частичные данные), либо фатально при полном
  отсутствии данных за период.
- Неизвестное поле в `grid.json` / неизвестная `-metric` — фатально на старте.

## Тестирование

Чистое ядро (`internal/domain/backtest`) — табличные тесты на синтетических свечах:
- `engine_test.go`: вход по `Buy`/выход по `Sell` на заданных барах; игнор Buy-в-позиции и
  Sell-без-позиции; недостаток истории (`< Lookback`) → пустой результат; открытая позиция в конце.
- `portfolio_test.go`: размер по `fraction`, целые лоты, списание/начисление комиссии, `PnL`/`PnLPct`,
  `Equity`, отказ от входа при нехватке кэша на лот.
- `metrics_test.go`: win rate, gross profit/loss, profit factor (включая `GrossLoss==0`),
  max drawdown на известной кривой, expectancy, exposure, CAGR.
- `report_test.go`: рендер содержит ключевые секции/заголовки (smoke).

`internal/service/backtest`:
- `candles_test.go`: merge+dedup по `Time`, срез окна `[from,to]`, поведение при отсутствии файла
  (на временном каталоге `t.TempDir()`); gRPC-клиент мокается интерфейсом.
- `calibrate_test.go`: декартово произведение сетки, применение полей через рефлексию,
  ранжирование по метрике, ошибка на неизвестном поле.

Стратегия RUSAL и индикаторы уже покрыты — не дублируем.

## Что НЕ входит (YAGNI)

- Короткие позиции, плечо, несколько одновременных позиций.
- Интрабар-исполнение SL/TP по High/Low (осознанно: живая система не ставит стоп-ордера).
- Оптимизаторы умнее grid search (genetic, walk-forward) — возможный follow-up.
- Графики (только CSV equity для внешнего построения).
- Калибровка не-RUSAL стратегий (биндинг добавляется по мере появления стратегий).
```
