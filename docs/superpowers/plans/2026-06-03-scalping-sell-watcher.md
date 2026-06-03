# Scalping Sell-Watcher Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Добавить отдельный, более частый «сторож выхода» — он шлёт уведомления о продаже по удерживаемым позициям каждые 5 минут, не меняя логику входа.

**Architecture:** Переиспользуем существующий `service.Trade`, добавив флаг `SellOnly` в `dto.Trade`. В sell-режиме `Trade` обрабатывает только тикеры с открытой позицией и оставляет лишь `SignalSell`. Второй cron-job (`*/5 8-23 * * 1-5`) поднимается в `internal/app/app.go`. Логика `Decide`/TP-SL и данные (1h-свечи с формирующимся баром, `to=now`) не меняются.

**Tech Stack:** Go 1.25, `robfig/cron/v3` (через `pkg/scheduler`), Telegram client, стандартный `testing` (табличные тесты).

**Spec:** `docs/superpowers/specs/2026-06-03-scalping-sell-watcher-design.md`

---

## File Structure

- `internal/service/trading_strategy/scalping/dto/trade.go` — добавить поле `SellOnly bool`.
- `internal/service/trading_strategy/scalping/notification/notifications.go` — добавить `SellWatch(...)` с отдельным заголовком; вынести общий рендер в `render(header, signals)`.
- `internal/service/trading_strategy/scalping/types.go` — добавить тест-seam `WithStrategies(...) Option`.
- `internal/service/trading_strategy/scalping/trade.go` — фильтрация по `SellOnly`; выбор рендера уведомления.
- `internal/service/trading_strategy/scalping/trade_test.go` — **создать**: табличные тесты `Trade` в sell-режиме + регрессия.
- `internal/app/app.go` — поднять второй worker с `SellOnly: true` и расписанием `*/5 8-23 * * 1-5`.

---

## Task 1: Notification — заголовок для режима мониторинга выхода

**Files:**
- Modify: `internal/service/trading_strategy/scalping/notification/notifications.go`
- Test: `internal/service/trading_strategy/scalping/notification/notifications_test.go`

- [ ] **Step 1: Write the failing test**

Добавить в конец `notifications_test.go`:

```go
func TestSellWatch_UsesMonitorHeaderAndRendersSells(t *testing.T) {
	signals := []model.Signal{
		{Kind: model.SignalSell, InstrumentName: "Gazprom", Ticker: "GAZP", Price: 104, TakeProfit: 103, StopLoss: 98, Reason: "TP"},
	}

	got := SellWatch(signals)

	for _, want := range []string{"Мониторинг выхода", "продажу", "Gazprom", "GAZP", "TP"} {
		if !strings.Contains(got, want) {
			t.Errorf("message missing %q\n---\n%s", want, got)
		}
	}
	if strings.Contains(got, "Скальпинг (1H)") {
		t.Errorf("sell-watch must not use the hourly header\n---\n%s", got)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/service/trading_strategy/scalping/notification/ -run TestSellWatch -v`
Expected: FAIL — `undefined: SellWatch` (compile error).

- [ ] **Step 3: Implement minimal code**

В `notifications.go` заменить функцию `Trade` так, чтобы общий рендер вынести в `render`, и добавить `SellWatch`. Итоговое содержимое функций:

```go
// Trade renders an aggregated HTML Telegram message for the hourly run.
func Trade(signals []model.Signal) string {
	return render("⚡️ <b>Скальпинг (1H)</b>\n\n", signals)
}

// SellWatch renders the same body under the out-of-schedule exit-monitor header.
func SellWatch(signals []model.Signal) string {
	return render("⚠️ <b>Мониторинг выхода (1H)</b>\n\n", signals)
}

func render(header string, signals []model.Signal) string {
	var buys, sells []model.Signal
	for _, s := range signals {
		switch s.Kind {
		case model.SignalBuy:
			buys = append(buys, s)
		case model.SignalSell:
			sells = append(sells, s)
		}
	}

	b := strings.Builder{}
	b.WriteString(header)

	if len(buys) > 0 {
		b.WriteString("<u><b>Сигналы на покупку:</b></u>\n")
		for _, s := range buys {
			b.WriteString(fmt.Sprintf(
				"🟢 <b>%s</b> (%s)\n  Цена: %.2f | TP: %.2f | SL: %.2f | RSI: %.0f\n",
				s.InstrumentName, s.Ticker, s.Price, s.TakeProfit, s.StopLoss, s.RSI,
			))
		}
		b.WriteString("\n")
	}

	if len(sells) > 0 {
		b.WriteString("<u><b>Сигналы на продажу:</b></u>\n")
		for _, s := range sells {
			b.WriteString(fmt.Sprintf(
				"🔴 <b>%s</b> (%s) [%s]\n  Цена: %.2f | TP: %.2f | SL: %.2f\n",
				s.InstrumentName, s.Ticker, s.Reason, s.Price, s.TakeProfit, s.StopLoss,
			))
		}
	}

	return b.String()
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/service/trading_strategy/scalping/notification/ -v`
Expected: PASS — новый тест и существующие (`TestTrade_*`) зелёные.

