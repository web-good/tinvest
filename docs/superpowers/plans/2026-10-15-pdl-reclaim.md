# pdl_reclaim Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Backtest-only long strategy `pdl_reclaim` on 5m bars: buy a false break of yesterday's low (A) or a bullish engulfing under it (B); exit on half-daily-ATR stop, yesterday's high, RSI(6) ≥ 70, a MACD histogram turning red after being green, or end of day.

**Architecture:** A pure, stateless core `internal/service/trading_strategy/pdl_reclaim/strategy/core` (modelled on `gap_fade/strategy/core`: intraday, levels and ATR from engine dailies, EOD guard) plus a per-ticker registry in `internal/service/backtest`, wired into `cmd/backtest -strategy pdl_reclaim -interval Minutes5`, a universe env var for the future live runner, and mechanics docs. Ticker packages come later with calibration (GAZP, SBER, NVTK first).

**Tech Stack:** Go 1.25, `tinvest/pkg/indicators` (`RSISeries`, `MACD`, `ATR`), backtest engine `tinvest/internal/domain/backtest`, `heetch/confita`.

**Spec:** `docs/superpowers/specs/2026-10-15-pdl-reclaim-design.md`

## Global Constraints

- Only long; only `Minutes5` (any other interval is an error from the registry).
- Calendar rules in MSK (`Europe/Moscow`, UTC fallback); entries Mon–Fri only.
- Entry buys at the next bar's open (`Config.EntryAtNextOpen`), `Signal.FillBy = bar time + 5m`.
- Defaults: `EntryA 1`, `EntryB 1`, `StopDailyATR 0.5`, `DailyATRPeriod 14`, `TakeProfitPDH 1`, `RSIPeriod 6`, `RSIExit 70`, `MACDFast/Slow/Signal 12/26/9`, `EntryFrom 07:00`, `EntryTo 18:00`, `Exit 18:30`.
- Every `Params` field is `int` or `float64` (reflection grid calibration).
- Stop is measured from the fill price (`Position.PurchasePrice − StopDailyATR · Position.EntryATR`), not from `Position.StopLoss`.
- Exit priority: SL → TP → RSI → MACD → EOD. Reasons: `"SL"` (stop-style, already in `model.IsStopReason`), `"TP"`, `"RSI"`, `"MACD"`, `"EOD"`.
- Go identifiers use the `PDL` acronym: `PDLReclaimLookupOrGeneric`, `pdlReclaimRegistry`, `PDLReclaimConfig` (the spec's `PdlReclaim…` spelling is replaced by idiomatic Go).
- Build with `go build ./internal/... ./pkg/... ./cmd/...` (never `./...` — `magefiles` has no `main`).
- Commit messages end with `Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>`.
- Docs under `docs/pdl_reclaim/` describe mechanics only — no per-ticker results.

## Review Focus

1. A signal late in the day with a widened window (`EntryToHour 24`): the fill bar would be the next date — must not enter (test in Task 1).
2. A held position whose `EntryTime` is zero or no longer in the window: MACD exit must silently stay off, no panic (test in Task 2).
3. A thin ticker with no 18:30 bar: the first bar of the next date must close the position (test in Task 2).
4. The last completed daily is a weekend session: PDL/PDH must come from Friday, not Sunday (test in Task 1).
5. Stop and target touched in one bar: stop wins (test in Task 2).

---

### Task 1: Core — params, levels, entries A and B

**Files:**
- Create: `internal/service/trading_strategy/pdl_reclaim/strategy/core/core.go`
- Test: `internal/service/trading_strategy/pdl_reclaim/strategy/core/core_test.go`

**Interfaces:**
- Produces:
  - `type Params struct { EntryA, EntryB int; StopDailyATR float64; DailyATRPeriod, TakeProfitPDH, RSIPeriod int; RSIExit float64; MACDFast, MACDSlow, MACDSignal, EntryFromHour, EntryFromMinute, EntryToHour, EntryToMinute, ExitHour, ExitMinute int }`
  - `func DefaultParams() Params`
  - `type Strategy`; `func NewWithParams(ticker string, step time.Duration, p Params) *Strategy`; `Ticker() string`; `Lookback() int`; `Decide(md strategy.MarketData) model.Signal`
  - package helpers used by Tasks 2–3: `mskLoc`, `dayKey(time.Time) int`, `minuteOf(time.Time) int`, `(s *Strategy) levels(md) (pdl, pdh float64, ok bool)`, `(s *Strategy) dailyATR(md) float64`, `(s *Strategy) inEntryWindow(t time.Time) bool`, `isA(md, i, pdl) bool`, `isB(md, i, pdl) bool`, `(s *Strategy) manage(md, sig) model.Signal` (stub here, real in Task 2)
  - test helpers in `core_test.go` used by Tasks 2–3: `eps`, `at`, `bar`, `flat`, `dailyBars`, `fixture`, `newS`, `decide`, `pre`, `barA`, `redB`, `greenB`

- [ ] **Step 1: Write the failing tests**

`internal/service/trading_strategy/pdl_reclaim/strategy/core/core_test.go`:

```go
package core

import (
	"math"
	"strings"
	"testing"
	"time"

	"tinvest/internal/service/trading_strategy/scalping/model"
	"tinvest/internal/service/trading_strategy/scalping/strategy"
)

const eps = 1e-9

// at parses "2006-01-02 15:04" as MSK.
func at(s string) time.Time {
	t, err := time.ParseInLocation("2006-01-02 15:04", s, mskLoc)
	if err != nil {
		panic(err)
	}
	return t
}

type bar struct {
	t          string
	o, h, l, c float64
}

// flat returns flat bars (o=h=l=c=p) every 5 minutes over [from, to], both MSK.
func flat(from, to string, p float64) []bar {
	var out []bar
	for t := at(from); !t.After(at(to)); t = t.Add(5 * time.Minute) {
		out = append(out, bar{t.Format("2006-01-02 15:04"), p, p, p, p})
	}
	return out
}

// dailyBars returns `days` calendar days of completed dailies ending the day before `before`, all
// closing at 100. Weekdays span 99..101 (PDL 99, PDH 101, Wilder ATR exactly 2); weekend sessions
// span 99.9..100.1, so a weekend daily leaking into the levels is visible.
func dailyBars(before time.Time, days int) (highs, lows, closes []float64, times []time.Time) {
	b := before.In(mskLoc)
	start := time.Date(b.Year(), b.Month(), b.Day(), 0, 0, 0, 0, mskLoc).AddDate(0, 0, -days)
	for i := 0; i < days; i++ {
		d := start.AddDate(0, 0, i)
		half := 1.0
		if d.Weekday() == time.Saturday || d.Weekday() == time.Sunday {
			half = 0.1
		}
		highs = append(highs, 100+half)
		lows = append(lows, 100-half)
		closes = append(closes, 100)
		times = append(times, d.Add(10*time.Hour))
	}
	return highs, lows, closes, times
}

// fixture builds MarketData from bars plus 40 days of dailies before the last bar.
func fixture(bars []bar) strategy.MarketData {
	var md strategy.MarketData
	for _, b := range bars {
		md.Times = append(md.Times, at(b.t))
		md.Opens = append(md.Opens, b.o)
		md.Highs = append(md.Highs, b.h)
		md.Lows = append(md.Lows, b.l)
		md.Closes = append(md.Closes, b.c)
	}
	md.Price = md.Closes[len(md.Closes)-1]
	md.DailyHighs, md.DailyLows, md.DailyCloses, md.DailyTimes = dailyBars(md.Times[len(md.Times)-1], 40)
	return md
}

func newS(p Params) *Strategy { return NewWithParams("TEST", 5*time.Minute, p) }

func decide(p Params, bars []bar) model.Signal { return newS(p).Decide(fixture(bars)) }

// Monday 2026-09-14. PDL 99, PDH 101 (Friday's daily), daily ATR 2.
var (
	pre    = bar{"2026-09-14 09:55", 99.5, 99.5, 99.5, 99.5}
	barA   = bar{"2026-09-14 10:00", 99.2, 99.5, 98.8, 99.3}   // green, low < PDL, close > PDL
	redB   = bar{"2026-09-14 09:55", 98.9, 98.95, 98.5, 98.6}  // red, close < PDL
	greenB = bar{"2026-09-14 10:00", 98.55, 98.95, 98.4, 98.92} // close > red open, low < red low, close < PDL (not A)
)

func TestEntryA(t *testing.T) {
	sig := decide(DefaultParams(), []bar{pre, barA})
	if sig.Kind != model.SignalBuy {
		t.Fatalf("Kind = %v, want Buy", sig.Kind)
	}
	if math.Abs(sig.StopLoss-98.3) > eps || math.Abs(sig.TakeProfit-101) > eps || math.Abs(sig.ATR-2) > eps || math.Abs(sig.Level-99) > eps {
		t.Fatalf("SL/TP/ATR/Level = %v/%v/%v/%v, want 98.3/101/2/99", sig.StopLoss, sig.TakeProfit, sig.ATR, sig.Level)
	}
	if !sig.FillBy.Equal(at("2026-09-14 10:05")) {
		t.Fatalf("FillBy = %v, want 10:05", sig.FillBy)
	}
	if !strings.HasPrefix(sig.EntryReason, "A") {
		t.Fatalf("EntryReason = %q, want prefix A", sig.EntryReason)
	}
}

func TestEntryANeedsEveryCondition(t *testing.T) {
	for name, b := range map[string]bar{
		"red":             {"2026-09-14 10:00", 99.4, 99.5, 98.8, 99.3},
		"low at PDL":      {"2026-09-14 10:00", 99.2, 99.5, 99.0, 99.3},
		"close at PDL":    {"2026-09-14 10:00", 98.9, 99.5, 98.8, 99.0},
		"close below PDL": {"2026-09-14 10:00", 98.85, 99.5, 98.8, 98.95},
	} {
		if sig := decide(DefaultParams(), []bar{pre, b}); sig.Kind != model.SignalNone {
			t.Errorf("%s: Kind = %v, want None", name, sig.Kind)
		}
	}
}

func TestEntryB(t *testing.T) {
	sig := decide(DefaultParams(), []bar{redB, greenB})
	if sig.Kind != model.SignalBuy {
		t.Fatalf("Kind = %v, want Buy", sig.Kind)
	}
	if math.Abs(sig.StopLoss-97.92) > eps || math.Abs(sig.TakeProfit-101) > eps {
		t.Fatalf("SL/TP = %v/%v, want 97.92/101", sig.StopLoss, sig.TakeProfit)
	}
	if !strings.HasPrefix(sig.EntryReason, "B") {
		t.Fatalf("EntryReason = %q, want prefix B", sig.EntryReason)
	}
}

func TestEntryBNeedsEveryCondition(t *testing.T) {
	for name, pair := range map[string][2]bar{
		"red closes above PDL": {{"2026-09-14 09:55", 99.4, 99.4, 99.05, 99.1}, {"2026-09-14 10:00", 99.05, 99.45, 99.0, 99.42}},
		"close equals red open": {redB, {"2026-09-14 10:00", 98.55, 98.95, 98.4, 98.9}},
		"low equals red low":    {redB, {"2026-09-14 10:00", 98.55, 98.95, 98.5, 98.92}},
		"previous is a doji":    {{"2026-09-14 09:55", 98.6, 98.95, 98.5, 98.6}, greenB},
		"current is red":        {redB, {"2026-09-14 10:00", 98.95, 98.95, 98.4, 98.92}},
		"different dates":       {{"2026-09-11 23:55", 98.9, 98.95, 98.5, 98.6}, {"2026-09-14 07:00", 98.55, 98.95, 98.4, 98.92}},
	} {
		if sig := decide(DefaultParams(), pair[:]); sig.Kind != model.SignalNone {
			t.Errorf("%s: Kind = %v, want None", name, sig.Kind)
		}
	}
}

func TestEntryBothPatternsReportA(t *testing.T) {
	both := []bar{{"2026-09-14 09:55", 99.2, 99.2, 98.7, 98.8}, {"2026-09-14 10:00", 98.75, 99.4, 98.6, 99.3}}
	if sig := decide(DefaultParams(), both); sig.Kind != model.SignalBuy || !strings.HasPrefix(sig.EntryReason, "A") {
		t.Fatalf("both: %v %q, want Buy with reason A", sig.Kind, sig.EntryReason)
	}
	p := DefaultParams()
	p.EntryA = 0
	if sig := decide(p, both); sig.Kind != model.SignalBuy || !strings.HasPrefix(sig.EntryReason, "B") {
		t.Fatalf("EntryA off: %v %q, want Buy with reason B", sig.Kind, sig.EntryReason)
	}
}

func TestEntrySwitches(t *testing.T) {
	p := DefaultParams()
	p.EntryA = 0
	if sig := decide(p, []bar{pre, barA}); sig.Kind != model.SignalNone {
		t.Errorf("EntryA off: Kind = %v, want None", sig.Kind)
	}
	p = DefaultParams()
	p.EntryB = 0
	if sig := decide(p, []bar{redB, greenB}); sig.Kind != model.SignalNone {
		t.Errorf("EntryB off: Kind = %v, want None", sig.Kind)
	}
}

func TestEntryWindow(t *testing.T) {
	for ts, want := range map[string]bool{
		"2026-09-14 06:55": false,
		"2026-09-14 07:00": true,
		"2026-09-14 17:50": true,  // buy at 17:55
		"2026-09-14 17:55": false, // buy at 18:00 is too late
		"2026-09-12 10:00": false, // Saturday
	} {
		b := barA
		b.t = ts
		p := pre
		p.t = at(ts).Add(-5 * time.Minute).Format("2006-01-02 15:04")
		if got := decide(DefaultParams(), []bar{p, b}).Kind == model.SignalBuy; got != want {
			t.Errorf("%s: buy = %v, want %v", ts, got, want)
		}
	}
}

// Review focus 1: with the window widened to midnight the fill bar of a 23:55 signal is the next
// date — no entry.
func TestEntryFillMustBeSameDate(t *testing.T) {
	p := DefaultParams()
	p.EntryToHour = 24
	b := barA
	b.t = "2026-09-14 23:55"
	q := pre
	q.t = "2026-09-14 23:50"
	if sig := decide(p, []bar{q, b}); sig.Kind != model.SignalNone {
		t.Fatalf("Kind = %v, want None", sig.Kind)
	}
	b.t, q.t = "2026-09-14 23:50", "2026-09-14 23:45"
	if sig := decide(p, []bar{q, b}); sig.Kind != model.SignalBuy {
		t.Fatalf("23:50: Kind = %v, want Buy", sig.Kind)
	}
}

func TestEntryCloseAtPDH(t *testing.T) {
	wide := bar{"2026-09-14 10:00", 99.2, 101.2, 98.8, 101.0}
	if sig := decide(DefaultParams(), []bar{pre, wide}); sig.Kind != model.SignalNone {
		t.Fatalf("target on: Kind = %v, want None", sig.Kind)
	}
	p := DefaultParams()
	p.TakeProfitPDH = 0
	sig := decide(p, []bar{pre, wide})
	if sig.Kind != model.SignalBuy || sig.TakeProfit != 0 {
		t.Fatalf("target off: %v TP %v, want Buy with no target", sig.Kind, sig.TakeProfit)
	}
}

// Review focus 4: Sunday's daily (99.9..100.1) is not the reference day; Friday's (99..101) is.
func TestEntryIgnoresWeekendDaily(t *testing.T) {
	b := bar{"2026-09-14 10:00", 99.6, 99.97, 99.5, 99.95} // a reclaim of 99.9, not of 99
	if sig := decide(DefaultParams(), []bar{pre, b}); sig.Kind != model.SignalNone {
		t.Fatalf("Kind = %v, want None", sig.Kind)
	}
}

func TestEntryNeedsDailiesAndStop(t *testing.T) {
	md := fixture([]bar{pre, barA})
	md.DailyHighs, md.DailyLows, md.DailyCloses, md.DailyTimes = nil, nil, nil, nil
	if sig := newS(DefaultParams()).Decide(md); sig.Kind != model.SignalNone {
		t.Errorf("no dailies: Kind = %v, want None", sig.Kind)
	}
	p := DefaultParams()
	p.StopDailyATR = 0
	if sig := decide(p, []bar{pre, barA}); sig.Kind != model.SignalNone {
		t.Errorf("no stop: Kind = %v, want None", sig.Kind)
	}
}

func TestLookbackCoversWarmupAndDay(t *testing.T) {
	if got := newS(DefaultParams()).Lookback(); got != 3*(26+9)+dayBars {
		t.Fatalf("Lookback = %d, want %d", got, 3*(26+9)+dayBars)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/service/trading_strategy/pdl_reclaim/strategy/core/`
Expected: FAIL to compile — `undefined: mskLoc`, `undefined: NewWithParams`, …

- [ ] **Step 3: Write the implementation**

`internal/service/trading_strategy/pdl_reclaim/strategy/core/core.go`:

```go
// Package core implements pdl_reclaim: a long-only intraday strategy on 5-minute bars around
// yesterday's range. Yesterday's low (PDL) is support, yesterday's high (PDH) the target. It buys
// the next bar's open after a green bar that dipped below PDL and closed back above it (entry A),
// or after a bullish engulfing whose red bar closed below PDL (entry B). Exits, in priority order:
// a stop StopDailyATR daily ATRs below the fill, PDH, RSI(RSIPeriod) at or above RSIExit, the
// first red MACD histogram bar after a green one inside the trade, and the end of day. Pure,
// stateless between bars and ticker-agnostic. Run with `-strategy pdl_reclaim -interval Minutes5`.
package core

import (
	"fmt"
	"time"

	"tinvest/internal/service/trading_strategy/scalping/model"
	"tinvest/internal/service/trading_strategy/scalping/strategy"
	"tinvest/pkg/indicators"
)

// dayBars is the number of 5-minute bars from 07:00 to the 18:30 exit bar inclusive: an open
// position is never older than that, so the window always reaches back to its entry bar.
const dayBars = 139

// minWarmup keeps the indicator warm-up sane when the periods are tiny.
const minWarmup = 64

// Params holds every tunable. All fields are int or float64 so reflection grid calibration can
// sweep them.
type Params struct {
	EntryA          int     // 1 — false break of PDL (green bar, low < PDL < close) enters
	EntryB          int     // 1 — bullish engulfing under PDL enters
	StopDailyATR    float64 // stop below the fill in daily ATR; 0 — no entries (no risk unit)
	DailyATRPeriod  int     // Wilder period over completed weekday dailies
	TakeProfitPDH   int     // 1 — target at yesterday's high; 0 — no target
	RSIPeriod       int     // 5m RSI for the exit; 0 — RSI exit off
	RSIExit         float64 // RSI at/above this sells; 0 — RSI exit off
	MACDFast        int     // 5m MACD for the exit; invalid periods — MACD exit off
	MACDSlow        int
	MACDSignal      int
	EntryFromHour   int // the signal bar starts at/after EntryFrom MSK
	EntryFromMinute int
	EntryToHour     int // the buy bar starts before EntryTo MSK, on the signal's date
	EntryToMinute   int
	ExitHour        int // the first bar starting at/after Exit MSK sells at its close
	ExitMinute      int
}

// DefaultParams returns the owner's rules.
func DefaultParams() Params {
	return Params{
		EntryA:          1,
		EntryB:          1,
		StopDailyATR:    0.5,
		DailyATRPeriod:  14,
		TakeProfitPDH:   1,
		RSIPeriod:       6,
		RSIExit:         70,
		MACDFast:        12,
		MACDSlow:        26,
		MACDSignal:      9,
		EntryFromHour:   7,
		EntryFromMinute: 0,
		EntryToHour:     18,
		EntryToMinute:   0,
		ExitHour:        18,
		ExitMinute:      30,
	}
}

// Strategy trades a single instrument with the pdl_reclaim rules. Ticker-agnostic and pure.
type Strategy struct {
	ticker string
	step   time.Duration
	p      Params
}

// NewWithParams returns the strategy for a ticker; step is the bar length (5m).
func NewWithParams(ticker string, step time.Duration, p Params) *Strategy {
	return &Strategy{ticker: ticker, step: step, p: p}
}

func (s *Strategy) Ticker() string { return s.ticker }

// Lookback warms MACD and RSI before the oldest possible entry bar of a held position.
func (s *Strategy) Lookback() int {
	return max(minWarmup, 3*(s.p.MACDSlow+s.p.MACDSignal), 4*s.p.RSIPeriod+2) + dayBars
}

// mskLoc anchors every calendar rule to Moscow (UTC fallback).
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

// dayKey is the MSK calendar date of t as yyyymmdd.
func dayKey(t time.Time) int {
	tl := t.In(mskLoc)
	return tl.Year()*10000 + int(tl.Month())*100 + tl.Day()
}

// minuteOf is the MSK minute of the day of t.
func minuteOf(t time.Time) int {
	tl := t.In(mskLoc)
	return tl.Hour()*60 + tl.Minute()
}

// weekdayDaily drops weekend (Sat/Sun MSK) dailies, keeping the price slices aligned. When times
// is empty or misaligned the series is returned untouched rather than guessed at.
func weekdayDaily(highs, lows, closes []float64, times []time.Time) (h, l, c []float64) {
	n := len(closes)
	if n == 0 || len(highs) != n || len(lows) != n || len(times) != n {
		return highs, lows, closes
	}
	h = make([]float64, 0, n)
	l = make([]float64, 0, n)
	c = make([]float64, 0, n)
	for i := 0; i < n; i++ {
		if isWeekend(times[i].In(mskLoc)) {
			continue
		}
		h = append(h, highs[i])
		l = append(l, lows[i])
		c = append(c, closes[i])
	}
	return h, l, c
}

// levels returns yesterday's low and high: the last completed weekday daily.
func (s *Strategy) levels(md strategy.MarketData) (pdl, pdh float64, ok bool) {
	h, l, c := weekdayDaily(md.DailyHighs, md.DailyLows, md.DailyCloses, md.DailyTimes)
	n := len(c)
	if n == 0 || len(h) != n || len(l) != n {
		return 0, 0, false
	}
	return l[n-1], h[n-1], l[n-1] > 0 && h[n-1] >= l[n-1]
}

// dailyATR is Wilder's ATR over completed weekday dailies; 0 when the data cannot support it.
func (s *Strategy) dailyATR(md strategy.MarketData) float64 {
	if s.p.DailyATRPeriod <= 0 {
		return 0
	}
	h, l, c := weekdayDaily(md.DailyHighs, md.DailyLows, md.DailyCloses, md.DailyTimes)
	if len(c) < s.p.DailyATRPeriod+1 || len(h) != len(c) || len(l) != len(c) {
		return 0
	}
	return indicators.ATR(h, l, c, s.p.DailyATRPeriod)
}

// inEntryWindow reports whether a signal on the bar starting at t may buy: a weekday, the bar
// starts at/after EntryFrom, and the buy bar (t+step) starts before EntryTo on the same date.
func (s *Strategy) inEntryWindow(t time.Time) bool {
	if isWeekend(t.In(mskLoc)) || minuteOf(t) < s.p.EntryFromHour*60+s.p.EntryFromMinute {
		return false
	}
	fill := t.Add(s.step)
	return dayKey(fill) == dayKey(t) && minuteOf(fill) < s.p.EntryToHour*60+s.p.EntryToMinute
}

// isA: bar i is green, dipped below PDL and closed back above it.
func isA(md strategy.MarketData, i int, pdl float64) bool {
	return md.Closes[i] > md.Opens[i] && md.Lows[i] < pdl && md.Closes[i] > pdl
}

// isB: bar i-1 is red and closed below PDL; bar i, on the same date, is green, closes above the
// red bar's open and trades below its low.
func isB(md strategy.MarketData, i int, pdl float64) bool {
	if i < 1 || dayKey(md.Times[i]) != dayKey(md.Times[i-1]) {
		return false
	}
	o0, c0, l0 := md.Opens[i-1], md.Closes[i-1], md.Lows[i-1]
	o1, c1, l1 := md.Opens[i], md.Closes[i], md.Lows[i]
	return c0 < o0 && c0 < pdl && c1 > o1 && c1 > o0 && l1 < l0
}

// pattern names the entry bar i matches ("A" wins over "B"), or "" for none.
func (s *Strategy) pattern(md strategy.MarketData, i int, pdl float64) string {
	switch {
	case s.p.EntryA > 0 && isA(md, i, pdl):
		return "A"
	case s.p.EntryB > 0 && isB(md, i, pdl):
		return "B"
	}
	return ""
}

func (s *Strategy) Decide(md strategy.MarketData) model.Signal {
	sig := model.Signal{Ticker: s.ticker, Price: md.Price}
	if md.Position != nil {
		return s.manage(md, sig)
	}
	return s.enter(md, sig)
}

// enter buys the next bar's open on entry A or B. The cheap bar gates run before the daily ATR.
func (s *Strategy) enter(md strategy.MarketData, sig model.Signal) model.Signal {
	n := len(md.Closes)
	if n < 2 || len(md.Times) != n || len(md.Opens) != n || len(md.Highs) != n || len(md.Lows) != n {
		return sig
	}
	if s.p.StopDailyATR <= 0 {
		return sig
	}
	i := n - 1
	if !s.inEntryWindow(md.Times[i]) {
		return sig
	}
	pdl, pdh, ok := s.levels(md)
	if !ok {
		return sig
	}
	kind := s.pattern(md, i, pdl)
	if kind == "" {
		return sig
	}
	closeP := md.Closes[i]
	if s.p.TakeProfitPDH > 0 && closeP >= pdh {
		return sig
	}
	atr := s.dailyATR(md)
	if atr <= 0 {
		return sig
	}
	stop := closeP - s.p.StopDailyATR*atr
	if stop <= 0 {
		return sig
	}
	sig.Kind = model.SignalBuy
	sig.StopLoss, sig.ATR, sig.Level = stop, atr, pdl
	if s.p.TakeProfitPDH > 0 {
		sig.TakeProfit = pdh
	}
	sig.FillBy = md.Times[i].Add(s.step)
	sig.EntryReason = fmt.Sprintf("%s: PDL %.4f, PDH %.4f, сигнал по close %.4f, дневной ATR %.4f, стоп %.4f",
		kind, pdl, pdh, closeP, atr, stop)
	return sig
}

// manage is filled in by Task 2.
func (s *Strategy) manage(md strategy.MarketData, sig model.Signal) model.Signal { return sig }
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/service/trading_strategy/pdl_reclaim/strategy/core/`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/service/trading_strategy/pdl_reclaim
git commit -m "feat(pdl_reclaim): core params, levels and entries A/B

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

---

### Task 2: Core — exits

**Files:**
- Modify: `internal/service/trading_strategy/pdl_reclaim/strategy/core/core.go` (replace the `manage` stub, add helpers)
- Test: `internal/service/trading_strategy/pdl_reclaim/strategy/core/exits_test.go`

**Interfaces:**
- Consumes (Task 1): `Params`, `DefaultParams`, `newS`, `fixture`, `at`, `bar`, `flat`, `eps`, `dayKey`, `minuteOf`, `mskLoc`, `(s *Strategy) dailyATR`.
- Produces: `func histogram(closes []float64, p Params) []float64`, `func entryIndex(times []time.Time, entry time.Time) int`, `func macdExit(hist []float64, e, i int) bool`, `(s *Strategy) eodReason(md, pos) string` — Task 3's Explain calls them.

- [ ] **Step 1: Write the failing tests**

`internal/service/trading_strategy/pdl_reclaim/strategy/core/exits_test.go`:

```go
package core

import (
	"math"
	"testing"
	"time"

	"tinvest/internal/service/trading_strategy/scalping/model"
	"tinvest/internal/service/trading_strategy/scalping/strategy"
	"tinvest/pkg/indicators"
)

// quiet switches off the RSI and MACD exits so stop/target/EOD are tested alone.
func quiet() Params {
	p := DefaultParams()
	p.RSIPeriod, p.MACDFast = 0, 0
	return p
}

func held(p Params, bars []bar, pos strategy.Position) model.Signal {
	md := fixture(bars)
	md.Position = &pos
	return newS(p).Decide(md)
}

// pos0 filled at 99.4 on Monday 10:05: stop 99.4 − 0.5·2 = 98.4, target PDH 101. Its frozen
// StopLoss (98.3, from the signal close) is deliberately different: the stop is measured from the fill.
func pos0() strategy.Position {
	return strategy.Position{PurchasePrice: 99.4, EntryATR: 2, TakeProfit: 101, StopLoss: 98.3, EntryTime: at("2026-09-14 10:05")}
}

var fillBar = bar{"2026-09-14 10:05", 99.35, 99.5, 99.3, 99.4}

func TestExitStopFromFill(t *testing.T) {
	sig := held(quiet(), []bar{fillBar, {"2026-09-14 10:10", 99.4, 99.5, 98.4, 98.6}}, pos0())
	if sig.Kind != model.SignalSell || sig.Reason != "SL" || math.Abs(sig.StopLoss-98.4) > eps {
		t.Fatalf("got %v %q SL %v, want Sell SL at 98.4", sig.Kind, sig.Reason, sig.StopLoss)
	}
}

// Review focus 5.
func TestExitStopBeatsTarget(t *testing.T) {
	sig := held(quiet(), []bar{fillBar, {"2026-09-14 10:10", 99.4, 101.2, 98.3, 100}}, pos0())
	if sig.Reason != "SL" {
		t.Fatalf("Reason = %q, want SL", sig.Reason)
	}
}

func TestExitTarget(t *testing.T) {
	up := bar{"2026-09-14 10:10", 99.4, 101.0, 99.3, 100.9}
	sig := held(quiet(), []bar{fillBar, up}, pos0())
	if sig.Kind != model.SignalSell || sig.Reason != "TP" || math.Abs(sig.TakeProfit-101) > eps {
		t.Fatalf("got %v %q TP %v, want Sell TP at 101", sig.Kind, sig.Reason, sig.TakeProfit)
	}
	noTarget := pos0()
	noTarget.TakeProfit = 0
	if sig := held(quiet(), []bar{fillBar, up}, noTarget); sig.Kind != model.SignalNone {
		t.Fatalf("no target: Kind = %v, want None", sig.Kind)
	}
}

func TestExitHoldsInsideRange(t *testing.T) {
	if sig := held(quiet(), []bar{fillBar, {"2026-09-14 10:10", 99.4, 99.6, 99.3, 99.5}}, pos0()); sig.Kind != model.SignalNone {
		t.Fatalf("Kind = %v, want None", sig.Kind)
	}
}

func TestExitEndOfDay(t *testing.T) {
	bars := append([]bar{fillBar}, flat("2026-09-14 10:10", "2026-09-14 18:25", 99.5)...)
	if sig := held(quiet(), bars, pos0()); sig.Kind != model.SignalNone {
		t.Fatalf("18:25: Kind = %v, want None", sig.Kind)
	}
	bars = append(bars, bar{"2026-09-14 18:30", 99.5, 99.5, 99.5, 99.5})
	if sig := held(quiet(), bars, pos0()); sig.Reason != "EOD" {
		t.Fatalf("18:30: Reason = %q, want EOD", sig.Reason)
	}
}

// Review focus 3: no 18:30 bar in the data — the first bar of the next date closes the position.
func TestExitNextDate(t *testing.T) {
	bars := []bar{fillBar, {"2026-09-14 16:00", 99.5, 99.5, 99.5, 99.5}, {"2026-09-15 07:00", 99.5, 99.5, 99.5, 99.5}}
	if sig := held(quiet(), bars, pos0()); sig.Reason != "EOD" {
		t.Fatalf("Reason = %q, want EOD", sig.Reason)
	}
}

func TestExitRSI(t *testing.T) {
	p := DefaultParams()
	p.MACDFast = 0
	bars := []bar{fillBar}
	for k := 1; k <= 10; k++ {
		c := 99.4 + 0.05*float64(k)
		bars = append(bars, bar{at("2026-09-14 10:05").Add(time.Duration(k) * 5 * time.Minute).Format("2006-01-02 15:04"), c - 0.05, c + 0.01, c - 0.06, c})
	}
	if sig := held(p, bars, pos0()); sig.Reason != "RSI" {
		t.Fatalf("Reason = %q, want RSI", sig.Reason)
	}
	p.RSIExit = 0
	if sig := held(p, bars, pos0()); sig.Kind != model.SignalNone {
		t.Fatalf("RSI off: Kind = %v, want None", sig.Kind)
	}
}

// macdPath: 60 flat bars at 100 from Monday 07:00, 12 bars rising by 0.1, 30 bars falling by 0.1.
func macdPath() []bar {
	var out []bar
	c := 100.0
	for k := 0; k < 102; k++ {
		switch {
		case k >= 72:
			c -= 0.1
		case k >= 60:
			c += 0.1
		}
		ts := at("2026-09-14 07:00").Add(time.Duration(k) * 5 * time.Minute).Format("2006-01-02 15:04")
		out = append(out, bar{ts, c, c, c, c})
	}
	return out
}

// firstRed is the first bar after the rise starts whose histogram is red right after a green one.
func firstRed(t *testing.T, bars []bar) (int, []float64) {
	t.Helper()
	closes := make([]float64, len(bars))
	for k, b := range bars {
		closes[k] = b.c
	}
	m, s := indicators.MACD(closes, 12, 26, 9)
	h := make([]float64, len(m))
	for k := range m {
		h[k] = m[k] - s[k]
	}
	for r := 61; r < len(h)-1; r++ {
		if h[r] < 0 && h[r-1] >= 0 {
			if h[r+1] >= 0 {
				t.Fatalf("histogram at %d is green again", r+1)
			}
			return r, h
		}
	}
	t.Fatal("no red-after-green bar in the path")
	return 0, nil
}

func macdPos(entry bar) strategy.Position {
	return strategy.Position{PurchasePrice: entry.c, EntryATR: 100, EntryTime: at(entry.t)} // stop far, no target
}

func TestExitMACDAfterGreen(t *testing.T) {
	p := DefaultParams()
	p.RSIPeriod = 0
	bars := macdPath()
	r, _ := firstRed(t, bars)
	if sig := held(p, bars[:r+1], macdPos(bars[60])); sig.Reason != "MACD" {
		t.Fatalf("first red: Reason = %q, want MACD", sig.Reason)
	}
	if sig := held(p, bars[:r], macdPos(bars[60])); sig.Kind != model.SignalNone {
		t.Fatalf("still green: Kind = %v, want None", sig.Kind)
	}
	// green on the entry bar itself counts
	if sig := held(p, bars[:r+1], macdPos(bars[r-1])); sig.Reason != "MACD" {
		t.Fatalf("green at entry: Reason = %q, want MACD", sig.Reason)
	}
	// green only before the entry does not
	if sig := held(p, bars[:r+2], macdPos(bars[r])); sig.Kind != model.SignalNone {
		t.Fatalf("green before entry: Kind = %v, want None", sig.Kind)
	}
}

// Review focus 2: no entry time, or an entry time outside the window — MACD exit stays off.
func TestExitMACDWithoutEntryBar(t *testing.T) {
	p := DefaultParams()
	p.RSIPeriod = 0
	bars := macdPath()
	r, _ := firstRed(t, bars)
	pos := macdPos(bars[60])
	pos.EntryTime = time.Time{}
	if sig := held(p, bars[:r+1], pos); sig.Kind != model.SignalNone {
		t.Fatalf("zero EntryTime: Kind = %v, want None", sig.Kind)
	}
	pos.EntryTime = at("2026-09-14 06:00")
	if sig := held(p, bars[:r+1], pos); sig.Kind != model.SignalNone {
		t.Fatalf("EntryTime before window: Kind = %v, want None", sig.Kind)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/service/trading_strategy/pdl_reclaim/strategy/core/ -run TestExit`
Expected: FAIL — stop/target/EOD/RSI/MACD tests report `Kind = 0` / `Reason = ""`.

- [ ] **Step 3: Write the implementation**

In `core.go` replace the stub

```go
// manage is filled in by Task 2.
func (s *Strategy) manage(md strategy.MarketData, sig model.Signal) model.Signal { return sig }
```

with:

```go
// manage exits in priority order: stop, target, RSI, MACD, end of day.
func (s *Strategy) manage(md strategy.MarketData, sig model.Signal) model.Signal {
	pos := md.Position
	n := len(md.Closes)
	if n == 0 || len(md.Highs) != n || len(md.Lows) != n {
		return sig
	}
	i := n - 1
	low, high, closeP := md.Lows[i], md.Highs[i], md.Closes[i]
	if stop := s.stop(md, pos); stop > 0 && low <= stop {
		sig.Kind, sig.Reason, sig.StopLoss = model.SignalSell, "SL", stop
		sig.ExitReason = fmt.Sprintf("SL: low %.4f ≤ стоп %.4f (вход %.4f)", low, stop, pos.PurchasePrice)
		return sig
	}
	if pos.TakeProfit > 0 && high >= pos.TakeProfit {
		sig.Kind, sig.Reason, sig.TakeProfit = model.SignalSell, "TP", pos.TakeProfit
		sig.ExitReason = fmt.Sprintf("TP: high %.4f ≥ PDH %.4f (вход %.4f)", high, pos.TakeProfit, pos.PurchasePrice)
		return sig
	}
	if s.p.RSIPeriod > 0 && s.p.RSIExit > 0 && n > s.p.RSIPeriod {
		if rsi := indicators.RSISeries(md.Closes, s.p.RSIPeriod)[i]; rsi >= s.p.RSIExit {
			sig.Kind, sig.Reason = model.SignalSell, "RSI"
			sig.ExitReason = fmt.Sprintf("RSI: RSI(%d) %.1f ≥ %.0f, выход по %.4f (вход %.4f)", s.p.RSIPeriod, rsi, s.p.RSIExit, closeP, pos.PurchasePrice)
			return sig
		}
	}
	if hist := histogram(md.Closes, s.p); hist != nil && len(md.Times) == n {
		if macdExit(hist, entryIndex(md.Times, pos.EntryTime), i) {
			sig.Kind, sig.Reason = model.SignalSell, "MACD"
			sig.ExitReason = fmt.Sprintf("MACD: гистограмма %.4f красная после зелёной в сделке, выход по %.4f (вход %.4f)", hist[i], closeP, pos.PurchasePrice)
			return sig
		}
	}
	if why := s.eodReason(md, pos); why != "" {
		sig.Kind, sig.Reason = model.SignalSell, "EOD"
		sig.ExitReason = fmt.Sprintf("EOD: %s, выход по %.4f (вход %.4f)", why, closeP, pos.PurchasePrice)
	}
	return sig
}

// stop is StopDailyATR daily ATRs below the fill, with the ATR frozen at entry (today's when the
// position carries none). 0 — no stop.
func (s *Strategy) stop(md strategy.MarketData, pos *strategy.Position) float64 {
	if s.p.StopDailyATR <= 0 {
		return 0
	}
	atr := pos.EntryATR
	if atr <= 0 {
		atr = s.dailyATR(md)
	}
	if atr <= 0 {
		return 0
	}
	return max(pos.PurchasePrice-s.p.StopDailyATR*atr, 0)
}

// histogram is MACD line minus signal line over closes; nil for invalid periods or short history.
func histogram(closes []float64, p Params) []float64 {
	m, sg := indicators.MACD(closes, p.MACDFast, p.MACDSlow, p.MACDSignal)
	if m == nil {
		return nil
	}
	h := make([]float64, len(m))
	for k := range m {
		h[k] = m[k] - sg[k]
	}
	return h
}

// entryIndex is the index of the bar that opened at entry, or -1 when entry is unset or not in
// the window.
func entryIndex(times []time.Time, entry time.Time) int {
	if entry.IsZero() {
		return -1
	}
	for j := len(times) - 1; j >= 0; j-- {
		if times[j].Equal(entry) {
			return j
		}
		if times[j].Before(entry) {
			return -1
		}
	}
	return -1
}

// macdExit: the histogram is red on bar i and was green (≥ 0) on some bar from the entry bar e
// up to i-1.
func macdExit(hist []float64, e, i int) bool {
	if e < 0 || e >= i || hist[i] >= 0 {
		return false
	}
	for k := e; k < i; k++ {
		if hist[k] >= 0 {
			return true
		}
	}
	return false
}

// eodReason explains why the position must close on the current bar, or returns "" when it may
// stay. It fails closed: no bar times, the first bar of a new MSK date, a bar of a later date
// than the entry, or a bar starting at/after Exit MSK.
func (s *Strategy) eodReason(md strategy.MarketData, pos *strategy.Position) string {
	n := len(md.Closes)
	if n < 2 || len(md.Times) != n {
		return "нет времени свечей — выход по безопасности"
	}
	now := md.Times[n-1]
	if dayKey(now) != dayKey(md.Times[n-2]) {
		return "первая свеча новой даты — позиция не переносится через ночь"
	}
	if !pos.EntryTime.IsZero() && dayKey(now) != dayKey(pos.EntryTime) {
		return "свеча следующего дня — позиция не переносится через ночь"
	}
	if minuteOf(now) >= s.p.ExitHour*60+s.p.ExitMinute {
		return fmt.Sprintf("свеча %s не раньше %02d:%02d MSK", now.In(mskLoc).Format("15:04"), s.p.ExitHour, s.p.ExitMinute)
	}
	return ""
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/service/trading_strategy/pdl_reclaim/strategy/core/`
Expected: PASS (Task 1 tests included)

- [ ] **Step 5: Commit**

```bash
git add internal/service/trading_strategy/pdl_reclaim
git commit -m "feat(pdl_reclaim): exits — stop, PDH, RSI, MACD, end of day

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

---

### Task 3: Engine fills and Explain

**Files:**
- Create: `internal/service/trading_strategy/pdl_reclaim/strategy/core/explain.go`
- Test: `internal/service/trading_strategy/pdl_reclaim/strategy/core/engine_test.go`, `internal/service/trading_strategy/pdl_reclaim/strategy/core/explain_test.go`

**Interfaces:**
- Consumes (Tasks 1–2): `levels`, `dailyATR`, `inEntryWindow`, `isA`, `isB`, `stop`, `histogram`, `entryIndex`, `macdExit`, `eodReason`, test helpers `flat`, `dailyBars`, `at`, `bar`, `pre`, `barA`, `newS`, `fixture`, `quiet`, `pos0`, `fillBar`.
- Produces: `func (s *Strategy) Explain(md strategy.MarketData) string` — the engine's `Trace` (`-explain`) finds it through its `Explain(md) string` interface.

- [ ] **Step 1: Write the failing tests**

`internal/service/trading_strategy/pdl_reclaim/strategy/core/engine_test.go`:

```go
// engine_test.go runs pdl_reclaim through the real backtest engine to pin the fills the strategy
// relies on: entry at the next bar's open, TP at max(target, open), SL at min(stop from the fill,
// open), EOD at the 18:30 bar's close, and a second trade on the same day after an exit.
package core

import (
	"math"
	"testing"
	"time"

	bt "tinvest/internal/domain/backtest"
)

func toCandles(bars []bar) []bt.Candle {
	out := make([]bt.Candle, len(bars))
	for i, b := range bars {
		out[i] = bt.Candle{Time: at(b.t), Open: b.o, High: b.h, Low: b.l, Close: b.c}
	}
	return out
}

func engineDaily() []bt.Candle {
	h, l, c, times := dailyBars(at("2026-09-14 00:00"), 40)
	out := make([]bt.Candle, len(c))
	for i := range c {
		out[i] = bt.Candle{Time: times[i], Open: c[i], High: h[i], Low: l[i], Close: c[i]}
	}
	return out
}

// history is Thursday and Friday 07:00–18:30 flat at 100: 278 bars, more than the 244-bar lookback.
func history() []bar {
	return append(flat("2026-09-10 07:00", "2026-09-10 18:30", 100), flat("2026-09-11 07:00", "2026-09-11 18:30", 100)...)
}

func run(t *testing.T, monday []bar) []bt.Trade {
	t.Helper()
	bars := append(history(), monday...)
	res := bt.Run(NewWithParams("TEST", 5*time.Minute, quiet()), toCandles(bars), engineDaily(), nil,
		bt.Config{InitialCash: 1e6, Fraction: 1, Commission: 0, Lot: 1, EntryAtNextOpen: true})
	return res.Trades
}

// mondayToA is Monday flat at 100 until 09:55, then the entry-A bar at 10:00.
func mondayToA() []bar { return append(flat("2026-09-14 07:00", "2026-09-14 09:55", 100), barA) }

func TestEngineEntryAAndTarget(t *testing.T) {
	mon := append(mondayToA(),
		bar{"2026-09-14 10:05", 99.35, 99.6, 99.3, 99.5},
		bar{"2026-09-14 10:10", 99.5, 101.2, 99.5, 101.0})
	mon = append(mon, flat("2026-09-14 10:15", "2026-09-14 10:30", 101)...)
	trades := run(t, mon)
	if len(trades) != 1 {
		t.Fatalf("trades = %d, want 1: %+v", len(trades), trades)
	}
	tr := trades[0]
	if math.Abs(tr.EntryPrice-99.35) > eps || !tr.EntryTime.Equal(at("2026-09-14 10:05")) {
		t.Fatalf("entry %v @ %v, want 99.35 @ 10:05", tr.EntryPrice, tr.EntryTime)
	}
	if tr.Reason != "TP" || math.Abs(tr.ExitPrice-101) > eps {
		t.Fatalf("exit %q @ %v, want TP @ 101", tr.Reason, tr.ExitPrice)
	}
}

// slDay: entry A fills at 99.35 (stop 98.35), the 10:10 bar gaps to 98.0 — SL at the open.
func slDay() []bar {
	mon := append(mondayToA(),
		bar{"2026-09-14 10:05", 99.35, 99.4, 99.3, 99.35},
		bar{"2026-09-14 10:10", 98.0, 98.1, 97.9, 98.0})
	return append(mon, flat("2026-09-14 10:15", "2026-09-14 10:55", 98)...)
}

func TestEngineStopGapFillsAtOpen(t *testing.T) {
	trades := run(t, slDay())
	if len(trades) != 1 {
		t.Fatalf("trades = %d, want 1: %+v", len(trades), trades)
	}
	if tr := trades[0]; tr.Reason != "SL" || math.Abs(tr.ExitPrice-98.0) > eps {
		t.Fatalf("exit %q @ %v, want SL @ 98.0", tr.Reason, tr.ExitPrice)
	}
}

func TestEngineReentersSameDayAndExitsAtEOD(t *testing.T) {
	mon := append(slDay(),
		bar{"2026-09-14 11:00", 98.9, 98.95, 98.5, 98.6},  // red under PDL
		bar{"2026-09-14 11:05", 98.55, 98.95, 98.4, 98.92}, // engulfing — entry B
		bar{"2026-09-14 11:10", 98.9, 98.95, 98.85, 98.9})
	mon = append(mon, flat("2026-09-14 11:15", "2026-09-14 18:30", 98.9)...)
	trades := run(t, mon)
	if len(trades) != 2 {
		t.Fatalf("trades = %d, want 2: %+v", len(trades), trades)
	}
	tr := trades[1]
	if !tr.EntryTime.Equal(at("2026-09-14 11:10")) || math.Abs(tr.EntryPrice-98.9) > eps {
		t.Fatalf("second entry %v @ %v, want 98.9 @ 11:10", tr.EntryPrice, tr.EntryTime)
	}
	if tr.Reason != "EOD" || !tr.ExitTime.Equal(at("2026-09-14 18:30")) {
		t.Fatalf("second exit %q @ %v, want EOD @ 18:30", tr.Reason, tr.ExitTime)
	}
}
```

`internal/service/trading_strategy/pdl_reclaim/strategy/core/explain_test.go`:

```go
package core

import (
	"strings"
	"testing"
)

func TestExplainEntryA(t *testing.T) {
	out := newS(DefaultParams()).Explain(fixture([]bar{pre, barA}))
	for _, want := range []string{"✓ окно", "PDL 99.0000", "✓ A", "ИТОГ: ПОКУПКА"} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q in:\n%s", want, out)
		}
	}
}

