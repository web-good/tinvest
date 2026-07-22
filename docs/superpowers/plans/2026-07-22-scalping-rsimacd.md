# Scalping RSI+MACD Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Реализовать 5-минутную лонговую скальпинг-стратегию «RSI выходит из перепроданности → MACD(3,6,9) подтверждает пересечением ниже нуля», со стопом под минимум свечи RSI-кросса, тейком кратно риску и выходом по стохастику, доступную в бэктесте как `-strategy scalping_rsimacd`.

**Architecture:** Один чистый пакет `.../scalping_rsimacd/strategy/core` реализует интерфейс `scalping/strategy.Strategy` (`Ticker/Lookback/Decide`) плюс `Explain`. `Decide` не хранит состояние между барами: «память» о сработавшем RSI-триггере восстанавливается из серий на каждом баре. Подключение к движку бэктеста — через `Binding` в `internal/service/backtest` и `switch` в `cmd/backtest/main.go`, как у `reversion`.

**Tech Stack:** Go 1.25; индикаторы из `pkg/indicators` (`RSISeries`, `MACD`, `StochasticSeries`, `ATR`); движок `internal/domain/backtest`; тесты `go test`; линт/CI через `./bin/mage ci`.

**Spec:** `docs/superpowers/specs/2026-07-22-scalping-rsimacd-design.md`

## Global Constraints

- Ветка: `feat/scalping-rsimacd` (уже создана от `main`, спека закоммичена как `60bc8d1`).
- Пакет размещается в `internal/service/trading_strategy/scalping_rsimacd/strategy/core`, имя Go-пакета — `core`.
- `Decide` обязана быть **чистой**: никакого I/O, никакого состояния между вызовами, никаких полей-аккумуляторов в `Strategy` кроме `ticker` и `p Params`.
- Все поля `Params` — только `int` и `float64` (булевы флаги как `int` 0/1): по ним идёт reflection-калибровка.
- Стратегия **только лонг**. Шортов нет.
- Комментарии в коде — на английском (как во всём репо); пользовательские строки (`EntryReason`, `ExitReason`, `Explain`) — на русском, как в `reversion`.
- Коды выхода (`Signal.Reason`): `"SL"`, `"TP"`, `"STOCH"`, `"EOD"`. **Не добавлять `"STOCH"`/`"EOD"` в `model.IsStopReason`** — они обязаны заполняться по цене закрытия бара.
- Проверка сборки: `go build ./internal/... ./pkg/... ./cmd/...` (не `go build ./...` — падает на `magefiles`).
- Финальный гейт: `./bin/mage ci`.
- Фиксированные значения из спеки, копировать дословно: MACD(3,6,9), стохастик (14,3,3) с уровнем 80, сессия Пн–Чт `[08:00, 17:00)`, Пт `[08:00, 14:00)` MSK, `Lookback() = 120`.

---

### Task 1: Каркас пакета и сессионные правила

**Files:**
- Create: `internal/service/trading_strategy/scalping_rsimacd/strategy/core/core.go`
- Test: `internal/service/trading_strategy/scalping_rsimacd/strategy/core/core_test.go`

**Interfaces:**
- Consumes: `tinvest/internal/service/trading_strategy/scalping/strategy` (`MarketData`, `Position`), `tinvest/internal/service/trading_strategy/scalping/model` (`Signal`, `SignalKind`).
- Produces: `type Params struct`, `func DefaultParams() Params`, `type Strategy struct`, `func NewWithParams(ticker string, p Params) *Strategy`, методы `Ticker() string`, `Lookback() int`, `inSession(t time.Time) bool`, `isDayEnd(t time.Time) bool`, `barTime(md strategy.MarketData) time.Time`, пакетная переменная `mskLoc *time.Location`, константа `barSpanMin = 5`.

- [ ] **Step 1: Написать падающий тест**

Создать `internal/service/trading_strategy/scalping_rsimacd/strategy/core/core_test.go`:

```go
package core

import (
	"testing"
	"time"

	"tinvest/internal/service/trading_strategy/scalping/strategy"
)

// mskAt builds an MSK bar open-time for the given date and clock.
func mskAt(y int, m time.Month, d, hh, mm int) time.Time {
	return time.Date(y, m, d, hh, mm, 0, 0, mskLoc)
}

func TestInSession(t *testing.T) {
	s := NewWithParams("TEST", DefaultParams())
	cases := []struct {
		name string
		at   time.Time
		want bool
	}{
		// 2026-07-20 is a Monday, 2026-07-24 a Friday, 2026-07-25 a Saturday.
		{"mon before open", mskAt(2026, 7, 20, 7, 55), false},
		{"mon at open", mskAt(2026, 7, 20, 8, 0), true},
		{"mon midday", mskAt(2026, 7, 20, 12, 30), true},
		{"mon last bar", mskAt(2026, 7, 20, 16, 55), true},
		{"mon at close", mskAt(2026, 7, 20, 17, 0), false},
		{"mon after close", mskAt(2026, 7, 20, 17, 5), false},
		{"fri before friday close", mskAt(2026, 7, 24, 13, 55), true},
		{"fri at friday close", mskAt(2026, 7, 24, 14, 0), false},
		{"fri after friday close", mskAt(2026, 7, 24, 16, 0), false},
		{"saturday", mskAt(2026, 7, 25, 12, 0), false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := s.inSession(tc.at); got != tc.want {
				t.Fatalf("inSession(%v) = %v want %v", tc.at, got, tc.want)
			}
		})
	}
}

func TestInSessionZeroTimeIsPermissive(t *testing.T) {
	s := NewWithParams("TEST", DefaultParams())
	if !s.inSession(time.Time{}) {
		t.Fatalf("zero time must not block the entry gate")
	}
}

func TestIsDayEnd(t *testing.T) {
	s := NewWithParams("TEST", DefaultParams())
	cases := []struct {
		name string
		at   time.Time
		want bool
	}{
		{"mon midday", mskAt(2026, 7, 20, 12, 0), false},
		{"mon 16:50", mskAt(2026, 7, 20, 16, 50), false},
		{"mon 16:55 last bar closes at 17:00", mskAt(2026, 7, 20, 16, 55), true},
		{"mon 17:05", mskAt(2026, 7, 20, 17, 5), true},
		{"fri 13:50", mskAt(2026, 7, 24, 13, 50), false},
		{"fri 13:55 last bar closes at 14:00", mskAt(2026, 7, 24, 13, 55), true},
		{"saturday always day end", mskAt(2026, 7, 25, 10, 0), true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := s.isDayEnd(tc.at); got != tc.want {
				t.Fatalf("isDayEnd(%v) = %v want %v", tc.at, got, tc.want)
			}
		})
	}
}

func TestIsDayEndZeroTimeIsNoOp(t *testing.T) {
	s := NewWithParams("TEST", DefaultParams())
	if s.isDayEnd(time.Time{}) {
		t.Fatalf("zero time must degrade the EOD exit to a no-op")
	}
}

func TestBarTimeRequiresAlignedTimes(t *testing.T) {
	s := NewWithParams("TEST", DefaultParams())
	md := strategy.MarketData{
		Closes: []float64{1, 2, 3},
		Times:  []time.Time{mskAt(2026, 7, 20, 10, 0)}, // misaligned on purpose
	}
	if got := s.barTime(md); !got.IsZero() {
		t.Fatalf("misaligned Times must yield zero time, got %v", got)
	}
	md.Times = []time.Time{
		mskAt(2026, 7, 20, 10, 0),
		mskAt(2026, 7, 20, 10, 5),
		mskAt(2026, 7, 20, 10, 10),
	}
	if got := s.barTime(md); !got.Equal(mskAt(2026, 7, 20, 10, 10)) {
		t.Fatalf("barTime = %v want the latest bar time", got)
	}
}

func TestLookbackCoversIndicatorWarmup(t *testing.T) {
	s := NewWithParams("TEST", DefaultParams())
	if got := s.Lookback(); got != 120 {
		t.Fatalf("Lookback = %d want 120", got)
	}
}

func TestTickerRoundTrip(t *testing.T) {
	if got := NewWithParams("SBER", DefaultParams()).Ticker(); got != "SBER" {
		t.Fatalf("Ticker = %q want SBER", got)
	}
}
```

