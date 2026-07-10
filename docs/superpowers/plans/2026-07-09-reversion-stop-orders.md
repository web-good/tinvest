# Reversion Stop Orders Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Биржевые стоп-заявки (Tinkoff StopOrdersService) для SL/TRAIL/ATRSL в live-раннере reversion + интрабарный триггер этих же стопов в бэктесте + информационный перепрогон walk-forward трёх тикеров.

**Architecture:** Одна stop-market заявка на позицию на уровне `max(SL, ATRSL, TRAIL)`, раннер отменяет/перевыставляет её на часовом тике при подъёме трейлинга; в бэктесте те же три стопа сливаются в одну интрабарную проверку `low ≤ уровень` с исполнением по `min(уровень, open)` и приоритетом выше OB. Общая формула уровня — экспортируемая `core.DesiredStop`, чтобы live и бэктест не разъезжались.

**Tech Stack:** Go 1.25, gRPC (protoc-стабы из `api/v1/stoporders.proto`), mockery v2, mage.

**Spec:** `docs/superpowers/specs/2026-07-09-reversion-stop-orders-design.md` — прочитать перед началом.

## Global Constraints

- Ветка: `feat/reversion-stop-orders` от `main` (создать в Task 1, шаг 1).
- Гейт качества: `./bin/mage ci` (lint + `go test -race ./...` + mock-drift). `go build ./...` падает на `magefiles` — использовать `go build ./internal/... ./pkg/... ./cmd/...`.
- После изменения мокаемого интерфейса: `./bin/mage mocks` (инструменты ставятся `./bin/mage tools`, если `./bin/mockery` отсутствует).
- Коммиты заканчивать `Co-Authored-By: Claude Fable 5 <noreply@anthropic.com>`.
- Тип заявки: `STOP_ORDER_TYPE_STOP_LOSS` + `EXCHANGE_ORDER_TYPE_MARKET` + `STOP_ORDER_EXPIRATION_TYPE_GOOD_TILL_CANCEL`, `order_id` = UUID (идемпотентность).
- BE (безубыток) НЕ переезжает ни на биржу, ни на интрабар — остаётся close-проверкой.
- Никаких новых env-переменных: используются существующие `REVERSION_*` (TradeEnabled = dry-run переключатель).

---

### Task 1: Стабы StopOrdersService + gRPC-обёртка с auth

**Files:**
- Modify: `Makefile` (после таргета `generate-orders-api`, ~строка 26)
- Create: `internal/pb/v1/stoporders.pb.go`, `internal/pb/v1/stoporders_grpc.pb.go` (генерируются)
- Create: `pkg/client/grpc/stop_orders_service_client.go`
- Modify: `pkg/client/grpc/grpc.go` (интерфейс `GrpcClient`, структура `Client`, `NewClientGrpc`)
- Test: `pkg/client/grpc/stop_orders_auth_test.go`

**Interfaces:**
- Consumes: `api/v1/stoporders.proto` (уже в репо), `NewAuth`/`NewRPCCredential` из `pkg/client/grpc/auth.go` (паттерн — `orders_service_client.go`).
- Produces:
  ```go
  type StopOrdersServiceClient interface {
      PostStopOrder(ctx context.Context, in *investapi.PostStopOrderRequest, opts ...grpc.CallOption) (*investapi.PostStopOrderResponse, error)
      GetStopOrders(ctx context.Context, in *investapi.GetStopOrdersRequest, opts ...grpc.CallOption) (*investapi.GetStopOrdersResponse, error)
      CancelStopOrder(ctx context.Context, in *investapi.CancelStopOrderRequest, opts ...grpc.CallOption) (*investapi.CancelStopOrderResponse, error)
  }
  func NewStopOrdersServiceClient(conn grpc.ClientConnInterface, token string) StopOrdersServiceClient
  // + метод GrpcClient.StopOrdersServiceClient() StopOrdersServiceClient
  ```

- [x] **Step 1: Создать ветку**

```bash
git checkout -b feat/reversion-stop-orders main
```

- [x] **Step 2: Makefile-таргет по шаблону generate-orders-api**

```makefile
generate-stoporders-api:
	mkdir -p internal/pb/v1
	protoc --proto_path api/v1 \
	--go_out=./internal/pb/v1 --experimental_allow_proto3_optional --go_opt=paths=source_relative \
	--plugin=protoc-gen-go=bin/protoc-gen-go \
	--go-grpc_out=internal/pb/v1 --go-grpc_opt=paths=source_relative \
	--plugin=protoc-gen-go-grpc=bin/protoc-gen-go-grpc \
	api/v1/stoporders.proto
```

Также добавить `make generate-stoporders-api` в агрегирующий таргет `generate` (рядом с остальными `generate-*-api`, см. начало Makefile).

- [x] **Step 3: Сгенерировать и проверить**

Перед генерацией убедиться, что в `api/v1/stoporders.proto` есть `option go_package` в том же стиле, что в `orders.proto` (`grep go_package api/v1/orders.proto api/v1/stoporders.proto`); если в stoporders.proto его нет — добавить идентичный orders.proto.

```bash
make generate-stoporders-api
go build ./internal/pb/...
```

Expected: появились `internal/pb/v1/stoporders.pb.go` и `stoporders_grpc.pb.go` (пакет `investapi`), сборка зелёная. Если `bin/protoc-gen-go` отсутствует — `make install-deps`.

- [x] **Step 4: Failing test — обёртка хранит токен и подмешивает auth**

Скопировать паттерн из `pkg/client/grpc/orders_auth_test.go` (fake, который проверяет `opts`):

```go
package grpc

import (
	"context"
	"testing"

	"google.golang.org/grpc"

	investapi "tinvest/internal/pb/v1"
)

type fakeStopOrdersAPI struct {
	investapi.StopOrdersServiceClient
	gotOpts []grpc.CallOption
}

func (f *fakeStopOrdersAPI) PostStopOrder(ctx context.Context, in *investapi.PostStopOrderRequest, opts ...grpc.CallOption) (*investapi.PostStopOrderResponse, error) {
	f.gotOpts = opts
	return &investapi.PostStopOrderResponse{}, nil
}

func TestNewStopOrdersServiceClient_StoresToken(t *testing.T) {
	c := NewStopOrdersServiceClient(nil, "tok-xyz").(*stopOrdersServiceClient)
	if c.auth.Token != "tok-xyz" {
		t.Fatalf("token = %q, want tok-xyz", c.auth.Token)
	}
}

func TestStopOrdersServiceClient_PostAttachesAuth(t *testing.T) {
	fake := &fakeStopOrdersAPI{}
	c := &stopOrdersServiceClient{api: fake, auth: NewAuth("tok")}
	if _, err := c.PostStopOrder(context.Background(), &investapi.PostStopOrderRequest{}); err != nil {
		t.Fatal(err)
	}
	if len(fake.gotOpts) == 0 {
		t.Fatal("auth call option not attached")
	}
}
```

Примечание: поле `auth.Token` — сверить фактическое имя поля в `pkg/client/grpc/auth.go` и с тестом `orders_auth_test.go`; повторить их структуру дословно.

- [x] **Step 5: Запустить — убедиться, что падает**

Run: `go test ./pkg/client/grpc/ -run StopOrders -v`
Expected: FAIL (undefined: `NewStopOrdersServiceClient`).

- [x] **Step 6: Реализация обёртки**

`pkg/client/grpc/stop_orders_service_client.go` — зеркало `orders_service_client.go` (таймаут 10s, `NewRPCCredential`):

```go
package grpc

import (
	"context"
	"time"

	"google.golang.org/grpc"

	investapi "tinvest/internal/pb/v1"
)

type StopOrdersServiceClient interface {
	PostStopOrder(ctx context.Context, in *investapi.PostStopOrderRequest, opts ...grpc.CallOption) (*investapi.PostStopOrderResponse, error)
	GetStopOrders(ctx context.Context, in *investapi.GetStopOrdersRequest, opts ...grpc.CallOption) (*investapi.GetStopOrdersResponse, error)
	CancelStopOrder(ctx context.Context, in *investapi.CancelStopOrderRequest, opts ...grpc.CallOption) (*investapi.CancelStopOrderResponse, error)
}

type stopOrdersServiceClient struct {
	api  investapi.StopOrdersServiceClient
	auth *Auth
}

func NewStopOrdersServiceClient(conn grpc.ClientConnInterface, token string) StopOrdersServiceClient {
	return &stopOrdersServiceClient{api: investapi.NewStopOrdersServiceClient(conn), auth: NewAuth(token)}
}

func (c *stopOrdersServiceClient) PostStopOrder(ctx context.Context, in *investapi.PostStopOrderRequest, opts ...grpc.CallOption) (*investapi.PostStopOrderResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	opts = append(opts, NewRPCCredential(c.auth))
	return c.api.PostStopOrder(ctx, in, opts...)
}

func (c *stopOrdersServiceClient) GetStopOrders(ctx context.Context, in *investapi.GetStopOrdersRequest, opts ...grpc.CallOption) (*investapi.GetStopOrdersResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	opts = append(opts, NewRPCCredential(c.auth))
	return c.api.GetStopOrders(ctx, in, opts...)
}

func (c *stopOrdersServiceClient) CancelStopOrder(ctx context.Context, in *investapi.CancelStopOrderRequest, opts ...grpc.CallOption) (*investapi.CancelStopOrderResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	opts = append(opts, NewRPCCredential(c.auth))
	return c.api.CancelStopOrder(ctx, in, opts...)
}
```

