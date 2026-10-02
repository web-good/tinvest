# gap_fade Screener Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** `cmd/gapscreen` picks the gap_fade universe `GAP_FADE_TICKERS` by structural liquidity of the 07:00 entry, and `cmd/gapwf` validates that universe by default.

**Architecture:** Pure core in `internal/service/backtest/gap_screen.go` (metrics + gates) and `gap_screen_report.go` (markdown); thin I/O command `cmd/gapscreen` modelled on `cmd/zonescreen`; `cmd/gapwf` switches its default env key. Signals per year come from a real engine run with shared gap_fade defaults — no second copy of the gap rule.

**Tech Stack:** Go 1.25, existing `domain/backtest` engine, `screenrun`, `CandleProvider`, `godotenv`.

**Spec:** `docs/superpowers/specs/2026-10-02-gap-fade-screener-design.md`

## Global Constraints

- Code comments in English; report text, reasons and docs in Russian.
- Morning bar = bar whose start time is exactly 07:00 Europe/Moscow; weekday = Mon–Fri in MSK.
- Gate defaults: `-min-morning-days 0.9`, `-min-morning-turnover 5` (млн ₽), `-max-tick-pct 0.2`, `-min-turnover 50` (млн ₽). A gate fails strictly below its minimum / strictly above its maximum; equality passes.
- `-max-tick-pct 0` disables the tick gate; unknown tick (`MinPriceIncrement == 0`) never fails it.
- Profit of the signal-count run is never shown or used.
- Report path: `reports/gap_fade/screen_<20060102_150405>.md`; final line format `GAP_FADE_TICKERS=A,B,C`.
- `docs/gap_fade/*.md` describe mechanics only — no run results.
- Commits end with `Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>`. Never push.
- Quality gate per task: `go test ./internal/service/backtest/... ./cmd/gapscreen/... ./cmd/gapwf/...` and `go vet` on touched packages; final: `./bin/mage ci` (known pre-existing failure `bonds/computable TestCalculateProfit/ОФЗ_с_дисконтом` is out of scope).

## Review Focus

- Duplicate 07:00 bars on one date (cache overlap) — counted once per date; covered in Task 1 test.
- Weekend sessions with 07:00-ish bars — never counted as weekdays or morning bars; Task 1 test.
- Ticker with zero candles — goes to «нет данных», not to rejected with zero metrics; Task 3 code.
- Zero passed tickers — report still written with empty `GAP_FADE_TICKERS=` and «Нет.»; Task 2 test.
- Unknown price tick — tick column «—», tick gate skipped; Task 1 and Task 2 tests.

---

### Task 1: Screening core

**Files:**
- Create: `internal/service/backtest/gap_screen.go`
- Test: `internal/service/backtest/gap_screen_test.go`

**Interfaces:**
- Consumes: `backtest.Candle`, `backtest.Run`, `backtest.Config` (package `tinvest/internal/domain/backtest`, imported as `backtest` like `gap_wf.go`); `GapFadeLookupOrGeneric`; `gapLoc` (MSK location from `gap_wf.go`); `medianF` from `pullback_screen.go`; `backtest.MeanDailyTurnoverM`.
- Produces:
  - `type GapScreenOptions struct { MinMorningDays, MinMorningTurnoverM, MaxTickPct, MinTurnoverM float64 }`
  - `func DefaultGapScreenOptions() GapScreenOptions`
  - `type GapScreenInput struct { Ticker, Name string; Bars []backtest.Candle; Lot int32; MinPriceIncrement, SignalsPerYear float64 }`
  - `type GapScreenRow struct { Ticker, Name string; MorningDays, MorningTurnoverM, TurnoverM, TickPct, SignalsPerYear float64; Pass bool; Reasons []string }`
  - `func GapMorningStats(bars []backtest.Candle, lot int32) (days, turnoverM float64)`
  - `func GapSignalsPerYear(ticker string, bars, daily []backtest.Candle, lot int32, months int) float64`
  - `func ApplyGapGates(row GapScreenRow, opts GapScreenOptions) GapScreenRow`
  - `func ScreenGap(in GapScreenInput, opts GapScreenOptions) GapScreenRow`

- [ ] **Step 1: Write the failing tests**

`internal/service/backtest/gap_screen_test.go`:

```go
package backtest

import (
	"math"
	"reflect"
	"strings"
	"testing"
	"time"

	"tinvest/internal/domain/backtest"
)

func gsDate(s string) time.Time {
	t, err := time.ParseInLocation("2006-01-02", s, gapLoc)
	if err != nil {
		panic(err)
	}
	return t
}

// gsBar is a flat 30m bar starting at hh:mm MSK on day.
func gsBar(day string, hh, mm int, price float64, vol int64) backtest.Candle {
	t := gsDate(day).Add(time.Duration(hh)*time.Hour + time.Duration(mm)*time.Minute)
	return backtest.Candle{Time: t, Open: price, High: price, Low: price, Close: price, Volume: vol}
}

func TestGapMorningStats(t *testing.T) {
	bars := []backtest.Candle{
		// Mon 2026-06-01: auction bar 06:30 is ignored, the 07:00 bar is 1000 lots·10·100 = 1 млн ₽.
		gsBar("2026-06-01", 6, 30, 100, 99999),
		gsBar("2026-06-01", 7, 0, 100, 1000),
		gsBar("2026-06-01", 7, 0, 100, 1000), // duplicate 07:00 bar from a cache overlap: counted once
		// Tue: session starts at 07:30 — a weekday without a morning bar.
		gsBar("2026-06-02", 7, 30, 100, 500),
		// Wed: 07:00 bar of 3 млн ₽.
		gsBar("2026-06-03", 7, 0, 100, 3000),
		// Sat: a weekend session is neither a weekday nor a morning bar.
		gsBar("2026-06-06", 7, 0, 100, 777777),
	}
	days, turnover := GapMorningStats(bars, 10)
	if math.Abs(days-2.0/3.0) > 1e-9 {
		t.Errorf("days = %v, want 2/3", days)
	}
	if math.Abs(turnover-2.0) > 1e-9 {
		t.Errorf("turnover = %v, want median(1, 3) = 2", turnover)
	}
	if d, m := GapMorningStats(nil, 10); d != 0 || m != 0 {
		t.Errorf("empty: %v %v", d, m)
	}
}

func passingRow() GapScreenRow {
	return GapScreenRow{Ticker: "AAA", MorningDays: 0.95, MorningTurnoverM: 6, TurnoverM: 60, TickPct: 0.1}
}

func TestApplyGapGatesPassAndBoundaries(t *testing.T) {
	opts := DefaultGapScreenOptions()
	if got := ApplyGapGates(passingRow(), opts); !got.Pass || len(got.Reasons) != 0 {
		t.Fatalf("passing row rejected: %+v", got)
	}
	edge := GapScreenRow{Ticker: "EDGE", MorningDays: 0.9, MorningTurnoverM: 5, TurnoverM: 50, TickPct: 0.2}
	if got := ApplyGapGates(edge, opts); !got.Pass {
		t.Fatalf("equality must pass: %+v", got.Reasons)
	}
	unknownTick := passingRow()
	unknownTick.TickPct = 0
	if got := ApplyGapGates(unknownTick, opts); !got.Pass {
		t.Fatalf("unknown tick must not fail: %+v", got.Reasons)
	}
	off := passingRow()
	off.TickPct = 5
	opts.MaxTickPct = 0
	if got := ApplyGapGates(off, opts); !got.Pass {
		t.Fatalf("-max-tick-pct 0 must disable the gate: %+v", got.Reasons)
	}
}

func TestApplyGapGatesEachFailsAlone(t *testing.T) {
	cases := map[string]struct {
		mut  func(*GapScreenRow)
		want string
	}{
		"morning days":     {func(r *GapScreenRow) { r.MorningDays = 0.5 }, "бар 07:00 в 50% будних дней < 90%"},
		"morning turnover": {func(r *GapScreenRow) { r.MorningTurnoverM = 1.24 }, "утренний оборот 1.2 млн ₽ < 5.0"},
		"tick":             {func(r *GapScreenRow) { r.TickPct = 0.31 }, "два шага 0.31% > 0.20%"},
		"daily turnover":   {func(r *GapScreenRow) { r.TurnoverM = 12 }, "дневной оборот 12 млн ₽ < 50"},
	}
	for name, c := range cases {
		r := passingRow()
		c.mut(&r)
		got := ApplyGapGates(r, DefaultGapScreenOptions())
		if got.Pass || !reflect.DeepEqual(got.Reasons, []string{c.want}) {
			t.Errorf("%s: pass=%v reasons=%q, want [%q]", name, got.Pass, got.Reasons, c.want)
		}
	}
}

func TestScreenGapWiresMetrics(t *testing.T) {
	var bars []backtest.Candle
	for _, d := range []string{"2026-06-01", "2026-06-02"} {
		bars = append(bars, gsBar(d, 7, 0, 200, 5000), gsBar(d, 7, 30, 200, 15000))
	}
	row := ScreenGap(GapScreenInput{
		Ticker: "BBB", Name: "Бэ", Bars: bars, Lot: 10, MinPriceIncrement: 0.1, SignalsPerYear: 3.5,
	}, DefaultGapScreenOptions())
	if row.Ticker != "BBB" || row.Name != "Бэ" || row.SignalsPerYear != 3.5 {
		t.Fatalf("identity not carried: %+v", row)
	}
	if row.MorningDays != 1 || math.Abs(row.MorningTurnoverM-10) > 1e-9 {
		t.Errorf("morning = %v / %v, want 1 / 10", row.MorningDays, row.MorningTurnoverM)
	}
	if math.Abs(row.TurnoverM-40) > 1e-9 {
		t.Errorf("turnover = %v, want 40", row.TurnoverM)
	}
	if math.Abs(row.TickPct-0.1) > 1e-9 {
		t.Errorf("tick = %v, want 2·0.1/200·100 = 0.1", row.TickPct)
	}
	if row.Pass || len(row.Reasons) != 1 || !strings.HasPrefix(row.Reasons[0], "дневной оборот 40") {
		t.Errorf("want only the daily-turnover gate to fail, got %q", row.Reasons)
	}
	if got := ScreenGap(GapScreenInput{Ticker: "NOTICK", Bars: bars, Lot: 10}, DefaultGapScreenOptions()); got.TickPct != 0 {
		t.Errorf("unknown tick: TickPct = %v, want 0", got.TickPct)
	}
}

// gsMarket covers every weekday in [from, to) with a 07:00–23:30 session flat at 100; on a gap
// day the 07:00 bar sits at 99.7 (0.6 ATR below 100 with ATR 0.5) and the 07:30 bar touches 100.
func gsMarket(from, to time.Time, gapDays map[string]bool) (bars, daily []backtest.Candle) {
	for d := from.AddDate(0, -3, 0); d.Before(to); d = d.AddDate(0, 0, 1) {
		if d.Weekday() == time.Saturday || d.Weekday() == time.Sunday {
			continue
		}
		daily = append(daily, backtest.Candle{Time: d.Add(10 * time.Hour), Open: 100, High: 100.25, Low: 99.75, Close: 100})
		if d.Before(from) {
			continue
		}
		for t := d.Add(7 * time.Hour); t.Before(d.Add(24 * time.Hour)); t = t.Add(30 * time.Minute) {
			bars = append(bars, backtest.Candle{Time: t, Open: 100, High: 100, Low: 100, Close: 100})
		}
		if gapDays[d.Format("2006-01-02")] {
			i := len(bars) - 34 // 34 bars per day; the day's first bar
			bars[i] = backtest.Candle{Time: bars[i].Time, Open: 99.7, High: 99.7, Low: 99.7, Close: 99.7}
			bars[i+1] = backtest.Candle{Time: bars[i+1].Time, Open: 99.8, High: 100, Low: 99.7, Close: 100}
		}
	}
	return bars, daily
}

func TestGapSignalsPerYearCountsEngineTrades(t *testing.T) {
	bars, daily := gsMarket(gsDate("2026-05-18"), gsDate("2026-09-01"),
		map[string]bool{"2026-06-10": true, "2026-07-15": true})
	if got := GapSignalsPerYear("SYN", bars, daily, 1, 6); math.Abs(got-4) > 1e-9 {
		t.Fatalf("signals per year = %v, want 2 trades / 0.5 year = 4", got)
	}
	if got := GapSignalsPerYear("SYN", bars, daily, 1, 0); got != 0 {
		t.Fatalf("months 0: %v, want 0", got)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/service/backtest/ -run 'TestGapMorningStats|TestApplyGapGates|TestScreenGap|TestGapSignalsPerYear' -count=1`
