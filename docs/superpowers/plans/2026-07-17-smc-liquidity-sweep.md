# SMC Liquidity Sweep — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Backtest-only long-only стратегия «liquidity sweep»: вход после снятия ликвидности под фрактальным swing-low и reclaim-close, с опциональными SMC-фильтрами (OB/FVG/discount), проверяемая walk-forward'ом.

**Architecture:** Калька с ORB — пакет `internal/service/trading_strategy/smc/strategy` (core + 8 пер-тикерных пакетов), интерфейсы `scalping/model`+`scalping/strategy`, реестр в `internal/service/backtest`, регистрация в `cmd/backtest` под `-strategy smc`. Одно расширение общего ядра: поле `strategy.Position.EntryTime` для тайм-стопа (заполняет движок бэктеста).

**Tech Stack:** Go 1.25, stdlib + `tinvest/pkg/indicators` (ATR). Никаких новых зависимостей.

**Spec:** `docs/superpowers/specs/2026-07-17-smc-liquidity-sweep-design.md` — при расхождении спека главнее.

## Global Constraints

- Ветка: `feat/smc-liquidity-sweep` (уже создана от main).
- Гейт качества: `./bin/mage ci` (lint + `go test -race ./...` + mock-drift). `go build ./...` падает на `magefiles` — собирать `go build ./internal/... ./pkg/... ./cmd/...`.
- Decide обязан быть pure: никакого I/O, всё из `MarketData`.
- Анти-lookahead: swing подтверждается только через `SwingK` баров; уровни пересчитываются каждый бар из окна.
- `MarketData` НЕ содержит Open — «медвежья свеча» для OB определяется как down-close (`Close[i] < Close[i-1]`).
- Тумблеры фильтров — `int` 0/1 (грид умеет только числа).
- Все коммиты заканчивать трейлером: `Co-Authored-By: Claude Fable 5 <noreply@anthropic.com>`.
- Комментарии в коде — в стиле ORB core (по-английски, объясняют constraint); тексты Reason/Explain — по-русски, как в ORB.

---

### Task 1: strategy.Position.EntryTime

Тайм-стопу нужен возраст позиции; портфель движка уже хранит `entryTime`, но не отдаёт его стратегии.

**Files:**
- Modify: `internal/service/trading_strategy/scalping/strategy/strategy.go` (struct Position)
- Modify: `internal/domain/backtest/portfolio.go` (метод `strategyPosition`)
- Test: `internal/domain/backtest/portfolio_test.go`

**Interfaces:**
- Consumes: существующие `portfolio.open`, `portfolio.strategyPosition`.
- Produces: `strategy.Position.EntryTime time.Time` — нулевое в live, заполнено в бэктесте. Task 4 читает его в manage-pass.

- [ ] **Step 1: Написать падающий тест**

В `internal/domain/backtest/portfolio_test.go` добавить:

```go
func TestStrategyPositionCarriesEntryTime(t *testing.T) {
	p := newPortfolio(Config{InitialCash: 1000, Fraction: 1})
	entry := time.Date(2026, 7, 1, 10, 0, 0, 0, time.UTC)
	p.open(100, entry, 0, 0, 0, 95, "")
	pos := p.strategyPosition()
	if pos == nil {
		t.Fatal("strategyPosition() = nil, want position")
	}
	if !pos.EntryTime.Equal(entry) {
		t.Fatalf("EntryTime = %v, want %v", pos.EntryTime, entry)
	}
}
```

(Если в файле нет импорта `time` — добавить.)

- [ ] **Step 2: Убедиться, что тест падает**

Run: `go test ./internal/domain/backtest/ -run TestStrategyPositionCarriesEntryTime -v`
Expected: FAIL с `pos.EntryTime undefined` (ошибка компиляции).

- [ ] **Step 3: Реализация**

В `strategy.go`, в struct `Position` после поля `PrevMaxFavorablePrice` добавить:

```go
	// EntryTime is the open-time of the entry bar. Zero means "not set" (live
	// trading does not persist it); the backtest engine always populates it.
	// Time-based exits must degrade to a no-op when zero.
	EntryTime time.Time
```

(`strategy.go` уже импортирует `time`.)

В `portfolio.go`, в `strategyPosition()` добавить в литерал:

```go
		EntryTime:             p.entryTime,
```

- [ ] **Step 4: Тест зелёный + ничего не сломано**

Run: `go test ./internal/domain/backtest/ ./internal/service/... -count=1`
Expected: PASS везде.

- [ ] **Step 5: Commit**

```bash
git add internal/service/trading_strategy/scalping/strategy/strategy.go internal/domain/backtest/portfolio.go internal/domain/backtest/portfolio_test.go
git commit -m "feat(scalping): Position.EntryTime — backtest engine exposes entry bar time to strategies"
```

---

### Task 2: core skeleton — Params, Strategy, календарные хелперы

**Files:**
- Create: `internal/service/trading_strategy/smc/strategy/core/core.go`
- Test: `internal/service/trading_strategy/smc/strategy/core/core_test.go`

**Interfaces:**
- Consumes: `strategy.MarketData`, `model.Signal` (из scalping core).
- Produces: `core.Params{SwingK, ReclaimBars int; Buffer, TPR float64; MaxHoldDays, UseOB, UseFVG, UseDiscount int}`; `core.NewWithParams(ticker string, p Params) *Strategy`; методы `Ticker() string`, `Lookback() int`; хелперы `mskLoc`, `isWeekend(t time.Time) bool`, `sameMSKDay(a, b time.Time) bool`, `startOfMSKDay(t time.Time) time.Time`, `windowStart(times []time.Time) int`, `tradingDaysSince(times []time.Time, entry time.Time) int`; тестовые хелперы `bar`, `mkMD`, `msk`, `advance`, `flatBars`, `next`.

- [ ] **Step 1: Написать падающие тесты (вместе с тестовыми хелперами)**

Создать `core_test.go`:

```go
package core

import (
	"testing"
	"time"

	"tinvest/internal/service/trading_strategy/scalping/strategy"
)

// bar — компактная спека свечи для тестов: время открытия MSK + H/L/C/V
// (Open в MarketData отсутствует).
type bar struct {
	t       time.Time
	h, l, c float64
	v       int64
}

// mkMD собирает MarketData из баров (oldest-first) и позиции.
func mkMD(bars []bar, pos *strategy.Position) strategy.MarketData {
	md := strategy.MarketData{Position: pos}
	for _, b := range bars {
		md.Highs = append(md.Highs, b.h)
		md.Lows = append(md.Lows, b.l)
		md.Closes = append(md.Closes, b.c)
		md.Volumes = append(md.Volumes, b.v)
		md.Times = append(md.Times, b.t)
	}
	if n := len(bars); n > 0 {
		md.Price = bars[n-1].c
	}
	return md
}

func msk(y int, m time.Month, d, hh int) time.Time {
	return time.Date(y, m, d, hh, 0, 0, 0, mskLoc)
}

// advance — следующий Hour1-бар: +1ч внутри основной сессии (10..17),
// иначе 10:00 следующего буднего дня.
func advance(t time.Time) time.Time {
	nt := t.Add(time.Hour)
	if nt.In(mskLoc).Hour() < 18 && sameMSKDay(nt, t) {
		return nt
	}
	nt = startOfMSKDay(t).AddDate(0, 0, 1).Add(10 * time.Hour)
	for nt.Weekday() == time.Saturday || nt.Weekday() == time.Sunday {
		nt = nt.AddDate(0, 0, 1)
	}
	return nt
}

// flatBars — n «тихих» баров (H=p+1, L=p-1, C=p) с шагом advance от start.
func flatBars(start time.Time, n int, p float64) []bar {
	out := make([]bar, 0, n)
	t := start
	for i := 0; i < n; i++ {
		out = append(out, bar{t: t, h: p + 1, l: p - 1, c: p, v: 100})
		t = advance(t)
	}
	return out
}

// next — время открытия бара, следующего за последним в срезе.
func next(bars []bar) time.Time { return advance(bars[len(bars)-1].t) }

func TestWindowStartKeepsLastNDays(t *testing.T) {
	// 12 торговых дней по 8 баров: окно должно начинаться с первого бара
	// последних levelWindowDays (10) дней = индекс 2*8.
	bars := flatBars(msk(2026, 6, 1, 10), 12*8, 100)
	md := mkMD(bars, nil)
	if got, want := windowStart(md.Times), 16; got != want {
		t.Fatalf("windowStart = %d, want %d", got, want)
	}
	// Короткая история — окно с нулевого бара.
	short := mkMD(flatBars(msk(2026, 6, 1, 10), 5, 100), nil)
	if got := windowStart(short.Times); got != 0 {
		t.Fatalf("windowStart(short) = %d, want 0", got)
	}
}

func TestTradingDaysSince(t *testing.T) {
	// 3 торговых дня по 8 баров; вход в первый день.
	bars := flatBars(msk(2026, 7, 6, 10), 3*8, 100)
	md := mkMD(bars, nil)
	entry := msk(2026, 7, 6, 12)
	if got := tradingDaysSince(md.Times, entry); got != 2 {
		t.Fatalf("tradingDaysSince = %d, want 2 (Tue, Wed)", got)
	}
	// Вход в последний день — 0 полных дней после.
	if got := tradingDaysSince(md.Times, msk(2026, 7, 8, 10)); got != 0 {
		t.Fatalf("tradingDaysSince(last day) = %d, want 0", got)
	}
}
```

