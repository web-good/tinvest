# Levels: внутрибаровое исполнение стопов — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Стопы стратегии levels (жёсткий SL и chandelier-трейл) срабатывают по внутрибаровому минимуму (`low`) и исполняются по уровню стопа с поправкой на гэп, а не по цене закрытия бара.

**Architecture:** Стратегия остаётся чистой: ядро `decide()` решает, выходить ли (триггер по `in.barLow`) и кладёт уровень стопа в `sig.StopLoss`. Движок бэктеста владеет механикой fill: для стоповых выходов цена исполнения = `min(уровень_стопа, open_бара)` — честный учёт гэп-вниз. Арминг трейла по-прежнему на цене закрытия (консервативно).

**Tech Stack:** Go 1.25 (встроенный `min`), стандартный `testing`. Затрагиваются два пакета: `internal/service/trading_strategy/levels/strategy/core` и `internal/domain/backtest`.

**Спецификация:** `docs/superpowers/specs/2026-06-07-levels-intrabar-stop-execution-design.md`

---

## Структура файлов

- `internal/service/trading_strategy/levels/strategy/core/core.go` — **изменить** `decide()`: триггеры стопов с `in.price` на `in.barLow`.
- `internal/service/trading_strategy/levels/strategy/core/core_test.go` — **изменить** существующие тесты управления позицией (они опираются на `in.price`) + **добавить** новые кейсы про `barLow`.
- `internal/domain/backtest/engine.go` — **изменить** ветку `SignalSell` в `Run()`: гэп-аккуратный fill по уровню стопа.
- `internal/domain/backtest/engine_test.go` — **добавить** тесты исполнения стопа (обычный пробой / гэп вниз / не-стоповый sell).
- `docs/levels/strategy.md` — **изменить** раздел 5 + добавить live-контракт.

---

## Task 1: Триггер стопов по `low` в ядре стратегии

**Files:**
- Modify: `internal/service/trading_strategy/levels/strategy/core/core.go:156-161`
- Modify/Test: `internal/service/trading_strategy/levels/strategy/core/core_test.go`

Контекст: сейчас в `decide()` (управление открытой позицией) триггеры стопов проверяют `in.price` (последний close). Меняем на `in.barLow` (внутрибаровый минимум). **Арминг трейла остаётся на `in.price`** — трейл не активируется, пока бар не закрылся в нужном плюсе. Уровни `hardSL` и `chandelier` считаются как сейчас.

Важно: текущие тесты `TestDecideHardStopExit` и `TestDecideInProfitHold` опираются на `in.price` и сломаются после смены триггера (в `bounceInput()` зашит `barLow: 99.8`, который ниже chandelier 107.5). Их нужно обновить в этой же задаче.

- [ ] **Step 1: Добавить новые провальные тесты (триггер по low)**

В конец `core_test.go` (перед строкой `// Interface satisfaction.`) добавить:

```go
func TestDecideHardStopOnLowWhileCloseAbove(t *testing.T) {
	s := newCore()
	in := bounceInput()
	in.pos = &strategy.Position{PurchasePrice: 100, Quantity: 1}
	in.price = 99.5  // close ещё выше hard SL (99) — по close выхода бы не было
	in.barLow = 98.8 // но low пробил hard SL внутри бара
	sig := s.decide(in)
	if sig.Kind != model.SignalSell || sig.Reason != "SL" {
		t.Fatalf("low pierce of hard SL must sell SL, got %v/%q", sig.Kind, sig.Reason)
	}
	if sig.StopLoss != 99 {
		t.Errorf("stop = %v, want 99", sig.StopLoss)
	}
}

func TestDecideTrailOnLowWhileCloseAbove(t *testing.T) {
	s := newCore()
	in := bounceInput()
	in.pos = &strategy.Position{PurchasePrice: 100, Quantity: 1}
	in.recentHigh = 110 // chandelier = 110 - 2.5 = 107.5
	in.price = 108      // close выше chandelier и armed (>= entry+1ATR=101)
	in.barLow = 107.0   // low пробил chandelier внутри бара
	sig := s.decide(in)
	if sig.Kind != model.SignalSell || sig.Reason != "TRAIL" {
		t.Fatalf("low pierce of chandelier must sell TRAIL, got %v/%q", sig.Kind, sig.Reason)
	}
	if sig.StopLoss != 107.5 {
		t.Errorf("trail stop = %v, want 107.5", sig.StopLoss)
	}
}

func TestDecideNoStopWhenLowAboveStops(t *testing.T) {
	s := newCore()
	in := bounceInput()
	in.pos = &strategy.Position{PurchasePrice: 100, Quantity: 1}
	in.recentHigh = 110 // chandelier 107.5
	in.price = 108
	in.barLow = 107.8 // low остался выше chandelier и hard SL -> держим
	sig := s.decide(in)
	if sig.Kind != model.SignalNone {
		t.Fatalf("low above both stops must hold, got %v/%q", sig.Kind, sig.Reason)
	}
}

func TestDecideTrailArmStaysOnClose(t *testing.T) {
	s := newCore()
	in := bounceInput()
	in.pos = &strategy.Position{PurchasePrice: 100, Quantity: 1}
	in.recentHigh = 110 // chandelier 107.5
	in.price = 100.5    // close < entry+1ATR=101 -> трейл НЕ взведён
	in.barLow = 100.2   // low ниже chandelier, но выше hard SL (99) -> арминг держит
	sig := s.decide(in)
	if sig.Kind != model.SignalNone {
		t.Fatalf("unarmed trail must hold even if low below chandelier, got %v/%q", sig.Kind, sig.Reason)
	}
}
```