Expected: FAIL — `undefined: GapMorningStats` (and the other new names).

- [ ] **Step 3: Implement**

`internal/service/backtest/gap_screen.go`:

```go
package backtest

// Pure core of cmd/gapscreen: structural metrics that decide whether a ticker's 07:00 gap_fade
// entry is executable, and the gates over them. Profit is deliberately not measured — a ticker
// has a handful of gap signals a year, and the universe-wide profit check is cmd/gapwf's job.
// Spec: docs/superpowers/specs/2026-10-02-gap-fade-screener-design.md.

import (
	"fmt"
	"time"

	"tinvest/internal/domain/backtest"
)

// GapScreenOptions are the gate thresholds.
type GapScreenOptions struct {
	MinMorningDays      float64 // minimum share of weekday dates with a 07:00 bar, 0..1
	MinMorningTurnoverM float64 // minimum median 07:00-bar turnover, millions of RUB
	MaxTickPct          float64 // maximum two price ticks as % of the last close; 0 = off
	MinTurnoverM        float64 // minimum mean daily turnover, millions of RUB
}

// DefaultGapScreenOptions are the spec defaults: a 100k RUB position is 2% of a 5M 07:00 bar.
func DefaultGapScreenOptions() GapScreenOptions {
	return GapScreenOptions{MinMorningDays: 0.9, MinMorningTurnoverM: 5, MaxTickPct: 0.2, MinTurnoverM: 50}
}

// GapScreenInput is one ticker's data for ScreenGap.
type GapScreenInput struct {
	Ticker, Name      string
	Bars              []backtest.Candle // 30m, oldest-first
	Lot               int32
	MinPriceIncrement float64 // 0 when the API did not report one
	SignalsPerYear    float64 // from GapSignalsPerYear; informational
}

// GapScreenRow is one ticker's metrics and gate outcome.
type GapScreenRow struct {
	Ticker, Name     string
	MorningDays      float64 // share of weekday dates with a 07:00 bar
	MorningTurnoverM float64 // median 07:00-bar turnover, millions of RUB
	TurnoverM        float64 // mean daily turnover, millions of RUB
	TickPct          float64 // two ticks as % of the last close; 0 = unknown tick
	SignalsPerYear   float64
	Pass             bool
	Reasons          []string // one per failed gate
}

// GapMorningStats returns the share of weekday MSK dates that have a bar starting at 07:00 (the
// first continuous-trading bar, where gap_fade buys) and the median turnover of those bars in
// millions of RUB. Weekend sessions are ignored; a duplicate 07:00 bar on one date counts once.
func GapMorningStats(bars []backtest.Candle, lot int32) (days, turnoverM float64) {
	weekdays := map[string]bool{}
	seen := map[string]bool{}
	var morning []float64
	for _, b := range bars {
		tl := b.Time.In(gapLoc)
		if tl.Weekday() == time.Saturday || tl.Weekday() == time.Sunday {
			continue
		}
		day := tl.Format("2006-01-02")
		weekdays[day] = true
		if tl.Hour() != 7 || tl.Minute() != 0 || seen[day] {
			continue
		}
		seen[day] = true
		morning = append(morning, float64(b.Volume)*float64(lot)*b.Close/1e6)
	}
	if len(weekdays) == 0 {
		return 0, 0
	}
	return float64(len(morning)) / float64(len(weekdays)), medianF(morning)
}

// GapSignalsPerYear counts gap_fade trades over the window with the shared default parameters,
// run exactly as cmd/gapwf runs them (next-open entry), so the screener never re-implements the
// gap rule. Only the count is used — the run's profit stays out of the screen. Dividend ex-days
// are not filtered here.
func GapSignalsPerYear(ticker string, bars, daily []backtest.Candle, lot int32, months int) float64 {
	if months <= 0 {
		return 0
	}
	b := GapFadeLookupOrGeneric(ticker)
	cfg := backtest.Config{InitialCash: 1e7, Fraction: 1.0, Commission: 0.0005, Lot: lot, EntryAtNextOpen: true}
	res := backtest.Run(b.Build(b.DefaultParams()), bars, daily, nil, cfg)
	return float64(len(res.Trades)) / (float64(months) / 12)
}

// ApplyGapGates sets Pass and Reasons from the row's metrics. Equality passes every gate.
func ApplyGapGates(row GapScreenRow, opts GapScreenOptions) GapScreenRow {
	var reasons []string
	if row.MorningDays < opts.MinMorningDays {
		reasons = append(reasons, fmt.Sprintf("бар 07:00 в %.0f%% будних дней < %.0f%%", row.MorningDays*100, opts.MinMorningDays*100))
	}
	if row.MorningTurnoverM < opts.MinMorningTurnoverM {
		reasons = append(reasons, fmt.Sprintf("утренний оборот %.1f млн ₽ < %.1f", row.MorningTurnoverM, opts.MinMorningTurnoverM))
	}
	if opts.MaxTickPct > 0 && row.TickPct > opts.MaxTickPct {
		reasons = append(reasons, fmt.Sprintf("два шага %.2f%% > %.2f%%", row.TickPct, opts.MaxTickPct))
	}
	if row.TurnoverM < opts.MinTurnoverM {
		reasons = append(reasons, fmt.Sprintf("дневной оборот %.0f млн ₽ < %.0f", row.TurnoverM, opts.MinTurnoverM))
	}
	row.Reasons = reasons
	row.Pass = len(reasons) == 0
	return row
}

// ScreenGap measures one ticker and applies the gates.
func ScreenGap(in GapScreenInput, opts GapScreenOptions) GapScreenRow {
	row := GapScreenRow{Ticker: in.Ticker, Name: in.Name, SignalsPerYear: in.SignalsPerYear}
	row.MorningDays, row.MorningTurnoverM = GapMorningStats(in.Bars, in.Lot)
	row.TurnoverM = backtest.MeanDailyTurnoverM(in.Bars, in.Lot)
	if n := len(in.Bars); n > 0 && in.MinPriceIncrement > 0 && in.Bars[n-1].Close > 0 {
		row.TickPct = 2 * in.MinPriceIncrement / in.Bars[n-1].Close * 100
	}
	return ApplyGapGates(row, opts)
}
```

