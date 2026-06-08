package indicators

import "testing"

func TestDonchian(t *testing.T) {
	tests := []struct {
		name      string
		highs     []float64
		lows      []float64
		period    int
		wantUpper float64
		wantLower float64
	}{
		{
			name:      "last 3 bars window",
			highs:     []float64{5, 7, 3, 9, 4},
			lows:      []float64{2, 1, 0, 4, 3},
			period:    3, // last 3 highs {3,9,4} -> 9 ; last 3 lows {0,4,3} -> 0
			wantUpper: 9,
			wantLower: 0,
		},
		{
			name:      "period equals length spans all bars",
			highs:     []float64{5, 7, 3},
			lows:      []float64{2, 1, 4},
			period:    3,
			wantUpper: 7,
			wantLower: 1,
		},
		{
			name:      "insufficient history is silent zero",
			highs:     []float64{5, 7},
			lows:      []float64{2, 1},
			period:    3,
			wantUpper: 0,
			wantLower: 0,
		},
		{
			name:      "length mismatch is silent zero",
			highs:     []float64{5, 7, 3},
			lows:      []float64{2, 1},
			period:    2,
			wantUpper: 0,
			wantLower: 0,
		},
		{
			name:      "period <= 0 is silent zero",
			highs:     []float64{5, 7, 3},
			lows:      []float64{2, 1, 4},
			period:    0,
			wantUpper: 0,
			wantLower: 0,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			upper, lower := Donchian(tc.highs, tc.lows, tc.period)
			if upper != tc.wantUpper || lower != tc.wantLower {
				t.Fatalf("Donchian = (%v, %v), want (%v, %v)", upper, lower, tc.wantUpper, tc.wantLower)
			}
		})
	}
}
