# RSI+EMA Entry Quality Filters Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add two optional, default-off entry sub-filters to the `rsi_ema` strategy — a "saw/chop" filter counting RSI crossings of the mid line, and a volume-regime filter comparing recent to background volume — and flip the existing fresh-entry filter to default-off so the grid can judge every quality filter against a filter-off control.

**Architecture:** All changes live in one pure, stateless core file (`internal/service/trading_strategy/rsi_ema/strategy/core/core.go`). Both new gates sit on the ENTRY path only, after the existing cross+trend confirmation, and are computed from data already present in `strategy.MarketData` (`RSISeries` and `md.Volumes`) — no new inter-bar state, no lookahead, no changes to exits. Each sub-filter is an independent predicate method with its own disable knob, composed by `freshEntry` (RSI sub-filters) or checked directly in `enter` (volume). Diagnostics go through `Explain`; sweeping happens through two new phases in `data/params/rsi_ema/grid.json`.

**Tech Stack:** Go 1.25, standard library only (`fmt`, `strconv`, `time`), existing in-repo helpers `indicators.RSISeries`, `ema.Compute`, package-local `mskLoc`/`isWeekend`. Tests are plain `go test` table-driven tests in `core_test.go`. Quality gate: `./bin/mage ci` (golangci-lint v2 + `go test -race ./...` + mock drift).

**Spec:** `docs/superpowers/specs/2026-07-25-rsi-ema-entry-quality-filters-design.md`

## Global Constraints

- **All entry quality filters are OPTIONAL and OFF by default.** `DefaultParams()` must trade the plain "cross + trend" entry. New defaults: `EntryAboveMidLimit = 0`, `EntryMaxMidCrossings = 0`, `UseVolume = 0`, `VolShortPeriod = 10`, `VolLongPeriod = 50`, `VolMult = 1.0`, `EntryLookbackBars` stays `5`.
- **Never block an entry on missing data.** Any gate whose inputs are absent, misaligned or unusable must degrade to "allow".
- **No lookahead.** Window computations may read `rsi[start..i-1]` only; the entry bar `i` is never part of an RSI window. The volume windows deliberately DO include bar `i` (its own volume is known at close).
- **Core stays pure and stateless between bars**, ticker-agnostic and timeframe-agnostic (all windows measured in bars).
- **Exits are untouched.** `manage()` and the `SL`/`EMAX`/`RSI70`/`RSI50`/`EOD` precedence must not change.
- **Grid keys must match `Params` field names char-for-char** — calibration sweeps them by reflection.
- All `Params` fields stay `int` or `float64` (reflection grid requirement). Booleans are expressed as int toggles (`UseVolume`), like `reversion` does.
- Comments and identifiers in Go code are in English (matches the file); user-facing strings (`EntryReason`, `Explain`, docs) are in Russian.
- Run scoped tests with `go test -race ./internal/service/trading_strategy/rsi_ema/... -v`. Do NOT run full `./bin/mage ci` expecting green: an unrelated uncommitted WIP in `internal/service/trading_strategy/scalping_rsimacd/strategy/core/core.go` currently fails it. Leave that file alone.

## File Structure

- `internal/service/trading_strategy/rsi_ema/strategy/core/core.go` (modify) — `Params` fields + defaults, `midCrossings`, `freshByBarsAbove`, `freshByCrossings`, `freshEntry` composition, `avgVolumeLastN`, `volumeRegimeOK`, the gate call in `enter`, `limitLabel`, `Explain` lines. Single core file, ~420 lines today, ~530 after; consistent with the sibling `scalping_rsimacd` core — do NOT split it.
- `internal/service/trading_strategy/rsi_ema/strategy/core/core_test.go` (modify) — new unit + integration tests, plus updates to four existing tests that assumed the fresh-entry filter was on by default.
- `data/params/rsi_ema/grid.json` (modify) — two new phases (`chop`, `volume`), updated `_comment` and combo count.
- `docs/rsi_ema/strategy.md` (modify) — entry rules, parameter list, grid phases/count.
- `docs/superpowers/specs/2026-07-23-rsi-ema-design.md` (modify) — one cross-reference line to the new spec.

**Out of scope:** `docs/rsi_ema/backtest-commands.md` is untracked and left untouched; the `scalping_rsimacd` WIP is left untouched; no calibration run is part of this plan.

---

### Task 1: Fresh-entry filter becomes default-off

Flips `EntryAboveMidLimit` from `3` to `0` so the baseline strategy trades the plain entry, and repairs the four existing tests that silently relied on the filter being on by default.

**Files:**
- Modify: `internal/service/trading_strategy/rsi_ema/strategy/core/core.go:50` (field comment), `:70` (default value)
- Test: `internal/service/trading_strategy/rsi_ema/strategy/core/core_test.go:432-501`

**Interfaces:**
- Consumes: nothing from earlier tasks.
- Produces: `DefaultParams().EntryAboveMidLimit == 0`; test helper `freshParams() Params` (defaults with `EntryAboveMidLimit = 3`) used by Task 2's tests.

- [ ] **Step 1: Update the failing tests**

In `core_test.go`, replace the body of `TestDefaultParamsFreshEntryFilter` (currently at line 432) with the new expectation and add the helper right above it:

```go
// freshParams returns the defaults with the bars-above-mid sub-filter explicitly ON.
// DefaultParams ships every entry quality filter off, so tests that exercise the filter must
// switch it on themselves.
func freshParams() Params {
	p := DefaultParams()
	p.EntryAboveMidLimit = 3
	return p
}

func TestDefaultParamsFreshEntryFilter(t *testing.T) {
	p := DefaultParams()
	if p.EntryLookbackBars != 5 {
		t.Fatalf("EntryLookbackBars = %d want 5", p.EntryLookbackBars)
	}
	if p.EntryAboveMidLimit != 0 {
		t.Fatalf("EntryAboveMidLimit = %d want 0 (quality filters are off by default)", p.EntryAboveMidLimit)
	}
}
```

