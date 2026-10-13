# drop_bounce Rework Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace the drop_bounce core with a 15-minute long-only strategy: buy after a red bar of ≥ `MinDropPct`% that closes with RSI(`RSIPeriod`) < `RSIEntry`, exit on a % stop, a fill-bar profit, a MACD histogram green→red flip during the trade, an RSI cross up through `RSIExit` or a stochastic %K cross up through `StochExit`; positions carry overnight.

**Architecture:** One pure, stateless `strategy.Strategy` in `internal/service/trading_strategy/drop_bounce/strategy/core`, driven by the existing backtest engine with `Config.EntryAtNextOpen`. Indicators come from `pkg/indicators` (`RSISeries`, `MACD`, `StochasticSeries`). The registry in `internal/service/backtest` binds it for `-interval Minutes15` only. The old ticker packages (GAZP, T) and the engine flag `SkipFillOutsideLevels` are removed.

**Tech Stack:** Go 1.25, `tinvest/pkg/indicators`, `tinvest/internal/domain/backtest`, `tinvest/internal/service/trading_strategy/scalping/{model,strategy}`.

**Spec:** `docs/superpowers/specs/2026-10-09-drop-bounce-rework-design.md`

## Global Constraints

- Timeframe 15m only; `DropBounceLookupOrGeneric` rejects any other interval with an error.
- Defaults exactly: `MinDropPct 0.6, RSIPeriod 6, RSIEntry 30, RSIExit 70, MACDExit 1, MACDFast 9, MACDSlow 14, MACDSignal 3, StochPeriod 14, StochSmooth 3, StochExit 80, ProfitPct 1, StopPct 1, StartHour 8`.
- All `Params` fields are `int` or `float64` (reflection grid calibration).
- Exit order on a bar: `SL`, `PROFIT`, `MACD`, `RSI`, `STOCH`. No EOD exit.
- Stop and profit are measured from `Position.PurchasePrice` (the fill price), never from the signal bar.
- A zero threshold disables its exit (`StopPct`, `ProfitPct`, `RSIExit`, `StochExit`; `MACDExit = 0`).
- Calendar rules in MSK (`Europe/Moscow`, UTC fallback).
- Code comments in English, `EntryReason`/`ExitReason` strings in Russian (house style); docs in Russian.
- Backtest runs follow the owner's CPU rule: tests only via `go test` on the touched packages; no long backtest runs in this plan.
- `go build ./...` fails on `magefiles`; build with `go build ./internal/... ./pkg/... ./cmd/...`.

## Review Focus

1. RSI warm-up zeros — `RSISeries` returns 0 before index `period`; an entry check on a short window must not read that 0 as "oversold". Pinned by `TestEnterNeedsRSIHistory` (Task 2).
2. MACD green before the entry — must not arm the exit; only green bars at or after `Position.EntryTime` count. Pinned by `TestGreenThenRed` rows and `TestDecideMACDIgnoresGreenBeforeEntry` (Task 3).
3. Overnight gap through the stop — the engine must sell at the gap open, not at the stop price. Pinned by `TestEngineOvernightGapStop` (Task 4).
4. Signal on the last bar of the day — the fill must not move to the next morning. Pinned by `TestEnterRejectsFillOnNextDate` (Task 2).
5. Invalid indicator periods in a grid (`MACDFast ≥ MACDSlow`, `StochPeriod 0`) — must disable that exit, not panic. Pinned by `TestDecideInvalidPeriodsDoNotPanic` (Task 3).

---

### Task 1: Remove `Config.SkipFillOutsideLevels` from the backtest engine

The flag served only the old drop_bounce (fixed target/stop on the signal). The new signal carries no levels.

**Files:**
- Modify: `internal/domain/backtest/types.go` (delete the `SkipFillOutsideLevels` field and its comment)
- Modify: `internal/domain/backtest/engine.go` (`fillPending`, `openOutsideLevels`, both call sites)
- Modify: `internal/domain/backtest/entry_next_open_test.go` (delete `skipOutsideCfg`, `buyAt100`, the three `TestSkipFillOutsideLevels*` tests)
- Modify: `cmd/backtest/main.go` (drop the `SkipFillOutsideLevels:` line and its comment)

**Interfaces:**
- Produces: `fillPending(p *portfolio, pending *model.Signal, c Candle) *model.Signal` (signature as before the flag).

- [ ] **Step 1: Delete the three tests and their fixtures** in `entry_next_open_test.go`: everything from `// skipOutsideCfg turns on Config.SkipFillOutsideLevels` to the end of `TestSkipFillOutsideLevelsKeepsOpenInside`.

- [ ] **Step 2: Restore `fillPending`** in `engine.go`:

```go
// fillPending opens a Buy deferred by Config.EntryAtNextOpen at bar c's open, before the bar is
// managed, and clears it. A pending Buy with a position already open is dropped. Returns the new
// pending state (always nil).
func fillPending(p *portfolio, pending *model.Signal, c Candle) *model.Signal {
	if pending != nil && p.qty == 0 {
		p.open(c.Open, c.Time, pending.Level, pending.TakeProfit, pending.ATR, pending.StopLoss, pending.EntryReason)
	}
	return nil
}
```

Change both call sites (`Run` and `Trace`) to `pending = fillPending(p, pending, candles[i])`. Delete `openOutsideLevels` at the end of the file.

- [ ] **Step 3: Delete the field** `SkipFillOutsideLevels bool` and its comment block from `Config` in `types.go`.

- [ ] **Step 4: Update `cmd/backtest/main.go`** — the `domain.Config{...}` literal keeps:

```go
		// gap_fade triggers on the opening-auction print itself and drop_bounce on a bar's close;
		// either way the first price an order can get is the next bar's open.
		EntryAtNextOpen: strategyName == "gap_fade" || strategyName == "drop_bounce"}
```

and loses the `SkipFillOutsideLevels` line and its two comment lines.

- [ ] **Step 5: Verify**

Run: `go build ./internal/... ./pkg/... ./cmd/... && go test ./internal/domain/backtest/...`
Expected: build OK, `ok tinvest/internal/domain/backtest`. `grep -rn SkipFillOutsideLevels internal cmd` prints nothing.

- [ ] **Step 6: Commit**

```bash
git add internal/domain/backtest cmd/backtest/main.go
git commit -m "refactor(backtest): drop SkipFillOutsideLevels, only the old drop_bounce used it"
```

---

### Task 2: New core — params, entry, stop; remove the old ticker packages; 15m registry

The old core's `Params` fields are referenced by the GAZP/T packages and the registry, so they go in the same task to keep the build green.

**Files:**
- Delete: `internal/service/trading_strategy/drop_bounce/strategy/gazp/`, `.../strategy/tbank/`, `data/params/drop_bounce/gazp/`, `data/params/drop_bounce/t/`, `docs/drop_bounce/2026-10-09-gazp-loss-analysis.md`, `.../strategy/core/engine_test.go` (rewritten in Task 4)
- Rewrite: `internal/service/trading_strategy/drop_bounce/strategy/core/core.go`
- Rewrite: `internal/service/trading_strategy/drop_bounce/strategy/core/core_test.go`
- Rewrite: `internal/service/backtest/drop_bounce_registry.go`, `internal/service/backtest/drop_bounce_registry_test.go`

