# rsi_pullback: стоп по закрытию бара в live-раннере — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** У 21 тикера rsi_pullback live-раннер держит на бирже только аварийный стоп (`CATSL`), а защитный уровень (SL/трейл) проверяет по закрытию бара и продаёт рыночной заявкой (`SLC`/`TRAILC`).

**Architecture:** Бэктестная часть уже есть (коммит a8957e51: `core.Params.StopOnClose/CatStopMult`, `core.CatastropheStop`, причины `SLC`/`TRAILC`/`CATSL`). Live меняется в двух местах: адаптер pullback отдаёт раннеру уровень биржевой заявки через новую чистую функцию `exchangeStop` (аварийный уровень при `StopOnClose=1`), а `pass.go:manage` форсирует перевыставление заявки при смене режима на открытой позиции. Выход `SLC`/`TRAILC` идёт существующим путём `sell`, нового кода не требует. Затем литералы 21 тикера, сторожевые тесты, сквозные тесты раннера и документация.

**Tech Stack:** Go 1.25, testify/mock (mockery v2), `./bin/mage ci`.

**Spec:** `docs/superpowers/specs/2026-10-13-pullback-stop-on-close-live-design.md`

Рабочая копия: worktree `/home/oleg/GolandProjects/tinvest-stoponclose`, ветка `feat/stop-on-close`. Все пути ниже — от корня этого worktree. Тесты гонять на ограниченном CPU: `GOMAXPROCS=2 nice -n 19 go test ...`.

## Global Constraints

- Режим получают ровно 21 тикер: AQUA, BSPB, CNRU, DIAS, ELFV, FESH, GAZP, IVAT, LENT, LSNGP, MDMG, SFIN, SPBE, SVAV, SVCB, T, TGKA, VSMO, WUSH, X5, YDEX — значения `StopOnClose: 1, CatStopMult: 2`.
- Не получают (литерал не трогать): UGLD, MVID, RAGR, NVTK, RENI, SOFL, SNGSP, BANEP, ASTR, NKHP, IRKT, MAGN, SIBN, DOMRF.
- Причина биржевой заявки в режиме — строка `"CATSL"` (та же, что в ядре и `model.IsStopReason`).
- rsi_zone и gap_fade (гости раннера) не меняются; тест `rsi_zone/live/stop_on_close_guard_test.go` не трогать.
- `docs/rsi_pullback/*.md` — только механика: никаких PF, списков тикеров и результатов замера (CLAUDE.md, «Documentation rules»).
- Комментарии в коде — по-русски, в стиле окружающего кода; сообщения коммитов — английские Conventional Commits с трейлером `Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>`.
- `go build ./...` падает на `magefiles` — собирать `go build ./internal/... ./pkg/... ./cmd/...`.

## Review Focus

- Литерал переключили при открытой позиции (на бирже `SL`, литерал требует `CATSL`, и обратно) — заявка обязана перевыставиться в нужном режиме, а не остаться старой (Task 2).
- Аварийная заявка сработала внутри бара, а портфель брокера ещё показывает позицию — никакой второй (рыночной) продажи, только уведомление о выходе `CATSL` (Task 4).
- Позиция без стейта у тикера в режиме — реконструкция ставит аварийную заявку, а не внутрибарный SL (Task 4).
- Трейл-тикер в режиме (T): рост `MaxFav` не двигает биржевую заявку (Task 1).
- Бар проколол защитный уровень тенью, но закрылся выше — никакой продажи (именно ради этого режим; Task 4).

---

### Task 1: Уровень биржевой заявки в адаптере pullback

**Files:**
- Modify: `internal/service/trading_strategy/rsi_pullback/live/adapter.go`
- Create: `internal/service/trading_strategy/rsi_pullback/live/adapter_test.go`

**Interfaces:**
- Produces: `const catReason = "CATSL"` и `func exchangeStop(p core.Params, e statestore.Entry) (float64, string)` в пакете `live` (Task 2 использует `catReason`).

- [ ] **Step 1: Написать падающий тест**

`internal/service/trading_strategy/rsi_pullback/live/adapter_test.go`:

```go
package live

import (
	"math"
	"testing"

	"tinvest/internal/service/trading_strategy/livecore/statestore"
	"tinvest/internal/service/trading_strategy/rsi_pullback/strategy/core"
)

func closeStopParams() core.Params {
	p := core.DefaultParams()
	p.StopDailyATR = 0.7
	p.StopOnClose, p.CatStopMult = 1, 2
	return p
}

// Без режима биржевая заявка — прежний защитный уровень ядра.
func TestExchangeStopWithoutCloseModeIsTheProtectiveLevel(t *testing.T) {
	p := core.DefaultParams()
	p.StopDailyATR = 0.7
	e := statestore.Entry{EntryPrice: 100, EntryATR: 10, MaxFav: 100}
	gotLevel, gotReason := exchangeStop(p, e)
	wantLevel, wantReason := core.DesiredStop(p, 100, 10, 100)
	if gotLevel != wantLevel || gotReason != wantReason {
		t.Fatalf("exchangeStop = %v/%q, want %v/%q", gotLevel, gotReason, wantLevel, wantReason)
	}
}

// В режиме на бирже стоит только аварийный стоп: вход − CatStopMult×StopDailyATR×ATR.
func TestExchangeStopInCloseModeIsTheCatastropheStop(t *testing.T) {
	p := closeStopParams()
	level, reason := exchangeStop(p, statestore.Entry{EntryPrice: 100, EntryATR: 10, MaxFav: 100})
	if reason != "CATSL" || math.Abs(level-86) > 1e-9 {
		t.Fatalf("exchangeStop = %v/%q, want 86/CATSL", level, reason)
	}
}

// Трейл в режиме проверяется ядром по закрытию; биржевую заявку он не двигает.
func TestExchangeStopInCloseModeIgnoresTheTrail(t *testing.T) {
	p := closeStopParams()
	p.UseTrail, p.TrailDailyATR = 1, 0.5
	low, _ := exchangeStop(p, statestore.Entry{EntryPrice: 100, EntryATR: 10, MaxFav: 100})
	high, reason := exchangeStop(p, statestore.Entry{EntryPrice: 100, EntryATR: 10, MaxFav: 130})
	if low != high || reason != "CATSL" {
		t.Fatalf("уровень при MaxFav 100 = %v, при 130 = %v/%q — трейл двигает аварийную заявку", low, high, reason)
	}
}

// Режим без аварийного множителя — заявки нет (такой литерал не пропустит сторожевой тест).
func TestExchangeStopInCloseModeWithoutCatHasNoOrder(t *testing.T) {
	p := closeStopParams()
	p.CatStopMult = 0
	if level, reason := exchangeStop(p, statestore.Entry{EntryPrice: 100, EntryATR: 10, MaxFav: 100}); level != 0 || reason != "" {
		t.Fatalf("exchangeStop = %v/%q, want 0/\"\"", level, reason)
	}
}
```

