# RSI pullback screener — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Скринер, который прогоняет настоящий движок `rsi_pullback` по вселенной торгуемых RUB-акций MOEX на фиксированной сетке из 24 конфигураций и выдаёт шортлист тикеров, достойных фазовой калибровки.

**Architecture:** Чистое ядро в `internal/service/backtest/` (сетка, статистика, агрегация, рендер — без сети и без флагов) плюс тонкая CLI-команда `cmd/pullscreen`, где живёт весь gRPC/файловый ввод-вывод. Ранжирование — медиана profit factor по сетке, без взвешенного композитного скора.

**Tech Stack:** Go 1.25, существующий движок `internal/domain/backtest` (`backtest.Run`), `pkg/indicators`, `pkg/semaphore`, `pkg/client/grpc`.

**Спека:** `docs/superpowers/specs/2026-08-03-rsi-pullback-screener-design.md` — читать целиком перед началом.

## Global Constraints

- Ветка-основание: `feat/rsi-pullback`. Работать на ней или на ветке от неё.
- Go 1.25; стиль пакета — table-driven тесты, комментарии на английском в коде, документация в `docs/` на русском.
- TDD: сначала падающий тест, потом минимальная реализация. Коммит после каждой задачи.
- Гейт качества — `./bin/mage ci` (lint + `go test -race ./...` + проверка дрейфа моков). `go build ./...` падает на пакете `magefiles` — собирать `go build ./internal/... ./pkg/... ./cmd/...`.
- Пакет `internal/service/backtest` называется `backtest` и при этом импортирует доменный `tinvest/internal/domain/backtest` **без алиаса**, обращаясь к нему как `backtest.Candle`, `backtest.Run`, `backtest.Config`. Это существующий приём пакета (см. `volatility_screen.go`) — повторять его, а не изобретать алиас.
- Пакет параметров стратегии импортируется как `"tinvest/internal/service/trading_strategy/rsi_pullback/strategy/core"` и используется как `core.Params`, `core.NewWithParams` (см. `rsi_pullback_registry.go`).
- Никаких новых зависимостей.

---

### Task 1: Сетка из 24 конфигураций

**Files:**
- Create: `internal/service/backtest/pullback_screen.go`
- Create: `internal/service/backtest/pullback_screen_test.go`

**Interfaces:**
- Consumes: `core.Params` (`internal/service/trading_strategy/rsi_pullback/strategy/core/core.go:30`).
- Produces: `PullbackGrid() []core.Params` — 24 конфигурации в детерминированном порядке; используется во всех последующих задачах и в CLI.

- [ ] **Step 1: Write the failing test**

Создать `internal/service/backtest/pullback_screen_test.go`:

```go
package backtest

import (
	"testing"

	"tinvest/internal/service/trading_strategy/rsi_pullback/strategy/core"
)

func TestPullbackGridHas24UniqueConfigs(t *testing.T) {
	grid := PullbackGrid()
	if len(grid) != 24 {
		t.Fatalf("grid size = %d, want 24 (2 RSIPeriod x 3 RSILower x 2 EMASlow x 2 TPDailyATR)", len(grid))
	}
	seen := make(map[core.Params]bool, len(grid))
	for _, p := range grid {
		if seen[p] {
			t.Fatalf("duplicate config in grid: %+v", p)
		}
		seen[p] = true
	}
}

func TestPullbackGridSweepsOnlyTheFourAxes(t *testing.T) {
	rsiPeriods := map[int]bool{}
	rsiLowers := map[float64]bool{}
	emaSlows := map[int]bool{}
	tps := map[float64]bool{}
	for _, p := range PullbackGrid() {
		rsiPeriods[p.RSIPeriod] = true
		rsiLowers[p.RSILower] = true
		emaSlows[p.EMASlow] = true
		tps[p.TPDailyATR] = true

		// Everything else is pinned: the screener compares tickers, not configurations.
		if p.EMAFast != 20 {
			t.Fatalf("EMAFast = %d, want pinned 20", p.EMAFast)
		}
		if p.StopDailyATR != 0.5 {
			t.Fatalf("StopDailyATR = %v, want pinned 0.5", p.StopDailyATR)
		}
		if p.DailyATRPeriod != 14 {
			t.Fatalf("DailyATRPeriod = %d, want pinned 14", p.DailyATRPeriod)
		}
		if p.UseDayATRGate != 1 || p.FreshDayATR != 0.3 || p.SpentDayATR != 0.8 {
			t.Fatalf("day gate = %d/%v/%v, want pinned 1/0.3/0.8", p.UseDayATRGate, p.FreshDayATR, p.SpentDayATR)
		}
		if p.UseVolume != 0 {
			t.Fatalf("UseVolume = %d, want pinned 0 (the volume gate starves an already thin sample)", p.UseVolume)
		}
		if p.UseRSIExit != 1 || p.RSIUpper != 60 {
			t.Fatalf("RSI exit = %d/%v, want pinned 1/60", p.UseRSIExit, p.RSIUpper)
		}
		if p.UseTrail != 0 || p.TrailDailyATR != 0 {
			t.Fatalf("trail = %d/%v, want pinned off (trailing is tuning, not ticker fitness)", p.UseTrail, p.TrailDailyATR)
		}
	}
	if len(rsiPeriods) != 2 || len(rsiLowers) != 3 || len(emaSlows) != 2 || len(tps) != 2 {
		t.Fatalf("axes = %d/%d/%d/%d, want 2/3/2/2", len(rsiPeriods), len(rsiLowers), len(emaSlows), len(tps))
	}
}

func TestPullbackGridNeverDisablesTheStop(t *testing.T) {
	// Same invariant TestRSIPullbackCalFilesValid enforces on the JSON grids: a
	// multi-day long without a stop is not a configuration the screener may pick.
	for _, p := range PullbackGrid() {
		if p.StopDailyATR <= 0 {
			t.Fatalf("StopDailyATR = %v, want > 0", p.StopDailyATR)
		}
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/service/backtest/ -run TestPullbackGrid -v`
Expected: FAIL — `undefined: PullbackGrid`.

- [ ] **Step 3: Write minimal implementation**

Создать `internal/service/backtest/pullback_screen.go`:

```go
package backtest

import (
	"tinvest/internal/service/trading_strategy/rsi_pullback/strategy/core"
)

// Pinned screener axes. The screener answers "is this ticker worth calibrating",
// so it sweeps only the parameters whose useful range is instrument-specific and
// pins everything else to one value shared by every ticker. See
// docs/superpowers/specs/2026-08-03-rsi-pullback-screener-design.md, section 3.
var (
	gridRSIPeriods = []int{4, 6}
	gridRSILowers  = []float64{10, 15, 20}
	gridEMASlows   = []int{100, 150}
	gridTPDailyATR = []float64{1.0, 1.5}
)

// PullbackGrid returns the 24 configurations every ticker is screened on, in a
// deterministic order. The volume gate is pinned OFF: it cuts trade count harder
// than any other filter, and the screening stage needs a sample, not a filter.
// Trailing is pinned OFF because it is a property of a tuned configuration, not
// of the instrument.
func PullbackGrid() []core.Params {
	out := make([]core.Params, 0, len(gridRSIPeriods)*len(gridRSILowers)*len(gridEMASlows)*len(gridTPDailyATR))
	for _, rsiPeriod := range gridRSIPeriods {
		for _, rsiLower := range gridRSILowers {
			for _, emaSlow := range gridEMASlows {
				for _, tp := range gridTPDailyATR {
					out = append(out, core.Params{
						RSIPeriod:       rsiPeriod,
						RSILower:        rsiLower,
						RSIUpper:        60,
						EMAFast:         20,
						EMASlow:         emaSlow,
						DailyATRPeriod:  14,
						UseDayATRGate:   1,
						FreshDayATR:     0.3,
						SpentDayATR:     0.8,
						StopDailyATR:    0.5,
						TPDailyATR:      tp,
						UseVolume:       0,
						VolBaseDays:     5,
						VolLookbackBars: 3,
						VolMult:         1.2,
						UseRSIExit:      1,
						UseTrail:        0,
						TrailDailyATR:   0,
					})
				}
			}
		}
	}
	return out
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/service/backtest/ -run TestPullbackGrid -v`
Expected: PASS (3 теста).

- [ ] **Step 5: Commit**

```bash
git add internal/service/backtest/pullback_screen.go internal/service/backtest/pullback_screen_test.go
git commit -m "feat(pullscreen): сетка из 24 конфигураций для скринера rsi_pullback"
```

---

### Task 2: Чистая статистика — PF, разрез по времени, медиана, плато

**Files:**
- Modify: `internal/service/backtest/pullback_screen.go`
- Modify: `internal/service/backtest/pullback_screen_test.go`

