# rsi_pullback: единая калибровка с выбором режима входа — план реализации

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Тип входа (`pullback`/`zone`) в сделке и отчётах бэктеста плюс единая процедура `/pullback-calibrate`, которая калибрует оба входа, выбирает режим `UseZoneEntry` 0/1/2 или отказ и всегда калибрует общие выходы.

**Architecture:** Поле `EntryKind` идёт по цепочке `model.Signal` → `portfolio.open` → `backtest.Trade`; общая функция `backtest.KindBreakdown` считает метрики по типам и используется одиночным отчётом и walk-forward. Процедура текстовая: §8.2 `strategy.md` (механика) и команда `.claude/commands/pullback-calibrate.md`; её инварианты держат два сторожевых теста в `internal/service/backtest/rsi_pullback_zone_grid_test.go` (режим файла и минимальная ширина осей).

**Tech Stack:** Go 1.25, стандартный `testing`, `./bin/mage ci`.

**Spec:** `docs/superpowers/specs/2026-09-30-rsi-pullback-unified-calibration-design.md`

## Global Constraints

- Ветка `feat/pullback-zone-entry`; не мержить и не пушить.
- Коммиты на русском, в конце строка `Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>`.
- `docs/rsi_pullback/*.md` — только механика: без результатов прогонов, PF, дат заведения тикеров.
- Отчёты стратегий без типа входа (`EntryKind == ""`) не меняются, кроме колонки «Вход» со значением `—`.
- CSV: колонка `entry_kind` — последней.
- Значения типа входа — ровно `"pullback"` и `"zone"`.
- `UseRSIExit` в сетках процедуры не свипается.
- Перекалибровка тикеров в объём не входит.
- После каждой задачи: `go build ./internal/... ./pkg/... ./cmd/...` и тесты затронутых пакетов зелёные; в последней задаче — `./bin/mage ci`.

## Review Focus

1. Режим 1, совпадение сигналов: сделка помечена `pullback` (приоритет), а не `zone` — тест в Task 1.
2. Смешанный журнал, где у части сделок тип пуст: пустой тип не образует строку в разделе «По типу входа» и не роняет порядок — тест в Task 2.
3. Позиционные скрипты журнала: все старые колонки CSV на прежних местах, `entry_kind` добавлена в конец — тест в Task 2.
4. Файл единой процедуры со свипом режима `{0,2}` в одной фазе или разными режимами по фазам — сторож ловит — тест в Task 4.
5. Каталоги старых калибровок (без `plateau_mode_both.json`) не падают на сторож ширины, а `plateau_point.json` в них не обязан задавать режим — тест в Task 4.

---

### Task 1: `EntryKind` в сигнале и ядре rsi_pullback

**Files:**
- Modify: `internal/service/trading_strategy/scalping/model/signal.go`
- Modify: `internal/service/trading_strategy/rsi_pullback/strategy/core/core.go`
- Test: `internal/service/trading_strategy/rsi_pullback/strategy/core/zone_test.go`

**Interfaces:**
- Produces: `model.Signal.EntryKind string`; `core.EntryKindPullback = "pullback"`, `core.EntryKindZone = "zone"`.

- [ ] **Step 1: Write the failing test** — дописать в конец `zone_test.go`:

```go
// TestEntryKindMarksWhichEntryBought: тип входа сделки — то, что отчёт показывает в колонке «Вход».
// При совпадении сигналов в режиме 1 входит pullback, в режиме 2 — только zone.
func TestEntryKindMarksWhichEntryBought(t *testing.T) {
	md := withDay(entryFixture(), 10.0, 101, 100)

	pullbackOff := zoneParams()
	pullbackOff.UseZoneEntry = 0

	pullbackBlocked := zoneParams()
	blockPullbackTrend(&pullbackBlocked)

	zoneOnly := zoneParams()
	zoneOnly.UseZoneEntry = ZoneEntryOnly

	cases := []struct {
		name string
		p    Params
		want string
	}{
		{"режим 0 — pullback", pullbackOff, EntryKindPullback},
		{"режим 1, срабатывают оба — pullback", zoneParams(), EntryKindPullback},
		{"режим 1, pullback закрыт — zone", pullbackBlocked, EntryKindZone},
		{"режим 2, срабатывают оба — zone", zoneOnly, EntryKindZone},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := NewWithParams("T", c.p).Decide(md)
			if got.Kind != model.SignalBuy {
				t.Fatalf("Kind = %v, want Buy", got.Kind)
			}
			if got.EntryKind != c.want {
				t.Fatalf("EntryKind = %q, want %q (reason %q)", got.EntryKind, c.want, got.EntryReason)
			}
		})
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/service/trading_strategy/rsi_pullback/strategy/core/ -run TestEntryKindMarksWhichEntryBought -count=1`
Expected: FAIL компиляции — `got.EntryKind undefined`, `undefined: EntryKindPullback`.

- [ ] **Step 3: Write minimal implementation**

В `signal.go` после строки `EntryReason    string ...` добавить поле (выровнять gofmt):

```go
	EntryKind      string  // which entry opened the position, for strategies with more than one (rsi_pullback: "pullback" / "zone"); empty otherwise
```

В `core.go` сразу после блока констант `ZoneEntryAlso`/`ZoneEntryOnly` добавить:

```go
// Entry kinds stamped on a Buy signal (model.Signal.EntryKind): the backtest report groups
// trades by them, which is how the unified calibration tells the two entries apart.
const (
	EntryKindPullback = "pullback"
	EntryKindZone     = "zone"
)
```

В pullback-ветке `enter()` после `sig.EntryReason = s.entryReason(rsi[i], fast[i], slow[i], entry, stop, target, atr, md)` добавить:

```go
	sig.EntryKind = EntryKindPullback
```

В `zoneEntry()` сразу после присваивания `sig.EntryReason = fmt.Sprintf(... )` (перед `return sig`) добавить:

```go
	sig.EntryKind = EntryKindZone
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/service/trading_strategy/rsi_pullback/... ./internal/service/trading_strategy/scalping/... -count=1`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/service/trading_strategy/scalping/model/signal.go internal/service/trading_strategy/rsi_pullback/strategy/core/core.go internal/service/trading_strategy/rsi_pullback/strategy/core/zone_test.go
git commit -m "feat(rsi_pullback): тип входа pullback/zone в сигнале покупки

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

---

### Task 2: тип входа в сделке и отчёте одиночного прогона

**Files:**
- Modify: `internal/domain/backtest/types.go` (поле `Trade.EntryKind`)
- Modify: `internal/domain/backtest/portfolio.go` (`open`, поле `entryKind`, сброс, перенос в `Trade`)
- Modify: `internal/domain/backtest/engine.go` (оба вызова `p.open`)
- Modify: `internal/domain/backtest/portfolio_test.go` и прочие вызовы `p.open(...)` в тестах пакета (все 16 мест: `grep -rn "\.open(" internal/domain/backtest`) — дописать аргумент `""`
- Modify: `internal/domain/backtest/report.go` (`KindStats`, `KindBreakdown`, колонка, раздел, CSV)
- Test: `internal/domain/backtest/engine_test.go`, создать `internal/domain/backtest/report_entrykind_test.go`, поправить `report_test.go`

**Interfaces:**
- Consumes: `model.Signal.EntryKind` (Task 1).
- Produces:
  ```go
  // KindStats — метрики сделок одного типа входа.
  type KindStats struct {
      Kind    string
      Metrics Metrics
      SLExits int
  }
  func KindBreakdown(trades []Trade) []KindStats
  ```
  Порядок: `pullback`, `zone`, затем прочие непустые по алфавиту; сделки с пустым типом не входят; нет ни одного типа — `nil`.

