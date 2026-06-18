# Daily-ATR Volatility Screener Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add a `-volrank` CLI mode to `cmd/backtest` that ranks the liquid MOEX share universe by normalized daily ATR (ATR%) over a lookback window and writes a markdown report.

**Architecture:** Three layers. (1) `pkg/indicators` gains an `ATRSeries` helper (full Wilder-ATR series); the existing `ATR` is refactored to call it. (2) `internal/service/backtest` gains pure, testable functions: `VolMetrics` (computes mean/last ATR% + ruble turnover from a candle slice) and `RenderVolatilityMarkdown` (sorted markdown table). (3) `cmd/backtest/main.go` gains flags and a `runVolRank` orchestrator that filters the universe via gRPC, fetches daily candles concurrently with a semaphore worker pool, and writes the report.

**Tech Stack:** Go 1.25, Tinkoff Invest gRPC client, `pkg/semaphore` for the worker pool, existing `CandleProvider` candle cache.

## Global Constraints

- Volatility metric is **ATR% = ATR / close · 100** (normalized), never absolute ATR.
- Headline ranking metric is **mean ATR% over the window**; report also shows the latest ATR%.
- Daily timeframe only: the screener hardcodes `enum.Day1` (ignores `-interval`).
- Universe filter: `Currency == "rub"` (lowercase, as in `internal/converter/share.go`) AND `Trading == true`.
- Turnover is in **millions of RUB**: `mean(volume · lot · close) / 1e6`.
- Markdown output only; no CSV, no charts, no realtime.
- Concurrency: different tickers write different cache files, so the worker pool is race-free for `CandleProvider.Load`; the shared result slice is guarded by a `sync.Mutex`.

---

### Task 1: `ATRSeries` helper in `pkg/indicators`

**Files:**
- Modify: `pkg/indicators/atr.go`
- Test: `pkg/indicators/atr_test.go`

**Interfaces:**
- Consumes: nothing new.
- Produces:
  - `func ATRSeries(highs, lows, closes []float64, period int) []float64` — returns a slice of length `len(closes)`; entries before the seed bar (`i < period`) are `0`; entry at `i == period` is the seed `mean(TR_1..TR_period)`; later entries are Wilder-smoothed. Returns an all-zero slice of length `len(closes)` for invalid input (`period <= 0`, mismatched lengths, `n < period+1`).
  - `func ATR(highs, lows, closes []float64, period int) float64` — unchanged behavior; now returns the last element of `ATRSeries` (or `0` for empty input).

- [ ] **Step 1: Write the failing tests**

Add to `pkg/indicators/atr_test.go` (create the file if it does not exist; if it exists, append these functions):

```go
package indicators

import (
	"math"
	"testing"
)

func TestATRSeries_LengthAndWarmup(t *testing.T) {
	highs := []float64{10, 11, 12, 13, 14, 15}
	lows := []float64{9, 9.5, 10, 11, 12, 13}
	closes := []float64{9.5, 10.5, 11.5, 12.5, 13.5, 14.5}
	period := 3

	s := ATRSeries(highs, lows, closes, period)

	if len(s) != len(closes) {
		t.Fatalf("len = %d, want %d", len(s), len(closes))
	}
	for i := 0; i < period; i++ {
		if s[i] != 0 {
			t.Errorf("s[%d] = %v, want 0 (warmup)", i, s[i])
		}
	}
	if s[period] == 0 {
		t.Errorf("s[%d] (seed) = 0, want non-zero", period)
	}
}

func TestATRSeries_LastEqualsATR(t *testing.T) {
	highs := []float64{10, 11, 12, 13, 14, 15}
	lows := []float64{9, 9.5, 10, 11, 12, 13}
	closes := []float64{9.5, 10.5, 11.5, 12.5, 13.5, 14.5}
	period := 3

	s := ATRSeries(highs, lows, closes, period)
	last := s[len(s)-1]
	single := ATR(highs, lows, closes, period)

	if math.Abs(last-single) > 1e-9 {
		t.Errorf("series last = %v, ATR = %v; want equal", last, single)
	}
}

func TestATRSeries_InvalidReturnsZeros(t *testing.T) {
	closes := []float64{1, 2, 3}
	// n < period+1
	s := ATRSeries([]float64{1, 2, 3}, []float64{1, 2, 3}, closes, 5)
	if len(s) != 3 {
		t.Fatalf("len = %d, want 3", len(s))
	}
	for i, v := range s {
		if v != 0 {
			t.Errorf("s[%d] = %v, want 0", i, v)
		}
	}
}
```

