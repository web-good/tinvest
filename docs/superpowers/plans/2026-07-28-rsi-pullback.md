# Откат по RSI в тренде (`rsi_pullback`) — план реализации

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** backtest-only лонговая стратегия `rsi_pullback` — покупка отката внутри восходящего тренда на 30-минутных барах, с сеткой калибровки и документацией, готовая к прогонам владельца.

**Architecture:** правила живут в stateless-ядре `internal/service/trading_strategy/rsi_pullback/strategy/core` (структурный клон `rsi_ema`); реестр `RSIPullbackLookupOrGeneric` связывает ядро с движком; `cmd/backtest` получает ветку `-strategy rsi_pullback`. Индикаторы (`indicators.RSISeries`, `indicators.ATR`, `ema.Compute`), движок бэктеста, метрики, walk-forward и отчёты НЕ меняются. Соседняя стратегия `rsi_ema` не трогается.

**Tech Stack:** Go 1.25, стандартная библиотека; `pkg/indicators`, `internal/domain/ema`; тесты — table-driven `testing`; сборка и проверки — `./bin/mage ci`.

**Спека:** `docs/superpowers/specs/2026-07-28-rsi-pullback-design.md`

## Global Constraints

- Ядро **stateless между барами**: всё пересчитывается из `strategy.MarketData`; состояние позиции приходит только в `md.Position`.
- Все поля `Params` — только `int` и `float64`: рефлексивная грид-калибровка (`applyField`) других типов не перебирает.
- Никакого lookahead: решение на баре `i` использует только бары `0..i`.
- Комментарии и сообщения в коде — на английском (как в `rsi_ema`); тексты `EntryReason`/`ExitReason`/`Explain` — на русском (их читает владелец в отчётах).
- Часовой пояс всех сессионных правил — `Europe/Moscow` с откатом на UTC.
- Комиссия в бэктесте — `0.0005` за сторону (0.1% за круг). Дефолт CLI менять нельзя.
- Стратегия backtest-only: в `internal/app`, live-раннеры и Telegram она НЕ подключается; старший таймфрейм (HTF) для неё не заводится.
- Коды выхода ровно: `"SL"`, `"RSI"`, `"TIME"`, `"EOD"`; приоритет SL → RSI → TIME → EOD.
- Каждая задача завершается зелёным `./bin/mage ci` и коммитом.

---

## File Structure

| Файл | Ответственность |
|---|---|
| `internal/service/trading_strategy/rsi_pullback/strategy/core/core.go` (создать) | `Params`, `DefaultParams`, `Strategy`, `Decide`/`enter`/`manage`/`Explain`, `Lookback`, сессионные хелперы. Ничего не знает о загрузке свечей. |
| `internal/service/trading_strategy/rsi_pullback/strategy/core/core_test.go` (создать) | Table-driven тесты правил входа, выходов, приоритета, сессионных границ и отсутствия lookahead. |
| `internal/service/backtest/rsi_pullback_registry.go` (создать) | `RSIPullbackLookupOrGeneric` — биндинг ядра к движку. |
| `internal/service/backtest/rsi_pullback_registry_test.go` (создать) | Тесты биндинга и разбора частичного JSON. |
| `internal/service/backtest/rsi_pullback_grid_test.go` (создать) | Тест: каждое имя поля в `grid.json` существует в `Params`; контрольная точка «выключено» на месте. |
| `cmd/backtest/main.go` (изменить, строки 41, 155-167) | Ветка `case "rsi_pullback"` и строки помощи. |
| `data/params/rsi_pullback/grid.json` (создать) | Фазовая сетка, 32 комбинации. |
| `docs/rsi_pullback/strategy.md` (создать) | Правила, параметры, команды запуска. |

---

### Task 1: Ядро — параметры, сессия и вход

**Files:**
- Create: `internal/service/trading_strategy/rsi_pullback/strategy/core/core.go`
- Test: `internal/service/trading_strategy/rsi_pullback/strategy/core/core_test.go`

**Interfaces:**
- Consumes: `indicators.RSISeries(closes []float64, period int) []float64`; `indicators.ATR(highs, lows, closes []float64, period int) float64`; `ema.Compute(closes []float64, period int) []float64`; `strategy.MarketData`, `strategy.Position`, `model.Signal`, `model.SignalBuy`.
- Produces: `core.Params` (поля перечислены ниже), `core.DefaultParams() Params`, `core.NewWithParams(ticker string, p Params) *Strategy`, `(*Strategy).Ticker() string`, `(*Strategy).Lookback() int`, `(*Strategy).Decide(md strategy.MarketData) model.Signal`; приватные `enter`, `manage` (в этой задаче — заглушка), `inSession`, `isDayEnd`, `barSpanMinutes`, `barTime`, `emaPair`, `crossedDown`, `crossedUp`.

- [ ] **Step 1: Написать падающие тесты входа**

Создать `internal/service/trading_strategy/rsi_pullback/strategy/core/core_test.go`:

```go
package core

import (
	"math"
	"testing"
	"time"

	"tinvest/internal/service/trading_strategy/scalping/model"
	"tinvest/internal/service/trading_strategy/scalping/strategy"
)

// msk is the timezone every session rule is anchored to.
var msk = func() *time.Location {
	loc, err := time.LoadLocation("Europe/Moscow")
	if err != nil {
		return time.UTC
	}
	return loc
}()

// barSeries builds MarketData from closes, stamping bars every 30 minutes starting at start.
// Highs/Lows are derived from the close with a fixed 0.3% envelope so ATR is always positive.
func barSeries(closes []float64, start time.Time) strategy.MarketData {
	n := len(closes)
	md := strategy.MarketData{
		Highs:   make([]float64, n),
		Lows:    make([]float64, n),
		Closes:  append([]float64(nil), closes...),
		Volumes: make([]int64, n),
		Times:   make([]time.Time, n),
	}
	for i, c := range closes {
		md.Highs[i] = c * 1.003
		md.Lows[i] = c * 0.997
		md.Volumes[i] = 1000
		md.Times[i] = start.Add(time.Duration(i) * 30 * time.Minute)
	}
	md.Price = closes[n-1]
	return md
}

// pullbackCloses builds a long uptrend (so the fast EMA sits above the slow one) followed by a
// sharp multi-bar drop that pushes a short RSI below the lower band on the LAST bar.
func pullbackCloses() []float64 {
	const rise = 400
	out := make([]float64, 0, rise+6)
	p := 100.0
	for i := 0; i < rise; i++ {
		p *= 1.0008
		out = append(out, p)
	}
	// Five consecutive down bars: RSI(4) collapses toward zero on the last one.
	for i := 0; i < 5; i++ {
		p *= 0.994
		out = append(out, p)
	}
	return out
}

// entryFixture returns market data whose LAST bar is a valid entry bar at 12:00 MSK Monday.
func entryFixture() strategy.MarketData {
	closes := pullbackCloses()
	// 2026-06-01 is a Monday. Place the last bar at 12:00 MSK.
	last := time.Date(2026, 6, 1, 12, 0, 0, 0, msk)
	start := last.Add(-time.Duration(len(closes)-1) * 30 * time.Minute)
	return barSeries(closes, start)
}

func TestDefaultParams(t *testing.T) {
	p := DefaultParams()
	want := Params{
		RSIPeriod: 4, RSILower: 15, RSIUpper: 70,
		EMAFast: 10, EMASlow: 100,
		StopATR: 1.2, ATRPeriod: 14, MaxHoldBars: 8,
		SessionStartMin: 420, SessionEndMin: 1020, DayEndMin: 1380,
	}
	if p != want {
		t.Fatalf("DefaultParams() = %+v, want %+v", p, want)
	}
}

func TestLookbackCoversSlowEMA(t *testing.T) {
	s := NewWithParams("T", DefaultParams())
	if got := s.Lookback(); got < 220 {
		t.Fatalf("Lookback() = %d, want >= 220 (2*EMASlow+20)", got)
	}
	p := DefaultParams()
	p.EMASlow = 200
	if got := NewWithParams("T", p).Lookback(); got != 420 {
		t.Fatalf("Lookback() with EMASlow=200 = %d, want 420", got)
	}
}

func TestEnterBuysThePullback(t *testing.T) {
	s := NewWithParams("T", DefaultParams())
	md := entryFixture()
	got := s.Decide(md)
	if got.Kind != model.SignalBuy {
		t.Fatalf("Kind = %v, want Buy (EntryReason %q)", got.Kind, got.EntryReason)
	}
	i := len(md.Closes) - 1
	wantStop := md.Closes[i] - DefaultParams().StopATR*got.ATR
	if got.ATR <= 0 {
		t.Fatalf("ATR = %v, want > 0", got.ATR)
	}
	if math.Abs(got.StopLoss-wantStop) > 1e-9 {
		t.Fatalf("StopLoss = %v, want %v", got.StopLoss, wantStop)
	}
	if got.EntryReason == "" {
		t.Fatal("EntryReason is empty")
	}
}

// TestEnterGates breaks exactly one precondition at a time and requires no entry.
func TestEnterGates(t *testing.T) {
	base := DefaultParams()
	tests := []struct {
		name  string
		tweak func(p *Params, md *strategy.MarketData)
	}{
		{"before the session opens", func(_ *Params, md *strategy.MarketData) {
			shiftTo(md, time.Date(2026, 6, 1, 6, 30, 0, 0, msk))
		}},
		{"after the entry window closes", func(_ *Params, md *strategy.MarketData) {
			shiftTo(md, time.Date(2026, 6, 1, 17, 0, 0, 0, msk))
		}},
		{"weekend", func(_ *Params, md *strategy.MarketData) {
			// 2026-06-06 is a Saturday.
			shiftTo(md, time.Date(2026, 6, 6, 12, 0, 0, 0, msk))
		}},
		{"downtrend: fast EMA below slow", func(_ *Params, md *strategy.MarketData) {
			for i := range md.Closes {
				md.Closes[i] = 200 - md.Closes[i]
				md.Highs[i] = md.Closes[i] * 1.003
				md.Lows[i] = md.Closes[i] * 0.997
			}
			md.Price = md.Closes[len(md.Closes)-1]
		}},
		{"RSI stays above the lower band", func(p *Params, _ *strategy.MarketData) {
			p.RSILower = 0.5
		}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			p := base
			md := entryFixture()
			tc.tweak(&p, &md)
			if got := NewWithParams("T", p).Decide(md); got.Kind != model.SignalNone {
				t.Fatalf("Kind = %v, want None (reason %q)", got.Kind, got.EntryReason)
			}
		})
	}
}

// shiftTo re-stamps the series so its LAST bar opens at last.
func shiftTo(md *strategy.MarketData, last time.Time) {
	start := last.Add(-time.Duration(len(md.Times)-1) * 30 * time.Minute)
	for i := range md.Times {
		md.Times[i] = start.Add(time.Duration(i) * 30 * time.Minute)
	}
}

// TestEnterCrossIsAnEventNotAState: once RSI already sits below the band on the PREVIOUS bar,
// there is no fresh cross and therefore no entry.
func TestEnterCrossIsAnEventNotAState(t *testing.T) {
	s := NewWithParams("T", DefaultParams())
	md := entryFixture()
	// Append one more down bar: now bar i-1 is already below the band.
	last := md.Closes[len(md.Closes)-1] * 0.994
	md = barSeries(append(md.Closes, last), md.Times[0])
	shiftTo(&md, time.Date(2026, 6, 1, 12, 30, 0, 0, msk))
	if got := s.Decide(md); got.Kind != model.SignalNone {
		t.Fatalf("Kind = %v, want None: RSI was already below the band on the previous bar", got.Kind)
	}
}

func TestEnterRejectsShortHistory(t *testing.T) {
	s := NewWithParams("T", DefaultParams())
	md := barSeries([]float64{100, 99}, time.Date(2026, 6, 1, 11, 0, 0, 0, msk))
	if got := s.Decide(md); got.Kind != model.SignalNone {
		t.Fatalf("Kind = %v, want None on a two-bar series", got.Kind)
	}
}
```

- [ ] **Step 2: Прогнать тесты и убедиться, что они падают**

Run: `go test ./internal/service/trading_strategy/rsi_pullback/... -v`
Expected: FAIL — пакет не компилируется, `undefined: Params`, `undefined: NewWithParams`.

- [ ] **Step 3: Реализовать параметры, сессионные хелперы и вход**

Создать `internal/service/trading_strategy/rsi_pullback/strategy/core/core.go`:

```go
// Package core implements a long-only intraday RSI pullback strategy. When flat it buys the dip
// inside an uptrend: the fast EMA must sit above the slow one and a short RSI must cross DOWN
// through its lower band on the current bar. The position is closed on the first of: the ATR
// stop, RSI crossing UP through the upper band, a time stop measured in bars, or the day-end
// force close — positions never survive into the next day. The decision logic is pure, stateless
// between bars and ticker-agnostic. The reference timeframe is 30 minutes; the EOD gate infers
// the bar span from the series, so other -interval values work as well. Run with
// `-strategy rsi_pullback -interval Minutes30`.
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
// open-times (a dead fallback in practice: the backtest and -explain paths always populate
// Times, so barSpanMinutes infers the real span). It matches this strategy's 30-minute
// reference timeframe so the EOD gate degrades sanely rather than under-detecting the day-end
// bar.
const defaultBarSpanMin = 30

// minLookback floors the candle window at roughly one trading week of 30-minute bars, so the
// session and RSI gates always see enough history even with short indicator periods.
const minLookback = 120

// Params holds every tunable. All fields are int or float64 so reflection grid calibration
// can sweep them.
type Params struct {
	RSIPeriod       int     // RSI length (grid; default 4)
	RSILower        float64 // lower band; a DOWNWARD cross of it is the entry (grid; default 15)
	RSIUpper        float64 // upper band; an UPWARD cross of it is the exit (grid; default 70)
	EMAFast         int     // fast EMA period (grid; default 10)
	EMASlow         int     // slow EMA period (grid; default 100)
	StopATR         float64 // stop = entry - StopATR*ATR; 0 disables the stop (grid; never 0 in the grid)
	ATRPeriod       int     // ATR length; used only when StopATR>0
	MaxHoldBars     int     // time stop in bars; 0 disables it (grid)
	SessionStartMin int     // entry window start, minutes from MSK midnight (420 = 07:00)
	SessionEndMin   int     // entry window end, minutes from MSK midnight (1020 = 17:00)
	DayEndMin       int     // day-end force-close boundary, minutes from MSK midnight (1380 = 23:00)
}

// DefaultParams returns the spec's baseline; swept values come from calibration.
func DefaultParams() Params {
	return Params{
		RSIPeriod:       4,
		RSILower:        15,
		RSIUpper:        70,
		EMAFast:         10,
		EMASlow:         100,
		StopATR:         1.2,
		ATRPeriod:       14,
		MaxHoldBars:     8,
		SessionStartMin: 420,
		SessionEndMin:   1020,
		DayEndMin:       1380,
	}
}

// Strategy trades a single instrument with the RSI pullback rules. Ticker-agnostic and pure.
type Strategy struct {
	ticker string
	p      Params
}

// NewWithParams returns the strategy for a ticker with explicit params.
func NewWithParams(ticker string, p Params) *Strategy { return &Strategy{ticker: ticker, p: p} }

func (s *Strategy) Ticker() string { return s.ticker }

// Lookback sizes the candle window the engine feeds Decide on every bar. It must cover the
// hungriest indicator with room to converge: ema.Compute seeds on an SMA over the first
// `period` closes, so a window of exactly `period` bars yields a bare seed — and a window
// SHORTER than the period yields an all-zero series, which silently fails the trend gate for
// the whole run instead of erroring. Doubling the largest period leaves as many recursion steps
// as the seed span; the +20 covers the two-bar cross lookups and the ATR's extra bar.
func (s *Strategy) Lookback() int {
	need := max(s.p.EMASlow, s.p.EMAFast, s.p.RSIPeriod, s.p.ATRPeriod)
	return max(minLookback, 2*need+20)
}

// mskLoc anchors the session windows to the Moscow calendar (UTC fallback).
var mskLoc = func() *time.Location {
	loc, err := time.LoadLocation("Europe/Moscow")
	if err != nil {
		return time.UTC
	}
	return loc
}()

// isWeekend reports whether tl (already in MSK) falls on a non-trading day.
func isWeekend(tl time.Time) bool {
	wd := tl.Weekday()
	return wd == time.Saturday || wd == time.Sunday
}

// inSession reports whether bar-time t falls inside the entry window in MSK. A zero time skips
// the gate — never block on missing data.
func (s *Strategy) inSession(t time.Time) bool {
	if t.IsZero() {
		return true
	}
	tl := t.In(mskLoc)
	if isWeekend(tl) {
		return false
	}
	m := tl.Hour()*60 + tl.Minute()
	return m >= s.p.SessionStartMin && m < s.p.SessionEndMin
}

// isDayEnd reports whether the bar opening at t, spanning spanMin minutes, is the last one
// before the day-end force-close boundary (DayEndMin). It is decoupled from the entry cutoff so
// a position opened inside the entry window is still managed through the evening session up to
// DayEndMin. A zero time degrades the EOD exit to a no-op.
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

// barSpanMinutes infers the bar length from the series' own open-times: the MEDIAN gap between
// consecutive bars (robust to session and weekend jumps). Falls back to defaultBarSpanMin when
// Times is absent or too short.
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

// emaPair computes the fast and slow EMA series (index-aligned to closes) and reports ok=true
// only when both are warmed at the last bar. ema.Compute zero-fills warm-up positions, so an
// unwarmed value would compare as a spurious 0.
func (s *Strategy) emaPair(closes []float64) (fast, slow []float64, ok bool) {
	fast = ema.Compute(closes, s.p.EMAFast)
	slow = ema.Compute(closes, s.p.EMASlow)
	i := len(closes) - 1
	if i < 0 || len(fast) != len(closes) || len(slow) != len(closes) {
		return nil, nil, false
	}
	return fast, slow, fast[i] > 0 && slow[i] > 0
}

// crossedDown reports whether series crossed down through level between i-1 and i: it sat at or
// above the level and is now strictly below. The series[i-1] > 0 guard rejects RSISeries warm-up
// zeros reading as "below the level".
func crossedDown(series []float64, i int, level float64) bool {
	return i >= 1 && series[i-1] > 0 && series[i-1] >= level && series[i] < level
}

// crossedUp reports whether series crossed up through level between i-1 and i: it sat at or
// below the level and is now strictly above. Mirrors crossedDown, so a bar sitting exactly ON
// the level is treated as "not yet crossed" in both directions.
func crossedUp(series []float64, i int, level float64) bool {
	return i >= 1 && series[i-1] > 0 && series[i-1] <= level && series[i] > level
}

// Decide routes to entry (flat) or position management (open).
func (s *Strategy) Decide(md strategy.MarketData) model.Signal {
	sig := model.Signal{Ticker: s.ticker, Price: md.Price}
	if md.Position != nil {
		return s.manage(md, sig)
	}
	return s.enter(md, sig)
}

// enter emits a long when a short RSI crosses DOWN through its lower band on the current bar
// while the fast EMA sits above the slow one. Everything is recomputed from md — no state
// survives between bars.
func (s *Strategy) enter(md strategy.MarketData, sig model.Signal) model.Signal {
	n := len(md.Closes)
	if n < 2 || len(md.Highs) != n || len(md.Lows) != n {
		return sig
	}
	// 1. session window, and never on the day-end bar (manage() only runs from the NEXT bar, so
	// an entry on the day-end bar could not be EOD-closed on its own bar).
	t := s.barTime(md)
	if !s.inSession(t) || s.isDayEnd(t, barSpanMinutes(md.Times)) {
		return sig
	}
	i := n - 1
	// 2. RSI crosses down through the lower band on the current bar.
	rsi := indicators.RSISeries(md.Closes, s.p.RSIPeriod)
	if len(rsi) != n || !crossedDown(rsi, i, s.p.RSILower) {
		return sig
	}
	// 3. trend confirmation: fast EMA above slow EMA (both warmed).
	fast, slow, ok := s.emaPair(md.Closes)
	if !ok || fast[i] <= slow[i] {
		return sig
	}
	// 4. optional ATR stop; a non-positive ATR means the data cannot support the stop, and an
	// entry without its planned protection is refused.
	entry := md.Closes[i]
	var stop, atr float64
	if s.p.StopATR > 0 {
		atr = indicators.ATR(md.Highs, md.Lows, md.Closes, s.p.ATRPeriod)
		if atr <= 0 {
			return sig
		}
		stop = entry - s.p.StopATR*atr
	}
	sig.Kind = model.SignalBuy
	sig.StopLoss = stop
	sig.ATR = atr
	sig.RSI = rsi[i]
	sig.EntryReason = s.entryReason(rsi[i], fast[i], slow[i], entry, stop, atr)
	return sig
}

// entryReason renders the human-readable rationale shown in the trade journal.
func (s *Strategy) entryReason(rsiNow, fastNow, slowNow, entry, stop, atr float64) string {
	stopHow := "стоп выключен (StopATR=0)"
	if s.p.StopATR > 0 {
		stopHow = fmt.Sprintf("стоп %.4f (вход − %.2f×ATR, ATR=%.4f)", stop, s.p.StopATR, atr)
	}
	return fmt.Sprintf(
		"RSI(%d) ушёл под %.0f (%.1f) на откате, EMA(%d) %.4f > EMA(%d) %.4f; вход %.4f, %s",
		s.p.RSIPeriod, s.p.RSILower, rsiNow, s.p.EMAFast, fastNow, s.p.EMASlow, slowNow, entry, stopHow,
	)
}

// manage handles an open long. Task 2 replaces this stub with the SL -> RSI -> TIME -> EOD
// precedence.
func (s *Strategy) manage(md strategy.MarketData, sig model.Signal) model.Signal {
	return sig
}
```

