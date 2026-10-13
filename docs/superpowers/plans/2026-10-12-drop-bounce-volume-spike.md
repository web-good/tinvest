# drop_bounce volume-spike rework Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace the drop_bounce rules with: red bar whose volume ≥ `VolDailyMult` × Wilder average weekday daily volume → buy next open; exit on RSI(6) crossing 70 upward or a `StopDailyATR` daily-ATR stop.

**Architecture:** The backtest engine starts exposing completed daily volumes (`MarketData.DailyVolumes`) alongside the daily closes/highs/lows/times it already exposes. The drop_bounce core is rewritten around them; the old ticker packages are archived in git and deleted, the registry mechanism stays with an empty map.

**Tech Stack:** Go 1.25, `go test`, repo backtest engine (`internal/domain/backtest`), `pkg/indicators`.

**Spec:** `docs/superpowers/specs/2026-10-12-drop-bounce-volume-spike-design.md`

## Global Constraints

- Params defaults exactly: `VolDailyMult 0.4`, `VolAvgPeriod 14`, `RSIPeriod 6`, `RSIExit 70`, `StopDailyATR 1.0`, `DailyATRPeriod 14`, `StartHour 8`.
- All Params fields are `int` or `float64` (reflection grid calibration).
- `VolAvg`: weekday (Mon–Fri MSK) completed dailies, zero-volume days dropped, Wilder (seed = SMA of first `VolAvgPeriod`, then `avg += (v−avg)/N`); fewer than `VolAvgPeriod` values → unknown → no entry.
- Thresholds compare with a `1e-9` epsilon (exact threshold passes), like the rest of the core.
- Accepted intervals stay Minutes5, Minutes15, Minutes30, Hour1; others rejected.
- Backtest-only; no live code.
- `go build ./...` fails on `magefiles`; build with `go build ./internal/... ./pkg/... ./cmd/...`.
- Do not run backtests in this plan (CPU rule: they need a built binary + `taskset -c 0` + `GOMAXPROCS=1`); unit tests only.
- Commit messages end with `Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>`.
- Leave `docs/ideas/*.md` (untracked, unrelated) alone — never `git add -A`.

## Review Focus

- Daily volumes misaligned with `DailyTimes` (different lengths) → no entry, never a panic or a guess.
- A day with zero daily volume (holiday print, missing data) → excluded from the average, so the threshold is not lowered.
- Weekend dailies with tiny volume → excluded; only weekday volume defines "an average day".
- Signal on the last bar of the evening session (fill would be the next morning) → no entry; and a missing next slot → engine drops the signal.
- RSI cross on the fill bar itself, below the entry → sells (no deferral any more).

Each of these is pinned by a test in Task 3 or Task 4.

---

### Task 1: Archive the old calibration state

**Files:**
- Commit (no edits): `CLAUDE.md`, `data/params/drop_bounce/grid_example.json`, `docs/drop_bounce/strategy.md`, `internal/service/backtest/drop_bounce_registry.go`, `internal/service/backtest/drop_bounce_registry_test.go`, `internal/service/trading_strategy/drop_bounce/strategy/` (all), `data/params/drop_bounce/{domrf,gazp,ivat,t,wush}/`, `.claude/skills/drop-bounce-calibrate/`

**Interfaces:** none.

Known: in this state `go test ./internal/service/trading_strategy/drop_bounce/strategy/tbank` fails (`TestLiteralsArePinned`: T Minutes15 literal differs from its snapshot). The commit is an archive of the working tree as-is; do not fix it — the package is deleted in Task 2.

- [ ] **Step 1: Drop the Python cache from the skill dir**

```bash
rm -rf .claude/skills/drop-bounce-calibrate/scripts/__pycache__
```

- [ ] **Step 2: Stage exactly the drop_bounce paths**

```bash
git add CLAUDE.md data/params/drop_bounce internal/service/backtest/drop_bounce_registry.go \
  internal/service/backtest/drop_bounce_registry_test.go \
  internal/service/trading_strategy/drop_bounce docs/drop_bounce .claude/skills/drop-bounce-calibrate
git status --short
```
Expected: `docs/ideas/2026-10-08-short-hold-reversal-ideas.md` and `docs/ideas/2026-10-12-osengine-ideas.md` still `??` (unstaged); nothing else unstaged.

- [ ] **Step 3: Commit**

```bash
git commit -m "chore(drop_bounce): archive the 2026-10-11/12 calibration state

Ticker packages GAZP/DOMRF/T/IVAT/WUSH, their grids and results, and the
calibration skill as they stood before the volume-spike rework. None passed
walk-forward. The T literal snapshot test is red in this state.

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

---

### Task 2: Delete the ticker packages, empty the registry

**Files:**
- Delete: `internal/service/trading_strategy/drop_bounce/strategy/{domrf,gazp,ivat,tbank,wush}/`
- Delete: `data/params/drop_bounce/{domrf,gazp,ivat,t,wush}/`
- Modify: `internal/service/backtest/drop_bounce_registry.go`
- Modify: `internal/service/backtest/drop_bounce_registry_test.go`

**Interfaces:**
- Produces: `dropBounceRegistry` is an empty `map[string]dropBounceLiterals`; `DropBounceLookupOrGeneric(ticker, interval)` unchanged in signature.

- [ ] **Step 1: Delete packages and grids**

```bash
git rm -rq internal/service/trading_strategy/drop_bounce/strategy/{domrf,gazp,ivat,tbank,wush} \
  data/params/drop_bounce/{domrf,gazp,ivat,t,wush}
```

- [ ] **Step 2: Empty the registry**

In `internal/service/backtest/drop_bounce_registry.go` remove the five ticker imports and replace the registry declaration with:

```go
// dropBounceRegistry maps tickers with their own package to their literals, one per timeframe:
// a literal is tuned for its own timeframe and never leaks into another. Empty since the
// 2026-10-12 volume-spike rework: the packages calibrated on the old rules were deleted, and a
// ticker comes back here only after a calibration on the new ones.
var dropBounceRegistry = map[string]dropBounceLiterals{}
```

Imports left: `encoding/json`, `fmt`, `tinvest/internal/enum`, `.../drop_bounce/strategy/core`, `.../scalping/strategy`.

- [ ] **Step 3: Update the registry test**

In `internal/service/backtest/drop_bounce_registry_test.go`: remove the five ticker imports and delete `TestDropBounceTickerLiteralPerTimeframe` entirely (and any other test referencing a ticker package). Add:

```go
// No ticker has a package since the 2026-10-12 rework: every ticker on every supported timeframe
// runs on the core defaults.
func TestDropBounceEveryTimeframeUsesCoreDefaults(t *testing.T) {
	for _, iv := range []enum.Interval{enum.Minutes5, enum.Minutes15, enum.Minutes30, enum.Hour1} {
		b, err := DropBounceLookupOrGeneric("GAZP", iv)
		if err != nil {
			t.Fatalf("%s: %v", iv, err)
		}
		if got, ok := b.DefaultParams().(core.Params); !ok || got != core.DefaultParams() {
			t.Fatalf("%s: defaults = %#v, want core.DefaultParams()", iv, b.DefaultParams())
		}
	}
}
```

- [ ] **Step 4: Run tests**

Run: `go test ./internal/service/backtest/ ./internal/service/trading_strategy/drop_bounce/...`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/service/backtest/drop_bounce_registry.go internal/service/backtest/drop_bounce_registry_test.go
git commit -m "refactor(drop_bounce): delete ticker packages ahead of the rework

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

---

### Task 3: Engine exposes completed daily volumes

**Files:**
- Modify: `internal/service/trading_strategy/scalping/strategy/strategy.go` (the `MarketData` struct, after `DailyTimes`)
- Modify: `internal/domain/backtest/engine.go` (`visibleDaily`, `dailyCursor`, `newDailyCursor`, `dailyCursor.visible`, Run/Trace per-bar assembly, `AssembleMarketDataWithHTFInterval`)
- Test: `internal/domain/backtest/engine_test.go`

**Interfaces:**
- Produces: `strategy.MarketData.DailyVolumes []int64` — oldest-first, index-aligned with `DailyCloses`/`DailyTimes`, completed days only.
- Produces: `visibleDaily(daily []Candle, t time.Time, loc *time.Location) (closes, highs, lows []float64, times []time.Time, volumes []int64)` and `(*dailyCursor).visible(t time.Time)` with the same five results.

- [ ] **Step 1: Write the failing tests**

In `engine_test.go`:

`TestVisibleDaily`: give the `day` helper `Volume: int64(d) * 100`, and replace the first call and its checks' prelude with:

```go
	closes, highs, lows, times, volumes := visibleDaily(daily, t3, msk)
	if len(closes) != 2 || closes[0] != 1 || closes[1] != 2 {
		t.Fatalf("closes on Jan 3 = %v, want [1 2]", closes)
	}
	if len(highs) != 2 || len(lows) != 2 || len(times) != 2 || len(volumes) != 2 {
		t.Fatalf("series not index-aligned: %d closes, %d highs, %d lows, %d times, %d volumes",
			len(closes), len(highs), len(lows), len(times), len(volumes))
	}
	if volumes[0] != 100 || volumes[1] != 200 {
		t.Fatalf("volumes on Jan 3 = %v, want [100 200]", volumes)
	}