- [ ] **Step 2: Убедиться, что тест падает**

Run: `go test ./internal/service/trading_strategy/scalping_rsimacd/... 2>&1 | tail -20`
Expected: FAIL — пакет не собирается, «undefined: NewWithParams», «undefined: mskLoc».

- [ ] **Step 3: Минимальная реализация**

Создать `internal/service/trading_strategy/scalping_rsimacd/strategy/core/core.go`:

```go
// Package core implements a long-only 5-minute RSI+MACD scalping strategy. When flat it
// looks for a fast RSI crossing up out of its oversold zone, confirmed within a few bars
// by a MACD(3,6,9) bullish line cross that happens BELOW zero. The stop is frozen at the
// low of the RSI-cross candle; the take-profit is RR times the entry risk. An optional
// stochastic exit closes the position when %K leaves the overbought zone downward, and
// every position is force-closed at the end of the trading day. The decision logic is
// pure, stateless between bars and ticker-agnostic. Run with
// `-strategy scalping_rsimacd -interval Minutes5`.
package core

import (
	"time"

	"tinvest/internal/service/trading_strategy/scalping/strategy"
)

// barSpanMin is the bar length in minutes. The strategy is defined on 5-minute candles;
// the EOD gate uses it to close on the LAST bar that still ends inside the session.
const barSpanMin = 5

// Params holds every tunable. All fields are int or float64 (flags as int 0/1) so
// reflection grid calibration can sweep them.
type Params struct {
	RSIPeriod       int     // fast RSI length (grid 3/4/5)
	RSIOversold     float64 // lower critical zone (grid 20/25/30)
	MACDFast        int     // MACD fast EMA (fixed 3)
	MACDSlow        int     // MACD slow EMA (fixed 6)
	MACDSignal      int     // MACD signal EMA (fixed 9)
	MACDConfirmBars int     // MACD cross accepted on the RSI bar or the next N bars (grid 2/3/4)
	ATRPeriod       int     // ATR length; the unit of the risk sanity bounds
	StopBufferATR   float64 // stop = low(RSI bar) - StopBufferATR*ATR (grid 0/0.1)
	RR              float64 // take-profit = entry + RR*(entry-stop) (grid 2/2.5/3)
	MinRiskATR      float64 // reject entries whose risk < MinRiskATR*ATR
	MaxRiskATR      float64 // reject entries whose risk > MaxRiskATR*ATR
	EnableStochExit int     // 1 = stochastic exit active; 0 = SL/TP/EOD only (grid 0/1)
	StochK          int     // stochastic %K period (fixed 14)
	StochD          int     // stochastic %D smoothing (fixed 3)
	StochOverbought float64 // upper critical zone for the exit (fixed 80)
	SessionStartMin int     // entry window start, minutes from MSK midnight (480 = 08:00)
	SessionEndMin   int     // Mon-Thu session end, minutes from MSK midnight (1020 = 17:00)
	FridayEndMin    int     // Friday session end, minutes from MSK midnight (840 = 14:00)
}

// DefaultParams returns the spec's baseline; the swept values come from calibration.
func DefaultParams() Params {
	return Params{
		RSIPeriod:       3,
		RSIOversold:     30,
		MACDFast:        3,
		MACDSlow:        6,
		MACDSignal:      9,
		MACDConfirmBars: 3,
		ATRPeriod:       14,
		StopBufferATR:   0,
		RR:              2.0,
		MinRiskATR:      0.1,
		MaxRiskATR:      3.0,
		EnableStochExit: 1,
		StochK:          14,
		StochD:          3,
		StochOverbought: 80,
		SessionStartMin: 480,
		SessionEndMin:   1020,
		FridayEndMin:    840,
	}
}

// Strategy trades a single instrument with the RSI+MACD rules. Ticker-agnostic and pure.
type Strategy struct {
	ticker string
	p      Params
}

// NewWithParams returns the strategy for a ticker with explicit params.
func NewWithParams(ticker string, p Params) *Strategy { return &Strategy{ticker: ticker, p: p} }

func (s *Strategy) Ticker() string { return s.ticker }

// Lookback sizes the candle window to warm every consumer with margin: Wilder RSI
// (period 3-5), MACD(3,6,9), the stochastic (14+3) and the confirmation window.
func (s *Strategy) Lookback() int { return 120 }

// mskLoc anchors the session windows to the Moscow calendar (UTC fallback).
var mskLoc = func() *time.Location {
	loc, err := time.LoadLocation("Europe/Moscow")
	if err != nil {
		return time.UTC
	}
	return loc
}()

// sessionEndMin returns the session end for the weekday of tl (Friday closes early).
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
// (live paths without Times) skips the gate — never block on missing data.
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

// isDayEnd reports whether the bar opening at t is the last one that still ends inside
// the session (or already sits outside it). A zero time degrades the EOD exit to a no-op.
func (s *Strategy) isDayEnd(t time.Time) bool {
	if t.IsZero() {
		return false
	}
	tl := t.In(mskLoc)
	if isWeekend(tl) {
		return true
	}
	m := tl.Hour()*60 + tl.Minute()
	return m+barSpanMin >= s.sessionEndMin(tl)
}

// barTime returns the open-time of the latest bar, or the zero time when Times is absent
// or misaligned with Closes (so the time-based gates degrade instead of misfiring).
func (s *Strategy) barTime(md strategy.MarketData) time.Time {
	n := len(md.Closes)
	if n == 0 || len(md.Times) != n {
		return time.Time{}
	}
	return md.Times[n-1]
}
```

- [ ] **Step 4: Убедиться, что тесты проходят**

Run: `go test ./internal/service/trading_strategy/scalping_rsimacd/... -v 2>&1 | tail -30`
Expected: PASS по всем тестам Task 1.

- [ ] **Step 5: Коммит**

```bash
git add internal/service/trading_strategy/scalping_rsimacd/
git commit -m "feat(scalping-rsimacd): package skeleton, params and session rules"
```

---

### Task 2: Логика входа

**Files:**
- Modify: `internal/service/trading_strategy/scalping_rsimacd/strategy/core/core.go`
- Test: `internal/service/trading_strategy/scalping_rsimacd/strategy/core/core_test.go`

**Interfaces:**
- Consumes: из Task 1 — `Params`, `DefaultParams()`, `Strategy`, `NewWithParams`, `inSession`, `barTime`, `mskLoc`; из `pkg/indicators` — `RSISeries(closes []float64, period int) []float64`, `MACD(closes []float64, fast, slow, signalPeriod int) (macdLine, signalLine []float64)`, `ATR(highs, lows, closes []float64, period int) float64`.
- Produces: `func (s *Strategy) Decide(md strategy.MarketData) model.Signal`, `func (s *Strategy) enter(md strategy.MarketData, sig model.Signal) model.Signal`, `func (s *Strategy) lastRSITrigger(rsi []float64, n int) (int, bool)`, `func (s *Strategy) triggerAlive(md strategy.MarketData, t, n int, stop float64) bool`, `func (s *Strategy) entryReason(barsAgo int, rsiNow, stop, entry, tp, atr float64) string`.

**Важно про выравнивание индикаторов:** `RSISeries` и `MACD` возвращают серии **длиной с `closes`**, с нулями на прогреве. Нулевой RSI на прогреве меньше любого `RSIOversold`, поэтому проверка «RSI был в зоне» обязана требовать `rsi[t-1] > 0` — иначе прогревочный ноль создаст фантомный триггер. Это ловушка, из-за которой тест `TestNoEntryFromWarmupZeros` в наборе обязателен.