**Interfaces:**
- Consumes: `backtest.Trade` (`internal/domain/backtest/types.go:40`) — поля `EntryTime time.Time`, `PnL float64`.
- Produces:
  - `profitFactor(trades []backtest.Trade) (pf float64, n int)` — `+Inf` при нулевом убытке и положительной прибыли;
  - `splitTrades(trades []backtest.Trade, split time.Time) (train, holdout []backtest.Trade)`;
  - `medianF(vals []float64) float64`;
  - `clampPF(pf, cap float64) (float64, bool)`.

- [ ] **Step 1: Write the failing test**

Дописать в `internal/service/backtest/pullback_screen_test.go`:

```go
func TestProfitFactor(t *testing.T) {
	tests := []struct {
		name    string
		pnl     []float64
		wantPF  float64
		wantInf bool
		wantN   int
	}{
		{name: "mixed", pnl: []float64{100, -50, 50, -50}, wantPF: 1.5, wantN: 4},
		{name: "no losses is infinite, not gross profit", pnl: []float64{100, 200}, wantInf: true, wantN: 2},
		{name: "no profit", pnl: []float64{-100, -50}, wantPF: 0, wantN: 2},
		{name: "empty", pnl: nil, wantPF: 0, wantN: 0},
		{name: "zero pnl counts as a win side", pnl: []float64{0, -100}, wantPF: 0, wantN: 2},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			trades := make([]backtest.Trade, 0, len(tt.pnl))
			for _, p := range tt.pnl {
				trades = append(trades, backtest.Trade{PnL: p})
			}
			pf, n := profitFactor(trades)
			if n != tt.wantN {
				t.Fatalf("n = %d, want %d", n, tt.wantN)
			}
			if tt.wantInf {
				if !math.IsInf(pf, 1) {
					t.Fatalf("pf = %v, want +Inf", pf)
				}
				return
			}
			if math.Abs(pf-tt.wantPF) > 1e-9 {
				t.Fatalf("pf = %v, want %v", pf, tt.wantPF)
			}
		})
	}
}

func TestSplitTradesByEntryTime(t *testing.T) {
	split := time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC)
	trades := []backtest.Trade{
		{EntryTime: split.AddDate(0, 0, -10), ExitTime: split.AddDate(0, 0, -9), PnL: 1},  // train
		{EntryTime: split.AddDate(0, 0, -1), ExitTime: split.AddDate(0, 0, 3), PnL: 2},    // train: straddles the split, classified by ENTRY
		{EntryTime: split, ExitTime: split.AddDate(0, 0, 1), PnL: 3},                      // holdout: exactly on the boundary
		{EntryTime: split.AddDate(0, 0, 5), ExitTime: split.AddDate(0, 0, 6), PnL: 4},     // holdout
	}
	train, holdout := splitTrades(trades, split)
	if len(train) != 2 || train[0].PnL != 1 || train[1].PnL != 2 {
		t.Fatalf("train = %+v, want the two trades entered before the split", train)
	}
	if len(holdout) != 2 || holdout[0].PnL != 3 || holdout[1].PnL != 4 {
		t.Fatalf("holdout = %+v, want the boundary trade and the later one", holdout)
	}
}

func TestSplitTradesEmptySides(t *testing.T) {
	split := time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC)
	all := []backtest.Trade{{EntryTime: split.AddDate(0, 0, 1)}}
	train, holdout := splitTrades(all, split)
	if len(train) != 0 || len(holdout) != 1 {
		t.Fatalf("train/holdout = %d/%d, want 0/1", len(train), len(holdout))
	}
	train, holdout = splitTrades(nil, split)
	if len(train) != 0 || len(holdout) != 0 {
		t.Fatalf("train/holdout = %d/%d on empty input, want 0/0", len(train), len(holdout))
	}
}

func TestMedianF(t *testing.T) {
	tests := []struct {
		name string
		in   []float64
		want float64
	}{
		{name: "odd", in: []float64{3, 1, 2}, want: 2},
		{name: "even averages the middles", in: []float64{4, 1, 3, 2}, want: 2.5},
		{name: "all zero", in: []float64{0, 0, 0}, want: 0},
		{name: "empty", in: nil, want: 0},
		{name: "single", in: []float64{7}, want: 7},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			in := append([]float64(nil), tt.in...)
			if got := medianF(in); math.Abs(got-tt.want) > 1e-9 {
				t.Fatalf("medianF = %v, want %v", got, tt.want)
			}
			if !reflect.DeepEqual(in, tt.in) {
				t.Fatalf("medianF mutated its input: %v != %v", in, tt.in)
			}
		})
	}
}

func TestClampPF(t *testing.T) {
	if got, capped := clampPF(3, 10); got != 3 || capped {
		t.Fatalf("clampPF(3,10) = %v,%v, want 3,false", got, capped)
	}
	if got, capped := clampPF(math.Inf(1), 10); got != 10 || !capped {
		t.Fatalf("clampPF(+Inf,10) = %v,%v, want 10,true", got, capped)
	}
	if got, capped := clampPF(25, 10); got != 10 || !capped {
		t.Fatalf("clampPF(25,10) = %v,%v, want 10,true", got, capped)
	}
	if got, capped := clampPF(math.Inf(1), 0); !math.IsInf(got, 1) || capped {
		t.Fatalf("clampPF with cap<=0 must pass through, got %v,%v", got, capped)
	}
}
```

Импорты файла тестов расширить до:

```go
import (
	"math"
	"reflect"
	"testing"
	"time"

	"tinvest/internal/domain/backtest"
	"tinvest/internal/service/trading_strategy/rsi_pullback/strategy/core"
)
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/service/backtest/ -run 'TestProfitFactor|TestSplitTrades|TestMedianF|TestClampPF' -v`
Expected: FAIL — `undefined: profitFactor`, `undefined: splitTrades`, `undefined: medianF`, `undefined: clampPF`.

- [ ] **Step 3: Write minimal implementation**

Дописать в `internal/service/backtest/pullback_screen.go` (и расширить импорты файла до `"math"`, `"sort"`, `"time"`, `"tinvest/internal/domain/backtest"`, `core`):

```go
// profitFactor is gross profit over gross loss on an arbitrary SUBSET of trades.
// It deliberately differs from ComputeMetrics on one point: with no losing trade
// it returns +Inf rather than gross profit. The screener takes a MEDIAN across 24
// configurations, and a currency amount masquerading as a ratio would poison it;
// the caller clamps the infinity with clampPF instead.
func profitFactor(trades []backtest.Trade) (float64, int) {
	var gross, loss float64
	for _, t := range trades {
		if t.PnL >= 0 {
			gross += t.PnL
			continue
		}
		loss += -t.PnL
	}
	switch {
	case len(trades) == 0:
		return 0, 0
	case loss == 0 && gross > 0:
		return math.Inf(1), len(trades)
	case loss == 0:
		return 0, len(trades)
	}
	return gross / loss, len(trades)
}

// splitTrades cuts a trade list into the selection window and the holdout by ENTRY
// time: a trade opened before the split belongs to train even if it closed after it,
// because the entry is the decision the screener is grading. A trade entered exactly
// at the split goes to the holdout.
func splitTrades(trades []backtest.Trade, split time.Time) (train, holdout []backtest.Trade) {
	for _, t := range trades {
		if t.EntryTime.Before(split) {
			train = append(train, t)
			continue
		}
		holdout = append(holdout, t)
	}
	return train, holdout
}

// medianF is the median of vals; it copies before sorting so callers keep their order.
func medianF(vals []float64) float64 {
	if len(vals) == 0 {
		return 0
	}
	s := append([]float64(nil), vals...)
	sort.Float64s(s)
	mid := len(s) / 2
	if len(s)%2 == 1 {
		return s[mid]
	}
	return (s[mid-1] + s[mid]) / 2
}

// clampPF caps a profit factor for ranking purposes and reports whether it bit.
// A limit of zero or less disables clamping. (The parameter is named `limit`, not
// `cap`, to avoid shadowing the builtin.)
func clampPF(pf, limit float64) (float64, bool) {
	if limit <= 0 {
		return pf, false
	}
	if pf > limit {
		return limit, true
	}
	return pf, false
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/service/backtest/ -run 'TestProfitFactor|TestSplitTrades|TestMedianF|TestClampPF' -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/service/backtest/pullback_screen.go internal/service/backtest/pullback_screen_test.go
git commit -m "feat(pullscreen): profit factor по подмножеству сделок, разрез train/holdout, медиана"
```

---

### Task 3: Агрегация строки тикера

**Files:**
- Modify: `internal/service/backtest/pullback_screen.go`
- Modify: `internal/service/backtest/pullback_screen_test.go`

**Interfaces:**
- Consumes: `profitFactor`, `splitTrades`, `medianF`, `clampPF` (Task 2); `core.Params` (Task 1).
- Produces:
  - `type ConfigResult struct { Params core.Params; Trades []backtest.Trade }`
  - `type ScreenOpts struct { PFCap, PlateauPF float64; PlateauTrades int; Cash, Fraction, Commission float64 }`
  - `type PullbackRow struct { ... }` (полный состав — в коде ниже)
  - `func DefaultScreenOpts() ScreenOpts`
  - `func Aggregate(ticker, name string, results []ConfigResult, split time.Time, opts ScreenOpts) PullbackRow`

