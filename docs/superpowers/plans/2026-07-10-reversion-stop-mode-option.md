# Reversion Stop Mode Option (close / интрабар) — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Сделать интрабарное исполнение защитного стопа в reversion опциональным: новый параметр `core.Params.UseIntrabarStop` (0 = стоп по close часовой свечи, дефолт; 1 = интрабар + биржевая стоп-заявка в live), NVTK переводится на 1, финал — перепрогон walk-forward и сводный отчёт.

**Архитектура:** Формула уровня (`DesiredStop`) общая для обеих моделей; различаются только триггер (`low ≤ уровень` vs `close ≤ уровень`), источник maxFav для трейла (PrevMaxFav vs MaxFav) и филл (`sig.StopLoss=уровень` → движок филлит `min(уровень, open)` vs `StopLoss=0` → филл по close, fallback уже в движке). В live при `UseIntrabarStop != 1` биржевой стоп не выставляется и не ведётся; оставшаяся после переключения заявка снимается. Спека: `docs/superpowers/specs/2026-07-10-reversion-stop-mode-option-design.md`.

**Tech Stack:** Go 1.25, mockery v2 (моки уже сгенерированы — интерфейсы не меняются), mage.

## Global Constraints

- Работаем на существующей ветке `feat/reversion-stop-orders` (НЕ создавать новую ветку/worktree — это продолжение несмерженной функциональности).
- Гейт качества: `./bin/mage ci` (lint + `go test -race ./...` + mock-drift) — обязателен в Task 4; для точечных проверок `go test ./internal/... -run <Name>`.
- `go build ./...` падает на `magefiles` — использовать `go build ./internal/... ./pkg/... ./cmd/...`.
- Коммиты завершать `Co-Authored-By: Claude Fable 5 <noreply@anthropic.com>`.
- Дефолт параметра — close-модель (zero-value 0): существующие пакеты тикеров, кроме NVTK, не трогаем.
- Порядок приоритетов выходов един для обеих моделей: STOP(SL|TRAIL|ATRSL) → OB → RSI50 → BE → RSIOS → EMAX.

---

### Task 1: core — параметр UseIntrabarStop и двухрежимная STOP-ветка

**Files:**
- Modify: `internal/service/trading_strategy/reversion/strategy/core/core.go`
- Test: `internal/service/trading_strategy/reversion/strategy/core/core_test.go`

**Interfaces:**
- Consumes: существующие `Params`, `DesiredStop(p, entryPrice, entryATR, maxFav) (float64, string)` — сигнатура НЕ меняется.
- Produces: поле `Params.UseIntrabarStop int` (0 = close-модель, 1 = интрабар) — Task 2 (live) и Task 3 (nvtk.go) читают его; семантика `manage()`: при 0 триггер `in.price <= level`, maxFav = `Position.MaxFavorablePrice`, `sig.StopLoss` не ставится; при 1 — прежнее поведение ветки.

- [ ] **Step 1: Написать падающие тесты close-модели**

В конец `core_test.go` добавить (helpers `defaultParams`, `openInput`, `atrStopParams` уже существуют в файле):

