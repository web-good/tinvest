# Reversion Live Runner — карта кода

Документ для разработчика: **какой кусок кода за что отвечает** в боевом (live)
запуске стратегии reversion. Операторская часть (env-переменные, флаги, запуск) —
в соседнем [`live-runner.md`](./live-runner.md). Логика самих сигналов входа/выхода —
в [`strategy.md`](./strategy.md).

Ветка: `feat/reversion-rsi-dip` (базовый live-раннер); биржевые стоп-заявки и интрабарная
модель стопа добавлены веткой `feat/reversion-stop-orders`.
Дизайн-спеки: `docs/superpowers/specs/2026-06-25-reversion-live-runner-design.md` (раннер) +
`docs/superpowers/specs/2026-07-09-reversion-stop-orders-design.md` (стоп-заявки).

---

## 1. Картина целиком за 30 секунд

Боевой запуск — это **два cron-воркера** поверх одного и того же сервиса:

- **buy-pass** — ищет, что бы купить (только по тикерам, которых ещё нет в позиции);
- **manage-pass** — ведёт уже открытые позиции и продаёт по сигналу выхода.

Оба воркера делят **один экземпляр** `service` и сериализуются мьютексом, чтобы не
затирать друг другу файл состояния.

Ключевой принцип: **решения принимает тот же код, что и в бэктесте**
(`reversion/strategy/core`). Live-пакет НЕ содержит торговой логики — он только
*добывает данные → зовёт `core.Decide()` → исполняет результат → сохраняет состояние*.

Третий, более тонкий контур поверх этих двух проходов — **биржевая стоп-заявка**
(`live/stoporders`) — но не для всех тикеров: она существует только у тикеров с
`UseIntrabarStop=1` (интрабарная модель, сейчас NVTK; см. `strategy.md` → «Модель
исполнения стопов»). Для них buy-pass выставляет заявку сразу после фактического входа, а
manage-pass на каждом тике синхронизирует её (List/Cancel/Place) с желаемым уровнем STOP от
`core.DesiredStop`. У тикеров на close-модели (`UseIntrabarStop=0`, дефолт; сейчас UGLD,
EUTR) заявки нет вовсе — выход по STOP идёт обычной рыночной продажей по сигналу ядра, как
и остальные close-based выходы. Подробности — в §11.

```
cron "0 7-23 * * 1-5"  ─┐
  (ModeBuy)             ├─► scheduler.Run ─► service.Run ─┬─► buyPass     ─► core.Decide ─► executor.Buy  ─► placeInitialStop: UseIntrabarStop=1? stoporders.Place : no-op ─► statestore.Save
cron "0 7-23,0 * * *"  ─┘                  (под мьютексом) └─► managePass  ─► stoporders.List ─► core.Decide ─┬─► SignalSell: stoporders.Cancel (если есть) ─► executor.Sell ─► statestore.Save
                                                                                                                ├─► UseIntrabarStop≠1: снять оставшуюся заявку (если есть) ─► statestore.Save
                                                                                                                └─► UseIntrabarStop=1: sync стоп-заявки (Cancel+Place при росте уровня/несовпадении лотов) ─► statestore.Save
```

---

## 2. Карта пакетов

Корень: `internal/service/trading_strategy/reversion/live/`

| Файл / пакет | Роль | Ключевые символы |
|---|---|---|
| `live.go` | Ядро сервиса: зависимости, мьютекс, диспетчеризация прохода, общие хелперы | `service`, `NewService`, `Run`, `heldByShareID`, `sharesByTicker`, `notify` |
| `buy.go` | Проход покупки (ModeBuy) | `buyPass` |
| `manage.go` | Проход ведения/продажи (ModeManage) | `managePass`, `atrPeriodFor` |
| `registry.go` | Реестр тикер → калиброванные параметры | `paramsByTicker`, `StrategyFor`, `MaxHTFTrendEMA` |
| `dto/run.go` | Описание одного запуска (режим + cron-строка) | `Run`, `Mode`, `ModeBuy`, `ModeManage` |
| `scheduler/scheduler.go` | Обёртка cron: регистрирует job и блокирует горутину | `NewSchedulerService`, `Run` |
| `marketdata/marketdata.go` | Сборка `MarketData` из свечей Тинькофф (как в бэктесте) | `Assemble`, `fetchCompleted`, `ToCandles` |
| `sizing/sizing.go` | Расчёт количества лотов из % счёта | `Lots` |
| `executor/executor.go` | Размещение рыночных ордеров (или dry-run) | `Executor`, `Buy`, `Sell`, `place` |
| `stoporders/stoporders.go` | Постановка/снятие/список биржевой стоп-market SELL-заявки (или dry-run) | `Executor`, `Place`, `Cancel`, `List`, `ActiveStop`, `Result` |
| `statestore/statestore.go` | Персистентность entry-state в JSON (атомарно), включая стоп-поля | `FileStore`, `Entry`, `Load`, `Save` |
| `reconstruct/reconstruct.go` | Восстановление state из API, если файл потерян | `Entry`, `atrAtEntry` |
| `notifier/notifier.go` | Рендер Telegram-сообщений (чистые функции) | `Entry`, `Exit`, `Skip`, `Alert`, `StopSet` |