Note for the implementer: `MeanDailyTurnoverM` groups by UTC date; MSK 07:00–23:30 bars span one UTC date except after 03:00 MSK, so the test's 07:00/07:30 bars sit on one UTC date. Do not change `MeanDailyTurnoverM`.

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/service/backtest/ -run 'TestGapMorningStats|TestApplyGapGates|TestScreenGap|TestGapSignalsPerYear' -count=1 -v`
Expected: PASS. If `TestGapSignalsPerYearCountsEngineTrades` gives a different count, debug the fixture (daily lead-in, gap-bar index) — do not loosen the assertion.

- [ ] **Step 5: Full package + vet, commit**

Run: `go test ./internal/service/backtest/... -count=1 && go vet ./internal/service/backtest/`

```bash
git add internal/service/backtest/gap_screen.go internal/service/backtest/gap_screen_test.go
git commit -m "feat(gap_fade): ядро скринера — утренняя ликвидность, гейты, сигналы в год

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

---

### Task 2: Screener report

**Files:**
- Create: `internal/service/backtest/gap_screen_report.go`
- Test: `internal/service/backtest/gap_screen_report_test.go`

**Interfaces:**
- Consumes: `GapScreenRow`, `GapScreenOptions` (Task 1).
- Produces:
  - `type GapScreenMeta struct { Generated time.Time; Months, Scanned int; Opts GapScreenOptions; NoData []string }`
  - `func GapScreenPassed(rows []GapScreenRow) []GapScreenRow` — passed rows, morning turnover desc, ticker asc on ties.
  - `func GapScreenTickersLine(rows []GapScreenRow) string` — `GAP_FADE_TICKERS=` + passed tickers in `GapScreenPassed` order, comma-joined.
  - `func RenderGapScreenMarkdown(rows []GapScreenRow, meta GapScreenMeta) string`

- [ ] **Step 1: Write the failing tests**

`internal/service/backtest/gap_screen_report_test.go`:

```go
package backtest

import (
	"strings"
	"testing"
	"time"
)

func reportRows() []GapScreenRow {
	return []GapScreenRow{
		{Ticker: "GAZP", Name: "Газпром", MorningDays: 1, MorningTurnoverM: 109.5, TurnoverM: 4000, TickPct: 0.01, SignalsPerYear: 2, Pass: true},
		{Ticker: "TGKA", Name: "ТГК-1", MorningDays: 0.95, MorningTurnoverM: 0.4, TurnoverM: 30, TickPct: 0, SignalsPerYear: 5,
			Reasons: []string{"утренний оборот 0.4 млн ₽ < 5.0", "дневной оборот 30 млн ₽ < 50"}},
		{Ticker: "SBER", Name: "Сбербанк", MorningDays: 1, MorningTurnoverM: 127.7, TurnoverM: 9000, TickPct: 0.003, SignalsPerYear: 1, Pass: true},
		{Ticker: "AAAA", Name: "Ничья", MorningDays: 1, MorningTurnoverM: 109.5, TurnoverM: 100, TickPct: 0, SignalsPerYear: 0, Pass: true},
	}
}

func TestGapScreenPassedOrderAndLine(t *testing.T) {
	got := GapScreenPassed(reportRows())
	var names []string
	for _, r := range got {
		names = append(names, r.Ticker)
	}
	if strings.Join(names, ",") != "SBER,AAAA,GAZP" {
		t.Fatalf("order = %v, want SBER,AAAA,GAZP (turnover desc, ticker asc on ties)", names)
	}
	if line := GapScreenTickersLine(reportRows()); line != "GAP_FADE_TICKERS=SBER,AAAA,GAZP" {
		t.Fatalf("line = %q", line)
	}
	if line := GapScreenTickersLine(nil); line != "GAP_FADE_TICKERS=" {
		t.Fatalf("empty line = %q", line)
	}
}

func TestRenderGapScreenMarkdown(t *testing.T) {
	meta := GapScreenMeta{
		Generated: time.Date(2026, 10, 2, 15, 4, 0, 0, gapLoc), Months: 12, Scanned: 5,
		Opts: DefaultGapScreenOptions(), NoData: []string{"XXXX: нет свечей"},
	}
	md := RenderGapScreenMarkdown(reportRows(), meta)
	for _, want := range []string{
		"# gap_fade — скринер тикеров",
		"- Сформирован: 2026-10-02 15:04",
		"просмотрено тикеров: 5, прошли: 3, отсеяны: 1, нет данных: 1",
		"бар 07:00 ≥ 90% будних дней; утренний оборот ≥ 5.0 млн ₽; два шага ≤ 0.20%; дневной оборот ≥ 50 млн ₽",
		"GAP_FADE_TICKERS=SBER,AAAA,GAZP",
		"| SBER | Сбербанк | 100% | 127.7 | 9000 | 0.00% | 1.0 |",
		"| AAAA | Ничья | 100% | 109.5 | 100 | — | 0.0 |",
		"| TGKA | ТГК-1 | утренний оборот 0.4 млн ₽ < 5.0; дневной оборот 30 млн ₽ < 50 |",
		"- XXXX: нет свечей",
	} {
		if !strings.Contains(md, want) {
			t.Errorf("report lacks %q\n---\n%s", want, md)
		}
	}
	if strings.Index(md, "| SBER |") > strings.Index(md, "| GAZP |") {
		t.Error("passed table not sorted by morning turnover")
	}
}

func TestRenderGapScreenMarkdownEmpty(t *testing.T) {
	opts := DefaultGapScreenOptions()
	opts.MaxTickPct = 0
	md := RenderGapScreenMarkdown(nil, GapScreenMeta{Generated: time.Now(), Months: 12, Opts: opts})
	for _, want := range []string{"GAP_FADE_TICKERS=\n", "прошли: 0", "два шага — выкл.", "Нет."} {
		if !strings.Contains(md, want) {
			t.Errorf("empty report lacks %q\n---\n%s", want, md)
		}
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/service/backtest/ -run 'TestGapScreenPassed|TestRenderGapScreen' -count=1`
Expected: FAIL — `undefined: GapScreenPassed`.

- [ ] **Step 3: Implement**

`internal/service/backtest/gap_screen_report.go`:

```go
package backtest

import (
	"fmt"
	"sort"
	"strings"
	"time"
)

// GapScreenMeta is the run context printed in the screener report header.
type GapScreenMeta struct {
	Generated time.Time
	Months    int
	Scanned   int
	Opts      GapScreenOptions
	NoData    []string // "TICKER: reason" for tickers without usable candles
}

// GapScreenPassed returns the rows that passed every gate, by median 07:00 turnover descending
// (ticker ascending on ties) — the most executable entries first.
func GapScreenPassed(rows []GapScreenRow) []GapScreenRow {
	var out []GapScreenRow
	for _, r := range rows {
		if r.Pass {
			out = append(out, r)
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].MorningTurnoverM != out[j].MorningTurnoverM {
			return out[i].MorningTurnoverM > out[j].MorningTurnoverM
		}
		return out[i].Ticker < out[j].Ticker
	})
	return out
}

// GapScreenTickersLine is the env line for env/prod.env: passed tickers in report order.
func GapScreenTickersLine(rows []GapScreenRow) string {
	passed := GapScreenPassed(rows)
	t := make([]string, len(passed))
	for i, r := range passed {
		t[i] = r.Ticker
	}
	return "GAP_FADE_TICKERS=" + strings.Join(t, ",")
}

// RenderGapScreenMarkdown renders the screener report.
func RenderGapScreenMarkdown(rows []GapScreenRow, meta GapScreenMeta) string {
	passed := GapScreenPassed(rows)
	var rejected []GapScreenRow
	for _, r := range rows {
		if !r.Pass {
			rejected = append(rejected, r)
		}
	}
	sort.Slice(rejected, func(i, j int) bool { return rejected[i].Ticker < rejected[j].Ticker })

	tick := fmt.Sprintf("два шага ≤ %.2f%%", meta.Opts.MaxTickPct)
	if meta.Opts.MaxTickPct == 0 {
		tick = "два шага — выкл."
	}
	var b strings.Builder
	b.WriteString("# gap_fade — скринер тикеров\n\n")
	fmt.Fprintf(&b, "- Сформирован: %s\n", meta.Generated.Format("2006-01-02 15:04"))
	fmt.Fprintf(&b, "- Окно: %d мес., просмотрено тикеров: %d, прошли: %d, отсеяны: %d, нет данных: %d\n",
		meta.Months, meta.Scanned, len(passed), len(rejected), len(meta.NoData))
	fmt.Fprintf(&b, "- Гейты: бар 07:00 ≥ %.0f%% будних дней; утренний оборот ≥ %.1f млн ₽; %s; дневной оборот ≥ %.0f млн ₽\n",
		meta.Opts.MinMorningDays*100, meta.Opts.MinMorningTurnoverM, tick, meta.Opts.MinTurnoverM)
	b.WriteString("- Сигналов в год — число сделок gap_fade с общими параметрами, справочно: дни дивидендных отсечек не выброшены, прибыль не считается.\n\n")

	b.WriteString("## Итог\n\n```\n")
	b.WriteString(GapScreenTickersLine(rows))
	b.WriteString("\n```\n\n")

	b.WriteString("## Прошли\n\n")
	if len(passed) == 0 {
		b.WriteString("Нет.\n\n")
	} else {
		b.WriteString("| Тикер | Название | Бар 07:00, дней | Утренний оборот, млн ₽ | Дневной оборот, млн ₽ | Два шага | Сигналов в год |\n")
		b.WriteString("|---|---|---|---|---|---|---|\n")
		for _, r := range passed {
			t := "—"
			if r.TickPct > 0 {
				t = fmt.Sprintf("%.2f%%", r.TickPct)
			}
			fmt.Fprintf(&b, "| %s | %s | %.0f%% | %.1f | %.0f | %s | %.1f |\n",
				r.Ticker, r.Name, r.MorningDays*100, r.MorningTurnoverM, r.TurnoverM, t, r.SignalsPerYear)
		}
		b.WriteString("\n")
	}

	b.WriteString("## Отсеяны\n\n")
	if len(rejected) == 0 {
		b.WriteString("Нет.\n\n")
	} else {
		b.WriteString("| Тикер | Название | Причины |\n|---|---|---|\n")
		for _, r := range rejected {
			fmt.Fprintf(&b, "| %s | %s | %s |\n", r.Ticker, r.Name, strings.Join(r.Reasons, "; "))
		}
		b.WriteString("\n")
	}

	b.WriteString("## Нет данных\n\n")
	if len(meta.NoData) == 0 {
		b.WriteString("Нет.\n")
	} else {
		for _, s := range meta.NoData {
			fmt.Fprintf(&b, "- %s\n", s)
		}
	}
	return b.String()
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/service/backtest/ -run 'TestGapScreenPassed|TestRenderGapScreen' -count=1 -v`
Expected: PASS.

- [ ] **Step 5: Full package + vet, commit**

Run: `go test ./internal/service/backtest/... -count=1 && go vet ./internal/service/backtest/`

```bash
git add internal/service/backtest/gap_screen_report.go internal/service/backtest/gap_screen_report_test.go
git commit -m "feat(gap_fade): отчёт скринера и строка GAP_FADE_TICKERS

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

---

### Task 3: `cmd/gapscreen`

**Files:**
- Create: `cmd/gapscreen/main.go`
- Test: `cmd/gapscreen/main_test.go`

**Interfaces:**
- Consumes: Task 1 (`DefaultGapScreenOptions`, `GapScreenOptions`, `GapScreenInput`, `GapSignalsPerYear`, `ScreenGap`, `GapScreenRow`), Task 2 (`GapScreenMeta`, `GapScreenPassed`, `GapScreenTickersLine`, `RenderGapScreenMarkdown`); `screenrun.{LoadToken, APIAddress, LoadUniverse, SplitCSV, EffectiveWorkers, PauseAfterTicker, CacheDir, ShareInfo}`; `svc.NewCandleProvider(...).Load(ctx, ticker, id, interval, from, to, refresh) ([]domain.Candle, error)`.
- Produces: binary `go run ./cmd/gapscreen`; `validate(runCfg) error`.

- [ ] **Step 1: Write the failing test**

`cmd/gapscreen/main_test.go`:

```go
package main

import (
	"testing"

	svc "tinvest/internal/service/backtest"
)

func okCfg() runCfg {
	return runCfg{months: 12, workers: 8, opts: svc.DefaultGapScreenOptions()}
}

func TestValidate(t *testing.T) {
	if err := validate(okCfg()); err != nil {
		t.Fatalf("valid cfg rejected: %v", err)
	}
	tickOff := okCfg()
	tickOff.opts.MaxTickPct = 0
	if err := validate(tickOff); err != nil {
		t.Fatalf("-max-tick-pct 0 rejected: %v", err)
	}
	bad := map[string]func(*runCfg){
		"months":           func(c *runCfg) { c.months = 0 },
		"workers":          func(c *runCfg) { c.workers = 0 },
		"days below 0":     func(c *runCfg) { c.opts.MinMorningDays = -0.1 },
		"days above 1":     func(c *runCfg) { c.opts.MinMorningDays = 1.1 },
		"morning turnover": func(c *runCfg) { c.opts.MinMorningTurnoverM = -1 },
		"tick":             func(c *runCfg) { c.opts.MaxTickPct = -0.1 },
		"daily turnover":   func(c *runCfg) { c.opts.MinTurnoverM = -1 },
	}
	for name, mut := range bad {
		c := okCfg()
		mut(&c)
		if err := validate(c); err == nil {
			t.Errorf("%s: accepted", name)
		}
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./cmd/gapscreen/ -count=1`
Expected: FAIL — no Go files / `undefined: runCfg`.

- [ ] **Step 3: Implement**

`cmd/gapscreen/main.go`:

```go
// Command gapscreen picks the gap_fade universe by structure, not by profit: a ticker passes when
// its 07:00 bar trades almost every weekday, that bar is liquid enough to absorb the entry, the
// price tick is small and the daily turnover is high. The profit check of the picked set is
// cmd/gapwf. All gRPC/file I/O lives here; the metrics, gates and report are pure
// (internal/service/backtest/gap_screen*.go). Spec: docs/superpowers/specs/2026-10-02-gap-fade-screener-design.md.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"

	"tinvest/internal/enum"
	svc "tinvest/internal/service/backtest"
	"tinvest/internal/service/backtest/screenrun"
	grpcclient "tinvest/pkg/client/grpc"
	"tinvest/pkg/logger"
	"tinvest/pkg/semaphore"
)

func main() {
	def := svc.DefaultGapScreenOptions()
	var (
		months          = flag.Int("months", 12, "lookback window in months")
		minMorningDays  = flag.Float64("min-morning-days", def.MinMorningDays, "gate: minimum share of weekdays with a 07:00 bar (0..1)")
		minMorningTurnM = flag.Float64("min-morning-turnover", def.MinMorningTurnoverM, "gate: minimum median 07:00-bar turnover in millions of RUB")
		maxTickPct      = flag.Float64("max-tick-pct", def.MaxTickPct, "gate: maximum two price ticks as a percentage of the last close (0 = off)")
		minTurnoverM    = flag.Float64("min-turnover", def.MinTurnoverM, "gate: minimum mean daily turnover in millions of RUB")
		tickersCSV      = flag.String("tickers", "", "comma-separated tickers instead of the full universe (diagnostics/smoke runs)")
		workers         = flag.Int("workers", 8, "concurrent tickers")
		outDir          = flag.String("out", "reports/gap_fade", "report output directory")
		refresh         = flag.Bool("refresh", false, "force candle refetch (ignore cache)")
		pause           = flag.Duration("pause", 0, "idle time after each ticker, e.g. 500ms or 2s: trades wall-clock for a cooler CPU")
	)
	flag.Parse()
	logger.Init()

	if err := run(context.Background(), runCfg{
		months: *months, workers: *workers,
		opts: svc.GapScreenOptions{
			MinMorningDays: *minMorningDays, MinMorningTurnoverM: *minMorningTurnM,
			MaxTickPct: *maxTickPct, MinTurnoverM: *minTurnoverM,
		},
		tickers: screenrun.SplitCSV(*tickersCSV), outDir: *outDir, refresh: *refresh, pause: *pause,
	}); err != nil {
		log.Fatalf("gapscreen: %v", err)
	}
}

type runCfg struct {
	months, workers int
	opts            svc.GapScreenOptions
	tickers         []string
	outDir          string
	refresh         bool
	pause           time.Duration
}

// validate rejects flag combinations that would silently produce a meaningless run.
func validate(cfg runCfg) error {
	switch {
	case cfg.months <= 0:
		return fmt.Errorf("-months must be > 0 (got %d)", cfg.months)
	case cfg.workers < 1:
		return fmt.Errorf("-workers must be >= 1 (got %d)", cfg.workers)
	case cfg.opts.MinMorningDays < 0 || cfg.opts.MinMorningDays > 1:
		return fmt.Errorf("-min-morning-days must be within [0, 1] (got %v)", cfg.opts.MinMorningDays)
	case cfg.opts.MinMorningTurnoverM < 0:
		return errors.New("-min-morning-turnover must be >= 0")
	case cfg.opts.MaxTickPct < 0:
		return errors.New("-max-tick-pct must be >= 0 (0 disables the gate)")
	case cfg.opts.MinTurnoverM < 0:
		return errors.New("-min-turnover must be >= 0")
	}
	return nil
}

func run(ctx context.Context, cfg runCfg) error {
	if err := validate(cfg); err != nil {
		return err
	}
	if workers, capped := screenrun.EffectiveWorkers(cfg.workers, cfg.refresh); capped {
		fmt.Printf("gapscreen: -refresh forces -workers %d -> %d to avoid punching holes in the local candle cache (see screenrun.RefreshWorkerCap)\n",
			cfg.workers, workers)
		cfg.workers = workers
	}
	token, err := screenrun.LoadToken()
	if err != nil {
		return err
	}
	client, err := grpcclient.NewClientGrpc(screenrun.APIAddress, token)
	if err != nil {
		return fmt.Errorf("grpc client: %w", err)
	}
	universe, err := screenrun.LoadUniverse(ctx, client, cfg.tickers)
	if err != nil {
		return err
	}

	to := time.Now()
	from := to.AddDate(0, -cfg.months, 0)
	dailyFrom := from.AddDate(-1, 0, 0) // a year of lead-in warms the daily ATR, as in cmd/gapwf
	provider := svc.NewCandleProvider(client.MarketDataServiceClient(), screenrun.CacheDir)

	sem := semaphore.New(cfg.workers)
	var wg sync.WaitGroup
	var mu sync.Mutex
	var rows []svc.GapScreenRow
	var noData []string
	skip := func(ticker, why string) {
		mu.Lock()
		noData = append(noData, ticker+": "+why)
		mu.Unlock()
		fmt.Printf("gapscreen %s: skip (%s)\n", ticker, why)
	}

	for _, u := range universe {
		wg.Add(1)
		sem.Acquire()
		go func(u screenrun.ShareInfo) {
			defer wg.Done()
			defer sem.Release()
			// Held inside the semaphore slot on purpose: releasing first would let the next
			// ticker start while this one idles, and the pool would stay just as hot.
			defer screenrun.PauseAfterTicker(ctx, cfg.pause)

			bars, err := provider.Load(ctx, u.Ticker, u.ID, enum.Minutes30, from, to, cfg.refresh)
			if err != nil {
				skip(u.Ticker, fmt.Sprintf("свечи 30m: %v", err))
				return
			}
			if len(bars) == 0 {
				skip(u.Ticker, "нет свечей 30m")
				return
			}
			daily, err := provider.Load(ctx, u.Ticker, u.ID, enum.Day1, dailyFrom, to, cfg.refresh)
			if err != nil {
				skip(u.Ticker, fmt.Sprintf("дневные свечи: %v", err))
				return
			}
			row := svc.ScreenGap(svc.GapScreenInput{
				Ticker: u.Ticker, Name: u.Name, Bars: bars, Lot: u.Lot, MinPriceIncrement: u.MinPriceIncrement,
				SignalsPerYear: svc.GapSignalsPerYear(u.Ticker, bars, daily, u.Lot, cfg.months),
			}, cfg.opts)
			fmt.Printf("gapscreen %s: pass=%v morning=%.0f%% morningTurnover=%.1fM turnover=%.0fM tick=%.2f%% signals/yr=%.1f\n",
				u.Ticker, row.Pass, row.MorningDays*100, row.MorningTurnoverM, row.TurnoverM, row.TickPct, row.SignalsPerYear)
			mu.Lock()
			rows = append(rows, row)
			mu.Unlock()
		}(u)
	}
	wg.Wait()
	sort.Strings(noData)

	md := svc.RenderGapScreenMarkdown(rows, svc.GapScreenMeta{
		Generated: to, Months: cfg.months, Scanned: len(universe), Opts: cfg.opts, NoData: noData,
	})
	if err := os.MkdirAll(cfg.outDir, 0o755); err != nil {
		return fmt.Errorf("mkdir out dir: %w", err)
	}
	path := filepath.Join(cfg.outDir, fmt.Sprintf("screen_%s.md", to.Format("20060102_150405")))
	if err := os.WriteFile(path, []byte(md), 0o644); err != nil {
		return fmt.Errorf("write %s: %w", path, err)
	}
	passed := len(svc.GapScreenPassed(rows))
	if passed == 0 {
		fmt.Println("gapscreen: ВНИМАНИЕ: ни один тикер не прошёл гейты")
	}
	fmt.Printf("gapscreen: прошли %d из %d (нет данных %d); отчёт %s\n%s\n",
		passed, len(universe), len(noData), path, svc.GapScreenTickersLine(rows))
	return nil
}
```

