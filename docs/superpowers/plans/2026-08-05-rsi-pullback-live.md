# rsi_pullback: прод-раннер — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Поднять live-раннер стратегии `rsi_pullback` на отдельном брокерском счёте так, чтобы он торговал те же сигналы, что воспроизводит бэктест, с настраиваемым процентом счёта на сделку (дефолт 5%).

**Architecture:** Общие с `reversion` части live-слоя (ордера, стоп-заявки, сайзинг, стейт, уведомления, конвертер свечей) выносятся в `internal/service/trading_strategy/livecore/`. Поверх них строится `rsi_pullback/live` с одним получасовым пассом (`в позиции ? manage : buy`), собственной сборкой `MarketData` (30m-окно чанками + дневные свечи + TodayHigh/Low) и биржевой стоп-заявкой для внутрибарных SL/TRAIL. Команда `cmd/pullparity` бар-в-бар доказывает, что живая сборка данных и вердикты `Decide` идентичны бэктесту.

**Tech Stack:** Go 1.25, gRPC (Tinkoff Invest API), `heetch/confita` (конфиг из env), `mockery` v2 (моки), `mage` (lint/test/mocks), Telegram Bot API.

## Global Constraints

- Спека: `docs/superpowers/specs/2026-08-05-rsi-pullback-live-design.md`. Ветка: `feat/rsi-pullback`.
- Гейт после каждой задачи: `./bin/mage ci` (golangci-lint v2 + `go test -race ./...` + проверка дрейфа моков). Сборка проверяется как `go build ./internal/... ./pkg/... ./cmd/...` — `go build ./...` падает на `magefiles`.
- `reversion` торгует реальными деньгами прямо сейчас. В переносимых пакетах **не меняется логика** — только расположение и две явно оговорённые правки API (`statestore.Entry.TakeProfit`, метка стратегии в `notifier.Alert`).
- Комментарии и сообщения — на русском, как в остальном live-коде; названия пакетов, типов и функций — на английском.
- TDD: сначала падающий тест, потом минимальная реализация. Коммит после каждой задачи.
- Вселенная на старте: `UGLD`, `T`, `GAZP`. Дефолт `BuyPct = 5`, `TradeEnabled = false`, `Schedule = "1,31 6-23 * * *"`.
- Выходные бары MOEX из 30m-окна **не отфильтровываются** — ядро отсеивает их само, и бэктест видит их в окне.
- TP исполняется по close-модели (market SELL на закрытии бара с `high ≥ цели`), биржевая take-profit заявка не ставится.

---

## Файловая структура

**Новое:**

| Файл | Ответственность |
|---|---|
| `internal/service/trading_strategy/livecore/executor/` | market BUY/SELL + dry-run (переезд) |
| `internal/service/trading_strategy/livecore/stoporders/` | биржевые стоп-заявки (переезд) |
| `internal/service/trading_strategy/livecore/sizing/` | лоты из процента счёта (переезд) |
| `internal/service/trading_strategy/livecore/statestore/` | атомарный JSON-стейт (переезд + `TakeProfit`) |
| `internal/service/trading_strategy/livecore/notifier/` | тексты Telegram (переезд + метка стратегии) |
| `internal/service/trading_strategy/livecore/candles/` | `CandleClient`, `ToCandles` (вынуто из `reversion/live/marketdata`) |
| `internal/service/trading_strategy/rsi_pullback/live/live.go` | `service`, `Run`, общие хелперы |
| `internal/service/trading_strategy/rsi_pullback/live/pass.go` | единственный пасс: вход и сопровождение |
| `internal/service/trading_strategy/rsi_pullback/live/registry.go` | параметры по тикеру |
| `internal/service/trading_strategy/rsi_pullback/live/dto/run.go` | `Run{Scheduler}` |
| `internal/service/trading_strategy/rsi_pullback/live/marketdata/` | сборка `MarketData` (30m + дневные + TodayHigh/Low) |
| `internal/service/trading_strategy/rsi_pullback/live/reconstruct/` | восстановление стейта по API |
| `internal/service/trading_strategy/rsi_pullback/live/scheduler/` | cron-обёртка |
| `internal/config/rsi_pullback.go` | `RSIPullbackConfig` |
| `cmd/pullparity/main.go` | сверка живой сборки с бэктестом |
| `docs/rsi_pullback/live.md` | справочник раннера |

**Изменяемое:** `internal/domain/backtest/engine.go` (экспорт `TodayExtent`), `rsi_pullback/strategy/core/core.go` (экспорт `DesiredStop`), `rsi_pullback/strategy/{tbank,gazp}/*.go` (комментарии), `reversion/live/*.go` (импорты), `internal/config/config.go`, `internal/app/init_config.go`, `internal/app/app.go`, `internal/service_provider/{client,service}.go`, `internal/config/telegram_client.go`, `.mockery.yaml`, `CLAUDE.md`.

---

### Task 1: Вынос `executor`, `stoporders`, `sizing` в `livecore`

Чистый переезд трёх пакетов без изменения их API. Ставит фундамент для `rsi_pullback/live` и первым проверяет, что тесты `reversion` держат перенос.

**Files:**
- Move: `internal/service/trading_strategy/reversion/live/executor/` → `internal/service/trading_strategy/livecore/executor/`
- Move: `internal/service/trading_strategy/reversion/live/stoporders/` → `internal/service/trading_strategy/livecore/stoporders/`
- Move: `internal/service/trading_strategy/reversion/live/sizing/` → `internal/service/trading_strategy/livecore/sizing/`
- Modify: `internal/service/trading_strategy/reversion/live/live.go`, `buy.go`, `manage.go`, `service_test.go` (импорты)
- Modify: `.mockery.yaml`

**Interfaces:**
- Consumes: ничего (первая задача).
- Produces:
  - `livecore/executor`: `type OrdersClient interface { PostOrder(ctx, *investapi.PostOrderRequest, ...grpc.CallOption) (*investapi.PostOrderResponse, error) }`; `type Result struct { Placed bool; FilledLots int64; FillPrice float64 }`; `func New(c OrdersClient, accountID string, tradeEnabled bool) *Executor`; `(*Executor).Buy(ctx, instrumentID string, lots int64) (Result, error)`; `(*Executor).Sell(...)` — та же сигнатура.
  - `livecore/stoporders`: `type Client interface { investapi.StopOrdersServiceClient }`; `type ActiveStop struct { InstrumentUID, StopOrderID string; StopPrice float64; Lots int64 }`; `type Result struct { Placed bool; OrderID string }`; `func New(c Client, accountID string, tradeEnabled bool) *Executor`; `(*Executor).Place(ctx, instrumentID string, lots int64, stopPrice, minPriceIncrement float64) (Result, error)`; `.Cancel(ctx, stopOrderID string) error`; `.Executed(ctx, stopOrderID string) (bool, error)`; `.List(ctx) ([]ActiveStop, error)`; `func RoundDownToIncrement(price, incr float64) float64`.
  - `livecore/sizing`: `func Lots(buyPct, accountValue, cash, price float64, lot int32) (int64, bool, string)`.

- [ ] **Step 1: Перенести пакеты и переписать импорты**

```bash
mkdir -p internal/service/trading_strategy/livecore
git mv internal/service/trading_strategy/reversion/live/executor internal/service/trading_strategy/livecore/executor
git mv internal/service/trading_strategy/reversion/live/stoporders internal/service/trading_strategy/livecore/stoporders
git mv internal/service/trading_strategy/reversion/live/sizing internal/service/trading_strategy/livecore/sizing

grep -rl 'reversion/live/\(executor\|stoporders\|sizing\)' --include='*.go' . \
  | xargs sed -i 's#trading_strategy/reversion/live/executor#trading_strategy/livecore/executor#g; s#trading_strategy/reversion/live/stoporders#trading_strategy/livecore/stoporders#g; s#trading_strategy/reversion/live/sizing#trading_strategy/livecore/sizing#g'
```

- [ ] **Step 2: Обновить doc-комментарии пакетов**

Три пакета описывают себя как принадлежащие reversion. Заменить первую строку каждого:

`livecore/executor/executor.go`:
```go
// Package executor places (or dry-runs) whole-position market orders for the live
// strategy runners. Each order carries a fresh UUID order_id for idempotency.
```

`livecore/stoporders/stoporders.go`:
```go
// Package stoporders places (or dry-runs) the single protective stop-market SELL
// order a live runner keeps on the exchange per open position. Each order carries a
// fresh UUID order_id for idempotency.
```

`livecore/sizing/sizing.go`:
```go
// Package sizing computes whole-lot order quantities for the live strategy runners from
// a percentage of total account value, capped by available cash.
```

- [ ] **Step 3: Обновить `.mockery.yaml`**

Заменить два блока путей (остальные записи не трогать):

```yaml
  tinvest/internal/service/trading_strategy/livecore/executor:
    interfaces:
      OrdersClient:
  tinvest/internal/service/trading_strategy/livecore/stoporders:
    interfaces:
      Client:
```

- [ ] **Step 4: Перегенерировать моки и собрать**

Run: `./bin/mage mocks && go build ./internal/... ./pkg/... ./cmd/...`
Expected: сборка без ошибок; `git status` показывает моки на новых путях.

- [ ] **Step 5: Прогнать тесты**

Run: `go test ./internal/service/trading_strategy/... -race`
Expected: PASS, включая `reversion/live` (931 строка `service_test.go`) и перенесённые тесты пакетов.

- [ ] **Step 6: Коммит**

```bash
git add -A
git commit -m "refactor(live): выношу executor/stoporders/sizing в livecore"
```

---

### Task 2: Вынос `statestore` и `notifier` в `livecore`

Переезд с двумя минимальными правками API: `Entry` получает поле `TakeProfit` (нужно для close-модели TP у `rsi_pullback`; `reversion` его никогда не пишет, `omitempty` держит формат файла прежним), а `Alert` — метку стратегии вместо захардкоженного «Reversion».

**Files:**
- Move: `internal/service/trading_strategy/reversion/live/statestore/` → `internal/service/trading_strategy/livecore/statestore/`
- Move: `internal/service/trading_strategy/reversion/live/notifier/` → `internal/service/trading_strategy/livecore/notifier/`
- Modify: `internal/service/trading_strategy/livecore/statestore/statestore.go` (+ поле), `statestore_test.go`
- Modify: `internal/service/trading_strategy/livecore/notifier/notifier.go` (+ параметр), `notifier_test.go`
- Modify: `internal/service/trading_strategy/reversion/live/{live,buy,manage}.go`, `service_test.go`, `reconstruct/reconstruct.go`, `reconstruct/reconstruct_test.go`

**Interfaces:**
- Consumes: ничего из Task 1.
- Produces:
  - `livecore/statestore`: `type Entry struct { Ticker string; EntryTime time.Time; EntryPrice, EntryATR, MaxFav float64; Quantity int64; TakeProfit float64; StopOrderID string; StopPrice float64; StopReason string }`; `type Store interface { Load() (map[string]Entry, error); Save(map[string]Entry) error }`; `func New(path string) *FileStore`.
  - `livecore/notifier`: `func Entry(ticker string, price float64, lots, qty int64, paper bool) string`; `func Exit(ticker, reason string, price float64, qty int64, paper bool) string`; `func Skip(ticker, reason string) string`; `func Alert(strategy, ticker, message string) string`; `func StopSet(ticker string, price float64, reason string, paper bool) string`.

- [ ] **Step 1: Перенести пакеты и переписать импорты**

```bash
git mv internal/service/trading_strategy/reversion/live/statestore internal/service/trading_strategy/livecore/statestore
git mv internal/service/trading_strategy/reversion/live/notifier internal/service/trading_strategy/livecore/notifier

grep -rl 'reversion/live/\(statestore\|notifier\)' --include='*.go' . \
  | xargs sed -i 's#trading_strategy/reversion/live/statestore#trading_strategy/livecore/statestore#g; s#trading_strategy/reversion/live/notifier#trading_strategy/livecore/notifier#g'
```

- [ ] **Step 2: Написать падающие тесты на обе правки API**

В `internal/service/trading_strategy/livecore/statestore/statestore_test.go` дописать:

```go
func TestEntryRoundTripsTakeProfit(t *testing.T) {
	path := filepath.Join(t.TempDir(), "state.json")
	s := New(path)
	want := map[string]Entry{"UGLD": {Ticker: "UGLD", EntryPrice: 0.6, TakeProfit: 0.72}}
	if err := s.Save(want); err != nil {
		t.Fatalf("Save: %v", err)
	}
	got, err := s.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if got["UGLD"].TakeProfit != 0.72 {
		t.Fatalf("TakeProfit = %v, want 0.72", got["UGLD"].TakeProfit)
	}
}

// Стейт, записанный reversion (без поля takeProfit), обязан читаться как TakeProfit=0,
// а не ломать разбор файла: формат общий для обеих стратегий.
func TestEntryWithoutTakeProfitLoadsAsZero(t *testing.T) {
	path := filepath.Join(t.TempDir(), "state.json")
	if err := os.WriteFile(path, []byte(`{"UGLD":{"ticker":"UGLD","entryPrice":0.6}}`), 0o600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	got, err := New(path).Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if got["UGLD"].TakeProfit != 0 {
		t.Fatalf("TakeProfit = %v, want 0", got["UGLD"].TakeProfit)
	}
}
```

В `internal/service/trading_strategy/livecore/notifier/notifier_test.go` дописать:

```go
func TestAlertUsesStrategyLabel(t *testing.T) {
	got := Alert("RSI Pullback", "UGLD", "стоп не выставлен")
	if !strings.Contains(got, "RSI Pullback UGLD") {
		t.Fatalf("Alert = %q, want it to name the strategy and the ticker", got)
	}
	if strings.Contains(got, "Reversion") {
		t.Fatalf("Alert = %q, must not hardcode Reversion", got)
	}
}
```

- [ ] **Step 3: Прогнать тесты — убедиться, что падают**

Run: `go test ./internal/service/trading_strategy/livecore/... -run 'TakeProfit|AlertUsesStrategyLabel' -v`
Expected: FAIL — `unknown field TakeProfit` и `too many arguments in call to Alert`.

- [ ] **Step 4: Внести обе правки**

В `livecore/statestore/statestore.go` в `Entry` после `Quantity`:

```go
	// TakeProfit — цель, замороженная на входе. Бэктест хранит её на позиции
	// (strategy.Position.TakeProfit), а живой раннер обязан пережить её через перезапуск,
	// иначе close-модель TP не с чем сравнивать high бара. reversion её не пишет —
	// omitempty оставляет формат файла прежним.
	TakeProfit float64 `json:"takeProfit,omitempty"`
```

В `livecore/notifier/notifier.go`:

```go
// Alert renders an operational alert (e.g. state reconstructed, order rejected).
// strategy — метка раннера в заголовке: пакет общий для нескольких стратегий, и по
// сообщению должно быть видно, чей раннер его прислал.
func Alert(strategy, ticker, message string) string {
	return fmt.Sprintf("⚠️ <b>%s %s</b>\n  %s", strategy, ticker, message)
}
```

- [ ] **Step 5: Обновить все вызовы `Alert` в reversion**

```bash
grep -rl 'notifier\.Alert(' --include='*.go' internal/service/trading_strategy/reversion \
  | xargs sed -i 's#notifier\.Alert(#notifier.Alert("Reversion", #g'
```

Проверить глазами: в `manage.go` есть вызов `notifier.Alert("reversion", "GetStopOrders недоступен: "+listErr.Error())` — после sed он станет `Alert("Reversion", "reversion", ...)`. Заменить второй аргумент на пустую строку не нужно: поправить вручную на `notifier.Alert("Reversion", "", "GetStopOrders недоступен: "+listErr.Error())`.

- [ ] **Step 6: Прогнать тесты**

Run: `./bin/mage mocks && go test ./internal/... -race`
Expected: PASS во всех пакетах.

- [ ] **Step 7: Коммит**

```bash
git add -A
git commit -m "refactor(live): выношу statestore/notifier в livecore, TakeProfit в стейте"
```

---

### Task 3: Вынос `CandleClient` и `ToCandles` в `livecore/candles`

`rsi_pullback/live/marketdata` нужен тот же интерфейс клиента и тот же конвертер API-свечей в доменные, что у reversion. Выносим только их: сама `Assemble` у reversion (часовое окно + 4H, кламп под кап интервала) остаётся нетронутой — переписывать её под чанки нельзя, это изменение поведения в торгующем коде.

**Files:**
- Create: `internal/service/trading_strategy/livecore/candles/candles.go`
- Create: `internal/service/trading_strategy/livecore/candles/candles_test.go`
- Modify: `internal/service/trading_strategy/reversion/live/marketdata/marketdata.go` (удалить `CandleClient` и `ToCandles`, использовать `candles.*`)
- Modify: `internal/service/trading_strategy/reversion/live/marketdata/marketdata_test.go`, `reconstruct/reconstruct.go`, `reversion/live/{live,buy,manage}.go`, `service_test.go`
- Modify: `.mockery.yaml`

**Interfaces:**
- Consumes: ничего из Task 1–2.
- Produces: `livecore/candles`: `type CandleClient interface { GetCandles(ctx context.Context, instrumentUID *string, interval int32, from, to *timestamppb.Timestamp, limit *int32, withHoliday bool) ([]*imodel.CandleItemTechAnalyse, error) }`; `func ToCandles(in []*imodel.CandleItemTechAnalyse, completedOnly bool) []backtest.Candle`. Мок — `livecore/candles/mocks.MockCandleClient`.

- [ ] **Step 1: Написать падающий тест конвертера**

`internal/service/trading_strategy/livecore/candles/candles_test.go`:

```go
package candles

import (
	"testing"
	"time"

	imodel "tinvest/internal/model"
)

// Quotation в CandleItemTechAnalyse — значение, не указатель (см. reversion/live/service_test.go).
func q(v int64) imodel.Quotation { return imodel.Quotation{Units: v} }

func TestToCandlesDropsIncompleteWhenAsked(t *testing.T) {
	in := []*imodel.CandleItemTechAnalyse{
		{Time: time.Unix(0, 0), Open: q(1), High: q(2), Low: q(1), Close: q(2), Volume: 10, IsComplete: true},
		{Time: time.Unix(1800, 0), Open: q(2), High: q(3), Low: q(2), Close: q(3), Volume: 20, IsComplete: false},
	}
	if got := ToCandles(in, true); len(got) != 1 || got[0].Volume != 10 {
		t.Fatalf("ToCandles(completedOnly) = %+v, want only the completed bar", got)
	}
	if got := ToCandles(in, false); len(got) != 2 {
		t.Fatalf("ToCandles(all) length = %d, want 2", len(got))
	}
}
```

Точный тип поля `Quotation` подсмотреть в `internal/model` — использовать тот, что реально лежит в `CandleItemTechAnalyse.Open`.

- [ ] **Step 2: Прогнать тест — убедиться, что падает**

Run: `go test ./internal/service/trading_strategy/livecore/candles/ -v`
Expected: FAIL — пакета нет.

- [ ] **Step 3: Создать пакет**

`internal/service/trading_strategy/livecore/candles/candles.go` — перенести дословно из `reversion/live/marketdata/marketdata.go` объявление `CandleClient` и функцию `ToCandles` вместе с их doc-комментариями, сменив вводный комментарий пакета на:

```go
// Package candles holds the market-data client surface the live runners share and the
// conversion from API candles to domain candles. Assembling a MarketData snapshot stays
// per-strategy: reversion needs an hourly window plus 4H, rsi_pullback a 30-minute window
// plus dailies, and the two fetch policies have nothing in common beyond these primitives.
package candles
```

- [ ] **Step 4: Переключить reversion на новый пакет**

В `reversion/live/marketdata/marketdata.go` удалить объявление `CandleClient` и функцию `ToCandles`, добавить импорт `"tinvest/internal/service/trading_strategy/livecore/candles"` и заменить использования на `candles.CandleClient` / `candles.ToCandles`. Сигнатуры `Assemble` и `fetchCompleted` меняются только в типе клиента:

```go
func Assemble(ctx context.Context, c candles.CandleClient, instrumentID string,
	lookbackBars, htfEMAPeriod int, now time.Time) (strategy.MarketData, error)
```

Аналогично в `reconstruct.Entry` параметр `cc marketdata.CandleClient` → `cc candles.CandleClient`, и в `reversion/live/live.go` поле `market marketdata.CandleClient` → `market candles.CandleClient`.

- [ ] **Step 5: Обновить `.mockery.yaml` и перегенерировать**

Заменить блок `reversion/live/marketdata` на:

```yaml
  tinvest/internal/service/trading_strategy/livecore/candles:
    interfaces:
      CandleClient:
```

Run: `./bin/mage mocks`
Затем поправить импорты моков в тестах reversion: `reversion/live/marketdata/mocks` → `livecore/candles/mocks`, тип `mocks.MockCandleClient` не меняется.

- [ ] **Step 6: Прогнать тесты**

Run: `go test ./internal/... -race`
Expected: PASS.

- [ ] **Step 7: Коммит**

```bash
git add -A
git commit -m "refactor(live): выношу CandleClient и ToCandles в livecore/candles"
```

---

### Task 4: Конфиг `RSIPullbackConfig` и реестр тикеров

Две статические декларации раннера: откуда берутся настройки запуска и какие параметры соответствуют тикеру. Сюда же — правка комментариев `tbank.go` и `gazp.go`, которые сейчас утверждают обратное принятому решению.

**Files:**
- Create: `internal/config/rsi_pullback.go`, `internal/config/rsi_pullback_test.go`
- Create: `internal/service/trading_strategy/rsi_pullback/live/registry.go`, `registry_test.go`
- Modify: `internal/config/config.go`, `internal/config/telegram_client.go`, `internal/app/init_config.go`
- Modify: `internal/service/trading_strategy/rsi_pullback/strategy/tbank/tbank.go`, `.../gazp/gazp.go` (комментарии)

**Interfaces:**
- Consumes: ничего.
- Produces:
  - `config`: `type RSIPullbackConfig struct { AccountID, Token string; Tickers []string; BuyPct float64; TradeEnabled, NotifyEnabled bool; Schedule string }`; `func NewRSIPullbackConfig() *RSIPullbackConfig`; поле `RSIPullback *RSIPullbackConfig` в `Config`; поле `TopicRSIPullback int` в `TelegramClient`.
  - `rsi_pullback/live`: `func ParamsFor(ticker string) (core.Params, bool)`; `func StrategyFor(ticker string) (*core.Strategy, bool)`.

- [ ] **Step 1: Написать падающие тесты конфига**

`internal/config/rsi_pullback_test.go`:

```go
package config

import "testing"

func TestNewRSIPullbackConfig_Defaults(t *testing.T) {
	c := NewRSIPullbackConfig()
	want := []string{"UGLD", "T", "GAZP"}
	if len(c.Tickers) != len(want) {
		t.Fatalf("default Tickers = %v, want %v", c.Tickers, want)
	}
	for i, w := range want {
		if c.Tickers[i] != w {
			t.Fatalf("default Tickers = %v, want %v", c.Tickers, want)
		}
	}
	if c.BuyPct != 5 {
		t.Fatalf("default BuyPct = %v, want 5", c.BuyPct)
	}
	if c.Schedule != "1,31 6-23 * * *" {
		t.Fatalf("default Schedule = %q, want \"1,31 6-23 * * *\"", c.Schedule)
	}
}

// Забытый флаг никогда не должен выставить реальный ордер.
func TestNewRSIPullbackConfig_TradeDisabledByDefault(t *testing.T) {
	if NewRSIPullbackConfig().TradeEnabled {
		t.Fatal("TradeEnabled default = true, want false")
	}
}

func TestNewRSIPullbackConfig_TokenHasNoDefault(t *testing.T) {
	if tok := NewRSIPullbackConfig().Token; tok != "" {
		t.Fatalf("default Token = %q, want empty (must come from RSI_PULLBACK_TOKEN env)", tok)
	}
}
```

- [ ] **Step 2: Написать падающие тесты реестра**

`internal/service/trading_strategy/rsi_pullback/live/registry_test.go`:

```go
package live

import (
	"testing"

	"tinvest/internal/config"
)

// Каждый тикер дефолтной вселенной обязан находиться в реестре, иначе раннер молча
// не будет торговать его вообще.
func TestEveryDefaultTickerIsRegistered(t *testing.T) {
	for _, ticker := range config.NewRSIPullbackConfig().Tickers {
		if _, ok := ParamsFor(ticker); !ok {
			t.Fatalf("ticker %s from the default universe is missing from the registry", ticker)
		}
	}
}

// Ловушка нулевого значения: тикерные пакеты задают core.Params литералом, и забытое
// поле молча даёт UseRSIExit=0 — то есть выключенный основной выход (61% выходов UGLD).
func TestRegisteredTickersKeepTheRSIExitArmed(t *testing.T) {
	for ticker := range paramsByTicker {
		p, _ := ParamsFor(ticker)
		if p.UseRSIExit != 1 {
			t.Fatalf("%s: UseRSIExit = %d, want 1", ticker, p.UseRSIExit)
		}
	}
}

// Стратегия должна строиться на параметрах тикера, а не на дефолтах ядра.
func TestStrategyForUsesTickerParams(t *testing.T) {
	st, ok := StrategyFor("UGLD")
	if !ok {
		t.Fatal("StrategyFor(UGLD) not ok")
	}
	if st.Ticker() != "UGLD" {
		t.Fatalf("Ticker() = %q, want UGLD", st.Ticker())
	}
	if st.Lookback() < 400 {
		t.Fatalf("Lookback() = %d, want >= 400 (UGLD's volume gate dominates)", st.Lookback())
	}
	if _, ok := StrategyFor("НЕТ-ТАКОГО"); ok {
		t.Fatal("StrategyFor returned ok for an unknown ticker")
	}
}
```

- [ ] **Step 3: Прогнать тесты — убедиться, что падают**

Run: `go test ./internal/config/ ./internal/service/trading_strategy/rsi_pullback/live/ -v`
Expected: FAIL — `NewRSIPullbackConfig` не определена, пакета `live` нет.

- [ ] **Step 4: Создать конфиг**

`internal/config/rsi_pullback.go`:

```go
package config

// RSIPullbackConfig configures the live rsi_pullback runner. Trade and notify are
// independent: both off = dry-run to log only.
type RSIPullbackConfig struct {
	AccountID     string   `config:"RSI_PULLBACK_ACCOUNT_ID,required,backend=env"`
	Token         string   `config:"RSI_PULLBACK_TOKEN,required,backend=env"`
	Tickers       []string `config:"RSI_PULLBACK_TICKERS,backend=env"`
	BuyPct        float64  `config:"RSI_PULLBACK_BUY_PCT,backend=env"`
	TradeEnabled  bool     `config:"RSI_PULLBACK_TRADE_ENABLED,backend=env"`
	NotifyEnabled bool     `config:"RSI_PULLBACK_NOTIFY_ENABLED,backend=env"`
	Schedule      string   `config:"RSI_PULLBACK_SCHEDULE,backend=env"`
}

// NewRSIPullbackConfig returns the config pre-seeded with safe defaults. confita
// overrides any field whose env var is set; unset fields keep these values.
// TradeEnabled defaults to false so a missing flag never places real orders.
//
// Schedule фиксирует три решения. Раз в полчаса — потому что рабочий таймфрейм
// стратегии 30 минут. Минута :01/:31, а не :00/:30 — запас на то, чтобы закрывшийся бар
// успел прийти из API с IsComplete=true. Все семь дней недели — позиция едет через
// выходные, и бэктест отрабатывает выходы на выходных барах MOEX; входы на выходных
// закрывает само ядро (tradingDay). Ночью 00:00-06:00 MSK торгов нет.
func NewRSIPullbackConfig() *RSIPullbackConfig {
	return &RSIPullbackConfig{
		Tickers:  []string{"UGLD", "T", "GAZP"},
		BuyPct:   5,
		Schedule: "1,31 6-23 * * *",
	}
}
```

В `internal/config/config.go` добавить в `Config` поле `RSIPullback *RSIPullbackConfig` (после `Reversion`). В `internal/config/telegram_client.go` добавить `TopicRSIPullback int \`config:"TELEGRAM_TOPIC_RSI_PULLBACK"\``. В `internal/app/init_config.go` в литерал конфига добавить `RSIPullback: config.NewRSIPullbackConfig(),`.

- [ ] **Step 5: Создать реестр**

`internal/service/trading_strategy/rsi_pullback/live/registry.go`:

```go
package live

import (
	"tinvest/internal/service/trading_strategy/rsi_pullback/strategy/core"
	"tinvest/internal/service/trading_strategy/rsi_pullback/strategy/domrf"
	"tinvest/internal/service/trading_strategy/rsi_pullback/strategy/gazp"
	"tinvest/internal/service/trading_strategy/rsi_pullback/strategy/nvtk"
	"tinvest/internal/service/trading_strategy/rsi_pullback/strategy/tbank"
	"tinvest/internal/service/trading_strategy/rsi_pullback/strategy/ugld"
)

// paramsByTicker maps every rsi_pullback ticker the runner knows to its params. The
// configured universe (RSI_PULLBACK_TICKERS) selects which of these actually trade;
// NVTK and DOMRF are registered for completeness but have no calibrated literal yet and
// must not be put into the universe.
var paramsByTicker = map[string]core.Params{
	ugld.Ticker:  ugld.DefaultParams(),
	tbank.Ticker: tbank.DefaultParams(),
	gazp.Ticker:  gazp.DefaultParams(),
	nvtk.Ticker:  nvtk.DefaultParams(),
	domrf.Ticker: domrf.DefaultParams(),
}

// ParamsFor returns the params for a known ticker, ok=false otherwise.
func ParamsFor(ticker string) (core.Params, bool) {
	p, ok := paramsByTicker[ticker]
	return p, ok
}

// StrategyFor returns the strategy for a known ticker, ok=false otherwise.
func StrategyFor(ticker string) (*core.Strategy, bool) {
	p, ok := paramsByTicker[ticker]
	if !ok {
		return nil, false
	}
	return core.NewWithParams(ticker, p), true
}
```

Если `TestRegisteredTickersKeepTheRSIExitArmed` падает на NVTK или DOMRF — не «чинить» их значения, а проверить: у `core.DefaultParams()` `UseRSIExit = 1`, значит падение означает реальную опечатку в литерале тикера.

- [ ] **Step 6: Прогнать тесты**

Run: `go test ./internal/config/ ./internal/service/trading_strategy/rsi_pullback/... -race -v`
Expected: PASS.

- [ ] **Step 7: Привести комментарии `tbank.go` и `gazp.go` в соответствие с решением**

В `tbank.go` заменить абзац, начинающийся «Second, in-sample PF is not an edge claim…», на:

```go
// Second, this literal goes to production WITHOUT walk-forward confirmation. The owner
// decided so on 2026-08-05 knowing the numbers: in-sample PF 1.312 over 67 trades, but two
// consecutive losing years inside that sample (2023 PF 0.43, 2024 PF 0.58) and no OOS run at
// all. Treat a live drawdown here as expected variance of an unvalidated configuration, not
// as evidence that something broke.
```

В `gazp.go` дописать в конец doc-комментария пакета:

```go
// Как и T, GAZP уходит в прод БЕЗ walk-forward подтверждения (решение владельца от
// 2026-08-05): есть только пост-грид литерал, OOS-прогона не было.
```

- [ ] **Step 8: Коммит**

```bash
git add -A
git commit -m "feat(rsi_pullback): конфиг раннера и реестр тикеров"
```

---

### Task 5: Сборка `MarketData` для live

Самая ответственная часть: расхождение здесь означает, что прод торгует другую стратегию. Три источника — 30m-окно чанками, дневные свечи, TodayHigh/Low.

**Files:**
- Create: `internal/service/trading_strategy/rsi_pullback/live/marketdata/marketdata.go`
- Create: `internal/service/trading_strategy/rsi_pullback/live/marketdata/marketdata_test.go`
- Modify: `internal/domain/backtest/engine.go` (экспорт `TodayExtent`), `internal/domain/backtest/engine_test.go` (вызовы)

**Interfaces:**
- Consumes: `livecore/candles.CandleClient`, `livecore/candles.ToCandles` (Task 3).
- Produces:
  - `internal/domain/backtest`: `func TodayExtent(candles []Candle, i int) (high, low float64)` — публичная обёртка, локация MSK внутри пакета.
  - `rsi_pullback/live/marketdata`: `func Assemble(ctx context.Context, c candles.CandleClient, instrumentID string, lookbackBars int, now time.Time) (strategy.MarketData, error)`.

- [ ] **Step 1: Экспортировать `TodayExtent`**

В `internal/domain/backtest/engine.go` переименовать `todayExtent` → `todayExtentIn` (оставив прежнюю сигнатуру с `loc`) и добавить рядом:

```go
// TodayExtent returns the high and low of the MSK calendar day that bar i belongs to,
// scanning back through the (oldest-first) window. Exported so a live runner fills
// MarketData.TodayHigh/TodayLow with the ENGINE's own rule instead of a lookalike:
// AssembleMarketData deliberately leaves those two fields to the caller.
func TodayExtent(candles []Candle, i int) (high, low float64) {
	return todayExtentIn(candles, i, mskLoc)
}
```

Обновить два внутренних вызова в `Run` (строки с `md.TodayHigh, md.TodayLow = todayExtent(...)`) на `todayExtentIn(...)` и вызовы в тестах пакета.

- [ ] **Step 2: Написать падающие тесты сборщика**

`internal/service/trading_strategy/rsi_pullback/live/marketdata/marketdata_test.go`. Хелпер строит поддельного клиента, отдающего 30m или дневные свечи по запрошенному интервалу и окну:

```go
package marketdata

import (
	"context"
	"testing"
	"time"

	"google.golang.org/protobuf/types/known/timestamppb"

	"tinvest/internal/enum"
	imodel "tinvest/internal/model"
)

var msk = func() *time.Location {
	l, _ := time.LoadLocation("Europe/Moscow")
	return l
}()

// fakeClient отдаёт свечи из заранее построенных серий, обрезая их запрошенным окном —
// как делает настоящий API. Записывает окна запросов, чтобы тест мог проверить чанкование.
type fakeClient struct {
	m30, day []*imodel.CandleItemTechAnalyse
	windows  []int32 // интервалы запросов в порядке вызова
	calls    int
}

func (f *fakeClient) GetCandles(_ context.Context, _ *string, interval int32,
	from, to *timestamppb.Timestamp, limit *int32, _ bool) ([]*imodel.CandleItemTechAnalyse, error) {
	f.calls++
	f.windows = append(f.windows, interval)
	src := f.m30
	if interval == enum.Day1.ToNumberInvestAPI() {
		src = f.day
	}
	var out []*imodel.CandleItemTechAnalyse
	for _, c := range src {
		if !c.Time.Before(from.AsTime()) && !c.Time.After(to.AsTime()) {
			out = append(out, c)
		}
	}
	return out, nil
}

// Quotation в CandleItemTechAnalyse — значение, не указатель.
func q(v int64) imodel.Quotation { return imodel.Quotation{Units: v} }

// series30m строит непрерывную ленту из n 30-минутных баров, последний открывается в end.
// Бары идут подряд по календарю, включая выходные, — ровно как лента MOEX в кэше свечей.
func series30m(end time.Time, n int) []*imodel.CandleItemTechAnalyse {
	out := make([]*imodel.CandleItemTechAnalyse, 0, n)
	for i := n - 1; i >= 0; i-- {
		t := end.Add(-time.Duration(i) * 30 * time.Minute)
		out = append(out, &imodel.CandleItemTechAnalyse{
			Time: t, Open: q(100), High: q(101), Low: q(99), Close: q(100),
			Volume: 1000, IsComplete: true,
		})
	}
	return out
}

// seriesDaily строит n дневных баров, последний открывается в MSK-полночь дня end.
func seriesDaily(end time.Time, n int) []*imodel.CandleItemTechAnalyse {
	e := end.In(msk)
	midnight := time.Date(e.Year(), e.Month(), e.Day(), 0, 0, 0, 0, msk)
	out := make([]*imodel.CandleItemTechAnalyse, 0, n)
	for i := n - 1; i >= 0; i-- {
		out = append(out, &imodel.CandleItemTechAnalyse{
			Time: midnight.AddDate(0, 0, -i), Open: q(100), High: q(105), Low: q(95), Close: q(100),
			Volume: 100000, IsComplete: true,
		})
	}
	return out
}
```

Собственно тесты:

```go
// Окно на 403 бара (UGLD) не помещается в один запрос 30m-свечей: API ограничивает
// окно примерно тремя неделями, а каникулы MOEX растягивают эти бары за кап. Сборщик
// обязан догружать чанками, а не молча возвращать короткое окно.
func TestAssembleChunksUntilLookbackIsFilled(t *testing.T) {
	now := time.Date(2026, 3, 10, 12, 0, 0, 0, msk)
	f := &fakeClient{m30: series30m(now.Add(-30*time.Minute), 900), day: seriesDaily(now, 90)}
	md, err := Assemble(context.Background(), f, "uid", 403, now)
	if err != nil {
		t.Fatalf("Assemble: %v", err)
	}
	if len(md.Closes) != 403 {
		t.Fatalf("len(Closes) = %d, want 403", len(md.Closes))
	}
	m30calls := 0
	for _, iv := range f.windows {
		if iv == enum.Minutes30.ToNumberInvestAPI() {
			m30calls++
		}
	}
	if m30calls < 2 {
		t.Fatalf("30m requests = %d, want >= 2 (chunked fetch)", m30calls)
	}
}

// Недобор баров — это ошибка, а не короткое окно: ядро на окне короче EMASlow
// возвращает нулевую серию EMA и молча заваливает трендовый гейт на весь прогон.
func TestAssembleFailsWhenHistoryIsShort(t *testing.T) {
	now := time.Date(2026, 3, 10, 12, 0, 0, 0, msk)
	f := &fakeClient{m30: series30m(now.Add(-30*time.Minute), 50), day: seriesDaily(now, 90)}
	if _, err := Assemble(context.Background(), f, "uid", 403, now); err == nil {
		t.Fatal("Assemble returned nil error on a short history, want an error")
	}
}

// Формирующийся бар не должен попадать в окно: решение принимается по закрытым барам.
func TestAssembleDropsIncompleteBar(t *testing.T) {
	now := time.Date(2026, 3, 10, 12, 0, 0, 0, msk)
	bars := series30m(now.Add(-30*time.Minute), 500)
	forming := *bars[len(bars)-1]
	forming.Time = now
	forming.IsComplete = false
	f := &fakeClient{m30: append(bars, &forming), day: seriesDaily(now, 90)}
	md, err := Assemble(context.Background(), f, "uid", 403, now)
	if err != nil {
		t.Fatalf("Assemble: %v", err)
	}
	if last := md.Times[len(md.Times)-1]; !last.Before(now) {
		t.Fatalf("last bar time = %v, want strictly before now (%v)", last, now)
	}
}

// Дневная свеча текущего дня закрыться не успела: если она попадёт в Daily*, дневной ATR
// и обе границы гейта дня будут посчитаны с заглядыванием в будущее.
func TestAssembleExcludesTodayFromDailies(t *testing.T) {
	now := time.Date(2026, 3, 10, 12, 0, 0, 0, msk)
	f := &fakeClient{m30: series30m(now.Add(-30*time.Minute), 900), day: seriesDaily(now, 90)}
	md, err := Assemble(context.Background(), f, "uid", 403, now)
	if err != nil {
		t.Fatalf("Assemble: %v", err)
	}
	if len(md.DailyTimes) == 0 {
		t.Fatal("DailyTimes is empty, want the completed dailies")
	}
	last := md.DailyTimes[len(md.DailyTimes)-1].In(msk)
	if last.Year() == now.Year() && last.YearDay() == now.YearDay() {
		t.Fatalf("last daily = %v, want strictly before today", last)
	}
}

// Выходные бары MOEX обязаны остаться в 30m-окне: ядро отсеивает их само (isWeekend),
// а бэктест видит их в окне. Фильтрация здесь сломала бы совпадение с бэктестом.
func TestAssembleKeepsWeekendBarsInTheWindow(t *testing.T) {
	now := time.Date(2026, 3, 10, 12, 0, 0, 0, msk)
	f := &fakeClient{m30: series30m(now.Add(-30*time.Minute), 900), day: seriesDaily(now, 90)}
	md, err := Assemble(context.Background(), f, "uid", 403, now)
	if err != nil {
		t.Fatalf("Assemble: %v", err)
	}
	weekend := 0
	for _, tm := range md.Times {
		if wd := tm.In(msk).Weekday(); wd == time.Saturday || wd == time.Sunday {
			weekend++
		}
	}
	if weekend == 0 {
		t.Fatal("no weekend bars in the window: the assembler must not filter them")
	}
}

// TodayHigh/Low обязаны считаться правилом движка, а не похожим кодом.
func TestAssembleFillsTodayExtentFromEngine(t *testing.T) {
	now := time.Date(2026, 3, 10, 12, 0, 0, 0, msk)
	f := &fakeClient{m30: series30m(now.Add(-30*time.Minute), 900), day: seriesDaily(now, 90)}
	md, err := Assemble(context.Background(), f, "uid", 403, now)
	if err != nil {
		t.Fatalf("Assemble: %v", err)
	}
	if md.TodayHigh <= 0 || md.TodayLow <= 0 || md.TodayHigh < md.TodayLow {
		t.Fatalf("TodayHigh/TodayLow = %v/%v, want a valid range", md.TodayHigh, md.TodayLow)
	}
}
```

- [ ] **Step 3: Прогнать тесты — убедиться, что падают**

Run: `go test ./internal/service/trading_strategy/rsi_pullback/live/marketdata/ -v`
Expected: FAIL — `undefined: Assemble`.

- [ ] **Step 4: Реализовать сборщик**

`internal/service/trading_strategy/rsi_pullback/live/marketdata/marketdata.go`:

```go
// Package marketdata assembles the live rsi_pullback MarketData snapshot from Tinkoff
// candles, reusing the backtest's AssembleMarketData and TodayExtent so live and backtest
// build identical inputs. Two series are fetched: a 30-minute window (the strategy's
// working timeframe, sized by Strategy.Lookback) and daily candles (the unit of risk —
// the stop, the target and both thresholds of the day gate are multiples of the daily ATR).
// No 4H series: the core never reads it.
package marketdata

import (
	"context"
	"fmt"
	"sort"
	"time"

	"google.golang.org/protobuf/types/known/timestamppb"

	"tinvest/internal/domain/backtest"
	"tinvest/internal/enum"
	"tinvest/internal/service/trading_strategy/livecore/candles"
	"tinvest/internal/service/trading_strategy/scalping/strategy"
)

// chunkDays bounds one 30-minute request. The API caps a 30-minute request window at
// roughly three weeks (see the CandleInterval doc comments in internal/pb/v1/marketdata.pb.go);
// 14 days keeps us clear of the cap the way the backtest's chunkDaysFor does.
const chunkDays = 14

// maxChunks bounds the walk backwards. UGLD's 403-bar window is about two calendar weeks
// of MOEX 30-minute bars, so two chunks normally suffice; eight leaves room for the New
// Year holidays and a halted instrument without turning a data outage into an endless loop.
const maxChunks = 8

// m30Limit is the API's per-request cap on 30-minute candles.
const m30Limit int32 = 1200

// dailyFetchDays sizes the single daily request. DailyATRPeriod+1 = 15 completed WEEKDAY
// dailies is the most any registered ticker needs; 90 calendar days covers that with room
// for the January holidays, and one request is enough because the daily interval allows
// windows up to six years.
const dailyFetchDays = 90

// dailyLimit caps the daily request; 200 comfortably exceeds dailyFetchDays.
const dailyLimit int32 = 200

// Assemble builds the MarketData snapshot as of `now`. lookbackBars is the 30-minute window
// size (Strategy.Lookback()). Position is left nil for the caller to set.
func Assemble(ctx context.Context, c candles.CandleClient, instrumentID string,
	lookbackBars int, now time.Time) (strategy.MarketData, error) {

	window, err := fetch30m(ctx, c, instrumentID, lookbackBars, now)
	if err != nil {
		return strategy.MarketData{}, err
	}
	if len(window) < lookbackBars {
		// Короткое окно опаснее ошибки: ema.Compute на окне короче периода возвращает
		// нулевую серию, трендовый гейт молча закрывается на весь прогон, и раннер
		// выглядит работающим, ничего не торгуя.
		return strategy.MarketData{}, fmt.Errorf(
			"rsi_pullback marketdata: %d completed 30m candles < lookback %d", len(window), lookbackBars)
	}

	daily, err := fetchDaily(ctx, c, instrumentID, now)
	if err != nil {
		return strategy.MarketData{}, err
	}

	i := len(window) - 1
	md := backtest.AssembleMarketData(window, daily, nil, window[i].Time)
	md.TodayHigh, md.TodayLow = backtest.TodayExtent(window, i)
	return md, nil
}

// fetch30m walks backwards in chunkDays windows until it holds lookbackBars completed bars,
// then returns the last lookbackBars (oldest-first). Chunk boundaries may overlap, so bars
// are de-duplicated by open-time.
func fetch30m(ctx context.Context, c candles.CandleClient, instrumentID string,
	lookbackBars int, now time.Time) ([]backtest.Candle, error) {

	seen := make(map[int64]struct{})
	var all []backtest.Candle
	to := now
	for chunk := 0; chunk < maxChunks && len(all) < lookbackBars; chunk++ {
		from := to.AddDate(0, 0, -chunkDays)
		limit := m30Limit
		raw, err := c.GetCandles(ctx, &instrumentID, enum.Minutes30.ToNumberInvestAPI(),
			timestamppb.New(from), timestamppb.New(to), &limit, true)
		if err != nil {
			return nil, fmt.Errorf("rsi_pullback marketdata: 30m candles: %w", err)
		}
		for _, cd := range candles.ToCandles(raw, true) {
			key := cd.Time.UnixNano()
			if _, dup := seen[key]; dup {
				continue
			}
			seen[key] = struct{}{}
			all = append(all, cd)
		}
		to = from
	}
	sort.Slice(all, func(a, b int) bool { return all[a].Time.Before(all[b].Time) })
	if len(all) > lookbackBars {
		all = all[len(all)-lookbackBars:]
	}
	return all, nil
}

// fetchDaily pulls the completed daily series in one request. visibleDaily inside
// AssembleMarketData then drops anything not closed before the current bar's MSK midnight,
// and the core drops weekend sessions itself, so nothing is filtered here.
func fetchDaily(ctx context.Context, c candles.CandleClient, instrumentID string,
	now time.Time) ([]backtest.Candle, error) {

	from := now.AddDate(0, 0, -dailyFetchDays)
	limit := dailyLimit
	raw, err := c.GetCandles(ctx, &instrumentID, enum.Day1.ToNumberInvestAPI(),
		timestamppb.New(from), timestamppb.New(now), &limit, true)
	if err != nil {
		return nil, fmt.Errorf("rsi_pullback marketdata: daily candles: %w", err)
	}
	return candles.ToCandles(raw, true), nil
}
```

- [ ] **Step 5: Прогнать тесты**

Run: `go test ./internal/service/trading_strategy/rsi_pullback/live/marketdata/ ./internal/domain/backtest/ -race -v`
Expected: PASS.

- [ ] **Step 6: Полный гейт и коммит**

Run: `./bin/mage ci`
Expected: EXIT=0.

```bash
git add -A
git commit -m "feat(rsi_pullback): live-сборка MarketData (30m чанками + дневные + TodayExtent)"
```

---

### Task 6: `cmd/pullparity` — сверка живой сборки с бэктестом

Приёмочный инструмент: доказывает, что `marketdata.Assemble` даёт бит-в-бит тот же снимок и те же вердикты `Decide`, что сборка движка. Стоит сразу после Task 5, чтобы расхождение вскрылось до того, как на сборку ляжет весь пасс.

**Files:**
- Create: `cmd/pullparity/main.go`
- Create: `cmd/pullparity/main_test.go`

**Interfaces:**
- Consumes: `marketdata.Assemble` (Task 5), `live.StrategyFor` (Task 4), `backtest.AssembleMarketData`, `backtest.TodayExtent`, `svc.NewCandleProvider(...).Load(ctx, ticker, instrumentID, interval, from, to, refresh)`.
- Produces: команда `go run ./cmd/pullparity -tickers UGLD,T,GAZP -months 24`; чистые функции `func diffMarketData(want, got strategy.MarketData) []string` и `func diffSignal(want, got model.Signal) []string` — тестируются юнитами.

- [ ] **Step 1: Написать падающие тесты сравнителей**

`cmd/pullparity/main_test.go`:

```go
package main

import (
	"testing"
	"time"

	"tinvest/internal/service/trading_strategy/scalping/model"
	"tinvest/internal/service/trading_strategy/scalping/strategy"
)

func TestDiffMarketDataFindsNothingWhenIdentical(t *testing.T) {
	md := strategy.MarketData{
		Price: 1, Closes: []float64{1, 2}, Highs: []float64{1, 2}, Lows: []float64{1, 2},
		Volumes: []int64{5, 6}, Times: []time.Time{time.Unix(0, 0), time.Unix(1800, 0)},
		DailyCloses: []float64{9}, DailyHighs: []float64{9}, DailyLows: []float64{8},
		DailyTimes: []time.Time{time.Unix(0, 0)}, TodayHigh: 2, TodayLow: 1,
	}
	if d := diffMarketData(md, md); len(d) != 0 {
		t.Fatalf("diffMarketData on identical snapshots = %v, want none", d)
	}
}

// Именно эти поля рвутся при неверной сборке: смещение окна, невидимая дневная свеча,
// посчитанный по-своему диапазон дня. Каждое обязано быть названо в отчёте.
func TestDiffMarketDataNamesTheDivergingField(t *testing.T) {
	base := strategy.MarketData{
		Closes: []float64{1, 2}, DailyCloses: []float64{9}, TodayHigh: 2, TodayLow: 1,
	}
	cases := map[string]func(m *strategy.MarketData){
		"Closes":      func(m *strategy.MarketData) { m.Closes = []float64{1, 3} },
		"DailyCloses": func(m *strategy.MarketData) { m.DailyCloses = []float64{9, 10} },
		"TodayHigh":   func(m *strategy.MarketData) { m.TodayHigh = 5 },
		"TodayLow":    func(m *strategy.MarketData) { m.TodayLow = 0.5 },
	}
	for field, mutate := range cases {
		got := base
		mutate(&got)
		d := diffMarketData(base, got)
		if len(d) == 0 {
			t.Fatalf("%s: diffMarketData found no divergence", field)
		}
		found := false
		for _, line := range d {
			if contains(line, field) {
				found = true
			}
		}
		if !found {
			t.Fatalf("%s: diff = %v, want the field named", field, d)
		}
	}
}

func TestDiffSignalComparesTradeRelevantFields(t *testing.T) {
	a := model.Signal{Kind: model.SignalBuy, Reason: "", StopLoss: 1, TakeProfit: 2, ATR: 0.5}
	if d := diffSignal(a, a); len(d) != 0 {
		t.Fatalf("diffSignal on identical signals = %v, want none", d)
	}
	b := a
	b.StopLoss = 1.1
	if d := diffSignal(a, b); len(d) == 0 {
		t.Fatal("diffSignal missed a StopLoss divergence")
	}
	c := a
	c.Kind = model.SignalSell
	if d := diffSignal(a, c); len(d) == 0 {
		t.Fatal("diffSignal missed a Kind divergence")
	}
}
```

`contains` — локальный хелпер над `strings.Contains`.

- [ ] **Step 2: Прогнать тесты — убедиться, что падают**

Run: `go test ./cmd/pullparity/ -v`
Expected: FAIL — `undefined: diffMarketData`.

- [ ] **Step 3: Реализовать команду**

`cmd/pullparity/main.go`. Структура (за образец взять `cmd/pullscreen/main.go`: `godotenv`, `grpcclient`, `svc.NewCandleProvider`, флаги, вывод в stdout):

```go
// Command pullparity proves that the live rsi_pullback MarketData assembler produces the
// same snapshot — and therefore the same Decide verdict — as the backtest engine's own
// per-bar assembly. It replays cached candles bar by bar and builds each snapshot twice:
// once the way internal/domain/backtest.Run does it, once through the live
// marketdata.Assemble fed by a cache-backed CandleClient that only ever returns candles
// visible at that bar's close. Any divergence is a live/backtest fidelity bug.
package main
```

Флаги: `-tickers` (CSV, дефолт `UGLD,T,GAZP`), `-months` (дефолт 24), `-examples` (сколько расхождений печатать, дефолт 10), `-cache` (дефолт `data/candles`).

Ядро прогона по тикеру:

```go
st, ok := live.StrategyFor(ticker)          // параметры тикера, не дефолты ядра
lookback := st.Lookback()
m30, _ := provider.Load(ctx, ticker, uid, enum.Minutes30, from, to, false)
day, _ := provider.Load(ctx, ticker, uid, enum.Day1, from, to, false)

for i := lookback - 1; i < len(m30); i++ {
    cur := m30[i].Time
    // эталон — сборка движка
    want := backtest.AssembleMarketData(m30[i-lookback+1:i+1], day, nil, cur)
    want.TodayHigh, want.TodayLow = backtest.TodayExtent(m30, i)

    // живой путь: клиент отдаёт ровно то, что было видно на закрытие бара i
    asOf := cur.Add(30 * time.Minute) // момент, когда бар i только что закрылся
    fake := newCacheClient(m30, day, asOf)
    got, err := marketdata.Assemble(ctx, fake, uid, lookback, asOf)
    if err != nil {
        // Ошибка сборки — тоже расхождение: движок на этом баре снимок построил.
        diffs = append(diffs, fmt.Sprintf("%v: Assemble: %v", cur, err))
        continue
    }

    lines := diffMarketData(want, got)
    lines = append(lines, diffSignal(st.Decide(want), st.Decide(got))...)
    for _, l := range lines {
        diffs = append(diffs, fmt.Sprintf("%v: %s", cur, l))
    }
}
```

`newCacheClient` реализует `candles.CandleClient` поверх срезов: отдаёт бары интервала с `Time` в `[from, to]` и строго раньше `asOf`, помечая их `IsComplete: true`. Именно `asOf = cur + 30m` — момент, когда бар `i` только что закрылся и раннер принимает решение.

`diffMarketData` сравнивает `Price`, `Closes`, `Highs`, `Lows`, `Volumes`, `Times`, `DailyCloses`, `DailyHighs`, `DailyLows`, `DailyTimes`, `TodayHigh`, `TodayLow` — сперва длины, потом поэлементно, возвращая строки вида `Closes[402]: want 0.6120, got 0.6115`. `diffSignal` сравнивает `Kind`, `Reason`, `StopLoss`, `TakeProfit`, `ATR`.

Выход: по тикеру строка `UGLD: 12480 баров, 0 расхождений`, при ненулевом — первые `-examples` строк. Код возврата 1, если расхождения есть хоть где-то.

- [ ] **Step 4: Прогнать юнит-тесты**

Run: `go test ./cmd/pullparity/ -race -v`
Expected: PASS.

- [ ] **Step 5: Прогнать сверку на реальных данных**

Run: `go run ./cmd/pullparity -tickers UGLD,T,GAZP -months 24`
Expected: `0 расхождений` по всем трём тикерам, код возврата 0.

Если расхождения есть — это **дефект сборщика из Task 5**, а не повод ослабить сравнение. Типовые причины и что смотреть: смещение окна на бар (границы среза `i-lookback+1:i+1` против дедупликации в `fetch30m`); отличающийся набор `Daily*` (`dailyFetchDays` не покрыл прогрев ATR на длинной серии праздников); `TodayHigh/Low`, посчитанные по обрезанному окну, тогда как движок сканирует полную серию (окно ≥ 160 баров всегда покрывает один календарный день, так что расхождение здесь означает дыру в данных).

- [ ] **Step 6: Коммит**

```bash
git add cmd/pullparity
git commit -m "feat(rsi_pullback): cmd/pullparity — сверка live-сборки с бэктестом"
```

---

### Task 7: Единственный пасс — сервис, вход и выходы

Весь торговый путь одной задачей. Часть А: сервис, пасс и вход (сигнал → сайзинг от
процента счёта → market BUY → стейт → немедленный защитный стоп). Часть Б: три выхода и
ведение биржевой стоп-заявки. Части разделены только ради порядка работы и промежуточного
коммита — гейт ревью один, на цельном пассе.

**Files:**
- Create: `internal/service/trading_strategy/rsi_pullback/live/dto/run.go`
- Create: `internal/service/trading_strategy/rsi_pullback/live/live.go`
- Create: `internal/service/trading_strategy/rsi_pullback/live/pass.go`
- Create: `internal/service/trading_strategy/rsi_pullback/live/service_test.go`
- Modify: `.mockery.yaml`

**Interfaces:**
- Consumes: `livecore/{executor,stoporders,sizing,statestore,notifier,candles}` (Task 1–3), `config.RSIPullbackConfig` и `StrategyFor` (Task 4), `marketdata.Assemble` (Task 5).
- Produces:
  - `rsi_pullback/live/dto`: `type Run struct { Scheduler string }`.
  - `rsi_pullback/live`: `type Service interface { Run(ctx context.Context, in dto.Run) error }`; `func NewService(instruments instrumentsClient, market candles.CandleClient, ops operationsClient, orders executor.OrdersClient, stops stoporders.Client, tg telegram.Client, cfg *config.RSIPullbackConfig) *service`; неэкспортируемые интерфейсы `instrumentsClient` (`Shares(ctx) ([]*imodel.Share, error)`) и `operationsClient` (`GetPortfolio`, `GetPortfolioTotal`, `GetAvailableCash`, `GetInstrumentTrades` — те же сигнатуры, что в `reversion/live/live.go`).

- [ ] **Step 1: Написать падающие тесты входа**

`internal/service/trading_strategy/rsi_pullback/live/service_test.go`. За образец взять `reversion/live/service_test.go` (моки `livemocks.NewMockinstrumentsClient`, `livemocks.NewMockoperationsClient`, `candlemocks.NewMockCandleClient`, `execmocks.NewMockOrdersClient`, `stopmocks.NewMockClient`, `tgmocks.NewMockClient`; путь к стейту подменяется неэкспортируемым полем `svc.statePath`).

Общая обвязка — вселенная из одного тикера и генераторы свечей:

```go
package live

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/mock"

	"tinvest/internal/config"
	imodel "tinvest/internal/model"
	"tinvest/internal/enum"
	"tinvest/internal/service/trading_strategy/livecore/statestore"
	"tinvest/internal/service/trading_strategy/rsi_pullback/live/dto"
	"tinvest/pkg/logger"
)

func TestMain(m *testing.M) {
	logger.Init()
	os.Exit(m.Run())
}

var msk = func() *time.Location {
	l, _ := time.LoadLocation("Europe/Moscow")
	return l
}()

func q(v int64) imodel.Quotation { return imodel.Quotation{Units: v} }

func isM30(interval int32) bool  { return interval == enum.Minutes30.ToNumberInvestAPI() }
func isDay1(interval int32) bool { return interval == enum.Day1.ToNumberInvestAPI() }

// cfgFor — вселенная из одного тикера. Тесты входа гоняются на GAZP: его Lookback (160)
// самый маленький из зарегистрированных и UseVolume=0, поэтому серии короче и читаемее.
// Тесты трейла обязаны гоняться на UGLD — у GAZP UseTrail=0, и там трейл вообще
// не считается, то есть тест был бы зелёным при любой реализации.
func cfgFor(ticker string) *config.RSIPullbackConfig {
	return &config.RSIPullbackConfig{
		AccountID: "acc", Tickers: []string{ticker}, BuyPct: 5,
		TradeEnabled: false, NotifyEnabled: true,
	}
}

func shareFor(ticker string) []*imodel.Share {
	return []*imodel.Share{{ID: "uid-" + ticker, Ticker: ticker,
		Lot: 10, Trading: true, MinPriceIncrement: 0.01}}
}

// flat30m — ровная лента без сигнала: RSI никуда не пересекает, вход невозможен.
func flat30m(end time.Time, n int) []*imodel.CandleItemTechAnalyse {
	out := make([]*imodel.CandleItemTechAnalyse, 0, n)
	for i := n - 1; i >= 0; i-- {
		out = append(out, &imodel.CandleItemTechAnalyse{
			Time: end.Add(-time.Duration(i) * 30 * time.Minute),
			Open: q(100), High: q(100), Low: q(100), Close: q(100),
			Volume: 1000, IsComplete: true,
		})
	}
	return out
}

// pullback30m — лента, на последнем баре которой ядро GAZP открывает лонг: длинный
// подъём поднимает EMA(10) над EMA(70), затем резкий провал последних баров загоняет
// RSI(4) вниз через 25. Точные значения подбираются прогоном — «зелёный по неверной
// причине» тут самая дорогая ошибка, поэтому фикстура проверяется отдельно в Step 5.
func pullback30m(end time.Time, n int) []*imodel.CandleItemTechAnalyse

// dailies — дневные свечи с ненулевым истинным диапазоном: без них dailyATR() = 0
// и enter() выходит на четвёртом гейте, не дойдя до проверяемого.
func dailies(end time.Time, n int) []*imodel.CandleItemTechAnalyse
```

Тела `pullback30m` и `dailies` строятся по образцу `barSeries` из `rsi_pullback/strategy/core/core_test.go` — там уже есть серия, доводящая ядро до входа; её надо перенести и подогнать под параметры GAZP.

Полный образец теста — вход:

```go
// Вход: BUY-сигнал обязан привести к ордеру, записи стейта с замороженной целью и
// немедленной постановке защитного стопа. Позиция не должна прожить без биржевой
// защиты ни одного тика: следующая проверка только через полчаса, а стоп внутрибарный.
func TestBuySignalPlacesOrderStateAndProtectiveStop(t *testing.T) {
	dir := t.TempDir()
	now := time.Date(2026, 3, 10, 12, 1, 0, 0, msk)
	lastBar := time.Date(2026, 3, 10, 11, 30, 0, 0, msk)

	instruments := livemocks.NewMockinstrumentsClient(t)
	instruments.EXPECT().Shares(mock.Anything).Return(shareFor("GAZP"), nil)

	market := candlemocks.NewMockCandleClient(t)
	market.EXPECT().GetCandles(mock.Anything, mock.Anything, mock.MatchedBy(isM30),
		mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(pullback30m(lastBar, 400), nil)
	market.EXPECT().GetCandles(mock.Anything, mock.Anything, mock.MatchedBy(isDay1),
		mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(dailies(now, 60), nil)

	ops := livemocks.NewMockoperationsClient(t)
	ops.EXPECT().GetPortfolio(mock.Anything, mock.Anything).Return(nil, nil)
	ops.EXPECT().GetPortfolioTotal(mock.Anything, mock.Anything).Return(1_000_000.0, nil)
	ops.EXPECT().GetAvailableCash(mock.Anything, mock.Anything).Return(1_000_000.0, nil)

	stops := stopmocks.NewMockClient(t)
	tg := tgmocks.NewMockClient(t)
	tg.EXPECT().SendMessage(mock.Anything).Return(nil).Maybe()

	svc := NewService(instruments, market, ops, nil, stops, tg, cfgFor("GAZP"))
	svc.statePath = filepath.Join(dir, "state.json")

	if err := svc.Run(context.Background(), dto.Run{}); err != nil {
		t.Fatalf("Run: %v", err)
	}

	st, _ := statestore.New(svc.statePath).Load()
	e, ok := st["GAZP"]
	if !ok {
		t.Fatal("после BUY-сигнала в стейте нет записи по GAZP")
	}
	if e.EntryATR <= 0 {
		t.Fatalf("EntryATR = %v, want > 0 (дневной ATR замораживается на входе)", e.EntryATR)
	}
	if e.TakeProfit <= e.EntryPrice {
		t.Fatalf("TakeProfit = %v при входе %v — цель обязана быть заморожена выше входа",
			e.TakeProfit, e.EntryPrice)
	}
	if e.StopReason == "" {
		t.Fatal("защитный стоп не выставлен сразу после входа")
	}
	if e.StopPrice <= 0 || e.StopPrice >= e.EntryPrice {
		t.Fatalf("StopPrice = %v при входе %v, want 0 < stop < entry", e.StopPrice, e.EntryPrice)
	}
}
```

Остальные тесты строятся по этому же шаблону; ниже — что именно каждый обязан проверить.

```go
// Сайзинг идёт от BuyPct конфига, а не от захардкоженного числа: тот же прогон с
// BuyPct=5 против BuyPct=50 обязан дать разное Quantity в стейте (при цене 100 и лоте 10
// это 500 против 5000 штук при счёте 1 000 000).
func TestLotsAreSizedFromConfiguredBuyPct(t *testing.T)

// Уже открытая позиция у брокера не должна порождать второй вход: GetPortfolio отдаёт
// позицию по uid-gazp, PostOrder не должен быть вызван ни разу.
func TestNoSecondEntryWhenBrokerAlreadyHolds(t *testing.T)

// То же, но позиция известна только из стейта (ордер прошёл, портфель ещё не догнал).
func TestNoSecondEntryWhenStateAlreadyHasEntry(t *testing.T)

// Отклонённый ордер не оставляет записи в стейте: следующий тик повторит попытку.
// TradeEnabled=true, PostOrder возвращает ошибку, стейт обязан остаться пустым.
func TestRejectedBuyLeavesStateUntouched(t *testing.T)

// Тикер из конфига, которого нет в реестре, даёт алерт и пропускается, а не паникует:
// Tickers = ["НЕТ-ТАКОГО"], SendMessage вызван, GetCandles — нет.
func TestUnknownTickerAlertsAndSkips(t *testing.T)

// Протухшие данные: последний завершённый бар старше maxBarAge — решений не принимаем.
// Лента заканчивается за 3 часа до now, PostOrder не должен быть вызван.
func TestStaleBarSkipsTicker(t *testing.T)
```

- [ ] **Step 2: Прогнать тесты — убедиться, что падают**

Run: `go test ./internal/service/trading_strategy/rsi_pullback/live/ -v`
Expected: FAIL — `undefined: NewService`.

- [ ] **Step 3: Реализовать `dto` и каркас сервиса**

`dto/run.go`:

```go
package dto

// Run is one scheduled invocation of the rsi_pullback runner. Unlike reversion there is no
// Mode: entries and exits are both evaluated on every 30-minute bar, so the runner makes a
// single pass — exactly one Decide per bar per ticker, as in the backtest engine.
type Run struct {
	Scheduler string
}
```

`live.go` — по образцу `reversion/live/live.go`: `mu sync.Mutex`, поля `instruments`, `market`, `ops`, `exec`, `stops`, `tg`, `cfg`, `statePath`; `NewService` собирает `executor.New(orders, cfg.AccountID, cfg.TradeEnabled)` и `stoporders.New(stops, cfg.AccountID, cfg.TradeEnabled)`, `statePath = filepath.Join("data", "state", "rsi_pullback_"+cfg.AccountID+".json")`. Хелперы `notify`, `sharesByTicker`, `heldByShareID`, `nowMSK`, `stateStore` — копии reversion'овских. Метка алертов:

```go
// alertLabel — заголовок операционных уведомлений; пакет notifier общий, и по сообщению
// должно быть видно, какой раннер его прислал.
const alertLabel = "RSI Pullback"
```

`Run` держит мьютекс на весь пасс и зовёт `s.pass(ctx)`.

- [ ] **Step 4: Реализовать пасс и вход**

`pass.go`:

```go
// maxBarAge is how stale the latest completed 30-minute bar may be before the runner
// refuses to act on it. The cron fires every half hour, so anything older than an hour
// means the feed is behind (holiday, halt, API outage) — and a decision taken on a stale
// bar is a decision taken on the wrong price.
const maxBarAge = 60 * time.Minute

func (s *service) pass(ctx context.Context) error {
	shares, err := s.sharesByTicker(ctx)
	if err != nil {
		return err
	}
	held, err := s.heldByShareID(ctx)
	if err != nil {
		return err
	}
	store := s.stateStore()
	state, err := store.Load()
	if err != nil {
		return fmt.Errorf("rsi_pullback: load state: %w", err)
	}

	activeStops, listErr := s.stops.List(ctx) // один вызов на весь пасс
	if listErr != nil {
		s.notify(notifier.Alert(alertLabel, "", "GetStopOrders недоступен: "+listErr.Error()))
	}
	stopByInstrument := map[string]stoporders.ActiveStop{}
	stopByID := map[string]stoporders.ActiveStop{}
	for _, a := range activeStops {
		stopByInstrument[a.InstrumentUID] = a
		stopByID[a.StopOrderID] = a
	}

	now := nowMSK()
	for _, ticker := range s.cfg.Tickers {
		st, ok := StrategyFor(ticker)
		if !ok {
			s.notify(notifier.Alert(alertLabel, ticker, "тикер не зарегистрирован в rsi_pullback — пропуск"))
			continue
		}
		sh, ok := shares[ticker]
		if !ok || !sh.Trading {
			logger.ErrorContext(ctx, fmt.Sprintf("rsi_pullback: %s not tradable, skip", ticker))
			continue
		}
		md, err := marketdata.Assemble(ctx, s.market, sh.ID, st.Lookback(), now)
		if err != nil {
			logger.ErrorContext(ctx, fmt.Sprintf("rsi_pullback: %s marketdata: %v", ticker, err))
			continue
		}
		if n := len(md.Times); n == 0 || now.Sub(md.Times[n-1]) > maxBarAge {
			logger.ErrorContext(ctx, fmt.Sprintf("rsi_pullback: %s stale bar, skip", ticker))
			continue
		}

		pc := &passCtx{
			state: state, store: store, stopByID: stopByID,
			stopByInstrument: stopByInstrument, listErr: listErr, now: now,
		}
		pos, isHeld := held[sh.ID]
		if _, hasState := state[ticker]; isHeld || hasState {
			if err := s.manage(ctx, pc, ticker, sh, st, md, pos, isHeld); err != nil {
				return err
			}
			continue
		}
		if err := s.buy(ctx, pc, ticker, sh, st, md); err != nil {
			return err
		}
	}
	return nil
}
```

Общее для всех тикеров одного пасса состояние переносится структурой, а не десятком
параметров:

```go
// passCtx несёт состояние, общее для всех тикеров одного пасса: снапшот биржевых заявок
// берётся ОДИН раз на пасс, иначе повторный Cancel одной и той же заявки в одном тике
// дал бы ложный алерт.
type passCtx struct {
	state            map[string]statestore.Entry
	store            statestore.Store
	stopByID         map[string]stoporders.ActiveStop
	stopByInstrument map[string]stoporders.ActiveStop
	listErr          error
	now              time.Time
}
```

Заглушка `manage` — промежуточное состояние ВНУТРИ этой задачи: часть Б ниже заменяет её
настоящей реализацией, и задача не считается сделанной, пока это не произошло.

```go
func (s *service) manage(ctx context.Context, pc *passCtx, ticker string, sh *imodel.Share,
	st *core.Strategy, md strategy.MarketData, pos *grpcmodel.Position, isHeld bool) error {
	return nil
}
```

Сигнатура входа: `func (s *service) buy(ctx context.Context, pc *passCtx, ticker string, sh *imodel.Share, st *core.Strategy, md strategy.MarketData) error`.

`buy` повторяет `reversion/live/buy.go` с тремя отличиями: `md.Position = nil`; в стейт кладётся `TakeProfit: sig.TakeProfit`; защитный стоп ставится всегда (у `rsi_pullback` нет переключателя `UseIntrabarStop` — стоп внутрибарный по определению стратегии):

```go
		state[ticker] = statestore.Entry{
			Ticker:     ticker,
			EntryTime:  now,
			EntryPrice: fillPrice,
			EntryATR:   sig.ATR, // дневной ATR: им же меряются стоп, цель и обе границы гейта дня
			TakeProfit: sig.TakeProfit,
			MaxFav:     fillPrice,
			Quantity:   qty,
		}
```

- [ ] **Step 5: Добавить моки и прогнать тесты**

В `.mockery.yaml`:

```yaml
  tinvest/internal/service/trading_strategy/rsi_pullback/live:
    config:
      all: false
    interfaces:
      Service:
      instrumentsClient:
      operationsClient:
```

Run: `./bin/mage mocks && go test ./internal/service/trading_strategy/rsi_pullback/live/ -race -v`
Expected: PASS.

- [ ] **Step 6: Проверить, что фикстура входа действительно доходит до входа**

Финальное ревью прошлого этапа стратегии нашло ровно этот дефект: `TestEnterGates` был
пустым, потому что фикстура не давала дневных данных, `dailyATR()` возвращал 0, и `enter()`
выходил на четвёртом гейте — мутационно четыре гейта из шести можно было удалить, не уронив
пакет. Прямая проверка:

```go
// Фикстура обязана доходить до самого входа, а не останавливаться на раннем гейте.
func TestPullbackFixtureActuallyProducesABuySignal(t *testing.T) {
	st, _ := StrategyFor("GAZP")
	lastBar := time.Date(2026, 3, 10, 11, 30, 0, 0, msk)
	md := assembleFromFixture(t, pullback30m(lastBar, 400), dailies(lastBar, 60))
	if sig := st.Decide(md); sig.Kind != model.SignalBuy {
		t.Fatalf("фикстура не даёт BUY: Kind=%v\n%s", sig.Kind, st.Explain(md))
	}
}
```

`assembleFromFixture` — хелпер поверх `marketdata.Assemble` с тем же поддельным клиентом,
что в тестах выше. `st.Explain(md)` в сообщении об ошибке печатает вердикт по каждому
гейту — по нему видно, на каком именно фикстура остановилась.

Run: `go test ./internal/service/trading_strategy/rsi_pullback/live/ -run FixtureActually -v`
Expected: PASS. При падении — чинить фикстуру, а не ослаблять тесты входа.

- [ ] **Step 7: Коммит**

