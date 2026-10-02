# gap_fade live Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Run gap_fade as a third guest adapter of the rsi_pullback account runner, in paper mode first.

**Architecture:** The runner gets three generic hooks: a frozen `StopLoss` in the state entry, an optional `EntryWindow` that skips data assembly outside a strategy's entry window, and an optional `EntryFilter` veto after a BUY signal. A new package `gap_fade/live` implements `adapter.Strategy` plus both optional interfaces, using one shared parameter literal for every ticker and the backtest's dividend ex-day rule. Wiring adds a config block, a Telegram topic, the pullparity registry, and env lines.

**Tech Stack:** Go 1.25, testify mocks (mockery v2), heetch/confita, Tinkoff Invest gRPC.

**Spec:** `docs/superpowers/specs/2026-10-02-gap-fade-live-design.md`

## Global Constraints

- Code comments in English or Russian matching the surrounding file (runner and livecore files use Russian comments; keep that). Docs in Russian, mechanics only, no run results, profit factors or per-ticker numbers.
- Strategy name in state: `gap_fade`; label `Gap Fade`; Telegram prefix `[Gap Fade] `.
- Shared params: `GapATR 0.7, MaxGapATR 3.0, EntryBar 0, TargetFill 0.75, StopDailyATR 1.0, DailyATRPeriod 14, EODHour 18, EODMinute 30`.
- Adapter order: `[rsi_pullback, rsi_zone, gap_fade]`.
- Config vars (none `required`): `GAP_FADE_TICKERS` (default empty), `GAP_FADE_BUY_PCT` (default 5), `GAP_FADE_TRADE_ENABLED` (default false), `GAP_FADE_NOTIFY_ENABLED` (default false), `TELEGRAM_TOPIC_GAP_FADE`.
- Entry window: weekday MSK and pass time strictly before 10:00 MSK.
- Dividend query window: `[last bar time − 30 days, last bar time + 7 days]`; fail closed on API error.
- Reconstruct: `StopLoss = EntryPrice − StopDailyATR·EntryATR`, `TakeProfit = 0`, `MaxFav = EntryPrice`; zero ATR is an error.
- Commits end with `Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>`. Never push. Work on branch `feat/gap-fade-live`.
- Build check: `go build ./internal/... ./pkg/... ./cmd/...` (never `go build ./...` — magefiles has no main).
- After changing a mocked interface run `./bin/mage mocks` (no mocked interface is expected to change in this plan).

## Review Focus

1. A gap_fade position held past 10:00 must still be managed — `EntryWindow` must gate only free-ticker entries. Pinned by Task 4's EOD-at-19:01 end-to-end test.
2. A state file written before this change (no `stopLoss` key) must load unchanged for rsi_pullback/rsi_zone entries. Pinned by Task 1's legacy JSON test.
3. A filtered or failed-filter BUY must not consume the bar: the next strategy by priority is still asked on that bar. Pinned by Task 1.
4. Dividend with an old `last_buy_date` (before the 64-bar window) must not block today's entry. Pinned by Task 3.
5. A nil dividends client must block entries (fail closed), not panic. Pinned by Task 3.

---

### Task 1: Runner hooks — frozen StopLoss, EntryWindow, EntryFilter

**Files:**
- Modify: `internal/service/trading_strategy/livecore/adapter/adapter.go`
- Modify: `internal/service/trading_strategy/livecore/statestore/statestore.go`
- Test: `internal/service/trading_strategy/livecore/statestore/statestore_test.go`
- Modify: `internal/service/trading_strategy/rsi_pullback/live/pass.go`
- Test: `internal/service/trading_strategy/rsi_pullback/live/shared_test.go`

**Interfaces:**
- Produces:
  - `statestore.Entry.StopLoss float64` (`json:"stopLoss,omitempty"`)
  - `adapter.EntryWindow interface { EntryPossible(now time.Time) bool }`
  - `adapter.EntryFilter interface { EntryBlocked(ctx context.Context, ticker, instrumentID string, md strategy.MarketData) (reason string, err error) }`
  - pass writes `Entry.StopLoss = sig.StopLoss` on entry and passes `Position.StopLoss = entry.StopLoss` on manage.

- [ ] **Step 1: Write failing statestore tests**

Append to `statestore_test.go` (keep its existing imports; add `os`, `path/filepath` if missing):

```go
// StopLoss — замороженный уровень стопа gap_fade — переживает перезапуск.
func TestStopLossRoundTrips(t *testing.T) {
	path := filepath.Join(t.TempDir(), "state.json")
	s := New(path)
	if err := s.Save(map[string]Entry{"GAZP": {Ticker: "GAZP", StopLoss: 78.5}}); err != nil {
		t.Fatalf("Save: %v", err)
	}
	got, err := s.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if got["GAZP"].StopLoss != 78.5 {
		t.Fatalf("StopLoss = %v, want 78.5", got["GAZP"].StopLoss)
	}
}

// Стейт, записанный до появления поля, читается как прежде: StopLoss = 0.
func TestLegacyStateWithoutStopLossLoads(t *testing.T) {
	path := filepath.Join(t.TempDir(), "state.json")
	legacy := `{"GAZP":{"ticker":"GAZP","entryTime":"2026-03-10T07:01:00+03:00","entryPrice":100,"entryATR":10,"maxFav":100,"quantity":10,"takeProfit":120}}`
	if err := os.WriteFile(path, []byte(legacy), 0o600); err != nil {
		t.Fatal(err)
	}
	got, err := New(path).Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	e := got["GAZP"]
	if e.StopLoss != 0 || e.TakeProfit != 120 || e.EntryPrice != 100 {
		t.Fatalf("legacy entry = %+v", e)
	}
}
```

Before writing, read `statestore.go` `Load`/`Save` to confirm the on-disk shape is a JSON object keyed by ticker. If it is wrapped (e.g. `{"positions":{...}}`), adapt the legacy literal to that shape — the point is a real pre-change file.

- [ ] **Step 2: Run, expect compile failure**

Run: `go test ./internal/service/trading_strategy/livecore/statestore/ -run 'StopLoss' -v`
Expected: FAIL — `unknown field StopLoss`.

- [ ] **Step 3: Add the field**

In `statestore.go`, `Entry`, right after the `TakeProfit` field:

```go
	// StopLoss — уровень стопа, замороженный на входе от цены сигнала (gap_fade меряет стоп от
	// close сигнального бара, а не от цены исполнения, поэтому из EntryPrice его не вывести).
	// Стратегии, считающие стоп от входа, его не читают — omitempty оставляет их записи прежними.
	StopLoss float64 `json:"stopLoss,omitempty"`
```

- [ ] **Step 4: Run statestore tests — PASS**

Run: `go test ./internal/service/trading_strategy/livecore/statestore/ -v`

- [ ] **Step 5: Add adapter interfaces**

In `adapter.go` add import `"tinvest/internal/service/trading_strategy/scalping/strategy"` (already imported — check) and append:

```go
// EntryWindow — необязательная подсказка стратегии: может ли она войти на пассе в момент now.
// Ложь — пасс по свободному тикеру не собирает данные и не спрашивает стратегию: сборка
// MarketData — запросы свечей, а стратегия с узким окном входа (gap_fade — только утро) иначе
// тянула бы их на каждом пассе. Сопровождение открытых позиций окно не ограничивает.
type EntryWindow interface {
	EntryPossible(now time.Time) bool
}

// EntryFilter — необязательное вето стратегии на вход после BUY-сигнала ядра (например,
// день дивидендной отсечки). Непустая причина — вход пропускается с уведомлением; ошибка —
// вход пропускается с алертом (fail-closed). В обоих случаях бар не расходуется: пасс
// спрашивает следующую по приоритету стратегию.
type EntryFilter interface {
	EntryBlocked(ctx context.Context, ticker, instrumentID string, md strategy.MarketData) (reason string, err error)
}
```

- [ ] **Step 6: Write failing pass tests**

Append to `shared_test.go`:

```go
// windowFake — гость с окном входа.
type windowFake struct {
	*fakeStrategy
	open bool
}

func (w *windowFake) EntryPossible(time.Time) bool { return w.open }

// filterFake — гость с вето на вход.
type filterFake struct {
	*fakeStrategy
	reason string
	err    error
	calls  int
}

func (f *filterFake) EntryBlocked(context.Context, string, string, strategy.MarketData) (string, error) {
	f.calls++
	return f.reason, f.err
}

func guestFake(name string, sig model.SignalKind, universe ...string) *fakeStrategy {
	f := zoneFake(sig, universe...)
	f.name = name
	return f
}

// Окно входа закрыто — свечи по свободному тикеру не запрашиваются, гостя не спрашивают.
// market-мок без ожиданий падает на любом GetCandles.
func TestClosedEntryWindowSkipsAssembly(t *testing.T) {
	gap := &windowFake{fakeStrategy: guestFake("gap_fade", model.SignalBuy, "GAZP"), open: false}
	e := newSharedEnv(t, sharedNow, nil, nil, gap)
	e.instruments.EXPECT().Shares(mock.Anything).Return(sharesOf("GAZP"), nil)
	e.ops.EXPECT().GetPortfolio(mock.Anything, mock.Anything).Return(nil, nil)
	e.tg.EXPECT().SendMessage(mock.Anything).Return(nil).Maybe()

	if err := e.svc.Run(context.Background(), dto.Run{}); err != nil {
		t.Fatalf("Run: %v", err)
	}
	if gap.decideCalls != 0 {
		t.Fatalf("гостя спросили %d раз при закрытом окне", gap.decideCalls)
	}
	if len(e.state(t)) != 0 {
		t.Fatal("вход при закрытом окне")
	}
}

// Окно открыто — обычный вход.
func TestOpenEntryWindowEnters(t *testing.T) {
	gap := &windowFake{fakeStrategy: guestFake("gap_fade", model.SignalBuy, "GAZP"), open: true}
	e := newSharedEnv(t, sharedNow, nil, nil, gap)
	e.instruments.EXPECT().Shares(mock.Anything).Return(sharesOf("GAZP"), nil)
	e.expectCandles(flatAt30m(sharedLastBar, 400, 100), dailies(sharedNow, 60))
	e.ops.EXPECT().GetPortfolio(mock.Anything, mock.Anything).Return(nil, nil)
	e.ops.EXPECT().GetPortfolioTotal(mock.Anything, mock.Anything).Return(1_000_000.0, nil)
	e.ops.EXPECT().GetAvailableCash(mock.Anything, mock.Anything).Return(1_000_000.0, nil)
	e.tg.EXPECT().SendMessage(mock.Anything).Return(nil).Maybe()

	if err := e.svc.Run(context.Background(), dto.Run{}); err != nil {
		t.Fatalf("Run: %v", err)
	}
	if got := e.state(t)["GAZP"].Strategy; got != "gap_fade" {
		t.Fatalf("владелец = %q, want gap_fade", got)
	}
}

// Окно входа не мешает сопровождению: позиция гостя ведётся и вне окна.
func TestClosedEntryWindowStillManagesPosition(t *testing.T) {
	gap := &windowFake{fakeStrategy: guestFake("gap_fade", model.SignalSell, "GAZP"), open: false}
	e := newSharedEnv(t, sharedNow, nil, nil, gap)
	e.seed(t, statestore.Entry{Ticker: "GAZP", Strategy: "gap_fade", EntryPrice: 100,
		EntryATR: 10, MaxFav: 100, Quantity: 100, EntryTime: sharedNow.Add(-3 * time.Hour)})
	e.instruments.EXPECT().Shares(mock.Anything).Return(sharesOf("GAZP"), nil)
	e.expectCandles(flatAt30m(sharedLastBar, 400, 100), dailies(sharedNow, 60))
	e.ops.EXPECT().GetPortfolio(mock.Anything, mock.Anything).Return([]*grpcmodel.Position{held("GAZP", 100)}, nil)
	e.tg.EXPECT().SendMessage(mock.Anything).Return(nil).Maybe()

	if err := e.svc.Run(context.Background(), dto.Run{}); err != nil {
		t.Fatalf("Run: %v", err)
	}
	if gap.decideCalls != 1 {
		t.Fatalf("Decide вызван %d раз, want 1", gap.decideCalls)
	}
	if _, ok := e.state(t)["GAZP"]; ok {
		t.Fatal("SELL владельца вне окна входа не закрыл позицию")
	}
}

// Вето с причиной: ордера и стейта нет, причина — уведомлением; следующий гость на этом же
// баре входит.
func TestEntryFilterReasonSkipsAndAsksNextStrategy(t *testing.T) {
	first := &filterFake{fakeStrategy: guestFake("rsi_zone", model.SignalBuy, "GAZP"), reason: "день дивидендной отсечки"}
	second := guestFake("gap_fade", model.SignalBuy, "GAZP")
	e := newSharedEnv(t, sharedNow, nil, nil, first, second)
	e.instruments.EXPECT().Shares(mock.Anything).Return(sharesOf("GAZP"), nil)
	e.expectCandles(flatAt30m(sharedLastBar, 400, 100), dailies(sharedNow, 60))
	e.ops.EXPECT().GetPortfolio(mock.Anything, mock.Anything).Return(nil, nil)
	e.ops.EXPECT().GetPortfolioTotal(mock.Anything, mock.Anything).Return(1_000_000.0, nil)
	e.ops.EXPECT().GetAvailableCash(mock.Anything, mock.Anything).Return(1_000_000.0, nil)
	e.tg.EXPECT().SendMessage(mock.Anything).Return(nil).Maybe()

	if err := e.svc.Run(context.Background(), dto.Run{}); err != nil {
		t.Fatalf("Run: %v", err)
	}
	if first.calls != 1 {
		t.Fatalf("EntryBlocked вызван %d раз, want 1", first.calls)
	}
	if !first.said("день дивидендной отсечки") {
		t.Fatalf("причина вето не ушла уведомлением: %v", first.msgs)
	}
	if got := e.state(t)["GAZP"].Strategy; got != "gap_fade" {
		t.Fatalf("владелец = %q, want gap_fade (следующий по приоритету)", got)
	}
}

// Ошибка вето: вход не делается (fail-closed), алерт; бар не расходуется.
func TestEntryFilterErrorFailsClosed(t *testing.T) {
	gap := &filterFake{fakeStrategy: guestFake("gap_fade", model.SignalBuy, "GAZP"), err: errors.New("api down")}
	e := newSharedEnv(t, sharedNow, nil, nil, gap)
	e.instruments.EXPECT().Shares(mock.Anything).Return(sharesOf("GAZP"), nil)
	e.expectCandles(flatAt30m(sharedLastBar, 400, 100), dailies(sharedNow, 60))
	e.ops.EXPECT().GetPortfolio(mock.Anything, mock.Anything).Return(nil, nil)
	e.tg.EXPECT().SendMessage(mock.Anything).Return(nil).Maybe()

	if err := e.svc.Run(context.Background(), dto.Run{}); err != nil {
		t.Fatalf("Run: %v", err)
	}
	if len(e.state(t)) != 0 {
		t.Fatal("вход при ошибке вето")
	}
	if !gap.said("api down") {
		t.Fatalf("ошибка вето не ушла алертом: %v", gap.msgs)
	}
}

// Стоп, замороженный сигналом, пишется в стейт при входе и возвращается ядру в Position.
func TestSignalStopLossIsFrozenAndPassedBack(t *testing.T) {
	gap := guestFake("gap_fade", model.SignalBuy, "GAZP")
	gap.sig.StopLoss = 77
	e := newSharedEnv(t, sharedNow, nil, nil, gap)
	e.instruments.EXPECT().Shares(mock.Anything).Return(sharesOf("GAZP"), nil)
	e.expectCandles(flatAt30m(sharedLastBar, 400, 100), dailies(sharedNow, 60))
	e.ops.EXPECT().GetPortfolio(mock.Anything, mock.Anything).Return(nil, nil)
	e.ops.EXPECT().GetPortfolioTotal(mock.Anything, mock.Anything).Return(1_000_000.0, nil)
	e.ops.EXPECT().GetAvailableCash(mock.Anything, mock.Anything).Return(1_000_000.0, nil)
	e.tg.EXPECT().SendMessage(mock.Anything).Return(nil).Maybe()
	if err := e.svc.Run(context.Background(), dto.Run{}); err != nil {
		t.Fatalf("Run: %v", err)
	}
	if got := e.state(t)["GAZP"].StopLoss; got != 77 {
		t.Fatalf("Entry.StopLoss = %v, want 77", got)
	}

	held2 := guestFake("gap_fade", model.SignalNone, "GAZP")
	e2 := newSharedEnv(t, sharedNow, nil, nil, held2)
	e2.seed(t, statestore.Entry{Ticker: "GAZP", Strategy: "gap_fade", EntryPrice: 100,
		EntryATR: 10, MaxFav: 100, Quantity: 100, StopLoss: 77, EntryTime: sharedNow.Add(-time.Hour)})
	e2.instruments.EXPECT().Shares(mock.Anything).Return(sharesOf("GAZP"), nil)
	e2.expectCandles(flatAt30m(sharedLastBar, 400, 100), dailies(sharedNow, 60))
	e2.ops.EXPECT().GetPortfolio(mock.Anything, mock.Anything).Return([]*grpcmodel.Position{held("GAZP", 100)}, nil)
	e2.tg.EXPECT().SendMessage(mock.Anything).Return(nil).Maybe()
	if err := e2.svc.Run(context.Background(), dto.Run{}); err != nil {
		t.Fatalf("Run: %v", err)
	}
	if held2.lastPosition == nil || held2.lastPosition.StopLoss != 77 {
		t.Fatalf("Position.StopLoss = %+v, want 77", held2.lastPosition)
	}
}
```

Add `"errors"` to the `shared_test.go` imports.

- [ ] **Step 7: Run, expect failures**

Run: `go test ./internal/service/trading_strategy/rsi_pullback/live/ -run 'EntryWindow|EntryFilter|SignalStopLoss' -v`
Expected: FAIL — window/filter ignored, StopLoss not stored.

- [ ] **Step 8: Implement in `pass.go`**

