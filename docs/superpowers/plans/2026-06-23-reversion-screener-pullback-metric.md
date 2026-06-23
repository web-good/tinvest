# Reversion-скринер: pullback-in-trend метрика (Hour1) — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Заменить VR-based «mean-reversion fitness» скринера на прямую симуляцию pullback-in-trend событий на Hour1, чтобы ранжирование отражало реальную цель стратегии (buy-the-dip в аптренде).

**Architecture:** Чистая функция `PullbackStats` (domain/backtest) считает recoveryRate/eventFreq по предрасчитанным close/EMA/RSI сериям. `VolMetrics` (service) переходит на Hour1, считает pullback-статистику + справочные ATR%/VR/autocorr и дневной оборот через группировку часовых баров по дате. `ScoreVolRows` смешивает recoveryRate/eventFreq/turnover. `runVolRank` грузит Hour1 и прокидывает новые флаги.

**Tech Stack:** Go 1.25, стандартная библиотека; внутренние пакеты `tinvest/internal/domain/backtest`, `tinvest/internal/domain/ema`, `tinvest/pkg/indicators`.

## Global Constraints

- TDD: сначала падающий тест, затем минимальная реализация.
- Go naming: MixedCaps, экспортируемые функции с doc-комментариями.
- Спека-источник: `docs/superpowers/specs/2026-06-23-reversion-screener-pullback-metric-design.md`.
- Дефолты: `RSIPeriod=14`, `RSIOversold=30`, `RecoverBars=24`, `RecoverRSI=50`, `EMAPeriod=200`, `wRecov=0.5`, `wFreq=0.3`, `wLiq=0.2`, `MinTurnover=50`.
- `PullbackStats` принимает предрасчитанные серии (не считает индикаторы сам) — для изолируемой тестируемости и чтобы domain/backtest не тянул новые импорты.
- eventFreq = `events / len(closes) * 1000` (события на 1000 баров).
- Незавершённое событие (нет `RecoverBars` баров впереди) **исключается** из числителя и знаменателя.
- Warm-up sentinel: бар участвует только если `ema[i] > 0` и `rsi[i] > 0` и `rsi[i-1] > 0`.

---

## File Structure

- **Create** `internal/domain/backtest/pullback.go` — `PullbackStats`, `MeanDailyTurnoverM`.
- **Create** `internal/domain/backtest/pullback_test.go` — тесты обеих функций.
- **Modify** `internal/service/backtest/volatility_screen.go` — `VolRow`, `VolParams`, `VolStats`, `VolMetrics`, `ScoreVolRows`, `VolMeta`, `RenderVolatilityMarkdown`.
- **Modify** `internal/service/backtest/volatility_screen_test.go` — переписать под новые сигнатуры.
- **Modify** `cmd/backtest/main.go` — флаги + `runVolRank` (Hour1, новые параметры).
- **Create** `docs/reversion/screener.md` — пользовательская документация.

---

## Task 1: Чистые функции PullbackStats и MeanDailyTurnoverM

**Files:**
- Create: `internal/domain/backtest/pullback.go`
- Test: `internal/domain/backtest/pullback_test.go`

**Interfaces:**
- Consumes: `backtest.Candle` (`internal/domain/backtest/types.go`: `Time time.Time`, `Close float64`, `Volume int64`).
- Produces:
  - `func PullbackStats(closes, ema, rsi []float64, oversold, recoverRSI float64, recoverBars int) (recoveryRate, eventFreq float64, events int)`
  - `func MeanDailyTurnoverM(candles []Candle, lot int32) float64`

- [ ] **Step 1: Write the failing test**

Create `internal/domain/backtest/pullback_test.go`:

```go
package backtest

import (
	"math"
	"testing"
	"time"
)

func approxEq(a, b float64) bool { return math.Abs(a-b) < 1e-9 }

func TestPullbackStats_NoEvents(t *testing.T) {
	// RSI never crosses down into oversold (stays high) → no events.
	n := 50
	closes := make([]float64, n)
	ema := make([]float64, n)
	rsi := make([]float64, n)
	for i := range closes {
		closes[i] = 100 + float64(i) // uptrend
		ema[i] = 50                  // close always > ema
		rsi[i] = 60                  // never oversold
	}
	rate, freq, events := PullbackStats(closes, ema, rsi, 30, 50, 24)
	if events != 0 || rate != 0 || freq != 0 {
		t.Fatalf("want no events; got rate=%v freq=%v events=%d", rate, freq, events)
	}
}

func TestPullbackStats_RecoveredAndFailed(t *testing.T) {
	// Two cross-down events; first recovers, second does not.
	n := 60
	closes := make([]float64, n)
	ema := make([]float64, n)
	rsi := make([]float64, n)
	for i := range closes {
		closes[i] = 100 // flat price
		ema[i] = 50     // uptrend context (close > ema) everywhere
		rsi[i] = 60     // default: not oversold
	}
	// Event A at i=10: rsi[9]=40(>=30), rsi[10]=20(<30) → cross-down. Recovers: rsi[12]=55 (>50) within 24.
	rsi[9] = 40
	rsi[10] = 20
	rsi[12] = 55
	// Event B at i=30: rsi[29]=40, rsi[30]=20 → cross-down. Never exceeds 50 in next 24 bars → fails.
	rsi[29] = 40
	rsi[30] = 20
	// keep rsi[31..54]=60? that would recover. Set the recovery window below 50.
	for j := 31; j <= 54; j++ {
		rsi[j] = 45
	}
	rate, freq, events := PullbackStats(closes, ema, rsi, 30, 50, 24)
	if events != 2 {
		t.Fatalf("events = %d, want 2", events)
	}
	if !approxEq(rate, 0.5) {
		t.Errorf("recoveryRate = %v, want 0.5", rate)
	}
	if !approxEq(freq, float64(2)/float64(n)*1000) {
		t.Errorf("eventFreq = %v, want %v", freq, float64(2)/float64(n)*1000)
	}
}

func TestPullbackStats_TrendFilterBlocks(t *testing.T) {
	// Same cross-down but price below EMA (downtrend) → event ignored.
	n := 60
	closes := make([]float64, n)
	ema := make([]float64, n)
	rsi := make([]float64, n)
	for i := range closes {
		closes[i] = 40 // below ema → not uptrend
		ema[i] = 50
		rsi[i] = 60
	}
	rsi[9] = 40
	rsi[10] = 20
	rsi[12] = 55
	_, _, events := PullbackStats(closes, ema, rsi, 30, 50, 24)
	if events != 0 {
		t.Fatalf("events = %d, want 0 (trend filter blocks)", events)
	}
}

func TestPullbackStats_IncompleteEventExcluded(t *testing.T) {
	// Cross-down too close to the end (no recoverBars lookahead) → excluded.
	n := 20
	closes := make([]float64, n)
	ema := make([]float64, n)
	rsi := make([]float64, n)
	for i := range closes {
		closes[i] = 100
		ema[i] = 50
		rsi[i] = 60
	}
	// recoverBars=24 but only 20 bars total → every event is incomplete.
	rsi[9] = 40
	rsi[10] = 20
	_, _, events := PullbackStats(closes, ema, rsi, 30, 50, 24)
	if events != 0 {
		t.Fatalf("events = %d, want 0 (incomplete excluded)", events)
	}
}

func TestPullbackStats_WarmupSentinelSkipped(t *testing.T) {
	// rsi[i-1]==0 (warm-up) must NOT be read as >= oversold → no false cross.
	n := 60
	closes := make([]float64, n)
	ema := make([]float64, n)
	rsi := make([]float64, n)
	for i := range closes {
		closes[i] = 100
		ema[i] = 50
		rsi[i] = 60
	}
	rsi[9] = 0  // warm-up sentinel
	rsi[10] = 20 // would look like a cross-down from 0, but prev is invalid
	rsi[12] = 55
	_, _, events := PullbackStats(closes, ema, rsi, 30, 50, 24)
	if events != 0 {
		t.Fatalf("events = %d, want 0 (warm-up sentinel must not trigger)", events)
	}
}

func TestMeanDailyTurnoverM_GroupsByDate(t *testing.T) {
	d1 := time.Date(2025, 1, 2, 10, 0, 0, 0, time.UTC)
	d1b := time.Date(2025, 1, 2, 11, 0, 0, 0, time.UTC)
	d2 := time.Date(2025, 1, 3, 10, 0, 0, 0, time.UTC)
	candles := []Candle{
		{Time: d1, Close: 100, Volume: 10},  // day1: 10*1*100 = 1000
		{Time: d1b, Close: 100, Volume: 20}, // day1: 20*1*100 = 2000 → day1 total 3000
		{Time: d2, Close: 100, Volume: 50},  // day2: 50*1*100 = 5000
	}
	// mean of {3000, 5000} = 4000 → /1e6
	got := MeanDailyTurnoverM(candles, 1)
	want := 4000.0 / 1e6
	if !approxEq(got, want) {
		t.Fatalf("MeanDailyTurnoverM = %v, want %v", got, want)
	}
}

func TestMeanDailyTurnoverM_Empty(t *testing.T) {
	if got := MeanDailyTurnoverM(nil, 1); got != 0 {
		t.Fatalf("empty = %v, want 0", got)
	}
}
```

- [ ] **Step 2: Run the test to verify it fails**

Run: `go test ./internal/domain/backtest/ -run 'Pullback|MeanDailyTurnover' -v`
Expected: FAIL — `undefined: PullbackStats`, `undefined: MeanDailyTurnoverM`.

- [ ] **Step 3: Write minimal implementation**

Create `internal/domain/backtest/pullback.go`:

```go
package backtest

// PullbackStats measures "buy-the-dip in an uptrend" opportunities on a single
// series. closes/ema/rsi must be equal-length, oldest-first, index-aligned
// (rsi is the RSI series, ema the trend EMA — e.g. EMA200). A pullback event is
// a bar i where the close is above its EMA (uptrend) and RSI crosses DOWN into
// the oversold zone (rsi[i-1] >= oversold && rsi[i] < oversold). An event is
// "recovered" if RSI exceeds recoverRSI within the next recoverBars bars.
//
// Warm-up sentinels (ema<=0 or rsi<=0) are skipped so leading-zero series entries
// cannot fake a cross. Events without a full recoverBars lookahead window are
// incomplete and excluded entirely. recoveryRate is recovered/events (0 when no
// events); eventFreq is events per 1000 bars over the whole series.
func PullbackStats(closes, ema, rsi []float64, oversold, recoverRSI float64, recoverBars int) (recoveryRate, eventFreq float64, events int) {
	n := len(closes)
	if n == 0 || len(ema) != n || len(rsi) != n || recoverBars <= 0 {
		return 0, 0, 0
	}
	recovered := 0
	for i := 1; i < n; i++ {
		if i+recoverBars >= n { // no full lookahead window left → all later events incomplete
			break
		}
		if ema[i] <= 0 || rsi[i] <= 0 || rsi[i-1] <= 0 { // warm-up sentinel
			continue
		}
		if closes[i] <= ema[i] { // not in an uptrend
			continue
		}
		if !(rsi[i-1] >= oversold && rsi[i] < oversold) { // not a fresh cross-down into oversold
			continue
		}
		events++
		for j := i + 1; j <= i+recoverBars; j++ {
			if rsi[j] > recoverRSI {
				recovered++
				break
			}
		}
	}
	if events == 0 {
		return 0, 0, 0
	}
	recoveryRate = float64(recovered) / float64(events)
	eventFreq = float64(events) / float64(n) * 1000
	return recoveryRate, eventFreq, events
}

// MeanDailyTurnoverM returns the mean daily turnover (in millions of currency
// units) from a series of candles, grouping by calendar date (UTC) of each
// candle's Time. Turnover per candle is Volume*lot*Close; per-day turnovers are
// summed, then averaged across distinct days. Returns 0 for an empty series.
func MeanDailyTurnoverM(candles []Candle, lot int32) float64 {
	if len(candles) == 0 {
		return 0
	}
	type dayKey struct{ y, m, d int }
	daySum := make(map[dayKey]float64)
	for _, c := range candles {
		k := dayKey{c.Time.Year(), int(c.Time.Month()), c.Time.Day()}
		daySum[k] += float64(c.Volume) * float64(lot) * c.Close
	}
	if len(daySum) == 0 {
		return 0
	}
	var total float64
	for _, v := range daySum {
		total += v
	}
	return total / float64(len(daySum)) / 1e6
}
```