Внешние зависимости:

- `internal/config/reversion.go` — `ReversionConfig` (env-конфиг, дефолты).
- `internal/app/app.go:327-346` — где воркеры стартуют.
- `reversion/strategy/core` — торговая логика (`Decide`, `Lookback`, `Params`,
  `DesiredStop`).
- `internal/domain/backtest` — `AssembleMarketData` переиспользуется в live.
- gRPC `StopOrdersService` (`internal/pb/v1`) — постановка/отмена/список биржевых
  стоп-заявок; клиент — `grpcClient.StopOrdersServiceClient()`, передаётся в
  `live.NewService` как параметр `stops`.

---

## 3. Точка входа: запуск воркеров

`internal/app/app.go:327-346` стартует **две** горутины, по одной на режим:

```go
// buy: рабочие дни, часы 07:00–23:00, минута 0
reversiondto.Run{Scheduler: "0 7-23 * * 1-5", Mode: reversiondto.ModeBuy}

// manage: ежедневно, часы 07:00–00:00, минута 0
reversiondto.Run{Scheduler: "0 7-23,0 * * *", Mode: reversiondto.ModeManage}
```

Обе горутины зовут `a.sp.GetReversionLiveService()` — это **один и тот же** мемоизированный
экземпляр `service` (см. `service_provider`). Поэтому мьютекс в `service` реально
защищает оба прохода (см. §4).

Почему окна разные:
- **buy** только по будням и не круглосуточно — нет смысла открывать новые позиции
  в выходные/глубокой ночью при тонкой ликвидности MOEX;
- **manage** шире (и по выходным) — защитные/прибыльные выходы должны иметь шанс
  сработать всегда, пока позиция открыта.

> ⚠️ Важно для сверки с бэктестом: бэктест проходит **каждый** бар, а live-buy
> ограничен этим cron-окном. Сигнал входа в час вне окна live пропустит.

`scheduler/scheduler.go`:
- `Run` (строка 27) регистрирует cron-job через `pkg/scheduler` и **блокирует**
  горутину до `ctx.Done()`;
- `jobTicker := time.NewTicker(time.Hour)` (строка 28) — просто heartbeat в лог,
  к торговле отношения не имеет;
- при срабатывании cron вызывается `s.service.Run(ctx, in)` (строка 33).

---

## 4. Ядро: `service` и сериализация проходов

`live.go:37-51` — структура `service` с зависимостями:

| Поле | Тип | Зачем |
|---|---|---|
| `mu` | `sync.Mutex` | сериализует buy- и manage-проходы |
| `instruments` | `instrumentsClient` | список торгуемых акций (`Shares`) |
| `market` | `marketdata.CandleClient` | свечи (`GetCandles`) |
| `ops` | `operationsClient` | портфель, кэш, сделки (gRPC) |
| `exec` | `*executor.Executor` | размещение рыночных ордеров |
| `stops` | `*stoporders.Executor` | постановка/снятие/список биржевой стоп-заявки (§11) |
| `tg` | `telegram.Client` | уведомления |
| `cfg` | `*config.ReversionConfig` | конфиг (account, тикеры, %, флаги) |
| `statePath` | `string` | путь к файлу состояния: `data/state/reversion_<accountID>.json` |

`NewService` (`live.go:55-74`) с веткой стоп-заявок получила новый параметр `stops
stoporders.Client` (пятый позиционный аргумент, между `orders executor.OrdersClient` и `tg
telegram.Client`):

```go
func NewService(
    instruments instrumentsClient,
    market marketdata.CandleClient,
    ops operationsClient,
    orders executor.OrdersClient,
    stops stoporders.Client,
    tg telegram.Client,
    cfg *config.ReversionConfig,
) *service
```