```bash
git add -A
git commit -m "feat(rsi_pullback): live-сервис и вход в позицию"
```

#### Часть Б: сопровождение позиции — выходы и синхронизация стоп-заявки

Три выхода и ведение биржевой заявки. Здесь же экспорт `DesiredStop` из ядра: уровень
обязан считаться той же функцией, что в бэктесте. **Задача НЕ завершена, пока часть Б не
заменила заглушку `manage` из части А** — иначе позиция остаётся без сопровождения.

**Files:**
- Modify: `internal/service/trading_strategy/rsi_pullback/strategy/core/core.go` (экспорт `DesiredStop`), `core_test.go`
- Modify: `internal/service/trading_strategy/rsi_pullback/live/pass.go` (реализация `manage`)
- Modify: `internal/service/trading_strategy/rsi_pullback/live/service_test.go`

**Interfaces:**
- Consumes: всё из Task 7.
- Produces: `core.DesiredStop(p Params, entry, dailyATR, maxFav float64) (float64, string)` — уровень защитного стопа и причина (`"SL"` | `"TRAIL"`), `(0, "")` когда стопа нет.

- [ ] **Step 8: Экспортировать `DesiredStop`**

В `rsi_pullback/strategy/core/core.go` переименовать `desiredStop` → `DesiredStop`, обновить два вызова (`manage`, и вызов внутри самого ядра) и вызовы в `core_test.go`. Doc-комментарий дополнить строкой:

```go
// Экспортирована ради живого раннера: уровень биржевой стоп-заявки обязан считаться этой
// же функцией, иначе прод и бэктест разъедутся по самому частому механизму выхода.
```

- [ ] **Step 9: Написать падающие тесты выходов**

Дописать в `service_test.go`. Понадобится ещё одна фикстура:

```go
// risingWithDeepLow30m — лента, где последний бар закрывается ВЫШЕ прежнего максимума
// (закрытие 112 при MaxFav 100), но его low проваливается между двумя кандидатами на
// уровень трейла: ниже уровня, посчитанного от нового закрытия, но выше уровня,
// посчитанного от прежнего максимума. Именно этот зазор разводит честное решение и
// завышенное; без него тест зелёный при обеих реализациях. n обязано быть >= 403:
// столько баров требует Lookback UGLD, и на более коротком окне Assemble вернёт ошибку.
func risingWithDeepLow30m(end time.Time, n int) []*imodel.CandleItemTechAnalyse
```

Конкретные уровни считаются от параметров UGLD (`TrailDailyATR = 0.5`,
`StopDailyATR = 0.5`) и `EntryATR = 10`: трейл от PrevMaxFav=100 даёт 95, от MaxFav=112 —
107, значит low последнего бара должен лежать между ними, например 100. Подобрать
прогоном и закрепить мутационной проверкой из Step 6.

Остальные тесты:

```go
// RSI-выход: перед рыночной продажей биржевой стоп обязан быть снят, иначе он позже
// продаст новую позицию по этому тикеру.
func TestRSIExitCancelsStopBeforeSelling(t *testing.T)

// TP по close-модели: бар, чей high достал цели из стейта, закрывает позицию по рынку.
func TestTakeProfitExitFiresOnBarHigh(t *testing.T)

// Отклонённая продажа обязана вернуть биржевую защиту: стоп уже снят, позиция голая.
func TestRejectedSellRestoresTheStop(t *testing.T)

// Трейл: уровень НОВОЙ заявки считается от обновлённого maxFav (она будет работать на
// следующем баре), а решение о срабатывании — от PrevMaxFavorablePrice (заявка,
// работавшая на ЭТОМ баре, выставлена после закрытия предыдущего и не могла знать его
// закрытия, а проверяется против low этого же бара). MaxFav >= PrevMaxFav всегда, поэтому
// чтение первого вместо второго завышает результат систематически и в обе стороны:
// уровень срабатывает не реже и заливается не хуже.
func TestTrailDecisionUsesPrevMaxFavorable(t *testing.T) {
	dir := t.TempDir()
	statePath := filepath.Join(dir, "state.json")
	now := time.Date(2026, 3, 10, 12, 1, 0, 0, msk)
	lastBar := time.Date(2026, 3, 10, 11, 30, 0, 0, msk)

	// UGLD, а не GAZP: у GAZP UseTrail=0, трейл не считается вовсе и тест был бы зелёным
	// при любой реализации. У UGLD UseTrail=1, TrailDailyATR=0.5 — как только максимум
	// закрытий уходит выше входа, связывает именно трейл, а не фиксированный SL.
	// Позиция открыта по 100, максимум закрытий пока 100, дневной ATR 10. Уровень от
	// PrevMaxFav=100 лежит НИЖЕ low последнего бара, а от MaxFav=112 (закрытие этого
	// бара) — уже ВЫШЕ. Стоп обязан не сработать.
	_ = statestore.New(statePath).Save(map[string]statestore.Entry{
		"UGLD": {Ticker: "UGLD", EntryPrice: 100, EntryATR: 10, MaxFav: 100,
			TakeProfit: 200, Quantity: 100},
	})

	instruments := livemocks.NewMockinstrumentsClient(t)
	instruments.EXPECT().Shares(mock.Anything).Return(shareFor("UGLD"), nil)

	market := candlemocks.NewMockCandleClient(t)
	market.EXPECT().GetCandles(mock.Anything, mock.Anything, mock.MatchedBy(isM30),
		mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(risingWithDeepLow30m(lastBar, 500), nil)
	market.EXPECT().GetCandles(mock.Anything, mock.Anything, mock.MatchedBy(isDay1),
		mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(dailies(now, 60), nil)

	ops := livemocks.NewMockoperationsClient(t)
	ops.EXPECT().GetPortfolio(mock.Anything, mock.Anything).Return(
		[]*grpcmodel.Position{{ShareID: "uid-UGLD", InstrumentType: "share", Quantity: 100,
			PurchasePrice: grpcmodel.Quotation{Units: 100}}}, nil)

	stops := stopmocks.NewMockClient(t)
	orders := execmocks.NewMockOrdersClient(t)
	tg := tgmocks.NewMockClient(t)
	tg.EXPECT().SendMessage(mock.Anything).Return(nil).Maybe()

	c := cfgFor("UGLD")
	c.TradeEnabled = true
	svc := NewService(instruments, market, ops, orders, stops, tg, c)
	svc.statePath = statePath

	if err := svc.Run(context.Background(), dto.Run{}); err != nil {
		t.Fatalf("Run: %v", err)
	}

	// Продажи быть не должно: от честного (Prev) уровня стоп не задет.
	orders.AssertNotCalled(t, "PostOrder", mock.Anything, mock.Anything)
	st, _ := statestore.New(statePath).Load()
	if _, ok := st["UGLD"]; !ok {
		t.Fatal("позиция закрыта: уровень посчитан от MaxFav вместо PrevMaxFavorablePrice")
	}
	// А вот НОВАЯ заявка обязана подтянуться к обновлённому максимуму.
	if st["UGLD"].MaxFav <= 100 {
		t.Fatalf("MaxFav = %v, want > 100 (максимум закрытий обязан подняться)", st["UGLD"].MaxFav)
	}
}

// Заявка исчезла из ACTIVE и найдена в EXECUTED — это сработавший стоп: уведомление
// о выходе и чистка стейта, без репоста.
func TestFiredStopClosesTheTradeInState(t *testing.T)

// Заявка исчезла, а EXECUTED недоступен — репост вслепую запрещён: он продал бы уже
// проданную позицию при касании уровня.
func TestVanishedStopWithUnavailableExecutedDefersRepost(t *testing.T)

// Позиция усохла (частичный стоп или ручная продажа) — размер в стейте
// реконсилируется, иначе следующая заявка окажется переразмеренной.
func TestShrunkPositionReconcilesQuantity(t *testing.T)
```

- [ ] **Step 10: Прогнать тесты — убедиться, что падают**

Run: `go test ./internal/service/trading_strategy/rsi_pullback/live/ -race -v`
Expected: FAIL — `manage` пустой, ни одного вызова моков.

- [ ] **Step 11: Реализовать `manage`**

Перенести структуру `reversion/live/manage.go` целиком, с четырьмя отличиями:

1. **Ветки `UseIntrabarStop != 1` нет.** У `rsi_pullback` защитный стоп внутрибарный всегда — блок «close-модель: снять оставшуюся заявку» удаляется.
2. **`md.Position` получает `TakeProfit` из стейта** — без него close-модель TP не с чем сравнивать:

```go
		md.Position = &strategy.Position{
			PurchasePrice:         entry.EntryPrice,
			Quantity:              pos.Quantity,
			EntryATR:              entry.EntryATR,
			TakeProfit:            entry.TakeProfit,
			MaxFavorablePrice:     entry.MaxFav,
			PrevMaxFavorablePrice: prevMaxFav,
		}
```

3. **Уровень считает `core.DesiredStop`** из `rsi_pullback/strategy/core`, а не из reversion'ового.
4. **Все `notifier.Alert(...)` получают первым аргументом `alertLabel`.**

Порядок работы с maxFav сохраняется дословно: `prevMaxFav := entry.MaxFav` до подъёма, подъём по `md.Price`, `Decide`, затем уровень новой заявки от **обновлённого** `entry.MaxFav`.

- [ ] **Step 12: Прогнать тесты**

Run: `go test ./internal/service/trading_strategy/rsi_pullback/... -race -v`
Expected: PASS.

- [ ] **Step 13: Мутационная проверка ключевого теста**

Временно заменить в `manage` `PrevMaxFavorablePrice: prevMaxFav` на `PrevMaxFavorablePrice: entry.MaxFav` и прогнать `TestTrailDecisionUsesPrevMaxFavorable`.

Run: `go test ./internal/service/trading_strategy/rsi_pullback/live/ -run TrailDecisionUsesPrev -v`
Expected: FAIL. Если тест проходит — он не проверяет то, ради чего написан (скорее всего `risingWithDeepLow30m` не разводит два уровня по разные стороны от `low` последнего бара); починить фикстуру, затем вернуть код.

- [ ] **Step 14: Полный гейт и коммит**

Run: `./bin/mage ci`
Expected: EXIT=0.

```bash
git add -A
git commit -m "feat(rsi_pullback): выходы SL/TRAIL/TP/RSI и ведение биржевой стоп-заявки"
```

---

### Task 8: Восстановление стейта по API

Позиция есть у брокера, локального стейта нет (переезд контейнера, потерянный том). Без восстановления раннер не знает ни `EntryATR`, ни цели — то есть не может ни защитить позицию, ни закрыть её.

**Files:**
- Create: `internal/service/trading_strategy/rsi_pullback/live/reconstruct/reconstruct.go`
- Create: `internal/service/trading_strategy/rsi_pullback/live/reconstruct/reconstruct_test.go`
- Modify: `internal/service/trading_strategy/rsi_pullback/live/pass.go` (вызов из `manage`)
- Modify: `.mockery.yaml`

**Interfaces:**
- Consumes: `livecore/candles.CandleClient`, `livecore/statestore.Entry`, `core.Params`.
- Produces: `reconstruct`: `type TradesClient interface { GetInstrumentTrades(ctx context.Context, accountID, instrumentID string, from, to time.Time) ([]grpcmodel.Trade, error) }`; `func Entry(ctx context.Context, tc TradesClient, cc candles.CandleClient, accountID, instrumentID, ticker string, purchasePrice float64, p core.Params, now time.Time) (statestore.Entry, error)`.

- [ ] **Step 1: Написать падающие тесты**

`reconstruct_test.go`:

Обвязка: мок `TradesClient` отдаёт одну BUY-сделку, `fakeCandles` — дневные и 30m серии
по интервалу запроса (тот же приём, что в тестах `marketdata` из Task 5).

```go
// fakeCandles отдаёт серию по запрошенному интервалу, обрезая её окном запроса.
type fakeCandles struct{ m30, day []*imodel.CandleItemTechAnalyse }

func (f *fakeCandles) GetCandles(_ context.Context, _ *string, interval int32,
	from, to *timestamppb.Timestamp, _ *int32, _ bool) ([]*imodel.CandleItemTechAnalyse, error) {
	src := f.m30
	if interval == enum.Day1.ToNumberInvestAPI() {
		src = f.day
	}
	var out []*imodel.CandleItemTechAnalyse
	for _, c := range src {
		if !c.Time.Before(from.AsTime()) && !c.Time.After(to.AsTime()) {
			out = append(out, c)
		}
	}
	return out, nil
}

// dailiesWithRange строит n дневных баров, у каждого high-low ровно rng и close посередине,
// последний — в MSK-полночь дня end. Истинный диапазон постоянен, поэтому ATR любой
// длины равен rng, и тест не зависит от подобранного DailyATRPeriod.
func dailiesWithRange(end time.Time, n int, rng int64) []*imodel.CandleItemTechAnalyse

// flat30m строит n ровных 30-минутных баров с закрытием price, последний — в end.
func flat30m(end time.Time, n int, price int64) []*imodel.CandleItemTechAnalyse
```