- [ ] **Step 1: Write the failing test**

Дописать в `internal/service/backtest/pullback_screen_test.go`:

```go
// screenTrade builds one trade entered `dayOffset` days from `base` with the given PnL.
func screenTrade(base time.Time, dayOffset int, pnl float64) backtest.Trade {
	entry := base.AddDate(0, 0, dayOffset)
	return backtest.Trade{EntryTime: entry, ExitTime: entry.AddDate(0, 0, 1), PnL: pnl}
}

func TestAggregateRanksByMedianNotBest(t *testing.T) {
	base := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	split := base.AddDate(0, 6, 0)
	grid := PullbackGrid()

	// 23 mediocre configurations (PF 1.0) and one lucky outlier (PF 5.0).
	results := make([]ConfigResult, 0, len(grid))
	for i, p := range grid {
		trades := []backtest.Trade{screenTrade(base, 1, 100), screenTrade(base, 2, -100)}
		if i == 0 {
			trades = []backtest.Trade{screenTrade(base, 1, 500), screenTrade(base, 2, -100)}
		}
		results = append(results, ConfigResult{Params: p, Trades: trades})
	}
	row := Aggregate("XXXX", "Test", results, split, DefaultScreenOpts())

	if math.Abs(row.PFMed-1.0) > 1e-9 {
		t.Fatalf("PFMed = %v, want 1.0 — the median must ignore the single lucky config", row.PFMed)
	}
	if math.Abs(row.BestPF-5.0) > 1e-9 {
		t.Fatalf("BestPF = %v, want 5.0", row.BestPF)
	}
	if row.Best != grid[0] {
		t.Fatalf("Best = %+v, want the outlier config %+v", row.Best, grid[0])
	}
}

func TestAggregatePlateauShare(t *testing.T) {
	base := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	split := base.AddDate(0, 6, 0)
	grid := PullbackGrid()
	opts := DefaultScreenOpts() // PlateauPF 1.3, PlateauTrades 10

	results := make([]ConfigResult, 0, len(grid))
	for i, p := range grid {
		var trades []backtest.Trade
		switch {
		case i < 12: // PF 2.0 on 12 trades -> counts toward the plateau
			for d := 0; d < 8; d++ {
				trades = append(trades, screenTrade(base, d, 100))
			}
			for d := 8; d < 12; d++ {
				trades = append(trades, screenTrade(base, d, -100))
			}
		case i < 18: // PF 2.0 but only 3 trades -> too thin to count
			trades = []backtest.Trade{screenTrade(base, 0, 100), screenTrade(base, 1, 100), screenTrade(base, 2, -100)}
		default: // PF 0.5 on 12 trades -> below the PF bar
			for d := 0; d < 4; d++ {
				trades = append(trades, screenTrade(base, d, 100))
			}
			for d := 4; d < 12; d++ {
				trades = append(trades, screenTrade(base, d, -100))
			}
		}
		results = append(results, ConfigResult{Params: p, Trades: trades})
	}
	row := Aggregate("XXXX", "Test", results, split, opts)

	if math.Abs(row.Plateau-0.5) > 1e-9 {
		t.Fatalf("Plateau = %v, want 0.5 (12 of 24 configs clear both PF>=1.3 and trades>=10)", row.Plateau)
	}
}

func TestAggregateClampsUnboundedPF(t *testing.T) {
	base := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	split := base.AddDate(0, 6, 0)
	results := make([]ConfigResult, 0, len(PullbackGrid()))
	for _, p := range PullbackGrid() {
		// Every configuration wins twice and never loses: PF is +Inf before clamping.
		results = append(results, ConfigResult{Params: p, Trades: []backtest.Trade{
			screenTrade(base, 1, 1000), screenTrade(base, 2, 2000),
		}})
	}
	row := Aggregate("XXXX", "Test", results, split, DefaultScreenOpts())

	if row.PFMed != 10 {
		t.Fatalf("PFMed = %v, want the 10.0 cap", row.PFMed)
	}
	if row.Capped != 24 {
		t.Fatalf("Capped = %d, want 24", row.Capped)
	}
}

func TestAggregateSplitsTrainAndHoldout(t *testing.T) {
	base := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	split := base.AddDate(0, 0, 100)
	results := make([]ConfigResult, 0, len(PullbackGrid()))
	for _, p := range PullbackGrid() {
		results = append(results, ConfigResult{Params: p, Trades: []backtest.Trade{
			screenTrade(base, 1, 200),   // train: PF 2.0
			screenTrade(base, 2, -100),  // train
			screenTrade(base, 150, 50),  // holdout: PF 0.5
			screenTrade(base, 151, -100), // holdout
		}})
	}
	row := Aggregate("XXXX", "Test", results, split, DefaultScreenOpts())

	if math.Abs(row.PFMed-2.0) > 1e-9 {
		t.Fatalf("PFMed(train) = %v, want 2.0", row.PFMed)
	}
	if math.Abs(row.PFMedHO-0.5) > 1e-9 {
		t.Fatalf("PFMed(holdout) = %v, want 0.5", row.PFMedHO)
	}
	if row.TradesMed != 2 || row.TradesMedHO != 2 {
		t.Fatalf("TradesMed/HO = %v/%v, want 2/2", row.TradesMed, row.TradesMedHO)
	}
}

func TestAggregateNoSignals(t *testing.T) {
	split := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)
	results := make([]ConfigResult, 0, len(PullbackGrid()))
	for _, p := range PullbackGrid() {
		results = append(results, ConfigResult{Params: p, Trades: nil})
	}
	row := Aggregate("XXXX", "Test", results, split, DefaultScreenOpts())
	if !row.NoSignals {
		t.Fatal("NoSignals = false, want true when every configuration produced zero trades")
	}
	if row.PFMed != 0 || row.Plateau != 0 {
		t.Fatalf("PFMed/Plateau = %v/%v, want 0/0", row.PFMed, row.Plateau)
	}
}

func TestAggregateOneTradingConfigIsNotNoSignals(t *testing.T) {
	base := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	split := base.AddDate(0, 6, 0)
	results := make([]ConfigResult, 0, len(PullbackGrid()))
	for i, p := range PullbackGrid() {
		var trades []backtest.Trade
		if i == 3 {
			trades = []backtest.Trade{screenTrade(base, 1, 100), screenTrade(base, 2, -50)}
		}
		results = append(results, ConfigResult{Params: p, Trades: trades})
	}
	row := Aggregate("XXXX", "Test", results, split, DefaultScreenOpts())
	if row.NoSignals {
		t.Fatal("NoSignals = true, want false when at least one configuration traded")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/service/backtest/ -run TestAggregate -v`
Expected: FAIL — `undefined: ConfigResult`, `undefined: Aggregate`, `undefined: DefaultScreenOpts`.

- [ ] **Step 3: Write minimal implementation**

Дописать в `internal/service/backtest/pullback_screen.go`:

```go
// ConfigResult is one grid configuration's trade list on one ticker.
type ConfigResult struct {
	Params core.Params
	Trades []backtest.Trade
}

// ScreenOpts are the knobs of one screening run.
type ScreenOpts struct {
	PFCap         float64 // ranking cap on profit factor; 0 disables clamping
	PlateauPF     float64 // a configuration joins the plateau at this profit factor
	PlateauTrades int     // ...and at this many trades
	Cash          float64 // mock portfolio starting cash
	Fraction      float64 // fraction of cash per entry
	Commission    float64 // commission as a fraction of turnover, per side
}

// DefaultScreenOpts are the screener's defaults; the CLI overrides them by flag.
func DefaultScreenOpts() ScreenOpts {
	return ScreenOpts{
		PFCap:         10,
		PlateauPF:     1.3,
		PlateauTrades: 10,
		Cash:          100000,
		Fraction:      1.0,
		Commission:    0.0005,
	}
}

// PullbackRow is one ticker's screening result.
type PullbackRow struct {
	Ticker      string
	Name        string
	TurnoverM   float64 // mean daily turnover, millions of RUB (filled by ScreenTicker)
	DailyATRPct float64 // mean weekday daily ATR as a percentage of close (filled by ScreenTicker)
	Bars        int     // 30-minute bars the run replayed (filled by ScreenTicker)

	PFMed     float64 // MEDIAN profit factor across the grid on the train window: the ranking key
	TradesMed float64 // median trade count on the train window
	Plateau   float64 // share of configurations clearing PlateauPF at PlateauTrades trades
	Capped    int     // configurations whose train profit factor hit PFCap

	PFMedHO     float64 // median profit factor on the holdout window: a red flag, never a ranking key
	TradesMedHO float64

	Best      core.Params // configuration with the highest train profit factor (reference only)
	BestPF    float64
	NoSignals bool // every configuration produced zero trades: profit factor does not exist
}

// Aggregate reduces one ticker's per-configuration results to a report row. The
// ranking key is the MEDIAN profit factor, never the best one: across 271 tickers
// and 24 configurations the maximum is a lottery, while the median asks whether the
// strategy works across a band of parameters.
func Aggregate(ticker, name string, results []ConfigResult, split time.Time, opts ScreenOpts) PullbackRow {
	row := PullbackRow{Ticker: ticker, Name: name, NoSignals: true}
	pfs := make([]float64, 0, len(results))
	counts := make([]float64, 0, len(results))
	pfsHO := make([]float64, 0, len(results))
	countsHO := make([]float64, 0, len(results))
	var plateau int

	for _, r := range results {
		train, holdout := splitTrades(r.Trades, split)

		pf, n := profitFactor(train)
		if n > 0 {
			row.NoSignals = false
		}
		if pf > row.BestPF {
			row.BestPF, row.Best = pf, r.Params
		}
		pf, capped := clampPF(pf, opts.PFCap)
		if capped {
			row.Capped++
		}
		if pf >= opts.PlateauPF && n >= opts.PlateauTrades {
			plateau++
		}
		pfs = append(pfs, pf)
		counts = append(counts, float64(n))

		pfHO, nHO := profitFactor(holdout)
		if nHO > 0 {
			row.NoSignals = false
		}
		pfHO, _ = clampPF(pfHO, opts.PFCap)
		pfsHO = append(pfsHO, pfHO)
		countsHO = append(countsHO, float64(nHO))
	}

	row.BestPF, _ = clampPF(row.BestPF, opts.PFCap)
	row.PFMed = medianF(pfs)
	row.TradesMed = medianF(counts)
	row.PFMedHO = medianF(pfsHO)
	row.TradesMedHO = medianF(countsHO)
	if len(results) > 0 {
		row.Plateau = float64(plateau) / float64(len(results))
	}
	return row
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/service/backtest/ -run TestAggregate -v`
Expected: PASS (6 тестов).

- [ ] **Step 5: Commit**

```bash
git add internal/service/backtest/pullback_screen.go internal/service/backtest/pullback_screen_test.go
git commit -m "feat(pullscreen): агрегация тикера — медиана PF, плато, holdout, секция без сигналов"
```

---

### Task 4: Прогон движка и дневной ATR%

**Files:**
- Modify: `internal/service/backtest/pullback_screen.go`
- Modify: `internal/service/backtest/pullback_screen_test.go`

**Interfaces:**
- Consumes: `backtest.Run(s strategy.Strategy, candles, dailyCandles, htfCandles []backtest.Candle, cfg backtest.Config) backtest.Result` (`internal/domain/backtest/engine.go:152`); `core.NewWithParams(ticker string, p core.Params) *core.Strategy` (`core.go:82`); `backtest.MeanDailyTurnoverM(candles []backtest.Candle, lot int32) float64` (`internal/domain/backtest/pullback.go:53`); `indicators.ATRSeries(highs, lows, closes []float64, period int) []float64` (`pkg/indicators/atr.go:11`).
- Produces:
  - `MeanDailyATRPct(daily []backtest.Candle, period int) float64`
  - `ScreenTicker(ticker, name string, bars, daily []backtest.Candle, lot int32, cfgs []core.Params, split time.Time, opts ScreenOpts) PullbackRow`

- [ ] **Step 1: Write the failing test**

Дописать в `internal/service/backtest/pullback_screen_test.go`:

```go
// dailyCandlesMSK builds `n` consecutive daily candles starting at 2026-01-05 (a
// Monday) with a constant true range of `rangePct` percent around `close`, in MSK.
func dailyCandlesMSK(n int, closePrice, rangePct float64) []backtest.Candle {
	msk, err := time.LoadLocation("Europe/Moscow")
	if err != nil {
		msk = time.UTC
	}
	base := time.Date(2026, 1, 5, 10, 0, 0, 0, msk) // Monday
	out := make([]backtest.Candle, 0, n)
	half := closePrice * rangePct / 100 / 2
	for i := 0; i < n; i++ {
		out = append(out, backtest.Candle{
			Time:  base.AddDate(0, 0, i),
			Open:  closePrice,
			High:  closePrice + half,
			Low:   closePrice - half,
			Close: closePrice,
			Volume: 1000,
		})
	}
	return out
}

func TestMeanDailyATRPct(t *testing.T) {
	// 40 days of a constant 2% range: ATR% must converge on 2%.
	got := MeanDailyATRPct(dailyCandlesMSK(40, 100, 2), 14)
	if math.Abs(got-2.0) > 0.2 {
		t.Fatalf("MeanDailyATRPct = %v, want ~2.0", got)
	}
}

func TestMeanDailyATRPctDropsWeekends(t *testing.T) {
	// Weekend bars are 10x narrower. Leaving them in drags the mean down; the
	// strategy itself filters them out (docs/rsi_pullback/strategy.md, section 5)
	// and the screener must measure the same ruler.
	full := dailyCandlesMSK(60, 100, 2)
	msk := full[0].Time.Location()
	for i := range full {
		wd := full[i].Time.In(msk).Weekday()
		if wd == time.Saturday || wd == time.Sunday {
			full[i].High = 100.1
			full[i].Low = 99.9
		}
	}
	got := MeanDailyATRPct(full, 14)
	if math.Abs(got-2.0) > 0.3 {
		t.Fatalf("MeanDailyATRPct = %v, want ~2.0 — weekend bars must be dropped before the ATR", got)
	}
}

func TestMeanDailyATRPctInsufficientData(t *testing.T) {
	if got := MeanDailyATRPct(dailyCandlesMSK(5, 100, 2), 14); got != 0 {
		t.Fatalf("MeanDailyATRPct = %v on 5 bars with period 14, want 0", got)
	}
	if got := MeanDailyATRPct(nil, 14); got != 0 {
		t.Fatalf("MeanDailyATRPct = %v on empty input, want 0", got)
	}
}

func TestScreenTickerFlatSeriesProducesNoSignals(t *testing.T) {
	// A dead-flat series never crosses RSI down through the lower band, so no
	// configuration can enter: the row must land in the "no signals" bucket rather
	// than in the ranking with a fabricated profit factor.
	bars := tinyCandles(600)
	for i := range bars {
		bars[i].Open, bars[i].High, bars[i].Low, bars[i].Close = 100, 100, 100, 100
	}
	daily := dailyCandlesMSK(60, 100, 2)
	split := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)

	row := ScreenTicker("XXXX", "Test", bars, daily, 1, PullbackGrid(), split, DefaultScreenOpts())

	if !row.NoSignals {
		t.Fatalf("NoSignals = false on a flat series, row = %+v", row)
	}
	if row.Bars != len(bars) {
		t.Fatalf("Bars = %d, want %d", row.Bars, len(bars))
	}
	if row.DailyATRPct <= 0 {
		t.Fatalf("DailyATRPct = %v, want the daily series to be measured even with no trades", row.DailyATRPct)
	}
	if row.TurnoverM <= 0 {
		t.Fatalf("TurnoverM = %v, want > 0", row.TurnoverM)
	}
}

func TestScreenTickerRunsTheGridParamsVerbatim(t *testing.T) {
	// The screener compares tickers on ONE grid. UGLD is a REGISTERED ticker whose
	// package carries a calibrated literal (RSIPeriod 6, EMASlow 150, UseVolume 1,
	// UseTrail 1); grading it on that literal instead of the grid config would make
	// its row incomparable with the other 268. This pins the equivalence: one grid
	// config through ScreenTicker must reproduce a direct engine run with that same
	// config, bit for bit.
	bars := tinyCandles(600)
	daily := dailyCandlesMSK(60, 100, 2)
	split := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)
	opts := DefaultScreenOpts()

	cfg := PullbackGrid()[0]
	want := backtest.Run(
		core.NewWithParams("UGLD", cfg),
		bars, daily, nil,
		backtest.Config{InitialCash: opts.Cash, Fraction: opts.Fraction, Commission: opts.Commission, Lot: 1},
	)
	wantPF, wantN := profitFactor(want.Trades)

	row := ScreenTicker("UGLD", "calibrated", bars, daily, 1, []core.Params{cfg}, split, opts)

	gotPF, _ := clampPF(wantPF, opts.PFCap)
	trainWant, _ := splitTrades(want.Trades, split)
	trainPF, trainN := profitFactor(trainWant)
	trainPF, _ = clampPF(trainPF, opts.PFCap)

	if row.PFMed != trainPF {
		t.Fatalf("PFMed = %v, want %v — ScreenTicker must run the grid config, not the ticker's calibrated literal (whole-window PF was %v on %d trades)",
			row.PFMed, trainPF, gotPF, wantN)
	}
	if row.TradesMed != float64(trainN) {
		t.Fatalf("TradesMed = %v, want %v", row.TradesMed, float64(trainN))
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/service/backtest/ -run 'TestMeanDailyATRPct|TestScreenTicker' -v`
Expected: FAIL — `undefined: MeanDailyATRPct`, `undefined: ScreenTicker`.