В `pkg/client/grpc/grpc.go`: добавить `StopOrdersServiceClient() StopOrdersServiceClient` в интерфейс `GrpcClient`, поле `stopOrdersServiceClient StopOrdersServiceClient` в `Client`, геттер и строку `stopOrdersServiceClient: NewStopOrdersServiceClient(conn, token),` в `NewClientGrpc`.

- [x] **Step 7: Тесты зелёные**

Run: `go test ./pkg/client/grpc/ -v`
Expected: PASS (включая старые).

- [x] **Step 8: Commit**

```bash
git add Makefile internal/pb/v1/stoporders*.go pkg/client/grpc/
git commit -m "feat(grpc): generate StopOrdersService stubs and add authed client wrapper"
```

---

### Task 2: `PrevMaxFavorablePrice` + исполнение "ATRSL" по уровню в движке

**Files:**
- Modify: `internal/service/trading_strategy/scalping/strategy/strategy.go` (тип `Position`)
- Modify: `internal/domain/backtest/portfolio.go` (поля/`open`/`mark`/`strategyPosition`)
- Modify: `internal/domain/backtest/engine.go:135-137` и зеркальная ветка в `Trace` (~строка 214)
- Test: `internal/domain/backtest/portfolio_test.go`, `internal/domain/backtest/engine_test.go`

**Interfaces:**
- Produces: `strategy.Position.PrevMaxFavorablePrice float64` — монотонный максимум закрытий **до** текущего бара (на баре входа = цене входа). Движок исполняет `sig.Reason` из набора `"SL", "TRAIL", "ATRSL"` по `min(sig.StopLoss, c.Open)`.

- [x] **Step 1: Failing tests**

В `portfolio_test.go` (по стилю соседних тестов):

```go
func TestPrevMaxFavorableLagsMark(t *testing.T) {
	p := newPortfolio(Config{InitialCash: 1000, Fraction: 1, Commission: 0, Lot: 1})
	p.open(100, time.Now(), 0, 0, 0, 0, "")
	pos := p.strategyPosition()
	if pos.PrevMaxFavorablePrice != 100 || pos.MaxFavorablePrice != 100 {
		t.Fatalf("after open: prev=%v max=%v, want 100/100", pos.PrevMaxFavorablePrice, pos.MaxFavorablePrice)
	}
	p.mark(105)
	pos = p.strategyPosition()
	if pos.PrevMaxFavorablePrice != 100 || pos.MaxFavorablePrice != 105 {
		t.Fatalf("after mark(105): prev=%v max=%v, want 100/105", pos.PrevMaxFavorablePrice, pos.MaxFavorablePrice)
	}
	p.mark(103) // не новый максимум: prev догоняет 105, max стоит
	pos = p.strategyPosition()
	if pos.PrevMaxFavorablePrice != 105 || pos.MaxFavorablePrice != 105 {
		t.Fatalf("after mark(103): prev=%v max=%v, want 105/105", pos.PrevMaxFavorablePrice, pos.MaxFavorablePrice)
	}
}
```

В `engine_test.go` (используя существующий `scriptedStrategy` и `flatCandles`):

```go
func TestEngineFillsATRSLAtStopLevel(t *testing.T) {
	// Вход по 100, затем sell ATRSL со StopLoss 95 на баре с open 97 и close 90:
	// исполнение должно быть по min(95, 97) = 95, а не по close 90.
	candles := flatCandles([]float64{10, 100, 90})
	candles[2].Open = 97
	s := scriptedStrategy{lookback: 1, decide: func(md strategy.MarketData) model.Signal {
		if md.Position == nil && md.Price == 100 {
			return model.Signal{Kind: model.SignalBuy}
		}
		if md.Position != nil && md.Price == 90 {
			return model.Signal{Kind: model.SignalSell, Reason: "ATRSL", StopLoss: 95}
		}
		return model.Signal{}
	}}
	res := Run(s, candles, nil, nil, Config{InitialCash: 1000, Fraction: 1, Commission: 0, Lot: 1})
	if len(res.Trades) != 1 {
		t.Fatalf("trades = %d, want 1", len(res.Trades))
	}
	if got := res.Trades[0].ExitPrice; got != 95 {
		t.Fatalf("ATRSL exit price = %v, want 95 (stop level)", got)
	}
}
```

Примечание: поле цены выхода в `Trade` сверить с фактическим именем в `internal/domain/backtest` (`grep "ExitPrice\|SellPrice" internal/domain/backtest/*.go`) и использовать реальное. `flatCandles`/`scriptedStrategy` уже есть в `engine_test.go` — если сигнатуры отличаются, адаптировать вызов, не меняя суть проверки.

- [x] **Step 2: Убедиться, что падают**

Run: `go test ./internal/domain/backtest/ -run 'PrevMaxFavorable|ATRSLAtStop' -v`
Expected: FAIL (`PrevMaxFavorablePrice` undefined; exit price 90 ≠ 95).

- [x] **Step 3: Реализация**

`strategy.go` — в `Position` после `MaxFavorablePrice`:

```go
	// PrevMaxFavorablePrice is MaxFavorablePrice as of the PREVIOUS bar (before the
	// current bar's close was marked). The reversion intrabar stop computes its trail
	// level from it: the exchange stop order working during bar i was placed after bar
	// i-1 closed, so its level knows nothing about bar i. Seeded to the entry price.
	PrevMaxFavorablePrice float64
```

`portfolio.go`:
- поле `prevMaxFavorable float64` рядом с `maxFavorable`;
- в `open()`: `p.prevMaxFavorable = price` (рядом с `p.maxFavorable = price`);
- в `mark(price)`: первой строкой `p.prevMaxFavorable = p.maxFavorable`, затем существующий подъём максимума;
- в `strategyPosition()`: `PrevMaxFavorablePrice: p.prevMaxFavorable,`;
- в сбросе позиции (там, где `p.maxFavorable = 0`): `p.prevMaxFavorable = 0`.

`engine.go` — в обоих местах (Run ~135 и Trace ~214):

```go
				switch sig.Reason {
				case "SL", "TRAIL", "ATRSL":
					exitPrice = min(sig.StopLoss, c.Open)
```

Гвард: если `sig.StopLoss == 0` при этих причинах, `min(0, open)=0` испортит сделку — добавить условие `if sig.StopLoss > 0` перед подстановкой (сохранив текущее поведение для нулевого стопа: исполнение по close).

- [x] **Step 4: Тесты зелёные**

Run: `go test ./internal/domain/backtest/ ./internal/service/trading_strategy/... -count=1`
Expected: PASS (в т.ч. существующие тесты движка/levels — их поведение не должно измениться).

- [x] **Step 5: Commit**

```bash
git add internal/service/trading_strategy/scalping/strategy/strategy.go internal/domain/backtest/
git commit -m "feat(backtest): PrevMaxFavorablePrice on Position and stop-level fills for ATRSL"
```

---

### Task 3: Ядро reversion — объединённый интрабарный стоп + `core.DesiredStop`

**Files:**
- Modify: `internal/service/trading_strategy/reversion/strategy/core/core.go` (`decideInput`, `buildInput`, `manage`, doc-comment пакета/manage)
- Test: `internal/service/trading_strategy/reversion/strategy/core/core_test.go`

**Interfaces:**
- Consumes: `strategy.Position.PrevMaxFavorablePrice` (Task 2).
- Produces:
  ```go
  // DesiredStop returns the single protective stop level for a reversion position
  // and the reason of the binding component ("SL" | "ATRSL" | "TRAIL"), or (0, "")
  // when no price stop is enabled/armed. maxFav is the monotonic max of closes the
  // stop may trail from (backtest passes PrevMaxFavorablePrice; live passes the
  // persisted MaxFav).
  func DesiredStop(p Params, entryPrice, entryATR, maxFav float64) (level float64, reason string)
  ```
  Новое поведение `manage`: приоритет `STOP(SL|TRAIL|ATRSL) → OB → RSI50 → BE → RSIOS → EMAX`; STOP триггерится по `low ≤ level`, кладёт `sig.StopLoss = level`, `sig.Reason = reason`.

