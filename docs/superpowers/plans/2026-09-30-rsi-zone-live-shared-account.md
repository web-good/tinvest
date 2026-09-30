# rsi_zone live на общем счёте с rsi_pullback — план реализации

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Живой раннер rsi_pullback становится раннером счёта с несколькими стратегиями; rsi_zone подключается к нему вторым адаптером, торгует на том же счёте, тикер с открытой позицией занят стратегией-владельцем.

**Architecture:** Пасс остаётся в `internal/service/trading_strategy/rsi_pullback/live` и обобщается на месте: вместо зашитого ядра rsi_pullback он ходит по упорядоченному списку «слотов» (адаптер `livecore/adapter.Strategy` + свои исполнители ордеров/стопов). rsi_pullback — первый слот (приоритет), rsi_zone подключается вариадиком `NewService(..., guests...)`. Владелец позиции — новое поле `strategy` в общем файле стейта; пустое значение = rsi_pullback (совместимость с прод-стейтом).

**Tech Stack:** Go 1.25, testify/mockery v2, heetch/confita, `./bin/mage ci`.

**Spec:** `docs/superpowers/specs/2026-09-30-rsi-zone-live-shared-account-design.md`

## Global Constraints

- Язык комментариев и сообщений — русский, как в окружающем коде; идентификаторы — английские.
- `rsi_pullback/live/service_test.go` не редактируется ни в одной задаче: ни хелперы, ни проверки. Это главная гарантия безопасности рефакторинга.
- Приоритет входа: `[rsi_pullback, rsi_zone]` — при двух BUY на одном баре входит rsi_pullback, второй адаптер на этом баре не спрашивается.
- Запись стейта без поля `strategy` принадлежит `rsi_pullback`.
- Бумажный вход пишет стейт, только если ни одна подключённая стратегия не боевая.
- Позиция без стейта на тикере из вселенных ≥2 стратегий: алерт, без реконструкции, стоп-заявки не трогаются.
- Ни одна новая переменная конфига не `required`.
- Расписание общее: `RSI_PULLBACK_SCHEDULE` (дефолт `1,31 6-23 * * *`).
- Вселенная rsi_zone в проде: `AFKS,BAZA,DIAS,DOMRF,LENT`; SBER не заводится.
- Сборка: `go build ./internal/... ./pkg/... ./cmd/...` (не `./...` — magefiles). Финальный гейт: `./bin/mage ci`.
- Коммиты: сообщение на русском в стиле репо (`feat(rsi_zone): ...`), в конце `Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>`. Коммитить только свои пути: `git commit -- <paths>` (в индексе ветки могут лежать чужие изменения). Новые файлы перед этим — `git add <path>` (иначе pathspec коммита их не найдёт).
- Документация `docs/rsi_*` — только механика, без результатов калибровок (правило CLAUDE.md).

## Review Focus

1. **Прод-стейт после выката** — файл `rsi_pullback_<acct>.json` без поля `strategy` у живых позиций: они должны вестись rsi_pullback, без алертов и реконструкции. Тест — Task 4 `TestLegacyEntryWithoutStrategyIsOwnedByPullback`.
2. **Смешанный режим pullback боевой + zone бумажный** — бумажный вход zone не должен занимать тикер и не должен звать `PostOrder`/`PostStopOrder`. Тест — Task 4 `TestPaperGuestEntryDoesNotClaimTickerWhenPullbackIsLive`.
3. **Тикер, убранный из вселенной, при открытой позиции** — владелец ведёт до выхода. Тест — Task 4 `TestPositionOutsideUniverseIsStillManagedByOwner`.
4. **Стейт с неизвестной стратегией** (ручная правка файла, откат версии) — алерт, стейт не трогается, ордеров нет. Тест — Task 4 `TestUnknownOwnerInStateAlertsAndKeepsState`.
5. **Пустая строка в `RSI_ZONE_TICKERS`** (`RSI_ZONE_TICKERS=` в env) — стратегия не подключается, а не подключается с тикером `""`. Тест — Task 5 `TestRSIZoneEnabledIgnoresBlankTickers`.

---

### Task 0: Предусловие — сборка ветки

На ветке `feat/rsi-zone-live` в индексе лежит staged-удаление `internal/service/trading_strategy/rsi_zone/strategy/sber/{sber.go,sber_test.go}` (унаследовано от `feat/lent-rsi-zone`), а `internal/service/backtest/rsi_zone_registry.go` всё ещё импортирует пакет sber — `go build ./internal/...` падает.

- [ ] **Step 1: Применить решение владельца** (зафиксировано в чате перед исполнением):
  - вариант «вернуть»: `git restore --staged --worktree internal/service/trading_strategy/rsi_zone/strategy/sber`
  - вариант «удалить совсем»: убрать импорт `rsizonesber` и строку `rsizonesber.Ticker: ...` из `internal/service/backtest/rsi_zone_registry.go`, поправить `internal/service/backtest/rsi_zone_registry_test.go` (убрать упоминания SBER), закоммитить удаление вместе с правкой реестра.
- [ ] **Step 2: Проверить сборку и тесты**

Run: `go build ./internal/... ./pkg/... ./cmd/... && go test ./internal/service/backtest/... ./internal/service/trading_strategy/...`
Expected: PASS

- [ ] **Step 3: Коммит** (только для варианта «удалить совсем»)

```bash
git add internal/service/backtest/rsi_zone_registry.go internal/service/backtest/rsi_zone_registry_test.go
git commit -m "chore(rsi_zone): пакет SBER убран из реестра бэктеста

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

---

### Task 1: Пакет `livecore/adapter` и владелец в стейте

**Files:**
- Create: `internal/service/trading_strategy/livecore/adapter/adapter.go`
- Create: `internal/service/trading_strategy/livecore/adapter/adapter_test.go`
- Modify: `internal/service/trading_strategy/livecore/statestore/statestore.go` (struct `Entry`)
- Test: `internal/service/trading_strategy/livecore/statestore/statestore_test.go` (добавить тест)

**Interfaces:**
- Produces:
  - `adapter.LegacyOwner = "rsi_pullback"`
  - `adapter.Owner(e statestore.Entry) string`
  - `adapter.Decider` (`Decide(strategy.MarketData) model.Signal`, `Lookback() int`)
  - `adapter.TradesClient` (`GetInstrumentTrades(ctx, accountID, instrumentID string, from, to time.Time) ([]grpcmodel.Trade, error)`)
  - `adapter.ReconstructInput{Trades TradesClient; Candles candles.CandleClient; AccountID, InstrumentID, Ticker string; PurchasePrice float64; Now time.Time}`
  - `adapter.Strategy` — см. код ниже
  - `statestore.Entry.Strategy string` (`json:"strategy,omitempty"`)

- [ ] **Step 1: Тест на совместимость формата стейта**

В `statestore_test.go` (посмотреть существующие тесты файла и повторить их стиль с `t.TempDir()`):

```go
// Прод-файл rsi_pullback записан до появления поля strategy. Он обязан читаться без
// ошибок, а поле — оставаться пустым: пустое значение и есть «позиция rsi_pullback».
func TestEntryWithoutStrategyFieldLoadsEmpty(t *testing.T) {
	path := filepath.Join(t.TempDir(), "state.json")
	raw := `{"GAZP":{"ticker":"GAZP","entryPrice":100,"entryATR":10,"quantity":10}}`
	if err := os.WriteFile(path, []byte(raw), 0o644); err != nil {
		t.Fatal(err)
	}
	st, err := New(path).Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if got := st["GAZP"].Strategy; got != "" {
		t.Fatalf("Strategy = %q, want пусто", got)
	}
}

// omitempty: reversion поле не пишет, и формат его файла меняться не должен.
func TestEmptyStrategyIsNotWritten(t *testing.T) {
	path := filepath.Join(t.TempDir(), "state.json")
	if err := New(path).Save(map[string]Entry{"UGLD": {Ticker: "UGLD"}}); err != nil {
		t.Fatal(err)
	}
	b, _ := os.ReadFile(path)
	if strings.Contains(string(b), "strategy") {
		t.Fatalf("пустое поле strategy попало в файл: %s", b)
	}
}
```

- [ ] **Step 2: Прогнать — падает компиляция** (`Entry` не имеет поля `Strategy`)

Run: `go test ./internal/service/trading_strategy/livecore/statestore/ -run 'Strategy' -v`
Expected: FAIL `st["GAZP"].Strategy undefined`

- [ ] **Step 3: Добавить поле в `Entry`** (после `PendingExit`)

```go
	// Strategy — стратегия-владелец позиции на счёте, где торгуют несколько стратегий
	// (docs/rsi_zone/live.md). Пустое значение — rsi_pullback: прод-файл записан до
	// появления поля, и его позиции должны вестись как прежде. reversion поле не пишет —
	// omitempty оставляет формат его файла прежним.
	Strategy string `json:"strategy,omitempty"`
```

- [ ] **Step 4: Тесты `adapter_test.go`**

```go
package adapter

import (
	"testing"

	"tinvest/internal/service/trading_strategy/livecore/statestore"
)

func TestOwnerOfLegacyEntryIsPullback(t *testing.T) {
	if got := Owner(statestore.Entry{}); got != LegacyOwner {
		t.Fatalf("Owner(пусто) = %q, want %q", got, LegacyOwner)
	}
}

func TestOwnerKeepsExplicitStrategy(t *testing.T) {
	if got := Owner(statestore.Entry{Strategy: "rsi_zone"}); got != "rsi_zone" {
		t.Fatalf("Owner = %q, want rsi_zone", got)
	}
}
```

- [ ] **Step 5: Прогнать — падает** (пакета нет)

Run: `go test ./internal/service/trading_strategy/livecore/adapter/ -v`
Expected: FAIL (no Go files / undefined Owner)

- [ ] **Step 6: Написать `adapter.go`**

```go
// Package adapter описывает, что живой раннер счёта ждёт от подключённой к нему стратегии.
// Раннер один на брокерский счёт (rsi_pullback/live); стратегии, торгующие на этом счёте,
// приходят к нему адаптерами — так rsi_zone/live не зависит от rsi_pullback/live.
package adapter

import (
	"context"
	"time"

	"tinvest/internal/service/trading_strategy/livecore/candles"
	"tinvest/internal/service/trading_strategy/livecore/statestore"
	"tinvest/internal/service/trading_strategy/scalping/model"
	"tinvest/internal/service/trading_strategy/scalping/strategy"
	grpcmodel "tinvest/pkg/client/grpc/model"
)

// LegacyOwner — владелец записей стейта без поля strategy: прод-файл rsi_pullback записан
// до того, как на счёт пришла вторая стратегия.
const LegacyOwner = "rsi_pullback"