In `TestFreshEntryFilter`, switch the strategy to the explicit-on params:

```go
	s := NewWithParams("TEST", freshParams()) // window 5, limit 3
```

In `TestFreshEntryFilterDisabled`, the second half must start from the ON params, otherwise it asserts nothing:

```go
func TestFreshEntryFilterDisabled(t *testing.T) {
	p := freshParams()
	p.EntryAboveMidLimit = 0 // off
	s := NewWithParams("TEST", p)
	if !s.freshEntry([]float64{55, 55, 55, 55, 47}, 5) {
		t.Fatalf("EntryAboveMidLimit<=0 must disable the filter")
	}
	p2 := freshParams()
	p2.EntryLookbackBars = 0 // off
	s2 := NewWithParams("TEST", p2)
	if !s2.freshEntry([]float64{55, 55, 55, 55, 47}, 5) {
		t.Fatalf("EntryLookbackBars<=0 must disable the filter")
	}
}
```

In `TestEnterFilterRejectsChopReentry`, the filter-off arm is now just the defaults — which also asserts the spec's "at defaults the chop re-entry is bought". Replace the first four lines of the body:

```go
	sOff := NewWithParams("TEST", DefaultParams()) // defaults: every quality filter off
	on := NewWithParams("TEST", freshParams())     // filter on (window 5, limit 3)
	closes, highs, lows := driftWalk(1500, 7)
	end := mskAt(2026, 7, 20, 12, 0)
```

and inside the loop replace the two `DefaultParams().EntryAboveMidLimit` references with `freshParams().EntryAboveMidLimit`.

- [ ] **Step 2: Run the tests to verify they fail**

Run: `go test -race ./internal/service/trading_strategy/rsi_ema/... -run 'TestDefaultParamsFreshEntryFilter|TestFreshEntryFilter' -v`
Expected: FAIL — `EntryAboveMidLimit = 3 want 0 (quality filters are off by default)`.

- [ ] **Step 3: Flip the default**

In `core.go`, change the field comment and the default:

```go
	EntryAboveMidLimit int // reject entry when >= this many bars in the window are above RSIMid; <=0 disables (grid; default 0 = off)
```

```go
		EntryLookbackBars:  5,
		EntryAboveMidLimit: 0,
```

- [ ] **Step 4: Run the whole package to verify it passes**

Run: `go test -race ./internal/service/trading_strategy/rsi_ema/... -v`
Expected: PASS, all tests.

- [ ] **Step 5: Commit**

```bash
git add internal/service/trading_strategy/rsi_ema/strategy/core/core.go internal/service/trading_strategy/rsi_ema/strategy/core/core_test.go
git commit -m "feat(rsi_ema): фильтр свежести входа выключен по умолчанию"
```

---

### Task 2: Chop filter — RSI mid-line crossings

Adds the `EntryMaxMidCrossings` sub-filter: count how many times RSI crossed `RSIMid` inside the lookback window before the entry bar, reject when there are too many. Closes the hole the bars-above-mid counter leaves for a genuine saw (`52, 48, 51, 49, 49` counts only 2 bars above 50 yet crosses the mid 3 times).

**Files:**
- Modify: `internal/service/trading_strategy/rsi_ema/strategy/core/core.go` — `Params` (after `EntryAboveMidLimit`), `DefaultParams`, new `midCrossings`/`freshByBarsAbove`/`freshByCrossings`, rewritten `freshEntry` (currently `:219-224`), `limitLabel`, `Explain` (currently `:402-405`)
- Test: `internal/service/trading_strategy/rsi_ema/strategy/core/core_test.go`

**Interfaces:**
- Consumes: `freshParams()` and `DefaultParams().EntryAboveMidLimit == 0` from Task 1; existing `barsAboveMid(rsi []float64, i int) int`, `freshEntry(rsi []float64, i int) bool`.
- Produces:
  - `Params.EntryMaxMidCrossings int` (default `0`)
  - `func (s *Strategy) midCrossings(rsi []float64, i int) int`
  - `func (s *Strategy) freshByBarsAbove(rsi []float64, i int) bool`
  - `func (s *Strategy) freshByCrossings(rsi []float64, i int) bool`
  - `func limitLabel(v int) string` — `"выключен"` when `v <= 0`, else the decimal number; reused by Task 3's Explain work.
  - `freshEntry` keeps its signature; it now ANDs both sub-filters.

- [ ] **Step 1: Write the failing unit tests**

Append to `core_test.go`:

```go
func TestDefaultParamsChopFilterOff(t *testing.T) {
	if got := DefaultParams().EntryMaxMidCrossings; got != 0 {
		t.Fatalf("EntryMaxMidCrossings = %d want 0 (filter off by default)", got)
	}
}

func TestMidCrossings(t *testing.T) {
	s := NewWithParams("TEST", DefaultParams()) // window 5, RSIMid 50
	cases := []struct {
		name string
		rsi  []float64 // indices 0..i-1 form the window when i = len(rsi)
		want int
	}{
		{"saw crosses three times", []float64{52, 48, 51, 49, 49}, 3},
		{"quiet below the mid", []float64{48, 49, 49, 49, 49}, 0},
		{"single cross down", []float64{55, 55, 55, 55, 47}, 1},
		{"warm-up zeros do not count", []float64{0, 0, 0, 55, 47}, 1},
		{"short history truncates, no panic", []float64{55, 47}, 1},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := s.midCrossings(tc.rsi, len(tc.rsi)); got != tc.want {
				t.Fatalf("midCrossings = %d want %d", got, tc.want)
			}
		})
	}
}

func TestChopFilterRejectsSaw(t *testing.T) {
	p := DefaultParams()
	p.EntryMaxMidCrossings = 3
	s := NewWithParams("TEST", p)
	// saw crosses the mid line 3 times, twoCross only twice.
	saw := []float64{52, 48, 51, 49, 49}
	twoCross := []float64{52, 48, 51, 51, 51}
	if s.freshEntry(saw, len(saw)) {
		t.Fatalf("saw with 3 >= 3 crossings must be rejected")
	}
	if !s.freshEntry(twoCross, len(twoCross)) {
		t.Fatalf("2 < 3 crossings must be allowed")
	}
}

func TestChopFilterDisabled(t *testing.T) {
	saw := []float64{52, 48, 51, 49, 49}
	s := NewWithParams("TEST", DefaultParams()) // EntryMaxMidCrossings = 0
	if !s.freshEntry(saw, len(saw)) {
		t.Fatalf("EntryMaxMidCrossings<=0 must disable the chop filter")
	}
	p := DefaultParams()
	p.EntryMaxMidCrossings = 3
	p.EntryLookbackBars = 0 // the shared window switch disables both sub-filters
	if !NewWithParams("TEST", p).freshEntry(saw, len(saw)) {
		t.Fatalf("EntryLookbackBars<=0 must disable both sub-filters")
	}
}

// TestEntrySubFiltersAreIndependent pins that each sub-filter cuts only its own pattern: the
// chop filter rejects the saw and passes the dip-after-a-run, the bars-above-mid filter does
// the opposite.
func TestEntrySubFiltersAreIndependent(t *testing.T) {
	saw := []float64{52, 48, 51, 49, 49} // 3 crossings, 2 bars above
	dip := []float64{55, 55, 55, 55, 47} // 1 crossing, 4 bars above

	chopOnly := DefaultParams()
	chopOnly.EntryMaxMidCrossings = 3
	sChop := NewWithParams("TEST", chopOnly)
	if sChop.freshEntry(saw, len(saw)) {
		t.Fatalf("chop filter must reject the saw")
	}
	if !sChop.freshEntry(dip, len(dip)) {
		t.Fatalf("chop filter must not reject the dip (only 1 crossing)")
	}

	aboveOnly := freshParams() // EntryAboveMidLimit = 3, EntryMaxMidCrossings = 0
	sAbove := NewWithParams("TEST", aboveOnly)
	if sAbove.freshEntry(dip, len(dip)) {
		t.Fatalf("bars-above-mid filter must reject the dip")
	}
	if !sAbove.freshEntry(saw, len(saw)) {
		t.Fatalf("bars-above-mid filter must not reject the saw (only 2 bars above)")
	}
}
```

- [ ] **Step 2: Run the tests to verify they fail**

Run: `go test -race ./internal/service/trading_strategy/rsi_ema/... -run 'Chop|MidCrossings|SubFilters' -v`
Expected: FAIL to compile — `p.EntryMaxMidCrossings undefined`, `s.midCrossings undefined`.

- [ ] **Step 3: Implement the chop sub-filter**

In `core.go`, add the param right after `EntryAboveMidLimit` in the struct:

```go
	EntryMaxMidCrossings int // reject entry when the window holds >= this many RSIMid crossings; <=0 disables (grid; default 0 = off)
```

and in `DefaultParams`, right after `EntryAboveMidLimit: 0,`:

```go
		EntryMaxMidCrossings: 0,
```

Add `midCrossings` immediately after `barsAboveMid`:

```go
// midCrossings counts how many times RSI crossed RSIMid — in EITHER direction — between
// consecutive bars inside the EntryLookbackBars window before bar i (indices
// i-EntryLookbackBars .. i-1, clamped at 0). The entry cross itself (i-1 -> i) lies outside the
// window and is never counted, so the maximum is EntryLookbackBars-1. The rsi[j-1] > 0 guard
// rejects RSISeries warm-up zeros (same discipline as crossedUp/crossedDown); a bar sitting
// exactly at RSIMid is treated as "no crossing".
func (s *Strategy) midCrossings(rsi []float64, i int) int {
	start := i - s.p.EntryLookbackBars
	if start < 0 {
		start = 0
	}
	end := i
	if end > len(rsi) {
		end = len(rsi)
	}
	n := 0
	for j := start + 1; j < end; j++ {
		prev, cur := rsi[j-1], rsi[j]
		if prev <= 0 {
			continue
		}
		if (prev < s.p.RSIMid && cur > s.p.RSIMid) || (prev > s.p.RSIMid && cur < s.p.RSIMid) {
			n++
		}
	}
	return n
}
```

Replace `freshEntry` (currently the whole `:215-224` block) with the two predicates plus the composition:

```go
// freshByBarsAbove reports whether the bars-above-mid sub-filter allows the entry: fewer than
// EntryAboveMidLimit of the preceding bars sat above RSIMid. Rejects choppy re-entries where RSI
// only briefly dipped below the mid line after a sustained run above it. Off (always true) when
// EntryAboveMidLimit<=0.
func (s *Strategy) freshByBarsAbove(rsi []float64, i int) bool {
	return s.p.EntryAboveMidLimit <= 0 || s.barsAboveMid(rsi, i) < s.p.EntryAboveMidLimit
}

// freshByCrossings reports whether the chop sub-filter allows the entry: fewer than
// EntryMaxMidCrossings crossings of RSIMid inside the window. Rejects saws where RSI flips
// across the mid line every couple of bars. Off (always true) when EntryMaxMidCrossings<=0.
func (s *Strategy) freshByCrossings(rsi []float64, i int) bool {
	return s.p.EntryMaxMidCrossings <= 0 || s.midCrossings(rsi, i) < s.p.EntryMaxMidCrossings
}

// freshEntry reports whether the RSI cross at bar i is "fresh" — it must clear BOTH sub-filters.
// EntryLookbackBars<=0 disables the whole gate; each sub-filter also has its own off switch, and
// both are off in DefaultParams (the grid turns them on).
func (s *Strategy) freshEntry(rsi []float64, i int) bool {
	if s.p.EntryLookbackBars <= 0 {
		return true
	}
	return s.freshByBarsAbove(rsi, i) && s.freshByCrossings(rsi, i)
}
```