```go
// Close-модель (дефолт): прокол low ниже уровня при close выше НЕ триггерит стоп.
func TestCloseStopIgnoresLowPoke(t *testing.T) {
	s := NewWithParams("T", atrStopParams()) // UseIntrabarStop=0 по умолчанию
	in := openInput()
	in.pos = &strategy.Position{PurchasePrice: 100, EntryATR: 5}
	in.price, in.low = 96, 94 // low проколол порог 95, close вернулся выше
	if sig := s.decide(in); sig.Kind == model.SignalSell {
		t.Fatalf("close-модель: прокол low не должен продавать, got %q", sig.Reason)
	}
}

// Close-модель: триггер по close, sig.StopLoss НЕ ставится (движок филлит по close).
func TestCloseStopFiresOnCloseWithoutStopLoss(t *testing.T) {
	s := NewWithParams("T", atrStopParams())
	in := openInput()
	in.pos = &strategy.Position{PurchasePrice: 100, EntryATR: 5}
	in.price, in.low = 94, 93 // close 94 <= порога 95
	sig := s.decide(in)
	if sig.Kind != model.SignalSell || sig.Reason != "ATRSL" {
		t.Fatalf("want ATRSL sell on close, got kind=%v reason=%q", sig.Kind, sig.Reason)
	}
	if sig.StopLoss != 0 {
		t.Fatalf("close-модель: StopLoss должен остаться 0 (филл по close), got %v", sig.StopLoss)
	}
}

// Close-модель: трейл считается от ТЕКУЩЕГО MaxFavorablePrice (решение на закрытии,
// весь бар известен), а не от PrevMaxFav, как в интрабарной модели.
func TestCloseTrailUsesCurrentMaxFav(t *testing.T) {
	p := defaultParams()
	p.UseTrail, p.TrailATRMult, p.ATRPeriod = 1, 1.5, 14
	s := NewWithParams("T", p)
	in := openInput()
	in.pos = &strategy.Position{PurchasePrice: 100, EntryATR: 2,
		MaxFavorablePrice: 120, PrevMaxFavorablePrice: 110} // close-уровень 117, интрабарный был бы 107
	in.price, in.low = 116.5, 116.5 // 116.5 <= 117 -> TRAIL; от prevMax (107) держали бы
	sig := s.decide(in)
	if sig.Kind != model.SignalSell || sig.Reason != "TRAIL" {
		t.Fatalf("want TRAIL from current MaxFav, got kind=%v reason=%q", sig.Kind, sig.Reason)
	}
	if sig.StopLoss != 0 {
		t.Fatalf("close-модель: StopLoss должен остаться 0, got %v", sig.StopLoss)
	}
}
```

- [ ] **Step 2: Убедиться, что тесты падают**

Run: `go test ./internal/service/trading_strategy/reversion/strategy/core/ -run 'TestCloseStop|TestCloseTrail' -v`
Expected: FAIL — `UseIntrabarStop` ещё нет, интрабарная семантика безусловна: `TestCloseStopIgnoresLowPoke` продаёт по проколу low, остальные два видят непустой `StopLoss`.

- [ ] **Step 3: Добавить поле в Params**

В `core.go` после поля `UseRSI50` в struct `Params`:

```go
	UseIntrabarStop int     // 0 = close-модель (дефолт): стоп проверяется по close часовой свечи, филл по close, в live биржевая стоп-заявка НЕ выставляется; 1 = интрабар: триггер low ≤ уровень, филл min(уровень, open), в live — реальная биржевая стоп-заявка
```

- [ ] **Step 4: Переписать STOP-ветку manage()**

В `core.go` заменить начало `manage()` (строки с вызовом `DesiredStop` и первым `case`):

```go
func (s *Strategy) manage(in decideInput, sig model.Signal) model.Signal {
	sig.RSI = in.rsiNow

	// Источник maxFav для трейла зависит от модели исполнения: интрабарная заявка
	// стояла на бирже с закрытия ПРОШЛОГО бара (PrevMaxFav); close-модель решает на
	// закрытии текущего бара, когда весь бар известен (MaxFav) — как июньская модель.
	maxFav := in.pos.MaxFavorablePrice
	if s.p.UseIntrabarStop == 1 {
		maxFav = in.pos.PrevMaxFavorablePrice
	}
	stopLevel, stopReason := DesiredStop(s.p, in.pos.PurchasePrice, in.pos.EntryATR, maxFav)

	stopHit := false
	if stopReason != "" {
		if s.p.UseIntrabarStop == 1 {
			stopHit = in.low > 0 && in.low <= stopLevel // биржевая заявка: касание low
		} else {
			stopHit = in.price <= stopLevel // close-модель: только закрытие бара
		}
	}

	switch {
	case stopHit:
		sig.Kind, sig.Reason = model.SignalSell, stopReason
		if s.p.UseIntrabarStop == 1 {
			sig.StopLoss = stopLevel // движок филлит min(уровень, open); в close-модели StopLoss=0 -> филл по close
			sig.ExitReason = fmt.Sprintf("%s: low %.4f ≤ стоп %.4f (вход %.4f, ATR %.4f, prevMaxFav %.4f)",
				stopReason, in.low, stopLevel, in.pos.PurchasePrice, in.pos.EntryATR, in.pos.PrevMaxFavorablePrice)
		} else {
			sig.ExitReason = fmt.Sprintf("%s: close %.4f ≤ стоп %.4f (вход %.4f, ATR %.4f, maxFav %.4f)",
				stopReason, in.price, stopLevel, in.pos.PurchasePrice, in.pos.EntryATR, in.pos.MaxFavorablePrice)
		}
```

