# pdl_rsi live Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** pdl_rsi (Hour1) становится гостем живого раннера счёта rsi_pullback и по умолчанию выключен.

**Architecture:** Раннер (`rsi_pullback/live`) остаётся 30m. Необязательный интерфейс `adapter.BarInterval` даёт гостю свой интервал свечей. От интервала зависят сборка MarketData, свежесть бара, время бара входа, окно входа и разовое решение ядра на каждый завершённый бар (поле `LastBar` в стейте). Новый пакет `pdl_rsi/live` реализует `adapter.Strategy` по образцу `rsi_zone/live`. Конфиг `PDL_RSI_*` по умолчанию пуст и выключен.

**Tech Stack:** Go 1.25, mockery v2 mocks (`./bin/mage mocks`), `./bin/mage ci`.

**Spec:** `docs/superpowers/specs/2026-10-27-pdl-rsi-live-design.md`

## Global Constraints

- Ни один ордер без `PDL_RSI_TRADE_ENABLED=true`; дефолт `NewPDLRSIConfig()` — `BuyPct: 5`, `Tickers` пусто, `TradeEnabled` false.
- Гость без `BarInterval` ведёт себя побайтно как сейчас: тот же запрос свечей `Minutes30`, `maxBarAge` 60m, `entryBarTime = fill − 30m`. Поле `LastBar` он не пишет и не читает.
- Новые поля стейта — только `omitempty`.
- Приоритет слотов: rsi_pullback → rsi_zone → gap_fade → pdl_rsi.
- Имя стратегии в стейте — `"pdl_rsi"`, метка алертов — `"PDL RSI"`, тема — `TELEGRAM_TOPIC_PDL_RSI`.
- LAG_BOOST к pdl_rsi не применяется. Проверь, как буст ограничен для gap_fade (`lagboost.go`), и держи pdl_rsi вне его тем же способом.
- Комментарии в коде — на русском, в плотности окружающего кода.
- Перед прогоном CI бэктесты и тесты запускаются по правилу памяти: собранный бинарь, `GOMAXPROCS=1`.

## Review Focus

1. Пасс в :31 после покупки в :01. Ядро не должно решать по сигнальному часу как по часу с позицией. Это закрыто тем, что `buy` пишет `LastBar` = время сигнального бара. Тест: `TestHourGuestDoesNotDecideSignalBarAfterBuy` (Task 2).
2. Час 23:00 решается на 06:01 следующего дня, а не теряется. Тест: `TestHourGuestCatchesUpMissedHourOnNextPass` (Task 2).
3. На пассе :31 перестановка снятого стопа должна работать, хотя ядро по часу не решает. Тест: `TestHourGuestReplacesStopOnSkippedPass` (Task 2).
4. Часовой бар на :31 имеет возраст 91 мин, и это не протухание. Алерта быть не должно. Тест: `TestHourGuestNoStaleAlertOnHalfHourPass` (Task 2).
5. `PendingExit` повторяется на каждом пассе, даже когда ядро по часу уже решило. Тест: `TestHourGuestRetriesPendingExitOnSkippedPass` (Task 2).

---

### Task 1: Сборка MarketData по интервалу

**Files:**
- Modify: `internal/service/trading_strategy/rsi_pullback/live/marketdata/marketdata.go`
- Modify: `internal/service/trading_strategy/rsi_pullback/live/pass.go:219` (вызов)
- Modify: `cmd/pullparity/main.go:187` (вызов, `enum.Minutes30`)
- Test: `internal/service/trading_strategy/rsi_pullback/live/marketdata/marketdata_test.go` (если файла нет — создать)

**Interfaces:**
- Produces: `func Assemble(ctx context.Context, c candles.CandleClient, instrumentID string, interval enum.Interval, lookbackBars int, now time.Time) (strategy.MarketData, error)`. Размер чанка: `Minutes30` — 14 дней (как сейчас), `Hour1` — 60 дней (лимит API 3 месяца). Другой интервал — ошибка. Текст ошибки окна: `"%d completed %s candles < lookback %d"`.

