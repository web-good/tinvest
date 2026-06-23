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
	rsi[9] = 0   // warm-up sentinel
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
