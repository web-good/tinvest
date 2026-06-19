# Reversion-fitness Screener Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Turn `-volrank` from a pure ATR% ranking into a reversion-fitness screener that ranks the liquid RUB universe by a composite of volatility + mean-reversion + liquidity, and hard-excludes trending tickers.

**Architecture:** Extend the existing per-ticker `VolMetrics` to also emit the Lo-MacKinlay variance ratio VR(2) and lag-1 autocorrelation (both already implemented in `internal/domain/backtest/variance_ratio.go`). A new pure `ScoreVolRows` blends percentile ranks of the three dimensions into a `Score`. `runVolRank` applies a hard trend gate during row collection, scores the survivors, and the renderer sorts by `Score`.

**Tech Stack:** Go 1.25, standard library only. Tests via `go test`.

## Global Constraints

- Reuse `domain/backtest` helpers — do NOT reimplement variance ratio / autocorrelation: `SimpleReturns(closes []float64) []float64`, `VarianceRatio(returns []float64, q int) float64` (VR<1 mean-reverting, >1 trending, 0 undefined), `Autocorr1(returns []float64) float64`, `MeanReversionVerdict(vr2 float64) string`.
- `internal/service/backtest/volatility_screen.go` already imports `"tinvest/internal/domain/backtest"` (used as `backtest.Candle`); the domain helpers are reachable as `backtest.SimpleReturns`, etc.
- Default composite weights: `wVol=0.4, wRev=0.4, wLiq=0.2`. Default trend threshold: `maxVR=1.05`.
- Percentiles are computed over the surviving (passed + non-trending) set only.
- Commit after each task. Run `go build ./...` before every commit.

## File Structure

- `internal/service/backtest/volatility_screen.go` — extend `VolRow`, `VolMetrics`, `VolMeta`; add `ScoreVolRows` + `percentileRanks`; rewrite `RenderVolatilityMarkdown`.
- `internal/service/backtest/volatility_screen_test.go` — extend tests for VR output, scoring, render ordering.
- `cmd/backtest/main.go` — add flags `-w-vol`/`-w-rev`/`-w-liq`/`-max-vr`; thread through `run`; apply trend gate + scoring in `runVolRank`.

---

### Task 1: VolMetrics emits VR(2) and Autocorr(1)

**Files:**
- Modify: `internal/service/backtest/volatility_screen.go` (`VolMetrics`, ~line 35-69)
- Modify: `internal/service/backtest/volatility_screen_test.go` (existing `VolMetrics` call sites)
- Modify: `cmd/backtest/main.go:516` (existing `VolMetrics` caller)

**Interfaces:**
- Produces: `func VolMetrics(candles []backtest.Candle, lot int32, atrPeriod int) (meanATRpct, lastATRpct, turnoverM, vr2, autocorr1 float64, bars int)` — two new return values inserted before `bars`.

- [ ] **Step 1: Write the failing test**

Add to `volatility_screen_test.go`:

```go
func TestVolMetrics_VarianceRatio(t *testing.T) {
	// Oscillating closes (up/down/up/down) → mean-reverting → VR(2) < 1.
	osc := []domainbt.Candle{}
	for i := 0; i < 40; i++ {
		c := 100.0
		if i%2 == 1 {
			c = 110.0
		}
		osc = append(osc, domainbt.Candle{High: c + 1, Low: c - 1, Close: c, Volume: 100})
	}
	_, _, _, vrOsc, acOsc, _ := VolMetrics(osc, 1, 14)
	if vrOsc <= 0 || vrOsc >= 1 {
		t.Errorf("oscillating VR(2) = %v, want in (0,1)", vrOsc)
	}
	if acOsc >= 0 {
		t.Errorf("oscillating autocorr = %v, want negative", acOsc)
	}

	// Steadily trending closes → VR(2) > 1.
	trend := []domainbt.Candle{}
	for i := 0; i < 40; i++ {
		c := 100.0 + float64(i)
		trend = append(trend, domainbt.Candle{High: c + 1, Low: c - 1, Close: c, Volume: 100})
	}
	_, _, _, vrTrend, _, _ := VolMetrics(trend, 1, 14)
	if vrTrend <= 1 {
		t.Errorf("trending VR(2) = %v, want > 1", vrTrend)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/service/backtest/ -run TestVolMetrics_VarianceRatio`
Expected: FAIL to compile — `VolMetrics` returns 4 values, test destructures 6.

