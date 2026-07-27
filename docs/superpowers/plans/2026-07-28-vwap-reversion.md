# VWAP-возврат (`vwap_rev`) — план реализации

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** backtest-only лонговая стратегия `vwap_rev` — внутридневной возврат к сессионному VWAP на 30-минутных барах, с грид-сеткой и документацией, готовая к walk-forward калибровке.

**Architecture:** сессионный VWAP выносится в `pkg/indicators` как чистая функция; правила живут в stateless-ядре `internal/service/trading_strategy/vwap_rev/strategy/core` (клон структуры `rsi_ema`); реестр `VWAPRevLookupOrGeneric` связывает ядро с движком; `cmd/backtest` получает ветку `-strategy vwap_rev`. Движок бэктеста, метрики, walk-forward и отчёты НЕ меняются.

**Tech Stack:** Go 1.25, стандартная библиотека; `pkg/indicators` (ATR), `internal/domain/ema`; тесты — table-driven `testing`; сборка и проверки — `./bin/mage ci`.

**Спека:** `docs/superpowers/specs/2026-07-28-vwap-reversion-design.md`

## Global Constraints

- Комиссия в бэктесте — `0.0005` за сторону (тариф «Трейдер»), 0.1% за круг. Дефолт CLI менять нельзя.
- Ядро **stateless между барами**: всё пересчитывается из `strategy.MarketData`; состояние позиции приходит только в `md.Position`.
- Все поля `Params` — только `int` и `float64`: рефлексивная грид-калибровка (`applyField`) других типов не перебирает.
- Никакого lookahead: решение на баре `i` использует только бары `0..i`, а цель `TP` — VWAP бара `i-1`.
- Комментарии и сообщения в коде — на английском (как в `rsi_ema`); тексты `EntryReason`/`ExitReason`/`Explain` — на русском (их читает владелец в отчётах).
- Часовой пояс всех сессионных правил — `Europe/Moscow` с откатом на UTC.
- Каждая задача завершается зелёным `./bin/mage ci` и коммитом.
- Стратегия backtest-only: в `internal/app`, live-раннеры и Telegram она НЕ подключается.

---

## File Structure

| Файл | Ответственность |
|---|---|
| `pkg/indicators/vwap.go` (создать) | Чистый индикатор `SessionVWAP`: якорь сессии, VWAP, σ, индекс бара внутри сессии. Ничего не знает о торговле. |
| `pkg/indicators/vwap_test.go` (создать) | Тесты индикатора: ручной расчёт, границы сессий, деградация. |
| `internal/service/trading_strategy/vwap_rev/strategy/core/core.go` (создать) | `Params`, `DefaultParams`, `Strategy`, `Decide`/`enter`/`manage`/`Explain`, `Lookback`. Ничего не знает о загрузке свечей. |
| `internal/service/trading_strategy/vwap_rev/strategy/core/core_test.go` (создать) | Table-driven тесты правил входа, выходов, приоритета и отсутствия lookahead. |
| `internal/service/backtest/vwap_rev_registry.go` (создать) | `VWAPRevLookupOrGeneric` — биндинг ядра к движку. |
| `internal/service/backtest/vwap_rev_registry_test.go` (создать) | Тесты биндинга и разбора частичного JSON. |
| `internal/service/backtest/vwap_rev_grid_test.go` (создать) | Тест: каждое имя поля в `grid.json` существует в `Params`, в каждой фазе есть контрольная точка «выключено». |
| `cmd/backtest/main.go` (изменить, ~строки 41, 156-166) | Ветка `case "vwap_rev"` и строки помощи. |
| `data/params/vwap_rev/grid.json` (создать) | Фазовая сетка, 124 комбинации. |
| `docs/vwap_rev/strategy.md` (создать) | Правила, параметры, команды запуска. |

---

### Task 1: Индикатор `SessionVWAP`

**Files:**
- Create: `pkg/indicators/vwap.go`
- Test: `pkg/indicators/vwap_test.go`

**Interfaces:**
- Consumes: ничего (первая задача).
- Produces: `indicators.SessionVWAP(highs, lows, closes []float64, volumes []int64, times []time.Time, loc *time.Location) (vwap, sigma []float64, barsFromOpen []int)`. Все три среза длины `len(closes)`, либо `nil, nil, nil` при невалидном входе.

- [ ] **Step 1: Написать падающий тест ручного расчёта и деградации**

Создать `pkg/indicators/vwap_test.go`:

```go
package indicators

import (
	"math"
	"testing"
	"time"
)

// msk is the calendar the session anchor is defined in.
var msk = time.FixedZone("MSK", 3*60*60)

// bar is one synthetic candle for the VWAP tests.
type bar struct {
	t          time.Time
	h, l, c    float64
	v          int64
}

// split explodes bars into the parallel slices SessionVWAP consumes.
func split(bars []bar) (h, l, c []float64, v []int64, ts []time.Time) {
	for _, b := range bars {
		h = append(h, b.h)
		l = append(l, b.l)
		c = append(c, b.c)
		v = append(v, b.v)
		ts = append(ts, b.t)
	}
	return
}

// at builds an MSK bar time on the given day.
func at(day, hour, min int) time.Time {
	return time.Date(2026, 3, day, hour, min, 0, 0, msk)
}

func TestSessionVWAPHandComputed(t *testing.T) {
	// Typical prices 100, 102, 104 with equal volume -> VWAP 102,
	// sigma = sqrt(((100-102)^2 + 0 + (104-102)^2)/3) = sqrt(8/3).
	bars := []bar{
		{at(2, 10, 0), 101, 99, 100, 100},
		{at(2, 10, 30), 103, 101, 102, 100},
		{at(2, 11, 0), 105, 103, 104, 100},
	}
	h, l, c, v, ts := split(bars)
	vwap, sigma, bfo := SessionVWAP(h, l, c, v, ts, msk)
	if len(vwap) != 3 || len(sigma) != 3 || len(bfo) != 3 {
		t.Fatalf("lengths = %d/%d/%d want 3/3/3", len(vwap), len(sigma), len(bfo))
	}
	if math.Abs(vwap[2]-102) > 1e-9 {
		t.Fatalf("vwap[2] = %v want 102", vwap[2])
	}
	if want := math.Sqrt(8.0 / 3.0); math.Abs(sigma[2]-want) > 1e-9 {
		t.Fatalf("sigma[2] = %v want %v", sigma[2], want)
	}
	if math.Abs(vwap[0]-100) > 1e-9 || sigma[0] != 0 {
		t.Fatalf("single-bar session: vwap=%v sigma=%v want 100/0", vwap[0], sigma[0])
	}
}

func TestSessionVWAPFirstSessionMarkedIncomplete(t *testing.T) {
	bars := []bar{
		{at(2, 10, 0), 101, 99, 100, 100},
		{at(2, 10, 30), 103, 101, 102, 100},
		{at(3, 10, 0), 201, 199, 200, 100},
		{at(3, 10, 30), 203, 201, 202, 100},
	}
	h, l, c, v, ts := split(bars)
	vwap, _, bfo := SessionVWAP(h, l, c, v, ts, msk)
	if bfo[0] != -1 || bfo[1] != -1 {
		t.Fatalf("first session bfo = %v/%v want -1/-1", bfo[0], bfo[1])
	}
	if bfo[2] != 0 || bfo[3] != 1 {
		t.Fatalf("second session bfo = %v/%v want 0/1", bfo[2], bfo[3])
	}
	// The anchor resets: day 2 must not be dragged by day 1's prices.
	if math.Abs(vwap[3]-201) > 1e-9 {
		t.Fatalf("vwap[3] = %v want 201 (day-2 anchor)", vwap[3])
	}
}

func TestSessionVWAPWeekendGapSplitsSessions(t *testing.T) {
	// 2026-03-06 is a Friday, 2026-03-09 the next Monday.
	bars := []bar{
		{time.Date(2026, 3, 5, 10, 0, 0, 0, msk), 101, 99, 100, 100},
		{time.Date(2026, 3, 6, 10, 0, 0, 0, msk), 101, 99, 100, 100},
		{time.Date(2026, 3, 9, 10, 0, 0, 0, msk), 301, 299, 300, 100},
		{time.Date(2026, 3, 9, 10, 30, 0, 0, msk), 301, 299, 300, 100},
	}
	h, l, c, v, ts := split(bars)
	vwap, _, bfo := SessionVWAP(h, l, c, v, ts, msk)
	if bfo[1] != 0 {
		t.Fatalf("Friday bfo = %d want 0 (own session)", bfo[1])
	}
	if bfo[2] != 0 || bfo[3] != 1 {
		t.Fatalf("Monday bfo = %d/%d want 0/1", bfo[2], bfo[3])
	}
	if math.Abs(vwap[3]-300) > 1e-9 {
		t.Fatalf("vwap[3] = %v want 300 (Monday anchor)", vwap[3])
	}
}

func TestSessionVWAPZeroVolumeSession(t *testing.T) {
	bars := []bar{
		{at(2, 10, 0), 101, 99, 100, 100},
		{at(3, 10, 0), 201, 199, 200, 0},
		{at(3, 10, 30), 203, 201, 202, 0},
	}
	h, l, c, v, ts := split(bars)
	vwap, sigma, _ := SessionVWAP(h, l, c, v, ts, msk)
	if vwap[2] != 0 || sigma[2] != 0 {
		t.Fatalf("zero-volume session: vwap=%v sigma=%v want 0/0", vwap[2], sigma[2])
	}
	if math.IsNaN(vwap[2]) || math.IsNaN(sigma[2]) {
		t.Fatalf("zero-volume session produced NaN")
	}
}

func TestSessionVWAPRejectsMisalignedInput(t *testing.T) {
	bars := []bar{
		{at(2, 10, 0), 101, 99, 100, 100},
		{at(2, 10, 30), 103, 101, 102, 100},
	}
	h, l, c, v, ts := split(bars)
	cases := map[string]func() (vwap, sigma []float64, bfo []int){
		"no times":      func() ([]float64, []float64, []int) { return SessionVWAP(h, l, c, v, nil, msk) },
		"short times":   func() ([]float64, []float64, []int) { return SessionVWAP(h, l, c, v, ts[:1], msk) },
		"short highs":   func() ([]float64, []float64, []int) { return SessionVWAP(h[:1], l, c, v, ts, msk) },
		"short volumes": func() ([]float64, []float64, []int) { return SessionVWAP(h, l, c, v[:1], ts, msk) },
		"empty":         func() ([]float64, []float64, []int) { return SessionVWAP(nil, nil, nil, nil, nil, msk) },
	}
	for name, call := range cases {
		t.Run(name, func(t *testing.T) {
			vwap, sigma, bfo := call()
			if vwap != nil || sigma != nil || bfo != nil {
				t.Fatalf("want nil,nil,nil; got %v,%v,%v", vwap, sigma, bfo)
			}
		})
	}
}

func TestSessionVWAPNilLocationDoesNotPanic(t *testing.T) {
	bars := []bar{{at(2, 10, 0), 101, 99, 100, 100}}
	h, l, c, v, ts := split(bars)
	if vwap, _, _ := SessionVWAP(h, l, c, v, ts, nil); len(vwap) != 1 {
		t.Fatalf("nil loc must degrade to UTC, got len=%d", len(vwap))
	}
}
```