Внутри конструктора `stops` оборачивается тем же флагом `cfg.TradeEnabled`, что и обычный
`exec`: `stoporders.New(stops, cfg.AccountID, cfg.TradeEnabled)` — отдельного тумблера
dry-run для стоп-заявок нет. Вызывающая сторона (`service_provider.GetReversionLiveService`,
`internal/service_provider/service.go:227-243`) передаёт
`grpcClient.StopOrdersServiceClient()`.

`Run` (`live.go:80-92`) — **единственная точка входа** прохода:

```go
func (s *service) Run(ctx context.Context, in dto.Run) error {
    s.mu.Lock()
    defer s.mu.Unlock()        // мьютекс держится ВЕСЬ проход
    switch in.Mode {
    case dto.ModeBuy:    return s.buyPass(ctx)
    case dto.ModeManage: return s.managePass(ctx)
    }
}
```

**Зачем мьютекс на весь проход** (`live.go:37-41`): buy и manage запланированы на
минуту 0 пересекающихся часов и делят один `service`. Каждый проход делает цикл
`Load → mutate → Save` над файлом состояния. Без блокировки два прохода могли бы
перетереть записи друг друга. Этот мьютекс — причина коммита
`fix(reversion/live): serialize buy/manage passes`.

Общие хелперы в `live.go`:
- `notify` (95) — шлёт в Telegram **только** если `NotifyEnabled`;
- `sharesByTicker` (102) — индекс `ticker → *Share` по всем торгуемым инструментам;
- `heldByShareID` (115) — позиции счёта с `Quantity > 0`, индекс по `ShareID`
  (так buy понимает, что уже в позиции, а manage — что вести);
- `nowMSK` (129) — текущее время в Europe/Moscow;
- `stateStore` (138) — `FileStore` по `statePath`.

---

## 5. Buy-pass: как открывается позиция

`buy.go:17-113`, функция `buyPass`. Поток по шагам:

1. **Подготовка** (18-30): грузим торгуемые акции (`sharesByTicker`), текущие
   позиции (`heldByShareID`), файл состояния (`store.Load`).
2. **Цикл по тикерам из конфига** (`cfg.Tickers`), для каждого:
   - **34-38** `StrategyFor(ticker)` — берём калиброванную стратегию из реестра.
     Нет в реестре → alert + пропуск.
   - **39-43** проверяем, что инструмент существует и `sh.Trading == true`.
   - **44-50** **фильтр «уже в позиции»**: если тикер есть в портфеле (`held`) или
     в нашем state — пропускаем, им займётся manage-pass.
   - **52-57** `marketdata.Assemble(...)` — собираем `MarketData` (закрытые часовые
     свечи + при необходимости 4H; см. §8). `md.Position = nil` — мы вне позиции.
   - **59-62** `st.Decide(md)` — **главный вызов**. Если это не `SignalBuy` — дальше.
   - **64-71** при сигнале берём из брокера полную стоимость счёта
     (`GetPortfolioTotal`) и свободный кэш (`GetAvailableCash`).
   - **72-76** `sizing.Lots(...)` — считаем лоты (см. §9). Не хватает на лот →
     `notifier.Skip` + пропуск.
   - **78-83** `exec.Buy(...)` — рыночный ордер (или dry-run). Ошибка → alert,
     **state не меняем** (повтор на следующем тике).
   - **85-95** определяем фактическую цену/количество: если ордер реально
     размещён и брокер вернул `FillPrice`/`FilledLots` — берём их, иначе fallback
     на цену сигнала и запрошенные лоты.
   - **97-107** записываем `statestore.Entry` (тикер, время, цена входа, **`EntryATR`
     из сигнала**, `MaxFav = fillPrice`, количество, стоп-поля ещё пустые) и
     атомарно сохраняем.
   - **108** уведомление `notifier.Entry`.
   - **110** `s.placeInitialStop(...)` — **только для тикеров с `UseIntrabarStop=1`**
     сразу выставляет защитную биржевую стоп-заявку на весь купленный объём (см. §11) и
     перезаписывает `state[ticker]` с заполненными `StopOrderID`/`StopPrice`/`StopReason`.
     Для тикера на close-модели (`UseIntrabarStop=0`, дефолт) функция делает ранний
     `return entry` сразу после проверки параметра (`buy.go:121-123`) — стоп-поля
     остаются пустыми/нулевыми, заявка не выставляется вовсе.

Ключевой момент — `EntryATR: sig.ATR` (строка 101). Стратегия «замораживает» ATR на
момент входа, и все защитные выходы (trailing/breakeven/STOP) считаются от этого
замороженного значения. Поэтому его обязательно надо сохранить — брокер его не отдаёт.

