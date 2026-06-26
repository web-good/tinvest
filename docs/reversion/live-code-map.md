# Reversion Live Runner — карта кода

Документ для разработчика: **какой кусок кода за что отвечает** в боевом (live)
запуске стратегии reversion. Операторская часть (env-переменные, флаги, запуск) —
в соседнем [`live-runner.md`](./live-runner.md). Логика самих сигналов входа/выхода —
в [`strategy.md`](./strategy.md).

Ветка: `feat/reversion-rsi-dip`.
Дизайн-спека: `docs/superpowers/specs/2026-06-25-reversion-live-runner-design.md`.

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

```
cron "0 8-23 * * 1-5"  ─┐
  (ModeBuy)             ├─► scheduler.Run ─► service.Run ─┬─► buyPass     ─► core.Decide ─► executor.Buy  ─► statestore.Save
cron "0 7-23,0 * * *"  ─┘                  (под мьютексом) └─► managePass  ─► core.Decide ─► executor.Sell ─► statestore.Save
  (ModeManage)
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
| `statestore/statestore.go` | Персистентность entry-state в JSON (атомарно) | `FileStore`, `Entry`, `Load`, `Save` |
| `reconstruct/reconstruct.go` | Восстановление state из API, если файл потерян | `Entry`, `atrAtEntry` |
| `notifier/notifier.go` | Рендер Telegram-сообщений (чистые функции) | `Entry`, `Exit`, `Skip`, `Alert` |

Внешние зависимости:

- `internal/config/reversion.go` — `ReversionConfig` (env-конфиг, дефолты).
- `internal/app/app.go:340-359` — где воркеры стартуют.
- `reversion/strategy/core` — торговая логика (`Decide`, `Lookback`, `Params`).
- `internal/domain/backtest` — `AssembleMarketData` переиспользуется в live.

---

## 3. Точка входа: запуск воркеров

`internal/app/app.go:340-359` стартует **две** горутины, по одной на режим:

```go
// buy: рабочие дни, часы 08:00–23:00, минута 0
reversiondto.Run{Scheduler: "0 8-23 * * 1-5", Mode: reversiondto.ModeBuy}

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

`live.go:36-49` — структура `service` с зависимостями:

| Поле | Тип | Зачем |
|---|---|---|
| `mu` | `sync.Mutex` | сериализует buy- и manage-проходы |
| `instruments` | `instrumentsClient` | список торгуемых акций (`Shares`) |
| `market` | `marketdata.CandleClient` | свечи (`GetCandles`) |
| `ops` | `operationsClient` | портфель, кэш, сделки (gRPC) |
| `exec` | `*executor.Executor` | размещение ордеров |
| `tg` | `telegram.Client` | уведомления |
| `cfg` | `*config.ReversionConfig` | конфиг (account, тикеры, %, флаги) |
| `statePath` | `string` | путь к файлу состояния: `data/state/reversion_<accountID>.json` |

`Run` (`live.go:76-88`) — **единственная точка входа** прохода:

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
- `notify` (90) — шлёт в Telegram **только** если `NotifyEnabled`;
- `sharesByTicker` (98) — индекс `ticker → *Share` по всем торгуемым инструментам;
- `heldByShareID` (111) — позиции счёта с `Quantity > 0`, индекс по `ShareID`
  (так buy понимает, что уже в позиции, а manage — что вести);
- `nowMSK` (125) — текущее время в Europe/Moscow;
- `stateStore` (134) — `FileStore` по `statePath`.

---

## 5. Buy-pass: как открывается позиция

`buy.go:15-109`, функция `buyPass`. Поток по шагам:

1. **Подготовка** (16-28): грузим торгуемые акции (`sharesByTicker`), текущие
   позиции (`heldByShareID`), файл состояния (`store.Load`).