- [ ] **Step 2: Убедиться, что тест падает**

Run: `cd /home/oleg/GolandProjects/tinvest-stoponclose && GOMAXPROCS=2 go test ./internal/service/trading_strategy/rsi_pullback/live/ -run TestExchangeStop`
Expected: FAIL компиляции — `undefined: exchangeStop`.

- [ ] **Step 3: Реализация**

В `adapter.go` заменить метод `DesiredStop` и добавить константу с функцией (после метода):

```go
// DesiredStop — уровень БИРЖЕВОЙ заявки (контракт adapter.Strategy), см. exchangeStop.
func (p *pullbackStrategy) DesiredStop(ticker string, e statestore.Entry) (float64, string) {
	return exchangeStop(mustParams(ticker), e)
}

// catReason — причина аварийной биржевой заявки, та же, что у ядра (core: CATSL).
const catReason = "CATSL"

// exchangeStop — уровень, который раннер держит на бирже. Обычно это защитный уровень ядра
// (SL или трейл, core.DesiredStop). У литерала со StopOnClose=1 защитный уровень проверяет
// ядро по закрытию бара (SLC/TRAILC, продажа рыночной заявкой на проходе), а на бирже стоит
// только аварийный стоп core.CatastropheStop: он заморожен на входе, и трейл его не двигает.
// Режим без аварийного уровня — заявки нет (reason ""); сторожевой тест реестра такой литерал
// в бой не пускает.
func exchangeStop(p core.Params, e statestore.Entry) (float64, string) {
	if p.StopOnClose == 1 {
		level := core.CatastropheStop(p, e.EntryPrice, e.EntryATR)
		if level <= 0 {
			return 0, ""
		}
		return level, catReason
	}
	return core.DesiredStop(p, e.EntryPrice, e.EntryATR, e.MaxFav)
}
```

- [ ] **Step 4: Тесты проходят**

Run: `GOMAXPROCS=2 go test ./internal/service/trading_strategy/rsi_pullback/live/`
Expected: PASS (весь пакет: ни один литерал ещё не в режиме, поведение прочих тестов не изменилось).

- [ ] **Step 5: Commit**

```bash
git add internal/service/trading_strategy/rsi_pullback/live/adapter.go internal/service/trading_strategy/rsi_pullback/live/adapter_test.go
git commit -m "feat(pullback/live): exchange stop is the catastrophe stop under StopOnClose

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

---

### Task 2: Перевыставление заявки при смене режима стопа

**Files:**
- Modify: `internal/service/trading_strategy/rsi_pullback/live/pass.go` (switch в `manage`, ~строки 644–670)
- Modify: `internal/service/trading_strategy/rsi_pullback/live/shared_test.go` (поле `stopReason` у `fakeStrategy`, три теста в конец файла)

**Interfaces:**
- Consumes: `catReason` из Task 1.
- Produces: `func catModeChanged(placed, wanted string) bool` в `pass.go`.

- [ ] **Step 1: Причина стопа у фейковой стратегии**

В `shared_test.go` в struct `fakeStrategy` после `stopLevel  float64` добавить поле:

```go
	stopReason string // причина DesiredStop; "" — "SL"
```

и заменить метод `DesiredStop`:

```go
func (f *fakeStrategy) DesiredStop(string, statestore.Entry) (float64, string) {
	if f.stopLevel <= 0 {
		return 0, ""
	}
	if f.stopReason != "" {
		return f.stopLevel, f.stopReason
	}
	return f.stopLevel, "SL"
}
```

- [ ] **Step 2: Написать падающие тесты** (в конец `shared_test.go`)

```go
// modeSwitchEnv — открытая позиция гостя на GAZP с живой биржевой заявкой so-1 (уровень
// placed, причина placedReason); гость теперь хочет заявку wantLevel/wantReason. Обе стратегии
// боевые, чтобы заявки шли в мок, а не в dry-run.
func modeSwitchEnv(t *testing.T, placed float64, placedReason string, wantLevel float64, wantReason string) (*env, *fakeStrategy) {
	t.Helper()
	zone := zoneFake(model.SignalNone, "GAZP")
	zone.trade = true
	zone.stopLevel, zone.stopReason = wantLevel, wantReason
	e := newSharedEnv(t, sharedNow, []string{"GAZP"}, func(c *config.RSIPullbackConfig) { c.TradeEnabled = true }, zone)
	e.seed(t, statestore.Entry{Ticker: "GAZP", Strategy: "rsi_zone", EntryPrice: 100,
		EntryATR: 10, MaxFav: 100, Quantity: 100, EntryTime: sharedNow.Add(-48 * time.Hour),
		StopOrderID: "so-1", StopPrice: placed, StopReason: placedReason})
	e.instruments.EXPECT().Shares(mock.Anything).Return(sharesOf("GAZP"), nil)
	e.expectCandles(flatAt30m(sharedLastBar, 400, 100), dailies(sharedNow, 60))
	e.ops.EXPECT().GetPortfolio(mock.Anything, mock.Anything).Return([]*grpcmodel.Position{held("GAZP", 100)}, nil)
	e.stops.EXPECT().GetStopOrders(mock.Anything, mock.MatchedBy(statusIs(investapi.StopOrderStatusOption_STOP_ORDER_STATUS_ACTIVE))).
		Return(activeList("uid-GAZP", "so-1", placed, 10), nil)
	e.tg.EXPECT().SendMessage(mock.Anything).Return(nil).Maybe()
	return e, zone
}