- [ ] **Step 2: Прогнать тест и убедиться, что он падает**

Run: `go test ./pkg/indicators/ -run TestSessionVWAP -v`
Expected: FAIL — `undefined: SessionVWAP`

- [ ] **Step 3: Реализовать индикатор**

Создать `pkg/indicators/vwap.go`:

```go
package indicators

import (
	"math"
	"time"
)

// SessionVWAP returns the session-anchored VWAP, the volume-weighted standard deviation of
// typical price ((H+L+C)/3) around it, and each bar's zero-based index within its session —
// all index-aligned to the input bars.
//
// Sessions are delimited by a change of calendar date in loc, so weekends and holidays split
// sessions without a separate rule. On bar i the sums run from that session's first bar
// through i inclusive.
//
// The FIRST session present in the window is always reported as incomplete (barsFromOpen = -1
// on every one of its bars), even when the window happens to start exactly at a session open.
// A rolling window usually starts mid-day, where the anchor would be arbitrary; keeping the
// rule unconditional makes every returned value independent of where the window was cut,
// which is what lets callers stay free of look-ahead.
//
// Returns nil, nil, nil when the inputs are empty or not index-aligned. Bars with
// non-positive volume contribute zero weight; a session whose total volume is zero keeps
// vwap = 0 and sigma = 0 rather than producing NaN. A nil loc degrades to UTC.
func SessionVWAP(highs, lows, closes []float64, volumes []int64,
	times []time.Time, loc *time.Location) (vwap, sigma []float64, barsFromOpen []int) {
	n := len(closes)
	if n == 0 || len(highs) != n || len(lows) != n || len(volumes) != n || len(times) != n {
		return nil, nil, nil
	}
	if loc == nil {
		loc = time.UTC
	}

	vwap = make([]float64, n)
	sigma = make([]float64, n)
	barsFromOpen = make([]int, n)

	// West's weighted incremental mean/variance: numerically stable for prices in the
	// hundreds carrying a variance in the fractions.
	var sumV, mean, m2 float64
	idx := 0
	firstSession := true
	var py, pd int
	var pm time.Month

	for i := 0; i < n; i++ {
		y, m, d := times[i].In(loc).Date()
		switch {
		case i == 0:
			// first bar opens the first (incomplete-by-rule) session
		case y != py || m != pm || d != pd:
			firstSession = false
			sumV, mean, m2 = 0, 0, 0
			idx = 0
		default:
			idx++
		}
		py, pm, pd = y, m, d

		if v := float64(volumes[i]); v > 0 {
			tp := (highs[i] + lows[i] + closes[i]) / 3
			sumV += v
			delta := tp - mean
			mean += delta * v / sumV
			m2 += v * delta * (tp - mean)
		}
		if sumV > 0 {
			vwap[i] = mean
			if variance := m2 / sumV; variance > 0 {
				sigma[i] = math.Sqrt(variance)
			}
		}
		if firstSession {
			barsFromOpen[i] = -1
		} else {
			barsFromOpen[i] = idx
		}
	}
	return vwap, sigma, barsFromOpen
}
```

- [ ] **Step 4: Прогнать тесты**

Run: `go test ./pkg/indicators/ -run TestSessionVWAP -v`
Expected: PASS, все шесть тестов зелёные

- [ ] **Step 5: Полный гейт**

Run: `./bin/mage ci`
Expected: PASS

- [ ] **Step 6: Коммит**

```bash
git add pkg/indicators/vwap.go pkg/indicators/vwap_test.go
git commit -m "feat(indicators): сессионный VWAP с объёмно-взвешенной сигмой"
```

---

### Task 2: Ядро — параметры и правила входа

**Files:**
- Create: `internal/service/trading_strategy/vwap_rev/strategy/core/core.go`
- Test: `internal/service/trading_strategy/vwap_rev/strategy/core/core_test.go`

**Interfaces:**
- Consumes: `indicators.SessionVWAP` (Task 1); `indicators.ATR(highs, lows, closes []float64, period int) float64`; `ema.Compute(closes []float64, period int) []float64`; `strategy.MarketData`, `strategy.Position`, `model.Signal`, `model.SignalBuy`.
- Produces: `core.Params` (поля перечислены ниже), `core.DefaultParams() Params`, `core.NewWithParams(ticker string, p Params) *Strategy`, `(*Strategy).Ticker() string`, `(*Strategy).Lookback() int`, `(*Strategy).Decide(md strategy.MarketData) model.Signal`.

- [ ] **Step 1: Написать падающие тесты входа**

Создать `internal/service/trading_strategy/vwap_rev/strategy/core/core_test.go`:

```go
package core

import (
	"testing"
	"time"

	"tinvest/internal/service/trading_strategy/scalping/model"
	"tinvest/internal/service/trading_strategy/scalping/strategy"
)

var msk = time.FixedZone("MSK", 3*60*60)

// series builds MarketData for bars laid out at a fixed 30-minute span starting at the given
// MSK day/hour. Volumes are uniform so VWAP is the plain mean of typical prices.
type barSpec struct {
	h, l, c float64
	v       int64
}

// buildMD lays bars out at 30-minute steps from 2026-03-day hour:min MSK and returns the
// MarketData the strategy sees on the LAST bar.
func buildMD(day, hour, min int, bars []barSpec, dailyCloses []float64) strategy.MarketData {
	md := strategy.MarketData{DailyCloses: dailyCloses}
	start := time.Date(2026, 3, day, hour, min, 0, 0, msk)
	for i, b := range bars {
		md.Highs = append(md.Highs, b.h)
		md.Lows = append(md.Lows, b.l)
		md.Closes = append(md.Closes, b.c)
		md.Volumes = append(md.Volumes, b.v)
		md.Times = append(md.Times, start.Add(time.Duration(i)*30*time.Minute))
	}
	md.Price = md.Closes[len(md.Closes)-1]
	return md
}

// flatDay returns n bars whose typical price is exactly p, each with volume 1000. The bars
// carry a small high/low range on purpose: a zero-range series would drive ATR to 0, and the
// entry rejects a non-positive ATR whenever the stop is armed.
func flatDay(n int, p float64) []barSpec {
	out := make([]barSpec, n)
	for i := range out {
		out[i] = barSpec{h: p + 0.15, l: p - 0.15, c: p, v: 1000}
	}
	return out
}

// setupEntry builds a window whose LAST bar is a textbook entry: a previous session, then a
// session trading at 100 that finally dips to 99.5 with the close in the upper half of the
// bar. Daily closes trend up so the daily gate passes.
//
// The resulting numbers on the last bar: VWAP ≈ 99.947, σ ≈ 0.16, deviation ≈ 0.447
// (≈ 2.8×σ, ≈ 0.449% of price) — clear of every default threshold with margin, so a single
// tweak in the table-driven test isolates exactly one gate.
func setupEntry() strategy.MarketData {
	bars := flatDay(10, 100) // previous session (window's first -> barsFromOpen -1)
	// Current session starts the next calendar day.
	cur := flatDay(9, 100)
	cur = append(cur, barSpec{h: 99.7, l: 99.2, c: 99.5, v: 1000}) // the dip bar
	all := append(bars, cur...)
	daily := make([]float64, 60)
	for i := range daily {
		daily[i] = 50 + float64(i) // strongly rising -> EMA far below 99.5
	}
	md := buildMD(2, 10, 0, all, daily)
	// Re-stamp the current session onto the next day starting at 07:00 MSK.
	base := time.Date(2026, 3, 3, 7, 0, 0, 0, msk)
	for i := range cur {
		md.Times[len(bars)+i] = base.Add(time.Duration(i) * 30 * time.Minute)
	}
	return md
}

func decide(t *testing.T, p Params, md strategy.MarketData) model.Signal {
	t.Helper()
	return NewWithParams("TEST", p).Decide(md)
}

func TestEnterFiresOnTextbookSetup(t *testing.T) {
	sig := decide(t, DefaultParams(), setupEntry())
	if sig.Kind != model.SignalBuy {
		t.Fatalf("Kind = %v want SignalBuy", sig.Kind)
	}
	if sig.StopLoss <= 0 || sig.StopLoss >= md0Close {
		t.Fatalf("StopLoss = %v must sit below the entry price", sig.StopLoss)
	}
}

// md0Close is the close of the dip bar built by setupEntry.
const md0Close = 99.5

func TestEnterGates(t *testing.T) {
	tests := []struct {
		name  string
		tweak func(p *Params)
		want  model.SignalKind
	}{
		{"baseline fires", func(p *Params) {}, model.SignalBuy},
		{"deviation too shallow", func(p *Params) { p.EntryK = 50 }, model.SignalNone},
		{"deviation deeper than MaxDevK", func(p *Params) { p.MaxDevK = 0.01 }, model.SignalNone},
		{"MaxDevK<=0 disables the cap", func(p *Params) { p.MaxDevK = 0 }, model.SignalBuy},
		{"MinEdgePct not met", func(p *Params) { p.MinEdgePct = 5 }, model.SignalNone},
		{"MinEdgePct=0 lets it through", func(p *Params) { p.MinEdgePct = 0 }, model.SignalBuy},
		{"too early in the session", func(p *Params) { p.MinBarsFromOpen = 50 }, model.SignalNone},
		{"close in the lower third", func(p *Params) { p.MinClosePos = 0.99 }, model.SignalNone},
		{"MinClosePos<=0 disables the filter", func(p *Params) { p.MinClosePos = 0 }, model.SignalBuy},
		{"daily trend gate off", func(p *Params) { p.UseDailyTrend = 0 }, model.SignalBuy},
		{"entry cutoff already passed", func(p *Params) { p.SessionEndMin = 421 }, model.SignalNone},
		{"stop disabled is still a valid entry", func(p *Params) { p.StopATR = 0 }, model.SignalBuy},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			p := DefaultParams()
			tc.tweak(&p)
			if got := decide(t, p, setupEntry()).Kind; got != tc.want {
				t.Fatalf("Kind = %v want %v", got, tc.want)
			}
		})
	}
}

func TestEnterRejectedWhenPriceBelowDailyEMA(t *testing.T) {
	md := setupEntry()
	// Falling daily series: EMA ends far ABOVE the intraday price.
	for i := range md.DailyCloses {
		md.DailyCloses[i] = 500 - float64(i)
	}
	if got := decide(t, DefaultParams(), md).Kind; got != model.SignalNone {
		t.Fatalf("Kind = %v want SignalNone below the daily EMA", got)
	}
}

// The daily trend gate is the one gate in this strategy that FAILS CLOSED: without a usable
// daily series there is no way to tell a pullback from a collapse.
func TestEnterRejectedWhenDailySeriesMissing(t *testing.T) {
	for _, name := range []string{"nil", "too short"} {
		t.Run(name, func(t *testing.T) {
			md := setupEntry()
			if name == "nil" {
				md.DailyCloses = nil
			} else {
				md.DailyCloses = md.DailyCloses[:3]
			}
			if got := decide(t, DefaultParams(), md).Kind; got != model.SignalNone {
				t.Fatalf("Kind = %v want SignalNone", got)
			}
		})
	}
}

func TestEnterRejectedOnFirstSessionOfWindow(t *testing.T) {
	// A window holding a single session: its bars carry barsFromOpen = -1 by the indicator's
	// rule, so no entry may fire however good the setup looks.
	cur := flatDay(9, 100)
	cur = append(cur, barSpec{h: 99.7, l: 99.2, c: 99.5, v: 1000})
	daily := make([]float64, 60)
	for i := range daily {
		daily[i] = 50 + float64(i)
	}
	md := buildMD(3, 7, 0, cur, daily)
	p := DefaultParams()
	p.MinBarsFromOpen = 0
	if got := decide(t, p, md).Kind; got != model.SignalNone {
		t.Fatalf("Kind = %v want SignalNone on the window's first session", got)
	}
}

func TestLookbackCoversAFullSession(t *testing.T) {
	if got := NewWithParams("T", DefaultParams()).Lookback(); got < 300 {
		t.Fatalf("Lookback = %d want >= 300", got)
	}
}
```

- [ ] **Step 2: Прогнать тест и убедиться, что он падает**

Run: `go test ./internal/service/trading_strategy/vwap_rev/... -v`
Expected: FAIL — пакет не существует

- [ ] **Step 3: Реализовать параметры, сессионные хелперы и вход**

Создать `internal/service/trading_strategy/vwap_rev/strategy/core/core.go`:

```go
// Package core implements a long-only intraday mean-reversion strategy anchored to the
// session VWAP. When flat it buys a close that has fallen EntryK sigmas below the running
// session VWAP — provided the move is large enough in percent to clear round-trip costs, the
// bar did not close on its low, and the daily trend is up. The target is the VWAP itself. The
// decision logic is pure, stateless between bars and ticker-agnostic. The reference timeframe
// is 30 minutes. Run with `-strategy vwap_rev -interval Minutes30`.
package core

import (
	"fmt"
	"sort"
	"time"

	"tinvest/internal/domain/ema"
	"tinvest/internal/service/trading_strategy/scalping/model"
	"tinvest/internal/service/trading_strategy/scalping/strategy"
	"tinvest/pkg/indicators"
)

// defaultBarSpanMin is the bar length in minutes assumed when the series carries no usable
// open-times. It matches this strategy's 30-minute reference timeframe; the real span is read
// from the data via barSpanMinutes whenever Times is present.
const defaultBarSpanMin = 30

// minLookback is the floor on the candle window handed to Decide. The session VWAP anchor is
// only correct when the current day's opening bar sits inside the window, and the indicator
// discards the window's first session outright, so the window must cover at least two full
// sessions. 300 bars clears that on 15-minute bars and coarser; this strategy is not meant for
// anything finer.
const minLookback = 300

// Params holds every tunable. All fields are int or float64 so reflection grid calibration
// can sweep them.
type Params struct {
	EntryK          float64 // entry requires Close <= VWAP - EntryK*sigma (grid; default 1.5)
	MaxDevK         float64 // reject when Close < VWAP - MaxDevK*sigma; <=0 disables (grid; default 4)
	MinEdgePct      float64 // entry requires (VWAP-Close)/Close*100 >= this, IN PERCENT (grid; default 0.35)
	MinBarsFromOpen int     // entry requires barsFromOpen >= this (grid; default 6)
	MinClosePos     float64 // entry requires Close >= Low + MinClosePos*(High-Low), a 0..1 fraction; <=0 disables (grid; default 0.33)
	UseDailyTrend   int     // 1 arms the daily EMA trend gate; any other value disables it (grid; default 1)
	DailyEMAPeriod  int     // daily EMA period for the trend gate (grid; default 50)
	StopATR         float64 // stop = entry - StopATR*ATR; 0 disables the stop (grid; default 1.2)
	ATRPeriod       int     // ATR length; used only when StopATR>0
	MaxHoldBars     int     // TIME exit after this many bars held; <=0 disables (grid; default 8)
	SessionStartMin int     // entry window start, minutes from MSK midnight (420 = 07:00)
	SessionEndMin   int     // Mon-Thu entry cutoff, minutes from MSK midnight (1080 = 18:00)
	FridayEndMin    int     // Friday entry cutoff, minutes from MSK midnight (840 = 14:00)
	DayEndMin       int     // day-end force-close boundary, minutes from MSK midnight (1380 = 23:00)
}

// DefaultParams returns the spec's baseline; swept values come from calibration.
func DefaultParams() Params {
	return Params{
		EntryK:          1.5,
		MaxDevK:         4.0,
		MinEdgePct:      0.35,
		MinBarsFromOpen: 6,
		MinClosePos:     0.33,
		UseDailyTrend:   1,
		DailyEMAPeriod:  50,
		StopATR:         1.2,
		ATRPeriod:       14,
		MaxHoldBars:     8,
		SessionStartMin: 420,
		SessionEndMin:   1080,
		FridayEndMin:    840,
		DayEndMin:       1380,
	}
}

// Strategy trades a single instrument with the VWAP-reversion rules. Ticker-agnostic and pure.
type Strategy struct {
	ticker string
	p      Params
}

// NewWithParams returns the strategy for a ticker with explicit params.
func NewWithParams(ticker string, p Params) *Strategy { return &Strategy{ticker: ticker, p: p} }

func (s *Strategy) Ticker() string { return s.ticker }

// Lookback sizes the candle window the engine feeds Decide on every bar. See minLookback for
// why two full sessions are the binding requirement; the ATR term only matters if someone
// calibrates an unusually long ATR.
func (s *Strategy) Lookback() int {
	return max(minLookback, 2*s.p.ATRPeriod+20)
}

// mskLoc anchors the session windows and the VWAP anchor to the Moscow calendar (UTC fallback).
var mskLoc = func() *time.Location {
	loc, err := time.LoadLocation("Europe/Moscow")
	if err != nil {
		return time.UTC
	}
	return loc
}()

// sessionEndMin returns the entry cutoff for the weekday of tl (Friday closes early).
func (s *Strategy) sessionEndMin(tl time.Time) int {
	if tl.Weekday() == time.Friday {
		return s.p.FridayEndMin
	}
	return s.p.SessionEndMin
}

// isWeekend reports whether tl (already in MSK) falls on a non-trading day.
func isWeekend(tl time.Time) bool {
	wd := tl.Weekday()
	return wd == time.Saturday || wd == time.Sunday
}

// inSession reports whether bar-time t falls inside the entry window in MSK. A zero time
// skips the gate — never block on missing data.
func (s *Strategy) inSession(t time.Time) bool {
	if t.IsZero() {
		return true
	}
	tl := t.In(mskLoc)
	if isWeekend(tl) {
		return false
	}
	m := tl.Hour()*60 + tl.Minute()
	return m >= s.p.SessionStartMin && m < s.sessionEndMin(tl)
}

// isDayEnd reports whether the bar opening at t, spanning spanMin minutes, is the last one
// before the day-end force-close boundary. A zero time degrades the EOD exit to a no-op.
func (s *Strategy) isDayEnd(t time.Time, spanMin int) bool {
	if t.IsZero() {
		return false
	}
	tl := t.In(mskLoc)
	if isWeekend(tl) {
		return true
	}
	m := tl.Hour()*60 + tl.Minute()
	return m+spanMin >= s.p.DayEndMin
}

// barSpanMinutes infers the bar length from the series' own open-times: the MEDIAN gap
// between consecutive bars (robust to session and weekend jumps).
func barSpanMinutes(times []time.Time) int {
	if len(times) < 2 {
		return defaultBarSpanMin
	}
	gaps := make([]int, 0, len(times)-1)
	for i := 1; i < len(times); i++ {
		if d := int(times[i].Sub(times[i-1]) / time.Minute); d > 0 {
			gaps = append(gaps, d)
		}
	}
	if len(gaps) == 0 {
		return defaultBarSpanMin
	}
	sort.Ints(gaps)
	return gaps[len(gaps)/2]
}

// barTime returns the open-time of the latest bar, or the zero time when Times is absent or
// misaligned with Closes (so time-based gates degrade instead of misfiring).
func (s *Strategy) barTime(md strategy.MarketData) time.Time {
	n := len(md.Closes)
	if n == 0 || len(md.Times) != n {
		return time.Time{}
	}
	return md.Times[n-1]
}

// sessionVWAP recomputes the session anchor from md. Returns ok=false whenever the anchor
// cannot be trusted at the last bar.
func (s *Strategy) sessionVWAP(md strategy.MarketData) (vwap, sigma []float64, bfo []int, ok bool) {
	n := len(md.Closes)
	vwap, sigma, bfo = indicators.SessionVWAP(md.Highs, md.Lows, md.Closes, md.Volumes, md.Times, mskLoc)
	if vwap == nil || len(vwap) != n {
		return nil, nil, nil, false
	}
	return vwap, sigma, bfo, true
}

// closePosOK reports whether the bar closed clear of its own low — a falling-knife filter. A
// zero-range bar carries no position information and is allowed through. Off when
// MinClosePos<=0.
func (s *Strategy) closePosOK(high, low, closeP float64) bool {
	if s.p.MinClosePos <= 0 {
		return true
	}
	rng := high - low
	if rng <= 0 {
		return true
	}
	return closeP >= low+s.p.MinClosePos*rng
}

// dailyTrendOK reports whether the daily trend allows the entry. Unlike most gates in this
// repo it FAILS CLOSED: with the gate armed and no usable daily series, the entry is rejected
// rather than allowed. Buying a deep intraday dip without knowing the daily trend is exactly
// the trade this strategy must not take.
func (s *Strategy) dailyTrendOK(dailyCloses []float64, price float64) bool {
	if s.p.UseDailyTrend != 1 {
		return true
	}
	if s.p.DailyEMAPeriod <= 0 || len(dailyCloses) < s.p.DailyEMAPeriod {
		return false
	}
	e := ema.Compute(dailyCloses, s.p.DailyEMAPeriod)
	if len(e) == 0 {
		return false
	}
	last := e[len(e)-1]
	return last > 0 && price > last
}

// Decide routes to entry (flat) or position management (open).
func (s *Strategy) Decide(md strategy.MarketData) model.Signal {
	sig := model.Signal{Ticker: s.ticker, Price: md.Price}
	if md.Position != nil {
		return s.manage(md, sig)
	}
	return s.enter(md, sig)
}

// enter emits a long when the close sits deep enough below the session VWAP, the move is
// worth more than the round-trip cost, the bar did not close on its low and the daily trend
// is up. Everything is recomputed from md — no state survives between bars.
func (s *Strategy) enter(md strategy.MarketData, sig model.Signal) model.Signal {
	n := len(md.Closes)
	if n < 2 || len(md.Highs) != n || len(md.Lows) != n {
		return sig
	}
	// 1. entry window, and never on the day-end bar (manage() only runs from the NEXT bar, so
	// an entry on the day-end bar could not be EOD-closed on its own bar).
	t := s.barTime(md)
	if !s.inSession(t) || s.isDayEnd(t, barSpanMinutes(md.Times)) {
		return sig
	}
	i := n - 1
	// 2. session anchor must be warmed and the session complete inside the window.
	vwap, sigma, bfo, ok := s.sessionVWAP(md)
	if !ok || vwap[i] <= 0 || sigma[i] <= 0 || bfo[i] < s.p.MinBarsFromOpen {
		return sig
	}
	closeP := md.Closes[i]
	if closeP <= 0 {
		return sig
	}
	dev := vwap[i] - closeP
	// 3. deep enough below the anchor, but not a collapse.
	if dev < s.p.EntryK*sigma[i] {
		return sig
	}
	if s.p.MaxDevK > 0 && dev > s.p.MaxDevK*sigma[i] {
		return sig
	}
	// 4. the move to the target must clear round-trip costs.
	if dev/closeP*100 < s.p.MinEdgePct {
		return sig
	}
	// 5. not a falling knife.
	if !s.closePosOK(md.Highs[i], md.Lows[i], closeP) {
		return sig
	}
	// 6. daily trend (fails closed).
	if !s.dailyTrendOK(md.DailyCloses, closeP) {
		return sig
	}
	// 7. optional ATR stop.
	var stop, atr float64
	if s.p.StopATR > 0 {
		atr = indicators.ATR(md.Highs, md.Lows, md.Closes, s.p.ATRPeriod)
		if atr <= 0 {
			return sig
		}
		stop = closeP - s.p.StopATR*atr
	}
	sig.Kind = model.SignalBuy
	sig.StopLoss = stop
	sig.TakeProfit = vwap[i]
	sig.ATR = atr
	sig.EntryReason = s.entryReason(vwap[i], sigma[i], closeP, stop, atr)
	return sig
}

// entryReason renders the human-readable rationale shown in the trade journal.
func (s *Strategy) entryReason(vwapNow, sigmaNow, entry, stop, atr float64) string {
	stopHow := "стоп выключен (StopATR=0)"
	if s.p.StopATR > 0 {
		stopHow = fmt.Sprintf("стоп %.4f (вход − %.2f×ATR, ATR=%.4f)", stop, s.p.StopATR, atr)
	}
	dev := vwapNow - entry
	return fmt.Sprintf(
		"цена %.4f ниже VWAP %.4f на %.4f (%.2f×σ, σ=%.4f; %.2f%% хода до цели); %s",
		entry, vwapNow, dev, dev/sigmaNow, sigmaNow, dev/entry*100, stopHow,
	)
}
```

Плюс временная заглушка управления позицией, чтобы пакет компилировался (полноценно реализуется в Task 3):

```go
// manage is filled in by the exits task; a flat no-op keeps the package compiling.
func (s *Strategy) manage(md strategy.MarketData, sig model.Signal) model.Signal {
	return sig
}
```

Заметка про заполнение ордеров: правок в `internal/service/trading_strategy/scalping/model` не требуется. `"SL"` уже входит в `model.IsStopReason` (заполнение по `min(StopLoss, open)`), `"TP"` движок обрабатывает отдельно (`max(TakeProfit, open)`), а `"TIME"` и `"EOD"` штатно заполняются по цене закрытия бара. `sig.TakeProfit`, выставленный на входе, движок только сохраняет в позицию для журнала — сам он по нему не выходит.

- [ ] **Step 4: Прогнать тесты**

Run: `go test ./internal/service/trading_strategy/vwap_rev/... -v`
Expected: PASS

- [ ] **Step 5: Полный гейт**

Run: `./bin/mage ci`
Expected: PASS

- [ ] **Step 6: Коммит**

```bash
git add internal/service/trading_strategy/vwap_rev/
git commit -m "feat(vwap_rev): ядро и правила входа по отклонению от сессионного VWAP"
```

---

### Task 3: Ядро — выходы, приоритет и `Explain`

**Files:**
- Modify: `internal/service/trading_strategy/vwap_rev/strategy/core/core.go` (заменить заглушку `manage`, добавить `barsHeld`, `exitReason`, `Explain`)
- Test: `internal/service/trading_strategy/vwap_rev/strategy/core/core_test.go` (дописать)