**Interfaces:**
- Produces (package `core`):
  - `const BarStep = 15 * time.Minute`
  - `type Params struct { MinDropPct float64; RSIPeriod int; RSIEntry, RSIExit float64; MACDExit, MACDFast, MACDSlow, MACDSignal, StochPeriod, StochSmooth int; StochExit, ProfitPct, StopPct float64; StartHour int }`
  - `func DefaultParams() Params`
  - `func NewWithParams(ticker string, p Params) *Strategy`
  - `func (s *Strategy) Ticker() string`, `Lookback() int`, `Decide(md strategy.MarketData) model.Signal`
  - unexported `func (s *Strategy) manage(md strategy.MarketData, sig model.Signal) model.Signal` (Task 3 extends it)
- Produces (package `backtest`): `func DropBounceLookupOrGeneric(ticker string, interval enum.Interval) (Binding, error)` — same signature as today.

- [ ] **Step 1: Delete the old files**

```bash
git rm -r -q internal/service/trading_strategy/drop_bounce/strategy/gazp internal/service/trading_strategy/drop_bounce/strategy/tbank data/params/drop_bounce/gazp data/params/drop_bounce/t docs/drop_bounce/2026-10-09-gazp-loss-analysis.md internal/service/trading_strategy/drop_bounce/strategy/core/engine_test.go
```

- [ ] **Step 2: Write the failing tests** — replace `core/core_test.go` entirely:

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
	t          time.Time
	o, h, l, c float64
}

// path builds consecutive 15m bars from start (MSK): each bar opens at the previous close (the
// first at its own close) and its high/low span the open and the close.
func path(start string, closes []float64) []bar {
	t0 := at(start)
	out := make([]bar, len(closes))
	prev := closes[0]
	for i, c := range closes {
		out[i] = bar{t0.Add(time.Duration(i) * BarStep), prev, max(prev, c), min(prev, c), c}
		prev = c
	}
	return out
}

// steps returns n closes, each pct percent from the previous one, starting from `from` (excluded).
func steps(from float64, n int, pct float64) []float64 {
	out := make([]float64, n)
	for i := range out {
		from *= 1 + pct/100
		out[i] = from
	}
	return out
}

// flatN returns n closes equal to p.
func flatN(n int, p float64) []float64 {
	out := make([]float64, n)
	for i := range out {
		out[i] = p
	}
	return out
}

func fixture(bars []bar) strategy.MarketData {
	var md strategy.MarketData
	for _, b := range bars {
		md.Times = append(md.Times, b.t)
		md.Opens = append(md.Opens, b.o)
		md.Highs = append(md.Highs, b.h)
		md.Lows = append(md.Lows, b.l)
		md.Closes = append(md.Closes, b.c)
	}
	md.Price = md.Closes[len(md.Closes)-1]
	return md
}

func decide(p Params, bars []bar) model.Signal {
	return NewWithParams("TEST", p).Decide(fixture(bars))
}

// held decides with a position bought at entry on the bar starting at entryTime.
func held(p Params, bars []bar, entry float64, entryTime time.Time) model.Signal {
	md := fixture(bars)
	md.Position = &strategy.Position{PurchasePrice: entry, Quantity: 1, EntryTime: entryTime}
	return NewWithParams("TEST", p).Decide(md)
}

// dropAfter appends to closes one bar falling pct percent from the last close.
func dropAfter(closes []float64, pct float64) []float64 {
	return append(closes, closes[len(closes)-1]*(1-pct/100))
}

// downtrend is 20 closes sliding 0.1% per bar: RSI(6) sits near 0.
func downtrend() []float64 { return steps(100, 20, -0.1) }

func TestDefaultParams(t *testing.T) {
	want := Params{MinDropPct: 0.6, RSIPeriod: 6, RSIEntry: 30, RSIExit: 70, MACDExit: 1, MACDFast: 9, MACDSlow: 14,
		MACDSignal: 3, StochPeriod: 14, StochSmooth: 3, StochExit: 80, ProfitPct: 1, StopPct: 1, StartHour: 8}
	if got := DefaultParams(); got != want {
		t.Fatalf("DefaultParams() = %+v, want %+v", got, want)
	}
}

func TestEnterBuysOversoldDrop(t *testing.T) {
	sig := decide(DefaultParams(), path("2026-09-14 08:00", dropAfter(downtrend(), 0.6)))
	if sig.Kind != model.SignalBuy {
		t.Fatalf("kind = %v, want Buy", sig.Kind)
	}
	if sig.StopLoss != 0 || sig.TakeProfit != 0 {
		t.Fatalf("levels = %v/%v, want none: they are measured from the fill", sig.StopLoss, sig.TakeProfit)
	}
	if !strings.HasPrefix(sig.EntryReason, "DROP:") {
		t.Fatalf("reason = %q", sig.EntryReason)
	}
}

func TestEnterRejectsShallowDrop(t *testing.T) {
	if sig := decide(DefaultParams(), path("2026-09-14 08:00", dropAfter(downtrend(), 0.59))); sig.Kind != model.SignalNone {
		t.Fatalf("kind = %v, want None for 0.59%%", sig.Kind)
	}
}

func TestEnterRejectsRSIAtOrAboveEntry(t *testing.T) {
	closes := dropAfter(steps(100, 20, 1.0), 0.6) // strong uptrend: RSI(6) stays high after the drop
	if sig := decide(DefaultParams(), path("2026-09-14 08:00", closes)); sig.Kind != model.SignalNone {
		t.Fatalf("kind = %v, want None: RSI is not oversold", sig.Kind)
	}
	p := DefaultParams()
	p.RSIEntry = 100
	if sig := decide(p, path("2026-09-14 08:00", closes)); sig.Kind != model.SignalBuy {
		t.Fatalf("kind = %v, want Buy with RSIEntry 100 (the drop itself qualifies)", sig.Kind)
	}
}

func TestEnterNeedsRSIHistory(t *testing.T) {
	closes := dropAfter(steps(100, 5, -0.1), 0.6) // 6 closes: RSI(6) has no value yet
	if sig := decide(DefaultParams(), path("2026-09-14 08:00", closes)); sig.Kind != model.SignalNone {
		t.Fatalf("kind = %v, want None on RSI warm-up", sig.Kind)
	}
}

func TestEnterWindow(t *testing.T) {
	closes := dropAfter(downtrend(), 0.6) // 21 bars: the drop is the bar 300 minutes after start
	cases := []struct {
		name, start string
		want        model.SignalKind
	}{
		{"monday 13:00", "2026-09-14 08:00", model.SignalBuy},
		{"drop 07:45, before StartHour", "2026-09-14 02:45", model.SignalNone},
		{"drop 08:00, at StartHour", "2026-09-14 03:00", model.SignalBuy},
		{"saturday", "2026-09-19 08:00", model.SignalNone},
		{"drop 23:30, fill 23:45 same date", "2026-09-14 18:30", model.SignalBuy},
	}
	for _, c := range cases {
		if sig := decide(DefaultParams(), path(c.start, closes)); sig.Kind != c.want {
			t.Errorf("%s: kind = %v, want %v", c.name, sig.Kind, c.want)
		}
	}
}