- [ ] **Step 3: Write minimal implementation**

Дописать в `internal/service/backtest/pullback_screen.go` (импорты дополнить `"tinvest/pkg/indicators"`):

```go
// screenMSK anchors the weekday rule to Moscow, matching the strategy core.
var screenMSK = func() *time.Location {
	loc, err := time.LoadLocation("Europe/Moscow")
	if err != nil {
		return time.UTC
	}
	return loc
}()

// MeanDailyATRPct is the mean of ATR(period)/close over WEEKDAY daily candles, in
// percent. Weekend bars are dropped for the same reason the strategy drops them:
// MOEX weekend sessions are 3-4x narrower, and leaving them in understates the daily
// ATR by 9-16% (docs/rsi_pullback/strategy.md, section 5). Returns 0 when the series
// cannot support the calculation.
func MeanDailyATRPct(daily []backtest.Candle, period int) float64 {
	if period <= 0 {
		return 0
	}
	highs := make([]float64, 0, len(daily))
	lows := make([]float64, 0, len(daily))
	closes := make([]float64, 0, len(daily))
	for _, c := range daily {
		switch c.Time.In(screenMSK).Weekday() {
		case time.Saturday, time.Sunday:
			continue
		}
		highs = append(highs, c.High)
		lows = append(lows, c.Low)
		closes = append(closes, c.Close)
	}
	if len(closes) < period+1 {
		return 0
	}
	atr := indicators.ATRSeries(highs, lows, closes, period)
	var sum float64
	var n int
	for i := range atr {
		if atr[i] <= 0 || closes[i] <= 0 {
			continue
		}
		sum += atr[i] / closes[i]
		n++
	}
	if n == 0 {
		return 0
	}
	return sum / float64(n) * 100
}

// ScreenTicker replays every grid configuration over one ticker and reduces the runs
// to a report row. The strategy is built directly with core.NewWithParams and NOT
// through RSIPullbackLookupOrGeneric: registered tickers (GAZP, T, UGLD) carry their
// own calibrated literals, and grading them on those would make their rows
// incomparable with the other 268.
func ScreenTicker(ticker, name string, bars, daily []backtest.Candle, lot int32,
	cfgs []core.Params, split time.Time, opts ScreenOpts,
) PullbackRow {
	cfg := backtest.Config{
		InitialCash: opts.Cash,
		Fraction:    opts.Fraction,
		Commission:  opts.Commission,
		Lot:         lot,
	}
	results := make([]ConfigResult, 0, len(cfgs))
	for _, p := range cfgs {
		// rsi_pullback needs no higher-timeframe series: htfCandles is nil.
		res := backtest.Run(core.NewWithParams(ticker, p), bars, daily, nil, cfg)
		results = append(results, ConfigResult{Params: p, Trades: res.Trades})
	}
	row := Aggregate(ticker, name, results, split, opts)
	row.Bars = len(bars)
	row.TurnoverM = backtest.MeanDailyTurnoverM(bars, lot)
	row.DailyATRPct = MeanDailyATRPct(daily, screenDailyATRPeriod)
	return row
}

// screenDailyATRPeriod matches the strategy's fixed DailyATRPeriod: the ATR% column
// must be measured with the same ruler the strategy sizes its stop with.
const screenDailyATRPeriod = 14
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/service/backtest/ -run 'TestMeanDailyATRPct|TestScreenTicker' -v`
Expected: PASS (5 тестов).

- [ ] **Step 5: Run the whole package and commit**

```bash
go test ./internal/service/backtest/ -race
git add internal/service/backtest/pullback_screen.go internal/service/backtest/pullback_screen_test.go
git commit -m "feat(pullscreen): прогон сетки через движок и дневной ATR% по будням"
```

---

### Task 5: Отчёт — гейты, распределение по вселенной, markdown

**Files:**
- Create: `internal/service/backtest/pullback_screen_report.go`
- Create: `internal/service/backtest/pullback_screen_report_test.go`

**Interfaces:**
- Consumes: `PullbackRow`, `ScreenOpts` (Task 3).
- Produces:
  - `type ScreenMeta struct { Months, HoldoutMonths, TopN int; Split time.Time; MinTurnoverM, MinATRPct, PFCap float64; Scanned, Passed, Skipped int }`
  - `func FilterAndRank(rows []PullbackRow, minTurnoverM, minATRPct float64) (ranked, noSignals, rejected []PullbackRow)`
  - `type PFDist struct { Min, Q1, Median, Q3, Max, ShareAbove15 float64; N int }`
  - `func Distribution(ranked []PullbackRow) PFDist`
  - `func RenderPullbackScreenMarkdown(ranked, noSignals []PullbackRow, meta ScreenMeta) string`

- [ ] **Step 1: Write the failing test**

Создать `internal/service/backtest/pullback_screen_report_test.go`:

```go
package backtest

import (
	"math"
	"strings"
	"testing"
	"time"
)

func TestFilterAndRankGates(t *testing.T) {
	rows := []PullbackRow{
		{Ticker: "GOOD", TurnoverM: 100, DailyATRPct: 2.0, PFMed: 1.2},
		{Ticker: "THIN", TurnoverM: 10, DailyATRPct: 2.0, PFMed: 3.0},  // liquidity gate
		{Ticker: "CALM", TurnoverM: 100, DailyATRPct: 0.8, PFMed: 3.0}, // ATR% gate
		{Ticker: "QUIET", TurnoverM: 100, DailyATRPct: 2.0, NoSignals: true},
		{Ticker: "BEST", TurnoverM: 100, DailyATRPct: 2.0, PFMed: 2.5},
	}
	ranked, noSignals, rejected := FilterAndRank(rows, 50, 1.5)

	if len(ranked) != 2 || ranked[0].Ticker != "BEST" || ranked[1].Ticker != "GOOD" {
		t.Fatalf("ranked = %+v, want BEST then GOOD sorted by PFMed desc", ranked)
	}
	if len(noSignals) != 1 || noSignals[0].Ticker != "QUIET" {
		t.Fatalf("noSignals = %+v, want QUIET", noSignals)
	}
	if len(rejected) != 2 {
		t.Fatalf("rejected = %+v, want THIN and CALM", rejected)
	}
}

func TestFilterAndRankNoSignalsGatedToo(t *testing.T) {
	// A no-signal ticker that also fails liquidity is a rejection, not a "no signals"
	// row: the report's no-signal section is about tickers that passed the gates.
	rows := []PullbackRow{{Ticker: "THIN", TurnoverM: 1, DailyATRPct: 2.0, NoSignals: true}}
	ranked, noSignals, rejected := FilterAndRank(rows, 50, 1.5)
	if len(ranked) != 0 || len(noSignals) != 0 || len(rejected) != 1 {
		t.Fatalf("ranked/noSignals/rejected = %d/%d/%d, want 0/0/1", len(ranked), len(noSignals), len(rejected))
	}
}

func TestDistribution(t *testing.T) {
	rows := []PullbackRow{
		{PFMed: 0.5}, {PFMed: 1.0}, {PFMed: 1.5}, {PFMed: 2.0}, {PFMed: 3.0},
	}
	d := Distribution(rows)
	if d.N != 5 {
		t.Fatalf("N = %d, want 5", d.N)
	}
	if math.Abs(d.Min-0.5) > 1e-9 || math.Abs(d.Max-3.0) > 1e-9 {
		t.Fatalf("min/max = %v/%v, want 0.5/3.0", d.Min, d.Max)
	}
	if math.Abs(d.Median-1.5) > 1e-9 {
		t.Fatalf("median = %v, want 1.5", d.Median)
	}
	if math.Abs(d.ShareAbove15-0.6) > 1e-9 {
		t.Fatalf("ShareAbove15 = %v, want 0.6 (three of five at or above 1.5)", d.ShareAbove15)
	}
}

func TestDistributionEmpty(t *testing.T) {
	d := Distribution(nil)
	if d.N != 0 || d.Median != 0 || d.ShareAbove15 != 0 {
		t.Fatalf("Distribution(nil) = %+v, want the zero value", d)
	}
}

func TestRenderPullbackScreenMarkdown(t *testing.T) {
	meta := ScreenMeta{
		Months: 36, HoldoutMonths: 6, TopN: 50,
		Split:        time.Date(2026, 2, 3, 0, 0, 0, 0, time.UTC),
		MinTurnoverM: 50, MinATRPct: 1.5, PFCap: 10,
		Scanned:      271, Passed: 2, Skipped: 3,
	}
	ranked := []PullbackRow{
		{Ticker: "BEST", Name: "Best Co", TurnoverM: 120.5, DailyATRPct: 2.4, Bars: 20000,
			TradesMed: 18, PFMed: 2.5, Plateau: 0.75, PFMedHO: 1.8, TradesMedHO: 4,
			Best: PullbackGrid()[0], BestPF: 4.1, Capped: 2},
		{Ticker: "GOOD", Name: "Good Co", TurnoverM: 80, DailyATRPct: 1.9, Bars: 19000,
			TradesMed: 12, PFMed: 1.2, Plateau: 0.2, PFMedHO: 0.4, TradesMedHO: 2,
			Best: PullbackGrid()[1], BestPF: 2.0},
	}
	noSignals := []PullbackRow{{Ticker: "QUIET", Name: "Quiet Co"}}

	md := RenderPullbackScreenMarkdown(ranked, noSignals, meta)

	for _, want := range []string{
		"# RSI pullback screener",
		"2026-02-03",   // the train/holdout split date
		"scanned=271",  // universe accounting
		"BEST", "GOOD", // ranking rows
		"QUIET",                    // no-signal section
		"Распределение PFmed",      // the universe backdrop
		"кандидаты на калибровку",  // the disclaimer must be in the report itself
	} {
		if !strings.Contains(md, want) {
			t.Fatalf("report is missing %q\n---\n%s", want, md)
		}
	}
}

func TestRenderPullbackScreenMarkdownOmitsEmptyNoSignalSection(t *testing.T) {
	meta := ScreenMeta{Months: 36, HoldoutMonths: 6, TopN: 50, Split: time.Now(), PFCap: 10}
	md := RenderPullbackScreenMarkdown([]PullbackRow{{Ticker: "ONLY", PFMed: 1.1}}, nil, meta)
	if strings.Contains(md, "Нет сигналов") {
		t.Fatal("empty no-signal section must be omitted entirely")
	}
}

func TestRenderPullbackScreenMarkdownRespectsTopN(t *testing.T) {
	meta := ScreenMeta{Months: 36, HoldoutMonths: 6, TopN: 1, Split: time.Now(), PFCap: 10}
	ranked := []PullbackRow{{Ticker: "FIRST", PFMed: 2}, {Ticker: "SECOND", PFMed: 1}}
	md := RenderPullbackScreenMarkdown(ranked, nil, meta)
	if !strings.Contains(md, "FIRST") || strings.Contains(md, "SECOND") {
		t.Fatal("TopN=1 must render exactly one ranking row")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/service/backtest/ -run 'TestFilterAndRank|TestDistribution|TestRenderPullbackScreen' -v`