(a) In `pass()`, the free-ticker loop, before `s.assemble(...)`:

```go
		for _, c := range ready {
			if w, ok := c.sl.strat.(adapter.EntryWindow); ok && !w.EntryPossible(now) {
				continue // окно входа закрыто: ни свечей, ни вопроса стратегии
			}
			md, ok := s.assemble(ctx, c.sl, ticker, sh, c.dec, now, false)
```

(b) In `buy()`, right after `if sig.Kind != model.SignalBuy { return false, nil }` and before `GetPortfolioTotal`:

```go
	// Вето стратегии (дивидендная отсечка у gap_fade) — до сайзинга и ордера. Бар не
	// расходуется (signaled=false): следующая по приоритету стратегия всё равно спрашивается.
	if f, ok := sl.strat.(adapter.EntryFilter); ok {
		reason, ferr := f.EntryBlocked(ctx, ticker, sh.ID, md)
		if ferr != nil {
			sl.alert(ticker, "проверка входа не удалась — вход пропущен: "+ferr.Error())
			logger.ErrorContext(ctx, fmt.Sprintf("%s: %s entry filter: %v", sl.strat.Name(), ticker, ferr))
			return false, nil
		}
		if reason != "" {
			sl.strat.Notify(notifier.Skip(ticker, reason))
			return false, nil
		}
	}
```

(c) In `buy()`, the state entry literal: add `StopLoss: sig.StopLoss,` after `TakeProfit: sig.TakeProfit,`.

(d) In `manage()`, the `md.Position = &strategy.Position{...}` literal: add `StopLoss: entry.StopLoss,` after `TakeProfit: entry.TakeProfit,`.

- [ ] **Step 9: Run the whole runner package — PASS**

Run: `go test ./internal/service/trading_strategy/rsi_pullback/live/... ./internal/service/trading_strategy/livecore/... -race`
Expected: PASS (all existing tests too).

- [ ] **Step 10: Commit**

```bash
git add internal/service/trading_strategy/livecore internal/service/trading_strategy/rsi_pullback/live
git commit -m "feat(livecore): замороженный StopLoss, окно входа и вето входа в раннере счёта

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

---

### Task 2: Config, shared params and the gap_fade adapter

**Files:**
- Create: `internal/config/gap_fade.go`, Test: `internal/config/gap_fade_test.go`
- Modify: `internal/config/config.go`, `internal/app/init_config.go`, `internal/config/telegram_client.go`
- Create: `internal/service/trading_strategy/gap_fade/strategy/shared/shared.go`
- Create: `internal/service/trading_strategy/gap_fade/live/registry.go`, `internal/service/trading_strategy/gap_fade/live/strategy.go`
- Test: `internal/service/trading_strategy/gap_fade/live/registry_test.go`, `internal/service/trading_strategy/gap_fade/live/strategy_test.go`

**Interfaces:**
- Consumes: `adapter.Strategy`, `adapter.EntryWindow` (Task 1), `statestore.Entry.StopLoss` (Task 1), `rebuild.LastBuyTime`, `rebuild.WeekdayDailyATRBefore`, `rebuild.ErrNoBuyFill`.
- Produces:
  - `config.GapFadeConfig{Tickers []string; BuyPct float64; TradeEnabled, NotifyEnabled bool}`, `config.NewGapFadeConfig() *GapFadeConfig`, `(*GapFadeConfig).Enabled() bool`; `Config.GapFade *GapFadeConfig`; `TelegramClient.TopicGapFade int`.
  - `shared.Params() core.Params` (package `tinvest/internal/service/trading_strategy/gap_fade/strategy/shared`).
  - package `gaplive` path `tinvest/internal/service/trading_strategy/gap_fade/live` (package name `live`): `const Name = "gap_fade"`, `ParamsFor(ticker string) (core.Params, bool)`, `StrategyFor(ticker string) (*core.Strategy, bool)`, `type DividendsClient interface { GetDividends(ctx context.Context, instrumentID string, from, to time.Time) ([]*pkgmodel.Dividend, error) }`, `New(cfg *config.GapFadeConfig, tg telegram.Client, divs DividendsClient) *Strategy`, and `*Strategy` methods of `adapter.Strategy` + `EntryPossible(now time.Time) bool`. `EntryBlocked` comes in Task 3.

- [ ] **Step 1: Config tests**

`internal/config/gap_fade_test.go`:

```go
package config

import (
	"context"
	"testing"

	"github.com/heetch/confita"
	"github.com/heetch/confita/backend/env"
)

// Без переменных gap_fade конфиг грузится, стратегия и торговля выключены.
func TestGapFadeConfigLoadsWithoutItsEnvVars(t *testing.T) {
	t.Setenv("GAP_FADE_TICKERS", "")
	t.Setenv("GAP_FADE_BUY_PCT", "")
	t.Setenv("GAP_FADE_TRADE_ENABLED", "")
	t.Setenv("GAP_FADE_NOTIFY_ENABLED", "")

	cfg := NewGapFadeConfig()
	if err := confita.NewLoader(env.NewBackend()).Load(context.Background(), cfg); err != nil {
		t.Fatalf("Load без GAP_FADE_* = %v, want nil", err)
	}
	if cfg.Enabled() || cfg.TradeEnabled {
		t.Fatalf("без переменных cfg = %+v, want стратегия и торговля выключены", cfg)
	}
}

func TestNewGapFadeConfig_Defaults(t *testing.T) {
	c := NewGapFadeConfig()
	if c.Enabled() || c.TradeEnabled || c.NotifyEnabled || c.BuyPct != 5 {
		t.Fatalf("дефолты = %+v, want выключено, BuyPct 5", c)
	}
}

func TestGapFadeEnabledIgnoresBlankTickers(t *testing.T) {
	c := NewGapFadeConfig()
	c.Tickers = []string{"", " "}
	if c.Enabled() {
		t.Fatal("вселенная из пустых строк включила стратегию")
	}
	c.Tickers = []string{"SBER"}
	if !c.Enabled() {
		t.Fatal("вселенная SBER не включила стратегию")
	}
}
```

Run: `go test ./internal/config/ -run GapFade -v` — expect compile FAIL.

- [ ] **Step 2: Config implementation**

`internal/config/gap_fade.go`:

```go
package config

import "strings"

// GapFadeConfig подключает gap_fade к живому раннеру счёта rsi_pullback гостем: счёт, токен и
// расписание общие (RSI_PULLBACK_*), здесь — только своё (docs/gap_fade/live.md). Ни одна
// переменная не required — ошибка загрузки конфига оборвала бы InitApp целиком.
type GapFadeConfig struct {
	Tickers       []string `config:"GAP_FADE_TICKERS,backend=env"`
	BuyPct        float64  `config:"GAP_FADE_BUY_PCT,backend=env"`
	TradeEnabled  bool     `config:"GAP_FADE_TRADE_ENABLED,backend=env"`
	NotifyEnabled bool     `config:"GAP_FADE_NOTIFY_ENABLED,backend=env"`
}

// NewGapFadeConfig: вселенная пуста — стратегия не подключается, пока её не включат в env;
// торговля выключена, чтобы отсутствующий флаг не выставил ордер.
func NewGapFadeConfig() *GapFadeConfig {
	return &GapFadeConfig{BuyPct: 5}
}

// Enabled — задана ли вселенная. GAP_FADE_TICKERS= в env приходит как [""].
func (c *GapFadeConfig) Enabled() bool {
	for _, t := range c.Tickers {
		if strings.TrimSpace(t) != "" {
			return true
		}
	}
	return false
}
```

`config.go`: add field `GapFade *GapFadeConfig` after `RSIZone`. `init_config.go`: add `GapFade: config.NewGapFadeConfig(),` after `RSIZone`. `telegram_client.go`: add `TopicGapFade int \`config:"TELEGRAM_TOPIC_GAP_FADE"\`` after `TopicRSIZone`.

Run: `go test ./internal/config/ -v` — PASS.

- [ ] **Step 3: Shared params literal**

`internal/service/trading_strategy/gap_fade/strategy/shared/shared.go`:

```go
// Package shared holds the single gap_fade parameter set that live trades on every ticker of
// GAP_FADE_TICKERS. There is no per-ticker calibration: a ticker sees a handful of gaps a year,
// so a per-ticker profit factor is noise; the set is validated once, pooled, by cmd/gapwf.
//
// Choice (2026-10-02, cmd/gapwf on the 46-ticker cmd/gapscreen universe): the walk-forward folds
// pick GapATR 0.5 or 0.7 with TargetFill 0.75 and StopDailyATR 1.0. GapATR 0.7 is taken because
// it makes the same sum of returns with less than half the trades, which leaves a wider margin
// for real morning costs — the main live risk of this strategy.
package shared

import "tinvest/internal/service/trading_strategy/gap_fade/strategy/core"

// Params returns the live parameter set.
func Params() core.Params {
	return core.Params{
		GapATR:         0.7,
		MaxGapATR:      3.0,
		EntryBar:       0,
		TargetFill:     0.75,
		StopDailyATR:   1.0,
		DailyATRPeriod: 14,
		EODHour:        18,
		EODMinute:      30,
	}
}
```