- [x] **Step 1: Failing tests**

Добавить в `core_test.go` (стиль `openInput()` + `s.decide(in)`; `openInput` дополнится полем `low` — см. Step 3):

```go
// Интрабарный прокол: low ниже уровня, close выше — раньше держали, теперь продаём.
func TestIntrabarStopFiresOnLowTouch(t *testing.T) {
	p := defaultParams()
	p.UseTrail, p.TrailATRMult, p.ATRPeriod = 1, 1.5, 14
	s := NewWithParams("T", p)
	in := openInput()
	in.pos = &strategy.Position{PurchasePrice: 100, EntryATR: 2,
		MaxFavorablePrice: 110, PrevMaxFavorablePrice: 110} // trail = 110-3 = 107
	in.price, in.low = 108, 106.5 // close выше уровня, low проколол
	sig := s.decide(in)
	if sig.Kind != model.SignalSell || sig.Reason != "TRAIL" {
		t.Fatalf("want TRAIL sell on low touch, got kind=%v reason=%q", sig.Kind, sig.Reason)
	}
	if sig.StopLoss != 107 {
		t.Fatalf("StopLoss = %v, want 107 (trail level)", sig.StopLoss)
	}
}

// Трейлинг считается от PrevMaxFav: спайк-бар с новым максимумом закрытия не должен
// выбивать по завышенному уровню, которого на «бирже» ещё не было.
func TestIntrabarTrailUsesPrevMaxFav(t *testing.T) {
	p := defaultParams()
	p.UseTrail, p.TrailATRMult, p.ATRPeriod = 1, 1.5, 14
	s := NewWithParams("T", p)
	in := openInput()
	in.pos = &strategy.Position{PurchasePrice: 100, EntryATR: 2,
		MaxFavorablePrice: 120, PrevMaxFavorablePrice: 110} // рабочий уровень 107, не 117
	in.price, in.low = 119, 116 // low 116 > 107 -> держим (по MaxFav 120 выбило бы)
	if sig := s.decide(in); sig.Kind == model.SignalSell {
		t.Fatalf("prev-max trail must hold, got sell %q", sig.Reason)
	}
}

// Из двух активных стопов побеждает больший уровень и его причина.
func TestDesiredStopPicksBindingComponent(t *testing.T) {
	p := defaultParams()
	p.UseATRStop, p.StopATRMult, p.ATRPeriod = 1, 1.2, 14 // 100-2.4 = 97.6
	p.UseTrail, p.TrailATRMult = 1, 1.5                   // 110-3 = 107
	level, reason := DesiredStop(p, 100, 2, 110)
	if level != 107 || reason != "TRAIL" {
		t.Fatalf("DesiredStop = %v/%q, want 107/TRAIL", level, reason)
	}
	level, reason = DesiredStop(p, 100, 2, 0) // maxFav неизвестен -> трейл не участвует
	if level != 97.6 || reason != "ATRSL" {
		t.Fatalf("DesiredStop = %v/%q, want 97.6/ATRSL", level, reason)
	}
	if level, reason = DesiredStop(p, 100, 0, 110); reason != "" || level != 0 {
		t.Fatalf("EntryATR=0 must disable price stops, got %v/%q", level, reason)
	}
}

// Стоп теперь старше OB: прокол и перекупленность в одном баре -> STOP.
func TestStopBeatsOverboughtSameBar(t *testing.T) {
	p := defaultParams()
	p.UseOverbought, p.RSIOverbought, p.StochOverbought = 1, 70, 80
	p.CatStopATRMult, p.ATRPeriod = 2, 14 // SL = 100-4 = 96
	s := NewWithParams("T", p)
	in := openInput()
	in.pos = &strategy.Position{PurchasePrice: 100, EntryATR: 2,
		MaxFavorablePrice: 100, PrevMaxFavorablePrice: 100}
	in.rsiPrev, in.rsiNow = 72, 75
	in.stochPrev, in.stochNow = 82, 85
	in.price, in.low = 101, 95.5
	if sig := s.decide(in); sig.Reason != "SL" {
		t.Fatalf("stop must outrank OB, got %q", sig.Reason)
	}
}

// Прогрев: low-сентинел 0 не триггерит стоп.
func TestIntrabarStopSkippedWithoutLow(t *testing.T) {
	p := defaultParams()
	p.CatStopATRMult, p.ATRPeriod = 2, 14
	s := NewWithParams("T", p)
	in := openInput()
	in.pos = &strategy.Position{PurchasePrice: 100, EntryATR: 2,
		MaxFavorablePrice: 100, PrevMaxFavorablePrice: 100}
	in.price, in.low = 101, 0
	if sig := s.decide(in); sig.Kind == model.SignalSell {
		t.Fatalf("low=0 sentinel must not fire the stop, got %q", sig.Reason)
	}
}
```

- [x] **Step 2: Убедиться, что падают**

Run: `go test ./internal/service/trading_strategy/reversion/strategy/core/ -run 'Intrabar|DesiredStop|BeatsOverbought' -v`
Expected: FAIL (нет поля `low`, нет `DesiredStop`, `PrevMaxFavorablePrice` не используется).

- [x] **Step 3: Реализация**

1. `decideInput`: добавить поле `low float64 // текущий бар: intraday low (0 на прогреве/пустой серии)`.
2. `buildInput`: заполнить `low` из `md.Lows[len(md.Lows)-1]` (0, если срез пуст); прокинуть в возвращаемый литерал.
3. Экспортируемая функция (рядом с `manage`):

```go
// DesiredStop ... (doc-comment из Interfaces выше)
func DesiredStop(p Params, entryPrice, entryATR, maxFav float64) (float64, string) {
	if entryATR <= 0 {
		return 0, ""
	}
	level, reason := 0.0, ""
	if p.CatStopATRMult > 0 {
		level, reason = entryPrice-p.CatStopATRMult*entryATR, "SL"
	}
	if p.UseATRStop == 1 && p.StopATRMult > 0 {
		if l := entryPrice - p.StopATRMult*entryATR; l > level {
			level, reason = l, "ATRSL"
		}
	}
	if p.UseTrail == 1 && p.TrailATRMult > 0 && maxFav > 0 {
		if l := maxFav - p.TrailATRMult*entryATR; l > level {
			level, reason = l, "TRAIL"
		}
	}
	if level <= 0 {
		return 0, ""
	}
	return level, reason
}
```

4. `manage()`: удалить три старых case (`SL` ~432, `TRAIL` ~439, `ATRSL` ~459) и первым case поставить объединённый стоп:

```go
	stopLevel, stopReason := DesiredStop(s.p, in.pos.PurchasePrice, in.pos.EntryATR, in.pos.PrevMaxFavorablePrice)

	switch {
	case stopReason != "" && in.low > 0 && in.low <= stopLevel:
		sig.Kind, sig.Reason = model.SignalSell, stopReason
		sig.StopLoss = stopLevel
		sig.ExitReason = fmt.Sprintf("%s: low %.4f ≤ стоп %.4f (вход %.4f, ATR %.4f, prevMaxFav %.4f)",
			stopReason, in.low, stopLevel, in.pos.PurchasePrice, in.pos.EntryATR, in.pos.PrevMaxFavorablePrice)
	case s.p.UseOverbought == 1 && ...: // OB — без изменений
	case s.p.UseRSI50 == 1 && ...:      // RSI50 — без изменений
	case s.p.UseBreakeven == 1 && ...:  // BE — без изменений (in.price, MaxFavorablePrice)
	case s.p.UseATRStop == 0 && ...:    // RSIOS — без изменений
	case in.emaOK && ...:               // EMAX — без изменений
	}
```

5. Обновить doc-comment пакета (строки ~1-10) и `manage` (~401-422): новый приоритет `STOP(SL|TRAIL|ATRSL) → OB → RSI50 → BE → RSIOS → EMAX`, интрабарный триггер по low, исполнение движком по `min(level, open)`, BE/RSIOS/OB/RSI50/EMAX — по close.
6. Починить существующие тесты: в `openInput()` добавить `low: <=цене price>` по умолчанию (чтобы нейтральные сценарии не проколоть); тесты, которые триггерили ATRSL/SL/TRAIL через `in.price` (`TestExitATRStopFires`, `TestNoATRStopAboveThreshold`, `TestATRStopSkippedWhenEntryATRZero`, `TestATRStopSkippedWhenMultZero`, catstop/trail/precedence-тесты), перевести на `in.low` и заполнить `PrevMaxFavorablePrice` там, где участвует трейлинг. Смысл каждой проверки сохранить (падение → низкий low; удержание → low выше уровня).

- [x] **Step 4: Тесты зелёные**