// Owner возвращает стратегию-владельца записи стейта.
func Owner(e statestore.Entry) string {
	if e.Strategy == "" {
		return LegacyOwner
	}
	return e.Strategy
}

// Decider — то, что раннеру нужно от ядра стратегии для одного тикера.
type Decider interface {
	Decide(md strategy.MarketData) model.Signal
	Lookback() int
}

// TradesClient — срез operations-клиента, нужный реконструкции входа.
type TradesClient interface {
	GetInstrumentTrades(ctx context.Context, accountID, instrumentID string, from, to time.Time) ([]grpcmodel.Trade, error)
}

// ReconstructInput — всё, что раннер знает о позиции без локального стейта.
type ReconstructInput struct {
	Trades        TradesClient
	Candles       candles.CandleClient
	AccountID     string
	InstrumentID  string
	Ticker        string
	PurchasePrice float64 // средняя цена покупки из портфеля брокера
	Now           time.Time
}

// Strategy — стратегия, подключённая к раннеру счёта.
type Strategy interface {
	// Name пишется в стейт как владелец позиции: "rsi_pullback" | "rsi_zone".
	Name() string
	// Label — заголовок алертов: "RSI Pullback" | "RSI Zone".
	Label() string
	// Tickers — вселенная входов. Позиции вне её владелец ведёт до выхода.
	Tickers() []string
	// Decider — ядро с параметрами тикера; false — тикер не зарегистрирован.
	Decider(ticker string) (Decider, bool)
	// DesiredStop — защитный уровень открытой позиции; reason "" — стопа нет.
	DesiredStop(ticker string, e statestore.Entry) (level float64, reason string)
	// Reconstruct — вход позиции без локального стейта, по API брокера.
	Reconstruct(ctx context.Context, in ReconstructInput) (statestore.Entry, error)
	BuyPct() float64
	TradeEnabled() bool
	// Notify — своя тема Telegram и свой рубильник NotifyEnabled.
	Notify(msg string)
}
```

- [ ] **Step 7: Прогнать оба пакета**

Run: `go test ./internal/service/trading_strategy/livecore/adapter/ ./internal/service/trading_strategy/livecore/statestore/ -v`
Expected: PASS

- [ ] **Step 8: Коммит**

```bash
git add internal/service/trading_strategy/livecore/adapter internal/service/trading_strategy/livecore/statestore
git commit -m "feat(livecore): интерфейс адаптера стратегии и владелец позиции в стейте

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>" -- internal/service/trading_strategy/livecore/adapter internal/service/trading_strategy/livecore/statestore
```

---

### Task 2: Общие части реконструкции входа — `livecore/rebuild`

Выносим из `rsi_pullback/live/reconstruct` то, что понадобится и rsi_zone: поиск последней BUY-сделки и дневной ATR по будним дневкам до дня входа. Поведение rsi_pullback не меняется: `reconstruct_test.go` проходит без правок.

**Files:**
- Create: `internal/service/trading_strategy/livecore/rebuild/rebuild.go`
- Create: `internal/service/trading_strategy/livecore/rebuild/rebuild_test.go`
- Modify: `internal/service/trading_strategy/rsi_pullback/live/reconstruct/reconstruct.go`

**Interfaces:**
- Consumes: `adapter.TradesClient` (Task 1), `candles.CandleClient`.
- Produces:
  - `rebuild.ErrNoBuyFill` (sentinel)
  - `rebuild.LastBuyTime(ctx, tc adapter.TradesClient, accountID, instrumentID string, now time.Time) (time.Time, error)` — окно 180 дней; `ErrNoBuyFill`, если BUY нет
  - `rebuild.WeekdayDailyATRBefore(ctx, cc candles.CandleClient, instrumentID string, entryTime time.Time, period int) (float64, error)` — окно 730 дней; `0, nil` при нехватке баров или `period <= 0`
  - `rebuild.TradesLookbackDays = 180`, `rebuild.DailyFetchDays = 730`

- [ ] **Step 1: Тесты `rebuild_test.go`**

Моки: `internal/service/trading_strategy/rsi_pullback/live/reconstruct/mocks` уже содержит мок `TradesClient`, а `livecore/candles/mocks` — `MockCandleClient`. Посмотреть `reconstruct_test.go` и взять оттуда фикстуры дневок (будни + выходные) и шаблон ожиданий. Минимальный набор:

```go
package rebuild

// TestLastBuyTimePicksLatestBuy: две BUY и одна SELL — возвращается поздняя BUY.
// TestLastBuyTimeWithoutBuyReturnsSentinel: только SELL — errors.Is(err, ErrNoBuyFill).
// TestWeekdayDailyATRBeforeSkipsWeekendsAndEntryDay: дневки выходных и дня входа не входят
//   в расчёт — ATR равен indicators.ATR по будним барам до MSK-полуночи дня входа
//   (эталон считается в тесте тем же indicators.ATR по вручную отобранным барам).
// TestWeekdayDailyATRBeforeShortHistoryIsZero: period+1 будних баров нет — 0, nil.
```

Тексты тестов писать полностью, по образцу соответствующих тестов из `reconstruct_test.go` (они проверяют то же самое через `reconstruct.Entry`) — те тесты остаются на месте как регрессия.

- [ ] **Step 2: Прогнать — падает** (пакета нет)

Run: `go test ./internal/service/trading_strategy/livecore/rebuild/ -v`
Expected: FAIL

- [ ] **Step 3: Написать `rebuild.go`** — перенести `dailyATRAtEntry` из `reconstruct.go` дословно (вместе с `mskLoc`, doc-комментариями констант `tradesLookbackDays`/`dailyFetchDays`) под именем `WeekdayDailyATRBefore`, и цикл поиска BUY из `Entry` — под именем `LastBuyTime`:

```go
// ErrNoBuyFill — в окне нет ни одной BUY-сделки по инструменту.
var ErrNoBuyFill = errors.New("no BUY fill found")

func LastBuyTime(ctx context.Context, tc adapter.TradesClient, accountID, instrumentID string,
	now time.Time) (time.Time, error) {

	trades, err := tc.GetInstrumentTrades(ctx, accountID, instrumentID, now.AddDate(0, 0, -TradesLookbackDays), now)
	if err != nil {
		return time.Time{}, fmt.Errorf("trades: %w", err)
	}
	var entryTime time.Time
	for _, tr := range trades {
		if tr.IsBuy && tr.Date.After(entryTime) {
			entryTime = tr.Date
		}
	}
	if entryTime.IsZero() {
		return time.Time{}, ErrNoBuyFill
	}
	return entryTime, nil
}
```

- [ ] **Step 4: Переписать `reconstruct.Entry` поверх `rebuild`** — тексты ошибок сохранить побайтно:

```go
	entryTime, err := rebuild.LastBuyTime(ctx, tc, accountID, instrumentID, now)
	switch {
	case errors.Is(err, rebuild.ErrNoBuyFill):
		return statestore.Entry{}, fmt.Errorf("reconstruct: no BUY fill found for %s", ticker)
	case err != nil:
		return statestore.Entry{}, fmt.Errorf("reconstruct: %w", err)
	}

	atr, err := rebuild.WeekdayDailyATRBefore(ctx, cc, instrumentID, entryTime, p.DailyATRPeriod)
	if err != nil {
		return statestore.Entry{}, fmt.Errorf("reconstruct: %w", err)
	}
```

Внутри `WeekdayDailyATRBefore` ошибка свечей оборачивается как `fmt.Errorf("daily candles: %w", err)`, чтобы итог совпал с прежним `"reconstruct: daily candles: ..."`. В `reconstruct.go` остаются `maxCloseSinceEntry`, `m30ChunkDays`, `maxM30Chunks`; `dailyFetchDays` и `tradesLookbackDays` удаляются (используются `rebuild.*`). Интерфейс `reconstruct.TradesClient` оставить — на нём висят сгенерированные моки.

- [ ] **Step 5: Прогнать**

Run: `go test ./internal/service/trading_strategy/livecore/rebuild/ ./internal/service/trading_strategy/rsi_pullback/live/... -v 2>&1 | tail -30`
Expected: PASS, `reconstruct_test.go` и `service_test.go` без правок

- [ ] **Step 6: Коммит**

```bash
git add internal/service/trading_strategy/livecore/rebuild internal/service/trading_strategy/rsi_pullback/live/reconstruct/reconstruct.go
git commit -m "refactor(livecore): общая часть реконструкции входа — последняя BUY и будний дневной ATR

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>" -- internal/service/trading_strategy/livecore/rebuild internal/service/trading_strategy/rsi_pullback/live/reconstruct/reconstruct.go
```

---

### Task 3: Раннер rsi_pullback через слоты (без изменения поведения)

Раннер перестаёт звать ядро rsi_pullback напрямую: всё, что зависит от стратегии, идёт через `adapter.Strategy` одного слота. Стратегия пока ровно одна. Приёмка задачи — `service_test.go` зелёный **без единой правки**.

**Files:**
- Create: `internal/service/trading_strategy/rsi_pullback/live/adapter.go`
- Modify: `internal/service/trading_strategy/rsi_pullback/live/live.go`
- Modify: `internal/service/trading_strategy/rsi_pullback/live/pass.go`

**Interfaces:**
- Consumes: `adapter.*` (Task 1), `reconstruct.Entry` (Task 2).
- Produces (для Task 4):
  - `type slot struct { strat adapter.Strategy; exec *executor.Executor; stops *stoporders.Executor }`
  - `func (sl *slot) alert(ticker, msg string)` — `sl.strat.Notify(notifier.Alert(sl.strat.Label(), ticker, msg))`
  - поля `service`: `slots []*slot`, `accountStops *stoporders.Executor`, `anyLive bool`
  - `NewService(instruments, market, ops, orders, stops, tg, cfg *config.RSIPullbackConfig, guests ...adapter.Strategy) *service` (в этой задаче guests уже принимаются и добавляются слотами после rsi_pullback, но пасс их ещё не использует — это Task 4)
  - сигнатуры: `buy(ctx, pc, sl *slot, ticker string, sh *imodel.Share, dec adapter.Decider, md strategy.MarketData) (signaled bool, err error)`, `manage(ctx, pc, sl *slot, ticker string, sh *imodel.Share, dec adapter.Decider, md strategy.MarketData, pos *grpcmodel.Position, isHeld bool) error`, `sell(ctx, pc, sl *slot, ticker, sh, entry, pos, sig) error`, `settleGonePosition(ctx, pc, sl *slot, ticker string)`, `replaceStop(ctx, sl *slot, ticker, sh, entry, level, reason) statestore.Entry`, `placeInitialStop(ctx, pc, sl *slot, ticker, sh, entry) statestore.Entry`

- [ ] **Step 1: Базовая линия** — зафиксировать, что тесты зелёные до правок:

Run: `go test ./internal/service/trading_strategy/rsi_pullback/live/... -count=1`
Expected: PASS

- [ ] **Step 2: `adapter.go` — адаптер rsi_pullback**

```go
package live