- [ ] **Step 2: Убедиться, что тесты падают**

Run: `go test ./internal/service/trading_strategy/smc/... -v`
Expected: FAIL — `undefined: mskLoc`, `windowStart`, и т.д. (компиляция).

- [ ] **Step 3: Реализация core.go**

```go
// Package core implements the SMC liquidity-sweep long-only swing strategy:
// a fractal swing-low is confirmed SwingK bars after its extreme; a bar
// piercing that level with its low and a close back above it within
// ReclaimBars is a stop-hunt — the reclaim bar buys at its close with a hard
// stop under the sweep extreme; exits are the intrabar stop, an R-multiple
// take-profit and a trading-day time-stop. Optional OB/FVG/discount filters
// are int toggles (0/1) so the calibration grid can sweep them. See
// docs/superpowers/specs/2026-07-17-smc-liquidity-sweep-design.md.
package core

import (
	"time"
)

// Fixed knobs — deliberately NOT part of Params: sweeping window/warm-up
// mechanics is how past strategies overfit.
const (
	atrPeriod        = 14 // stop unit
	levelWindowDays  = 10 // sliding window of distinct MSK days where levels live
	sessionOpenHour  = 10 // MSK: bars opening before 10:00 never enter
	eveningStartHour = 19 // MSK: bars opening at/after 19:00 (evening session) never enter
	// barsPerDayMax is the worst-case Hour1 bar count of one MSK day
	// (morning + main + evening sessions). Lookback must cover the whole
	// level window plus indicator warm-up.
	barsPerDayMax = 17
)

// Params are the SMC tunables. UseOB/UseFVG/UseDiscount are int toggles
// (grid values are numeric): 0 = off, 1 = on.
type Params struct {
	SwingK      int     // fractal wing: a swing-low needs SwingK strictly higher lows on each side
	ReclaimBars int     // max bars from pierce to reclaim close (0 = same-bar sweep only)
	Buffer      float64 // hard stop = sweepLow - Buffer*ATR(atrPeriod) at entry
	TPR         float64 // take-profit = entry + TPR*(entry - stop); <=0 disables
	MaxHoldDays int     // time-stop after this many distinct trading days; <=0 disables
	UseOB       int     // 1 = reclaim close must sit inside an unmitigated bullish order block
	UseFVG      int     // 1 = a bullish FVG must form between pierce and reclaim
	UseDiscount int     // 1 = entry close must be below the level-window range midpoint
}

// Strategy is the SMC rule bound to one ticker.
type Strategy struct {
	ticker string
	p      Params
}

// NewWithParams builds the strategy for a ticker with explicit params.
func NewWithParams(ticker string, p Params) *Strategy {
	return &Strategy{ticker: ticker, p: p}
}

// Ticker returns the bound instrument ticker.
func (s *Strategy) Ticker() string { return s.ticker }

// Lookback returns the candle window the strategy needs: the full level
// window plus ATR warm-up.
func (s *Strategy) Lookback() int { return levelWindowDays*barsPerDayMax + atrPeriod + 2 }

// mskLoc anchors all session logic to the Moscow trading calendar (UTC
// fallback if the tz DB is absent), mirroring the backtest engine.
var mskLoc = func() *time.Location {
	loc, err := time.LoadLocation("Europe/Moscow")
	if err != nil {
		return time.UTC
	}
	return loc
}()

// isWeekend reports whether t falls on Saturday or Sunday in mskLoc.
func isWeekend(t time.Time) bool {
	wd := t.In(mskLoc).Weekday()
	return wd == time.Saturday || wd == time.Sunday
}

// sameMSKDay reports whether a and b share an MSK calendar day.
func sameMSKDay(a, b time.Time) bool {
	al, bl := a.In(mskLoc), b.In(mskLoc)
	return al.Year() == bl.Year() && al.YearDay() == bl.YearDay()
}

// startOfMSKDay returns midnight of t's MSK calendar day.
func startOfMSKDay(t time.Time) time.Time {
	tl := t.In(mskLoc)
	return time.Date(tl.Year(), tl.Month(), tl.Day(), 0, 0, 0, 0, mskLoc)
}

// windowStart returns the index of the oldest bar belonging to the last
// levelWindowDays distinct MSK days of times. Levels whose swing bar is
// older are forgotten.
func windowStart(times []time.Time) int {
	days := 0
	for i := len(times) - 1; i >= 0; i-- {
		if i == len(times)-1 || !sameMSKDay(times[i], times[i+1]) {
			days++
			if days > levelWindowDays {
				return i + 1
			}
		}
	}
	return 0
}

// tradingDaysSince counts distinct MSK days among times strictly after
// entry's day. Only days visible in the window count — safe while
// MaxHoldDays*barsPerDayMax stays well inside Lookback.
func tradingDaysSince(times []time.Time, entry time.Time) int {
	entryDay := startOfMSKDay(entry)
	days := 0
	var lastDay time.Time
	for _, t := range times {
		d := startOfMSKDay(t)
		if !d.After(entryDay) || d.Equal(lastDay) {
			continue
		}
		days++
		lastDay = d
	}
	return days
}
```

- [ ] **Step 4: Тесты зелёные**

Run: `go test ./internal/service/trading_strategy/smc/... -v`
Expected: PASS оба теста.

- [ ] **Step 5: Commit**

```bash
git add internal/service/trading_strategy/smc/
git commit -m "feat(smc): core skeleton — params, MSK calendar helpers, level window"
```

---

### Task 3: уровни — фрактальные swing-low и жизненный цикл sweep

**Files:**
- Modify: `internal/service/trading_strategy/smc/strategy/core/core.go`
- Test: `internal/service/trading_strategy/smc/strategy/core/core_test.go`

**Interfaces:**
- Consumes: `windowStart` (Task 2).
- Produces: `type level struct{price float64; barIdx, confirmIdx, pierceIdx, reclaimIdx int; sweepLow float64}`; `levelStates(lows, closes []float64, times []time.Time, k int) []level`; `reclaimCandidate(levels []level, cur, maxBars int) (level, bool)`. Task 5 строит вход из candidate.

- [ ] **Step 1: Написать падающие тесты**

Добавить в `core_test.go`:

```go
func TestSwingLowConfirmedOnlyAfterK(t *testing.T) {
	base := flatBars(msk(2026, 7, 6, 10), 6, 100)
	dip := bar{t: next(base), h: 101, l: 97, c: 100, v: 100}
	bars := append(append([]bar{}, base...), dip)
	// 1 бар после дна (k=2): уровень ещё не подтверждён.
	bars = append(bars, bar{t: next(bars), h: 100.6, l: 97.9, c: 100.3, v: 100})
	md := mkMD(bars, nil)
	if lvls := levelStates(md.Lows, md.Closes, md.Times, 2); len(lvls) != 0 {
		t.Fatalf("levels before confirmation = %d, want 0", len(lvls))
	}
	// 2 бара после дна: уровень 97 подтверждён.
	bars = append(bars, bar{t: next(bars), h: 100.5, l: 97.8, c: 100.1, v: 100})
	md = mkMD(bars, nil)
	lvls := levelStates(md.Lows, md.Closes, md.Times, 2)
	if len(lvls) != 1 || lvls[0].price != 97 {
		t.Fatalf("levels = %+v, want one level at 97", lvls)
	}
	if lvls[0].pierceIdx != -1 || lvls[0].reclaimIdx != -1 {
		t.Fatalf("fresh level must be untouched, got %+v", lvls[0])
	}
}

func TestSweepReclaimLifecycle(t *testing.T) {
	base := flatBars(msk(2026, 7, 6, 10), 6, 100)
	bars := append(append([]bar{}, base...),
		bar{t: next(base), h: 101, l: 97, c: 100, v: 100}, // swing low 97
	)
	bars = append(bars, bar{t: next(bars), h: 100.6, l: 97.9, c: 100.3, v: 100})
	bars = append(bars, bar{t: next(bars), h: 100.5, l: 97.8, c: 100.1, v: 100}) // confirm
	bars = append(bars, bar{t: next(bars), h: 99, l: 96.5, c: 96.9, v: 100})     // pierce, close under
	md := mkMD(bars, nil)
	lvls := levelStates(md.Lows, md.Closes, md.Times, 2)
	if len(lvls) != 1 || lvls[0].pierceIdx != len(bars)-1 || lvls[0].reclaimIdx != -1 {
		t.Fatalf("after pierce: %+v", lvls)
	}
	bars = append(bars, bar{t: next(bars), h: 99.4, l: 97.7, c: 98.5, v: 100}) // reclaim close
	md = mkMD(bars, nil)
	lvls = levelStates(md.Lows, md.Closes, md.Times, 2)
	lv := lvls[0]
	if lv.reclaimIdx != len(bars)-1 {
		t.Fatalf("reclaimIdx = %d, want %d", lv.reclaimIdx, len(bars)-1)
	}
	if lv.sweepLow != 96.5 {
		t.Fatalf("sweepLow = %v, want 96.5", lv.sweepLow)
	}
	// Однобарный sweep: прокол и reclaim одной свечой.
	sb := append(append([]bar{}, base...),
		bar{t: next(base), h: 101, l: 97, c: 100, v: 100},
	)
	sb = append(sb, bar{t: next(sb), h: 100.6, l: 97.9, c: 100.3, v: 100})
	sb = append(sb, bar{t: next(sb), h: 100.5, l: 97.8, c: 100.1, v: 100})
	sb = append(sb, bar{t: next(sb), h: 99, l: 96.5, c: 98.2, v: 100}) // wick under, close above
	md = mkMD(sb, nil)
	lv = levelStates(md.Lows, md.Closes, md.Times, 2)[0]
	if lv.pierceIdx != lv.reclaimIdx || lv.reclaimIdx != len(sb)-1 {
		t.Fatalf("same-bar sweep: %+v", lv)
	}
}

func TestReclaimCandidateWindowAndDepth(t *testing.T) {
	lvls := []level{
		{price: 97, pierceIdx: 10, reclaimIdx: 12, sweepLow: 96.5},
		{price: 96, pierceIdx: 11, reclaimIdx: 12, sweepLow: 95.8},
	}
	// Оба reclaim-ятся баром 12 — берём с более глубоким sweepLow.
	cand, ok := reclaimCandidate(lvls, 12, 4)
	if !ok || cand.sweepLow != 95.8 {
		t.Fatalf("cand = %+v ok=%v, want deepest sweepLow 95.8", cand, ok)
	}
	// Просроченный reclaim (gap 2 > maxBars 1) не кандидат.
	if _, ok := reclaimCandidate(lvls[:1], 12, 1); ok {
		t.Fatal("stale reclaim (gap 2 > 1) must not be a candidate")
	}
	// Текущий бар не совпадает с reclaim — не кандидат.
	if _, ok := reclaimCandidate(lvls, 13, 4); ok {
		t.Fatal("reclaim on an earlier bar must not be a candidate")
	}
}
```

- [ ] **Step 2: Убедиться, что тесты падают**

Run: `go test ./internal/service/trading_strategy/smc/... -run 'TestSwing|TestSweep|TestReclaim' -v`
Expected: FAIL — `undefined: levelStates`, `level`, `reclaimCandidate`.

- [ ] **Step 3: Реализация (добавить в core.go)**

```go
// level is one confirmed swing-low and its sweep lifecycle inside the window.
type level struct {
	price      float64 // the swing-low value — the liquidity line
	barIdx     int     // bar of the swing-low extreme
	confirmIdx int     // barIdx + SwingK: first bar the level is visible on (anti-lookahead)
	pierceIdx  int     // first bar > confirmIdx with Low < price; -1 while untouched
	reclaimIdx int     // first bar >= pierceIdx with Close > price; -1 while under water
	sweepLow   float64 // lowest Low from pierce through reclaim (or through the window end)
}

// levelStates finds confirmed fractal swing-lows inside the level window and
// classifies each one's sweep lifecycle. Bars within SwingK of the extreme
// cannot pierce it by construction (their lows are strictly higher), so the
// pierce scan starts after confirmation. A level is consumed by its FIRST
// reclaim; later sweeps of the same line never signal again.
func levelStates(lows, closes []float64, times []time.Time, k int) []level {
	n := len(lows)
	start := windowStart(times)
	var out []level
	for i := max(k, start); i+k < n; i++ {
		swing := true
		for d := 1; d <= k; d++ {
			if lows[i] >= lows[i-d] || lows[i] >= lows[i+d] {
				swing = false
				break
			}
		}
		if !swing {
			continue
		}
		lv := level{price: lows[i], barIdx: i, confirmIdx: i + k, pierceIdx: -1, reclaimIdx: -1}
		for j := lv.confirmIdx + 1; j < n; j++ {
			if lows[j] < lv.price {
				lv.pierceIdx = j
				break
			}
		}
		if lv.pierceIdx >= 0 {
			lv.sweepLow = lows[lv.pierceIdx]
			for j := lv.pierceIdx; j < n; j++ {
				lv.sweepLow = min(lv.sweepLow, lows[j])
				if closes[j] > lv.price {
					lv.reclaimIdx = j
					break
				}
			}
		}
		out = append(out, lv)
	}
	return out
}

// reclaimCandidate returns the level whose FIRST reclaim lands exactly on bar
// cur within maxBars of its pierce. When several levels reclaim on the same
// bar the deepest sweepLow wins — its stop covers the others.
func reclaimCandidate(levels []level, cur, maxBars int) (level, bool) {
	var best level
	found := false
	for _, lv := range levels {
		if lv.pierceIdx < 0 || lv.reclaimIdx != cur || lv.reclaimIdx-lv.pierceIdx > maxBars {
			continue
		}
		if !found || lv.sweepLow < best.sweepLow {
			best = lv
			found = true
		}
	}
	return best, found
}
```

- [ ] **Step 4: Тесты зелёные**

Run: `go test ./internal/service/trading_strategy/smc/... -count=1`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/service/trading_strategy/smc/
git commit -m "feat(smc): level detection — confirmed fractal swing-lows and sweep/reclaim lifecycle"
```

---

### Task 4: manage-pass — интрабарный стоп, RR-тейк, тайм-стоп

**Files:**
- Modify: `internal/service/trading_strategy/smc/strategy/core/core.go`
- Test: `internal/service/trading_strategy/smc/strategy/core/core_test.go`

**Interfaces:**
- Consumes: `Position.EntryTime` (Task 1), `tradingDaysSince` (Task 2).
- Produces: `(s *Strategy) Decide(md strategy.MarketData) model.Signal` (пока: позиция → manage, flat → SignalNone); `manage(md, sig) model.Signal`; `takeProfit(pos *strategy.Position, tpr float64) (float64, bool)`. Reason-коды: `"SL"` (стоп, уже в `model.IsStopReason`), `"TP"`, `"TIME"` (fill по close — в `IsStopReason` НЕ добавлять).

- [ ] **Step 1: Написать падающие тесты**

Добавить в `core_test.go` (импортировать `"tinvest/internal/service/trading_strategy/scalping/model"`):

```go
// heldMD — 2 торговых дня флэта + текущий бар с заданными H/L; позиция
// открыта в первый день по 100 со стопом 95.
func heldMD(h, l, c float64) strategy.MarketData {
	bars := flatBars(msk(2026, 7, 6, 10), 16, 100)
	bars = append(bars, bar{t: next(bars), h: h, l: l, c: c, v: 100})
	return mkMD(bars, &strategy.Position{
		PurchasePrice: 100, Quantity: 1, StopLoss: 95,
		EntryTime: msk(2026, 7, 6, 11),
	})
}