- [ ] **Step 3: Update `VolMetrics` signature and body**

In `volatility_screen.go`, change the signature and add VR/autocorr computation from the `closes` slice it already builds:

```go
func VolMetrics(candles []backtest.Candle, lot int32, atrPeriod int) (meanATRpct, lastATRpct, turnoverM, vr2, autocorr1 float64, bars int) {
	bars = len(candles)
	if bars == 0 {
		return 0, 0, 0, 0, 0, 0
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

	returns := backtest.SimpleReturns(closes)
	vr2 = backtest.VarianceRatio(returns, 2)
	autocorr1 = backtest.Autocorr1(returns)

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
		return 0, 0, turnoverM, vr2, autocorr1, bars
	}
	meanATRpct = pctSum / float64(count)
	return meanATRpct, lastATRpct, turnoverM, vr2, autocorr1, bars
}
```

- [ ] **Step 4: Fix the two existing test call sites**

In `volatility_screen_test.go`:
- `TestVolMetrics_BasicShape`: change `mean, last, turn, bars := VolMetrics(candles, 1, 3)` to `mean, last, turn, _, _, bars := VolMetrics(candles, 1, 3)`.
- `TestVolMetrics_InsufficientHistory`: change `mean, last, _, bars := VolMetrics(candles, 1, 14)` to `mean, last, _, _, _, bars := VolMetrics(candles, 1, 14)`.

- [ ] **Step 5: Fix the main.go caller**

In `cmd/backtest/main.go:516`, change:

```go
mean, last, turn, bars := svc.VolMetrics(candles, u.Lot, atrPeriod)
```
to:
```go
mean, last, turn, vr2, ac1, bars := svc.VolMetrics(candles, u.Lot, atrPeriod)
```
(`vr2`/`ac1` are wired into the row in Task 4; until then add `_ = vr2; _ = ac1` right after this line to keep the build green.)

- [ ] **Step 6: Run tests + build**

Run: `go test ./internal/service/backtest/ -run TestVolMetrics && go build ./...`
Expected: PASS, build clean.

- [ ] **Step 7: Commit**

```bash
git add internal/service/backtest/volatility_screen.go internal/service/backtest/volatility_screen_test.go cmd/backtest/main.go
git commit -m "feat(backtest): VolMetrics emits VR(2) and autocorr for reversion screening"
```

---

### Task 2: VolRow fields + ScoreVolRows composite

**Files:**
- Modify: `internal/service/backtest/volatility_screen.go` (`VolRow` struct; add `percentileRanks`, `ScoreVolRows`)
- Modify: `internal/service/backtest/volatility_screen_test.go`

**Interfaces:**
- Consumes: `VolMetrics` VR(2) output (Task 1).
- Produces:
  - `VolRow` gains fields `VR2 float64`, `Autocorr1 float64`, `Score float64`.
  - `func ScoreVolRows(rows []VolRow, wVol, wRev, wLiq float64)` — mutates `rows[i].Score` in place using percentile-rank blending; reversion percentile uses `-VR2` (lower VR2 ⇒ higher percentile).

- [ ] **Step 1: Write the failing test**