Expected: FAIL — `undefined: FilterAndRank`, `undefined: Distribution`, `undefined: RenderPullbackScreenMarkdown`, `undefined: ScreenMeta`.

- [ ] **Step 3: Write minimal implementation**

Создать `internal/service/backtest/pullback_screen_report.go`:

```go
package backtest

import (
	"fmt"
	"sort"
	"strings"
	"time"
)

// ScreenMeta carries the run context shown in the report header.
type ScreenMeta struct {
	Months        int
	HoldoutMonths int
	TopN          int
	Split         time.Time // train/holdout boundary
	MinTurnoverM  float64
	MinATRPct     float64
	PFCap         float64
	Scanned       int // universe size after the currency/trading filter
	Passed        int // rows that cleared both gates
	Skipped       int // tickers whose candles failed to load
}

// FilterAndRank applies the two hard gates and splits the survivors into the ranking
// (sorted by median profit factor, descending) and the no-signal bucket. Minimum
// history and minimum trade count are deliberately NOT gates — they are columns the
// reader judges; only liquidity and daily ATR% exclude a ticker.
func FilterAndRank(rows []PullbackRow, minTurnoverM, minATRPct float64) (ranked, noSignals, rejected []PullbackRow) {
	for _, r := range rows {
		if r.TurnoverM < minTurnoverM || r.DailyATRPct < minATRPct {
			rejected = append(rejected, r)
			continue
		}
		if r.NoSignals {
			noSignals = append(noSignals, r)
			continue
		}
		ranked = append(ranked, r)
	}
	sort.SliceStable(ranked, func(i, j int) bool { return ranked[i].PFMed > ranked[j].PFMed })
	sort.SliceStable(noSignals, func(i, j int) bool { return noSignals[i].Ticker < noSignals[j].Ticker })
	return ranked, noSignals, rejected
}

// PFDist is the spread of median profit factor across the ranked universe. Without
// it the top of the table is unreadable: 271 tickers times 24 configurations is 6504
// trials, so some rows are lucky by construction. If half the universe clears 1.5,
// the bar means nothing — and that must be visible in the report, not discovered a
// month of calibrations later.
type PFDist struct {
	Min, Q1, Median, Q3, Max float64
	ShareAbove15             float64 // share of ranked tickers with PFMed >= 1.5
	N                        int
}

// Distribution summarizes PFMed across the ranked rows.
func Distribution(ranked []PullbackRow) PFDist {
	if len(ranked) == 0 {
		return PFDist{}
	}
	vals := make([]float64, 0, len(ranked))
	var above int
	for _, r := range ranked {
		vals = append(vals, r.PFMed)
		if r.PFMed >= 1.5 {
			above++
		}
	}
	sort.Float64s(vals)
	return PFDist{
		Min:          vals[0],
		Q1:           medianF(vals[:len(vals)/2]),
		Median:       medianF(vals),
		Q3:           medianF(vals[(len(vals)+1)/2:]),
		Max:          vals[len(vals)-1],
		ShareAbove15: float64(above) / float64(len(ranked)),
		N:            len(ranked),
	}
}

// RenderPullbackScreenMarkdown renders the screening report.
func RenderPullbackScreenMarkdown(ranked, noSignals []PullbackRow, meta ScreenMeta) string {
	var b strings.Builder
	d := Distribution(ranked)

	b.WriteString("# RSI pullback screener\n\n")
	fmt.Fprintf(&b, "Окно: %d мес., holdout: последние %d мес. (срез %s).\n",
		meta.Months, meta.HoldoutMonths, meta.Split.Format("2006-01-02"))
	fmt.Fprintf(&b, "Сетка: %d конфигураций (RSIPeriod x RSILower x EMASlow x TPDailyATR), объёмный гейт и трейл выключены.\n",
		len(PullbackGrid()))
	fmt.Fprintf(&b, "Гейты: оборот >= %.0f млн ₽/день, дневной ATR >= %.2f%%. PF зажат сверху на %.1f.\n",
		meta.MinTurnoverM, meta.MinATRPct, meta.PFCap)
	fmt.Fprintf(&b, "Вселенная: scanned=%d passed=%d no-signal=%d skipped=%d.\n\n",
		meta.Scanned, meta.Passed, len(noSignals), meta.Skipped)

	b.WriteString("## Распределение PFmed по прошедшей вселенной\n\n")
	fmt.Fprintf(&b, "min %.2f · Q1 %.2f · медиана %.2f · Q3 %.2f · max %.2f · доля PFmed >= 1.5: %.0f%% (n=%d)\n\n",
		d.Min, d.Q1, d.Median, d.Q3, d.Max, d.ShareAbove15*100, d.N)
	b.WriteString("Читать эту строку раньше первой строки топа: если планку проходит половина вселенной, планка ничего не значит.\n\n")

	b.WriteString("## Рейтинг\n\n")
	b.WriteString("| # | Ticker | Name | Оборот, млн | ATR% дн | Бары | TradesMed | PFmed | Plateau | PFmed HO | Trades HO | Лучшая конфигурация |\n")
	b.WriteString("|---|---|---|---|---|---|---|---|---|---|---|---|\n")
	limit := len(ranked)
	if meta.TopN > 0 && meta.TopN < limit {
		limit = meta.TopN
	}
	for i, r := range ranked[:limit] {
		best := fmt.Sprintf("RSI %d/%.0f, EMA %d/%d, TP %.1f",
			r.Best.RSIPeriod, r.Best.RSILower, r.Best.EMAFast, r.Best.EMASlow, r.Best.TPDailyATR)
		fmt.Fprintf(&b, "| %d | %s | %s | %.0f | %.2f | %d | %.0f | %.2f | %.0f%% | %.2f | %.0f | %s |\n",
			i+1, r.Ticker, r.Name, r.TurnoverM, r.DailyATRPct, r.Bars,
			r.TradesMed, r.PFMed, r.Plateau*100, r.PFMedHO, r.TradesMedHO, best)
	}
	b.WriteString("\nКолонка «лучшая конфигурация» — справочная стартовая точка для ручной калибровки, а не рекомендация.\n")
	b.WriteString("`PFmed HO` в сортировке не участвует: это красный флаг («работало и развалилось»), а не критерий отбора.\n\n")

	if len(noSignals) > 0 {
		b.WriteString("## Нет сигналов\n\n")
		b.WriteString("Тикеры, прошедшие гейты, но не давшие ни одной сделки ни в одной из конфигураций — profit factor у них не существует:\n\n")
		names := make([]string, 0, len(noSignals))
		for _, r := range noSignals {
			names = append(names, r.Ticker)
		}
		fmt.Fprintf(&b, "%s\n\n", strings.Join(names, ", "))
	}

	b.WriteString("## Как это читать\n\n")
	b.WriteString("Шортлист — это **кандидаты на калибровку**, а не доказательство edge. ")
	b.WriteString("Верх такого рейтинга по построению содержит везунчиков: испытаний столько же, сколько тикеров умножить на конфигурации. ")
	b.WriteString("Планка приёмки не меняется — pooled OOS profit factor >= 1.5 в персональном walk-forward (docs/rsi_pullback/strategy.md, §8).\n")
	return b.String()
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/service/backtest/ -run 'TestFilterAndRank|TestDistribution|TestRenderPullbackScreen' -v`
Expected: PASS (7 тестов).