func newStrat(p Params) *Strategy { return NewWithParams("TEST", p) }

func TestManageStopBeforeTP(t *testing.T) {
	s := newStrat(Params{SwingK: 2, ReclaimBars: 4, TPR: 2, MaxHoldDays: 5})
	// TP = 100 + 2*(100-95) = 110; бар задевает и стоп, и тейк — стоп первым.
	sig := s.Decide(heldMD(111, 94, 100))
	if sig.Kind != model.SignalSell || sig.Reason != "SL" {
		t.Fatalf("sig = %+v, want Sell/SL", sig)
	}
	if sig.StopLoss != 95 {
		t.Fatalf("StopLoss = %v, want 95", sig.StopLoss)
	}
}

func TestManageTP(t *testing.T) {
	s := newStrat(Params{SwingK: 2, ReclaimBars: 4, TPR: 2, MaxHoldDays: 5})
	sig := s.Decide(heldMD(110.5, 99, 109))
	if sig.Kind != model.SignalSell || sig.Reason != "TP" || sig.TakeProfit != 110 {
		t.Fatalf("sig = %+v, want Sell/TP@110", sig)
	}
	// TPR <= 0 выключает тейк.
	s = newStrat(Params{SwingK: 2, ReclaimBars: 4, TPR: 0, MaxHoldDays: 5})
	if sig := s.Decide(heldMD(120, 99, 119)); sig.Kind != model.SignalNone {
		t.Fatalf("TPR=0: sig = %+v, want None", sig)
	}
}

func TestManageTimeStop(t *testing.T) {
	s := newStrat(Params{SwingK: 2, ReclaimBars: 4, TPR: 2, MaxHoldDays: 2})
	// heldMD: вход Пн, текущий бар Ср → 2 полных торговых дня после входа.
	sig := s.Decide(heldMD(101, 99, 100))
	if sig.Kind != model.SignalSell || sig.Reason != "TIME" {
		t.Fatalf("sig = %+v, want Sell/TIME", sig)
	}
	// Без EntryTime тайм-стоп деградирует в no-op.
	md := heldMD(101, 99, 100)
	md.Position.EntryTime = time.Time{}
	if sig := s.Decide(md); sig.Kind != model.SignalNone {
		t.Fatalf("zero EntryTime: sig = %+v, want None", sig)
	}
	// MaxHoldDays=5 ещё не истёк.
	s = newStrat(Params{SwingK: 2, ReclaimBars: 4, TPR: 2, MaxHoldDays: 5})
	if sig := s.Decide(heldMD(101, 99, 100)); sig.Kind != model.SignalNone {
		t.Fatalf("not expired: sig = %+v, want None", sig)
	}
}
```

- [ ] **Step 2: Убедиться, что тесты падают**

Run: `go test ./internal/service/trading_strategy/smc/... -run TestManage -v`
Expected: FAIL — `s.Decide undefined`.

- [ ] **Step 3: Реализация (добавить в core.go; импорты: `fmt`, model, strategy)**

```go
// Decide is pure: it computes everything from md and performs no I/O. Times
// are mandatory; when Times is missing or misaligned the strategy is a
// deliberate no-op (can't see the calendar — don't trade).
func (s *Strategy) Decide(md strategy.MarketData) model.Signal {
	sig := model.Signal{Kind: model.SignalNone, Ticker: s.ticker, Price: md.Price}
	n := len(md.Closes)
	if n == 0 || len(md.Times) != n || len(md.Highs) != n || len(md.Lows) != n {
		return sig
	}
	if md.Position != nil {
		return s.manage(md, sig)
	}
	return sig // entryCheck подключается в Task 5
}

// manage handles an open position: the intrabar hard stop first (it fires
// earlier in real time), then the take-profit, then the trading-day
// time-stop (close-fill).
func (s *Strategy) manage(md strategy.MarketData, sig model.Signal) model.Signal {
	n := len(md.Closes)
	pos := md.Position
	if pos.StopLoss > 0 && md.Lows[n-1] <= pos.StopLoss {
		sig.Kind = model.SignalSell
		sig.Reason = "SL"
		sig.StopLoss = pos.StopLoss
		sig.ExitReason = fmt.Sprintf("интрабарный стоп %.4f (low бара %.4f)", pos.StopLoss, md.Lows[n-1])
		return sig
	}
	if tp, ok := takeProfit(pos, s.p.TPR); ok && md.Highs[n-1] >= tp {
		sig.Kind = model.SignalSell
		sig.Reason = "TP"
		sig.TakeProfit = tp
		sig.ExitReason = fmt.Sprintf("тейк-профит %.4f (high бара %.4f)", tp, md.Highs[n-1])
		return sig
	}
	if s.p.MaxHoldDays > 0 && !pos.EntryTime.IsZero() &&
		tradingDaysSince(md.Times, pos.EntryTime) >= s.p.MaxHoldDays {
		sig.Kind = model.SignalSell
		sig.Reason = "TIME"
		sig.ExitReason = fmt.Sprintf("тайм-стоп: позиция старше %d торговых дней", s.p.MaxHoldDays)
		return sig
	}
	return sig
}

// takeProfit derives the frozen TP from the position itself (stateless): the
// entry stop distance times TPR. ok=false when TPR is off or the stop is
// missing/inverted — then no TP exit exists.
func takeProfit(pos *strategy.Position, tpr float64) (float64, bool) {
	if tpr <= 0 || pos.StopLoss <= 0 || pos.PurchasePrice <= pos.StopLoss {
		return 0, false
	}
	return pos.PurchasePrice + tpr*(pos.PurchasePrice-pos.StopLoss), true
}
```

- [ ] **Step 4: Тесты зелёные**

Run: `go test ./internal/service/trading_strategy/smc/... -count=1`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/service/trading_strategy/smc/
git commit -m "feat(smc): manage pass — intrabar stop, RR take-profit, trading-day time-stop"
```

---

### Task 5: entry-pass — сигнал sweep+reclaim со структурным стопом

**Files:**
- Modify: `internal/service/trading_strategy/smc/strategy/core/core.go`
- Test: `internal/service/trading_strategy/smc/strategy/core/core_test.go`

**Interfaces:**
- Consumes: `levelStates`, `reclaimCandidate` (Task 3), `indicators.ATR`.
- Produces: `entryCheck(md strategy.MarketData, sig model.Signal) (model.Signal, string)` — второй результат — человекочитаемая причина отказа (для Explain, Task 7); Decide вызывает его при flat. На Buy: `StopLoss`, `TakeProfit`, `ATR`, `Level`, `EntryReason` заполнены. Тестовый хелпер `sweepScenario() []bar` (переиспользуют Tasks 6–7).

- [ ] **Step 1: Написать падающие тесты**

Добавить в `core_test.go` (импортировать `"tinvest/pkg/indicators"`):