import (
	"context"
	"fmt"

	"tinvest/internal/config"
	"tinvest/internal/service/trading_strategy/livecore/adapter"
	"tinvest/internal/service/trading_strategy/livecore/statestore"
	"tinvest/internal/service/trading_strategy/rsi_pullback/live/reconstruct"
	"tinvest/internal/service/trading_strategy/rsi_pullback/strategy/core"
	"tinvest/pkg/client/telegram"
	"tinvest/pkg/logger"
)

// pullbackStrategy — rsi_pullback как адаптер раннера счёта. Первый слот раннера: его
// порядок и есть приоритет входа, когда две стратегии дают BUY по тикеру на одном баре.
type pullbackStrategy struct {
	cfg *config.RSIPullbackConfig
	tg  telegram.Client
}

var _ adapter.Strategy = (*pullbackStrategy)(nil)

func (p *pullbackStrategy) Name() string      { return adapter.LegacyOwner }
func (p *pullbackStrategy) Label() string     { return alertLabel }
func (p *pullbackStrategy) Tickers() []string { return p.cfg.Tickers }
func (p *pullbackStrategy) BuyPct() float64   { return p.cfg.BuyPct }
func (p *pullbackStrategy) TradeEnabled() bool { return p.cfg.TradeEnabled }

func (p *pullbackStrategy) Decider(ticker string) (adapter.Decider, bool) {
	st, ok := StrategyFor(ticker)
	if !ok {
		return nil, false // не *core.Strategy(nil): типизированный nil в интерфейсе != nil
	}
	return st, true
}

func (p *pullbackStrategy) DesiredStop(ticker string, e statestore.Entry) (float64, string) {
	return core.DesiredStop(mustParams(ticker), e.EntryPrice, e.EntryATR, e.MaxFav)
}

func (p *pullbackStrategy) Reconstruct(ctx context.Context, in adapter.ReconstructInput) (statestore.Entry, error) {
	return reconstruct.Entry(ctx, in.Trades, in.Candles, in.AccountID, in.InstrumentID, in.Ticker,
		in.PurchasePrice, mustParams(in.Ticker), in.Now)
}

// Notify — перенесённое без изменений тело прежнего service.notify (весь doc-комментарий
// про «молчание о молчании» переносится вместе с ним).
func (p *pullbackStrategy) Notify(msg string) {
	if !p.cfg.NotifyEnabled {
		return
	}
	if err := p.tg.SendMessage(msg); err != nil {
		logger.ErrorContext(context.Background(),
			fmt.Sprintf("rsi_pullback: уведомление не доставлено: %v", err))
	}
}
```

- [ ] **Step 3: `live.go` — слоты вместо `exec`/`stops`/`tg`**

В `service` удалить поля `exec`, `stops`, `tg`; добавить:

```go
	// slots — стратегии счёта в порядке приоритета входа; slots[0] — rsi_pullback.
	slots []*slot
	// accountStops читает список активных стоп-заявок счёта — один раз на пасс. Боевой,
	// если боевая хоть одна стратегия: в dry-run List возвращает пустой список.
	accountStops *stoporders.Executor
	// anyLive — боевая хоть одна стратегия. Тогда бумажные входы стейт не пишут: бумажная
	// запись заняла бы тикер и заблокировала боевой вход соседки.
	anyLive bool
```

и тип:

```go
// slot — стратегия счёта со своими исполнителями: TradeEnabled у каждой свой.
type slot struct {
	strat adapter.Strategy
	exec  *executor.Executor
	stops *stoporders.Executor
}

func (sl *slot) alert(ticker, msg string) {
	sl.strat.Notify(notifier.Alert(sl.strat.Label(), ticker, msg))
}
```

`NewService`:

```go
func NewService(
	instruments instrumentsClient,
	market candles.CandleClient,
	ops operationsClient,
	orders executor.OrdersClient,
	stops stoporders.Client,
	tg telegram.Client,
	cfg *config.RSIPullbackConfig,
	guests ...adapter.Strategy,
) *service {
	strats := append([]adapter.Strategy{&pullbackStrategy{cfg: cfg, tg: tg}}, guests...)
	s := &service{
		instruments: instruments,
		market:      market,
		ops:         ops,
		cfg:         cfg,
		statePath:   filepath.Join("data", "state", "rsi_pullback_"+cfg.AccountID+".json"),
		now:         nowMSK,
	}
	for _, st := range strats {
		s.slots = append(s.slots, &slot{
			strat: st,
			exec:  executor.New(orders, cfg.AccountID, st.TradeEnabled()),
			stops: stoporders.New(stops, cfg.AccountID, st.TradeEnabled()),
		})
		s.anyLive = s.anyLive || st.TradeEnabled()
	}
	s.accountStops = stoporders.New(stops, cfg.AccountID, s.anyLive)
	return s
}
```

`notify(msg)` становится `func (s *service) notify(msg string) { s.slots[0].strat.Notify(msg) }` (счётные сообщения — в тему rsi_pullback). `Announce` пока без изменений по смыслу: `s.slots[0].strat.Notify(notifier.Startup(...))` — расширение на гостей в Task 4.

- [ ] **Step 4: `pass.go` — провести слот через все функции.** Механические замены в `buy`/`manage`/`sell`/`settleGonePosition`/`replaceStop`/`placeInitialStop`:
  - `s.exec` → `sl.exec`; `s.stops.Place/Cancel/Executed` → `sl.stops.*`; в `pass` — `s.accountStops.List(ctx)`;
  - `s.notify(notifier.Alert(alertLabel, ticker, X))` → `sl.alert(ticker, X)`; прочие `s.notify(...)` внутри этих функций → `sl.strat.Notify(...)`;
  - `core.DesiredStop(mustParams(ticker), entry.EntryPrice, entry.EntryATR, entry.MaxFav)` → `sl.strat.DesiredStop(ticker, entry)` (обе точки: `placeInitialStop` и `manage`);
  - `reconstruct.Entry(...)` в `manage` → 
    ```go
    rebuilt, rerr := sl.strat.Reconstruct(ctx, adapter.ReconstructInput{
        Trades: s.ops, Candles: s.market, AccountID: s.cfg.AccountID,
        InstrumentID: sh.ID, Ticker: ticker,
        PurchasePrice: utils.CombinePrice(pos.PurchasePrice.Units, pos.PurchasePrice.Nano),
        Now: pc.now,
    })
    ```
  - `s.cfg.BuyPct` → `sl.strat.BuyPct()`;
  - `st *core.Strategy` → `dec adapter.Decider` (`st.Decide` → `dec.Decide`, `st.Lookback()` → `dec.Lookback()`);
  - в `buy` добавить `Strategy: sl.strat.Name()` в создаваемый `statestore.Entry`; после успешной реконструкции в `manage` — `rebuilt.Strategy = sl.strat.Name()`;
  - `buy` возвращает `(signaled bool, err error)`: `false, nil` при отсутствии BUY, `true, ...` во всех ветках после `sig.Kind == SignalBuy`;
  - префиксы логов `"rsi_pullback: ..."` внутри per-slot функций → `sl.strat.Name()+": ..."` (текст после префикса не менять).

  `pass` в этой задаче сохраняет прежнюю форму (цикл по `s.cfg.Tickers`, `StrategyFor` → `s.slots[0].strat.Decider(ticker)`), работая через `sl := s.slots[0]`. Импорт `core` из `pass.go` уходит; `mustParams` остаётся (им пользуется адаптер).

- [ ] **Step 5: Сборка и прежние тесты без правок**

Run: `go build ./internal/... && go test ./internal/service/trading_strategy/rsi_pullback/live/... -count=1 -race`
Expected: PASS. `git diff --stat -- internal/service/trading_strategy/rsi_pullback/live/service_test.go` — пусто.

- [ ] **Step 6: Мутационная проверка** — временно заменить в `NewService` `st.TradeEnabled()` у `exec` на `true`; `TestBuySignalPlacesOrderStateAndProtectiveStop` (dry-run) обязан упасть на неожиданном вызове `PostOrder`. Вернуть.

- [ ] **Step 7: Коммит**

```bash
git commit -m "refactor(rsi_pullback): раннер счёта ходит в стратегию через адаптер

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>" -- internal/service/trading_strategy/rsi_pullback/live/adapter.go internal/service/trading_strategy/rsi_pullback/live/live.go internal/service/trading_strategy/rsi_pullback/live/pass.go
```

---

### Task 4: Несколько стратегий на счёте — владение, приоритет, бумажный режим

**Files:**
- Modify: `internal/service/trading_strategy/rsi_pullback/live/pass.go` (функция `pass` + новые хелперы)
- Modify: `internal/service/trading_strategy/rsi_pullback/live/live.go` (`Announce`, `slotByName`)
- Create: `internal/service/trading_strategy/rsi_pullback/live/shared_test.go`

**Interfaces:**
- Consumes: всё из Task 3.
- Produces: `(s *service) slotByName(name string) *slot`, `(s *service) universeSlots(ticker string) []*slot`, `(s *service) passTickers(state map[string]statestore.Entry) []string`.

- [ ] **Step 1: Тестовая обвязка `shared_test.go`** — фейковый адаптер и окружение с гостями (хелперы `shareFor`, `flatAt30m`, `pullback30m`, `dailies`, `gq`, `emptyStopList`, `msk`, моки — из `service_test.go` того же пакета):

```go
package live

// fakeStrategy — гость раннера с заранее заданным вердиктом. Реальное ядро rsi_zone здесь
// не нужно: проверяется маршрутизация раннера, а не решения стратегии.
type fakeStrategy struct {
	name       string
	universe   []string
	registered map[string]bool
	sig        model.Signal // Kind/Reason/ATR; Price подставляется из md
	stopLevel  float64
	trade      bool

	decideCalls  int
	rebuildCalls int
	msgs         []string
}

func (f *fakeStrategy) Name() string       { return f.name }
func (f *fakeStrategy) Label() string      { return "FAKE" }
func (f *fakeStrategy) Tickers() []string  { return f.universe }
func (f *fakeStrategy) BuyPct() float64    { return 5 }
func (f *fakeStrategy) TradeEnabled() bool { return f.trade }
func (f *fakeStrategy) Notify(msg string)  { f.msgs = append(f.msgs, msg) }

func (f *fakeStrategy) Decider(ticker string) (adapter.Decider, bool) {
	if !f.registered[ticker] {
		return nil, false
	}
	return fakeDecider{f}, true
}