- [ ] **Step 4: Registry tests**

`internal/service/trading_strategy/gap_fade/live/registry_test.go`:

```go
package live

import (
	"testing"

	"tinvest/internal/service/trading_strategy/gap_fade/strategy/core"
	"tinvest/internal/service/trading_strategy/gap_fade/strategy/shared"
)

// Сторожевой тест литерала: живые параметры правятся только осознанно, вместе с этим тестом.
func TestSharedParamsLiteral(t *testing.T) {
	want := core.Params{GapATR: 0.7, MaxGapATR: 3.0, EntryBar: 0, TargetFill: 0.75,
		StopDailyATR: 1.0, DailyATRPeriod: 14, EODHour: 18, EODMinute: 30}
	if got := shared.Params(); got != want {
		t.Fatalf("shared.Params() = %+v, want %+v", got, want)
	}
}

// Любой непустой тикер получает общий литерал: позиция тикера, убранного из вселенной,
// доводится до выхода. Входы ограничивает вселенная, а не реестр.
func TestRegistryServesSharedParamsForAnyTicker(t *testing.T) {
	for _, tk := range []string{"SBER", "NOT_IN_UNIVERSE"} {
		p, ok := ParamsFor(tk)
		if !ok || p != shared.Params() {
			t.Fatalf("ParamsFor(%q) = %+v, %v", tk, p, ok)
		}
		st, ok := StrategyFor(tk)
		if !ok || st.Ticker() != tk || st.Lookback() != 64 {
			t.Fatalf("StrategyFor(%q) = %v, %v", tk, st, ok)
		}
	}
}

func TestRegistryRejectsBlankTicker(t *testing.T) {
	if _, ok := ParamsFor(" "); ok {
		t.Fatal("пустой тикер зарегистрирован")
	}
	if st, ok := StrategyFor(""); ok || st != nil {
		t.Fatal("пустой тикер дал стратегию")
	}
}
```

- [ ] **Step 5: Registry implementation**

`internal/service/trading_strategy/gap_fade/live/registry.go`:

```go
package live

import (
	"strings"

	"tinvest/internal/service/trading_strategy/gap_fade/strategy/core"
	"tinvest/internal/service/trading_strategy/gap_fade/strategy/shared"
)

// ParamsFor — общий литерал для любого непустого тикера. Калибровки по тикерам у gap_fade нет;
// вход ограничивает вселенная GAP_FADE_TICKERS, а позиция тикера, убранного из неё, должна
// дойти до выхода, поэтому реестр её не отвергает.
func ParamsFor(ticker string) (core.Params, bool) {
	if strings.TrimSpace(ticker) == "" {
		return core.Params{}, false
	}
	return shared.Params(), true
}

// StrategyFor — ядро тикера с общим литералом.
func StrategyFor(ticker string) (*core.Strategy, bool) {
	p, ok := ParamsFor(ticker)
	if !ok {
		return nil, false
	}
	return core.NewWithParams(ticker, p), true
}
```

Run: `go test ./internal/service/trading_strategy/gap_fade/live/ -run Registry -v` and `-run SharedParams` — PASS.

- [ ] **Step 6: Adapter tests**

`internal/service/trading_strategy/gap_fade/live/strategy_test.go`:

```go
package live

import (
	"context"
	"errors"
	"os"
	"strings"
	"testing"
	"time"

	"google.golang.org/protobuf/types/known/timestamppb"

	"tinvest/internal/config"
	imodel "tinvest/internal/model"
	"tinvest/internal/service/trading_strategy/livecore/adapter"
	"tinvest/internal/service/trading_strategy/livecore/statestore"
	grpcmodel "tinvest/pkg/client/grpc/model"
	tgmocks "tinvest/pkg/client/telegram/mocks"
	"tinvest/pkg/logger"
)

func TestMain(m *testing.M) {
	logger.Init()
	os.Exit(m.Run())
}

var msk = func() *time.Location {
	loc, err := time.LoadLocation("Europe/Moscow")
	if err != nil {
		return time.UTC
	}
	return loc
}()

type fakeTrades struct{ trades []grpcmodel.Trade }

func (f *fakeTrades) GetInstrumentTrades(_ context.Context, _, _ string, _, _ time.Time) ([]grpcmodel.Trade, error) {
	return f.trades, nil
}

type fakeDaily struct{ bars []*imodel.CandleItemTechAnalyse }

func (f *fakeDaily) GetCandles(_ context.Context, _ *string, _ int32,
	_, _ *timestamppb.Timestamp, _ *int32, _ bool) ([]*imodel.CandleItemTechAnalyse, error) {
	return f.bars, nil
}

func q(f float64) imodel.Quotation { return imodel.Quotation{Units: int64(f)} }

// weekdayBars — будние дневки за 60 дней до полуночи MSK дня входа, истинный диапазон 8.
func weekdayBars(entry time.Time) []*imodel.CandleItemTechAnalyse {
	e := entry.In(msk)
	midnight := time.Date(e.Year(), e.Month(), e.Day(), 0, 0, 0, 0, msk)
	var bars []*imodel.CandleItemTechAnalyse
	for off := 60; off >= 1; off-- {
		d := midnight.AddDate(0, 0, -off)
		if wd := d.Weekday(); wd == time.Saturday || wd == time.Sunday {
			continue
		}
		bars = append(bars, &imodel.CandleItemTechAnalyse{
			Time: d, Open: q(100), High: q(104), Low: q(96), Close: q(100),
			Volume: 1000, IsComplete: true,
		})
	}
	return bars
}

func newGap(t *testing.T) *Strategy {
	t.Helper()
	return New(&config.GapFadeConfig{Tickers: []string{"SBER"}, BuyPct: 5}, nil, nil)
}

func TestIdentity(t *testing.T) {
	s := New(&config.GapFadeConfig{Tickers: []string{"SBER", "GAZP"}, BuyPct: 7, TradeEnabled: true}, nil, nil)
	if s.Name() != "gap_fade" || s.Label() != "Gap Fade" || s.BuyPct() != 7 || !s.TradeEnabled() ||
		strings.Join(s.Tickers(), ",") != "SBER,GAZP" {
		t.Fatalf("identity = %s/%s/%v/%v/%v", s.Name(), s.Label(), s.BuyPct(), s.TradeEnabled(), s.Tickers())
	}
	var _ adapter.Strategy = s
	var _ adapter.EntryWindow = s
}

// Стоп заморожен сигналом и хранится в стейте; без него стопа нет.
func TestDesiredStopIsFrozenStopLoss(t *testing.T) {
	s := newGap(t)
	if lvl, why := s.DesiredStop("SBER", statestore.Entry{EntryPrice: 90, StopLoss: 78}); lvl != 78 || why != "SL" {
		t.Fatalf("DesiredStop = %v/%q, want 78/SL", lvl, why)
	}
	if lvl, why := s.DesiredStop("SBER", statestore.Entry{EntryPrice: 90}); lvl != 0 || why != "" {
		t.Fatalf("DesiredStop без StopLoss = %v/%q, want 0/\"\"", lvl, why)
	}
}

// Реконструкция: стоп от цены входа (close сигнала неизвестен), цели нет.
func TestReconstructStopFromEntryNoTarget(t *testing.T) {
	s := newGap(t)
	entry := time.Date(2026, 3, 10, 7, 1, 0, 0, msk)
	e, err := s.Reconstruct(context.Background(), adapter.ReconstructInput{
		Trades:  &fakeTrades{trades: []grpcmodel.Trade{{IsBuy: true, Date: entry}}},
		Candles: &fakeDaily{bars: weekdayBars(entry)},
		Ticker:  "SBER", PurchasePrice: 90, Now: entry.Add(time.Hour),
	})
	if err != nil {
		t.Fatalf("Reconstruct: %v", err)
	}
	if e.EntryATR <= 0 || e.TakeProfit != 0 || e.MaxFav != 90 || !e.EntryTime.Equal(entry) {
		t.Fatalf("entry = %+v", e)
	}
	if want := 90 - 1.0*e.EntryATR; e.StopLoss != want {
		t.Fatalf("StopLoss = %v, want %v", e.StopLoss, want)
	}
}

func TestReconstructWithoutATRFails(t *testing.T) {
	s := newGap(t)
	entry := time.Date(2026, 3, 10, 7, 1, 0, 0, msk)
	_, err := s.Reconstruct(context.Background(), adapter.ReconstructInput{
		Trades:  &fakeTrades{trades: []grpcmodel.Trade{{IsBuy: true, Date: entry}}},
		Candles: &fakeDaily{}, Ticker: "SBER", PurchasePrice: 90, Now: entry,
	})
	if err == nil {
		t.Fatal("нулевой ATR дал реконструкцию")
	}
}

func TestReconstructWithoutBuyFails(t *testing.T) {
	s := newGap(t)
	now := time.Date(2026, 3, 10, 7, 1, 0, 0, msk)
	_, err := s.Reconstruct(context.Background(), adapter.ReconstructInput{
		Trades: &fakeTrades{}, Candles: &fakeDaily{bars: weekdayBars(now)},
		Ticker: "SBER", PurchasePrice: 90, Now: now,
	})
	if err == nil || !strings.Contains(err.Error(), "no BUY fill") {
		t.Fatalf("err = %v, want no BUY fill", err)
	}
}

// Окно входа: будний день до 10:00 MSK.
func TestEntryPossible(t *testing.T) {
	s := newGap(t)
	cases := []struct {
		at   time.Time
		want bool
	}{
		{time.Date(2026, 3, 10, 7, 1, 0, 0, msk), true},   // вторник 07:01
		{time.Date(2026, 3, 10, 9, 59, 0, 0, msk), true},  // до 10:00
		{time.Date(2026, 3, 10, 10, 0, 0, 0, msk), false}, // ровно 10:00 — уже нет
		{time.Date(2026, 3, 10, 19, 1, 0, 0, msk), false},
		{time.Date(2026, 3, 14, 7, 1, 0, 0, msk), false},          // суббота
		{time.Date(2026, 3, 10, 4, 1, 0, 0, time.UTC), true},      // 07:01 MSK в UTC
	}
	for _, c := range cases {
		if got := s.EntryPossible(c.at); got != c.want {
			t.Errorf("EntryPossible(%v) = %v, want %v", c.at, got, c.want)
		}
	}
}

func TestNotifyPrefixAndSwitch(t *testing.T) {
	tg := tgmocks.NewMockClient(t)
	tg.EXPECT().SendMessage("[Gap Fade] привет").Return(nil).Once()
	New(&config.GapFadeConfig{NotifyEnabled: true}, tg, nil).Notify("привет")

	off := tgmocks.NewMockClient(t) // без ожиданий: любой вызов — падение
	New(&config.GapFadeConfig{NotifyEnabled: false}, off, nil).Notify("тишина")
	New(&config.GapFadeConfig{NotifyEnabled: true}, nil, nil).Notify("nil-клиент не паникует")
}

func TestNotifyDeliveryFailureDoesNotPanic(t *testing.T) {
	tg := tgmocks.NewMockClient(t)
	tg.EXPECT().SendMessage("[Gap Fade] x").Return(errors.New("down")).Once()
	New(&config.GapFadeConfig{NotifyEnabled: true}, tg, nil).Notify("x")
}
```