- [ ] **Step 1: Написать падающие тесты**

Дописать в `core_test.go` (импорты дополнить: `math`, `tinvest/internal/service/trading_strategy/scalping/model`, `tinvest/pkg/indicators`):

```go
// testParams are permissive defaults for entry tests: the risk sanity bounds are opened
// up so synthetic series never trip them, and the stop sits exactly on the trigger low.
func testParams() Params {
	p := DefaultParams()
	p.MinRiskATR = 0
	p.MaxRiskATR = 100
	p.StopBufferATR = 0
	return p
}

// declineThenRally builds a synthetic 5m series: `down` bars falling by step from start,
// then `up` bars rising by 2*step. Each bar's high/low is close +/- wick, so lows rise
// monotonically during the rally (the trigger is never invalidated by the rally itself).
func declineThenRally(down, up int, start, step, wick float64) (highs, lows, closes []float64) {
	price := start
	for i := 0; i < down; i++ {
		price -= step
		closes = append(closes, price)
	}
	for i := 0; i < up; i++ {
		price += 2 * step
		closes = append(closes, price)
	}
	for _, c := range closes {
		highs = append(highs, c+wick)
		lows = append(lows, c-wick)
	}
	return highs, lows, closes
}

// sessionTimes returns len(n) MSK bar open-times, 5 minutes apart, starting Mon 10:00.
func sessionTimes(n int) []time.Time {
	out := make([]time.Time, n)
	base := mskAt(2026, 7, 20, 10, 0)
	for i := range out {
		out[i] = base.Add(time.Duration(i*barSpanMin) * time.Minute)
	}
	return out
}

// mdPrefix builds MarketData over bars [0, i] of the given series (flat position).
func mdPrefix(highs, lows, closes []float64, times []time.Time, i int) strategy.MarketData {
	return strategy.MarketData{
		Price:  closes[i],
		Highs:  highs[:i+1],
		Lows:   lows[:i+1],
		Closes: closes[:i+1],
		Times:  times[:i+1],
	}
}

// firstBuy walks the series bar by bar and returns the first Buy signal and its index.
func firstBuy(s *Strategy, highs, lows, closes []float64, times []time.Time) (model.Signal, int, bool) {
	for i := 40; i < len(closes); i++ {
		sig := s.Decide(mdPrefix(highs, lows, closes, times, i))
		if sig.Kind == model.SignalBuy {
			return sig, i, true
		}
	}
	return model.Signal{}, 0, false
}

func TestEntryFiresOnRSIThenMACDCross(t *testing.T) {
	highs, lows, closes := declineThenRally(60, 8, 100, 0.5, 0.2)
	times := sessionTimes(len(closes))
	s := NewWithParams("TEST", testParams())

	sig, i, ok := firstBuy(s, highs, lows, closes, times)
	if !ok {
		t.Fatalf("no entry fired on a decline-then-rally series")
	}
	if i < 60 || i > 66 {
		t.Fatalf("entry at bar %d, want it inside the rally (60..66)", i)
	}

	// The MACD cross must be on the entry bar and both lines below zero.
	macd, signal := indicators.MACD(closes[:i+1], 3, 6, 9)
	if !(macd[i-1] <= signal[i-1] && macd[i] > signal[i]) {
		t.Fatalf("entry bar %d is not a MACD bullish cross", i)
	}
	if macd[i] >= 0 || signal[i] >= 0 {
		t.Fatalf("MACD lines must be below zero at entry: macd=%v signal=%v", macd[i], signal[i])
	}

	// The stop is the low of the RSI-cross bar, which lies within the confirm window.
	rsi := indicators.RSISeries(closes[:i+1], s.p.RSIPeriod)
	trig, found := s.lastRSITrigger(rsi, i+1)
	if !found {
		t.Fatalf("no RSI trigger found for entry bar %d", i)
	}
	if math.Abs(sig.StopLoss-lows[trig]) > 1e-9 {
		t.Fatalf("stop = %v want low of the RSI bar %v", sig.StopLoss, lows[trig])
	}
	wantTP := closes[i] + s.p.RR*(closes[i]-sig.StopLoss)
	if math.Abs(sig.TakeProfit-wantTP) > 1e-9 {
		t.Fatalf("tp = %v want %v", sig.TakeProfit, wantTP)
	}
	if sig.EntryReason == "" {
		t.Fatalf("EntryReason must be filled for the trade journal")
	}
}

func TestNoEntryOutsideSession(t *testing.T) {
	highs, lows, closes := declineThenRally(60, 8, 100, 0.5, 0.2)
	// Same series, but every bar sits on a Saturday.
	times := make([]time.Time, len(closes))
	base := mskAt(2026, 7, 25, 10, 0)
	for i := range times {
		times[i] = base.Add(time.Duration(i*barSpanMin) * time.Minute)
	}
	s := NewWithParams("TEST", testParams())
	if _, _, ok := firstBuy(s, highs, lows, closes, times); ok {
		t.Fatalf("entry fired on a Saturday")
	}
}

func TestEveryEntryHasBothMACDLinesBelowZero(t *testing.T) {
	// A long rally eventually drags MACD above zero; no Buy may fire on such a cross.
	highs, lows, closes := declineThenRally(60, 40, 100, 0.5, 0.2)
	times := sessionTimes(len(closes))
	s := NewWithParams("TEST", testParams())

	var buys int
	for i := 40; i < len(closes); i++ {
		sig := s.Decide(mdPrefix(highs, lows, closes, times, i))
		if sig.Kind != model.SignalBuy {
			continue
		}
		buys++
		macd, signal := indicators.MACD(closes[:i+1], 3, 6, 9)
		if macd[i] >= 0 || signal[i] >= 0 {
			t.Fatalf("Buy at bar %d with MACD above zero: macd=%v signal=%v", i, macd[i], signal[i])
		}
	}
	if buys == 0 {
		t.Fatalf("no Buy fired at all; the assertion above would be vacuous")
	}
}

func TestLastRSITriggerWindowBoundary(t *testing.T) {
	p := testParams()
	p.RSIOversold = 30
	p.MACDConfirmBars = 3
	s := NewWithParams("TEST", p)

	// rsi[3] crosses up through 30 (25 -> 40). n-1 is the current bar.
	rsi := []float64{20, 22, 25, 40, 45, 50, 55, 60}
	if idx, ok := s.lastRSITrigger(rsi, 7); !ok || idx != 3 { // current bar 6 -> t=3 is the edge
		t.Fatalf("cross at the window edge: idx=%d ok=%v want 3,true", idx, ok)
	}
	if _, ok := s.lastRSITrigger(rsi, 8); ok { // current bar 7 -> t=3 is one bar too old
		t.Fatalf("cross one bar past the window must not be accepted")
	}
}

func TestLastRSITriggerPicksMostRecent(t *testing.T) {
	p := testParams()
	p.RSIOversold = 30
	p.MACDConfirmBars = 3
	s := NewWithParams("TEST", p)
	// Two crosses: at index 1 and at index 3. The most recent one wins.
	rsi := []float64{20, 40, 25, 45, 50}
	if idx, ok := s.lastRSITrigger(rsi, 5); !ok || idx != 3 {
		t.Fatalf("idx=%d ok=%v want 3,true", idx, ok)
	}
}

func TestNoEntryFromWarmupZeros(t *testing.T) {
	s := NewWithParams("TEST", testParams())
	// RSISeries fills warm-up positions with 0. A zero is below any oversold level, so a
	// naive comparison would read bar 1 as "crossed up out of the zone".
	rsi := []float64{0, 55, 60, 62}
	if idx, ok := s.lastRSITrigger(rsi, 4); ok {
		t.Fatalf("warm-up zero produced a phantom trigger at %d", idx)
	}
}

func TestTriggerInvalidatedByStopBreak(t *testing.T) {
	s := NewWithParams("TEST", testParams())
	md := strategy.MarketData{
		Lows:   []float64{10, 9, 8.5, 9.5},
		Closes: []float64{10.5, 9.5, 9.0, 10.0},
	}
	// Trigger at bar 1 with stop 9: bar 2's low 8.5 breaks it before the entry.
	if s.triggerAlive(md, 1, 4, 9) {
		t.Fatalf("a low below the stop between the trigger and the entry must invalidate it")
	}
	// Trigger at bar 2 with stop 8.5: nothing after it breaks the level.
	if !s.triggerAlive(md, 2, 4, 8.5) {
		t.Fatalf("intact trigger reported as invalidated")
	}
	// Entry close at or below the stop is never valid.
	if s.triggerAlive(md, 2, 4, 10.0) {
		t.Fatalf("close at the stop level must invalidate the entry")
	}
}

func TestRiskSanityBounds(t *testing.T) {
	highs, lows, closes := declineThenRally(60, 8, 100, 0.5, 0.2)
	times := sessionTimes(len(closes))

	// Baseline: the permissive params do produce an entry.
	if _, _, ok := firstBuy(NewWithParams("TEST", testParams()), highs, lows, closes, times); !ok {
		t.Fatalf("baseline entry missing; the other assertions would be vacuous")
	}

	tight := testParams()
	tight.MinRiskATR = 100 // no realistic risk can clear this
	if _, _, ok := firstBuy(NewWithParams("TEST", tight), highs, lows, closes, times); ok {
		t.Fatalf("entry fired despite risk below MinRiskATR")
	}

	wide := testParams()
	wide.MaxRiskATR = 0.001 // any realistic risk exceeds this
	if _, _, ok := firstBuy(NewWithParams("TEST", wide), highs, lows, closes, times); ok {
		t.Fatalf("entry fired despite risk above MaxRiskATR")
	}
}

func TestStopBufferWidensTheStop(t *testing.T) {
	highs, lows, closes := declineThenRally(60, 8, 100, 0.5, 0.2)
	times := sessionTimes(len(closes))

	plain, _, ok := firstBuy(NewWithParams("TEST", testParams()), highs, lows, closes, times)
	if !ok {
		t.Fatalf("baseline entry missing")
	}
	p := testParams()
	p.StopBufferATR = 0.5
	buffered, _, ok := firstBuy(NewWithParams("TEST", p), highs, lows, closes, times)
	if !ok {
		t.Fatalf("buffered entry missing")
	}
	if !(buffered.StopLoss < plain.StopLoss) {
		t.Fatalf("buffered stop %v must sit below the plain stop %v", buffered.StopLoss, plain.StopLoss)
	}
}

func TestNoEntryWhenAlreadyInPosition(t *testing.T) {
	highs, lows, closes := declineThenRally(60, 8, 100, 0.5, 0.2)
	times := sessionTimes(len(closes))
	s := NewWithParams("TEST", testParams())
	for i := 40; i < len(closes); i++ {
		md := mdPrefix(highs, lows, closes, times, i)
		md.Position = &strategy.Position{PurchasePrice: closes[i], Quantity: 1}
		if sig := s.Decide(md); sig.Kind == model.SignalBuy {
			t.Fatalf("Buy emitted while a position is open (bar %d)", i)
		}
	}
}
```

