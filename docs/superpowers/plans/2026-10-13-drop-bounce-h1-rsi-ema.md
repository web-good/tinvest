# drop_bounce 1h RSI-zone rewrite Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace the drop_bounce volume-spike rules with a 1h long strategy: RSI oversold zone (or an upward cross of it under a cap) above an EMA, entered only while the day's range has not used up its share of the daily ATR; exits unchanged (daily-ATR stop, RSI cross up).

**Architecture:** The pure core `internal/service/trading_strategy/drop_bounce/strategy/core` is rewritten in place (same `Strategy`/`Params`/`NewWithParams` shape, new fields). Everything around it shrinks to one timeframe: the backtest registry accepts only `Hour1` and holds one literal per ticker (none today), the live-universe config keeps only `DROP_BOUNCE_TICKERS_H1`, the screener grid is rebuilt on the new fields, and the calibration skill/docs follow.

**Tech Stack:** Go 1.25, `tinvest/internal/domain/ema.Compute`, `tinvest/pkg/indicators` (`RSISeries`, `ATR`), backtest engine `tinvest/internal/domain/backtest`, Python 3 scripts of the skill.

**Spec:** `docs/superpowers/specs/2026-10-13-drop-bounce-h1-rsi-ema-design.md`

## Global Constraints

- Timeframe: `Hour1` only; every other interval is rejected with an error.
- Defaults: `RSIPeriod` 6, `RSIZone` 30, `RSICap` 50, `EMAPeriod` 200, `DaySpentATR` 0.7, `StopDailyATR` 1.0, `DailyATRPeriod` 14, `RSIExit` 70, `StartHour` 7.
- All `Params` fields are `int` or `float64` (reflection grids sweep them).
- `Lookback = max(64, 3·EMAPeriod, 4·RSIPeriod+2)`.
- Calibration phases (owner's lists, never changed without a request): entry `RSIPeriod` 4/5/6/7 × `RSIZone` 25/30/35 × `RSICap` 40/50/60; filter `EMAPeriod` 100/150/200 × `DaySpentATR` 0.3/0.5/0.7/1.0; risk `StopDailyATR` 0.5/0.7/1.0/1.5/2.0 × `DailyATRPeriod` 7/10/14; exit `RSIExit` 60/65/70/75/80; `keepTop 3`; `StartHour` pinned 7.
- Screener grid: `RSIZone` {25, 35} × `RSICap` {40, 60} × `DaySpentATR` {0.5, 1.0} × `RSIExit` {60, 75}; pinned `RSIPeriod` 6, `EMAPeriod` 200, `StopDailyATR` 1.0, `DailyATRPeriod` 14, `StartHour` 7.
- Code comments and identifiers in English, user-facing reason strings and docs in Russian — as the current core does.
- Backtests/tests on one core when run by hand: `GOMAXPROCS=1`, `taskset -c 0`, `nice -n 19` (owner rule). Plain `go test` for unit tests is fine.
- Quality gate: `./bin/mage ci` green at the end (`go build ./...` fails on `magefiles` — build with `go build ./internal/... ./pkg/... ./cmd/...`).

## Review Focus

1. RSI warm-up zeros: `RSISeries` returns 0 before index `RSIPeriod`; a 0 must never pass as "RSI < RSIZone". Expect no entry when `len(closes) < RSIPeriod+2`. Test in Task 2 (`TestEnterNeedsWarmRSI`).
2. Unwarmed EMA: `ema.Compute` returns 0 before `EMAPeriod-1`; "close > 0" must not pass as an uptrend. Expect no entry with fewer than `EMAPeriod` bars. Test in Task 2 (`TestEnterNeedsWarmEMA`).
3. Day extent missing or inverted (`TodayHigh`/`TodayLow` zero, or high < low — e.g. a strategy called outside the engine). Expect no entry. Test in Task 2 (`TestEnterDayGate`).
4. A flat RSI of exactly 50 (`wilderRSI` returns 50 when there were no losses) on a quiet tape must not look like a CROSS through 30. Expect no entry on a flat series. Test in Task 5 (`TestScreenDropTickerFlatSeriesProducesNoSignals`) and Task 2 (`TestEnterNoSignalWhenRSIAboveZoneWithoutCross`).
5. A registered-literal JSON override (`-params`/`-calibrate`) with an old field name (`VolDailyMult`) must not silently pass. `json.Unmarshal` ignores unknown fields, so expect the run to proceed on defaults — this is accepted behaviour, pinned so nobody assumes otherwise. Test in Task 3 (`TestDropBounceParseParamsIgnoresRemovedFields`).

---

### Task 0: Freeze the branch's pending work

**Files:** everything currently modified/untracked on `feat/drop-bounce` (see `git status`).

- [ ] **Step 1: Check the tree is green before freezing it**

Run: `go test ./internal/service/backtest/ ./internal/config/ ./internal/service/trading_strategy/drop_bounce/...`
Expected: PASS. If something fails, stop and report to the owner — do not commit a red tree.

- [ ] **Step 2: Commit as-is**

```bash
git add -A .claude/skills/drop-bounce-calibrate CLAUDE.md cmd/dropscreen docs/drop_bounce docs/ideas env/local.env.example env/prod.env env/prod.env.example internal data/params/drop_bounce
git status --short   # must be empty except files the owner keeps out of git
git commit -m "chore(drop_bounce): freeze 15m volume-spike calibrations before the 1h rewrite

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

---

### Task 1: Test fixtures on 1h bars

The exit tests stay valid for the new core (exits do not change), but they and the shared fixture are built on 15m bars and volume. This task moves the shared helpers to 1h and removes the volume fixture, before the core changes, so Task 2 starts from a compiling test package.

**Files:**
- Modify: `internal/service/trading_strategy/drop_bounce/strategy/core/core_test.go` (helpers only; entry tests are replaced in Task 2)
- Modify: `internal/service/trading_strategy/drop_bounce/strategy/core/exits_test.go`

**Interfaces:**
- Produces (test package `core`): `stepH1 = time.Hour`; `at(s string) time.Time`; `type bar struct{t time.Time; o,h,l,c float64}`; `path(start string, closes []float64) []bar` (1h bars); `flatN(n int, p float64) []float64`; `steps(from float64, n int, pct float64) []float64`; `fixtureATR = 2.0`; `dailies(before time.Time, n int, rng float64) []bar`; `withDailies(md *strategy.MarketData, ds []bar)`; `fixture(bars []bar) strategy.MarketData`; `held(p Params, bars []bar, entry float64, entryTime time.Time) model.Signal`; `eps = 1e-9`.

- [ ] **Step 1: Rewrite the helper block of `core_test.go`**

Replace everything from `// step15 is the timeframe` down to the end of `held` with:

```go
// stepH1 is the only timeframe drop_bounce runs on.
const stepH1 = time.Hour

// path builds consecutive 1h bars from start (MSK): each bar opens at the previous close (the
// first at its own close) and its high/low span the open and the close.
func path(start string, closes []float64) []bar {
	t0 := at(start)
	out := make([]bar, len(closes))
	prev := closes[0]
	for i, c := range closes {
		out[i] = bar{t0.Add(time.Duration(i) * stepH1), prev, max(prev, c), min(prev, c), c}
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

// fixtureUsed is the day's range so far every fixture reports: well inside the default gate
// (DaySpentATR 0.7 · fixtureATR 2 = 1.4).
const fixtureUsed = 0.5

// fixture builds the market data over bars with 20 weekday dailies of range fixtureATR before the
// first bar, and a day extent of fixtureUsed around the last close.
func fixture(bars []bar) strategy.MarketData {
	var md strategy.MarketData
	for _, b := range bars {
		md.Times = append(md.Times, b.t)
		md.Opens = append(md.Opens, b.o)
		md.Highs = append(md.Highs, b.h)
		md.Lows = append(md.Lows, b.l)
		md.Closes = append(md.Closes, b.c)
		md.Volumes = append(md.Volumes, 10)
	}
	md.Price = md.Closes[len(md.Closes)-1]
	md.TodayLow = md.Price - fixtureUsed/2
	md.TodayHigh = md.Price + fixtureUsed/2
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

// withDailies replaces the daily series of md with ds.
func withDailies(md *strategy.MarketData, ds []bar) {
	md.DailyHighs, md.DailyLows, md.DailyCloses, md.DailyTimes = nil, nil, nil, nil
	for _, d := range ds {
		md.DailyTimes = append(md.DailyTimes, d.t)
		md.DailyHighs = append(md.DailyHighs, d.h)
		md.DailyLows = append(md.DailyLows, d.l)
		md.DailyCloses = append(md.DailyCloses, d.c)
	}
}

// held decides with a position bought at entry on the bar starting at entryTime, with the daily
// ATR frozen at entry equal to fixtureATR.
func held(p Params, bars []bar, entry float64, entryTime time.Time) model.Signal {
	md := fixture(bars)
	md.Position = &strategy.Position{PurchasePrice: entry, Quantity: 1, EntryTime: entryTime, EntryATR: fixtureATR}
	return NewWithParams("TEST", stepH1, p).Decide(md)
}
```

Delete every test function in `core_test.go` after `held` (`TestDefaultParams` … `TestLookback`) — Task 2 writes the new ones. Delete `fixtureDayVol`, `quietVol`, `redAt`, `spikeMD`, `spike`.

- [ ] **Step 2: Move `exits_test.go` to 1h**

In `exits_test.go` replace every `step15` with `stepH1` (`flatWithLast`, `TestStopUsesATRFrozenAtEntry`, `TestStopFallsBackToCurrentATR`, `TestStopUnplaceableLeavesRSIExit`, and any other occurrence). No other change: the exit rules are the same.

- [ ] **Step 3: Delete `engine_test.go` for now**

`git rm internal/service/trading_strategy/drop_bounce/strategy/core/engine_test.go` — it is rebuilt in Task 2 on the new entry. (It references the removed volume helpers and would not compile.)

- [ ] **Step 4: Run the exit tests on the old core**

Run: `go test ./internal/service/trading_strategy/drop_bounce/strategy/core/ -run 'Stop|RSIExit|CrossUp|EOD'`
Expected: PASS (exits are timeframe-independent).

- [ ] **Step 5: Commit**

```bash
git add -A internal/service/trading_strategy/drop_bounce/strategy/core
git commit -m "test(drop_bounce): move core fixtures to 1h bars

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

---

### Task 2: New entry rules in the core

**Files:**
- Modify: `internal/service/trading_strategy/drop_bounce/strategy/core/core.go`
- Test: `internal/service/trading_strategy/drop_bounce/strategy/core/core_test.go`
- Create: `internal/service/trading_strategy/drop_bounce/strategy/core/engine_test.go`

**Interfaces:**
- Consumes: Task 1 helpers.
- Produces:
  ```go
  type Params struct {
      RSIPeriod      int
      RSIZone        float64
      RSICap         float64
      EMAPeriod      int
      DaySpentATR    float64
      StopDailyATR   float64
      DailyATRPeriod int
      RSIExit        float64
      StartHour      int
  }
  func DefaultParams() Params
  func NewWithParams(ticker string, step time.Duration, p Params) *Strategy
  func (s *Strategy) Lookback() int
  func (s *Strategy) Decide(md strategy.MarketData) model.Signal
  ```
  Entry reasons start with `ZONE:` or `CROSS:`; exit reasons stay `SL` / `RSI`.

- [ ] **Step 1: Write the failing entry tests**

Append to `core_test.go` (add `"tinvest/pkg/indicators"` and `"tinvest/internal/domain/ema"` to its imports; keep `math`, `strings`):

```go
// trendCloses is 60 zigzag bars (+2%, −0.5%) from 100, then `drops` bars of −1% each. With
// RSI(6) and EMA(30): drops=5 leaves RSI 33.52 (above the zone, no cross), drops=6 → RSI 28.34
// with the close 146.73 above EMA 144.43, drops=12 → RSI deep in the zone with the close below
// the EMA.
func trendCloses(drops int) []float64 {
	c := []float64{100}
	for i := range 60 {
		k := 1.02
		if i%2 == 1 {
			k = 0.995
		}
		c = append(c, c[len(c)-1]*k)
	}
	for range drops {
		c = append(c, c[len(c)-1]*0.99)
	}
	return c
}

// withLast appends one more close mult times the last.
func withLast(c []float64, mult float64) []float64 {
	return append(append([]float64(nil), c...), c[len(c)-1]*mult)
}

// entryEnd is the start of the decision bar: Wednesday 12:00 MSK, inside the window.
const entryEnd = "2026-09-16 12:00"

// endingAt builds 1h bars over closes whose last bar starts at end (MSK).
func endingAt(end string, closes []float64) []bar {
	start := at(end).Add(-time.Duration(len(closes)-1) * stepH1)
	return path(start.Format("2006-01-02 15:04"), closes)
}

// entryParams is the default params with EMA(30): the fixtures are ~70 bars long.
func entryParams() Params {
	p := DefaultParams()
	p.EMAPeriod = 30
	return p
}

// decide runs a fresh strategy over closes ending at end.
func decide(p Params, end string, closes []float64) model.Signal {
	return NewWithParams("TEST", stepH1, p).Decide(fixture(endingAt(end, closes)))
}

// rsiAt / emaAt are the fixture's indicator values at the last bar, for guards.
func rsiAt(c []float64, period int) (prev, cur float64) {
	r := indicators.RSISeries(c, period)
	return r[len(r)-2], r[len(r)-1]
}

func emaAt(c []float64, period int) float64 { return ema.Compute(c, period)[len(c)-1] }

func TestDefaultParams(t *testing.T) {
	want := Params{RSIPeriod: 6, RSIZone: 30, RSICap: 50, EMAPeriod: 200, DaySpentATR: 0.7,
		StopDailyATR: 1.0, DailyATRPeriod: 14, RSIExit: 70, StartHour: 7}
	if got := DefaultParams(); got != want {
		t.Fatalf("DefaultParams() = %+v, want %+v", got, want)
	}
}

func TestLookback(t *testing.T) {
	cases := []struct {
		p    Params
		want int
	}{
		{DefaultParams(), 600},
		{Params{RSIPeriod: 6, EMAPeriod: 10}, 64},
		{Params{RSIPeriod: 20, EMAPeriod: 10}, 82},
	}
	for _, c := range cases {
		if got := NewWithParams("T", stepH1, c.p).Lookback(); got != c.want {
			t.Fatalf("Lookback(%+v) = %d, want %d", c.p, got, c.want)
		}
	}
}

func TestEnterZoneAboveEMA(t *testing.T) {
	c := trendCloses(6)
	if _, cur := rsiAt(c, 6); cur >= 30 || c[len(c)-1] <= emaAt(c, 30) {
		t.Fatalf("fixture: RSI %.2f must be < 30 and close %.2f > EMA %.2f", cur, c[len(c)-1], emaAt(c, 30))
	}
	sig := decide(entryParams(), entryEnd, c)
	if sig.Kind != model.SignalBuy || !strings.HasPrefix(sig.EntryReason, "ZONE") {
		t.Fatalf("got %v %q, want Buy ZONE", sig.Kind, sig.EntryReason)
	}
	if math.Abs(sig.ATR-fixtureATR) > eps {
		t.Fatalf("ATR = %v, want %v", sig.ATR, fixtureATR)
	}
	if want := at(entryEnd).Add(stepH1); !sig.FillBy.Equal(want) {
		t.Fatalf("FillBy = %v, want %v", sig.FillBy, want)
	}
}

func TestEnterZoneBelowEMANoSignal(t *testing.T) {
	c := trendCloses(12)
	if _, cur := rsiAt(c, 6); cur >= 30 || c[len(c)-1] >= emaAt(c, 30) {
		t.Fatalf("fixture: RSI %.2f must be < 30 and close %.2f < EMA %.2f", cur, c[len(c)-1], emaAt(c, 30))
	}
	if sig := decide(entryParams(), entryEnd, c); sig.Kind != model.SignalNone {
		t.Fatalf("below EMA: got %v %q, want none", sig.Kind, sig.EntryReason)
	}
}

func TestTrendUpIsStrict(t *testing.T) {
	if trendUp(100, 100) || trendUp(100, 0) || !trendUp(100.01, 100) {
		t.Fatal("trendUp must need close strictly above a warmed (non-zero) EMA")
	}
}

func TestEnterCrossUnderCap(t *testing.T) {
	c := withLast(trendCloses(6), 1.01)
	if prev, cur := rsiAt(c, 6); !(prev < 30 && cur >= 30 && cur <= 50) {
		t.Fatalf("fixture: RSI %.2f → %.2f must cross 30 and stay ≤ 50", prev, cur)
	}
	sig := decide(entryParams(), entryEnd, c)
	if sig.Kind != model.SignalBuy || !strings.HasPrefix(sig.EntryReason, "CROSS") {
		t.Fatalf("got %v %q, want Buy CROSS", sig.Kind, sig.EntryReason)
	}
}

func TestEnterCrossAboveCap(t *testing.T) {
	c := withLast(trendCloses(6), 1.03)
	if prev, cur := rsiAt(c, 6); !(prev < 30 && cur > 50 && cur <= 60) {
		t.Fatalf("fixture: RSI %.2f → %.2f must cross 30 and land in (50, 60]", prev, cur)
	}
	if sig := decide(entryParams(), entryEnd, c); sig.Kind != model.SignalNone {
		t.Fatalf("above cap 50: got %v %q, want none", sig.Kind, sig.EntryReason)
	}
	p := entryParams()
	p.RSICap = 60
	if sig := decide(p, entryEnd, c); sig.Kind != model.SignalBuy {
		t.Fatalf("cap 60: got %v, want Buy", sig.Kind)
	}
}

func TestEnterNoSignalWhenRSIAboveZoneWithoutCross(t *testing.T) {
	c := trendCloses(5)
	if prev, cur := rsiAt(c, 6); prev < 30 || cur < 30 {
		t.Fatalf("fixture: RSI %.2f → %.2f must stay at/above 30", prev, cur)
	}
	if sig := decide(entryParams(), entryEnd, c); sig.Kind != model.SignalNone {
		t.Fatalf("got %v %q, want none", sig.Kind, sig.EntryReason)
	}
	// A flat tape has RSI exactly 50 on every bar (no losses): never a zone, never a cross.
	if sig := decide(entryParams(), entryEnd, flatN(70, 100)); sig.Kind != model.SignalNone {
		t.Fatalf("flat: got %v, want none", sig.Kind)
	}
}

func TestEnterNeedsWarmRSI(t *testing.T) {
	p := entryParams()
	p.EMAPeriod = 2 // keep the EMA warm so only the RSI warm-up is under test
	c := steps(100, 7, -1) // 7 closes: RSI(6) has one value, no previous one
	if sig := decide(p, entryEnd, c); sig.Kind != model.SignalNone {
		t.Fatalf("got %v, want none before RSI has two values", sig.Kind)
	}
}

func TestEnterNeedsWarmEMA(t *testing.T) {
	c := trendCloses(6) // 67 closes
	p := entryParams()
	p.EMAPeriod = len(c) + 1
	if sig := decide(p, entryEnd, c); sig.Kind != model.SignalNone {
		t.Fatalf("got %v, want none with an unwarmed EMA", sig.Kind)
	}
}

func TestEnterDayGate(t *testing.T) {
	c := trendCloses(6)
	p := entryParams()
	p.DaySpentATR = 0.5 // threshold 0.5 · fixtureATR 2 = 1.0
	run := func(hi, lo float64) model.Signal {
		md := fixture(endingAt(entryEnd, c))
		md.TodayHigh, md.TodayLow = hi, lo
		return NewWithParams("TEST", stepH1, p).Decide(md)
	}
	if sig := run(100.99, 100); sig.Kind != model.SignalBuy {
		t.Fatalf("used 0.99 < 1.0: got %v, want Buy", sig.Kind)
	}
	if sig := run(101, 100); sig.Kind != model.SignalNone {
		t.Fatalf("used exactly 1.0: got %v, want none (spent)", sig.Kind)
	}
	if sig := run(103, 100); sig.Kind != model.SignalNone {
		t.Fatalf("used 3.0: got %v, want none", sig.Kind)
	}
	for _, ext := range [][2]float64{{0, 0}, {101, 0}, {0, 100}, {100, 101}} {
		if sig := run(ext[0], ext[1]); sig.Kind != model.SignalNone {
			t.Fatalf("broken extent %v: got %v, want none", ext, sig.Kind)
		}
	}
	md := fixture(endingAt(entryEnd, c))
	md.DailyHighs, md.DailyLows, md.DailyCloses, md.DailyTimes = nil, nil, nil, nil
	if sig := NewWithParams("TEST", stepH1, p).Decide(md); sig.Kind != model.SignalNone {
		t.Fatalf("no daily ATR: got %v, want none", sig.Kind)
	}
}

func TestEnterZeroParamsGiveNoEntries(t *testing.T) {
	c := trendCloses(6)
	for name, mut := range map[string]func(*Params){
		"DaySpentATR": func(p *Params) { p.DaySpentATR = 0 },
		"EMAPeriod":   func(p *Params) { p.EMAPeriod = 0 },
		"RSIPeriod":   func(p *Params) { p.RSIPeriod = 0 },
	} {
		p := entryParams()
		mut(&p)
		if sig := decide(p, entryEnd, c); sig.Kind != model.SignalNone {
			t.Fatalf("%s = 0: got %v, want none", name, sig.Kind)
		}
	}
}

func TestEnterStopGuard(t *testing.T) {
	c := trendCloses(6)
	p := entryParams()
	p.StopDailyATR = c[len(c)-1]/fixtureATR + 1 // stop below zero
	if sig := decide(p, entryEnd, c); sig.Kind != model.SignalNone {
		t.Fatalf("stop ≤ 0: got %v, want none", sig.Kind)
	}
}

func TestEnterWindow(t *testing.T) {
	c := trendCloses(6)
	for _, end := range []string{
		"2026-09-19 12:00", // Saturday
		"2026-09-20 12:00", // Sunday
		"2026-09-16 06:00", // before StartHour 7
		"2026-09-16 23:00", // fill bar would be the next date
	} {
		if sig := decide(entryParams(), end, c); sig.Kind != model.SignalNone {
			t.Fatalf("%s: got %v, want none", end, sig.Kind)
		}
	}
	if sig := decide(entryParams(), "2026-09-16 07:00", c); sig.Kind != model.SignalBuy {
		t.Fatalf("07:00: got %v, want Buy", sig.Kind)
	}
}
```

- [ ] **Step 2: Run to see them fail**

Run: `go test ./internal/service/trading_strategy/drop_bounce/strategy/core/`
Expected: compile errors — unknown fields `RSIZone`, `RSICap`, `EMAPeriod`, `DaySpentATR`, undefined `trendUp`.

- [ ] **Step 3: Rewrite `core.go`**

Replace the package doc, `minLookback` comment, `Params`, `DefaultParams`, `Lookback`, delete `avgDailyVolume`, and replace `enter`. Keep `mskLoc`, `dayKey`, `isWeekend`, `weekdayDaily`, `dailyATR`, `inWindow`, `Decide`, `manage`, `crossUp` unchanged.

```go
// Package core implements drop_bounce: a long-only 1h buy of an oversold RSI inside an uptrend
// while the day still has range left. On the close of a Mon–Fri MSK bar starting at or after
// StartHour whose fill bar lies on the same date, it buys when (1) the close is strictly above
// EMA(EMAPeriod); (2) RSI(RSIPeriod) is below RSIZone (ZONE), or has just crossed RSIZone upward
// and is not above RSICap (CROSS); (3) the day's range so far, TodayHigh − TodayLow, is below
// DaySpentATR daily ATRs (Wilder, DailyATRPeriod, completed weekday dailies). The backtest fills
// at the next bar's open (Config.EntryAtNextOpen); the daily ATR is frozen for the trade. Exits,
// in order: a stop StopDailyATR daily ATRs below the fill price, then RSI(RSIPeriod) crossing
// RSIExit upward whatever the price. No end-of-day exit: positions carry overnight. Pure,
// stateless between bars and ticker-agnostic. Run with `-strategy drop_bounce -interval Hour1`.
package core

import (
	"fmt"
	"time"

	"tinvest/internal/domain/ema"
	"tinvest/internal/service/trading_strategy/scalping/model"
	"tinvest/internal/service/trading_strategy/scalping/strategy"
	"tinvest/pkg/indicators"
)

// minLookback keeps 64 bars so the RSI converges even with a short EMA.
const minLookback = 64

// Params holds every tunable. All fields are int or float64 so reflection grid calibration can
// sweep them.
type Params struct {
	RSIPeriod int     // RSI length of the entry and the exit (default 6); 0 = no entries
	RSIZone   float64 // oversold level: ZONE below it, CROSS through it upward (default 30)
	RSICap    float64 // a CROSS counts only if RSI on the cross bar is ≤ this (default 50)
	EMAPeriod int     // trend EMA on the bar closes (default 200); 0 = no entries
	// DaySpentATR is the share of the daily ATR the day's range may have used before entry: entry
	// needs TodayHigh − TodayLow < DaySpentATR · dailyATR (default 0.7); 0 = no entries.
	DaySpentATR float64
	// StopDailyATR is the stop distance below the fill price in daily ATRs frozen at the signal;
	// 0 = off (default 1).
	StopDailyATR float64
	// DailyATRPeriod is the Wilder ATR length over completed weekday dailies (default 14).
	DailyATRPeriod int
	RSIExit        float64 // sell when RSI crosses this upward; 0 = off (default 70)
	StartHour      int     // signal bars must start at/after StartHour:00 MSK (default 7)
}

// DefaultParams returns the owner's initial configuration.
func DefaultParams() Params {
	return Params{
		RSIPeriod:      6,
		RSIZone:        30,
		RSICap:         50,
		EMAPeriod:      200,
		DaySpentATR:    0.7,
		StopDailyATR:   1,
		DailyATRPeriod: 14,
		RSIExit:        70,
		StartHour:      7,
	}
}
```

`Lookback`:

```go
// Lookback is enough history for the EMA (three lengths: the SMA seed has faded) and the Wilder
// RSI to converge, plus the previous bar the cross compares against.
func (s *Strategy) Lookback() int {
	return max(minLookback, 3*s.p.EMAPeriod, 4*s.p.RSIPeriod+2)
}
```

`enter` and `trendUp`:

```go
// enter buys an oversold RSI above the trend EMA while the day still has range left. The signal
// carries no levels: the stop is measured from the fill. It carries the daily ATR for the engine
// to freeze. Without a daily ATR or a sane day extent there is no entry — the day gate needs both.
func (s *Strategy) enter(md strategy.MarketData, sig model.Signal) model.Signal {
	p := s.p
	n := len(md.Closes)
	if p.RSIPeriod <= 0 || p.EMAPeriod <= 0 || p.DaySpentATR <= 0 || len(md.Times) != n {
		return sig
	}
	// RSISeries is 0 up to index RSIPeriod−1: the cross needs bars RSIPeriod and later, and a
	// warm-up zero must never read as "below the zone".
	if n < p.RSIPeriod+2 {
		return sig
	}
	i := n - 1
	c := md.Closes[i]
	if !s.inWindow(md.Times[i]) {
		return sig
	}
	trend := ema.Compute(md.Closes, p.EMAPeriod)[i]
	if !trendUp(c, trend) {
		return sig
	}
	rsi := indicators.RSISeries(md.Closes, p.RSIPeriod)
	prev, cur := rsi[i-1], rsi[i]
	var branch string
	switch {
	case cur < p.RSIZone:
		branch = "ZONE"
	case crossUp(prev, cur, p.RSIZone) && cur <= p.RSICap:
		branch = "CROSS"
	default:
		return sig
	}
	atr := s.dailyATR(md)
	hi, lo := md.TodayHigh, md.TodayLow
	if atr <= 0 || hi <= 0 || lo <= 0 || hi < lo {
		return sig
	}
	used := hi - lo
	if used >= p.DaySpentATR*atr {
		return sig
	}
	if p.StopDailyATR > 0 && c-p.StopDailyATR*atr <= 0 {
		return sig
	}
	sig.Kind = model.SignalBuy
	sig.ATR = atr
	// The fill must be the very next slot: if that candle is missing from the data, the engine
	// drops the signal rather than buying at a later session's open.
	sig.FillBy = md.Times[i].Add(s.step)
	sig.EntryReason = fmt.Sprintf("%s: RSI(%d) %.1f → %.1f (зона %.0f, планка %.0f), close %.4f > EMA(%d) %.4f, ход дня %.4f = %.2f ATR < %.2f (ATR %.4f)",
		branch, p.RSIPeriod, prev, cur, p.RSIZone, p.RSICap, c, p.EMAPeriod, trend, used, used/atr, p.DaySpentATR, atr)
	return sig
}

// trendUp reports whether the close sits strictly above a warmed EMA. ema.Compute zero-fills
// warm-up positions, so an unwarmed EMA must not pass as "price above zero".
func trendUp(closeP, emaNow float64) bool {
	return emaNow > 0 && closeP > emaNow
}
```

Update the `Strategy.step` field comment to: `// step is the bar timeframe (1h in production; tests and the screener pass it). It bounds the fill: the fill bar starts one step after the signal bar, must lie on the same MSK date, and bounds Signal.FillBy.`

- [ ] **Step 4: Run the core tests**

Run: `go test ./internal/service/trading_strategy/drop_bounce/strategy/core/`
Expected: PASS. If a fixture guard (`fixture: ...`) fails, the fixture numbers drifted — fix the fixture, never the guard.

- [ ] **Step 5: Write the engine test**

Create `engine_test.go`:

```go
// engine_test.go runs drop_bounce through the real backtest engine on 1h bars to pin what it
// relies on: the day extent and dailies reaching the strategy, entry at the next bar's open, the
// daily-ATR stop filled at the gap open overnight, the RSI exit, and a signal dropped when the
// next slot is missing.
package core

import (
	"math"
	"testing"
	"time"

	bt "tinvest/internal/domain/backtest"
)

// engineParams opens the day gate wide: the fixture's trend runs far beyond one daily ATR within
// a day, and the gate itself is unit-tested in core_test.go.
func engineParams() Params {
	p := entryParams()
	p.DaySpentATR = 100
	return p
}

// toCandles converts bars to candles.
func toCandles(bars []bar) []bt.Candle {
	out := make([]bt.Candle, len(bars))
	for i, b := range bars {
		out[i] = bt.Candle{Time: b.t, Open: b.o, High: b.h, Low: b.l, Close: b.c, Volume: 10}
	}
	return out
}

// signalBars is Lookback flat bars at 100, then trendCloses(6) scaled from 100, with the last
// (signal) bar starting at end. The engine decides only from bar Lookback()−1 on.
func signalBars(end string) []bar {
	lead := NewWithParams("TEST", stepH1, engineParams()).Lookback()
	closes := append(flatN(lead, 100), trendCloses(6)[1:]...)
	return endingAt(end, closes)
}

func runEngine(bars []bar) bt.Result {
	return bt.Run(NewWithParams("TEST", stepH1, engineParams()), toCandles(bars),
		toCandles(dailies(bars[0].t, 20, fixtureATR)), nil,
		bt.Config{InitialCash: 1e6, Fraction: 1, Commission: 0, Lot: 1, EntryAtNextOpen: true})
}

func TestEngineEntryAndRSIExit(t *testing.T) {
	bars := signalBars("2026-09-16 12:00")
	last := bars[len(bars)-1].c
	f := last * 1.001
	bars = append(bars, bar{at("2026-09-16 13:00"), f, f, f, f})
	for i, c := range steps(f, 6, 2) {
		bars = append(bars, bar{at("2026-09-16 14:00").Add(time.Duration(i) * stepH1), c / 1.02, c, c / 1.02, c})
	}
	res := runEngine(bars)
	if len(res.Trades) != 1 {
		t.Fatalf("trades = %d, want 1: %+v", len(res.Trades), res.Trades)
	}
	tr := res.Trades[0]
	if math.Abs(tr.EntryPrice-f) > eps || !tr.EntryTime.Equal(at("2026-09-16 13:00")) {
		t.Fatalf("entry %v @ %v, want %v @ 13:00 (next bar's open)", tr.EntryPrice, tr.EntryTime, f)
	}
	if tr.Reason != "RSI" {
		t.Fatalf("exit reason %q, want RSI", tr.Reason)
	}
}

func TestEngineOvernightGapStop(t *testing.T) {
	bars := signalBars("2026-09-16 21:00")
	f := bars[len(bars)-1].c
	gap := f - 3*fixtureATR
	bars = append(bars,
		bar{at("2026-09-16 22:00"), f, f, f, f},
		bar{at("2026-09-17 07:00"), gap, gap, gap * 0.999, gap},
	)
	res := runEngine(bars)
	if len(res.Trades) != 1 {
		t.Fatalf("trades = %d, want 1: %+v", len(res.Trades), res.Trades)
	}
	tr := res.Trades[0]
	if tr.Reason != "SL" || math.Abs(tr.ExitPrice-gap) > eps {
		t.Fatalf("exit %q @ %v, want SL at the gap open %v (stop %v)", tr.Reason, tr.ExitPrice, gap, f-fixtureATR)
	}
}

func TestEngineDropsSignalWhenNextSlotMissing(t *testing.T) {
	bars := signalBars("2026-09-16 12:00")
	f := bars[len(bars)-1].c
	// 13:00 is missing: the next bar is 15:00.
	bars = append(bars, bar{at("2026-09-16 15:00"), f, f, f, f})
	if res := runEngine(bars); len(res.Trades) != 0 {
		t.Fatalf("trades = %d, want 0 (fill slot missing): %+v", len(res.Trades), res.Trades)
	}
}
```

Note for the implementer: in `TestEngineOvernightGapStop` and `TestEngineDropsSignalWhenNextSlotMissing` the run may end with an open position; the engine closes open positions at the end of data — check how `bt.Run` reports that (an `EOD`/end-of-data trade). If it does, assert on `res.Trades[0]` for the gap test as written and, in the missing-slot test, assert that no trade has `EntryTime` on 2026-09-16. Adjust the assertion, not the rule.

- [ ] **Step 6: Run all core tests**

Run: `go test ./internal/service/trading_strategy/drop_bounce/strategy/core/`
Expected: PASS.

- [ ] **Step 7: Commit**

```bash
git add internal/service/trading_strategy/drop_bounce/strategy/core
git commit -m "feat(drop_bounce): 1h RSI zone above EMA with unspent daily range

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

(The repo does not build after this commit until Task 3: ticker packages and the screener still use removed fields. Tasks 2–5 can be squashed later if the owner wants a bisectable history.)

---

### Task 3: Registry on Hour1, ticker packages removed

**Files:**
- Modify: `internal/service/backtest/drop_bounce_registry.go`
- Modify: `internal/service/backtest/drop_bounce_registry_test.go`
- Delete: `internal/service/trading_strategy/drop_bounce/strategy/{enpg,gazp,magn,moex,nlmk,nvtk,plzl,rual,tbank,vsmo}/`
- Delete: `data/params/drop_bounce/{enpg,gazp,magn,moex,nlmk,nvtk,plzl,rual,t,vsmo}/`
- Modify: `data/params/drop_bounce/grid_example.json`
- Modify: `cmd/backtest/main.go` (comment at line ~227 only if it mentions volume)

**Interfaces:**
- Produces: `var dropBounceRegistry map[string]func() core.Params` (empty); `func DropBounceLookupOrGeneric(ticker string, interval enum.Interval) (Binding, error)` — `Hour1` only.

- [ ] **Step 1: Write the failing registry tests**

Replace `drop_bounce_registry_test.go` with:

```go
package backtest

import (
	"testing"

	"tinvest/internal/enum"
	"tinvest/internal/service/trading_strategy/drop_bounce/strategy/core"
)

func TestDropBounceGenericTickerUsesCoreDefaults(t *testing.T) {
	b, err := DropBounceLookupOrGeneric("ZZZZ", enum.Hour1)
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

func TestDropBounceRunsOnHour1Only(t *testing.T) {
	for _, iv := range []enum.Interval{enum.Minutes5, enum.Minutes15, enum.Minutes30, enum.Hour4, enum.Day1, enum.Week1} {
		if _, err := DropBounceLookupOrGeneric("GAZP", iv); err == nil {
			t.Fatalf("%s: want error", iv)
		}
	}
}

func TestDropBounceParseParamsLayersOverDefaults(t *testing.T) {
	b, err := DropBounceLookupOrGeneric("ZZZZ", enum.Hour1)
	if err != nil {
		t.Fatalf("lookup: %v", err)
	}
	parsed, err := b.ParseParams([]byte(`{"RSIZone": 25, "EMAPeriod": 150}`))
	if err != nil {
		t.Fatalf("ParseParams: %v", err)
	}
	got := parsed.(core.Params)
	want := core.DefaultParams()
	want.RSIZone, want.EMAPeriod = 25, 150
	if got != want {
		t.Fatalf("got %+v, want %+v", got, want)
	}
}

// json.Unmarshal ignores unknown fields: a grid or params file written for the volume-spike rules
// runs on the defaults instead of failing. Pinned so nobody assumes the old fields still apply.
func TestDropBounceParseParamsIgnoresRemovedFields(t *testing.T) {
	b, err := DropBounceLookupOrGeneric("ZZZZ", enum.Hour1)
	if err != nil {
		t.Fatalf("lookup: %v", err)
	}
	parsed, err := b.ParseParams([]byte(`{"VolDailyMult": 0.15, "VolAvgPeriod": 10}`))
	if err != nil {
		t.Fatalf("ParseParams: %v", err)
	}
	if got := parsed.(core.Params); got != core.DefaultParams() {
		t.Fatalf("got %+v, want core.DefaultParams()", got)
	}
}

// Every registered ticker resolves to its own literal on Hour1.
func TestDropBounceTickerLiteral(t *testing.T) {
	for ticker, literal := range dropBounceRegistry {
		b, err := DropBounceLookupOrGeneric(ticker, enum.Hour1)
		if err != nil {
			t.Fatalf("%s: lookup: %v", ticker, err)
		}
		if got := b.DefaultParams().(core.Params); got != literal() {
			t.Fatalf("%s: defaults = %+v, want %+v", ticker, got, literal())
		}
	}
}
```

- [ ] **Step 2: Run to see them fail**

Run: `go test ./internal/service/backtest/ -run DropBounce`
Expected: compile failure (old registry types, ticker packages use removed fields).

- [ ] **Step 3: Delete the ticker packages and their params**

```bash
git rm -r -q internal/service/trading_strategy/drop_bounce/strategy/{enpg,gazp,magn,moex,nlmk,nvtk,plzl,rual,tbank,vsmo}
git rm -r -q data/params/drop_bounce/{enpg,gazp,magn,moex,nlmk,nvtk,plzl,rual,t,vsmo}
```

- [ ] **Step 4: Rewrite the registry**

Replace `drop_bounce_registry.go` with:

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
func dropBounceBindingFor(ticker string, interval enum.Interval, defaults func() core.Params) Binding {
	step := interval.ToTimeDuration()
	return Binding{
		DefaultParams: func() any { return defaults() },
		Build: func(params any) strategy.Strategy {
			return core.NewWithParams(ticker, step, params.(core.Params))
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

// dropBounceRegistry maps tickers with their own package to their Hour1 literal. The packages
// calibrated on the volume-spike rules (15m) were deleted on 2026-10-13 with the rewrite to the
// 1h RSI-zone rules; a ticker comes back here with a calibration on the new rules
// (skill drop-bounce-calibrate).
var dropBounceRegistry = map[string]func() core.Params{}

// DropBounceLookupOrGeneric returns the binding for a ticker: the ticker package's literal, or
// core.DefaultParams when the ticker has no package. Partial JSON overrides layer over whichever
// defaults apply. drop_bounce runs on Hour1 only; other timeframes are rejected.
func DropBounceLookupOrGeneric(ticker string, interval enum.Interval) (Binding, error) {
	if interval != enum.Hour1 {
		return Binding{}, fmt.Errorf("drop_bounce runs on Hour1 only, not %s", interval)
	}
	defaults, ok := dropBounceRegistry[ticker]
	if !ok {
		defaults = core.DefaultParams
	}
	return dropBounceBindingFor(ticker, interval, defaults), nil
}
```

- [ ] **Step 5: Rewrite `grid_example.json`**

```json
{
  "_comment": "drop_bounce (только 1h), стартовая фазовая сетка поверх core.DefaultParams(): вход — RSI(RSIPeriod) ниже RSIZone или крест RSIZone вверх не выше RSICap; фильтр — close > EMA(EMAPeriod) и ход дня < DaySpentATR дневных ATR; риск — стоп StopDailyATR дневных ATR (нуля нет); выход — RSI пересёк RSIExit вверх. Боевая сетка калибровки — scripts/grids.py скилла drop-bounce-calibrate.",
  "phases": [
    {"name": "entry", "keepTop": 3, "grid": {"RSIPeriod": [4, 5, 6, 7], "RSIZone": [25, 30, 35], "RSICap": [40, 50, 60]}},
    {"name": "filter", "keepTop": 3, "grid": {"EMAPeriod": [100, 150, 200], "DaySpentATR": [0.3, 0.5, 0.7, 1.0]}},
    {"name": "risk", "keepTop": 3, "grid": {"StopDailyATR": [0.5, 0.7, 1.0, 1.5, 2.0], "DailyATRPeriod": [7, 10, 14]}},
    {"name": "exit", "keepTop": 3, "grid": {"RSIExit": [60, 65, 70, 75, 80]}}
  ]
}
```

- [ ] **Step 6: Check `cmd/backtest/main.go`**

Read the comment near `EntryAtNextOpen: strategyName == "gap_fade" || strategyName == "drop_bounce"`. If it describes drop_bounce as triggering on a volume spike, change the drop_bounce part to "drop_bounce on a 1h bar's close". No code change.

- [ ] **Step 7: Run registry tests**

Run: `go test ./internal/service/backtest/ -run DropBounce`
Expected: still compile failure only from `drop_screen*.go` / `drop_bounce_live_env_test.go` (Tasks 4–5). If anything else fails, fix it here.

- [ ] **Step 8: Commit**

```bash
git add -A internal/service/backtest/drop_bounce_registry.go internal/service/backtest/drop_bounce_registry_test.go internal/service/trading_strategy/drop_bounce data/params/drop_bounce cmd/backtest/main.go
git commit -m "refactor(drop_bounce): Hour1-only registry, drop volume-spike ticker packages

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

---

### Task 4: Live universe config — H1 only

**Files:**
- Modify: `internal/config/drop_bounce.go`
- Modify: `internal/config/drop_bounce_test.go`
- Modify: `internal/service/backtest/drop_bounce_live_env_test.go`
- Modify: `env/prod.env`, `env/prod.env.example`, `env/local.env.example`

**Interfaces:**
- Produces: `type DropBounceConfig struct { TickersHour1 []string \`config:"DROP_BOUNCE_TICKERS_H1,backend=env"\` }`; `func (c *DropBounceConfig) Enabled() bool`.

- [ ] **Step 1: Write the failing config tests**

Replace `internal/config/drop_bounce_test.go` with:

```go
package config

import (
	"context"
	"slices"
	"testing"

	"github.com/heetch/confita"
	"github.com/heetch/confita/backend/env"
)

// Без переменной drop_bounce конфиг грузится, вселенная пуста.
func TestDropBounceConfigLoadsWithoutItsEnvVar(t *testing.T) {
	t.Setenv("DROP_BOUNCE_TICKERS_H1", "")

	cfg := NewDropBounceConfig()
	if err := confita.NewLoader(env.NewBackend()).Load(context.Background(), cfg); err != nil {
		t.Fatalf("Load без DROP_BOUNCE_TICKERS_H1 = %v, want nil", err)
	}
	if cfg.Enabled() {
		t.Fatalf("без переменной cfg = %+v, want выключено", cfg)
	}
}

func TestDropBounceConfigLoadsHour1(t *testing.T) {
	t.Setenv("DROP_BOUNCE_TICKERS_H1", "GAZP,T")

	cfg := NewDropBounceConfig()
	if err := confita.NewLoader(env.NewBackend()).Load(context.Background(), cfg); err != nil {
		t.Fatalf("Load = %v", err)
	}
	if !slices.Equal(cfg.TickersHour1, []string{"GAZP", "T"}) {
		t.Fatalf("cfg = %+v, want H1 [GAZP T]", cfg)
	}
	if !cfg.Enabled() {
		t.Fatal("заданная вселенная не включила стратегию")
	}
}

func TestDropBounceEnabledIgnoresBlankTickers(t *testing.T) {
	c := NewDropBounceConfig()
	c.TickersHour1 = []string{"", " "}
	if c.Enabled() {
		t.Fatal("вселенная из пустых строк включила стратегию")
	}
}
```

- [ ] **Step 2: Run to see them fail**

Run: `go test ./internal/config/ -run DropBounce`
Expected: PASS for the first two is possible; the point is the struct change in Step 3 — then `TickersMinutes15` uses elsewhere fail to compile. Proceed.

- [ ] **Step 3: Shrink the config**

Replace `internal/config/drop_bounce.go` with:

```go
package config

import "strings"

// DropBounceConfig — вселенная drop_bounce для будущего живого раннера. Стратегия работает только
// на 1h; тикер попадает сюда, только если прошёл калибровку (скилл drop-bounce-calibrate
// записывает его). Живого раннера пока нет — конфиг только хранит список. Переменная не
// required: ошибка загрузки конфига оборвала бы InitApp целиком.
type DropBounceConfig struct {
	TickersHour1 []string `config:"DROP_BOUNCE_TICKERS_H1,backend=env"`
}

// NewDropBounceConfig: вселенная пуста — стратегия ничего не торгует, пока её не заполнят.
func NewDropBounceConfig() *DropBounceConfig {
	return &DropBounceConfig{}
}

// Enabled — задан ли хоть один тикер. DROP_BOUNCE_TICKERS_H1= в env приходит как [""].
func (c *DropBounceConfig) Enabled() bool {
	for _, t := range c.TickersHour1 {
		if strings.TrimSpace(t) != "" {
			return true
		}
	}
	return false
}
```

Run `grep -rn "TickersMinutes15\|TickersMinutes30" --include=*.go .` — expected: no hits. Fix any.

- [ ] **Step 4: Env files**

In each of `env/prod.env`, `env/prod.env.example`, `env/local.env.example`: delete the lines `DROP_BOUNCE_TICKERS_M15=...` and `DROP_BOUNCE_TICKERS_M30=...`; set `DROP_BOUNCE_TICKERS_H1=` (empty). If a comment above them mentions 15m/30m, change it to: `# drop_bounce: вселенная 1h для будущего live-раннера (ведёт скилл drop-bounce-calibrate)`.

- [ ] **Step 5: Live-env test**

In `internal/service/backtest/drop_bounce_live_env_test.go` change

```go
// dropBounceLiveKeys — вселенные drop_bounce по таймфреймам в env/prod.env.
var dropBounceLiveKeys = []string{"DROP_BOUNCE_TICKERS_M15", "DROP_BOUNCE_TICKERS_M30", "DROP_BOUNCE_TICKERS_H1"}
```

to

```go
// dropBounceLiveKeys — вселенная drop_bounce в env/prod.env (стратегия только на 1h).
var dropBounceLiveKeys = []string{"DROP_BOUNCE_TICKERS_H1"}
```

and append this test, which keeps the removed keys from coming back:

```go
// Ключи прежних таймфреймов удалены вместе с 15m/30m-правилами: их возвращение в env значило бы,
// что кто-то торгует drop_bounce не на 1h.
func TestDropBounceNoLegacyTimeframeKeys(t *testing.T) {
	lists := dropBounceLiveTickers(t)
	for _, key := range []string{"DROP_BOUNCE_TICKERS_M15", "DROP_BOUNCE_TICKERS_M30"} {
		if _, ok := lists[key]; ok {
			t.Errorf("%s есть в env/prod.env, drop_bounce работает только на 1h", key)
		}
	}
}
```

- [ ] **Step 6: Run**

Run: `go test ./internal/config/ && go vet ./internal/app/`
Expected: PASS. (`internal/service/backtest` still waits for Task 5.)

- [ ] **Step 7: Commit**

```bash
git add internal/config/drop_bounce.go internal/config/drop_bounce_test.go internal/service/backtest/drop_bounce_live_env_test.go env/prod.env env/prod.env.example env/local.env.example
git commit -m "refactor(config): drop_bounce live universe is Hour1 only

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

---

### Task 5: Screener on the new core

**Files:**
- Modify: `internal/service/backtest/drop_screen.go:12-54`
- Modify: `internal/service/backtest/drop_screen_report.go:110-128,174`
- Modify: `internal/service/backtest/drop_screen_test.go`
- Modify: `internal/service/backtest/drop_screen_report_test.go` (interval/text)
- Modify: `cmd/dropscreen/main.go:31,83-91`

**Interfaces:**
- Consumes: `dropcore.Params` from Task 2.
- Produces: `func DropGrid(interval enum.Interval) ([]dropcore.Params, error)` — `Hour1` only, 16 configs, order `RSIZone` outermost → `RSICap` → `DaySpentATR` → `RSIExit` innermost.

- [ ] **Step 1: Write the failing grid tests**

In `drop_screen_test.go` replace `TestDropGridPerTimeframe`, `TestDropGridOrder`, `TestDropGridRejectsOtherTimeframes` with:

```go
func TestDropGridHour1(t *testing.T) {
	grid, err := DropGrid(enum.Hour1)
	if err != nil {
		t.Fatalf("DropGrid error: %v", err)
	}
	if len(grid) != 16 {
		t.Fatalf("len = %d, want 16", len(grid))
	}
	uniq := make(map[[4]float64]bool)
	for _, p := range grid {
		if p.RSIPeriod != 6 || p.EMAPeriod != 200 || p.StopDailyATR != 1.0 || p.DailyATRPeriod != 14 || p.StartHour != 7 {
			t.Fatalf("pinned fields drifted: %+v", p)
		}
		uniq[[4]float64{p.RSIZone, p.RSICap, p.DaySpentATR, p.RSIExit}] = true
	}
	if len(uniq) != 16 {
		t.Fatalf("%d distinct configurations, want 16", len(uniq))
	}
}

func TestDropGridOrder(t *testing.T) {
	grid, err := DropGrid(enum.Hour1)
	if err != nil {
		t.Fatal(err)
	}
	first, second, last := grid[0], grid[1], grid[15]
	if first.RSIZone != 25 || first.RSICap != 40 || first.DaySpentATR != 0.5 || first.RSIExit != 60 {
		t.Fatalf("grid[0] = %+v, want zone 25 cap 40 day 0.5 exit 60", first)
	}
	if second.RSIExit != 75 || second.DaySpentATR != 0.5 {
		t.Fatalf("grid[1] = %+v, want RSIExit as the innermost axis", second)
	}
	if last.RSIZone != 35 || last.RSICap != 60 || last.DaySpentATR != 1.0 || last.RSIExit != 75 {
		t.Fatalf("grid[15] = %+v, want zone 35 cap 60 day 1.0 exit 75", last)
	}
}

func TestDropGridRejectsOtherTimeframes(t *testing.T) {
	for _, iv := range []enum.Interval{enum.Minutes5, enum.Minutes15, enum.Minutes30, enum.Day1, enum.Hour4} {
		if _, err := DropGrid(iv); err == nil {
			t.Fatalf("DropGrid(%s) = nil error, want an error", iv)
		}
	}
}
```

Replace `dropFixture` (and its doc comment) with an hourly fixture, and change every `DropGrid(enum.Minutes15)` → `DropGrid(enum.Hour1)`, `Interval: enum.Minutes15` → `Interval: enum.Hour1`, `15*time.Minute` → `time.Hour`, `dailyCandlesMSK(60, 100, 2)` → `dailyCandlesMSK(100, 100, 2)` in this file:

```go
// dropFixture is 1h bars 07:00–22:00 MSK on every weekday from 2026-01-12 to 2026-03-31: a slow
// uptrend (+0.03% a bar) with a dip at 10:00–12:00 (−0.25% each) and a rebound at 13:00–15:00
// (+0.3% each). The 12:00 bar has RSI(6) ≈ 20 above EMA(200) with a day range ≈ 0.8 — under the
// gate 0.5 · daily ATR 2 = 1.0 of grid[0] — and the rebound crosses RSI 60, so grid[0] trades
// once a day after the EMA(200) lookback (600 bars) is filled.
func dropFixture() []backtest.Candle {
	msk, err := time.LoadLocation("Europe/Moscow")
	if err != nil {
		msk = time.UTC
	}
	var out []backtest.Candle
	p := 100.0
	for d := time.Date(2026, 1, 12, 0, 0, 0, 0, msk); d.Before(time.Date(2026, 4, 1, 0, 0, 0, 0, msk)); d = d.AddDate(0, 0, 1) {
		if wd := d.Weekday(); wd == time.Saturday || wd == time.Sunday {
			continue
		}
		for h := 7; h <= 22; h++ {
			o := p
			switch {
			case h >= 10 && h <= 12:
				p *= 0.9975
			case h >= 13 && h <= 15:
				p *= 1.003
			default:
				p *= 1.0003
			}
			out = append(out, backtest.Candle{Time: d.Add(time.Duration(h) * time.Hour),
				Open: o, High: max(o, p), Low: min(o, p), Close: p, Volume: 10})
		}
	}
	return out
}
```

In `TestScreenDropTickerRunsTheGridParamsVerbatim` change the comment `// vol 0.05, RSI(4)@55` to `// zone 25, cap 40, day 0.5, exit 60` and the comment about GAZP being registered to: `// The screener must grade every ticker on the grid config, registered or not, so one grid config through ScreenDropTicker must reproduce a direct engine run with that config.` In `TestScreenDropTickerFlatSeriesProducesNoSignals` change the failure text `(no red bar)` to `(RSI pinned at 50, close on the EMA)`.

- [ ] **Step 2: Run to see them fail**

Run: `go test ./internal/service/backtest/ -run 'DropGrid|ScreenDrop|DropScreen'`
Expected: compile errors on `VolDailyMult` in `drop_screen.go`.

- [ ] **Step 3: Rewrite the grid**

In `drop_screen.go` replace the `var (...)` block, `dropGridVols` and `DropGrid` with:

```go
// Pinned drop-screener axes. Like the zone screener, it answers "is this ticker worth
// calibrating", so it sweeps only the axes whose useful range depends on the paper: how deep the
// oversold zone sits and how high a cross may land, how much of the day's range is still allowed,
// and the two opposite exit shapes. See
// docs/superpowers/specs/2026-10-13-drop-bounce-h1-rsi-ema-design.md.
var (
	dropGridZones     = []float64{25, 35}
	dropGridCaps      = []float64{40, 60}
	dropGridDaySpent  = []float64{0.5, 1.0}
	dropGridRSIExits  = []float64{60, 75}
)

// DropGrid returns the 16 configurations every ticker is screened on, in a deterministic order
// (RSIZone outermost, RSIExit innermost). RSI(6), EMA(200) and the stop at 1.0 daily ATR(14) are
// pinned: the core defaults, properties of a tuned point rather than of the paper. drop_bounce
// runs on Hour1 only.
func DropGrid(interval enum.Interval) ([]dropcore.Params, error) {
	if interval != enum.Hour1 {
		return nil, fmt.Errorf("drop screener runs on Hour1 only, not %s", interval)
	}
	out := make([]dropcore.Params, 0, len(dropGridZones)*len(dropGridCaps)*len(dropGridDaySpent)*len(dropGridRSIExits))
	for _, zone := range dropGridZones {
		for _, capV := range dropGridCaps {
			for _, day := range dropGridDaySpent {
				for _, exit := range dropGridRSIExits {
					p := dropcore.DefaultParams()
					p.RSIZone, p.RSICap, p.DaySpentATR, p.RSIExit = zone, capV, day, exit
					out = append(out, p)
				}
			}
		}
	}
	return out, nil
}
```

(`DefaultParams` gives `RSIPeriod` 6, `EMAPeriod` 200, `StopDailyATR` 1.0, `DailyATRPeriod` 14, `StartHour` 7 — exactly the pinned set; `TestDropGridHour1` guards it.)

- [ ] **Step 4: Report text**

In `RenderDropScreenMarkdown` delete the `vols`/`volStr` lines and replace the `Сетка:` line with:

```go
	fmt.Fprintf(&b, "Сетка: %d конфигураций — RSIZone {25, 35} × RSICap {40, 60} × DaySpentATR {0.5, 1.0} × RSIExit {60, 75}; "+
		"прибито: RSI(6), EMA(200), стоп 1.0 дневного ATR(14), StartHour 7.\n", gridSize)
```

and the `best` line with:

```go
		best := fmt.Sprintf("зона %.0f, планка %.0f, день %g, выход %.0f", r.Best.RSIZone, r.Best.RSICap, r.Best.DaySpentATR, r.Best.RSIExit)
```

In `drop_screen_report_test.go` change `enum.Minutes15` → `enum.Hour1` and any expected `"Minutes15"` string → `"Hour1"`; update any expected `VolDailyMult`/`vol ` substring to the new text.

- [ ] **Step 5: CLI**

In `cmd/dropscreen/main.go`:

```go
		intervalS     = flag.String("interval", "Hour1", "timeframe: Hour1 (drop_bounce runs on 1h only)")
```

```go
// parseInterval accepts the one timeframe drop_bounce runs on.
func parseInterval(s string) (enum.Interval, error) {
	if strings.EqualFold(s, enum.Hour1.String()) {
		return enum.Hour1, nil
	}
	return 0, fmt.Errorf("unknown -interval %q (want Hour1: drop_bounce runs on 1h only)", s)
}
```

If `cmd/dropscreen` has a test for `parseInterval`, update its cases to: `Hour1` accepted, `Minutes15`/`Minutes30`/`Minutes5` rejected.

- [ ] **Step 6: Run the whole backtest package and build**

Run: `go test ./internal/service/backtest/ ./cmd/... && go build ./internal/... ./pkg/... ./cmd/...`
Expected: PASS. If `TestScreenDropTickerRunsTheGridParamsVerbatim` fails with "fixture produces no train trades", re-check the fixture against its doc comment (RSI ≈ 20 at 12:00, close above EMA200, day range < 1.0) by logging `Decide` on the 12:00 bars — fix the fixture, not the guard.

- [ ] **Step 7: Commit**

```bash
git add internal/service/backtest/drop_screen.go internal/service/backtest/drop_screen_report.go internal/service/backtest/drop_screen_test.go internal/service/backtest/drop_screen_report_test.go cmd/dropscreen
git commit -m "feat(dropscreen): screen the 1h RSI-zone drop_bounce

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

---

### Task 6: Calibration skill on 1h

**Files:**
- Modify: `.claude/skills/drop-bounce-calibrate/scripts/grids.py`
- Modify: `.claude/skills/drop-bounce-calibrate/scripts/report.py:64`
- Modify: `.claude/skills/drop-bounce-calibrate/scripts/live_env.py`
- Delete: `.claude/skills/drop-bounce-calibrate/scripts/overlap.py` (compared two timeframes; one is left)
- Modify: `.claude/skills/drop-bounce-calibrate/SKILL.md`

- [ ] **Step 1: `grids.py`**

Replace the docstring line `<tf> must be Minutes15: ...` with `<tf> must be Hour1: drop_bounce runs on 1h only (owner, 2026-10-13).` Replace `DEFAULTS` … `PHASES` with:

```python
DEFAULTS = dict(RSIPeriod=6, RSIZone=30, RSICap=50, EMAPeriod=200, DaySpentATR=0.7,
                StopDailyATR=1.0, DailyATRPeriod=14, RSIExit=70, StartHour=7)
ZONES = [25, 30, 35]
CAPS = [40, 50, 60]
DAY_SPENT = [0.3, 0.5, 0.7, 1.0]
STOPS = [0.5, 0.7, 1.0, 1.5, 2.0]  # daily ATRs
RSI_EXITS = [60, 65, 70, 75, 80]
# owner's lists, 2026-10-13; do not change without a request
PHASES = [
    ("entry", {"RSIPeriod": [4, 5, 6, 7], "RSIZone": ZONES, "RSICap": CAPS}),
    ("filter", {"EMAPeriod": [100, 150, 200], "DaySpentATR": DAY_SPENT}),
    ("risk", {"StopDailyATR": STOPS, "DailyATRPeriod": [7, 10, 14]}),
    ("exit", {"RSIExit": RSI_EXITS}),
]
```

In `main()`: the timeframe check becomes

```python
    if tf != "Hour1":
        sys.exit("калибруется только Hour1: drop_bounce работает только на 1h (решение владельца 2026-10-13)")
```

`calstop`: `phases = PHASES` (no call) and the `_comment` text `entry → risk → exits` → `entry → filter → risk → exit`. `plateau`: replace the two neighbour lines of the entry plateau with

```python
        e["RSIZone"] = neighbours(ZONES, p["RSIZone"], 5)
        e["DaySpentATR"] = neighbours(DAY_SPENT, p["DaySpentATR"], 0.1)
```

and its `_comment` text `(±1 шаг VolDailyMult × RSIExit)` → `(±1 шаг RSIZone × DaySpentATR)`; `plateau_entry` name stays. Stop plateau: `neighbours(STOPS, p["StopDailyATR"], 0.25)` stays. Delete `VOL`, `VOL_STEP`.

Check: `python3 -I .claude/skills/drop-bounce-calibrate/scripts/grids.py calstop ZZZZ Hour1 && python3 -I -c "import json;d=json.load(open('data/params/drop_bounce/zzzz/calstop_hour1.json'));print([p['name'] for p in d['phases']], d['phases'][0]['grid']['EMAPeriod'])"` → `['entry', 'filter', 'risk', 'exit'] [200]`. Then `rm -r data/params/drop_bounce/zzzz`.

- [ ] **Step 2: `report.py`**

Line 64: replace the key tuple with `("RSIPeriod", "RSIZone", "RSICap", "EMAPeriod", "DaySpentATR", "StopDailyATR", "RSIExit")`.

- [ ] **Step 3: `live_env.py`**

```python
"""Writes a ticker into or out of the drop_bounce live universe.

Usage: live_env.py <TICKER> [Hour1]   (no TF = remove)

drop_bounce runs on 1h only (owner, 2026-10-13): Hour1 is the only timeframe accepted. The
_M15/_M30 keys of the old rules are deleted from the files if found.

Sets membership exactly: with Hour1 the ticker ends up in DROP_BOUNCE_TICKERS_H1, without it the
ticker is taken out. Edits env/prod.env, env/prod.env.example and env/local.env.example (run from
the repo root); a missing key is appended. Order of the other tickers is kept; a new one goes last.
"""
import os, sys

KEY = "DROP_BOUNCE_TICKERS_H1"
LEGACY = {"DROP_BOUNCE_TICKERS_M15", "DROP_BOUNCE_TICKERS_M30"}
FILES = ["env/prod.env", "env/prod.env.example", "env/local.env.example"]

if len(sys.argv) < 2:
    sys.exit(__doc__)
ticker = sys.argv[1].upper()
bad = [tf for tf in sys.argv[2:] if tf != "Hour1"]
if bad:
    sys.exit(f"timeframe {bad} not traded: drop_bounce is 1h only, want Hour1")
want = bool(sys.argv[2:])

for f in FILES:
    if not os.path.exists(f):
        print(f"{f}: нет файла, пропущен")
        continue
    out, seen = [], False
    for line in open(f).read().splitlines():
        key, sep, v = line.partition("=")
        key = key.strip()
        if sep and key in LEGACY:
            continue
        if sep and key == KEY:
            seen = True
            tickers = [t.strip() for t in v.split(",") if t.strip() and t.strip() != ticker]
            if want:
                tickers.append(ticker)
            line = f"{KEY}={','.join(tickers)}"
        out.append(line)
    if not seen:
        out.append(f"{KEY}={ticker if want else ''}")
    open(f, "w").write("\n".join(out) + "\n")
    print(f"{f}: " + "; ".join(l for l in out if l.split("=")[0] == KEY))
```

Check on a copy: `d=$(mktemp -d -p /tmp/claude-1000); mkdir $d/env; printf 'A=1\nDROP_BOUNCE_TICKERS_M15=NVTK\nDROP_BOUNCE_TICKERS_H1=\n' > $d/env/prod.env; (cd $d && python3 -I $OLDPWD/.claude/skills/drop-bounce-calibrate/scripts/live_env.py gazp Hour1); cat $d/env/prod.env` → `A=1` and `DROP_BOUNCE_TICKERS_H1=GAZP`, no `_M15` line.

- [ ] **Step 4: Delete `overlap.py`**

`git rm .claude/skills/drop-bounce-calibrate/scripts/overlap.py`; `grep -n overlap .claude/skills/drop-bounce-calibrate/SKILL.md` and remove every mention.

- [ ] **Step 5: `SKILL.md`**

Edit the skill to the new rules; every other section (planka A/B/C/D, consensus point, plateau, costs, single question) stays word for word.

- Front-matter `description`: `Калибровка стратегии drop_bounce под один тикер на 1h с решением, торговать ли его. …` (rest unchanged).
- Title: `# drop_bounce: калибровка тикера на 1h и решение, торговать ли`. Screener reference: `cmd/dropscreen -interval Hour1`.
- §0 first bullet becomes: `**Таймфрейм — только Hour1** (решение владельца 2026-10-13: drop_bounce переписан под 1h — RSI-зона над EMA при неизрасходованном дневном ходе). Другие таймфреймы ядро отвергает.` Remove the sentence about `ParamsMinutes5/30/Hour1` literals.
- §0 window bullet: `**Окно 36 месяцев**, walk-forward 12/6 (4 фолда).` — drop the reason about volume spikes; replace with `Окно не сокращай: на 1h сделок мало, короче — фолды вырождаются.`
- All `-interval Minutes15` → `-interval Hour1`, `Minutes15` paths/filenames (`cal_minutes15_baseline`, `calstop_minutes15`, `point_minutes15`, `plateau_minutes15_*`, `reports/.../Minutes15`) → `Hour1` / `hour1`.
- §1 package: `один литерал \`Params()\` — явно выписанные \`core.DefaultParams()\`; тест \`TestLiteralIsPinned\` со снимком; строка \`<pkg>.Ticker: <pkg>.Params\` в \`dropBounceRegistry\` (\`TestDropBounceTickerLiteral\` проверит сам); тикер в списке drop_bounce в \`CLAUDE.md\`.` Remove "Образец — пакет `tbank`" (deleted); say `Образец — пакет rsi_zone любого тикера (одна функция литерала + снимок).`
- §2 baseline: remove the sentence about the 0.4-day volume threshold; keep "Baseline не судья".
- §3 phases paragraph becomes:
  `\`grids.py calstop <T> Hour1\` → \`calstop_hour1.json\` (entry → filter → risk → exit, keepTop 3): entry — \`RSIPeriod\` (4, 5, 6, 7) × \`RSIZone\` (25, 30, 35) × \`RSICap\` (40, 50, 60); filter — \`EMAPeriod\` (100, 150, 200) × \`DaySpentATR\` (0.3, 0.5, 0.7, 1.0); risk — \`StopDailyATR\` (0.5, 0.7, 1.0, 1.5, 2.0) × \`DailyATRPeriod\` (7, 10, 14); exit — \`RSIExit\` (60, 65, 70, 75, 80). \`StartHour\` всегда 7. **Эти списки задал владелец 2026-10-13.** Сетка одна для всех тикеров: не меняй её без просьбы владельца.`
  Delete the GREENVOL paragraph and the 2026-10-12 list paragraph.
- §3 "Почему не счёт фолдов": replace `(их задают всплески объёма)` with `(на 1h их немного)`.
- §4.3 plateau: `плато по входу и фильтру (±1 шаг RSIZone × DaySpentATR) и по стопу`.
- §4 end: `Прошёл §3 и §4 целиком — торговать на 1h; иначе — не торговать.`
- §5: `\`Params()\` = \`point_hour1.json\``; universe `DROP_BOUNCE_TICKERS_H1`; commands `live_env.py <T> Hour1` / `live_env.py <T>`; the sentence about removing from `_M30`/`_H1` becomes `Скрипт ставит членство точно и удаляет ключи прежних таймфреймов _M15/_M30, если найдёт.`
- §6: `Одна строка таблицы 1h …` and `DROP_BOUNCE_TICKERS_H1`.
- Red flags: `«15m не взял, проверю 30m/1h»` → `«1h не взял, проверю 15m/30m»` / `Ядро работает только на 1h. Другой таймфрейм — подгонка.`; `«Давай по темам, как в rsi_zone» | Работает лишь над прибыльным baseline. Здесь baseline < 1.` → `| Темы меряют ось поверх дефолтов; фазовая сетка меряет связку. Судья — фазовый WF.`; `DROP_BOUNCE_TICKERS_M15` → `DROP_BOUNCE_TICKERS_H1`.

Check: `grep -nE "Minutes15|M15|VolDaily|VolAvg|GREENVOL|объём|15m" .claude/skills/drop-bounce-calibrate/SKILL.md` → no hits (except a historical note if you kept one deliberately — then say so in the commit body).

- [ ] **Step 6: Commit**

```bash
git add -A .claude/skills/drop-bounce-calibrate
git commit -m "chore(skill): drop-bounce-calibrate on the 1h RSI-zone rules

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

---

### Task 7: Docs, CLAUDE.md, final gate

**Files:**
- Modify: `docs/drop_bounce/strategy.md` (full rewrite)
- Modify: `docs/drop_bounce/screener.md` (grid, interval, launch)
- Modify: `CLAUDE.md` (the `drop_bounce` entry in Layout)

- [ ] **Step 1: `docs/drop_bounce/strategy.md`**

Replace the whole file with:

````markdown
# drop_bounce — RSI-зона над EMA при неизрасходованном дневном ходе

Бэктест-стратегия, только лонг, таймфрейм **только 1h** (`-interval Hour1`, другие cmd/backtest отклоняет). Позиция может переноситься через ночь. Код: `internal/service/trading_strategy/drop_bounce/strategy/core`, привязка к бэктесту — `internal/service/backtest/drop_bounce_registry.go`. Спека: `docs/superpowers/specs/2026-10-13-drop-bounce-h1-rsi-ema-design.md`. Прежняя версия (красная свеча на всплеске объёма, 15m) заменена 2026-10-13, история — в git.

## Вход

На закрытии часовой свечи, все условия вместе:

1. **Окно:** свеча Пн–Пт МСК, начинается не раньше `StartHour:00`, свеча исполнения (через час) лежит в ту же дату. Сигнал на последней свече дня на утро не переносится.
2. **Тренд:** `close > EMA(EMAPeriod)`, строго. Пока EMA не прогрелась (меньше `EMAPeriod` свечей), входа нет.
3. **Зона RSI** (`RSI(RSIPeriod)`, Уайлдер), одна из веток:
   - **ZONE** — `RSI < RSIZone`;
   - **CROSS** — RSI пересёк `RSIZone` снизу вверх на этой свече (`prev < RSIZone ≤ cur`) и `cur ≤ RSICap`.
4. **Дневной ход не израсходован:** `used = TodayHigh − TodayLow` (диапазон текущего дня МСК по эту свечу включительно); вход только при `used < DaySpentATR · ATR`. Ровно на пороге день считается израсходованным. Нет дневного ATR или битый экстент дня — входа нет.
5. Стоп от close сигнала выше нуля.

Дневной ATR — Уайлдер `DailyATRPeriod` по завершённым дневкам Пн–Пт (выходные сессии отброшены: они в разы уже и занижают ATR). Он уходит в сигнал (`Signal.ATR`), движок замораживает его на позицию (`Position.EntryATR`).

Покупка по open следующей свечи (`Config.EntryAtNextOpen`), `FillBy = время сигнала + 1h`: нет этой свечи в данных — сигнал отброшен. Пока позиция открыта, сигналы игнорируются. Паузы после стопа нет: следующая свеча может войти снова, если условия держатся.

## Выход

По приоритету на каждой свече:

1. **SL** — внутри свечи: `low ≤ вход − StopDailyATR · ATR`, ATR замороженный на сигнале (если позиция его не несёт — сегодняшний). Движок исполняет по `min(стоп, open)`, так что ночной гэп ниже стопа закрывается по open. `StopDailyATR = 0` выключает стоп.
2. **RSI** — на close: RSI(`RSIPeriod`) пересёк `RSIExit` снизу вверх. Продаёт при любой цене. `RSIExit = 0` выключает выход.

Выхода в конце дня нет.

## Параметры

| поле | по умолчанию | смысл |
|---|---|---|
| `RSIPeriod` | 6 | длина RSI входа и выхода; 0 — входов нет |
| `RSIZone` | 30 | уровень перепроданности: ZONE ниже, CROSS через него вверх |
| `RSICap` | 50 | CROSS засчитывается, только если RSI на свече креста ≤ этого |
| `EMAPeriod` | 200 | EMA тренда на часовых close; 0 — входов нет |
| `DaySpentATR` | 0.7 | вход, пока ход дня < это · дневной ATR; 0 — входов нет |
| `StopDailyATR` | 1.0 | стоп ниже цены входа в дневных ATR; 0 — выключен |
| `DailyATRPeriod` | 14 | длина дневного ATR Уайлдера по будним дням |
| `RSIExit` | 70 | выход при пересечении вверх; 0 — выключен |
| `StartHour` | 7 | свеча-сигнал начинается не раньше этого часа МСК |

`Lookback = max(64, 3 · EMAPeriod, 4 · RSIPeriod + 2)` — при EMA 200 это 600 часовых свечей.

## Калибровка

Фазовая сетка (`keepTop 3`), каждая фаза перебирает свои оси поверх трёх лучших точек предыдущей: entry (`RSIPeriod` × `RSIZone` × `RSICap`) → filter (`EMAPeriod` × `DaySpentATR`) → risk (`StopDailyATR` × `DailyATRPeriod`) → exit (`RSIExit`). Списки значений и процедура — скилл `drop-bounce-calibrate`; стартовая сетка — `data/params/drop_bounce/grid_example.json`.

## Пакеты тикеров

Реестр `dropBounceRegistry` держит по одному литералу `Params()` на тикер, заведённый калибровкой на этих правилах; тикер без пакета работает на `core.DefaultParams()`; JSON `-params` / `-calibrate` ложится поверх. Поля прежних правил (`VolDailyMult`, `VolAvgPeriod`) в JSON молча игнорируются.

Кандидатов на калибровку подбирает скринер `cmd/dropscreen` — `docs/drop_bounce/screener.md`.

## Вселенная для live

Живого раннера пока нет; вселенная под него — `DROP_BOUNCE_TICKERS_H1` (`config.DropBounceConfig`). Список ведёт скилл `drop-bounce-calibrate` (`scripts/live_env.py`). Тест `TestDropBounceLiveTickersAreRegistered` требует, чтобы каждый тикер из `env/prod.env` имел пакет в `dropBounceRegistry`.

## Запуск

```bash
go run ./cmd/backtest -strategy drop_bounce -interval Hour1 -ticker GAZP -months 36
go run ./cmd/backtest -strategy drop_bounce -interval Hour1 -ticker GAZP -months 36 \
  -calibrate data/params/drop_bounce/grid_example.json -metric profit_factor
```
Для калибровочных прогонов — собранный бинарь на одном ядре (см. скилл `drop-bounce-calibrate`).
````

- [ ] **Step 2: `docs/drop_bounce/screener.md`**

- Spec line: add `Правила стратегии с 2026-10-13: docs/superpowers/specs/2026-10-13-drop-bounce-h1-rsi-ema-design.md.`
- «Зачем»: `три таймфрейма` → `фазовый walk-forward на одном ядре`.
- «Сетка» section replaced with:

```markdown
## Сетка

Только 1h. 16 конфигураций: `RSIZone` {25, 35} × `RSICap` {40, 60} × `DaySpentATR` {0.5, 1.0} × `RSIExit` {60, 75}.

Прибито: `RSIPeriod` 6, `EMAPeriod` 200, `StopDailyATR` 1.0, `DailyATRPeriod` 14, `StartHour` 7 (дефолты ядра). Двигаются только оси, чей полезный диапазон зависит от бумаги: глубина зоны перепроданности и высота креста, доля дневного хода, при которой ещё входим, и выход по RSI — быстрый на откате и удерживающий. Стоп и длины индикаторов — свойства настроенной точки, поэтому в сетку не входят.
```

- Launch: `-interval Minutes15` → `-interval Hour1`; flags line `-interval` (`Minutes15 | Minutes30 | Hour1; 5m не скринится`) → `-interval` (`Hour1` — единственный).
- «Дальше»: delete the sentence `Скринер гоняется отдельно на каждом таймфрейме; …`.
- `grep -nE "Minutes15|Minutes30|VolDaily|VolAvg|объём" docs/drop_bounce/screener.md` — only turnover (оборот) mentions may remain.

- [ ] **Step 3: `CLAUDE.md`**

Replace the whole `` `drop_bounce` — backtest-only long after a red bar … docs/drop_bounce/screener.md. `` sentence group (from `` `drop_bounce` `` up to and including `docs/drop_bounce/screener.md.` before `Note: backtest-only strategies`) with:

```
`drop_bounce` — backtest-only 1h long (rewritten 2026-10-13; the 15m volume-spike version and its ticker packages are in git history): on a Mon–Fri bar from `StartHour` (7) MSK, buy at the next bar's open when the close is above EMA(`EMAPeriod`=200) and RSI(`RSIPeriod`=6) is below `RSIZone` (30) or has just crossed it upward staying ≤ `RSICap` (50), and the day's range so far is below `DaySpentATR` (0.7) daily ATRs (Wilder `DailyATRPeriod`=14 over weekday dailies, frozen at the signal); exits — stop `StopDailyATR` (1.0) daily ATRs below the fill, RSI crossing `RSIExit` (70) upward; holds overnight (`-strategy drop_bounce -interval Hour1`, docs: `docs/drop_bounce/strategy.md`); calibration is a phased grid entry → filter → risk → exit (skill `drop-bounce-calibrate`), no ticker packages yet, starter grid `data/params/drop_bounce/grid_example.json`; live-раннера нет, вселенная под него — `DROP_BOUNCE_TICKERS_H1` (`config.DropBounceConfig`), её ведёт скилл (`scripts/live_env.py`), тикер обязан иметь пакет в реестре (`TestDropBounceLiveTickersAreRegistered`); подбор тикеров — `cmd/dropscreen -interval Hour1` (шортлист `DROP_BOUNCE_CANDIDATES`), docs/drop_bounce/screener.md.
```

- [ ] **Step 4: Final gate**

Run: `./bin/mage ci`
Expected: lint + `go test -race ./...` + mock-drift all green. Also: `grep -rnE "VolDailyMult|VolAvgPeriod|TickersMinutes15|DROP_BOUNCE_TICKERS_M15" --include=*.go --include=*.md --include=*.py --include=*.env* . | grep -v docs/superpowers` → no hits.

- [ ] **Step 5: Smoke run on real candles (one core)**

```bash
GOMAXPROCS=1 nice -n 19 go build -o reports/drop_bounce_cal/backtest ./cmd/backtest
taskset -c 0 nice -n 19 env GOMAXPROCS=1 reports/drop_bounce_cal/backtest -strategy drop_bounce -interval Hour1 -ticker GAZP -months 36 -out reports/GAZP_drop_bounce/smoke/Hour1
```
Expected: the run completes and reports a non-zero number of trades with entry reasons starting `ZONE:`/`CROSS:`. Report the trade count and PF to the owner as a smoke fact, not a verdict. If it reports 0 trades, do not tune anything — report it.

- [ ] **Step 6: Commit**

```bash
git add docs/drop_bounce CLAUDE.md
git commit -m "docs(drop_bounce): 1h RSI-zone rules, screener and CLAUDE.md

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

- [ ] **Step 7: Memory (controller only, after the branch review)**

Update the auto-memory: mark `feedback_drop_bounce_15m_only.md` and `feedback_drop_bounce_grid.md` as superseded by the 1h rewrite (new grid = the phase lists above, set by the owner 2026-10-13); add a project memory `project_drop_bounce_h1_rewrite.md` pointing to the spec and plan; update the MEMORY.md lines.