Before relying on `grpcmodel.Trade{IsBuy, Date}` field names, confirm them in `pkg/client/grpc/model` (they are used by `rebuild.LastBuyTime`).

Run: `go test ./internal/service/trading_strategy/gap_fade/live/ -v` — expect compile FAIL.

- [ ] **Step 7: Adapter implementation**

`internal/service/trading_strategy/gap_fade/live/strategy.go`:

```go
// Package live connects gap_fade to the rsi_pullback account runner as a guest adapter
// (docs/gap_fade/live.md): same process, pass, state file and account; its own Telegram topic.
package live

import (
	"context"
	"errors"
	"fmt"
	"time"

	"tinvest/internal/config"
	"tinvest/internal/service/trading_strategy/livecore/adapter"
	"tinvest/internal/service/trading_strategy/livecore/rebuild"
	"tinvest/internal/service/trading_strategy/livecore/statestore"
	pkgmodel "tinvest/pkg/client/grpc/model"
	"tinvest/pkg/client/telegram"
	"tinvest/pkg/logger"
)

// Name — имя стратегии в стейте счёта.
const Name = "gap_fade"

const alertLabel = "Gap Fade"

// entryCutoffMinute — пасс с этого времени MSK (10:00) gap_fade не спрашивает: сигнал — бар 06:30,
// запас покрывает сдвиг расписания сессии и опоздавший пасс.
const entryCutoffMinute = 10 * 60

// DividendsClient — источник дивидендов для фильтра дня отсечки.
type DividendsClient interface {
	GetDividends(ctx context.Context, instrumentID string, from, to time.Time) ([]*pkgmodel.Dividend, error)
}

// Strategy — gap_fade как гость раннера счёта.
type Strategy struct {
	cfg  *config.GapFadeConfig
	tg   telegram.Client
	divs DividendsClient
}

var (
	_ adapter.Strategy    = (*Strategy)(nil)
	_ adapter.EntryWindow = (*Strategy)(nil)
)

func New(cfg *config.GapFadeConfig, tg telegram.Client, divs DividendsClient) *Strategy {
	return &Strategy{cfg: cfg, tg: tg, divs: divs}
}

var mskLoc = func() *time.Location {
	loc, err := time.LoadLocation("Europe/Moscow")
	if err != nil {
		return time.UTC
	}
	return loc
}()

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

// EntryPossible — будний день MSK до 10:00: gap_fade входит только по бару 06:30.
func (s *Strategy) EntryPossible(now time.Time) bool {
	t := now.In(mskLoc)
	if wd := t.Weekday(); wd == time.Saturday || wd == time.Sunday {
		return false
	}
	return t.Hour()*60+t.Minute() < entryCutoffMinute
}

// DesiredStop — стоп, замороженный на входе от close сигнального бара (Entry.StopLoss). Уровень
// не меняется за всю жизнь позиции.
func (s *Strategy) DesiredStop(_ string, e statestore.Entry) (float64, string) {
	if e.StopLoss <= 0 {
		return 0, ""
	}
	return e.StopLoss, "SL"
}

// Reconstruct поднимает вход по API. Close сигнального бара неизвестен, поэтому стоп меряется
// от цены входа; цели нет — позицию закроет EOD или первый бар новой даты.
func (s *Strategy) Reconstruct(ctx context.Context, in adapter.ReconstructInput) (statestore.Entry, error) {
	p, ok := ParamsFor(in.Ticker)
	if !ok {
		return statestore.Entry{}, fmt.Errorf("gap_fade reconstruct: %q не зарегистрирован", in.Ticker)
	}
	entryTime, err := rebuild.LastBuyTime(ctx, in.Trades, in.AccountID, in.InstrumentID, in.Now)
	switch {
	case errors.Is(err, rebuild.ErrNoBuyFill):
		return statestore.Entry{}, fmt.Errorf("gap_fade reconstruct: no BUY fill found for %s", in.Ticker)
	case err != nil:
		return statestore.Entry{}, fmt.Errorf("gap_fade reconstruct: %w", err)
	}
	atr, err := rebuild.WeekdayDailyATRBefore(ctx, in.Candles, in.InstrumentID, entryTime, p.DailyATRPeriod)
	if err != nil {
		return statestore.Entry{}, fmt.Errorf("gap_fade reconstruct: %w", err)
	}
	if atr <= 0 {
		return statestore.Entry{}, fmt.Errorf("gap_fade reconstruct: %s: cannot rebuild daily ATR as of %s", in.Ticker, entryTime.Format(time.RFC3339))
	}
	return statestore.Entry{
		Ticker:     in.Ticker,
		EntryTime:  entryTime,
		EntryPrice: in.PurchasePrice,
		EntryATR:   atr,
		StopLoss:   in.PurchasePrice - p.StopDailyATR*atr,
		MaxFav:     in.PurchasePrice,
	}, nil
}

// Notify шлёт в тему gap_fade с меткой стратегии. Сбой доставки — ERROR-лог, как у соседей.
func (s *Strategy) Notify(msg string) {
	if !s.cfg.NotifyEnabled || s.tg == nil {
		return
	}
	if err := s.tg.SendMessage("[" + alertLabel + "] " + msg); err != nil {
		logger.ErrorContext(context.Background(), fmt.Sprintf("gap_fade: уведомление не доставлено: %v", err))
	}
}
```

Note: `rebuild.WeekdayDailyATRBefore` with `fakeDaily{}` (no bars) — confirm it returns `0, nil` (doc says so for too few bars); the adapter then errors on `atr <= 0`.

- [ ] **Step 8: Run — PASS**

Run: `go test ./internal/service/trading_strategy/gap_fade/... ./internal/config/... -race`
Run: `go build ./internal/... ./pkg/... ./cmd/...`

- [ ] **Step 9: Commit**