Run: `go test ./internal/service/trading_strategy/reversion/... -count=1`
Expected: PASS.

- [x] **Step 5: Commit**

```bash
git add internal/service/trading_strategy/reversion/strategy/core/
git commit -m "feat(reversion): combined intrabar stop (low-triggered) with DesiredStop and new exit precedence"
```

---

### Task 4: Пакет `live/stoporders` + стейт + уведомление

**Files:**
- Create: `internal/service/trading_strategy/reversion/live/stoporders/stoporders.go`
- Create: `internal/service/trading_strategy/reversion/live/stoporders/stoporders_test.go`
- Modify: `internal/service/trading_strategy/reversion/live/statestore/statestore.go` (поля Entry)
- Modify: `internal/service/trading_strategy/reversion/live/notifier/notifier.go` (+ тест в `notifier_test.go`)
- Modify: `.mockery.yaml` (+ сгенерировать мок)

**Interfaces:**
- Consumes: `investapi` стабы (Task 1), `utils.SplitPrice(price float64) (int64, int32)`, `utils.CombinePrice(units int64, nano int32) float64`, `uuid.NewString()`.
- Produces:
  ```go
  package stoporders
  type Client interface { // мокается mockery
      PostStopOrder(ctx context.Context, in *investapi.PostStopOrderRequest, opts ...grpc.CallOption) (*investapi.PostStopOrderResponse, error)
      GetStopOrders(ctx context.Context, in *investapi.GetStopOrdersRequest, opts ...grpc.CallOption) (*investapi.GetStopOrdersResponse, error)
      CancelStopOrder(ctx context.Context, in *investapi.CancelStopOrderRequest, opts ...grpc.CallOption) (*investapi.CancelStopOrderResponse, error)
  }
  type ActiveStop struct{ InstrumentUID, StopOrderID string; StopPrice float64 }
  type Result struct{ Placed bool; OrderID string } // Placed=false -> dry-run
  func New(c Client, accountID string, tradeEnabled bool) *Executor
  func (e *Executor) Place(ctx context.Context, instrumentID string, lots int64, stopPrice, minPriceIncrement float64) (Result, error)
  func (e *Executor) Cancel(ctx context.Context, stopOrderID string) error // dry-run: no-op nil
  func (e *Executor) List(ctx context.Context) ([]ActiveStop, error)      // dry-run: (nil, nil); только SELL-заявки
  ```
  `statestore.Entry` += `StopOrderID string`, `StopPrice float64`, `StopReason string` (json: `stopOrderId`/`stopPrice`/`stopReason`, omitempty).
  `notifier.StopSet(ticker string, price float64, reason string, paper bool) string`.

- [x] **Step 1: Failing tests**

`stoporders_test.go` (мок появится в Step 3 — писать тест сразу под `mocks.NewMockClient(t)`):

```go
package stoporders

import (
	"context"
	"testing"

	"github.com/stretchr/testify/mock"

	investapi "tinvest/internal/pb/v1"
	"tinvest/internal/service/trading_strategy/reversion/live/stoporders/mocks"
	"tinvest/internal/utils"
)

func TestPlaceBuildsStopMarketSellRoundedDown(t *testing.T) {
	c := mocks.NewMockClient(t)
	var got *investapi.PostStopOrderRequest
	c.EXPECT().PostStopOrder(mock.Anything, mock.Anything).
		Run(func(_ context.Context, in *investapi.PostStopOrderRequest, _ ...interface{}) { got = in }).
		Return(&investapi.PostStopOrderResponse{StopOrderId: "so-1"}, nil)

	e := New(c, "acc", true)
	res, err := e.Place(context.Background(), "uid-1", 3, 107.037, 0.05) // 107.037 -> 107.00
	if err != nil || !res.Placed || res.OrderID != "so-1" {
		t.Fatalf("res=%+v err=%v", res, err)
	}
	if got.GetInstrumentId() != "uid-1" || got.GetQuantity() != 3 || got.GetAccountId() != "acc" {
		t.Fatalf("req fields: %+v", got)
	}
	if got.GetDirection() != investapi.StopOrderDirection_STOP_ORDER_DIRECTION_SELL ||
		got.GetStopOrderType() != investapi.StopOrderType_STOP_ORDER_TYPE_STOP_LOSS ||
		got.GetExchangeOrderType() != investapi.ExchangeOrderType_EXCHANGE_ORDER_TYPE_MARKET ||
		got.GetExpirationType() != investapi.StopOrderExpirationType_STOP_ORDER_EXPIRATION_TYPE_GOOD_TILL_CANCEL {
		t.Fatalf("order type fields: %+v", got)
	}
	if p := utils.CombinePrice(got.GetStopPrice().GetUnits(), got.GetStopPrice().GetNano()); p != 107.00 {
		t.Fatalf("stop price = %v, want 107.00 (rounded down to 0.05)", p)
	}
	if got.GetOrderId() == "" {
		t.Fatal("idempotency order_id must be set")
	}
}

func TestDryRunTouchesNoAPI(t *testing.T) {
	c := mocks.NewMockClient(t) // без EXPECT: любой вызов = провал теста
	e := New(c, "acc", false)
	res, err := e.Place(context.Background(), "uid-1", 1, 100, 0.01)
	if err != nil || res.Placed {
		t.Fatalf("dry-run Place: res=%+v err=%v", res, err)
	}
	if err := e.Cancel(context.Background(), "so-1"); err != nil {
		t.Fatalf("dry-run Cancel: %v", err)
	}
	if list, err := e.List(context.Background()); err != nil || list != nil {
		t.Fatalf("dry-run List: %v %v", list, err)
	}
}

func TestListReturnsOnlyActiveSellStops(t *testing.T) {
	c := mocks.NewMockClient(t)
	c.EXPECT().GetStopOrders(mock.Anything, mock.MatchedBy(func(in *investapi.GetStopOrdersRequest) bool {
		return in.GetAccountId() == "acc" && in.GetStatus() == investapi.StopOrderStatusOption_STOP_ORDER_STATUS_ACTIVE
	})).Return(&investapi.GetStopOrdersResponse{StopOrders: []*investapi.StopOrder{
		{StopOrderId: "so-1", InstrumentUid: "uid-1",
			Direction: investapi.StopOrderDirection_STOP_ORDER_DIRECTION_SELL,
			StopPrice: &investapi.MoneyValue{Units: 107}},
		{StopOrderId: "so-2", InstrumentUid: "uid-2",
			Direction: investapi.StopOrderDirection_STOP_ORDER_DIRECTION_BUY,
			StopPrice: &investapi.MoneyValue{Units: 50}},
	}}, nil)

	e := New(c, "acc", true)
	list, err := e.List(context.Background())
	if err != nil || len(list) != 1 || list[0].StopOrderID != "so-1" || list[0].StopPrice != 107 {
		t.Fatalf("list=%v err=%v", list, err)
	}
}
```

Примечания для реализатора: точные имена геттеров/типов (`MoneyValue` vs `Quotation` у `StopPrice` в `StopOrder`, тип `Nano`) сверить со сгенерированным `stoporders.pb.go`; `PostStopOrderRequest.StopPrice` — `*Quotation`, `StopOrder.StopPrice` — `*MoneyValue` (см. proto). Сигнатура `Run(...)` мока может требовать `...grpc.CallOption` вместо `...interface{}` — подстроиться под сгенерированный мок.

В `notifier_test.go` дополнить существующий стиль проверкой `StopSet("UGLD", 107.5, "TRAIL", true)`: строка содержит тикер, "107.5", "TRAIL" и пометку dry-run (какую именно — скопировать из `paperTag`).

- [x] **Step 2: Реализация + мок**

`stoporders.go`:

```go
// Package stoporders places (or dry-runs) the single protective stop-market SELL
// order the reversion runner keeps on the exchange per open position. Each order
// carries a fresh UUID order_id for idempotency.
package stoporders

import (
	"context"
	"fmt"
	"math"

	"github.com/google/uuid"
	"google.golang.org/grpc"

	investapi "tinvest/internal/pb/v1"
	"tinvest/internal/utils"
)

type Client interface { /* из Interfaces выше */ }

type ActiveStop struct {
	InstrumentUID string
	StopOrderID   string
	StopPrice     float64
}

type Result struct {
	Placed  bool
	OrderID string
}

type Executor struct {
	client       Client
	accountID    string
	tradeEnabled bool
}

func New(c Client, accountID string, tradeEnabled bool) *Executor {
	return &Executor{client: c, accountID: accountID, tradeEnabled: tradeEnabled}
}

// roundDownToIncrement snaps price DOWN to the instrument's min price increment —
// conservative for a sell stop (never above the strategy level). incr<=0 keeps price.
func roundDownToIncrement(price, incr float64) float64 {
	if incr <= 0 {
		return price
	}
	return math.Floor(price/incr+1e-9) * incr
}

func (e *Executor) Place(ctx context.Context, instrumentID string, lots int64, stopPrice, minPriceIncrement float64) (Result, error) {
	if !e.tradeEnabled {
		return Result{Placed: false}, nil
	}
	rounded := roundDownToIncrement(stopPrice, minPriceIncrement)
	units, nano := utils.SplitPrice(rounded)
	req := &investapi.PostStopOrderRequest{
		InstrumentId:      instrumentID,
		Quantity:          lots,
		StopPrice:         &investapi.Quotation{Units: units, Nano: nano},
		Direction:         investapi.StopOrderDirection_STOP_ORDER_DIRECTION_SELL,
		AccountId:         e.accountID,
		ExpirationType:    investapi.StopOrderExpirationType_STOP_ORDER_EXPIRATION_TYPE_GOOD_TILL_CANCEL,
		StopOrderType:     investapi.StopOrderType_STOP_ORDER_TYPE_STOP_LOSS,
		ExchangeOrderType: investapi.ExchangeOrderType_EXCHANGE_ORDER_TYPE_MARKET,
		OrderId:           uuid.NewString(),
	}
	resp, err := e.client.PostStopOrder(ctx, req)
	if err != nil {
		return Result{}, fmt.Errorf("post stop order: %w", err)
	}
	return Result{Placed: true, OrderID: resp.GetStopOrderId()}, nil
}

func (e *Executor) Cancel(ctx context.Context, stopOrderID string) error {
	if !e.tradeEnabled {
		return nil
	}
	_, err := e.client.CancelStopOrder(ctx, &investapi.CancelStopOrderRequest{
		AccountId: e.accountID, StopOrderId: stopOrderID,
	})
	if err != nil {
		return fmt.Errorf("cancel stop order %s: %w", stopOrderID, err)
	}
	return nil
}

func (e *Executor) List(ctx context.Context) ([]ActiveStop, error) {
	if !e.tradeEnabled {
		return nil, nil
	}
	resp, err := e.client.GetStopOrders(ctx, &investapi.GetStopOrdersRequest{
		AccountId: e.accountID,
		Status:    investapi.StopOrderStatusOption_STOP_ORDER_STATUS_ACTIVE,
	})
	if err != nil {
		return nil, fmt.Errorf("get stop orders: %w", err)
	}
	var out []ActiveStop
	for _, so := range resp.GetStopOrders() {
		if so.GetDirection() != investapi.StopOrderDirection_STOP_ORDER_DIRECTION_SELL {
			continue
		}
		out = append(out, ActiveStop{
			InstrumentUID: so.GetInstrumentUid(),
			StopOrderID:   so.GetStopOrderId(),
			StopPrice:     utils.CombinePrice(so.GetStopPrice().GetUnits(), so.GetStopPrice().GetNano()),
		})
	}
	return out, nil
}
```

Проверить фактическую сигнатуру `utils.SplitPrice` (возвращает `(int64, int32)`; если nano-тип иной — привести).

`statestore.go` — в `Entry`:

```go
	StopOrderID string  `json:"stopOrderId,omitempty"` // активная биржевая стоп-заявка ("" = нет)
	StopPrice   float64 `json:"stopPrice,omitempty"`   // уровень выставленной заявки
	StopReason  string  `json:"stopReason,omitempty"`  // компонент уровня: SL | ATRSL | TRAIL
```

`notifier.go`:

```go
// StopSet reports a protective stop order (re)placed at price for reason.
func StopSet(ticker string, price float64, reason string, paper bool) string {
	return fmt.Sprintf("🛡 %s: стоп-заявка %s на %.4f%s", ticker, reason, price, paperTag(paper))
}
```

(формат `paperTag` — как в соседних функциях).

`.mockery.yaml` — добавить:

```yaml
  tinvest/internal/service/trading_strategy/reversion/live/stoporders:
    interfaces:
      Client:
```

Сгенерировать: `./bin/mage mocks`.

- [x] **Step 3: Тесты зелёные**

Run: `go test ./internal/service/trading_strategy/reversion/live/... -count=1`
Expected: PASS.

- [x] **Step 4: Commit**

```bash
git add internal/service/trading_strategy/reversion/live/ .mockery.yaml
git commit -m "feat(reversion/live): stoporders executor, stop fields in state, StopSet notify"
```

---

### Task 5: buyPass ставит стоп сразу после покупки + wiring

**Files:**
- Modify: `internal/service/trading_strategy/reversion/live/live.go` (`NewService`, структура `service`)
- Modify: `internal/service/trading_strategy/reversion/live/registry.go` (+`ParamsFor`)
- Modify: `internal/service/trading_strategy/reversion/live/buy.go`
- Modify: `internal/service_provider/service.go:227-243` (`GetReversionLiveService`)
- Test: `internal/service/trading_strategy/reversion/live/service_test.go`, `registry_test.go`

**Interfaces:**
- Consumes: `stoporders.New/Place/Result` (Task 4), `core.DesiredStop` (Task 3), `grpcClient.StopOrdersServiceClient()` (Task 1).
- Produces:
  ```go
  func NewService(instruments instrumentsClient, market marketdata.CandleClient,
      ops operationsClient, orders executor.OrdersClient, stops stoporders.Client,
      tg telegram.Client, cfg *config.ReversionConfig) *service
  // поле service.stops *stoporders.Executor
  func ParamsFor(ticker string) (core.Params, bool) // registry.go
  ```

- [x] **Step 1: Failing test**

В `service_test.go` — тест «покупка ставит стоп». Данные: серия, дающая buy-сигнал, уже есть? Нет — существующие тесты только no-signal. Проще протестировать через seed: вызвать неэкспортируемый хелпер напрямую. Поэтому выделить постановку стопа в метод и тестировать его юнитом:

```go
// в buy.go появится метод:
// func (s *service) placeInitialStop(ctx context.Context, ticker string,
//     sh *imodel.Share, entry statestore.Entry, state map[string]statestore.Entry,
//     store statestore.Store) statestore.Entry

func TestPlaceInitialStopPersistsIDAndLevel(t *testing.T) {
	dir := t.TempDir()
	statePath := filepath.Join(dir, "state.json")
	stops := stopmocks.NewMockClient(t)
	stops.EXPECT().PostStopOrder(mock.Anything, mock.Anything).
		Return(&investapi.PostStopOrderResponse{StopOrderId: "so-9"}, nil)

	c := cfg(dir)
	c.TradeEnabled = true
	svc := NewService(livemocks.NewMockinstrumentsClient(t), mdmocks.NewMockCandleClient(t),
		livemocks.NewMockoperationsClient(t), nil, stops, tgmocks.NewMockClient(t), c)
	svc.statePath = statePath

	store := statestore.New(statePath)
	entry := statestore.Entry{Ticker: "UGLD", EntryPrice: 100, EntryATR: 2, MaxFav: 100, Quantity: 10}
	state := map[string]statestore.Entry{"UGLD": entry}
	sh := &imodel.Share{ID: "uid-ugld", Ticker: "UGLD", Lot: 1, MinPriceIncrement: 0.01}

	got := svc.placeInitialStop(context.Background(), "UGLD", sh, entry, state, store)
	// UGLD DefaultParams: UseTrail=1/TrailATRMult=1.5, CatStop=0, UseATRStop=0
	// -> want = 100 - 1.5*2 = 97, reason TRAIL
	if got.StopOrderID != "so-9" || got.StopPrice != 97 || got.StopReason != "TRAIL" {
		t.Fatalf("entry after stop: %+v", got)
	}
	persisted, _ := store.Load()
	if persisted["UGLD"].StopOrderID != "so-9" {
		t.Fatalf("state not persisted: %+v", persisted["UGLD"])
	}
}
```

(тикер UGLD: подтвердить флаги в `strategy/ugld/ugld.go` — UseTrail=1, TrailATRMult=1.5, CatStopATRMult=0, UseATRStop=0; если отличаются — скорректировать ожидание по формуле.)
Импорты: `stopmocks "tinvest/internal/service/trading_strategy/reversion/live/stoporders/mocks"`, `investapi "tinvest/internal/pb/v1"`.

Плюс тест реестра в `registry_test.go`:

```go
func TestParamsForKnownAndUnknown(t *testing.T) {
	if _, ok := ParamsFor("UGLD"); !ok {
		t.Fatal("UGLD must be registered")
	}
	if _, ok := ParamsFor("NOPE"); ok {
		t.Fatal("unknown ticker must return ok=false")
	}
}
```

- [x] **Step 2: Убедиться, что падают**