- [ ] **Step 2: Убедиться, что тесты падают**

Run: `go test ./internal/service/trading_strategy/scalping_rsimacd/... 2>&1 | tail -20`
Expected: FAIL — «undefined: (*Strategy).Decide», «undefined: lastRSITrigger», «undefined: triggerAlive».

- [ ] **Step 3: Реализовать вход**

Дописать в `core.go` (в импорты добавить `fmt`, `tinvest/internal/service/trading_strategy/scalping/model`, `tinvest/pkg/indicators`):

```go
// Decide routes to entry (flat) or position management (open).
func (s *Strategy) Decide(md strategy.MarketData) model.Signal {
	sig := model.Signal{Ticker: s.ticker, Price: md.Price}
	if md.Position != nil {
		return s.manage(md, sig) // implemented in Task 3
	}
	return s.enter(md, sig)
}

// enter emits a long when the current bar carries a below-zero MACD bullish cross that
// confirms a recent RSI cross up out of the oversold zone, and the trigger's stop level
// has held since. Everything is recomputed from md — no state survives between bars.
func (s *Strategy) enter(md strategy.MarketData, sig model.Signal) model.Signal {
	n := len(md.Closes)
	if n < 2 || len(md.Highs) != n || len(md.Lows) != n {
		return sig
	}
	// 1. session window (skipped when Times is absent or misaligned).
	if !s.inSession(s.barTime(md)) {
		return sig
	}
	// 2. MACD bullish cross on the current bar, both lines below zero.
	macd, signal := indicators.MACD(md.Closes, s.p.MACDFast, s.p.MACDSlow, s.p.MACDSignal)
	if len(macd) != n || len(signal) != n {
		return sig
	}
	i := n - 1
	if !(macd[i-1] <= signal[i-1] && macd[i] > signal[i]) {
		return sig
	}
	if macd[i] >= 0 || signal[i] >= 0 {
		return sig
	}
	// 3. the RSI trigger that this cross confirms.
	rsi := indicators.RSISeries(md.Closes, s.p.RSIPeriod)
	if len(rsi) != n {
		return sig
	}
	trig, ok := s.lastRSITrigger(rsi, n)
	if !ok {
		return sig
	}
	// 4. ATR unit for the stop buffer and the risk sanity bounds.
	atr := indicators.ATR(md.Highs, md.Lows, md.Closes, s.p.ATRPeriod)
	if atr <= 0 {
		return sig
	}
	stop := md.Lows[trig] - s.p.StopBufferATR*atr
	// 5. the trigger's stop level must have held since the trigger bar.
	if !s.triggerAlive(md, trig, n, stop) {
		return sig
	}
	// 6. risk sanity: reject degenerate and abnormally wide stops.
	entry := md.Closes[i]
	risk := entry - stop
	if risk < s.p.MinRiskATR*atr || risk > s.p.MaxRiskATR*atr {
		return sig
	}

	tp := entry + s.p.RR*risk
	sig.Kind = model.SignalBuy
	sig.StopLoss = stop
	sig.TakeProfit = tp
	sig.ATR = atr
	sig.RSI = rsi[i]
	sig.Level = md.Lows[trig]
	sig.EntryReason = s.entryReason(i-trig, rsi[i], stop, entry, tp, atr)
	return sig
}

// lastRSITrigger returns the most recent bar in [n-1-MACDConfirmBars, n-1] on which RSI
// crossed up through the oversold level and closed above it. The rsi[j-1] > 0 guard is
// load-bearing: RSISeries fills warm-up positions with 0, which would otherwise read as
// "was inside the oversold zone" and manufacture a phantom trigger.
func (s *Strategy) lastRSITrigger(rsi []float64, n int) (int, bool) {
	lo := n - 1 - s.p.MACDConfirmBars
	if lo < 1 {
		lo = 1
	}
	for j := n - 1; j >= lo; j-- {
		if rsi[j-1] > 0 && rsi[j-1] < s.p.RSIOversold && rsi[j] > s.p.RSIOversold {
			return j, true
		}
	}
	return 0, false
}

// triggerAlive reports whether the stop level has held on every bar after the trigger and
// the entry close still sits above it. A bar that already traded through the stop means
// the setup was stopped out before it could be entered.
func (s *Strategy) triggerAlive(md strategy.MarketData, t, n int, stop float64) bool {
	for j := t + 1; j <= n-1; j++ {
		if md.Lows[j] <= stop {
			return false
		}
	}
	return md.Closes[n-1] > stop
}

// entryReason renders the rationale shown in the trade journal. barsAgo is the distance
// from the RSI trigger bar to the entry bar (0 = same bar).
func (s *Strategy) entryReason(barsAgo int, rsiNow, stop, entry, tp, atr float64) string {
	return fmt.Sprintf(
		"RSI(%d) вышел вверх из зоны %.0f (сейчас %.1f), через %d бар(ов) MACD(%d,%d,%d) пересёкся ниже нуля; вход %.4f, стоп %.4f (лоу свечи кросса, буфер %.2f×ATR), тейк %.4f (RR=%.2f), ATR=%.4f",
		s.p.RSIPeriod, s.p.RSIOversold, rsiNow, barsAgo, s.p.MACDFast, s.p.MACDSlow, s.p.MACDSignal,
		entry, stop, s.p.StopBufferATR, tp, s.p.RR, atr,
	)
}
```