---

## 6. Manage-pass: как ведётся и закрывается позиция

`manage.go:20-273`, функция `managePass`. Поток:

1. **Подготовка** (21-33): акции, позиции, state — как в buy.
2. **Один снимок активных стоп-заявок** (35-44): `s.stops.List(ctx)` — единственный
   вызов `GetStopOrders` на весь проход (не на тикер), проиндексированный и по
   `StopOrderID` (`stopByID`), и по инструменту (`stopByInstrument`). `List`-ошибка
   не прерывает проход — просто отключает синхронизацию стоп-заявок в этом тике
   (`listErr`) и шлёт alert.
3. **Цикл по тикерам**, для каждого:
   - **48-55** стратегия из реестра + проверка торгуемости.
   - **57-69** `held[sh.ID]` — **если позиции в портфеле НЕТ**: если в state тикер
     ещё числится с непустым `StopOrderID`, это трактуется как **сработавшая
     стоп-заявка** — `notifier.Exit` с уже сохранёнными `StopReason`/`StopPrice`
     (детект срабатывания, live-runner.md → «Стоп-заявки»); иначе — обычная ручная
     продажа. В обоих случаях запись чистится из state. У close-тикеров
     `StopOrderID` пуст всегда, так что этот детект их не касается: их собственный
     выход по STOP уже прошёл через обычный `SignalSell`-путь (шаг «150-186» ниже) в
     каком-то из предыдущих проходов, стерев запись из state там же.
   - **71-87** **восстановление state** (`reconstruct.Entry`): позиция в портфеле
     есть, а локального state нет (потеряли файл / перезапуск без него). Тогда
     восстанавливаем `EntryTime/EntryATR/MaxFav` из API (см. §12) и шлём alert.
   - **113-123** **частичное исполнение стопа** (`pos.Quantity < entry.Quantity &&
     entry.StopOrderID != ""`): bookkeeping-only — `entry.Quantity` подрезается до
     фактического остатка, alert шлётся, но `StopOrderID` **не отменяется здесь**
     (снимок из шага 2 уже сделан один раз на пасс; повторная отмена той же заявки
     дала бы ложный alert). Реконсиляция размера живой заявки происходит ниже, в
     общем level-switch (см. следующий пункт). Актуально только для тикеров с
     `UseIntrabarStop=1` — у close-тикеров `StopOrderID` всегда пуст, условие не
     срабатывает.
   - **125-129** `marketdata.Assemble(...)` — свежий снимок рынка.
   - **131** `prevMaxFav := entry.MaxFav` — уровень, от которого была посчитана
     заявка, стоящая сейчас на бирже (нужен ниже как `PrevMaxFavorablePrice`).
   - **131-139** **обновление `MaxFav`**: если текущая цена (последний закрытый
     close) выше сохранённого максимума — поднимаем и сохраняем. Монотонный рост,
     нужен для TRAIL и breakeven (в обеих моделях — close-модель тоже поднимает
     `MaxFav`, просто trail-компонента `manage()` читает его без лага, см.
     `strategy.md`).
   - **141-147** собираем `md.Position` из state: цена входа, количество,
     **замороженный `EntryATR`**, `MaxFavorablePrice` (обновлённый) и
     `PrevMaxFavorablePrice` (до обновления этого тика) — оба поля идут в
     `core.DesiredStop`/`manage()`, см. `strategy.md` → «Модель исполнения стопов»
     (`manage()` сам выбирает, какое из двух полей использовать, по
     `UseIntrabarStop`).
   - **149** `sig := st.Decide(md)`.
   - **150-186** **если `SignalSell`** (`OB`/`RSI50`/`BE`/`RSIOS`/`EMAX`/`STOP` —
     оба варианта STOP: интрабарный триггер по low у `UseIntrabarStop=1` и
     close-триггер у `UseIntrabarStop=0`, оба возвращаются из `core.Decide` как
     обычный `SignalSell`): сначала `s.stops.Cancel(entry.StopOrderID)`, **если
     заявка есть** (`entry.StopOrderID != ""` — для close-тикера обычно пусто,
     шаг no-op) — провал `Cancel` алертит и **пропускает продажу в этом тике**
     (`continue`, без снятия заявки продавать нельзя). Затем guard `sh.Lot <= 0`
     (причина коммита `guard sell divide-by-zero`), `exec.Sell(...)`. Ошибка
     продажи → alert, state не трогаем (retry на следующем тике). Успех → удаляем
     запись из state, `notifier.Exit`.
   - **188-208** **гейт модели** (`p := mustParams(ticker); if p.UseIntrabarStop != 1`)
     — весь остаток функции (синхронизация биржевой заявки) относится только к
     тикерам на интрабарной модели. Для close-тикера этот блок вместо синхронизации
     лишь **подчищает переходное состояние**: если в `entry.StopOrderID` завалялась
     заявка, оставшаяся с тех пор, когда тикер ещё был на интрабаре (переключили
     `UseIntrabarStop` с `1` на `0` в его пакете параметров), и `List` в шаге 2 не
     упал — заявку, если она ещё жива, снимает (`Cancel`), затем в любом случае
     очищает `StopOrderID`/`StopPrice`/`StopReason` и сохраняет state; при
     недоступном `List` или провале `Cancel` — алерт, ничего не трогаем, ретрай на
     следующем тике. `continue` после этого блока — до конца итерации остаток
     функции (синхронизация) не выполняется.
   - **210-229** (только когда `List` в шаге 2 не упал, и только для
     `UseIntrabarStop=1`) **синхронизация**: если `StopOrderID` есть, но его нет
     среди живых заявок биржи — считаем, что потерялась (сбрасываем `StopOrderID`,
     alert); если своей заявки нет, а на бирже висит чужая/устаревшая на этот
     инструмент (например, после `reconstruct`) — отменяем её.
   - **232** `level, reason := core.DesiredStop(...)` от **обновлённого** `MaxFav`.
   - **238-246** `sizeMismatch` — живая заявка держит не те лоты, что реально в
     позиции (`live.Lots != entry.Quantity/sh.Lot`), например после частичного
     исполнения из шага «частичное исполнение».
   - **248-266** финальный `switch`: `reason==""` → ценовые стопы выключены,
     ничего не делаем; `StopOrderID==""` → `s.replaceStop` выставляет первую
     заявку; `sizeMismatch || level > entry.StopPrice` → `Cancel` старой +
     `s.replaceStop` новой (оверсайз или подъём трейла). Провал `Cancel` здесь —
     alert, **старая заявка не трогается**, ретрай на следующем часовом тике.
   - **267-270** сохраняем итоговый `state[ticker]`.