```go
// sweepScenario — канонический валидный сетап (k=2): ATR-прогрев (2 дня
// флэта), подтверждённый swing-low 97, прокол до 96.5 и reclaim-close 98.5
// последним баром (среда, основная сессия).
func sweepScenario() []bar {
	bars := flatBars(msk(2026, 7, 6, 10), 16, 100)
	bars = append(bars, bar{t: next(bars), h: 101, l: 97, c: 100, v: 100})      // swing low 97
	bars = append(bars, bar{t: next(bars), h: 100.6, l: 97.9, c: 100.3, v: 100})
	bars = append(bars, bar{t: next(bars), h: 100.5, l: 97.8, c: 100.1, v: 100}) // confirm
	bars = append(bars, bar{t: next(bars), h: 99, l: 96.5, c: 96.9, v: 100})     // pierce
	bars = append(bars, bar{t: next(bars), h: 99.4, l: 97.7, c: 98.5, v: 100})   // reclaim
	return bars
}

func defParams() Params {
	return Params{SwingK: 2, ReclaimBars: 4, Buffer: 0.5, TPR: 2, MaxHoldDays: 3}
}

func TestEntryOnSweepReclaim(t *testing.T) {
	md := mkMD(sweepScenario(), nil)
	sig := newStrat(defParams()).Decide(md)
	if sig.Kind != model.SignalBuy {
		t.Fatalf("sig = %+v, want Buy", sig)
	}
	atr := indicators.ATR(md.Highs, md.Lows, md.Closes, 14)
	wantStop := 96.5 - 0.5*atr
	if sig.StopLoss != wantStop {
		t.Fatalf("StopLoss = %v, want %v", sig.StopLoss, wantStop)
	}
	if sig.Level != 97 || sig.ATR != atr {
		t.Fatalf("Level/ATR = %v/%v, want 97/%v", sig.Level, sig.ATR, atr)
	}
	wantTP := 98.5 + 2*(98.5-wantStop)
	if sig.TakeProfit != wantTP {
		t.Fatalf("TakeProfit = %v, want %v", sig.TakeProfit, wantTP)
	}
	if sig.EntryReason == "" {
		t.Fatal("EntryReason must be set on Buy")
	}
}

func TestNoEntryWhenReclaimTooLate(t *testing.T) {
	p := defParams()
	p.ReclaimBars = 0 // только однобарный sweep; в сценарии gap = 1
	if sig := newStrat(p).Decide(mkMD(sweepScenario(), nil)); sig.Kind != model.SignalNone {
		t.Fatalf("sig = %+v, want None", sig)
	}
}

func TestNoEntryWhenReclaimOnEarlierBar(t *testing.T) {
	bars := sweepScenario()
	bars = append(bars, bar{t: next(bars), h: 101, l: 99, c: 100, v: 100})
	if sig := newStrat(defParams()).Decide(mkMD(bars, nil)); sig.Kind != model.SignalNone {
		t.Fatalf("sig = %+v, want None (reclaim был баром раньше)", sig)
	}
}

func TestNoEntryOutsideMainSession(t *testing.T) {
	// Reclaim-бар в вечернюю сессию (19:00 MSK того же дня).
	bars := sweepScenario()
	ev := bars[len(bars)-1].t.In(mskLoc)
	bars[len(bars)-1].t = time.Date(ev.Year(), ev.Month(), ev.Day(), 19, 0, 0, 0, mskLoc)
	if sig := newStrat(defParams()).Decide(mkMD(bars, nil)); sig.Kind != model.SignalNone {
		t.Fatalf("evening bar: sig = %+v, want None", sig)
	}
	// Reclaim-бар в субботу.
	bars = sweepScenario()
	sat := startOfMSKDay(bars[len(bars)-1].t)
	for sat.Weekday() != time.Saturday {
		sat = sat.AddDate(0, 0, 1)
	}
	bars[len(bars)-1].t = sat.Add(12 * time.Hour)
	if sig := newStrat(defParams()).Decide(mkMD(bars, nil)); sig.Kind != model.SignalNone {
		t.Fatalf("saturday bar: sig = %+v, want None", sig)
	}
}
```

- [ ] **Step 2: Убедиться, что тесты падают**

Run: `go test ./internal/service/trading_strategy/smc/... -run TestEntry -v` и `-run TestNoEntry -v`
Expected: FAIL — Buy не генерируется (Decide пока возвращает None при flat).

- [ ] **Step 3: Реализация**

В `Decide` заменить `return sig // entryCheck подключается в Task 5` на:

```go
	sig, _ = s.entryCheck(md, sig)
	return sig
```

Добавить (импортировать `"tinvest/pkg/indicators"`):

```go
// entryCheck runs the full entry pipeline. On rejection the second result
// holds a human-readable reason (consumed by Explain). On success
// sig.Kind == model.SignalBuy with the frozen structural stop in sig.StopLoss.
func (s *Strategy) entryCheck(md strategy.MarketData, sig model.Signal) (model.Signal, string) {
	n := len(md.Closes)
	t := md.Times[n-1].In(mskLoc)
	if isWeekend(t) {
		return sig, "выходной (Сб/Вс MSK) — входы запрещены"
	}
	if h := t.Hour(); h < sessionOpenHour {
		return sig, fmt.Sprintf("час бара %d < %d MSK — утренняя сессия без входов", h, sessionOpenHour)
	} else if h >= eveningStartHour {
		return sig, fmt.Sprintf("час бара %d ≥ %d MSK — вечерняя сессия без входов", h, eveningStartHour)
	}
	if s.p.SwingK <= 0 {
		return sig, "SwingK ≤ 0 — вход выключен"
	}
	levels := levelStates(md.Lows, md.Closes, md.Times, s.p.SwingK)
	cand, ok := reclaimCandidate(levels, n-1, s.p.ReclaimBars)
	if !ok {
		return sig, "текущий бар не reclaim-ит ни один уровень в окне ReclaimBars"
	}
	if why, ok := s.passFilters(md, cand); !ok { // no-op до Task 6
		return sig, why
	}
	atr := indicators.ATR(md.Highs, md.Lows, md.Closes, atrPeriod)
	if atr <= 0 {
		return sig, "ATR не прогрет — не с чего считать стоп"
	}
	stop := cand.sweepLow - s.p.Buffer*atr
	sig.Kind = model.SignalBuy
	sig.StopLoss = stop
	sig.ATR = atr
	sig.Level = cand.price
	if s.p.TPR > 0 {
		sig.TakeProfit = md.Price + s.p.TPR*(md.Price-stop)
	}
	sig.EntryReason = fmt.Sprintf(
		"sweep-и-reclaim уровня %.4f: прокол до %.4f (%d бар(ов) от прокола), close %.4f выше уровня; стоп %.4f",
		cand.price, cand.sweepLow, cand.reclaimIdx-cand.pierceIdx, md.Price, stop)
	return sig, ""
}

// passFilters applies the optional SMC filters to a reclaim candidate.
// Implemented in the filters task; the core entry is filter-free.
func (s *Strategy) passFilters(_ strategy.MarketData, _ level) (string, bool) {
	return "", true
}
```

- [ ] **Step 4: Тесты зелёные**

Run: `go test ./internal/service/trading_strategy/smc/... -count=1`
Expected: PASS все.

- [ ] **Step 5: Commit**

```bash
git add internal/service/trading_strategy/smc/
git commit -m "feat(smc): entry — sweep+reclaim signal with structural stop and RR target"
```

---

### Task 6: фильтры UseFVG / UseDiscount / UseOB

**Files:**
- Modify: `internal/service/trading_strategy/smc/strategy/core/core.go`
- Test: `internal/service/trading_strategy/smc/strategy/core/core_test.go`

**Interfaces:**
- Consumes: `sweepScenario`, `entryCheck`/`passFilters` (Task 5), `windowStart` (Task 2).
- Produces: реальный `passFilters`; `hasBullishFVG(highs, lows []float64, pierce, reclaim int) bool`; `inDiscount(highs, lows []float64, start int, close float64) bool`; `bullishOBZones(highs, lows, closes []float64, times []time.Time, k int) [][2]float64`; `inOBZone(zones [][2]float64, price float64) bool`.

- [ ] **Step 1: Написать падающие тесты**

