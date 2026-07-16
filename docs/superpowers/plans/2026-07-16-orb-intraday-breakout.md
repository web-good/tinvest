# ORB Intraday Breakout — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Новая long-only внутридневная стратегия ORB (пробой утреннего диапазона) для бэктеста: ядро + реестр + wiring в `cmd/backtest` + сетки калибровки, по спеке `docs/superpowers/specs/2026-07-16-orb-intraday-breakout-design.md`.

**Architecture:** Чистое ядро `internal/service/trading_strategy/orb/strategy/core` поверх общих типов `scalping/strategy.MarketData/Position` и `scalping/model.Signal` (паттерн reversion). 8 тонких пер-тикерных пакетов + `ORBLookupOrGeneric` в `internal/service/backtest/orb_registry.go` + `case "orb"` в `cmd/backtest`. Движок бэктеста НЕ меняется: reason `"SL"` уже в `model.IsStopReason` (филл `min(stop, open)`), reason `"EOD"` — не-стоповый (филл по close), `sig.StopLoss` уже замораживается движком в `Position.StopLoss` (`internal/domain/backtest/engine.go:126`).

**Tech Stack:** Go 1.25, стандартный `testing` (in-package тесты, как в reversion core), `tinvest/pkg/indicators` (ATR), `./bin/mage ci` как приёмочный гейт.

**Замечание по скоупу:** шаг А спеки (kill-test adaptive-ядра) — ручные прогоны пользователя с токеном, в план кода НЕ входит. Сетки для него уже восстановлены в `data/params/scalping/`.

## Global Constraints

- Long-only, шортов нет (решение пользователя, навсегда).
- Ровно 4 параметра: `ORBars, VolMult, Buffer, MaxRangeATR`. Константы ядра НЕ тюнятся: `atrPeriod=14`, `volAvgPeriod=20`, `sessionOpenHour=10`, `eodHour=18`, `dayBarsMax=32`.
- Все временные правила — MSK (`Europe/Moscow`, UTC-фолбэк). Без `MarketData.Times` (или при рассинхроне длин) стратегия — no-op.
- Никаких trail/breakeven/take-profit в v1.
- Движок бэктеста и общие типы (`scalping/model`, `scalping/strategy`) не модифицируются.
- Каждая задача заканчивается коммитом; финальный гейт — `./bin/mage ci` (lint + `go test -race ./...` + mock-drift).
- `go build ./...` падает на `magefiles` (нет `main`) — для сборки использовать `go build ./internal/... ./pkg/... ./cmd/...`.

---

### Task 1: Ядро — типы, константы, календарные хелперы

**Files:**
- Create: `internal/service/trading_strategy/orb/strategy/core/core.go`
- Create: `internal/service/trading_strategy/orb/strategy/core/core_test.go`

**Interfaces:**
- Consumes: `tinvest/internal/service/trading_strategy/scalping/strategy` (типы `MarketData`, `Position`), `time`, `slices`.
- Produces (нужно задачам 2–4): `type Params struct{ ORBars int; VolMult, Buffer, MaxRangeATR float64 }`; `NewWithParams(ticker string, p Params) *Strategy`; методы `Ticker() string`, `Lookback() int`; неэкспортируемые хелперы `mskLoc *time.Location`, `isWeekend(t time.Time) bool`, `isFirstBarOfDay(times []time.Time, i int) bool`, `sessionBarsToday(times []time.Time, cur int) []int`, `rangeOf(highs, lows []float64, idx []int) (hi, lo float64)`, `averageVolumeExcludingWeekends(vols []int64, times []time.Time, period int) (avg float64, ok bool)`; тестовые хелперы `mskT`, `mskBar`, `mdOf`, `warmupDay`.

- [ ] **Step 1: Write the failing tests**

Создать `core_test.go` (in-package `core`, как у reversion):

```go
package core

import (
	"slices"
	"testing"
	"time"

	"tinvest/internal/service/trading_strategy/scalping/strategy"
)

// mskBar — компактная спека свечи для тестов: время открытия MSK + H/L/C/V
// (Open в MarketData отсутствует).
type mskBar struct {
	t       time.Time
	h, l, c float64
	v       int64
}

func mskT(y int, m time.Month, d, hh, mm int) time.Time {
	return time.Date(y, m, d, hh, mm, 0, 0, mskLoc)
}

func mdOf(bars []mskBar, pos *strategy.Position) strategy.MarketData {
	md := strategy.MarketData{Position: pos}
	for _, b := range bars {
		md.Times = append(md.Times, b.t)
		md.Highs = append(md.Highs, b.h)
		md.Lows = append(md.Lows, b.l)
		md.Closes = append(md.Closes, b.c)
		md.Volumes = append(md.Volumes, b.v)
	}
	if n := len(bars); n > 0 {
		md.Price = bars[n-1].c
	}
	return md
}

// warmupDay — полный основной Min30-день (16 баров 10:00–17:30) с постоянной
// ценой 100 и диапазоном h101/l99: TR каждого бара = 2, так что ATR(14) на
// прогретой серии равен ровно 2. Объём 1000.
func warmupDay(y int, m time.Month, d int) []mskBar {
	var bars []mskBar
	for i := 0; i < 16; i++ {
		t := mskT(y, m, d, 10+i/2, (i%2)*30)
		bars = append(bars, mskBar{t: t, h: 101, l: 99, c: 100, v: 1000})
	}
	return bars
}

func TestIsWeekend(t *testing.T) {
	if !isWeekend(mskT(2026, 7, 18, 12, 0)) { // суббота
		t.Fatal("суббота должна быть выходным")
	}
	if isWeekend(mskT(2026, 7, 14, 12, 0)) { // вторник
		t.Fatal("вторник не выходной")
	}
}

func TestIsFirstBarOfDay(t *testing.T) {
	times := []time.Time{
		mskT(2026, 7, 13, 17, 30),
		mskT(2026, 7, 14, 10, 0),
		mskT(2026, 7, 14, 10, 30),
	}
	if !isFirstBarOfDay(times, 1) {
		t.Fatal("бар 10:00 нового дня — первый бар дня")
	}
	if isFirstBarOfDay(times, 2) {
		t.Fatal("бар 10:30 — не первый бар дня")
	}
	if !isFirstBarOfDay(times, 0) {
		t.Fatal("край окна (i==0) считается началом дня — защитный дефолт")
	}
}

func TestSessionBarsTodaySkipsMorningEveningAndOtherDays(t *testing.T) {
	times := []time.Time{
		mskT(2026, 7, 13, 17, 30), // вчера — отсекается
		mskT(2026, 7, 14, 9, 0),   // утренняя сессия — игнор
		mskT(2026, 7, 14, 10, 0),
		mskT(2026, 7, 14, 10, 30),
		mskT(2026, 7, 14, 11, 0),
	}
	got := sessionBarsToday(times, 4)
	want := []int{2, 3, 4}
	if !slices.Equal(got, want) {
		t.Fatalf("sessionBarsToday = %v, want %v", got, want)
	}
}

func TestSessionBarsTodayExcludesEODAndEvening(t *testing.T) {
	times := []time.Time{
		mskT(2026, 7, 14, 10, 0),
		mskT(2026, 7, 14, 17, 30),
		mskT(2026, 7, 14, 18, 0), // EOD-бар — вне [10,18)
		mskT(2026, 7, 14, 19, 0), // вечерняя сессия — игнор
	}
	got := sessionBarsToday(times, 3)
	want := []int{0, 1}
	if !slices.Equal(got, want) {
		t.Fatalf("sessionBarsToday = %v, want %v", got, want)
	}
}

func TestRangeOf(t *testing.T) {
	highs := []float64{5, 7, 6}
	lows := []float64{3, 4, 2}
	hi, lo := rangeOf(highs, lows, []int{0, 1})
	if hi != 7 || lo != 3 {
		t.Fatalf("rangeOf = (%v, %v), want (7, 3)", hi, lo)
	}
}

func TestAverageVolumeExcludesWeekends(t *testing.T) {
	vols := []int64{1000, 9000, 1000, 2000}
	times := []time.Time{
		mskT(2026, 7, 17, 10, 0), // пятница
		mskT(2026, 7, 18, 10, 0), // суббота — исключается
		mskT(2026, 7, 20, 10, 0), // понедельник
		mskT(2026, 7, 20, 10, 30), // входной бар — не в своей базе
	}
	avg, ok := averageVolumeExcludingWeekends(vols, times, 3)
	if !ok || avg != 1000 {
		t.Fatalf("avg = %v ok = %v, want 1000 true", avg, ok)
	}
}

func TestAverageVolumeNoTimesKeepsAllBars(t *testing.T) {
	vols := []int64{1000, 3000, 2000}
	avg, ok := averageVolumeExcludingWeekends(vols, nil, 2)
	if !ok || avg != 2000 {
		t.Fatalf("avg = %v ok = %v, want 2000 true", avg, ok)
	}
}

func TestNewWithParamsAndLookback(t *testing.T) {
	s := NewWithParams("SBER", Params{ORBars: 2, Buffer: 0.25})
	if s.Ticker() != "SBER" {
		t.Fatalf("Ticker = %q, want SBER", s.Ticker())
	}
	// Lookback обязан покрыть полный день Min30 (32 бара) + прогрев объёмной
	// базы (20) и ATR (14) с запасом.
	if lb := s.Lookback(); lb < 32+20+14 {
		t.Fatalf("Lookback = %d, слишком мал", lb)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/service/trading_strategy/orb/... -run . -v`