- [ ] **Step 1: тесты** `TestAssembleHour1RequestsHourCandles`: мок `CandleClient` принимает только `enum.Hour1.ToNumberInvestAPI()` и `Day1`, `lookbackBars` 30; результат — `len(md.Times) == 30`. `TestAssembleRejectsUnsupportedInterval` проверяет, что `enum.Minutes5` даёт ошибку. Существующие тесты Assemble передают `enum.Minutes30`.
- [ ] **Step 2:** `go test ./internal/service/trading_strategy/rsi_pullback/live/marketdata/` — ожидается FAIL (не компилируется).
- [ ] **Step 3:** реализовать. `fetch30m` переименовать в `fetchBars(ctx, c, id, interval, lookback, now)`; префикс ошибок `rsi_pullback marketdata:` оставить. Обновить вызов в `pass.go`: пока `enum.Minutes30` (Task 2 заменит). Обновить вызов в `cmd/pullparity`.
- [ ] **Step 4:** `go build ./internal/... ./cmd/... && go test ./internal/service/trading_strategy/rsi_pullback/...` — PASS.
- [ ] **Step 5:** коммит `feat(livecore): interval-aware live market data assembly`.

### Task 2: Часовой гость в раннере

**Files:**
- Modify: `internal/service/trading_strategy/livecore/adapter/adapter.go`
- Modify: `internal/service/trading_strategy/livecore/statestore/statestore.go` (поле `LastBar`)
- Modify: `internal/service/trading_strategy/rsi_pullback/live/pass.go`
- Test: `internal/service/trading_strategy/rsi_pullback/live/hour_guest_test.go` (новый; фейк `hourFake` — `fakeStrategy` с методом `BarInterval() enum.Interval { return enum.Hour1 }`, по образцу `windowFake` в `shared_test.go`)

**Interfaces:**
- Consumes: `marketdata.Assemble(..., interval, ...)` из Task 1.
- Produces:
  - `adapter.BarInterval interface { BarInterval() enum.Interval }` — doc-комментарий: стратегия без него работает на `enum.Minutes30`.
  - `statestore.Entry.LastBar time.Time` с тегом `json:"lastBar,omitempty"` — время открытия последнего бара, по которому ядро уже решило. Пишут только гости с `BarInterval`.
  - В `pass.go`: `func intervalOf(sl *slot) enum.Interval`; `func barAgeLimit(iv enum.Interval) time.Duration` = `max(maxBarAge, 2*iv.ToTimeDuration())` (30m → 60m без изменений, Hour1 → 120m); `func entryBarTime(fill time.Time, iv enum.Interval) time.Time` — для `Minutes30` прежнее `fill − 30m`, для остальных `fill.Truncate(iv.ToTimeDuration())` (часовой гость покупает на открытии следующего бара, как `EntryAtNextOpen` движка: бар входа — тот, внутри которого заявка); `const cronStep = 30 * time.Minute`; `func freshClose(lastOpen time.Time, iv enum.Interval, now time.Time) bool` = `now.Sub(lastOpen.Add(iv.ToTimeDuration())) <= cronStep`.

Правила (только для слота с `BarInterval`; слоты без него идут прежним путём):
- `assemble` собирает данные по `intervalOf(sl)` и проверяет протухание по `barAgeLimit`.
- Свободный тикер: после `assemble`, если `!freshClose(md.Times[n-1], iv, now)` → `continue` без алерта и без Decide (лишний запрос свечей на :31 допустим).
- `buy`: в новую запись стейта `LastBar = md.Times[n-1]`.
- `manage`: страховочная часть (судьба стоп-заявки, реконсиляция размера, MaxFav, `DesiredStop`/перестановка) идёт всегда. `dec.Decide` зовётся, только если `entry.LastBar` нулевое или `md.Times[n-1].After(entry.LastBar)`. После Decide `LastBar` обновляется и сохраняется вместе с остальными изменениями записи. Если Decide пропущен, `sig` пустой, и ветка `PendingExit` срабатывает как сейчас.

