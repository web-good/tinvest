# Levels: ATR и уровни в отчёте + калибровочный грид — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** В журнал сделок бэктеста добавить уровень входа (HVN-support), цель (HVN-resistance) и ATR на входе; переориентировать калибровочный грид RUAL на знобы ширины стопа/трейла.

**Architecture:** Контекст входа известен в `decide()` на Buy, но `Trade` создаётся на выходе. Протаскиваем контекст: `Signal.Level/ATR` (resistance = существующий `TakeProfit`) → `portfolio.open(... level, target, atr)` запоминает → `close` штампует в `Trade` → рендер выводит в Markdown + CSV. Грид — это data-файл, меняется отдельно.

**Tech Stack:** Go 1.25, стандартный `testing`. Затрагиваются пакеты `scalping/model`, `levels/strategy/core`, `internal/domain/backtest` и data/docs.

**Спецификация:** `docs/superpowers/specs/2026-06-07-levels-report-context-and-calibration-design.md`

---

## Структура файлов

- `internal/service/trading_strategy/scalping/model/signal.go` — **изменить**: добавить поля `Level`, `ATR` в `Signal`.
- `internal/service/trading_strategy/levels/strategy/core/core.go` — **изменить** `decide()`: в Buy-ветке выставить `sig.Level`, `sig.ATR`.
- `internal/service/trading_strategy/levels/strategy/core/core_test.go` — **добавить** тест на `Level`/`ATR` в Buy.
- `internal/domain/backtest/types.go` — **изменить** `Trade`: поля `SupportLevel`, `ResistanceLevel`, `ATR`.
- `internal/domain/backtest/portfolio.go` — **изменить** `open` (сигнатура + поля) и `close` (штамп).
- `internal/domain/backtest/engine.go` — **изменить** Buy-ветку `Run()`: передать контекст из сигнала.
- `internal/domain/backtest/engine_test.go` — **добавить** тест штампа контекста на `Trade`.
- `internal/domain/backtest/report.go` — **изменить** Markdown-журнал и `RenderTradesCSV`.
- `internal/domain/backtest/report_test.go` — **изменить** тесты заголовков/строк.
- `data/params/rusal/levels_grid.json` — **переписать** грид.
- `docs/levels/strategy.md` — **изменить**: команда калибровки + новые колонки.

---

## Task 1: Уровень входа и ATR в сигнале стратегии

**Files:**
- Modify: `internal/service/trading_strategy/scalping/model/signal.go`
- Modify: `internal/service/trading_strategy/levels/strategy/core/core.go`
- Modify/Test: `internal/service/trading_strategy/levels/strategy/core/core_test.go`

Контекст: `Signal` уже несёт `Price/TakeProfit/StopLoss/RSI/Reason`. Добавляем `Level` (HVN-support, от которого вошли) и `ATR` (на момент входа). В `decide()` Buy-ветка уже считает `support` и `in.atr` — кладём их в сигнал. Resistance уже едет в `sig.TakeProfit`.

- [ ] **Step 1: Написать падающий тест**

В `core_test.go` сразу после `TestDecideBounceBuy` (заканчивается на строке с `}` после проверки TakeProfit) добавить:

```go
func TestDecideBuySetsLevelAndATR(t *testing.T) {
	s := newCore()
	sig := s.decide(bounceInput())
	if sig.Kind != model.SignalBuy {
		t.Fatalf("want Buy, got %v", sig.Kind)
	}
	if sig.Level != 100 {
		t.Errorf("level = %v, want support 100", sig.Level)
	}
	if sig.ATR != 1.0 {
		t.Errorf("atr = %v, want 1.0", sig.ATR)
	}
}
```

- [ ] **Step 2: Запустить — убедиться, что не компилируется / падает**

Run: `go test ./internal/service/trading_strategy/levels/strategy/core/ -run TestDecideBuySetsLevelAndATR -v`
Expected: FAIL — у `model.Signal` ещё нет полей `Level`/`ATR` (ошибка компиляции `unknown field`).

- [ ] **Step 3: Добавить поля в `Signal`**

В `internal/service/trading_strategy/scalping/model/signal.go` заменить определение структуры:

```go
// Signal is a rendered trade alert for one instrument.
type Signal struct {
	Kind           SignalKind
	InstrumentID   string
	InstrumentName string
	Ticker         string
	Price          float64
	TakeProfit     float64
	StopLoss       float64
	RSI            float64
	Level          float64 // entry support level (HVN); 0 when n/a
	ATR            float64 // ATR at entry; 0 when n/a
	Reason         string // "TP" or "SL" for sells
}
```

