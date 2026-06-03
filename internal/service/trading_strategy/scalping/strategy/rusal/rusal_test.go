package rusal

import (
	"testing"

	"tinvest/internal/service/trading_strategy/scalping/model"
	"tinvest/internal/service/trading_strategy/scalping/strategy"
)

func TestTickerAndLookback(t *testing.T) {
	s := New()
	if s.Ticker() != "RUAL" {
		t.Errorf("Ticker = %q, want RUAL", s.Ticker())
	}
	// 6*14 + 20 + 50 = 154
	if got := s.Lookback(); got != 154 {
		t.Errorf("Lookback = %d, want 154", got)
	}
}

func TestDefaultParams(t *testing.T) {
	p := DefaultParams()
	if p.ADXTrendLevel <= p.ADXRangeLevel {
		t.Errorf("ADXTrendLevel (%v) must exceed ADXRangeLevel (%v)", p.ADXTrendLevel, p.ADXRangeLevel)
	}
	if p.EMAPeriod <= 0 || p.ADXPeriod <= 0 || p.RSIPeriod <= 0 || p.DonchianPeriod <= 0 || p.ATRPeriod <= 0 {
		t.Errorf("all periods must be positive: %+v", p)
	}
}

func TestEMATouched(t *testing.T) {
	ema := []float64{10, 10, 10, 10, 10}
	// A low at index 3 dips to the EMA within tolerance.
	lows := []float64{12, 12, 12, 10.01, 12}
	if !emaTouched(lows, ema, 3, 0.002) { // window covers indices 2,3,4
		t.Error("expected touch within last 3 bars")
	}
	if emaTouched(lows, ema, 1, 0.002) { // window covers only index 4 (low 12, no touch)
		t.Error("did not expect touch within last 1 bar")
	}
	if emaTouched(nil, nil, 3, 0.002) {
		t.Error("empty input must not touch")
	}
}

func TestRecentHigh(t *testing.T) {
	highs := []float64{5, 9, 3, 7, 4}
	if got := recentHigh(highs, 3); got != 7 { // last 3 -> {3,7,4} -> 7
		t.Errorf("recentHigh = %v, want 7", got)
	}
	if got := recentHigh(highs, 10); got != 9 { // window > len -> all -> 9
		t.Errorf("recentHigh = %v, want 9", got)
	}
	if got := recentHigh(nil, 3); got != 0 {
		t.Errorf("recentHigh(nil) = %v, want 0", got)
	}
	if got := recentHigh([]float64{5, 9, 3, 7, 4}, 0); got != 4 { // window<=0 clamps to last bar, no panic
		t.Errorf("recentHigh(window=0) = %v, want 4", got)
	}
}

func TestRegimeOf(t *testing.T) {
	s := New() // ADXTrendLevel 25, ADXRangeLevel 20
	cases := []struct {
		adx  float64
		want regime
	}{
		{0, regimeDead},   // sentinel / insufficient history
		{-5, regimeDead},  // defensive
		{30, regimeTrend}, // >= 25
		{25, regimeTrend}, // boundary
		{15, regimeRange}, // <= 20
		{20, regimeRange}, // boundary
		{22, regimeDead},  // dead zone between 20 and 25
	}
	for _, c := range cases {
		if got := s.regimeOf(c.adx); got != c.want {
			t.Errorf("regimeOf(%v) = %v, want %v", c.adx, got, c.want)
		}
	}
}

// TestDecide_FlatUptrendIsNone: a monotonic uptrend keeps RSI high (no upward cross)
// and price runs above the EMA, so no entry fires. Holds for the stub and the real core.
func TestDecide_FlatUptrendIsNone(t *testing.T) {
	s := New()
	highs := make([]float64, 200)
	lows := make([]float64, 200)
	closes := make([]float64, 200)
	for i := 0; i < 200; i++ {
		base := 100.0 + float64(i)
		highs[i] = base + 1
		lows[i] = base - 1
		closes[i] = base
	}
	md := strategy.MarketData{
		Price:  closes[199],
		Highs:  highs,
		Lows:   lows,
		Closes: closes,
	}
	got := s.Decide(md)
	if got.Kind != model.SignalNone {
		t.Fatalf("Kind = %v, want None", got.Kind)
	}
	if got.Ticker != "RUAL" {
		t.Errorf("Ticker = %q, want RUAL", got.Ticker)
	}
}