Run: `go test ./internal/service/trading_strategy/reversion/live/ -run 'PlaceInitialStop|ParamsFor' -v`
Expected: FAIL (нет `stops` в NewService, нет `placeInitialStop`, нет `ParamsFor`).

- [x] **Step 3: Реализация**

`registry.go`:

```go
// ParamsFor returns the calibrated params for a known ticker, ok=false otherwise.
func ParamsFor(ticker string) (core.Params, bool) {
	p, ok := paramsByTicker[ticker]
	return p, ok
}
```

`live.go`: параметр `stops stoporders.Client` в `NewService` (между `orders` и `tg`), поле `stops *stoporders.Executor`, инициализация `stops: stoporders.New(stops, cfg.AccountID, cfg.TradeEnabled)`. Doc-comment: stops может быть nil только при TradeEnabled=false.

`buy.go` — после сохранения стейта покупки (строка ~105) вызвать:

```go
		state[ticker] = s.placeInitialStop(ctx, ticker, sh, state[ticker], state, store)
```

и добавить метод:

```go
// placeInitialStop puts the protective exchange stop right after a fill so the
// position is never unprotected for the first hour. On failure the entry keeps an
// empty StopOrderID and managePass retries next tick.
func (s *service) placeInitialStop(ctx context.Context, ticker string, sh *imodel.Share,
	entry statestore.Entry, state map[string]statestore.Entry, store statestore.Store) statestore.Entry {

	p, ok := ParamsFor(ticker)
	if !ok {
		return entry
	}
	level, reason := core.DesiredStop(p, entry.EntryPrice, entry.EntryATR, entry.MaxFav)
	if reason == "" {
		return entry
	}
	lots := entry.Quantity / int64(sh.Lot)
	res, err := s.stops.Place(ctx, sh.ID, lots, level, sh.MinPriceIncrement)
	if err != nil {
		s.notify(notifier.Alert(ticker, "стоп-заявка не выставлена: "+err.Error()))
		logger.ErrorContext(ctx, fmt.Sprintf("reversion: %s place stop: %v", ticker, err))
		return entry
	}
	if res.Placed {
		entry.StopOrderID = res.OrderID
	}
	entry.StopPrice, entry.StopReason = level, reason
	state[ticker] = entry
	_ = store.Save(state)
	s.notify(notifier.StopSet(ticker, level, reason, !res.Placed))
	return entry
}
```

(guard `sh.Lot <= 0` уже есть выше по buy-пути перед sizing; если нет — добавить перед делением.)

`service_provider/service.go`: в `GetReversionLiveService` добавить аргумент `grpcClient.StopOrdersServiceClient(),` между `OrdersServiceClient()` и `tgClient`.

Все существующие вызовы `NewService(...)` в тестах: добавить пятым аргументом `nil` (dry-run) или мок.

- [x] **Step 4: Тесты зелёные + сборка**

Run: `go build ./internal/... ./pkg/... ./cmd/... && go test ./internal/service/trading_strategy/reversion/live/... ./internal/service_provider/... -count=1`
Expected: PASS.

- [x] **Step 5: Commit**

```bash
git add internal/service/trading_strategy/reversion/live/ internal/service_provider/
git commit -m "feat(reversion/live): place protective stop order right after entry fill"
```

---

### Task 6: managePass — сверка, перевыставление, детект срабатывания, защита от двойной продажи

**Files:**
- Modify: `internal/service/trading_strategy/reversion/live/manage.go`
- Test: `internal/service/trading_strategy/reversion/live/service_test.go`

**Interfaces:**
- Consumes: `stoporders.Executor.{Place,Cancel,List}`, `core.DesiredStop`, `ParamsFor`, `statestore.Entry.{StopOrderID,StopPrice,StopReason}`, `strategy.Position.PrevMaxFavorablePrice`.
- Produces: поведение managePass по секции 3 спеки (шесть случаев). Новых экспортируемых имён нет.

- [x] **Step 1: Failing tests**

Тесты уровня managePass через `svc.Run(ModeManage)` с моками (стиль `TestManagePass_UpdatesMaxFavAndPersists`; `TradeEnabled=true`, `stops` = мок). Общий сборщик окружения + четыре сценария:

```go
// manageEnv wires a UGLD manage-pass with an open position and a stops mock.
// hourlyLastClose управляет последним закрытием (MaxFav/сигналы ядра).
type manageEnv struct {
	svc       *service
	statePath string
	stops     *stopmocks.MockClient
	tg        *tgmocks.MockClient
	orders    *execmocks.MockOrdersClient
}

func newManageEnv(t *testing.T, seed statestore.Entry, hourly []*imodel.CandleItemTechAnalyse) *manageEnv {
	t.Helper()
	dir := t.TempDir()
	statePath := filepath.Join(dir, "state.json")
	if err := statestore.New(statePath).Save(map[string]statestore.Entry{"UGLD": seed}); err != nil {
		t.Fatal(err)
	}

	instruments := livemocks.NewMockinstrumentsClient(t)
	instruments.EXPECT().Shares(mock.Anything).
		Return([]*imodel.Share{{ID: "uid-ugld", Ticker: "UGLD", Name: "ЮГК", Lot: 1,
			Trading: true, MinPriceIncrement: 0.01}}, nil)

	market := mdmocks.NewMockCandleClient(t)
	market.EXPECT().
		GetCandles(mock.Anything, mock.Anything, mock.MatchedBy(isHourly1), mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(hourly, nil)

	ops := livemocks.NewMockoperationsClient(t)
	ops.EXPECT().GetPortfolio(mock.Anything, mock.Anything).
		Return([]*grpcmodel.Position{{ShareID: "uid-ugld", InstrumentType: "share",
			Quantity: seed.Quantity, PurchasePrice: gq(seed.EntryPrice)}}, nil)

	stops := stopmocks.NewMockClient(t)
	tg := tgmocks.NewMockClient(t)
	tg.EXPECT().SendMessage(mock.Anything).Return(nil).Maybe()
	orders := execmocks.NewMockOrdersClient(t)

	c := cfg(dir)
	c.TradeEnabled = true
	svc := NewService(instruments, market, ops, orders, stops, tg, c)
	svc.statePath = statePath
	return &manageEnv{svc: svc, statePath: statePath, stops: stops, tg: tg, orders: orders}
}

// hourlySeries строит n флэт-баров по 100 и заменяет последние len(tail) закрытий.
func hourlySeries(n int, tail ...float64) []*imodel.CandleItemTechAnalyse {
	out := flatHourly(n)
	for i, c := range tail {
		idx := n - len(tail) + i
		out[idx].Open, out[idx].High, out[idx].Low, out[idx].Close = q(c), q(c), q(c), q(c)
	}
	return out
}

func seedEntry(stopID string, stopPrice float64) statestore.Entry {
	return statestore.Entry{Ticker: "UGLD", EntryPrice: 100, EntryATR: 2, MaxFav: 100,
		Quantity: 10, EntryTime: time.Date(2026, 6, 1, 10, 0, 0, 0, time.UTC),
		StopOrderID: stopID, StopPrice: stopPrice, StopReason: "TRAIL"}
}

func activeList(id string, price float64) *investapi.GetStopOrdersResponse {
	return &investapi.GetStopOrdersResponse{StopOrders: []*investapi.StopOrder{{
		StopOrderId: id, InstrumentUid: "uid-ugld",
		Direction: investapi.StopOrderDirection_STOP_ORDER_DIRECTION_SELL,
		StopPrice: &investapi.MoneyValue{Units: int64(price)},
	}}}
}

// 1. Трейлинг подтянулся: cancel старой + post новой, стейт несёт новый ID/уровень.
// UGLD params: UseTrail=1, TrailATRMult=1.5 -> close 110 => MaxFav 110, want 110-3=107 > 97.
func TestManagePass_RepostsStopWhenTrailRises(t *testing.T) {
	env := newManageEnv(t, seedEntry("so-old", 97), hourlySeries(400, 110))
	env.stops.EXPECT().GetStopOrders(mock.Anything, mock.Anything).Return(activeList("so-old", 97), nil)
	env.stops.EXPECT().CancelStopOrder(mock.Anything, mock.MatchedBy(func(in *investapi.CancelStopOrderRequest) bool {
		return in.GetStopOrderId() == "so-old"
	})).Return(&investapi.CancelStopOrderResponse{}, nil)
	env.stops.EXPECT().PostStopOrder(mock.Anything, mock.Anything).
		Return(&investapi.PostStopOrderResponse{StopOrderId: "so-new"}, nil)

	if err := env.svc.Run(context.Background(), dto.Run{Mode: dto.ModeManage}); err != nil {
		t.Fatalf("Run: %v", err)
	}
	st, _ := statestore.New(env.statePath).Load()
	got := st["UGLD"]
	if got.StopOrderID != "so-new" || got.StopPrice != 107 || got.StopReason != "TRAIL" {
		t.Fatalf("state after repost: %+v", got)
	}
}

// 2. Уровень не вырос: только List; отсутствие EXPECT на Cancel/Post ловит лишние вызовы.
func TestManagePass_KeepsStopWhenLevelUnchanged(t *testing.T) {
	env := newManageEnv(t, seedEntry("so-1", 97), hourlySeries(400)) // flat 100: want 97 == 97
	env.stops.EXPECT().GetStopOrders(mock.Anything, mock.Anything).Return(activeList("so-1", 97), nil)

	if err := env.svc.Run(context.Background(), dto.Run{Mode: dto.ModeManage}); err != nil {
		t.Fatalf("Run: %v", err)
	}
	st, _ := statestore.New(env.statePath).Load()
	if st["UGLD"].StopOrderID != "so-1" {
		t.Fatalf("stop must be kept, got %+v", st["UGLD"])
	}
}

// 3. Позиция исчезла + StopOrderID в стейте -> Exit с причиной из стейта, стейт очищен,
// Cancel не вызывается (заявка уже исполнена биржей).
func TestManagePass_DetectsFiredStop(t *testing.T) {
	env := newManageEnv(t, seedEntry("so-1", 97), hourlySeries(400))
	// Перекрыть GetPortfolio пустым ответом: newManageEnv уже поставил EXPECT c позицией,
	// поэтому для этого теста собрать окружение вручную по образцу newManageEnv, но
	// ops.EXPECT().GetPortfolio(...).Return(nil, nil) и tg с явной проверкой:
	//   tg.EXPECT().SendMessage(mock.MatchedBy(func(s string) bool {
	//       return strings.Contains(s, "TRAIL") && strings.Contains(s, "UGLD")
	//   })).Return(nil)
	// (без .Maybe() — уведомление обязано уйти). market/GetCandles не понадобится —
	// поставить .Maybe(). Никаких EXPECT на stops.Cancel/Post; List допускается .Maybe().
	_ = env // окружение из хелпера в этом тесте не используется — собрано вручную
	// assert: statestore пуст после Run.
}

// 4. SELL ядра при живой заявке и падающем Cancel -> рыночная продажа НЕ отправляется,
// позиция остаётся в стейте, уходит Alert.
func TestManagePass_CancelFailBlocksMarketSell(t *testing.T) {
	// Серия с обвалом хвоста заставляет ядро дать SELL (у UGLD включён trail: close 90
	// при MaxFav 100 пробивает close-бэкстоп... надёжнее выключить неоднозначность:
	// хвост 110, затем 90 — low-стоп ядра сработает по close-падению).
	env := newManageEnv(t, seedEntry("so-1", 97), hourlySeries(400, 110, 90))
	env.stops.EXPECT().GetStopOrders(mock.Anything, mock.Anything).Return(activeList("so-1", 97), nil).Maybe()
	env.stops.EXPECT().CancelStopOrder(mock.Anything, mock.Anything).
		Return(nil, fmt.Errorf("boom")).Maybe()
	// env.orders без EXPECT: вызов PostOrder провалит тест = продажи не было.

	if err := env.svc.Run(context.Background(), dto.Run{Mode: dto.ModeManage}); err != nil {
		t.Fatalf("Run: %v", err)
	}
	st, _ := statestore.New(env.statePath).Load()
	if _, ok := st["UGLD"]; !ok {
		t.Fatal("position must stay in state when cancel fails")
	}
}
```

