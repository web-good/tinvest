# Rolling walk-forward для cmd/backtest — дизайн

Дата: 2026-06-16

## Проблема

Текущий бэктестер поддерживает только **одиночный train/test holdout** через флаг
`-test-months`: грид-калибровка гоняется на ранней части истории, а единственный
победитель прогоняется на последних N месяцах (`cmd/backtest/main.go:248-298`).
Это даёт одну OOS-оценку на одном окне — она чувствительна к тому, на какой
рыночный режим пришёлся последний отрезок, и не показывает, устойчивы ли подобранные
параметры во времени.

Настоящего скользящего walk-forward (несколько окон с переподбором параметров на
каждом шаге и склейкой OOS-результатов) в коде нет, хотя help-текст флага и
basket-отчёт его так называют.

Цель: добавить rolling walk-forward для одного тикера, чтобы отделить настоящий edge
от in-sample подгонки.

## Решения (зафиксированы при брейншторминге)

1. **Схема окон — rolling**: train фиксированной длины, train и OOS-окно скользят
   вперёд вместе.
2. **Параметризация — минимум флагов**: новый `-train-months`; шаг равен `-test-months`
   (OOS-фолды идут встык, без перекрытия); обратная совместимость сохраняется.
3. **Отчёт — три блока**: pooled OOS + таблица по фолдам + стабильность параметров.
   Scope: один тикер через путь `-calibrate` (корзину не трогаем).
4. **Прогрев — lead-in + фильтр сделок**: OOS-прогон получает train-часть для прогрева
   индикаторов, но в OOS-метрики идут только сделки со входом ≥ начала OOS-окна.

## Алгоритм окон

- `to = now`, `from = to − months`, шаг `step = testMonths`.
- Фолд `k` (k = 0, 1, 2, …):
  - `trainFrom = from + k·step`
  - `trainTo   = trainFrom + trainMonths`
  - `testFrom  = trainTo`
  - `testTo    = testFrom + testMonths`
- Фолды генерируются, пока `testTo ≤ to`.
- Число фолдов = `⌊(months − trainMonths − testMonths) / testMonths⌋ + 1`.
- Границы окон считаются в месяцах через `time.Time.AddDate(0, ±m, 0)` от `from`/`to`,
  как в существующем коде (`cmd/backtest/main.go:138, 249`).

Пример `-months 24 -train-months 12 -test-months 3` → 4 фолда:
`[0–12]→[12–15]`, `[3–15]→[15–18]`, `[6–18]→[18–21]`, `[9–21]→[21–24]`.

### Guard-условия

- Если ни одного полного фолда не помещается (`trainMonths + testMonths > months`) →
  ошибка с понятным текстом (сколько месяцев нужно).
- Если train-срез конкретного фолда короче `Lookback` стратегии → ошибка по образцу
  существующей проверки (`cmd/backtest/main.go:265-270`). Доминирующий lookback
  (например `SlowEMA`) в гриде не свипуется, поэтому дефолтные параметры
  репрезентативны для оценки.

## Прогрев индикаторов

- **Калибровка фолда**: `RunPhases` запускается на train-срезе `[trainFrom, trainTo)`.
- **OOS-прогон**: `domain.Run` запускается на срезе `[trainFrom, testTo)` — train-часть
  прогревает часовые индикаторы (EMA200 ≈ 200 баров ≈ ~месяц торговых часов). Затем из
  `res.Trades` оставляются только сделки со **временем входа ≥ testFrom**; именно они
  формируют OOS-метрики фолда и пул. Это эквивалентно живой торговле, где к началу
  периода индикаторы уже прогреты историей.
- `daily` и `htf` свечи передаются целиком (они уже грузятся с годовым lead-in,
  `cmd/backtest/main.go:146, 156`; стратегия выбирает их по времени бара, прогрев не
  теряется).