func TestEnterRejectsFillOnNextDate(t *testing.T) {
	closes := dropAfter(downtrend(), 0.6)
	if sig := decide(DefaultParams(), path("2026-09-14 18:45", closes)); sig.Kind != model.SignalNone { // drop at 23:45
		t.Fatalf("kind = %v, want None: the fill would be the next morning", sig.Kind)
	}
}

func TestEnterInvalidParams(t *testing.T) {
	bars := path("2026-09-14 08:00", dropAfter(downtrend(), 0.6))
	for name, mut := range map[string]func(*Params){
		"MinDropPct 0": func(p *Params) { p.MinDropPct = 0 },
		"RSIPeriod 0":  func(p *Params) { p.RSIPeriod = 0 },
		"RSIEntry 0":   func(p *Params) { p.RSIEntry = 0 },
	} {
		p := DefaultParams()
		mut(&p)
		if sig := decide(p, bars); sig.Kind != model.SignalNone {
			t.Errorf("%s: kind = %v, want None", name, sig.Kind)
		}
	}
}

// exitsOff leaves only the stop on.
func exitsOff() Params {
	p := DefaultParams()
	p.RSIExit, p.StochExit, p.MACDExit, p.ProfitPct = 0, 0, 0, 0
	return p
}

// flatWithLast is 30 flat bars at 100 and a last bar with the given open/high/low/close.
func flatWithLast(o, h, l, c float64) []bar {
	bars := path("2026-09-14 08:00", flatN(30, 100))
	last := bars[len(bars)-1]
	return append(bars, bar{last.t.Add(BarStep), o, h, l, c})
}

func TestStopFromFillPrice(t *testing.T) {
	bars := flatWithLast(100, 100, 99.0, 99.5)
	sig := held(exitsOff(), bars, 100, bars[20].t)
	if sig.Kind != model.SignalSell || sig.Reason != "SL" || math.Abs(sig.StopLoss-99.0) > eps {
		t.Fatalf("got %v %q stop %v, want SL at 99.0", sig.Kind, sig.Reason, sig.StopLoss)
	}
	bars = flatWithLast(100, 100, 99.01, 99.5)
	if sig := held(exitsOff(), bars, 100, bars[20].t); sig.Kind != model.SignalNone {
		t.Fatalf("low 99.01: kind = %v, want None", sig.Kind)
	}
	p := exitsOff()
	p.StopPct = 0
	bars = flatWithLast(100, 100, 50, 60)
	if sig := held(p, bars, 100, bars[20].t); sig.Kind != model.SignalNone {
		t.Fatalf("StopPct 0: kind = %v, want None", sig.Kind)
	}
}

func TestNoEODExit(t *testing.T) {
	// Overnight: entry Monday 23:45, the bar is Tuesday 07:00 with nothing triggered.
	bars := path("2026-09-14 16:15", flatN(31, 100)) // last bar 23:45
	bars = append(bars, bar{at("2026-09-15 07:00"), 100, 100.2, 99.8, 100})
	if sig := held(exitsOff(), bars, 100, at("2026-09-14 23:45")); sig.Kind != model.SignalNone {
		t.Fatalf("kind = %v %q, want None: positions carry overnight", sig.Kind, sig.Reason)
	}
}
```

- [ ] **Step 3: Run tests to verify they fail**

Run: `go test ./internal/service/trading_strategy/drop_bounce/...`
Expected: FAIL — compile errors (`BarStep` undefined, unknown `Params` fields).

- [ ] **Step 4: Write `core/core.go`**

```go
// Package core implements drop_bounce: a long-only 15-minute bounce after a large red bar that
// closes oversold. A bar falling at least MinDropPct percent from its open to its close, with
// RSI(RSIPeriod) at its close below RSIEntry, signals a buy; the backtest fills it at the next
// bar's open (Config.EntryAtNextOpen). Signals are taken on Mon–Fri MSK bars starting at or after
// StartHour whose fill bar lies on the same date. Exits, in order: a stop StopPct below the fill
// price, a ProfitPct profit at the fill bar's close, the first red MACD histogram bar after a
// green one closed during the trade, RSI crossing RSIExit upward, the smoothed stochastic %K
// crossing StochExit upward. There is no end-of-day exit: positions carry overnight. Pure,
// stateless between bars and ticker-agnostic. Run with `-strategy drop_bounce -interval Minutes15`.
package core

import (
	"fmt"
	"time"

	"tinvest/internal/service/trading_strategy/scalping/model"
	"tinvest/internal/service/trading_strategy/scalping/strategy"
	"tinvest/pkg/indicators"
)

// BarStep is the only timeframe drop_bounce runs on. It matters only for the entry window: the
// fill bar starts one step after the signal bar and must lie on the same MSK date.
const BarStep = 15 * time.Minute

// minLookback keeps 16 hours of 15m bars so the indicators converge and the MACD scan sees the
// holding bars of a typical trade.
const minLookback = 64

// Params holds every tunable. All fields are int or float64 so reflection grid calibration can
// sweep them. A zero threshold disables its exit.
type Params struct {
	MinDropPct float64 // minimum (open−close)/open of the signal bar, percent (default 0.6)
	RSIPeriod  int     // RSI length for the entry and the RSI exit (default 6)
	RSIEntry   float64 // the signal bar's RSI must be below this (default 30)
	RSIExit    float64 // sell when RSI crosses this upward; 0 = off (default 70)
	// MACDExit 1 sells on the first red histogram bar (MACD − signal < 0) after a green one (> 0)
	// that closed while holding; bars before the entry do not count; 0 = off (default 1).
	MACDExit    int
	MACDFast    int     // default 9
	MACDSlow    int     // default 14
	MACDSignal  int     // default 3
	StochPeriod int     // stochastic %K period (default 14)
	StochSmooth int     // SMA smoothing of %K; 1 = raw %K (default 3)
	StochExit   float64 // sell when smoothed %K crosses this upward; 0 = off (default 80)
	ProfitPct   float64 // sell when the fill bar closes this percent above the fill; 0 = off (default 1)
	StopPct     float64 // stop this percent below the fill price; 0 = off (default 1)
	StartHour   int     // signal bars must start at/after StartHour:00 MSK (default 8)
}

// DefaultParams returns the owner's initial configuration.
func DefaultParams() Params {
	return Params{
		MinDropPct:  0.6,
		RSIPeriod:   6,
		RSIEntry:    30,
		RSIExit:     70,
		MACDExit:    1,
		MACDFast:    9,
		MACDSlow:    14,
		MACDSignal:  3,
		StochPeriod: 14,
		StochSmooth: 3,
		StochExit:   80,
		ProfitPct:   1,
		StopPct:     1,
		StartHour:   8,
	}
}