- [ ] **Step 4: Run the test to verify it passes**

Run: `go test ./internal/domain/backtest/ -run 'Pullback|MeanDailyTurnover' -v`
Expected: PASS (all 7 subtests).

- [ ] **Step 5: Commit**

```bash
git add internal/domain/backtest/pullback.go internal/domain/backtest/pullback_test.go
git commit -m "feat(screener): pullback-in-trend stats + daily turnover from intraday candles

Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>"
```

---

## Task 2: Перевод VolMetrics/VolRow/ScoreVolRows/VolMeta на pullback-метрику (Hour1)

**Files:**
- Modify: `internal/service/backtest/volatility_screen.go`
- Test: `internal/service/backtest/volatility_screen_test.go`

**Interfaces:**
- Consumes: `backtest.PullbackStats`, `backtest.MeanDailyTurnoverM` (Task 1); `backtest.SimpleReturns/VarianceRatio/Autocorr1`; `indicators.ATRSeries`, `indicators.RSISeries`; `ema.Compute`.
- Produces:
  - `type VolParams struct { ATRPeriod, EMAPeriod, RSIPeriod int; RSIOversold, RecoverRSI float64; RecoverBars int }`
  - `type VolStats struct { MeanATRpct, LastATRpct, TurnoverM, VR2, Autocorr1, RecoveryRate, EventFreq float64; Events, Bars int }`
  - `func VolMetrics(candles []backtest.Candle, lot int32, p VolParams) VolStats`
  - `func ScoreVolRows(rows []VolRow, wRecov, wFreq, wLiq float64)`
  - `VolRow` with new fields `RecoveryRate, EventFreq float64; Events int` (keeps `MeanATRpct, LastATRpct, TurnoverM, VR2, Autocorr1, Bars` as reference).
  - `VolMeta` with `WRecov, WFreq, WLiq float64; RSIPeriod, RecoverBars int; RSIOversold, RecoverRSI float64` (drops `MaxVR, DroppedTrending, WVol, WRev`).

- [ ] **Step 1: Rewrite the test file (failing)**

Replace the entire contents of `internal/service/backtest/volatility_screen_test.go`:

```go
package backtest

import (
	"strings"
	"testing"
	"time"

	domainbt "tinvest/internal/domain/backtest"
)

func screenParams() VolParams {
	return VolParams{ATRPeriod: 14, EMAPeriod: 5, RSIPeriod: 3, RSIOversold: 30, RecoverRSI: 50, RecoverBars: 5}
}

func TestVolMetrics_BasicShape(t *testing.T) {
	// 40 bars, short periods so EMA/RSI/ATR series are valid; turnover positive.
	candles := make([]domainbt.Candle, 40)
	base := time.Date(2025, 1, 1, 10, 0, 0, 0, time.UTC)
	for i := range candles {
		c := 100.0 + float64(i%5) // oscillate a little so RSI moves
		candles[i] = domainbt.Candle{
			Time: base.Add(time.Duration(i) * time.Hour),
			High: c + 1, Low: c - 1, Close: c, Volume: 100,
		}
	}
	got := VolMetrics(candles, 1, screenParams())
	if got.Bars != 40 {
		t.Fatalf("bars = %d, want 40", got.Bars)
	}
	if got.MeanATRpct <= 0 || got.LastATRpct <= 0 {
		t.Errorf("ATR%% mean=%v last=%v, want both > 0", got.MeanATRpct, got.LastATRpct)
	}
	if got.TurnoverM <= 0 {
		t.Errorf("turnover = %v, want > 0", got.TurnoverM)
	}
}

func TestVolMetrics_InsufficientHistory(t *testing.T) {
	candles := []domainbt.Candle{
		{Time: time.Now(), High: 10, Low: 9, Close: 9.5, Volume: 100},
		{Time: time.Now(), High: 11, Low: 9.5, Close: 10.5, Volume: 100},
	}
	got := VolMetrics(candles, 1, VolParams{ATRPeriod: 14, EMAPeriod: 200, RSIPeriod: 14, RSIOversold: 30, RecoverRSI: 50, RecoverBars: 24})
	if got.Bars != 2 {
		t.Fatalf("bars = %d, want 2", got.Bars)
	}
	if got.MeanATRpct != 0 || got.LastATRpct != 0 {
		t.Errorf("ATR%% mean=%v last=%v, want 0 (no valid ATR)", got.MeanATRpct, got.LastATRpct)
	}
	if got.Events != 0 {
		t.Errorf("events = %d, want 0 (no history)", got.Events)
	}
}

func TestVolMetrics_DetectsPullbackRecovery(t *testing.T) {
	// Build an uptrend (close > EMA) with a sharp RSI dip that then recovers.
	// 60 bars rising slowly; inject a multi-bar drop around i=40 to push RSI
	// into oversold, then resume rising to recover RSI above 50.
	n := 60
	candles := make([]domainbt.Candle, n)
	base := time.Date(2025, 1, 1, 10, 0, 0, 0, time.UTC)
	price := 100.0
	for i := 0; i < n; i++ {
		switch {
		case i >= 40 && i < 44:
			price -= 4 // sharp dip → RSI falls
		default:
			price += 1 // steady uptrend
		}
		candles[i] = domainbt.Candle{
			Time: base.Add(time.Duration(i) * time.Hour),
			High: price + 0.5, Low: price - 0.5, Close: price, Volume: 100,
		}
	}
	got := VolMetrics(candles, 1, VolParams{ATRPeriod: 14, EMAPeriod: 10, RSIPeriod: 6, RSIOversold: 35, RecoverRSI: 50, RecoverBars: 10})
	if got.Events < 1 {
		t.Fatalf("events = %d, want >= 1 (a recoverable dip in uptrend)", got.Events)
	}
	if got.EventFreq <= 0 {
		t.Errorf("eventFreq = %v, want > 0", got.EventFreq)
	}
}

func TestScoreVolRows_RewardsRecoveryFreqLiquidity(t *testing.T) {
	rows := []VolRow{
		{Ticker: "GOOD", RecoveryRate: 0.9, EventFreq: 10, TurnoverM: 500},
		{Ticker: "WEAK", RecoveryRate: 0.2, EventFreq: 1, TurnoverM: 50},
		{Ticker: "MID", RecoveryRate: 0.5, EventFreq: 5, TurnoverM: 200},
	}
	ScoreVolRows(rows, 0.5, 0.3, 0.2)
	byTicker := map[string]float64{}
	for _, r := range rows {
		byTicker[r.Ticker] = r.Score
	}
	if !(byTicker["GOOD"] > byTicker["MID"] && byTicker["MID"] > byTicker["WEAK"]) {
		t.Errorf("score order wrong: GOOD=%.3f MID=%.3f WEAK=%.3f", byTicker["GOOD"], byTicker["MID"], byTicker["WEAK"])
	}
	if byTicker["GOOD"] < 0.999 {
		t.Errorf("GOOD = %.3f, want ~1.0 (top on every dimension)", byTicker["GOOD"])
	}
}

func TestScoreVolRows_IgnoresATRAndVR(t *testing.T) {
	// ATR% and VR2 differ wildly but recovery/freq/liq are identical → equal score.
	rows := []VolRow{
		{Ticker: "A", RecoveryRate: 0.5, EventFreq: 5, TurnoverM: 100, MeanATRpct: 9.0, VR2: 0.3},
		{Ticker: "B", RecoveryRate: 0.5, EventFreq: 5, TurnoverM: 100, MeanATRpct: 0.5, VR2: 1.4},
	}
	ScoreVolRows(rows, 0.5, 0.3, 0.2)
	if rows[0].Score != rows[1].Score {
		t.Errorf("ATR%%/VR must not affect score: A=%.4f B=%.4f", rows[0].Score, rows[1].Score)
	}
}

func TestScoreVolRows_SingleRow(t *testing.T) {
	rows := []VolRow{{Ticker: "ONLY", RecoveryRate: 0.5, EventFreq: 5, TurnoverM: 100}}
	ScoreVolRows(rows, 0.5, 0.3, 0.2)
	if rows[0].Score == 0 {
		t.Errorf("single row score = 0, want > 0")
	}
}

func TestRenderVolatilityMarkdown_SortsByScore(t *testing.T) {
	rows := []VolRow{
		{Ticker: "AAA", Name: "Alpha Co", RecoveryRate: 0.3, EventFreq: 2, TurnoverM: 200, MeanATRpct: 1.0, VR2: 0.9, Autocorr1: -0.1, Score: 0.20, Bars: 300},
		{Ticker: "BBB", Name: "Beta Co", RecoveryRate: 0.8, EventFreq: 6, TurnoverM: 50, MeanATRpct: 3.0, VR2: 0.7, Autocorr1: -0.3, Score: 0.80, Bars: 300},
	}
	meta := VolMeta{Months: 12, ATRPeriod: 14, MinTurnover: 50, WRecov: 0.5, WFreq: 0.3, WLiq: 0.2, RSIPeriod: 14, RSIOversold: 30, RecoverBars: 24, RecoverRSI: 50, Scanned: 100, Passed: 2}
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
	for _, want := range []string{"Alpha Co", "Beta Co", "Score", "Восстановл", "VR(2)", "Autocorr"} {
		if !strings.Contains(out, want) {
			t.Errorf("expected %q in output; got: %q", want, out)
		}
	}
}

func TestRenderVolatilityMarkdown_TopN(t *testing.T) {
	rows := []VolRow{
		{Ticker: "AAA", Score: 0.1, Bars: 300},
		{Ticker: "BBB", Score: 0.9, Bars: 300},
		{Ticker: "CCC", Score: 0.5, Bars: 300},
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

- [ ] **Step 2: Run the test to verify it fails**

Run: `go test ./internal/service/backtest/ -run 'VolMetrics|ScoreVolRows|RenderVolatility' -v`
Expected: FAIL — compile errors (`VolParams`, `VolStats` undefined; old `VolMetrics` signature mismatch).

- [ ] **Step 3: Rewrite `volatility_screen.go`**

Replace the entire contents of `internal/service/backtest/volatility_screen.go`:

```go
package backtest

import (
	"fmt"
	"sort"
	"strings"

	"tinvest/internal/domain/backtest"
	"tinvest/internal/domain/ema"
	"tinvest/pkg/indicators"
)