- [ ] **Step 4: Run the tests to verify they pass**

Run: `go test -race ./internal/service/trading_strategy/rsi_ema/... -v`
Expected: PASS, all tests.

- [ ] **Step 5: Write the failing integration test**

The unit tests exercise `freshEntry` directly; this one proves the gate is wired onto the real entry path. Append to `core_test.go`:

```go
// TestEnterFilterRejectsSawEntry finds a bar the defaults (all filters off) buy but whose
// preceding window holds >= EntryMaxMidCrossings mid-line crossings, and asserts the chop-ON
// strategy rejects that exact bar.
func TestEnterFilterRejectsSawEntry(t *testing.T) {
	off := NewWithParams("TEST", DefaultParams()) // every quality filter off
	chop := DefaultParams()
	chop.EntryMaxMidCrossings = 3
	on := NewWithParams("TEST", chop)
	closes, highs, lows := driftWalk(1500, 7)
	end := mskAt(2026, 7, 20, 12, 0)
	for k := 60; k < len(closes); k++ {
		md := mdEndingAt(closes, highs, lows, k, end, nil)
		if off.Decide(md).Kind != model.SignalBuy {
			continue
		}
		rsi := indicators.RSISeries(md.Closes, DefaultParams().RSIPeriod)
		if on.midCrossings(rsi, k) >= chop.EntryMaxMidCrossings {
			if on.Decide(md).Kind == model.SignalBuy {
				t.Fatalf("saw entry at bar %d must be filtered out", k)
			}
			return // found and verified a saw candidate
		}
	}
	t.Fatalf("no saw candidate found in series (adjust the driftWalk seed)")
}
```

- [ ] **Step 6: Run it**

Run: `go test -race ./internal/service/trading_strategy/rsi_ema/... -run TestEnterFilterRejectsSawEntry -v`
Expected: PASS — `freshEntry` is already wired into `enter` at the `3.5` gate, so no production change is needed; this test locks that wiring in.

If it fails with `no saw candidate found in series`, the seed produced no qualifying bar: try `driftWalk(1500, 3)`, then `11`, then `21`, and hard-code the first seed that yields a candidate. Do NOT weaken the assertion or lower the crossing threshold to make it pass.

- [ ] **Step 7: Write the failing Explain test**

Append to `core_test.go`:

```go
func TestExplainReportsChopFilter(t *testing.T) {
	p := DefaultParams()
	p.EntryMaxMidCrossings = 3
	s := NewWithParams("TEST", p)
	closes, highs, lows := driftWalk(800, 1)
	out := s.Explain(mdEndingAt(closes, highs, lows, 200, mskAt(2026, 7, 20, 12, 0), nil))
	if !strings.Contains(out, "фильтр пилы") {
		t.Fatalf("Explain must report the chop filter:\n%s", out)
	}
	sOff := NewWithParams("TEST", DefaultParams())
	outOff := sOff.Explain(mdEndingAt(closes, highs, lows, 200, mskAt(2026, 7, 20, 12, 0), nil))
	if !strings.Contains(outOff, "выключен") {
		t.Fatalf("Explain must mark disabled filters as выключен:\n%s", outOff)
	}
}
```

- [ ] **Step 8: Run it to verify it fails**

Run: `go test -race ./internal/service/trading_strategy/rsi_ema/... -run TestExplainReportsChopFilter -v`
Expected: FAIL — `Explain must report the chop filter`.

- [ ] **Step 9: Add the Explain lines**

Add `"strconv"` to the import block in `core.go` (keep the stdlib group sorted: `fmt`, `math`, `sort`, `strconv`, `strings`, `time`).

Add the label helper next to `midCrossings`:

```go
// limitLabel renders an optional integer gate limit for Explain: the number when the sub-filter
// is armed, "выключен" when it is off.
func limitLabel(v int) string {
	if v <= 0 {
		return "выключен"
	}
	return strconv.Itoa(v)
}
```

In `Explain`, replace the existing freshness block:

```go
	if len(rsi) == n {
		fmt.Fprintf(&sb, "фильтр свежести: баров выше %.0f в окне %d = %d (лимит %s); прошёл? %v\n",
			s.p.RSIMid, s.p.EntryLookbackBars, s.barsAboveMid(rsi, i),
			limitLabel(s.p.EntryAboveMidLimit), s.freshByBarsAbove(rsi, i))
		fmt.Fprintf(&sb, "фильтр пилы: пересечений %.0f в окне %d = %d (лимит %s); прошёл? %v\n",
			s.p.RSIMid, s.p.EntryLookbackBars, s.midCrossings(rsi, i),
			limitLabel(s.p.EntryMaxMidCrossings), s.freshByCrossings(rsi, i))
	}
```

- [ ] **Step 10: Run the whole package**

Run: `go test -race ./internal/service/trading_strategy/rsi_ema/... -v`
Expected: PASS, all tests.

- [ ] **Step 11: Lint**

Run: `./bin/golangci-lint run ./internal/service/trading_strategy/rsi_ema/...`
Expected: no findings. (If `./bin/golangci-lint` is missing, run `./bin/mage tools` first.)

- [ ] **Step 12: Commit**

```bash
git add internal/service/trading_strategy/rsi_ema/strategy/core/core.go internal/service/trading_strategy/rsi_ema/strategy/core/core_test.go
git commit -m "feat(rsi_ema): фильтр пилы — лимит пересечений линии 50 в окне входа"
```

---

### Task 3: Volume-regime entry gate

Adds the `UseVolume` gate: the average volume of the last `VolShortPeriod` bars (including the entry bar) must hold at least `VolMult` times the `VolLongPeriod` background average, otherwise the entry into a dead tape is skipped. Off by default, and degrades to "allow" on any data problem.

