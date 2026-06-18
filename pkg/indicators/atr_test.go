package indicators

import (
	"math"
	"testing"
)

func TestATR(t *testing.T) {
	tests := []struct {
		name    string
		highs   []float64
		lows    []float64
		closes  []float64
		period  int
		want    float64
		tol     float64
	}{
		{
			name:   "constant TR=2: ATR equals 2 regardless of close drift",
			highs:  []float64{12, 12, 12, 12, 12, 12, 12, 12, 12, 12, 12, 12, 12, 12, 12},
			lows:   []float64{10, 10, 10, 10, 10, 10, 10, 10, 10, 10, 10, 10, 10, 10, 10},
			closes: []float64{11, 11, 11, 11, 11, 11, 11, 11, 11, 11, 11, 11, 11, 11, 11},
			period: 14,
			want:   2.0,
			tol:    1e-9,
		},
		{
			name:   "len equals period — insufficient (need period+1)",
			highs:  []float64{12, 12, 12, 12, 12, 12, 12, 12, 12, 12, 12, 12, 12, 12},
			lows:   []float64{10, 10, 10, 10, 10, 10, 10, 10, 10, 10, 10, 10, 10, 10},
			closes: []float64{11, 11, 11, 11, 11, 11, 11, 11, 11, 11, 11, 11, 11, 11},
			period: 14,
			want:   0.0,
			tol:    0,
		},
		{
			name:   "len equals period+1 — returns the seed (single TR)",
			highs:  []float64{12, 12, 12, 12},
			lows:   []float64{10, 10, 10, 10},
			closes: []float64{11, 11, 11, 11},
			period: 3,
			want:   2.0,
			tol:    1e-9,
		},
		{
			name:   "Wilder smoothing nudges ATR toward larger TR",
			// Period 3 so the example is small enough to hand-check.
			// TRs (i=1..6): 2, 2, 2, 6, 2, 2.
			//   Seed ATR_3 = mean(2,2,2) = 2.
			//   ATR_4 = (2*2 + 6) / 3 = 10/3 ≈ 3.3333.
			//   ATR_5 = (3.3333*2 + 2) / 3 ≈ 2.8889.
			//   ATR_6 = (2.8889*2 + 2) / 3 ≈ 2.5926.
			highs:  []float64{12, 12, 12, 12, 16, 12, 12},
			lows:   []float64{10, 10, 10, 10, 10, 10, 10},
			closes: []float64{11, 11, 11, 11, 11, 11, 11},
			period: 3,
			want:   2.5925925925925926,
			tol:    1e-9,
		},
		{
			name:   "period <= 0 is silent zero",
			highs:  []float64{12, 12, 12},
			lows:   []float64{10, 10, 10},
			closes: []float64{11, 11, 11},
			period: 0,
			want:   0.0,
			tol:    0,
		},
		{
			name:   "negative period is silent zero",
			highs:  []float64{12, 12, 12},
			lows:   []float64{10, 10, 10},
			closes: []float64{11, 11, 11},
			period: -3,
			want:   0.0,
			tol:    0,
		},
		{
			name:   "length mismatch (highs shorter) is silent zero",
			highs:  []float64{12, 12, 12},
			lows:   []float64{10, 10, 10, 10},
			closes: []float64{11, 11, 11, 11},
			period: 3,
			want:   0.0,
			tol:    0,
		},
		{
			name:   "length mismatch (closes shorter) is silent zero",
			highs:  []float64{12, 12, 12, 12},
			lows:   []float64{10, 10, 10, 10},
			closes: []float64{11, 11, 11},
			period: 3,
			want:   0.0,
			tol:    0,
		},
		{
			name:   "empty input is silent zero",
			highs:  nil,
			lows:   nil,
			closes: nil,
			period: 14,
			want:   0.0,
			tol:    0,
		},
		{
			name:   "gap up beats high-low: TR uses |high - prevClose|",
			// Bar 0: high=10, low=8, close=9.
			// Bar 1: high=12, low=11, close=11.5. TR = max(12-11, |12-9|, |11-9|) = 3.
			// Period 1 means seed ATR_1 = TR_1 = 3.
			highs:  []float64{10, 12},
			lows:   []float64{8, 11},
			closes: []float64{9, 11.5},
			period: 1,
			want:   3.0,
			tol:    1e-9,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := ATR(tc.highs, tc.lows, tc.closes, tc.period)
			if math.Abs(got-tc.want) > tc.tol {
				t.Fatalf("ATR = %v, want %v (tol %v)", got, tc.want, tc.tol)
			}
		})
	}
}

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