```go
func TestFVGFilter(t *testing.T) {
	// База: между проколом (high 99) и reclaim (low 97.7) разрыва нет — режем.
	p := defParams()
	p.UseFVG = 1
	if sig := newStrat(p).Decide(mkMD(sweepScenario(), nil)); sig.Kind != model.SignalNone {
		t.Fatalf("no FVG: sig = %+v, want None", sig)
	}
	// Вариант с разрывом: X(h97.6) → A(прокол) → R(low 97.7 > 97.6) — FVG есть.
	bars := flatBars(msk(2026, 7, 6, 10), 16, 100)
	bars = append(bars, bar{t: next(bars), h: 101, l: 97, c: 100, v: 100})
	bars = append(bars, bar{t: next(bars), h: 100.6, l: 97.9, c: 100.3, v: 100})
	bars = append(bars, bar{t: next(bars), h: 97.6, l: 97.1, c: 97.3, v: 100})  // X — confirm + сжатие
	bars = append(bars, bar{t: next(bars), h: 97.4, l: 96.5, c: 96.9, v: 100})  // A — прокол
	bars = append(bars, bar{t: next(bars), h: 99, l: 97.7, c: 98.5, v: 100})    // R — reclaim, FVG
	if sig := newStrat(p).Decide(mkMD(bars, nil)); sig.Kind != model.SignalBuy {
		t.Fatalf("with FVG: sig = %+v, want Buy", sig)
	}
}

func TestDiscountFilter(t *testing.T) {
	// База: окно [96.5..101], mid 98.75, вход 98.5 < mid — проходит.
	p := defParams()
	p.UseDiscount = 1
	if sig := newStrat(p).Decide(mkMD(sweepScenario(), nil)); sig.Kind != model.SignalBuy {
		t.Fatalf("discount entry: sig = %+v, want Buy", sig)
	}
	// Вход 99.2 > mid — режем.
	bars := sweepScenario()
	bars[len(bars)-1].c = 99.2
	md := mkMD(bars, nil)
	if sig := newStrat(p).Decide(md); sig.Kind != model.SignalNone {
		t.Fatalf("premium entry: sig = %+v, want None", sig)
	}
}

// obScenario — сетап с непогашенным бычьим OB [96, 100.8]: swing-high 103
// пробит закрытием 103.5, последняя down-close свеча перед пробоем — зона.
func obScenario() []bar {
	bars := flatBars(msk(2026, 7, 6, 10), 16, 100)
	bars = append(bars, bar{t: next(bars), h: 103, l: 99.5, c: 102, v: 100})     // H — swing high
	bars = append(bars, bar{t: next(bars), h: 101, l: 99, c: 100.4, v: 100})     // S1
	bars = append(bars, bar{t: next(bars), h: 101, l: 99, c: 100.2, v: 100})     // S2 — confirm H
	bars = append(bars, bar{t: next(bars), h: 100.8, l: 96, c: 99.8, v: 100})    // O — down-close, зона [96,100.8]
	bars = append(bars, bar{t: next(bars), h: 104, l: 99.9, c: 103.5, v: 100})   // B1 — пробой 103
	bars = append(bars, bar{t: next(bars), h: 101, l: 99, c: 100.5, v: 100})     // F1
	bars = append(bars, bar{t: next(bars), h: 101, l: 99, c: 100.2, v: 100})     // F2
	bars = append(bars, bar{t: next(bars), h: 101, l: 97, c: 100, v: 100})       // D — swing low 97
	bars = append(bars, bar{t: next(bars), h: 100.6, l: 97.9, c: 100.3, v: 100}) // D1
	bars = append(bars, bar{t: next(bars), h: 100.5, l: 97.8, c: 100.1, v: 100}) // D2 — confirm D
	bars = append(bars, bar{t: next(bars), h: 99, l: 96.5, c: 96.9, v: 100})     // A — прокол
	bars = append(bars, bar{t: next(bars), h: 99.4, l: 97.7, c: 98.5, v: 100})   // R — reclaim ∈ зоны
	return bars
}

func TestOBFilter(t *testing.T) {
	p := defParams()
	p.UseOB = 1
	// База: ни один swing-high не пробит — зон нет, режем.
	if sig := newStrat(p).Decide(mkMD(sweepScenario(), nil)); sig.Kind != model.SignalNone {
		t.Fatalf("no OB: sig = %+v, want None", sig)
	}
	// Сетап с зоной: вход 98.5 ∈ [96, 100.8] — проходит.
	if sig := newStrat(p).Decide(mkMD(obScenario(), nil)); sig.Kind != model.SignalBuy {
		t.Fatalf("with OB: sig = %+v, want Buy", sig)
	}
}
```

- [ ] **Step 2: Убедиться, что тесты падают**

Run: `go test ./internal/service/trading_strategy/smc/... -run 'TestFVG|TestDiscount|TestOB' -v`
Expected: FAIL — фильтры пока no-op, «режущие» ассерты получают Buy.

- [ ] **Step 3: Реализация — заменить заглушку passFilters и добавить хелперы**

```go
// passFilters applies the optional SMC filters to a reclaim candidate. Each
// filter is a pure predicate over the same window; the first failing one
// names itself in the rejection reason.
func (s *Strategy) passFilters(md strategy.MarketData, cand level) (string, bool) {
	if s.p.UseFVG == 1 && !hasBullishFVG(md.Highs, md.Lows, cand.pierceIdx, cand.reclaimIdx) {
		return "фильтр FVG: между проколом и reclaim нет бычьего разрыва", false
	}
	if s.p.UseDiscount == 1 && !inDiscount(md.Highs, md.Lows, windowStart(md.Times), md.Price) {
		return "фильтр discount: вход выше середины диапазона окна", false
	}
	if s.p.UseOB == 1 {
		zones := bullishOBZones(md.Highs, md.Lows, md.Closes, md.Times, s.p.SwingK)
		if !inOBZone(zones, md.Price) {
			return "фильтр OB: close входа вне непогашенных бычьих order block", false
		}
	}
	return "", true
}

// hasBullishFVG reports a three-bar bullish imbalance (Low[i+1] > High[i-1])
// whose middle bar lies in [pierce, reclaim-1] — the pattern completes no
// later than the reclaim bar. A same-bar sweep has no room for displacement
// and always fails this filter.
func hasBullishFVG(highs, lows []float64, pierce, reclaim int) bool {
	for i := max(pierce, 1); i <= reclaim-1 && i+1 < len(lows); i++ {
		if lows[i+1] > highs[i-1] {
			return true
		}
	}
	return false
}

// inDiscount reports whether close sits below the midpoint of the
// level-window price range — the "buy cheap" half.
func inDiscount(highs, lows []float64, start int, close float64) bool {
	hi, lo := highs[start], lows[start]
	for i := start + 1; i < len(highs); i++ {
		hi = max(hi, highs[i])
		lo = min(lo, lows[i])
	}
	return close < (hi+lo)/2
}

// bullishOBZones returns unmitigated bullish order blocks in the level
// window: for each confirmed fractal swing-high later broken by a close, the
// last down-close bar (Close[o] < Close[o-1] — MarketData has no Open) before
// the breaking bar forms the zone [low, high]; any later close below the zone
// low mitigates (removes) it.
func bullishOBZones(highs, lows, closes []float64, times []time.Time, k int) [][2]float64 {
	n := len(closes)
	start := windowStart(times)
	var zones [][2]float64
	for i := max(k, start); i+k < n; i++ {
		swing := true
		for d := 1; d <= k; d++ {
			if highs[i] <= highs[i-d] || highs[i] <= highs[i+d] {
				swing = false
				break
			}
		}
		if !swing {
			continue
		}
		for b := i + k + 1; b < n; b++ {
			if closes[b] <= highs[i] {
				continue
			}
			for o := b - 1; o > 0; o-- {
				if closes[o] >= closes[o-1] {
					continue
				}
				mitigated := false
				for m := o + 1; m < n; m++ {
					if closes[m] < lows[o] {
						mitigated = true
						break
					}
				}
				if !mitigated {
					zones = append(zones, [2]float64{lows[o], highs[o]})
				}
				break
			}
			break
		}
	}
	return zones
}

// inOBZone reports whether price falls inside any zone (inclusive).
func inOBZone(zones [][2]float64, price float64) bool {
	for _, z := range zones {
		if price >= z[0] && price <= z[1] {
			return true
		}
	}
	return false
}
```

- [ ] **Step 4: Тесты зелёные (все, включая прежние)**

