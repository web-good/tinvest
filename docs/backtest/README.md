# Бэктест и калибровка торговых стратегий

Документация по инструменту `cmd/backtest` и пакетам `internal/domain/backtest`
(чистое ядро) и `internal/service/backtest` (I/O-слой).

## Что это

`cmd/backtest` прогоняет торговую стратегию на исторических свечах за указанный
период, симулирует торговлю на мок-портфеле и пишет отчёт с журналом сделок,
движением капитала и метриками качества.

Стратегия выбирается флагом `-strategy`, таймфрейм — флагом `-interval`
(дефолт `Hour1`):

| `-strategy` | Движок | Пакет |
|---|---|---|
| `scalping` (дефолт) | RSI-скальпинг (ADX-режимы) | `internal/service/trading_strategy/scalping` |
| `levels` | Volume-Profile / ATR-room | `internal/service/trading_strategy/levels` |
| `momentum` | EMA200 + MACD + объём | `internal/service/trading_strategy/momentum` |
| `reversion` | mean-reversion RSI + Stochastic | `internal/service/trading_strategy/reversion` |

Поддерживаются режимы:
- **одиночный прогон** одного набора параметров (`-params` / `DefaultParams`);
- **калибровка** — перебор по сетке (`-calibrate`, grid search);
- **walk-forward** — калибровка с проверкой вне выборки: одиночный holdout
  (`-test-months`) или скользящий rolling (`-train-months`), см. ниже;
- **basket** — пул OOS-сделок по нескольким тикерам (`-basket`, только `momentum`);
- **explain** — диагностика одного бара (`-explain`): почему вошёл/не вошёл.

Ключевой принцип: **симулятор повторяет ровно путь живого раннера**. На каждом
баре он строит `MarketData` из окна свечей, вызывает `Decide`, действует только по
`Buy`/`Sell` и исполняет по `close` текущего бара. Никаких интрабар-стопов:
`SL`/`TRAIL`/`TP` и прочие выходы срабатывают на закрытии бара — тогда, когда их
вернёт сама стратегия. Это даёт результат, согласованный с тем, как система
торгует в проде (живая система не ставит стоп-ордера).

## Архитектура

| Слой | Путь | Ответственность | I/O |
|---|---|---|---|
| CLI | `cmd/backtest/main.go` | флаги, env+gRPC, оркестрация, запись файлов | да |
| Свечи | `internal/service/backtest/candles.go` | файл-кэш, чанковая дозагрузка, merge/срез | да |
| Реестр | `internal/service/backtest/registry.go` | `тикер → (DefaultParams, Build, ParseParams)` | нет |
| Калибровка | `internal/service/backtest/calibrate.go` | перебор сетки, рефлексия, ранжирование | нет |
| Движок | `internal/domain/backtest/engine.go` | реплей бар-за-баром, вызов `Decide` | нет |
| Портфель | `internal/domain/backtest/portfolio.go` | мок-кэш, лоты, комиссия, PnL | нет |
| Метрики | `internal/domain/backtest/metrics.go` | расчёт метрик из сделок и equity | нет |
| Отчёт | `internal/domain/backtest/report.go` | рендер Markdown + CSV (возвращает строки) | нет |

Доменный пакет (`internal/domain/backtest`) полностью чистый и покрыт табличными
тестами; весь gRPC/файловый I/O изолирован в `cmd/backtest` и
`internal/service/backtest`.

## Подготовка

Нужен токен Tinkoff Invest в переменной окружения `T_BANK`. CLI читает его из
(в порядке проверки) текущего окружения, `./env/local.env`, `./env/token.env`:

```bash
cp env/local.env.example env/local.env   # если ещё не сделано
# впишите T_BANK=... в env/local.env
```

Прогон требует сетевого доступа к `invest-public-api.tinkoff.ru:443`.

## Быстрый старт

Одиночный прогон RUAL за 12 месяцев на дефолтных (НЕ откалиброванных) параметрах
скальпинг-стратегии:

```bash
go run ./cmd/backtest -ticker RUAL -months 12
```

Reversion-стратегия на часовках:

```bash
go run ./cmd/backtest -ticker NVTK -strategy reversion -interval Hour1 -months 12 -out ./reports/NVTK
```

В консоль выводится сводка, а в каталог `-out` (дефолт `reports/`) пишутся файлы
(см. ниже).

## Флаги CLI

| Флаг | Дефолт | Назначение |
|---|---|---|
| `-ticker` | — | **обязательный** (кроме `-basket`). Тикер; ищется через `Shares()` → `ID`, `Lot`. RUSAL: тикер `RUAL` |
| `-strategy` | `scalping` | движок: `scalping` \| `levels` \| `momentum` \| `reversion` |
| `-interval` | `Hour1` | таймфрейм: `Minutes15` \| `Minutes30` \| `Hour1` \| `Hour4` \| `Day1` \| `Week1` |
| `-months` | `12` | период прогона в месяцах (`from = now - N мес`, `to = now`) |
| `-cash` | `100000` | стартовый кэш мок-портфеля |
| `-fraction` | `1.0` | доля кэша на вход (`1.0` = весь кэш) |
| `-commission` | `0.0005` | комиссия как доля оборота (≈0.05%) |
| `-params` | — | путь к JSON c `Params` (поверх `DefaultParams`; взаимоисключающ с `-calibrate`) |
| `-calibrate` | — | путь к grid JSON (режим перебора) |
| `-metric` | `expectancy` | метрика ранжирования при калибровке (см. таблицу ниже) |
| `-min-trades` | `15` | калибровка: комбинации с меньшим числом сделок тонут ниже квалифицированных |
| `-test-months` | `0` | одиночный holdout: калибровка на ранней части, отчёт лучшей комбо на последних N мес |
| `-train-months` | `0` | rolling walk-forward (вместе с `-calibrate`): фикс. длина train-окна в мес; шаг = `-test-months` |
| `-out` | `reports/` | каталог отчётов |
| `-refresh` | `false` | форсировать перезагрузку свечей (игнор кэша) |
| `-explain` | — | диагностика одного бара: MSK-время `'YYYY-MM-DD HH:MM'` |
| `-basket` | — | basket-режим: список тикеров через запятую (только `momentum`; игнорирует `-ticker`) |
| `-grid-dir` | `data/params` | basket: каталог с `<lower-ticker>/momentum_grid.json` |

## Одиночный прогон со своими параметрами

`Params` для RUSAL заданы в `rusal.DefaultParams()`. Чтобы переопределить часть
параметров, передайте JSON с нужными полями — остальные останутся дефолтными
(JSON применяется поверх `DefaultParams`):

`params.json`:
```json
{
  "EMAPeriod": 34,
  "SLMult": 1.5,
  "TrailMult": 3.0
}
```

```bash
go run ./cmd/backtest -ticker RUAL -months 18 -params params.json \
  -cash 200000 -fraction 0.5 -commission 0.0004
```

Полный список из 15 полей `Params` с дефолтами и смыслом — см.
[../scalping/settings.md](../scalping/settings.md).

## Калибровка (grid search)

Режим `-calibrate` перебирает декартово произведение значений по перечисленным
полям. В сетке указываются только варьируемые поля; остальные берутся из
`DefaultParams`.

`grid.json` (имя поля `Params` → список значений):
```json
{
  "EMAPeriod":      [12, 21, 34],
  "RSITrendLevel":  [40, 45, 50],
  "SLMult":         [1.0, 1.5, 2.0],
  "TrailMult":      [2.0, 2.5, 3.0]
}
```

```bash
go run ./cmd/backtest -ticker RUAL -months 24 -calibrate grid.json \
  -metric profit_factor
```