- [ ] **Step 5: Commit**

```bash
git add internal/service/trading_strategy/scalping/notification/notifications.go internal/service/trading_strategy/scalping/notification/notifications_test.go
git commit -m "feat(scalping): exit-monitor notification header"
```

---

## Task 2: `dto.SellOnly` + тест-seam `WithStrategies`

Подготовительные изменения, чтобы Task 3 можно было покрыть тестами. Поведения ещё не добавляем.

**Files:**
- Modify: `internal/service/trading_strategy/scalping/dto/trade.go`
- Modify: `internal/service/trading_strategy/scalping/types.go`

- [ ] **Step 1: Добавить поле `SellOnly`**

Заменить содержимое `dto/trade.go`:

```go
package dto

// Trade carries per-run parameters for the scalping strategy.
type Trade struct {
	Scheduler string
	// SellOnly turns the run into an out-of-schedule exit watcher: it processes
	// only instruments currently held and emits only Sell signals.
	SellOnly bool
}
```

- [ ] **Step 2: Добавить Option `WithStrategies`**

В `types.go` после функции `WithSettings` добавить:

```go
// WithStrategies overrides the per-share strategy set (used in tests).
func WithStrategies(s []strategy.Strategy) Option {
	return func(svc *service) {
		svc.strategies = s
	}
}
```

(`strategy` уже импортирован в `types.go`.)

- [ ] **Step 3: Verify build**

Run: `go build ./...`
Expected: успех, без ошибок.

- [ ] **Step 4: Commit**

```bash
git add internal/service/trading_strategy/scalping/dto/trade.go internal/service/trading_strategy/scalping/types.go
git commit -m "feat(scalping): SellOnly flag and WithStrategies test seam"
```

---

## Task 3: Фильтрация `Trade` в sell-режиме + выбор рендера

**Files:**
- Modify: `internal/service/trading_strategy/scalping/trade.go`
- Test: `internal/service/trading_strategy/scalping/trade_test.go` (создать)

- [ ] **Step 1: Write the failing tests**

Создать `internal/service/trading_strategy/scalping/trade_test.go`:

```go
package scalping

import (
	"context"
	"strings"
	"testing"

	"github.com/golang/protobuf/ptypes/timestamp"

	imodel "tinvest/internal/model"
	"tinvest/internal/service/trading_strategy/scalping/dto"
	"tinvest/internal/service/trading_strategy/scalping/model"
	"tinvest/internal/service/trading_strategy/scalping/strategy"
	grpcmodel "tinvest/pkg/client/grpc/model"
)

// --- fakes ---

type fakeStrategy struct {
	ticker   string
	lookback int
	sig      model.Signal
}

func (f fakeStrategy) Ticker() string                              { return f.ticker }
func (f fakeStrategy) Lookback() int                               { return f.lookback }
func (f fakeStrategy) Decide(strategy.MarketData) model.Signal     { return f.sig }

type stubInstruments struct{ shares []*imodel.Share }

func (s stubInstruments) Shares(context.Context) ([]*imodel.Share, error) { return s.shares, nil }

type stubMarket struct {
	called  int
	candles []*imodel.CandleItemTechAnalyse
}

func (m *stubMarket) GetCandles(_ context.Context, _ *string, _ int32, _ *timestamp.Timestamp, _ *timestamp.Timestamp, _ *int32, _ bool) ([]*imodel.CandleItemTechAnalyse, error) {
	m.called++
	return m.candles, nil
}

type stubOps struct{ positions []*grpcmodel.Position }

func (o stubOps) GetPortfolio(context.Context, string) ([]*grpcmodel.Position, error) {
	return o.positions, nil
}

type stubTg struct{ msgs []string }

func (t *stubTg) SendMessage(msg string) error              { t.msgs = append(t.msgs, msg); return nil }
func (t *stubTg) SendMessageToChat(int64, string) error     { return nil }

// share with one tradable instrument matching the fake strategy ticker.
func tradableShare() *imodel.Share {
	return &imodel.Share{ID: "share-1", Ticker: "TEST", Name: "Test Co", Trading: true, Lot: 1}
}

func heldPosition() *grpcmodel.Position {
	return &grpcmodel.Position{InstrumentType: "share", ShareID: "share-1", Quantity: 10}
}

func oneCandle() []*imodel.CandleItemTechAnalyse {
	return []*imodel.CandleItemTechAnalyse{{
		Open:  imodel.Quotation{Units: 100},
		Close: imodel.Quotation{Units: 100},
		Low:   imodel.Quotation{Units: 99},
		High:  imodel.Quotation{Units: 101},
	}}
}

func TestTrade_SellOnly(t *testing.T) {
	tests := []struct {
		name         string
		sellOnly     bool
		positions    []*grpcmodel.Position
		sig          model.Signal
		wantMsgs     int
		wantFetched  bool
		wantContains string
	}{
		{
			name:         "held position with sell signal sends one alert",
			sellOnly:     true,
			positions:    []*grpcmodel.Position{heldPosition()},
			sig:          model.Signal{Kind: model.SignalSell, Reason: "TP"},
			wantMsgs:     1,
			wantFetched:  true,
			wantContains: "Мониторинг выхода",
		},
		{
			name:        "held position without signal stays silent",
			sellOnly:    true,
			positions:   []*grpcmodel.Position{heldPosition()},
			sig:         model.Signal{Kind: model.SignalNone},
			wantMsgs:    0,
			wantFetched: true,
		},
		{
			name:        "no position skips instrument before fetching candles",
			sellOnly:    true,
			positions:   nil,
			sig:         model.Signal{Kind: model.SignalSell, Reason: "TP"},
			wantMsgs:    0,
			wantFetched: false,
		},
		{
			name:        "buy signal is ignored in sell-only mode",
			sellOnly:    true,
			positions:   []*grpcmodel.Position{heldPosition()},
			sig:         model.Signal{Kind: model.SignalBuy},
			wantMsgs:    0,
			wantFetched: true,
		},
		{
			name:         "default mode still emits buy without a position",
			sellOnly:     false,
			positions:    nil,
			sig:          model.Signal{Kind: model.SignalBuy},
			wantMsgs:     1,
			wantFetched:  true,
			wantContains: "Скальпинг (1H)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			market := &stubMarket{candles: oneCandle()}
			tg := &stubTg{}
			svc := NewService(
				stubInstruments{shares: []*imodel.Share{tradableShare()}},
				market,
				stubOps{positions: tt.positions},
				tg,
				"acc-1",
				WithStrategies([]strategy.Strategy{fakeStrategy{ticker: "TEST", lookback: 1, sig: tt.sig}}),
			)

			if err := svc.Trade(context.Background(), dto.Trade{SellOnly: tt.sellOnly}); err != nil {
				t.Fatalf("Trade returned error: %v", err)
			}

			if len(tg.msgs) != tt.wantMsgs {
				t.Errorf("messages = %d, want %d (msgs=%v)", len(tg.msgs), tt.wantMsgs, tg.msgs)
			}
			if (market.called > 0) != tt.wantFetched {
				t.Errorf("candles fetched = %v, want %v (called=%d)", market.called > 0, tt.wantFetched, market.called)
			}
			if tt.wantContains != "" && (len(tg.msgs) == 0 || !strings.Contains(tg.msgs[0], tt.wantContains)) {
				t.Errorf("message missing %q (msgs=%v)", tt.wantContains, tg.msgs)
			}
		})
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/service/trading_strategy/scalping/ -run TestTrade_SellOnly -v`
Expected: FAIL — в sell-режиме сейчас уведомления уходят без фильтра (кейсы «no position…», «buy signal is ignored…» и заголовок «Мониторинг выхода» не выполняются).