func TestExplainOutsideWindow(t *testing.T) {
	b := barA
	b.t = "2026-09-14 06:55"
	q := pre
	q.t = "2026-09-14 06:50"
	out := newS(DefaultParams()).Explain(fixture([]bar{q, b}))
	if !strings.Contains(out, "✗ окно") || !strings.Contains(out, "ИТОГ: входа нет") {
		t.Fatalf("got:\n%s", out)
	}
}

func TestExplainHeldStop(t *testing.T) {
	md := fixture([]bar{fillBar, {"2026-09-14 10:10", 99.4, 99.5, 98.4, 98.6}})
	pos := pos0()
	md.Position = &pos
	out := newS(quiet()).Explain(md)
	for _, want := range []string{"--- выходы ---", "стоп", "ИТОГ: ПРОДАЖА SL"} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q in:\n%s", want, out)
		}
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/service/trading_strategy/pdl_reclaim/strategy/core/`
Expected: FAIL to compile — `newS(...).Explain undefined`. (The engine tests alone would already pass; they pin behaviour of Tasks 1–2 through the engine.)

- [ ] **Step 3: Write the implementation**

`internal/service/trading_strategy/pdl_reclaim/strategy/core/explain.go`:

```go
package core

import (
	"fmt"
	"strings"

	"tinvest/internal/service/trading_strategy/scalping/model"
	"tinvest/internal/service/trading_strategy/scalping/strategy"
	"tinvest/pkg/indicators"
)