Expected: FAIL — пакет не существует / undefined identifiers.

- [ ] **Step 3: Write minimal implementation**

Создать `core.go`:

```go
// Package core implements the ORB (opening range breakout) intraday long-only
// strategy: the first ORBars main-session bars of the MSK day form a range;
// the first bar of the day to CLOSE above the range high triggers a buy with a
// structural stop below the range low; the position is force-closed at the end
// of the main session (never held overnight). Deliberately only 4 tunable
// params — exit timing is a constant by design. See
// docs/superpowers/specs/2026-07-16-orb-intraday-breakout-design.md.
package core

import (
	"fmt"
	"slices"
	"time"

	"tinvest/internal/service/trading_strategy/scalping/model"
	"tinvest/internal/service/trading_strategy/scalping/strategy"
)

// Fixed knobs — deliberately NOT part of Params: sweeping exit/warm-up
// mechanics is how past strategies overfit.
const (
	atrPeriod       = 14 // stop unit and range-filter unit
	volAvgPeriod    = 20 // volume-gate baseline window (bars)
	sessionOpenHour = 10 // MSK: bars opening before 10:00 (morning session) are ignored
	eodHour         = 18 // MSK: the first bar opening at/after 18:00 force-closes the day
	// dayBarsMax is the worst-case bar count of one MSK day on the finest
	// supported timeframe (Minutes30 incl. evening session). Lookback must
	// cover the whole current day plus indicator warm-up.
	dayBarsMax = 32
)

// Params are the ORB tunables. Zero VolMult / MaxRangeATR disable the
// corresponding optional gate.
type Params struct {
	ORBars      int     // bars in the opening range, counted from the first bar at/after 10:00 MSK
	VolMult     float64 // breakout-bar volume must be >= VolMult*avg(volAvgPeriod); <=0 disables
	Buffer      float64 // hard stop = ORL - Buffer*ATR(atrPeriod) at entry
	MaxRangeATR float64 // skip the day when ORH-ORL > MaxRangeATR*ATR at OR formation; <=0 disables
}

// Strategy is the ORB rule bound to one ticker.
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

// Lookback returns the candle window the strategy needs: a full worst-case
// trading day plus volume-average and ATR warm-up.
func (s *Strategy) Lookback() int { return dayBarsMax + volAvgPeriod + atrPeriod + 2 }

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

// isFirstBarOfDay reports whether bar i opens a new MSK day relative to bar
// i-1. The window edge (i == 0) counts as a day start: with a position open it
// forces the protective EOD exit instead of risking a silent overnight carry.
func isFirstBarOfDay(times []time.Time, i int) bool {
	if i == 0 {
		return true
	}
	a, b := times[i].In(mskLoc), times[i-1].In(mskLoc)
	return a.Year() != b.Year() || a.YearDay() != b.YearDay()
}

// sessionBarsToday returns the indexes (oldest-first) of the current MSK day's
// main-session bars — open-time hour in [sessionOpenHour, eodHour) — walking
// back from bar cur. Pre-market (morning-session) and evening bars are skipped.
func sessionBarsToday(times []time.Time, cur int) []int {
	t := times[cur].In(mskLoc)
	var idx []int
	for i := cur; i >= 0; i-- {
		ti := times[i].In(mskLoc)
		if ti.Year() != t.Year() || ti.YearDay() != t.YearDay() {
			break
		}
		if h := ti.Hour(); h >= sessionOpenHour && h < eodHour {
			idx = append(idx, i)
		}
	}
	slices.Reverse(idx)
	return idx
}

// rangeOf returns the highest high and lowest low across the given bar indexes.
// idx must be non-empty.
func rangeOf(highs, lows []float64, idx []int) (hi, lo float64) {
	hi, lo = highs[idx[0]], lows[idx[0]]
	for _, j := range idx[1:] {
		hi = max(hi, highs[j])
		lo = min(lo, lows[j])
	}
	return hi, lo
}

// averageVolumeExcludingWeekends averages the volumes of the `period` bars that
// PRECEDE the final (entry) bar of vols. The entry bar is never part of its own
// average. When times is supplied and index-aligned to vols, weekend bars
// (Sat/Sun MSK) are dropped; when times is empty or misaligned, weekend
// exclusion is skipped (all preceding bars count). Non-positive volumes are
// ignored. ok is false when no sample survives — the caller must then skip the
// gate (never block an entry on missing data). Mirrors the reversion core.
func averageVolumeExcludingWeekends(vols []int64, times []time.Time, period int) (avg float64, ok bool) {
	n := len(vols)
	if n < 2 || period <= 0 {
		return 0, false
	}
	lo := n - 1 - period // window = the `period` bars before the entry bar: [lo, n-1)
	if lo < 0 {
		lo = 0
	}
	haveTimes := len(times) == n
	var sum float64
	var count int
	for j := lo; j < n-1; j++ {
		if haveTimes && isWeekend(times[j]) {
			continue
		}
		if vols[j] <= 0 {
			continue
		}
		sum += float64(vols[j])
		count++
	}
	if count == 0 {
		return 0, false
	}
	return sum / float64(count), true
}

// model is referenced by tasks 2-4; keep the import pinned until then.
var _ = model.SignalNone
var _ = fmt.Sprintf
```