```bash
git add internal/config internal/app/init_config.go internal/service/trading_strategy/gap_fade
git commit -m "feat(gap_fade): конфиг, общий литерал параметров и адаптер гостя раннера

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

---

### Task 3: Dividend ex-day filter

**Files:**
- Modify: `internal/service/backtest/gap_wf.go` (add `GapDividendExDays`)
- Test: `internal/service/backtest/gap_wf_test.go` (append)
- Modify: `cmd/gapwf/main.go` (use the shared function, delete local `dividendExDays`)
- Modify: `cmd/gapwf/main_test.go` only if it tests `dividendExDays` directly — move those cases to the svc test.
- Modify: `internal/service/trading_strategy/gap_fade/live/strategy.go` (add `EntryBlocked`)
- Test: `internal/service/trading_strategy/gap_fade/live/strategy_test.go` (append)

**Interfaces:**
- Consumes: `adapter.EntryFilter` (Task 1), `Strategy`/`DividendsClient` (Task 2), existing `svc.GapDay`, `svc.DividendExDaysFromBars`.
- Produces: `svc.GapDividendExDays(divs []*pkgmodel.Dividend, barTimes []time.Time) map[string]bool`; `(*Strategy).EntryBlocked(ctx, ticker, instrumentID string, md strategy.MarketData) (string, error)`.

- [ ] **Step 1: Move the rule into svc — test first**

Append to `internal/service/backtest/gap_wf_test.go` (add imports `pkgmodel "tinvest/pkg/client/grpc/model"` and `time` if missing):

```go
func TestGapDividendExDays(t *testing.T) {
	loc, _ := time.LoadLocation("Europe/Moscow")
	bar := func(d, h int) time.Time { return time.Date(2026, 3, d, h, 0, 0, 0, loc) }
	bars := []time.Time{bar(6, 18), bar(9, 18), bar(10, 7), bar(10, 8)}
	divs := []*pkgmodel.Dividend{
		nil,
		{LastBuyDate: time.Date(2026, 3, 9, 0, 0, 0, 0, time.UTC), DividendType: "Regular Cash"}, // → 2026-03-10
		{LastBuyDate: time.Date(2026, 3, 5, 0, 0, 0, 0, time.UTC), DividendType: "Cancelled"},    // отменён
		{RecordDate: time.Date(2026, 3, 12, 0, 0, 0, 0, time.UTC)},                               // по дате реестра
	}
	got := GapDividendExDays(divs, bars)
	if !got["2026-03-10"] || !got["2026-03-12"] || got["2026-03-06"] || len(got) != 2 {
		t.Fatalf("GapDividendExDays = %v", got)
	}
}
```

Run: `go test ./internal/service/backtest/ -run GapDividendExDays -v` — compile FAIL.

- [ ] **Step 2: Implement by moving `cmd/gapwf.dividendExDays`**

Add to `gap_wf.go` (import `pkgmodel "tinvest/pkg/client/grpc/model"`):

```go
// GapDividendExDays maps every non-cancelled dividend to its ex-day: the first bar's date after
// last_buy_date (DividendExDaysFromBars). A dividend without last_buy_date falls back to its
// record date, which under T+1 settlement is itself the ex-day. Shared by cmd/gapwf and the
// live gap_fade entry filter, so backtest and live drop the same days.
func GapDividendExDays(divs []*pkgmodel.Dividend, barTimes []time.Time) map[string]bool {
	var lastBuy, recordEx []time.Time
	for _, d := range divs {
		switch {
		case d == nil || d.DividendType == "Cancelled":
		case !d.LastBuyDate.IsZero():
			lastBuy = append(lastBuy, d.LastBuyDate)
		case !d.RecordDate.IsZero():
			recordEx = append(recordEx, d.RecordDate)
		}
	}
	out := DividendExDaysFromBars(lastBuy, barTimes)
	for _, r := range recordEx {
		out[GapDay(r)] = true
	}
	return out
}
```

In `cmd/gapwf/main.go`: replace `exDays: dividendExDays(divs, bars)` with

```go
	barTimes := make([]time.Time, len(bars))
	for i, b := range bars {
		barTimes[i] = b.Time
	}
	return tickerData{ticker: u.Ticker, lot: u.Lot, bars: bars, daily: daily, exDays: svc.GapDividendExDays(divs, barTimes)}, nil
```

and delete `func dividendExDays`. Remove the now-unused `pkgmodel` import if nothing else uses it. If `cmd/gapwf/main_test.go` calls `dividendExDays`, switch those calls to `svc.GapDividendExDays` with a `[]time.Time` built from the test bars.

Run: `go test ./internal/service/backtest/ ./cmd/gapwf/ -race` — PASS.

- [ ] **Step 3: EntryBlocked tests**

Append to `gap_fade/live/strategy_test.go` (add imports `pkgmodel "tinvest/pkg/client/grpc/model"` and `"tinvest/internal/service/trading_strategy/scalping/strategy"`):

```go
type fakeDivs struct {
	divs      []*pkgmodel.Dividend
	err       error
	from, to  time.Time
	calls     int
}

func (f *fakeDivs) GetDividends(_ context.Context, _ string, from, to time.Time) ([]*pkgmodel.Dividend, error) {
	f.calls++
	f.from, f.to = from, to
	return f.divs, f.err
}

// gapMD — окно из баров 9 марта (до 23:30) и сигнального бара 06:30 10 марта.
func gapMD() strategy.MarketData {
	var times []time.Time
	for t := time.Date(2026, 3, 9, 7, 0, 0, 0, msk); !t.After(time.Date(2026, 3, 9, 23, 30, 0, 0, msk)); t = t.Add(30 * time.Minute) {
		times = append(times, t)
	}
	times = append(times, time.Date(2026, 3, 10, 6, 30, 0, 0, msk))
	return strategy.MarketData{Times: times}
}

func utcDay(d int) time.Time { return time.Date(2026, 3, d, 0, 0, 0, 0, time.UTC) }

func TestEntryBlockedOnExDay(t *testing.T) {
	f := &fakeDivs{divs: []*pkgmodel.Dividend{{LastBuyDate: utcDay(9), DividendType: "Regular Cash"}}}
	s := New(&config.GapFadeConfig{}, nil, f)
	reason, err := s.EntryBlocked(context.Background(), "SBER", "uid", gapMD())
	if err != nil || !strings.Contains(reason, "отсечк") {
		t.Fatalf("EntryBlocked = %q, %v; want причина про отсечку", reason, err)
	}
	last := gapMD().Times[len(gapMD().Times)-1]
	if !f.from.Equal(last.AddDate(0, 0, -30)) || !f.to.Equal(last.AddDate(0, 0, 7)) {
		t.Fatalf("окно запроса %v—%v, want [−30д, +7д] от %v", f.from, f.to, last)
	}
}

func TestEntryNotBlockedByCancelledDividend(t *testing.T) {
	f := &fakeDivs{divs: []*pkgmodel.Dividend{{LastBuyDate: utcDay(9), DividendType: "Cancelled"}}}
	reason, err := New(&config.GapFadeConfig{}, nil, f).EntryBlocked(context.Background(), "SBER", "uid", gapMD())
	if err != nil || reason != "" {
		t.Fatalf("EntryBlocked = %q, %v; want вход разрешён", reason, err)
	}
}

func TestEntryBlockedByRecordDateWithoutLastBuy(t *testing.T) {
	f := &fakeDivs{divs: []*pkgmodel.Dividend{{RecordDate: utcDay(10)}}}
	reason, err := New(&config.GapFadeConfig{}, nil, f).EntryBlocked(context.Background(), "SBER", "uid", gapMD())
	if err != nil || reason == "" {
		t.Fatalf("EntryBlocked = %q, %v; want блок по дате реестра", reason, err)
	}
}

// Старая отсечка до начала окна помечает первый день окна, а не сегодняшний: вход разрешён.
func TestOldDividendDoesNotBlockToday(t *testing.T) {
	f := &fakeDivs{divs: []*pkgmodel.Dividend{{LastBuyDate: utcDay(2), DividendType: "Regular Cash"}}}
	reason, err := New(&config.GapFadeConfig{}, nil, f).EntryBlocked(context.Background(), "SBER", "uid", gapMD())
	if err != nil || reason != "" {
		t.Fatalf("EntryBlocked = %q, %v; want вход разрешён", reason, err)
	}
}

func TestEntryBlockedAPIErrorFailsClosed(t *testing.T) {
	f := &fakeDivs{err: errors.New("unavailable")}
	if _, err := New(&config.GapFadeConfig{}, nil, f).EntryBlocked(context.Background(), "SBER", "uid", gapMD()); err == nil {
		t.Fatal("ошибка API не вернулась")
	}
}

func TestEntryBlockedWithoutClientFailsClosed(t *testing.T) {
	if _, err := New(&config.GapFadeConfig{}, nil, nil).EntryBlocked(context.Background(), "SBER", "uid", gapMD()); err == nil {
		t.Fatal("nil-клиент не дал ошибку")
	}
}
```

Also add to `TestIdentity`: `var _ adapter.EntryFilter = s`.

Run: `go test ./internal/service/trading_strategy/gap_fade/live/ -run 'EntryBlocked|Dividend' -v` — compile FAIL.

- [ ] **Step 4: Implement EntryBlocked**

Append to `strategy.go` (imports: `svc "tinvest/internal/service/backtest"`, `"tinvest/internal/service/trading_strategy/scalping/strategy"`); extend the `var (...)` assertions with `_ adapter.EntryFilter = (*Strategy)(nil)`:

```go
// EntryBlocked не пускает вход в день дивидендной отсечки: гэп там — дивиденд, а не шум, и
// бэктест такие сделки выбрасывает (cmd/gapwf). Правило дня отсечки общее с бэктестом —
// svc.GapDividendExDays по барам окна. Без клиента или при сбое API — ошибка: раннер
// пропускает вход (fail-closed).
func (s *Strategy) EntryBlocked(ctx context.Context, ticker, instrumentID string, md strategy.MarketData) (string, error) {
	n := len(md.Times)
	if n == 0 {
		return "", nil // ядро без баров BUY не даёт
	}
	if s.divs == nil {
		return "", fmt.Errorf("gap_fade: нет клиента дивидендов для %s", ticker)
	}
	last := md.Times[n-1]
	divs, err := s.divs.GetDividends(ctx, instrumentID, last.AddDate(0, 0, -30), last.AddDate(0, 0, 7))
	if err != nil {
		return "", fmt.Errorf("gap_fade: дивиденды %s: %w", ticker, err)
	}
	today := svc.GapDay(last)
	if svc.GapDividendExDays(divs, md.Times)[today] {
		return fmt.Sprintf("день дивидендной отсечки %s — гэп дивидендный, вход пропущен", today), nil
	}
	return "", nil
}
```

Verify no import cycle: `go list -deps ./internal/service/trading_strategy/gap_fade/live | grep rsi_pullback/live` must print nothing.

- [ ] **Step 5: Run — PASS**

Run: `go test ./internal/service/trading_strategy/gap_fade/... ./internal/service/backtest/... ./cmd/gapwf/... -race`

- [ ] **Step 6: Commit**

```bash
git add internal/service/backtest cmd/gapwf internal/service/trading_strategy/gap_fade
git commit -m "feat(gap_fade): фильтр дня дивидендной отсечки, общее с gapwf правило

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