- **Принятый компромисс**: train-калибровка стартует «холодной» на `trainFrom` (часовой
  EMA200 не прогрет на первых ~200 барах train-окна). Это in-sample-оценка на большом
  окне (train ≫ warm-up), ранжирование комбо страдает одинаково для всех — приемлемо.
  Прогревать train отдельно не будем (усложнение `RunPhases` без значимой пользы).

## Компоненты

### `internal/service/backtest/walkforward.go` (новый файл)

```go
// WalkForwardFold — результат одного фолда.
type WalkForwardFold struct {
    Index          int
    TrainFrom      time.Time
    TrainTo        time.Time
    TestFrom       time.Time
    TestTo         time.Time
    InSampleMetric float64              // значение метрики ранжирования на train
    InSamplePF     float64              // PF лучшей комбо на train (для сравнения с OOS)
    OOS            backtest.Metrics     // trade-based метрики по OOS-сделкам фолда
    OOSNetPnLPct   float64              // сумма OOS-PnL / InitialCash
    OOSMaxDDPct    float64              // max drawdown по реплею OOS-сделок
    OOSTrades      int
    WinnerParams   any                  // параметры-победителя (для анализа стабильности)
    WinnerRows     []backtest.ParamLine // те же параметры в человекочитаемом виде
    Note           string               // напр. «0 OOS-сделок»
}

// WalkForwardSummary — агрегат по всем фолдам.
type WalkForwardSummary struct {
    Folds               []WalkForwardFold
    PooledOOS           backtest.Metrics // PooledMetrics по всем склеенным OOS-сделкам
    CompoundedReturnPct float64          // произведение (1 + OOSNetPnLPct_фолд) − 1
}

// RunWalkForward гоняет rolling walk-forward для одного тикера.
func RunWalkForward(
    b Binding, phases []Phase,
    candles, dailyCandles, htfCandles []backtest.Candle,
    cfg backtest.Config, metric string, minTrades int,
    from, to time.Time, trainMonths, testMonths int,
) (WalkForwardSummary, error)
```

Логика `RunWalkForward`:
1. Вычислить список фолдов по датам; если пусто → ошибка.
2. Для каждого фолда:
   - `trainSlice = sliceByRange(candles, trainFrom, trainTo)`; guard на `Lookback`.
   - `results = RunPhases(b, phases, trainSlice, dailyCandles, htfCandles, cfg, metric, minTrades, trainDays, nil)`.
   - если `len(results) == 0` → пометить фолд `Note`, пропустить OOS.
   - `best = results[0].Params`; зафиксировать `InSampleMetric`, `InSamplePF`.
   - `warmSlice = sliceByRange(candles, trainFrom, testTo)`.
   - `res = domain.Run(b.Build(best), warmSlice, dailyCandles, htfCandles, cfg)`.
   - `oosTrades = trades с Trade.EntryTime ≥ testFrom` (фильтр; поле подтверждено в
     `internal/domain/backtest/types.go:27`).
   - метрики фолда: `OOS = PooledMetrics(oosTrades)` (trade-based);
     `OOSNetPnLPct = Σ PnL / InitialCash`; `OOSMaxDDPct` — по реплею OOS-сделок
     (бегущий минимум кумулятивного PnL относительно пика).
   - накопить `oosTrades` в общий пул.
3. `PooledOOS = PooledMetrics(pool)`; `CompoundedReturnPct` — компаунд по фолдам.

Вспомогательное:

```go
// sliceByRange возвращает свечи в [from, to): старшая граница исключающая.
func sliceByRange(candles []backtest.Candle, from, to time.Time) []backtest.Candle
```
Реализуется двумя `SplitByTime`: `_, tail := SplitByTime(candles, from)`;
`head, _ := SplitByTime(tail, to)`.

Фильтр строится по `Trade.EntryTime` (`internal/domain/backtest/types.go:27`).

### `internal/service/backtest/walkforward.go` — рендер

```go
// RenderWalkForwardMarkdown рендерит pooled OOS + таблицу по фолдам + стабильность.
func RenderWalkForwardMarkdown(ticker, metric string, s WalkForwardSummary,
    trainMonths, testMonths int) string
```