// Explain returns a gate-by-gate verdict for one bar, consumed by the engine's Trace (-explain).
// It recomputes the values enter() and manage() use and ends with the signal Decide actually
// returns on this bar, so the verdict can never disagree with a real run.
func (s *Strategy) Explain(md strategy.MarketData) string {
	var sb strings.Builder
	p := s.p
	n := len(md.Closes)
	if n < 2 || len(md.Times) != n || len(md.Opens) != n || len(md.Highs) != n || len(md.Lows) != n {
		return "недостаточно свечей или нет времени/open свечей\n"
	}
	i := n - 1
	t := md.Times[i].In(mskLoc)
	fmt.Fprintf(&sb, "%s окно: сигнал %s %s, покупка %s (сигнал с %02d:%02d, покупка до %02d:%02d, Пн–Пт)\n",
		mark(s.inEntryWindow(md.Times[i])), t.Weekday(), t.Format("15:04"), md.Times[i].Add(s.step).In(mskLoc).Format("15:04"),
		p.EntryFromHour, p.EntryFromMinute, p.EntryToHour, p.EntryToMinute)

	o, h, l, c := md.Opens[i], md.Highs[i], md.Lows[i], md.Closes[i]
	fmt.Fprintf(&sb, "свеча: open %.4f high %.4f low %.4f close %.4f\n", o, h, l, c)
	if pdl, pdh, ok := s.levels(md); ok {
		fmt.Fprintf(&sb, "уровни вчера: PDL %.4f, PDH %.4f\n", pdl, pdh)
		fmt.Fprintf(&sb, "%s A (включён %d): зелёная %s, low < PDL %s, close > PDL %s\n",
			mark(isA(md, i, pdl)), p.EntryA, mark(c > o), mark(l < pdl), mark(c > pdl))
		po, pc, pl := md.Opens[i-1], md.Closes[i-1], md.Lows[i-1]
		fmt.Fprintf(&sb, "%s B (включён %d): одна дата %s, пред. красная %s, её close %.4f < PDL %s, текущая зелёная %s, close > open пред. %.4f %s, low < low пред. %.4f %s\n",
			mark(isB(md, i, pdl)), p.EntryB, mark(dayKey(md.Times[i]) == dayKey(md.Times[i-1])), mark(pc < po), pc, mark(pc < pdl),
			mark(c > o), po, mark(c > po), pl, mark(l < pl))
		if p.TakeProfitPDH > 0 {
			fmt.Fprintf(&sb, "%s цель: close %.4f < PDH %.4f\n", mark(c < pdh), c, pdh)
		}
	} else {
		sb.WriteString("✗ уровни: нет будней дневки\n")
	}
	if atr := s.dailyATR(md); atr > 0 {
		fmt.Fprintf(&sb, "✓ дневной ATR(%d): %.4f, стоп от close %.4f\n", p.DailyATRPeriod, atr, c-p.StopDailyATR*atr)
	} else {
		fmt.Fprintf(&sb, "✗ дневной ATR(%d): не посчитан\n", p.DailyATRPeriod)
	}

	if md.Position != nil {
		s.explainExits(&sb, md)
	}

	sig := s.Decide(md)
	switch sig.Kind {
	case model.SignalBuy:
		fmt.Fprintf(&sb, "ИТОГ: ПОКУПКА по open следующей свечи — %s\n", sig.EntryReason)
	case model.SignalSell:
		fmt.Fprintf(&sb, "ИТОГ: ПРОДАЖА %s — %s\n", sig.Reason, sig.ExitReason)
	default:
		if md.Position != nil {
			sb.WriteString("ИТОГ: держим (в позиции новые входы не ищутся)\n")
		} else {
			sb.WriteString("ИТОГ: входа нет\n")
		}
	}
	return sb.String()
}