- [ ] **Step 5: Commit**

```bash
git add internal/service/backtest/pullback_screen_report.go internal/service/backtest/pullback_screen_report_test.go
git commit -m "feat(pullscreen): гейты, распределение PFmed по вселенной и markdown-отчёт"
```

---

### Task 6: CLI `cmd/pullscreen` и документация

**Files:**
- Create: `cmd/pullscreen/main.go`
- Create: `docs/rsi_pullback/screener.md`
- Modify: `docs/rsi_pullback/strategy.md` (ссылка на скринер в §1)
- Modify: `CLAUDE.md` (упоминание команды в описании `rsi_pullback`)

**Interfaces:**
- Consumes: `PullbackGrid`, `ScreenTicker`, `DefaultScreenOpts`, `FilterAndRank`, `RenderPullbackScreenMarkdown`, `ScreenMeta` (Tasks 1–5); `grpcclient.NewClientGrpc(apiAddress, token)`, `client.InstrumentsServiceClient().Shares(ctx)`, `svc.NewCandleProvider(client.MarketDataServiceClient(), cacheDir)` и `provider.Load(ctx, ticker, id, interval, from, to, refresh)` — образец в `cmd/backtest/main.go:178-190` и `cmd/backtest/main.go:567-641`.
- Produces: команда `go run ./cmd/pullscreen`.

- [ ] **Step 1: Write the CLI**

Создать `cmd/pullscreen/main.go`:

```go
// Command pullscreen ranks the tradable RUB share universe by how well the
// rsi_pullback strategy fits each ticker. Unlike the older -volrank screener it does
// NOT use a cheap entry proxy: it replays the real strategy engine over a fixed
// 24-configuration grid and ranks by the MEDIAN profit factor, because a proxy that
// only measured the entry side failed to predict realized walk-forward results
// (docs/reversion/screener.md). All gRPC/file I/O lives here; the scoring is pure.
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/joho/godotenv"

	"tinvest/internal/enum"
	svc "tinvest/internal/service/backtest"
	grpcclient "tinvest/pkg/client/grpc"
	"tinvest/pkg/logger"
	"tinvest/pkg/semaphore"
)

const (
	apiAddress = "invest-public-api.tinkoff.ru:443"
	cacheDir   = "data/candles"
)

func main() {
	var (
		months        = flag.Int("months", 36, "lookback period in months")
		holdoutMonths = flag.Int("holdout-months", 6, "trailing months held out of the ranking window")
		minTurnoverM  = flag.Float64("min-turnover", 50, "gate: minimum mean daily turnover in millions of RUB")
		minATRPct     = flag.Float64("min-atr-pct", 1.5, "gate: minimum mean weekday daily ATR as a percentage of close")
		topN          = flag.Int("top", 50, "ranking rows in the report (0 = all)")
		workers       = flag.Int("workers", 8, "concurrent tickers")
		pfCap         = flag.Float64("pf-cap", 10, "cap on profit factor for ranking (0 = no cap)")
		cash          = flag.Float64("cash", 100000, "starting mock cash")
		fraction      = flag.Float64("fraction", 1.0, "fraction of cash per entry")
		commission    = flag.Float64("commission", 0.0005, "commission as a fraction of turnover, per side")
		tickersCSV    = flag.String("tickers", "", "comma-separated tickers instead of the full universe (diagnostics/smoke runs)")
		outDir        = flag.String("out", "reports/pullback_screen", "report output directory")
		refresh       = flag.Bool("refresh", false, "force candle refetch (ignore cache)")
	)
	flag.Parse()
	logger.Init()

	opts := svc.DefaultScreenOpts()
	opts.PFCap = *pfCap
	opts.Cash = *cash
	opts.Fraction = *fraction
	opts.Commission = *commission

	if err := run(context.Background(), runCfg{
		months: *months, holdoutMonths: *holdoutMonths, topN: *topN, workers: *workers,
		minTurnoverM: *minTurnoverM, minATRPct: *minATRPct,
		tickers: splitCSV(*tickersCSV), outDir: *outDir, refresh: *refresh, opts: opts,
	}); err != nil {
		log.Fatalf("pullscreen: %v", err)
	}
}

type runCfg struct {
	months, holdoutMonths, topN, workers int
	minTurnoverM, minATRPct              float64
	tickers                              []string
	outDir                               string
	refresh                              bool
	opts                                 svc.ScreenOpts
}

// shareInfo is the per-ticker metadata the worker pool needs.
type shareInfo struct {
	Ticker string
	Name   string
	ID     string
	Lot    int32
}

func run(ctx context.Context, cfg runCfg) error {
	if cfg.holdoutMonths >= cfg.months {
		return fmt.Errorf("-holdout-months (%d) must be smaller than -months (%d)", cfg.holdoutMonths, cfg.months)
	}
	if cfg.workers < 1 {
		return fmt.Errorf("-workers must be at least 1")
	}
	token, err := loadToken()
	if err != nil {
		return err
	}
	client, err := grpcclient.NewClientGrpc(apiAddress, token)
	if err != nil {
		return fmt.Errorf("grpc client: %w", err)
	}

	universe, err := loadUniverse(ctx, client, cfg.tickers)
	if err != nil {
		return err
	}

	to := time.Now()
	from := to.AddDate(0, -cfg.months, 0)
	// A year of lead-in warms the daily ATR, exactly as cmd/backtest does.
	dailyFrom := from.AddDate(-1, 0, 0)
	split := to.AddDate(0, -cfg.holdoutMonths, 0)

	provider := svc.NewCandleProvider(client.MarketDataServiceClient(), cacheDir)
	grid := svc.PullbackGrid()

	sem := semaphore.New(cfg.workers)
	var wg sync.WaitGroup
	var mu sync.Mutex
	var rows []svc.PullbackRow
	var done, skipped int32

	for _, u := range universe {
		wg.Add(1)
		sem.Acquire()
		go func(u shareInfo) {
			defer wg.Done()
			defer sem.Release()

			bars, err := provider.Load(ctx, u.Ticker, u.ID, enum.Minutes30, from, to, cfg.refresh)
			if err != nil {
				atomic.AddInt32(&skipped, 1)
				fmt.Printf("pullscreen %s: skip (load 30m: %v)\n", u.Ticker, err)
				return
			}
			daily, err := provider.Load(ctx, u.Ticker, u.ID, enum.Day1, dailyFrom, to, cfg.refresh)
			if err != nil {
				atomic.AddInt32(&skipped, 1)
				fmt.Printf("pullscreen %s: skip (load daily: %v)\n", u.Ticker, err)
				return
			}

			row := svc.ScreenTicker(u.Ticker, u.Name, bars, daily, u.Lot, grid, split, cfg.opts)
			n := atomic.AddInt32(&done, 1)
			fmt.Printf("pullscreen [%d/%d] %s: PFmed=%.2f trades=%.0f plateau=%.0f%% turnover=%.0fM atr=%.2f%% bars=%d\n",
				n, len(universe), u.Ticker, row.PFMed, row.TradesMed, row.Plateau*100, row.TurnoverM, row.DailyATRPct, row.Bars)

			mu.Lock()
			rows = append(rows, row)
			mu.Unlock()
		}(u)
	}
	wg.Wait()

	ranked, noSignals, rejected := svc.FilterAndRank(rows, cfg.minTurnoverM, cfg.minATRPct)
	meta := svc.ScreenMeta{
		Months: cfg.months, HoldoutMonths: cfg.holdoutMonths, TopN: cfg.topN, Split: split,
		MinTurnoverM: cfg.minTurnoverM, MinATRPct: cfg.minATRPct, PFCap: cfg.opts.PFCap,
		Scanned: len(universe), Passed: len(ranked), Skipped: int(skipped),
	}

	if err := os.MkdirAll(cfg.outDir, 0o755); err != nil {
		return fmt.Errorf("mkdir out dir: %w", err)
	}
	path := filepath.Join(cfg.outDir, fmt.Sprintf("pullback_screen_Minutes30_%s.md", time.Now().Format("20060102_150405")))
	if err := os.WriteFile(path, []byte(svc.RenderPullbackScreenMarkdown(ranked, noSignals, meta)), 0o644); err != nil {
		return fmt.Errorf("write %s: %w", path, err)
	}
	fmt.Printf("pullscreen report: %s (scanned=%d ranked=%d no-signal=%d rejected=%d skipped=%d)\n",
		path, len(universe), len(ranked), len(noSignals), len(rejected), skipped)
	return nil
}

// loadUniverse returns the tradable RUB share universe, or just the requested tickers.
func loadUniverse(ctx context.Context, client grpcclient.GrpcClient, only []string) ([]shareInfo, error) {
	shares, err := client.InstrumentsServiceClient().Shares(ctx)
	if err != nil {
		return nil, fmt.Errorf("load shares: %w", err)
	}
	want := make(map[string]bool, len(only))
	for _, t := range only {
		want[strings.ToUpper(t)] = true
	}
	var universe []shareInfo
	for _, s := range shares {
		if len(want) > 0 {
			if !want[strings.ToUpper(s.Ticker)] {
				continue
			}
		} else if !strings.EqualFold(s.Currency, "rub") || !s.Trading {
			continue
		}
		universe = append(universe, shareInfo{Ticker: s.Ticker, Name: s.Name, ID: s.ID, Lot: s.Lot})
	}
	if len(universe) == 0 {
		return nil, fmt.Errorf("no matching shares found")
	}
	return universe, nil
}

func splitCSV(s string) []string {
	if strings.TrimSpace(s) == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}

func loadToken() (string, error) {
	_ = godotenv.Load("./env/local.env")
	_ = godotenv.Load("./env/token.env")
	token := os.Getenv("T_BANK")
	if token == "" {
		return "", fmt.Errorf("T_BANK is not set (checked env + ./env/local.env, ./env/token.env)")
	}
	return token, nil
}
```