`atrPeriodFor` (276-281) — отдаёт `ATRPeriod` тикера для пересчёта ATR в
reconstruct (дефолт 14). `mustParams` (284-287) — тонкая обёртка над `ParamsFor` для
кейса, где тикер уже прошёл `StrategyFor` выше по стеку, так что `ok` гарантированно
`true`. `replaceStop` (291-315) — хелпер `manage.go`, вызывает `s.stops.Place`,
штампует `entry.StopOrderID` **только если реально размещено** (`res.Placed`), но
`entry.StopPrice`/`entry.StopReason` — **всегда** (в том числе в dry-run, чтобы
бумажный state отражал уровень, который был бы на бирже), и шлёт `notifier.StopSet`,
только если уровень/причина реально изменились. `buy.go`'s `placeInitialStop` (§5, шаг
110) делает то же самое для самой первой заявки, но отдельным телом функции — а не
вызовом `replaceStop` (код почти идентичен, но не переиспользован буквально).

---

## 7. Реестр стратегий: `registry.go`

Связывает тикер с его **калиброванными параметрами** и собирает конкретную стратегию.

- `paramsByTicker` (15-21) — карта `ticker → core.Params`. Зарегистрированы
  `UGLD`, `EUTR`, `NVTK`, `ASTR`, `SFIN`.
  - ⚠️ `SFIN` — **DO NOT TRADE** (калибровка провалена). Числится для полноты, но
    его не должно быть в `REVERSION_TICKERS`.
  - `ASTR` — baseline (без калибровки).
- `ParamsFor(ticker)` (24-27) — возвращает голый `core.Params` (без обёртки в
  `*core.Strategy`), `ok=false` для неизвестного тикера. Используется там, где нужны
  именно параметры, а не готовая стратегия: `manage.go` (`mustParams`, вызов
  `core.DesiredStop` в level-sync) и `buy.go` (`placeInitialStop`, тот же
  `core.DesiredStop` для самой первой заявки).
- `StrategyFor(ticker)` (30-36) — возвращает `*core.Strategy`
  (`core.NewWithParams(ticker, p)`) или `ok=false`.