Временно добавить заглушку `manage`, чтобы пакет собирался (полная реализация — Task 3):

```go
// manage handles an open long. Implemented in Task 3.
func (s *Strategy) manage(md strategy.MarketData, sig model.Signal) model.Signal {
	return sig
}
```

- [ ] **Step 4: Убедиться, что тесты проходят**

Run: `go test ./internal/service/trading_strategy/scalping_rsimacd/... -v 2>&1 | tail -40`
Expected: PASS по всем тестам Task 1 и Task 2.

Если `TestEntryFiresOnRSIThenMACDCross` падает с «no entry fired» — **не подгонять пороги вслепую**. Добавить временный `t.Logf` внутрь цикла `firstBuy`, печатающий `i`, `rsi[i]`, `macd[i]`, `signal[i]` для баров 55..68, запустить `go test ... -run TestEntryFires -v` и определить, какой именно гейт не пускает. После диагностики лог удалить. Если окажется, что синтетический ралли слишком пологий для MACD-кросса, увеличить в тесте параметр `up` (число баров ралли) или `step` — но не менять `RSIOversold`/`MACDConfirmBars`: они фиксированы спекой.

- [ ] **Step 5: Коммит**

```bash
git add internal/service/trading_strategy/scalping_rsimacd/
git commit -m "feat(scalping-rsimacd): entry — RSI oversold exit confirmed by a below-zero MACD cross"
```

---

### Task 3: Выходы — SL, TP, стохастик, EOD

**Files:**
- Modify: `internal/service/trading_strategy/scalping_rsimacd/strategy/core/core.go`
- Test: `internal/service/trading_strategy/scalping_rsimacd/strategy/core/core_test.go`

**Interfaces:**
- Consumes: из Task 1 — `isDayEnd`, `barTime`; из Task 2 — `Decide` (маршрутизация в `manage`); из `pkg/indicators` — `StochasticSeries(highs, lows, closes []float64, kPeriod, dSmooth int) (ks, ds []float64)`; из `scalping/strategy` — `Position` (`StopLoss`, `TakeProfit`, `PurchasePrice`).
- Produces: полная реализация `func (s *Strategy) manage(md strategy.MarketData, sig model.Signal) model.Signal` и `func (s *Strategy) stochExit(md strategy.MarketData) bool`.

**Важно про выравнивание стохастика:** `StochasticSeries` **правовыровнена и короче** серии цен (`len(ks) = n - kPeriod + 1`). Текущий бар — `ks[len(ks)-1]`, предыдущий — `ks[len(ks)-2]`. Индексировать `ks` номером бара нельзя.

**Важно про коды выхода:** `"SL"` заполняется движком по `min(StopLoss, open)`, `"TP"` — по `max(TakeProfit, open)`, всё остальное — по закрытию бара. `"STOCH"` и `"EOD"` обязаны заполняться по закрытию, поэтому в `model.IsStopReason` их добавлять **нельзя**.

- [ ] **Step 1: Написать падающие тесты**

Дописать в `core_test.go`:

```go
// openPos builds MarketData for an open long on a single trailing bar.
func openPos(high, low, close float64, at time.Time, stop, tp float64) strategy.MarketData {
	md := strategy.MarketData{
		Price:  close,
		Highs:  []float64{high},
		Lows:   []float64{low},
		Closes: []float64{close},
		Times:  []time.Time{at},
		Position: &strategy.Position{
			PurchasePrice: 100,
			Quantity:      1,
			StopLoss:      stop,
			TakeProfit:    tp,
		},
	}
	return md
}

func TestExitStopLoss(t *testing.T) {
	s := NewWithParams("TEST", DefaultParams())
	md := openPos(101, 97, 98, mskAt(2026, 7, 20, 12, 0), 98.5, 110)
	sig := s.Decide(md)
	if sig.Kind != model.SignalSell || sig.Reason != "SL" {
		t.Fatalf("kind=%v reason=%q want Sell/SL", sig.Kind, sig.Reason)
	}
	if sig.StopLoss != 98.5 {
		t.Fatalf("StopLoss=%v want 98.5 (engine fills stops at min(level, open))", sig.StopLoss)
	}
}

func TestExitTakeProfit(t *testing.T) {
	s := NewWithParams("TEST", DefaultParams())
	md := openPos(110.5, 104, 110, mskAt(2026, 7, 20, 12, 0), 95, 110)
	sig := s.Decide(md)
	if sig.Kind != model.SignalSell || sig.Reason != "TP" {
		t.Fatalf("kind=%v reason=%q want Sell/TP", sig.Kind, sig.Reason)
	}
	if sig.TakeProfit != 110 {
		t.Fatalf("TakeProfit=%v want 110", sig.TakeProfit)
	}
}

func TestStopLossWinsWhenBothTouch(t *testing.T) {
	s := NewWithParams("TEST", DefaultParams())
	// The bar spans both levels; the conservative assumption is that the stop hit first.
	md := openPos(112, 94, 105, mskAt(2026, 7, 20, 12, 0), 95, 110)
	sig := s.Decide(md)
	if sig.Reason != "SL" {
		t.Fatalf("reason=%q want SL to win a same-bar SL/TP touch", sig.Reason)
	}
}

// stochSeries builds a series whose %K sits above 80 on the second-to-last bar and drops
// below 80 on the last one: a long flat-high range, then a close near the top, then a
// close near the bottom of the same range.
func stochSeries() (highs, lows, closes []float64) {
	const n = 20
	for i := 0; i < n; i++ {
		highs = append(highs, 110)
		lows = append(lows, 100)
		closes = append(closes, 105)
	}
	closes[n-2] = 109.5 // %K = 95
	closes[n-1] = 102   // %K = 20
	return highs, lows, closes
}

func TestExitStochasticCrossDown(t *testing.T) {
	highs, lows, closes := stochSeries()
	n := len(closes)
	times := sessionTimes(n)
	s := NewWithParams("TEST", DefaultParams())

	md := strategy.MarketData{
		Price:  closes[n-1],
		Highs:  highs,
		Lows:   lows,
		Closes: closes,
		Times:  times,
		Position: &strategy.Position{
			PurchasePrice: 100, Quantity: 1, StopLoss: 90, TakeProfit: 200,
		},
	}
	sig := s.Decide(md)
	if sig.Kind != model.SignalSell || sig.Reason != "STOCH" {
		t.Fatalf("kind=%v reason=%q want Sell/STOCH", sig.Kind, sig.Reason)
	}
	if model.IsStopReason(sig.Reason) {
		t.Fatalf("STOCH must not be a stop-style reason: it fills at the bar close")
	}
}

func TestStochasticExitDisabled(t *testing.T) {
	highs, lows, closes := stochSeries()
	n := len(closes)
	p := DefaultParams()
	p.EnableStochExit = 0
	s := NewWithParams("TEST", p)

	md := strategy.MarketData{
		Price:  closes[n-1],
		Highs:  highs,
		Lows:   lows,
		Closes: closes,
		Times:  sessionTimes(n),
		Position: &strategy.Position{
			PurchasePrice: 100, Quantity: 1, StopLoss: 90, TakeProfit: 200,
		},
	}
	if sig := s.Decide(md); sig.Kind == model.SignalSell {
		t.Fatalf("stochastic exit fired while EnableStochExit=0 (reason=%q)", sig.Reason)
	}
}

func TestExitEndOfDay(t *testing.T) {
	s := NewWithParams("TEST", DefaultParams())
	cases := []struct {
		name string
		at   time.Time
		want bool
	}{
		{"monday midday", mskAt(2026, 7, 20, 12, 0), false},
		{"monday last bar", mskAt(2026, 7, 20, 16, 55), true},
		{"friday midday", mskAt(2026, 7, 24, 12, 0), false},
		{"friday last bar", mskAt(2026, 7, 24, 13, 55), true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			md := openPos(101, 99, 100, tc.at, 90, 200)
			sig := s.Decide(md)
			got := sig.Kind == model.SignalSell && sig.Reason == "EOD"
			if got != tc.want {
				t.Fatalf("EOD exit = %v want %v (reason=%q)", got, tc.want, sig.Reason)
			}
		})
	}
}

func TestNoExitWithoutTimes(t *testing.T) {
	s := NewWithParams("TEST", DefaultParams())
	md := openPos(101, 99, 100, time.Time{}, 90, 200)
	md.Times = nil
	if sig := s.Decide(md); sig.Kind == model.SignalSell {
		t.Fatalf("missing Times must degrade the EOD exit to a no-op, got reason=%q", sig.Reason)
	}
}
```