// explainExits reports every exit gate of a held position in priority order.
func (s *Strategy) explainExits(sb *strings.Builder, md strategy.MarketData) {
	p := s.p
	pos := md.Position
	i := len(md.Closes) - 1
	sb.WriteString("--- выходы ---\n")
	stop := s.stop(md, pos)
	fmt.Fprintf(sb, "%s стоп: вход %.4f, стоп %.4f, low %.4f\n", mark(stop > 0 && md.Lows[i] <= stop), pos.PurchasePrice, stop, md.Lows[i])
	if pos.TakeProfit > 0 {
		fmt.Fprintf(sb, "%s цель PDH %.4f, high %.4f\n", mark(md.Highs[i] >= pos.TakeProfit), pos.TakeProfit, md.Highs[i])
	} else {
		sb.WriteString("цель: нет\n")
	}
	if p.RSIPeriod > 0 && p.RSIExit > 0 && len(md.Closes) > p.RSIPeriod {
		rsi := indicators.RSISeries(md.Closes, p.RSIPeriod)[i]
		fmt.Fprintf(sb, "%s RSI(%d) %.1f ≥ %.0f\n", mark(rsi >= p.RSIExit), p.RSIPeriod, rsi, p.RSIExit)
	} else {
		sb.WriteString("RSI-выход: выключен или не прогрет\n")
	}
	if hist := histogram(md.Closes, p); hist != nil {
		e := entryIndex(md.Times, pos.EntryTime)
		fmt.Fprintf(sb, "%s MACD: гистограмма %.4f, свеча входа в окне %s\n", mark(macdExit(hist, e, i)), hist[i], mark(e >= 0))
	} else {
		sb.WriteString("MACD-выход: выключен\n")
	}
	why := s.eodReason(md, pos)
	fmt.Fprintf(sb, "%s конец дня: %s\n", mark(why != ""), orNone(why))
}

