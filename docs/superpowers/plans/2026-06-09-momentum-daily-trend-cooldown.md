# Momentum daily-trend filter + cooldown — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Оживить два мёртвых параметра momentum-стратегии — `DailyTrendPeriod` (дневной трендовый фильтр по наклону EMA) и `CooldownBars` (пауза после выхода) — чтобы убрать кластер из 6 убытков подряд, дающий всю просадку RUAL.

**Architecture:** Дневной фильтр — чистый, считается в `buildInput` из `md.DailyCloses` и проверяется гейтом в `decide()`. Кулдаун требует состояния: счётчик `barsSinceExit` живёт в `Strategy` (грязная оболочка), обновляется в `Decide`, передаётся в `decideInput` — `decide()` остаётся чистой функцией. Оба фильтра выключены при значении `0`, что сохраняет обратную совместимость с текущим грид-победителем и замороженным baseline-тестом.

**Tech Stack:** Go 1.25, стандартный `testing`. Затрагивается один пакет `internal/service/trading_strategy/momentum/strategy/core`, плюс docs и грид-json.

**Spec:** `docs/superpowers/specs/2026-06-09-momentum-daily-trend-cooldown-design.md`

---

## File Structure

- `internal/service/trading_strategy/momentum/strategy/core/core.go` — добавить константы, поля `decideInput`, расчёт в `buildInput`, два гейта в `decide()` и `Explain()`, состояние в `Strategy` + обновление в `Decide`, обновить докстринги.
- `internal/service/trading_strategy/momentum/strategy/core/core_test.go` — новые table-driven тесты; существующие тесты не трогаем.
- `docs/momentum/strategy.md` — обновить таблицу гейтов, таблицу параметров, секцию «заготовок».
- `data/params/rusal/momentum_grid.json` — добавить sweep по `DailyTrendPeriod` (шаг калибровки).
- `internal/service/trading_strategy/momentum/strategy/rusal/rusal.go` — `DailyTrendPeriod` проставляется после калибровки (Task 6, вне кода этой итерации).

**Факты для реализации (проверены в коде):**
- `ema.Compute(closes []float64, period int) []float64` — длина = длине входа, позиции до `period-1` нулевые, сид = SMA первых `period` значений (`internal/domain/ema/compute.go`).
- `decide()` первым делом делает `if in.pos != nil { return s.manage(...) }`, поэтому к entry-гейтам `in.pos` всегда `nil` — гейту кулдауна проверка позиции не нужна.
- `defaultParams()` в тестах не задаёт `DailyTrendPeriod`/`CooldownBars` → они `0` → существующие тесты остаются зелёными без правок.
- Замороженный baseline-тест `TestGenericMomentumDefaultsAreFrozenBaseline` пинит `CooldownBars: 0, DailyTrendPeriod: 0` — он остаётся зелёным, т.к. дефолты не меняются.

---

## Task 1: Дневной трендовый фильтр — расчёт и гейт в `decide()`

**Files:**
- Modify: `internal/service/trading_strategy/momentum/strategy/core/core.go`
- Test: `internal/service/trading_strategy/momentum/strategy/core/core_test.go`

- [ ] **Step 1: Написать падающие тесты**

Добавить в конец `core_test.go`:

```go
func TestEntryFiresWithRisingDailyTrend(t *testing.T) {
	p := defaultParams()
	p.DailyTrendPeriod = 5 // дневные closes в buildEntryMD растут -> наклон вверх
	s := NewWithParams("TEST", p)
	if sig := s.Decide(buildEntryMD()); sig.Kind != model.SignalBuy {
		t.Fatal("entry should fire when daily EMA slope is up")
	}
}

func TestEntryBlockedByDailyTrendFilter(t *testing.T) {
	md := buildEntryMD()
	for i := range md.DailyCloses { // плоские дневные closes -> EMA не растёт
		md.DailyCloses[i] = 110
	}
	p := defaultParams()
	p.DailyTrendPeriod = 5
	s := NewWithParams("TEST", p)
	if sig := s.Decide(md); sig.Kind == model.SignalBuy {
		t.Fatal("entry should be blocked when daily EMA is not rising")
	}
}

func TestDailyTrendFilterDisabledIgnoresSlope(t *testing.T) {
	md := buildEntryMD()
	for i := range md.DailyCloses { // падающие дневные closes
		md.DailyCloses[i] = 200 - float64(i)
	}
	p := defaultParams() // DailyTrendPeriod = 0 -> фильтр выключен
	s := NewWithParams("TEST", p)
	if sig := s.Decide(md); sig.Kind != model.SignalBuy {
		t.Fatal("entry should fire when daily filter is disabled, regardless of slope")
	}
}

func TestDailyTrendFilterPassesWithInsufficientHistory(t *testing.T) {
	md := buildEntryMD()
	md.DailyCloses = md.DailyCloses[:5] // меньше, чем period+slopeBars (5+3=8)
	md.DailyHighs = md.DailyHighs[:5]
	md.DailyLows = md.DailyLows[:5]
	p := defaultParams()
	p.DailyTrendPeriod = 5
	s := NewWithParams("TEST", p)
	if sig := s.Decide(md); sig.Kind != model.SignalBuy {
		t.Fatal("entry should fire (filter passes) when daily history is insufficient")
	}
}
```

- [ ] **Step 2: Запустить — убедиться, что падает на компиляции/логике**

Run: `go test ./internal/service/trading_strategy/momentum/strategy/core/ -run 'DailyTrend' -v`
Expected: FAIL — без гейта `TestEntryBlockedByDailyTrendFilter` не блокирует вход (тест падает).

- [ ] **Step 3: Добавить константу наклона**

В `core.go`, сразу после блока `import` (перед `type Params struct`), добавить:

```go
// dailyTrendSlopeBars is the fixed horizon (in completed daily candles) over which
// the daily-EMA slope filter measures direction. Held as an in-package policy
// constant (not a grid knob) to keep calibration combinatorics small.
const dailyTrendSlopeBars = 3
```

- [ ] **Step 4: Расширить `decideInput`**

В `core.go` в `type decideInput struct` добавить поля (после `recentHigh float64`):

```go
	dailyEMANow     float64 // last daily-EMA value (0 if unavailable)
	dailyEMAPast    float64 // daily-EMA value dailyTrendSlopeBars back (0 if unavailable)
	dailyTrendKnown bool    // true when daily history sufficed to compute both points
```

- [ ] **Step 5: Считать дневную EMA в `buildInput`**

В `core.go` в `buildInput`, перед `return decideInput{...}`, добавить:

```go
	dailyEMANow, dailyEMAPast, dailyTrendKnown := 0.0, 0.0, false
	if s.p.DailyTrendPeriod > 0 {
		if de := ema.Compute(md.DailyCloses, s.p.DailyTrendPeriod); len(de) >= s.p.DailyTrendPeriod+dailyTrendSlopeBars {
			n := len(de)
			dailyEMANow, dailyEMAPast, dailyTrendKnown = de[n-1], de[n-1-dailyTrendSlopeBars], true
		}
	}
```

И в литерале `return decideInput{...}` добавить поля:

```go
		dailyEMANow:     dailyEMANow,
		dailyEMAPast:    dailyEMAPast,
		dailyTrendKnown: dailyTrendKnown,
```

- [ ] **Step 6: Добавить гейт в `decide()`**

В `core.go` в `decide()`, сразу после часового uptrend-гейта (после блока `if !(in.emaTrend > 0 && in.price > in.emaTrend) { return sig }`), добавить:

```go
	if s.p.DailyTrendPeriod > 0 && in.dailyTrendKnown && !(in.dailyEMANow > in.dailyEMAPast) {
		return sig // daily trend not rising
	}
```

- [ ] **Step 7: Запустить новые тесты — должны пройти**