**Files:**
- Modify: `internal/service/trading_strategy/rsi_ema/strategy/core/core.go` — `Params`, `DefaultParams`, new `avgVolumeLastN`/`volumeRegimeOK`, gate call in `enter` (after the `3.5` freshness gate, before the ATR block at `:255`), `Explain`
- Test: `internal/service/trading_strategy/rsi_ema/strategy/core/core_test.go`

**Interfaces:**
- Consumes: `limitLabel` from Task 2 is NOT needed here; uses the existing package-level `mskLoc`, `isWeekend(tl time.Time) bool` (note: it expects a time ALREADY converted to MSK — call `isWeekend(times[j].In(mskLoc))`), and `strategy.MarketData.Volumes []int64` / `.Times []time.Time`.
- Produces:
  - `Params.UseVolume int` (default `0`), `Params.VolShortPeriod int` (default `10`), `Params.VolLongPeriod int` (default `50`), `Params.VolMult float64` (default `1.0`)
  - `func avgVolumeLastN(vols []int64, times []time.Time, n int) (avg float64, ok bool)`
  - `func (s *Strategy) volumeRegimeOK(md strategy.MarketData) bool`
  - test helper `withVolumes(md strategy.MarketData, background, recent int64, short int) strategy.MarketData`

- [ ] **Step 1: Write the failing unit tests for the averaging helper**

Append to `core_test.go`:

```go
func TestDefaultParamsVolumeFilterOff(t *testing.T) {
	p := DefaultParams()
	if p.UseVolume != 0 {
		t.Fatalf("UseVolume = %d want 0 (filter off by default)", p.UseVolume)
	}
	if p.VolShortPeriod != 10 || p.VolLongPeriod != 50 || p.VolMult != 1.0 {
		t.Fatalf("volume defaults wrong: %+v", p)
	}
}

func TestAvgVolumeLastN(t *testing.T) {
	vols := []int64{100, 200, 300, 400}
	if avg, ok := avgVolumeLastN(vols, nil, 2); !ok || avg != 350 {
		t.Fatalf("avg = %v ok = %v want 350 true", avg, ok)
	}
	if avg, ok := avgVolumeLastN(vols, nil, 10); !ok || avg != 250 {
		t.Fatalf("window longer than the series must truncate: avg = %v ok = %v want 250 true", avg, ok)
	}
	if avg, ok := avgVolumeLastN([]int64{0, 0, 300}, nil, 3); !ok || avg != 300 {
		t.Fatalf("non-positive volumes must be ignored: avg = %v ok = %v want 300 true", avg, ok)
	}
	if _, ok := avgVolumeLastN(nil, nil, 5); ok {
		t.Fatalf("empty series must report ok=false")
	}
	if _, ok := avgVolumeLastN([]int64{0, 0}, nil, 2); ok {
		t.Fatalf("no positive sample must report ok=false")
	}
	if _, ok := avgVolumeLastN(vols, nil, 0); ok {
		t.Fatalf("non-positive window must report ok=false")
	}
}

func TestAvgVolumeLastNExcludesWeekends(t *testing.T) {
	// 2026-07-25 is a Saturday; its bar must be dropped when Times is aligned.
	times := []time.Time{
		mskAt(2026, 7, 24, 12, 0),
		mskAt(2026, 7, 25, 12, 0), // Saturday
		mskAt(2026, 7, 27, 12, 0),
	}
	vols := []int64{100, 9000, 300}
	avg, ok := avgVolumeLastN(vols, times, 3)
	if !ok || avg != 200 {
		t.Fatalf("weekend bar must be excluded: avg = %v ok = %v want 200 true", avg, ok)
	}
	// Misaligned Times → weekend exclusion is skipped, all bars count.
	avg2, ok2 := avgVolumeLastN(vols, times[:2], 3)
	if !ok2 || avg2 != (100+9000+300)/3.0 {
		t.Fatalf("misaligned Times must skip weekend exclusion: avg = %v ok = %v", avg2, ok2)
	}
}
```

- [ ] **Step 2: Run to verify failure**

Run: `go test -race ./internal/service/trading_strategy/rsi_ema/... -run 'Volume|AvgVolume' -v`
Expected: FAIL to compile — `avgVolumeLastN undefined`, `p.UseVolume undefined`.

- [ ] **Step 3: Add the params and the averaging helper**

In `core.go`, add to `Params` right after `EntryMaxMidCrossings`:

```go
	UseVolume      int     // 1 arms the volume-regime entry gate; any other value disables it (grid; default 0 = off)
	VolShortPeriod int     // recent-activity window in bars, INCLUDING the entry bar (default 10)
	VolLongPeriod  int     // background window in bars; must exceed VolShortPeriod (default 50)
	VolMult        float64 // entry requires shortAvg >= longAvg*VolMult (grid; default 1.0)
```

and to `DefaultParams` right after `EntryMaxMidCrossings: 0,`:

```go
		UseVolume:      0,
		VolShortPeriod: 10,
		VolLongPeriod:  50,
		VolMult:        1.0,
```

Add the helper right before `volumeRegimeOK` (place both after `freshEntry`):

```go
// avgVolumeLastN averages the volumes of the last n bars of vols, INCLUDING the final (entry)
// bar — unlike reversion's average, the entry bar's own volume is part of the "recent activity"
// this gate measures. When times is index-aligned to vols, weekend bars (Sat/Sun MSK) are
// dropped; when times is empty or misaligned, weekend exclusion is skipped. Non-positive volumes
// are ignored. ok is false when no sample survives — the caller must then skip the gate (never
// block an entry on missing data).
func avgVolumeLastN(vols []int64, times []time.Time, n int) (avg float64, ok bool) {
	if len(vols) == 0 || n <= 0 {
		return 0, false
	}
	lo := len(vols) - n
	if lo < 0 {
		lo = 0
	}
	haveTimes := len(times) == len(vols)
	var sum float64
	var count int
	for j := lo; j < len(vols); j++ {
		if haveTimes && isWeekend(times[j].In(mskLoc)) {
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
```