Остальные `case` (OB/RSI50/BE/RSIOS/EMAX) — без изменений.

- [ ] **Step 5: Обновить doc-комментарии**

1. Package-comment (`core.go`, шапка): фразу про STOP «modeled as an exchange stop order, it triggers intrabar the instant the bar's LOW touches/pierces the level, pre-empting every close-based exit including OB» заменить на описание двух моделей: «execution model is selected per ticker by UseIntrabarStop: 0 (default) checks the stop against the bar CLOSE and fills at close; 1 models an exchange stop order that triggers intrabar the instant the bar's LOW touches/pierces the level and fills at min(level, open). Either way STOP pre-empts every close-based exit including OB».
2. Doc-комментарий `manage()`: абзац про STOP переписать — уровень общий (`DesiredStop`), далее две модели: интрабар (триггер low, prevMaxFav, филл min(level, open)) и close (триггер close, текущий MaxFav, филл по close; sig.StopLoss не ставится). Упомянуть, что порядок приоритетов одинаков в обеих моделях.
3. Комментарий поля `atr` в `decideInput` и `DesiredStop` — упоминание «backtest passes PrevMaxFavorablePrice» уточнить: «intrabar model passes PrevMaxFavorablePrice, close model passes MaxFavorablePrice».

- [ ] **Step 6: Проставить UseIntrabarStop=1 в тестах интрабарной семантики**

Эти тесты написаны про интрабар (прокол low при close выше, prevMaxFav, sentinel low=0, assert `sig.StopLoss`) и после смены дефолта обязаны явно включать флаг — добавить строку `p.UseIntrabarStop = 1` после создания params в каждом:

- `TestManageCatStopExit` (assert StopLoss 94)
- `TestManageTrailExit` (assert StopLoss 114)
- `TestIntrabarStopFiresOnLowTouch`
- `TestIntrabarTrailUsesPrevMaxFav`
- `TestStopBeatsOverboughtSameBar` (close 101 выше уровня — только интрабар)
- `TestIntrabarStopSkippedWithoutLow` (low-сентинел)

Пример для `TestIntrabarStopFiresOnLowTouch`:

```go
	p := defaultParams()
	p.UseTrail, p.TrailATRMult, p.ATRPeriod = 1, 1.5, 14
	p.UseIntrabarStop = 1
```

Остальные стоп-тесты (`TestExitATRStopFires`, `TestNoATRStopAboveThreshold`, `TestExitPrecedenceSTOPOverRSI50`, `TestExitPrecedenceATROverEMA`, `TestExitPrecedenceSTOPOverBreakeven`, `TestATRStopSkipped*`, `TestRSIOSInertWhenATRStopOn`) ставят price и low по одну сторону уровня и не ассертят StopLoss — они обязаны проходить в ОБЕИХ моделях, флаг в них НЕ добавлять (это бесплатная проверка эквивалентности моделей на «обычных» барах).

- [ ] **Step 7: Прогнать тесты пакета**

Run: `go test ./internal/service/trading_strategy/reversion/strategy/core/ -v 2>&1 | tail -20`
Expected: PASS все, включая три новых close-теста.

- [ ] **Step 8: Commit**