(Строки `var _ =` — временные пины импортов, задачи 2–4 их удалят, когда появятся Decide/manage/Explain.)

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/service/trading_strategy/orb/... -v`
Expected: PASS (все тесты Task 1).

- [ ] **Step 5: Commit**

```bash
git add internal/service/trading_strategy/orb/
git commit -m "feat(orb): core package skeleton — params, session calendar helpers"
```

---

### Task 2: Decide + manage — интрабарный стоп и принудительный EOD-выход

**Files:**
- Modify: `internal/service/trading_strategy/orb/strategy/core/core.go`
- Modify: `internal/service/trading_strategy/orb/strategy/core/core_test.go`

**Interfaces:**
- Consumes: типы и хелперы Task 1; `scalping/model.Signal{Kind, Ticker, Price, StopLoss, Reason, ExitReason}`, `model.SignalNone/SignalBuy/SignalSell`; `strategy.Position{StopLoss}`.
- Produces: `func (s *Strategy) Decide(md strategy.MarketData) model.Signal` (полный контракт `strategy.Strategy`); `func (s *Strategy) manage(md strategy.MarketData, t time.Time, sig model.Signal) model.Signal`. Flat-путь Decide пока делегирует `s.entryCheck` — в этой задаче заглушка, возвращающая `(sig, "вход реализуется в задаче 3")`; Task 3 её заменит.

- [ ] **Step 1: Write the failing tests**

Добавить в `core_test.go`:

```go
func posAt(stop float64) *strategy.Position {
	return &strategy.Position{PurchasePrice: 103.5, Quantity: 10, StopLoss: stop}
}

func TestManageStopFiresIntrabar(t *testing.T) {
	bars := append(warmupDay(2026, 7, 13),
		mskBar{t: mskT(2026, 7, 14, 12, 0), h: 103, l: 98, c: 99, v: 1000})
	md := mdOf(bars, posAt(98.5))
	sig := NewWithParams("SBER", Params{ORBars: 2, Buffer: 0.25}).Decide(md)
	if sig.Kind != model.SignalSell || sig.Reason != "SL" {
		t.Fatalf("kind=%v reason=%q, want Sell/SL", sig.Kind, sig.Reason)
	}
	if sig.StopLoss != 98.5 {
		t.Fatalf("sig.StopLoss = %v, want 98.5 (нужен движку для филла min(stop, open))", sig.StopLoss)
	}
}

func TestManageEODExitAtEODHour(t *testing.T) {
	bars := append(warmupDay(2026, 7, 13),
		mskBar{t: mskT(2026, 7, 14, 18, 0), h: 104, l: 103, c: 103.5, v: 1000})
	md := mdOf(bars, posAt(98.5))
	sig := NewWithParams("SBER", Params{ORBars: 2, Buffer: 0.25}).Decide(md)
	if sig.Kind != model.SignalSell || sig.Reason != "EOD" {
		t.Fatalf("kind=%v reason=%q, want Sell/EOD", sig.Kind, sig.Reason)
	}
}

func TestManageStopPrecedesEOD(t *testing.T) {
	// На EOD-баре зацеплен и стоп: интрабарный стоп раньше по времени, чем close.
	bars := append(warmupDay(2026, 7, 13),
		mskBar{t: mskT(2026, 7, 14, 18, 0), h: 104, l: 98, c: 103.5, v: 1000})
	md := mdOf(bars, posAt(98.5))
	sig := NewWithParams("SBER", Params{ORBars: 2, Buffer: 0.25}).Decide(md)
	if sig.Reason != "SL" {
		t.Fatalf("reason = %q, want SL (стоп важнее EOD)", sig.Reason)
	}
}

func TestManageEODExitOnFirstBarOfNewDay(t *testing.T) {
	// Короткая сессия без 18:00-бара: позиция дотянула до утра — принудительный
	// выход на первом баре нового дня, а не тихий овернайт.
	bars := append(warmupDay(2026, 7, 13),
		mskBar{t: mskT(2026, 7, 14, 10, 0), h: 104, l: 103, c: 103.5, v: 1000})
	md := mdOf(bars, posAt(98.5))
	sig := NewWithParams("SBER", Params{ORBars: 2, Buffer: 0.25}).Decide(md)
	if sig.Kind != model.SignalSell || sig.Reason != "EOD" {
		t.Fatalf("kind=%v reason=%q, want Sell/EOD", sig.Kind, sig.Reason)
	}
}

func TestManageHoldsMidday(t *testing.T) {
	// Последний warmup-бар — 17:30 того же дня: не первый бар дня, час < 18,
	// low 99 выше стопа 98.5 → удерживаем.
	md := mdOf(warmupDay(2026, 7, 13), posAt(98.5))
	sig := NewWithParams("SBER", Params{ORBars: 2, Buffer: 0.25}).Decide(md)
	if sig.Kind != model.SignalNone {
		t.Fatalf("kind = %v, want None (держим до стопа или EOD)", sig.Kind)
	}
}