func expectRepost(e *env, level int64) {
	e.stops.EXPECT().CancelStopOrder(mock.Anything, mock.MatchedBy(func(in *investapi.CancelStopOrderRequest) bool {
		return in.GetStopOrderId() == "so-1"
	})).Return(&investapi.CancelStopOrderResponse{}, nil).Once()
	e.stops.EXPECT().PostStopOrder(mock.Anything, mock.MatchedBy(func(in *investapi.PostStopOrderRequest) bool {
		return in.GetStopPrice().GetUnits() == level
	})).Return(&investapi.PostStopOrderResponse{StopOrderId: "so-2"}, nil).Once()
}

// Литерал перевели в режим стопа по закрытию при открытой позиции: на бирже стоит SL 95,
// нужен аварийный 80 — НИЖЕ. Обычное правило «только вверх» оставило бы тесную внутрибарную
// заявку, о которой бэктест режима не знает. Заявка обязана перевыставиться.
func TestStopModeSwitchToCatastropheReposts(t *testing.T) {
	e, _ := modeSwitchEnv(t, 95, "SL", 80, "CATSL")
	expectRepost(e, 80)
	if err := e.svc.Run(context.Background(), dto.Run{}); err != nil {
		t.Fatalf("Run: %v", err)
	}
	got := e.state(t)["GAZP"]
	if got.StopOrderID != "so-2" || got.StopPrice != 80 || got.StopReason != "CATSL" {
		t.Fatalf("заявка после смены режима: %q/%v/%q, want so-2/80/CATSL", got.StopOrderID, got.StopPrice, got.StopReason)
	}
}

// Обратное переключение: на бирже аварийный 80, нужен SL 70 — тоже ниже; перевыставить.
func TestStopModeSwitchFromCatastropheReposts(t *testing.T) {
	e, _ := modeSwitchEnv(t, 80, "CATSL", 70, "SL")
	expectRepost(e, 70)
	if err := e.svc.Run(context.Background(), dto.Run{}); err != nil {
		t.Fatalf("Run: %v", err)
	}
	if got := e.state(t)["GAZP"]; got.StopOrderID != "so-2" || got.StopReason != "SL" {
		t.Fatalf("заявка после смены режима: %q/%q, want so-2/SL", got.StopOrderID, got.StopReason)
	}
}

// Контроль: режим тот же, уровень ниже — заявку не трогаем (стоп только подтягивается).
func TestLowerStopInTheSameModeIsNotReposted(t *testing.T) {
	e, _ := modeSwitchEnv(t, 95, "SL", 80, "SL")
	if err := e.svc.Run(context.Background(), dto.Run{}); err != nil {
		t.Fatalf("Run: %v", err)
	}
	e.stops.AssertNotCalled(t, "CancelStopOrder", mock.Anything, mock.Anything)
	e.stops.AssertNotCalled(t, "PostStopOrder", mock.Anything, mock.Anything)
}
```

Если `investapi` ещё не импортирован в `shared_test.go`, он уже есть в списке импортов (`investapi "tinvest/internal/pb/v1"`) — проверить и не дублировать.

- [ ] **Step 3: Убедиться, что падают**

Run: `GOMAXPROCS=2 go test ./internal/service/trading_strategy/rsi_pullback/live/ -run 'TestStopModeSwitch|TestLowerStopInTheSameMode' -v`
Expected: `TestStopModeSwitchToCatastropheReposts` и `TestStopModeSwitchFromCatastropheReposts` FAIL (мок не получил `CancelStopOrder`), контрольный PASS.

- [ ] **Step 4: Реализация в `pass.go`**

В `manage` заменить строку

```go
	case sizeMismatch, desired > current:
```

на

```go
	case sizeMismatch, desired > current, catModeChanged(entry.StopReason, reason):
```

и добавить функцию после `manage` (перед комментарием к `sell`):

```go
// catModeChanged — заявка выставлена в одном режиме стопа, а литерал требует другого: на бирже
// внутрибарный SL/TRAIL, а тикер перевели на стоп по закрытию с аварийной заявкой (или
// наоборот). Аварийный уровень ниже защитного, и правило «переносить только вверх» оставило
// бы на бирже заявку не того режима — позиция шла бы со стопом, которого нет в бэктесте её
// литерала. Пустая причина в стейте (запись старше этого поля) режимом не считается.
func catModeChanged(placed, wanted string) bool {
	return placed != "" && wanted != "" && (placed == catReason) != (wanted == catReason)
}
```

- [ ] **Step 5: Тесты проходят**

Run: `GOMAXPROCS=2 go test ./internal/service/trading_strategy/rsi_pullback/live/`
Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add internal/service/trading_strategy/rsi_pullback/live/pass.go internal/service/trading_strategy/rsi_pullback/live/shared_test.go
git commit -m "fix(pullback/live): repost the exchange stop when the stop mode changes

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

---

### Task 3: Литералы 21 тикера и сторожевые тесты

**Files:**
- Modify: `internal/service/trading_strategy/rsi_pullback/strategy/<ticker>/<ticker>.go` для 21 тикера (пакет T — `strategy/tbank/tbank.go`, X5 — `strategy/x5/x5.go`; прочие — имя тикера строчными)
- Modify: `internal/service/trading_strategy/rsi_pullback/strategy/<ticker>/<ticker>_test.go` — там, где тест пинит литерал целиком
- Modify: `internal/service/trading_strategy/rsi_pullback/live/stop_on_close_guard_test.go`
- Modify: `internal/service/trading_strategy/rsi_pullback/live/service_test.go` (`TestMain`, `realParams`)

**Interfaces:**
- Produces: `var realParams map[string]core.Params` (реестр как в бою) и подмена GAZP на внутрибарную копию в `TestMain` пакета `live` — Task 4 возвращает GAZP в боевой вид хелпером на основе `realParams`.

- [ ] **Step 1: Сторожевые тесты (падающие)**

Заменить содержимое `internal/service/trading_strategy/rsi_pullback/live/stop_on_close_guard_test.go`:

```go
package live

