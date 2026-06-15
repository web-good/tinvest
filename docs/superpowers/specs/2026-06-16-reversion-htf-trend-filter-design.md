# Дизайн: HTF-фильтр тренда (4H) для входа стратегии reversion

Дата: 2026-06-16
Ветка: `feat/reversion-rsi-dip`
Статус: дизайн согласован, ожидает ревью спека

## Проблема

Бэктест-анализ 5 тикеров (AFKS, RUAL, MDMG, PLZL, NVTK, Hour1, 12 мес. OOS)
показал: **вся просадка стратегии приходит из защитного выхода**, который
срабатывает после входа в продолжающееся падение («падающий нож»):

- Где включён ATR-стоп (`UseATRStop=1`): выход `ATRSL` — 100% убыточных сделок
  (50/50), доминирующий отрицательный бакет (RUAL −16397, NVTK −17566,
  MDMG −6619, PLZL −2355).
- Где ATR-стоп выключен (AFKS): ту же роль играет выход `RSIOS` — 17/19 убыточных,
  −11848.
- Прибыльную сторону несут выходы `OB` (24 сделки, 0 убыточных) и `RSI50`.
  Тикер выходит в плюс только если эти выигрыши перевешивают убытки от стопов.

Стопы срабатывают быстро (в среднем 4.0–4.5 бара против ~7 у выигрышных) — вход
происходит в момент ухода осцилляторов в перепроданность, а цена продолжает
падать. Просадка кластеризуется в нисходящих фазах рынка (окт–ноя 2025;
апр–июн 2026), когда существующий hour1-фильтр тренда (`UseTrend`: быстрая EMA >
медленной + цена > медленной на том же hour1) показывал «вверх», тогда как
старший тренд был вниз.

## Цель

Добавить **опциональный фильтр старшего тренда на 4-часовом таймфрейме (HTF)**,
который блокирует вход, когда 4H-тренд направлен вниз. Это адресует корневую
причину стоп-аутов: не покупать провалы внутри настоящего нисходящего тренда.

Не-цели (вне рамок этого изменения):
- Логика выходов (`manage`) не меняется.
- Подтверждение разворота, фильтр волатильности/локации — отдельные направления,
  не входят сюда.
- Дневной HTF-путь и его потребители (scalping/momentum/levels) не затрагиваются.

## Решения по дизайну

### 1. Новый параметр

В `core.Params` добавляется:

```go
HTFTrendEMA int // период EMA на 4H для фильтра старшего тренда; 0 = выкл
```

- Тип `int` (как все прочие knobs) → попадает в reflection-грид калибровки
  автоматически (грид применяет значения через `FieldByName`).
- `0` отключает gate — это значение по умолчанию во всех `DefaultParams`, поэтому
  текущее поведение тикеров сохраняется до повторной калибровки.

### 2. Источник данных — реальные Hour4-свечи через движок (Маршрут 1)

`enum.Hour4` уже поддержан провайдером. Мы грузим настоящие биржевые 4H-свечи
(не агрегируем hour1 в стратегии) и прокидываем их в движок отдельной серией,
**не затрагивая** существующие дневные поля.

Изменения инфраструктуры:

- **MarketData** (`internal/service/trading_strategy/scalping/strategy/strategy.go`):
  новые поля `HTFCloses []float64`, `HTFHighs []float64`, `HTFLows []float64`
  — oldest-first закрытия/хаи/лои завершённых 4H-баров, выровненные так, что
  последний элемент — самый свежий 4H-бар, полностью закрытый к моменту текущего
  hour1-бара. Пустые, если HTF-данные не поданы.

- **Движок** (`internal/domain/backtest/engine.go`):
  - Новая функция `visibleCompletedHTF(htf []Candle, cur time.Time, interval time.Duration)`
    с правилом no-lookahead: 4H-бар видим, когда `c.Time.Add(interval) <= cur`
    (бар полностью закрылся к времени текущего hour1-бара). Возвращает три
    index-выровненные серии (closes/highs/lows), по аналогии с
    `visibleDailyCloses` / `visibleDailyHighsLows`.
  - `Run` и `Trace` получают дополнительную серию `htfCandles []Candle`; на каждом
    баре заполняют `md.HTFCloses/HTFHighs/HTFLows` через `visibleCompletedHTF`
    с `interval = 4h`. Дневные поля заполняются как прежде.

- **Калибровка** (`internal/service/backtest/calibrate.go` — `RunPhases`, и
  `split.go` — `SplitByTime` для HTF-серии при walk-forward) и **basket-путь**
  прокидывают `htfCandles` так же, как `dailyCandles`.

- **Загрузчик** (`cmd/backtest/main.go`): для `-strategy reversion` дополнительно
  грузим `enum.Hour4` с тем же годовым лид-ином, что и дневка (warm-up 4H-EMA), и
  передаём как `htfCandles`. Для остальных стратегий `htfCandles = nil`
  (HTF-поля остаются пустыми — поведение не меняется).

Замечание по churn: добавление позиционного параметра в `Run`/`Trace`/`RunPhases`
затрагивает все их вызовы и тесты (scalping/levels/momentum передают `nil`). Это
механическая правка; альтернатива со скрытым состоянием отвергнута ради явности.