func TestDecideNoOpWithoutTimes(t *testing.T) {
	md := mdOf(append(warmupDay(2026, 7, 13),
		mskBar{t: mskT(2026, 7, 14, 12, 0), h: 103, l: 98, c: 99, v: 1000}), posAt(98.5))
	md.Times = nil // без временных меток сессии не определить — no-op
	sig := NewWithParams("SBER", Params{ORBars: 2, Buffer: 0.25}).Decide(md)
	if sig.Kind != model.SignalNone {
		t.Fatalf("kind = %v, want None (защитный дефолт без Times)", sig.Kind)
	}
}
```

И импорт `"tinvest/internal/service/trading_strategy/scalping/model"` в core_test.go.

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/service/trading_strategy/orb/... -run TestManage -v`
Expected: FAIL — `Decide`/`manage` не определены.

- [ ] **Step 3: Write minimal implementation**

В `core.go` удалить пины `var _ = model.SignalNone` / `var _ = fmt.Sprintf` и добавить:

```go
// Decide is pure: it computes everything from md and performs no I/O. Times
// are mandatory for the session logic; when Times is missing or misaligned the
// strategy is a deliberate no-op (protective default — can't see the session,
// don't trade).
func (s *Strategy) Decide(md strategy.MarketData) model.Signal {
	sig := model.Signal{Kind: model.SignalNone, Ticker: s.ticker, Price: md.Price}
	n := len(md.Closes)
	if n == 0 || len(md.Times) != n || len(md.Highs) != n || len(md.Lows) != n {
		return sig
	}
	t := md.Times[n-1].In(mskLoc)
	if md.Position != nil {
		return s.manage(md, t, sig)
	}
	sig, _ = s.entryCheck(md, t, sig)
	return sig
}

// manage handles an open position: the intrabar hard stop first (it fires
// earlier in real time than any close-based rule), then the forced end-of-day
// exit. EOD also covers weekends and a new-day first bar (short sessions
// without an 18:00 bar) so a position can never silently carry overnight.
func (s *Strategy) manage(md strategy.MarketData, t time.Time, sig model.Signal) model.Signal {
	n := len(md.Closes)
	pos := md.Position
	if pos.StopLoss > 0 && md.Lows[n-1] <= pos.StopLoss {
		sig.Kind = model.SignalSell
		sig.Reason = "SL"
		sig.StopLoss = pos.StopLoss
		sig.ExitReason = fmt.Sprintf("интрабарный стоп %.4f (low бара %.4f)", pos.StopLoss, md.Lows[n-1])
		return sig
	}
	if t.Hour() >= eodHour || isWeekend(t) || isFirstBarOfDay(md.Times, n-1) {
		sig.Kind = model.SignalSell
		sig.Reason = "EOD"
		sig.ExitReason = "конец сессии — принудительный внутридневной выход"
		return sig
	}
	return sig
}

// entryCheck is implemented in the entry task; until then the flat path never
// enters.
func (s *Strategy) entryCheck(md strategy.MarketData, t time.Time, sig model.Signal) (model.Signal, string) {
	return sig, "вход реализуется в задаче 3"
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/service/trading_strategy/orb/... -v`
Expected: PASS (Task 1 + Task 2 тесты).

- [ ] **Step 5: Commit**

```bash
git add internal/service/trading_strategy/orb/
git commit -m "feat(orb): manage pass — intrabar hard stop and forced EOD exit"
```

---

### Task 3: Вход — первый пробой дня, фильтр ширины OR, объёмный гейт, стоп

**Files:**
- Modify: `internal/service/trading_strategy/orb/strategy/core/core.go`
- Modify: `internal/service/trading_strategy/orb/strategy/core/core_test.go`

**Interfaces:**
- Consumes: хелперы Task 1; `indicators.ATR(highs, lows, closes []float64, period int) float64` из `tinvest/pkg/indicators`.
- Produces: полная реализация `entryCheck(md strategy.MarketData, t time.Time, sig model.Signal) (model.Signal, string)` — заменяет заглушку Task 2. На успехе `sig.Kind == model.SignalBuy`, `sig.StopLoss = ORL − Buffer×ATR`, `sig.ATR = ATR`, `sig.EntryReason` заполнен; на отказе второй результат — человекочитаемая причина (нужна Explain в Task 4).

- [ ] **Step 1: Write the failing tests**

Добавить в `core_test.go` (импорт `"tinvest/pkg/indicators"`):