---

### Task 4: Wiring, pullparity, env, end-to-end runner tests

**Files:**
- Modify: `internal/service_provider/client.go` (add `GetGapFadeSender`)
- Modify: `internal/service_provider/service.go` (attach guest)
- Modify: `cmd/pullparity/main.go`, Test: `cmd/pullparity/main_test.go`
- Modify: `env/prod.env`, `env/prod.env.example`, `env/local.env.example`
- Create: `internal/service/trading_strategy/rsi_pullback/live/gapfade_e2e_test.go`

**Interfaces:**
- Consumes: `gaplive.New`, `gaplive.StrategyFor`, `gaplive.Name` (Tasks 2–3), `config.GapFade`, `TelegramClient.TopicGapFade`.

- [ ] **Step 1: End-to-end runner tests (real adapter, real core)**

`internal/service/trading_strategy/rsi_pullback/live/gapfade_e2e_test.go`:

```go
package live

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/mock"

	"tinvest/internal/config"
	imodel "tinvest/internal/model"
	gaplive "tinvest/internal/service/trading_strategy/gap_fade/live"
	"tinvest/internal/service/trading_strategy/livecore/statestore"
	"tinvest/internal/service/trading_strategy/rsi_pullback/live/dto"
	grpcmodel "tinvest/pkg/client/grpc/model"
)

// gapTape — лента gap_fade: 120 баров по 100 до 23:30 9 марта, затем бары 10 марта с 06:30 по
// last. Первый бар дня открывается гэпом 85 и закрывается 88 (дневной ATR фикстуры dailies = 10,
// гэп −1.5 ATR), остальные стоят на 90 с диапазоном ±0.5 — ни цели 97, ни стопа 78.
func gapTape(last time.Time) []*imodel.CandleItemTechAnalyse {
	closes := make([]float64, 120)
	for i := range closes {
		closes[i] = 100
	}
	out := tape30m(time.Date(2026, 3, 9, 23, 30, 0, 0, msk), closes)
	for t := time.Date(2026, 3, 10, 6, 30, 0, 0, msk); !t.After(last); t = t.Add(30 * time.Minute) {
		o, c := 90.0, 90.0
		if t.Hour() == 6 {
			o, c = 85, 88
		}
		out = append(out, &imodel.CandleItemTechAnalyse{Time: t, Open: qf(o), High: qf(max(o, c) + 0.5),
			Low: qf(min(o, c) - 0.5), Close: qf(c), Volume: 1000, IsComplete: true})
	}
	return out
}

type noDivs struct{}

func (noDivs) GetDividends(context.Context, string, time.Time, time.Time) ([]*grpcmodel.Dividend, error) {
	return nil, nil
}

func gapGuest() *gaplive.Strategy {
	return gaplive.New(&config.GapFadeConfig{Tickers: []string{"GAZP"}, BuyPct: 5}, nil, noDivs{})
}

// Гэп-день: пасс 07:01 входит (dry-run, боевых стратегий нет), стоп и цель заморожены от close
// сигнала 88: стоп 88 − 1.0·10 = 78, цель 88 + 0.75·(100 − 88) = 97.
func TestGapFadeEntersOnGapMorning(t *testing.T) {
	now := time.Date(2026, 3, 10, 7, 1, 0, 0, msk)
	e := newSharedEnv(t, now, nil, nil, gapGuest())
	e.instruments.EXPECT().Shares(mock.Anything).Return(sharesOf("GAZP"), nil)
	e.expectCandles(gapTape(time.Date(2026, 3, 10, 6, 30, 0, 0, msk)), dailies(now, 60))
	e.ops.EXPECT().GetPortfolio(mock.Anything, mock.Anything).Return(nil, nil)
	e.ops.EXPECT().GetPortfolioTotal(mock.Anything, mock.Anything).Return(1_000_000.0, nil)
	e.ops.EXPECT().GetAvailableCash(mock.Anything, mock.Anything).Return(1_000_000.0, nil)
	e.tg.EXPECT().SendMessage(mock.Anything).Return(nil).Maybe()

	if err := e.svc.Run(context.Background(), dto.Run{}); err != nil {
		t.Fatalf("Run: %v", err)
	}
	en, ok := e.state(t)["GAZP"]
	if !ok || en.Strategy != "gap_fade" {
		t.Fatalf("нет входа gap_fade: %+v", e.state(t))
	}
	if en.StopLoss != 78 || en.TakeProfit != 97 || en.StopPrice != 78 || en.StopReason != "SL" {
		t.Fatalf("уровни = stopLoss %v, tp %v, stop %v/%q; want 78/97/78/SL", en.StopLoss, en.TakeProfit, en.StopPrice, en.StopReason)
	}
}

func seededGap(now time.Time) statestore.Entry {
	return statestore.Entry{Ticker: "GAZP", Strategy: "gap_fade", EntryTime: time.Date(2026, 3, 10, 7, 1, 0, 0, msk),
		EntryPrice: 88, EntryATR: 10, TakeProfit: 97, StopLoss: 78, MaxFav: 88, Quantity: 100}
}

// Цель: бар с high ≥ 97 — SELL на следующем пассе, запись удалена.
func TestGapFadeTakeProfitExit(t *testing.T) {
	now := time.Date(2026, 3, 10, 8, 1, 0, 0, msk)
	e := newSharedEnv(t, now, nil, nil, gapGuest())
	e.seed(t, seededGap(now))
	tape := gapTape(time.Date(2026, 3, 10, 7, 30, 0, 0, msk))
	tape[len(tape)-1].High = qf(98)
	e.instruments.EXPECT().Shares(mock.Anything).Return(sharesOf("GAZP"), nil)
	e.expectCandles(tape, dailies(now, 60))
	e.ops.EXPECT().GetPortfolio(mock.Anything, mock.Anything).Return([]*grpcmodel.Position{held("GAZP", 100)}, nil)
	e.tg.EXPECT().SendMessage(mock.Anything).Return(nil).Maybe()

	if err := e.svc.Run(context.Background(), dto.Run{}); err != nil {
		t.Fatalf("Run: %v", err)
	}
	if _, ok := e.state(t)["GAZP"]; ok {
		t.Fatal("TP не закрыл позицию")
	}
}

// EOD: пасс 19:01 после бара 18:30 закрывает позицию — вне окна входа сопровождение работает.
func TestGapFadeEODExitOutsideEntryWindow(t *testing.T) {
	now := time.Date(2026, 3, 10, 19, 1, 0, 0, msk)
	e := newSharedEnv(t, now, nil, nil, gapGuest())
	e.seed(t, seededGap(now))
	e.instruments.EXPECT().Shares(mock.Anything).Return(sharesOf("GAZP"), nil)
	e.expectCandles(gapTape(time.Date(2026, 3, 10, 18, 30, 0, 0, msk)), dailies(now, 60))
	e.ops.EXPECT().GetPortfolio(mock.Anything, mock.Anything).Return([]*grpcmodel.Position{held("GAZP", 100)}, nil)
	e.tg.EXPECT().SendMessage(mock.Anything).Return(nil).Maybe()

	if err := e.svc.Run(context.Background(), dto.Run{}); err != nil {
		t.Fatalf("Run: %v", err)
	}
	if _, ok := e.state(t)["GAZP"]; ok {
		t.Fatal("EOD не закрыл позицию")
	}
}

// Тот же гэп в 12:01 — вне окна: свечи не запрашиваются (market-мок без ожиданий), входа нет.
func TestGapFadeDoesNotFetchOutsideEntryWindow(t *testing.T) {
	now := time.Date(2026, 3, 10, 12, 1, 0, 0, msk)
	e := newSharedEnv(t, now, nil, nil, gapGuest())
	e.instruments.EXPECT().Shares(mock.Anything).Return(sharesOf("GAZP"), nil)
	e.ops.EXPECT().GetPortfolio(mock.Anything, mock.Anything).Return(nil, nil)
	e.tg.EXPECT().SendMessage(mock.Anything).Return(nil).Maybe()

	if err := e.svc.Run(context.Background(), dto.Run{}); err != nil {
		t.Fatalf("Run: %v", err)
	}
	if len(e.state(t)) != 0 {
		t.Fatal("вход вне окна")
	}
}
```