Run: `go test ./internal/service/trading_strategy/momentum/strategy/core/ -run 'DailyTrend' -v`
Expected: PASS (все 4 теста).

- [ ] **Step 8: Прогнать весь пакет — регрессий нет**

Run: `go test ./internal/service/trading_strategy/momentum/...`
Expected: PASS (старые тесты зелёные, т.к. `DailyTrendPeriod=0`).

- [ ] **Step 9: Коммит**

```bash
git add internal/service/trading_strategy/momentum/strategy/core/core.go internal/service/trading_strategy/momentum/strategy/core/core_test.go
git commit -m "feat(momentum): daily-EMA slope trend filter (DailyTrendPeriod)

Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>"
```

---

## Task 2: Дневной фильтр — зеркало в `Explain()`

**Files:**
- Modify: `internal/service/trading_strategy/momentum/strategy/core/core.go`
- Test: `internal/service/trading_strategy/momentum/strategy/core/core_test.go`

- [ ] **Step 1: Написать падающий тест на диагностику**

Добавить в `core_test.go`:

```go
func TestExplainReportsDailyTrendBlock(t *testing.T) {
	md := buildEntryMD()
	for i := range md.DailyCloses {
		md.DailyCloses[i] = 110 // плоско -> фильтр блокирует
	}
	p := defaultParams()
	p.DailyTrendPeriod = 5
	s := NewWithParams("TEST", p)
	out := s.Explain(md)
	if !strings.Contains(out, "Дневной тренд") {
		t.Fatalf("Explain should mention daily trend gate, got: %q", out)
	}
}
```

- [ ] **Step 2: Запустить — убедиться, что падает**

Run: `go test ./internal/service/trading_strategy/momentum/strategy/core/ -run 'TestExplainReportsDailyTrendBlock' -v`
Expected: FAIL — `Explain` ещё не упоминает дневной тренд.

- [ ] **Step 3: Добавить гейт в `Explain()`**

В `core.go` в `Explain()`, сразу после блока `pass("Тренд: close %.4f > EMA%d %.4f", ...)` (gate 1) и перед `// 2. MACD bullish cross`, добавить:

```go
	// 1b. Daily trend slope.
	if s.p.DailyTrendPeriod > 0 {
		switch {
		case !in.dailyTrendKnown:
			pass("Дневной тренд: недостаточно дневной истории — фильтр пропущен")
		case !(in.dailyEMANow > in.dailyEMAPast):
			return block("Дневной тренд: EMA%d не растёт (%.4f ≤ %.4f, %d дн назад)",
				s.p.DailyTrendPeriod, in.dailyEMANow, in.dailyEMAPast, dailyTrendSlopeBars)
		default:
			pass("Дневной тренд: EMA%d растёт (%.4f > %.4f)", s.p.DailyTrendPeriod, in.dailyEMANow, in.dailyEMAPast)
		}
	}
```

- [ ] **Step 4: Запустить тест — должен пройти**

Run: `go test ./internal/service/trading_strategy/momentum/strategy/core/ -run 'TestExplainReportsDailyTrendBlock' -v`
Expected: PASS.

- [ ] **Step 5: Коммит**

```bash
git add internal/service/trading_strategy/momentum/strategy/core/core.go internal/service/trading_strategy/momentum/strategy/core/core_test.go
git commit -m "feat(momentum): surface daily-trend gate in Explain

Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>"
```

---

## Task 3: Кулдаун — состояние, трекинг и гейт в `decide()`

**Files:**
- Modify: `internal/service/trading_strategy/momentum/strategy/core/core.go`
- Test: `internal/service/trading_strategy/momentum/strategy/core/core_test.go`

- [ ] **Step 1: Написать падающие тесты**

Добавить в `core_test.go`:

```go
func TestCooldownBlocksReentryAfterExit(t *testing.T) {
	p := defaultParams()
	p.CooldownBars = 3
	s := NewWithParams("TEST", p)

	// Бар 1: открываем позицию (md.Position == nil).
	if sig := s.Decide(buildEntryMD()); sig.Kind != model.SignalBuy {
		t.Fatal("expected entry on bar 1")
	}
	// Бар 2: в позиции, ловим SL -> выход.
	pos := &strategy.Position{PurchasePrice: 100, StopLoss: 95, EntryATR: 1, MaxFavorablePrice: 100}
	if sig := s.Decide(inPositionMD(94, 101, 100, pos)); sig.Reason != "SL" {
		t.Fatalf("expected SL exit, got %q", sig.Reason)
	}
	// Бары 3-5: flat, кулдаун активен (barsSinceExit 0,1,2 < 3) -> вход блокируется.
	for i := 0; i < 3; i++ {
		if sig := s.Decide(buildEntryMD()); sig.Kind == model.SignalBuy {
			t.Fatalf("entry should be blocked during cooldown, offset %d", i)
		}
	}
	// Бар 6: кулдаун истёк (barsSinceExit 3 >= 3) -> вход снова разрешён.
	if sig := s.Decide(buildEntryMD()); sig.Kind != model.SignalBuy {
		t.Fatal("entry should fire after cooldown elapses")
	}
}

func TestCooldownDisabledAllowsImmediateReentry(t *testing.T) {
	p := defaultParams() // CooldownBars = 0
	s := NewWithParams("TEST", p)
	if sig := s.Decide(buildEntryMD()); sig.Kind != model.SignalBuy {
		t.Fatal("expected entry on bar 1")
	}
	pos := &strategy.Position{PurchasePrice: 100, StopLoss: 95, EntryATR: 1, MaxFavorablePrice: 100}
	s.Decide(inPositionMD(94, 101, 100, pos)) // выход по SL
	if sig := s.Decide(buildEntryMD()); sig.Kind != model.SignalBuy {
		t.Fatal("entry should fire immediately when cooldown disabled")
	}
}
```

- [ ] **Step 2: Запустить — убедиться, что падает**

Run: `go test ./internal/service/trading_strategy/momentum/strategy/core/ -run 'Cooldown' -v`
Expected: FAIL — `TestCooldownBlocksReentryAfterExit` не блокирует (нет гейта/счётчика).

- [ ] **Step 3: Добавить константу-насыщение**

В `core.go` расширить блок констант (рядом с `dailyTrendSlopeBars`) — заменить одиночную const на блок:

```go
const (
	// dailyTrendSlopeBars is the fixed horizon (in completed daily candles) over which
	// the daily-EMA slope filter measures direction. Held as an in-package policy
	// constant (not a grid knob) to keep calibration combinatorics small.
	dailyTrendSlopeBars = 3
	// cooldownSaturate seeds and caps barsSinceExit so it never overflows and starts
	// "cooldown satisfied" (no entry blocked before the first exit).
	cooldownSaturate = 1 << 30
)
```

- [ ] **Step 4: Добавить состояние в `Strategy` и инициализацию**

В `core.go` заменить `type Strategy struct { ... }` на:

```go
// Strategy trades a single instrument with the momentum rules. Ticker-agnostic.
// It carries the cooldown counter as mutable state in the impure shell; the pure
// decide() core stays a function of its input.
type Strategy struct {
	ticker         string
	p              Params
	barsSinceExit  int  // bars elapsed since the last exit; gates re-entry
	prevInPosition bool // whether the previous Decide saw an open position
}
```

И заменить `NewWithParams` на:

```go
// NewWithParams returns the momentum strategy for a ticker with explicit params.
func NewWithParams(ticker string, p Params) *Strategy {
	return &Strategy{ticker: ticker, p: p, barsSinceExit: cooldownSaturate}
}
```

- [ ] **Step 5: Обновлять счётчик в `Decide` и пробрасывать в input**

В `core.go` заменить `Decide` на:

```go
// Decide computes every indicator from md, packs them, and delegates to the pure core.
func (s *Strategy) Decide(md strategy.MarketData) model.Signal {
	s.trackCooldown(md.Position)
	sig := s.decide(s.buildInput(md))
	sig.Ticker = s.ticker
	return sig
}

// trackCooldown advances the post-exit bar counter. It detects the in-position ->
// flat edge to reset the counter, then increments while flat. Called once per bar
// from Decide (the trading path); Explain never mutates this state.
func (s *Strategy) trackCooldown(pos *strategy.Position) {
	switch {
	case pos != nil:
		// In a position: cooldown is irrelevant, leave the counter as-is.
	case s.prevInPosition:
		s.barsSinceExit = 0 // just exited this bar
	case s.barsSinceExit < cooldownSaturate:
		s.barsSinceExit++
	}
	s.prevInPosition = pos != nil
}
```

В `buildInput`, в литерале `return decideInput{...}`, добавить поле:

```go
		barsSinceExit: s.barsSinceExit,
```

И в `type decideInput struct` добавить поле (после `dailyTrendKnown bool`):

```go
	barsSinceExit int // bars since the last exit, for the cooldown gate
```

- [ ] **Step 6: Добавить гейт кулдауна в `decide()`**

В `core.go` в `decide()`, как **первый** entry-гейт — сразу после `if in.pos != nil { return s.manage(in, sig) }` и перед `// Entry gates (all must pass).`, добавить:

```go
	if s.p.CooldownBars > 0 && in.barsSinceExit < s.p.CooldownBars {
		return sig // still cooling down after the last exit
	}
```

- [ ] **Step 7: Запустить тесты кулдауна — должны пройти**

Run: `go test ./internal/service/trading_strategy/momentum/strategy/core/ -run 'Cooldown' -v`
Expected: PASS (оба теста).

- [ ] **Step 8: Прогнать весь пакет**

Run: `go test ./internal/service/trading_strategy/momentum/...`
Expected: PASS.

- [ ] **Step 9: Коммит**

```bash
git add internal/service/trading_strategy/momentum/strategy/core/core.go internal/service/trading_strategy/momentum/strategy/core/core_test.go
git commit -m "feat(momentum): post-exit cooldown gate (CooldownBars)

Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>"
```

---

## Task 4: Кулдаун — отображение в `Explain()`

**Files:**
- Modify: `internal/service/trading_strategy/momentum/strategy/core/core.go`
- Test: `internal/service/trading_strategy/momentum/strategy/core/core_test.go`

- [ ] **Step 1: Написать падающий тест**

Добавить в `core_test.go`:

```go
func TestExplainReportsCooldownBlock(t *testing.T) {
	p := defaultParams()
	p.CooldownBars = 3
	s := NewWithParams("TEST", p)
	// Открыть и закрыть позицию, чтобы barsSinceExit обнулился.
	s.Decide(buildEntryMD())
	pos := &strategy.Position{PurchasePrice: 100, StopLoss: 95, EntryATR: 1, MaxFavorablePrice: 100}
	s.Decide(inPositionMD(94, 101, 100, pos)) // выход
	// Сразу после выхода кулдаун активен -> Explain должен это показать.
	out := s.Explain(buildEntryMD())
	if !strings.Contains(out, "Кулдаун") {
		t.Fatalf("Explain should mention cooldown gate, got: %q", out)
	}
}
```

- [ ] **Step 2: Запустить — убедиться, что падает**

Run: `go test ./internal/service/trading_strategy/momentum/strategy/core/ -run 'TestExplainReportsCooldownBlock' -v`
Expected: FAIL — `Explain` ещё не упоминает кулдаун.

- [ ] **Step 3: Добавить гейт кулдауна в `Explain()`**

В `core.go` в `Explain()`, сразу после блока `if in.pos != nil { return ... }` и перед `var b strings.Builder` — нет, `b` используется в гейте. Вставить **после** `block := func(...) string {...}` определения и **перед** `// 1. Uptrend.`:

```go
	// 0. Cooldown.
	if s.p.CooldownBars > 0 {
		if in.barsSinceExit < s.p.CooldownBars {
			return block("Кулдаун: после выхода прошло %d бар(ов) из %d", in.barsSinceExit, s.p.CooldownBars)
		}
		pass("Кулдаун: после выхода прошло %d бар(ов) ≥ %d", in.barsSinceExit, s.p.CooldownBars)
	}
```

Примечание: `in.barsSinceExit` берётся из `buildInput`, который читает текущее `s.barsSinceExit`. `Explain` не вызывает `trackCooldown`, поэтому состояние не мутируется — диагностика остаётся read-only.

- [ ] **Step 4: Запустить тест — должен пройти**

Run: `go test ./internal/service/trading_strategy/momentum/strategy/core/ -run 'TestExplainReportsCooldownBlock' -v`
Expected: PASS.

- [ ] **Step 5: Полный прогон пакета + go vet**

Run: `go test ./internal/service/trading_strategy/momentum/... && go vet ./internal/service/trading_strategy/momentum/...`
Expected: PASS, без замечаний vet.

- [ ] **Step 6: Коммит**

```bash
git add internal/service/trading_strategy/momentum/strategy/core/core.go internal/service/trading_strategy/momentum/strategy/core/core_test.go
git commit -m "feat(momentum): surface cooldown gate in Explain

Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>"
```

---

## Task 5: Обновить докстринги и документацию

**Files:**
- Modify: `internal/service/trading_strategy/momentum/strategy/core/core.go`
- Modify: `docs/momentum/strategy.md`

- [ ] **Step 1: Убрать пометки `reserved` в `Params`**

В `core.go` в `type Params struct` заменить две строки:

```go
	CooldownBars      int     // reserved; not yet enforced
	DailyTrendPeriod  int     // reserved; not yet enforced
```

на:

```go
	CooldownBars      int     // bars after an exit before a new entry is allowed (0 disables)
	DailyTrendPeriod  int     // daily-EMA period for the higher-timeframe slope filter (0 disables)
```

- [ ] **Step 2: Обновить докстринг пакета**

В `core.go` в комментарии пакета (строки 1-7) после предложения про exits добавить упоминание новых фильтров. Заменить фразу `MACD just crossed up (optionally below zero), volume is above its recent average, and the day still has room left within its typical daily-ATR range.` на:

```
// MACD just crossed up (optionally below zero), volume is above its recent
// average, the higher-timeframe daily EMA is sloping up, the day still has room
// left within its typical daily-ATR range, and any post-exit cooldown has elapsed.
```

- [ ] **Step 3: Прогнать тесты (докстринги не ломают сборку, но проверим)**

Run: `go build ./... && go test ./internal/service/trading_strategy/momentum/...`
Expected: PASS.

- [ ] **Step 4: Обновить таблицу гейтов в `docs/momentum/strategy.md`**

В секции `## 2. Когда стратегия ПОКУПАЕТ (вход)` добавить строку дневного фильтра после строки про тренд (gate 1) и строку кулдауна. Конкретно добавить в таблицу:

```
| 1б | **Дневной тренд растёт:** дневная EMA(`DailyTrendPeriod`) сегодня выше, чем 3 дня назад (при `DailyTrendPeriod>0`). | Старший таймфрейм подтверждает тренд — не входим, когда дневной тренд выполаживается/разворачивается. |
```

И после таблицы (рядом со строкой про анти-черн `| — |`) добавить:

```
| — | **Кулдаун:** после выхода должно пройти ≥ `CooldownBars` баров (при `CooldownBars>0`). | Не перезаходим сразу в тот же ломающийся диапазон после стопа. |
```

- [ ] **Step 5: Обновить таблицу параметров в `docs/momentum/strategy.md`**

В секции `## 5. Параметры стратегии` заменить строку:

```
| `DailyTrendPeriod` | 0 | Заготовка: доп. фильтр «выше дневной EMA» (0 = выкл). |
```