```bash
git add internal/service/trading_strategy/reversion/strategy/core/
git commit -m "feat(reversion/core): UseIntrabarStop param — optional close-model stop execution (default)

Co-Authored-By: Claude Fable 5 <noreply@anthropic.com>"
```

---

### Task 2: live — гейт стоп-механики по UseIntrabarStop + снятие оставшейся заявки

**Files:**
- Modify: `internal/service/trading_strategy/reversion/live/buy.go` (`placeInitialStop`)
- Modify: `internal/service/trading_strategy/reversion/live/manage.go` (`managePass`)
- Test: `internal/service/trading_strategy/reversion/live/service_test.go`

**Interfaces:**
- Consumes: `core.Params.UseIntrabarStop` из Task 1; существующие `ParamsFor(ticker)`, `mustParams(ticker)`, `s.stops.Cancel/Place/List`, `statestore.Entry{StopOrderID, StopPrice, StopReason}`.
- Produces: поведение live: close-тикер никогда не ставит и не ведёт биржевой стоп; оставшаяся заявка снимается в managePass. Никаких новых экспортируемых имён.

- [ ] **Step 1: Тест-хелпер переключения UGLD на интрабар + падающие тесты**

Тесты стоп-механики написаны против UGLD-окружения (`newManageEnv`, `cfg` с `Tickers: ["UGLD"]`), а UGLD теперь close-модель. В `service_test.go` добавить хелпер:

```go
// withIntrabarUGLD временно переводит UGLD в реестре на интрабарную модель — тесты
// биржевой стоп-механики написаны против UGLD-окружения (newManageEnv), а боевой UGLD
// работает по close-модели (без биржевых стопов).
func withIntrabarUGLD(t *testing.T) {
	t.Helper()
	old := paramsByTicker["UGLD"]
	p := old
	p.UseIntrabarStop = 1
	paramsByTicker["UGLD"] = p
	t.Cleanup(func() { paramsByTicker["UGLD"] = old })
}
```

И два новых теста:

```go
// Close-модель (дефолт UGLD): placeInitialStop не выставляет биржевую заявку.
// stops-мок без ожиданий — любой вызов Place уронит тест (mockery strict).
func TestPlaceInitialStopSkippedOnCloseModel(t *testing.T) {
	dir := t.TempDir()
	statePath := filepath.Join(dir, "state.json")
	stops := stopmocks.NewMockClient(t)
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
	if got.StopOrderID != "" || got.StopPrice != 0 || got.StopReason != "" {
		t.Fatalf("close-модель: стоп-заявка не должна выставляться, got %+v", got)
	}
}

// Close-модель: заявка, оставшаяся после переключения с интрабара, снимается,
// стоп-поля стейта чистятся, позиция остаётся (SELL ядро не сигналит).
func TestManagePass_CancelsLeftoverStopOnCloseModel(t *testing.T) {
	env := newManageEnv(t, seedEntry("so-old", 97), hourlySeries(400, 101))
	env.stops.EXPECT().GetStopOrders(mock.Anything, mock.Anything).
		Return(activeList("so-old", 97, 10), nil)
	env.stops.EXPECT().CancelStopOrder(mock.Anything, mock.MatchedBy(func(in *investapi.CancelStopOrderRequest) bool {
		return in.GetStopOrderId() == "so-old"
	})).Return(&investapi.CancelStopOrderResponse{}, nil)

	if err := env.svc.Run(context.Background(), dto.Run{Mode: dto.ModeManage}); err != nil {
		t.Fatalf("Run: %v", err)
	}
	st, _ := statestore.New(env.statePath).Load()
	e := st["UGLD"]
	if e.StopOrderID != "" || e.StopPrice != 0 || e.StopReason != "" {
		t.Fatalf("стоп-поля должны очиститься: %+v", e)
	}
	if e.Ticker == "" {
		t.Fatalf("позиция не должна пропасть из стейта")
	}
}
```

(Импорты `context`, `filepath`, `mock`, `investapi`, `imodel`, `dto`, `statestore` и моки уже есть в файле.)

- [ ] **Step 2: Пометить существующие тесты стоп-механики как интрабарные**