Примечания: `execmocks` = `tinvest/internal/service/trading_strategy/reversion/live/executor/mocks`; `stopmocks` = `.../stoporders/mocks`; `investapi "tinvest/internal/pb/v1"`. Тест 3 собрать вручную по телу `newManageEnv` (другие ожидания ops/tg — см. комментарий). В тесте 4 сигнал ядра проверить локально (`go test -run CancelFail -v`); если серия `…110, 90` не даёт SELL — усилить обвал (несколько падающих закрытий 110→105→95→90), суть теста от формы сигнала не зависит.

- [x] **Step 2: Убедиться, что падают**

Run: `go test ./internal/service/trading_strategy/reversion/live/ -run ManagePass -v`
Expected: новые FAIL (логики нет), старые PASS.

- [x] **Step 3: Реализация**

Перестроить цикл managePass (`manage.go`), сохранив существующие шаги и добавив стоп-логику. Полный новый скелет тела цикла:

```go
	activeStops, listErr := s.stops.List(ctx) // один вызов на весь пасс
	if listErr != nil {
		s.notify(notifier.Alert("reversion", "GetStopOrders недоступен: "+listErr.Error()))
	}
	stopByInstrument := map[string]stoporders.ActiveStop{}
	stopByID := map[string]stoporders.ActiveStop{}
	for _, a := range activeStops {
		stopByInstrument[a.InstrumentUID] = a
		stopByID[a.StopOrderID] = a
	}

	for _, ticker := range s.cfg.Tickers {
		// ... StrategyFor / shares / (без изменений) ...

		pos, isHeld := held[sh.ID]
		if !isHeld {
			entry, hadState := state[ticker]
			if hadState && entry.StopOrderID != "" {
				// Наш биржевой стоп исполнился.
				s.notify(notifier.Exit(ticker, entry.StopReason, entry.StopPrice, entry.Quantity, false))
			}
			if hadState {
				delete(state, ticker)
				_ = store.Save(state)
			}
			continue
		}

		// ... reconstruct (без изменений) ...

		// Частичное исполнение: биржа продала часть позиции.
		if pos.Quantity < entry.Quantity && entry.StopOrderID != "" {
			entry.Quantity = pos.Quantity
			state[ticker] = entry
			_ = store.Save(state)
			s.notify(notifier.Alert(ticker, fmt.Sprintf("стоп исполнился частично, осталось %d", pos.Quantity)))
			// перевыставление на остаток произойдёт ниже общим путём (cancel+place)
			if err := s.stops.Cancel(ctx, entry.StopOrderID); err == nil {
				entry.StopOrderID = ""
				state[ticker] = entry
				_ = store.Save(state)
			}
		}

		// ... marketdata.Assemble (без изменений) ...

		prevMaxFav := entry.MaxFav // уровень, от которого считалась стоящая на бирже заявка
		// ... подъём MaxFav от md.Price и сохранение (без изменений) ...

		md.Position = &strategy.Position{
			PurchasePrice:         entry.EntryPrice,
			Quantity:              pos.Quantity,
			EntryATR:              entry.EntryATR,
			MaxFavorablePrice:     entry.MaxFav,
			PrevMaxFavorablePrice: prevMaxFav,
		}

		sig := st.Decide(md)
		if sig.Kind == model.SignalSell {
			// Любой SELL ядра: сначала снять биржевой стоп, потом рыночная продажа.
			if entry.StopOrderID != "" {
				if err := s.stops.Cancel(ctx, entry.StopOrderID); err != nil {
					s.notify(notifier.Alert(ticker, "не удалось снять стоп-заявку перед продажей: "+err.Error()))
					logger.ErrorContext(ctx, fmt.Sprintf("reversion: %s cancel before sell: %v", ticker, err))
					continue // без снятия продавать нельзя — двойная продажа
				}
				entry.StopOrderID = ""
				state[ticker] = entry
				_ = store.Save(state)
			}
			// ... существующий код продажи + delete(state)+Save+notify (без изменений) ...
			continue
		}

		// Синхронизация стоп-заявки (только при работающем List).
		if listErr == nil {
			if entry.StopOrderID != "" {
				if _, alive := stopByID[entry.StopOrderID]; !alive {
					s.notify(notifier.Alert(ticker, "стоп-заявка исчезла с биржи — перевыставляю"))
					entry.StopOrderID = ""
				}
			} else if stray, ok := stopByInstrument[sh.ID]; ok {
				// Чужая/устаревшая заявка (например, после reconstruct) — снять.
				if err := s.stops.Cancel(ctx, stray.StopOrderID); err != nil {
					s.notify(notifier.Alert(ticker, "не удалось снять неизвестную стоп-заявку: "+err.Error()))
				}
			}
		}

		// Желаемый уровень от ОБНОВЛЁННОГО MaxFav.
		level, reason := core.DesiredStop(mustParams(ticker), entry.EntryPrice, entry.EntryATR, entry.MaxFav)
		switch {
		case reason == "":
			// ценовые стопы выключены параметрами — нечего вести
		case entry.StopOrderID == "":
			entry = s.replaceStop(ctx, ticker, sh, entry, level, reason)
		case level > entry.StopPrice:
			if err := s.stops.Cancel(ctx, entry.StopOrderID); err != nil {
				s.notify(notifier.Alert(ticker, "не удалось снять стоп для переноса: "+err.Error()))
				break // старая заявка продолжает защищать
			}
			entry.StopOrderID = ""
			entry = s.replaceStop(ctx, ticker, sh, entry, level, reason)
		}
		state[ticker] = entry
		if err := store.Save(state); err != nil {
			return fmt.Errorf("reversion: save stop state %s: %w", ticker, err)
		}
	}
```

