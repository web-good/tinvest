package golden_x

import (
	"math"
	"testing"
)

func TestPercentile_R7(t *testing.T) {
	tests := []struct {
		name   string
		sorted []float64
		p      float64
		want   float64
	}{
		{
			// R-7 reference: percentile([1..100], 5) = 1 + 0.05*99 = 5.95
			name:   "p5 of 1..100",
			sorted: rangeFloat(1, 100),
			p:      5,
			want:   5.95,
		},
		{
			// R-7 reference: percentile([1..100], 15) = 1 + 0.15*99 = 15.85
			name:   "p15 of 1..100",
			sorted: rangeFloat(1, 100),
			p:      15,
			want:   15.85,
		},
		{
			name:   "single element returns itself",
			sorted: []float64{42},
			p:      5,
			want:   42,
		},
		{
			name:   "all equal returns the common value",
			sorted: []float64{5, 5, 5, 5, 5},
			p:      50,
			want:   5,
		},
		{
			name:   "p=0 returns the smallest",
			sorted: []float64{10, 20, 30, 40},
			p:      0,
			want:   10,
		},
		{
			name:   "p=100 returns the largest",
			sorted: []float64{10, 20, 30, 40},
			p:      100,
			want:   40,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := percentile(tc.sorted, tc.p)
			if math.Abs(got-tc.want) > 1e-9 {
				t.Fatalf("percentile(%v, %v) = %v, want %v", tc.sorted, tc.p, got, tc.want)
			}
		})
	}
}

// rangeFloat returns [from, from+1, …, to] inclusive.
func rangeFloat(from, to int) []float64 {
	out := make([]float64, 0, to-from+1)
	for i := from; i <= to; i++ {
		out = append(out, float64(i))
	}
	return out
}

func TestTierFromAdaptive(t *testing.T) {
	tests := []struct {
		name string
		rsi  float64
		p5   float64
		p15  float64
		want alertTier
	}{
		{"rsi strictly below p5 → Green", 20, 24, 31, tierGreen},
		{"rsi == p5 → Yellow (strict <)", 24, 24, 31, tierYellow},
		{"rsi between p5 and p15 → Yellow", 28, 24, 31, tierYellow},
		{"rsi == p15 → None (strict <)", 31, 24, 31, tierNone},
		{"rsi above p15 → None", 40, 24, 31, tierNone},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := tierFromAdaptive(tc.rsi, tc.p5, tc.p15)
			if got != tc.want {
				t.Fatalf("tierFromAdaptive(%v, %v, %v) = %v, want %v", tc.rsi, tc.p5, tc.p15, got, tc.want)
			}
		})
	}
}

func TestAdaptiveThresholds(t *testing.T) {
	// percentile([1..100], 5)  = 5.95
	// percentile([1..100], 15) = 15.85
	rsi := rangeFloat(1, 100)
	got := adaptiveThresholds(rsi)
	if math.Abs(got.P5-5.95) > 1e-9 {
		t.Errorf("P5 = %v, want 5.95", got.P5)
	}
	if math.Abs(got.P15-15.85) > 1e-9 {
		t.Errorf("P15 = %v, want 15.85", got.P15)
	}
}

func TestAdaptiveThresholds_DoesNotMutateInput(t *testing.T) {
	// Input may arrive in any order; helper must sort defensively without
	// scrambling the caller's slice.
	in := []float64{50, 10, 30, 20, 40}
	original := append([]float64(nil), in...)
	_ = adaptiveThresholds(in)
	for i := range in {
		if in[i] != original[i] {
			t.Fatalf("input mutated at %d: got %v, want %v", i, in[i], original[i])
		}
	}
}

func TestAdaptiveSellThresholds(t *testing.T) {
	// R-7 reference (mirrors TestAdaptiveThresholds p5=5.95, p15=15.85):
	//   percentile([1..100], 80) = 1 + 0.80*99 = 80.20
	//   percentile([1..100], 90) = 1 + 0.90*99 = 90.10
	//   percentile([1..100], 95) = 1 + 0.95*99 = 95.05
	rsi := rangeFloat(1, 100)
	got := adaptiveSellThresholds(rsi)
	if math.Abs(got.P80-80.20) > 1e-9 {
		t.Errorf("P80 = %v, want 80.20", got.P80)
	}
	if math.Abs(got.P90-90.10) > 1e-9 {
		t.Errorf("P90 = %v, want 90.10", got.P90)
	}
	if math.Abs(got.P95-95.05) > 1e-9 {
		t.Errorf("P95 = %v, want 95.05", got.P95)
	}
}

func TestAdaptiveSellThresholds_DoesNotMutateInput(t *testing.T) {
	in := []float64{50, 10, 30, 20, 40, 90, 70, 60, 80, 100}
	original := append([]float64(nil), in...)
	_ = adaptiveSellThresholds(in)
	for i := range in {
		if in[i] != original[i] {
			t.Fatalf("input mutated at %d: got %v, want %v", i, in[i], original[i])
		}
	}
}
