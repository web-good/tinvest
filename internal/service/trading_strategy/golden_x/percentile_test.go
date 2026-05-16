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