- [ ] **Step 2: Запустить новые тесты — убедиться, что падают**

Run: `go test ./internal/service/trading_strategy/levels/strategy/core/ -run 'TestDecideHardStopOnLowWhileCloseAbove|TestDecideTrailOnLowWhileCloseAbove' -v`
Expected: FAIL — текущий код триггерит по `in.price`, поэтому `price 99.5` (> 99) и `price 108` (> 107.5) не дают выхода (`SignalNone` вместо `Sell`).

- [ ] **Step 3: Сменить триггеры стопов на `in.barLow` в `decide()`**

В `core.go` заменить блок `switch` (строки ~156-161):

```go
		switch {
		case in.price <= hardSL:
			sig.Kind, sig.Reason = model.SignalSell, "SL"
		case armed && in.price <= chandelier:
			sig.Kind, sig.Reason, sig.StopLoss = model.SignalSell, "TRAIL", chandelier
		}
```

на:

```go
		switch {
		case in.barLow <= hardSL:
			sig.Kind, sig.Reason = model.SignalSell, "SL"
		case armed && in.barLow <= chandelier:
			sig.Kind, sig.Reason, sig.StopLoss = model.SignalSell, "TRAIL", chandelier
		}
```

(Строки `hardSL := ...`, `chandelier := ...`, `armed := ...`, `sig.StopLoss = hardSL` выше по коду НЕ трогаем — `armed` остаётся на `in.price`.)

- [ ] **Step 4: Обновить существующие тесты управления позицией под новую семантику**

В `core_test.go`:

`TestDecideHardStopExit` — добавить установку `barLow` (иначе зашитый `barLow: 99.8 > 99` не триггерит):

```go
func TestDecideHardStopExit(t *testing.T) {
	s := newCore()
	in := bounceInput()
	in.pos = &strategy.Position{PurchasePrice: 100, Quantity: 1}
	in.price = 98.9  // <= entry - SLMult*atr = 99
	in.barLow = 98.9 // low тоже пробил hard SL
	sig := s.decide(in)
	if sig.Kind != model.SignalSell || sig.Reason != "SL" {
		t.Fatalf("want Sell/SL, got %v/%q", sig.Kind, sig.Reason)
	}
	if sig.StopLoss != 99 {
		t.Errorf("stop = %v, want 99", sig.StopLoss)
	}
}
```

`TestDecideTrailExit` — добавить реалистичный `barLow` (был неявный 99.8):

```go
func TestDecideTrailExit(t *testing.T) {
	s := newCore()
	in := bounceInput()
	in.pos = &strategy.Position{PurchasePrice: 100, Quantity: 1}
	in.recentHigh = 110
	// armed: price >= entry + TrailArmATR*atr = 101. chandelier = 110 - 2.5 = 107.5.
	in.price = 107
	in.barLow = 107 // low на уровне триггера трейла
	sig := s.decide(in)
	if sig.Kind != model.SignalSell || sig.Reason != "TRAIL" {
		t.Fatalf("want Sell/TRAIL, got %v/%q", sig.Kind, sig.Reason)
	}
	if sig.StopLoss != 107.5 {
		t.Errorf("trail stop = %v, want 107.5", sig.StopLoss)
	}
}
```