Run: `go test ./internal/service/trading_strategy/smc/... -count=1`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/service/trading_strategy/smc/
git commit -m "feat(smc): optional filters — bullish FVG, discount half, unmitigated order block"
```

---

### Task 7: Explain — пер-барная диагностика

**Files:**
- Modify: `internal/service/trading_strategy/smc/strategy/core/core.go`
- Test: `internal/service/trading_strategy/smc/strategy/core/core_test.go`

**Interfaces:**
- Consumes: `entryCheck` (возвращает причину отказа), `manage`, `levelStates`, `takeProfit`.
- Produces: `(s *Strategy) Explain(md strategy.MarketData) string` — сигнатура, которую type-assert'ит `domain.Trace` (интерфейс `explainer` в `internal/domain/backtest/engine.go`); `levelSummary(levels []level) string`.

- [ ] **Step 1: Написать падающий тест**

```go
func TestExplain(t *testing.T) {
	s := newStrat(defParams())
	// Вход.
	if got := s.Explain(mkMD(sweepScenario(), nil)); !strings.Contains(got, "ВХОД") {
		t.Fatalf("Explain(entry) = %q, want contains ВХОД", got)
	}
	// Отказ: показывает причину и сводку уровней.
	bars := sweepScenario()
	bars = append(bars, bar{t: next(bars), h: 101, l: 99, c: 100, v: 100})
	got := s.Explain(mkMD(bars, nil))
	if !strings.Contains(got, "входа нет") || !strings.Contains(got, "уровн") {
		t.Fatalf("Explain(reject) = %q", got)
	}
	// Позиция: показывает стоп и тейк.
	got = s.Explain(heldMD(101, 99, 100))
	if !strings.Contains(got, "удерживается") {
		t.Fatalf("Explain(hold) = %q", got)
	}
	// Выход.
	if got := s.Explain(heldMD(101, 94, 95)); !strings.Contains(got, "ВЫХОД SL") {
		t.Fatalf("Explain(exit) = %q", got)
	}
}
```

(Импортировать `"strings"` в тестах.)

- [ ] **Step 2: Убедиться, что тест падает**

Run: `go test ./internal/service/trading_strategy/smc/... -run TestExplain -v`
Expected: FAIL — `s.Explain undefined`.

- [ ] **Step 3: Реализация (добавить в core.go; импортировать `"strings"`)**

```go
// Explain renders a human-readable verdict for the LAST bar of md — why the
// strategy did or did not act. Wired into `-explain` via domain.Trace, which
// type-asserts exactly this signature.
func (s *Strategy) Explain(md strategy.MarketData) string {
	sig := model.Signal{Kind: model.SignalNone, Ticker: s.ticker, Price: md.Price}
	n := len(md.Closes)
	if n == 0 || len(md.Times) != n || len(md.Highs) != n || len(md.Lows) != n {
		return "SMC: нет данных или Times не заполнен/не выровнен — стратегия без временных меток не работает"
	}
	stamp := md.Times[n-1].In(mskLoc).Format("2006-01-02 15:04")
	if md.Position != nil {
		out := s.manage(md, sig)
		if out.Kind == model.SignalSell {
			return fmt.Sprintf("SMC %s %s: ВЫХОД %s — %s", s.ticker, stamp, out.Reason, out.ExitReason)
		}
		tp, _ := takeProfit(md.Position, s.p.TPR)
		return fmt.Sprintf("SMC %s %s: позиция удерживается (стоп %.4f, тейк %.4f, low бара %.4f)",
			s.ticker, stamp, md.Position.StopLoss, tp, md.Lows[n-1])
	}
	out, why := s.entryCheck(md, sig)
	if out.Kind == model.SignalBuy {
		return fmt.Sprintf("SMC %s %s: ВХОД — %s", s.ticker, stamp, out.EntryReason)
	}
	levels := levelStates(md.Lows, md.Closes, md.Times, s.p.SwingK)
	return fmt.Sprintf("SMC %s %s: входа нет — %s; уровней в окне: %d (%s)",
		s.ticker, stamp, why, len(levels), levelSummary(levels))
}

// levelSummary renders a compact per-level status line for Explain.
func levelSummary(levels []level) string {
	if len(levels) == 0 {
		return "нет"
	}
	parts := make([]string, 0, len(levels))
	for _, lv := range levels {
		st := "активен"
		switch {
		case lv.reclaimIdx >= 0:
			st = fmt.Sprintf("reclaim на баре %d", lv.reclaimIdx)
		case lv.pierceIdx >= 0:
			st = fmt.Sprintf("в снятии с бара %d", lv.pierceIdx)
		}
		parts = append(parts, fmt.Sprintf("%.4f: %s", lv.price, st))
	}
	return strings.Join(parts, "; ")
}
```

- [ ] **Step 4: Тесты зелёные**

Run: `go test ./internal/service/trading_strategy/smc/... -count=1`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/service/trading_strategy/smc/
git commit -m "feat(smc): Explain — per-bar entry/exit diagnostics for -explain"
```

---

### Task 8: пер-тикерные пакеты, реестр, cmd/backtest, гриды

**Files:**
- Create: `internal/service/trading_strategy/smc/strategy/{afks,gazp,mdmg,nvtk,plzl,rusal,sber,ydex}/<pkg>.go` (8 файлов)
- Create: `internal/service/backtest/smc_registry.go`
- Test: `internal/service/backtest/smc_registry_test.go`
- Modify: `cmd/backtest/main.go` (флаг-usage `-strategy`, switch в `run`, doc-comment файла)
- Create: `data/params/{afks,gazp,mdmg,nvtk,plzl,rual,sber,ydex}/smc_grid.json` (8 файлов; заметь: папка RUAL — `rual`, пакет — `rusal`, как у ORB)

**Interfaces:**
- Consumes: `core.NewWithParams`, `core.Params` (Tasks 2–7); `backtest.Binding` (существует).
- Produces: `<pkg>.Ticker` (const string), `<pkg>.DefaultParams() core.Params`; `svc.SMCLookupOrGeneric(ticker string) Binding`; `-strategy smc` в CLI.

- [ ] **Step 1: Написать падающий тест реестра**

`internal/service/backtest/smc_registry_test.go`:

```go
package backtest

import (
	"testing"

	"tinvest/internal/service/trading_strategy/smc/strategy/core"
	smcsber "tinvest/internal/service/trading_strategy/smc/strategy/sber"
)

func TestSMCLookupOrGeneric(t *testing.T) {
	b := SMCLookupOrGeneric(smcsber.Ticker)
	if got := b.Build(b.DefaultParams()).Ticker(); got != "SBER" {
		t.Fatalf("Ticker = %q, want SBER", got)
	}
	if b.DefaultParams().(core.Params) != smcsber.DefaultParams() {
		t.Fatal("registered ticker must use its package defaults")
	}
	// Незнакомый тикер получает generic-байндинг, привязанный к нему.
	g := SMCLookupOrGeneric("XXXX")
	if got := g.Build(g.DefaultParams()).Ticker(); got != "XXXX" {
		t.Fatalf("generic Ticker = %q, want XXXX", got)
	}
	// Частичный JSON перекрывает только свои поля.
	p, err := b.ParseParams([]byte(`{"SwingK": 5}`))
	if err != nil {
		t.Fatal(err)
	}
	want := smcsber.DefaultParams()
	want.SwingK = 5
	if p.(core.Params) != want {
		t.Fatalf("ParseParams = %+v, want %+v", p, want)
	}
}
```

- [ ] **Step 2: Убедиться, что тест падает**

Run: `go test ./internal/service/backtest/ -run TestSMCLookup -v`
Expected: FAIL — пакеты и `SMCLookupOrGeneric` не существуют.

- [ ] **Step 3: Пер-тикерные пакеты**

8 одинаковых по форме файлов; образец `internal/service/trading_strategy/smc/strategy/sber/sber.go`:

```go
// Package sber supplies the ticker and starting SMC Params for SBER (Sberbank).
// Starting values mirror the generic defaults; calibrate with -calibrate and
// then hardcode the winning combination here.
package sber

import "tinvest/internal/service/trading_strategy/smc/strategy/core"

// Ticker is the instrument this package configures.
const Ticker = "SBER"

// DefaultParams returns SBER's starting SMC parameters (pre-calibration).
func DefaultParams() core.Params {
	return core.Params{SwingK: 3, ReclaimBars: 4, Buffer: 0.5, TPR: 2, MaxHoldDays: 3}
}
```

Остальные семь: `afks` (AFKS, АФК Система), `gazp` (GAZP, Газпром), `mdmg` (MDMG, MD Medical), `nvtk` (NVTK, Новатэк), `plzl` (PLZL, Полюс), `rusal` (RUAL, Русал — пакет `rusal`, тикер `RUAL`), `ydex` (YDEX, Яндекс) — те же дефолты, меняется только имя пакета/тикер/первая строка комментария.