```
and the two later calls become `closes, _, _, times, _ := visibleDaily(...)`.

`TestRunPopulatesDailyTimes`: add `Volume: 500` / `Volume: 600` to its two dailies, capture `seenVols = md.DailyVolumes` in `decide`, and assert after `Run`:

```go
	if !reflect.DeepEqual(seenVols, []int64{500, 600}) {
		t.Fatalf("DailyVolumes on the last bar = %v, want [500 600]", seenVols)
	}
```
(declare `var seenVols []int64`; import `reflect` is already in the file.)

Cursor-equivalence test (the one at ~line 764): give its dailies `Volume` values (e.g. `Volume: 1000`, `2000`, `3000`, `4000`), then

```go
		gotC, gotH, gotL, gotT, gotV := cur.visible(at)
		wantC, wantH, wantL, wantT, wantV := visibleDaily(daily, at, msk)
		if !reflect.DeepEqual(gotC, wantC) || !reflect.DeepEqual(gotH, wantH) ||
			!reflect.DeepEqual(gotL, wantL) || !reflect.DeepEqual(gotT, wantT) || !reflect.DeepEqual(gotV, wantV) {
			t.Fatalf("t=%v: курсор дал (%v %v %v %v %v), visibleDaily — (%v %v %v %v %v)",
				at, gotC, gotH, gotL, gotT, gotV, wantC, wantH, wantL, wantT, wantV)
		}