### 3. Логика gate

В `buildInput` (когда `HTFTrendEMA > 0` и `len(md.HTFCloses) >= HTFTrendEMA`):

```
htfEMA   = EMA(md.HTFCloses, HTFTrendEMA)  // последнее значение
htfClose = md.HTFCloses[last]
htfOK    = true
```

Значения прокидываются в `decideInput` как `htfClose`, `htfEMA`, `htfOK` — по той
же дисциплине warm-up-флагов, что `rsiOK`/`stochOK`/`emaOK` (warm-up-ноль не
считается валидным чтением).

Правило восходящего HTF-тренда:

```go
func htfUptrend(in decideInput) bool {
    return in.htfOK && in.htfClose > in.htfEMA
}
```

В `decide` gate вставляется **шагом 0, перед** существующим hour1-фильтром
`UseTrend` (отдельный независимый gate; `UseTrend` и фильтр объёма остаются как
есть):

```go
// 0. Optional higher-timeframe (4H) trend filter.
if s.p.HTFTrendEMA > 0 && !htfUptrend(in) {
    return sig // нет покупки
}
```

### 4. Поведение при нехватке HTF-данных — блокировать вход

Если `HTFTrendEMA > 0`, но 4H-данных недостаточно для EMA (`htfOK == false`),
вход **блокируется** (старший тренд не подтверждён → не покупаем). Обоснование:
это защитный фильтр, и безопаснее не торговать при невозможности подтвердить HTF
(прецедент — `scalping/strategy/adaptive/adaptive.go`, где `trendUp` остаётся
`false` при нехватке дневных данных и вход блокируется). В бэктесте это не
проявляется: 4H-EMA всегда прогрета годовым лид-ином. Влияет только на live/edge.

Это отличается от фильтра объёма (тот пропускает gate при отсутствии данных), и
отличие намеренно: объём — не-защитный фильтр, HTF-тренд — защитный.

### 5. Прозрачность

- `entryReason` дополняется HTF-частью (период, значения close/EMA, вердикт),
  чтобы журнал сделок объяснял вход.
- `Explain` получает gate шагом 0 (✓ pass / ✗ block) с фактическими значениями.

### 6. Калибровка

В каждый `data/params/<ticker>/reversion_grid.json` добавляется свип
`"HTFTrendEMA": [0, 20, 50, 100]` (4H-периоды: 0=выкл, ~20≈3.3 дня, ~50≈8 дней,
~100≈17 дней). Калибровка сама решит per-ticker, помогает ли фильтр и какой
период. `DefaultParams` остаются `0` (фильтр выключен до калибровки).

### 7. Тесты

Table-driven, в `core_test.go` (и engine_test.go для visibility):

- `visibleCompletedHTF`: правило no-lookahead — текущий формирующийся 4H-бар
  невидим; полностью закрытые видимы; выравнивание closes/highs/lows.
- gate выключен при `HTFTrendEMA=0` (поведение неизменно).
- блок входа при `htfClose < htfEMA`, несмотря на пройденные hour1-условия.
- пропуск (покупка) при `htfClose > htfEMA` и прочих пройденных условиях.
- блок входа при нехватке HTF-данных (`htfOK=false`) и `HTFTrendEMA>0`.
- gate не влияет на выходы (путь `manage` не меняется).
- `buildInput`: HTF-EMA считается только при `HTFTrendEMA>0`; warm-up-дисциплина.

### 8. Валидация

- Перепрогнать ту же калибровку per-ticker:
  `go run ./cmd/backtest -ticker <T> -strategy reversion -calibrate
  data/params/<t>/reversion_grid.json -out ./reports/<T> -months 50
  -test-months 12 -metric net_pnl`.
- Сравнить до/после: `net_pnl`, и особенно убыточные бакеты `ATRSL`/`RSIOS`
  (ожидаем сокращение числа и суммы стоп-аутов).
- Walk-forward уже встроен (`-test-months 12`) — следить, чтобы улучшение
  держалось на OOS, а не было переподгонкой (ср. с уроком momentum).

## Затронутые компоненты (сводка)

| Компонент | Изменение |
|---|---|
| `core.Params` | + поле `HTFTrendEMA int` |
| `core.decideInput` | + `htfClose`, `htfEMA`, `htfOK` |
| `core.buildInput` | расчёт 4H-EMA при включённом gate |
| `core.decide` | gate шаг 0 перед `UseTrend` |
| `core.entryReason` / `Explain` | HTF-часть в выводе |
| `MarketData` (scalping/strategy) | + `HTFCloses/HTFHighs/HTFLows` |
| `backtest.engine` | `visibleCompletedHTF`; `Run`/`Trace` + HTF-серия |
| `service/backtest` | `RunPhases`/`SplitByTime` + HTF-серия |
| `cmd/backtest` | загрузка Hour4 для reversion; basket-путь |
| `data/params/*/reversion_grid.json` | свип `HTFTrendEMA` |
| `core_test.go`, `engine_test.go` | новые тесты |

Реестр reversion (`reversion_registry.go`) **не меняется** — `ParseParams`
анмаршалит в `core.Params`, новое поле подхватывается автоматически.