Примечание для исполнителя: `manage` в этой задаче — намеренная заглушка (выходы делает Task 2), поэтому пакеты `math` и `strings` пока не импортируются — Task 2 добавит их вместе с `barsHeld` и `Explain`. Параметр `md` заглушка не использует; если линтер потребует, назови его `_`, а Task 2 вернёт имя.

- [ ] **Step 4: Прогнать тесты**

Run: `go test ./internal/service/trading_strategy/rsi_pullback/... -v`
Expected: PASS — все тесты Step 1 зелёные.

- [ ] **Step 5: Полный гейт**

Run: `./bin/mage ci`
Expected: PASS

- [ ] **Step 6: Коммит**

```bash
git add internal/service/trading_strategy/rsi_pullback/
git commit -m "feat(rsi_pullback): ядро и правила входа по откату RSI в тренде"
```

---

### Task 2: Выходы, приоритет и `Explain`

**Files:**
- Modify: `internal/service/trading_strategy/rsi_pullback/strategy/core/core.go` (заменить заглушку `manage`, добавить `barsHeld`, `Explain`, удалить временные `var _ =`)
- Test: `internal/service/trading_strategy/rsi_pullback/strategy/core/core_test.go` (дописать)

**Interfaces:**
- Consumes: всё из Task 1; `strategy.Position{PurchasePrice, StopLoss, TakeProfit, EntryATR, EntryTime}`.
- Produces: `(*Strategy).Explain(md strategy.MarketData) string` — необязательный интерфейс, который движок подхватывает в `Trace` (`-explain`). Коды выхода: `"SL"`, `"RSI"`, `"TIME"`, `"EOD"`; приватный `barsHeld(md) int` (возвращает `holdUnknown` = -1, когда время входа неизвестно).

- [ ] **Step 1: Написать падающие тесты выходов**

Дописать в `core_test.go`:

```go
// withPosition attaches an open long entered heldBars bars before the last bar of md.
func withPosition(md strategy.MarketData, entryPrice, stop float64, heldBars int) strategy.MarketData {
	last := md.Times[len(md.Times)-1]
	md.Position = &strategy.Position{
		PurchasePrice: entryPrice,
		StopLoss:      stop,
		EntryATR:      entryPrice * 0.003,
		EntryTime:     last.Add(-time.Duration(heldBars) * 30 * time.Minute),
	}
	return md
}

func TestExitStopLoss(t *testing.T) {
	s := NewWithParams("T", DefaultParams())
	md := entryFixture()
	i := len(md.Closes) - 1
	// Stop sits just above the bar's low: the stop must fire.
	stop := md.Lows[i] * 1.0001
	md = withPosition(md, md.Closes[i]*1.02, stop, 1)
	got := s.Decide(md)
	if got.Kind != model.SignalSell || got.Reason != "SL" {
		t.Fatalf("Kind/Reason = %v/%q, want Sell/SL", got.Kind, got.Reason)
	}
	if math.Abs(got.StopLoss-stop) > 1e-9 {
		t.Fatalf("StopLoss = %v, want the frozen position stop %v", got.StopLoss, stop)
	}
}

// upperCrossFixture builds an uptrend whose LAST bar pushes RSI(4) above the upper band.
func upperCrossFixture() strategy.MarketData {
	closes := make([]float64, 0, 410)
	p := 100.0
	for i := 0; i < 400; i++ {
		p *= 1.0008
		closes = append(closes, p)
	}
	// Three down bars pull RSI below the upper band, then two sharp up bars push it back above.
	for i := 0; i < 3; i++ {
		p *= 0.994
		closes = append(closes, p)
	}
	for i := 0; i < 2; i++ {
		p *= 1.010
		closes = append(closes, p)
	}
	last := time.Date(2026, 6, 1, 14, 0, 0, 0, msk)
	start := last.Add(-time.Duration(len(closes)-1) * 30 * time.Minute)
	return barSeries(closes, start)
}

func TestExitOnRSIEnteringUpperBand(t *testing.T) {
	s := NewWithParams("T", DefaultParams())
	md := upperCrossFixture()
	i := len(md.Closes) - 1
	// Stop far below so only the RSI exit can fire.
	md = withPosition(md, md.Closes[i]*0.97, md.Lows[i]*0.5, 2)
	got := s.Decide(md)
	if got.Kind != model.SignalSell || got.Reason != "RSI" {
		t.Fatalf("Kind/Reason = %v/%q, want Sell/RSI", got.Kind, got.Reason)
	}
}

func TestExitStopWinsOverRSIOnTheSameBar(t *testing.T) {
	s := NewWithParams("T", DefaultParams())
	md := upperCrossFixture()
	i := len(md.Closes) - 1
	// Stop inside the bar AND the RSI cross on the same bar: SL must win.
	md = withPosition(md, md.Closes[i]*0.97, md.Lows[i]*1.0001, 2)
	got := s.Decide(md)
	if got.Reason != "SL" {
		t.Fatalf("Reason = %q, want SL to take precedence over RSI", got.Reason)
	}
}

func TestExitTimeStop(t *testing.T) {
	p := DefaultParams()
	p.MaxHoldBars = 3
	s := NewWithParams("T", p)
	md := entryFixture()
	i := len(md.Closes) - 1
	md = withPosition(md, md.Closes[i]*0.99, md.Lows[i]*0.5, 3)
	got := s.Decide(md)
	if got.Kind != model.SignalSell || got.Reason != "TIME" {
		t.Fatalf("Kind/Reason = %v/%q, want Sell/TIME after 3 bars", got.Kind, got.Reason)
	}
}

func TestExitTimeStopDisabledAtZero(t *testing.T) {
	p := DefaultParams()
	p.MaxHoldBars = 0
	s := NewWithParams("T", p)
	md := entryFixture()
	i := len(md.Closes) - 1
	md = withPosition(md, md.Closes[i]*0.99, md.Lows[i]*0.5, 50)
	if got := s.Decide(md); got.Kind != model.SignalNone {
		t.Fatalf("Kind = %v (%q), want None: the time stop is disabled at MaxHoldBars=0", got.Kind, got.Reason)
	}
}

func TestExitTimeStopSilentOnUnknownEntryTime(t *testing.T) {
	p := DefaultParams()
	p.MaxHoldBars = 1
	s := NewWithParams("T", p)
	md := entryFixture()
	i := len(md.Closes) - 1
	md = withPosition(md, md.Closes[i]*0.99, md.Lows[i]*0.5, 5)
	md.Position.EntryTime = time.Time{}
	if got := s.Decide(md); got.Reason == "TIME" {
		t.Fatal("TIME fired with an unknown entry time; it must stay silent")
	}
}

func TestExitEndOfDay(t *testing.T) {
	s := NewWithParams("T", DefaultParams())
	md := entryFixture()
	shiftTo(&md, time.Date(2026, 6, 1, 22, 30, 0, 0, msk))
	i := len(md.Closes) - 1
	md = withPosition(md, md.Closes[i]*0.99, md.Lows[i]*0.5, 2)
	got := s.Decide(md)
	if got.Kind != model.SignalSell || got.Reason != "EOD" {
		t.Fatalf("Kind/Reason = %v/%q, want Sell/EOD on the 22:30 bar", got.Kind, got.Reason)
	}
}

func TestExitHoldsThroughTheEveningSession(t *testing.T) {
	p := DefaultParams()
	p.MaxHoldBars = 0
	s := NewWithParams("T", p)
	md := entryFixture()
	shiftTo(&md, time.Date(2026, 6, 1, 19, 0, 0, 0, msk))
	i := len(md.Closes) - 1
	md = withPosition(md, md.Closes[i]*0.99, md.Lows[i]*0.5, 2)
	if got := s.Decide(md); got.Kind != model.SignalNone {
		t.Fatalf("Kind = %v (%q), want None: 19:00 is past the entry window but before DayEndMin",
			got.Kind, got.Reason)
	}
}

func TestExplainMentionsEveryGate(t *testing.T) {
	s := NewWithParams("T", DefaultParams())
	out := s.Explain(entryFixture())
	for _, want := range []string{"сессия", "RSI", "EMA", "стоп", "удержание"} {
		if !strings.Contains(out, want) {
			t.Fatalf("Explain() is missing %q; got:\n%s", want, out)
		}
	}
}
```