- [ ] **Step 2: Убедиться, что тесты падают**

Run: `go test ./internal/service/trading_strategy/scalping_rsimacd/... 2>&1 | tail -20`
Expected: FAIL — `kind=0 reason="" want Sell/SL` и аналогичные (заглушка `manage` из Task 2 ничего не возвращает).

- [ ] **Step 3: Реализовать выходы**

Заменить заглушку `manage` в `core.go` на полную реализацию:

```go
// manage handles an open long, exiting in precedence SL -> TP -> stochastic -> EOD. SL and
// TP are read from the position (frozen at entry by the engine), never recomputed from
// Params. The stochastic and EOD exits fill at the bar close — their reasons are
// deliberately absent from model.IsStopReason.
func (s *Strategy) manage(md strategy.MarketData, sig model.Signal) model.Signal {
	pos := md.Position
	n := len(md.Lows)
	if pos == nil || n == 0 || len(md.Highs) != n || len(md.Closes) != n {
		return sig
	}
	low, high := md.Lows[n-1], md.Highs[n-1]

	switch {
	case pos.StopLoss > 0 && low <= pos.StopLoss:
		sig.Kind, sig.Reason = model.SignalSell, "SL"
		sig.StopLoss = pos.StopLoss
		sig.ExitReason = fmt.Sprintf("SL: low %.4f ≤ стоп %.4f (вход %.4f)", low, pos.StopLoss, pos.PurchasePrice)
	case pos.TakeProfit > 0 && high >= pos.TakeProfit:
		sig.Kind, sig.Reason = model.SignalSell, "TP"
		sig.TakeProfit = pos.TakeProfit
		sig.ExitReason = fmt.Sprintf("TP: high %.4f ≥ тейк %.4f (вход %.4f)", high, pos.TakeProfit, pos.PurchasePrice)
	case s.p.EnableStochExit == 1 && s.stochExit(md):
		sig.Kind, sig.Reason = model.SignalSell, "STOCH"
		sig.ExitReason = fmt.Sprintf("STOCH: %%K вышел вниз из зоны %.0f, выход по закрытию %.4f (вход %.4f)",
			s.p.StochOverbought, md.Closes[n-1], pos.PurchasePrice)
	case s.isDayEnd(s.barTime(md)):
		sig.Kind, sig.Reason = model.SignalSell, "EOD"
		sig.ExitReason = fmt.Sprintf("EOD: закрытие на конец торгового дня по %.4f (вход %.4f)",
			md.Closes[n-1], pos.PurchasePrice)
	}
	return sig
}

// stochExit reports whether %K left the overbought zone downward on the current bar.
// StochasticSeries is right-aligned and shorter than the price series, so the two latest
// values are read from the tail — never by bar index.
func (s *Strategy) stochExit(md strategy.MarketData) bool {
	ks, _ := indicators.StochasticSeries(md.Highs, md.Lows, md.Closes, s.p.StochK, s.p.StochD)
	if len(ks) < 2 {
		return false
	}
	prev, now := ks[len(ks)-2], ks[len(ks)-1]
	return prev >= s.p.StochOverbought && now < s.p.StochOverbought
}
```

- [ ] **Step 4: Убедиться, что тесты проходят**

Run: `go test ./internal/service/trading_strategy/scalping_rsimacd/... -v 2>&1 | tail -40`
Expected: PASS по всем тестам Tasks 1–3.

- [ ] **Step 5: Коммит**

```bash
git add internal/service/trading_strategy/scalping_rsimacd/
git commit -m "feat(scalping-rsimacd): exits — SL/TP precedence, stochastic cross-down, forced EOD close"
```

---

### Task 4: Explain для режима `-explain`

**Files:**
- Modify: `internal/service/trading_strategy/scalping_rsimacd/strategy/core/core.go`
- Test: `internal/service/trading_strategy/scalping_rsimacd/strategy/core/core_test.go`

**Interfaces:**
- Consumes: из Tasks 1–3 — `inSession`, `barTime`, `lastRSITrigger`, `triggerAlive`, `stochExit`.
- Produces: `func (s *Strategy) Explain(md strategy.MarketData) string` — удовлетворяет интерфейсу `Explainer` из `internal/domain/backtest/engine.go:162` (`Explain(md strategy.MarketData) string`), который использует `domain.Trace`.

- [ ] **Step 1: Написать падающий тест**

Дописать в `core_test.go` (в импорты добавить `strings`):

```go
func TestExplainReportsEveryGate(t *testing.T) {
	highs, lows, closes := declineThenRally(60, 8, 100, 0.5, 0.2)
	times := sessionTimes(len(closes))
	s := NewWithParams("TEST", testParams())

	out := s.Explain(mdPrefix(highs, lows, closes, times, 63))
	for _, want := range []string{"сессия", "MACD", "RSI", "ATR", "стоп", "риск"} {
		if !strings.Contains(out, want) {
			t.Fatalf("Explain output missing %q:\n%s", want, out)
		}
	}
}

func TestExplainHandlesShortHistory(t *testing.T) {
	s := NewWithParams("TEST", DefaultParams())
	md := strategy.MarketData{
		Price:  10,
		Highs:  []float64{10},
		Lows:   []float64{9},
		Closes: []float64{10},
		Times:  []time.Time{mskAt(2026, 7, 20, 12, 0)},
	}
	if out := s.Explain(md); out == "" {
		t.Fatalf("Explain must always return a diagnosis, even on short history")
	}
}
```

- [ ] **Step 2: Убедиться, что тест падает**

Run: `go test ./internal/service/trading_strategy/scalping_rsimacd/... -run Explain 2>&1 | tail -10`
Expected: FAIL — «s.Explain undefined».

- [ ] **Step 3: Реализовать Explain**

Дописать в `core.go` (в импорты добавить `strings`):

