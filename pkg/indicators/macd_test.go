package indicators

import (
	"math"
	"testing"
)

func approx(a, b float64) bool { return math.Abs(a-b) < 1e-9 }

func TestMACDInsufficientHistory(t *testing.T) {
	m, s := MACD([]float64{1, 2, 3}, 12, 26, 9)
	if m != nil || s != nil {
		t.Fatalf("want nil,nil for short history; got %v,%v", m, s)
	}
}

func TestMACDInvalidPeriods(t *testing.T) {
	closes := make([]float64, 100)
	for i := range closes {
		closes[i] = float64(i)
	}
	if m, s := MACD(closes, 26, 12, 9); m != nil || s != nil { // fast>=slow
		t.Fatalf("want nil for fast>=slow")
	}
	if m, s := MACD(closes, 0, 26, 9); m != nil || s != nil {
		t.Fatalf("want nil for non-positive period")
	}
}

func TestMACDRisingSeriesPositive(t *testing.T) {
	// A steadily rising series: fast EMA above slow EMA => MACD line positive at the end.
	closes := make([]float64, 100)
	for i := range closes {
		closes[i] = float64(i)
	}
	m, s := MACD(closes, 12, 26, 9)
	if len(m) != len(closes) || len(s) != len(closes) {
		t.Fatalf("lengths: macd=%d signal=%d want %d", len(m), len(s), len(closes))
	}
	if m[len(m)-1] <= 0 {
		t.Fatalf("rising series: macd[last]=%f want >0", m[len(m)-1])
	}
}

func TestMACDLineEqualsFastMinusSlowEMA(t *testing.T) {
	closes := make([]float64, 60)
	for i := range closes {
		closes[i] = math.Sin(float64(i)/5) * 10
	}
	m, _ := MACD(closes, 12, 26, 9)
	fast := ema(closes, 12)
	slow := ema(closes, 26)
	last := len(closes) - 1
	if !approx(m[last], fast[last]-slow[last]) {
		t.Fatalf("macd[last]=%f want fast-slow=%f", m[last], fast[last]-slow[last])
	}
}