- `MaxHTFTrendEMA(tickers)` (40-48) — максимальный период 4H-EMA среди тикеров;
  нужен, чтобы заранее знать, грузить ли 4H-свечи в `marketdata.Assemble`.

Параметры каждого тикера живут в `reversion/strategy/<ticker>/<ticker>.go`
(`DefaultParams()`). Именно тут отличаются `RSIOversold`, `UseTrend`, `UseVolume`,
`HTFTrendEMA`, выходы (`UseTrail`/`CatStopATRMult`/`UseATRStop` — компоненты STOP) и
т.д. Live и бэктест берут их из **одного и того же** места.

---

## 8. Сборка рыночных данных: `marketdata/marketdata.go`

Цель — построить такой же `MarketData`, как в бэктесте, чтобы `core.Decide` вёл себя
идентично. Для этого live **переиспользует** `backtest.AssembleMarketData` (строка 141).

- `Assemble` (119-142):
  1. `fetchCompleted(... Hour1, lookbackBars ...)` — тянет **закрытые** часовые свечи
     в количестве `Strategy.Lookback()`. Если их меньше lookback — ошибка
     (125-127), проход для тикера пропускается.
  2. если `htfEMAPeriod > 0` — дополнительно тянет 4H-свечи
     (`max(htfEMAPeriod + 20, htfWarmupBars)` для прогрева EMA — см. док-коммент
     `htfWarmupBars` о сходимости live-EMA с full-history EMA бэктеста).
  3. `cur := window[len-1].Time` — «сейчас» = время последней закрытой свечи.
- `fetchCompleted` (85-114): рассчитывает календарное окно с запасом
  (`warmupBufferFactor = 3`), запрашивает свечи и оставляет **последние `bars`**.
  Запас нужен, чтобы пережить выходные/праздники MOEX.
- `ToCandles` (64-80): конвертирует API-свечи в доменные; при `completedOnly=true`
  **отбрасывает ещё формирующийся бар** (`IsComplete=false`).

Константы прогрева (31-35): `barsPerCalendarDayHourly=6`, `barsPerCalendarDayHTF=2`,
`warmupBufferFactor=3` — намеренно пессимистичны (перебор по свечам дёшев, снимок
всё равно обрезается до точного lookback).

> Ключевое: live работает **только по закрытым свечам**, текущий незавершённый бар
> игнорируется. Это совпадает с бэктестом и исключает заглядывание в будущее.

---

## 9. Сайзинг: `sizing/sizing.go`

`Lots(buyPct, accountValue, cash, price, lot)` (14-32):

```
budget    = buyPct/100 * accountValue   // % от полной стоимости счёта
lotCost   = price * lot
lots      = floor(budget / lotCost)
affordable= floor(cash / lotCost)        // ограничение по свободному кэшу
lots      = min(lots, affordable)
```

Возвращает `ok=false` с человекочитаемой причиной, если:
- `price<=0` или `lot<=0` (некорректные данные);
- бюджет не покрывает один лот;
- кэша не хватает на один лот.

> Отличие от бэктеста: в бэктесте дефолт `Fraction=1.0` = **100% кэша** на позицию
> (так считались PF по одному тикеру). В live дефолт `BuyPct=10` — **10% от счёта**,
> и тикеры делят общий капитал.

---

## 10. Исполнение: `executor/executor.go`

Размещает **рыночные** ордера на всю позицию.

- `Executor` (28-33): клиент ордеров, `accountID`, флаг `tradeEnabled`.
- `Buy`/`Sell` (40-48) — обёртки над `place` с направлением.
- `place` (50-71):
  - **51-53** если `!tradeEnabled` → `Result{Placed:false}` без обращения к API
    (бумажный/dry-run режим). Caller тогда использует цену сигнала.
  - **54-61** строит `PostOrderRequest`: `OrderType=ORDER_TYPE_MARKET`,
    `OrderId=uuid.NewString()` — **свежий UUID для идемпотентности**.
  - **62-69** отправляет `PostOrder`, читает `LotsExecuted` и `ExecutedOrderPrice`.

`Result` (22-26): `Placed`, `FilledLots`, `FillPrice`. Когда `Placed=false`,
вызывающая сторона откатывается на цену сигнала и запрошенное количество.

> Отличие от бэктеста: бэктест «заполняет» по `Close` сигнального бара (и
> gap-protected для стопов). Live шлёт рыночный ордер уже **после** закрытия бара,
> поэтому реальный филл — это ближайшая рыночная цена + спред/проскальзывание,
> которые бэктест не моделирует.

---

## 11. Стоп-заявки: `stoporders/stoporders.go`