Это 3×3×3×3 = 81 комбинация. Для каждой движок прогоняется отдельно, считаются
метрики, и комбинации ранжируются по `-metric`. Значения в сетке — числа JSON
(`float64`); для целочисленных полей (`EMAPeriod`, `RSIPeriod`, …) они приводятся
к `int`. Неизвестное имя поля или неизвестная метрика → ошибка на старте.

### Метрики ранжирования (`-metric`)

| Значение | Смысл | Лучше |
|---|---|---|
| `expectancy` (дефолт) | средний PnL на сделку | больше |
| `profit_factor` | gross profit / gross loss | больше |
| `net_pnl` | чистый PnL за период | больше |
| `win_rate` | доля прибыльных сделок | больше |
| `sortino` | средний PnL / downside-отклонение PnL сделок | больше |
| `max_drawdown` | максимальная просадка капитала | **меньше** |

Гриды лежат по-тикерно: `data/params/<lower-ticker>/<strategy>_grid.json`
(например `data/params/nvtk/reversion_grid.json`). Поддерживается как плоский
формат (`{"Field":[...]}`), так и фазовый (`{"phases":[{"name","keepTop","grid"}]}`),
где каждая следующая фаза свипует свою под-сетку поверх топ-`keepTop` выживших
предыдущей.

## Проверка вне выборки (walk-forward)

Калибровка на всём периоде даёт **in-sample** результат — параметры подобраны на
тех же данных, на которых и оцениваются, поэтому метрики завышены (переобучение).
Чтобы оценить параметры честно, есть два режима out-of-sample (OOS).

### Одиночный holdout (`-test-months`)

Калибровка идёт на ранней части `[from, boundary)`, а лучшая комбинация
прогоняется на последних `N` месяцах `[boundary, to)`, которые грид не видел.
`_best.md` тогда показывает именно OOS-результат, а `_calibration.md` — in-sample
ранжирование (train-окно). Сравнение этих двух чисел и есть диагностика
переобучения.

```bash
go run ./cmd/backtest -ticker NVTK -strategy reversion \
  -calibrate data/params/nvtk/reversion_grid.json \
  -out ./reports/NVTK -months 24 -test-months 6 -min-trades 20 -metric profit_factor
```

Ограничение: окно одно — оценка чувствительна к тому, на какой рыночный режим
пришёлся последний отрезок. Train-окно должно быть длиннее `Lookback` стратегии,
иначе калибровка падает с ошибкой (для коротких историй уменьшите `-test-months`
или возьмите больше истории через `-months`).

### Скользящий rolling walk-forward (`-train-months`)

`-train-months N` (только вместе с `-calibrate`) включает скользящий walk-forward:
train-окно фиксированной длины `N` месяцев и OOS-окно длиной `-test-months`
скользят вперёд вместе, OOS-окна идут встык без перекрытия. На КАЖДОМ фолде
параметры подбираются заново на его train-окне и проверяются на следующем
OOS-окне. Это честнее одиночного holdout: несколько независимых OOS-окон + видно,
устойчивы ли параметры во времени.

- **Шаг** = `-test-months`. **Число фолдов** = `⌊(months − train − test) / test⌋ + 1`.
- **Прогрев индикаторов:** OOS-прогон каждого фолда получает свечи `[trainFrom, testTo)`
  (train-часть прогревает EMA200 и т.п.), но в OOS-метрики идут только сделки со
  **временем входа ≥ начала OOS-окна** — как в живой торговле.

Пример `-months 24 -train-months 12 -test-months 3` → 4 фолда:
`[0–12]→[12–15]`, `[3–15]→[15–18]`, `[6–18]→[18–21]`, `[9–21]→[21–24]`.

```bash
go run ./cmd/backtest -ticker NVTK -strategy reversion \
  -calibrate data/params/nvtk/reversion_grid.json \
  -out ./reports/NVTK -months 24 -train-months 12 -test-months 3 \
  -min-trades 10 -metric profit_factor
```