// orNone is s, or "нет" when empty.
func orNone(s string) string {
	if s == "" {
		return "нет"
	}
	return s
}

// mark is ✓ for a passed gate and ✗ for a failed one.
func mark(ok bool) string {
	if ok {
		return "✓"
	}
	return "✗"
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/service/trading_strategy/pdl_reclaim/...`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/service/trading_strategy/pdl_reclaim
git commit -m "feat(pdl_reclaim): engine fill tests and -explain

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

---

### Task 4: Registry, cmd/backtest wiring, starter grid

**Files:**
- Create: `internal/service/backtest/pdl_reclaim_registry.go`
- Test: `internal/service/backtest/pdl_reclaim_registry_test.go`
- Modify: `cmd/backtest/main.go` (lines ~41, ~57, ~175–184, ~233–235)
- Create: `data/params/pdl_reclaim/grid_example.json`

**Interfaces:**
- Consumes (Task 1): `core.Params`, `core.DefaultParams`, `core.NewWithParams(ticker string, step time.Duration, p Params)`.
- Produces: `var pdlReclaimRegistry map[string]func() core.Params`, `func PDLReclaimLookupOrGeneric(ticker string, interval enum.Interval) (Binding, error)` — Task 5's env test reads `pdlReclaimRegistry`.

- [ ] **Step 1: Write the failing test**

`internal/service/backtest/pdl_reclaim_registry_test.go`:

```go
package backtest

import (
	"testing"

	"tinvest/internal/enum"
	"tinvest/internal/service/trading_strategy/pdl_reclaim/strategy/core"
)

func TestPDLReclaimGenericTickerUsesCoreDefaults(t *testing.T) {
	b, err := PDLReclaimLookupOrGeneric("ZZZZ", enum.Minutes5)
	if err != nil {
		t.Fatalf("lookup: %v", err)
	}
	got, ok := b.DefaultParams().(core.Params)
	if !ok || got != core.DefaultParams() {
		t.Fatalf("defaults = %#v, want core.DefaultParams()", b.DefaultParams())
	}
	if s := b.Build(got); s.Ticker() != "ZZZZ" {
		t.Fatalf("ticker = %q, want ZZZZ", s.Ticker())
	}
}

func TestPDLReclaimRunsOnMinutes5Only(t *testing.T) {
	for _, iv := range []enum.Interval{enum.Minutes15, enum.Minutes30, enum.Hour1, enum.Hour4, enum.Day1, enum.Week1} {
		if _, err := PDLReclaimLookupOrGeneric("GAZP", iv); err == nil {
			t.Fatalf("%s: want error", iv)
		}
	}
}

func TestPDLReclaimParseParamsLayersOverDefaults(t *testing.T) {
	b, err := PDLReclaimLookupOrGeneric("ZZZZ", enum.Minutes5)
	if err != nil {
		t.Fatalf("lookup: %v", err)
	}
	parsed, err := b.ParseParams([]byte(`{"StopDailyATR": 0.7, "RSIExit": 75}`))
	if err != nil {
		t.Fatalf("ParseParams: %v", err)
	}
	want := core.DefaultParams()
	want.StopDailyATR, want.RSIExit = 0.7, 75
	if got := parsed.(core.Params); got != want {
		t.Fatalf("got %+v, want %+v", got, want)
	}
}

// A registered ticker resolves to its own literal. The registry is empty until the first
// calibration, so the test injects one.
func TestPDLReclaimTickerLiteral(t *testing.T) {
	lit := core.DefaultParams()
	lit.RSIExit = 77
	pdlReclaimRegistry["TEST"] = func() core.Params { return lit }
	t.Cleanup(func() { delete(pdlReclaimRegistry, "TEST") })
	b, err := PDLReclaimLookupOrGeneric("TEST", enum.Minutes5)
	if err != nil {
		t.Fatalf("lookup: %v", err)
	}
	if got := b.DefaultParams().(core.Params); got != lit {
		t.Fatalf("got %+v, want %+v", got, lit)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/service/backtest/ -run PDLReclaim`
Expected: FAIL to compile — `undefined: PDLReclaimLookupOrGeneric`.

- [ ] **Step 3: Write the implementation**

`internal/service/backtest/pdl_reclaim_registry.go`:

```go
package backtest

import (
	"encoding/json"
	"fmt"

	"tinvest/internal/enum"
	"tinvest/internal/service/trading_strategy/pdl_reclaim/strategy/core"
	"tinvest/internal/service/trading_strategy/scalping/strategy"
)

// pdlReclaimBindingFor builds a pdl_reclaim binding for one ticker whose baseline params come from
// defaults: a ticker package literal, or core.DefaultParams when the ticker has none.
func pdlReclaimBindingFor(ticker string, interval enum.Interval, defaults func() core.Params) Binding {
	step := interval.ToTimeDuration()
	return Binding{
		DefaultParams: func() any { return defaults() },
		Build: func(params any) strategy.Strategy {
			return core.NewWithParams(ticker, step, params.(core.Params))
		},
		ParseParams: func(raw []byte) (any, error) {
			p := defaults() // start from defaults so partial JSON overrides
			if err := json.Unmarshal(raw, &p); err != nil {
				return nil, fmt.Errorf("backtest: parse pdl_reclaim params: %w", err)
			}
			return p, nil
		},
	}
}

// pdlReclaimRegistry maps tickers with their own package to their Minutes5 literal. A ticker
// comes here with its calibration; GAZP, SBER and NVTK are the first in line.
var pdlReclaimRegistry = map[string]func() core.Params{}

// PDLReclaimLookupOrGeneric returns the binding for a ticker: the ticker package's literal, or
// core.DefaultParams when the ticker has no package. Partial JSON overrides layer over whichever
// defaults apply. pdl_reclaim runs on Minutes5 only; other timeframes are rejected.
func PDLReclaimLookupOrGeneric(ticker string, interval enum.Interval) (Binding, error) {
	if interval != enum.Minutes5 {
		return Binding{}, fmt.Errorf("pdl_reclaim runs on Minutes5 only, not %s", interval)
	}
	defaults, ok := pdlReclaimRegistry[ticker]
	if !ok {
		defaults = core.DefaultParams
	}
	return pdlReclaimBindingFor(ticker, interval, defaults), nil
}
```

In `cmd/backtest/main.go`:

1. Flag help (line ~41): replace `|gap_fade|drop_bounce")` with `|gap_fade|drop_bounce|pdl_reclaim")`.
2. `-risk-pct` help (line ~57): replace `drop_bounce StopDailyATR>0;` with `drop_bounce/pdl_reclaim StopDailyATR>0;`.
3. After the `case "drop_bounce": … binding = b` block add:

```go
	case "pdl_reclaim":
		b, err := svc.PDLReclaimLookupOrGeneric(ticker, interval)
		if err != nil {
			return err
		}
		binding = b
```

4. The `default:` error string: replace `|gap_fade|drop_bounce)"` with `|gap_fade|drop_bounce|pdl_reclaim)"`.
5. The config (line ~233):

```go
		// gap_fade triggers on the opening-auction print itself, drop_bounce and pdl_reclaim on a
		// bar's close; either way the first price an order can get is the next bar's open.
		EntryAtNextOpen: strategyName == "gap_fade" || strategyName == "drop_bounce" || strategyName == "pdl_reclaim"}