- [ ] **Step 1: тесты** в `hour_guest_test.go`, время — MSK через существующие хелперы `shared_test.go`:
  - `TestHourGuestBuysOnlyOnPassAfterHourClose`: `now` 10:01, последний Hour1 бар 09:00 → BUY, `decideCalls == 1`, в стейте `LastBar == 09:00`. При `now` 10:31 (тот же бар 09:00) → `decideCalls == 0`, ордеров нет, алертов нет.
  - `TestHourGuestDoesNotDecideSignalBarAfterBuy`: стейт `{Strategy: hourFake, LastBar: 09:00}`, позиция у брокера, `now` 10:31, бар 09:00 → `decideCalls == 0`, стоп-заявка на месте.
  - `TestHourGuestCatchesUpMissedHourOnNextPass`: `LastBar` 22:00 вчера, `now` 06:01, последний бар 23:00 вчера; ядро отдаёт SELL → продажа, запись удалена.
  - `TestHourGuestReplacesStopOnSkippedPass`: `LastBar == последний бар`, `StopOrderID` нет в ACTIVE, `Executed` false → новая стоп-заявка поставлена, `decideCalls == 0`.
  - `TestHourGuestNoStaleAlertOnHalfHourPass`: позиция, `now` 10:31, бар 09:00 → нет ни одного сообщения с «протух».
  - `TestHourGuestRetriesPendingExitOnSkippedPass`: `PendingExit "RSI"`, `LastBar == последний бар` → SELL отправлен.
  - `TestHourGuestPositionEntryTimeIsFillHour`: позиция с `EntryTime` 10:01, Decide на баре 10:00 → `lastPosition.EntryTime == 10:00`.
  - `TestHalfHourGuestStateHasNoLastBar`: обычный `fakeStrategy` покупает → `LastBar.IsZero()`, а JSON файла стейта не содержит `lastBar`.
- [ ] **Step 2:** `go test ./internal/service/trading_strategy/rsi_pullback/live/ -run 'HourGuest|HalfHourGuest'` — FAIL.
- [ ] **Step 3:** реализовать интерфейс, поле и правила выше. Doc-комментарий `maxBarAge` и `freshEntryGrace` дополнить строкой о часовом госте.
- [ ] **Step 4:** `go test -race ./internal/service/trading_strategy/rsi_pullback/... ./internal/service/trading_strategy/livecore/...` — PASS, все старые тесты зелёные без правок.
- [ ] **Step 5:** коммит `feat(rsi_pullback/live): hourly guests on the 30m account runner`.

### Task 3: Уровень стопа pdl_rsi одной функцией

**Files:**
- Modify: `internal/service/trading_strategy/pdl_rsi/strategy/core/exits.go`
- Test: `internal/service/trading_strategy/pdl_rsi/strategy/core/exits_test.go`

**Interfaces:**
- Produces: `func StopLevel(p Params, fill, atr float64) float64` — `fill − stopDistance(fill, atr)` (с `StopATR` и `MaxStopPct`). Возвращает 0, если `fill <= 0`, `atr <= 0`, `p.StopATR <= 0` или уровень вне `(0, fill)`. `exitLevels` берёт стоп из `StopLevel`, когда `pos.EntryATR > 0 && s.p.StopATR > 0`. `stopDistance` — метод на `*Strategy`; при необходимости вынеси тело в свободную функцию от `Params`, а метод сделай обёрткой.

- [ ] **Step 1: тесты** `TestStopLevelMatchesExitLevels`: на параметрах `gazp.Params()`-подобного набора (`StopATR` 0.7, `MaxStopPct` 1.5) `StopLevel(p, 100, 5) == 98.5` (кап 1.5% сильнее 3.5) и `StopLevel(p, 100, 1) == 99.3`. `TestStopLevelZeroWithoutATR`: `atr` 0 → 0.
- [ ] **Step 2:** `go test ./internal/service/trading_strategy/pdl_rsi/strategy/core/ -run StopLevel` — FAIL.
- [ ] **Step 3:** реализовать.
- [ ] **Step 4:** `go test ./internal/service/trading_strategy/pdl_rsi/...` — PASS (старые тесты exits не меняются).
- [ ] **Step 5:** коммит `refactor(pdl_rsi): export StopLevel`.

### Task 4: Пакет pdl_rsi/live

**Files:**
- Create: `internal/service/trading_strategy/pdl_rsi/live/registry.go`, `strategy.go`
- Create: `internal/service/trading_strategy/pdl_rsi/live/registry_test.go`, `strategy_test.go`
- Create: `internal/config/pdl_rsi.go`, `internal/config/pdl_rsi_test.go`