func (f *fakeStrategy) DesiredStop(string, statestore.Entry) (float64, string) {
	if f.stopLevel <= 0 {
		return 0, ""
	}
	return f.stopLevel, "SL"
}

func (f *fakeStrategy) Reconstruct(_ context.Context, in adapter.ReconstructInput) (statestore.Entry, error) {
	f.rebuildCalls++
	return statestore.Entry{Ticker: in.Ticker, EntryTime: in.Now.Add(-24 * time.Hour),
		EntryPrice: in.PurchasePrice, EntryATR: 10, MaxFav: in.PurchasePrice}, nil
}

type fakeDecider struct{ f *fakeStrategy }

func (d fakeDecider) Lookback() int { return 50 }
func (d fakeDecider) Decide(md strategy.MarketData) model.Signal {
	d.f.decideCalls++
	sig := d.f.sig
	sig.Price = md.Price
	return sig
}

func (f *fakeStrategy) said(sub string) bool {
	for _, m := range f.msgs {
		if strings.Contains(m, sub) {
			return true
		}
	}
	return false
}

// newSharedEnv — newEnv с гостями: вселенная rsi_pullback задаётся явно.
func newSharedEnv(t *testing.T, now time.Time, pullTickers []string,
	tweak func(*config.RSIPullbackConfig), guests ...adapter.Strategy) *env {
	t.Helper()
	c := cfgFor("GAZP")
	c.Tickers = pullTickers
	if tweak != nil {
		tweak(c)
	}
	e := &env{
		statePath:   filepath.Join(t.TempDir(), "state.json"),
		instruments: livemocks.NewMockinstrumentsClient(t),
		market:      candlemocks.NewMockCandleClient(t),
		ops:         livemocks.NewMockoperationsClient(t),
		orders:      execmocks.NewMockOrdersClient(t),
		stops:       stopmocks.NewMockClient(t),
		tg:          tgmocks.NewMockClient(t),
	}
	e.svc = NewService(e.instruments, e.market, e.ops, e.orders, e.stops, e.tg, c, guests...)
	e.svc.statePath = e.statePath
	e.svc.now = func() time.Time { return now }
	return e
}

func sharesOf(tickers ...string) []*imodel.Share {
	var out []*imodel.Share
	for _, t := range tickers {
		out = append(out, shareFor(t)...)
	}
	return out
}

func held(ticker string, qty int64) *grpcmodel.Position {
	return &grpcmodel.Position{ShareID: "uid-" + ticker, InstrumentType: "share",
		Quantity: qty, PurchasePrice: gq(100)}
}

func zoneFake(sig model.SignalKind, universe ...string) *fakeStrategy {
	reg := map[string]bool{}
	for _, t := range universe {
		reg[t] = true
	}
	return &fakeStrategy{name: "rsi_zone", universe: universe, registered: reg,
		sig: model.Signal{Kind: sig, Reason: "FAKE", ATR: 10}, stopLevel: 90}
}

var (
	sharedNow     = time.Date(2026, 3, 10, 12, 1, 0, 0, msk)
	sharedLastBar = time.Date(2026, 3, 10, 11, 30, 0, 0, msk)
)
```

- [ ] **Step 2: Тесты сценариев** (в тот же файл). Каждый выставляет свои ожидания; `e.tg.EXPECT().SendMessage(mock.Anything).Return(nil).Maybe()`, где сообщения pullback не проверяются.

```go
// Оба дают BUY по свободному тикеру на одном баре — входит rsi_pullback, гостя не спрашивают.
func TestPullbackWinsTheSameBarBuy(t *testing.T) {
	zone := zoneFake(model.SignalBuy, "GAZP")
	e := newSharedEnv(t, sharedNow, []string{"GAZP"}, nil, zone)
	e.instruments.EXPECT().Shares(mock.Anything).Return(sharesOf("GAZP"), nil)
	e.expectCandles(pullback30m(sharedLastBar, 400), dailies(sharedNow, 60))
	e.ops.EXPECT().GetPortfolio(mock.Anything, mock.Anything).Return(nil, nil)
	e.ops.EXPECT().GetPortfolioTotal(mock.Anything, mock.Anything).Return(1_000_000.0, nil)
	e.ops.EXPECT().GetAvailableCash(mock.Anything, mock.Anything).Return(1_000_000.0, nil)
	e.tg.EXPECT().SendMessage(mock.Anything).Return(nil).Maybe()

	if err := e.svc.Run(context.Background(), dto.Run{}); err != nil {
		t.Fatalf("Run: %v", err)
	}
	if got := e.state(t)["GAZP"].Strategy; got != "rsi_pullback" {
		t.Fatalf("владелец = %q, want rsi_pullback", got)
	}
	if zone.decideCalls != 0 {
		t.Fatalf("гостя спросили %d раз после BUY rsi_pullback", zone.decideCalls)
	}
}

// rsi_pullback молчит — входит гость, запись помечена его именем, стоп — от его DesiredStop.
func TestGuestEntersWhenPullbackIsSilent(t *testing.T) {
	zone := zoneFake(model.SignalBuy, "GAZP")
	e := newSharedEnv(t, sharedNow, []string{"GAZP"}, nil, zone)
	e.instruments.EXPECT().Shares(mock.Anything).Return(sharesOf("GAZP"), nil)
	e.expectCandles(flatAt30m(sharedLastBar, 400, 100), dailies(sharedNow, 60))
	e.ops.EXPECT().GetPortfolio(mock.Anything, mock.Anything).Return(nil, nil)
	e.ops.EXPECT().GetPortfolioTotal(mock.Anything, mock.Anything).Return(1_000_000.0, nil)
	e.ops.EXPECT().GetAvailableCash(mock.Anything, mock.Anything).Return(1_000_000.0, nil)
	e.tg.EXPECT().SendMessage(mock.Anything).Return(nil).Maybe()

	if err := e.svc.Run(context.Background(), dto.Run{}); err != nil {
		t.Fatalf("Run: %v", err)
	}
	entry := e.state(t)["GAZP"]
	if entry.Strategy != "rsi_zone" {
		t.Fatalf("владелец = %q, want rsi_zone", entry.Strategy)
	}
	if entry.StopPrice != 90 || entry.StopReason != "SL" {
		t.Fatalf("стоп = %v/%q, want 90/SL от DesiredStop гостя", entry.StopPrice, entry.StopReason)
	}
}

// Позицию ведёт владелец из стейта: SELL гостя закрывает позицию, которую открыл гость.
func TestOwnerFromStateManagesThePosition(t *testing.T) {
	zone := zoneFake(model.SignalSell, "GAZP")
	e := newSharedEnv(t, sharedNow, []string{"GAZP"}, nil, zone)
	e.seed(t, statestore.Entry{Ticker: "GAZP", Strategy: "rsi_zone", EntryPrice: 100,
		EntryATR: 10, MaxFav: 100, Quantity: 100, EntryTime: sharedNow.Add(-48 * time.Hour)})
	e.instruments.EXPECT().Shares(mock.Anything).Return(sharesOf("GAZP"), nil)
	e.expectCandles(flatAt30m(sharedLastBar, 400, 100), dailies(sharedNow, 60))
	e.ops.EXPECT().GetPortfolio(mock.Anything, mock.Anything).Return([]*grpcmodel.Position{held("GAZP", 100)}, nil)
	e.tg.EXPECT().SendMessage(mock.Anything).Return(nil).Maybe()

	if err := e.svc.Run(context.Background(), dto.Run{}); err != nil {
		t.Fatalf("Run: %v", err)
	}
	if _, ok := e.state(t)["GAZP"]; ok {
		t.Fatal("SELL владельца-гостя не закрыл позицию")
	}
	if zone.decideCalls != 1 {
		t.Fatalf("Decide гостя вызван %d раз, want 1", zone.decideCalls)
	}
}

// Занятый тикер: позиция rsi_pullback открыта — гость по нему не входит и даже не спрашивается.
func TestOccupiedTickerIsClosedForTheOtherStrategy(t *testing.T) {
	zone := zoneFake(model.SignalBuy, "GAZP")
	e := newSharedEnv(t, sharedNow, []string{"GAZP"}, nil, zone)
	e.seed(t, openEntry("")) // запись без Strategy — позиция rsi_pullback
	e.instruments.EXPECT().Shares(mock.Anything).Return(sharesOf("GAZP"), nil)
	e.expectCandles(flatAt30m(sharedLastBar, 400, 100), dailies(sharedNow, 60))
	e.ops.EXPECT().GetPortfolio(mock.Anything, mock.Anything).Return(heldGAZP(100), nil)
	e.tg.EXPECT().SendMessage(mock.Anything).Return(nil).Maybe()

	if err := e.svc.Run(context.Background(), dto.Run{}); err != nil {
		t.Fatalf("Run: %v", err)
	}
	if zone.decideCalls != 0 {
		t.Fatalf("гостя спросили по занятому тикеру %d раз", zone.decideCalls)
	}
}

// Прод-стейт до выката: запись без поля strategy ведёт rsi_pullback — без алертов о
// неизвестном владельце и без реконструкции.
func TestLegacyEntryWithoutStrategyIsOwnedByPullback(t *testing.T) {
	zone := zoneFake(model.SignalSell, "GAZP")
	e := newSharedEnv(t, sharedNow, []string{"GAZP"}, nil, zone)
	e.seed(t, openEntry(""))
	e.instruments.EXPECT().Shares(mock.Anything).Return(sharesOf("GAZP"), nil)
	e.expectCandles(flatAt30m(sharedLastBar, 400, 100), dailies(sharedNow, 60))
	e.ops.EXPECT().GetPortfolio(mock.Anything, mock.Anything).Return(heldGAZP(100), nil)
	var sent []string
	e.tg.EXPECT().SendMessage(mock.Anything).RunAndReturn(func(s string) error {
		sent = append(sent, s)
		return nil
	}).Maybe()

	if err := e.svc.Run(context.Background(), dto.Run{}); err != nil {
		t.Fatalf("Run: %v", err)
	}
	for _, s := range sent {
		if strings.Contains(s, "неизвестн") {
			t.Fatalf("запись без strategy дала алерт о неизвестном владельце: %s", s)
		}
	}
	if _, ok := e.state(t)["GAZP"]; !ok {
		t.Fatal("позицию закрыл SELL гостя — запись без strategy досталась не rsi_pullback")
	}
	if zone.decideCalls != 0 || zone.rebuildCalls != 0 {
		t.Fatalf("гость тронул чужую позицию: decide=%d rebuild=%d", zone.decideCalls, zone.rebuildCalls)
	}
}