```go
// entryDay строит день 2026-07-14 (вторник): 2 OR-бара + пробойный бар.
// ORH = 103 (max high), ORL = 99.5 (min low); breakout close 104 > 103.
func entryDay(breakoutClose float64, breakoutVol int64) []mskBar {
	return []mskBar{
		{t: mskT(2026, 7, 14, 10, 0), h: 102, l: 99.5, c: 101.5, v: 1000},
		{t: mskT(2026, 7, 14, 10, 30), h: 103, l: 100, c: 102, v: 1000},
		{t: mskT(2026, 7, 14, 11, 0), h: 104.5, l: 102, c: breakoutClose, v: breakoutVol},
	}
}

func TestEntryFirstBreakout(t *testing.T) {
	bars := append(warmupDay(2026, 7, 13), entryDay(104, 1000)...)
	md := mdOf(bars, nil)
	sig := NewWithParams("SBER", Params{ORBars: 2, Buffer: 0.25}).Decide(md)
	if sig.Kind != model.SignalBuy {
		t.Fatalf("kind = %v, want Buy", sig.Kind)
	}
	atr := indicators.ATR(md.Highs, md.Lows, md.Closes, 14)
	if atr <= 0 {
		t.Fatal("тестовые данные должны давать прогретый ATR")
	}
	wantStop := 99.5 - 0.25*atr
	if sig.StopLoss != wantStop {
		t.Fatalf("StopLoss = %v, want %v (ORL − Buffer×ATR)", sig.StopLoss, wantStop)
	}
	if sig.ATR != atr {
		t.Fatalf("sig.ATR = %v, want %v", sig.ATR, atr)
	}
	if sig.EntryReason == "" {
		t.Fatal("EntryReason должен быть заполнен")
	}
}

func TestNoEntryDuringORFormation(t *testing.T) {
	bars := append(warmupDay(2026, 7, 13), entryDay(104, 1000)[:2]...)
	sig := NewWithParams("SBER", Params{ORBars: 2, Buffer: 0.25}).Decide(mdOf(bars, nil))
	if sig.Kind != model.SignalNone {
		t.Fatalf("kind = %v, want None (диапазон ещё формируется)", sig.Kind)
	}
}

func TestNoEntryCloseAtOrBelowORH(t *testing.T) {
	bars := append(warmupDay(2026, 7, 13), entryDay(103, 1000)...) // close == ORH
	sig := NewWithParams("SBER", Params{ORBars: 2, Buffer: 0.25}).Decide(mdOf(bars, nil))
	if sig.Kind != model.SignalNone {
		t.Fatalf("kind = %v, want None (нужен строго close > ORH)", sig.Kind)
	}
}

func TestNoEntrySecondBreakoutSameDay(t *testing.T) {
	// Бар 11:00 уже закрылся выше ORH (первый пробой), 11:30 упал, 12:00 снова
	// выше — повторного входа нет (stateless «один вход в день»).
	bars := append(warmupDay(2026, 7, 13), entryDay(104, 1000)...)
	bars = append(bars,
		mskBar{t: mskT(2026, 7, 14, 11, 30), h: 104, l: 101, c: 102, v: 1000},
		mskBar{t: mskT(2026, 7, 14, 12, 0), h: 104.5, l: 102, c: 104, v: 1000})
	sig := NewWithParams("SBER", Params{ORBars: 2, Buffer: 0.25}).Decide(mdOf(bars, nil))
	if sig.Kind != model.SignalNone {
		t.Fatalf("kind = %v, want None (первый пробой дня уже был)", sig.Kind)
	}
}

func TestNoEntryAtEODHourOrWeekend(t *testing.T) {
	// EOD-час.
	bars := append(warmupDay(2026, 7, 13), entryDay(104, 1000)...)
	bars[len(bars)-1].t = mskT(2026, 7, 14, 18, 0)
	if sig := NewWithParams("SBER", Params{ORBars: 2, Buffer: 0.25}).Decide(mdOf(bars, nil)); sig.Kind != model.SignalNone {
		t.Fatalf("EOD-час: kind = %v, want None", sig.Kind)
	}
	// Выходной (суббота 2026-07-18): весь день переносим на субботу.
	wk := append(warmupDay(2026, 7, 17), []mskBar{
		{t: mskT(2026, 7, 18, 10, 0), h: 102, l: 99.5, c: 101.5, v: 1000},
		{t: mskT(2026, 7, 18, 10, 30), h: 103, l: 100, c: 102, v: 1000},
		{t: mskT(2026, 7, 18, 11, 0), h: 104.5, l: 102, c: 104, v: 1000},
	}...)
	if sig := NewWithParams("SBER", Params{ORBars: 2, Buffer: 0.25}).Decide(mdOf(wk, nil)); sig.Kind != model.SignalNone {
		t.Fatalf("суббота: kind = %v, want None", sig.Kind)
	}
}

func TestRangeFilterBlocksWideOR(t *testing.T) {
	wide := append(warmupDay(2026, 7, 13), []mskBar{
		{t: mskT(2026, 7, 14, 10, 0), h: 106, l: 99, c: 105, v: 1000},
		{t: mskT(2026, 7, 14, 10, 30), h: 106.5, l: 99.5, c: 105.5, v: 1000},
		{t: mskT(2026, 7, 14, 11, 0), h: 108, l: 105, c: 107, v: 1000},
	}...) // OR range = 106.5 − 99 = 7.5 >> ATR
	if sig := NewWithParams("SBER", Params{ORBars: 2, Buffer: 0.25, MaxRangeATR: 1.5}).Decide(mdOf(wide, nil)); sig.Kind != model.SignalNone {
		t.Fatalf("широкий OR: kind = %v, want None", sig.Kind)
	}
	// Тот же день с выключенным фильтром — вход есть.
	if sig := NewWithParams("SBER", Params{ORBars: 2, Buffer: 0.25}).Decide(mdOf(wide, nil)); sig.Kind != model.SignalBuy {
		t.Fatalf("фильтр выключен: kind = %v, want Buy", sig.Kind)
	}
}

func TestVolumeGateBlocksThinBreakout(t *testing.T) {
	// avg объёма 20 предшествующих баров = 1000 → порог 1.5×1000 = 1500.
	blocked := append(warmupDay(2026, 7, 13), entryDay(104, 1200)...)
	if sig := NewWithParams("SBER", Params{ORBars: 2, Buffer: 0.25, VolMult: 1.5}).Decide(mdOf(blocked, nil)); sig.Kind != model.SignalNone {
		t.Fatalf("тонкий пробой: kind = %v, want None", sig.Kind)
	}
	passed := append(warmupDay(2026, 7, 13), entryDay(104, 1600)...)
	if sig := NewWithParams("SBER", Params{ORBars: 2, Buffer: 0.25, VolMult: 1.5}).Decide(mdOf(passed, nil)); sig.Kind != model.SignalBuy {
		t.Fatalf("объёмный пробой: kind = %v, want Buy", sig.Kind)
	}
}

func TestNoEntryColdATR(t *testing.T) {
	// Всего 3 бара — ATR(14) не прогрет → входа нет (не с чем считать стоп).
	sig := NewWithParams("SBER", Params{ORBars: 2, Buffer: 0.25}).Decide(mdOf(entryDay(104, 1000), nil))
	if sig.Kind != model.SignalNone {
		t.Fatalf("холодный ATR: kind = %v, want None", sig.Kind)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/service/trading_strategy/orb/... -run 'TestEntry|TestNoEntry|TestRangeFilter|TestVolumeGate' -v`
Expected: FAIL — заглушка entryCheck никогда не входит.

- [ ] **Step 3: Write the implementation**

Заменить заглушку `entryCheck` в `core.go` (добавить импорт `"tinvest/pkg/indicators"`):