```

`data/params/pdl_reclaim/grid_example.json`:

```json
{
  "_comment": "pdl_reclaim (только 5m), одна фаза поверх core.DefaultParams(): вход A — зелёная свеча с low ниже вчерашнего лоя и close выше него, вход B — бычье поглощение под вчерашним лоем; покупка по open следующей 5m свечи 07:00–18:00 МСК, будни; стоп StopDailyATR дневных ATR от цены входа; выход — PDH, RSI(RSIPeriod) ≥ RSIExit, первая красная гистограмма MACD(12,26,9) после зелёной в сделке, конец дня 18:30. Ветки A/B проверяются отдельными прогонами с EntryA/EntryB = 0.",
  "phases": [
    {"name": "sweep", "grid": {"StopDailyATR": [0.3, 0.5, 0.7], "RSIExit": [65, 70, 75, 80], "RSIPeriod": [4, 6, 8]}}
  ]
}
```

- [ ] **Step 4: Run tests and build**

Run: `go test ./internal/service/backtest/ -run PDLReclaim && go build ./internal/... ./pkg/... ./cmd/...`
Expected: PASS, build OK.

Run: `go run ./cmd/backtest -strategy pdl_reclaim -interval Minutes15 -ticker GAZP -months 1`
Expected: exits with `pdl_reclaim runs on Minutes5 only, not …` (before any network call).

- [ ] **Step 5: Commit**

```bash
git add internal/service/backtest/pdl_reclaim_registry.go internal/service/backtest/pdl_reclaim_registry_test.go cmd/backtest/main.go data/params/pdl_reclaim/grid_example.json
git commit -m "feat(pdl_reclaim): registry, cmd/backtest -strategy pdl_reclaim, starter grid

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