Ставит, снимает и листает **единственную** биржевую stop-market SELL-заявку на позицию —
живую реализацию компоненты STOP из `core.DesiredStop` (`strategy.md`), но только для
тикеров с `UseIntrabarStop=1` (сейчас NVTK); вызывающая сторона (`buy.go`, `manage.go`)
сама решает, звать ли `Place`/`Cancel` — сам пакет ничего не знает про модель тикера.
Симметричен `executor/executor.go`, но говорит с отдельным gRPC-сервисом
(`StopOrdersService`), а не `OrdersService`. `List` вызывается на каждом manage-тике
безусловно (единый снимок нужен и для detect-триггера, и для cleanup close-тикеров, см.
§6), но `Place`/`Cancel` для close-тикера в обычном режиме не зовутся вовсе — только при
переходном снятии заявки после переключения модели.

- `Client` (19-23) — срез клиента, нужный экзекьютору: `PostStopOrder`, `GetStopOrders`,
  `CancelStopOrder`.
- `ActiveStop` (26-31) — одна активная заявка из ответа `List`: `InstrumentUID`,
  `StopOrderID`, `StopPrice`, `Lots`.
- `Result` (35-38) — `Placed`, `OrderID`; при `Placed=false` (dry-run) вызывающая сторона
  продолжает полагаться на собственный state, а не на биржевую заявку.
- `Executor` / `New` (42-51) — обёртка над `Client` + `accountID` + `tradeEnabled` (тот же
  флаг `cfg.TradeEnabled`, что и у обычного `executor`).
- `roundDownToIncrement` (55-60) — округляет цену **вниз** к `MinPriceIncrement`
  инструмента: для sell-стопа округление вниз консервативно (заявка не окажется выше
  желаемого уровня стратегии).
- `Place` (64-86): при `!tradeEnabled` — сразу `Result{Placed:false}, nil` без обращения к
  API. Иначе строит `PostStopOrderRequest`: `Direction=SELL`,
  `ExpirationType=GOOD_TILL_CANCEL` (GTC — висит, пока не отменят или не исполнят),
  `StopOrderType=STOP_LOSS`, `ExchangeOrderType=MARKET` (после срабатывания заявка
  становится рыночной), `OrderId=uuid.NewString()` — свежий UUID на каждый вызов, для
  идемпотентности так же, как у рыночных ордеров в `executor`.
- `Cancel` (89-100) — при `!tradeEnabled` no-op (`nil`); иначе `CancelStopOrder`.
- `List` (104-128) — при `!tradeEnabled` возвращает `(nil, nil)`, поэтому в dry-run
  синхронизация с биржей в `manage.go` просто ничего не делает (сверять нечего). Иначе
  запрашивает `GetStopOrders(Status=ACTIVE)` и оставляет только SELL-заявки
  (`GetDirection() == SELL`), конвертируя `Quotation` в `float64` через `utils.CombinePrice`.

Вызывающая сторона (`manage.go`, `buy.go`) не работает с `Client`/gRPC напрямую — только
через `*stoporders.Executor` (поле `s.stops` в `service`, §4).

---

## 12. Состояние позиции: `statestore` и `reconstruct`

Брокер **не отдаёт** `EntryATR`, `MaxFavorablePrice` и наши стоп-поля — а без них не
посчитать trailing/breakeven/STOP или синхронизировать биржевую заявку. Поэтому их храним
сами.

### `statestore/statestore.go`
- `Entry` (16-27): `Ticker`, `EntryTime`, `EntryPrice`, `EntryATR`, `MaxFav`, `Quantity` —
  как раньше, плюс три новых поля стоп-заявки: `StopOrderID` (ID активной заявки на бирже,
  `""` = сейчас нет заявки — dry-run или временный провал `Place`), `StopPrice` (уровень
  выставленной или, в dry-run, *номинально* выставленной заявки — обновляется независимо от
  того, реально ли заявка на бирже), `StopReason` (`SL`/`ATRSL`/`TRAIL` — какая компонента
  STOP сейчас связывает уровень). Все три поля — часть контракта только тикеров с
  `UseIntrabarStop=1`: `placeInitialStop` (`buy.go`) их не заполняет для close-тикеров, а
  блок ведения заявки в `managePass` (`manage.go:210-266`) для них не выполняется вовсе.
  Единственная запись в эти поля у close-тикера — разовая переходная чистка (гейт модели,
  `manage.go:188-208`, §6): если после переключения тикера с интрабара осталась заявка,
  она снимается, а поля **обнуляются** в `""`/`0`/`""`. В устоявшемся состоянии у
  UGLD/EUTR они так и остаются `""`/`0`/`""` всю жизнь позиции.