`TestDecideTrailNotArmed` — добавить реалистичный `barLow` выше hard SL:

```go
func TestDecideTrailNotArmed(t *testing.T) {
	s := newCore()
	in := bounceInput()
	in.pos = &strategy.Position{PurchasePrice: 100, Quantity: 1}
	in.recentHigh = 110 // chandelier 107.5
	in.price = 100.5    // below entry + 1 ATR -> not armed, above hard SL -> hold
	in.barLow = 100.5   // low выше hard SL (99) -> ничего не триггерит
	sig := s.decide(in)
	if sig.Kind != model.SignalNone {
		t.Fatalf("unarmed trail with price above hard SL must hold, got %v/%q", sig.Kind, sig.Reason)
	}
}
```

`TestDecideInProfitHold` — добавить `barLow` выше chandelier (иначе зашитый 99.8 триггерит трейл):

```go
func TestDecideInProfitHold(t *testing.T) {
	s := newCore()
	in := bounceInput()
	in.pos = &strategy.Position{PurchasePrice: 100, Quantity: 1}
	in.recentHigh = 110 // chandelier 107.5
	in.price = 108.5    // armed (>=101) но close выше chandelier
	in.barLow = 108.0   // low тоже выше chandelier -> hold
	sig := s.decide(in)
	if sig.Kind != model.SignalNone {
		t.Fatalf("in-profit above chandelier must hold, got %v/%q", sig.Kind, sig.Reason)
	}
}
```

- [ ] **Step 5: Запустить весь пакет core — все тесты зелёные**

Run: `go test ./internal/service/trading_strategy/levels/strategy/core/ -v`
Expected: PASS (новые кейсы про low + обновлённые кейсы управления позицией + не тронутые кейсы входа).

- [ ] **Step 6: Commit**

```bash
git add internal/service/trading_strategy/levels/strategy/core/core.go internal/service/trading_strategy/levels/strategy/core/core_test.go
git commit -m "feat(levels): trigger stops on intrabar low instead of close"
```

---

## Task 2: Гэп-аккуратное исполнение стопов в движке бэктеста

**Files:**
- Modify: `internal/domain/backtest/engine.go:66-69`
- Test: `internal/domain/backtest/engine_test.go`

Контекст: сейчас `Run()` исполняет любой `SignalSell` по `c.Close`. Для стоповых выходов (`Reason` = "SL"/"TRAIL") fill должен идти по уровню стопа с поправкой на гэп: `min(sig.StopLoss, c.Open)`. Бар открылся выше стопа и пробил внутри → fill по уровню; бар открылся гэпом ниже стопа → fill по `open` (хуже стопа, честный гэп-риск). `min(уровень, open)` всегда ≥ `c.Low`, т.к. триггер означает `low ≤ уровень`, а `open ≥ low`. Не-стоповые выходы по-прежнему по `c.Close`.

- [ ] **Step 1: Добавить провальные тесты исполнения стопа**

В конец `engine_test.go` (после `TestEngineSuppliesDailyCloses`) добавить:

```go
// stopExitStrategy покупает на первом баре (когда flat), затем на каждом
// следующем баре отдаёт Sell с заданными reason и StopLoss.
type stopExitStrategy struct {
	reason   string
	stopLoss float64
}

func (s stopExitStrategy) Ticker() string  { return "TEST" }
func (s stopExitStrategy) Lookback() int    { return 1 }
func (s stopExitStrategy) Decide(md strategy.MarketData) model.Signal {
	if md.Position == nil {
		return model.Signal{Kind: model.SignalBuy}
	}
	return model.Signal{Kind: model.SignalSell, Reason: s.reason, StopLoss: s.stopLoss}
}

func TestEngineStopFillsAtLevelOnIntrabarPierce(t *testing.T) {
	base := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	candles := []Candle{
		{Time: base, Open: 100, High: 100, Low: 100, Close: 100, Volume: 1},                  // buy here
		{Time: base.Add(time.Hour), Open: 100, High: 100.5, Low: 98, Close: 98.5, Volume: 1}, // SL pierced intrabar
	}
	s := stopExitStrategy{reason: "SL", stopLoss: 99}
	res := Run(s, candles, nil, Config{InitialCash: 100000, Fraction: 1.0, Lot: 1})
	if len(res.Trades) != 1 {
		t.Fatalf("trades = %d, want 1", len(res.Trades))
	}
	if got := res.Trades[0].ExitPrice; got != 99 {
		t.Fatalf("exit = %v, want 99 (filled at stop level, not close 98.5)", got)
	}
}

func TestEngineStopFillsAtOpenOnGapDown(t *testing.T) {
	base := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	candles := []Candle{
		{Time: base, Open: 100, High: 100, Low: 100, Close: 100, Volume: 1},               // buy here
		{Time: base.Add(time.Hour), Open: 97, High: 97, Low: 96, Close: 96.5, Volume: 1},  // gap below stop
	}
	s := stopExitStrategy{reason: "SL", stopLoss: 99}
	res := Run(s, candles, nil, Config{InitialCash: 100000, Fraction: 1.0, Lot: 1})
	if len(res.Trades) != 1 {
		t.Fatalf("trades = %d, want 1", len(res.Trades))
	}
	if got := res.Trades[0].ExitPrice; got != 97 {
		t.Fatalf("exit = %v, want 97 (filled at gap open, worse than stop 99)", got)
	}
}

func TestEngineNonStopSellFillsAtClose(t *testing.T) {
	base := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	candles := []Candle{
		{Time: base, Open: 100, High: 100, Low: 100, Close: 100, Volume: 1},                  // buy here
		{Time: base.Add(time.Hour), Open: 100, High: 101, Low: 98, Close: 98.5, Volume: 1},   // TP-style exit
	}
	// StopLoss is set but must be ignored for a non-stop reason.
	s := stopExitStrategy{reason: "TP", stopLoss: 99}
	res := Run(s, candles, nil, Config{InitialCash: 100000, Fraction: 1.0, Lot: 1})
	if len(res.Trades) != 1 {
		t.Fatalf("trades = %d, want 1", len(res.Trades))
	}
	if got := res.Trades[0].ExitPrice; got != 98.5 {
		t.Fatalf("exit = %v, want 98.5 (non-stop sell fills at close)", got)
	}
}
```

- [ ] **Step 2: Запустить новые тесты — убедиться, что падают**

Run: `go test ./internal/domain/backtest/ -run 'TestEngineStopFillsAtLevelOnIntrabarPierce|TestEngineStopFillsAtOpenOnGapDown' -v`
Expected: FAIL — текущий код исполняет по `c.Close`, поэтому exit = 98.5 (не 99) и exit = 96.5 (не 97).

- [ ] **Step 3: Изменить ветку `SignalSell` в `Run()`**

В `engine.go` заменить (строки ~66-69):

```go
		case model.SignalSell:
			if p.qty != 0 {
				res.Trades = append(res.Trades, p.close(c.Close, c.Time, sig.Reason))
			}
```

на:

```go
		case model.SignalSell:
			if p.qty != 0 {
				exitPrice := c.Close
				// Stop exits fill at the stop level, adjusted for a gap-down open:
				// min(level, open) lands inside the bar (always >= c.Low) and charges
				// real gap risk when the bar opened below the stop.
				if sig.Reason == "SL" || sig.Reason == "TRAIL" {
					exitPrice = min(sig.StopLoss, c.Open)
				}
				res.Trades = append(res.Trades, p.close(exitPrice, c.Time, sig.Reason))
			}
```

- [ ] **Step 4: Запустить весь пакет backtest — все тесты зелёные**

Run: `go test ./internal/domain/backtest/ -v`
Expected: PASS (новые три теста + существующие; `TestEngineBuysFlatSellsInPosition` использует reason "TP" → по-прежнему fill по close 110).

- [ ] **Step 5: Commit**

```bash
git add internal/domain/backtest/engine.go internal/domain/backtest/engine_test.go
git commit -m "feat(backtest): fill stop exits at stop level with gap adjustment"
```

---

## Task 3: Документация — внутрибаровая семантика и live-контракт