```go
// entryCheck runs the full entry pipeline. On rejection the second result
// holds a human-readable reason (consumed by Explain). On success
// sig.Kind == model.SignalBuy with the frozen structural stop in sig.StopLoss.
func (s *Strategy) entryCheck(md strategy.MarketData, t time.Time, sig model.Signal) (model.Signal, string) {
	n := len(md.Closes)
	if isWeekend(t) {
		return sig, "выходной (Сб/Вс MSK) — входы запрещены"
	}
	if h := t.Hour(); h >= eodHour {
		return sig, fmt.Sprintf("час бара %d ≥ %d MSK — входы после EOD-часа запрещены", h, eodHour)
	} else if h < sessionOpenHour {
		return sig, fmt.Sprintf("час бара %d < %d MSK — утренняя сессия игнорируется", h, sessionOpenHour)
	}
	if s.p.ORBars <= 0 {
		return sig, "ORBars ≤ 0 — вход выключен"
	}
	session := sessionBarsToday(md.Times, n-1)
	if len(session) < s.p.ORBars+1 {
		return sig, fmt.Sprintf("диапазон не сформирован: %d из %d сессионных баров", len(session), s.p.ORBars+1)
	}
	or := session[:s.p.ORBars]
	orh, orl := rangeOf(md.Highs, md.Lows, or)
	if s.p.MaxRangeATR > 0 {
		formEnd := or[len(or)-1]
		atrForm := indicators.ATR(md.Highs[:formEnd+1], md.Lows[:formEnd+1], md.Closes[:formEnd+1], atrPeriod)
		if atrForm > 0 && orh-orl > s.p.MaxRangeATR*atrForm {
			return sig, fmt.Sprintf("день пропущен: ширина OR %.4f > %.2f×ATR %.4f — риск непропорционален тейку",
				orh-orl, s.p.MaxRangeATR, atrForm)
		}
	}
	// Stateless "one entry per day": only the FIRST close above ORH may enter.
	// After a stop-out the earlier breakout bar is still in today's history, so
	// a re-entry is impossible by construction.
	for _, j := range session[s.p.ORBars : len(session)-1] {
		if md.Closes[j] > orh {
			return sig, fmt.Sprintf("первый пробой дня уже был (бар %s, close %.4f > ORH %.4f) — повторный вход запрещён",
				md.Times[j].In(mskLoc).Format("15:04"), md.Closes[j], orh)
		}
	}
	if md.Closes[n-1] <= orh {
		return sig, fmt.Sprintf("close %.4f ≤ ORH %.4f — пробоя нет", md.Closes[n-1], orh)
	}
	if s.p.VolMult > 0 {
		if avg, ok := averageVolumeExcludingWeekends(md.Volumes, md.Times, volAvgPeriod); ok && avg > 0 &&
			float64(md.Volumes[n-1]) < s.p.VolMult*avg {
			return sig, fmt.Sprintf("объём пробойного бара %d < %.2f×среднего %.0f — пробой без участников",
				md.Volumes[n-1], s.p.VolMult, avg)
		}
	}
	atr := indicators.ATR(md.Highs, md.Lows, md.Closes, atrPeriod)
	if atr <= 0 {
		return sig, "ATR не прогрет — вход запрещён (не с чего считать стоп)"
	}
	sig.Kind = model.SignalBuy
	sig.StopLoss = orl - s.p.Buffer*atr
	sig.ATR = atr
	sig.EntryReason = fmt.Sprintf("пробой утреннего диапазона: close %.4f > ORH %.4f (OR из %d баров, ORL %.4f), стоп %.4f",
		md.Closes[n-1], orh, s.p.ORBars, orl, sig.StopLoss)
	return sig, ""
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/service/trading_strategy/orb/... -v`
Expected: PASS (все тесты задач 1–3).

- [ ] **Step 5: Commit**

```bash
git add internal/service/trading_strategy/orb/
git commit -m "feat(orb): entry — first daily breakout with structural stop, range and volume gates"
```

---

### Task 4: Explain — диагностика бара для `-explain`

**Files:**
- Modify: `internal/service/trading_strategy/orb/strategy/core/core.go`
- Modify: `internal/service/trading_strategy/orb/strategy/core/core_test.go`

**Interfaces:**
- Consumes: `entryCheck` (Task 3), `manage` (Task 2).
- Produces: `func (s *Strategy) Explain(md strategy.MarketData) string` — сигнатура, которую type-assert'ит `domain.Trace` (`internal/domain/backtest/engine.go:162`: `Explain(md strategy.MarketData) string`).

- [ ] **Step 1: Write the failing tests**

Добавить в `core_test.go` (импорт `"strings"`):

```go
func TestExplainReportsEntryBlock(t *testing.T) {
	bars := append(warmupDay(2026, 7, 13), entryDay(102.5, 1000)...) // close ниже ORH
	out := NewWithParams("SBER", Params{ORBars: 2, Buffer: 0.25}).Explain(mdOf(bars, nil))
	if !strings.Contains(out, "ORH") || !strings.Contains(out, "входа нет") {
		t.Fatalf("Explain должен объяснить отсутствие пробоя, got: %s", out)
	}
}

func TestExplainReportsEntry(t *testing.T) {
	bars := append(warmupDay(2026, 7, 13), entryDay(104, 1000)...)
	out := NewWithParams("SBER", Params{ORBars: 2, Buffer: 0.25}).Explain(mdOf(bars, nil))
	if !strings.Contains(out, "ВХОД") {
		t.Fatalf("Explain должен показать вход, got: %s", out)
	}
}

func TestExplainReportsExitAndHold(t *testing.T) {
	out := NewWithParams("SBER", Params{ORBars: 2, Buffer: 0.25}).Explain(mdOf(warmupDay(2026, 7, 13), posAt(98.5)))
	if !strings.Contains(out, "удерживается") {
		t.Fatalf("Explain должен показать удержание, got: %s", out)
	}
	stop := append(warmupDay(2026, 7, 13),
		mskBar{t: mskT(2026, 7, 14, 12, 0), h: 103, l: 98, c: 99, v: 1000})
	out = NewWithParams("SBER", Params{ORBars: 2, Buffer: 0.25}).Explain(mdOf(stop, posAt(98.5)))
	if !strings.Contains(out, "ВЫХОД SL") {
		t.Fatalf("Explain должен показать выход по стопу, got: %s", out)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/service/trading_strategy/orb/... -run TestExplain -v`
Expected: FAIL — `Explain` не определён.

- [ ] **Step 3: Write the implementation**

Добавить в `core.go`:

```go
// Explain renders a human-readable verdict for the LAST bar of md — why the
// strategy did or did not act. Wired into `-explain` via domain.Trace, which
// type-asserts exactly this signature.
func (s *Strategy) Explain(md strategy.MarketData) string {
	sig := model.Signal{Kind: model.SignalNone, Ticker: s.ticker, Price: md.Price}
	n := len(md.Closes)
	if n == 0 || len(md.Times) != n || len(md.Highs) != n || len(md.Lows) != n {
		return "ORB: нет данных или Times не заполнен/не выровнен — стратегия без временных меток не работает"
	}
	t := md.Times[n-1].In(mskLoc)
	stamp := t.Format("2006-01-02 15:04")
	if md.Position != nil {
		out := s.manage(md, t, sig)
		if out.Kind == model.SignalSell {
			return fmt.Sprintf("ORB %s %s: ВЫХОД %s — %s", s.ticker, stamp, out.Reason, out.ExitReason)
		}
		return fmt.Sprintf("ORB %s %s: позиция удерживается (стоп %.4f, low бара %.4f)",
			s.ticker, stamp, md.Position.StopLoss, md.Lows[n-1])
	}
	out, why := s.entryCheck(md, t, sig)
	if out.Kind == model.SignalBuy {
		return fmt.Sprintf("ORB %s %s: ВХОД — %s", s.ticker, stamp, out.EntryReason)
	}
	return fmt.Sprintf("ORB %s %s: входа нет — %s", s.ticker, stamp, why)
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/service/trading_strategy/orb/... -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/service/trading_strategy/orb/
git commit -m "feat(orb): Explain — per-bar diagnostics for -explain"
```

---

### Task 5: Пер-тикерные пакеты, реестр, wiring в cmd/backtest, сетки