**Interfaces:**
- Consumes: всё из Task 2; `strategy.Position{PurchasePrice, StopLoss, TakeProfit, EntryATR, EntryTime}`.
- Produces: `(*Strategy).Explain(md strategy.MarketData) string` — необязательный интерфейс, который движок подхватывает в `Trace` (`internal/domain/backtest/engine.go:225`). Коды выхода: `"SL"`, `"TP"`, `"TIME"`, `"EOD"`.

- [ ] **Step 1: Написать падающие тесты выходов**

Дописать в `core_test.go`:

```go
// openAt returns MarketData for a held position entered `heldBars` bars before the last bar.
func withPosition(md strategy.MarketData, entryPrice, stop float64, heldBars int) strategy.MarketData {
	last := md.Times[len(md.Times)-1]
	md.Position = &strategy.Position{
		PurchasePrice: entryPrice,
		Quantity:      10,
		StopLoss:      stop,
		EntryTime:     last.Add(-time.Duration(heldBars) * 30 * time.Minute),
	}
	return md
}

func TestExitStopWinsOverTargetOnTheSameBar(t *testing.T) {
	md := setupEntry()
	i := len(md.Closes) - 1
	// The bar sweeps both the stop below and the VWAP above.
	md.Lows[i] = 90
	md.Highs[i] = 120
	md = withPosition(md, 99.5, 95, 1)
	sig := decide(t, DefaultParams(), md)
	if sig.Kind != model.SignalSell || sig.Reason != "SL" {
		t.Fatalf("Kind/Reason = %v/%q want Sell/SL", sig.Kind, sig.Reason)
	}
	if sig.StopLoss != 95 {
		t.Fatalf("StopLoss = %v want 95 (frozen at entry)", sig.StopLoss)
	}
}

func TestExitTargetIsPreviousBarVWAP(t *testing.T) {
	md := setupEntry()
	i := len(md.Closes) - 1
	md.Highs[i] = 1000 // certainly reaches any target
	md.Lows[i] = 99
	md = withPosition(md, 99.5, 1, 1)

	vwap, _, _, ok := NewWithParams("T", DefaultParams()).sessionVWAP(md)
	if !ok {
		t.Fatalf("sessionVWAP not usable in the fixture")
	}
	sig := decide(t, DefaultParams(), md)
	if sig.Kind != model.SignalSell || sig.Reason != "TP" {
		t.Fatalf("Kind/Reason = %v/%q want Sell/TP", sig.Kind, sig.Reason)
	}
	if sig.TakeProfit != vwap[i-1] {
		t.Fatalf("TakeProfit = %v want previous-bar VWAP %v (current bar's is %v)",
			sig.TakeProfit, vwap[i-1], vwap[i])
	}
}

func TestExitTimeStop(t *testing.T) {
	base := setupEntry()
	i := len(base.Closes) - 1
	base.Highs[i] = 99.7 // never reaches the VWAP
	base.Lows[i] = 99.4

	tests := []struct {
		name     string
		held     int
		maxHold  int
		wantKind model.SignalKind
	}{
		{"not held long enough", 2, 8, model.SignalNone},
		{"held exactly the limit", 8, 8, model.SignalSell},
		{"MaxHoldBars<=0 disables", 50, 0, model.SignalNone},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			p := DefaultParams()
			p.MaxHoldBars = tc.maxHold
			md := withPosition(base, 99.5, 1, tc.held)
			sig := decide(t, p, md)
			if sig.Kind != tc.wantKind {
				t.Fatalf("Kind = %v want %v", sig.Kind, tc.wantKind)
			}
			if tc.wantKind == model.SignalSell && sig.Reason != "TIME" {
				t.Fatalf("Reason = %q want TIME", sig.Reason)
			}
		})
	}
}

// A zero EntryTime must NOT be read as "held forever" — the time exit degrades to off.
func TestExitTimeStopSilentOnUnknownEntryTime(t *testing.T) {
	md := setupEntry()
	i := len(md.Closes) - 1
	md.Highs[i] = 99.7
	md.Lows[i] = 99.4
	md = withPosition(md, 99.5, 1, 1)
	md.Position.EntryTime = time.Time{}
	if got := decide(t, DefaultParams(), md).Kind; got != model.SignalNone {
		t.Fatalf("Kind = %v want SignalNone when EntryTime is unknown", got)
	}
}

func TestExitEndOfDay(t *testing.T) {
	md := setupEntry()
	i := len(md.Closes) - 1
	md.Highs[i] = 99.7
	md.Lows[i] = 99.4
	// Move the last bar to 22:30 MSK: it is the last one before DayEndMin (23:00).
	md.Times[i] = time.Date(2026, 3, 3, 22, 30, 0, 0, msk)
	md = withPosition(md, 99.5, 1, 1)
	p := DefaultParams()
	p.MaxHoldBars = 0 // isolate the EOD rule
	sig := decide(t, p, md)
	if sig.Kind != model.SignalSell || sig.Reason != "EOD" {
		t.Fatalf("Kind/Reason = %v/%q want Sell/EOD", sig.Kind, sig.Reason)
	}
}

func TestExplainMentionsEveryGate(t *testing.T) {
	out := NewWithParams("T", DefaultParams()).Explain(setupEntry())
	for _, want := range []string{"сессия", "VWAP", "σ", "MinEdgePct", "дневной тренд", "стоп"} {
		if !strings.Contains(out, want) {
			t.Fatalf("Explain missing %q; got:\n%s", want, out)
		}
	}
}
```

Дополнить импорты `core_test.go`: `"strings"`.

- [ ] **Step 2: Прогнать тесты и убедиться, что они падают**

Run: `go test ./internal/service/trading_strategy/vwap_rev/... -run 'TestExit|TestExplain' -v`
Expected: FAIL — выходы не реализованы

- [ ] **Step 3: Заменить заглушку `manage` реализацией**

В `core.go` добавить в импорты `"math"` и `"strings"` (в Task 2 их не было), удалить заглушку `manage` и добавить:

```go
// barsHeld counts bars from the position's entry to the current bar, purely from EntryTime,
// the current bar time and the data-inferred span. Returns -1 when either time is unknown, so
// the TIME exit degrades to "do not fire" instead of closing the position on its first managed
// bar. Positions never survive the EOD close, so the window is always inside one session and
// the uniform span is exact.
func (s *Strategy) barsHeld(md strategy.MarketData) int {
	pos := md.Position
	t := s.barTime(md)
	if pos == nil || pos.EntryTime.IsZero() || t.IsZero() {
		return -1
	}
	span := barSpanMinutes(md.Times)
	if span <= 0 {
		return -1
	}
	return int(math.Round(t.Sub(pos.EntryTime).Minutes() / float64(span)))
}

// manage handles an open long, exiting in precedence SL -> TP -> TIME -> EOD.
//
// SL is read from the position (frozen at entry) and is checked BEFORE the target: the
// intrabar order of the two is unknown, so the worst case is assumed. The target is the
// PREVIOUS bar's session VWAP — the level a resting limit order could have carried into this
// bar; using the current bar's VWAP would be look-ahead. SL fills at the stop (gap-adjusted by
// the engine), TP at max(target, open), TIME and EOD at the bar close.
func (s *Strategy) manage(md strategy.MarketData, sig model.Signal) model.Signal {
	pos := md.Position
	n := len(md.Closes)
	if pos == nil || n < 2 || len(md.Highs) != n || len(md.Lows) != n {
		return sig
	}
	i := n - 1
	high, low, closeP := md.Highs[i], md.Lows[i], md.Closes[i]

	// 1. hard stop (always active, checked first by design).
	if pos.StopLoss > 0 && low <= pos.StopLoss {
		sig.Kind, sig.Reason = model.SignalSell, "SL"
		sig.StopLoss = pos.StopLoss
		sig.ExitReason = fmt.Sprintf("SL: low %.4f ≤ стоп %.4f (вход %.4f)", low, pos.StopLoss, pos.PurchasePrice)
		return sig
	}
	// 2. target — the previous bar's VWAP, reachable intrabar.
	if vwap, _, _, ok := s.sessionVWAP(md); ok && vwap[i-1] > 0 && high >= vwap[i-1] {
		sig.Kind, sig.Reason = model.SignalSell, "TP"
		sig.TakeProfit = vwap[i-1]
		sig.ExitReason = fmt.Sprintf("TP: high %.4f достиг VWAP предыдущего бара %.4f (вход %.4f)",
			high, vwap[i-1], pos.PurchasePrice)
		return sig
	}
	// 3. time stop: a reversion that has not happened by now probably will not.
	if s.p.MaxHoldBars > 0 {
		if held := s.barsHeld(md); held >= 0 && held >= s.p.MaxHoldBars {
			sig.Kind, sig.Reason = model.SignalSell, "TIME"
			sig.ExitReason = fmt.Sprintf("TIME: удержание %d баров ≥ %d, выход по %.4f (вход %.4f)",
				held, s.p.MaxHoldBars, closeP, pos.PurchasePrice)
			return sig
		}
	}
	// 4. end of day (always active).
	if s.isDayEnd(s.barTime(md), barSpanMinutes(md.Times)) {
		sig.Kind, sig.Reason = model.SignalSell, "EOD"
		sig.ExitReason = fmt.Sprintf("EOD: закрытие на конец дня по %.4f (вход %.4f)", closeP, pos.PurchasePrice)
	}
	return sig
}

// Explain returns a gate-by-gate verdict for one bar, consumed by the engine's Trace
// (-explain). It recomputes the same values enter()/manage() do and reports each gate.
func (s *Strategy) Explain(md strategy.MarketData) string {
	var sb strings.Builder
	n := len(md.Closes)
	if n < 2 || len(md.Highs) != n || len(md.Lows) != n {
		sb.WriteString("недостаточно свечей\n")
		return sb.String()
	}
	i := n - 1
	barT := s.barTime(md)
	span := barSpanMinutes(md.Times)
	fmt.Fprintf(&sb, "сессия: вход разрешён? %v (бар %v); конец дня? %v\n",
		s.inSession(barT), barT, s.isDayEnd(barT, span))

	vwap, sigma, bfo, ok := s.sessionVWAP(md)
	if !ok || vwap[i] <= 0 || sigma[i] <= 0 {
		sb.WriteString("VWAP: не прогрет (нет времён, нет объёма или окно невалидно)\n")
		return sb.String()
	}
	closeP := md.Closes[i]
	dev := vwap[i] - closeP
	fmt.Fprintf(&sb, "бар в сессии №%d (нужно ≥ %d); VWAP %.4f, σ %.4f\n",
		bfo[i], s.p.MinBarsFromOpen, vwap[i], sigma[i])
	fmt.Fprintf(&sb, "отклонение вниз %.4f = %.2f×σ; порог входа %.2f×σ? %v; не глубже %.2f×σ? %v\n",
		dev, dev/sigma[i], s.p.EntryK, dev >= s.p.EntryK*sigma[i],
		s.p.MaxDevK, s.p.MaxDevK <= 0 || dev <= s.p.MaxDevK*sigma[i])
	fmt.Fprintf(&sb, "ход до цели %.2f%%; MinEdgePct %.2f%%? %v\n",
		dev/closeP*100, s.p.MinEdgePct, dev/closeP*100 >= s.p.MinEdgePct)
	fmt.Fprintf(&sb, "закрытие не в нижней части бара (порог %.2f)? %v\n",
		s.p.MinClosePos, s.closePosOK(md.Highs[i], md.Lows[i], closeP))
	if s.p.UseDailyTrend != 1 {
		sb.WriteString("дневной тренд: гейт выключен (UseDailyTrend=0)\n")
	} else {
		fmt.Fprintf(&sb, "дневной тренд: цена выше EMA(%d) по дневкам (дней %d)? %v\n",
			s.p.DailyEMAPeriod, len(md.DailyCloses), s.dailyTrendOK(md.DailyCloses, closeP))
	}
	if s.p.StopATR > 0 {
		atr := indicators.ATR(md.Highs, md.Lows, md.Closes, s.p.ATRPeriod)
		fmt.Fprintf(&sb, "ATR(%d) %.4f; стоп %.4f (вход − %.2f×ATR)\n",
			s.p.ATRPeriod, atr, closeP-s.p.StopATR*atr, s.p.StopATR)
	} else {
		sb.WriteString("стоп: выключен (StopATR=0)\n")
	}
	if md.Position != nil {
		fmt.Fprintf(&sb, "позиция открыта: удержано баров %d (лимит %d); цель = VWAP пред. бара %.4f\n",
			s.barsHeld(md), s.p.MaxHoldBars, vwap[i-1])
	}
	return sb.String()
}
```

- [ ] **Step 4: Прогнать тесты**

Run: `go test ./internal/service/trading_strategy/vwap_rev/... -v`
Expected: PASS

- [ ] **Step 5: Полный гейт**

Run: `./bin/mage ci`
Expected: PASS

- [ ] **Step 6: Коммит**

```bash
git add internal/service/trading_strategy/vwap_rev/
git commit -m "feat(vwap_rev): выходы SL/TP/TIME/EOD и Explain"
```

---

### Task 4: Тест на отсутствие lookahead

**Files:**
- Test: `internal/service/trading_strategy/vwap_rev/strategy/core/core_test.go` (дописать)

**Interfaces:**
- Consumes: всё из Task 2 и Task 3.
- Produces: ничего для последующих задач — это проверочная сеть.

Это самый важный тест плана. Ядро обязано быть функцией от `[0..i]`, а не от того, где обрезано окно. Правило «первая сессия в окне неполная» намеренно делает результат детерминированным, и тест обязан это уважать: срезы берутся только такие, где до сессии последнего бара остаётся хотя бы одна полная сессия.

- [ ] **Step 1: Написать падающий тест**

Дополнить импорты `core_test.go`: `"math"`. Даты подобраны так, что 2026-03-02 — понедельник, а сессии 2-5 марта приходятся на будни: выходные не попадают в фикстуру и не мешают счёту сессий.

```go
// multiDay builds `days` sessions of `perDay` 30-minute bars starting at 07:00 MSK, with a
// deterministic zig-zag so deviations from the VWAP actually occur.
func multiDay(days, perDay int) strategy.MarketData {
	md := strategy.MarketData{}
	for d := 0; d < days; d++ {
		day := time.Date(2026, 3, 2+d, 7, 0, 0, 0, msk)
		for b := 0; b < perDay; b++ {
			p := 100 + float64(d) + math.Sin(float64(b)/3)*0.6
			md.Highs = append(md.Highs, p+0.2)
			md.Lows = append(md.Lows, p-0.25)
			md.Closes = append(md.Closes, p)
			md.Volumes = append(md.Volumes, 1000+int64(b))
			md.Times = append(md.Times, day.Add(time.Duration(b)*30*time.Minute))
		}
	}
	md.Price = md.Closes[len(md.Closes)-1]
	md.DailyCloses = make([]float64, 60)
	for i := range md.DailyCloses {
		md.DailyCloses[i] = 50 + float64(i)
	}
	return md
}

// slice returns md restricted to bars [from:to).
func sliceMD(md strategy.MarketData, from, to int) strategy.MarketData {
	out := md
	out.Highs = md.Highs[from:to]
	out.Lows = md.Lows[from:to]
	out.Closes = md.Closes[from:to]
	out.Volumes = md.Volumes[from:to]
	out.Times = md.Times[from:to]
	out.Price = out.Closes[len(out.Closes)-1]
	return out
}

// TestNoLookaheadAcrossWindowCuts is the load-bearing safety net: the decision on bar i must
// not depend on how much history precedes it, as long as one full session still precedes the
// session bar i belongs to. Any use of future bars, or any dependence on the window length,
// breaks this.
func TestNoLookaheadAcrossWindowCuts(t *testing.T) {
	const perDay = 20
	full := multiDay(4, perDay)
	s := NewWithParams("T", DefaultParams())

	checked := 0
	// Bars of the last two sessions: each has at least one whole session before its own.
	for i := 2 * perDay; i < len(full.Closes); i++ {
		want := s.Decide(sliceMD(full, 0, i+1))
		// Cut so that exactly one full session precedes bar i's session.
		sessionStart := (i / perDay) * perDay
		for _, from := range []int{sessionStart - perDay, sessionStart - perDay + 3} {
			if from < 0 {
				continue
			}
			got := s.Decide(sliceMD(full, from, i+1))
			if got.Kind != want.Kind || got.Reason != want.Reason {
				t.Fatalf("bar %d: cut at %d gave %v/%q, full window gave %v/%q",
					i, from, got.Kind, got.Reason, want.Kind, want.Reason)
			}
			if math.Abs(got.StopLoss-want.StopLoss) > 1e-9 || math.Abs(got.TakeProfit-want.TakeProfit) > 1e-9 {
				t.Fatalf("bar %d: cut at %d gave stop/target %v/%v, full window %v/%v",
					i, from, got.StopLoss, got.TakeProfit, want.StopLoss, want.TakeProfit)
			}
			checked++
		}
	}
	if checked == 0 {
		t.Fatalf("fixture produced no comparisons")
	}
}

// The mirror case: a window that leaves bar i's session FIRST must refuse to trade, because
// the anchor of that session is unknown.
func TestFirstSessionInWindowNeverTrades(t *testing.T) {
	const perDay = 20
	full := multiDay(4, perDay)
	s := NewWithParams("T", DefaultParams())
	for i := 3*perDay + 1; i < len(full.Closes); i++ {
		got := s.Decide(sliceMD(full, 3*perDay, i+1))
		if got.Kind != model.SignalNone {
			t.Fatalf("bar %d traded (%v) while its session was the window's first", i, got.Kind)
		}
	}
}
```

- [ ] **Step 2: Прогнать тест**

Run: `go test ./internal/service/trading_strategy/vwap_rev/... -run 'TestNoLookahead|TestFirstSession' -v`
Expected: PASS сразу, если Task 1-3 сделаны верно. **Если падает — чинить ядро, а не тест.** Типичная причина: где-то используется `vwap[i]` вместо `vwap[i-1]` в качестве цели, либо индикатор не помечает первую сессию окна.

- [ ] **Step 3: Убедиться, что тест действительно ловит регрессию**

Временно заменить в `manage` цель `vwap[i-1]` на `vwap[i]` и прогнать:

Run: `go test ./internal/service/trading_strategy/vwap_rev/... -run TestExitTargetIsPreviousBarVWAP -v`
Expected: FAIL. Вернуть `vwap[i-1]` обратно и убедиться, что снова PASS. Мутацию НЕ коммитить.