```go
// EntryATR обязан быть ДНЕВНЫМ ATR по будним дневным свечам, закрывшимся до
// MSK-полуночи дня входа — той же величиной, которой стратегия мерила стоп на входе.
// Часовой или 30-минутный ATR дал бы уровень в разы теснее, и восстановленная позиция
// поехала бы с защитой, которой стратегия никогда не задавала.
func TestEntryRebuildsDailyATRAsOfEntryDay(t *testing.T) {
	entryTime := time.Date(2026, 3, 10, 11, 30, 0, 0, msk)
	now := entryTime.Add(4 * time.Hour)

	tc := recmocks.NewMockTradesClient(t)
	tc.EXPECT().GetInstrumentTrades(mock.Anything, "acc", "uid-GAZP", mock.Anything, mock.Anything).
		Return([]grpcmodel.Trade{{IsBuy: true, Date: entryTime}}, nil)

	// Дневные свечи с истинным диапазоном ровно 10 у каждой: ATR любой длины даёт 10.
	cc := &fakeCandles{
		day:  dailiesWithRange(entryTime, 60, 10),
		m30:  flat30m(now, 20, 100),
	}

	p := gazp.DefaultParams()
	got, err := Entry(context.Background(), tc, cc, "acc", "uid-GAZP", "GAZP", 100, p, now)
	if err != nil {
		t.Fatalf("Entry: %v", err)
	}
	if got.EntryATR < 9.5 || got.EntryATR > 10.5 {
		t.Fatalf("EntryATR = %v, want ~10 (дневной ATR, не внутридневной)", got.EntryATR)
	}
	if !got.EntryTime.Equal(entryTime) {
		t.Fatalf("EntryTime = %v, want %v (последняя BUY-сделка)", got.EntryTime, entryTime)
	}
	if got.EntryPrice != 100 {
		t.Fatalf("EntryPrice = %v, want 100 (средняя цена покупки от брокера)", got.EntryPrice)
	}
}

// Выходные дневные свечи MOEX в 2.9-3.8 раза уже будних: оставленные в расчёте, они
// занижают ATR на 9-16% и вместе с ним весь защитный уровень.
func TestEntrySkipsWeekendDailies(t *testing.T)

// Цель восстанавливается из параметров тикера, иначе TP по close-модели не сработает
// никогда и сделку дотянет только стоп.
func TestEntryRebuildsTakeProfitFromParams(t *testing.T)

// MaxFav — максимум ЗАКРЫТИЙ 30m с момента входа: движок марк-ту-маркетит позицию по
// close, и трейл обязан отсчитываться от той же величины.
func TestEntryRebuildsMaxFavFromClosesAfterEntry(t *testing.T)

// Без BUY-сделки восстанавливать нечего — ошибка, а не молчаливый нулевой стейт.
func TestEntryFailsWithoutABuyFill(t *testing.T)
```

- [ ] **Step 2: Прогнать тесты — убедиться, что падают**

Run: `go test ./internal/service/trading_strategy/rsi_pullback/live/reconstruct/ -v`
Expected: FAIL — пакета нет.

- [ ] **Step 3: Реализовать пакет**

За образец — `reversion/live/reconstruct/reconstruct.go`, с заменой расчёта ATR:

```go
// Package reconstruct rebuilds rsi_pullback entry-state from the broker API when the local
// state file is missing but a position is open. EntryPrice uses the broker's average
// purchase price; EntryTime is the most recent BUY fill; EntryATR is the DAILY ATR over
// weekday dailies completed before the entry day's MSK midnight — the same unit the core
// used to size the stop and the target at entry; TakeProfit is rebuilt from it and the
// ticker's TPDailyATR; MaxFav is the highest 30-minute CLOSE since entry.
package reconstruct
```

Два запроса: `enum.Day1` за `entryTime - 150 дней … entryTime` (150 календарных дней с запасом покрывают `DailyATRPeriod+1` будних дней через каникулы) и `enum.Minutes30` за `entryTime … now` для `MaxFav`. Фильтрация выходных — та же логика, что в `core.weekdayDaily`: отбрасывать бары, чей MSK-день — суббота или воскресенье. ATR — `indicators.ATR(highs, lows, closes, p.DailyATRPeriod)`. `TakeProfit = purchasePrice + p.TPDailyATR*atr` при `p.TPDailyATR > 0`, иначе 0.

- [ ] **Step 4: Подключить в `manage`**

В ветке «позиция есть, стейта нет» (по образцу `reversion/live/manage.go:104-118`):

```go
			rebuilt, err := reconstruct.Entry(ctx, s.ops, s.market, s.cfg.AccountID, sh.ID, ticker,
				utils.CombinePrice(pos.PurchasePrice.Units, pos.PurchasePrice.Nano),
				mustParams(ticker), now)
			if err != nil {
				s.notify(notifier.Alert(alertLabel, ticker, "позиция без локального стейта, реконструкция не удалась: "+err.Error()))
				logger.ErrorContext(ctx, fmt.Sprintf("rsi_pullback: reconstruct %s: %v", ticker, err))
				continue
			}
			rebuilt.Quantity = pos.Quantity
```

`mustParams(ticker)` — хелпер в `pass.go`: `p, _ := ParamsFor(ticker); return p` (тикер уже прошёл `StrategyFor` выше).

- [ ] **Step 5: Добавить мок, прогнать тесты**

В `.mockery.yaml`:

```yaml
  tinvest/internal/service/trading_strategy/rsi_pullback/live/reconstruct:
    interfaces:
      TradesClient:
```

Run: `./bin/mage mocks && go test ./internal/service/trading_strategy/rsi_pullback/... -race`
Expected: PASS.

- [ ] **Step 6: Коммит**

```bash
git add -A
git commit -m "feat(rsi_pullback): восстановление стейта по API (дневной ATR)"
```

---

### Task 9: Планировщик, проводка в приложение и документация

Последняя миля: cron-обёртка, отдельный gRPC-клиент под токен второго счёта, тема Telegram, горутины в `runDev`/`runProd` и справочник раннера.

**Files:**
- Create: `internal/service/trading_strategy/rsi_pullback/live/scheduler/scheduler.go`, `scheduler_test.go`
- Create: `docs/rsi_pullback/live.md`
- Modify: `internal/service_provider/client.go`, `internal/service_provider/service.go`, `internal/app/app.go`, `CLAUDE.md`, `docs/rsi_pullback/strategy.md`, `env/local.env.example`

**Interfaces:**
- Consumes: `live.Service`, `dto.Run`, `config.RSIPullbackConfig`.
- Produces: `scheduler.NewSchedulerService(service live.Service) live.Service`; `(*ServiceProvider).GetRSIPullbackLiveService() live.Service`; `(*ServiceProvider).GetRSIPullbackGrpcClient() (internalgrpc.GrpcClient, error)`; `(*ServiceProvider).GetRSIPullbackSender() (telegram.Client, error)`.

- [ ] **Step 1: Написать падающий тест планировщика**

`scheduler_test.go` — по образцу `reversion/live/scheduler/scheduler_test.go`: `NewSchedulerService` оборачивает `live.Service` и при отменённом контексте завершается без ошибки, зарегистрировав джоб с переданным cron-выражением.

- [ ] **Step 2: Прогнать — убедиться, что падает**

Run: `go test ./internal/service/trading_strategy/rsi_pullback/live/scheduler/ -v`
Expected: FAIL — пакета нет.

- [ ] **Step 3: Реализовать планировщик**

Копия `reversion/live/scheduler/scheduler.go` с заменой импортов на `rsi_pullback/live` и `rsi_pullback/live/dto` и лог-текстов на «Воркер RSI Pullback начал работу» / «Ошибка в ходе работы job RSI Pullback» / «Worker RSI Pullback is running».

- [ ] **Step 4: Проводка в `service_provider`**

В `client.go` — поле `rsiPullbackGrpcClient internalgrpc.GrpcClient` в `client`, метод по образцу `GetReversionGrpcClient` на `s.appConfig.RSIPullback.Token`, и:

```go
func (s *ServiceProvider) GetRSIPullbackSender() (telegram.Client, error) {
	return s.topicSender(s.appConfig.TelegramClient.TopicRSIPullback, "rsi_pullback")
}
```

В `service.go` — поле `rsiPullbackLiveService rsipullbacklive.Service` в `service` и:

```go
func (*ServiceProvider) GetRSIPullbackLiveService() rsipullbacklive.Service {
	if serviceProvider.service.rsiPullbackLiveService == nil {
		grpcClient, _ := serviceProvider.GetRSIPullbackGrpcClient()
		tgClient, _ := serviceProvider.GetRSIPullbackSender()
		serviceProvider.service.rsiPullbackLiveService = rsipullbacklive.NewService(
			grpcClient.InstrumentsServiceClient(),
			grpcClient.MarketDataServiceClient(),
			grpcClient.OperationsServiceClient(),
			grpcClient.OrdersServiceClient(),
			grpcClient.StopOrdersServiceClient(),
			tgClient,
			serviceProvider.appConfig.RSIPullback,
		)
	}

	return serviceProvider.service.rsiPullbackLiveService
}
```

Импорт `rsipullbacklive "tinvest/internal/service/trading_strategy/rsi_pullback/live"` — алиас обязателен, `reversion/live` уже занял имя `live`.

- [ ] **Step 5: Проводка в `app.go`**

В `runProd` увеличить `wg.Add(6)` → `wg.Add(7)` и добавить горутину; в `runDev` — `wg.Add(2)` → `wg.Add(3)` и такую же:

```go
	go func() {
		defer wg.Done()
		err := rsipullbackscheduler.NewSchedulerService(a.sp.GetRSIPullbackLiveService()).Run(
			ctx,
			rsipullbackdto.Run{Scheduler: a.config.RSIPullback.Schedule},
		)
		if err != nil {
			logger.ErrorContext(ctx, "Error in worker RSI Pullback", err.Error())
		}
	}()
```

Проверить фактические значения `wg.Add` перед правкой — они могли измениться.

- [ ] **Step 6: Собрать и прогнать полный гейт**

Run: `go build ./internal/... ./pkg/... ./cmd/... && ./bin/mage ci`
Expected: EXIT=0.

- [ ] **Step 7: Написать `docs/rsi_pullback/live.md`**

Разделы: назначение и отличия от бэктеста; расписание и почему `1,31 6-23 * * *` (все дни недели); полный список env-переменных с дефолтами; формат файла стейта `data/state/rsi_pullback_<accountID>.json` с описанием каждого поля; таблица трёх выходов и их механизмов (биржевая заявка для SL/TRAIL, close-модель для TP и RSI) с долями на 36 in-sample сделках UGLD; порядок восстановления стейта; инструкция по `cmd/pullparity`; порядок выката (dry-run → сверка → торговля); принятые риски из §11 спеки.

В `CLAUDE.md` в описании `rsi_pullback` дописать: «live-раннер — `rsi_pullback/live`, отдельный счёт, docs/rsi_pullback/live.md». В `docs/rsi_pullback/strategy.md` — ссылку на `live.md`. В `env/local.env.example` — семь новых переменных с комментариями и безопасными значениями (`RSI_PULLBACK_TRADE_ENABLED=false`).

- [ ] **Step 8: Коммит**

```bash
git add -A
git commit -m "feat(rsi_pullback): планировщик, проводка в приложение, справочник раннера"
```

---

## Приёмка

После Task 9:

1. `./bin/mage ci` — EXIT=0.
2. `go run ./cmd/pullparity -tickers UGLD,T,GAZP -months 24` — ноль расхождений по всем трём тикерам.
3. `go build ./internal/... ./pkg/... ./cmd/...` — без ошибок.
4. Запуск с `RSI_PULLBACK_TRADE_ENABLED=false`, `RSI_PULLBACK_NOTIFY_ENABLED=true` на реальном счёте: в теме Telegram появляются сигналы, ордера не выставляются, файл стейта пишется.

Пункт 4 выполняет владелец; включение торговли — отдельное решение после периода наблюдения.