// VolRow is one ticker's Hour1 reversion-fitness result.
type VolRow struct {
	Ticker       string
	Name         string  // company / instrument name
	RecoveryRate float64 // share of pullback-in-trend dips that recover within RecoverBars
	EventFreq    float64 // pullback events per 1000 bars
	Events       int     // completed pullback events
	MeanATRpct   float64 // mean Hour1 ATR% (reference column, not scored)
	LastATRpct   float64 // latest Hour1 ATR% (reference)
	TurnoverM    float64 // mean daily turnover in millions of RUB
	VR2          float64 // Lo-MacKinlay variance ratio at q=2 (reference column, not scored)
	Autocorr1    float64 // lag-1 autocorrelation (reference)
	Score        float64 // composite reversion-fitness score (filled by ScoreVolRows)
	Bars         int
}

// VolParams holds the pullback-metric configuration for one screening run.
type VolParams struct {
	ATRPeriod   int     // ATR period for the reference ATR% column
	EMAPeriod   int     // trend EMA period (uptrend context, e.g. 200)
	RSIPeriod   int     // RSI period for the dip trigger
	RSIOversold float64 // oversold threshold RSI crosses down through
	RecoverRSI  float64 // RSI level that marks a recovered dip
	RecoverBars int     // bars allowed for a dip to recover
}

// VolStats are the per-ticker metrics computed by VolMetrics.
type VolStats struct {
	MeanATRpct   float64
	LastATRpct   float64
	TurnoverM    float64
	VR2          float64
	Autocorr1    float64
	RecoveryRate float64
	EventFreq    float64
	Events       int
	Bars         int
}

// VolMeta carries the run parameters shown in the report header.
type VolMeta struct {
	Months      int
	ATRPeriod   int
	MinTurnover float64
	RSIPeriod   int
	RSIOversold float64
	RecoverBars int
	RecoverRSI  float64
	WRecov      float64 // composite weights
	WFreq       float64
	WLiq        float64
	Scanned     int // universe size after the currency/trading filter
	Passed      int // rows that cleared liquidity/history filters (scored)
}

// VolMetrics computes the Hour1 reversion-fitness metrics for one ticker. The
// headline metrics are RecoveryRate/EventFreq (pullback-in-trend opportunity);
// ATR%, VR2 and Autocorr1 are reference-only. TurnoverM is the mean daily
// turnover (Volume*lot*Close summed per calendar day, averaged), in millions.
func VolMetrics(candles []backtest.Candle, lot int32, p VolParams) VolStats {
	out := VolStats{Bars: len(candles)}
	if out.Bars == 0 {
		return out
	}

	highs := make([]float64, out.Bars)
	lows := make([]float64, out.Bars)
	closes := make([]float64, out.Bars)
	for i, c := range candles {
		highs[i] = c.High
		lows[i] = c.Low
		closes[i] = c.Close
	}
	out.TurnoverM = backtest.MeanDailyTurnoverM(candles, lot)

	returns := backtest.SimpleReturns(closes)
	out.VR2 = backtest.VarianceRatio(returns, 2)
	out.Autocorr1 = backtest.Autocorr1(returns)

	atrSeries := indicators.ATRSeries(highs, lows, closes, p.ATRPeriod)
	pctSum := 0.0
	count := 0
	for i, atr := range atrSeries {
		if atr > 0 && closes[i] > 0 {
			pct := atr / closes[i] * 100
			pctSum += pct
			count++
			out.LastATRpct = pct
		}
	}
	if count > 0 {
		out.MeanATRpct = pctSum / float64(count)
	}

	trendEMA := ema.Compute(closes, p.EMAPeriod)
	rsi := indicators.RSISeries(closes, p.RSIPeriod)
	out.RecoveryRate, out.EventFreq, out.Events = backtest.PullbackStats(
		closes, trendEMA, rsi, p.RSIOversold, p.RecoverRSI, p.RecoverBars,
	)
	return out
}

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
// across recovery rate, event frequency and liquidity (turnover). ATR% and VR2
// are reference-only and do NOT contribute. Percentiles are relative to the rows
// passed in.
func ScoreVolRows(rows []VolRow, wRecov, wFreq, wLiq float64) {
	n := len(rows)
	if n == 0 {
		return
	}
	recov := make([]float64, n)
	freq := make([]float64, n)
	liq := make([]float64, n)
	for i, r := range rows {
		recov[i] = r.RecoveryRate
		freq[i] = r.EventFreq
		liq[i] = r.TurnoverM
	}
	pRecov, pFreq, pLiq := percentileRanks(recov), percentileRanks(freq), percentileRanks(liq)
	for i := range rows {
		rows[i].Score = wRecov*pRecov[i] + wFreq*pFreq[i] + wLiq*pLiq[i]
	}
}