// Позиция без стейта на общем тикере: владельца не определить — алерт, без реконструкции,
// стоп-заявки по инструменту не трогаются.
func TestHeldSharedTickerWithoutStateIsNotReconstructed(t *testing.T) {
	zone := zoneFake(model.SignalNone, "GAZP")
	e := newSharedEnv(t, sharedNow, []string{"GAZP"}, nil, zone)
	e.instruments.EXPECT().Shares(mock.Anything).Return(sharesOf("GAZP"), nil)
	e.ops.EXPECT().GetPortfolio(mock.Anything, mock.Anything).Return(heldGAZP(100), nil)
	e.tg.EXPECT().SendMessage(mock.MatchedBy(func(s string) bool {
		return strings.Contains(s, "владелец неизвестен") && strings.Contains(s, "GAZP")
	})).Return(nil).Once()

	if err := e.svc.Run(context.Background(), dto.Run{}); err != nil {
		t.Fatalf("Run: %v", err)
	}
	if st := e.state(t); len(st) != 0 {
		t.Fatalf("стейт создан без владельца: %+v", st)
	}
	if zone.rebuildCalls != 0 {
		t.Fatal("гость реконструировал позицию на общем тикере")
	}
	// market без ожиданий: свечи по такому тикеру не запрашиваются; stops без ожиданий на
	// Cancel/Place — mockery провалит тест на любом вызове.
}

// Позиция без стейта на тикере одной стратегии — реконструирует она.
func TestHeldGuestOnlyTickerIsReconstructedByGuest(t *testing.T) {
	zone := zoneFake(model.SignalNone, "AFKS")
	e := newSharedEnv(t, sharedNow, []string{"GAZP"}, nil, zone)
	e.instruments.EXPECT().Shares(mock.Anything).Return(sharesOf("GAZP", "AFKS"), nil)
	e.expectCandles(flatAt30m(sharedLastBar, 400, 100), dailies(sharedNow, 60))
	e.ops.EXPECT().GetPortfolio(mock.Anything, mock.Anything).Return([]*grpcmodel.Position{held("AFKS", 100)}, nil)
	e.tg.EXPECT().SendMessage(mock.Anything).Return(nil).Maybe()

	if err := e.svc.Run(context.Background(), dto.Run{}); err != nil {
		t.Fatalf("Run: %v", err)
	}
	if zone.rebuildCalls != 1 {
		t.Fatalf("rebuild гостя = %d, want 1", zone.rebuildCalls)
	}
	if got := e.state(t)["AFKS"].Strategy; got != "rsi_zone" {
		t.Fatalf("владелец восстановленной позиции = %q, want rsi_zone", got)
	}
}

// Смешанный режим: rsi_pullback боевой, гость бумажный. Бумажный вход — только уведомление:
// стейт не пишется, тикер не занимается, ордеров нет.
func TestPaperGuestEntryDoesNotClaimTickerWhenPullbackIsLive(t *testing.T) {
	zone := zoneFake(model.SignalBuy, "GAZP")
	e := newSharedEnv(t, sharedNow, []string{"GAZP"},
		func(c *config.RSIPullbackConfig) { c.TradeEnabled = true }, zone)
	e.instruments.EXPECT().Shares(mock.Anything).Return(sharesOf("GAZP"), nil)
	e.expectCandles(flatAt30m(sharedLastBar, 400, 100), dailies(sharedNow, 60))
	e.ops.EXPECT().GetPortfolio(mock.Anything, mock.Anything).Return(nil, nil)
	e.ops.EXPECT().GetPortfolioTotal(mock.Anything, mock.Anything).Return(1_000_000.0, nil)
	e.ops.EXPECT().GetAvailableCash(mock.Anything, mock.Anything).Return(1_000_000.0, nil)
	e.stops.EXPECT().GetStopOrders(mock.Anything, mock.Anything).Return(emptyStopList(), nil)
	e.tg.EXPECT().SendMessage(mock.Anything).Return(nil).Maybe()

	if err := e.svc.Run(context.Background(), dto.Run{}); err != nil {
		t.Fatalf("Run: %v", err)
	}
	if st := e.state(t); len(st) != 0 {
		t.Fatalf("бумажный вход гостя занял тикер: %+v", st)
	}
	if !zone.said("GAZP") {
		t.Fatal("бумажный вход гостя не дошёл до уведомления")
	}
	e.orders.AssertNotCalled(t, "PostOrder", mock.Anything, mock.Anything)
}

// Тикер ушёл из вселенной владельца, позиция открыта — владелец ведёт её до выхода.
func TestPositionOutsideUniverseIsStillManagedByOwner(t *testing.T) {
	zone := zoneFake(model.SignalSell) // вселенная пуста
	zone.registered = map[string]bool{"AFKS": true}
	e := newSharedEnv(t, sharedNow, []string{"GAZP"}, nil, zone)
	e.seed(t, statestore.Entry{Ticker: "AFKS", Strategy: "rsi_zone", EntryPrice: 100,
		EntryATR: 10, MaxFav: 100, Quantity: 100, EntryTime: sharedNow.Add(-48 * time.Hour)})
	e.instruments.EXPECT().Shares(mock.Anything).Return(sharesOf("GAZP", "AFKS"), nil)
	e.expectCandles(flatAt30m(sharedLastBar, 400, 100), dailies(sharedNow, 60))
	e.ops.EXPECT().GetPortfolio(mock.Anything, mock.Anything).Return([]*grpcmodel.Position{held("AFKS", 100)}, nil)
	e.tg.EXPECT().SendMessage(mock.Anything).Return(nil).Maybe()

	if err := e.svc.Run(context.Background(), dto.Run{}); err != nil {
		t.Fatalf("Run: %v", err)
	}
	if _, ok := e.state(t)["AFKS"]; ok {
		t.Fatal("позиция вне вселенной брошена: SELL владельца не исполнен")
	}
}

// Стейт с неизвестной стратегией: алерт в тему счёта, стейт и биржа не трогаются.
func TestUnknownOwnerInStateAlertsAndKeepsState(t *testing.T) {
	e := newSharedEnv(t, sharedNow, []string{"GAZP"}, nil)
	ghost := openEntry("")
	ghost.Strategy = "ghost"
	e.seed(t, ghost)
	e.instruments.EXPECT().Shares(mock.Anything).Return(sharesOf("GAZP"), nil)
	e.ops.EXPECT().GetPortfolio(mock.Anything, mock.Anything).Return(heldGAZP(100), nil)
	e.tg.EXPECT().SendMessage(mock.MatchedBy(func(s string) bool {
		return strings.Contains(s, "ghost") && strings.Contains(s, "GAZP")
	})).Return(nil).Once()

	if err := e.svc.Run(context.Background(), dto.Run{}); err != nil {
		t.Fatalf("Run: %v", err)
	}
	if got := e.state(t)["GAZP"].Strategy; got != "ghost" {
		t.Fatalf("стейт неизвестного владельца изменён: %q", got)
	}
}

// Гость объявляет о себе при старте в своей теме.
func TestAnnounceIncludesGuests(t *testing.T) {
	zone := zoneFake(model.SignalNone, "AFKS")
	e := newSharedEnv(t, sharedNow, []string{"GAZP"}, nil, zone)
	e.tg.EXPECT().SendMessage(mock.Anything).Return(nil).Once()

	e.svc.Announce()
	if !zone.said("FAKE") || !zone.said("AFKS") {
		t.Fatalf("гость не объявил о себе: %v", zone.msgs)
	}
}
```

Если `flatAt30m` на 400 барах с дневками фикстуры всё же даёт BUY у ядра rsi_pullback (проверить `TestGuestEntersWhenPullbackIsSilent` первым прогоном), подобрать ровную ленту, на которой ядро молчит, и оставить комментарий, почему она молчит.

- [ ] **Step 3: Прогнать — новые тесты падают**

Run: `go test ./internal/service/trading_strategy/rsi_pullback/live/ -run 'Pullback|Guest|Owner|Occupied|Legacy|Shared|Paper|Outside|Unknown|Announce' -v 2>&1 | tail -40`
Expected: FAIL на сценариях гостя (пасс ходит только по `slots[0]`)

- [ ] **Step 4: Хелперы в `pass.go`**

```go
// passTickers — объединение вселенных в порядке приоритета стратегий, затем тикеры стейта
// вне всех вселенных (отсортированно): владелец ведёт позицию до выхода, даже если тикер
// убрали из его вселенной.
func (s *service) passTickers(state map[string]statestore.Entry) []string {
	seen := map[string]bool{}
	var out []string
	for _, sl := range s.slots {
		for _, t := range sl.strat.Tickers() {
			if !seen[t] {
				seen[t] = true
				out = append(out, t)
			}
		}
	}
	var extra []string
	for t := range state {
		if !seen[t] {
			extra = append(extra, t)
		}
	}
	sort.Strings(extra)
	return append(out, extra...)
}