Добавить `withIntrabarUGLD(t)` первой строкой в:
- `TestPlaceInitialStopPersistsIDAndLevel`
- `TestManagePass_RepostsStopWhenTrailRises`
- `TestManagePass_KeepsStopWhenLevelUnchanged`
- `TestManagePass_StrayCancelFailBlocksNewStopPlacement`
- `TestManagePass_ReconcilesStopSizeAfterPartialFill`

НЕ трогать: `TestManagePass_DetectsFiredStop`, `TestManagePass_OrphanedStopWhenSoldExternally` (ветка «позиция исчезла» не консультирует params), `TestManagePass_CancelFailBlocksMarketSell` (cancel-перед-продажей стоит ДО гейта модели и обязан работать для close-тикера с оставшейся заявкой), `TestManagePass_UpdatesMaxFavAndPersists`, `TestBuyPass_NoSignal_NoOrderNoState`.

- [ ] **Step 3: Убедиться, что новые тесты падают**

Run: `go test ./internal/service/trading_strategy/reversion/live/ -run 'CloseModel' -v`
Expected: FAIL — `TestPlaceInitialStopSkippedOnCloseModel` падает на неожиданном `PostStopOrder` (mockery strict), `TestManagePass_CancelsLeftoverStopOnCloseModel` — стоп-поля не чистятся (сейчас sync ведёт заявку).

- [ ] **Step 4: buy.go — не ставить стоп для close-тикера**

В `placeInitialStop` заменить гейт:

```go
	p, ok := ParamsFor(ticker)
	if !ok || p.UseIntrabarStop != 1 {
		return entry // close-модель: биржевой стоп не выставляется, выходы — по сигналам ядра на закрытии часа
	}
```

- [ ] **Step 5: manage.go — гейт синхронизации + переходное снятие**

В `managePass` перед блоком «Синхронизация стоп-заявки» (сразу после `continue` SELL-ветки) вставить:

```go
		p := mustParams(ticker)
		if p.UseIntrabarStop != 1 {
			// Close-модель: биржевой стоп не ведём. Снять заявку, оставшуюся после
			// переключения модели; мёртвую (не в живых по List) — просто вычистить из
			// стейта. При недоступном List ничего не трогаем — ретрай на следующем тике.
			if entry.StopOrderID != "" && listErr == nil {
				if _, alive := stopByID[entry.StopOrderID]; alive {
					if err := s.stops.Cancel(ctx, entry.StopOrderID); err != nil {
						s.notify(notifier.Alert(ticker, "close-модель: не удалось снять оставшуюся стоп-заявку: "+err.Error()))
						continue // заявка жива — стейт не трогаем, ретрай на следующем тике
					}
					s.notify(notifier.Alert(ticker, "close-модель: снял оставшуюся биржевую стоп-заявку"))
				}
				entry.StopOrderID, entry.StopPrice, entry.StopReason = "", 0, ""
				state[ticker] = entry
				if err := store.Save(state); err != nil {
					return fmt.Errorf("reversion: save stop state %s: %w", ticker, err)
				}
			}
			continue
		}
```

Существующий вызов `core.DesiredStop(mustParams(ticker), ...)` ниже заменить на `core.DesiredStop(p, ...)` (params уже подняты). Остальной sync-блок без изменений.

- [ ] **Step 6: Прогнать live-тесты**

Run: `go test ./internal/service/trading_strategy/reversion/live/ 2>&1 | tail -5`
Expected: PASS (включая оба новых и все существующие с `withIntrabarUGLD`).

- [ ] **Step 7: Commit**

```bash
git add internal/service/trading_strategy/reversion/live/
git commit -m "feat(reversion/live): gate exchange stop orders by UseIntrabarStop, cancel leftover stop on close-model ticker

Co-Authored-By: Claude Fable 5 <noreply@anthropic.com>"
```

---

### Task 3: NVTK → интрабар; документация