- [ ] **Step 4: Выставить уровень и ATR в Buy-ветке `decide()`**

В `internal/service/trading_strategy/levels/strategy/core/core.go` в блоке входа заменить:

```go
		if s.entryQualifies(in.price, stop, target, in.atr) {
			sig.Kind, sig.StopLoss, sig.TakeProfit = model.SignalBuy, stop, target
			if in.recentlyBelow {
				sig.Reason = "RETEST"
			} else {
				sig.Reason = "BOUNCE"
			}
		}
```

на:

```go
		if s.entryQualifies(in.price, stop, target, in.atr) {
			sig.Kind, sig.StopLoss, sig.TakeProfit = model.SignalBuy, stop, target
			sig.Level, sig.ATR = support.Price, in.atr
			if in.recentlyBelow {
				sig.Reason = "RETEST"
			} else {
				sig.Reason = "BOUNCE"
			}
		}
```

- [ ] **Step 5: Запустить пакеты core и model — зелёные**

Run: `go test ./internal/service/trading_strategy/levels/strategy/core/ ./internal/service/trading_strategy/scalping/model/ -v`
Expected: PASS (новый тест + все существующие).

- [ ] **Step 6: Commit**

```bash
git add internal/service/trading_strategy/scalping/model/signal.go internal/service/trading_strategy/levels/strategy/core/core.go internal/service/trading_strategy/levels/strategy/core/core_test.go
git commit -m "feat(levels): carry entry support level and ATR in the signal"
```

---

## Task 2: Протащить контекст входа в Trade через портфель и движок

**Files:**
- Modify: `internal/domain/backtest/types.go`
- Modify: `internal/domain/backtest/portfolio.go`
- Modify: `internal/domain/backtest/engine.go:60-63`
- Test: `internal/domain/backtest/engine_test.go`

Контекст: `portfolio.open(price, t)` сейчас не принимает контекст входа, а `Trade` его не хранит. Расширяем `open` до `open(price, t, level, target, atr)`, запоминаем в полях портфеля, штампуем в `Trade` при `close`. Движок передаёт значения из сигнала. Изменение сигнатуры `open` — атомарное (ломает компиляцию `engine.go`, поэтому правки types/portfolio/engine идут вместе).

- [ ] **Step 1: Написать падающий тест штампа контекста**

В конец `internal/domain/backtest/engine_test.go` добавить:

```go
func TestEngineStampsEntryContextOnTrade(t *testing.T) {
	candles := flatCandles([]float64{10, 100, 110})
	s := scriptedStrategy{lookback: 1, decide: func(md strategy.MarketData) model.Signal {
		if md.Position == nil && md.Price == 100 {
			return model.Signal{Kind: model.SignalBuy, Level: 98, TakeProfit: 108, ATR: 1.5}
		}
		if md.Position != nil && md.Price == 110 {
			return model.Signal{Kind: model.SignalSell, Reason: "TP"}
		}
		return model.Signal{Kind: model.SignalNone}
	}}
	res := Run(s, candles, nil, Config{InitialCash: 100000, Fraction: 1.0, Lot: 1})
	if len(res.Trades) != 1 {
		t.Fatalf("trades = %d, want 1", len(res.Trades))
	}
	tr := res.Trades[0]
	if tr.SupportLevel != 98 || tr.ResistanceLevel != 108 || tr.ATR != 1.5 {
		t.Fatalf("entry context = {%v %v %v}, want {98 108 1.5}", tr.SupportLevel, tr.ResistanceLevel, tr.ATR)
	}
}
```

- [ ] **Step 2: Запустить — убедиться, что не компилируется**

Run: `go test ./internal/domain/backtest/ -run TestEngineStampsEntryContextOnTrade -v`
Expected: FAIL — у `Trade` нет полей `SupportLevel/ResistanceLevel/ATR` (ошибка компиляции).

- [ ] **Step 3: Добавить поля в `Trade`**

В `internal/domain/backtest/types.go` заменить определение `Trade`:

```go
// Trade is one completed round-trip (entry -> exit).
type Trade struct {
	EntryTime       time.Time
	EntryPrice      float64
	ExitTime        time.Time
	ExitPrice       float64
	Quantity        int64   // shares (lots * Lot)
	Reason          string  // exit reason: "SL" / "TRAIL" / "TP"
	PnL             float64 // net of commission, in currency
	PnLPct          float64 // PnL relative to entry cost
	BarsHeld        int
	SupportLevel    float64 // HVN support the entry bounced off; 0 when n/a
	ResistanceLevel float64 // HVN resistance / target at entry; 0 when n/a
	ATR             float64 // ATR at entry; 0 when n/a
}
```

- [ ] **Step 4: Запомнить контекст в `portfolio.open` и обнулять/штамповать в `close`**

В `internal/domain/backtest/portfolio.go` заменить определение структуры `portfolio`:

```go
// portfolio is the long-only mock account the engine trades against.
type portfolio struct {
	cfg         Config
	cash        float64
	qty         int64
	entryPrice  float64
	entryTime   time.Time
	entryBar    int
	entryLevel  float64 // support level captured at entry
	entryTarget float64 // resistance/target captured at entry
	entryATR    float64 // ATR captured at entry
	bar         int // current bar index, set by the engine each iteration
}
```

Заменить сигнатуру и хвост `open` (математику лотов/кэша НЕ трогаем):

```go
func (p *portfolio) open(price float64, t time.Time, level, target, atr float64) {
	if p.qty != 0 {
		return
	}
	lotCost := price * float64(p.cfg.Lot) * (1 + p.cfg.Commission)
	if lotCost <= 0 {
		return
	}
	budget := p.cfg.Fraction * p.cash
	lots := int64(math.Floor(budget / lotCost))
	if lots <= 0 {
		return
	}
	qty := lots * int64(p.cfg.Lot)
	cost := float64(qty) * price
	commission := cost * p.cfg.Commission
	p.cash -= cost + commission
	p.qty = qty
	p.entryPrice = price
	p.entryTime = t
	p.entryBar = p.bar
	p.entryLevel = level
	p.entryTarget = target
	p.entryATR = atr
}
```

В `close` добавить три поля в собираемый `Trade` и обнулить контекст после закрытия. Заменить блок построения `tr` и сброса:

```go
	tr := Trade{
		EntryTime:       p.entryTime,
		EntryPrice:      p.entryPrice,
		ExitTime:        t,
		ExitPrice:       price,
		Quantity:        p.qty,
		Reason:          reason,
		PnL:             pnl,
		PnLPct:          pnlPct,
		BarsHeld:        p.bar - p.entryBar,
		SupportLevel:    p.entryLevel,
		ResistanceLevel: p.entryTarget,
		ATR:             p.entryATR,
	}
	p.qty = 0
	p.entryPrice = 0
	p.entryLevel = 0
	p.entryTarget = 0
	p.entryATR = 0
	return tr
```

- [ ] **Step 5: Передать контекст из сигнала в движке**

В `internal/domain/backtest/engine.go` заменить Buy-ветку:

```go
		case model.SignalBuy:
			if p.qty == 0 {
				p.open(c.Close, c.Time)
			}
```

на:

```go
		case model.SignalBuy:
			if p.qty == 0 {
				p.open(c.Close, c.Time, sig.Level, sig.TakeProfit, sig.ATR)
			}
```

- [ ] **Step 6: Запустить весь пакет backtest — зелёные**

Run: `go test ./internal/domain/backtest/ -v`
Expected: PASS (новый тест штампа + все существующие; `SupportLevel/ResistanceLevel/ATR` у старых тестов = 0).

- [ ] **Step 7: Commit**

```bash
git add internal/domain/backtest/types.go internal/domain/backtest/portfolio.go internal/domain/backtest/engine.go internal/domain/backtest/engine_test.go
git commit -m "feat(backtest): stamp entry support/resistance/ATR onto each trade"
```

---

## Task 3: Вывести уровни и ATR в отчёт (Markdown + CSV)

**Files:**
- Modify: `internal/domain/backtest/report.go`
- Modify/Test: `internal/domain/backtest/report_test.go`

Контекст: журнал сделок (Markdown) и `RenderTradesCSV` сейчас не печатают новые поля `Trade`. Добавляем три колонки **в конец** строки журнала («Support», «Resist», «ATR», формат `%.4f`) и три колонки в CSV (`support_level,resistance_level,atr`, формат `%.6f`).

- [ ] **Step 1: Обновить тесты под новые колонки (падающие)**

В `internal/domain/backtest/report_test.go` заменить `TestRenderTradesCSVHeaderAndRow` целиком:

```go
func TestRenderTradesCSVHeaderAndRow(t *testing.T) {
	trades := []Trade{{
		EntryTime: time.Unix(0, 0), EntryPrice: 100, ExitTime: time.Unix(3600, 0),
		ExitPrice: 110, Quantity: 10, Reason: "TP", PnL: 100, PnLPct: 0.1, BarsHeld: 1,
		SupportLevel: 99, ResistanceLevel: 112, ATR: 1.25,
	}}
	out := RenderTradesCSV(trades)
	lines := strings.Split(strings.TrimSpace(out), "\n")
	if len(lines) != 2 {
		t.Fatalf("csv lines = %d, want 2 (header + 1 row)", len(lines))
	}
	if !strings.HasSuffix(lines[0], "support_level,resistance_level,atr") {
		t.Fatalf("header missing new columns: %q", lines[0])
	}
	if !strings.Contains(lines[1], "99.000000,112.000000,1.250000") {
		t.Fatalf("row missing entry-context values: %q", lines[1])
	}
}
```

И в `TestRenderMarkdownHasSections` дополнить список `want` колонкой `"Support"`:

```go
	for _, want := range []string{"RUAL", "EMAPeriod", "Сводка метрик", "Журнал сделок", "Движение капитала", "TP", "Support"} {
```

- [ ] **Step 2: Запустить — убедиться, что падают**

Run: `go test ./internal/domain/backtest/ -run 'TestRenderTradesCSVHeaderAndRow|TestRenderMarkdownHasSections' -v`
Expected: FAIL — текущий заголовок CSV не оканчивается на новые колонки, а Markdown не содержит «Support».

- [ ] **Step 3: Обновить Markdown-журнал в `RenderMarkdown`**

В `internal/domain/backtest/report.go` заменить заголовок журнала и форматирование строки. Заменить блок:

```go
	b.WriteString("\n## Журнал сделок\n\n| № | Вход | Цена входа | Выход | Цена выхода | Причина | Баров | PnL | PnL %% |\n|---|---|---|---|---|---|---|---|---|\n")
	for i, t := range trades {
		fmt.Fprintf(&b, "| %d | %s | %.4f | %s | %.4f | %s | %d | %.2f | %.2f%% |\n",
			i+1, t.EntryTime.Format(tsLayout), t.EntryPrice, t.ExitTime.Format(tsLayout),
			t.ExitPrice, t.Reason, t.BarsHeld, t.PnL, t.PnLPct*100)
	}
```

на:

```go
	b.WriteString("\n## Журнал сделок\n\n| № | Вход | Цена входа | Выход | Цена выхода | Причина | Баров | PnL | PnL %% | Support | Resist | ATR |\n|---|---|---|---|---|---|---|---|---|---|---|---|\n")
	for i, t := range trades {
		fmt.Fprintf(&b, "| %d | %s | %.4f | %s | %.4f | %s | %d | %.2f | %.2f%% | %.4f | %.4f | %.4f |\n",
			i+1, t.EntryTime.Format(tsLayout), t.EntryPrice, t.ExitTime.Format(tsLayout),
			t.ExitPrice, t.Reason, t.BarsHeld, t.PnL, t.PnLPct*100,
			t.SupportLevel, t.ResistanceLevel, t.ATR)
	}
```

- [ ] **Step 4: Обновить `RenderTradesCSV`**

В `internal/domain/backtest/report.go` заменить:

```go
	b.WriteString("idx,entry_time,entry_price,exit_time,exit_price,qty,reason,pnl,pnl_pct,bars_held\n")
	for i, t := range trades {
		fmt.Fprintf(&b, "%d,%s,%.6f,%s,%.6f,%d,%s,%.6f,%.6f,%d\n",
			i+1, t.EntryTime.UTC().Format(time.RFC3339), t.EntryPrice,
			t.ExitTime.UTC().Format(time.RFC3339), t.ExitPrice, t.Quantity,
			t.Reason, t.PnL, t.PnLPct, t.BarsHeld)
	}
```

на:

```go
	b.WriteString("idx,entry_time,entry_price,exit_time,exit_price,qty,reason,pnl,pnl_pct,bars_held,support_level,resistance_level,atr\n")
	for i, t := range trades {
		fmt.Fprintf(&b, "%d,%s,%.6f,%s,%.6f,%d,%s,%.6f,%.6f,%d,%.6f,%.6f,%.6f\n",
			i+1, t.EntryTime.UTC().Format(time.RFC3339), t.EntryPrice,
			t.ExitTime.UTC().Format(time.RFC3339), t.ExitPrice, t.Quantity,
			t.Reason, t.PnL, t.PnLPct, t.BarsHeld,
			t.SupportLevel, t.ResistanceLevel, t.ATR)
	}
```

- [ ] **Step 5: Запустить весь пакет backtest — зелёные**

Run: `go test ./internal/domain/backtest/ -v`
Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add internal/domain/backtest/report.go internal/domain/backtest/report_test.go
git commit -m "feat(backtest): render entry support/resist/ATR in trade journal and CSV"
```

---

## Task 4: Переориентировать калибровочный грид и обновить docs

**Files:**
- Modify: `data/params/rusal/levels_grid.json`
- Modify: `docs/levels/strategy.md`

Контекст: грид не свипает `SLMult` — главный зноб после перехода на внутрибаровое исполнение. Переписываем грид на знобы ширины стопа/трейла/качества. Грид — data-файл (парсится `RunGrid` через reflection по именам полей `core.Params`), поэтому имена ключей должны точно совпадать с полями: `SLMult`, `TrailArmATR`, `TrailMult`, `MinRR`, `RoomATR`.

- [ ] **Step 1: Переписать грид**

Заменить содержимое `data/params/rusal/levels_grid.json` на:

```json
{
  "SLMult": [1.0, 1.5, 2.0, 2.5],
  "TrailArmATR": [0.5, 1.0, 1.5],
  "TrailMult": [2.0, 2.5, 3.0],
  "MinRR": [1.2, 1.5, 2.0],
  "RoomATR": [1.5, 2.0, 2.5]
}
```

- [ ] **Step 2: Проверить, что JSON валиден и ключи — реальные поля `core.Params`**

Run: `go vet ./... && python3 -m json.tool data/params/rusal/levels_grid.json`
Expected: JSON печатается без ошибок (валиден). Ключи `SLMult/TrailArmATR/TrailMult/MinRR/RoomATR` присутствуют в `core.Params` (см. `core.go`), поэтому reflection их найдёт.

- [ ] **Step 3: Обновить `docs/levels/strategy.md`**

В разделе 6, сразу после строки:

```
Все эти числа — **стартовые** и подлежат калибровке на истории RUAL.
```

вставить:

```
**Калибровка под RUAL** (grid-search по ширине стопа/трейла, walk-forward против
переобучения):

​```
go run ./cmd/backtest -ticker RUAL -strategy levels -interval Hour1 -months 25 \
  -calibrate data/params/rusal/levels_grid.json -metric expectancy -min-trades 20 -test-months 3
​```

Победитель — в `reports/..._best.md`; его значения переносятся в `rusal.go`
`DefaultParams`.

**Журнал сделок** (Markdown и `_trades.csv`) теперь для каждой сделки показывает
уровень входа **Support** (HVN-поддержка), цель **Resist** (HVN-сопротивление) и
**ATR** на момент входа — виден контекст риск/прибыли и волатильность.
```

(Примечание для исполнителя: символы `​` вокруг внутренних ``` — это zero-width
заглушки, чтобы внешний блок плана не закрылся раньше времени; в реальный файл
вставь обычные тройные backtick без них.)

- [ ] **Step 4: Commit**

```bash
git add data/params/rusal/levels_grid.json docs/levels/strategy.md
git commit -m "feat(levels): retarget RUAL calibration grid at stop/trail width; document"
```

---

## Калибровка — ручной шаг владельца (вне кода)

После реализации владелец сам запускает:

```
go run ./cmd/backtest -ticker RUAL -strategy levels -interval Hour1 -months 25 \
  -calibrate data/params/rusal/levels_grid.json -metric expectancy -min-trades 20 -test-months 3
```

и присылает победителя из `*_best.md`; его значения зашиваются в
`internal/service/trading_strategy/levels/strategy/rusal/rusal.go` `DefaultParams()`.
Это отдельный шаг **после** прогона — не входит в задачи плана.

---

## Вне рамок (YAGNI)

- ATR-сводка в шапке отчёта (per-trade достаточно).
- Авто-запуск калибровки из кода.
- Второй проход грида (HVNFactor/MaxExtensionATR/LevelTolATR/ATRPeriod).
- Условное скрытие нулевых колонок для scalping-отчётов.