на:

```
| `DailyTrendPeriod` | 0 | Период дневной EMA для фильтра по наклону: вход только если EMA растёт за 3 дня (0 = выкл). |
| `CooldownBars` | 0 | Сколько баров после выхода вход заблокирован (0 = выкл). |
```

- [ ] **Step 6: Обновить секцию «заготовок»**

В `docs/momentum/strategy.md` в секции `## 8. Заготовки на будущее (выключены по умолчанию)` убрать пункт про `DailyTrendPeriod` (он теперь реализован), оставив только всё ещё не реализованные заготовки. Строку `- **`DailyTrendPeriod`** — доп. конфлюэнс: цена выше дневной EMA.` удалить.

- [ ] **Step 7: Коммит**

```bash
git add internal/service/trading_strategy/momentum/strategy/core/core.go docs/momentum/strategy.md
git commit -m "docs(momentum): document daily-trend filter and cooldown gates

Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>"
```

---

## Task 6: Калибровка — измерить дневной фильтр, решить по кулдауну (решение C)

> Ручной шаг пользователя (бэктест/грид гоняет он сам). Кодовые правки — только хардкод победителя в `rusal.go` после прогона.

**Files:**
- Modify: `data/params/rusal/momentum_grid.json`
- Modify (после прогона): `internal/service/trading_strategy/momentum/strategy/rusal/rusal.go`

- [ ] **Step 1: Добавить sweep по `DailyTrendPeriod` в грид-json**

В `data/params/rusal/momentum_grid.json` добавить ключ (умножает комбинаторику ×3 — приемлемо; кулдаун в грид НЕ добавляем по решению C):

```json
  "DailyTrendPeriod": [0, 10, 20]
```

- [ ] **Step 2: Прогнать калибровку (пользователь)**

Run: `go run ./cmd/backtest -ticker RUAL -strategy momentum -calibrate`
Expected: новый `reports/.../_calibration.md` и `_best.md`. Сравнить best с текущим (PF 1.90, DD 10.9%, 23 сделки) — проверить, ужалась ли просадка и уцелел ли кластер №9–14 (по журналу сделок в `_best.md`).

- [ ] **Step 3: Захардкодить победителя в `rusal.go`**

Обновить `DefaultParams()` в `rusal.go` значениями из `_best.md` (включая `DailyTrendPeriod`). Если кластер убытков ушёл — `CooldownBars` оставить `0`. Если кластер уцелел — отдельно прогнать с `CooldownBars` 3/5/8 (через `-params '{"CooldownBars":5}'` или временно в гриде) и взять лучший.

- [ ] **Step 4: Прогнать тесты после хардкода**

Run: `go test ./internal/service/...`
Expected: PASS. (Замороженный baseline-тест проверяет `genericMomentumDefaults`, а не `rusal.DefaultParams`, поэтому правка RUAL его не задевает.)

- [ ] **Step 5: Коммит**

```bash
git add data/params/rusal/momentum_grid.json internal/service/trading_strategy/momentum/strategy/rusal/rusal.go
git commit -m "feat(momentum): calibrate RUAL with daily-trend filter

Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>"
```

---

## Self-Review (заполнено автором плана)

- **Покрытие спека:** дневной фильтр (Task 1-2), кулдаун (Task 3-4), докстринги/docs (Task 5), последовательность калибровки C (Task 6), обратная совместимость (проверяется в Step 8 каждой кодовой задачи + baseline-тест не трогается). ✓
- **Плейсхолдеры:** код во всех кодовых шагах полный; Task 6 — ручной, значения берутся из отчёта (это нормально для калибровки). ✓
- **Согласованность типов:** `dailyEMANow/dailyEMAPast/dailyTrendKnown/barsSinceExit` объявлены в Task 1/3 и используются единообразно; `dailyTrendSlopeBars`/`cooldownSaturate` — в блоке const; `trackCooldown` определён и вызван в `Decide`. ✓