- [ ] **Step 1: Write the failing tests**

В `engine_test.go` после `TestEngineStampsEntryContextOnTrade`:

```go
func TestEngineStampsEntryKindOnTrade(t *testing.T) {
	candles := flatCandles([]float64{10, 100, 110})
	s := scriptedStrategy{lookback: 1, decide: func(md strategy.MarketData) model.Signal {
		if md.Position == nil && md.Price == 100 {
			return model.Signal{Kind: model.SignalBuy, EntryKind: "zone"}
		}
		if md.Position != nil && md.Price == 110 {
			return model.Signal{Kind: model.SignalSell, Reason: "RSI"}
		}
		return model.Signal{Kind: model.SignalNone}
	}}
	res := Run(s, candles, nil, nil, Config{InitialCash: 100000, Fraction: 1.0, Lot: 1})
	if len(res.Trades) != 1 {
		t.Fatalf("trades = %d, want 1", len(res.Trades))
	}
	if got := res.Trades[0].EntryKind; got != "zone" {
		t.Fatalf("EntryKind = %q, want zone", got)
	}
}
```

Создать `report_entrykind_test.go`:

```go
package backtest

import (
	"strings"
	"testing"
	"time"
)

func kindTrade(kind, reason string, pnl float64) Trade {
	return Trade{
		EntryTime: time.Unix(0, 0), EntryPrice: 100, ExitTime: time.Unix(3600, 0),
		ExitPrice: 100 + pnl, Quantity: 1, Reason: reason, PnL: pnl, BarsHeld: 1, EntryKind: kind,
	}
}

func TestKindBreakdownOrdersAndSkipsEmpty(t *testing.T) {
	trades := []Trade{
		kindTrade("zone", "SL", -50),
		kindTrade("", "TP", 10),
		kindTrade("pullback", "TP", 100),
		kindTrade("alpha", "TP", 5),
		kindTrade("zone", "TP", 150),
		kindTrade("pullback", "SL", -40),
	}
	got := KindBreakdown(trades)
	var kinds []string
	for _, k := range got {
		kinds = append(kinds, k.Kind)
	}
	if strings.Join(kinds, ",") != "pullback,zone,alpha" {
		t.Fatalf("kinds = %v, want pullback,zone,alpha", kinds)
	}
	zone := got[1]
	want := Compute(Result{Trades: []Trade{trades[0], trades[4]}}, 0, 0, 0)
	if zone.Metrics.TotalTrades != 2 || zone.Metrics.ProfitFactor != want.ProfitFactor || zone.Metrics.NetPnL != want.NetPnL {
		t.Fatalf("zone metrics = %+v, want Compute над подмножеством %+v", zone.Metrics, want)
	}
	if zone.SLExits != 1 || got[0].SLExits != 1 || got[2].SLExits != 0 {
		t.Fatalf("SLExits = %d/%d/%d, want 1/1/0", got[0].SLExits, zone.SLExits, got[2].SLExits)
	}
}

func TestKindBreakdownNilWithoutKinds(t *testing.T) {
	if got := KindBreakdown([]Trade{kindTrade("", "TP", 10)}); got != nil {
		t.Fatalf("KindBreakdown = %v, want nil", got)
	}
}

func TestRenderMarkdownEntryKindColumnAndSection(t *testing.T) {
	trades := []Trade{kindTrade("pullback", "TP", 100), kindTrade("zone", "SL", -50), kindTrade("", "TP", 10)}
	out := RenderMarkdown(sampleMeta(), Metrics{}, trades, nil)
	if !strings.Contains(out, "| № | Вход | Время входа | Цена входа |") {
		t.Fatalf("журнал без колонки «Вход» второй: %q", out)
	}
	if !strings.Contains(out, "| 1 | pullback |") || !strings.Contains(out, "| 2 | zone |") || !strings.Contains(out, "| 3 | — |") {
		t.Fatalf("колонка «Вход» не заполнена или пустой тип не «—»: %q", out)
	}
	sec := strings.Index(out, "## По типу входа")
	journal := strings.Index(out, "## Журнал сделок")
	if sec < 0 || sec > journal {
		t.Fatalf("раздел «По типу входа» отсутствует или стоит после журнала")
	}
	if !strings.Contains(out, "| pullback | 1 |") || !strings.Contains(out, "| zone | 1 |") {
		t.Fatalf("строки раздела по типам не найдены: %q", out[sec:journal])
	}
}

func TestRenderMarkdownNoKindSectionWithoutKinds(t *testing.T) {
	out := RenderMarkdown(sampleMeta(), Metrics{}, []Trade{kindTrade("", "TP", 10)}, nil)
	if strings.Contains(out, "## По типу входа") {
		t.Fatal("раздел по типам выведен для стратегии без типа входа")
	}
	if !strings.Contains(out, "| 1 | — |") {
		t.Fatalf("пустой тип должен печататься «—»: %q", out)
	}
}

func TestRenderTradesCSVEntryKindIsLastColumn(t *testing.T) {
	out := RenderTradesCSV([]Trade{kindTrade("zone", "SL", -50)})
	lines := strings.Split(strings.TrimSpace(out), "\n")
	if lines[0] != "idx,entry_time,entry_price,exit_time,exit_price,qty,reason,pnl,pnl_pct,bars_held,support_level,resistance_level,atr,entry_reason,exit_reason,entry_kind" {
		t.Fatalf("header = %q: старые колонки обязаны остаться на местах, entry_kind — последней", lines[0])
	}
	if !strings.HasSuffix(lines[1], ",zone") {
		t.Fatalf("row = %q, want suffix ,zone", lines[1])
	}
}
```

Существующие заголовки журнала «Вход»/«Выход» (время) переименовываются в «Время входа»/«Время выхода», чтобы новая колонка «Вход» (тип) с ними не путалась.

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/domain/backtest/ -count=1`
Expected: FAIL компиляции — `unknown field EntryKind in struct literal of type Trade`, `undefined: KindBreakdown`.

- [ ] **Step 3: Implement**

`types.go`, в `Trade` после `ExitReason`:

```go
	EntryKind       string  // which entry opened the trade (rsi_pullback: "pullback" / "zone"); empty for single-entry strategies