- [ ] **Step 4: Run tests and build**

Run: `go test ./cmd/gapscreen/ -count=1 -v && go vet ./cmd/gapscreen/ && go build -o /dev/null ./cmd/gapscreen`
Expected: PASS, build OK.

- [ ] **Step 5: Commit**

```bash
git add cmd/gapscreen
git commit -m "feat(gap_fade): команда cmd/gapscreen

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

---

### Task 4: `cmd/gapwf` default universe + docs

**Files:**
- Modify: `cmd/gapwf/main.go` (flag help at line 35, `defaultTickers` at lines 93–104)
- Modify: `cmd/gapwf/main_test.go` (`TestDefaultTickers`, lines 40–55)
- Create: `docs/gap_fade/screener.md`
- Modify: `docs/gap_fade/strategy.md` (universe sentences in «Как торгует» and «Запуск»)
- Modify: `CLAUDE.md` (gap_fade entry in Layout)

**Interfaces:**
- Consumes: Task 3 command name `cmd/gapscreen`, report path, flag names.
- Produces: `defaultTickers(path string) ([]string, error)` reading `GAP_FADE_TICKERS`.

- [ ] **Step 1: Update the test first**

Replace `TestDefaultTickers` in `cmd/gapwf/main_test.go`:

```go
func TestDefaultTickers(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "prod.env")
	if err := os.WriteFile(path, []byte("RSI_PULLBACK_TICKERS=UGLD\nGAP_FADE_TICKERS=SBER, T ,GAZP\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	got, err := defaultTickers(path)
	if err != nil || !reflect.DeepEqual(got, []string{"SBER", "T", "GAZP"}) {
		t.Fatalf("got %v, %v", got, err)
	}
	// No fallback to the rsi_pullback universe: that set was picked for another strategy.
	pullbackOnly := filepath.Join(dir, "pullback.env")
	_ = os.WriteFile(pullbackOnly, []byte("RSI_PULLBACK_TICKERS=UGLD,T\n"), 0o600)
	_, err = defaultTickers(pullbackOnly)
	if err == nil || !strings.Contains(err.Error(), "GAP_FADE_TICKERS") || !strings.Contains(err.Error(), "cmd/gapscreen") {
		t.Fatalf("missing key: err = %v, want GAP_FADE_TICKERS and a cmd/gapscreen hint", err)
	}
}
```

- [ ] **Step 2: Run to verify it fails**

Run: `go test ./cmd/gapwf/ -run TestDefaultTickers -count=1`
Expected: FAIL — got `[UGLD]`.

- [ ] **Step 3: Implement**

In `cmd/gapwf/main.go` change the flag help:

```go
		tickersCSV  = flag.String("tickers", "", "comma-separated universe (default: GAP_FADE_TICKERS from -env-file)")
```

Replace `defaultTickers`:

```go
// defaultTickers reads the gap_fade universe picked by cmd/gapscreen. There is no fallback to
// RSI_PULLBACK_TICKERS: that set was chosen for another strategy and is mostly too thin at 07:00.
func defaultTickers(path string) ([]string, error) {
	env, err := godotenv.Read(path)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", path, err)
	}
	t := screenrun.SplitCSV(env["GAP_FADE_TICKERS"])
	if len(t) == 0 {
		return nil, fmt.Errorf("%s: GAP_FADE_TICKERS is empty — run go run ./cmd/gapscreen and add its GAP_FADE_TICKERS line, or pass -tickers", path)
	}
	return t, nil
}
```

- [ ] **Step 4: Run tests**

Run: `go test ./cmd/gapwf/ -count=1 && go vet ./cmd/gapwf/`
Expected: PASS.

- [ ] **Step 5: Docs**

Create `docs/gap_fade/screener.md`:

```markdown
# gap_fade — скринер тикеров