```go
// Explain returns a gate-by-gate verdict for one bar, consumed by the engine's Trace
// (-explain). It recomputes the same values enter() does and reports pass/fail per gate.
func (s *Strategy) Explain(md strategy.MarketData) string {
	var sb strings.Builder
	n := len(md.Closes)
	if n < 2 || len(md.Highs) != n || len(md.Lows) != n {
		return "недостаточно свечей\n"
	}
	fmt.Fprintf(&sb, "сессия: %v (бар %v)\n", s.inSession(s.barTime(md)), s.barTime(md))

	macd, signal := indicators.MACD(md.Closes, s.p.MACDFast, s.p.MACDSlow, s.p.MACDSignal)
	if len(macd) != n || len(signal) != n {
		sb.WriteString("MACD: недостаточно истории\n")
		return sb.String()
	}
	i := n - 1
	crossed := macd[i-1] <= signal[i-1] && macd[i] > signal[i]
	fmt.Fprintf(&sb, "MACD(%d,%d,%d): линия %.5f сигнал %.5f, пересечение вверх? %v; обе ниже нуля? %v\n",
		s.p.MACDFast, s.p.MACDSlow, s.p.MACDSignal, macd[i], signal[i], crossed, macd[i] < 0 && signal[i] < 0)

	rsi := indicators.RSISeries(md.Closes, s.p.RSIPeriod)
	if len(rsi) != n {
		sb.WriteString("RSI: недостаточно истории\n")
		return sb.String()
	}
	fmt.Fprintf(&sb, "RSI(%d) сейчас %.1f, зона %.0f\n", s.p.RSIPeriod, rsi[i], s.p.RSIOversold)
	trig, ok := s.lastRSITrigger(rsi, n)
	if !ok {
		fmt.Fprintf(&sb, "RSI-триггер в окне %d бар(ов): нет\n", s.p.MACDConfirmBars)
		return sb.String()
	}
	fmt.Fprintf(&sb, "RSI-триггер: бар -%d (RSI %.1f -> %.1f)\n", i-trig, rsi[trig-1], rsi[trig])

	atr := indicators.ATR(md.Highs, md.Lows, md.Closes, s.p.ATRPeriod)
	fmt.Fprintf(&sb, "ATR(%d): %.4f\n", s.p.ATRPeriod, atr)
	if atr <= 0 {
		return sb.String()
	}
	stop := md.Lows[trig] - s.p.StopBufferATR*atr
	fmt.Fprintf(&sb, "стоп %.4f (лоу свечи кросса %.4f - %.2f×ATR); уровень удержан? %v\n",
		stop, md.Lows[trig], s.p.StopBufferATR, s.triggerAlive(md, trig, n, stop))

	risk := md.Closes[i] - stop
	fmt.Fprintf(&sb, "риск %.4f в границах [%.4f..%.4f]? %v; тейк %.4f (RR=%.2f)\n",
		risk, s.p.MinRiskATR*atr, s.p.MaxRiskATR*atr,
		risk >= s.p.MinRiskATR*atr && risk <= s.p.MaxRiskATR*atr,
		md.Closes[i]+s.p.RR*risk, s.p.RR)

	if s.p.EnableStochExit == 1 {
		fmt.Fprintf(&sb, "выход по стохастику(%d,%d) на этом баре? %v\n", s.p.StochK, s.p.StochD, s.stochExit(md))
	}
	fmt.Fprintf(&sb, "конец дня? %v\n", s.isDayEnd(s.barTime(md)))
	return sb.String()
}
```

- [ ] **Step 4: Убедиться, что тесты проходят**

Run: `go test ./internal/service/trading_strategy/scalping_rsimacd/... -v 2>&1 | tail -20`
Expected: PASS по всем тестам Tasks 1–4.

- [ ] **Step 5: Коммит**

```bash
git add internal/service/trading_strategy/scalping_rsimacd/
git commit -m "feat(scalping-rsimacd): Explain reports every entry gate for -explain"
```

---

### Task 5: Подключение к бэктесту, грид и документация

**Files:**
- Create: `internal/service/backtest/scalping_rsimacd_registry.go`
- Create: `internal/service/backtest/scalping_rsimacd_registry_test.go`
- Create: `data/params/scalping_rsimacd/grid.json`
- Create: `docs/scalping_rsimacd/strategy.md`
- Modify: `cmd/backtest/main.go` (флаг `-strategy` на строке 41, `switch` на строках 153–159)
- Modify: `CLAUDE.md` (строка про `trading_strategy/` в разделе Layout)

**Interfaces:**
- Consumes: `core.Params`, `core.DefaultParams()`, `core.NewWithParams(ticker, p)` из Tasks 1–4; `backtest.Binding` (поля `DefaultParams func() any`, `Build func(any) strategy.Strategy`, `ParseParams func([]byte) (any, error)`).
- Produces: `func ScalpingRSIMACDLookupOrGeneric(ticker string) Binding` в пакете `backtest`; CLI-значение `-strategy scalping_rsimacd`.

- [ ] **Step 1: Написать падающий тест реестра**

Создать `internal/service/backtest/scalping_rsimacd_registry_test.go`:

```go
package backtest

import (
	"testing"

	"tinvest/internal/service/trading_strategy/scalping_rsimacd/strategy/core"
)

func TestScalpingRSIMACDBindingDefaults(t *testing.T) {
	b := ScalpingRSIMACDLookupOrGeneric("SBER")
	got, ok := b.DefaultParams().(core.Params)
	if !ok {
		t.Fatalf("DefaultParams type = %T want core.Params", b.DefaultParams())
	}
	if got != core.DefaultParams() {
		t.Fatalf("defaults = %+v\nwant %+v", got, core.DefaultParams())
	}
	if s := b.Build(got); s.Ticker() != "SBER" {
		t.Fatalf("ticker = %q want SBER", s.Ticker())
	}
}

func TestScalpingRSIMACDParseParamsLayersOverDefaults(t *testing.T) {
	b := ScalpingRSIMACDLookupOrGeneric("SBER")
	raw := []byte(`{"RSIPeriod": 5, "RR": 3.0}`)
	parsed, err := b.ParseParams(raw)
	if err != nil {
		t.Fatalf("ParseParams: %v", err)
	}
	got := parsed.(core.Params)
	if got.RSIPeriod != 5 || got.RR != 3.0 {
		t.Fatalf("overrides not applied: %+v", got)
	}
	if got.MACDSlow != core.DefaultParams().MACDSlow {
		t.Fatalf("untouched field must keep its default: MACDSlow=%d", got.MACDSlow)
	}
}

func TestScalpingRSIMACDParseParamsRejectsGarbage(t *testing.T) {
	b := ScalpingRSIMACDLookupOrGeneric("SBER")
	if _, err := b.ParseParams([]byte(`not json`)); err == nil {
		t.Fatalf("want an error on malformed params JSON")
	}
}
```

- [ ] **Step 2: Убедиться, что тест падает**

Run: `go test ./internal/service/backtest/ -run ScalpingRSIMACD 2>&1 | tail -10`
Expected: FAIL — «undefined: ScalpingRSIMACDLookupOrGeneric».

- [ ] **Step 3: Реализовать реестр**

Создать `internal/service/backtest/scalping_rsimacd_registry.go`:

```go
package backtest

import (
	"encoding/json"
	"fmt"

	"tinvest/internal/service/trading_strategy/scalping/strategy"
	"tinvest/internal/service/trading_strategy/scalping_rsimacd/strategy/core"
)

// scalpingRSIMACDBindingFor builds a Binding for a ticker on the scalping_rsimacd engine.
// The strategy is ticker-agnostic; only the ticker label differs, so a single generic
// default suffices until calibration proves per-ticker params are needed.
func scalpingRSIMACDBindingFor(ticker string) Binding {
	return Binding{
		DefaultParams: func() any { return core.DefaultParams() },
		Build: func(params any) strategy.Strategy {
			return core.NewWithParams(ticker, params.(core.Params))
		},
		ParseParams: func(raw []byte) (any, error) {
			p := core.DefaultParams() // start from defaults so partial JSON overrides
			if err := json.Unmarshal(raw, &p); err != nil {
				return nil, fmt.Errorf("backtest: parse scalping_rsimacd params: %w", err)
			}
			return p, nil
		},
	}
}

// ScalpingRSIMACDLookupOrGeneric returns a scalping_rsimacd binding bound to the ticker.
// There are no per-ticker packages yet (calibration pending), so every ticker gets the
// generic defaults.
func ScalpingRSIMACDLookupOrGeneric(ticker string) Binding {
	return scalpingRSIMACDBindingFor(ticker)
}
```