// Strategy trades a single instrument with the drop_bounce rules. Ticker-agnostic and pure.
type Strategy struct {
	ticker string
	p      Params
}

// NewWithParams returns the strategy for a ticker with explicit params.
func NewWithParams(ticker string, p Params) *Strategy {
	return &Strategy{ticker: ticker, p: p}
}

func (s *Strategy) Ticker() string { return s.ticker }

// Lookback is enough history for every indicator to converge (Wilder RSI and the EMA-based MACD
// need several periods) plus the previous bar each cross compares against.
func (s *Strategy) Lookback() int {
	return max(minLookback, 4*s.p.RSIPeriod+2, 3*s.p.MACDSlow+s.p.MACDSignal+2, s.p.StochPeriod+s.p.StochSmooth+2)
}

// mskLoc anchors every calendar rule to Moscow (UTC fallback).
var mskLoc = func() *time.Location {
	loc, err := time.LoadLocation("Europe/Moscow")
	if err != nil {
		return time.UTC
	}
	return loc
}()

// dayKey is the MSK calendar date of t as yyyymmdd.
func dayKey(t time.Time) int {
	tl := t.In(mskLoc)
	return tl.Year()*10000 + int(tl.Month())*100 + tl.Day()
}

// inWindow reports whether a signal on the bar starting at t may open a position: a weekday, at
// or after StartHour, and the fill bar (one step later) on the same MSK date.
func (s *Strategy) inWindow(t time.Time) bool {
	tl := t.In(mskLoc)
	if wd := tl.Weekday(); wd == time.Saturday || wd == time.Sunday {
		return false
	}
	if tl.Hour() < s.p.StartHour {
		return false
	}
	return dayKey(tl.Add(BarStep)) == dayKey(tl)
}

func (s *Strategy) Decide(md strategy.MarketData) model.Signal {
	sig := model.Signal{Ticker: s.ticker, Price: md.Price}
	if md.Position != nil {
		return s.manage(md, sig)
	}
	return s.enter(md, sig)
}

// enter buys after a red bar falling at least MinDropPct percent whose close leaves RSI below
// RSIEntry. The signal carries no levels: the stop and the profit are measured from the fill.
func (s *Strategy) enter(md strategy.MarketData, sig model.Signal) model.Signal {
	p := s.p
	n := len(md.Closes)
	if n == 0 || len(md.Times) != n || len(md.Opens) != n {
		return sig
	}
	// RSISeries is 0 up to index RSIPeriod−1; n > RSIPeriod keeps the warm-up out of the check.
	if p.MinDropPct <= 0 || p.RSIPeriod <= 0 || p.RSIEntry <= 0 || n <= p.RSIPeriod {
		return sig
	}
	i := n - 1
	if !s.inWindow(md.Times[i]) {
		return sig
	}
	o, c := md.Opens[i], md.Closes[i]
	if o <= 0 || c >= o {
		return sig
	}
	dropPct := (o - c) / o * 100
	// The tiny epsilon keeps a fall of exactly MinDropPct from failing on float rounding.
	if dropPct < p.MinDropPct-1e-9 {
		return sig
	}
	rsi := indicators.RSISeries(md.Closes, p.RSIPeriod)[i]
	if rsi >= p.RSIEntry {
		return sig
	}
	sig.Kind = model.SignalBuy
	sig.EntryReason = fmt.Sprintf("DROP: open %.4f → close %.4f, падение %.2f%%, RSI(%d) %.1f < %.0f",
		o, c, dropPct, p.RSIPeriod, rsi, p.RSIEntry)
	return sig
}

// manage exits in priority order: the stop below the fill price; Task 3 adds the rest.
func (s *Strategy) manage(md strategy.MarketData, sig model.Signal) model.Signal {
	pos := md.Position
	n := len(md.Closes)
	if n == 0 || len(md.Highs) != n || len(md.Lows) != n || pos.PurchasePrice <= 0 {
		return sig
	}
	i := n - 1
	entry := pos.PurchasePrice
	if s.p.StopPct > 0 {
		stop := entry * (1 - s.p.StopPct/100)
		if md.Lows[i] <= stop+1e-9 {
			sig.Kind, sig.Reason, sig.StopLoss = model.SignalSell, "SL", stop
			sig.ExitReason = fmt.Sprintf("SL: low %.4f ≤ стоп %.4f (вход %.4f, −%.2f%%)", md.Lows[i], stop, entry, s.p.StopPct)
			return sig
		}
	}
	return sig
}
```

Note: the comment on `manage` says "Task 3 adds the rest" — Task 3 Step 3 replaces that comment; it must not survive into the final code.

- [ ] **Step 5: Rewrite the registry** `internal/service/backtest/drop_bounce_registry.go`:

```go
package backtest

import (
	"encoding/json"
	"fmt"

	"tinvest/internal/enum"
	"tinvest/internal/service/trading_strategy/drop_bounce/strategy/core"
	"tinvest/internal/service/trading_strategy/scalping/strategy"
)

// dropBounceBindingFor builds a drop_bounce binding for one ticker whose baseline params come from
// defaults: a ticker package literal, or core.DefaultParams when the ticker has none.
func dropBounceBindingFor(ticker string, defaults func() core.Params) Binding {
	return Binding{
		DefaultParams: func() any { return defaults() },
		Build: func(params any) strategy.Strategy {
			return core.NewWithParams(ticker, params.(core.Params))
		},
		ParseParams: func(raw []byte) (any, error) {
			p := defaults() // start from defaults so partial JSON overrides
			if err := json.Unmarshal(raw, &p); err != nil {
				return nil, fmt.Errorf("backtest: parse drop_bounce params: %w", err)
			}
			return p, nil
		},
	}
}

// dropBounceRegistry maps tickers with their own package to their 15m literal. Empty until a
// ticker is calibrated.
var dropBounceRegistry = map[string]func() core.Params{}

// DropBounceLookupOrGeneric returns the binding for a ticker: the ticker package's literal, or
// core.DefaultParams when there is none. Partial JSON overrides layer over whichever defaults
// apply. drop_bounce runs on Minutes15 only; other timeframes are rejected.
func DropBounceLookupOrGeneric(ticker string, interval enum.Interval) (Binding, error) {
	if interval != enum.Minutes15 {
		return Binding{}, fmt.Errorf("drop_bounce runs on Minutes15 only, not %s", interval)
	}
	defaults := core.DefaultParams
	if d, ok := dropBounceRegistry[ticker]; ok {
		defaults = d
	}
	return dropBounceBindingFor(ticker, defaults), nil
}
```

And `drop_bounce_registry_test.go`:

```go
package backtest

import (
	"testing"

	"tinvest/internal/enum"
	"tinvest/internal/service/trading_strategy/drop_bounce/strategy/core"
)