- `FileStore` хранит **всю карту** `ticker → Entry` одним JSON-файлом
  (`data/state/reversion_<accountID>.json`).
- `Load` (44-60): отсутствующий файл = **пустая карта, не ошибка**.
- `Save` (63-86): **атомарная** запись — пишем во временный файл в той же
  директории и `rename`. Гарантирует, что файл никогда не остаётся «полуписаным».

### `reconstruct/reconstruct.go`
Фолбэк, когда позиция в портфеле есть, а локального state нет (потеря файла,
переезд на новую машину). `Entry` (29-72):
- `EntryPrice` ← средняя цена покупки от брокера;
- `EntryTime` ← **последняя BUY-сделка** за 120 дней (`GetInstrumentTrades`, 33-45);
- грузит часовые свечи от `entryTime - (lookback/4+10)` до now (47-55);
- `EntryATR` ← `atrAtEntry` (97-119): ATR на часовом окне, заканчивающемся на баре
  входа — повторяет то, как `core` штампует ATR при входе;
- `MaxFav` ← максимум close с момента входа (59-63).

Восстановленная запись выходит с **пустыми стоп-полями** (`reconstruct.Entry` их не
знает — на бирже могла остаться чужая/устаревшая заявка на этот инструмент, её подчистит
шаг синхронизации в `managePass`, §6). Восстановленные значения приблизительны (особенно
ATR), поэтому manage-pass шлёт alert о реконструкции (`manage.go:110`).

---

## 13. Конфиг и уведомления

### `internal/config/reversion.go`
`ReversionConfig` (5-11) + `NewReversionConfig` (16-21) с безопасными дефолтами:
- `Tickers = [UGLD, EUTR, NVTK]`, `BuyPct = 10`;
- `TradeEnabled` по умолчанию `false` — **отсутствие флага никогда не приведёт к
  реальному ордеру**;
- `TradeEnabled` и `NotifyEnabled` независимы (матрица режимов — в `live-runner.md`).

### `notifier/notifier.go`
Чистые функции рендера Telegram (вызывающая сторона шлёт только при `NotifyEnabled`):
- `Entry` (15) — 🟢 вход; `Exit` (21) — 🔴 выход с кодом причины;
- `Skip` (27) — ⏭️ пропуск входа (нехватка бюджета/кэша);
- `Alert` (32) — ⚠️ операционный алерт (реконструкция, отклонённый ордер, провал
  Place/Cancel/List стоп-заявки);
- `StopSet` (37) — 🛡 стоп-заявка (пере)выставлена на уровень с кодом причины
  (`SL`/`ATRSL`/`TRAIL`); шлётся из `replaceStop`/`placeInitialStop` только когда
  уровень или причина реально изменились (не на каждом тике).
- `paperTag` (7) добавляет пометку «БУМАЖНАЯ сделка», когда ордер (рыночный или
  стоп-заявка) не выставлялся — общая для всех пяти функций.

---

## 14. Где что искать (шпаргалка)

| Вопрос | Файл |
|---|---|
| Когда запускаются воркеры? | `internal/app/app.go:327-346` |
| Как cron вызывает сервис? | `live/scheduler/scheduler.go` |
| Логика «купить»? | `live/buy.go` |
| Логика «вести/продать»? | `live/manage.go` |
| Сами правила входа/выхода? | `reversion/strategy/core/core.go` (+ `strategy.md`) |
| Комбинированный уровень стопа (SL/ATRSL/TRAIL)? | `core.DesiredStop` (`core.go`) |
| Параметры конкретного тикера? | `reversion/strategy/<ticker>/<ticker>.go` |
| Параметры по тикеру без стратегии? | `registry.ParamsFor` (`live/registry.go`) |
| Откуда берутся свечи? | `live/marketdata/marketdata.go` |
| Сколько лотов покупать? | `live/sizing/sizing.go` |
| Как выставляется рыночный ордер? | `live/executor/executor.go` |
| Как выставляется/снимается/листается стоп-заявка? | `live/stoporders/stoporders.go` |
| Где хранится EntryATR/MaxFav/стоп-поля? | `live/statestore/statestore.go` |
| Что если файл состояния потерян? | `live/reconstruct/reconstruct.go` |
| Какие env-переменные? | `internal/config/reversion.go` + `live-runner.md` |