// RenderVolatilityMarkdown renders the reversion-fitness screen as a Markdown
// table ranked by Score descending. When topN > 0 the table is truncated to the
// top N rows.
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
	b.WriteString("# Скринер пригодности под reversion (Hour1, pullback-in-trend)\n\n")
	fmt.Fprintf(&b, "Окно: %d мес (Hour1); EMA-тренд + RSI(%d), отскок: вход при пересечении вниз %.0f, успех если RSI > %.0f за %d баров; порог ликвидности: %.0f млн ₽/день.\n",
		meta.Months, meta.RSIPeriod, meta.RSIOversold, meta.RecoverRSI, meta.RecoverBars, meta.MinTurnover)
	fmt.Fprintf(&b, "Просканировано %d тикеров (RUB, торгуемые); прошло фильтр: %d.\n\n", meta.Scanned, meta.Passed)
	fmt.Fprintf(&b, "Score = %.2g·перцентиль(восстановление) + %.2g·перцентиль(частота) + %.2g·перцентиль(оборот). ATR%%/VR(2) — справочные, в Score не входят. Ранжир по Score (убыв.).\n\n",
		meta.WRecov, meta.WFreq, meta.WLiq)
	b.WriteString("| # | Тикер | Название | Score | Восстановл., % | События/1000 | Кол-во соб. | Оборот, млн ₽/день | ATR% (H1) | VR(2) | Autocorr | Баров |\n")
	b.WriteString("|---|---|---|---|---|---|---|---|---|---|---|---|\n")
	for i, r := range sorted {
		fmt.Fprintf(&b, "| %d | %s | %s | %.3f | %.1f | %.2f | %d | %.1f | %.2f | %.3f | %+.3f | %d |\n",
			i+1, r.Ticker, r.Name, r.Score, r.RecoveryRate*100, r.EventFreq, r.Events,
			r.TurnoverM, r.MeanATRpct, r.VR2, r.Autocorr1, r.Bars)
	}
	return b.String()
}
```

- [ ] **Step 4: Run the test to verify it passes**

Run: `go test ./internal/service/backtest/ -run 'VolMetrics|ScoreVolRows|RenderVolatility' -v`
Expected: PASS.

- [ ] **Step 5: Run the full package test + vet to catch fallout**

Run: `go test ./internal/service/backtest/... && go vet ./internal/service/backtest/...`
Expected: PASS (no other callers of the changed symbols inside the package besides `cmd`, which is fixed in Task 3 — `cmd` is a separate package so this still builds).

- [ ] **Step 6: Commit**

```bash
git add internal/service/backtest/volatility_screen.go internal/service/backtest/volatility_screen_test.go
git commit -m "feat(screener): pullback-in-trend fitness metric, drop VR scoring/trend-exclusion

Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>"
```

---

## Task 3: Перевод runVolRank и флагов на Hour1 + новые параметры

**Files:**
- Modify: `cmd/backtest/main.go` (flags block ~57-64; dispatch ~137-138; `runVolRank` ~478-558)

**Interfaces:**
- Consumes: `svc.VolParams`, `svc.VolMetrics`, `svc.VolRow`, `svc.VolStats`, `svc.ScoreVolRows`, `svc.VolMeta`, `svc.RenderVolatilityMarkdown` (Task 2); `enum.Hour1`.
- Produces: updated CLI surface (no exported Go API).

- [ ] **Step 1: Check for flag-name collisions**

Run: `grep -n 'flag\.\(Int\|Float64\|Bool\|String\)("rsi-period"\|"rsi-oversold"\|"recover-bars"\|"recover-rsi"\|"w-recov"\|"w-freq"' cmd/backtest/main.go`
Expected: no output (names are free). If any collide, prefix with `screen-` and adjust the steps below.

- [ ] **Step 2: Replace the volrank flag block**

In `cmd/backtest/main.go`, find the existing volrank flags (currently):

```go
		maxVR        = flag.Float64("max-vr", 1.05, "volrank: drop tickers whose VR(2) exceeds this (trend exclusion)")
		wVol         = flag.Float64("w-vol", 0.4, "volrank: composite weight on ATR%% percentile")
		wRev         = flag.Float64("w-rev", 0.4, "volrank: composite weight on mean-reversion percentile")
		wLiq         = flag.Float64("w-liq", 0.2, "volrank: composite weight on turnover percentile")
```

Replace those four lines with:

```go
		wRecov       = flag.Float64("w-recov", 0.5, "volrank: composite weight on recovery-rate percentile")
		wFreq        = flag.Float64("w-freq", 0.3, "volrank: composite weight on event-frequency percentile")
		wLiq         = flag.Float64("w-liq", 0.2, "volrank: composite weight on turnover percentile")
		rsiPeriodS   = flag.Int("rsi-period", 14, "volrank: RSI period for the pullback trigger")
		rsiOversold  = flag.Float64("rsi-oversold", 30, "volrank: RSI oversold threshold (cross-down trigger)")
		recoverBars  = flag.Int("recover-bars", 24, "volrank: bars allowed for a dip's RSI to recover")
		recoverRSI   = flag.Float64("recover-rsi", 50, "volrank: RSI level marking a recovered dip")
```

- [ ] **Step 3: Update the dispatch call**

Find:

```go
	if volRank {
		return runVolRank(ctx, client, months, atrPeriod, topN, minTurnoverM, maxVR, wVol, wRev, wLiq, outDir, refresh)
	}
```

Replace with:

```go
	if volRank {
		sp := svc.VolParams{
			ATRPeriod:   *atrPeriod,
			EMAPeriod:   screenTrendEMA,
			RSIPeriod:   *rsiPeriodS,
			RSIOversold: *rsiOversold,
			RecoverRSI:  *recoverRSI,
			RecoverBars: *recoverBars,
		}
		return runVolRank(ctx, client, months, topN, minTurnoverM, *wRecov, *wFreq, *wLiq, sp, outDir, refresh)
	}
```

NOTE: `atrPeriod`, `topN`, `months`, `minTurnoverM`, `outDir`, `refresh` are already dereferenced/named in the surrounding code exactly as the current call uses them — match the existing local names (e.g. if the current call passes `atrPeriod` not `*atrPeriod`, then `sp.ATRPeriod` must use the same form). Verify against the current call site before editing.

- [ ] **Step 4: Add the trend-EMA constant**

Near the top `const` block that defines `volWorkers` (~line 32), add:

```go
	screenTrendEMA = 200 // volrank: trend EMA period (uptrend context for pullback events)