- [ ] **Step 4: Убедиться, что тесты реестра проходят**

Run: `go test ./internal/service/backtest/ -run ScalpingRSIMACD -v 2>&1 | tail -15`
Expected: PASS (три теста).

- [ ] **Step 5: Подключить CLI**

В `cmd/backtest/main.go` заменить описание флага (строка 41):

```go
		strategyName = flag.String("strategy", "scalping", "strategy engine: scalping|reversion|scalping_rsimacd")
```

и расширить `switch` (строки 153–159):

```go
	switch strategyName {
	case "reversion":
		binding = svc.ReversionLookupOrGeneric(ticker)
	case "scalping":
		binding = svc.LookupOrGeneric(ticker)
	case "scalping_rsimacd":
		binding = svc.ScalpingRSIMACDLookupOrGeneric(ticker)
	default:
		return fmt.Errorf("unknown strategy %q (want scalping|reversion|scalping_rsimacd)", strategyName)
	}
```

- [ ] **Step 6: Проверить сборку и полный набор тестов**

Run: `go build ./internal/... ./pkg/... ./cmd/... && go test ./internal/... ./pkg/... 2>&1 | grep -v "^ok\|no test files" | head -20`
Expected: сборка без ошибок; в выводе тестов нет строк FAIL.

- [ ] **Step 7: Создать грид-файл**

Создать `data/params/scalping_rsimacd/grid.json`:

```json
{
  "_comment": "scalping_rsimacd phased grid. Phase 1 sweeps the entry geometry (RSI length/oversold zone + the MACD confirmation window); phase 2 sweeps risk and exits (RR, stop buffer, stochastic-exit ablation). MACD(3,6,9), the stochastic (14,3,3) and the session bounds are fixed by the strategy definition and are deliberately NOT swept — the grid stays at 27+12 combos to keep the fitting surface small. Judge on pooled OOS profit factor from a walk-forward, never on the in-sample best.",
  "phases": [
    {
      "name": "entry",
      "keepTop": 6,
      "grid": {
        "RSIPeriod": [3, 4, 5],
        "RSIOversold": [20, 25, 30],
        "MACDConfirmBars": [2, 3, 4]
      }
    },
    {
      "name": "risk",
      "grid": {
        "RR": [2.0, 2.5, 3.0],
        "StopBufferATR": [0, 0.1],
        "EnableStochExit": [0, 1]
      }
    }
  ]
}
```

- [ ] **Step 8: Прогнать бэктест на живых данных**

Run:

```bash
go run ./cmd/backtest -ticker SBER -strategy scalping_rsimacd -interval Minutes5 \
  -out ./reports/SBER -months 6 2>&1 | tail -20
```

Expected: команда завершается без ошибок и печатает путь к отчёту. Если сделок ноль — это **не** повод править пороги: зафиксировать факт и продиагностировать одним баром через `-explain "YYYY-MM-DD HH:MM"`, результат вынести в отчёт по задаче.

- [ ] **Step 9: Написать документацию**

Создать `docs/scalping_rsimacd/strategy.md`:

```markdown
# Scalping RSI+MACD (5m, лонг)

Дизайн: `docs/superpowers/specs/2026-07-22-scalping-rsimacd-design.md`
Код: `internal/service/trading_strategy/scalping_rsimacd/strategy/core`

## Правила

**Вход** (только лонг, только внутри сессии Пн–Чт 08:00–17:00 MSK, Пт 08:00–14:00):

1. На текущем баре MACD(3,6,9) пересекает сигнальную линию снизу вверх, **обе линии ниже нуля**.
2. В окне из `MACDConfirmBars` баров до текущего (включая сам бар кросса) RSI(`RSIPeriod`)
   пересёк уровень `RSIOversold` снизу вверх и закрылся выше него.
3. Уровень стопа держался с момента RSI-кросса, а закрытие текущего бара выше него.
4. Риск входа лежит в границах `[MinRiskATR×ATR, MaxRiskATR×ATR]`.

**Стоп:** минимум свечи RSI-кросса минус `StopBufferATR×ATR`.
**Тейк:** `вход + RR × риск`.

**Выходы** в порядке приоритета: `SL` → `TP` → `STOCH` (%K классического стохастика (14,3,3)
выходит вниз из зоны 80, закрытие по цене закрытия бара) → `EOD` (принудительное закрытие
на последнем баре сессии; позиция не переносится через ночь и выходные).

## Запуск

```
go run ./cmd/backtest -ticker <TICKER> -strategy scalping_rsimacd -interval Minutes5 \
  -calibrate data/params/scalping_rsimacd/grid.json -out ./reports/<TICKER> \
  -months 6 -test-months 3 -min-trades 20 -metric profit_factor
```

Диагностика одного бара: добавить `-explain "2026-07-20 12:35"` (время MSK).

## Критерий приёмки

Решение принимается только по pooled OOS profit factor на walk-forward: PF ≥ 1.5 при
≥ 30 OOS-сделок — кандидат на live; 1.0–1.5 — edge не подтверждён; сходимость всех комбо
или < 10 сделок — сетап слишком редок, диагностировать через `-explain`.
```

- [ ] **Step 10: Обновить CLAUDE.md**

В разделе `## Layout`, в пункте про `internal/service`, дописать к строке про `trading_strategy/` упоминание нового движка. Заменить строку:

```
  - `trading_strategy/` — `golden_x`, `reversion`, `bonds` (live strategies); `scalping/model/` and `scalping/strategy/` (shared core used by reversion (live) and the backtest engine; scalping live layer removed). Note: backtest-only strategies `levels`, `momentum`, `smc` were removed after walk-forward validation failed (2026-07-20).
```

на:

```
  - `trading_strategy/` — `golden_x`, `reversion`, `bonds` (live strategies); `scalping/model/` and `scalping/strategy/` (shared core used by reversion (live) and the backtest engine; scalping live layer removed); `scalping_rsimacd/` — backtest-only 5m long RSI+MACD scalper (`-strategy scalping_rsimacd`, docs: `docs/scalping_rsimacd/strategy.md`). Note: backtest-only strategies `levels`, `momentum`, `smc` were removed after walk-forward validation failed (2026-07-20).
```

- [ ] **Step 11: Прогнать полный CI-гейт**

Run: `./bin/mage ci 2>&1 | tail -20`
Expected: линт чистый, `go test -race ./...` зелёный, mock-drift пусто. Если `./bin/mage` отсутствует — порядок восстановления описан в `docs/tooling/mage.md`.

- [ ] **Step 12: Коммит**

```bash
git add internal/service/backtest/scalping_rsimacd_registry.go \
        internal/service/backtest/scalping_rsimacd_registry_test.go \
        cmd/backtest/main.go data/params/scalping_rsimacd/grid.json \
        docs/scalping_rsimacd/strategy.md CLAUDE.md
git commit -m "feat(scalping-rsimacd): backtest binding, CLI flag, calibration grid and docs"
```

---

## После плана: калибровка (не часть реализации)

Реализация закончена, когда прошёл Task 5. Дальше — отдельный шаг с решением человека:

```bash
go run ./cmd/backtest -ticker <TICKER> -strategy scalping_rsimacd -interval Minutes5 \
  -calibrate data/params/scalping_rsimacd/grid.json -out ./reports/<TICKER> \
  -months 6 -test-months 3 -min-trades 20 -metric profit_factor
```

Судить строго по pooled OOS PF по корзине тикеров, а не по лучшему in-sample результату.
Пороги приёмки — в спеке, раздел «Критерий приёмки». Уроки daylow и scalping-adaptive
применимы напрямую: сходимость всех комбо к одному результату означает, что сетап слишком
редок, а не что параметры не важны.