```go
func TestScoreVolRows_RewardsAllThree(t *testing.T) {
	rows := []VolRow{
		// strong: high ATR%, low VR2 (mean-reverting), high turnover
		{Ticker: "GOOD", MeanATRpct: 4.0, VR2: 0.7, TurnoverM: 500},
		// weak: low ATR%, high-ish VR2, low turnover
		{Ticker: "WEAK", MeanATRpct: 1.0, VR2: 1.0, TurnoverM: 50},
		// middle
		{Ticker: "MID", MeanATRpct: 2.5, VR2: 0.9, TurnoverM: 200},
	}
	ScoreVolRows(rows, 0.4, 0.4, 0.2)

	byTicker := map[string]float64{}
	for _, r := range rows {
		byTicker[r.Ticker] = r.Score
	}
	if !(byTicker["GOOD"] > byTicker["MID"] && byTicker["MID"] > byTicker["WEAK"]) {
		t.Errorf("score order wrong: GOOD=%.3f MID=%.3f WEAK=%.3f",
			byTicker["GOOD"], byTicker["MID"], byTicker["WEAK"])
	}
	// best on all dims ⇒ max blended score 1.0
	if byTicker["GOOD"] < 0.999 {
		t.Errorf("GOOD score = %.3f, want ~1.0 (top on every dimension)", byTicker["GOOD"])
	}
}

func TestScoreVolRows_WeightShiftsOrder(t *testing.T) {
	// A is more volatile; B mean-reverts harder. Weighting reversion flips the winner.
	mk := func() []VolRow {
		return []VolRow{
			{Ticker: "A", MeanATRpct: 5.0, VR2: 0.95, TurnoverM: 100},
			{Ticker: "B", MeanATRpct: 2.0, VR2: 0.60, TurnoverM: 100},
		}
	}
	volHeavy := mk()
	ScoreVolRows(volHeavy, 0.9, 0.1, 0.0)
	if volHeavy[0].Score <= volHeavy[1].Score {
		t.Errorf("vol-heavy weights: A must outscore B")
	}
	revHeavy := mk()
	ScoreVolRows(revHeavy, 0.1, 0.9, 0.0)
	if revHeavy[1].Score <= revHeavy[0].Score {
		t.Errorf("rev-heavy weights: B must outscore A")
	}
}

func TestScoreVolRows_SingleRow(t *testing.T) {
	rows := []VolRow{{Ticker: "ONLY", MeanATRpct: 2.0, VR2: 0.8, TurnoverM: 100}}
	ScoreVolRows(rows, 0.4, 0.4, 0.2) // must not panic; single candidate scores top
	if rows[0].Score == 0 {
		t.Errorf("single row score = 0, want > 0")
	}
}
```

- [ ] **Step 2: Run to verify it fails**

Run: `go test ./internal/service/backtest/ -run TestScoreVolRows`
Expected: FAIL — `VolRow` has no `VR2` field; `ScoreVolRows` undefined.

- [ ] **Step 3: Add fields + implement scoring**

Add the three fields to the `VolRow` struct:

```go
type VolRow struct {
	Ticker     string
	Name       string
	MeanATRpct float64
	LastATRpct float64
	TurnoverM  float64
	VR2        float64 // Lo-MacKinlay variance ratio at q=2 (<1 mean-reverting)
	Autocorr1  float64 // lag-1 autocorrelation (negative = mean-reverting)
	Score      float64 // composite reversion-fitness score (filled by ScoreVolRows)
	Bars       int
}
```

Add the helpers (place above `RenderVolatilityMarkdown`):

```go
// percentileRanks maps each value to its fractional rank in [0,1]: the smallest
// value gets 0, the largest 1, ties share a rank. A single value scores 1 (it is
// the sole — hence top — candidate).
func percentileRanks(vals []float64) []float64 {
	n := len(vals)
	out := make([]float64, n)
	if n == 1 {
		out[0] = 1
		return out
	}
	for i, v := range vals {
		less := 0
		for _, w := range vals {
			if w < v {
				less++
			}
		}
		out[i] = float64(less) / float64(n-1)
	}
	return out
}

// ScoreVolRows fills each row's Score with a weighted blend of percentile ranks
// across volatility (ATR%), mean reversion (-VR2, so lower VR2 ranks higher) and
// liquidity (turnover). Percentiles are relative to the rows passed in.
func ScoreVolRows(rows []VolRow, wVol, wRev, wLiq float64) {
	n := len(rows)
	if n == 0 {
		return
	}
	atr := make([]float64, n)
	rev := make([]float64, n)
	liq := make([]float64, n)
	for i, r := range rows {
		atr[i] = r.MeanATRpct
		rev[i] = -r.VR2
		liq[i] = r.TurnoverM
	}
	pa, pr, pl := percentileRanks(atr), percentileRanks(rev), percentileRanks(liq)
	for i := range rows {
		rows[i].Score = wVol*pa[i] + wRev*pr[i] + wLiq*pl[i]
	}
}
```

- [ ] **Step 4: Run to verify it passes**