Команда: `cmd/gapscreen`. Ядро: `internal/service/backtest/gap_screen.go`, отчёт —
`gap_screen_report.go`. Спека: `docs/superpowers/specs/2026-10-02-gap-fade-screener-design.md`.

## Зачем

gap_fade покупает в 07:00 — на первом баре непрерывных торгов после аукциона открытия. Если
утренний стакан тонкий, бэктест покупает по open, по которому реально не купить. Скринер
отбирает тикеры, где вход в 07:00 исполним, и даёт набор `GAP_FADE_TICKERS`.

## Принцип

Тикеры отбираются по устройству бумаги, а не по прибыли стратегии на тикере: на один тикер
приходятся единицы сигналов в год, PF по ним — шум. Прибыль проверяется одним прогоном
`cmd/gapwf` на всём отобранном наборе.

## Метрики

За окно `-months` по 30m-свечам, время MSK:

- **Бар 07:00, дней** — доля будних дат (дата с хотя бы одним баром), в которых есть бар с
  началом ровно в 07:00. Сессии выходного дня не считаются.
- **Утренний оборот** — медиана оборота бара 07:00 (`объём · лот · close`), млн ₽.
- **Дневной оборот** — средний оборот за день, млн ₽.
- **Два шага** — `2 · шаг цены / последний close`, %. Если шаг неизвестен — «—».
- **Сигналов в год** — число сделок gap_fade с общими параметрами тем же прогоном движка, что в
  `cmd/gapwf`, на год окна. Справочно: в отборе не участвует, дни дивидендных отсечек не
  выброшены, прибыль этого прогона не показывается.

## Гейты

| Флаг | Дефолт | Отсеивает, если |
|---|---|---|
| `-min-morning-days` | 0.9 | бар 07:00 есть реже |
| `-min-morning-turnover` | 5 | утренний оборот ниже, млн ₽ (позиция 100 тыс. ₽ — 2% такого бара) |
| `-max-tick-pct` | 0.2 | два шага дороже, %; 0 — гейт выключен; неизвестный шаг не отсеивается |
| `-min-turnover` | 50 | дневной оборот ниже, млн ₽ |

Равенство порогу проходит.

## Запуск

`go run ./cmd/gapscreen` — вся торгуемая RUB-вселенная за 12 месяцев. Флаги: `-months`,
`-tickers` (пробный прогон по списку), `-workers`, `-pause` (меньше нагрев процессора),
`-refresh` (перекачать свечи), `-out` (по умолчанию `reports/gap_fade`).

Отчёт `reports/gap_fade/screen_<время>.md`: прошедшие по убыванию утреннего оборота, отсеянные с
причинами, тикеры без данных и строка `GAP_FADE_TICKERS=...`. Та же строка печатается в консоль.

## Дальше

1. Строка `GAP_FADE_TICKERS` из отчёта вписывается в `env/prod.env`.
2. `go run ./cmd/gapwf` берёт вселенную из `GAP_FADE_TICKERS` и выносит вердикт гейта.
```

In `docs/gap_fade/strategy.md`, «Как торгует», replace:

```
видит, зависит от запуска: `cmd/backtest` — один тикер из `-ticker`, `cmd/gapwf` — вселенная
`RSI_PULLBACK_TICKERS`. Параметры для всех тикеров общие (см. «Почему без калибровки по тикерам»).
```

with:

```
видит, зависит от запуска: `cmd/backtest` — один тикер из `-ticker`, `cmd/gapwf` — вселенная
`GAP_FADE_TICKERS`, которую отбирает скринер `cmd/gapscreen` (`screener.md`). Параметры для всех
тикеров общие (см. «Почему без калибровки по тикерам»).
```

In «Запуск», replace:

```
  доходности сделок. Вселенная — `-tickers` через запятую, по умолчанию `RSI_PULLBACK_TICKERS` из
  `-env-file env/prod.env`. Сделки дня отсечки выброшены: дивиденды берутся из T-Invest
```

with:

```
  доходности сделок. Вселенная — `-tickers` через запятую, по умолчанию `GAP_FADE_TICKERS` из
  `-env-file env/prod.env` (строку даёт `cmd/gapscreen`; без неё команда падает с подсказкой).
  Сделки дня отсечки выброшены: дивиденды берутся из T-Invest
```

In `CLAUDE.md`, in the gap_fade entry replace:

```
universe validation with shared params and pooled walk-forward + gate — `cmd/gapwf`.
```

with:

```
universe validation with shared params and pooled walk-forward + gate — `cmd/gapwf` (default universe `GAP_FADE_TICKERS`); подбор тикеров по утренней ликвидности — `cmd/gapscreen`, docs/gap_fade/screener.md.
```

- [ ] **Step 6: Full check, commit**

Run: `go test ./internal/service/backtest/... ./cmd/gapscreen/... ./cmd/gapwf/... -count=1 && ./bin/mage ci`
Expected: all green except the known pre-existing `bonds/computable TestCalculateProfit/ОФЗ_с_дисконтом`.

```bash
git add cmd/gapwf/main.go cmd/gapwf/main_test.go docs/gap_fade/screener.md docs/gap_fade/strategy.md CLAUDE.md
git commit -m "feat(gap_fade): gapwf берёт вселенную из GAP_FADE_TICKERS, дока скринера

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

---

## After the plan (controller, not a task)

Run `go run ./cmd/gapscreen -workers 2`, then `go run ./cmd/gapwf -workers 2 -tickers <line>`; report both to the owner. Write `GAP_FADE_TICKERS` into `env/prod.env` only if the gate passes and the owner agrees.