```

- [ ] **Step 5: Rewrite runVolRank**

Replace the whole `runVolRank` function (~478-558) with:

```go
// runVolRank ranks the liquid RUB share universe by Hour1 pullback-in-trend
// fitness over the last `months`, writing a markdown report. It fetches Hour1
// candles concurrently (volWorkers) — different tickers hit different cache
// files, so the only shared state guarded is the result slice.
func runVolRank(ctx context.Context, client grpcclient.GrpcClient, months, topN int,
	minTurnoverM, wRecov, wFreq, wLiq float64, sp svc.VolParams, outDir string, refresh bool,
) error {
	shares, err := client.InstrumentsServiceClient().Shares(ctx)
	if err != nil {
		return fmt.Errorf("load shares: %w", err)
	}
	var universe []shareInfoT
	for _, s := range shares {
		if strings.EqualFold(s.Currency, "rub") && s.Trading {
			universe = append(universe, shareInfoT{Ticker: s.Ticker, Name: s.Name, ID: s.ID, Lot: s.Lot})
		}
	}
	if len(universe) == 0 {
		return fmt.Errorf("-volrank: no tradable RUB shares found")
	}

	provider := svc.NewCandleProvider(client.MarketDataServiceClient(), cacheDir)
	to := time.Now()
	from := to.AddDate(0, -months, 0)
	minBars := sp.EMAPeriod + sp.RecoverBars

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
			candles, err := provider.Load(ctx, u.Ticker, u.ID, enum.Hour1, from, to, refresh)
			if err != nil {
				fmt.Printf("volrank %s: skip (load: %v)\n", u.Ticker, err)
				return
			}
			st := svc.VolMetrics(candles, u.Lot, sp)
			n := atomic.AddInt32(&done, 1)
			fmt.Printf("volrank [%d/%d] %s: recov=%.0f%% freq=%.2f turnover=%.0fM bars=%d\n",
				n, len(universe), u.Ticker, st.RecoveryRate*100, st.EventFreq, st.TurnoverM, st.Bars)
			if st.Bars < minBars || st.TurnoverM < minTurnoverM {
				return
			}
			mu.Lock()
			rows = append(rows, svc.VolRow{
				Ticker: u.Ticker, Name: u.Name,
				RecoveryRate: st.RecoveryRate, EventFreq: st.EventFreq, Events: st.Events,
				MeanATRpct: st.MeanATRpct, LastATRpct: st.LastATRpct, TurnoverM: st.TurnoverM,
				VR2: st.VR2, Autocorr1: st.Autocorr1, Bars: st.Bars,
			})
			mu.Unlock()
		}(u)
	}
	wg.Wait()

	svc.ScoreVolRows(rows, wRecov, wFreq, wLiq)

	meta := svc.VolMeta{
		Months: months, ATRPeriod: sp.ATRPeriod, MinTurnover: minTurnoverM,
		RSIPeriod: sp.RSIPeriod, RSIOversold: sp.RSIOversold, RecoverBars: sp.RecoverBars, RecoverRSI: sp.RecoverRSI,
		WRecov: wRecov, WFreq: wFreq, WLiq: wLiq,
		Scanned: len(universe), Passed: len(rows),
	}
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return fmt.Errorf("mkdir out dir: %w", err)
	}
	stamp := time.Now().Format("20060102_150405")
	path := filepath.Join(outDir, fmt.Sprintf("volatility_Hour1_%s.md", stamp))
	if err := writeFile(path, svc.RenderVolatilityMarkdown(rows, meta, topN)); err != nil {
		return err
	}
	fmt.Printf("volrank report: %s (scanned=%d passed=%d)\n", path, len(universe), len(rows))
	return nil
}
```

- [ ] **Step 6: Build & vet**

Run: `go build ./... && go vet ./cmd/backtest/...`
Expected: clean (no unused-variable errors — every removed flag must be gone, every new flag used).

- [ ] **Step 7: Smoke run (manual, optional but recommended)**

Run a tiny smoke against the existing cache (no network needed if cache is warm):
`go run ./cmd/backtest -volrank -months 12 -top 20 -out ./reports/volotility`
Expected: prints `volrank [n/N] TICKER: recov=..% freq=.. turnover=..M bars=..` lines and writes `reports/volotility/volatility_Hour1_<stamp>.md` with the new columns. (If no cache and no network, this step is skipped — note it in the task report.)

- [ ] **Step 8: Commit**

```bash
git add cmd/backtest/main.go
git commit -m "feat(screener): -volrank runs on Hour1 with pullback-metric flags

Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>"
```

---

## Task 4: Документация скринера

**Files:**
- Create: `docs/reversion/screener.md`

**Interfaces:** none (docs only). Mirror the tone/structure of `docs/reversion/strategy.md`.

- [ ] **Step 1: Write `docs/reversion/screener.md`**

Create the file:

````markdown
# Скринер пригодности под reversion (`-volrank`)

Инструмент предварительного отбора кандидатов для mean-reversion / buy-the-dip
стратегии (`docs/reversion/strategy.md`). Сканирует торгуемую RUB-вселенную акций
и ранжирует тикеры по тому, **как часто на них случаются ловибельные откаты RSI
внутри восходящего тренда и как часто эти откаты восстанавливаются** — то есть по
самой возможности, которую эксплуатирует стратегия, а не по абстрактному
mean-reversion.

> **Статус и дисклеймер.** Это **prefilter кандидатов**, а не ранжир, которому
> можно слепо верить. Финальное решение по тикеру всегда принимается отдельным
> per-ticker walk-forward (`-calibrate -train-months ...`). Скринер отвечает на
> вопрос «есть ли тут вообще что ловить и хватает ли ликвидности», а не «заработает
> ли стратегия».

## Что считается (на Hour1)

Все метрики считаются на **часовых** свечах (`Hour1`) — на том же таймфрейме, что
торгует стратегия.

### Pullback-событие (headline-метрика)

Бар `i` считается pullback-событием, если одновременно:

1. **Тренд-контекст:** `close[i] > EMA(EMAPeriod)[i]` — цена выше трендовой EMA
   (по умолчанию EMA200), т.е. мы в аптренде.
2. **Триггер просадки:** `RSI(RSIPeriod)` пересекает **вниз** порог `RSIOversold`
   (`rsi[i-1] >= oversold && rsi[i] < oversold`).

Событие считается **восстановившимся**, если в течение следующих `RecoverBars`
баров `RSI` поднимается **выше** `RecoverRSI` (по умолчанию 50 — это первичный
выход самой стратегии `RSI50`).

Из этого получаем две величины на тикер:

- **Восстановл., %** (`RecoveryRate`) — доля событий, которые восстановились
  (качество edge);
- **События/1000** (`EventFreq`) — число событий на 1000 баров (торгуемость:
  достаточно ли сделок).

События у самого конца истории (без полного окна `RecoverBars` впереди)
считаются незавершёнными и исключаются.

### Score

```
Score = w-recov·перцентиль(восстановление)
      + w-freq ·перцентиль(частота событий)
      + w-liq  ·перцентиль(оборот)