// universeSlots — стратегии, в чьей вселенной тикер, в порядке приоритета.
func (s *service) universeSlots(ticker string) []*slot {
	var out []*slot
	for _, sl := range s.slots {
		if slices.Contains(sl.strat.Tickers(), ticker) {
			out = append(out, sl)
		}
	}
	return out
}
```

В `live.go`:

```go
func (s *service) slotByName(name string) *slot {
	for _, sl := range s.slots {
		if sl.strat.Name() == name {
			return sl
		}
	}
	return nil
}
```

- [ ] **Step 5: Новый цикл `pass`** (загрузка shares/held/state/List — как раньше; `pc` создаётся так же):

```go
	for _, ticker := range s.passTickers(state) {
		entry, hasState := state[ticker]

		// Кто вправе действовать по тикеру. Запись в стейте — владелец из неё; иначе —
		// стратегии, в чьей вселенной тикер, по приоритету.
		var cands []*slot
		if hasState {
			owner := s.slotByName(adapter.Owner(entry))
			if owner == nil {
				s.slots[0].alert(ticker, fmt.Sprintf("позиция принадлежит неизвестной стратегии %q — сопровождение пропущено", adapter.Owner(entry)))
				continue
			}
			cands = []*slot{owner}
		} else {
			cands = s.universeSlots(ticker)
		}
		// Незарегистрированный тикер — алерт от той стратегии, в чьей он вселенной (или
		// чья запись), и пропуск; свечи по нему не запрашиваются.
		type cand struct {
			sl  *slot
			dec adapter.Decider
		}
		var ready []cand
		for _, sl := range cands {
			dec, ok := sl.strat.Decider(ticker)
			if !ok {
				sl.alert(ticker, "тикер не зарегистрирован в "+sl.strat.Name()+" — пропуск")
				continue
			}
			ready = append(ready, cand{sl, dec})
		}
		if len(ready) == 0 {
			continue
		}

		sh, ok := shares[ticker]
		if !ok || !sh.Trading {
			if hasState {
				ready[0].sl.alert(ticker, "инструмент недоступен для торгов — сопровождение пропущено")
			}
			logger.ErrorContext(ctx, fmt.Sprintf("%s: %s not tradable, skip", ready[0].sl.strat.Name(), ticker))
			continue
		}
		pos, isHeld := held[sh.ID]

		if isHeld || hasState {
			// Позиция без стейта на тикере нескольких стратегий: чья она — из API не узнать.
			// Угаданный владелец повёл бы её чужим стопом и чужими выходами; реконструкция
			// и чужие стоп-заявки остаются нетронутыми до ручной разметки.
			if !hasState && len(s.universeSlots(ticker)) > 1 {
				s.slots[0].alert(ticker, "позиция без стейта на общем тикере, владелец неизвестен — нужна ручная разметка стейта; сопровождение пропущено")
				continue
			}
			c := ready[0]
			md, ok := s.assemble(ctx, c.sl, ticker, sh, c.dec, now, true)
			if !ok {
				continue
			}
			if perr := s.manage(ctx, pc, c.sl, ticker, sh, c.dec, md, pos, isHeld); perr != nil {
				c.sl.alert(ticker, "пасс по тикеру прерван: "+perr.Error())
				logger.ErrorContext(ctx, fmt.Sprintf("%s: %s pass: %v", c.sl.strat.Name(), ticker, perr))
				failed = append(failed, ticker)
			}
			continue
		}

		// Свободный тикер: стратегии по приоритету; первая с BUY входит, остальных на
		// этом баре не спрашивают.
		for _, c := range ready {
			md, ok := s.assemble(ctx, c.sl, ticker, sh, c.dec, now, false)
			if !ok {
				continue
			}
			signaled, perr := s.buy(ctx, pc, c.sl, ticker, sh, c.dec, md)
			if perr != nil {
				c.sl.alert(ticker, "пасс по тикеру прерван: "+perr.Error())
				logger.ErrorContext(ctx, fmt.Sprintf("%s: %s pass: %v", c.sl.strat.Name(), ticker, perr))
				failed = append(failed, ticker)
			}
			if signaled {
				break
			}
		}
	}
```

и хелпер сборки (тела алертов и логов — из прежнего пасса):

```go
// assemble собирает MarketData окном Lookback стратегии и отбрасывает протухший бар.
// Своя сборка на стратегию, а не срез общей: окно ровно то, что видит движок бэктеста.
// watched — по тикеру есть позиция или запись: только тогда пропуск будит владельца.
func (s *service) assemble(ctx context.Context, sl *slot, ticker string, sh *imodel.Share,
	dec adapter.Decider, now time.Time, watched bool) (strategy.MarketData, bool) {

	md, err := marketdata.Assemble(ctx, s.market, sh.ID, dec.Lookback(), now)
	if err != nil {
		if watched {
			sl.alert(ticker, "рыночные данные недоступны — сопровождение пропущено: "+err.Error())
		}
		logger.ErrorContext(ctx, fmt.Sprintf("%s: %s marketdata: %v", sl.strat.Name(), ticker, err))
		return strategy.MarketData{}, false
	}
	if n := len(md.Times); n == 0 || now.Sub(md.Times[n-1]) > maxBarAge {
		if watched {
			sl.alert(ticker, "последний завершённый бар протух — сопровождение пропущено")
		}
		logger.ErrorContext(ctx, fmt.Sprintf("%s: %s stale bar, skip", sl.strat.Name(), ticker))
		return strategy.MarketData{}, false
	}
	return md, true
}
```

Текст алерта незарегистрированного тикера у rsi_pullback обязан остаться с подстрокой «не зарегистрирован» и тикером (`TestUnknownTickerAlertsAndSkips`).

- [ ] **Step 6: Бумажный режим в `buy`** — сразу после успешного `sizing.Lots` и до `sl.exec.Buy`:

```go
	// Бумажный вход при боевой соседке — только уведомление. Запись в стейте заняла бы
	// тикер до чистки по freshEntryGrace и заблокировала бы боевой вход другой стратегии.
	// Когда боевых стратегий на счёте нет вовсе, работает прежний dry-run: стейт пишется.
	if !sl.strat.TradeEnabled() && s.anyLive {
		sl.strat.Notify(notifier.Entry(ticker, sig.Price, lots, lots*int64(sh.Lot), true))
		return true, nil
	}
```

- [ ] **Step 7: `Announce` для всех слотов**

```go
func (s *service) Announce() {
	for _, sl := range s.slots {
		sl.strat.Notify(notifier.Startup(sl.strat.Label(), sl.strat.Tickers(), !sl.strat.TradeEnabled()))
	}
}
```

- [ ] **Step 8: Прогнать весь пакет**

Run: `go test ./internal/service/trading_strategy/rsi_pullback/live/... -count=1 -race`
Expected: PASS, включая все прежние тесты `service_test.go` без правок.

- [ ] **Step 9: Мутационные проверки** (каждую вернуть после проверки):
  - в пассе не делать `break` после `signaled` → падает `TestPullbackWinsTheSameBarBuy`;
  - убрать проверку `len(s.universeSlots(ticker)) > 1` → падает `TestHeldSharedTickerWithoutStateIsNotReconstructed`;
  - убрать ветку бумажного режима из шага 6 → падает `TestPaperGuestEntryDoesNotClaimTickerWhenPullbackIsLive`;
  - в `passTickers` не добавлять тикеры стейта → падает `TestPositionOutsideUniverseIsStillManagedByOwner`.

- [ ] **Step 10: Коммит**

```bash
git commit -m "feat(rsi_pullback): раннер счёта ведёт несколько стратегий — владение тикером и приоритет входа

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>" -- internal/service/trading_strategy/rsi_pullback/live
```

---

### Task 5: Конфиг rsi_zone, тема Telegram, env-файлы

**Files:**
- Create: `internal/config/rsi_zone.go`
- Create: `internal/config/rsi_zone_test.go`
- Modify: `internal/config/config.go` (поле `RSIZone *RSIZoneConfig`)
- Modify: `internal/config/telegram_client.go` (`TopicRSIZone int \`config:"TELEGRAM_TOPIC_RSI_ZONE"\``)
- Modify: `internal/app/init_config.go` (`RSIZone: config.NewRSIZoneConfig(),`)
- Modify: `env/prod.env`, `env/prod.env.example`, `env/local.env.example`

**Interfaces:**
- Produces: `config.RSIZoneConfig{Tickers []string; BuyPct float64; TradeEnabled, NotifyEnabled bool}`, `config.NewRSIZoneConfig() *RSIZoneConfig`, `(*RSIZoneConfig).Enabled() bool`, `config.Config.RSIZone`, `config.TelegramClient.TopicRSIZone`.

- [ ] **Step 1: Тесты `rsi_zone_test.go`** (стиль — как в `rsi_pullback_test.go`, посмотреть `TestRSIPullbackConfigLoadsWithoutItsEnvVars` и повторить загрузку через confita):

```go
// Без переменных rsi_zone конфиг грузится, стратегия выключена, торговля выключена.
func TestRSIZoneConfigLoadsWithoutItsEnvVars(t *testing.T) { /* по образцу rsi_pullback */ }

func TestNewRSIZoneConfig_Defaults(t *testing.T) {
	c := NewRSIZoneConfig()
	if c.Enabled() || c.TradeEnabled || c.NotifyEnabled || c.BuyPct != 5 {
		t.Fatalf("дефолты = %+v, want выключено, BuyPct 5", c)
	}
}

// RSI_ZONE_TICKERS= (пустое значение в env) даёт [""] — это не вселенная.
func TestRSIZoneEnabledIgnoresBlankTickers(t *testing.T) {
	c := NewRSIZoneConfig()
	c.Tickers = []string{"", " "}
	if c.Enabled() {
		t.Fatal("вселенная из пустых строк включила стратегию")
	}
	c.Tickers = []string{"AFKS"}
	if !c.Enabled() {
		t.Fatal("вселенная AFKS не включила стратегию")
	}
}

// env/prod.env и оба образца несут одну и ту же вселенную rsi_zone (прод-файл уже дважды
// молча терял тикеры rsi_pullback — см. TestRSIPullbackTickersMatchEnvFiles).
func TestRSIZoneTickersMatchAcrossEnvFiles(t *testing.T) {
	want := "AFKS,BAZA,DIAS,DOMRF,LENT"
	for _, path := range []string{"../../env/prod.env", "../../env/prod.env.example", "../../env/local.env.example"} {
		// прочитать файл, найти строку с префиксом "RSI_ZONE_TICKERS=", сравнить значение с want
	}
}
```

Последний тест дописать целиком по образцу `TestRSIPullbackTickersMatchEnvFiles` (тот же парсер строки).

- [ ] **Step 2: Прогнать — падает компиляция**

Run: `go test ./internal/config/ -run RSIZone -v`
Expected: FAIL `undefined: NewRSIZoneConfig`

- [ ] **Step 3: `rsi_zone.go`**

```go
package config

import "strings"

// RSIZoneConfig подключает rsi_zone к живому раннеру счёта rsi_pullback: счёт, токен и
// расписание у стратегий общие (RSI_PULLBACK_*), здесь — только своё (docs/rsi_zone/live.md).
// Ни одна переменная не required — по той же причине, что у RSIPullbackConfig: ошибка
// загрузки конфига оборвала бы InitApp целиком.
type RSIZoneConfig struct {
	Tickers       []string `config:"RSI_ZONE_TICKERS,backend=env"`
	BuyPct        float64  `config:"RSI_ZONE_BUY_PCT,backend=env"`
	TradeEnabled  bool     `config:"RSI_ZONE_TRADE_ENABLED,backend=env"`
	NotifyEnabled bool     `config:"RSI_ZONE_NOTIFY_ENABLED,backend=env"`
}

// NewRSIZoneConfig: вселенная по умолчанию пуста — стратегия не подключается, пока её не
// включат в env; торговля выключена, чтобы отсутствующий флаг не выставил ордер.
func NewRSIZoneConfig() *RSIZoneConfig {
	return &RSIZoneConfig{BuyPct: 5}
}

// Enabled — задана ли вселенная. Пустые строки не считаются: RSI_ZONE_TICKERS= в env
// приходит как [""].
func (c *RSIZoneConfig) Enabled() bool {
	for _, t := range c.Tickers {
		if strings.TrimSpace(t) != "" {
			return true
		}
	}
	return false
}
```

- [ ] **Step 4: Поля в `Config`, `TelegramClient`, `init_config.go`** — по образцу `RSIPullback`/`TopicRSIPullback`.