func TestDropBounceGenericTickerUsesCoreDefaults(t *testing.T) {
	b, err := DropBounceLookupOrGeneric("ZZZZ", enum.Minutes15)
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

func TestDropBounceRejectsOtherTimeframes(t *testing.T) {
	for _, iv := range []enum.Interval{enum.Minutes5, enum.Minutes30, enum.Hour1, enum.Day1} {
		if _, err := DropBounceLookupOrGeneric("GAZP", iv); err == nil {
			t.Fatalf("%s: want error", iv)
		}
	}
}

func TestDropBounceParseParamsLayersOverDefaults(t *testing.T) {
	b, err := DropBounceLookupOrGeneric("ZZZZ", enum.Minutes15)
	if err != nil {
		t.Fatalf("lookup: %v", err)
	}
	parsed, err := b.ParseParams([]byte(`{"MinDropPct": 2.5}`))
	if err != nil {
		t.Fatalf("ParseParams: %v", err)
	}
	got := parsed.(core.Params)
	if got.MinDropPct != 2.5 || got.StopPct != core.DefaultParams().StopPct {
		t.Fatalf("got %+v", got)
	}
}
```

- [ ] **Step 6: Run tests to verify they pass**

Run: `go build ./internal/... ./pkg/... ./cmd/... && go test ./internal/service/trading_strategy/drop_bounce/... ./internal/service/backtest/...`
Expected: build OK, both packages `ok`.

- [ ] **Step 7: Commit**

```bash
git add -A internal/service/trading_strategy/drop_bounce internal/service/backtest/drop_bounce_registry.go internal/service/backtest/drop_bounce_registry_test.go data/params/drop_bounce docs/drop_bounce
git commit -m "feat(drop_bounce): new 15m core — oversold drop entry, stop from the fill; drop GAZP/T literals"
```

---

### Task 3: Exits — PROFIT on the fill bar, MACD green→red, RSI and stochastic crosses

**Files:**
- Modify: `internal/service/trading_strategy/drop_bounce/strategy/core/core.go` (`manage`, new helpers `greenThenRed`, `crossUp`)
- Create: `internal/service/trading_strategy/drop_bounce/strategy/core/exits_test.go`

**Interfaces:**
- Consumes: `Params`, `DefaultParams`, `NewWithParams`, `BarStep`; test helpers `at`, `bar`, `path`, `steps`, `flatN`, `fixture`, `held`, `exitsOff`, `flatWithLast`, `eps` from `core_test.go` (Task 2).
- Produces: `func greenThenRed(h []float64, times []time.Time, entry time.Time) bool`, `func crossUp(prev, cur, level float64) bool`.

- [ ] **Step 1: Write the failing tests** — `core/exits_test.go`:

```go
package core

import (
	"testing"
	"time"

	"tinvest/internal/service/trading_strategy/scalping/model"
	"tinvest/pkg/indicators"
)

func TestProfitOnlyOnFillBar(t *testing.T) {
	p := exitsOff()
	p.ProfitPct = 1
	bars := flatWithLast(100, 101, 100, 101)
	fill := bars[len(bars)-1].t
	if sig := held(p, bars, 100, fill); sig.Kind != model.SignalSell || sig.Reason != "PROFIT" {
		t.Fatalf("fill bar +1%%: got %v %q, want PROFIT", sig.Kind, sig.Reason)
	}
	bars = flatWithLast(100, 100.99, 100, 100.99)
	if sig := held(p, bars, 100, bars[len(bars)-1].t); sig.Kind != model.SignalNone {
		t.Fatalf("fill bar +0.99%%: kind = %v, want None", sig.Kind)
	}
	bars = flatWithLast(100, 101, 100, 101)
	if sig := held(p, bars, 100, bars[len(bars)-2].t); sig.Kind != model.SignalNone {
		t.Fatalf("second bar +1%%: kind = %v, want None (only the fill bar counts)", sig.Kind)
	}
}

func TestStopBeatsProfit(t *testing.T) {
	p := exitsOff()
	p.ProfitPct = 1
	bars := flatWithLast(100, 101.5, 98.9, 101.2)
	if sig := held(p, bars, 100, bars[len(bars)-1].t); sig.Reason != "SL" {
		t.Fatalf("reason = %q, want SL first", sig.Reason)
	}
}

func TestGreenThenRed(t *testing.T) {
	t0 := at("2026-09-14 10:00")
	times := make([]time.Time, 6)
	for i := range times {
		times[i] = t0.Add(time.Duration(i) * BarStep)
	}
	entry := times[2] // bars 2..5 are in the trade
	cases := []struct {
		name  string
		h     []float64
		entry time.Time
		want  bool
	}{
		{"green then red in trade", []float64{-1, -1, -1, 0.5, 0.2, -0.1}, entry, true},
		{"never green in trade", []float64{-1, -1, -1, -0.5, -0.2, -0.1}, entry, false},
		{"green only before entry", []float64{0.5, 0.4, -0.1, -0.2, -0.3, -0.4}, entry, false},
		{"green on the fill bar counts", []float64{-1, -1, 0.3, -0.1, -0.2, -0.3}[:4], entry, true},
		{"still green", []float64{-1, -1, -1, 0.5, 0.6, 0.7}, entry, false},
		{"zero between green and red", []float64{-1, -1, -1, 0.5, 0, -0.1}, entry, true},
		{"unknown entry scans the window", []float64{0.5, -1, -1, -1, -1, -0.1}, time.Time{}, false},
		{"unknown entry, green then red", []float64{-1, -1, -1, -1, 0.1, -0.1}, time.Time{}, true},
	}
	for _, c := range cases {
		if got := greenThenRed(c.h, times[:len(c.h)], c.entry); got != c.want {
			t.Errorf("%s: got %v, want %v", c.name, got, c.want)
		}
	}
}

func TestCrossUp(t *testing.T) {
	if !crossUp(69.9, 70, 70) || !crossUp(10, 90, 70) {
		t.Fatal("want cross")
	}
	if crossUp(70, 75, 70) || crossUp(60, 69.9, 70) {
		t.Fatal("want no cross")
	}
}

// macdHist is MACD − signal over closes with the default periods.
func macdHist(closes []float64) []float64 {
	p := DefaultParams()
	m, s := indicators.MACD(closes, p.MACDFast, p.MACDSlow, p.MACDSignal)
	h := make([]float64, len(closes))
	for i := range h {
		h[i] = m[i] - s[i]
	}
	return h
}

// firstSell walks the path bar by bar holding from bar entryIdx and returns the index and signal
// of the first sell, or −1.
func firstSell(p Params, bars []bar, entryIdx int) (int, model.Signal) {
	for k := entryIdx; k < len(bars); k++ {
		if sig := held(p, bars[:k+1], bars[entryIdx].o, bars[entryIdx].t); sig.Kind == model.SignalSell {
			return k, sig
		}
	}
	return -1, model.Signal{}
}

func macdOnly() Params {
	p := exitsOff()
	p.StopPct, p.MACDExit = 0, 1
	return p
}

func TestDecideMACDExitAfterGreenInTrade(t *testing.T) {
	closes := append(flatN(60, 100), steps(100, 6, 0.5)...)
	closes = append(closes, steps(closes[len(closes)-1], 15, -0.5)...)
	bars := path("2026-09-14 08:00", closes)
	k, sig := firstSell(macdOnly(), bars, 60)
	if k < 0 || sig.Reason != "MACD" {
		t.Fatalf("first sell at %d %q, want a MACD exit", k, sig.Reason)
	}
	h := macdHist(closes[:k+1])
	if h[k] >= 0 || h[k-1] <= 0 {
		t.Fatalf("exit at %d: hist %v → %v, want the first red after green", k, h[k-1], h[k])
	}
}

func TestDecideMACDIgnoresGreenBeforeEntry(t *testing.T) {
	closes := append(flatN(40, 100), steps(100, 6, 0.5)...)
	closes = append(closes, steps(closes[len(closes)-1], 10, -0.5)...)
	h := macdHist(closes)
	r := -1
	for i := 46; i < len(h); i++ {
		if h[i] < 0 {
			r = i
			break
		}
	}
	if r < 0 {
		t.Fatal("fixture: histogram never turned red after the rise")
	}
	for i := r; i < len(h); i++ {
		if h[i] >= 0 {
			t.Fatalf("fixture: histogram green again at %d", i)
		}
	}
	bars := path("2026-09-14 08:00", closes)
	if k, sig := firstSell(macdOnly(), bars, r); k >= 0 {
		t.Fatalf("sell at %d %q, want hold: the trade never saw a green bar", k, sig.Reason)
	}
}

func TestDecideRSICross(t *testing.T) {
	closes := append(flatN(30, 100), steps(100, 6, -0.3)...)
	closes = append(closes, steps(closes[len(closes)-1], 10, 0.4)...)
	bars := path("2026-09-14 08:00", closes)
	p := exitsOff()
	p.StopPct, p.RSIExit = 0, 70
	k, sig := firstSell(p, bars, 36)
	if k < 0 || sig.Reason != "RSI" {
		t.Fatalf("first sell at %d %q, want RSI", k, sig.Reason)
	}
	r := indicators.RSISeries(closes[:k+1], p.RSIPeriod)
	if !(r[k-1] < 70 && r[k] >= 70) {
		t.Fatalf("exit at %d: RSI %v → %v, want a cross of 70", k, r[k-1], r[k])
	}
}

func TestDecideStochCross(t *testing.T) {
	closes := append(flatN(30, 100), steps(100, 6, -0.3)...)
	closes = append(closes, steps(closes[len(closes)-1], 10, 0.4)...)
	bars := path("2026-09-14 08:00", closes)
	p := exitsOff()
	p.StopPct, p.StochExit = 0, 80
	k, sig := firstSell(p, bars, 36)
	if k < 0 || sig.Reason != "STOCH" {
		t.Fatalf("first sell at %d %q, want STOCH", k, sig.Reason)
	}
	md := fixture(bars[:k+1])
	_, ds := indicators.StochasticSeries(md.Highs, md.Lows, md.Closes, p.StochPeriod, p.StochSmooth)
	if !(ds[len(ds)-2] < 80 && ds[len(ds)-1] >= 80) {
		t.Fatalf("exit at %d: %%K %v → %v, want a cross of 80", k, ds[len(ds)-2], ds[len(ds)-1])
	}
}

func TestDecideAllExitsOffHolds(t *testing.T) {
	closes := append(flatN(30, 100), steps(100, 6, -0.3)...)
	closes = append(closes, steps(closes[len(closes)-1], 10, 0.4)...)
	p := exitsOff()
	p.StopPct = 0
	if k, sig := firstSell(p, path("2026-09-14 08:00", closes), 36); k >= 0 {
		t.Fatalf("sell at %d %q, want hold with every exit off", k, sig.Reason)
	}
}

func TestDecideInvalidPeriodsDoNotPanic(t *testing.T) {
	closes := append(flatN(30, 100), steps(100, 6, -0.3)...)
	closes = append(closes, steps(closes[len(closes)-1], 10, 0.4)...)
	bars := path("2026-09-14 08:00", closes)
	p := DefaultParams()
	p.StopPct, p.ProfitPct = 0, 0
	p.MACDFast, p.MACDSlow = 14, 9
	p.StochPeriod = 0
	p.RSIExit = 0
	if k, sig := firstSell(p, bars, 36); k >= 0 {
		t.Fatalf("sell at %d %q, want hold: invalid MACD/stochastic periods disable those exits", k, sig.Reason)
	}
}
```

Note on `"green on the fill bar counts"`: the slice is 4 values `{-1, -1, 0.3, -0.1}` with entry at index 2 — the fill bar is green, the next bar red.

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/service/trading_strategy/drop_bounce/...`
Expected: FAIL — `undefined: greenThenRed`, `undefined: crossUp`.

- [ ] **Step 3: Implement** — in `core.go` replace `manage` and add the helpers:

```go
// manage exits in priority order: the stop below the fill price (intrabar, filled by the engine at
// min(stop, open)), then at the bar's close the fill-bar profit, the MACD green→red flip, the RSI
// cross and the stochastic cross. There is no end-of-day exit.
func (s *Strategy) manage(md strategy.MarketData, sig model.Signal) model.Signal {
	pos := md.Position
	n := len(md.Closes)
	if n == 0 || len(md.Highs) != n || len(md.Lows) != n || pos.PurchasePrice <= 0 {
		return sig
	}
	i := n - 1
	entry, closeP := pos.PurchasePrice, md.Closes[i]
	sell := func(reason, why string) model.Signal {
		sig.Kind, sig.Reason = model.SignalSell, reason
		sig.ExitReason = fmt.Sprintf("%s: %s, выход по %.4f (вход %.4f)", reason, why, closeP, entry)
		return sig
	}
	if s.p.StopPct > 0 {
		stop := entry * (1 - s.p.StopPct/100)
		// The tiny epsilon keeps a low of exactly the stop from slipping on float rounding.
		if md.Lows[i] <= stop+1e-9 {
			sig.Kind, sig.Reason, sig.StopLoss = model.SignalSell, "SL", stop
			sig.ExitReason = fmt.Sprintf("SL: low %.4f ≤ стоп %.4f (вход %.4f, −%.2f%%)", md.Lows[i], stop, entry, s.p.StopPct)
			return sig
		}
	}
	if s.p.ProfitPct > 0 && len(md.Times) == n && md.Times[i].Equal(pos.EntryTime) &&
		closeP >= entry*(1+s.p.ProfitPct/100)-1e-9 {
		return sell("PROFIT", fmt.Sprintf("бар входа закрылся на +%.2f%%", (closeP/entry-1)*100))
	}
	if s.p.MACDExit != 0 {
		if m, sg := indicators.MACD(md.Closes, s.p.MACDFast, s.p.MACDSlow, s.p.MACDSignal); m != nil {
			h := make([]float64, n)
			for j := range h {
				h[j] = m[j] - sg[j]
			}
			if greenThenRed(h, md.Times, pos.EntryTime) {
				return sell("MACD", fmt.Sprintf("гистограмма %.4f — первый красный бар после зелёного", h[i]))
			}
		}
	}
	if s.p.RSIExit > 0 && s.p.RSIPeriod > 0 && n >= s.p.RSIPeriod+2 {
		r := indicators.RSISeries(md.Closes, s.p.RSIPeriod)
		if crossUp(r[i-1], r[i], s.p.RSIExit) {
			return sell("RSI", fmt.Sprintf("RSI(%d) %.1f → %.1f пересёк %.0f вверх", s.p.RSIPeriod, r[i-1], r[i], s.p.RSIExit))
		}
	}
	if s.p.StochExit > 0 {
		_, k := indicators.StochasticSeries(md.Highs, md.Lows, md.Closes, s.p.StochPeriod, s.p.StochSmooth)
		if m := len(k); m >= 2 && crossUp(k[m-2], k[m-1], s.p.StochExit) {
			return sell("STOCH", fmt.Sprintf("%%K %.1f → %.1f пересёк %.0f вверх", k[m-2], k[m-1], s.p.StochExit))
		}
	}
	return sig
}

// greenThenRed reports whether the histogram's last bar is red (< 0) and the latest non-zero bar
// before it, among bars at or after entry, is green (> 0). A still-open position has not yet
// seen a red bar after a green one — it would have exited — so this is exactly "the first red
// after a green during the trade". Bars before entry never count; with an unknown entry or
// misaligned times the whole window counts.
func greenThenRed(h []float64, times []time.Time, entry time.Time) bool {
	last := len(h) - 1
	if last < 1 || h[last] >= 0 {
		return false
	}
	checkEntry := !entry.IsZero() && len(times) == len(h)
	for j := last - 1; j >= 0; j-- {
		if checkEntry && times[j].Before(entry) {
			return false
		}
		if h[j] > 0 {
			return true
		}
		if h[j] < 0 {
			return false
		}
	}
	return false
}

// crossUp reports whether a series moved from below level to at or above it.
func crossUp(prev, cur, level float64) bool { return prev < level && cur >= level }
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/service/trading_strategy/drop_bounce/...`
Expected: `ok`. If a `TestDecide*` fixture assertion (`fixture: ...`) fails, the price path does not produce the shape the test needs — adjust the step sizes in that test's fixture (not the production code) until the precondition holds, and keep the behavioural assertions unchanged.

- [ ] **Step 5: Commit**

```bash
git add internal/service/trading_strategy/drop_bounce/strategy/core
git commit -m "feat(drop_bounce): exits — fill-bar profit, MACD green→red, RSI and stochastic crosses"
```

---

### Task 4: Engine tests, starter grid, docs, full gate

**Files:**
- Create: `internal/service/trading_strategy/drop_bounce/strategy/core/engine_test.go`
- Create: `data/params/drop_bounce/grid_example.json`
- Rewrite: `docs/drop_bounce/strategy.md`
- Modify: `CLAUDE.md` (the `drop_bounce` sentence in the `trading_strategy/` bullet)

**Interfaces:**
- Consumes: `core.NewWithParams`, `core.DefaultParams`, `BarStep`, test helpers `at`, `bar`, `path`, `flatN`, `downtrend`, `dropAfter`, `eps` (Task 2); `bt.Run`, `bt.Candle`, `bt.Config{EntryAtNextOpen: true}`.

- [ ] **Step 1: Write the engine tests** — `core/engine_test.go`:

```go
// engine_test.go runs drop_bounce through the real backtest engine to pin the fills it relies on:
// entry at the next bar's open, the stop from the fill price filled at the gap open overnight, and
// the fill-bar profit at the close.
package core

import (
	"math"
	"testing"

	bt "tinvest/internal/domain/backtest"
)

func toCandles(bars []bar) []bt.Candle {
	out := make([]bt.Candle, len(bars))
	for i, b := range bars {
		out[i] = bt.Candle{Time: b.t, Open: b.o, High: b.h, Low: b.l, Close: b.c}
	}
	return out
}

// runAfterDrop runs the engine over the oversold drop at Monday 13:00 followed by after and
// returns the single trade. 50 flat bars lead the downtrend: the engine decides only from bar
// Lookback()−1 (63) on, so the drop must sit past it (71 bars, the drop at index 70).
func runAfterDrop(t *testing.T, after func(entry float64) []bar) (bt.Trade, float64) {
	t.Helper()
	bars := path("2026-09-13 19:30", dropAfter(append(flatN(50, 100), downtrend()...), 0.6))
	entry := bars[len(bars)-1].c
	bars = append(bars, after(entry)...)
	res := bt.Run(NewWithParams("TEST", DefaultParams()), toCandles(bars), nil, nil,
		bt.Config{InitialCash: 1e6, Fraction: 1, Commission: 0, Lot: 1, EntryAtNextOpen: true})
	if len(res.Trades) != 1 {
		t.Fatalf("trades = %d, want 1: %+v", len(res.Trades), res.Trades)
	}
	tr := res.Trades[0]
	if math.Abs(tr.EntryPrice-entry) > eps || !tr.EntryTime.Equal(at("2026-09-14 13:15")) {
		t.Fatalf("entry %v @ %v, want %v @ 13:15 (next bar's open)", tr.EntryPrice, tr.EntryTime, entry)
	}
	return tr, entry
}

func TestEngineOvernightGapStop(t *testing.T) {
	tr, entry := runAfterDrop(t, func(e float64) []bar {
		gap := e * 0.97
		return []bar{
			{at("2026-09-14 13:15"), e, e, e, e},
			{at("2026-09-15 07:00"), gap, gap, gap * 0.999, gap},
		}
	})
	if tr.Reason != "SL" || math.Abs(tr.ExitPrice-entry*0.97) > 1e-6 || !tr.ExitTime.Equal(at("2026-09-15 07:00")) {
		t.Fatalf("exit %q @ %v %v, want SL at the gap open %v next morning", tr.Reason, tr.ExitPrice, tr.ExitTime, entry*0.97)
	}
}

func TestEngineProfitOnFillBar(t *testing.T) {
	tr, entry := runAfterDrop(t, func(e float64) []bar {
		c := e * 1.012
		return []bar{{at("2026-09-14 13:15"), e, c, e, c}}
	})
	if tr.Reason != "PROFIT" || math.Abs(tr.ExitPrice-entry*1.012) > 1e-6 {
		t.Fatalf("exit %q @ %v, want PROFIT at the fill bar's close %v", tr.Reason, tr.ExitPrice, entry*1.012)
	}
}
```

- [ ] **Step 2: Run them**

Run: `go test ./internal/service/trading_strategy/drop_bounce/... -run Engine -v`
Expected: both PASS (the code already exists; these pin the engine wiring). If `TestEngineOvernightGapStop` fails because the trade is still open at the end of the data (no trade), check that the engine reports trades closed by `SL` with `exitPrice = min(stop, open)` — do not change production code to satisfy it without reporting.

- [ ] **Step 3: Starter grid** — `data/params/drop_bounce/grid_example.json`:

```json
{
  "_comment": "drop_bounce 15m, стартовая сетка поверх core.DefaultParams(): глубина падения, порог входа RSI, процентные стоп и профит. Индикаторные выходы перебираются отдельной фазой (0 выключает выход).",
  "phases": [
    {
      "name": "entry",
      "grid": {
        "MinDropPct": [0.5, 0.6, 0.8, 1.0],
        "RSIPeriod": [4, 6, 9],
        "RSIEntry": [20, 25, 30]
      }
    },
    {
      "name": "risk",
      "grid": {
        "StopPct": [0.7, 1.0, 1.5, 2.0],
        "ProfitPct": [0, 0.7, 1.0, 1.5]
      }
    },
    {
      "name": "exits",
      "grid": {
        "MACDExit": [0, 1],
        "RSIExit": [0, 65, 70, 75],
        "StochExit": [0, 80, 90]
      }
    }
  ]
}
```

- [ ] **Step 4: Rewrite `docs/drop_bounce/strategy.md`**

```markdown
# drop_bounce — отскок после большой красной 15m-свечи в перепроданности

Бэктест-стратегия, только лонг, таймфрейм 15m (`-interval Minutes15`; другие cmd/backtest отклоняет). Позиция может переноситься через ночь. Код: `internal/service/trading_strategy/drop_bounce/strategy/core`, привязка к бэктесту — `internal/service/backtest/drop_bounce_registry.go`. Спека: `docs/superpowers/specs/2026-10-09-drop-bounce-rework-design.md`.

## Вход

1. Последняя закрытая свеча красная и её тело не меньше `MinDropPct` процентов: `(open − close) / open · 100 ≥ MinDropPct`.
2. RSI(`RSIPeriod`) по Уайлдеру на close этой свечи ниже `RSIEntry`. Пока истории меньше `RSIPeriod + 1` свечей, сигнала нет.
3. Окно: свеча-сигнал — Пн–Пт МСК, начинается не раньше `StartHour:00`, а свеча исполнения (через 15 минут) лежит в ту же дату. Сигнал на последней свече дня на утро не переносится.
4. Покупка по open следующей свечи (`Config.EntryAtNextOpen`). Сигнал уровней не несёт: стоп и профит считаются от цены исполнения.

Пока позиция открыта, новые сигналы игнорируются.

## Выход

Проверяется на каждой свече удержания, включая свечу входа, по порядку; срабатывает первое условие.

1. **SL** — low свечи ≤ `вход · (1 − StopPct/100)`. Движок исполняет по `min(стоп, open)`: ночной гэп ниже стопа продаётся по open.
2. **PROFIT** — только на свече входа: её close ≥ `вход · (1 + ProfitPct/100)`. Продажа по close. На следующих свечах правило не действует.
3. **MACD** — гистограмма `MACD − signal` (`MACDFast/MACDSlow/MACDSignal`). Зелёная — больше нуля, красная — меньше. Цвет до входа не важен. Если за время сделки (с свечи входа включительно) гистограмма хоть раз закрылась зелёной, выход по close первой следующей красной свечи. Пока зелёной в сделке не было, красные свечи выхода не дают.
4. **RSI** — RSI(`RSIPeriod`) пересёк `RSIExit` снизу вверх: на прошлой свече ниже, на текущей не ниже. Продажа по close.
5. **STOCH** — сглаженный %K стохастика (`StochPeriod`, сглаживание SMA(`StochSmooth`), 1 — сырой %K) пересёк `StochExit` снизу вверх. Продажа по close.

Выхода по концу дня нет. Нулевой порог выключает свой выход (`StopPct`, `ProfitPct`, `RSIExit`, `StochExit`; у MACD — `MACDExit = 0`). Неверные периоды MACD или стохастика выключают этот выход.

## Параметры (`core.Params`)

| Поле | По умолчанию | Смысл |
|---|---|---|
| `MinDropPct` | 0.6 | минимальное падение тела свечи-сигнала, % |
| `RSIPeriod` | 6 | длина RSI для входа и выхода |
| `RSIEntry` | 30 | вход, если RSI ниже |
| `RSIExit` | 70 | выход при пересечении снизу вверх (0 — выключен) |
| `MACDExit` | 1 | 1 — выход MACD включён, 0 — выключен |
| `MACDFast`, `MACDSlow`, `MACDSignal` | 9, 14, 3 | периоды MACD |
| `StochPeriod`, `StochSmooth` | 14, 3 | период %K и его сглаживание |
| `StochExit` | 80 | выход при пересечении %K снизу вверх (0 — выключен) |
| `ProfitPct` | 1 | профит на close свечи входа, % (0 — выключен) |
| `StopPct` | 1 | стоп ниже цены входа, % (0 — выключен) |
| `StartHour` | 8 | начало окна сигналов, час МСК |

Литерал тикера — пакет `internal/service/trading_strategy/drop_bounce/strategy/<ticker>/` с `DefaultParams()`, регистрация — `dropBounceRegistry` (тикер → литерал); без литерала берётся `core.DefaultParams`. JSON из `-params` / `-calibrate` частично переопределяет поля. Стартовая сетка — `data/params/drop_bounce/grid_example.json`, сетки тикера — `data/params/drop_bounce/<ticker>/`.

## Запуск

```
go run ./cmd/backtest -ticker GAZP -strategy drop_bounce -interval Minutes15 -months 12
go run ./cmd/backtest -ticker GAZP -strategy drop_bounce -interval Minutes15 -months 24 -calibrate data/params/drop_bounce/grid_example.json
```
```

- [ ] **Step 5: Update `CLAUDE.md`** — replace the whole sentence starting with `` `drop_bounce` — backtest-only 5m/15m/1h long bounce`` up to and including `grids in \`data/params/drop_bounce/<ticker>/\`.` with:

```
`drop_bounce` — backtest-only 15m long bounce after a red bar of ≥ `MinDropPct`% (0.6) closing with RSI(`RSIPeriod`=6) < `RSIEntry` (30): entry at the next open; exits — stop `StopPct`% below the fill, fill-bar close profit `ProfitPct`%, MACD histogram green→red during the trade (9/14/3), RSI cross up 70, stochastic %K cross up 80; holds overnight (`-strategy drop_bounce -interval Minutes15`, docs: `docs/drop_bounce/strategy.md`); no ticker literals yet, starter grid `data/params/drop_bounce/grid_example.json`.
```

- [ ] **Step 6: Full gate**

Run: `gofmt -l internal cmd pkg` (expected: no output), then `./bin/mage ci`
Expected: lint clean, all tests pass, no mock drift. Fix any lint finding in the files this plan touched.

- [ ] **Step 7: Commit**

```bash
git add internal/service/trading_strategy/drop_bounce data/params/drop_bounce docs/drop_bounce CLAUDE.md
git commit -m "test(drop_bounce): engine fills; docs and starter grid for the 15m rework"
```