Хелперы в `manage.go`:

```go
// mustParams: ParamsFor гарантированно ok — тикер прошёл StrategyFor выше.
func mustParams(ticker string) core.Params {
	p, _ := ParamsFor(ticker)
	return p
}

// replaceStop places a stop at level and stamps the entry (id only when actually
// placed; price/reason always, so dry-run state mirrors what WOULD be on exchange).
func (s *service) replaceStop(ctx context.Context, ticker string, sh *imodel.Share,
	entry statestore.Entry, level float64, reason string) statestore.Entry {

	lots := entry.Quantity / int64(sh.Lot)
	res, err := s.stops.Place(ctx, sh.ID, lots, level, sh.MinPriceIncrement)
	if err != nil {
		s.notify(notifier.Alert(ticker, "стоп-заявка не выставлена: "+err.Error()))
		logger.ErrorContext(ctx, fmt.Sprintf("reversion: %s place stop: %v", ticker, err))
		return entry
	}
	if res.Placed {
		entry.StopOrderID = res.OrderID
	}
	changed := level != entry.StopPrice || reason != entry.StopReason
	entry.StopPrice, entry.StopReason = level, reason
	if changed {
		s.notify(notifier.StopSet(ticker, level, reason, !res.Placed))
	}
	return entry
}
```

Примечания:
- guard `sh.Lot <= 0` перед `lots :=` в replaceStop (тот же паттерн, что в sell-пути).
- Существующий silent-случай «позиция исчезла без StopOrderID» остаётся тихим (как сегодня).
- **Требование спеки про reconstruct** («найденную по инструменту заявку отменить, поставить свежую») реализуется здесь же, а не в пакете reconstruct: после реконструкции стейта `entry.StopOrderID == ""`, sync-ветка снимает stray-заявку инструмента через `stopByInstrument`, а общий путь ниже ставит свежую от восстановленных EntryATR/MaxFav. Пакет `reconstruct` не меняется.
- В dry-run `List` возвращает `(nil, nil)` → sync-ветки инертны, `replaceStop` обновляет только StopPrice/StopReason и шлёт StopSet при изменении уровня — уведомление не чаще реального переноса.

- [x] **Step 4: Тесты зелёные**

Run: `go test ./internal/service/trading_strategy/reversion/live/... -count=1`
Expected: PASS (новые 4 + все старые).

- [x] **Step 5: Commit**

```bash
git add internal/service/trading_strategy/reversion/live/
git commit -m "feat(reversion/live): hourly stop-order sync — repost on trail rise, fired-stop detect, no double sell"
```

---

### Task 7: Документация

**Files:**
- Modify: `docs/reversion/strategy.md`
- Modify: `docs/reversion/live-runner.md`
- Modify: `docs/reversion/live-code-map.md`

**Interfaces:** нет кода; фиксирует поведение Task 3–6.

- [x] **Step 1: strategy.md**

- В шапке и разделе «Выход»: новый приоритет `STOP(SL|TRAIL|ATRSL) → OB → RSI50 → BE → RSIOS → EMAX`; SL/TRAIL/ATRSL — одна объединённая проверка `low ≤ max(уровни)` (интрабар), исполнение по `min(уровень, open)`, причина = компонент-максимум; трейлинг-уровень считается от максимума закрытий **по предыдущий бар** (PrevMaxFav); BE/OB/RSI50/RSIOS/EMAX — по close, как раньше.
- Новый подраздел «Модель исполнения стопов»: в live этим стопам соответствует одна биржевая stop-market заявка (GTC), которую раннер перевыставляет на часовом тике при подъёме трейлинга; поэтому интрабарная модель бэктеста = модель реального исполнения.

- [x] **Step 2: live-runner.md и live-code-map.md**

- live-runner.md: раздел «Стоп-заявки» — постановка сразу после покупки, часовой sync (List/Cancel/Place), детект срабатывания (позиция исчезла + StopOrderID), правило «cancel перед любой рыночной продажей, cancel-fail блокирует продажу», частичное исполнение, поведение dry-run, бэкстоп через close-проверку ядра.
- live-code-map.md: пакет `live/stoporders`, новые поля `statestore.Entry` (StopOrderID/StopPrice/StopReason), `notifier.StopSet`, `core.DesiredStop`, `registry.ParamsFor`, обновлённая сигнатура `NewService`.

- [x] **Step 3: Commit**

```bash
git add docs/reversion/
git commit -m "docs(reversion): intrabar stop execution model and live stop-order lifecycle"
```

---

### Task 8: Гейт качества + информационный перепрогон walk-forward

**Files:**
- Create: `docs/reversion/intrabar-rerun-2026-07.md`
- Create (генерируются): `reports/UGLD/intrabar/*`, `reports/EUTR/intrabar/*`, `reports/NVTK/intrabar/*`

**Interfaces:** Consumes: всё предыдущее. Тикеры из live-реестра НЕ убираются независимо от цифр (решение пользователя в спеке).

- [x] **Step 1: Полный гейт**

Run: `./bin/mage ci`
Expected: lint OK, `go test -race ./...` PASS, mock-drift OK. Починить, если нет.

- [x] **Step 2: Перепрогон fixed walk-forward (те же окна, что в отчётах 2026-06-18/22)**

```bash
mkdir -p reports/UGLD/intrabar reports/EUTR/intrabar reports/NVTK/intrabar
go run ./cmd/backtest -ticker UGLD -strategy reversion -interval Hour1 \
  -calibrate data/params/ugld/reversion_fixed.json -out ./reports/UGLD/intrabar \
  -months 30 -train-months 12 -test-months 6 -min-trades 1 -metric profit_factor
go run ./cmd/backtest -ticker EUTR -strategy reversion -interval Hour1 \
  -calibrate data/params/eutr/reversion_fixed.json -out ./reports/EUTR/intrabar \
  -months 31 -train-months 12 -test-months 6 -min-trades 1 -metric profit_factor
go run ./cmd/backtest -ticker NVTK -strategy reversion -interval Hour1 \
  -calibrate data/params/nvtk/reversion_fixed_current.json -out ./reports/NVTK/intrabar \
  -months 36 -train-months 18 -test-months 6 -min-trades 1 -metric profit_factor
```

Ожидаемый выход: `*_walkforward.md` в каждой папке. Если раннер отвергнет единственную комбинацию из-за `-min-trades` — снизить до 0. Свечные кэши в `data/candles/` уже есть; окна сместятся на ~3 недели относительно июньских отчётов (данные до текущей даты) — это допустимо, фиксировать фактические окна в сравнении.

- [x] **Step 3: Одиночные прогоны (полная история, для журнала сделок)**

```bash
go run ./cmd/backtest -ticker UGLD -strategy reversion -interval Hour1 -months 30 -out ./reports/UGLD/intrabar
go run ./cmd/backtest -ticker EUTR -strategy reversion -interval Hour1 -months 31 -out ./reports/EUTR/intrabar
go run ./cmd/backtest -ticker NVTK -strategy reversion -interval Hour1 -months 36 -out ./reports/NVTK/intrabar
```

- [x] **Step 4: Сравнительный документ**

`docs/reversion/intrabar-rerun-2026-07.md`: таблица «close-модель (июнь) vs интрабар (сейчас)» по пулу OOS каждого тикера — PF, сделки, win rate, NetPnL%, MaxDD%, per-fold PF. Baseline из старых отчётов: UGLD PF 3.385 / 25tr / win 32.0% (reports/UGLD/fixed/...141934), EUTR PF 2.529 / 47tr / win 55.3% (reports/EUTR/...234155), NVTK PF 5.762 / 33tr / win 54.5% (reports/NVTK/result/...133935). Вывод — только констатация (тикеры остаются в live по решению пользователя); если какой-то тикер уходит в PF<1 — отдельно пометить это в документе как кандидата на перекалибровку (вне скоупа).

- [x] **Step 5: Commit**

```bash
git add docs/reversion/intrabar-rerun-2026-07.md reports/UGLD/intrabar reports/EUTR/intrabar reports/NVTK/intrabar
git commit -m "docs(reversion): intrabar walk-forward rerun — close-model vs intrabar comparison"
```

- [x] **Step 6: Финальная проверка ветки**

Run: `./bin/mage ci && git log --oneline main..HEAD`
Expected: CI зелёный; 8 коммитов по задачам. Ветку НЕ мержить — предложить пользователю ревью (skill superpowers:finishing-a-development-branch).