2. **Цикл по тикерам из конфига** (`cfg.Tickers`), для каждого:
   - **31-36** `StrategyFor(ticker)` — берём калиброванную стратегию из реестра.
     Нет в реестре → alert + пропуск.
   - **37-41** проверяем, что инструмент существует и `sh.Trading == true`.
   - **42-48** **фильтр «уже в позиции»**: если тикер есть в портфеле (`held`) или
     в нашем state — пропускаем, им займётся manage-pass.
   - **50-55** `marketdata.Assemble(...)` — собираем `MarketData` (закрытые часовые
     свечи + при необходимости 4H; см. §8). `md.Position = nil` — мы вне позиции.
   - **57-60** `st.Decide(md)` — **главный вызов**. Если это не `SignalBuy` — дальше.
   - **62-69** при сигнале берём из брокера полную стоимость счёта
     (`GetPortfolioTotal`) и свободный кэш (`GetAvailableCash`).
   - **70-74** `sizing.Lots(...)` — считаем лоты (см. §9). Не хватает на лот →
     `notifier.Skip` + пропуск.
   - **76-81** `exec.Buy(...)` — рыночный ордер (или dry-run). Ошибка → alert,
     **state не меняем** (повтор на следующем тике).
   - **83-93** определяем фактическую цену/количество: если ордер реально
     размещён и брокер вернул `FillPrice`/`FilledLots` — берём их, иначе fallback
     на цену сигнала и запрошенные лоты.
   - **95-105** записываем `statestore.Entry` (тикер, время, цена входа, **`EntryATR`
     из сигнала**, `MaxFav = fillPrice`, количество) и атомарно сохраняем.
   - **106** уведомление `notifier.Entry`.

Ключевой момент — `EntryATR: sig.ATR` (строка 99). Стратегия «замораживает» ATR на
момент входа, и все защитные выходы (trailing/breakeven/stop) считаются от этого
замороженного значения. Поэтому его обязательно надо сохранить — брокер его не отдаёт.

---

## 6. Manage-pass: как ведётся и закрывается позиция

`manage.go:16-120`, функция `managePass`. Поток:

1. **Подготовка** (17-29): акции, позиции, state — как в buy.
2. **Цикл по тикерам**, для каждого:
   - **33-40** стратегия из реестра + проверка торгуемости.
   - **41-49** `held[sh.ID]` — **если позиции в портфеле НЕТ**, а в state она ещё
     висит (продали где-то ещё) — чистим устаревшую запись и пропускаем.
   - **51-67** **восстановление state** (`reconstruct.Entry`): позиция в портфеле
     есть, а локального state нет (потеряли файл / перезапуск без него). Тогда
     восстанавливаем `EntryTime/EntryATR/MaxFav` из API (см. §11) и шлём alert.
   - **69-73** `marketdata.Assemble(...)` — свежий снимок рынка.
   - **75-82** **обновление `MaxFav`**: если текущая цена (последний закрытый
     close) выше сохранённого максимума — поднимаем и сохраняем. Монотонный рост,
     нужен для trailing-stop и breakeven.
   - **84-89** собираем `md.Position` из state: цена входа, количество,
     **замороженный `EntryATR`**, `MaxFavorablePrice`.
   - **91-94** `st.Decide(md)` — если не `SignalSell`, дальше.
   - **96-100** guard от деления на ноль: `sh.Lot <= 0` → alert + пропуск
     (это причина части коммита `guard sell divide-by-zero`).
   - **101-107** `lots = pos.Quantity / sh.Lot`; `exec.Sell(...)`. Ошибка → alert,
     state не трогаем.
   - **109-117** определяем цену выхода (фактический `FillPrice` либо `sig.Price`),
     **удаляем тикер из state**, сохраняем, шлём `notifier.Exit` с кодом причины
     (`OB`/`TRAIL`/`BE`/`ATRSL`/`RSIOS`/`EMAX` — см. `strategy.md`).

`atrPeriodFor` (123-128) — отдаёт `ATRPeriod` тикера для пересчёта ATR в
reconstruct (дефолт 14).

---

## 7. Реестр стратегий: `registry.go`

Связывает тикер с его **калиброванными параметрами** и собирает конкретную стратегию.

- `paramsByTicker` (15-21) — карта `ticker → core.Params`. Зарегистрированы
  `UGLD`, `EUTR`, `NVTK`, `ASTR`, `SFIN`.
  - ⚠️ `SFIN` — **DO NOT TRADE** (калибровка провалена). Числится для полноты, но
    его не должно быть в `REVERSION_TICKERS`.
  - `ASTR` — baseline (без калибровки).
- `StrategyFor(ticker)` (24-30) — возвращает `*core.Strategy`
  (`core.NewWithParams(ticker, p)`) или `ok=false`.
- `MaxHTFTrendEMA(tickers)` (34-42) — максимальный период 4H-EMA среди тикеров;
  нужен, чтобы заранее знать, грузить ли 4H-свечи в `marketdata.Assemble`.