```

`TestDailyCursorNilForEmptyDailies`: `if c, h, l, ts, v := cur.visible(time.Now()); c != nil || h != nil || l != nil || ts != nil || v != nil {` with the message listing five values.

`TestDailyCursorPrefixIsAppendSafe`: the two calls get one more `_`. Add a volume variant at the end:

```go
	vols := []Candle{
		{Time: time.Date(2026, 1, 5, 0, 0, 0, 0, msk), Volume: 10},
		{Time: time.Date(2026, 1, 6, 0, 0, 0, 0, msk), Volume: 12},
	}
	vc := newDailyCursor(vols, msk)
	_, _, _, _, v := vc.visible(time.Date(2026, 1, 6, 12, 0, 0, 0, msk))
	_ = append(v, 999) //nolint:gocritic // намеренно: append не должен портить общий массив
	if _, _, _, _, got := vc.visible(time.Date(2026, 1, 7, 12, 0, 0, 0, msk)); len(got) != 2 || got[1] != 12 {
		t.Fatalf("volumes = %v, want [10 12] — append испортил общий массив", got)
	}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/domain/backtest/`
Expected: build failure — `assignment mismatch: 5 variables but visibleDaily returns 4 values` and `md.DailyVolumes undefined`.

- [ ] **Step 3: Implement**

`strategy.go`, after the `DailyTimes` field:

```go
	// DailyVolumes are the volumes (lots) of the same COMPLETED daily candles as DailyCloses
	// (aligned index-for-index). Empty when no daily data is supplied. Consumers must degrade
	// gracefully when it is empty or not length-aligned with DailyTimes.
	DailyVolumes []int64
```

`engine.go`:

```go
// visibleDaily returns the closes, highs, lows, open-times and volumes of daily candles whose
// calendar day (in loc) is strictly before t's calendar day — i.e. days that have fully
// closed by t. This is the no-lookahead rule: the current, still-forming day is never
// visible. The five series are index-aligned, oldest-first.
func visibleDaily(daily []Candle, t time.Time, loc *time.Location) (closes, highs, lows []float64, times []time.Time, volumes []int64) {
	bound := startOfDay(t, loc)
	for _, c := range daily {
		if c.Time.Before(bound) {
			closes = append(closes, c.Close)
			highs = append(highs, c.High)
			lows = append(lows, c.Low)
			times = append(times, c.Time)
			volumes = append(volumes, c.Volume)
		}
	}
	return closes, highs, lows, times, volumes
}
```

In `dailyCursor` add `volumes []int64` after `times`; update its doc "the same four series" → "the same five series". In `newDailyCursor` add `volumes: make([]int64, len(daily)),` and in the loop `c.volumes[i] = d.Volume`. `visible`:

```go
// visible returns exactly what visibleDaily(daily, t, loc) would, provided t is
// non-decreasing across calls. Returns five nils for a nil cursor or an empty prefix,
// matching visibleDaily.
func (d *dailyCursor) visible(t time.Time) (closes, highs, lows []float64, times []time.Time, volumes []int64) {
	if d == nil {
		return nil, nil, nil, nil, nil
	}
	bound := startOfDay(t, d.loc)
	for d.idx < len(d.daily) && d.daily[d.idx].Time.Before(bound) {
		d.idx++
	}
	if d.idx == 0 {
		return nil, nil, nil, nil, nil
	}
	// Full slice expressions for the same reason as in htfCursor.visible: a consumer that
	// appends to the returned series must reallocate instead of scribbling on the shared
	// backing arrays that later bars still read.
	return d.closes[:d.idx:d.idx], d.highs[:d.idx:d.idx], d.lows[:d.idx:d.idx], d.times[:d.idx:d.idx], d.volumes[:d.idx:d.idx]
}
```

The three assembly sites (Run ~233, Trace ~321, `AssembleMarketDataWithHTFInterval` ~423) become:

```go
		md.DailyCloses, md.DailyHighs, md.DailyLows, md.DailyTimes, md.DailyVolumes = daily.visible(candles[i].Time)
```
```go
	md.DailyCloses, md.DailyHighs, md.DailyLows, md.DailyTimes, md.DailyVolumes = visibleDaily(daily, cur, mskLoc)
```

- [ ] **Step 4: Run tests**

Run: `go test ./internal/domain/backtest/ && go build ./internal/... ./pkg/... ./cmd/...`
Expected: PASS, build OK (no other package calls the unexported helpers).

- [ ] **Step 5: Commit**

```bash
git add internal/service/trading_strategy/scalping/strategy/strategy.go internal/domain/backtest/engine.go internal/domain/backtest/engine_test.go
git commit -m "feat(backtest): expose completed daily volumes in MarketData

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

---

### Task 4: Rewrite the drop_bounce core

**Files:**
- Rewrite: `internal/service/trading_strategy/drop_bounce/strategy/core/core.go`
- Rewrite: `internal/service/trading_strategy/drop_bounce/strategy/core/core_test.go` (helpers + entry tests)
- Rewrite: `internal/service/trading_strategy/drop_bounce/strategy/core/exits_test.go`
- Rewrite: `internal/service/trading_strategy/drop_bounce/strategy/core/engine_test.go`
- Modify: `internal/service/backtest/drop_bounce_registry_test.go` (`TestDropBounceParseParamsLayersOverDefaults`)

**Interfaces:**
- Consumes: `strategy.MarketData.DailyVolumes []int64` (Task 3).
- Produces: `core.Params{VolDailyMult float64; VolAvgPeriod int; RSIPeriod int; RSIExit float64; StopDailyATR float64; DailyATRPeriod int; StartHour int}`, `core.DefaultParams() Params`, `core.NewWithParams(ticker string, step time.Duration, p Params) *Strategy` (unchanged signature).

- [ ] **Step 1: Write the failing tests — `core_test.go`**

Replace the whole file with:

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

// step15 is the timeframe the fixtures are built on unless a test names another.
const step15 = 15 * time.Minute

// path builds consecutive 15m bars from start (MSK): each bar opens at the previous close (the
// first at its own close) and its high/low span the open and the close.
func path(start string, closes []float64) []bar {
	t0 := at(start)
	out := make([]bar, len(closes))
	prev := closes[0]
	for i, c := range closes {
		out[i] = bar{t0.Add(time.Duration(i) * step15), prev, max(prev, c), min(prev, c), c}
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

// fixtureATR is the daily ATR every fixture carries: its dailies range exactly this much.
const fixtureATR = 2.0

// fixtureDayVol is the volume of every fixture daily, so the average daily volume is exactly it.
const fixtureDayVol = 1000

// quietVol is the volume of every fixture bar a test does not set: far below any threshold.
const quietVol = 10

// fixture builds the market data over bars, every bar trading quietVol, with 20 weekday dailies
// of range fixtureATR and volume fixtureDayVol before the first bar.
func fixture(bars []bar) strategy.MarketData {
	var md strategy.MarketData
	for _, b := range bars {
		md.Times = append(md.Times, b.t)
		md.Opens = append(md.Opens, b.o)
		md.Highs = append(md.Highs, b.h)
		md.Lows = append(md.Lows, b.l)
		md.Closes = append(md.Closes, b.c)
		md.Volumes = append(md.Volumes, quietVol)
	}
	md.Price = md.Closes[len(md.Closes)-1]
	withDailies(&md, dailies(bars[0].t, 20, fixtureATR))
	return md
}

// dailies returns n completed weekday daily bars ending the weekday before `before`, each
// spanning `rng` around a close of 100, so their ATR is exactly rng.
func dailies(before time.Time, n int, rng float64) []bar {
	d := time.Date(before.In(mskLoc).Year(), before.In(mskLoc).Month(), before.In(mskLoc).Day(), 0, 0, 0, 0, mskLoc)
	out := make([]bar, 0, n)
	for len(out) < n {
		d = d.AddDate(0, 0, -1)
		if wd := d.Weekday(); wd == time.Saturday || wd == time.Sunday {
			continue
		}
		out = append([]bar{{d, 100, 100 + rng/2, 100 - rng/2, 100}}, out...)
	}
	return out
}

// withDailies replaces the daily series of md with ds, each day trading fixtureDayVol.
func withDailies(md *strategy.MarketData, ds []bar) {
	md.DailyHighs, md.DailyLows, md.DailyCloses, md.DailyTimes, md.DailyVolumes = nil, nil, nil, nil, nil
	for _, d := range ds {
		md.DailyTimes = append(md.DailyTimes, d.t)
		md.DailyHighs = append(md.DailyHighs, d.h)
		md.DailyLows = append(md.DailyLows, d.l)
		md.DailyCloses = append(md.DailyCloses, d.c)
		md.DailyVolumes = append(md.DailyVolumes, fixtureDayVol)
	}
}

// redAt is 30 flat bars at 100 followed by a red bar 100 → 99.5 starting at t (MSK).
func redAt(t string) []bar {
	end := at(t)
	bars := path(end.Add(-30*step15).Format("2006-01-02 15:04"), flatN(30, 100))
	return append(bars, bar{end, 100, 100, 99.5, 99.5})
}

// spikeMD is the fixture over bars with the last bar trading vol.
func spikeMD(bars []bar, vol int64) strategy.MarketData {
	md := fixture(bars)
	md.Volumes[len(md.Volumes)-1] = vol
	return md
}

// spike decides on bars whose last bar traded vol.
func spike(p Params, bars []bar, vol int64) model.Signal {
	return NewWithParams("TEST", step15, p).Decide(spikeMD(bars, vol))
}

// held decides with a position bought at entry on the bar starting at entryTime, with the daily
// ATR frozen at entry equal to fixtureATR.
func held(p Params, bars []bar, entry float64, entryTime time.Time) model.Signal {
	md := fixture(bars)
	md.Position = &strategy.Position{PurchasePrice: entry, Quantity: 1, EntryTime: entryTime, EntryATR: fixtureATR}
	return NewWithParams("TEST", step15, p).Decide(md)
}

func TestDefaultParams(t *testing.T) {
	want := Params{VolDailyMult: 0.4, VolAvgPeriod: 14, RSIPeriod: 6, RSIExit: 70, StopDailyATR: 1.0, DailyATRPeriod: 14, StartHour: 8}
	if got := DefaultParams(); got != want {
		t.Fatalf("DefaultParams() = %+v, want %+v", got, want)
	}
}

// Monday 2026-09-14 15:30: a red bar trading 0.4 of an average day buys at the next slot.
func TestEnterBuysRedBarOnVolumeSpike(t *testing.T) {
	bars := redAt("2026-09-14 15:30")
	sig := spike(DefaultParams(), bars, 400)
	if sig.Kind != model.SignalBuy {
		t.Fatalf("kind = %v, want Buy", sig.Kind)
	}
	if math.Abs(sig.ATR-fixtureATR) > eps {
		t.Fatalf("ATR = %v, want %v", sig.ATR, fixtureATR)
	}
	if want := at("2026-09-14 15:45"); !sig.FillBy.Equal(want) {
		t.Fatalf("FillBy = %v, want %v", sig.FillBy, want)
	}
	if !strings.HasPrefix(sig.EntryReason, "VOLUME:") || !strings.Contains(sig.EntryReason, "0.40×") {
		t.Fatalf("entry reason = %q, want VOLUME with the 0.40× ratio", sig.EntryReason)
	}
}

func TestEnterVolumeThreshold(t *testing.T) {
	bars := redAt("2026-09-14 15:30")
	if sig := spike(DefaultParams(), bars, 399); sig.Kind != model.SignalNone {
		t.Fatalf("volume 399 < 0.4×1000: kind = %v, want None", sig.Kind)
	}
	if sig := spike(DefaultParams(), bars, 400); sig.Kind != model.SignalBuy {
		t.Fatalf("volume exactly at the threshold: kind = %v, want Buy", sig.Kind)
	}
	p := DefaultParams()
	p.VolDailyMult = 0
	if sig := spike(p, bars, 1_000_000); sig.Kind != model.SignalNone {
		t.Fatalf("VolDailyMult 0: kind = %v, want None", sig.Kind)
	}
}

func TestEnterNeedsRedBar(t *testing.T) {
	bars := redAt("2026-09-14 15:30")
	last := len(bars) - 1
	bars[last] = bar{bars[last].t, 100, 100.5, 100, 100.5} // green
	if sig := spike(DefaultParams(), bars, 1000); sig.Kind != model.SignalNone {
		t.Fatalf("green bar: kind = %v, want None", sig.Kind)
	}
	bars[last] = bar{bars[last].t, 100, 100, 100, 100} // doji
	if sig := spike(DefaultParams(), bars, 1000); sig.Kind != model.SignalNone {
		t.Fatalf("doji: kind = %v, want None", sig.Kind)
	}
}

// Volume is the signal itself, so without a known daily average there is no entry.
func TestEnterNeedsAverageDailyVolume(t *testing.T) {
	bars := redAt("2026-09-14 15:30")
	s := NewWithParams("TEST", step15, DefaultParams())

	md := spikeMD(bars, 1000)
	md.DailyVolumes = nil
	if sig := s.Decide(md); sig.Kind != model.SignalNone {
		t.Fatalf("no daily volumes: kind = %v, want None", sig.Kind)
	}
	md = spikeMD(bars, 1000)
	md.DailyVolumes = md.DailyVolumes[1:]
	if sig := s.Decide(md); sig.Kind != model.SignalNone {
		t.Fatalf("misaligned daily volumes: kind = %v, want None", sig.Kind)
	}

	// The stop is off so that only the volume history matters (the ATR needs 15 dailies).
	p := DefaultParams()
	p.StopDailyATR = 0
	md = spikeMD(bars, 1000)
	withDailies(&md, dailies(bars[0].t, 13, fixtureATR))
	if sig := NewWithParams("TEST", step15, p).Decide(md); sig.Kind != model.SignalNone {
		t.Fatalf("13 days of volume < VolAvgPeriod 14: kind = %v, want None", sig.Kind)
	}
	withDailies(&md, dailies(bars[0].t, 14, fixtureATR))
	if sig := NewWithParams("TEST", step15, p).Decide(md); sig.Kind != model.SignalBuy {
		t.Fatalf("14 days of volume: kind = %v, want Buy", sig.Kind)
	}
}

// Weekend and zero-volume days stay out of the Wilder average.
func TestAvgDailyVolumeWilder(t *testing.T) {
	md := strategy.MarketData{
		DailyTimes: []time.Time{
			at("2026-09-07 00:00"), // Mon 100
			at("2026-09-08 00:00"), // Tue 0 — no trading, dropped
			at("2026-09-09 00:00"), // Wed 200
			at("2026-09-10 00:00"), // Thu 300
			at("2026-09-12 00:00"), // Sat 1e6 — weekend, dropped
			at("2026-09-14 00:00"), // Mon 400
		},
		DailyVolumes: []int64{100, 0, 200, 300, 1_000_000, 400},
	}
	p := DefaultParams()
	p.VolAvgPeriod = 3
	// seed SMA(100, 200, 300) = 200, then 200 + (400 − 200)/3
	want := 200 + 200.0/3
	if got := NewWithParams("TEST", step15, p).avgDailyVolume(md); math.Abs(got-want) > 1e-9 {
		t.Fatalf("avgDailyVolume = %v, want %v", got, want)
	}
	p.VolAvgPeriod = 5 // only four usable days
	if got := NewWithParams("TEST", step15, p).avgDailyVolume(md); got != 0 {
		t.Fatalf("too few days: avgDailyVolume = %v, want 0", got)
	}
}

func TestEnterWindow(t *testing.T) {
	cases := []struct {
		at   string
		want model.SignalKind
	}{
		{"2026-09-14 07:45", model.SignalNone}, // before StartHour 8
		{"2026-09-14 08:00", model.SignalBuy},
		{"2026-09-19 12:00", model.SignalNone}, // Saturday
		{"2026-09-14 23:30", model.SignalBuy},  // fill at 23:45, same date
		{"2026-09-14 23:45", model.SignalNone}, // fill would be tomorrow
	}
	for _, c := range cases {
		if sig := spike(DefaultParams(), redAt(c.at), 1000); sig.Kind != c.want {
			t.Fatalf("%s: kind = %v, want %v", c.at, sig.Kind, c.want)
		}
	}
}

func TestEnterStopGuards(t *testing.T) {
	bars := redAt("2026-09-14 15:30")
	md := spikeMD(bars, 1000)
	md.DailyHighs = nil // the daily ATR becomes unknown, the daily volumes stay
	if sig := NewWithParams("TEST", step15, DefaultParams()).Decide(md); sig.Kind != model.SignalNone {
		t.Fatalf("stop on, ATR unknown: kind = %v, want None", sig.Kind)
	}
	p := DefaultParams()
	p.StopDailyATR = 0
	if sig := NewWithParams("TEST", step15, p).Decide(md); sig.Kind != model.SignalBuy {
		t.Fatalf("stop off, ATR unknown: kind = %v, want Buy", sig.Kind)
	}
	p.StopDailyATR = 50 // 99.5 − 50×2 < 0
	if sig := spike(p, bars, 1000); sig.Kind != model.SignalNone {
		t.Fatalf("stop below zero: kind = %v, want None", sig.Kind)
	}
}

// The step only sets the fill slot.
func TestEnterOnOtherSteps(t *testing.T) {
	sig := NewWithParams("TEST", 5*time.Minute, DefaultParams()).Decide(spikeMD(redAt("2026-09-14 15:30"), 1000))
	if want := at("2026-09-14 15:35"); sig.Kind != model.SignalBuy || !sig.FillBy.Equal(want) {
		t.Fatalf("5m: kind %v FillBy %v, want Buy by %v", sig.Kind, sig.FillBy, want)
	}
}

func TestLookback(t *testing.T) {
	if got := NewWithParams("TEST", step15, DefaultParams()).Lookback(); got != 64 {
		t.Fatalf("default Lookback = %d, want 64", got)
	}
	p := DefaultParams()
	p.RSIPeriod = 20
	if got := NewWithParams("TEST", step15, p).Lookback(); got != 82 {
		t.Fatalf("RSIPeriod 20: Lookback = %d, want 82", got)
	}
}
```

- [ ] **Step 2: Write the failing tests — `exits_test.go`**

Replace the whole file with:

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

// position is a held long bought at entry on the bar starting at entryTime with the frozen ATR.
func position(entry float64, entryTime time.Time, atr float64) *strategy.Position {
	return &strategy.Position{PurchasePrice: entry, Quantity: 1, EntryTime: entryTime, EntryATR: atr}
}

// stopOnly is the default params with the RSI exit off.
func stopOnly() Params {
	p := DefaultParams()
	p.RSIExit = 0
	return p
}

// flatWithLast is 30 flat bars at 100 and a last bar with the given open/high/low/close.
func flatWithLast(o, h, l, c float64) []bar {
	bars := path("2026-09-14 08:00", flatN(30, 100))
	last := bars[len(bars)-1]
	return append(bars, bar{last.t.Add(step15), o, h, l, c})
}

// The default stop is 1.0 daily ATR (fixtureATR 2) below the fill: 98 for a fill at 100.
func TestStopFromFillPrice(t *testing.T) {
	bars := flatWithLast(100, 100, 98.0, 99)
	sig := held(stopOnly(), bars, 100, bars[20].t)
	if sig.Kind != model.SignalSell || sig.Reason != "SL" || math.Abs(sig.StopLoss-98) > eps {
		t.Fatalf("got %v %q stop %v, want SL at 98", sig.Kind, sig.Reason, sig.StopLoss)
	}
	bars = flatWithLast(100, 100, 98.01, 99)
	if sig := held(stopOnly(), bars, 100, bars[20].t); sig.Kind != model.SignalNone {
		t.Fatalf("low 98.01: kind = %v, want None", sig.Kind)
	}
	p := stopOnly()
	p.StopDailyATR = 0
	bars = flatWithLast(100, 100, 50, 60)
	if sig := held(p, bars, 100, bars[20].t); sig.Kind != model.SignalNone {
		t.Fatalf("StopDailyATR 0: kind = %v, want None", sig.Kind)
	}
}

// Dailies that closed after the entry must not move the stop.
func TestStopUsesATRFrozenAtEntry(t *testing.T) {
	bars := flatWithLast(100, 100, 97.5, 99)
	md := fixture(bars)
	withDailies(&md, dailies(bars[0].t, 20, 4)) // today's ATR is 4: a live stop would sit at 96
	md.Position = position(100, bars[20].t, fixtureATR)
	sig := NewWithParams("TEST", step15, stopOnly()).Decide(md)
	if sig.Reason != "SL" || math.Abs(sig.StopLoss-98) > eps {
		t.Fatalf("got %q stop %v, want SL at 98 from the entry ATR", sig.Reason, sig.StopLoss)
	}
}

// A position without a frozen ATR keeps a stop from today's ATR rather than running naked.
func TestStopFallsBackToCurrentATR(t *testing.T) {
	bars := flatWithLast(100, 100, 98, 99)
	md := fixture(bars)
	md.Position = position(100, bars[20].t, 0)
	sig := NewWithParams("TEST", step15, stopOnly()).Decide(md)
	if sig.Reason != "SL" || math.Abs(sig.StopLoss-98) > eps {
		t.Fatalf("got %q stop %v, want SL at 98 from the current ATR", sig.Reason, sig.StopLoss)
	}
}

func TestCrossUp(t *testing.T) {
	if !crossUp(69, 70, 70) || !crossUp(10, 90, 70) {
		t.Fatal("a move from below to at/above the level must cross")
	}
	if crossUp(70, 75, 70) || crossUp(60, 69.9, 70) || crossUp(80, 60, 70) {
		t.Fatal("no cross from at/above, below-to-below or downward")
	}
}

// slideAndRise is 30 flat bars at 100, 10 bars sliding 0.5% each (RSI(6) near 0), then 15 bars
// rising 0.3% each, which carry RSI(6) up through 70.
func slideAndRise() []bar {
	closes := append(flatN(30, 100), steps(100, 10, -0.5)...)
	closes = append(closes, steps(closes[len(closes)-1], 15, 0.3)...)
	return path("2026-09-14 08:00", closes)
}

// firstCross returns the first bar index ≥ from whose RSI(period) crosses level upward, or −1.
func firstCross(bars []bar, period int, level float64, from int) int {
	closes := make([]float64, len(bars))
	for i, b := range bars {
		closes[i] = b.c
	}
	rsi := indicators.RSISeries(closes, period)
	for k := max(from, period+1); k < len(bars); k++ {
		if crossUp(rsi[k-1], rsi[k], level) {
			return k
		}
	}
	return -1
}

// The RSI exit sells at the cross whatever the price: here far below the entry.
func TestRSIExitSellsAtTheCross(t *testing.T) {
	bars := slideAndRise()
	k := firstCross(bars, 6, 70, 40)
	if k < 0 {
		t.Fatal("fixture: RSI(6) never crosses 70 on the rise")
	}
	p := DefaultParams()
	p.StopDailyATR = 0
	if sig := held(p, bars[:k], 200, bars[30].t); sig.Kind != model.SignalNone {
		t.Fatalf("bar before the cross: kind = %v %q, want None", sig.Kind, sig.Reason)
	}
	if sig := held(p, bars[:k+1], 200, bars[30].t); sig.Kind != model.SignalSell || sig.Reason != "RSI" {
		t.Fatalf("cross bar: got %v %q, want Sell RSI below the entry", sig.Kind, sig.Reason)
	}
	p.RSIExit = 0
	if sig := held(p, bars[:k+1], 200, bars[30].t); sig.Kind != model.SignalNone {
		t.Fatalf("RSIExit 0: kind = %v, want None", sig.Kind)
	}
}

// On a bar that both crosses RSI 70 and touches the stop, the stop wins.
func TestStopBeatsRSIExit(t *testing.T) {
	bars := slideAndRise()
	k := firstCross(bars, 6, 70, 40)
	if k < 0 {
		t.Fatal("fixture: RSI(6) never crosses 70 on the rise")
	}
	bars = bars[:k+1]
	entry := bars[k].c
	bars[k].l = entry - 2.5 // stop = entry − 1.0 × fixtureATR
	sig := held(DefaultParams(), bars, entry, bars[30].t)
	if sig.Reason != "SL" || math.Abs(sig.StopLoss-(entry-2)) > eps {
		t.Fatalf("got %q stop %v, want SL at %v", sig.Reason, sig.StopLoss, entry-2)
	}
}

// Positions carry overnight: nothing sells the last evening bar on its own.
func TestNoEODExit(t *testing.T) {
	bars := path("2026-09-14 16:15", flatN(31, 100)) // last bar 23:45
	if sig := held(DefaultParams(), bars, 100, bars[20].t); sig.Kind != model.SignalNone {
		t.Fatalf("23:45 flat: kind = %v %q, want None", sig.Kind, sig.Reason)
	}
}
```

- [ ] **Step 3: Write the failing tests — core `engine_test.go`**

Replace the whole file with:

```go
// engine_test.go runs drop_bounce through the real backtest engine to pin what it relies on: daily
// volumes reaching the strategy, entry at the next bar's open, the daily-ATR stop filled at the gap
// open overnight, the RSI exit, and a signal dropped when the next slot is missing.
package core

import (
	"math"
	"testing"
	"time"

	bt "tinvest/internal/domain/backtest"
)

// toCandles converts bars to candles that all trade vol.
func toCandles(bars []bar, vol int64) []bt.Candle {
	out := make([]bt.Candle, len(bars))
	for i, b := range bars {
		out[i] = bt.Candle{Time: b.t, Open: b.o, High: b.h, Low: b.l, Close: b.c, Volume: vol}
	}
	return out
}

// spikeBars is Lookback flat bars at 100 then a red bar 100 → 99.5 starting at t: the engine
// decides only from bar Lookback()−1 on, so the lead puts the signal past it.
func spikeBars(t string) []bar {
	n := NewWithParams("TEST", step15, DefaultParams()).Lookback()
	end := at(t)
	bars := path(end.Add(-time.Duration(n)*step15).Format("2006-01-02 15:04"), flatN(n, 100))
	return append(bars, bar{end, 100, 100, 99.5, 99.5})
}

// runEngine runs the default drop_bounce over bars where the bar at spike trades 0.5 of an average
// day (dailies of range fixtureATR and volume fixtureDayVol before the first bar).
func runEngine(bars []bar, spike int) bt.Result {
	candles := toCandles(bars, quietVol)
	candles[spike].Volume = fixtureDayVol / 2
	return bt.Run(NewWithParams("TEST", step15, DefaultParams()), candles,
		toCandles(dailies(bars[0].t, 20, fixtureATR), fixtureDayVol), nil,
		bt.Config{InitialCash: 1e6, Fraction: 1, Commission: 0, Lot: 1, EntryAtNextOpen: true})
}

func TestEngineOvernightGapStop(t *testing.T) {
	bars := spikeBars("2026-09-14 13:00")
	sig := len(bars) - 1
	f := 99.7 // the fill bar opens away from the signal close 99.5
	gap := f * 0.97
	bars = append(bars,
		bar{at("2026-09-14 13:15"), f, f, f, f},
		bar{at("2026-09-15 07:00"), gap, gap, gap * 0.999, gap},
	)
	res := runEngine(bars, sig)
	if len(res.Trades) != 1 {
		t.Fatalf("trades = %d, want 1: %+v", len(res.Trades), res.Trades)
	}
	tr := res.Trades[0]
	if math.Abs(tr.EntryPrice-f) > eps || !tr.EntryTime.Equal(at("2026-09-14 13:15")) {
		t.Fatalf("entry %v @ %v, want %v @ 13:15 (next bar's open)", tr.EntryPrice, tr.EntryTime, f)
	}
	if stop := f - fixtureATR; gap >= stop {
		t.Fatalf("fixture: gap open %v must be below the stop %v", gap, stop)
	}
	if tr.Reason != "SL" || math.Abs(tr.ExitPrice-gap) > 1e-6 || !tr.ExitTime.Equal(at("2026-09-15 07:00")) {
		t.Fatalf("exit %q @ %v %v, want SL at the gap open %v next morning", tr.Reason, tr.ExitPrice, tr.ExitTime, gap)
	}
}

func TestEngineRSIExit(t *testing.T) {
	bars := spikeBars("2026-09-14 13:00")
	sig := len(bars) - 1
	bars = append(bars, path("2026-09-14 13:15", steps(99.7, 15, 0.3))...)
	k := firstCross(bars, 6, 70, sig+1)
	if k < 0 {
		t.Fatal("fixture: RSI(6) never crosses 70 after the fill")
	}
	res := runEngine(bars, sig)
	if len(res.Trades) != 1 {
		t.Fatalf("trades = %d, want 1: %+v", len(res.Trades), res.Trades)
	}
	if tr := res.Trades[0]; tr.Reason != "RSI" || !tr.ExitTime.Equal(bars[k].t) {
		t.Fatalf("exit %q @ %v, want RSI at %v", tr.Reason, tr.ExitTime, bars[k].t)
	}
}

// Without daily volumes the engine-fed strategy has no average and never enters.
func TestEngineNoDailyVolumeNoTrade(t *testing.T) {
	bars := append(spikeBars("2026-09-14 13:00"), bar{at("2026-09-14 13:15"), 99.7, 99.7, 99.7, 99.7})
	candles := toCandles(bars, fixtureDayVol) // every bar trades a whole average day — if one were known
	res := bt.Run(NewWithParams("TEST", step15, DefaultParams()), candles,
		toCandles(dailies(bars[0].t, 20, fixtureATR), 0), nil,
		bt.Config{InitialCash: 1e6, Fraction: 1, Commission: 0, Lot: 1, EntryAtNextOpen: true})
	if len(res.Trades) != 0 || res.BarsInMarket != 0 {
		t.Fatalf("trades = %d, barsInMarket = %d, want no position", len(res.Trades), res.BarsInMarket)
	}
}

// A signal on the 23:30 bar may fill only at 23:45; when that candle is missing the next candle
// is the next morning and the signal is dropped instead of carrying over.
func TestEngineDropsSignalWhenNextSlotMissing(t *testing.T) {
	bars := spikeBars("2026-09-14 23:30")
	sig := len(bars) - 1
	bars = append(bars,
		bar{at("2026-09-15 07:00"), 99.5, 99.5, 99.5, 99.5},
		bar{at("2026-09-15 07:15"), 99.5, 99.5, 99.5, 99.5},
	)
	res := runEngine(bars, sig)
	if len(res.Trades) != 0 || res.BarsInMarket != 0 {
		t.Fatalf("trades = %d, barsInMarket = %d, want no position: %+v", len(res.Trades), res.BarsInMarket, res.Trades)
	}
}
```

Note on `TestEngineRSIExit`: `bars[k]` with `k ≥ sig+1` is the fill bar or later; if `k` is the fill bar itself the exit is on the fill bar's close — still RSI at `bars[k].t`. The stop is `99.999.. − 2`, far below every low of the rise.

- [ ] **Step 4: Update the registry parse test**

In `internal/service/backtest/drop_bounce_registry_test.go`, `TestDropBounceParseParamsLayersOverDefaults`:

```go
	parsed, err := b.ParseParams([]byte(`{"VolDailyMult": 0.15}`))
	if err != nil {
		t.Fatalf("ParseParams: %v", err)
	}
	got := parsed.(core.Params)
	if got.VolDailyMult != 0.15 || got.StopDailyATR != core.DefaultParams().StopDailyATR {
		t.Fatalf("got %+v", got)
	}
```

- [ ] **Step 5: Run tests to verify they fail**

Run: `go test ./internal/service/trading_strategy/drop_bounce/... ./internal/service/backtest/`
Expected: build failure — `unknown field VolDailyMult in struct literal`, `s.avgDailyVolume undefined`.

- [ ] **Step 6: Rewrite `core.go`**

Replace the whole file with:

```go
// Package core implements drop_bounce: a long-only bounce after a red bar that trades a sizeable
// share of an average day's volume. A bar closing below its open, whose volume reaches
// VolDailyMult times the average weekday daily volume (Wilder over VolAvgPeriod completed
// Mon–Fri dailies, zero-volume days left out), signals a buy; the backtest fills at the next
// bar's open (Config.EntryAtNextOpen). Signals are taken on Mon–Fri MSK bars starting at or after
// StartHour whose fill bar lies on the same date. The daily ATR (Wilder, DailyATRPeriod,
// completed weekday dailies) is taken at the signal and frozen for the trade. Exits, in order: a
// stop StopDailyATR daily ATRs below the fill price, then RSI(RSIPeriod) crossing RSIExit upward
// whatever the price. There is no end-of-day exit: positions carry overnight. Pure, stateless
// between bars and ticker-agnostic. The bar step (5m, 15m, 30m or 1h) is fixed at construction.
// Run with `-strategy drop_bounce -interval Minutes5|Minutes15|Minutes30|Hour1`.
package core

import (
	"fmt"
	"time"

	"tinvest/internal/service/trading_strategy/scalping/model"
	"tinvest/internal/service/trading_strategy/scalping/strategy"
	"tinvest/pkg/indicators"
)

// minLookback keeps 64 bars (16 hours of 15m) so the RSI converges.
const minLookback = 64

// Params holds every tunable. All fields are int or float64 so reflection grid calibration can
// sweep them.
type Params struct {
	// VolDailyMult is the share of the average weekday daily volume the signal bar must trade
	// (default 0.4); 0 = no entries.
	VolDailyMult float64
	// VolAvgPeriod is the Wilder length of the average over completed weekday dailies with
	// nonzero volume (default 14).
	VolAvgPeriod int
	RSIPeriod    int     // RSI length of the exit (default 6)
	RSIExit      float64 // sell when RSI crosses this upward; 0 = off (default 70)
	// StopDailyATR is the stop distance below the fill price in daily ATRs frozen at the signal;
	// 0 = off (default 1).
	StopDailyATR float64
	// DailyATRPeriod is the Wilder ATR length over completed weekday dailies (default 14).
	DailyATRPeriod int
	StartHour      int // signal bars must start at/after StartHour:00 MSK (default 8)
}

// DefaultParams returns the owner's initial configuration.
func DefaultParams() Params {
	return Params{
		VolDailyMult:   0.4,
		VolAvgPeriod:   14,
		RSIPeriod:      6,
		RSIExit:        70,
		StopDailyATR:   1,
		DailyATRPeriod: 14,
		StartHour:      8,
	}
}

// Strategy trades a single instrument with the drop_bounce rules. Ticker-agnostic and pure.
type Strategy struct {
	ticker string
	// step is the bar timeframe. It matters only for the entry: the fill bar starts one step after
	// the signal bar, must lie on the same MSK date, and bounds Signal.FillBy.
	step time.Duration
	p    Params
}

// NewWithParams returns the strategy for a ticker on bars of the given step with explicit params.
func NewWithParams(ticker string, step time.Duration, p Params) *Strategy {
	return &Strategy{ticker: ticker, step: step, p: p}
}

func (s *Strategy) Ticker() string { return s.ticker }

// Lookback is enough history for the Wilder RSI to converge plus the previous bar its cross
// compares against.
func (s *Strategy) Lookback() int { return max(minLookback, 4*s.p.RSIPeriod+2) }

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

// isWeekend reports whether t falls on Saturday or Sunday MSK.
func isWeekend(t time.Time) bool {
	wd := t.In(mskLoc).Weekday()
	return wd == time.Saturday || wd == time.Sunday
}

// weekdayDaily drops weekend (Sat/Sun MSK) bars from the daily series, keeping the three price
// slices aligned: MOEX weekend sessions are several times narrower and thinner than weekday ones
// and would understate the ATR. Without aligned times there is nothing to filter by and the
// series is returned untouched.
func weekdayDaily(highs, lows, closes []float64, times []time.Time) (h, l, c []float64) {
	n := len(closes)
	if n == 0 || len(highs) != n || len(lows) != n || len(times) != n {
		return highs, lows, closes
	}
	h, l, c = make([]float64, 0, n), make([]float64, 0, n), make([]float64, 0, n)
	for i := range n {
		if isWeekend(times[i]) {
			continue
		}
		h, l, c = append(h, highs[i]), append(l, lows[i]), append(c, closes[i])
	}
	return h, l, c
}

// dailyATR is Wilder's ATR over the completed weekday dailies the engine exposes (no lookahead),
// or 0 when the data cannot support it.
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

// avgDailyVolume is the Wilder average of the completed weekday daily volumes: weekend sessions
// and zero-volume days are left out (both would drag the yardstick down and fire the threshold on
// nothing), the first VolAvgPeriod values seed it as a plain mean, each later one moves it by
// (v − avg)/VolAvgPeriod. It is 0 — unknown — when the volumes are missing or not aligned with
// DailyTimes, or fewer than VolAvgPeriod days are left.
func (s *Strategy) avgDailyVolume(md strategy.MarketData) float64 {
	n := s.p.VolAvgPeriod
	if n <= 0 || len(md.DailyVolumes) == 0 || len(md.DailyVolumes) != len(md.DailyTimes) {
		return 0
	}
	vols := make([]float64, 0, len(md.DailyVolumes))
	for i, v := range md.DailyVolumes {
		if v > 0 && !isWeekend(md.DailyTimes[i]) {
			vols = append(vols, float64(v))
		}
	}
	if len(vols) < n {
		return 0
	}
	var avg float64
	for _, v := range vols[:n] {
		avg += v
	}
	avg /= float64(n)
	for _, v := range vols[n:] {
		avg += (v - avg) / float64(n)
	}
	return avg
}

// inWindow reports whether a signal on the bar starting at t may open a position: a weekday, at
// or after StartHour, and the fill bar (one step later) on the same MSK date.
func (s *Strategy) inWindow(t time.Time) bool {
	if isWeekend(t) || t.In(mskLoc).Hour() < s.p.StartHour {
		return false
	}
	return dayKey(t.Add(s.step)) == dayKey(t)
}

func (s *Strategy) Decide(md strategy.MarketData) model.Signal {
	sig := model.Signal{Ticker: s.ticker, Price: md.Price}
	if md.Position != nil {
		return s.manage(md, sig)
	}
	return s.enter(md, sig)
}

// enter buys after a red bar whose volume reaches VolDailyMult average days. The signal carries no
// levels: the stop is measured from the fill. It carries the daily ATR for the engine to freeze.
// Without a known average daily volume there is no entry — the volume is the signal itself — and
// neither is there one when the stop needs an unknown ATR or would land at or below zero.
func (s *Strategy) enter(md strategy.MarketData, sig model.Signal) model.Signal {
	p := s.p
	n := len(md.Closes)
	if n == 0 || len(md.Times) != n || len(md.Opens) != n || len(md.Volumes) != n || p.VolDailyMult <= 0 {
		return sig
	}
	i := n - 1
	o, c := md.Opens[i], md.Closes[i]
	if !s.inWindow(md.Times[i]) || o <= 0 || c >= o {
		return sig
	}
	avg := s.avgDailyVolume(md)
	if avg <= 0 {
		return sig
	}
	ratio := float64(md.Volumes[i]) / avg
	// The tiny epsilon keeps a volume of exactly the threshold from failing on float rounding.
	if ratio < p.VolDailyMult-1e-9 {
		return sig
	}
	atr := s.dailyATR(md)
	if p.StopDailyATR > 0 && (atr <= 0 || c-p.StopDailyATR*atr <= 0) {
		return sig
	}
	sig.Kind = model.SignalBuy
	sig.ATR = atr
	// The fill must be the very next slot: if that candle is missing from the data, the engine
	// drops the signal rather than buying at a later session's open.
	sig.FillBy = md.Times[i].Add(s.step)
	sig.EntryReason = fmt.Sprintf("VOLUME: open %.4f → close %.4f, объём %d = %.2f× среднего дня (%.0f за %d дн., порог %.2f), дневной ATR %.4f",
		o, c, md.Volumes[i], ratio, avg, p.VolAvgPeriod, p.VolDailyMult, atr)
	return sig
}

// manage exits in priority order: the stop StopDailyATR daily ATRs below the fill price (the ATR
// frozen at entry, today's ATR when the position carries none; intrabar, filled by the engine at
// min(stop, open)), then at the bar's close RSI(RSIPeriod) crossing RSIExit upward, whatever the
// price. There is no end-of-day exit.
func (s *Strategy) manage(md strategy.MarketData, sig model.Signal) model.Signal {
	pos := md.Position
	n := len(md.Closes)
	if n == 0 || len(md.Lows) != n || pos.PurchasePrice <= 0 {
		return sig
	}
	i := n - 1
	entry := pos.PurchasePrice
	if s.p.StopDailyATR > 0 {
		atr := pos.EntryATR
		if atr <= 0 {
			atr = s.dailyATR(md)
		}
		// The tiny epsilon keeps a low of exactly the stop from slipping on float rounding.
		if stop := entry - s.p.StopDailyATR*atr; atr > 0 && stop > 0 && md.Lows[i] <= stop+1e-9 {
			sig.Kind, sig.Reason, sig.StopLoss = model.SignalSell, "SL", stop
			sig.ExitReason = fmt.Sprintf("SL: low %.4f ≤ стоп %.4f (вход %.4f, −%.2f ATR %.4f = −%.2f%%)",
				md.Lows[i], stop, entry, s.p.StopDailyATR, atr, (1-stop/entry)*100)
			return sig
		}
	}
	// RSISeries is 0 up to index RSIPeriod−1, so the cross needs bars RSIPeriod and later.
	if s.p.RSIExit > 0 && s.p.RSIPeriod > 0 && n >= s.p.RSIPeriod+2 {
		rsi := indicators.RSISeries(md.Closes, s.p.RSIPeriod)
		if prev, cur := rsi[i-1], rsi[i]; crossUp(prev, cur, s.p.RSIExit) {
			sig.Kind, sig.Reason = model.SignalSell, "RSI"
			sig.ExitReason = fmt.Sprintf("RSI: RSI(%d) %.1f → %.1f пересёк %.0f вверх, выход по %.4f (вход %.4f)",
				s.p.RSIPeriod, prev, cur, s.p.RSIExit, md.Closes[i], entry)
		}
	}
	return sig
}

// crossUp reports whether a series moved from below level to at or above it.
func crossUp(prev, cur, level float64) bool { return prev < level && cur >= level }
```

- [ ] **Step 7: Run tests**

Run: `go test ./internal/service/trading_strategy/drop_bounce/... ./internal/service/backtest/ ./internal/domain/backtest/`
Expected: PASS. If `TestEngineRSIExit` or `TestRSIExitSellsAtTheCross` reports "fixture: RSI(6) never crosses 70", lengthen the rise (`steps(..., 25, 0.3)`) — do not change the strategy.

- [ ] **Step 8: Build and lint**

Run: `go build ./internal/... ./pkg/... ./cmd/... && go vet ./internal/service/trading_strategy/drop_bounce/... && ./bin/mage lint`
Expected: no errors. (If `./bin/mage` is missing, run `./bin/golangci-lint run ./internal/service/trading_strategy/drop_bounce/... ./internal/domain/backtest/... ./internal/service/backtest/...` instead, or note that lint was not run.)

- [ ] **Step 9: Commit**

```bash
git add internal/service/trading_strategy/drop_bounce/strategy/core internal/service/backtest/drop_bounce_registry_test.go
git commit -m "feat(drop_bounce): red bar on a daily-volume spike, RSI and ATR-stop exits

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

---

### Task 5: Docs, starter grid, CLAUDE.md

**Files:**
- Rewrite: `docs/drop_bounce/strategy.md`
- Rewrite: `data/params/drop_bounce/grid_example.json`
- Modify: `CLAUDE.md` (the drop_bounce sentence inside the `trading_strategy/` bullet)

**Interfaces:** consumes `core.Params` field names from Task 4.

- [ ] **Step 1: Rewrite `docs/drop_bounce/strategy.md`**

```markdown
# drop_bounce — красная свеча на всплеске дневного объёма

Бэктест-стратегия, только лонг. Основной таймфрейм 15m; ядро от шага не зависит, поэтому cmd/backtest принимает и 5m, 30m, 1h (`-interval Minutes5|Minutes15|Minutes30|Hour1`, другие отклоняет). Порог объёма меряется в долях среднего дневного объёма, поэтому на другом таймфрейме то же число означает другое событие: доля дня в одной 5m-свече примерно втрое меньше, чем в 15m. Позиция может переноситься через ночь. Код: `internal/service/trading_strategy/drop_bounce/strategy/core`, привязка к бэктесту — `internal/service/backtest/drop_bounce_registry.go`. Спека: `docs/superpowers/specs/2026-10-12-drop-bounce-volume-spike-design.md`.

## Вход

1. Свеча-сигнал красная: `close < open`.
2. Объём свечи-сигнала не меньше `VolDailyMult` средних дневных объёмов: `Volume ≥ VolDailyMult · VolAvg`. Объём ровно на пороге проходит.
3. `VolAvg` — средний дневной объём по Уайлдеру за `VolAvgPeriod` дней: завершённые дневки Пн–Пт (сессии выходного дня отброшены — они в разы тоньше), дни с нулевым объёмом отброшены (иначе среднее занижено и порог срабатывает на пустом месте). Первые `VolAvgPeriod` дней — простое среднее, каждый следующий сдвигает его на `(V − VolAvg) / VolAvgPeriod`. У объёма нет high/low, поэтому «ATR объёма» — это само среднее. Пока таких дней меньше `VolAvgPeriod` или дневных объёмов нет, сигнала нет: объём здесь и есть сигнал.
4. Окно: свеча-сигнал — Пн–Пт МСК, начинается не раньше `StartHour:00`; свеча исполнения (через один шаг) лежит в ту же дату. Сигнал на последней свече дня на утро не переносится.
5. Дневной ATR: Уайлдер `DailyATRPeriod` по завершённым дневкам Пн–Пт. Считается на свече-сигнале и уходит в сигнал (`Signal.ATR`); движок замораживает его на позицию (`Position.EntryATR`). Если стоп включён, а ATR посчитать нельзя или стоп от close сигнала ушёл бы в ноль и ниже — сигнала нет.
6. Покупка по open следующей свечи (`Config.EntryAtNextOpen`). Сигнал несёт `FillBy = время свечи-сигнала + шаг`: если этой свечи в данных нет, движок сигнал отбрасывает, а не покупает по open следующей сессии.

Пока позиция открыта, новые сигналы игнорируются.

## Выход

По приоритету на каждой свече:

1. **SL** — внутри свечи: `low ≤ вход − StopDailyATR · ATR`, ATR замороженный на сигнале (если позиция его не несёт — сегодняшний). Движок исполняет по `min(стоп, open)`, так что ночной гэп ниже стопа закрывается по open. `StopDailyATR = 0` выключает стоп.
2. **RSI** — на close: RSI(`RSIPeriod`) пересёк `RSIExit` снизу вверх (`prev < RSIExit ≤ cur`). Продаёт при любой цене, в том числе ниже входа. `RSIExit = 0` выключает выход.

Выхода в конце дня нет.

## Параметры

| поле | по умолчанию | смысл |
|---|---|---|
| `VolDailyMult` | 0.4 | объём свечи-сигнала ≥ это · `VolAvg`; 0 — входов нет |
| `VolAvgPeriod` | 14 | длина среднего Уайлдера по будним дневным объёмам |
| `RSIPeriod` | 6 | длина RSI выхода |
| `RSIExit` | 70 | выход при пересечении вверх; 0 — выключен |
| `StopDailyATR` | 1.0 | стоп ниже цены входа в дневных ATR; 0 — выключен |
| `DailyATRPeriod` | 14 | длина дневного ATR Уайлдера по будним дням |
| `StartHour` | 8 | свеча-сигнал начинается не раньше этого часа МСК |

`Lookback = max(64, 4 · RSIPeriod + 2)`.

## Пакеты тикеров

Реестр `dropBounceRegistry` пуст: пакеты, откалиброванные на прежних правилах, удалены 2026-10-12 (история — в git). Тикер без пакета работает на `core.DefaultParams()` на любом из четырёх таймфреймов; JSON `-params` / `-calibrate` ложится поверх. Новый пакет заводится только по итогам калибровки на этих правилах (скилл `drop-bounce-calibrate`).

## Запуск

```bash
go run ./cmd/backtest -strategy drop_bounce -interval Minutes15 -ticker GAZP -months 36
go run ./cmd/backtest -strategy drop_bounce -interval Minutes15 -ticker GAZP -months 36 \
  -calibrate data/params/drop_bounce/grid_example.json -metric profit_factor
```
Для калибровочных прогонов — собранный бинарь на одном ядре (см. скилл `drop-bounce-calibrate`).
```

- [ ] **Step 2: Rewrite `data/params/drop_bounce/grid_example.json`**

```json
{
  "_comment": "drop_bounce (основной 15m; 5m, 30m, 1h тоже принимаются), стартовая сетка поверх core.DefaultParams(): порог объёма свечи-сигнала в долях среднего дневного объёма (VolDailyMult; среднее Уайлдера за VolAvgPeriod будних дней с ненулевым объёмом), стоп в дневных ATR (StopDailyATR, ATR заморожен на сигнале; нуля в сетке нет), выход по пересечению RSI(RSIPeriod) уровня RSIExit вверх.",
  "phases": [
    {
      "name": "entry",
      "grid": {
        "VolDailyMult": [0.05, 0.1, 0.15, 0.2, 0.3, 0.4, 0.6],
        "VolAvgPeriod": [10, 14, 20]
      }
    },
    {
      "name": "risk",
      "grid": {
        "StopDailyATR": [0.5, 0.75, 1.0, 1.5, 2.0]
      }
    },
    {
      "name": "exits",
      "grid": {
        "RSIPeriod": [4, 6, 9],
        "RSIExit": [60, 65, 70, 75, 80]
      }
    }
  ]
}
```

- [ ] **Step 3: Update `CLAUDE.md`**

In the `internal/service` → `trading_strategy/` bullet, replace the whole span that starts with `` `drop_bounce` — backtest-only 5m/15m/30m/1h long bounce after a red bar`` and ends with `` starter grid `data/params/drop_bounce/grid_example.json`.`` with:

```
`drop_bounce` — backtest-only long after a red bar whose volume reaches `VolDailyMult` (0.4) of the average weekday daily volume (Wilder `VolAvgPeriod`=14 over completed Mon–Fri dailies, zero-volume days left out; engine field `MarketData.DailyVolumes`): entry at the next bar's open on the same date, bars from `StartHour` (8) MSK; exits — stop `StopDailyATR` (1.0) daily ATRs (Wilder `DailyATRPeriod`=14 over weekday dailies, frozen at the signal) below the fill, RSI(`RSIPeriod`=6) crossing `RSIExit` (70) upward whatever the price; holds overnight; main timeframe 15m, also accepts 5m/30m/1h (`-strategy drop_bounce -interval Minutes15`, docs: `docs/drop_bounce/strategy.md`); no ticker packages since the 2026-10-12 rework (`dropBounceRegistry` empty, old GAZP/DOMRF/T/IVAT/WUSH calibration archived in git), starter grid `data/params/drop_bounce/grid_example.json`.
```

- [ ] **Step 4: Verify the grid parses**

Run: `python3 -c "import json; json.load(open('data/params/drop_bounce/grid_example.json'))" && go test ./internal/service/backtest/`
Expected: no output from python, PASS.

- [ ] **Step 5: Commit**

```bash
git add docs/drop_bounce/strategy.md data/params/drop_bounce/grid_example.json CLAUDE.md
git commit -m "docs(drop_bounce): volume-spike rules, starter grid, CLAUDE.md

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

---

### Task 6: Calibration skill on the new params

**Files:**
- Modify: `.claude/skills/drop-bounce-calibrate/scripts/grids.py`
- Modify: `.claude/skills/drop-bounce-calibrate/SKILL.md`

**Interfaces:** consumes `core.Params` field names and defaults from Task 4.

- [ ] **Step 1: Rewrite the grid constants and plateau in `grids.py`**

Replace the block from `DEFAULTS = dict(` through the end of `PHASES = lambda tf: [...]` with:

```python
DEFAULTS = dict(VolDailyMult=0.4, VolAvgPeriod=14, RSIPeriod=6, RSIExit=70, StopDailyATR=1.0,
                DailyATRPeriod=14, StartHour=8)
# share of an average weekday daily volume traded by the signal bar
VOL = {"Minutes5": [0.02, 0.04, 0.06, 0.1, 0.15, 0.2], "Minutes15": [0.05, 0.1, 0.15, 0.2, 0.3, 0.4, 0.6],
       "Minutes30": [0.1, 0.15, 0.2, 0.3, 0.4, 0.6], "Hour1": [0.15, 0.2, 0.3, 0.4, 0.6, 0.8]}
# plateau step of VolDailyMult past the grid's edge
VOL_STEP = {"Minutes5": 0.02, "Minutes15": 0.05, "Minutes30": 0.1, "Hour1": 0.1}
STOPS = [0.5, 0.75, 1.0, 1.5, 2.0]  # daily ATRs
PHASES = lambda tf: [
    ("entry", {"VolDailyMult": VOL[tf], "VolAvgPeriod": [10, 14, 20]}),
    ("risk", {"StopDailyATR": STOPS}),
    ("exits", {"RSIPeriod": [4, 6, 9], "RSIExit": [60, 65, 70, 75, 80]}),
]
```

Replace `if tf not in DROP:` with `if tf not in VOL:`. Delete `gate_neighbours` (no gates left). In the `calstop` branch change the comment text `entry → risk → exits` (already correct) and keep the rest. Replace the whole `elif kind == "plateau":` branch with:

```python
    elif kind == "plateau":
        p = json.load(open(path(t, f"point_{low}")))
        e = pinned({k: [v] for k, v in p.items()})
        e["VolDailyMult"] = neighbours(VOL[tf], p["VolDailyMult"], VOL_STEP[tf])
        e["RSIExit"] = neighbours([60, 65, 70, 75, 80], p["RSIExit"], 5)
        dump(path(t, f"plateau_{low}_entry"), {
            "_comment": f"drop_bounce {t} {tf}: плато точки по входу и выходу (±1 шаг VolDailyMult × RSIExit), in-sample. Запуск: {cmd(t, tf, f'plateau_{low}_entry', wf=False)}. РЕЗУЛЬТАТ: прогон не делался.",
            "phases": [{"name": "plateau_entry", "grid": e}]})
        s = pinned({k: [v] for k, v in p.items()})
        s["StopDailyATR"] = neighbours(STOPS, p["StopDailyATR"], 0.25)
        dump(path(t, f"plateau_{low}_stop"), {
            "_comment": f"drop_bounce {t} {tf}: плато точки по стопу (±1 шаг StopDailyATR), in-sample. Запуск: {cmd(t, tf, f'plateau_{low}_stop', wf=False)}. РЕЗУЛЬТАТ: прогон не делался.",
            "phases": [{"name": "plateau_stop", "grid": s}]})
```

Update the module docstring's `plateau` line to: `-> plateau_<tf>_entry.json, plateau_<tf>_stop.json`.

- [ ] **Step 2: Smoke-test the script into the scratchpad-free path and clean up**

```bash
python3 -I .claude/skills/drop-bounce-calibrate/scripts/grids.py calstop ZZTEST Minutes15
python3 -I -c "import json; d=json.load(open('data/params/drop_bounce/zztest/calstop_minutes15.json')); print([p['name'] for p in d['phases']], sorted(d['phases'][0]['grid']))"
rm -rf data/params/drop_bounce/zztest .claude/skills/drop-bounce-calibrate/scripts/__pycache__
```
Expected: `['entry', 'risk', 'exits'] ['DailyATRPeriod', 'RSIExit', 'RSIPeriod', 'StartHour', 'StopDailyATR', 'VolAvgPeriod', 'VolDailyMult']`.

- [ ] **Step 3: Update `SKILL.md`**

Edits (keep everything else):
- §0 first bullet: replace the sentence `30m и 1h не калибруй: на 2026-10-11 30m провалил walk-forward на трёх тикерах из четырёх.` with `30m и 1h не калибруй без просьбы владельца: основной таймфрейм стратегии — 15m.`; replace `(`grids.py` знает их сетки глубины падения)` with `(`grids.py` знает их сетки порога объёма)`.
- §0: delete the two bullets starting `- **Фильтр размаха дня `MaxDayRangeATR`**` and `- **Объёмный фильтр `VolMult`/`VolBars`**`.
- §0 «Окно 36 месяцев» bullet: replace `число сделок задают падения, а не число баров (≈ 40 сделок за 36 мес)` with `число сделок задают всплески объёма, а не число баров`.
- §2: replace the paragraph starting `**Тематическую калибровку по одной оси поверх дефолтов не делай.**` with `**Тематическую калибровку по одной оси поверх дефолтов не делай.** На прежних правилах (до 2026-10-12) все темы на 15m были ниже 1, точки провалились: ось, полезная в связке, на убыточном baseline не видна.`
- §3: replace the description from `` `grids.py calstop <T> <TF>` → `` through `(убраны 2026-10-12).` with: `` `grids.py calstop <T> <TF>` → `calstop_<tf>.json` (entry → risk → exits, keepTop 3): entry — `VolDailyMult` (сетка своя у каждого таймфрейма) × `VolAvgPeriod` (10, 14, 20); risk — `StopDailyATR` (0.5–2.0); exits — `RSIPeriod` (4, 6, 9) × `RSIExit` (60–80). ``
- §4 step 3: replace from `` `plateau_<tf>_stop.json` и для каждого фильтра`` through `нижний сосед фильтра может быть 0 — выключен.` with `` `plateau_<tf>_stop.json`.`` (the sentence then reads: «…in-sample прогоны `plateau_<tf>_entry.json`, `plateau_<tf>_stop.json`. Все с `-min-trades 1`, …»).
- §6 doc-comment bullet: replace `стоп в ATR, фильтр размаха дня и объёмный фильтр точки (или «выключен»), плато,` with `порог объёма, стоп в ATR, выход RSI, плато,`.
- Red flags table: delete the row starting `| «Точка готова, добавлю фильтр размаха дня / объёма сверху»`.

Then confirm nothing stale is left:

Run: `grep -nE "MinDropPct|RSIEntry|GreenWait|ProfitPct|ExitInProfit|Stoch|MaxDayRange|VolMult|VolBars|DROP" .claude/skills/drop-bounce-calibrate/SKILL.md .claude/skills/drop-bounce-calibrate/scripts/*.py`
Expected: no matches. (If `report.py` / `overlap.py` match, they only read reports — fix the names the same way or leave them if they are generic; report what you did.)

- [ ] **Step 4: Commit**

```bash
git add .claude/skills/drop-bounce-calibrate
git commit -m "chore(drop_bounce): calibration skill on the volume-spike params

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

---

### Task 7: Final verification

- [ ] **Step 1:** `go build ./internal/... ./pkg/... ./cmd/...` — OK.
- [ ] **Step 2:** `go test -race ./internal/domain/backtest/ ./internal/service/backtest/ ./internal/service/trading_strategy/... ./cmd/...` — PASS.
- [ ] **Step 3:** `./bin/mage lint` (or report it was not run) — clean.
- [ ] **Step 4:** `git status --short` — only `docs/ideas/*.md` untracked.