**Interfaces:**
- Consumes: `core.StopLevel` (Task 3), `adapter.BarInterval` (Task 2).
- Produces:
  - `config.PDLRSIConfig{Tickers []string "PDL_RSI_TICKERS"; BuyPct float64 "PDL_RSI_BUY_PCT"; TradeEnabled bool "PDL_RSI_TRADE_ENABLED"; NotifyEnabled bool "PDL_RSI_NOTIFY_ENABLED"}`, `NewPDLRSIConfig()`, `(*PDLRSIConfig).Enabled() bool` — копия `RSIZoneConfig`.
  - `live.Name = "pdl_rsi"`; `func ParamsFor(ticker string) (core.Params, bool)`; `func StrategyFor(ticker string) (*core.Strategy, bool)` = `core.NewWithParams(ticker, time.Hour, p)`. Реестр: GAZP, LENT, MDMG, NVTK, SVCB, VTBR → `<pkg>.Params`. У пакетов нет константы `Ticker`, ключи — строковые литералы, как в `internal/service/backtest/pdl_rsi_registry.go`.
  - `func New(cfg *config.PDLRSIConfig, tg telegram.Client) *Strategy`, реализует `adapter.Strategy` и `adapter.BarInterval` (`enum.Hour1`).
  - `DesiredStop` → `core.StopLevel(p, e.EntryPrice, e.EntryATR)`, причина `"SL"`, а при 0 — `(0, "")`.
  - `Reconstruct` — как `rsi_zone/live.Reconstruct`, префикс ошибок `pdl_rsi reconstruct:`, `MaxFav` = `PurchasePrice`.

- [ ] **Step 1: тесты.**
  - `TestRegistryMatchesBacktestRegistry`: множество тикеров live совпадает с ключами `backtest` реестра и для каждого `ParamsFor` равен литералу пакета. Если реестр backtest не экспортирован, сравни с явным списком из шести тикеров.
  - `TestRegisteredLiteralsHaveExchangeStop`: у каждого литерала `StopATR > 0`.
  - `TestDesiredStopUsesStopLevel`: `Entry{EntryPrice:100, EntryATR:5}` по SVCB → `(core.StopLevel(svcb.Params(),100,5), "SL")`; при `EntryATR` 0 → `(0, "")`.
  - `TestReconstruct...`: три случая по образцу `rsi_zone/live/strategy_test.go` — успех, нет BUY, нулевой ATR.
  - `TestStrategyIsHourly`: `New(...).BarInterval() == enum.Hour1`.
  - `TestNotifyRespectsSwitch`: при `NotifyEnabled` false в tg ничего не уходит.
  - `config`: `TestPDLRSIConfigDefaultsOff`: `Enabled()` false, `TradeEnabled` false, `BuyPct` 5; `[""]` → `Enabled()` false.
- [ ] **Step 2:** `go test ./internal/service/trading_strategy/pdl_rsi/live/ ./internal/config/` — FAIL.
- [ ] **Step 3:** реализовать. Doc-комментарий пакета: гость раннера счёта rsi_pullback, ссылка на `docs/pdl_rsi/live.md`.
- [ ] **Step 4:** тот же `go test` — PASS.
- [ ] **Step 5:** коммит `feat(pdl_rsi): live adapter for the shared account runner`.

### Task 5: Подключение, env и сквозной тест

**Files:**
- Modify: `internal/config/config.go` (поле `PDLRSI *PDLRSIConfig`), `internal/app/init_config.go` (`PDLRSI: config.NewPDLRSIConfig()`)
- Modify: `internal/config/telegram_client.go` (`TopicPDLRSI int "TELEGRAM_TOPIC_PDL_RSI"`)
- Modify: `internal/service_provider/client.go` (`GetPDLRSISender`), `internal/service_provider/service.go` (гость после gap_fade, комментарий в стиле соседей)
- Modify: `env/local.env.example`, `env/prod.env.example`: ключи `PDL_RSI_TICKERS=` (пусто), `PDL_RSI_BUY_PCT=5`, `PDL_RSI_TRADE_ENABLED=false`, `PDL_RSI_NOTIFY_ENABLED=true`, `TELEGRAM_TOPIC_PDL_RSI=`. `env/prod.env` не трогать.
- Test: `internal/service/trading_strategy/rsi_pullback/live/pdlrsi_e2e_test.go` (по образцу `gapfade_e2e_test.go`)

**Interfaces:**
- Consumes: `pdlrsilive.New` (Task 4), правила Task 2.