- [ ] **Step 4: Полный гейт**

Run: `./bin/mage ci`
Expected: PASS

- [ ] **Step 5: Коммит**

```bash
git add internal/service/trading_strategy/vwap_rev/strategy/core/core_test.go
git commit -m "test(vwap_rev): решения не зависят от границы окна (нет lookahead)"
```

---

### Task 5: Реестр и ветка в CLI

**Files:**
- Create: `internal/service/backtest/vwap_rev_registry.go`
- Create: `internal/service/backtest/vwap_rev_registry_test.go`
- Modify: `cmd/backtest/main.go` (строка ~41 — текст флага `-strategy`; строки ~156-166 — `switch strategyName`)

**Interfaces:**
- Consumes: `core.DefaultParams()`, `core.NewWithParams` (Task 2); `svc.Binding{DefaultParams func() any; Build func(any) strategy.Strategy; ParseParams func([]byte) (any, error)}`.
- Produces: `backtest.VWAPRevLookupOrGeneric(ticker string) Binding`; CLI-значение `-strategy vwap_rev`.

- [ ] **Step 1: Написать падающий тест реестра**

Создать `internal/service/backtest/vwap_rev_registry_test.go`:

```go
package backtest

import (
	"testing"

	"tinvest/internal/service/trading_strategy/vwap_rev/strategy/core"
)

func TestVWAPRevBindingDefaults(t *testing.T) {
	b := VWAPRevLookupOrGeneric("GAZP")
	got, ok := b.DefaultParams().(core.Params)
	if !ok {
		t.Fatalf("DefaultParams type = %T want core.Params", b.DefaultParams())
	}
	if got != core.DefaultParams() {
		t.Fatalf("defaults = %+v want %+v", got, core.DefaultParams())
	}
	if s := b.Build(got); s.Ticker() != "GAZP" {
		t.Fatalf("ticker = %q want GAZP", s.Ticker())
	}
}

func TestVWAPRevParseParamsLayersOverDefaults(t *testing.T) {
	b := VWAPRevLookupOrGeneric("GAZP")
	parsed, err := b.ParseParams([]byte(`{"EntryK": 2.5, "MaxHoldBars": 12}`))
	if err != nil {
		t.Fatalf("ParseParams: %v", err)
	}
	got := parsed.(core.Params)
	if got.EntryK != 2.5 || got.MaxHoldBars != 12 {
		t.Fatalf("overrides not applied: %+v", got)
	}
	if got.MinEdgePct != core.DefaultParams().MinEdgePct {
		t.Fatalf("untouched field must keep default: MinEdgePct=%v", got.MinEdgePct)
	}
}

func TestVWAPRevParseParamsRejectsGarbage(t *testing.T) {
	b := VWAPRevLookupOrGeneric("GAZP")
	if _, err := b.ParseParams([]byte(`not json`)); err == nil {
		t.Fatalf("want error on malformed JSON")
	}
}
```

- [ ] **Step 2: Прогнать тест и убедиться, что он падает**

Run: `go test ./internal/service/backtest/ -run TestVWAPRev -v`
Expected: FAIL — `undefined: VWAPRevLookupOrGeneric`

- [ ] **Step 3: Реализовать реестр**

Создать `internal/service/backtest/vwap_rev_registry.go`:

```go
package backtest

import (
	"encoding/json"
	"fmt"

	"tinvest/internal/service/trading_strategy/scalping/strategy"
	"tinvest/internal/service/trading_strategy/vwap_rev/strategy/core"
)

// vwapRevBindingFor builds a Binding for a ticker on the vwap_rev engine. The strategy is
// ticker-agnostic; only the ticker label differs, so a single generic default suffices until
// calibration proves per-ticker params are needed.
func vwapRevBindingFor(ticker string) Binding {
	return Binding{
		DefaultParams: func() any { return core.DefaultParams() },
		Build: func(params any) strategy.Strategy {
			return core.NewWithParams(ticker, params.(core.Params))
		},
		ParseParams: func(raw []byte) (any, error) {
			p := core.DefaultParams() // start from defaults so partial JSON overrides
			if err := json.Unmarshal(raw, &p); err != nil {
				return nil, fmt.Errorf("backtest: parse vwap_rev params: %w", err)
			}
			return p, nil
		},
	}
}

// VWAPRevLookupOrGeneric returns a vwap_rev binding bound to the ticker. There are no
// per-ticker packages yet (calibration pending), so every ticker gets the generic defaults.
func VWAPRevLookupOrGeneric(ticker string) Binding {
	return vwapRevBindingFor(ticker)
}
```

- [ ] **Step 4: Подключить ветку в CLI**

В `cmd/backtest/main.go` в `switch strategyName` добавить перед `default`:

```go
	case "vwap_rev":
		binding = svc.VWAPRevLookupOrGeneric(ticker)
```

И обновить две строки со списком стратегий — описание флага и текст ошибки:

```go
		strategyName = flag.String("strategy", "scalping", "strategy engine: scalping|reversion|scalping_rsimacd|rsi_ema|vwap_rev")
```

```go
		return fmt.Errorf("unknown strategy %q (want scalping|reversion|scalping_rsimacd|rsi_ema|vwap_rev)", strategyName)
```

HTF-серию для `vwap_rev` грузить НЕ нужно: в `switch strategyName` для HTF (около строки 196) ветку не добавлять — дефолтная ветка оставляет `htfCandles` пустым, а дневные свечи грузятся безусловно выше.

- [ ] **Step 5: Прогнать тесты и убедиться, что CLI собирается**

Run: `go test ./internal/service/backtest/ -run TestVWAPRev -v && go build ./cmd/...`
Expected: PASS и успешная сборка

- [ ] **Step 6: Полный гейт**

Run: `./bin/mage ci`
Expected: PASS

- [ ] **Step 7: Коммит**

```bash
git add internal/service/backtest/vwap_rev_registry.go internal/service/backtest/vwap_rev_registry_test.go cmd/backtest/main.go
git commit -m "feat(vwap_rev): реестр стратегии и ветка -strategy vwap_rev"
```

---

### Task 6: Сетка калибровки и документация

**Files:**
- Create: `data/params/vwap_rev/grid.json`
- Create: `internal/service/backtest/vwap_rev_grid_test.go`
- Create: `docs/vwap_rev/strategy.md`

**Interfaces:**
- Consumes: `core.Params` (Task 2); `backtest.ParsePhases(raw []byte) ([]Phase, error)`, `backtest.applyField(p any, name string, v float64) (any, error)`, `Phase{Name string; KeepTop int; Grid Grid}`.
- Produces: `data/params/vwap_rev/grid.json` — 124 комбинации в 6 фазах.

- [ ] **Step 1: Написать падающий тест сетки**

Создать `internal/service/backtest/vwap_rev_grid_test.go`:

```go
package backtest

import (
	"os"
	"testing"

	"tinvest/internal/service/trading_strategy/vwap_rev/strategy/core"
)

// TestVWAPRevGridFieldsExist parses the shipped calibration grid and applies every swept
// value to core.Params. A typo in the JSON would otherwise silently collapse a whole phase.
func TestVWAPRevGridFieldsExist(t *testing.T) {
	raw, err := os.ReadFile("../../../data/params/vwap_rev/grid.json")
	if err != nil {
		t.Fatalf("read grid: %v", err)
	}
	phases, err := ParsePhases(raw)
	if err != nil {
		t.Fatalf("parse phases: %v", err)
	}
	var names []string
	for _, ph := range phases {
		for name, values := range ph.Grid {
			names = append(names, name)
			for _, v := range values {
				if _, err := applyField(core.DefaultParams(), name, v); err != nil {
					t.Fatalf("phase %q: apply %s=%v: %v", ph.Name, name, v, err)
				}
			}
		}
	}
	wantSwept := []string{
		"EntryK", "MinEdgePct", "UseDailyTrend", "DailyEMAPeriod",
		"MaxDevK", "MinClosePos", "StopATR", "MaxHoldBars", "MinBarsFromOpen",
	}
	for _, want := range wantSwept {
		found := false
		for _, got := range names {
			if got == want {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("grid must sweep %s", want)
		}
	}
}

// Every optional filter must carry its own "off" control point so the un-filtered baseline
// competes on the leaderboard instead of being assumed away.
func TestVWAPRevGridKeepsFilterOffControlPoints(t *testing.T) {
	raw, err := os.ReadFile("../../../data/params/vwap_rev/grid.json")
	if err != nil {
		t.Fatalf("read grid: %v", err)
	}
	phases, err := ParsePhases(raw)
	if err != nil {
		t.Fatalf("parse phases: %v", err)
	}
	needOff := map[string]float64{
		"MaxDevK":       0,
		"MinClosePos":   0,
		"UseDailyTrend": 0,
		"MaxHoldBars":   0,
	}
	for _, ph := range phases {
		for name, values := range ph.Grid {
			off, ok := needOff[name]
			if !ok {
				continue
			}
			for _, v := range values {
				if v == off {
					delete(needOff, name)
					break
				}
			}
		}
	}
	for name := range needOff {
		t.Fatalf("grid must include the filter-off control point for %s", name)
	}
}
```

- [ ] **Step 2: Прогнать тест и убедиться, что он падает**

Run: `go test ./internal/service/backtest/ -run TestVWAPRevGrid -v`
Expected: FAIL — файла `data/params/vwap_rev/grid.json` нет