- [ ] **Step 3: Implement the filtering**

В `trade.go`, внутри цикла `for _, st := range s.strategies {`, сразу после блока проверки `byTicker`/`!sh.Trading` и строки `id := sh.ID`, добавить ранний пропуск не-удерживаемых бумаг в sell-режиме (до `time.Sleep`/`GetCandles`):

```go
		id := sh.ID
		if in.SellOnly {
			if _, held := posByID[id]; !held {
				continue
			}
		}
		lookback := st.Lookback()
```

Далее, после строки `sig := st.Decide(md)` и существующей проверки `SignalNone`, добавить фильтр buy-сигналов:

```go
		sig := st.Decide(md)
		if sig.Kind == model.SignalNone {
			continue
		}
		if in.SellOnly && sig.Kind != model.SignalSell {
			continue
		}
```

Наконец, заменить отправку сообщения (выбор рендера по режиму). Было:

```go
	if err := s.tgClient.SendMessage(notification.Trade(signals)); err != nil {
		return fmt.Errorf("scalping: send message: %w", err)
	}
```

Стало:

```go
	msg := notification.Trade(signals)
	if in.SellOnly {
		msg = notification.SellWatch(signals)
	}
	if err := s.tgClient.SendMessage(msg); err != nil {
		return fmt.Errorf("scalping: send message: %w", err)
	}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/service/trading_strategy/scalping/... -v`
Expected: PASS — `TestTrade_SellOnly` (все подкейсы) и существующие тесты (`notification`, `rusal`) зелёные.

- [ ] **Step 5: Commit**

```bash
git add internal/service/trading_strategy/scalping/trade.go internal/service/trading_strategy/scalping/trade_test.go
git commit -m "feat(scalping): sell-only filtering in Trade with exit-monitor message"
```

---

## Task 4: Поднять второй worker в `app.go`

**Files:**
- Modify: `internal/app/app.go`

- [ ] **Step 1: Добавить sell-watch worker в `runDev`**

В `runDev`, сразу после существующего scalping-блока (горутина с `Scheduler: "0 8-23 * * 1-5"`, около строк 203–211), добавить:

```go
	wg.Add(1)
	go func() {
		defer wg.Done()
		err := a.sp.GetScalpingTradingService().Trade(ctx, scalpingdto.Trade{
			Scheduler: "*/5 8-23 * * 1-5",
			SellOnly:  true,
		})
		if err != nil {
			logger.ErrorContext(ctx, "Error in worker Scalping sell-watch", err.Error())
		}
	}()
```

(В dev раннер вызывает сервис напрямую — прогон один раз, удобно для smoke-теста.)

- [ ] **Step 2: Добавить sell-watch cron-job в `runProd`**

В блок импортов `app.go` добавить:

```go
	scalpingscheduler "tinvest/internal/service/trading_strategy/scalping/scheduler"
```

В `runProd` увеличить счётчик: заменить `wg.Add(6)` на `wg.Add(7)`. Затем добавить активную горутину (вне закомментированных блоков, перед `wg.Wait()`):

```go
	go func() {
		defer wg.Done()
		err := scalpingscheduler.NewSchedulerService(a.sp.GetScalpingTradingService()).Trade(
			ctx,
			scalpingdto.Trade{
				Scheduler: "*/5 8-23 * * 1-5",
				SellOnly:  true,
			},
		)
		if err != nil {
			logger.ErrorContext(ctx, "Error in worker Scalping sell-watch", err.Error())
		}
	}()
```

- [ ] **Step 3: Verify build and vet**

Run: `go build ./... && go vet ./internal/app/...`
Expected: успех. (Проверка: число активных горутин в `runProd` совпадает с `wg.Add(7)` — добавлена ровно одна активная горутина.)

- [ ] **Step 4: Commit**

```bash
git add internal/app/app.go
git commit -m "feat(app): wire scalping sell-watcher (*/5 cron, SellOnly)"
```

---

## Final Verification

- [ ] Run: `go test ./internal/service/trading_strategy/scalping/...` → PASS
- [ ] Run: `go build ./...` → успех
- [ ] Run: `go vet ./internal/service/trading_strategy/scalping/... ./internal/app/...` → без замечаний