Добавить `"strings"` в импорты теста.

- [ ] **Step 2: Прогнать тесты и убедиться, что они падают**

Run: `go test ./internal/service/trading_strategy/rsi_pullback/... -run 'TestExit|TestExplain' -v`
Expected: FAIL — `manage` возвращает пустой сигнал, `Explain` не определён.

- [ ] **Step 3: Реализовать выходы и `Explain`**

В `core.go` добавить в импорты `"math"` и `"strings"`, заменить заглушку `manage` и добавить:

```go
// holdUnknown is returned by barsHeld when the position's entry time is unknown. The time stop
// treats it as "do not fire": a missing EntryTime must never close a position by itself.
const holdUnknown = -1

// barsHeld counts bars from the position's entry to the current bar, purely from EntryTime, the
// current bar time and the data-inferred span. Positions never survive the EOD close, so the
// window is always within one session and the uniform span is exact.
func (s *Strategy) barsHeld(md strategy.MarketData) int {
	pos := md.Position
	t := s.barTime(md)
	if pos == nil || pos.EntryTime.IsZero() || t.IsZero() {
		return holdUnknown
	}
	span := barSpanMinutes(md.Times)
	if span <= 0 {
		return holdUnknown
	}
	return int(math.Round(t.Sub(pos.EntryTime).Minutes() / float64(span)))
}

// manage handles an open long, exiting in precedence SL -> RSI -> TIME -> EOD. SL is read from
// the position (frozen at entry), never recomputed: the stop the trade was opened with is the
// stop it dies by. RSI fires on an UPWARD cross of RSIUpper — taking the bounce into overbought.
// RSI/TIME/EOD fill at the bar close; SL fills at the stop level (the engine handles that via
// model.IsStopReason).
func (s *Strategy) manage(md strategy.MarketData, sig model.Signal) model.Signal {
	pos := md.Position
	n := len(md.Closes)
	if pos == nil || n < 2 || len(md.Highs) != n || len(md.Lows) != n {
		return sig
	}
	i := n - 1
	low := md.Lows[i]
	closeP := md.Closes[i]

	// 1. hard stop (always active, wins any same-bar tie).
	if pos.StopLoss > 0 && low <= pos.StopLoss {
		sig.Kind, sig.Reason = model.SignalSell, "SL"
		sig.StopLoss = pos.StopLoss
		sig.ExitReason = fmt.Sprintf("SL: low %.4f ≤ стоп %.4f (вход %.4f)", low, pos.StopLoss, pos.PurchasePrice)
		return sig
	}
	// 2. RSI crosses UP through the upper band — the bounce reached overbought.
	rsi := indicators.RSISeries(md.Closes, s.p.RSIPeriod)
	if len(rsi) == n && crossedUp(rsi, i, s.p.RSIUpper) {
		sig.Kind, sig.Reason = model.SignalSell, "RSI"
		sig.RSI = rsi[i]
		sig.ExitReason = fmt.Sprintf("RSI: RSI(%d) пересёк %.0f снизу вверх (%.1f), выход по %.4f (вход %.4f)",
			s.p.RSIPeriod, s.p.RSIUpper, rsi[i], closeP, pos.PurchasePrice)
		return sig
	}
	// 3. time stop: the setup had its chance and did not deliver.
	if held := s.barsHeld(md); s.p.MaxHoldBars > 0 && held != holdUnknown && held >= s.p.MaxHoldBars {
		sig.Kind, sig.Reason = model.SignalSell, "TIME"
		sig.ExitReason = fmt.Sprintf("TIME: удержано %d баров ≥ %d, выход по %.4f (вход %.4f)",
			held, s.p.MaxHoldBars, closeP, pos.PurchasePrice)
		return sig
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

	rsi := indicators.RSISeries(md.Closes, s.p.RSIPeriod)
	if len(rsi) == n {
		fmt.Fprintf(&sb, "RSI(%d) пред %.1f тек %.1f; вход-крест вниз через %.0f? %v; выход-крест вверх через %.0f? %v\n",
			s.p.RSIPeriod, rsi[i-1], rsi[i], s.p.RSILower, crossedDown(rsi, i, s.p.RSILower),
			s.p.RSIUpper, crossedUp(rsi, i, s.p.RSIUpper))
	} else {
		sb.WriteString("RSI: недостаточно истории\n")
	}

	if fast, slow, ok := s.emaPair(md.Closes); ok {
		fmt.Fprintf(&sb, "EMA(%d) %.4f vs EMA(%d) %.4f: тренд вверх? %v\n",
			s.p.EMAFast, fast[i], s.p.EMASlow, slow[i], fast[i] > slow[i])
	} else {
		sb.WriteString("EMA: не прогрето\n")
	}

	if s.p.StopATR > 0 {
		atr := indicators.ATR(md.Highs, md.Lows, md.Closes, s.p.ATRPeriod)
		fmt.Fprintf(&sb, "стоп: вход − %.2f×ATR (ATR=%.4f)\n", s.p.StopATR, atr)
	} else {
		sb.WriteString("стоп: выключен (StopATR=0)\n")
	}

	if md.Position == nil {
		fmt.Fprintf(&sb, "удержание: позиции нет; тайм-стоп %s\n", holdLabel(s.p.MaxHoldBars))
	} else {
		fmt.Fprintf(&sb, "удержание: %d баров; тайм-стоп %s\n", s.barsHeld(md), holdLabel(s.p.MaxHoldBars))
	}
	return sb.String()
}

// holdLabel renders the time stop for Explain: the bar count when armed, "выключен" when off.
func holdLabel(v int) string {
	if v <= 0 {
		return "выключен"
	}
	return fmt.Sprintf("%d баров", v)
}
```

- [ ] **Step 4: Прогнать тесты**

Run: `go test ./internal/service/trading_strategy/rsi_pullback/... -v`
Expected: PASS

- [ ] **Step 5: Полный гейт**

Run: `./bin/mage ci`
Expected: PASS

- [ ] **Step 6: Коммит**

```bash
git add internal/service/trading_strategy/rsi_pullback/
git commit -m "feat(rsi_pullback): выходы SL/RSI/TIME/EOD и Explain"
```

---

### Task 3: Тест на отсутствие lookahead

**Files:**
- Test: `internal/service/trading_strategy/rsi_pullback/strategy/core/core_test.go` (дописать)

**Interfaces:**
- Consumes: всё из Task 1 и Task 2.
- Produces: ничего для последующих задач — это проверочная сеть.