```

Перцентили — относительно прошедших фильтр тикеров. По умолчанию
`w-recov=0.5, w-freq=0.3, w-liq=0.2`.

### Справочные колонки (в Score НЕ входят)

- **ATR% (H1)** — средний часовой ATR в процентах от цены. Раньше был осью Score,
  но высокий ATR — это риск, а не edge, поэтому теперь только справочно.
- **VR(2)** — variance ratio Lo-MacKinlay (q=2), `<1` ⇒ mean-reverting, `>1` ⇒
  трендовость. Раньше использовался для отсева «трендовых» имён — этот фильтр
  **убран**, потому что стратегия как раз торгует откаты внутри тренда. Колонка
  оставлена справочно.
- **Autocorr** — лаг-1 автокорреляция доходностей.

## Фильтры (флоры гигиены)

- **Ликвидность:** средний **дневной** оборот (`Volume·lot·Close`, суммируется по
  календарным дням и усредняется) ≥ `min-turnover` (по умолчанию 50 млн ₽/день).
- **История:** минимум `EMAPeriod + RecoverBars` баров (прогрев EMA + окно
  восстановления). Более короткие тикеры пропускаются.

## Запуск

```bash
go run ./cmd/backtest -volrank -months 12 -top 50 -out ./reports/volotility
```

Отчёт пишется в `<out>/volatility_Hour1_<timestamp>.md`.

### Флаги

| Флаг | По умолч. | Назначение |
|---|---|---|
| `-volrank` | — | включить режим скринера (standalone) |
| `-months` | 12 | окно истории |
| `-top` | 50 | строк в отчёте (0 = все) |
| `-min-turnover` | 50 | порог дневного оборота, млн ₽ |
| `-atr-period` | 14 | период ATR для справочной колонки |
| `-rsi-period` | 14 | период RSI для триггера просадки |
| `-rsi-oversold` | 30 | порог oversold (пересечение вниз) |
| `-recover-bars` | 24 | сколько баров даётся на восстановление |
| `-recover-rsi` | 50 | уровень RSI, считающийся восстановлением |
| `-w-recov` | 0.5 | вес перцентиля восстановления в Score |
| `-w-freq` | 0.3 | вес перцентиля частоты в Score |
| `-w-liq` | 0.2 | вес перцентиля оборота в Score |

Трендовая EMA (контекст аптренда) зафиксирована на 200 и флагом не настраивается.

## Известные ограничения

- История не нормирована полностью: есть только грубый порог по числу баров;
  перцентили всё ещё сравнивают тикеры с разной длиной истории.
- Веса Score (`0.5/0.3/0.2`) подобраны эвристически, не калиброваны против
  реализованных walk-forward PF.
- VR считается только на q=2; пороги вердикта узкие. Это справочные величины.
- Дневной оборот группируется по UTC-дате — границы дня приблизительные, для
  порога ликвидности это несущественно.
````

- [ ] **Step 2: Commit**

```bash
git add docs/reversion/screener.md
git commit -m "docs(screener): user guide for the pullback-in-trend -volrank screener

Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>"
```

---

## Self-Review

**Spec coverage:**
- Пункт 1 (Hour1) → Task 3 (load `enum.Hour1`), Task 2 (метрики на Hour1). ✅
- Пункт 2 (pullback цель + удалить VR-фильтр) → Task 1 (`PullbackStats`), Task 2 (Score без VR, фильтр удалён). ✅
- ATR% из Score в колонку (пункт 5) → Task 2. ✅
- Дневной оборот из Hour1 → Task 1 (`MeanDailyTurnoverM`), Task 2 (вызов). ✅
- Порог истории `200+RecoverBars` → Task 3. ✅
- Флаги → Task 3. ✅
- Отчёт/колонки → Task 2 (`RenderVolatilityMarkdown`). ✅
- Документация → Task 4. ✅
- Тесты (TDD) → Task 1 (7 тестов), Task 2 (8 тестов). ✅

**Placeholder scan:** нет TBD/TODO; весь код приведён целиком.

**Type consistency:** `VolParams`/`VolStats`/`VolRow`/`VolMeta` поля и сигнатуры
`PullbackStats`/`MeanDailyTurnoverM`/`VolMetrics`/`ScoreVolRows`/`RenderVolatilityMarkdown`
совпадают между Task 1→2→3. `screenTrendEMA` (Task 3 const) = `EMAPeriod` (VolParams).
`RecoverBars`/`RecoverRSI`/`RSIOversold`/`RSIPeriod` именованы одинаково везде.
````