**Files:**
- Create: `internal/service/trading_strategy/orb/strategy/sber/sber.go` (и аналогично `gazp/gazp.go`, `nvtk/nvtk.go`, `plzl/plzl.go`, `ydex/ydex.go`, `afks/afks.go`, `rusal/rusal.go`, `mdmg/mdmg.go`)
- Create: `internal/service/backtest/orb_registry.go`
- Create: `internal/service/backtest/orb_registry_test.go`
- Modify: `cmd/backtest/main.go:1` (doc-comment), `cmd/backtest/main.go:40` (flag help), `cmd/backtest/main.go:163-175` (switch + сообщение об ошибке)
- Create: `data/params/{sber,gazp,nvtk,plzl,ydex,afks,rual,mdmg}/orb_grid.json` (папка для RUAL — `rual`, как у momentum/reversion)

**Interfaces:**
- Consumes: `core.Params`, `core.NewWithParams` (Task 1), `backtest.Binding{DefaultParams, Build, ParseParams}` (`internal/service/backtest/registry.go:17`).
- Produces: `ORBLookupOrGeneric(ticker string) Binding`; per-ticker `Ticker` const + `DefaultParams() core.Params`; строка стратегии `"orb"` в CLI.

- [ ] **Step 1: Write the failing registry tests**

Создать `internal/service/backtest/orb_registry_test.go`:

```go
package backtest

import (
	"testing"

	"tinvest/internal/service/trading_strategy/orb/strategy/core"
)

func TestORBLookupOrGenericRegistered(t *testing.T) {
	b := ORBLookupOrGeneric("SBER")
	if got := b.Build(b.DefaultParams()).Ticker(); got != "SBER" {
		t.Fatalf("Ticker = %q, want SBER", got)
	}
}

func TestORBLookupOrGenericFallback(t *testing.T) {
	b := ORBLookupOrGeneric("XXXX")
	if got := b.Build(b.DefaultParams()).Ticker(); got != "XXXX" {
		t.Fatalf("generic binding должен привязаться к запрошенному тикеру, got %q", got)
	}
}

func TestORBParseParamsPartialOverride(t *testing.T) {
	b := ORBLookupOrGeneric("SBER")
	p, err := b.ParseParams([]byte(`{"ORBars":3}`))
	if err != nil {
		t.Fatal(err)
	}
	cp := p.(core.Params)
	if cp.ORBars != 3 {
		t.Fatalf("ORBars = %d, want 3", cp.ORBars)
	}
	if cp.Buffer != 0.25 {
		t.Fatalf("Buffer = %v, want дефолт 0.25 (частичный JSON поверх дефолтов)", cp.Buffer)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/service/backtest/ -run TestORB -v`
Expected: FAIL — `ORBLookupOrGeneric` не определён.

- [ ] **Step 3: Write the implementation**

8 пер-тикерных пакетов, идентичная форма (показан SBER; остальные отличаются только именем пакета, константой и комментарием: GAZP «Газпром», NVTK «Новатэк», PLZL «Полюс», YDEX «Яндекс», AFKS «АФК Система», RUAL «Русал» — пакет `rusal`, MDMG «Мать и дитя»):

```go
// Package sber supplies the ticker and starting ORB Params for SBER (Sberbank).
// Starting values mirror the generic defaults; calibrate with -calibrate and
// then hardcode the winning combination here.
package sber

import "tinvest/internal/service/trading_strategy/orb/strategy/core"

// Ticker is the instrument this package configures.
const Ticker = "SBER"

// DefaultParams returns SBER's starting ORB parameters (pre-calibration).
func DefaultParams() core.Params {
	return core.Params{ORBars: 2, VolMult: 0, Buffer: 0.25, MaxRangeATR: 0}
}
```

`internal/service/backtest/orb_registry.go` (зеркало `reversion_registry.go`):

```go
package backtest

import (
	"encoding/json"
	"fmt"

	orbafks "tinvest/internal/service/trading_strategy/orb/strategy/afks"
	"tinvest/internal/service/trading_strategy/orb/strategy/core"
	orbgazp "tinvest/internal/service/trading_strategy/orb/strategy/gazp"
	orbmdmg "tinvest/internal/service/trading_strategy/orb/strategy/mdmg"
	orbnvtk "tinvest/internal/service/trading_strategy/orb/strategy/nvtk"
	orbplzl "tinvest/internal/service/trading_strategy/orb/strategy/plzl"
	orbrusal "tinvest/internal/service/trading_strategy/orb/strategy/rusal"
	orbsber "tinvest/internal/service/trading_strategy/orb/strategy/sber"
	orbydex "tinvest/internal/service/trading_strategy/orb/strategy/ydex"
	"tinvest/internal/service/trading_strategy/scalping/strategy"
)

// orbBindingFor builds a Binding for a ticker whose defaults come from defaults().
// All ORB tickers share the core engine; only ticker + defaults differ.
func orbBindingFor(ticker string, defaults func() core.Params) Binding {
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

var orbRegistry = map[string]Binding{
	orbsber.Ticker:  orbBindingFor(orbsber.Ticker, orbsber.DefaultParams),
	orbgazp.Ticker:  orbBindingFor(orbgazp.Ticker, orbgazp.DefaultParams),
	orbnvtk.Ticker:  orbBindingFor(orbnvtk.Ticker, orbnvtk.DefaultParams),
	orbplzl.Ticker:  orbBindingFor(orbplzl.Ticker, orbplzl.DefaultParams),
	orbydex.Ticker:  orbBindingFor(orbydex.Ticker, orbydex.DefaultParams),
	orbafks.Ticker:  orbBindingFor(orbafks.Ticker, orbafks.DefaultParams),
	orbrusal.Ticker: orbBindingFor(orbrusal.Ticker, orbrusal.DefaultParams),
	orbmdmg.Ticker:  orbBindingFor(orbmdmg.Ticker, orbmdmg.DefaultParams),
}

// genericORBDefaults are neutral baseline params for tickers without a dedicated
// ORB config. Intentionally independent of any per-ticker defaults so calibrating
// one ticker never drifts the generic baseline.
func genericORBDefaults() core.Params {
	return core.Params{ORBars: 2, VolMult: 0, Buffer: 0.25, MaxRangeATR: 0}
}

// ORBLookupOrGeneric returns the registered ORB binding for a ticker, or a
// generic binding bound to that ticker (with genericORBDefaults) when none is
// registered.
func ORBLookupOrGeneric(ticker string) Binding {
	if b, ok := orbRegistry[ticker]; ok {
		return b
	}
	return orbBindingFor(ticker, genericORBDefaults)
}
```

`cmd/backtest/main.go` — три правки:

1. Флаг (строка 40):
```go
strategyName = flag.String("strategy", "scalping", "strategy engine: scalping|levels|momentum|reversion|orb")
```
2. Switch (после `case "reversion":`):
```go
	case "orb":
		binding = svc.ORBLookupOrGeneric(ticker)
```
3. Сообщение об ошибке:
```go
		return fmt.Errorf("unknown strategy %q (want scalping|levels|momentum|reversion|orb)", strategyName)
```
Плюс doc-comment строки 1: заменить «(scalping or levels)» на «(scalping, levels, momentum, reversion or orb)».

8 файлов сеток `data/params/<папка>/orb_grid.json` (папки: sber, gazp, nvtk, plzl, ydex, afks, **rual**, mdmg) — одинаковое содержимое, плоский формат (легаси-формат `ParsePhases` принимает):

```json
{
  "ORBars": [1, 2, 3],
  "VolMult": [0, 1.0, 1.5],
  "Buffer": [0.1, 0.25, 0.5],
  "MaxRangeATR": [0, 1.5, 2.5]
}
```

- [ ] **Step 4: Run tests and build to verify**

Run: `go test ./internal/service/backtest/ ./internal/service/trading_strategy/orb/... && go build ./internal/... ./pkg/... ./cmd/...`
Expected: PASS + сборка без ошибок.

- [ ] **Step 5: Commit**

```bash
git add internal/service/trading_strategy/orb/ internal/service/backtest/orb_registry.go internal/service/backtest/orb_registry_test.go cmd/backtest/main.go data/params/
git commit -m "feat(orb): per-ticker packages, registry and cmd/backtest wiring + calibration grids"
```

---

### Task 6: Документация + финальный CI

**Files:**
- Create: `docs/orb/strategy.md`
- Modify: `CLAUDE.md` (Layout: строка про trading_strategy; Development Notes: команда запуска)

**Interfaces:**
- Consumes: всё реализованное в задачах 1–5.
- Produces: документацию; зелёный `./bin/mage ci`.

- [ ] **Step 1: Write docs/orb/strategy.md**

```markdown
# ORB — пробой утреннего диапазона (внутридневная, long-only)

Спека: `docs/superpowers/specs/2026-07-16-orb-intraday-breakout-design.md`.
Ядро: `internal/service/trading_strategy/orb/strategy/core`.

## Как работает

1. **Диапазон.** Первые `ORBars` баров основной сессии MOEX (время открытия
   ≥ 10:00 MSK; бары утренней сессии игнорируются) задают ORH = максимум
   high и ORL = минимум low.
2. **Фильтр дня** (`MaxRangeATR > 0`): если ширина диапазона больше
   `MaxRangeATR × ATR(14)` на момент формирования — день пропускается.
3. **Вход.** Первый бар дня, ЗАКРЫВШИЙСЯ выше ORH → покупка по close.
   Повторных входов в тот же день нет: правило stateless — если более ранний
   бар дня уже закрывался выше ORH, вход запрещён (в том числе после
   стоп-аута). Опциональный объёмный гейт (`VolMult > 0`): объём пробойного
   бара ≥ `VolMult ×` средний объём 20 предшествующих баров (выходные
   исключаются из базы).
4. **Стоп.** `ORL − Buffer × ATR(14)`, замораживается на входе
   (`Position.StopLoss`). Триггер интрабарный (`low ≤ stop`), филл в бэктесте
   по `min(stop, open)` — честная цена с учётом гэпа.
5. **Выход по времени.** Первый бар с открытием ≥ 18:00 MSK принудительно
   закрывает позицию по close (reason `EOD`). Страховки: выход также на
   первом баре нового дня (короткая сессия без 18:00-бара) и в выходные.
   Позиция никогда не переносится через ночь; trail/BE/TP отсутствуют
   сознательно.

## Параметры (все 4)

| Параметр | Дефолт | Смысл |
|---|---|---|
| `ORBars` | 2 | баров в утреннем диапазоне |
| `VolMult` | 0 (выкл) | объёмный гейт входа, множитель к среднему |
| `Buffer` | 0.25 | отступ стопа под ORL в долях ATR(14) |
| `MaxRangeATR` | 0 (выкл) | пропуск дня со слишком широким диапазоном |

Константы (не тюнятся): ATR(14), объёмная база 20 баров, сессия 10:00,
EOD 18:00, подтверждение пробоя — close бара.

## Запуск

Основной ТФ — Minutes30, контрольный — Hour1 (edge должен жить на обоих):

    go run ./cmd/backtest -ticker SBER -strategy orb -interval Minutes30 \
      -months 24 -out ./reports/SBER

Walk-forward калибровка (приёмка: pooled OOS PF > 1.5 на корзине из 8
тикеров при стабильных параметрах по фолдам):

    go run ./cmd/backtest -ticker SBER -strategy orb -interval Minutes30 \
      -months 24 -calibrate data/params/sber/orb_grid.json \
      -train-months 12 -test-months 6 -min-trades 10 -metric profit_factor

Диагностика бара: `-explain '2026-07-14 11:00'` (MSK).

Требование к данным: стратегия не работает без временных меток баров
(`MarketData.Times`) — бэктест заполняет их всегда; live-раннера в v1 нет.
```

- [ ] **Step 2: Update CLAUDE.md**

В Layout, в строке про `trading_strategy/`, добавить к перечню backtest-only стратегий `orb`:
`golden_x`, `reversion`, `bonds` (live strategies); `levels`, `momentum`, `orb` (backtest-only) — и упомянуть `orb` как «внутридневной пробой утреннего диапазона, Minutes30/Hour1, см. docs/orb/strategy.md».

В Development Notes добавить строку-пример запуска:
```
go run ./cmd/backtest -ticker SBER -strategy orb -interval Minutes30 -months 24 -calibrate data/params/sber/orb_grid.json -train-months 12 -test-months 6 -min-trades 10 -metric profit_factor
```

- [ ] **Step 3: Run full CI gate**

Run: `./bin/mage ci`
Expected: lint + `go test -race ./...` + mock-drift — всё зелёное. (Если `./bin/mage` отсутствует — сначала `make install-deps` или `go run mage.go tools` согласно `docs/tooling/mage.md`.)

- [ ] **Step 4: Commit**

```bash
git add docs/orb/ CLAUDE.md
git commit -m "docs(orb): strategy explainer + CLAUDE.md wiring"
```

---

## Manual follow-up (пользователь, вне плана)

1. Шаг А спеки — kill-test adaptive: прогоны с `data/params/scalping/{rusal,afks}_grid.json` (нужен `T_BANK`-токен).
2. Первая загрузка Minutes30-кэша и walk-forward ORB по 8 тикерам; сверка Hour1-контроля.
3. По результатам — решение о калибровке/live (отдельная спека).