Ядро обязано быть функцией от баров `[0..i]`, а не от того, где обрезано окно. Один нюанс, проверенный на предыдущей стратегии: `indicators.ATR` — рекурсия Уайлдера с неограниченной памятью, поэтому ATR-зависимый стоп при разной длине прогрева отличается на малую величину. Это не lookahead (будущее в расчёт не течёт, а движок вообще подаёт окно фиксированной длины `Lookback()`), поэтому сигнал и RSI-зависимые поля сравниваются строго, а `StopLoss` — с относительным допуском.

- [ ] **Step 1: Написать тест**

Дописать в `core_test.go`:

```go
// sliceMD returns md restricted to bars [from:to).
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

// TestNoLookaheadAcrossWindowCuts is the load-bearing safety net: the decision on bar i must not
// depend on how much history precedes it. Cuts stay far enough from bar i that every indicator
// is warmed in both windows; the ATR-derived stop is compared with a relative tolerance because
// indicators.ATR is an unbounded Wilder recursion whose value depends on the warm-up length —
// but never on future bars. In production the engine always feeds a fixed-length Lookback()
// window, so this discrepancy cannot arise there.
func TestNoLookaheadAcrossWindowCuts(t *testing.T) {
	s := NewWithParams("T", DefaultParams())
	full := entryFixture()
	n := len(full.Closes)

	checked := 0
	for _, from := range []int{0, 40, 80} {
		want := s.Decide(full)
		got := s.Decide(sliceMD(full, from, n))
		if got.Kind != want.Kind || got.Reason != want.Reason {
			t.Fatalf("cut at %d gave %v/%q, full window gave %v/%q",
				from, got.Kind, got.Reason, want.Kind, want.Reason)
		}
		if math.Abs(got.RSI-want.RSI) > 1e-9 {
			t.Fatalf("cut at %d gave RSI %v, full window %v", from, got.RSI, want.RSI)
		}
		if want.StopLoss > 0 {
			if rel := math.Abs(got.StopLoss-want.StopLoss) / want.StopLoss; rel > 1e-3 {
				t.Fatalf("cut at %d gave stop %v, full window %v (relative %g)",
					from, got.StopLoss, want.StopLoss, rel)
			}
		}
		checked++
	}
	if checked == 0 {
		t.Fatal("fixture produced no comparisons")
	}
}

// TestNoLookaheadWithOpenPosition covers the manage() path the same way.
func TestNoLookaheadWithOpenPosition(t *testing.T) {
	s := NewWithParams("T", DefaultParams())
	full := upperCrossFixture()
	n := len(full.Closes)
	i := n - 1
	full = withPosition(full, full.Closes[i]*0.97, full.Lows[i]*0.5, 2)

	want := s.Decide(full)
	if want.Kind == model.SignalNone {
		t.Fatal("fixture produced no exit; the managed sub-case would assert nothing")
	}
	for _, from := range []int{40, 80} {
		cut := sliceMD(full, from, n)
		cut.Position = full.Position
		got := s.Decide(cut)
		if got.Kind != want.Kind || got.Reason != want.Reason {
			t.Fatalf("cut at %d gave %v/%q, full window gave %v/%q",
				from, got.Kind, got.Reason, want.Kind, want.Reason)
		}
	}
}
```

- [ ] **Step 2: Прогнать тест**

Run: `go test ./internal/service/trading_strategy/rsi_pullback/... -run TestNoLookahead -v`
Expected: PASS сразу, если Task 1-2 сделаны верно. **Если падает — чинить ядро, а не тест.** Типичная причина: в решение затесалось значение, зависящее от начала окна (например, индекс, посчитанный от нуля среза, а не от конца).

- [ ] **Step 3: Убедиться, что тест ловит регрессию**

Временно заменить в `enter` условие входа `crossedDown(rsi, i, s.p.RSILower)` на `rsi[i] < s.p.RSILower` (крест превращается в состояние) и прогнать:

Run: `go test ./internal/service/trading_strategy/rsi_pullback/... -run 'TestEnterCrossIsAnEventNotAState' -v`
Expected: FAIL. Вернуть `crossedDown` обратно и убедиться, что снова PASS. Мутацию НЕ коммитить.

- [ ] **Step 4: Полный гейт**

Run: `./bin/mage ci`
Expected: PASS

- [ ] **Step 5: Коммит**

```bash
git add internal/service/trading_strategy/rsi_pullback/strategy/core/core_test.go
git commit -m "test(rsi_pullback): решения не зависят от границы окна (нет lookahead)"
```

---

### Task 4: Реестр и ветка в CLI

**Files:**
- Create: `internal/service/backtest/rsi_pullback_registry.go`
- Create: `internal/service/backtest/rsi_pullback_registry_test.go`
- Modify: `cmd/backtest/main.go` (строка ~41 — текст флага `-strategy`; строки ~155-167 — `switch strategyName`)

**Interfaces:**
- Consumes: `core.DefaultParams()`, `core.NewWithParams` (Task 1); `Binding{DefaultParams func() any; Build func(any) strategy.Strategy; ParseParams func([]byte) (any, error)}`.
- Produces: `backtest.RSIPullbackLookupOrGeneric(ticker string) Binding`; CLI-значение `-strategy rsi_pullback`.

- [ ] **Step 1: Написать падающие тесты биндинга**

Создать `internal/service/backtest/rsi_pullback_registry_test.go`:

```go
package backtest

import (
	"testing"

	"tinvest/internal/service/trading_strategy/rsi_pullback/strategy/core"
)

func TestRSIPullbackBindingBuildsForTicker(t *testing.T) {
	b := RSIPullbackLookupOrGeneric("GAZP")
	p, ok := b.DefaultParams().(core.Params)
	if !ok {
		t.Fatalf("DefaultParams() returned %T, want core.Params", b.DefaultParams())
	}
	if p != core.DefaultParams() {
		t.Fatalf("DefaultParams() = %+v, want %+v", p, core.DefaultParams())
	}
	s := b.Build(p)
	if s.Ticker() != "GAZP" {
		t.Fatalf("Ticker() = %q, want GAZP", s.Ticker())
	}
	if s.Lookback() < 220 {
		t.Fatalf("Lookback() = %d, want >= 220", s.Lookback())
	}
}

func TestRSIPullbackParseParamsLayersOverDefaults(t *testing.T) {
	b := RSIPullbackLookupOrGeneric("GAZP")
	got, err := b.ParseParams([]byte(`{"RSILower": 10}`))
	if err != nil {
		t.Fatalf("ParseParams: %v", err)
	}
	p := got.(core.Params)
	if p.RSILower != 10 {
		t.Fatalf("RSILower = %v, want the JSON value 10", p.RSILower)
	}
	if p.RSIUpper != core.DefaultParams().RSIUpper {
		t.Fatalf("RSIUpper = %v, want the default %v (partial JSON must not zero other fields)",
			p.RSIUpper, core.DefaultParams().RSIUpper)
	}
}

func TestRSIPullbackParseParamsRejectsGarbage(t *testing.T) {
	b := RSIPullbackLookupOrGeneric("GAZP")
	if _, err := b.ParseParams([]byte(`{"RSILower":`)); err == nil {
		t.Fatal("ParseParams accepted malformed JSON, want an error")
	}
}
```

- [ ] **Step 2: Прогнать тесты и убедиться, что они падают**

Run: `go test ./internal/service/backtest/ -run TestRSIPullback -v`
Expected: FAIL — `undefined: RSIPullbackLookupOrGeneric`.

- [ ] **Step 3: Реализовать реестр**

Создать `internal/service/backtest/rsi_pullback_registry.go`:

```go
package backtest

import (
	"encoding/json"
	"fmt"

	"tinvest/internal/service/trading_strategy/rsi_pullback/strategy/core"
	"tinvest/internal/service/trading_strategy/scalping/strategy"
)

// rsiPullbackBindingFor builds a Binding for a ticker on the rsi_pullback engine. The strategy
// is ticker-agnostic; only the ticker label differs, so a single generic default suffices until
// calibration proves per-ticker params are needed.
func rsiPullbackBindingFor(ticker string) Binding {
	return Binding{
		DefaultParams: func() any { return core.DefaultParams() },
		Build: func(params any) strategy.Strategy {
			return core.NewWithParams(ticker, params.(core.Params))
		},
		ParseParams: func(raw []byte) (any, error) {
			p := core.DefaultParams() // start from defaults so partial JSON overrides
			if err := json.Unmarshal(raw, &p); err != nil {
				return nil, fmt.Errorf("backtest: parse rsi_pullback params: %w", err)
			}
			return p, nil
		},
	}
}

// RSIPullbackLookupOrGeneric returns an rsi_pullback binding bound to the ticker. There are no
// per-ticker packages yet (calibration pending), so every ticker gets the generic defaults.
func RSIPullbackLookupOrGeneric(ticker string) Binding {
	return rsiPullbackBindingFor(ticker)
}
```

- [ ] **Step 4: Подключить ветку в CLI**

В `cmd/backtest/main.go`:

1. Строка ~41 — добавить имя в текст флага:

```go
strategyName = flag.String("strategy", "scalping", "strategy engine: scalping|reversion|scalping_rsimacd|rsi_ema|vwap_rev|rsi_pullback")
```

2. В `switch strategyName` (строки ~155-167) добавить ветку рядом с `case "vwap_rev"`, ровно в том же стиле, что соседние ветки:

```go
	case "rsi_pullback":
		binding = svc.RSIPullbackLookupOrGeneric(ticker)
```

3. В строке ошибки неизвестной стратегии добавить имя:

```go
		return fmt.Errorf("unknown strategy %q (want scalping|reversion|scalping_rsimacd|rsi_ema|vwap_rev|rsi_pullback)", strategyName)
```

Ветку загрузки HTF (`switch strategyName` около строки ~197) НЕ трогать: старший таймфрейм этой стратегии не нужен. Дневные свечи (`dailyCandles`) загружаются безусловно выше и ей тоже не требуются.

- [ ] **Step 5: Прогнать тесты**

Run: `go test ./internal/service/backtest/ -run TestRSIPullback -v && go build ./cmd/backtest`
Expected: PASS и успешная сборка.

- [ ] **Step 6: Проверить, что CLI видит стратегию**

Run: `go run ./cmd/backtest -strategy rsi_pullback -ticker GAZP -interval Minutes30 -months 1 -out ./reports/_smoke`
Expected: прогон завершается без ошибки «unknown strategy». Сделок может не быть — на одном месяце это нормально. Каталог `./reports/` в репозиторий не коммитить.

- [ ] **Step 7: Полный гейт**

Run: `./bin/mage ci`
Expected: PASS

- [ ] **Step 8: Коммит**

```bash
git add internal/service/backtest/rsi_pullback_registry.go internal/service/backtest/rsi_pullback_registry_test.go cmd/backtest/main.go
git commit -m "feat(rsi_pullback): реестр стратегии и ветка -strategy rsi_pullback"
```

---

### Task 5: Сетка калибровки и документация

**Files:**
- Create: `data/params/rsi_pullback/grid.json`
- Create: `internal/service/backtest/rsi_pullback_grid_test.go`
- Create: `docs/rsi_pullback/strategy.md`

**Interfaces:**
- Consumes: `core.Params` (Task 1); `ParsePhases(raw []byte) ([]Phase, error)`, `applyField(p any, name string, v float64) (any, error)`, `Phase{Name string; KeepTop int; Grid Grid}` — все из пакета `internal/service/backtest`.
- Produces: `data/params/rsi_pullback/grid.json` — 32 комбинации в 5 фазах; `docs/rsi_pullback/strategy.md`.

- [ ] **Step 1: Написать падающий тест сетки**

Создать `internal/service/backtest/rsi_pullback_grid_test.go`:

```go
package backtest

import (
	"os"
	"path/filepath"
	"testing"

	"tinvest/internal/service/trading_strategy/rsi_pullback/strategy/core"
)

// rsiPullbackGrid loads the shipped grid file.
func rsiPullbackGrid(t *testing.T) []Phase {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join("..", "..", "..", "data", "params", "rsi_pullback", "grid.json"))
	if err != nil {
		t.Fatalf("read grid: %v", err)
	}
	phases, err := ParsePhases(raw)
	if err != nil {
		t.Fatalf("parse grid: %v", err)
	}
	if len(phases) == 0 {
		t.Fatal("grid has no phases")
	}
	return phases
}

// TestRSIPullbackGridFieldsExist drives every swept value through applyField, which errors on an
// unknown field name — so a typo in the grid fails this test.
func TestRSIPullbackGridFieldsExist(t *testing.T) {
	for _, ph := range rsiPullbackGrid(t) {
		for name, values := range ph.Grid {
			if len(values) == 0 {
				t.Fatalf("phase %q: field %q has no values", ph.Name, name)
			}
			for _, v := range values {
				if _, err := applyField(core.DefaultParams(), name, v); err != nil {
					t.Fatalf("phase %q: applyField(%s=%v): %v", ph.Name, name, v, err)
				}
			}
		}
	}
}

// TestRSIPullbackGridHasTimeStopOffPoint pins the one deliberate control point: the time stop
// must be sweepable to "off" (0), while the ATR stop must NOT be — calibration may never choose
// to trade without protection.
func TestRSIPullbackGridHasTimeStopOffPoint(t *testing.T) {
	var sawHoldOff, sawStop bool
	for _, ph := range rsiPullbackGrid(t) {
		for _, v := range ph.Grid["MaxHoldBars"] {
			if v == 0 {
				sawHoldOff = true
			}
		}
		for _, v := range ph.Grid["StopATR"] {
			sawStop = true
			if v == 0 {
				t.Fatal("StopATR=0 is in the grid: calibration must not be able to disable the stop")
			}
		}
	}
	if !sawHoldOff {
		t.Fatal("no MaxHoldBars=0 control point in the grid")
	}
	if !sawStop {
		t.Fatal("the grid never sweeps StopATR")
	}
}

// TestRSIPullbackGridCombos pins the documented size so a silent grid edit is visible.
func TestRSIPullbackGridCombos(t *testing.T) {
	total := 0
	for _, ph := range rsiPullbackGrid(t) {
		n := 1
		for _, values := range ph.Grid {
			n *= len(values)
		}
		total += n
	}
	if total != 32 {
		t.Fatalf("grid has %d combos, want the documented 32", total)
	}
}
```

Примечание: путь к `grid.json` в `rsiPullbackGrid` собран из корня репозитория относительно каталога пакета (`internal/service/backtest` → три уровня вверх). Если соседний grid-тест (`vwap_rev_grid_test.go`) использует другой способ добраться до файла — повтори его способ, чтобы в пакете был один приём, а не два.

- [ ] **Step 2: Прогнать тест и убедиться, что он падает**

Run: `go test ./internal/service/backtest/ -run TestRSIPullbackGrid -v`
Expected: FAIL — файла `data/params/rsi_pullback/grid.json` нет.

- [ ] **Step 3: Написать сетку**

Создать `data/params/rsi_pullback/grid.json`:

```json
{
  "_comment": "rsi_pullback phased grid. Phase 1 (entry) sweeps the entry geometry: the RSI length and the lower band whose DOWNWARD cross opens the trade. Phase 2 (trend) sweeps the EMA pair that gates the entry. Phase 3 (exit) sweeps the upper band whose UPWARD cross closes the trade. Phase 4 (risk) sweeps the ATR stop multiplier — 0 is deliberately ABSENT: calibration must never be able to trade without a stop, even though the core supports StopATR=0. Phase 5 (hold) sweeps the time stop, whose 0 IS a real control point (time stop off). RunPhases expands each phase over the previous phase's keepTop seeds: 12 + 9 + 3 + 4 + 4 = 32 combos. The session bounds (SessionStartMin/SessionEndMin/DayEndMin) and ATRPeriod are fixed by the strategy definition and are deliberately NOT swept. Judge on pooled OOS profit factor from a walk-forward, never on the in-sample best.",
  "phases": [
    {
      "name": "entry",
      "keepTop": 6,
      "grid": {
        "RSIPeriod": [4, 5, 6],
        "RSILower": [5, 10, 15, 20]
      }
    },
    {
      "name": "trend",
      "keepTop": 5,
      "grid": {
        "EMAFast": [5, 10, 20],
        "EMASlow": [50, 100, 200]
      }
    },
    {
      "name": "exit",
      "keepTop": 5,
      "grid": {
        "RSIUpper": [60, 70, 80]
      }
    },
    {
      "name": "risk",
      "keepTop": 5,
      "grid": {
        "StopATR": [0.8, 1.2, 1.6, 2.0]
      }
    },
    {
      "name": "hold",
      "grid": {
        "MaxHoldBars": [0, 4, 8, 12]
      }
    }
  ]
}
```

- [ ] **Step 4: Прогнать тесты сетки**

Run: `go test ./internal/service/backtest/ -run TestRSIPullbackGrid -v`
Expected: PASS

- [ ] **Step 5: Написать документацию**

Создать `docs/rsi_pullback/strategy.md` на русском. Раздел за разделом:

1. **Идея.** Покупка отката внутри восходящего тренда на 30m: тренд задаёт пара EMA, момент входа — заход короткого RSI в нижнюю зону, выход — заход того же RSI в верхнюю. Отличие от `rsi_ema`: там вход по кресту RSI ВВЕРХ через 50 (продолжение импульса), здесь — по кресту ВНИЗ через нижний порог (откат).
2. **Правила входа** (все условия на закрытии одного бара): будний день и 07:00 ≤ время < 17:00 MSK; `EMAFast > EMASlow`; `RSI[i-1] ≥ RSILower` и `RSI[i] < RSILower`; позиции нет. Отдельно указать, что вход — это событие (крест), а не состояние («RSI ниже порога»), и что повторный вход в тот же день разрешён.
3. **Правила выхода** с приоритетом SL → RSI → TIME → EOD, с пояснением, что при одновременном срабатывании стопа и RSI-выхода на одном баре побеждает стоп, и что позиция не переносится на следующий день (`DayEndMin` = 23:00 MSK, то есть позиция живёт и всю вечернюю сессию).
4. **Таблица параметров** с дефолтами и единицами измерения: `RSILower`/`RSIUpper` — пункты RSI, `StopATR` — множитель ATR, `MaxHoldBars` — бары, `SessionStartMin`/`SessionEndMin`/`DayEndMin` — минуты от полуночи MSK. В колонке «в гриде» — да/нет строго по факту `grid.json`. Явно написать: risk-фаза перебирает только множители 0.8–2.0 и стоп никогда не выключает — вариант «без стопа» намеренно не является кандидатом калибровки, хотя ядро выключение поддерживает (`StopATR = 0`). Для `MaxHoldBars` отметить, что `0` в сетке есть и означает «тайм-стоп выключен».
5. **Ограничение по таймфрейму:** 30m — референс; `Lookback` = `max(120, 2×max(EMASlow, RSIPeriod, ATRPeriod) + 20)`, при `EMASlow=200` это 420 баров (~21 сессия). На более мелких интервалах правила работают, но окно прогрева растёт по числу баров, а не по времени.
6. **Команды:**

```
# разведочный прогон на дефолтах
go run ./cmd/backtest -ticker GAZP -strategy rsi_pullback -interval Minutes30 \
  -out ./reports/GAZP_rsipb -months 24

# диагностика одного бара
go run ./cmd/backtest -ticker GAZP -strategy rsi_pullback -interval Minutes30 \
  -months 24 -explain "2026-05-14 14:30"

# walk-forward калибровка
go run ./cmd/backtest -ticker GAZP -strategy rsi_pullback -interval Minutes30 \
  -calibrate data/params/rsi_pullback/grid.json -out ./reports/GAZP_rsipb \
  -months 24 -train-months 12 -test-months 3 -metric profit_factor -min-trades 40

# проверка переносимости победивших параметров без пере-калибровки
go run ./cmd/backtest -ticker NVTK -strategy rsi_pullback -interval Minutes30 \
  -params data/params/rsi_pullback/gazp_cal.json -out ./reports/NVTK_rsipb -months 24
```

7. **Заметка про вердикт:** вердикт о жизнеспособности выносит владелец по своим прогонам; вердикт читается по pooled OOS walk-forward, а не по in-sample лучшему. Комиссию (`-commission`, дефолт `0.0005` за сторону) не переопределять.

- [ ] **Step 6: Полный гейт**

Run: `./bin/mage ci`
Expected: PASS

- [ ] **Step 7: Коммит**

```bash
git add data/params/rsi_pullback/ internal/service/backtest/rsi_pullback_grid_test.go docs/rsi_pullback/
git commit -m "feat(rsi_pullback): фазовая сетка калибровки и документация стратегии"
```

---

## Проверка плана против спеки

| Требование спеки | Задача |
|---|---|
| Вход: окно 07:00–17:00 MSK, будни, `EMAFast > EMASlow`, крест RSI вниз через `RSILower` | Task 1 |
| Вход — событие, а не состояние; повторный вход в тот же день разрешён | Task 1 (тест `TestEnterCrossIsAnEventNotAState`), Task 5 (дока) |
| Выходы SL → RSI → TIME → EOD, приоритет стопа на общем баре | Task 2 |
| Тайм-стоп выключается нулём; неизвестное время входа его не запускает | Task 2 |
| Позиция не переносится через ночь (`DayEndMin` = 23:00 MSK), живёт всю вечернюю сессию | Task 2 |
| `Explain` для `-explain` | Task 2 |
| Все поля `Params` — `int`/`float64`; `Lookback` покрывает `EMASlow` | Task 1 |
| Анти-lookahead сеть, включая ветку с открытой позицией | Task 3 |
| Реестр и ветка CLI, без HTF | Task 4 |
| Фазовая сетка 32 комбо; `MaxHoldBars=0` есть, `StopATR=0` намеренно нет | Task 5 |
| Документация стратегии с командами | Task 5 |
| Стратегия backtest-only (нет правок в `internal/app`, live, Telegram) | Task 4 (ограничение соблюдается во всех задачах) |
| Вердикт выносит владелец сам | вне плана, отмечено в доке (Task 5) |