Run: `go test ./internal/service/backtest/ -run TestScoreVolRows`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/service/backtest/volatility_screen.go internal/service/backtest/volatility_screen_test.go
git commit -m "feat(backtest): add ScoreVolRows composite reversion-fitness score"
```

---

### Task 3: Render by Score with reversion columns

**Files:**
- Modify: `internal/service/backtest/volatility_screen.go` (`VolMeta`, `RenderVolatilityMarkdown`)
- Modify: `internal/service/backtest/volatility_screen_test.go` (existing render tests)

**Interfaces:**
- Consumes: `VolRow.Score`, `VolRow.VR2`, `VolRow.Autocorr1` (Task 2); `backtest.MeanReversionVerdict` (domain).
- Produces: `VolMeta` gains `WVol, WRev, WLiq, MaxVR float64` and `DroppedTrending int`; `RenderVolatilityMarkdown` sorts by `Score` desc and renders VR/autocorr/verdict/score columns.

- [ ] **Step 1: Update the existing render tests to expect Score ordering**

Replace `TestRenderVolatilityMarkdown_SortsDescAndTrend` and `TestRenderVolatilityMarkdown_TopN` with:

```go
func TestRenderVolatilityMarkdown_SortsByScore(t *testing.T) {
	rows := []VolRow{
		{Ticker: "AAA", Name: "Alpha Co", MeanATRpct: 1.0, LastATRpct: 1.5, TurnoverM: 200, VR2: 0.9, Autocorr1: -0.1, Score: 0.20, Bars: 120},
		{Ticker: "BBB", Name: "Beta Co", MeanATRpct: 3.0, LastATRpct: 2.0, TurnoverM: 50, VR2: 0.7, Autocorr1: -0.3, Score: 0.80, Bars: 120},
	}
	meta := VolMeta{Months: 6, ATRPeriod: 14, MinTurnover: 50, MaxVR: 1.05, WVol: 0.4, WRev: 0.4, WLiq: 0.2, Scanned: 100, Passed: 2}

	out := RenderVolatilityMarkdown(rows, meta, 0)

	bbb, aaa := strings.Index(out, "BBB"), strings.Index(out, "AAA")
	if bbb == -1 || aaa == -1 {
		t.Fatalf("both tickers must appear; out=%q", out)
	}
	if bbb > aaa {
		t.Errorf("BBB (score 0.80) must rank before AAA (score 0.20)")
	}
	if strings.Contains(out, "%%") {
		t.Errorf("rendered output must not contain literal '%%%%'; got: %q", out)
	}
	for _, want := range []string{"Alpha Co", "Beta Co", "Score", "VR(2)", "Autocorr", "Вердикт"} {
		if !strings.Contains(out, want) {
			t.Errorf("expected %q in output; got: %q", want, out)
		}
	}
}