**Files:**
- Modify: `internal/service/trading_strategy/reversion/strategy/nvtk/nvtk.go`
- Modify: `docs/reversion/strategy.md`, `docs/reversion/live-runner.md`, `docs/reversion/live-code-map.md`

**Interfaces:**
- Consumes: `Params.UseIntrabarStop` из Task 1.
- Produces: `nvtk.DefaultParams().UseIntrabarStop == 1` — Task 4 полагается на это: WF-прогон NVTK автоматически идёт в интрабарной модели (grid-калибровка стартует с `b.DefaultParams()` тикера, fixed-грид `reversion_fixed_current.json` поле не перечисляет → наследуется).

- [ ] **Step 1: nvtk.go**

В struct-литерал `DefaultParams()` добавить строку (рядом с выходами):

```go
		// Модель исполнения стопа: интрабар (биржевая стоп-заявка в live). Интрабарный
		// перепрогон 2026-07-10 показал минимальную деградацию (пул OOS PF 5.762 -> 4.842),
		// защита биржевым стопом почти бесплатна. UGLD/EUTR остаются на close-модели
		// своей июньской калибровки (дефолт UseIntrabarStop=0).
		UseIntrabarStop: 1,
```

Проверить: `go test ./internal/service/trading_strategy/reversion/... 2>&1 | tail -3` → PASS.

- [ ] **Step 2: docs/reversion/strategy.md**

Раздел про STOP/интрабарную модель переписать под опцию: уровень (`DesiredStop`) общий; `UseIntrabarStop=1` — интрабар (low-триггер, prevMaxFav, филл `min(уровень, open)`, в live биржевая заявка); `UseIntrabarStop=0` (дефолт) — close-модель (close-триггер, текущий MaxFav, филл по close, в live заявка не выставляется). Отметить два сознательных отличия close-модели от июньской (единый приоритет STOP-первым; честный close-филл вместо `min(уровень, open)` у SL/TRAIL) со ссылкой на спеку `2026-07-10-reversion-stop-mode-option-design.md`.

- [ ] **Step 3: docs/reversion/live-runner.md и live-code-map.md**

В live-runner.md (раздел «Стоп-заявки») добавить: механика активна только для тикеров с `UseIntrabarStop=1` (сейчас NVTK); для close-тикеров (UGLD/EUTR) стоп-заявки не выставляются, выходы — рыночной продажей по сигналу ядра на закрытии часа; при переключении тикера на close-модель managePass снимает оставшуюся заявку и чистит стоп-поля стейта. В live-code-map.md поправить описания `placeInitialStop`/`managePass`, если они утверждают безусловную установку/ведение стопа.

- [ ] **Step 4: Commit**

```bash
git add internal/service/trading_strategy/reversion/strategy/nvtk/ docs/reversion/
git commit -m "feat(reversion): NVTK on intrabar stop model; document stop mode option

Co-Authored-By: Claude Fable 5 <noreply@anthropic.com>"
```

---

### Task 4: CI-гейт, финальные прогоны, сводный отчёт

**Files:**
- Create: `docs/reversion/stop-mode-final-2026-07.md`
- Create: `reports/UGLD/stopmode/`, `reports/EUTR/stopmode/`, `reports/NVTK/stopmode/` (артефакты прогонов)

**Interfaces:**
- Consumes: раскладку из Task 3 (NVTK интрабар, UGLD/EUTR close по дефолту); свечные кэши `data/candles/*` (валидны с перепрогона 2026-07-10; при ошибке парсинга JSON — перекачать `-refresh`, токен из `env/token.env`).
- Produces: сводный отчёт для финального ответа пользователю.

- [ ] **Step 1: Полный гейт качества**

Run: `./bin/mage ci`
Expected: lint OK, `go test -race ./...` PASS, mock-drift OK. Починить, если нет.

- [ ] **Step 2: Walk-forward в итоговой конфигурации (те же окна, что в intrabar-rerun)**