Пишется один файл `<...>_walkforward.md` с тремя блоками:
1. **Пул сделок (агрегат OOS)** — PF / win rate / expectancy / sortino / лучшая-худшая
   сделка по всем склеенным OOS-сделкам + **compounded return** (компаунд по фолдам).
2. **Результаты по фолдам** — даты train/test-окон, in-sample PF против OOS PF,
   число OOS-сделок, OOS NetPnL%, OOS MaxDD%.
3. **Стабильность параметров** — какие кнобы совпали во всех фолдах (стабильны), а
   какие гуляли (с перечислением значений по фолдам). Если грид на каждом фолде
   выбирает разные параметры — стратегия неустойчива/переобучена.

При `-train-months 0` (дефолт) поведение `-calibrate`/`-test-months` не меняется.

## Что попадает в отчёт

Файлы пишутся в `-out` (дефолт `reports/`) с временной меткой в имени
(`<TICKER>_<Interval>_<timestamp>`):

**Одиночный прогон:**

| Файл | Содержимое |
|---|---|
| `..._<timestamp>.md` | заголовок прогона, таблица `Params`, сводка метрик, журнал сделок, движение капитала |
| `..._<timestamp>_trades.csv` | журнал сделок построчно для внешнего анализа |
| `..._<timestamp>_equity.csv` | кривая капитала (`time,equity`) для построения графика |

**Калибровка** (и одиночный holdout `-test-months`):

| Файл | Содержимое |
|---|---|
| `..._<timestamp>_calibration.md` | таблица топ-20 комбинаций по выбранной метрике + параметры лучшей (in-sample / train-окно) |
| `..._<timestamp>_best.md` | полный отчёт прогона лучшей комбинации (при `-test-months` — на OOS-окне) |

**Rolling walk-forward** (`-train-months`):

| Файл | Содержимое |
|---|---|
| `..._<timestamp>_walkforward.md` | пул OOS-сделок + compounded return, таблица по фолдам (in-sample PF vs OOS PF), стабильность параметров |

### Метрики в сводке

`TotalTrades`, число выигрышных/проигрышных, `WinRate`/`LossRate`,
`GrossProfit`/`GrossLoss`, `ProfitFactor`, чистый `NetPnL` (₽ и %), стартовый и
финальный капитал, `MaxDrawdown` (₽ и %), средняя прибыль/убыток, `Expectancy`,
`Sortino`, лучшая/худшая сделка, `ExposurePct` (доля баров в рынке) и `CAGR`.

Особые случаи: при нулевом `GrossLoss` и положительной прибыли `ProfitFactor`
равен `GrossProfit` (не делим на ноль); `CAGR` считается только при положительной
длительности и капитале.

## Как симулируется торговля

**Движок** (`engine.go`):
- Берёт `L = strategy.Lookback()`. Если свечей меньше `L` — пустой отчёт
  (нет данных для прогрева; в консоль выводится предупреждение).
- Идёт по барам `i = L-1 .. конец`. На каждом баре строит окно `[i-L+1 .. i]`,
  собирает `MarketData`, подставляет текущую позицию, вызывает `Decide`.
- `Buy` при отсутствии позиции → открытие по `close[i]`. `Sell` при наличии
  позиции → закрытие по `close[i]` с записью сделки. `Buy` в позиции и `Sell` без
  позиции игнорируются (как в живом раннере).
- Открытая в конце прогона позиция **не** закрывается принудительно — она
  оценивается mark-to-market по последнему `close`, в отчёте ставится пометка.

**Портфель** (`portfolio.go`):
- Бюджет на вход = `fraction × cash`. Число лотов = `floor(бюджет / (цена × Lot ×
  (1 + commission)))`. Если не хватает даже на один лот — вход не совершается.
- Комиссия списывается и на входе, и на выходе. `PnL` — чистый после комиссий,
  `PnLPct` — относительно стоимости входа (с учётом комиссии входа).