---

### Task 5: Live universe config

**Files:**
- Create: `internal/config/pdl_reclaim.go`, `internal/config/pdl_reclaim_test.go`
- Modify: `internal/config/config.go` (add field after `DropBounce`), `internal/app/init_config.go` (add constructor after `DropBounce`)
- Modify: `env/prod.env`, `env/prod.env.example`, `env/local.env.example` (append the variable)
- Create: `internal/service/backtest/pdl_reclaim_live_env_test.go`
- Modify: `internal/service/backtest/drop_bounce_live_env_test.go` (rename the env reader so both tests share it)

**Interfaces:**
- Consumes (Task 4): `pdlReclaimRegistry`.
- Produces: `type PDLReclaimConfig struct { TickersMinutes5 []string }`, `func NewPDLReclaimConfig() *PDLReclaimConfig`, `func (c *PDLReclaimConfig) Enabled() bool`, `func prodEnvTickerLists(t *testing.T) map[string][]string` (renamed from `dropBounceLiveTickers`).

- [ ] **Step 1: Write the failing tests**

`internal/config/pdl_reclaim_test.go`:

```go
package config

import (
	"context"
	"slices"
	"testing"

	"github.com/heetch/confita"
	"github.com/heetch/confita/backend/env"
)

// Без переменной pdl_reclaim конфиг грузится, вселенная пуста.
func TestPDLReclaimConfigLoadsWithoutItsEnvVar(t *testing.T) {
	t.Setenv("PDL_RECLAIM_TICKERS_M5", "")

	cfg := NewPDLReclaimConfig()
	if err := confita.NewLoader(env.NewBackend()).Load(context.Background(), cfg); err != nil {
		t.Fatalf("Load без PDL_RECLAIM_TICKERS_M5 = %v, want nil", err)
	}
	if cfg.Enabled() {
		t.Fatalf("без переменной cfg = %+v, want выключено", cfg)
	}
}

func TestPDLReclaimConfigLoadsMinutes5(t *testing.T) {
	t.Setenv("PDL_RECLAIM_TICKERS_M5", "GAZP,SBER")

	cfg := NewPDLReclaimConfig()
	if err := confita.NewLoader(env.NewBackend()).Load(context.Background(), cfg); err != nil {
		t.Fatalf("Load = %v", err)
	}
	if !slices.Equal(cfg.TickersMinutes5, []string{"GAZP", "SBER"}) {
		t.Fatalf("cfg = %+v, want [GAZP SBER]", cfg)
	}
	if !cfg.Enabled() {
		t.Fatal("заданная вселенная не включила стратегию")
	}
}

func TestPDLReclaimEnabledIgnoresBlankTickers(t *testing.T) {
	c := NewPDLReclaimConfig()
	c.TickersMinutes5 = []string{"", " "}
	if c.Enabled() {
		t.Fatal("вселенная из пустых строк включила стратегию")
	}
}
```

First rename the shared reader in `internal/service/backtest/drop_bounce_live_env_test.go`: every `dropBounceLiveTickers` → `prodEnvTickerLists`, and its doc comment to `// prodEnvTickerLists читает списки тикеров из env/prod.env: ключ — переменная.` Check nothing else uses the old name:

Run: `grep -rn dropBounceLiveTickers internal/ cmd/`
Expected: no output after the rename.

`internal/service/backtest/pdl_reclaim_live_env_test.go`:

```go
package backtest

import "testing"

// Вселенная pdl_reclaim есть в env/prod.env, и каждый её тикер имеет пакет с литералом: тикер
// без пакета торговал бы на дефолтах ядра, которые никто не калибровал.
func TestPDLReclaimLiveTickersAreRegistered(t *testing.T) {
	const key = "PDL_RECLAIM_TICKERS_M5"
	tickers, ok := prodEnvTickerLists(t)[key]
	if !ok {
		t.Fatalf("%s не найден в env/prod.env", key)
	}
	for _, tk := range tickers {
		if _, ok := pdlReclaimRegistry[tk]; !ok {
			t.Errorf("%s есть в %s env/prod.env, но не зарегистрирован в pdlReclaimRegistry", tk, key)
		}
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/config/ -run PDLReclaim; go test ./internal/service/backtest/ -run 'PDLReclaimLive|DropBounceLive'`
Expected: config FAIL to compile (`undefined: NewPDLReclaimConfig`); backtest `TestPDLReclaimLiveTickersAreRegistered` FAIL with `PDL_RECLAIM_TICKERS_M5 не найден в env/prod.env`; drop_bounce live tests PASS.

- [ ] **Step 3: Write the implementation**

`internal/config/pdl_reclaim.go`:

```go
package config

import "strings"

// PDLReclaimConfig — вселенная pdl_reclaim для будущего живого раннера. Стратегия работает только
// на 5m; тикер попадает сюда, только если прошёл калибровку. Живого раннера пока нет — конфиг
// только хранит список. Переменная не required: ошибка загрузки конфига оборвала бы InitApp целиком.
type PDLReclaimConfig struct {
	TickersMinutes5 []string `config:"PDL_RECLAIM_TICKERS_M5,backend=env"`
}

// NewPDLReclaimConfig: вселенная пуста — стратегия ничего не торгует, пока её не заполнят.
func NewPDLReclaimConfig() *PDLReclaimConfig {
	return &PDLReclaimConfig{}
}

// Enabled — задан ли хоть один тикер. PDL_RECLAIM_TICKERS_M5= в env приходит как [""].
func (c *PDLReclaimConfig) Enabled() bool {
	for _, t := range c.TickersMinutes5 {
		if strings.TrimSpace(t) != "" {
			return true
		}
	}
	return false
}
```

`internal/config/config.go` — after `DropBounce     *DropBounceConfig` add:

```go
	PDLReclaim     *PDLReclaimConfig
```

`internal/app/init_config.go` — after `DropBounce:     config.NewDropBounceConfig(),` add:

```go
		PDLReclaim:     config.NewPDLReclaimConfig(),
```

Append to the end of each of `env/prod.env`, `env/prod.env.example`, `env/local.env.example`:

```
# pdl_reclaim: вселенная 5m для будущего live-раннера (тикер — только с пакетом в pdlReclaimRegistry)
PDL_RECLAIM_TICKERS_M5=
```

(Read each file's tail first; keep exactly one blank line before the comment.)

- [ ] **Step 4: Run tests and build**

Run: `gofmt -l internal/ cmd/ && go test ./internal/config/ ./internal/service/backtest/ && go build ./internal/... ./pkg/... ./cmd/...`
Expected: no gofmt output, PASS, build OK.

- [ ] **Step 5: Commit**

```bash
git add internal/config/pdl_reclaim.go internal/config/pdl_reclaim_test.go internal/config/config.go internal/app/init_config.go env/prod.env env/prod.env.example env/local.env.example internal/service/backtest/pdl_reclaim_live_env_test.go internal/service/backtest/drop_bounce_live_env_test.go
git commit -m "feat(pdl_reclaim): live universe PDL_RECLAIM_TICKERS_M5

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

---

### Task 6: Docs and final gate

**Files:**
- Create: `docs/pdl_reclaim/strategy.md`
- Modify: `CLAUDE.md` (the `internal/service/trading_strategy/` bullet in Layout)

**Interfaces:**
- Consumes: everything above (names and defaults must match the code).

- [ ] **Step 1: Write `docs/pdl_reclaim/strategy.md`**

```markdown
# pdl_reclaim — возврат над вчерашним лоем на 5m

Бэктест-стратегия, только лонг, внутридневная. Прогон **только на 5m** (`-interval Minutes5`, другие cmd/backtest отклоняет). Уровни — лой (PDL) и хай (PDH) вчерашнего буднего дня: PDL — поддержка, у которой ищется вход, PDH — цель. Позиция через ночь не переносится. Код: `internal/service/trading_strategy/pdl_reclaim/strategy/core`, привязка к бэктесту — `internal/service/backtest/pdl_reclaim_registry.go`. Спека: `docs/superpowers/specs/2026-10-15-pdl-reclaim-design.md`.

## Уровни

- **PDL / PDH** — low и high последней завершённой будней дневки (выходные дневки отброшены; в понедельник это пятница). Дневка биржевая — со всеми сессиями дня.
- **Дневной ATR** — Уайлдер `DailyATRPeriod` по завершённым будним дневкам; уходит в сигнал и замораживается на позицию.
- Нет дневок для уровней или ATR — входов нет.

## Вход

На закрытии 5m свечи без открытой позиции, общие условия:

- будни МСК, свеча начинается не раньше `EntryFromHour:EntryFromMinute` (07:00);
- свеча покупки (следующая) начинается раньше `EntryToHour:EntryToMinute` (18:00) в ту же дату — последний сигнал на свече 17:50;
- при включённой цели close свечи ниже PDH;
- стоп `close − StopDailyATR · ATR` выше нуля.

**Вход A — ложный пробой** (`EntryA = 1`): свеча зелёная, low < PDL, close > PDL.

**Вход B — поглощение под PDL** (`EntryB = 1`): предыдущая свеча той же даты красная и закрылась ниже PDL; текущая зелёная, её close выше open красной, её low ниже low красной.

Обе ветки сразу — причина `A`. Покупка по open следующей 5m свечи (`Config.EntryAtNextOpen`, `FillBy` = время свечи + 5m). Сигнал несёт стоп (для `-risk-pct`), цель PDH и ATR. После выхода перевход в тот же день разрешён.

## Выход

По приоритету на каждой 5m свече:

1. **SL** — `low ≤ вход − StopDailyATR · ATR` (стоп от цены исполнения, ATR замороженный). Исполнение `min(стоп, open)`.
2. **TP** — `high ≥ PDH` (заморожен на входе). Исполнение `max(PDH, open)`. Стоп и цель в одной свече — стоп.
3. **RSI** — RSI(`RSIPeriod`) по 5m close ≥ `RSIExit`. Продажа по close.
4. **MACD** — гистограмма MACD(`MACDFast`, `MACDSlow`, `MACDSignal`) по 5m close была зелёной (`≥ 0`) хотя бы на одной свече от свечи входа до предыдущей, а на текущей красная. Продажа по close. Зелёная уже на свече входа — считается.
5. **EOD** — первая свеча, начавшаяся не раньше `ExitHour:ExitMinute` (18:30), продаётся по close; первая свеча новой даты или свеча даты позже входа — тоже.

## Параметры

| поле | по умолчанию | смысл |
|---|---|---|
| `EntryA` | 1 | вход A; 0 — выключен |
| `EntryB` | 1 | вход B; 0 — выключен |
| `StopDailyATR` | 0.5 | стоп ниже входа в дневных ATR; 0 — входов нет |
| `DailyATRPeriod` | 14 | длина дневного ATR Уайлдера по будням |
| `TakeProfitPDH` | 1 | цель PDH; 0 — без цели |
| `RSIPeriod` | 6 | длина RSI выхода на 5m; 0 — RSI-выход выключен |
| `RSIExit` | 70 | уровень RSI-выхода; 0 — выключен |
| `MACDFast` / `MACDSlow` / `MACDSignal` | 12 / 26 / 9 | MACD выхода на 5m; некорректные периоды — выключен |
| `EntryFromHour` / `EntryFromMinute` | 7 / 0 | сигнальная свеча не раньше |
| `EntryToHour` / `EntryToMinute` | 18 / 0 | свеча покупки раньше |
| `ExitHour` / `ExitMinute` | 18 / 30 | выход в конце дня |

`Lookback = max(64, 3 · (MACDSlow + MACDSignal), 4 · RSIPeriod + 2) + 139` 5m свечей (139 — свечи от 07:00 до 18:30): при дефолтах 244.

## Пакеты тикеров

Реестр `pdlReclaimRegistry` держит по одному литералу `Params()` на откалиброванный тикер (`internal/service/trading_strategy/pdl_reclaim/strategy/<ticker>/`); тикер без пакета работает на `core.DefaultParams()`; JSON `-params` / `-calibrate` ложится поверх. Стартовая сетка — `data/params/pdl_reclaim/grid_example.json`.

## Вселенная для live

Живого раннера нет; вселенная под него — `PDL_RECLAIM_TICKERS_M5` (`config.PDLReclaimConfig`). Тест `TestPDLReclaimLiveTickersAreRegistered` требует пакет в `pdlReclaimRegistry` у каждого тикера из `env/prod.env`.

## Запуск

```bash
go run ./cmd/backtest -strategy pdl_reclaim -interval Minutes5 -ticker GAZP -months 36
go run ./cmd/backtest -strategy pdl_reclaim -interval Minutes5 -ticker GAZP -months 36 \
  -calibrate data/params/pdl_reclaim/grid_example.json -metric profit_factor
# почему на свече был (или не был) вход/выход
go run ./cmd/backtest -strategy pdl_reclaim -interval Minutes5 -ticker GAZP -months 36 -explain "2026-09-03 10:00"
```
Для калибровочных прогонов — собранный бинарь на одном ядре.
```

- [ ] **Step 2: Add the CLAUDE.md line**

In `CLAUDE.md`, in the `trading_strategy/` bullet, insert before ` Note: backtest-only strategies \`levels\`…`:

```
`pdl_reclaim` — backtest-only 5m long intraday around yesterday's range: entry A on a green bar that dipped below yesterday's low (PDL) and closed back above it, entry B on a bullish engulfing whose red bar closed below PDL; buys the next 5m open 07:00–18:00 MSK, Mon–Fri; exits — stop `StopDailyATR` (0.5) daily ATRs below the fill, target yesterday's high, RSI(6) ≥ 70, the first red MACD(12,26,9) histogram bar after a green one in the trade, EOD 18:30; re-entry after an exit allowed (`-strategy pdl_reclaim -interval Minutes5`, docs: `docs/pdl_reclaim/strategy.md`); per-ticker literals in `pdl_reclaim/strategy/<ticker>/` after calibration (GAZP, SBER, NVTK first), universe `PDL_RECLAIM_TICKERS_M5` (`config.PDLReclaimConfig`).
```

- [ ] **Step 3: Final gate**

Run: `./bin/mage ci`
Expected: lint clean, `go test -race ./...` PASS, mocks unchanged. (If `./bin/mage` is missing: `go run github.com/magefile/mage@latest tools` is NOT allowed — instead run `go vet ./internal/... ./pkg/... ./cmd/... && go test -race ./internal/... ./pkg/... ./cmd/...` and report that lint was not run.)

- [ ] **Step 4: Commit**

```bash
git add docs/pdl_reclaim/strategy.md CLAUDE.md
git commit -m "docs(pdl_reclaim): strategy mechanics, CLAUDE.md entry

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```