- [ ] **Step 1: тесты** в `pdlrsi_e2e_test.go` на настоящем `pdlrsilive.Strategy` с GAZP в вселенной и `newSharedEnv`:
  - `TestPDLRSIGuestPaperDoesNotTrade`: `TradeEnabled` false, сигнал есть → ордеров нет.
  - `TestPDLRSIGuestTakesTickerOnlyWhenPullbackSilent`: rsi_pullback по GAZP молчит, гость с фейковым Decider, отдающим BUY (обёртка над адаптером, где подменён `Decider`), входит; в стейте `Strategy == "pdl_rsi"`, `LastBar` выставлен, биржевой стоп на уровне `core.StopLevel`.
  - `TestAnnounceIncludesPDLRSI`: стартовое объявление уходит и в тему гостя.
  Мок свечей отдаёт Hour1 и Day1, `isM30` по Hour1 не матчится — добавь матчер `isH1`.
- [ ] **Step 2:** `go test ./internal/service/trading_strategy/rsi_pullback/live/ -run PDLRSI` — FAIL.
- [ ] **Step 3:** реализовать подключение и env.
- [ ] **Step 4:** `go test -race ./internal/... ./cmd/...` — PASS; `go build ./internal/... ./pkg/... ./cmd/...` — OK.
- [ ] **Step 5:** коммит `feat(pdl_rsi): wire live guest, config and Telegram topic (off by default)`.

### Task 6: Сверка pullparity, документация, CI

**Files:**
- Modify: `cmd/pullparity/main.go`, `cmd/pullparity/main_test.go`
- Create: `docs/pdl_rsi/live.md`
- Modify: `CLAUDE.md` (в описании `pdl_rsi` заменить «backtest-only» на live-гостя: «live — гость раннера счёта rsi_pullback (`pdl_rsi/live`, Hour1, общий счёт; docs/pdl_rsi/live.md), сверка — `cmd/pullparity -strategy pdl_rsi`»), `docs/pdl_rsi/strategy.md` (ссылка на live.md)

**Interfaces:**
- Consumes: `pdlrsilive.StrategyFor`, `marketdata.Assemble(..., enum.Hour1, ...)`.
- `resolveDecider` возвращает и интервал (или отдельная `intervalFor(strategyName) enum.Interval`). `checkTicker` и `cacheClient` работают с рядом этого интервала вместо жёсткого `m30`; `asOf = cur.Add(interval)`.

- [ ] **Step 1: тест** в `main_test.go`: `TestResolveDeciderPDLRSI` — GAZP регистрируется, интервал `enum.Hour1`; `cacheClient` отвечает на запрос Hour1 и отказывает на Minutes30 для этого прогона.
- [ ] **Step 2:** `go test ./cmd/pullparity/` — FAIL.
- [ ] **Step 3:** реализовать; флаг `-strategy` дополнить `| pdl_rsi`.
- [ ] **Step 4:** `go build -o /tmp/pullparity ./cmd/pullparity && GOMAXPROCS=1 /tmp/pullparity -strategy pdl_rsi -ticker GAZP` (флаги выбора тикера и окна — по `-h`), то же для MDMG и SVCB. Ожидается: расхождений нет. Если они есть, не править тест, а разобрать причину и доложить. Также `-strategy rsi_zone` по одному тикеру из его реестра должен сойтись, как до изменений.
- [ ] **Step 5:** `docs/pdl_rsi/live.md`: только механика. Разделы: место в раннере и приоритет; env-ключи с дефолтами; часовой ритм (вход на :01, разовое решение на час, догоняние часа 23:00 на 06:01, страховка стопа на каждом пассе); стоп и реконструкция; известные расхождения с бэктестом (цель и выход RSI исполняются по рынку после закрытия часа; при нескольких пропущенных часах решается только последний; пропущенный пасс :01 вход не догоняет); строка «перед включением тикера проверь, что его литерал откалиброван на действующем ядре». Без PF и результатов калибровок.
- [ ] **Step 6:** `./bin/mage ci` — зелёный (lint + `go test -race ./...` + mock drift; при дрейфе — `./bin/mage mocks`).
- [ ] **Step 7:** коммит `feat(pullparity): pdl_rsi parity; docs: pdl_rsi live`.