Параметры каждого тикера живут в `reversion/strategy/<ticker>/<ticker>.go`
(`DefaultParams()`). Именно тут отличаются `RSIOversold`, `UseTrend`, `UseVolume`,
`HTFTrendEMA`, выходы и т.д. Live и бэктест берут их из **одного и того же** места.

---

## 8. Сборка рыночных данных: `marketdata/marketdata.go`

Цель — построить такой же `MarketData`, как в бэктесте, чтобы `core.Decide` вёл себя
идентично. Для этого live **переиспользует** `backtest.AssembleMarketData` (строка 104).

- `Assemble` (84-105):
  1. `fetchCompleted(... Hour1, lookbackBars ...)` — тянет **закрытые** часовые свечи
     в количестве `Strategy.Lookback()`. Если их меньше lookback — ошибка
     (90-92), проход для тикера пропускается.
  2. если `htfEMAPeriod > 0` — дополнительно тянет 4H-свечи (`htfEMAPeriod + 20`
     для прогрева EMA).
  3. `cur := window[len-1].Time` — «сейчас» = время последней закрытой свечи.
- `fetchCompleted` (60-79): рассчитывает календарное окно с запасом
  (`warmupBufferFactor = 3`), запрашивает свечи и оставляет **последние `bars`**.
  Запас нужен, чтобы пережить выходные/праздники MOEX.
- `ToCandles` (39-55): конвертирует API-свечи в доменные; при `completedOnly=true`
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

## 11. Состояние позиции: `statestore` и `reconstruct`

Брокер **не отдаёт** `EntryATR` и `MaxFavorablePrice` — а без них не посчитать
trailing/breakeven/stop. Поэтому их храним сами.

### `statestore/statestore.go`
- `Entry` (16-23): `Ticker`, `EntryTime`, `EntryPrice`, `EntryATR`, `MaxFav`,
  `Quantity`.
- `FileStore` хранит **всю карту** `ticker → Entry` одним JSON-файлом
  (`data/state/reversion_<accountID>.json`).
- `Load` (40-56): отсутствующий файл = **пустая карта, не ошибка**.
- `Save` (59-82): **атомарная** запись — пишем во временный файл в той же
  директории и `rename`. Гарантирует, что файл никогда не остаётся «полуписаным».

### `reconstruct/reconstruct.go`
Фолбэк, когда позиция в портфеле есть, а локального state нет (потеря файла,
переезд на новую машину). `Entry` (29-72):
- `EntryPrice` ← средняя цена покупки от брокера;
- `EntryTime` ← **последняя BUY-сделка** за 120 дней (`GetInstrumentTrades`, 33-45);
- грузит часовые свечи от `entryTime - (lookback/4+10)` до now (47-55);
- `EntryATR` ← `atrAtEntry` (95-119): ATR на часовом окне, заканчивающемся на баре
  входа — повторяет то, как `core` штампует ATR при входе;
- `MaxFav` ← максимум close с момента входа (59-63).

Восстановленные значения приблизительны (особенно ATR), поэтому manage-pass шлёт
alert о реконструкции (`manage.go:66`).

---

## 12. Конфиг и уведомления

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
- `Alert` (32) — ⚠️ операционный алерт (реконструкция, отклонённый ордер).
- `paperTag` (7) добавляет пометку «БУМАЖНАЯ сделка», когда ордер не выставлялся.

---

## 13. Где что искать (шпаргалка)

| Вопрос | Файл |
|---|---|
| Когда запускаются воркеры? | `internal/app/app.go:340-359` |
| Как cron вызывает сервис? | `live/scheduler/scheduler.go` |
| Логика «купить»? | `live/buy.go` |
| Логика «вести/продать»? | `live/manage.go` |
| Сами правила входа/выхода? | `reversion/strategy/core/core.go` (+ `strategy.md`) |
| Параметры конкретного тикера? | `reversion/strategy/<ticker>/<ticker>.go` |
| Откуда берутся свечи? | `live/marketdata/marketdata.go` |
| Сколько лотов покупать? | `live/sizing/sizing.go` |
| Как выставляется ордер? | `live/executor/executor.go` |
| Где хранится EntryATR/MaxFav? | `live/statestore/statestore.go` |
| Что если файл состояния потерян? | `live/reconstruct/reconstruct.go` |
| Какие env-переменные? | `internal/config/reversion.go` + `live-runner.md` |