func TestRenderVolatilityMarkdown_TopN(t *testing.T) {
	rows := []VolRow{
		{Ticker: "AAA", Score: 0.1, Bars: 120},
		{Ticker: "BBB", Score: 0.9, Bars: 120},
		{Ticker: "CCC", Score: 0.5, Bars: 120},
	}
	out := RenderVolatilityMarkdown(rows, VolMeta{}, 2)
	if strings.Contains(out, "AAA") {
		t.Errorf("topN=2 must drop AAA (lowest score); out=%q", out)
	}
	if !strings.Contains(out, "BBB") || !strings.Contains(out, "CCC") {
		t.Errorf("topN=2 must keep BBB and CCC")
	}
}
```

- [ ] **Step 2: Run to verify it fails**

Run: `go test ./internal/service/backtest/ -run TestRenderVolatilityMarkdown`
Expected: FAIL — `VolMeta` has no `MaxVR`/`WVol` fields; output lacks `Score`/`VR(2)` columns; sort still by `MeanATRpct`.

- [ ] **Step 3: Extend VolMeta**

```go
type VolMeta struct {
	Months          int
	ATRPeriod       int
	MinTurnover     float64
	MaxVR           float64 // trend-exclusion threshold on VR(2)
	WVol            float64 // composite weights
	WRev            float64
	WLiq            float64
	Scanned         int // universe size after the currency/trading filter
	Passed          int // rows that cleared liquidity/history/trend filters (scored)
	DroppedTrending int // rows excluded because VR2 > MaxVR or VR2 <= 0
}
```

- [ ] **Step 4: Rewrite RenderVolatilityMarkdown**

```go
func RenderVolatilityMarkdown(rows []VolRow, meta VolMeta, topN int) string {
	sorted := make([]VolRow, len(rows))
	copy(sorted, rows)
	sort.SliceStable(sorted, func(i, j int) bool {
		return sorted[i].Score > sorted[j].Score
	})
	if topN > 0 && len(sorted) > topN {
		sorted = sorted[:topN]
	}

	var b strings.Builder
	b.WriteString("# Скринер пригодности под reversion (дневной ТФ)\n\n")
	fmt.Fprintf(&b, "Окно: %d мес; ATR(%d); порог ликвидности: %.0f млн ₽/день; трендовый отсев: VR(2) > %.2f.\n",
		meta.Months, meta.ATRPeriod, meta.MinTurnover, meta.MaxVR)
	fmt.Fprintf(&b, "Просканировано %d тикеров (RUB, торгуемые); прошло фильтр: %d; отсеяно как трендовые: %d.\n\n",
		meta.Scanned, meta.Passed, meta.DroppedTrending)
	fmt.Fprintf(&b, "Score = %.2g·перцентиль(ATR%%) + %.2g·перцентиль(возврат, −VR2) + %.2g·перцентиль(оборот). Ранжир по Score (убыв.).\n\n",
		meta.WVol, meta.WRev, meta.WLiq)
	b.WriteString("| # | Тикер | Название | Score | Ср. ATR% | Тек. ATR% | Тренд | VR(2) | Autocorr | Вердикт | Оборот, млн ₽/день | Баров |\n")
	b.WriteString("|---|---|---|---|---|---|---|---|---|---|---|---|\n")
	for i, r := range sorted {
		trend := "↓"
		if r.LastATRpct > r.MeanATRpct {
			trend = "↑"
		}
		fmt.Fprintf(&b, "| %d | %s | %s | %.3f | %.2f | %.2f | %s | %.3f | %+.3f | %s | %.1f | %d |\n",
			i+1, r.Ticker, r.Name, r.Score, r.MeanATRpct, r.LastATRpct, trend,
			r.VR2, r.Autocorr1, backtest.MeanReversionVerdict(r.VR2), r.TurnoverM, r.Bars)
	}
	return b.String()
}
```

- [ ] **Step 5: Run to verify it passes**

Run: `go test ./internal/service/backtest/ && go build ./...`
Expected: PASS, build clean.

- [ ] **Step 6: Commit**

```bash
git add internal/service/backtest/volatility_screen.go internal/service/backtest/volatility_screen_test.go
git commit -m "feat(backtest): render volatility screen by reversion-fitness score"
```

---

### Task 4: Wire flags, trend gate, and scoring into runVolRank

**Files:**
- Modify: `cmd/backtest/main.go` (flag block ~line 56-60; `run` call ~line 70-73; `run` signature ~line 99-101; `runVolRank` call ~line 134; `runVolRank` definition ~line 478-545)

**Interfaces:**
- Consumes: `svc.VolMetrics` (Task 1), `svc.ScoreVolRows` (Task 2), `svc.VolRow`/`svc.VolMeta` new fields (Tasks 2-3).
- Produces: new flags `-w-vol`, `-w-rev`, `-w-liq`, `-max-vr`; `runVolRank` signature gains `wVol, wRev, wLiq, maxVR float64`.

- [ ] **Step 1: Add the flags**

After the `topN` flag (line 60) in the `var (...)` flag block:

```go
		maxVR        = flag.Float64("max-vr", 1.05, "volrank: drop tickers whose VR(2) exceeds this (trend exclusion)")
		wVol         = flag.Float64("w-vol", 0.4, "volrank: composite weight on ATR%% percentile")
		wRev         = flag.Float64("w-rev", 0.4, "volrank: composite weight on mean-reversion percentile")
		wLiq         = flag.Float64("w-liq", 0.2, "volrank: composite weight on turnover percentile")