import (
	"sort"
	"testing"

	"tinvest/internal/service/trading_strategy/rsi_pullback/strategy/core"
)

// closeStopTickers — тикеры, торгующие стопом по закрытию бара с аварийной биржевой заявкой
// (спека docs/superpowers/specs/2026-10-13-pullback-stop-on-close-live-design.md). Список
// пинится, чтобы режим не включился и не выключился у тикера молча, правкой одного литерала.
var closeStopTickers = []string{
	"AQUA", "BSPB", "CNRU", "DIAS", "ELFV", "FESH", "GAZP", "IVAT", "LENT", "LSNGP", "MDMG",
	"SFIN", "SPBE", "SVAV", "SVCB", "T", "TGKA", "VSMO", "WUSH", "X5", "YDEX",
}

// TestCloseStopLiteralsKeepAnExchangeStop: в режиме StopOnClose на бирже стоит только аварийная
// заявка. Литерал, у которого она не получается (CatStopMult или StopDailyATR не положительны),
// оставил бы позицию между закрытиями баров вовсе без биржевой защиты. CatStopMult без режима —
// мёртвое поле, которое вводит в заблуждение.
func TestCloseStopLiteralsKeepAnExchangeStop(t *testing.T) {
	for ticker, p := range realParams {
		switch {
		case p.StopOnClose == 1 && core.CatastropheStop(p, 100, 10) <= 0:
			t.Errorf("%s: StopOnClose=1, но аварийного уровня нет (CatStopMult=%v, StopDailyATR=%v)", ticker, p.CatStopMult, p.StopDailyATR)
		case p.StopOnClose != 1 && p.CatStopMult != 0:
			t.Errorf("%s: CatStopMult=%v без StopOnClose=1", ticker, p.CatStopMult)
		case p.StopOnClose != 0 && p.StopOnClose != 1:
			t.Errorf("%s: StopOnClose=%d, want 0 или 1", ticker, p.StopOnClose)
		}
	}
}

// TestCloseStopUniverseIsPinned: режим включён ровно у тикеров closeStopTickers.
func TestCloseStopUniverseIsPinned(t *testing.T) {
	var got []string
	for ticker, p := range realParams {
		if p.StopOnClose == 1 {
			got = append(got, ticker)
		}
	}
	sort.Strings(got)
	want := append([]string(nil), closeStopTickers...)
	sort.Strings(want)
	if len(got) != len(want) {
		t.Fatalf("режим StopOnClose у %v, want %v", got, want)
	}
	for i := range got {
		if got[i] != want[i] {
			t.Fatalf("режим StopOnClose у %v, want %v", got, want)
		}
	}
}
```

- [ ] **Step 2: `realParams` и подмена GAZP в `TestMain`**

В `service_test.go` заменить `TestMain`:

```go
// realParams — реестр в боевом виде, снятый до подмены ниже; по нему сторожевые тесты судят
// настоящие литералы.
var realParams map[string]core.Params

func TestMain(m *testing.M) {
	logger.Init()
	realParams = maps.Clone(paramsByTicker)
	// Фикстуры этого пакета построены на GAZP и пинят механику ВНУТРИБАРНОГО стопа (заявка на
	// защитном уровне, подтяжка трейла, продажа по SL). Боевой GAZP торгует стопом по
	// закрытию — для механических тестов он подменён копией без режима; тесты режима
	// возвращают боевой литерал хелпером withRealGAZP.
	p := paramsByTicker[gazp.Ticker]
	p.StopOnClose, p.CatStopMult = 0, 0
	paramsByTicker[gazp.Ticker] = p
	os.Exit(m.Run())
}
```

Добавить в импорты `service_test.go` `"maps"` и `"tinvest/internal/service/trading_strategy/rsi_pullback/strategy/core"`, если их нет (`gazp` уже импортирован — им считается `gazpStopFromSignal`).

`gazpStopFromSignal`/`gazpStopFromFill` считаются от `gazp.DefaultParams().StopDailyATR` — поле режим не меняет, оставить как есть.

- [ ] **Step 3: Убедиться, что сторожевые тесты падают**

Run: `GOMAXPROCS=2 go test ./internal/service/trading_strategy/rsi_pullback/live/ -run 'TestCloseStop' -v`
Expected: `TestCloseStopUniverseIsPinned` FAIL (режим ни у кого), `TestCloseStopLiteralsKeepAnExchangeStop` PASS.

- [ ] **Step 4: Литералы**

В `DefaultParams()` каждого из 21 пакета добавить в литерал `core.Params{...}` последними строками:

```go
		StopOnClose:     1,
		CatStopMult:     2,