- [ ] **Step 4: Run to verify the helper tests pass**

Run: `go test -race ./internal/service/trading_strategy/rsi_ema/... -run 'Volume|AvgVolume' -v`
Expected: PASS.

- [ ] **Step 5: Write the failing gate tests**

Append to `core_test.go`:

```go
// withVolumes attaches a volume series to md: the last `short` bars carry `recent`, every
// earlier bar carries `background`.
func withVolumes(md strategy.MarketData, background, recent int64, short int) strategy.MarketData {
	n := len(md.Closes)
	vols := make([]int64, n)
	for i := range vols {
		vols[i] = background
		if i >= n-short {
			vols[i] = recent
		}
	}
	md.Volumes = vols
	return md
}

func TestVolumeRegimeGate(t *testing.T) {
	closes, highs, lows := driftWalk(300, 1)
	base := mdEndingAt(closes, highs, lows, 250, mskAt(2026, 7, 20, 12, 0), nil)

	on := DefaultParams()
	on.UseVolume = 1
	sOn := NewWithParams("TEST", on)
	// Recent 10 bars at 12000 over an 8000 background: the long window mixes both, so
	// shortAvg (12000) clears longAvg (~8800) comfortably.
	if !sOn.volumeRegimeOK(withVolumes(base, 8000, 12000, 10)) {
		t.Fatalf("a live tape must pass the volume gate")
	}
	// Recent 10 bars at 5000: shortAvg falls well under the ~7400 background.
	if sOn.volumeRegimeOK(withVolumes(base, 8000, 5000, 10)) {
		t.Fatalf("a fading tape must be rejected")
	}
	// VolMult 1.2 with a barely-elevated tape (9000 vs a ~8200 long average): 9000 clears the
	// plain average but not the 1.2x threshold.
	mult := on
	mult.VolMult = 1.2
	if NewWithParams("TEST", mult).volumeRegimeOK(withVolumes(base, 8000, 9000, 10)) {
		t.Fatalf("VolMult 1.2 must reject a merely-flat tape")
	}
}

func TestVolumeRegimeGateDegrades(t *testing.T) {
	closes, highs, lows := driftWalk(300, 1)
	base := mdEndingAt(closes, highs, lows, 250, mskAt(2026, 7, 20, 12, 0), nil)
	fading := withVolumes(base, 8000, 5000, 10) // would be rejected when armed

	if !NewWithParams("TEST", DefaultParams()).volumeRegimeOK(fading) {
		t.Fatalf("UseVolume=0 must disable the gate")
	}

	on := DefaultParams()
	on.UseVolume = 1
	if !NewWithParams("TEST", on).volumeRegimeOK(base) {
		t.Fatalf("missing Volumes must never block an entry")
	}

	bad := on
	bad.VolLongPeriod = bad.VolShortPeriod // long window must exceed short
	if !NewWithParams("TEST", bad).volumeRegimeOK(fading) {
		t.Fatalf("VolLongPeriod <= VolShortPeriod must disable the gate")
	}

	zeroVols := base
	zeroVols.Volumes = make([]int64, len(base.Closes)) // all zero → no usable sample
	if !NewWithParams("TEST", on).volumeRegimeOK(zeroVols) {
		t.Fatalf("an all-zero volume series must never block an entry")
	}
}

// TestEnterVolumeGateWiredOnEntryPath proves the gate sits on the real entry path: the same bar
// that the defaults buy is rejected once the gate is armed against a fading tape, and still
// bought when the tape is alive.
func TestEnterVolumeGateWiredOnEntryPath(t *testing.T) {
	closes, highs, lows := driftWalk(800, 1)
	sDef := NewWithParams("TEST", DefaultParams())
	k := firstBuyBar(t, sDef, closes, highs, lows)
	md := mdEndingAt(closes, highs, lows, k, mskAt(2026, 7, 20, 12, 0), nil)

	if sDef.Decide(withVolumes(md, 8000, 5000, 10)).Kind != model.SignalBuy {
		t.Fatalf("defaults must ignore the volume background")
	}

	on := DefaultParams()
	on.UseVolume = 1
	sOn := NewWithParams("TEST", on)
	if sOn.Decide(withVolumes(md, 8000, 5000, 10)).Kind == model.SignalBuy {
		t.Fatalf("armed gate must reject an entry into a fading tape")
	}
	if sOn.Decide(withVolumes(md, 8000, 12000, 10)).Kind != model.SignalBuy {
		t.Fatalf("armed gate must allow an entry on a live tape")
	}
}
```

- [ ] **Step 6: Run to verify failure**

Run: `go test -race ./internal/service/trading_strategy/rsi_ema/... -run 'VolumeRegime|EnterVolume' -v`
Expected: FAIL to compile — `s.volumeRegimeOK undefined`.

- [ ] **Step 7: Implement the gate and wire it into enter()**

Add after `avgVolumeLastN` in `core.go`:

```go
// volumeRegimeOK reports whether the volume background allows an entry: the recent average
// volume must hold at least VolMult times the longer background average, so entries into a dead
// tape are skipped. The gate degrades to "allow" whenever it is disabled (UseVolume != 1),
// misconfigured (non-positive or non-increasing windows) or unsupported by the data (no usable
// volumes) — a missing volume series must never block an entry.
func (s *Strategy) volumeRegimeOK(md strategy.MarketData) bool {
	if s.p.UseVolume != 1 || s.p.VolShortPeriod <= 0 || s.p.VolLongPeriod <= s.p.VolShortPeriod {
		return true
	}
	shortAvg, okShort := avgVolumeLastN(md.Volumes, md.Times, s.p.VolShortPeriod)
	longAvg, okLong := avgVolumeLastN(md.Volumes, md.Times, s.p.VolLongPeriod)
	if !okShort || !okLong || longAvg <= 0 {
		return true
	}
	return shortAvg >= longAvg*s.p.VolMult
}
```