Доменный пакет `internal/domain/backtest` здесь не импортируется намеренно: свечи от
`provider.Load` уходят в `svc.ScreenTicker` транзитом, ни одного явного упоминания типа
не требуется.

- [ ] **Step 2: Build and smoke-run on three tickers**

```bash
go build ./internal/... ./pkg/... ./cmd/...
go run ./cmd/pullscreen -tickers UGLD,GAZP,SBER -months 24 -top 10 -out reports/pullback_screen_smoke
```

Expected: команда печатает по строке на тикер (`PFmed=... trades=...`) и путь к отчёту. Открыть отчёт и убедиться: шапка заполнена, таблица содержит прошедшие гейты тикеры, дисклеймер на месте. Если все три отсеялись гейтами — перезапустить с `-min-turnover 0 -min-atr-pct 0` и проверить, что строки появились (это диагностика, а не исправление гейтов).

- [ ] **Step 3: Write the user documentation**

Создать `docs/rsi_pullback/screener.md` со следующим содержанием (полный текст, не конспект):

```markdown
# Скринер тикеров под rsi_pullback

Код: `cmd/pullscreen/main.go` (ввод-вывод), `internal/service/backtest/pullback_screen.go`
и `pullback_screen_report.go` (чистое ядро).
Дизайн: `docs/superpowers/specs/2026-08-03-rsi-pullback-screener-design.md`.

## Что он делает

Прогоняет **настоящий движок `rsi_pullback`** по каждому тикеру торгуемой RUB-вселенной
MOEX на фиксированной сетке из 24 конфигураций и ранжирует тикеры по медиане profit
factor. Отвечает ровно на один вопрос: **кому стоит потратить фазовую калибровку**.

Это не доказательство edge. Планка приёмки не меняется: pooled OOS profit factor >= 1.5
в персональном walk-forward (`docs/rsi_pullback/strategy.md`, §8).

## Запуск

```bash
# Полная вселенная, 36 месяцев, holdout — последние 6.
go run ./cmd/pullscreen -months 36 -holdout-months 6 -top 50

# Диагностика по конкретным тикерам.
go run ./cmd/pullscreen -tickers UGLD,GAZP,T -months 24 -top 10
```

Отчёт — `reports/pullback_screen/pullback_screen_Minutes30_<stamp>.md`.
Полный прогон вселенной занимает 20–25 минут на 8 воркерах и идёт из локального кэша
`data/candles/`; `-refresh` заставляет перекачать свечи и делает прогон в разы дольше.

## Флаги

| Флаг | Дефолт | Смысл |
|---|---|---|
| `-months` | 36 | глубина окна |
| `-holdout-months` | 6 | хвост окна, не участвующий в ранжировании |
| `-min-turnover` | 50 | гейт: средний дневной оборот, млн ₽ |
| `-min-atr-pct` | 1.5 | гейт: средний дневной ATR, % от цены |
| `-top` | 50 | строк в рейтинге (0 = все) |
| `-workers` | 8 | параллельных тикеров |
| `-pf-cap` | 10 | потолок PF при ранжировании (0 — без потолка) |
| `-tickers` | — | список тикеров вместо всей вселенной |
| `-out` | `reports/pullback_screen` | каталог отчёта |
| `-refresh` | false | перекачать свечи, игнорируя кэш |

## Как читать отчёт

- **`PFmed`** — медиана profit factor по 24 конфигурациям на train-окне. Единственный
  ключ сортировки. Медиана, а не максимум: максимум по 6.5 тыс. испытаний — это лотерея.
- **`Plateau`** — доля конфигураций с PF >= 1.3 при >= 10 сделках. Ширина рабочей зоны:
  отличает «работает полосой» от «работает в одной точке».
- **`TradesMed`** — медиана числа сделок. Минимум сделок сознательно не является гейтом,
  поэтому строку на 4 сделках отсеивает читатель, а не программа.
- **`PFmed HO`** — то же на последних `-holdout-months`. **В сортировке не участвует**:
  на шести месяцах у стратегии с ~14 сделками в год выборка слишком мала, чтобы ей
  ранжировать. Это красный флаг «работало и развалилось», а не критерий.
- **Распределение `PFmed` по вселенной** в шапке — читать раньше первой строки топа. Если
  планку 1.5 проходит половина вселенной, планка ничего не значит.
- **«Лучшая конфигурация»** — справочная стартовая точка для ручной калибровки.

## Что скринер не меряет

- Гейт объёмов и трейлинг в сетке выключены: первый режет и без того редкие сделки,
  второй — свойство настроенной конфигурации, а не инструмента. Их цена меряется на
  этапе калибровки конкретного тикера (`data/params/rsi_pullback/<ticker>/`).
- Проскальзывание. Движок заливает стоп по `min(уровень, open)`, внутрибарные проколы не
  моделируются — как и во всех прочих прогонах репозитория.
- Лот, размер счёта, конфликт с другими стратегиями за капитал (UGLD уже торгуется живым
  `reversion`). Это решается при выводе в прод, не здесь.

## Почему не как `-volrank`

Предыдущий скринер (`docs/reversion/screener.md`) ранжировал по дешёвому прокси входа
и не предсказал ни одного из реализованных walk-forward победителей: прокси меряет вход,
а edge живёт в выходах и экспектанси. Здесь гоняется настоящий движок — это стало по
карману, потому что кэш 30-минуток по всей вселенной уже локальный.
```

- [ ] **Step 4: Link the screener from the strategy docs**

В `docs/rsi_pullback/strategy.md` в конец §1 «Идея» добавить абзац:

```markdown
Подбор тикеров под эту стратегию делает отдельный скринер: `go run ./cmd/pullscreen`,
справочник — `docs/rsi_pullback/screener.md`. Он прогоняет это же ядро по вселенной
RUB-акций на фиксированной сетке из 24 конфигураций и выдаёт шортлист кандидатов на
калибровку; доказательством edge его рейтинг не является.
```

В `CLAUDE.md` в описании `rsi_pullback` (раздел `internal/service` → `trading_strategy/`) после `docs: docs/rsi_pullback/strategy.md` дописать: `; подбор тикеров — cmd/pullscreen, docs/rsi_pullback/screener.md`.

- [ ] **Step 5: Full gate and commit**

```bash
./bin/mage ci
git add cmd/pullscreen/main.go docs/rsi_pullback/screener.md docs/rsi_pullback/strategy.md CLAUDE.md
git commit -m "feat(pullscreen): CLI скринера тикеров под rsi_pullback и документация"
```

Expected: `mage ci` EXIT=0 (lint + `go test -race ./...` + проверка дрейфа моков).

---

## Приёмка всего плана

- `./bin/mage ci` зелёный.
- `go run ./cmd/pullscreen -tickers UGLD,GAZP,T -months 24 -top 10` даёт отчёт с непустым рейтингом.
- Полный прогон `go run ./cmd/pullscreen -months 36` завершается за разумное время и пишет отчёт; шапка содержит распределение `PFmed` по вселенной.
- В отчёте UGLD (одобренный владельцем прод-тикер) присутствует в рейтинге — если он отсеялся гейтами или попал в «нет сигналов», это сигнал о дефекте скринера, а не о UGLD: разобраться до того, как доверять остальным строкам.