`Lot` берётся из `Shares()` (`model.Share.Lot`) автоматически по тикеру.

## Кэш свечей

Свечи кэшируются в один файл на пару (тикер, интервал):
`data/candles/<TICKER>_<Interval>.json` (JSON-массив, oldest-first). Алгоритм:

- файла нет → тянем весь `[from, to]` кусками по 30 дней, сохраняем;
- файл есть → читаем; если последний бар старше `to`, дотягиваем только хвост,
  мержим с дедупликацией по времени, сохраняем;
- `-refresh` → игнорируем кэш, перетягиваем весь период, перезаписываем файл.

Берутся только закрытые свечи (`IsComplete == true`); незакрытый последний бар
отбрасывается. Между запросами к API — пауза 300 мс, чтобы не упереться в лимиты.

Каталоги `data/candles/` и `reports/` в `.gitignore` — это локальные артефакты.

## Добавление новой стратегии/тикера в калибровку

Движок (`Run`) общий и принимает любую `strategy.Strategy`. Калибровка завязана на
конкретный тип `Params` через `Binding` (`DefaultParams` / `Build` / `ParseParams`).
У каждой стратегии свой реестр и lookup-функция, которую вызывает
`cmd/backtest/main.go` по `-strategy`:

| Стратегия | Реестр | Lookup |
|---|---|---|
| `scalping` | `registry.go` | `Lookup` / `LookupOrGeneric` |
| `levels` | `levels_registry.go` | `LevelsLookupOrGeneric` |
| `momentum` | `momentum_registry.go` | `MomentumLookupOrGeneric` |
| `reversion` | `reversion_registry.go` | `ReversionLookupOrGeneric` |

`...OrGeneric` означает: тикеры без выделенного конфига получают нейтральные
дефолты стратегии. Чтобы добавить бумагу, реализуйте её `Params` (экспортированные
`int`/`float64`-поля) и зарегистрируйте `Binding` в нужном реестре — поля грида
применяются рефлексией по именам.

## Предупреждение о дефолтных параметрах

`DefaultParams()` — **стандартные, ещё не откалиброванные** на истории RUAL
значения, основанные на общих практиках теханализа. Именно для их подбора и
существует этот инструмент: гоняйте `-calibrate` на достаточном периоде, смотрите
сводку, выбирайте устойчивую (а не переоптимизированную) комбинацию и фиксируйте
её. Калибровка меняет только числа в `Params`, не трогая логику `Decide`.

## Что НЕ входит (осознанно)

- Короткие позиции, плечо, несколько одновременных позиций.
- Интрабар-исполнение `SL`/`TP` по `High`/`Low` (живая система не ставит стопы).
- Оптимизаторы умнее grid search (genetic и т.п.). Walk-forward (одиночный holdout
  и rolling) **уже поддержан**, см. раздел выше.
- Графики (есть только CSV equity для внешнего построения).
- Известное ограничение одиночного holdout: OOS-прогон стартует с холодными
  индикаторами на границе (rolling walk-forward этого лишён за счёт lead-in).

## Навигация

| Документ | О чём |
|---|---|
| [../scalping/README.md](../scalping/README.md) | Архитектура скальпинг-сервиса и контракт стратегии |
| [../scalping/settings.md](../scalping/settings.md) | Параметры `Params` скальпинга с дефолтами |
| [../reversion/strategy.md](../reversion/strategy.md) | Стратегия reversion: входы/выходы и параметры |
| [../momentum/](../momentum/) | Документация momentum-стратегии |
| [../levels/](../levels/) | Документация levels-стратегии |
| [../superpowers/specs/2026-06-03-scalping-backtest-design.md](../superpowers/specs/2026-06-03-scalping-backtest-design.md) | Дизайн-спецификация бэктеста |
| [../superpowers/specs/2026-06-16-walk-forward-calibration-design.md](../superpowers/specs/2026-06-16-walk-forward-calibration-design.md) | Дизайн rolling walk-forward |