```

`portfolio.go`: поле `entryKind string` рядом с `entryReason`; сигнатура

```go
func (p *portfolio) open(price float64, t time.Time, level, target, atr, stop float64, entryReason, entryKind string) {
```

в теле после `p.entryReason = entryReason` — `p.entryKind = entryKind`; при построении `Trade` рядом с `EntryReason: p.entryReason,` — `EntryKind: p.entryKind,`; при сбросе рядом с `p.entryReason = ""` — `p.entryKind = ""`.

`engine.go`: оба вызова `p.open(c.Close, c.Time, sig.Level, sig.TakeProfit, sig.ATR, sig.StopLoss, sig.EntryReason)` → добавить последним аргументом `sig.EntryKind`.

Тесты пакета: во всех вызовах `p.open(...)` дописать последний аргумент `""`.

`report.go` — добавить перед `RenderMarkdown`:

```go
// KindStats holds the metrics of the trades opened by one entry kind.
type KindStats struct {
	Kind    string
	Metrics Metrics
	SLExits int
}

// KindBreakdown groups trades by EntryKind: "pullback" first, "zone" second, any other kind
// after them in alphabetical order. Trades without a kind are left out; nil when no trade has
// one, which is how single-entry strategies keep their reports unchanged.
func KindBreakdown(trades []Trade) []KindStats {
	groups := make(map[string][]Trade)
	for _, t := range trades {
		if t.EntryKind != "" {
			groups[t.EntryKind] = append(groups[t.EntryKind], t)
		}
	}
	if len(groups) == 0 {
		return nil
	}
	rank := func(k string) int {
		switch k {
		case "pullback":
			return 0
		case "zone":
			return 1
		}
		return 2
	}
	kinds := make([]string, 0, len(groups))
	for k := range groups {
		kinds = append(kinds, k)
	}
	sort.Slice(kinds, func(i, j int) bool {
		if rank(kinds[i]) != rank(kinds[j]) {
			return rank(kinds[i]) < rank(kinds[j])
		}
		return kinds[i] < kinds[j]
	})
	out := make([]KindStats, 0, len(kinds))
	for _, k := range kinds {
		ks := KindStats{Kind: k, Metrics: Compute(Result{Trades: groups[k]}, 0, 0, 0)}
		for _, t := range groups[k] {
			if t.Reason == "SL" {
				ks.SLExits++
			}
		}
		out = append(out, ks)
	}
	return out
}

// renderKindTable writes the per-entry-kind table; the caller skips it when kinds is empty.
func renderKindTable(b *strings.Builder, kinds []KindStats) {
	b.WriteString("| Вход | Сделок | Win rate | Profit factor | Net PnL | SL-выходов |\n|---|---|---|---|---|---|\n")
	for _, k := range kinds {
		fmt.Fprintf(b, "| %s | %d | %.2f%% | %.3f | %.2f | %d |\n",
			k.Kind, k.Metrics.TotalTrades, k.Metrics.WinRate*100, k.Metrics.ProfitFactor, k.Metrics.NetPnL, k.SLExits)
	}
}
```

(импорт `sort`). В `RenderMarkdown` перед `b.WriteString("\n## Журнал сделок...` вставить:

```go
	if kinds := KindBreakdown(trades); len(kinds) > 0 {
		b.WriteString("\n## По типу входа\n\n")
		renderKindTable(&b, kinds)
	}
```

Заголовок журнала и строка:

```go
	b.WriteString("\n## Журнал сделок\n\n| № | Вход | Время входа | Цена входа | Время выхода | Цена выхода | Причина | Баров | PnL | PnL %% | Support | Resist | ATR | Причина входа | Причина выхода |\n|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|\n")
	for i, t := range trades {
		kind := t.EntryKind
		if kind == "" {
			kind = "—"
		}
		fmt.Fprintf(&b, "| %d | %s | %s | %.4f | %s | %.4f | %s | %d | %.2f | %.2f%% | %.4f | %.4f | %.4f | %s | %s |\n",
			i+1, kind, t.EntryTime.Format(tsLayout), t.EntryPrice, t.ExitTime.Format(tsLayout),
			t.ExitPrice, t.Reason, t.BarsHeld, t.PnL, t.PnLPct*100,
			t.SupportLevel, t.ResistanceLevel, t.ATR, t.EntryReason, t.ExitReason)
	}
```

`RenderTradesCSV`: заголовок дополнить `,entry_kind`, формат строки — `,%s` в конце и аргумент `csvField(t.EntryKind)`.

В `report_test.go` в `TestRenderTradesCSVHeaderAndRow` суффикс заголовка сменить на `"support_level,resistance_level,atr,entry_reason,exit_reason,entry_kind"`.

- [ ] **Step 4: Run tests**

Run: `go test ./internal/domain/backtest/ ./internal/service/backtest/ ./cmd/... -count=1 && go build ./internal/... ./pkg/... ./cmd/...`
Expected: PASS. Если упал чужой тест, сравнивавший журнал markdown по заголовку «Вход»/«Выход», — поправить ожидание на «Время входа»/«Время выхода».

- [ ] **Step 5: Commit**

```bash
git add internal/domain/backtest/
git commit -m "feat(backtest): тип входа в сделке, колонка «Вход» и раздел «По типу входа» в отчёте

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

---

### Task 3: пул walk-forward по типу входа

**Files:**
- Modify: `internal/service/backtest/walkforward.go`
- Test: `internal/service/backtest/walkforward_test.go`

**Interfaces:**
- Consumes: `backtest.KindBreakdown`, `backtest.KindStats` (Task 2).
- Produces: `WalkForwardSummary.PooledByKind []backtest.KindStats` (срез, а не map из спеки: порядок строк отчёта задаёт `KindBreakdown`).

- [ ] **Step 1: Write the failing tests** — в `walkforward_test.go`:

```go
func TestRenderWalkForwardMarkdownKindSection(t *testing.T) {
	s := WalkForwardSummary{
		PooledOOS: backtest.Metrics{ProfitFactor: 1.2, TotalTrades: 3},
		PooledByKind: []backtest.KindStats{
			{Kind: "pullback", Metrics: backtest.Metrics{TotalTrades: 2, ProfitFactor: 1.5}},
			{Kind: "zone", Metrics: backtest.Metrics{TotalTrades: 1, ProfitFactor: 0.7}, SLExits: 1},
		},
	}
	md := RenderWalkForwardMarkdown("T", "profit_factor", s, 12, 6)
	sec := strings.Index(md, "## Пул по типу входа")
	folds := strings.Index(md, "## Результаты по фолдам")
	if sec < 0 || sec > folds {
		t.Fatalf("раздел «Пул по типу входа» отсутствует или стоит после фолдов:\n%s", md)
	}
	if !strings.Contains(md, "| zone | 1 |") {
		t.Fatalf("строка zone не найдена:\n%s", md)
	}
	if strings.Contains(RenderWalkForwardMarkdown("T", "profit_factor", WalkForwardSummary{}, 12, 6), "## Пул по типу входа") {
		t.Fatal("раздел выведен без типов входа")
	}
}
```

И в `TestRunWalkForward` после проверки `pooled trades` добавить:

```go
	if s.PooledByKind != nil {
		t.Fatalf("PooledByKind = %v, want nil: тестовая стратегия не ставит тип входа", s.PooledByKind)
	}
```

Плюс тест на реальное заполнение пула:

```go
// kindedStrategy торгует как alternatingStrategy, но помечает каждую покупку типом входа zone.
type kindedStrategy struct{}

func (kindedStrategy) Ticker() string { return "TEST" }
func (kindedStrategy) Lookback() int  { return 1 }
func (kindedStrategy) Decide(md strategy.MarketData) model.Signal {
	if md.Position == nil {
		return model.Signal{Kind: model.SignalBuy, EntryKind: "zone"}
	}
	return model.Signal{Kind: model.SignalSell, Reason: "TP"}
}

func TestRunWalkForwardPoolsByKind(t *testing.T) {
	from, to := date(2025, time.January, 1), date(2025, time.October, 1)
	b := fakeBinding()
	b.Build = func(any) strategy.Strategy { return kindedStrategy{} }
	cfg := backtest.Config{InitialCash: 100000, Fraction: 1, Commission: 0.0005, Lot: 1}
	s, err := RunWalkForward(b, []Phase{{Grid: Grid{"Threshold": {1, 2}}}}, genHourly(from, to), nil, nil, cfg,
		"profit_factor", 0, from, to, 3, 3)
	if err != nil {
		t.Fatal(err)
	}
	if len(s.PooledByKind) != 1 || s.PooledByKind[0].Kind != "zone" ||
		s.PooledByKind[0].Metrics.TotalTrades != s.PooledOOS.TotalTrades {
		t.Fatalf("PooledByKind = %+v, want один тип zone со всеми %d сделками пула", s.PooledByKind, s.PooledOOS.TotalTrades)
	}
}
```

- [ ] **Step 2: Run to verify fail**

Run: `go test ./internal/service/backtest/ -run 'WalkForward' -count=1`
Expected: FAIL компиляции — `unknown field PooledByKind`.

- [ ] **Step 3: Implement** — в `walkforward.go`:

в `WalkForwardSummary` после `PooledOOS`:

```go
	PooledByKind        []backtest.KindStats // OOS pool split by entry kind; nil for single-entry strategies
```

после `summary.PooledOOS = PooledMetrics(pool)`:

```go
	summary.PooledByKind = backtest.KindBreakdown(pool)
```

в `RenderWalkForwardMarkdown` перед `b.WriteString("## Результаты по фолдам\n\n")`:

```go
	if len(s.PooledByKind) > 0 {
		b.WriteString("## Пул по типу входа\n\n")
		b.WriteString("| Вход | Сделок | Win rate | Profit factor | Net PnL | SL-выходов |\n|---|---|---|---|---|---|\n")
		for _, k := range s.PooledByKind {
			fmt.Fprintf(&b, "| %s | %d | %.2f%% | %.3f | %.2f | %d |\n",
				k.Kind, k.Metrics.TotalTrades, k.Metrics.WinRate*100, k.Metrics.ProfitFactor, k.Metrics.NetPnL, k.SLExits)
		}
		b.WriteString("\n")
	}
```

- [ ] **Step 4: Run tests**

Run: `go test ./internal/service/backtest/ -count=1`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/service/backtest/walkforward.go internal/service/backtest/walkforward_test.go
git commit -m "feat(backtest): пул OOS walk-forward по типу входа

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

---

### Task 4: сторожа единой процедуры — режим файла и ширина осей

**Files:**
- Modify: `internal/service/backtest/rsi_pullback_zone_grid_test.go`

**Interfaces:**
- Produces:
  ```go
  func rsiPullbackZoneFileViolations(name string, unified bool, phases []Phase) []string
  var rsiPullbackUnifiedMinAxes map[string]map[string][]float64 // имя файла → поле → обязательные узлы
  func rsiPullbackUnifiedAxisViolations(name string, phases []Phase) []string
  func TestRSIPullbackUnifiedGridsStayWide(t *testing.T)
  ```
  `unified` — в каталоге файла лежит `plateau_mode_both.json`.

Правила режима (спека §5), проверяются по каждой фазе:

1. Если хоть одна фаза задаёт `UseZoneEntry`, его задаёт каждая фаза, ровно одним значением, одинаковым во всех фазах файла.
2. Значение ∈ {0, 1, 2}; при 1 и 2 — три zone-поля заданы и строго > 0; при 0 zone-поля не требуются.
3. zone-имена (`cal_zone_`, `cal2_zone_`, `plateau_zone_`, `plateau_entry_zone`, `r2_point_zone`, `r2_probe_zone`) — режим обязателен и ∈ {1, 2}.
4. `plateau_base_<m>`, `plateau_mode_<m>`, `plateau_final_<m>` (`<m>` ∈ pullback, zone, both) — режим обязателен и равен 0 / 2 / 1 соответственно; суффикс вне этих трёх — нарушение.
5. `plateau_entry_pullback` — режим не задан или 0.
6. `cal_out_`, `cal2_out_` — режим обязателен; при `unified` то же для `plateau_point.json` и `r2_point.json`.

- [ ] **Step 1: Write the failing tests**

В `TestRSIPullbackZoneFileViolations` сменить вызов на `rsiPullbackZoneFileViolations(c.file, c.unified, c.phases)`, добавить в структуру поле `unified bool` (старые случаи — `false`), случай «UseZoneEntry свипает выкл» `{0, 2}` оставить нарушением (правило 1). Дописать случаи:

```go
		{"явный режим 0 допустим", "cal_out_risk.json", false,
			[]Phase{{Name: "risk", Grid: Grid{"UseZoneEntry": {0}, "StopDailyATR": {0.5, 0.7}}}}, false},
		{"cal_out без режима", "cal_out_risk.json", false,
			[]Phase{{Name: "risk", Grid: Grid{"StopDailyATR": {0.5}}}}, true},
		{"разные режимы по фазам", "cal_out_exit.json", false,
			[]Phase{zoneArmedPhase("a", nil), zoneArmedPhase("b", Grid{"UseZoneEntry": {core.ZoneEntryAlso}})}, true},
		{"режим 3", "cal_out_exit.json", false,
			[]Phase{zoneArmedPhase("exit", Grid{"UseZoneEntry": {3}})}, true},
		{"mode_both = 1", "plateau_mode_both.json", false,
			[]Phase{zoneArmedPhase("point", Grid{"UseZoneEntry": {core.ZoneEntryAlso}})}, false},
		{"mode_both с режимом 2", "plateau_mode_both.json", false,
			[]Phase{zoneArmedPhase("point", nil)}, true},
		{"final_pullback = 0", "plateau_final_pullback.json", false,
			[]Phase{{Name: "point", Grid: Grid{"UseZoneEntry": {0}}}}, false},
		{"base_zone с режимом 0", "plateau_base_zone.json", false,
			[]Phase{{Name: "point", Grid: Grid{"UseZoneEntry": {0}}}}, true},
		{"неизвестный суффикс режима", "plateau_mode_mixed.json", false,
			[]Phase{{Name: "point", Grid: Grid{"UseZoneEntry": {0}}}}, true},
		{"entry_pullback с режимом 2", "plateau_entry_pullback.json", false,
			[]Phase{zoneArmedPhase("point", nil)}, true},
		{"entry_zone с режимом 0", "plateau_entry_zone.json", false,
			[]Phase{{Name: "point", Grid: Grid{"UseZoneEntry": {0}}}}, true},
		{"точка старой калибровки без режима", "plateau_point.json", false,
			[]Phase{{Name: "point", Grid: Grid{"RSIUpper": {70}}}}, false},
		{"точка единой процедуры без режима", "plateau_point.json", true,
			[]Phase{{Name: "point", Grid: Grid{"RSIUpper": {70}}}}, true},
```

Тест ширины:

```go
func TestRSIPullbackUnifiedAxisViolations(t *testing.T) {
	wide := Grid{"StopDailyATR": {0.3, 0.4, 0.5, 0.6, 0.7, 0.8, 1.0, 1.2, 1.5, 2.0},
		"TPDailyATR": {0.1, 0.15, 0.2, 0.3, 0.4, 0.5, 0.6, 0.8, 1.0, 1.5, 2.5}, "UseZoneEntry": {0}}
	if got := rsiPullbackUnifiedAxisViolations("cal_out_risk.json", []Phase{{Name: "risk", Grid: wide}}); len(got) != 0 {
		t.Fatalf("широкая сетка: ложное нарушение %v", got)
	}
	narrow := Grid{"StopDailyATR": {0.5, 0.7}, "TPDailyATR": {0.6}, "UseZoneEntry": {0}}
	if got := rsiPullbackUnifiedAxisViolations("cal_out_risk.json", []Phase{{Name: "risk", Grid: narrow}}); len(got) == 0 {
		t.Fatal("узкая сетка не поймана")
	}
	exitMode0 := Grid{"RSIUpper": {30, 35, 40, 45, 50, 55, 60, 65, 70, 75, 80, 85, 90, 95}, "UseZoneEntry": {0}}
	if got := rsiPullbackUnifiedAxisViolations("cal_out_exit.json", []Phase{{Name: "exit", Grid: exitMode0}}); len(got) != 0 {
		t.Fatalf("exit режима 0 не обязан свипать RSIPeriod: %v", got)
	}
	exitMode2 := zoneArmedPhase("exit", Grid{"RSIUpper": exitMode0["RSIUpper"]})
	if got := rsiPullbackUnifiedAxisViolations("cal_out_exit.json", []Phase{exitMode2}); len(got) == 0 {
		t.Fatal("exit режима 2 без оси RSIPeriod не пойман")
	}
	if got := rsiPullbackUnifiedAxisViolations("plateau_point.json", []Phase{{Name: "p", Grid: Grid{"RSIUpper": {70}}}}); len(got) != 0 {
		t.Fatalf("файл вне таблицы тем не проверяется: %v", got)
	}
}
```

- [ ] **Step 2: Run to verify fail**

Run: `go test ./internal/service/backtest/ -run 'Zone|Unified' -count=1`
Expected: FAIL компиляции (лишний аргумент, `undefined: rsiPullbackUnifiedAxisViolations`).

- [ ] **Step 3: Implement**

Переписать `rsiPullbackZoneFileViolations` по правилам 1–6 (doc-comment обновить: теперь это сторож режима файла для обеих процедур). Имя-режим:

```go
// rsiPullbackModeBySuffix — режим, который обязаны задавать plateau_{base,mode,final}_<m>.
var rsiPullbackModeBySuffix = map[string]float64{"pullback": 0, "zone": core.ZoneEntryOnly, "both": core.ZoneEntryAlso}
```

Разбор имени: для префиксов `plateau_base_`, `plateau_mode_`, `plateau_final_` суффикс = имя без префикса и без `.json`; суффикса нет в карте — нарушение `"%s: суффикс режима %q не из pullback/zone/both"`.

Таблица ширины (узлы — дословно из спеки §4.6):

```go
// rsiPullbackUnifiedMinAxes — минимальные оси тем единой процедуры (спека 2026-09-30, §4.6):
// шире можно, уже нельзя. Ключ — имя файла темы.
var rsiPullbackUnifiedMinAxes = map[string]map[string][]float64{
	"cal_screen.json":     {"UseDayATRGate": {0, 1}, "UseVolume": {0, 1}},
	"cal_entry.json":      {"RSIPeriod": {2, 3, 4, 5, 6, 7, 8, 10, 12, 14}, "RSILower": {5, 10, 15, 20, 25, 30, 35, 40, 45, 50}},
	"cal_trend.json":      {"EMAFast": {3, 5, 8, 10, 15, 20, 30, 40}, "EMASlow": {50, 75, 100, 150, 200, 250}},
	"cal_trend_low.json":  {"EMAFast": {1, 2, 3, 5, 8, 10}, "EMASlow": {12, 15, 20, 25, 30, 35, 40, 45}},
	"cal_day.json":        {"FreshDayATR": {0, 0.05, 0.1, 0.15, 0.2, 0.3, 0.4, 0.5}, "SpentDayATR": {0.4, 0.5, 0.6, 0.7, 0.8, 0.9, 1.0, 1.1, 1.25, 1.5, 2.0, 2.5}},
	"cal_volume.json":     {"UseVolume": {1}, "VolMult": {1.0, 1.2, 1.5, 2.0, 2.5, 3.0, 3.5, 4.0}, "VolBaseDays": {3, 5, 10, 14, 20, 30}},
	"cal_vol_window.json": {"UseVolume": {1}, "VolLookbackBars": {1, 2, 3, 5, 8, 12, 16, 24, 32}, "VolMult": {1.0, 1.2, 2.0, 3.0}},
	"cal_trend_spent.json": {"EMAFast": {1, 2, 3, 5, 10}, "SpentDayATR": {0.7, 0.8, 0.9, 1.0, 1.1, 1.25, 1.5}},
	"cal_trend_day.json":  {"EMASlow": {15, 20, 25, 30, 40, 50, 75, 100}, "FreshDayATR": {0, 0.05, 0.1, 0.15, 0.2}},
	"cal_zone_zone_entry.json": {"ZoneRSIPeriod": {2, 3, 4, 5, 6, 8, 10, 14}, "ZoneRSILower": {5, 10, 15, 20, 25, 30, 35, 40, 45}},
	"cal_zone_zone_trend.json": {"ZoneEMAPeriod": {20, 30, 50, 75, 100, 150, 200, 250, 300, 400}},
	"cal_out_exit.json":   {"RSIUpper": {30, 35, 40, 45, 50, 55, 60, 65, 70, 75, 80, 85, 90, 95}},
	"cal_out_risk.json":   {"StopDailyATR": {0.3, 0.4, 0.5, 0.6, 0.7, 0.8, 1.0, 1.2, 1.5, 2.0}, "TPDailyATR": {0.1, 0.15, 0.2, 0.3, 0.4, 0.5, 0.6, 0.8, 1.0, 1.5, 2.5}},
	"cal_out_trail.json":  {"UseTrail": {0, 1}, "TrailDailyATR": {0.2, 0.3, 0.4, 0.5, 0.7, 1.0, 1.5}},
}

// rsiPullbackExitZoneOnlyRSIPeriod — в режиме 2 RSIPeriod только выходной и свипается темой exit.
var rsiPullbackExitZoneOnlyRSIPeriod = []float64{2, 3, 4, 5, 6, 8, 10, 14}
```

`rsiPullbackUnifiedAxisViolations(name, phases)`: файл вне таблицы — `nil`; иначе объединение значений каждого поля по всем фазам; каждый обязательный узел должен присутствовать (сравнение с допуском `1e-9`); для `cal_out_exit.json`, если режим файла 2, дополнительно `RSIPeriod` ⊇ `rsiPullbackExitZoneOnlyRSIPeriod`. Сообщение: `"%s: ось %s без узла %v — сетка уже минимальной (спека §4.6)"`. gofmt выровняет литерал.

`TestRSIPullbackZoneFilesPinZone` — вычислять `unified` как наличие `plateau_mode_both.json` в `filepath.Dir(path)`.

Новый тест:

```go
// TestRSIPullbackUnifiedGridsStayWide проверяет ширину осей в каталогах единой процедуры —
// тех, где лежит plateau_mode_both.json. Каталоги старых калибровок не проверяются.
func TestRSIPullbackUnifiedGridsStayWide(t *testing.T) {
	for _, path := range rsiPullbackGridFiles(t) {
		if _, err := os.Stat(filepath.Join(filepath.Dir(path), "plateau_mode_both.json")); err != nil {
			continue
		}
		for _, v := range rsiPullbackUnifiedAxisViolations(filepath.Base(path), rsiPullbackPhases(t, path)) {
			t.Errorf("%s/%s", filepath.Base(filepath.Dir(path)), v)
		}
	}
}
```

- [ ] **Step 4: Run tests**

Run: `go test ./internal/service/backtest/ -count=1`
Expected: PASS (включая `TestRSIPullbackZoneFilesPinZone` и `TestRSIPullbackCalFilesValid` на всех существующих каталогах).

- [ ] **Step 5: Commit**

```bash
git add internal/service/backtest/rsi_pullback_zone_grid_test.go
git commit -m "test(rsi_pullback): сторожа единой калибровки — режим файла и минимальная ширина осей

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

---

### Task 5: §8.2, §9, команда `/pullback-calibrate`, CLAUDE.md

**Files:**
- Modify: `docs/rsi_pullback/strategy.md` (строки таблицы §2 для zone-полей, §8.2 целиком, §9 — пункт)
- Create: `.claude/commands/pullback-calibrate.md`
- Delete: `.claude/commands/pullback-zone-calibrate.md`
- Modify: `CLAUDE.md`
- Modify: `docs/superpowers/specs/2026-09-30-rsi-pullback-zone-only-calibration-design.md` (пометка в шапке)

- [ ] **Step 1: §2.** В таблице параметров заменить колонку-примечание:
  - `UseZoneEntry`: `режим 2 — zone-only процедура (§8.2); для режима 1 процедуры пока нет` → `режим выбирает единая калибровка (§8.2)`;
  - `ZoneRSIPeriod`, `ZoneRSILower`, `ZoneEMAPeriod`: `в zone-only процедуре (§8.2)` → `тема zone_entry/zone_trend единой калибровки (§8.2)` (для `ZoneEMAPeriod` — `zone_trend`, для двух других — `zone_entry`).

- [ ] **Step 2: §8.2.** Заменить раздел от `### 8.2. Калибровка тикера в режиме «только zone»` до строки перед `## 9.` на:

```markdown
### 8.2. Единая калибровка: выбор режима входа

У тикера два входа — pullback (§3) и zone (§3.1) — и общие выходы (§4). Единая процедура
калибрует оба входа, выбирает режим `UseZoneEntry` и калибрует выходы поверх входа выбранного
режима; пошаговый протокол — команда `/pullback-calibrate <TICKER>`
(`.claude/commands/pullback-calibrate.md`). Исходов четыре: 0 (только pullback), 1 (оба входа),
2 (только zone) и отказ.

**Три baseline** на дефолтных выходах: `pullback` — дефолты ядра; `zone` — якорь
`UseZoneEntry = 2`, `ZoneRSIPeriod = 4`, `ZoneRSILower = 25`, `ZoneEMAPeriod = 200`; `both` —
дефолты ядра плюс якорь zone при `UseZoneEntry = 1`. Вход, чей baseline на полном окне дал меньше
20 сделок, неторгуем: его темы не гоняются.

**Входы калибруются независимо.** Pullback — темами §8.0–§8.1 при `UseZoneEntry = 0`; zone — темами
`zone_entry` (`ZoneRSIPeriod` × `ZoneRSILower`) и `zone_trend` (`ZoneEMAPeriod`) над якорем. Выходы
на обоих этапах — дефолты ядра.

**Выбор режима.** Три фиксированные точки (`plateau_mode_{pullback,zone,both}`) — walk-forward.
Режим допущен при pooled OOS PF ≥ 1.0 и не меньше 20 сделках пула. Режим 1 дополнительно требует
PF ≥ 1.0 у обоих подпулов раздела «Пул по типу входа»: иначе один вход разбавляет другой. Режим 1
не равен сумме двух других — пока позиция открыта, второй вход молчит, а при совпадении сигналов
входит pullback, — поэтому он меряется только прямым прогоном. Среди допущенных — больший PF, при
разнице меньше 0.05 меньшая просадка, затем одиночный режим раньше режима 1.

**Выходы** — темы `out_exit`, `out_risk` (`StopDailyATR` × `TPDailyATR`), `out_trail`
(`UseTrail` 0/1 × `TrailDailyATR`) поверх входа выбранного режима; каждая фаза задаёт режим явно.
`out_exit` свипает `RSIUpper`, а в режиме 2 — ещё и `RSIPeriod`: в режимах 0 и 1 это длина RSI
pullback-входа и её подбирает тема `entry`. `UseRSIExit` не свипается — RSI-выход обязателен у
каждого зарегистрированного тикера.

**Перепроверка.** Три точки с откалиброванными выходами (`plateau_final_*`) сравниваются тем же
правилом; победил другой режим — переход на него без перекалибровки выходов, один раз. Затем
стоп-условие канона плюс восьмой пункт — ноль SL-выходов у точки на полном окне означает отказ:
стоп, который ни разу не сработал, не защищает. Второй круг — по темам выбранного режима. В прод
идёт точка или один из трёх baseline, прошедший все восемь пунктов.

**Сторожа.** `TestRSIPullbackZoneFilesPinZone` держит режим файла: фаза, задающая `UseZoneEntry`
1 или 2, обязана задать все три zone-поля строго положительными (ноль молча выключает вход);
режим один на файл; `plateau_{base,mode,final}_<m>` задают режим своего суффикса; zone-имена —
только 1 или 2; темы выходов и точка единой процедуры задают режим явно. У незарегистрированного
тикера `UseZoneEntry = 0`, поэтому сетка, забывшая режим, молча считает pullback.
`TestRSIPullbackUnifiedGridsStayWide` держит минимальную ширину осей в каталогах, где лежит
`plateau_mode_both.json`. При первом включении zone-входа на тикере обязательна сверка
`cmd/pullparity` (окно `Lookback` меняется, §7).
```

- [ ] **Step 3: §9.** Первым пунктом списка вставить:

```markdown
- **Раздел «По типу входа».** Одиночный отчёт и walk-forward разбивают сделки по входу, открывшему
  позицию (`pullback`/`zone`, колонка «Вход» журнала): сделок, win rate, PF, net PnL, SL-выходов.
  В режиме 1 это единственный способ увидеть, тянет ли каждый вход свой вес.
```

- [ ] **Step 4: команда.** `git mv .claude/commands/pullback-zone-calibrate.md .claude/commands/pullback-calibrate.md`, затем переписать файл целиком:

````markdown
---
description: Единая калибровка тикера rsi_pullback — входы pullback и zone, выбор режима UseZoneEntry 0/1/2 или отказ, выходы, walk-forward, пакет тикера
argument-hint: <TICKER> [заметки владельца]
---

# Единая калибровка rsi_pullback: $ARGUMENTS

Ты калибруешь rsi_pullback под тикер из аргумента: оба входа (pullback и zone), выбор режима
`UseZoneEntry` (0 — только pullback, 1 — оба, 2 — только zone) или отказ, общие выходы. Сам строишь
сетки, сам гоняешь бэктесты, сам выбираешь и доводишь тикер до пакета. Владелец хочет вердикт и
готовый код, а не меню: выбирай сам и обосновывай числами. Останавливайся только в точках **СТОП**.

Прочитай до начала:

- `docs/superpowers/specs/2026-09-30-rsi-pullback-unified-calibration-design.md` — спека процедуры;
- `docs/rsi_pullback/strategy.md` — §3, §3.1, §4, §7, §8–§8.2, §9;
- `internal/service/trading_strategy/rsi_pullback/strategy/core/core.go` — `Params`, режимы, `Lookback`;
- `docs/superpowers/specs/2026-09-26-mdmg-rsi-pullback-prep-design.md` — образец канона (темы входа,
  правило большинства, соседи плато, гейты A–D, стоп-условие, второй круг, правило прода);
- пакет тикера `internal/service/trading_strategy/rsi_pullback/strategy/<t>/`, `data/params/rsi_pullback/<t>/`
  и память проекта про тикер, если есть.

## 0. Жёсткие правила каждого прогона

- **`-strategy rsi_pullback -interval Minutes30`** в каждой команде `cmd/backtest` (дефолт CLI — `Hour1`).
- **`-months`, `-train-months`, `-test-months`** в каждой команде walk-forward (дефолт `-months` — 12).
- **`-refresh` — только в прогреве кэша** (§1, шаг 2).
- **Прогоны строго последовательно** — параллельные запуски ломают файл кэша.
- Сравниваемые прогоны — в один день; годы и хвосты режь фиксированными датами; дату прогона пиши в
  `_comment` и в отчёт.
- Прогон, упавший на ошибке API (`rpc error: code = Internal ...`), перезапусти, его числа не пиши.
- Отчёты — в `./reports/<TICKER>/...` (вне git).
- **Режим входа.** У незарегистрированного тикера `UseZoneEntry = 0`: сетка, забывшая режим, молча
  считает pullback, забывшая zone-поле при режиме 1/2 — молча даёт ноль сделок. Правила файлов —
  спека §5, сторож `TestRSIPullbackZoneFilesPinZone`.
- **Если литерал тикера уводит поля от дефолтов ядра**, каждая сетка перечисляет их явно значениями
  дефолтов ядра: калибровка идёт с нуля.
- Документация `docs/rsi_pullback/*.md` — только механика; результаты — в doc-comment пакета тикера и
  в `_comment` сеток.
- Ветка `feat/<ticker>-calibration` от `main` (или от ветки с единой процедурой, пока она не в `main`).
  Коммиты на русском, в конце строка `Co-Authored-By` из системной подсказки.

## 1. Разведка и три baseline

1. Ветка.
2. **Прогрев кэша** — единственный прогон с `-refresh`:
   ```
   go run ./cmd/backtest -ticker <T> -strategy rsi_pullback -interval Minutes30 -months 60 -refresh \
     -out ./reports/<T>/warmup
   ```
3. Длина истории — первый бар `data/candles/<T>_Minutes30.json`; схема и `<M>`:
   - ≥ 36 мес — основная **36/12/6** (4 фолда), контрольная **36/18/6** (3 фолда);
   - 24–36 мес — **24/12/3** (4 фолда), контрольная **24/15/3** (3 фолда);
   - меньше 24 мес — **СТОП**, доложи.
4. Файлы `data/params/rsi_pullback/<t>/plateau_base_{pullback,zone,both}.json`: одна фаза `point`,
   все поля выхода по `core.DefaultParams()`;
   - `pullback` — `UseZoneEntry 0`;
   - `zone` — `UseZoneEntry 2`, `ZoneRSIPeriod 4`, `ZoneRSILower 25`, `ZoneEMAPeriod 200`;
   - `both` — `UseZoneEntry 1` и те же zone-поля.
   Сторож ширины включится, когда на этапе 3 появится `plateau_mode_both.json`. `_comment` каждого
   файла — что это и команды.
5. На каждом baseline: полное окно (`-calibrate <файл> -months <M> -min-trades 1`), walk-forward
   основной, контрольной и основной с `-commission 0.001` (`-min-trades 1`). Сверь «Фолдов: N».
6. Сверка типа входа по колонке «Вход»: у `pullback` — только `pullback`, у `zone` — только `zone`, у
   `both` — оба типа. Иначе — **СТОП**.
7. Вход, чей baseline на полном окне дал меньше 20 сделок, — **неторгуемый**: его темы не гоняются,
   режимы с ним выбывают. Оба неторгуемы — **СТОП**, предложи отказ.
8. Профиль каждого baseline: сделки, PF, net, max DD (₽ и %), win rate, expectancy, доли выходов
   SL/TRAIL/TP/RSI, удержание (медиана / p90 / максимум), ночёвки, входы и выходы в выходные и в часы
   02–06, календарные годы и полугодия, хвосты 6 и 12 месяцев, раздел «По типу входа».
9. **Реальный круг издержек:** 2 × `MinPriceIncrement` / цена. Выше 0.2% — порог пункта 4 (§7)
   ужесточается до реального круга.
10. **Дивидендные отсечки** окна — из T-Invest `GetDividends` (образец `reports/_analysis/divprobe/`).
    Объяви до прогонов, не меняй.
11. **Скрипт журнала** `reports/_analysis/<t>_journal.py` (вне git) по образцу
    `reports/_analysis/mdmg_journal.py`: понимает выходы TP/TRAIL/SL/RSI и колонку `entry_kind`
    (последняя в CSV), считает всё из шага 8 и по типам входа. Сверь его вывод с шапкой отчёта baseline.

## 2. Сетки входов

Файлы в `data/params/rsi_pullback/<t>/`. `_comment` каждого: что меряет тема, полная команда запуска
с путём к самому файлу (требует `TestRSIPullbackCalFilesValid`), после прогона — строка
`РЕЗУЛЬТАТ ПРОГОНА <дата>: …` (pooled OOS, пул, пофолдовые PF, голоса фолдов).

Pullback-темы (`UseZoneEntry` не задан или 0): `cal_screen`, `cal_entry`, `cal_trend`, `cal_trend_low`,
`cal_day`, `cal_volume`, `cal_vol_window`, арбитры `cal_trend_spent`, `cal_trend_day`.
Zone-темы (каждая фаза — `UseZoneEntry 2` и три zone-поля): `cal_zone_zone_entry`, `cal_zone_zone_trend`.

Минимальные оси — таблица спеки §4.6 (в коде — `rsiPullbackUnifiedMinAxes`, сторож
`TestRSIPullbackUnifiedGridsStayWide`). Шире можно, уже нельзя. Инварианты: `StopDailyATR` нигде
не 0 и свипается только `cal_out_risk`; периоды ≥ 2 (кроме `EMAFast` 1 в `trend_low`/`trend_spent`);
узел, чьё окно `Lookback` съедает больше ~10% окна при истории < 36 мес, выбрасывается с записью в
`_comment`; упор в край — зонд за краем, не обрезка оси.

Первый коммит — baseline + сетки входов; `go test ./internal/service/backtest/ -run 'RSIPullback' -count=1`.

## 3. Этапы 1–2: walk-forward тем входов и сборка

Каждая тема:
```
go run ./cmd/backtest -ticker <T> -strategy rsi_pullback -interval Minutes30 \
  -calibrate data/params/rsi_pullback/<t>/<файл>.json -out ./reports/<T>/<тема> \
  -months <M> -train-months <TR> -test-months <TE> -min-trades 20 -metric profit_factor
```
(`cal_screen` — с `-min-trades 1`).

**Вырожденный фолд** — меньше пяти сделок OOS или ни одной убыточной: не голосует.
**Правило большинства:** поле уходит от baseline своего входа, только если за значение ≥ 3 из 4
невырожденных фолдов основной схемы (на 3 фолдах — единогласно); 2/2 — не большинство. Источники
полей — как в каноне; `ZoneRSIPeriod`, `ZoneRSILower` — `zone_entry`; `ZoneEMAPeriod` — `zone_trend`.
Таблица: поле → тема → голоса → вырожденные → принято → baseline.

**Соседи плато** для каждого ушедшего поля — значение и два соседа (`plateau_<поле>_<v>.json` или
`plateau_zone_<поле>_<v>.json`), годы и хвосты; плато уже 0.05 PF — «сигнала нет», прямым текстом.

Сборка: `plateau_entry_pullback.json` (baseline `pullback` + принятые поля входа) и
`plateau_entry_zone.json` (якорь + принятые zone-поля). Коммит.

## 4. Этап 3: выбор режима

`plateau_mode_pullback.json` (= `plateau_entry_pullback`, `UseZoneEntry 0`), `plateau_mode_zone.json`
(= `plateau_entry_zone`, `UseZoneEntry 2`), `plateau_mode_both.json` (поля обоих, `UseZoneEntry 1`),
выходы — дефолты ядра. Режимы с неторгуемым входом не создаются. Walk-forward основной схемы,
`-min-trades 1`, в один день.

- Допущен: pooled OOS PF ≥ 1.0 и ≥ 20 сделок пула.
- `both` дополнительно: в «Пул по типу входа» оба подпула PF ≥ 1.0 (с числом сделок в сводке).
- Выбор: больший PF → при разнице < 0.05 меньшая max DD % на полном окне → одиночный раньше `both`.
- Никто не допущен — дальше с режимом наибольшего PF; это пишется в сводку.

Таблица режимов (PF, пул, подпулы, DD) — в `_comment` `plateau_mode_both.json`. Коммит.

## 5. Этап 4: выходы

`cal_out_exit.json`, `cal_out_risk.json`, `cal_out_trail.json` — вход выбранного режима
зафиксирован, каждая фаза задаёт `UseZoneEntry` режима (и zone-поля при 1/2):

- `out_exit`: `RSIUpper`; в режиме 2 — `RSIPeriod` × `RSIUpper`;
- `out_risk`: `StopDailyATR` × `TPDailyATR`;
- `out_trail`: `UseTrail` 0/1 × `TrailDailyATR`. `UseRSIExit` не свипается.

Walk-forward и большинство — как в §3. **Гейт A** (потолок стопа по выживаемости уровня и таблице
срабатываний поверх зафиксированного входа, к `min(StopDailyATR, TrailDailyATR при UseTrail=1)`),
**гейт B** (max DD точки не выше baseline выбранного режима ни в ₽, ни в %), **гейт C** (профиль:
падение SL с ростом удержания и ночёвок — капкан, бери более узкий стоп в пределах плато; входов в
выходные — ноль), **гейт D** (сделки через отсечки — ноль) — как в каноне. Сборка — `plateau_point.json`
(режим задан явно). Коммит.

## 6. Этап 5: перепроверка режима

`plateau_final_{pullback,zone,both}.json` — вход каждого режима из §3, выходы из §5. Правило §4 заново.
Победил другой режим — переходи на него (выходы общие), выходы не перекалибровывай; переход один.
Точка = `plateau_final_<победитель>`, её копия — `plateau_point.json`.

## 7. Проверка точки — восемь пунктов стоп-условия

`plateau_point.json` (`-min-trades 1`), в один день: основная схема, контрольная, основная с
`-commission 0.001` (или реальным кругом), полное окно.

1. pooled OOS PF < 1.0 на основной схеме;
2. меньше 20 сделок в пуле OOS основной;
3. pooled OOS PF < 1.0 на контрольной;
4. pooled OOS PF < 1.0 при удвоенных (или реальных) издержках;
5. убыточен один из двух последних полных календарных лет или текущий год;
6. хвост 6 или 12 (по дате входа) с PF < 1.0, либо меньше пяти сделок в хвосте 6; знак хвоста 6 сверь
   с объединённым OOS PF двух последних фолдов (net = `OOS NetPnL%` × 1000, `GL = net/(PF−1)`,
   `GP = PF·GL`; формулу сначала проверь на baseline) — знаки разошлись: **СТОП**;
7. провал гейта C или D;
8. **ноль SL-выходов у точки на полном окне.**

Гейты A и B — ограничения сборки; точку, которую внутри них не собрать, считай сработавшей по пункту 1.

## 8. Второй круг — только при провале первого

Темы выбранного режима: `cal2_<тема>.json` (входа) и `cal2_out_<тема>.json` (выходов, режим задан
явно), точка — `r2_point.json`. Зоны — по фактическим голосам первого круга, шаг — половина шага
первого круга, не больше четырёх тем, каждое поле свипает ровно одна тема. Большинство — в зоне ±1 шаг.
`-min-trades 1`. Пункт 2 снят (пул меньше десяти закрывает работу); пункты 1, 3–8 в силе.

## 9. Правило прода

Кандидаты — точка (первого или второго круга) и три baseline; допущен прошедший все восемь пунктов.
Выбор: больший pooled OOS основной → при разнице < 0.05 меньшая max DD % → меньше убыточных полугодий
→ baseline `pullback`. Никто не допущен — **отказ**: литерал не меняется, сетки остаются протоколом,
в памяти — причина, сводка владельцу, без мержа.

## 10. Финал при допуске

1. Литерал пакета `internal/service/trading_strategy/rsi_pullback/strategy/<t>/<t>.go`: `UseZoneEntry`
   вердикта (`0`, `core.ZoneEntryAlso` или `core.ZoneEntryOnly`), zone-поля при 1/2, поля победителя;
   doc-comment — дата, протокол, таблицы голосов и выбора режима (этапы 3 и 5), плато, гейты A–D,
   восемь пунктов с числами, кандидат против baseline, риски. Тест-снимок литерала обновить. Нет
   пакета — завести по образцу соседей и алиас в `internal/service/backtest/rsi_pullback_registry.go`
   и `internal/service/trading_strategy/rsi_pullback/live/registry.go`.
2. При `UseZoneEntry` 1 или 2 — `go run ./cmd/pullparity -tickers <T> -months <M>`: ноль расхождений.
3. `go run ./cmd/backtest -ticker <T> -strategy rsi_pullback -interval Minutes30 -months <M>` без
   `-calibrate` — ровно числа точки (реестр взял литерал).
4. `./bin/mage ci` зелёный.
5. Коммиты по смыслу.
6. **СТОП:** сводка владельцу и вопрос про мерж и `RSI_PULLBACK_TICKERS` (`env/prod.env`,
   `.example`-файлы, сторож env).
7. Память: проектная запись тикера и строка в `MEMORY.md`.

## 11. Сводка владельцу

Коротко, по-русски: вердикт (режим 0/1/2 или отказ) и кандидат; литерал и отличие от baseline;
таблица выбора режима на этапах 3 и 5 с подпулами; таблица тем (pooled/пул/голоса); кандидат против
baseline на всех схемах; восемь пунктов с числами; главные риски строкой; что осталось владельцу.
````

- [ ] **Step 5: CLAUDE.md.** В строке Layout заменить
`режим \`UseZoneEntry=2\` — только zone-вход, калибровка такого тикера — \`/pullback-zone-calibrate <TICKER>\` (§8.2)`
на
`режимы \`UseZoneEntry\` 0/1/2 (pullback / оба / только zone), калибровка тикера с выбором режима — \`/pullback-calibrate <TICKER>\` (§8.2)`.

- [ ] **Step 6: пометка в старой спеке.** Под строкой `Дата: 2026-09-30. Ветка: ...` в
`docs/superpowers/specs/2026-09-30-rsi-pullback-zone-only-calibration-design.md` вставить абзац:

```markdown
> **Заменена** спекой `2026-09-30-rsi-pullback-unified-calibration-design.md`: режим «только zone»
> теперь один из исходов единой процедуры `/pullback-calibrate`, команда `/pullback-zone-calibrate` удалена.
```

- [ ] **Step 7: проверки.**

Run: `grep -rn "pullback-zone-calibrate" --include=*.md --include=*.go . | grep -v "^./docs/superpowers/"` — пусто.
Run: `./bin/mage ci`
Expected: exit 0, lint 0 issues.

- [ ] **Step 8: Commit**

```bash
git add -A docs/rsi_pullback/strategy.md .claude/commands/ CLAUDE.md docs/superpowers/specs/2026-09-30-rsi-pullback-zone-only-calibration-design.md
git commit -m "docs(rsi_pullback): единая калибровка — §8.2, раздел по типу входа в §9, команда /pullback-calibrate

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```