- [ ] **Step 5: env-файлы.** В `env/prod.env` после блока RSI Pullback:

```
# RSI Zone на счёте rsi_pullback (docs/rsi_zone/live.md): счёт, токен и расписание общие.
# Выкат — сначала бумажный режим при боевом rsi_pullback: бумажные входы zone приходят
# только уведомлениями и тикер не занимают.
RSI_ZONE_TICKERS=AFKS,BAZA,DIAS,DOMRF,LENT
RSI_ZONE_BUY_PCT=5
RSI_ZONE_TRADE_ENABLED=false
RSI_ZONE_NOTIFY_ENABLED=true
```

`TELEGRAM_TOPIC_RSI_ZONE` в `prod.env` не добавлять, пока владелец не завёл тему (без неё сообщения уходят в General общим механизмом `SendMessageToTopic`). В `prod.env.example` и `local.env.example` — тот же блок плюс пустая `TELEGRAM_TOPIC_RSI_ZONE=` рядом с `TELEGRAM_TOPIC_RSI_PULLBACK`. В `local.env.example` поправить комментарий «Отдельный счёт и токен: reversion тоже торгует UGLD…» — дописать, что на этом счёте торгует и rsi_zone.

- [ ] **Step 6: Прогнать**

Run: `go test ./internal/config/ ./internal/app/... -count=1`
Expected: PASS

- [ ] **Step 7: Коммит**

```bash
git commit -m "feat(config): конфиг rsi_zone на общем счёте и тема Telegram

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>" -- internal/config internal/app/init_config.go env/prod.env env/prod.env.example env/local.env.example
```

---

### Task 6: Адаптер rsi_zone — `rsi_zone/live`

**Files:**
- Create: `internal/service/trading_strategy/rsi_zone/live/registry.go`
- Create: `internal/service/trading_strategy/rsi_zone/live/registry_test.go`
- Create: `internal/service/trading_strategy/rsi_zone/live/strategy.go`
- Create: `internal/service/trading_strategy/rsi_zone/live/strategy_test.go`

**Interfaces:**
- Consumes: `adapter.*` (Task 1), `rebuild.*` (Task 2), `config.RSIZoneConfig` (Task 5), `core.StopLevel`, `core.NewWithParams`, `core.DefaultParams` (`internal/service/trading_strategy/rsi_zone/strategy/core`).
- Produces: `live.Name = "rsi_zone"`, `live.ParamsFor(ticker) (core.Params, bool)`, `live.StrategyFor(ticker) (*core.Strategy, bool)`, `live.New(cfg *config.RSIZoneConfig, tg telegram.Client) *Strategy` (реализует `adapter.Strategy`).

- [ ] **Step 1: Тесты реестра `registry_test.go`**

```go
package live

// Каждый тикер из RSI_ZONE_TICKERS env/prod.env зарегистрирован — иначе раннер его молча
// не торгует (алерт «не зарегистрирован» на каждом пассе).
func TestEveryProdTickerIsRegistered(t *testing.T) { /* прочитать ../../../../../env/prod.env, строку RSI_ZONE_TICKERS=, для каждого тикера ParamsFor ok */ }

// Тикер на дефолтах ядра в боевой вселенной — только именным исключением с причиной.
var baselineByDesignTickers = map[string]string{
	"BAZA": "калибровка 2026-09-29: точка большинства совпала с дефолтами ядра — литерала, отличного от них, нет (док пакета strategy/baza)",
	"LENT": "калибровка 2026-09-29: большинства нет ни по одному полю, заведён на дефолтах решением владельца (док пакета strategy/lent)",
}

func TestBaselineTickersInProdAreNamedExceptions(t *testing.T) {
	// для каждого тикера prod-вселенной: p == core.DefaultParams() допустимо только для
	// ключей baselineByDesignTickers; и наоборот — исключение с p != DefaultParams — ошибка.
}

func TestSBERIsNotRegistered(t *testing.T) {
	if _, ok := ParamsFor("SBER"); ok {
		t.Fatal("SBER планку не взял и в живой реестр rsi_zone не заводится")
	}
}
```

Формулировки причин BAZA/LENT сверить с doc-комментариями пакетов `rsi_zone/strategy/baza` и `rsi_zone/strategy/lent` и памятью проекта; если там сказано иначе — взять оттуда. Тела тестов дописать полностью.

- [ ] **Step 2: Тесты адаптера `strategy_test.go`**

```go
func TestDesiredStopIsFrozenAtEntry(t *testing.T) {
	s := New(&config.RSIZoneConfig{Tickers: []string{"AFKS"}}, nil)
	p, _ := ParamsFor("AFKS")
	e := statestore.Entry{EntryPrice: 100, EntryATR: 4, MaxFav: 150}
	level, reason := s.DesiredStop("AFKS", e)
	if want := core.StopLevel(p, 100, 4); level != want || reason != "SL" {
		t.Fatalf("DesiredStop = %v/%q, want %v/SL (MaxFav не участвует — трейла нет)", level, reason, want)
	}
}

func TestDesiredStopDisabledReturnsNoReason(t *testing.T) {
	// EntryATR = 0 → core.StopLevel = 0 → ("", 0): стоп-заявка не ставится.
}

func TestDeciderUsesTickerParams(t *testing.T) {
	// Decider("DIAS").Lookback() == core.NewWithParams("DIAS", dias.DefaultParams()).Lookback()
	// Decider("НЕТ") → ok=false и nil-интерфейс.
}

func TestReconstructRebuildsEntryWithoutTarget(t *testing.T) {
	// моки: reconstruct/mocks TradesClient (или свой минимальный фейк adapter.TradesClient) —
	// одна BUY; candlemocks дневки как в rebuild_test. Ожидание: Strategy пусто (проставляет
	// раннер), TakeProfit 0, EntryPrice = PurchasePrice, MaxFav = PurchasePrice, EntryATR > 0.
}

func TestReconstructFailsWithoutBuy(t *testing.T) {
	// только SELL → ошибка, содержащая тикер.
}

func TestNotifyIsLabelledAndRespectsSwitch(t *testing.T) {
	// NotifyEnabled=false → tg без ожиданий; true → сообщение начинается с "[RSI Zone] ".
}
```

Тела дописать полностью; моки Telegram — `tgmocks.NewMockClient(t)`.

- [ ] **Step 3: Прогнать — падает** (пакета нет)

Run: `go test ./internal/service/trading_strategy/rsi_zone/live/ -v`
Expected: FAIL

- [ ] **Step 4: `registry.go`**

```go
// Package live подключает rsi_zone к живому раннеру счёта rsi_pullback
// (rsi_pullback/live) адаптером livecore/adapter.Strategy. Механика — docs/rsi_zone/live.md.
package live

import (
	"tinvest/internal/service/trading_strategy/rsi_zone/strategy/afks"
	"tinvest/internal/service/trading_strategy/rsi_zone/strategy/baza"
	"tinvest/internal/service/trading_strategy/rsi_zone/strategy/core"
	"tinvest/internal/service/trading_strategy/rsi_zone/strategy/dias"
	"tinvest/internal/service/trading_strategy/rsi_zone/strategy/domrf"
	"tinvest/internal/service/trading_strategy/rsi_zone/strategy/lent"
)

// paramsByTicker — тикеры, которые раннер знает. SBER сюда не заводится: калибровка планку
// не взяла. Торгуют только перечисленные в RSI_ZONE_TICKERS.
var paramsByTicker = map[string]core.Params{
	afks.Ticker:  afks.DefaultParams(),
	baza.Ticker:  baza.DefaultParams(),
	dias.Ticker:  dias.DefaultParams(),
	domrf.Ticker: domrf.DefaultParams(),
	lent.Ticker:  lent.DefaultParams(),
}

func ParamsFor(ticker string) (core.Params, bool) {
	p, ok := paramsByTicker[ticker]
	return p, ok
}

func StrategyFor(ticker string) (*core.Strategy, bool) {
	p, ok := paramsByTicker[ticker]
	if !ok {
		return nil, false
	}
	return core.NewWithParams(ticker, p), true
}
```

(Если в пакетах тикеров константа называется не `Ticker` — свериться с `internal/service/backtest/rsi_zone_registry.go`, там те же идентификаторы.)

- [ ] **Step 5: `strategy.go`**

```go
package live

import (
	"context"
	"errors"
	"fmt"

	"tinvest/internal/config"
	"tinvest/internal/service/trading_strategy/livecore/adapter"
	"tinvest/internal/service/trading_strategy/livecore/rebuild"
	"tinvest/internal/service/trading_strategy/livecore/statestore"
	"tinvest/internal/service/trading_strategy/rsi_zone/strategy/core"
	"tinvest/pkg/client/telegram"
	"tinvest/pkg/logger"
)

// Name — имя стратегии в стейте счёта.
const Name = "rsi_zone"

const alertLabel = "RSI Zone"

// Strategy — rsi_zone как гость раннера счёта.
type Strategy struct {
	cfg *config.RSIZoneConfig
	tg  telegram.Client
}

var _ adapter.Strategy = (*Strategy)(nil)

func New(cfg *config.RSIZoneConfig, tg telegram.Client) *Strategy {
	return &Strategy{cfg: cfg, tg: tg}
}

func (s *Strategy) Name() string       { return Name }
func (s *Strategy) Label() string      { return alertLabel }
func (s *Strategy) Tickers() []string  { return s.cfg.Tickers }
func (s *Strategy) BuyPct() float64    { return s.cfg.BuyPct }
func (s *Strategy) TradeEnabled() bool { return s.cfg.TradeEnabled }

func (s *Strategy) Decider(ticker string) (adapter.Decider, bool) {
	st, ok := StrategyFor(ticker)
	if !ok {
		return nil, false
	}
	return st, true
}

// DesiredStop — стоп, замороженный на входе: у rsi_zone нет трейла, и уровень не меняется
// за всю жизнь позиции. Раннер переставляет заявку только при расхождении размера или после
// внешнего снятия.
func (s *Strategy) DesiredStop(ticker string, e statestore.Entry) (float64, string) {
	p, ok := ParamsFor(ticker)
	if !ok {
		return 0, ""
	}
	level := core.StopLevel(p, e.EntryPrice, e.EntryATR)
	if level <= 0 {
		return 0, ""
	}
	return level, "SL"
}

// Reconstruct поднимает вход по API: средняя цена брокера, последняя BUY-сделка, дневной
// ATR по будним дневкам до дня входа. Цели и трейла у стратегии нет — TakeProfit 0,
// MaxFav равен цене входа и ядром не читается.
func (s *Strategy) Reconstruct(ctx context.Context, in adapter.ReconstructInput) (statestore.Entry, error) {
	p, ok := ParamsFor(in.Ticker)
	if !ok {
		return statestore.Entry{}, fmt.Errorf("rsi_zone reconstruct: %s не зарегистрирован", in.Ticker)
	}
	entryTime, err := rebuild.LastBuyTime(ctx, in.Trades, in.AccountID, in.InstrumentID, in.Now)
	switch {
	case errors.Is(err, rebuild.ErrNoBuyFill):
		return statestore.Entry{}, fmt.Errorf("rsi_zone reconstruct: no BUY fill found for %s", in.Ticker)
	case err != nil:
		return statestore.Entry{}, fmt.Errorf("rsi_zone reconstruct: %w", err)
	}
	atr, err := rebuild.WeekdayDailyATRBefore(ctx, in.Candles, in.InstrumentID, entryTime, p.DailyATRPeriod)
	if err != nil {
		return statestore.Entry{}, fmt.Errorf("rsi_zone reconstruct: %w", err)
	}
	if atr <= 0 {
		// Нулевой ATR — стоп вплотную к цене входа; лучше отказ и решение человека.
		return statestore.Entry{}, fmt.Errorf("rsi_zone reconstruct: %s: cannot rebuild daily ATR as of %s", in.Ticker, entryTime.Format("2006-01-02T15:04:05Z07:00"))
	}
	return statestore.Entry{
		Ticker:     in.Ticker,
		EntryTime:  entryTime,
		EntryPrice: in.PurchasePrice,
		EntryATR:   atr,
		MaxFav:     in.PurchasePrice,
	}, nil
}

// Notify шлёт в тему rsi_zone с меткой стратегии: сообщения входа/выхода/стопа из общего
// notifier метки не несут. Сбой доставки — ERROR-лог, как у rsi_pullback.
func (s *Strategy) Notify(msg string) {
	if !s.cfg.NotifyEnabled || s.tg == nil {
		return
	}
	if err := s.tg.SendMessage("[" + alertLabel + "] " + msg); err != nil {
		logger.ErrorContext(context.Background(), fmt.Sprintf("rsi_zone: уведомление не доставлено: %v", err))
	}
}
```