In `enter`, insert the gate between the freshness gate and the ATR block:

```go
	// 3.6 volume regime: skip breakouts of the mid line on a dead tape (off by default; degrades
	// to "allow" whenever the volume data cannot support the comparison).
	if !s.volumeRegimeOK(md) {
		return sig
	}
```

- [ ] **Step 8: Run to verify they pass**

Run: `go test -race ./internal/service/trading_strategy/rsi_ema/... -v`
Expected: PASS, all tests.

- [ ] **Step 9: Write the failing Explain test**

Append to `core_test.go`:

```go
func TestExplainReportsVolumeGate(t *testing.T) {
	closes, highs, lows := driftWalk(300, 1)
	base := mdEndingAt(closes, highs, lows, 250, mskAt(2026, 7, 20, 12, 0), nil)

	if out := NewWithParams("TEST", DefaultParams()).Explain(base); !strings.Contains(out, "фон объёмов: выключен") {
		t.Fatalf("Explain must mark the volume gate as off:\n%s", out)
	}

	on := DefaultParams()
	on.UseVolume = 1
	sOn := NewWithParams("TEST", on)
	if out := sOn.Explain(withVolumes(base, 8000, 12000, 10)); !strings.Contains(out, "отношение") {
		t.Fatalf("Explain must report the volume ratio:\n%s", out)
	}
	if out := sOn.Explain(base); !strings.Contains(out, "нет данных") {
		t.Fatalf("Explain must report missing volume data:\n%s", out)
	}
}
```

- [ ] **Step 10: Run to verify failure**

Run: `go test -race ./internal/service/trading_strategy/rsi_ema/... -run TestExplainReportsVolumeGate -v`
Expected: FAIL — `Explain must mark the volume gate as off`.

- [ ] **Step 11: Add the Explain block**

In `Explain`, insert after the chop-filter block (before the `StopATR` block):

```go
	switch {
	case s.p.UseVolume != 1:
		sb.WriteString("фон объёмов: выключен (UseVolume=0)\n")
	default:
		shortAvg, okShort := avgVolumeLastN(md.Volumes, md.Times, s.p.VolShortPeriod)
		longAvg, okLong := avgVolumeLastN(md.Volumes, md.Times, s.p.VolLongPeriod)
		if okShort && okLong && longAvg > 0 {
			fmt.Fprintf(&sb, "фон объёмов: short(%d) %.0f vs long(%d) %.0f, отношение %.2f, порог %.2f; прошёл? %v\n",
				s.p.VolShortPeriod, shortAvg, s.p.VolLongPeriod, longAvg,
				shortAvg/longAvg, s.p.VolMult, s.volumeRegimeOK(md))
		} else {
			sb.WriteString("фон объёмов: нет данных → гейт пропущен\n")
		}
	}
```

- [ ] **Step 12: Run the whole package and lint**

Run: `go test -race ./internal/service/trading_strategy/rsi_ema/... -v && ./bin/golangci-lint run ./internal/service/trading_strategy/rsi_ema/...`
Expected: PASS, no lint findings.

- [ ] **Step 13: Commit**

```bash
git add internal/service/trading_strategy/rsi_ema/strategy/core/core.go internal/service/trading_strategy/rsi_ema/strategy/core/core_test.go
git commit -m "feat(rsi_ema): гейт фона объёмов на входе (по умолчанию выключен)"
```

---

### Task 4: Grid phases and documentation

Exposes both new filters to walk-forward calibration and brings the docs in line with the new defaults.

**Files:**
- Modify: `data/params/rsi_ema/grid.json`
- Modify: `docs/rsi_ema/strategy.md:19-23` (entry rules), `:63-65` (grid phases)
- Modify: `docs/superpowers/specs/2026-07-23-rsi-ema-design.md` (one cross-reference line in the entry rules)

**Interfaces:**
- Consumes: `Params.EntryMaxMidCrossings` (Task 2), `Params.UseVolume` / `Params.VolMult` (Task 3). Grid keys must match those field names exactly.
- Produces: nothing consumed by later tasks.

- [ ] **Step 1: Add the two grid phases**

In `data/params/rsi_ema/grid.json`, append after the `freshness` phase object (inside `phases`):

```json
    {
      "name": "chop",
      "grid": {
        "EntryMaxMidCrossings": [0, 2, 3]
      }
    },
    {
      "name": "volume",
      "grid": {
        "UseVolume": [0, 1],
        "VolMult": [1.0, 1.2]
      }
    }
```

- [ ] **Step 2: Update the grid `_comment`**

Replace the `_comment` value with:

```
rsi_ema phased grid. Phase 1 (entry) sweeps the indicator geometry: RSI length and the fast/slow EMA periods. Phase 2 (exits) sweeps the upper RSI exit level (crossed upward) and the RSI50 cooldown length on top of each phase-1 seed. Phase 3 (risk) sweeps the optional ATR stop (0 = stop off, plus a few multipliers) on top of each phase-2 seed. Phases 4-6 sweep the OPTIONAL entry quality filters, all of which are OFF in DefaultParams so every phase carries its own filter-off control point and the baseline competes on the leaderboard: phase 4 (freshness) sweeps the fresh-entry window/threshold (EntryAboveMidLimit=0 = off), phase 5 (chop) sweeps the RSIMid-crossings limit (EntryMaxMidCrossings=0 = off), phase 6 (volume) sweeps the volume-regime gate (UseVolume=0 = off; VolShortPeriod/VolLongPeriod stay at their defaults and are deliberately NOT swept). Note: at EntryAboveMidLimit=0 the freshness filter is off regardless of EntryLookbackBars, so its three lookback values collapse to one identical control config, and at UseVolume=0 both VolMult values collapse likewise - duplicate filter-off rows on the leaderboard are expected. RunPhases expands each phase over the previous phase's keepTop seeds: 27 (phase 1) + 6x12 (phase 2) + 6x4 (phase 3, keepTop defaults to 5) + 5x12 (phase 4) + 5x3 (phase 5) + 5x4 (phase 6) = 218 combos. The mid RSI level (50) and the session bounds are fixed by the strategy definition and are deliberately NOT swept. Judge on pooled OOS profit factor from a walk-forward, never on the in-sample best.
```