```

(выравнивание — как у соседних полей; `gofmt` поправит). У TGKA и CNRU `DefaultParams()` сейчас `return core.DefaultParams()` — заменить на:

```go
func DefaultParams() core.Params {
	p := core.DefaultParams()
	p.StopOnClose, p.CatStopMult = 1, 2 // стоп по закрытию бара, 2026-10-13 (см. doc пакета)
	return p
}
```

В doc-комментарий каждого из 21 пакета (в конец комментария перед `package`) добавить абзац:

```go
//
// 2026-10-13: стоп по закрытию бара с аварийной биржевой заявкой (StopOnClose=1, CatStopMult=2).
// Замер на этом литерале, 24 месяца, in-sample: PF <base> → <cat2> (<n> сделок).
```

с числами из таблицы:

| Тикер | base | cat2 | сделок |
|---|---|---|---|
| AQUA | 1.76 | 2.47 | 120 |
| BSPB | 2.12 | 2.38 | 43 |
| CNRU | 1.91 | 2.65 | 118 |
| DIAS | 2.63 | 2.80 | 108 |
| ELFV | 2.85 | 4.50 | 62 |
| FESH | 4.25 | 8.57 | 52 |
| GAZP | 2.61 | 3.96 | 42 |
| IVAT | 2.05 | 2.32 | 107 |
| LENT | 2.65 | 3.03 | 113 |
| LSNGP | 2.92 | 3.53 | 264 |
| MDMG | 2.00 | 2.35 | 56 |
| SFIN | 1.54 | 1.94 | 51 |
| SPBE | 1.64 | 1.83 | 59 |
| SVAV | 3.01 | 3.92 | 90 |
| SVCB | 1.39 | 1.71 | 72 |
| T | 2.85 | 2.93 | 30 |
| TGKA | 2.21 | 2.46 | 122 |
| VSMO | 1.83 | 2.14 | 105 |
| WUSH | 2.37 | 2.66 | 133 |
| X5 | 2.21 | 3.48 | 70 |
| YDEX | 2.74 | 2.86 | 110 |

У TGKA и CNRU абзац дополнить фразой: «Решение торговать дефолты ядра без калибровки не меняется: от core.DefaultParams() литерал отличается только двумя полями режима.»

- [ ] **Step 5: Тесты пакетов тикеров**

Run: `GOMAXPROCS=2 go test ./internal/service/trading_strategy/rsi_pullback/strategy/...`

Для каждого упавшего `<ticker>_test.go`, который сравнивает `DefaultParams()` с полным литералом `want`, добавить в `want` те же `StopOnClose: 1, CatStopMult: 2`. У TGKA (`TestParamsStayAtTheCoreBaselineByDesign`) и CNRU (`TestParamsAreTheCoreBaselineByDesign`) сравнение заменить на:

```go
	got, want := DefaultParams(), core.DefaultParams()
	want.StopOnClose, want.CatStopMult = 1, 2 // режим стопа по закрытию — не калибровка
	if got != want {
```

(сохранив существующий текст `t.Fatalf` и doc-комментарий теста; к комментарию теста дописать строку «Поля режима стопа по закрытию (2026-10-13) — исключение из равенства: это не калибровка».)

Повторить прогон до PASS.

- [ ] **Step 6: Прогон всего репозитория и разбор последствий**

Run: `GOMAXPROCS=2 nice -n 19 go test ./internal/... ./pkg/... ./cmd/... 2>&1 | grep -v '^ok\|no test files'`

Литералы общие для бэктеста (`internal/service/backtest/rsi_pullback_registry.go`) и раннера, поэтому возможны падения вне `live`. Правило разбора:
- тест пинит механику внутрибарного стопа на конкретном тикере из списка — подменить в этом тесте параметры копией с `StopOnClose, CatStopMult = 0, 0` и написать в комментарии почему (как в `TestMain` Step 2);
- тест пинит литерал целиком (равенство структуре) — добавить два поля в ожидание;
- любое другое падение — остановиться и сообщить, не чинить вслепую.

Expected после правок: пустой вывод (все `ok`).

- [ ] **Step 7: Commit**

```bash
git add internal/service/trading_strategy/rsi_pullback internal/service/backtest cmd
git commit -m "feat(pullback): close-based stop with 2x catastrophe stop on 21 live tickers

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

---

### Task 4: Сквозные тесты раннера на боевом GAZP

**Files:**
- Create: `internal/service/trading_strategy/rsi_pullback/live/close_stop_test.go`

**Interfaces:**
- Consumes: `realParams` и подмена GAZP из Task 3; фикстуры `service_test.go`: `newEnv`, `openEntry`, `shareFor`, `heldGAZP`, `tape30m`, `qf`, `flat30m`, `dailies`, `emptyStopList`, `activeList`, `executedList`, `statusIs`, `filledOrder`, `pullback30m`, `msk`.

Уровни боевого GAZP на фикстурах (вход 100, дневной ATR 10, `StopDailyATR` = 0.7): защитный 93, аварийный 86. В тесте они считаются из литерала, не числом.

- [ ] **Step 1: Написать тесты**

`internal/service/trading_strategy/rsi_pullback/live/close_stop_test.go`:

```go
package live

import (
	"context"
	"math"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/mock"
	"google.golang.org/grpc"

	"tinvest/internal/config"
	imodel "tinvest/internal/model"
	investapi "tinvest/internal/pb/v1"
	"tinvest/internal/service/trading_strategy/livecore/stoporders"
	"tinvest/internal/service/trading_strategy/rsi_pullback/live/dto"
	"tinvest/internal/service/trading_strategy/rsi_pullback/strategy/core"
	"tinvest/internal/service/trading_strategy/rsi_pullback/strategy/gazp"
	"tinvest/internal/utils"
	grpcmodel "tinvest/pkg/client/grpc/model"
)

// withRealGAZP возвращает GAZP боевой литерал (стоп по закрытию) на время теста: TestMain
// подменяет его внутрибарной копией ради механических тестов пакета.
func withRealGAZP(t *testing.T) core.Params {
	t.Helper()
	pinned := paramsByTicker[gazp.Ticker]
	real := realParams[gazp.Ticker]
	if real.StopOnClose != 1 {
		t.Fatalf("боевой GAZP без StopOnClose — тест режима потерял смысл: %+v", real)
	}
	paramsByTicker[gazp.Ticker] = real
	t.Cleanup(func() { paramsByTicker[gazp.Ticker] = pinned })
	return real
}

// gazpLevels — защитный и аварийный уровни боевого GAZP от входа entry при дневном ATR 10,
// округлённые вниз до шага 0.01 (как их ставит биржевая заявка).
func gazpLevels(p core.Params, entry float64) (protective, cat float64) {
	prot, _ := core.DesiredStop(p, entry, 10, entry)
	return stoporders.RoundDownToIncrement(prot, 0.01),
		stoporders.RoundDownToIncrement(core.CatastropheStop(p, entry, 10), 0.01)
}

// lastBar30m — ровная лента по 100 с заданным последним баром.
func lastBar30m(end time.Time, n int, o, h, l, c float64) []*imodel.CandleItemTechAnalyse {
	bars := flatAt30m(end, n, 100)
	last := bars[n-1]
	last.Open, last.High, last.Low, last.Close = qf(o), qf(h), qf(l), qf(c)
	return bars
}

// Вход у тикера в режиме ставит на биржу аварийную заявку (CATSL), а не защитный SL.
func TestCloseStopEntryPlacesTheCatastropheOrder(t *testing.T) {
	p := withRealGAZP(t)
	_, cat := gazpLevels(p, 101) // фил по 101
	now := time.Date(2026, 3, 10, 12, 1, 0, 0, msk)
	lastBar := time.Date(2026, 3, 10, 11, 30, 0, 0, msk)

	e := newEnv(t, "GAZP", now, func(c *config.RSIPullbackConfig) { c.TradeEnabled = true })
	e.instruments.EXPECT().Shares(mock.Anything).Return(shareFor("GAZP"), nil)
	e.expectCandles(pullback30m(lastBar, 400), dailies(now, 60))
	e.ops.EXPECT().GetPortfolio(mock.Anything, mock.Anything).Return(nil, nil)
	e.ops.EXPECT().GetPortfolioTotal(mock.Anything, mock.Anything).Return(1_000_000.0, nil)
	e.ops.EXPECT().GetAvailableCash(mock.Anything, mock.Anything).Return(1_000_000.0, nil)
	e.stops.EXPECT().GetStopOrders(mock.Anything, mock.Anything).Return(emptyStopList(), nil)
	e.orders.EXPECT().PostOrder(mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(filledOrder(40, 101), nil).Once()
	e.stops.EXPECT().PostStopOrder(mock.Anything, mock.MatchedBy(func(in *investapi.PostStopOrderRequest) bool {
		return math.Abs(utils.CombinePrice(in.GetStopPrice().GetUnits(), in.GetStopPrice().GetNano())-cat) < 1e-9
	})).Return(&investapi.PostStopOrderResponse{StopOrderId: "so-cat"}, nil).Once()
	e.tg.EXPECT().SendMessage(mock.Anything).Return(nil).Maybe()

	if err := e.svc.Run(context.Background(), dto.Run{}); err != nil {
		t.Fatalf("Run: %v", err)
	}
	got := e.state(t)["GAZP"]
	if got.StopOrderID != "so-cat" || got.StopReason != "CATSL" || math.Abs(got.StopPrice-cat) > 1e-9 {
		t.Fatalf("заявка после входа: %q/%v/%q, want so-cat/%v/CATSL", got.StopOrderID, got.StopPrice, got.StopReason, cat)
	}
}

// closeStopEnv — открытая позиция GAZP (вход 100) с аварийной заявкой so-1 в стейте; боевой
// литерал GAZP включён. Возвращает аварийный уровень.
func closeStopEnv(t *testing.T, bars []*imodel.CandleItemTechAnalyse) (*env, float64) {
	t.Helper()
	p := withRealGAZP(t)
	_, cat := gazpLevels(p, 100)
	now := time.Date(2026, 3, 10, 12, 1, 0, 0, msk)
	e := newEnv(t, "GAZP", now, func(c *config.RSIPullbackConfig) { c.TradeEnabled = true })
	entry := openEntry("so-1")
	entry.StopPrice, entry.StopReason = cat, "CATSL"
	e.seed(t, entry)
	e.instruments.EXPECT().Shares(mock.Anything).Return(shareFor("GAZP"), nil)
	e.expectCandles(bars, dailies(now, 60))
	return e, cat
}

// Бар закрылся ниже защитного уровня (но выше аварийного) — SLC: снять аварийную заявку,
// затем продать рыночной заявкой; стейт очищен, выход с причиной SLC.
func TestCloseStopSellsOnTheCloseBelowTheLevel(t *testing.T) {
	lastBar := time.Date(2026, 3, 10, 11, 30, 0, 0, msk)
	p := realParams[gazp.Ticker]
	prot, cat := gazpLevels(p, 100)
	e, _ := closeStopEnv(t, lastBar30m(lastBar, 400, prot+1, prot+1, cat+2, prot-1))
	e.ops.EXPECT().GetPortfolio(mock.Anything, mock.Anything).Return(heldGAZP(100), nil)
	e.stops.EXPECT().GetStopOrders(mock.Anything, mock.MatchedBy(statusIs(investapi.StopOrderStatusOption_STOP_ORDER_STATUS_ACTIVE))).
		Return(activeList("uid-GAZP", "so-1", cat, 10), nil)
	var seq []string
	e.stops.EXPECT().CancelStopOrder(mock.Anything, mock.MatchedBy(func(in *investapi.CancelStopOrderRequest) bool {
		return in.GetStopOrderId() == "so-1"
	})).Run(func(_ context.Context, _ *investapi.CancelStopOrderRequest, _ ...grpc.CallOption) {
		seq = append(seq, "cancel")
	}).Return(&investapi.CancelStopOrderResponse{}, nil).Once()
	e.orders.EXPECT().PostOrder(mock.Anything, mock.MatchedBy(func(in *investapi.PostOrderRequest) bool {
		return in.GetDirection() == investapi.OrderDirection_ORDER_DIRECTION_SELL
	}), mock.Anything, mock.Anything).Run(func(_ context.Context, _ *investapi.PostOrderRequest, _ ...grpc.CallOption) {
		seq = append(seq, "sell")
	}).Return(filledOrder(10, int64(prot-1)), nil).Once()
	e.tg.EXPECT().SendMessage(mock.MatchedBy(func(s string) bool {
		return strings.Contains(s, "Выход") && strings.Contains(s, "SLC")
	})).Return(nil).Once()

	if err := e.svc.Run(context.Background(), dto.Run{}); err != nil {
		t.Fatalf("Run: %v", err)
	}
	if len(seq) != 2 || seq[0] != "cancel" || seq[1] != "sell" {
		t.Fatalf("порядок вызовов %v, want [cancel sell]", seq)
	}
	if st := e.state(t); len(st) != 0 {
		t.Fatalf("после SLC стейт обязан очиститься, got %+v", st)
	}
}

// Тень проколола защитный уровень, бар закрылся выше — ради этого режим и сделан: ни продажи,
// ни переноса заявки.
func TestCloseStopIgnoresAWickThroughTheLevel(t *testing.T) {
	lastBar := time.Date(2026, 3, 10, 11, 30, 0, 0, msk)
	p := realParams[gazp.Ticker]
	prot, cat := gazpLevels(p, 100)
	e, _ := closeStopEnv(t, lastBar30m(lastBar, 400, 100, 100, cat+1, prot+2))
	e.ops.EXPECT().GetPortfolio(mock.Anything, mock.Anything).Return(heldGAZP(100), nil)
	e.stops.EXPECT().GetStopOrders(mock.Anything, mock.MatchedBy(statusIs(investapi.StopOrderStatusOption_STOP_ORDER_STATUS_ACTIVE))).
		Return(activeList("uid-GAZP", "so-1", cat, 10), nil)
	e.tg.EXPECT().SendMessage(mock.Anything).Return(nil).Maybe()

	if err := e.svc.Run(context.Background(), dto.Run{}); err != nil {
		t.Fatalf("Run: %v", err)
	}
	e.orders.AssertNotCalled(t, "PostOrder", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
	e.stops.AssertNotCalled(t, "CancelStopOrder", mock.Anything, mock.Anything)
	if _, ok := e.state(t)["GAZP"]; !ok {
		t.Fatal("позиция закрыта тенью — режим стопа по закрытию не работает")
	}
}

// Аварийная заявка сработала внутри бара, а портфель брокера ещё показывает позицию. Ядро на
// этом баре тоже скажет CATSL, но продавать второй раз нельзя: только уведомление о выходе.
func TestFiredCatastropheStopDoesNotSellTwice(t *testing.T) {
	lastBar := time.Date(2026, 3, 10, 11, 30, 0, 0, msk)
	p := realParams[gazp.Ticker]
	_, cat := gazpLevels(p, 100)
	e, _ := closeStopEnv(t, lastBar30m(lastBar, 400, 100, 100, cat-2, cat+1))
	e.ops.EXPECT().GetPortfolio(mock.Anything, mock.Anything).Return(heldGAZP(100), nil)
	e.stops.EXPECT().GetStopOrders(mock.Anything, mock.MatchedBy(statusIs(investapi.StopOrderStatusOption_STOP_ORDER_STATUS_ACTIVE))).
		Return(emptyStopList(), nil)
	e.stops.EXPECT().GetStopOrders(mock.Anything, mock.MatchedBy(statusIs(investapi.StopOrderStatusOption_STOP_ORDER_STATUS_EXECUTED))).
		Return(executedList("so-1"), nil)
	e.tg.EXPECT().SendMessage(mock.MatchedBy(func(s string) bool {
		return strings.Contains(s, "Выход") && strings.Contains(s, "CATSL")
	})).Return(nil).Once()

	if err := e.svc.Run(context.Background(), dto.Run{}); err != nil {
		t.Fatalf("Run: %v", err)
	}
	e.orders.AssertNotCalled(t, "PostOrder", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
	if st := e.state(t); len(st) != 0 {
		t.Fatalf("после сработавшей аварийной заявки стейт обязан очиститься, got %+v", st)
	}
}

// Позиция без стейта у тикера в режиме: реконструкция ставит аварийную заявку, а не SL.
func TestReconstructedCloseStopPositionGetsTheCatastropheOrder(t *testing.T) {
	p := withRealGAZP(t)
	_, cat := gazpLevels(p, 100)
	now := time.Date(2026, 3, 10, 12, 1, 0, 0, msk)
	lastBar := time.Date(2026, 3, 10, 11, 30, 0, 0, msk)
	fill := time.Date(2026, 3, 5, 11, 30, 0, 0, msk)

	e := newEnv(t, "GAZP", now, func(c *config.RSIPullbackConfig) { c.TradeEnabled = true })
	e.instruments.EXPECT().Shares(mock.Anything).Return(shareFor("GAZP"), nil)
	e.expectCandles(flat30m(lastBar, 400), dailies(now, 60))
	e.ops.EXPECT().GetPortfolio(mock.Anything, mock.Anything).Return(
		[]*grpcmodel.Position{{ShareID: "uid-GAZP", InstrumentType: "share", Quantity: 500,
			PurchasePrice: gq(100)}}, nil)
	e.ops.EXPECT().GetInstrumentTrades(mock.Anything, "acc", "uid-GAZP", mock.Anything, mock.Anything).
		Return([]grpcmodel.Trade{{IsBuy: true, Date: fill, Price: 100, Quantity: 500}}, nil)
	e.stops.EXPECT().GetStopOrders(mock.Anything, mock.Anything).Return(emptyStopList(), nil)
	e.stops.EXPECT().PostStopOrder(mock.Anything, mock.Anything).
		Return(&investapi.PostStopOrderResponse{StopOrderId: "so-rec"}, nil).Once()
	e.tg.EXPECT().SendMessage(mock.Anything).Return(nil).Maybe()

	if err := e.svc.Run(context.Background(), dto.Run{}); err != nil {
		t.Fatalf("Run: %v", err)
	}
	got := e.state(t)["GAZP"]
	// EntryATR реконструкции ≈ 10, но не обязан быть ровно 10: уровень сверяется с тем, что
	// ядро даёт от восстановленной записи.
	want := stoporders.RoundDownToIncrement(core.CatastropheStop(p, got.EntryPrice, got.EntryATR), 0.01)
	if got.StopReason != "CATSL" || math.Abs(got.StopPrice-want) > 1e-9 {
		t.Fatalf("заявка после реконструкции: %v/%q, want %v/CATSL (ориентир %v)", got.StopPrice, got.StopReason, want, cat)
	}
}
```

- [ ] **Step 2: Прогнать тесты**

Run: `GOMAXPROCS=2 go test ./internal/service/trading_strategy/rsi_pullback/live/ -run 'TestCloseStop|TestFiredCatastrophe|TestReconstructedCloseStop' -v`
Expected: PASS. Режим уже реализован Tasks 1–3 — тесты пинят поведение. Если какой-то падает, это находка: разобрать причину (не ослаблять тест) и сообщить.

Проверка, что тесты не вакуумные: временно закомментировать `paramsByTicker[gazp.Ticker] = real` в `withRealGAZP` и прогнать — `TestCloseStopEntryPlacesTheCatastropheOrder`, `TestCloseStopIgnoresAWickThroughTheLevel` (продажа по SL тенью) и `TestReconstructedCloseStopPositionGetsTheCatastropheOrder` обязаны упасть. Вернуть строку.

- [ ] **Step 3: Весь пакет и линтер**

Run: `GOMAXPROCS=2 go test ./internal/service/trading_strategy/rsi_pullback/... && ../tinvest/bin/golangci-lint run ./internal/service/trading_strategy/rsi_pullback/...`
Expected: PASS, `0 issues.`

- [ ] **Step 4: Commit**

```bash
git add internal/service/trading_strategy/rsi_pullback/live/close_stop_test.go
git commit -m "test(pullback/live): close-based stop end to end on the live GAZP literal

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

---

### Task 5: Документация и финальный гейт

**Files:**
- Modify: `docs/rsi_pullback/strategy.md` (§2 таблица параметров, §4 выходы)
- Modify: `docs/rsi_pullback/live.md` (§1 «Отличия от бэктеста», §6)
- Modify: `CLAUDE.md` (строка про `rsi_pullback` в Layout)

- [ ] **Step 1: `strategy.md` §2** — в таблицу после строки `TrailDailyATR` добавить:

```markdown
| `StopOnClose` | 0/1 | 0 | нет (режим стопа, задаётся литералом) |
| `CatStopMult` | множитель расстояния стопа | 0 | нет (только при `StopOnClose = 1`) |
```

- [ ] **Step 2: `strategy.md` §4** — после абзаца о кодах причин (перед «Позиция переносится через ночь…») вставить:

```markdown
**Стоп по закрытию бара (`StopOnClose = 1`).** Защитный уровень (`SL` или `TRAIL`) проверяется
не по `low`, а по `close` бара, и выход заполняется по закрытию с причиной `SLC` / `TRAILC` —
прокол уровня тенью игнорируется. Внутри бара остаётся только аварийный стоп `CATSL` на уровне
`вход − CatStopMult × StopDailyATR × дневной ATR` (при `CatStopMult > 0`); он заморожен на входе,
трейлом не двигается и, как `SL`, побеждает цель на одном баре. Порядок проверок в этом режиме:
`CATSL → TP → SLC|TRAILC → RSI`. `model.IsStopReason` знает `CATSL` (заливка
`min(уровень, open)`); `SLC`/`TRAILC` стоповыми не считаются и заливаются по close.
```

- [ ] **Step 3: `live.md` §1** — в таблицу «Отличия от бэктеста» после строки `SL / TRAIL` добавить:

```markdown
| CATSL (тикер со `StopOnClose = 1`) | **биржевая стоп-заявка** на аварийном уровне, ставится при входе и не двигается | момент — да (внутрибарно); цена заливки — рыночная |
| SLC / TRAILC (тикер со `StopOnClose = 1`) | market SELL на ближайшем проходе после закрытия бара | момент — нет: бэктест заливает по close сигнального бара, прод — через 1–2 минуты, около open следующего |
```

и под таблицей одну фразу: «У тикера со `StopOnClose = 1` на бирже стоит только аварийная заявка: защитный уровень ядро проверяет по закрытию бара, и раннер продаёт по нему рыночной заявкой».

- [ ] **Step 4: `live.md` §6** — после пункта «Перенос стопа» добавить пункт:

```markdown
- **Смена режима стопа на открытой позиции** (литерал перевели на `StopOnClose` или обратно) —
  если заявка на бирже стоит в одном режиме (`CATSL` или внутрибарный `SL`/`TRAIL`), а литерал
  требует другого, заявка снимается и ставится заново, даже когда новый уровень ниже: правило
  «только вверх» оставило бы позицию со стопом, которого нет в бэктесте её литерала. Запись без
  причины в стейте режимом не считается.
```

- [ ] **Step 5: `CLAUDE.md`** — в описании `rsi_pullback` в разделе Layout после «daily-ATR stop/target» вставить: «(у части тикеров — стоп по закрытию бара `StopOnClose` с аварийной биржевой заявкой `CatStopMult`, спека `docs/superpowers/specs/2026-10-13-pullback-stop-on-close-live-design.md`)».

- [ ] **Step 6: Финальный гейт**

Run: `cd /home/oleg/GolandProjects/tinvest-stoponclose && GOMAXPROCS=2 nice -n 19 ../tinvest/bin/mage ci`
Если `mage` из основного репо не находит `./bin` worktree: `ln -s ../tinvest/bin bin` не делать — вместо этого запустить `GOMAXPROCS=2 nice -n 19 ../tinvest/bin/golangci-lint run ./... && GOMAXPROCS=2 nice -n 19 go test -race ./internal/... ./pkg/... ./cmd/...`.
Expected: зелёный (lint 0 issues, все тесты ok, моки без дрейфа — интерфейсы не менялись).

- [ ] **Step 7: Commit**

```bash
git add docs/rsi_pullback/strategy.md docs/rsi_pullback/live.md CLAUDE.md
git commit -m "docs(pullback): close-based stop mechanics in strategy, live runner and CLAUDE.md

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```