```bash
mkdir -p reports/UGLD/stopmode reports/EUTR/stopmode reports/NVTK/stopmode
go run ./cmd/backtest -ticker UGLD -strategy reversion -interval Hour1 \
  -calibrate data/params/ugld/reversion_fixed.json -out ./reports/UGLD/stopmode \
  -months 30 -train-months 12 -test-months 6 -min-trades 1 -metric profit_factor
go run ./cmd/backtest -ticker EUTR -strategy reversion -interval Hour1 \
  -calibrate data/params/eutr/reversion_fixed.json -out ./reports/EUTR/stopmode \
  -months 31 -train-months 12 -test-months 6 -min-trades 1 -metric profit_factor
go run ./cmd/backtest -ticker NVTK -strategy reversion -interval Hour1 \
  -calibrate data/params/nvtk/reversion_fixed_current.json -out ./reports/NVTK/stopmode \
  -months 36 -train-months 18 -test-months 6 -min-trades 1 -metric profit_factor
```

Expected: `*_walkforward.md` в каждой папке. Модель берётся из `DefaultParams()` тикера автоматически (fixed-гриды не перечисляют `UseIntrabarStop`): UGLD/EUTR — close, NVTK — интрабар.

- [ ] **Step 3: Одиночные полноисторийные прогоны (журнал сделок)**

```bash
go run ./cmd/backtest -ticker UGLD -strategy reversion -interval Hour1 -months 30 -out ./reports/UGLD/stopmode
go run ./cmd/backtest -ticker EUTR -strategy reversion -interval Hour1 -months 31 -out ./reports/EUTR/stopmode
go run ./cmd/backtest -ticker NVTK -strategy reversion -interval Hour1 -months 36 -out ./reports/NVTK/stopmode
```

- [ ] **Step 4: Сводный отчёт `docs/reversion/stop-mode-final-2026-07.md`**

Структура (по образцу `docs/reversion/intrabar-rerun-2026-07.md`):
- Шапка: статус, ссылка на спеку, раскладка моделей по тикерам.
- Сводная таблица по пулу OOS на тикер, ТРИ строки на тикер: июньский close-baseline (цифры из `intrabar-rerun-2026-07.md`: UGLD 3.385/25/32.0%/26.4%; EUTR 2.529/47/55.3%/18.7%; NVTK 5.762/33/54.5%/33.5%) | интрабарный перепрогон 2026-07-10 (1.418/27; 1.084/51; 4.842/31) | финальная конфигурация (новые цифры). Колонки: PF, сделки, win rate, compounded return, худший фолд MaxDD%.
- Per-fold таблицы по каждому тикеру.
- Одиночные прогоны: сделки/NetPnL/PF.
- **Контроль корректности (из спеки, §6):** UGLD/EUTR (close) должны вернуться ≈ к июньским цифрам — не бит-в-бит (окна те же, что в июльском перепрогоне, но vs июнь сдвинуты ~3 недели; филл SL/TRAIL теперь честный по close, а не `min(уровень, open)`, поэтому лёгкое ухудшение против июня ожидаемо); NVTK — совпасть с интрабарным перепрогоном 2026-07-10 практически точно (его параметры и модель не менялись; расхождение возможно только от новых свечей в кэше). Существенные расхождения с ожиданием — разобрать в отчёте, не замалчивать.
- Вывод: итоговая раскладка live-реестра, EUTR остаётся кандидатом на перекалибровку под интрабар (вне скоупа).

- [ ] **Step 5: Commit**

```bash
git add docs/reversion/stop-mode-final-2026-07.md reports/UGLD/stopmode reports/EUTR/stopmode reports/NVTK/stopmode
git commit -m "docs(reversion): final stop-mode walk-forward — close (UGLD/EUTR) vs intrabar (NVTK) configuration

Co-Authored-By: Claude Fable 5 <noreply@anthropic.com>"
```

- [ ] **Step 6: Презентация отчёта пользователю**

В финальном ответе пользователю ОБЯЗАТЕЛЬНО привести содержание сводного отчёта (таблицы и выводы), а не только путь к файлу — прямое требование пользователя. Ветку НЕ мержить: предложить ревью (`/code-review ultra` запускает только пользователь).