- [ ] **Step 3: Verify the grid parses and every key resolves**

Run: `go run ./cmd/backtest -ticker SBER -strategy rsi_ema -interval Minutes15 -calibrate data/params/rsi_ema/grid.json -out ./reports/SBER -months 3 -test-months 1 -min-trades 1 -metric profit_factor 2>&1 | head -30`
Expected: the run starts and reports phases without an "unknown param" / JSON parse error. A short window may produce few or zero trades — that is fine; you are checking that the grid loads and the keys bind, not the results. If the cached candles are missing, add `-refresh`.

- [ ] **Step 4: Update `docs/rsi_ema/strategy.md`**

Replace entry rule 4 (currently lines 19-23) with:

```markdown
4. **Фильтры качества входа — все опциональны и по умолчанию ВЫКЛЮЧЕНЫ** (как `StopATR`);
   включаются только гридом. Общее окно — `EntryLookbackBars` (дефолт 5) баров перед баром-крестом;
   `EntryLookbackBars ≤ 0` выключает оба RSI-фильтра сразу.
   - **Свежесть входа**: в окне считаются бары с `RSI > RSIMid`; если их `≥ EntryAboveMidLimit`
     (дефолт 0 = выкл) — вход отклоняется как перезаход-чоп (RSI недолго нырнул под 50 и тут же
     вернулся).
   - **Пила**: в окне считаются пересечения `RSIMid` в любую сторону; если их
     `≥ EntryMaxMidCrossings` (дефолт 0 = выкл) — вход отклоняется как боковик у средней линии.
     Сам крест `i-1 → i` в окно не входит, максимум пересечений — `EntryLookbackBars - 1`.
   - **Фон объёмов**: при `UseVolume = 1` (дефолт 0 = выкл) вход требует
     `shortAvg ≥ longAvg × VolMult`, где `shortAvg` — средний объём последних `VolShortPeriod`
     (10) баров включая бар входа, `longAvg` — средний за `VolLongPeriod` (50) баров. Выходные
     исключаются. Если данных нет или окна невалидны — гейт пропускается, вход не блокируется.

   Обоснование фильтра свежести — `docs/superpowers/specs/2026-07-24-rsi-ema-fresh-entry-filter-design.md`;
   пила и фон объёмов — `docs/superpowers/specs/2026-07-25-rsi-ema-entry-quality-filters-design.md`.
```

- [ ] **Step 5: Update the grid section of `docs/rsi_ema/strategy.md`**

Replace the grid paragraph (currently lines 63-65) with:

```markdown
Грид `data/params/rsi_ema/grid.json`: фазы `entry` (RSIPeriod × EMAFast × EMASlow),
`exits` (RSIUpper × EntryCooldownBars), `risk` (StopATR), `freshness`
(EntryLookbackBars × EntryAboveMidLimit), `chop` (EntryMaxMidCrossings), `volume`
(UseVolume × VolMult). 27 + 6×12 + 6×4 + 5×12 + 5×3 + 5×4 = 218 комбинаций. В каждой
фазе фильтров есть контрольная точка «выключено», совпадающая с дефолтом, — так базовая
стратегия без фильтров участвует в сравнении наравне.
```

- [ ] **Step 6: Cross-reference the new spec from the original design**

In `docs/superpowers/specs/2026-07-23-rsi-ema-design.md`, entry rule 2 ends at line 64 with the reference to the fresh-entry spec. Extend that rule — replace lines 61-64:

```markdown
   прочитался бы как «был ниже уровня» и создал фантомный крест). Крест дополнительно
   гейтится **фильтрами качества входа** (`EntryLookbackBars`/`EntryAboveMidLimit` —
   перезаходы-чоп у линии 50; `EntryMaxMidCrossings` — пила по числу пересечений линии 50;
   `UseVolume`/`VolMult` — фон объёмов). Все они опциональны и выключены по умолчанию. См.
   `docs/superpowers/specs/2026-07-24-rsi-ema-fresh-entry-filter-design.md` и
   `docs/superpowers/specs/2026-07-25-rsi-ema-entry-quality-filters-design.md`.
```

- [ ] **Step 7: Final verification**

Run: `go test -race ./internal/service/trading_strategy/rsi_ema/... && ./bin/golangci-lint run ./internal/service/trading_strategy/rsi_ema/... && go build ./internal/... ./pkg/... ./cmd/...`
Expected: tests PASS, no lint findings, build succeeds.

- [ ] **Step 8: Commit**

```bash
git add data/params/rsi_ema/grid.json docs/rsi_ema/strategy.md docs/superpowers/specs/2026-07-23-rsi-ema-design.md
git commit -m "feat(rsi_ema): фазы грида chop/volume + доки по фильтрам качества входа"
```

---

## Acceptance

- `DefaultParams()` trades the plain cross+trend entry: `EntryAboveMidLimit`, `EntryMaxMidCrossings`, `UseVolume` all `0`.
- Each of the three quality filters can be armed independently and cuts only its own pattern.
- No gate can block an entry when its input data is missing, misaligned or unusable.
- Exits are byte-for-byte unchanged.
- `go test -race ./internal/service/trading_strategy/rsi_ema/...` green; `golangci-lint` clean on the package.
- Grid loads with 218 combos and every key binds to a real `Params` field.

**Not part of this plan:** the walk-forward calibration itself. The branch stays unmerged until pooled OOS PF ≥ 1.5 with ≥ 30 OOS trades, judged with the filter-off control points as the baseline.