```

- [ ] **Step 2: Thread flags through the `run` call**

Change the `run(...)` invocation (line 70-73) final line from:

```go
		*volRank, *minTurnover, *atrPeriod, *topN); err != nil {
```
to:
```go
		*volRank, *minTurnover, *atrPeriod, *topN, *maxVR, *wVol, *wRev, *wLiq); err != nil {
```

- [ ] **Step 3: Extend the `run` signature**

Change line 101 from:

```go
	volRank bool, minTurnoverM float64, atrPeriod, topN int,
```
to:
```go
	volRank bool, minTurnoverM float64, atrPeriod, topN int, maxVR, wVol, wRev, wLiq float64,
```

- [ ] **Step 4: Pass them to `runVolRank`**

Change the call (line 134) from:

```go
		return runVolRank(ctx, client, months, atrPeriod, topN, minTurnoverM, outDir, refresh)
```
to:
```go
		return runVolRank(ctx, client, months, atrPeriod, topN, minTurnoverM, maxVR, wVol, wRev, wLiq, outDir, refresh)
```

- [ ] **Step 5: Update `runVolRank` — signature, trend gate, scoring**

Change the signature (line 478-480):

```go
func runVolRank(ctx context.Context, client grpcclient.GrpcClient, months, atrPeriod, topN int,
	minTurnoverM, maxVR, wVol, wRev, wLiq float64, outDir string, refresh bool,
) error {
```

Replace the goroutine body's metrics+filter+append block (the `mean, last, turn, ... := svc.VolMetrics(...)` line through the `mu.Unlock()`, including the temporary `_ = vr2; _ = ac1` from Task 1) with a trend-gated version. Add a `var droppedTrending int32` declaration next to `var done int32` (line 503):

```go
			mean, last, turn, vr2, ac1, bars := svc.VolMetrics(candles, u.Lot, atrPeriod)
			n := atomic.AddInt32(&done, 1)
			fmt.Printf("volrank [%d/%d] %s: ATR%%=%.2f turnover=%.0fM VR2=%.2f\n", n, len(universe), u.Ticker, mean, turn, vr2)
			if bars < atrPeriod+1 || turn < minTurnoverM || mean <= 0 {
				return
			}
			if vr2 <= 0 || vr2 > maxVR { // undefined or trending → exclude
				atomic.AddInt32(&droppedTrending, 1)
				return
			}
			mu.Lock()
			rows = append(rows, svc.VolRow{
				Ticker: u.Ticker, Name: u.Name, MeanATRpct: mean, LastATRpct: last,
				TurnoverM: turn, VR2: vr2, Autocorr1: ac1, Bars: bars,
			})
			mu.Unlock()
```

After `wg.Wait()` (line 529), score the survivors and build the extended meta:

```go
	svc.ScoreVolRows(rows, wVol, wRev, wLiq)

	meta := svc.VolMeta{
		Months: months, ATRPeriod: atrPeriod, MinTurnover: minTurnoverM, MaxVR: maxVR,
		WVol: wVol, WRev: wRev, WLiq: wLiq,
		Scanned: len(universe), Passed: len(rows), DroppedTrending: int(droppedTrending),
	}
```
(Delete the old `meta := svc.VolMeta{...}` block at line 531-534.)

- [ ] **Step 6: Build + vet**

Run: `go build ./... && go vet ./cmd/backtest/ ./internal/service/backtest/`
Expected: clean (no output).

- [ ] **Step 7: Full test suite**

Run: `go test ./internal/service/backtest/`
Expected: PASS (all volatility-screen tests).

- [ ] **Step 8: Smoke run (optional, needs API token + network)**

Run: `go run ./cmd/backtest -volrank -months 12 -top 15 -out ./reports/screener`
Expected: a `volatility_Day1_*.md` report ranked by Score, with VR(2)/Autocorr/Вердикт columns, a "отсеяно как трендовые: N" line, and no row whose verdict is "trending".

- [ ] **Step 9: Commit**

```bash
git add cmd/backtest/main.go
git commit -m "feat(backtest): volrank ranks by reversion-fitness with trend exclusion"
```

---

## Self-Review

**Spec coverage:**
- Volatility/liquidity metrics retained → Task 1 keeps `meanATRpct`/`turnoverM`. ✓
- VR(2)/Autocorr(1) from existing daily closes via domain helpers → Task 1. ✓
- Percentile-blend composite, weights 0.4/0.4/0.2, reversion uses −VR2 → Task 2. ✓
- Hard trend gate (`VR2 > maxVR` or `VR2 <= 0`), runs before scoring, percentiles over survivors → Task 4 (gate) + Task 2 (scores survivors only). ✓
- Report: rank by Score, new columns, header documents weights + threshold + dropped count → Task 3. ✓
- Flags `-w-vol/-w-rev/-w-liq/-max-vr` → Task 4. ✓
- Tests: VR output, scoring order, weight shift, edge cases, trend gate count → Tasks 1-4 (gate count surfaced via smoke run Step 8; unit-level gate logic is trivial inequality in main.go, not separately unit-tested as main.go has no test harness). ✓
- Out of scope (`-screen`, engine, volume-activity) untouched. ✓

**Placeholder scan:** No TBD/TODO; every code step shows full code. ✓

**Type consistency:** `VolMetrics` 6-return signature used identically in Tasks 1 and 4; `ScoreVolRows(rows, wVol, wRev, wLiq)` signature matches across Tasks 2 and 4; `VolRow` fields `VR2`/`Autocorr1`/`Score` and `VolMeta` fields `MaxVR`/`WVol`/`WRev`/`WLiq`/`DroppedTrending` defined in Tasks 2-3 and consumed in Tasks 3-4. ✓