- [ ] **Step 4: Реестр `internal/service/backtest/smc_registry.go`**

```go
package backtest

import (
	"encoding/json"
	"fmt"

	"tinvest/internal/service/trading_strategy/scalping/strategy"
	smcafks "tinvest/internal/service/trading_strategy/smc/strategy/afks"
	"tinvest/internal/service/trading_strategy/smc/strategy/core"
	smcgazp "tinvest/internal/service/trading_strategy/smc/strategy/gazp"
	smcmdmg "tinvest/internal/service/trading_strategy/smc/strategy/mdmg"
	smcnvtk "tinvest/internal/service/trading_strategy/smc/strategy/nvtk"
	smcplzl "tinvest/internal/service/trading_strategy/smc/strategy/plzl"
	smcrusal "tinvest/internal/service/trading_strategy/smc/strategy/rusal"
	smcsber "tinvest/internal/service/trading_strategy/smc/strategy/sber"
	smcydex "tinvest/internal/service/trading_strategy/smc/strategy/ydex"
)

// smcBindingFor builds a Binding for a ticker whose defaults come from defaults().
// All SMC tickers share the core engine; only ticker + defaults differ.
func smcBindingFor(ticker string, defaults func() core.Params) Binding {
	return Binding{
		DefaultParams: func() any { return defaults() },
		Build: func(params any) strategy.Strategy {
			return core.NewWithParams(ticker, params.(core.Params))
		},
		ParseParams: func(raw []byte) (any, error) {
			p := defaults() // start from defaults so partial JSON overrides
			if err := json.Unmarshal(raw, &p); err != nil {
				return nil, fmt.Errorf("backtest: parse params: %w", err)
			}
			return p, nil
		},
	}
}

var smcRegistry = map[string]Binding{
	smcsber.Ticker:  smcBindingFor(smcsber.Ticker, smcsber.DefaultParams),
	smcgazp.Ticker:  smcBindingFor(smcgazp.Ticker, smcgazp.DefaultParams),
	smcnvtk.Ticker:  smcBindingFor(smcnvtk.Ticker, smcnvtk.DefaultParams),
	smcplzl.Ticker:  smcBindingFor(smcplzl.Ticker, smcplzl.DefaultParams),
	smcydex.Ticker:  smcBindingFor(smcydex.Ticker, smcydex.DefaultParams),
	smcafks.Ticker:  smcBindingFor(smcafks.Ticker, smcafks.DefaultParams),
	smcrusal.Ticker: smcBindingFor(smcrusal.Ticker, smcrusal.DefaultParams),
	smcmdmg.Ticker:  smcBindingFor(smcmdmg.Ticker, smcmdmg.DefaultParams),
}

// genericSMCDefaults are neutral baseline params for tickers without a dedicated
// SMC config. Intentionally independent of any per-ticker defaults so calibrating
// one ticker never drifts the generic baseline.
func genericSMCDefaults() core.Params {
	return core.Params{SwingK: 3, ReclaimBars: 4, Buffer: 0.5, TPR: 2, MaxHoldDays: 3}
}

// SMCLookupOrGeneric returns the registered SMC binding for a ticker, or a
// generic binding bound to that ticker (with genericSMCDefaults) when none is
// registered.
func SMCLookupOrGeneric(ticker string) Binding {
	if b, ok := smcRegistry[ticker]; ok {
		return b
	}
	return smcBindingFor(ticker, genericSMCDefaults)
}
```

- [ ] **Step 5: Wiring в `cmd/backtest/main.go`**

Три правки (по образцу коммита ORB `e29708c`):
1. Doc-comment файла: `... (scalping, levels, momentum, reversion or smc) ...`.
2. Usage флага: `"strategy engine: scalping|levels|momentum|reversion|smc"`.
3. В `run`, в switch по `strategyName` добавить перед `case "scalping"`:

```go
	case "smc":
		binding = svc.SMCLookupOrGeneric(ticker)
```

и дополнить сообщение об ошибке default-ветки: `(want scalping|levels|momentum|reversion|smc)`.

- [ ] **Step 6: Гриды**

8 одинаковых файлов `data/params/<t>/smc_grid.json` (папки: afks, gazp, mdmg, nvtk, plzl, rual, sber, ydex):

```json
{
  "phases": [
    {
      "name": "core",
      "keepTop": 5,
      "grid": {
        "SwingK": [2, 3, 5],
        "ReclaimBars": [2, 4, 6],
        "Buffer": [0.25, 0.5, 1.0],
        "TPR": [1.5, 2, 3],
        "MaxHoldDays": [2, 3, 5]
      }
    },
    {
      "name": "filters",
      "grid": {
        "UseOB": [0, 1],
        "UseFVG": [0, 1],
        "UseDiscount": [0, 1]
      }
    }
  ]
}
```

- [ ] **Step 7: Тесты зелёные + сборка**

Run: `go test ./internal/service/backtest/ ./internal/service/trading_strategy/smc/... -count=1 && go build ./internal/... ./pkg/... ./cmd/...`
Expected: PASS, сборка чистая.

- [ ] **Step 8: Commit**

```bash
git add internal/service/trading_strategy/smc/ internal/service/backtest/smc_registry.go internal/service/backtest/smc_registry_test.go cmd/backtest/main.go data/params/
git commit -m "feat(smc): per-ticker packages, registry and cmd/backtest wiring + calibration grids"
```

---

### Task 9: документация + полный CI-гейт

**Files:**
- Create: `docs/smc/strategy.md`
- Modify: `CLAUDE.md` (список стратегий в Layout + пример команды)

**Interfaces:**
- Consumes: всё готовое.
- Produces: докуменация; зелёный `./bin/mage ci`.

- [ ] **Step 1: `docs/smc/strategy.md`**

Написать explainer по образцу `docs/orb/strategy.md` с ветки ORB (`git show feat/orb-intraday-breakout:docs/orb/strategy.md` — для стиля). Обязательные секции: Идея (стоп-хант / liquidity sweep — почему это может быть edge), Механика сигнала (уровни → sweep → reclaim, с числовым примером), Фильтры (OB/FVG/discount и их тумблеры), Выходы (SL/TP/TIME с приоритетом), Параметры и константы (таблица: имя, смысл, дефолт, значения грида), Калибровка и walk-forward (команда из спеки), Критерий судьбы (pooled OOS PF ≥ 1.5 — иначе хороним). Ссылка на спеку `docs/superpowers/specs/2026-07-17-smc-liquidity-sweep-design.md`.

- [ ] **Step 2: CLAUDE.md**

В строке Layout про `trading_strategy/` добавить `smc` в перечень backtest-only стратегий с припиской: `` `smc` — свинговый liquidity-sweep (Hour1), см. `docs/smc/strategy.md` ``. В конец файла добавить пример команды:

```
go run ./cmd/backtest -ticker SBER -strategy smc -interval Hour1 -months 24 -calibrate data/params/sber/smc_grid.json -train-months 12 -test-months 6 -min-trades 10 -metric profit_factor
```

- [ ] **Step 3: Полный гейт**

Run: `./bin/mage ci`
Expected: lint OK, все тесты PASS, mock-drift OK. Любое падение чинится до коммита.

- [ ] **Step 4: Commit**

```bash
git add docs/smc/ CLAUDE.md
git commit -m "docs(smc): strategy explainer + CLAUDE.md wiring"
```

---

## Валидация после реализации (запускает пользователь / отдельная сессия)

Пер-тикерный walk-forward (24 мес, Hour1, train 12 / test 6, min-trades 10):

```bash
for t in sber gazp nvtk plzl ydex afks rual mdmg; do
  T=$(echo "$t" | tr a-z A-Z); [ "$t" = rual ] && T=RUAL
  go run ./cmd/backtest -ticker "$T" -strategy smc -interval Hour1 -months 24 \
    -calibrate "data/params/$t/smc_grid.json" -train-months 12 -test-months 6 \
    -min-trades 10 -metric profit_factor -out "./reports/$T"
done
```

`-basket` жёстко зашит под momentum и для smc недоступен — pooled OOS сводится вручную из пер-тикерных отчётов (как делали для ORB). Вердикт по спеке: pooled OOS PF ≥ 1.5 — живёт; ниже — хороним и фиксируем в памяти.