- [ ] **Step 2: Run the tests to verify they fail**

Run: `go test ./pkg/indicators/ -run TestATRSeries -v`
Expected: FAIL — `undefined: ATRSeries`.

- [ ] **Step 3: Implement `ATRSeries` and refactor `ATR`**

Replace the body of `pkg/indicators/atr.go` (keep `package indicators` and `import "math"`) with:

```go
package indicators

import "math"

// ATRSeries returns Wilder's Average True Range at every bar of the input
// series. The result has length len(closes): entries before the seed bar
// (index < period) are 0, the entry at index == period is the seed
// mean(TR_1..TR_period), and later entries are Wilder-smoothed. Returns an
// all-zero slice for invalid input (period <= 0, mismatched lengths, or
// len(closes) < period+1) — the insufficient-history rule is silent.
func ATRSeries(highs, lows, closes []float64, period int) []float64 {
	n := len(closes)
	out := make([]float64, n)
	if period <= 0 || len(highs) != n || len(lows) != n || n < period+1 {
		return out
	}

	trueRange := func(i int) float64 {
		hl := highs[i] - lows[i]
		hc := math.Abs(highs[i] - closes[i-1])
		lc := math.Abs(lows[i] - closes[i-1])
		return math.Max(hl, math.Max(hc, lc))
	}

	// Seed: mean of the first `period` TR values (bars 1..period).
	sum := 0.0
	for i := 1; i <= period; i++ {
		sum += trueRange(i)
	}
	atr := sum / float64(period)
	out[period] = atr

	// Wilder smoothing for every subsequent bar.
	for i := period + 1; i < n; i++ {
		atr = (atr*float64(period-1) + trueRange(i)) / float64(period)
		out[i] = atr
	}
	return out
}

// ATR returns Wilder's Average True Range at the last bar of the input series.
// It is the final element of ATRSeries; returns 0 for empty/invalid input.
//
// Algorithm:
//   - True Range at bar i (i >= 1):
//       TR_i = max(High_i - Low_i, |High_i - Close_{i-1}|, |Low_i - Close_{i-1}|)
//   - Seed ATR_{period} = mean(TR_1 .. TR_{period}).
//   - For i > period: ATR_i = (ATR_{i-1} * (period - 1) + TR_i) / period.
func ATR(highs, lows, closes []float64, period int) float64 {
	s := ATRSeries(highs, lows, closes, period)
	if len(s) == 0 {
		return 0
	}
	return s[len(s)-1]
}
```

- [ ] **Step 4: Run the full indicators test suite to verify it passes**

Run: `go test ./pkg/indicators/ -v`
Expected: PASS — both the new `ATRSeries` tests and any pre-existing `ATR` tests (regression check that the refactor preserved behavior).

- [ ] **Step 5: Commit**

```bash
git add pkg/indicators/atr.go pkg/indicators/atr_test.go
git commit -m "feat(indicators): ATRSeries helper; ATR delegates to it"
```

---

### Task 2: `VolMetrics` + `VolRow` + `RenderVolatilityMarkdown`

**Files:**
- Create: `internal/service/backtest/volatility_screen.go`
- Test: `internal/service/backtest/volatility_screen_test.go`

**Interfaces:**
- Consumes:
  - `indicators.ATRSeries` (Task 1).
  - `backtest.Candle` from `tinvest/internal/domain/backtest` (fields `High, Low, Close float64`, `Volume int64`). Note: this service file lives in `package backtest`; it imports the *domain* `backtest` package and refers to it as `backtest.Candle`, exactly like `internal/service/backtest/candles.go` does.
- Produces:
  - `type VolRow struct { Ticker string; MeanATRpct, LastATRpct, TurnoverM float64; Bars int }`
  - `type VolMeta struct { Months, ATRPeriod int; MinTurnover float64; Scanned, Passed int }`
  - `func VolMetrics(candles []backtest.Candle, lot int32, atrPeriod int) (meanATRpct, lastATRpct, turnoverM float64, bars int)`
  - `func RenderVolatilityMarkdown(rows []VolRow, meta VolMeta, topN int) string`

- [ ] **Step 1: Write the failing tests**

Create `internal/service/backtest/volatility_screen_test.go`:

```go
package backtest

import (
	"strings"
	"testing"

	domainbt "tinvest/internal/domain/backtest"
)

func TestVolMetrics_BasicShape(t *testing.T) {
	// 6 bars, period 3 → series valid from index 3; ATR% positive.
	candles := []domainbt.Candle{
		{High: 10, Low: 9, Close: 9.5, Volume: 100},
		{High: 11, Low: 9.5, Close: 10.5, Volume: 100},
		{High: 12, Low: 10, Close: 11.5, Volume: 100},
		{High: 13, Low: 11, Close: 12.5, Volume: 100},
		{High: 14, Low: 12, Close: 13.5, Volume: 100},
		{High: 15, Low: 13, Close: 14.5, Volume: 100},
	}
	mean, last, turn, bars := VolMetrics(candles, 1, 3)

	if bars != 6 {
		t.Fatalf("bars = %d, want 6", bars)
	}
	if mean <= 0 || last <= 0 {
		t.Errorf("mean=%v last=%v, want both > 0", mean, last)
	}
	// turnover = mean(volume*lot*close)/1e6 ; closes avg ~12 → ~100*12/1e6
	if turn <= 0 {
		t.Errorf("turnover = %v, want > 0", turn)
	}
}

func TestVolMetrics_InsufficientHistory(t *testing.T) {
	candles := []domainbt.Candle{
		{High: 10, Low: 9, Close: 9.5, Volume: 100},
		{High: 11, Low: 9.5, Close: 10.5, Volume: 100},
	}
	mean, last, _, bars := VolMetrics(candles, 1, 14)
	if bars != 2 {
		t.Fatalf("bars = %d, want 2", bars)
	}
	if mean != 0 || last != 0 {
		t.Errorf("mean=%v last=%v, want 0 (no valid ATR)", mean, last)
	}
}

func TestRenderVolatilityMarkdown_SortsDescAndTrend(t *testing.T) {
	rows := []VolRow{
		{Ticker: "AAA", MeanATRpct: 1.0, LastATRpct: 1.5, TurnoverM: 200, Bars: 120}, // trend up
		{Ticker: "BBB", MeanATRpct: 3.0, LastATRpct: 2.0, TurnoverM: 50, Bars: 120},  // trend down
	}
	meta := VolMeta{Months: 6, ATRPeriod: 14, MinTurnover: 50, Scanned: 100, Passed: 2}

	out := RenderVolatilityMarkdown(rows, meta, 0)

	bbb := strings.Index(out, "BBB")
	aaa := strings.Index(out, "AAA")
	if bbb == -1 || aaa == -1 {
		t.Fatalf("both tickers must appear; out=%q", out)
	}
	if bbb > aaa {
		t.Errorf("BBB (mean 3.0) must rank before AAA (mean 1.0)")
	}
	if !strings.Contains(out, "↑") || !strings.Contains(out, "↓") {
		t.Errorf("expected both trend arrows in output")
	}
}

func TestRenderVolatilityMarkdown_TopN(t *testing.T) {
	rows := []VolRow{
		{Ticker: "AAA", MeanATRpct: 1.0, Bars: 120},
		{Ticker: "BBB", MeanATRpct: 3.0, Bars: 120},
		{Ticker: "CCC", MeanATRpct: 2.0, Bars: 120},
	}
	out := RenderVolatilityMarkdown(rows, VolMeta{}, 2)
	if strings.Contains(out, "AAA") {
		t.Errorf("topN=2 must drop AAA (lowest mean); out=%q", out)
	}
	if !strings.Contains(out, "BBB") || !strings.Contains(out, "CCC") {
		t.Errorf("topN=2 must keep BBB and CCC")
	}
}
```

- [ ] **Step 2: Run the tests to verify they fail**

Run: `go test ./internal/service/backtest/ -run 'TestVolMetrics|TestRenderVolatility' -v`
Expected: FAIL — `undefined: VolMetrics` / `undefined: RenderVolatilityMarkdown`.

- [ ] **Step 3: Implement the service file**

Create `internal/service/backtest/volatility_screen.go`:

```go
package backtest

import (
	"fmt"
	"sort"
	"strings"

	"tinvest/internal/domain/backtest"
	"tinvest/pkg/indicators"
)

// VolRow is one ticker's daily-ATR volatility result.
type VolRow struct {
	Ticker     string
	MeanATRpct float64 // mean ATR% over the window (headline ranking metric)
	LastATRpct float64 // latest ATR% (regime: heating up vs cooling)
	TurnoverM  float64 // mean daily turnover in millions of RUB
	Bars       int
}

// VolMeta carries the run parameters shown in the report header.
type VolMeta struct {
	Months      int
	ATRPeriod   int
	MinTurnover float64
	Scanned     int // universe size after the currency/trading filter
	Passed      int // rows that cleared the liquidity/history filter
}

// VolMetrics computes the daily-ATR volatility metrics for one ticker from its
// daily candle slice. meanATRpct/lastATRpct are 0 when there is not enough
// history for a valid ATR series (len < atrPeriod+1). turnoverM is the mean of
// volume*lot*close across all candles, in millions of RUB.
func VolMetrics(candles []backtest.Candle, lot int32, atrPeriod int) (meanATRpct, lastATRpct, turnoverM float64, bars int) {
	bars = len(candles)
	if bars == 0 {
		return 0, 0, 0, 0
	}

	highs := make([]float64, bars)
	lows := make([]float64, bars)
	closes := make([]float64, bars)
	turnoverSum := 0.0
	for i, c := range candles {
		highs[i] = c.High
		lows[i] = c.Low
		closes[i] = c.Close
		turnoverSum += float64(c.Volume) * float64(lot) * c.Close
	}
	turnoverM = turnoverSum / float64(bars) / 1e6

	series := indicators.ATRSeries(highs, lows, closes, atrPeriod)
	pctSum := 0.0
	count := 0
	for i, atr := range series {
		if atr > 0 && closes[i] > 0 {
			pct := atr / closes[i] * 100
			pctSum += pct
			count++
			lastATRpct = pct
		}
	}
	if count == 0 {
		return 0, 0, turnoverM, bars
	}
	meanATRpct = pctSum / float64(count)
	return meanATRpct, lastATRpct, turnoverM, bars
}

// RenderVolatilityMarkdown renders the volatility screen as a Markdown table
// ranked by MeanATRpct descending (most volatile first). When topN > 0 the
// table is truncated to the top N rows.
func RenderVolatilityMarkdown(rows []VolRow, meta VolMeta, topN int) string {
	sorted := make([]VolRow, len(rows))
	copy(sorted, rows)
	sort.SliceStable(sorted, func(i, j int) bool {
		return sorted[i].MeanATRpct > sorted[j].MeanATRpct
	})
	if topN > 0 && len(sorted) > topN {
		sorted = sorted[:topN]
	}

	var b strings.Builder
	b.WriteString("# Волатильность акций по дневному ATR\n\n")
	fmt.Fprintf(&b, "Окно: %d мес; ATR(%d) на дневном ТФ; порог ликвидности: %.0f млн ₽/день.\n",
		meta.Months, meta.ATRPeriod, meta.MinTurnover)
	fmt.Fprintf(&b, "Просканировано %d тикеров (RUB, торгуемые); прошло фильтр: %d.\n\n",
		meta.Scanned, meta.Passed)
	b.WriteString("Метрика — ATR%% = ATR / цена. Ранжир по средней ATR%% за окно (убыв.).\n\n")
	b.WriteString("| # | Тикер | Ср. ATR% | Тек. ATR% | Тренд | Ликвидность, млн ₽/день | Баров |\n")
	b.WriteString("|---|---|---|---|---|---|---|\n")
	for i, r := range sorted {
		trend := "↓"
		if r.LastATRpct > r.MeanATRpct {
			trend = "↑"
		}
		fmt.Fprintf(&b, "| %d | %s | %.2f | %.2f | %s | %.1f | %d |\n",
			i+1, r.Ticker, r.MeanATRpct, r.LastATRpct, trend, r.TurnoverM, r.Bars)
	}
	return b.String()
}
```

- [ ] **Step 4: Run the tests to verify they pass**

Run: `go test ./internal/service/backtest/ -run 'TestVolMetrics|TestRenderVolatility' -v`
Expected: PASS — all four tests.

- [ ] **Step 5: Commit**

```bash
git add internal/service/backtest/volatility_screen.go internal/service/backtest/volatility_screen_test.go
git commit -m "feat(backtest): VolMetrics + volatility markdown renderer"
```

---

### Task 3: CLI wiring — `-volrank` flag and `runVolRank` orchestrator

**Files:**
- Modify: `cmd/backtest/main.go` (flag block ~lines 32-53; `run()` signature/guards ~lines 62-118; add `runVolRank` near `runScreen` ~line 396; imports at top)