- [ ] **Step 6: Прогнать**

Run: `go test ./internal/service/trading_strategy/rsi_zone/... -count=1`
Expected: PASS

- [ ] **Step 7: Коммит**

```bash
git commit -m "feat(rsi_zone): адаптер живого раннера — реестр, стоп, реконструкция входа

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>" -- internal/service/trading_strategy/rsi_zone/live
```

---

### Task 7: Подключение в приложении и сверка `pullparity -strategy`

**Files:**
- Modify: `internal/service_provider/service.go` (`GetRSIPullbackLiveService`)
- Modify: `internal/service_provider/client.go` (`GetRSIZoneSender`)
- Modify: `cmd/pullparity/main.go`
- Modify: `cmd/pullparity/main_test.go`

**Interfaces:**
- Consumes: `rsizonelive.New` (Task 6), `config.Config.RSIZone`, `TopicRSIZone` (Task 5), вариадик `NewService` (Task 3).

- [ ] **Step 1: `GetRSIZoneSender`** в `client.go` (рядом с `GetRSIPullbackSender`):

```go
func (s *ServiceProvider) GetRSIZoneSender() (telegram.Client, error) {
	return s.topicSender(s.appConfig.TelegramClient.TopicRSIZone, "rsi_zone")
}
```

- [ ] **Step 2: Гость в `GetRSIPullbackLiveService`**

```go
		var guests []adapter.Strategy
		// rsi_zone торгует на том же счёте (docs/rsi_zone/live.md): тот же gRPC-клиент и
		// файл стейта, своя тема Telegram. Пустая вселенная — стратегия не подключается.
		if zone := serviceProvider.appConfig.RSIZone; zone.Enabled() {
			zoneTG, _ := serviceProvider.GetRSIZoneSender()
			guests = append(guests, rsizonelive.New(zone, zoneTG))
		}
		serviceProvider.service.rsiPullbackLiveService = rsipullbacklive.NewService(
			// ...прежние аргументы...
			serviceProvider.appConfig.RSIPullback,
			guests...,
		)
```

Doc-комментарий метода переписать: раннер счёта, rsi_pullback — отдельный от reversion счёт, rsi_zone — гость на нём.

- [ ] **Step 3: `pullparity -strategy`.** Тест на резолвер в `main_test.go`:

```go
func TestResolveDeciderPicksStrategyRegistry(t *testing.T) {
	if _, err := resolveDecider("rsi_pullback", "GAZP"); err != nil {
		t.Fatalf("rsi_pullback GAZP: %v", err)
	}
	if _, err := resolveDecider("rsi_zone", "AFKS"); err != nil {
		t.Fatalf("rsi_zone AFKS: %v", err)
	}
	if _, err := resolveDecider("rsi_zone", "GAZP"); err == nil {
		t.Fatal("GAZP не в реестре rsi_zone — ждали ошибку")
	}
	if _, err := resolveDecider("nope", "GAZP"); err == nil {
		t.Fatal("неизвестная стратегия — ждали ошибку")
	}
}
```

Run: `go test ./cmd/pullparity/ -run ResolveDecider -v` — Expected: FAIL (`undefined: resolveDecider`)

Реализация в `main.go`: флаг `strategyName = flag.String("strategy", "rsi_pullback", "rsi_pullback | rsi_zone")`, поле `strategy string` в `runCfg`, в `run` — `st, err := resolveDecider(cfg.strategy, ticker)`; `checkTicker` принимает `st adapter.Decider` вместо `*core.Strategy`; импорт `core` удаляется.

```go
// resolveDecider берёт ядро тикера из живого реестра выбранной стратегии — того же, из
// которого его берёт раннер.
func resolveDecider(strategyName, ticker string) (adapter.Decider, error) {
	switch strategyName {
	case "rsi_pullback":
		if st, ok := live.StrategyFor(ticker); ok {
			return st, nil
		}
	case "rsi_zone":
		if st, ok := rsizonelive.StrategyFor(ticker); ok {
			return st, nil
		}
	default:
		return nil, fmt.Errorf("unknown -strategy %q (rsi_pullback | rsi_zone)", strategyName)
	}
	return nil, fmt.Errorf("%s: not registered in %s live registry — add calibrated params before checking parity", ticker, strategyName)
}
```

Doc-комментарий пакета дополнить строкой про `-strategy`.

- [ ] **Step 4: Прогнать**

Run: `go build ./internal/... ./pkg/... ./cmd/... && go test ./cmd/pullparity/ ./internal/service_provider/... ./internal/app/... -count=1`
Expected: PASS

- [ ] **Step 5: Живая сверка по кэшу** (нужен `env/token.env`; если кэша свечей по тикеру нет, утилита его докачает):

Run: `go run ./cmd/pullparity -strategy rsi_zone -tickers AFKS,BAZA,DIAS,DOMRF,LENT -months 24`
Expected: по каждому тикеру `0 расхождений`, exit 0. Любое расхождение — СТОП, задача не закрывается, эскалация владельцу.
Регрессия: `go run ./cmd/pullparity -tickers GAZP -months 24` — `0 расхождений`.

- [ ] **Step 6: Коммит**

```bash
git commit -m "feat(rsi_zone): подключение к раннеру счёта и сверка pullparity -strategy rsi_zone

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>" -- internal/service_provider cmd/pullparity
```

---

### Task 8: Документация и финальный гейт

**Files:**
- Create: `docs/rsi_zone/live.md`
- Modify: `docs/rsi_pullback/live.md`
- Modify: `docs/rsi_zone/strategy.md`
- Modify: `CLAUDE.md`

- [ ] **Step 1: `docs/rsi_zone/live.md`** — разделы (механика, без результатов калибровок):
  1. Что делает: гость раннера счёта rsi_pullback; ссылка на `docs/rsi_pullback/live.md` §§1–7 как на общую механику (пасс, стоп-заявки, grace, pendingExit, реконструкция).
  2. Общий счёт: владение тикером (поле `strategy`, пусто = rsi_pullback), приоритет rsi_pullback на одном баре, тикер вне вселенной ведётся владельцем до выхода, позиция без стейта на общем тикере — алерт и ручная разметка (пример JSON-записи с `"strategy": "rsi_zone"`), неизвестный владелец — алерт.
  3. Отличия от rsi_pullback: стоп фиксирован на входе (`core.StopLevel`), нет цели и трейла; выходы SL (биржевая заявка) и RSI (market SELL на закрытии бара).
  4. Бумажный режим при боевом rsi_pullback: только уведомления о входах, стейт не пишется, выходы не ведутся.
  5. Деньги: `BuyPct` каждой стратегии от полной стоимости счёта с ограничением кэшем, покупки внутри пасса последовательны; при `RSI_PULLBACK_BUY_PCT=50` две позиции занимают весь счёт.
  6. Конфигурация: таблица `RSI_ZONE_*` и `TELEGRAM_TOPIC_RSI_ZONE` (из Task 5), общие `RSI_PULLBACK_ACCOUNT_ID/TOKEN/SCHEDULE`.
  7. Сверка: `go run ./cmd/pullparity -strategy rsi_zone -tickers ... -months 24`, приёмка — ноль расхождений.
  8. Порядок выката — из §9 спеки.
- [ ] **Step 2: `docs/rsi_pullback/live.md`** — абзац «**Почему отдельный счёт.**» переписать: счёт отдельный от reversion (причина та же), но на нём же торгует rsi_zone — ссылка на `docs/rsi_zone/live.md`. В §1 добавить строку, что раннер ведёт несколько стратегий через `livecore/adapter`. В §7 — флаг `-strategy`.
- [ ] **Step 3: `docs/rsi_zone/strategy.md`** — строку «Только бэктест: живого раннера нет.» заменить на «Живой раннер — гость раннера счёта rsi_pullback: [live.md](live.md).»
- [ ] **Step 4: `CLAUDE.md`** — в описании `rsi_zone` заменить «backtest-only» на упоминание live-раннера: «live — гость раннера счёта rsi_pullback (`rsi_zone/live`, общий счёт, владение тикером; docs/rsi_zone/live.md), сверка — `cmd/pullparity -strategy rsi_zone`».
- [ ] **Step 5: Финальный гейт**

Run: `./bin/mage ci`
Expected: lint + `go test -race ./...` + проверка дрейфа моков — всё зелёное. Если дрейф моков — `./bin/mage mocks` и включить результат в коммит.

- [ ] **Step 6: Коммит**

```bash
git commit -m "docs(rsi_zone): живой раннер на общем счёте

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>" -- docs/rsi_zone docs/rsi_pullback/live.md CLAUDE.md
```