**Files:**
- Modify: `docs/levels/strategy.md:98-118` (раздел 5) и live-контракт.

Контекст: спецификация требует зафиксировать семантику, чтобы будущий боевой раннер совпал с бэктестом. Кода live не пишем — только документируем контракт.

- [ ] **Step 1: Обновить раздел 5 (триггер по low) и добавить блок про исполнение + live**

В `docs/levels/strategy.md` заменить пункты 1 и 2 раздела 5 (строки 102-107):

```markdown
1. **Жёсткий стоп (SL).** `hardSL = цена_входа - SLMult * ATR`. Если **минимум бара**
   `low <= hardSL` — выход с причиной **SL**. Это страховка, если сразу пошло против.

2. **Chandelier-трейлинг (TRAIL).** `chandelier = (максимум за ChandelierWindow баров) -
   TrailMult * ATR`. Это стоп, который **подтягивается вверх** вслед за новыми максимумами и
   никогда не опускается. Если **минимум бара** `low <= chandelier` — выход **TRAIL**.
```

И сразу после блока «Логика выходов имеет приоритет…» (после строки 116, перед `---`) вставить:

```markdown

**Исполнение стопа (бэктест).** Триггер считается по `low`, а цена выхода = `min(уровень_стопа,
open_бара)`. Если бар открылся выше стопа и пробил его внутри — выход ровно по уровню стопа.
Если бар открылся гэпом ниже стопа — выход по `open` (хуже стопа): это честный учёт гэп-риска,
раньше прятавшийся за исполнением по цене закрытия.

**Live-контракт (на будущее, кода пока нет).** Стратегия каждый бар отдаёт `sig.StopLoss` как
уровень отложенной стоп-заявки; боевой раннер выставит стоп-маркет через Tinkoff `StopOrders` API
и будет двигать его вверх вслед за трейлингом. Оговорка: в реале гэп и проскальзывание означают,
что «убыток ≤ 1 ATR» — это цель/норма, а не математическая гарантия (в отличие от бэктеста без
slippage). Арминг трейла остаётся по цене закрытия.
```

- [ ] **Step 2: Commit**

```bash
git add docs/levels/strategy.md
git commit -m "docs(levels): document intrabar stop execution and live contract"
```

---

## Task 4: Перегнать RUAL-бэктест и сравнить отчёты

**Files:** нет правок кода — это верификация ожидаемого эффекта из спецификации.

Контекст: смена триггера и fill меняет число сделок и win-rate (стопы срабатывают раньше, по low). Это более реалистичный результат, не регрессия. Нужно сравнить хвосты убытков: ожидаем, что SL-выходы подтянутся к уровню (`поддержка − 1·ATR`), а −2.2% ужмутся до ~−1.x% (кроме гэпов).

- [ ] **Step 1: Запустить бэктест RUAL на Hour1**

Run: `go run ./cmd/backtest -ticker RUAL -strategy levels -interval Hour1`

Примечание: команда тянет свечи через Tinkoff API (нужен токен в `env/local.env` / окружении); при наличии кэша в `data/candles` повторная загрузка не требуется. Новый отчёт появится в `reports/` под именем `RUAL_levels_Hour1_<stamp>.md` (+ `_trades.csv`, `_equity.csv`).

- [ ] **Step 2: Сравнить с предыдущим отчётом**

Сравнить новый `reports/RUAL_levels_Hour1_<новый stamp>.md` со старым
`reports/level/RUAL_levels_Hour1_20260607_085846.md`:
- Хвосты убыточных сделок (`*_trades.csv`): были −1.3% / −2.2% при «стопе в 1 ATR» — должны ужаться к ~−1.x% для не-гэповых выходов.
- WinRate / число сделок / Expectancy изменятся — это ожидаемо (стопы срабатывают раньше).
- Гэп-выходы могут по-прежнему превышать 1 ATR — это корректно.

Зафиксировать наблюдения (в ответе пользователю или в заметку), решения по параметрам — вне рамок этой задачи.

---

## Вне рамок (YAGNI)

- Take-profit как выход (его сейчас нет).
- Отдельное моделирование slippage.
- Сам live-раннер и интеграция со `StopOrders`.
- Перенос стопа на базу входа (`вход − 1·ATR`) — сознательно отклонено в спецификации.