**Interfaces:**
- Consumes: `svc.VolMetrics`, `svc.VolRow`, `svc.VolMeta`, `svc.RenderVolatilityMarkdown` (Task 2); `svc.NewCandleProvider`, `resolveShare`/`shareInfo` pattern, `writeFile`, `semaphore.New` from `tinvest/pkg/semaphore`, `enum.Day1`.
- Produces: a working `go run ./cmd/backtest -volrank ...` command that writes `reports/volatility/volatility_Day1_<timestamp>.md`.

- [ ] **Step 1: Add the worker-count constant and confirm imports**

At the top of `cmd/backtest/main.go`, ensure these imports are present (add any missing): `"sync"`, `"sync/atomic"`, and `"tinvest/pkg/semaphore"`. Add a package-level constant near the other consts:

```go
const volWorkers = 6 // concurrent candle fetches for -volrank
```

- [ ] **Step 2: Add the flags**

In the `flag` var block in `main()` (after the `screen` flag, ~line 52), add:

```go
		volRank     = flag.Bool("volrank", false, "volatility mode: rank the liquid RUB universe by daily ATR%% (ignores -ticker/-strategy/-interval)")
		minTurnover = flag.Float64("min-turnover", 50, "volrank: minimum mean daily turnover in millions of RUB")
		atrPeriod   = flag.Int("atr-period", 14, "volrank: ATR period on the daily timeframe")
		topN        = flag.Int("top", 50, "volrank: rows in the report (0 = all)")
```

- [ ] **Step 3: Thread the new flags into the `run` call**

Update the `run(...)` invocation in `main()` (currently ending `*basket, *gridDir, *riskPct, *screen)`) to append the four new values:

```go
	if err := run(*ticker, *strategyName, interval, *months, *cash, *fraction, *commission,
		*paramsPath, *calibrate, *metric, *minTrades, *testMonths, *trainMonths, *outDir, *refresh, *explain,
		*basket, *gridDir, *riskPct, *screen,
		*volRank, *minTurnover, *atrPeriod, *topN); err != nil {
		log.Fatalf("backtest: %v", err)
	}
```

- [ ] **Step 4: Extend the `run` signature and add the guard + dispatch**

Change the `run` signature line to append the new parameters:

```go
func run(ticker, strategyName string, interval enum.Interval, months int, cash, fraction, commission float64,
	paramsPath, calibratePath, metric string, minTrades, testMonths, trainMonths int, outDir string, refresh bool, explain string,
	basketCSV, gridDir string, riskPct float64, screenCSV string,
	volRank bool, minTurnoverM float64, atrPeriod, topN int,
) error {
```

Immediately after the existing `screenCSV` guard block (the `if screenCSV != "" && (...)` return, ~line 104), add:

```go
	if volRank && (screenCSV != "" || basketCSV != "" || calibratePath != "" || explain != "") {
		return fmt.Errorf("-volrank is standalone (not combined with -screen/-basket/-calibrate/-explain)")
	}
```

Then, right after the `if screenCSV != "" { return runScreen(...) }` block (~line 118), add the dispatch:

```go
	if volRank {
		return runVolRank(ctx, client, months, atrPeriod, topN, minTurnoverM, outDir, refresh)
	}
```

- [ ] **Step 5: Implement `runVolRank`**

Add this function next to `runScreen` in `cmd/backtest/main.go`:

```go
// runVolRank ranks the liquid RUB share universe by normalized daily ATR (ATR%)
// over the last `months`, writing a markdown report. It fetches daily candles
// concurrently (volWorkers) — different tickers hit different cache files, so
// the only shared state guarded is the result slice.
func runVolRank(ctx context.Context, client grpcclient.GrpcClient, months, atrPeriod, topN int,
	minTurnoverM float64, outDir string, refresh bool,
) error {
	shares, err := client.InstrumentsServiceClient().Shares(ctx)
	if err != nil {
		return fmt.Errorf("load shares: %w", err)
	}
	var universe []shareInfoT
	for _, s := range shares {
		if strings.EqualFold(s.Currency, "rub") && s.Trading {
			universe = append(universe, shareInfoT{Ticker: s.Ticker, ID: s.ID, Lot: s.Lot})
		}
	}
	if len(universe) == 0 {
		return fmt.Errorf("-volrank: no tradable RUB shares found")
	}

	provider := svc.NewCandleProvider(client.MarketDataServiceClient(), cacheDir)
	to := time.Now()
	from := to.AddDate(0, -months, 0)

	sem := semaphore.New(volWorkers)
	var wg sync.WaitGroup
	var mu sync.Mutex
	var rows []svc.VolRow
	var done int32

	for _, u := range universe {
		wg.Add(1)
		sem.Acquire()
		go func(u shareInfoT) {
			defer wg.Done()
			defer sem.Release()
			candles, err := provider.Load(ctx, u.Ticker, u.ID, enum.Day1, from, to, refresh)
			if err != nil {
				fmt.Printf("volrank %s: skip (load: %v)\n", u.Ticker, err)
				return
			}
			mean, last, turn, bars := svc.VolMetrics(candles, u.Lot, atrPeriod)
			n := atomic.AddInt32(&done, 1)
			fmt.Printf("volrank [%d/%d] %s: ATR%%=%.2f turnover=%.0fM\n", n, len(universe), u.Ticker, mean, turn)
			if bars < atrPeriod+1 || turn < minTurnoverM || mean <= 0 {
				return
			}
			mu.Lock()
			rows = append(rows, svc.VolRow{
				Ticker: u.Ticker, MeanATRpct: mean, LastATRpct: last, TurnoverM: turn, Bars: bars,
			})
			mu.Unlock()
		}(u)
	}
	wg.Wait()

	meta := svc.VolMeta{
		Months: months, ATRPeriod: atrPeriod, MinTurnover: minTurnoverM,
		Scanned: len(universe), Passed: len(rows),
	}
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return fmt.Errorf("mkdir out dir: %w", err)
	}
	stamp := time.Now().Format("20060102_150405")
	path := filepath.Join(outDir, fmt.Sprintf("volatility_Day1_%s.md", stamp))
	if err := writeFile(path, svc.RenderVolatilityMarkdown(rows, meta, topN)); err != nil {
		return err
	}
	fmt.Printf("volrank report: %s (scanned=%d passed=%d)\n", path, len(universe), len(rows))
	return nil
}

// shareInfoT carries the per-ticker data the volrank worker pool needs.
type shareInfoT struct {
	Ticker string
	ID     string
	Lot    int32
}
```

- [ ] **Step 6: Build and vet**

Run: `go build ./... && go vet ./cmd/backtest/`
Expected: no output (success). Fix any unused-import or signature-mismatch errors before proceeding.

- [ ] **Step 7: Smoke test the default invocation**

Run (requires a valid `T_BANK` token in `env/local.env`):
`go run ./cmd/backtest -volrank -months 6 -top 30 -out ./reports/volatility`
Expected: progress lines `volrank [n/N] TICKER: ATR%=… turnover=…M`, then `volrank report: reports/volatility/volatility_Day1_<timestamp>.md (scanned=… passed=…)`. Open the report and confirm a sorted table with ATR%, trend arrows, and turnover.

- [ ] **Step 8: Commit**

```bash
git add cmd/backtest/main.go
git commit -m "feat(backtest): -volrank daily-ATR volatility screener over liquid universe"
```

---

## Self-Review

**Spec coverage:**
- ATR% metric, mean (headline) + last → Task 2 `VolMetrics`, Task 2 render. ✓
- CLI flags `-volrank/-min-turnover/-atr-period/-top/-months/-out`, Day1 hardcoded → Task 3. ✓
- Universe via gRPC `Shares()`, filter RUB + Trading → Task 3 `runVolRank`. ✓
- Concurrent fetch via `pkg/semaphore` worker pool, mutex-guarded slice → Task 3. ✓
- Liquidity (turnover) + history filter → Task 2 computes turnover, Task 3 applies threshold. ✓
- Markdown report under `reports/volatility/` in `screen.go` style → Task 2 render + Task 3 write. ✓
- `ATRSeries` helper, `ATR` refactor with regression test → Task 1. ✓
- Tests for ATRSeries, VolMetrics, renderer → Tasks 1-2. ✓
- YAGNI (no CSV/charts/realtime/static FIGI) → respected; nothing added. ✓

**Placeholder scan:** No TBD/TODO; every code step has complete code. ✓

**Type consistency:** `VolRow`/`VolMeta`/`VolMetrics`/`RenderVolatilityMarkdown` signatures match between Task 2 definition and Task 3 use. `shareInfoT` is introduced (distinct from the existing `shareInfo` to avoid colliding with that type's `ID`/`Lot`-only shape). `enum.Day1`, `semaphore.New`, `provider.Load` signatures verified against the current source. ✓