- [ ] **Step 3: Создать сетку**

Создать `data/params/vwap_rev/grid.json`:

```json
{
  "_comment": "vwap_rev phased grid, 124 combos. Phase 1 (entry) sweeps the two numbers that define the setup: how deep below the session VWAP the close must sit (EntryK, in sigmas) and how large that move must be in percent to clear the 0.1% round-trip cost (MinEdgePct). Phase 2 (context) sweeps the daily trend gate, including UseDailyTrend=0 which turns it off - at 0 the three DailyEMAPeriod values collapse to one identical control config, so duplicate rows on the leaderboard are expected. Phase 3 (quality) sweeps the two optional entry filters, each with its own off switch (MaxDevK=0, MinClosePos=0). Phase 4 (risk) sweeps the ATR stop multiplier. Phase 5 (hold) sweeps the time exit, MaxHoldBars=0 = off. Phase 6 (timing) sweeps how far into the session trading may start; on 30m bars anchored at 07:00 MSK the values map to 09:00 / 10:00 / 11:00 / 12:00. The session bounds, DayEndMin and the VWAP target itself are fixed by the strategy definition and deliberately NOT swept - the target is the thesis, not a knob. RunPhases expands each phase over the previous phase's keepTop seeds: 12 + 4x6 + 4x9 + 4x4 + 4x5 + 4x4 = 124 combos. Judge on pooled OOS profit factor from a walk-forward, never on the in-sample best.",
  "phases": [
    {
      "name": "entry",
      "keepTop": 4,
      "grid": {
        "EntryK": [1.0, 1.5, 2.0, 2.5],
        "MinEdgePct": [0.25, 0.35, 0.5]
      }
    },
    {
      "name": "context",
      "keepTop": 4,
      "grid": {
        "UseDailyTrend": [0, 1],
        "DailyEMAPeriod": [20, 50, 100]
      }
    },
    {
      "name": "quality",
      "keepTop": 4,
      "grid": {
        "MaxDevK": [0, 3.0, 4.0],
        "MinClosePos": [0, 0.25, 0.33]
      }
    },
    {
      "name": "risk",
      "keepTop": 4,
      "grid": {
        "StopATR": [0.8, 1.2, 1.6, 2.0]
      }
    },
    {
      "name": "hold",
      "keepTop": 4,
      "grid": {
        "MaxHoldBars": [0, 4, 6, 8, 12]
      }
    },
    {
      "name": "timing",
      "grid": {
        "MinBarsFromOpen": [4, 6, 8, 10]
      }
    }
  ]
}
```

- [ ] **Step 4: Прогнать тесты сетки**

Run: `go test ./internal/service/backtest/ -run TestVWAPRevGrid -v`
Expected: PASS

- [ ] **Step 5: Написать документацию**

Создать `docs/vwap_rev/strategy.md` — правила (вход по семи пунктам, выходы SL/TP/TIME/EOD с приоритетом), таблицу параметров с дефолтами и единицами измерения (`MinEdgePct` в процентах, `MinClosePos` — доля 0..1), пояснение про якорь VWAP на первом баре календарного дня и про правило «первая сессия в окне неполная», ограничение по таймфрейму (30m — референс; ниже 15m пола `Lookback` не хватит), а также команды:

```
# разведочный прогон на дефолтах
go run ./cmd/backtest -ticker GAZP -strategy vwap_rev -interval Minutes30 \
  -out ./reports/GAZP_vwap -months 24

# диагностика одного бара
go run ./cmd/backtest -ticker GAZP -strategy vwap_rev -interval Minutes30 \
  -months 24 -explain "2026-05-14 14:30"

# калибровка walk-forward (единственный источник вердикта)
go run ./cmd/backtest -ticker GAZP -strategy vwap_rev -interval Minutes30 \
  -calibrate data/params/vwap_rev/grid.json -out ./reports/GAZP_vwap \
  -months 24 -train-months 12 -test-months 3 -metric profit_factor -min-trades 40
```

В доке явно перечислить шесть критериев приёмки из спеки и критерий смерти.

- [ ] **Step 6: Полный гейт**

Run: `./bin/mage ci`
Expected: PASS

- [ ] **Step 7: Коммит**

```bash
git add data/params/vwap_rev/ internal/service/backtest/vwap_rev_grid_test.go docs/vwap_rev/
git commit -m "feat(vwap_rev): фазовая сетка калибровки и документация стратегии"
```

---

### Task 7: Калибровка и вердикт

**Files:**
- Create: `reports/GAZP_vwap/` (артефакты прогонов, генерируются CLI)
- Create: `data/params/vwap_rev/gazp_cal.json` (победившие параметры, если приёмка пройдена)
- Modify: `docs/vwap_rev/strategy.md` (раздел с результатами)

**Interfaces:**
- Consumes: всё из Task 1-6.
- Produces: вердикт — стратегия принята или закрыта. Никакого кода для последующих задач.

Это задача про измерение, а не про код. Правило одно: **вердикт выносится по pooled OOS, никогда по in-sample лучшему**.

- [ ] **Step 1: Разведочный прогон на дефолтах**

Run:
```bash
go run ./cmd/backtest -ticker GAZP -strategy vwap_rev -interval Minutes30 \
  -out ./reports/GAZP_vwap -months 24
```
Expected: прогон завершается, отчёт создан. Проверить в отчёте, что сделок **не менее 200** за 24 месяца. Если их десятки — сетап слишком редкий, и вся посылка про «частоту вместо размера» не выполняется: прежде чем калибровать, разобраться через `-explain`, какой гейт режет входы.

- [ ] **Step 2: Walk-forward калибровка**

Run:
```bash
go run ./cmd/backtest -ticker GAZP -strategy vwap_rev -interval Minutes30 \
  -calibrate data/params/vwap_rev/grid.json -out ./reports/GAZP_vwap \
  -months 24 -train-months 12 -test-months 3 -metric profit_factor -min-trades 40
```
Expected: отчёт walk-forward с pooled OOS и разбивкой по фолдам

- [ ] **Step 3: Свести шесть критериев приёмки**

Выписать из отчёта и проверить **все шесть сразу**:

| # | Критерий | Порог |
|---|---|---|
| 1 | pooled OOS profit factor | ≥ 1.5 |
| 2 | OOS-сделок | ≥ 150 |
| 3 | win rate по пулу OOS | ≥ 55% |
| 4 | средняя чистая сделка | ≥ +0.15% |
| 5 | фолдов с OOS PF > 1.0 | ≥ 3 из 4 |
| 6 | разброс `EntryK` и `MinEdgePct` между фолдами | ≤ один шаг сетки |

- [ ] **Step 4: Бонус-чек переносимости (не блокирующий)**

Прогнать победившие параметры GAZP **как есть, без пере-калибровки**:

```bash
go run ./cmd/backtest -ticker NVTK -strategy vwap_rev -interval Minutes30 \
  -params data/params/vwap_rev/gazp_cal.json -out ./reports/NVTK_vwap -months 24
go run ./cmd/backtest -ticker RUAL -strategy vwap_rev -interval Minutes30 \
  -params data/params/vwap_rev/gazp_cal.json -out ./reports/RUAL_vwap -months 24
```
Expected: информация к размышлению. Если PF схлопывается в ноль на обоих — edge, скорее всего, подогнан под GAZP, даже когда формальные шесть критериев пройдены. Это повод насторожиться, а не автоматически отклонить.

- [ ] **Step 5: Вынести вердикт и записать его**

Дописать в `docs/vwap_rev/strategy.md` раздел «Результаты калибровки»: дату, команду, фактические значения всех шести метрик, вывод.

**Если все шесть пройдены:** сохранить победившие параметры в `data/params/vwap_rev/gazp_cal.json` и предложить владельцу мерж (по конвенции репо мерж только после приёмки).

**Если хоть один провален:** стратегия закрывается. Не расширять сетку, не добавлять седьмой фильтр, не перебирать тикеры в поисках того, где сработало. Записать в док, ЧТО именно не прошло и с какими числами, — это и есть ценность отрицательного результата.

- [ ] **Step 6: Коммит**

```bash
git add docs/vwap_rev/strategy.md data/params/vwap_rev/
git commit -m "docs(vwap_rev): результаты walk-forward калибровки на GAZP и вердикт"
```

---

## Проверка плана против спеки

| Требование спеки | Задача |
|---|---|
| `SessionVWAP` с якорем на календарном дне, σ, `barsFromOpen` | Task 1 |
| Деградация индикатора: nil-вход, нулевой объём, первая сессия неполная | Task 1 (шаг 1, тесты) |
| Семь правил входа, включая `MinEdgePct` и fail-closed дневной гейт | Task 2 |
| `Lookback` ≥ 300 | Task 2 |
| Выходы SL → TP → TIME → EOD, приоритет SL над TP | Task 3 |
| Цель = VWAP предыдущего бара | Task 3 (+ Task 4, мутационная проверка) |
| `Explain` для `-explain` | Task 3 |
| Анти-lookahead тест | Task 4 |
| Реестр и ветка CLI, без HTF | Task 5 |
| Фазовая сетка 124 комбо с контрольными точками «выключено» | Task 6 |
| Документация стратегии | Task 6 |
| Walk-forward и шесть критериев приёмки, бонус-чек, критерий смерти | Task 7 |