If the TP or EOD test does not close the position, debug before changing the fixture: in dry-run all sells short-circuit, so a remaining entry means `Decide` returned no SELL — check bar times (`Position.EntryTime` is fill minus 30m = 06:31, same date) and that `md.Times[n-1]` / `md.Times[n-2]` are both 10 March.

Run: `go test ./internal/service/trading_strategy/rsi_pullback/live/ -run GapFade -v` — expect PASS already (Tasks 1–3 implement everything); if a test fails, the failure is a real defect in Tasks 1–3 to fix here.

- [ ] **Step 2: service_provider wiring**

`client.go`, after `GetRSIZoneSender`:

```go
func (s *ServiceProvider) GetGapFadeSender() (telegram.Client, error) {
	return s.topicSender(s.appConfig.TelegramClient.TopicGapFade, "gap_fade")
}
```

`service.go`, in `GetRSIPullbackLiveService`, after the rsi_zone block (import `gaplive "tinvest/internal/service/trading_strategy/gap_fade/live"`):

```go
			// gap_fade — третий гость того же счёта (docs/gap_fade/live.md): дивиденды для
			// фильтра дня отсечки берутся instruments-клиентом счёта rsi_pullback.
			if gap := serviceProvider.appConfig.GapFade; gap.Enabled() {
				gapTG, _ := serviceProvider.GetGapFadeSender()
				guests = append(guests, gaplive.New(gap, gapTG, grpcClient.InstrumentsServiceClient()))
			}
```

Update the function's doc comment: "rsi_zone and gap_fade join it as guests".

Confirm `grpcClient.InstrumentsServiceClient()` satisfies `gaplive.DividendsClient` — `go build ./internal/...` proves it.

- [ ] **Step 3: pullparity**

In `cmd/pullparity/main.go`: import `gaplive "tinvest/internal/service/trading_strategy/gap_fade/live"`; flag help `"rsi_pullback | rsi_zone | gap_fade"`; package doc line `rsi_pullback (default), rsi_zone or gap_fade`; in `resolveDecider` add

```go
	case "gap_fade":
		if st, ok := gaplive.StrategyFor(ticker); ok {
			return st, nil
		}
```

and the default error text `(rsi_pullback | rsi_zone | gap_fade)`.

In `cmd/pullparity/main_test.go`, extend `TestResolveDeciderPicksStrategyRegistry` (read it first and follow its table/asserts) with: `resolveDecider("gap_fade", "SBER")` returns a decider with `Lookback() == 64` and no error; `resolveDecider("gap_fade", "")` returns an error.

Run: `go test ./cmd/pullparity/ -v` — PASS.

- [ ] **Step 4: env files**

`env/prod.env` — replace the gap_fade block (currently the comment and `GAP_FADE_TICKERS=` line written by the screener commit) with:

```
# gap_fade на счёте rsi_pullback (docs/gap_fade/live.md): счёт, токен и расписание общие.
# Вселенная — из cmd/gapscreen, проверена cmd/gapwf. Выкат — бумажный режим при боевом
# rsi_pullback: входы приходят только уведомлениями и тикер не занимают.
GAP_FADE_TICKERS=<keep the existing 46-ticker value verbatim>
GAP_FADE_BUY_PCT=5
GAP_FADE_TRADE_ENABLED=false
GAP_FADE_NOTIFY_ENABLED=true
```

`env/prod.env.example` — replace the `# gap_fade (backtest-only)...` block with the same four lines, but `GAP_FADE_TICKERS=` left empty, and add `TELEGRAM_TOPIC_GAP_FADE=` after `TELEGRAM_TOPIC_RSI_ZONE=`. `env/local.env.example` — add `TELEGRAM_TOPIC_GAP_FADE=` after `TELEGRAM_TOPIC_RSI_ZONE=` and the same block with empty tickers after the RSI Zone block. Do not add `TELEGRAM_TOPIC_GAP_FADE` to `env/prod.env` (the owner creates the topic; unset means General).

Run: `go test ./internal/config/ ./cmd/gapwf/ -race` — PASS (gapwf reads `GAP_FADE_TICKERS` from prod.env).

- [ ] **Step 5: Full check**

Run: `go build ./internal/... ./pkg/... ./cmd/... && go test -race ./internal/... ./cmd/... ./pkg/...`
Expected: PASS except the known pre-existing `bonds/computable TestCalculateProfit/ОФЗ_с_дисконтом` failure.

- [ ] **Step 6: Commit**

```bash
git add internal/service_provider cmd/pullparity env internal/service/trading_strategy/rsi_pullback/live/gapfade_e2e_test.go
git commit -m "feat(gap_fade): гость раннера rsi_pullback в бумажном режиме, сверка pullparity

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

---

### Task 5: Documentation

**Files:**
- Create: `docs/gap_fade/live.md`
- Modify: `docs/gap_fade/strategy.md`, `docs/rsi_zone/live.md`, `docs/rsi_pullback/live.md`, `CLAUDE.md`

**Interfaces:** none (docs only). Rules: Russian, mechanics only — no run results, PF, trade counts, onboarding dates.

- [ ] **Step 1: Write `docs/gap_fade/live.md`**

Model it on `docs/rsi_zone/live.md` (read it first). Sections, each with the concrete content below:

1. **Что делает** — гость раннера счёта rsi_pullback; общая механика — `../rsi_pullback/live.md` §§1–7, совместная работа — `../rsi_zone/live.md` §2; код `internal/service/trading_strategy/gap_fade/live` (адаптер `strategy.go`, реестр `registry.go`), литерал `gap_fade/strategy/shared`.
2. **Общий счёт** — порядок адаптеров `[rsi_pullback, rsi_zone, gap_fade]`; занятый тикер закрыт; позиция тикера, убранного из вселенной, ведётся до выхода (реестр отдаёт общий литерал любому тикеру); пример ручной разметки записи `gap_fade` с полями `strategy`, `entryTime`, `entryPrice`, `entryATR`, `takeProfit`, `stopLoss`, `maxFav`, `quantity`.
3. **Параметры** — таблица общего литерала (значения из Global Constraints) и правило: калибровки по тикерам нет, литерал меняется только вместе со сторожевым тестом.
4. **Тайминг входа** — сигнал по close бара 06:30, покупка рынком на пассе 07:01 = аналог `EntryAtNextOpen`; окно входа (будний день, до 10:00 MSK) и зачем оно (нагрузка на свечной API).
5. **Стоп и выходы** — стоп заморожен от close сигнала в `stopLoss`, биржевая заявка GTC, уровень не меняется; таблица выходов SL/TP/EOD/новая дата из спеки; почему TP рыночный, а не лимитной заявкой (две SELL-заявки без OCO).
6. **Дивидендная отсечка** — `EntryFilter`, правило `svc.GapDividendExDays` общее с `cmd/gapwf`, окно запроса, fail-closed при сбое API, бар не расходуется.
7. **Реконструкция** — поля из спеки, стоп от цены входа, без цели.
8. **Бумажный режим** — ссылка на `../rsi_zone/live.md` §4 (правила те же).
9. **Конфигурация** — таблица пяти переменных из Global Constraints; Telegram-префикс «[Gap Fade] », тема `TELEGRAM_TOPIC_GAP_FADE`.
10. **Пауза** — `GAP_FADE_BUY_PCT=0` останавливает входы с сопровождением; остальные способы и их цена — как у rsi_zone (`../rsi_zone/live.md`, «Пауза стратегии»).
11. **Сверка** — `go run ./cmd/pullparity -strategy gap_fade -tickers <GAP_FADE_TICKERS> -months 24`, приёмка ноль расхождений.
12. **Порядок выката** — четыре шага из спеки.
13. **Пограничные случаи** — тикер без вечерней сессии (`pendingExit`, стоп GTC через ночь).

- [ ] **Step 2: Update other docs**

- `docs/gap_fade/strategy.md`: replace the bullet "Live нет. …" (around line 94–96) with one line: live — гость раннера rsi_pullback, см. [live.md](live.md).
- `docs/rsi_zone/live.md` §2 «Приоритет входа»: order `[rsi_pullback, rsi_zone, gap_fade]`, one sentence that gap_fade is the third guest with a link to `../gap_fade/live.md`.
- `docs/rsi_pullback/live.md` line ~25: "остальные (`rsi_zone`, `gap_fade`) подключаются гостями".
- `CLAUDE.md`, the gap_fade entry: replace "backtest-only 30m long intraday fade" with "30m long intraday fade" and append: "live — гость раннера счёта rsi_pullback (`gap_fade/live`, общий счёт, окно входа и фильтр дивидендной отсечки; docs/gap_fade/live.md), сверка — `cmd/pullparity -strategy gap_fade`".

- [ ] **Step 3: Check and commit**

Run: `grep -nE 'PF|profit factor|[0-9]+ сделок' docs/gap_fade/live.md` — expect no output.

```bash
git add docs CLAUDE.md
git commit -m "docs(gap_fade): live гостем раннера rsi_pullback

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```