Структура отчёта `<base>_walkforward.md`:

1. **Заголовок**: тикер, train/test месяцы, число фолдов, метрика ранжирования.
2. **Pooled OOS** (агрегат): Всего сделок, Выигрышных/проигрышных, Win rate,
   Profit factor, Gross profit/loss, Expectancy, Sortino, Лучшая/худшая сделка,
   **Compounded return %**. (По образцу блока «Пул сделок» в `RenderBasketMarkdown`,
   `basket.go:52-61`; equity-based поля в пуле — нули, это ожидаемо, см. `PooledMetrics`.)
3. **Таблица по фолдам**:
   `| # | Train-окно | Test-окно | In-sample PF | OOS PF | OOS сделок | OOS NetPnL% | OOS MaxDD% |`
   Пропущенные фолды (0 комбинаций / 0 сделок) показываются с прочерками и `Note`.
4. **Стабильность параметров**: для каждого swept-кноба (объединение ключей из
   `WinnerRows` всех фолдов) — совпало ли значение во всех фолдах. Стабильные кнобы
   перечисляются одной строкой; «гулявшие» — с перечислением значений по фолдам
   (`knob: f1=.., f2=.., …`).

### `cmd/backtest/main.go` — проводка

- Новый флаг: `trainMonths = flag.Int("train-months", 0, "rolling walk-forward: fixed train window length in months; step = -test-months")`.
- Прокинуть `trainMonths` в `run(...)` и в `runCalibration(...)`.
- В `runCalibration`: если `trainMonths > 0`:
  - вызвать `svc.RunWalkForward(...)` с `from`/`to`;
  - записать `base + "_walkforward.md"` через `RenderWalkForwardMarkdown`;
  - напечатать сводку в stdout (число фолдов, pooled PF, compounded return);
  - вернуться (не писать `_best.md` / `_calibration.md` в этом режиме).
- Если `trainMonths == 0` — текущая ветка (одиночный holdout / полный прогон) без изменений.
- Валидация: `trainMonths > 0` без `-calibrate` → ошибка (walk-forward требует грид).

## Тесты (`internal/service/backtest/walkforward_test.go`)

Table-driven, по конвенциям `golang-testing`:

1. **Математика окон**: при разных `months/trainMonths/testMonths` — корректные число и
   границы фолдов; случай 0 фолдов → ошибка.
2. **Фильтрация сделок по времени входа**: синтетический `Run`-результат со сделками до и
   после `testFrom` — в OOS попадают только сделки со входом ≥ `testFrom` (lead-in не
   протекает в метрики).
3. **Пул-агрегация и compounded return**: несколько фолдов с известными PnL → корректные
   `PooledOOS` (PF/win/expectancy) и `CompoundedReturnPct`.
4. **Стабильность параметров**: фолды с совпадающими и расходящимися победителями →
   корректное разделение на стабильные/гулявшие кнобы.

`sliceByRange` тестируется на полузакрытость интервала `[from, to)`.

## Вне scope (опционально, отдельным коммитом)

Существующий одиночный `-test-months` holdout страдает тем же холодным стартом OOS
(`cmd/backtest/main.go:287` запускает `domain.Run` на `bestCandles`, начинающихся с
`boundary`). OOS-прогон с lead-in+фильтром можно вынести в общий helper и переключить на
него holdout-ветку. Это улучшение не блокирует основную фичу и помечено как
необязательное; делать только если не раздувает диф.

## Затрагиваемые файлы

- `internal/service/backtest/walkforward.go` — новый (логика + рендер).
- `internal/service/backtest/walkforward_test.go` — новый (тесты).
- `cmd/backtest/main.go` — флаг `-train-months`, проводка в `runCalibration`.
- (опц.) рефактор OOS-прогона в общий helper для holdout-ветки.

Переиспользуются без изменений: `SplitByTime`, `RunPhases`, `ParsePhases`,
`PooledMetrics`, `domain.Run`, `domain.Compute`, `ParamRows`.
