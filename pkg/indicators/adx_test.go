package indicators

import (
	"math"
	"testing"
)

func TestADX(t *testing.T) {
	tests := []struct {
		name           string
		highs          []float64
		lows           []float64
		closes         []float64
		period         int
		wantADX        float64 // checked with tol
		wantADXTol     float64
		assertDirected func(t *testing.T, diPlus, diMinus float64)
	}{
		{
			// Strict uptrend: every -DM is 0, so DX == 100 on every bar => ADX == 100,
			// -DI == 0, +DI > 0. Robust to the exact Wilder seed convention.
			name:       "strict uptrend -> adx 100, diMinus 0",
			highs:      []float64{10, 11, 12, 13, 14, 15},
			lows:       []float64{9, 10, 11, 12, 13, 14},
			closes:     []float64{9.5, 10.5, 11.5, 12.5, 13.5, 14.5},
			period:     2,
			wantADX:    100,
			wantADXTol: 1e-9,
			assertDirected: func(t *testing.T, diPlus, diMinus float64) {
				if diMinus != 0 {
					t.Errorf("diMinus = %v, want 0", diMinus)
				}
				if diPlus <= 0 {
					t.Errorf("diPlus = %v, want > 0", diPlus)
				}
			},
		},
		{
			// Strict downtrend: mirror image. +DM is 0 => DX == 100 => ADX == 100,
			// +DI == 0, -DI > 0.
			name:       "strict downtrend -> adx 100, diPlus 0",
			highs:      []float64{15, 14, 13, 12, 11, 10},
			lows:       []float64{14, 13, 12, 11, 10, 9},
			closes:     []float64{14.5, 13.5, 12.5, 11.5, 10.5, 9.5},
			period:     2,
			wantADX:    100,
			wantADXTol: 1e-9,
			assertDirected: func(t *testing.T, diPlus, diMinus float64) {
				if diPlus != 0 {
					t.Errorf("diPlus = %v, want 0", diPlus)
				}
				if diMinus <= 0 {
					t.Errorf("diMinus = %v, want > 0", diMinus)
				}
			},
		},
		{
			// Flat bars: no directional movement, +DI == -DI == 0 => DX guarded to 0 => ADX 0.
			name:       "flat -> adx 0",
			highs:      []float64{12, 12, 12, 12, 12, 12},
			lows:       []float64{10, 10, 10, 10, 10, 10},
			closes:     []float64{11, 11, 11, 11, 11, 11},
			period:     2,
			wantADX:    0,
			wantADXTol: 1e-9,
			assertDirected: func(t *testing.T, diPlus, diMinus float64) {
				if diPlus != 0 || diMinus != 0 {
					t.Errorf("diPlus=%v diMinus=%v, want 0,0", diPlus, diMinus)
				}
			},
		},
		{
			name:       "period <= 0 is silent zero",
			highs:      []float64{10, 11, 12, 13, 14},
			lows:       []float64{9, 10, 11, 12, 13},
			closes:     []float64{9.5, 10.5, 11.5, 12.5, 13.5},
			period:     0,
			wantADX:    0,
			wantADXTol: 0,
			assertDirected: func(t *testing.T, diPlus, diMinus float64) {
				if diPlus != 0 || diMinus != 0 {
					t.Errorf("diPlus=%v diMinus=%v, want 0,0", diPlus, diMinus)
				}
			},
		},
		{
			name:       "insufficient history (need 2*period+1) is silent zero",
			highs:      []float64{10, 11, 12, 13}, // n=4 < 2*2+1=5
			lows:       []float64{9, 10, 11, 12},
			closes:     []float64{9.5, 10.5, 11.5, 12.5},
			period:     2,
			wantADX:    0,
			wantADXTol: 0,
			assertDirected: func(t *testing.T, diPlus, diMinus float64) {
				if diPlus != 0 || diMinus != 0 {
					t.Errorf("diPlus=%v diMinus=%v, want 0,0", diPlus, diMinus)
				}
			},
		},
		{
			name:       "length mismatch is silent zero",
			highs:      []float64{10, 11, 12, 13, 14},
			lows:       []float64{9, 10, 11, 12}, // shorter
			closes:     []float64{9.5, 10.5, 11.5, 12.5, 13.5},
			period:     2,
			wantADX:    0,
			wantADXTol: 0,
			assertDirected: func(t *testing.T, diPlus, diMinus float64) {
				if diPlus != 0 || diMinus != 0 {
					t.Errorf("diPlus=%v diMinus=%v, want 0,0", diPlus, diMinus)
				}
			},
		},
		{
			name:       "negative period is silent zero",
			highs:      []float64{10, 11, 12, 13, 14},
			lows:       []float64{9, 10, 11, 12, 13},
			closes:     []float64{9.5, 10.5, 11.5, 12.5, 13.5},
			period:     -3,
			wantADX:    0,
			wantADXTol: 0,
			assertDirected: func(t *testing.T, diPlus, diMinus float64) {
				if diPlus != 0 || diMinus != 0 {
					t.Errorf("diPlus=%v diMinus=%v, want 0,0", diPlus, diMinus)
				}
			},
		},
		{
			name:       "nil/empty input is silent zero",
			highs:      nil,
			lows:       nil,
			closes:     nil,
			period:     2,
			wantADX:    0,
			wantADXTol: 0,
			assertDirected: func(t *testing.T, diPlus, diMinus float64) {
				if diPlus != 0 || diMinus != 0 {
					t.Errorf("diPlus=%v diMinus=%v, want 0,0", diPlus, diMinus)
				}
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			adx, diPlus, diMinus := ADX(tc.highs, tc.lows, tc.closes, tc.period)
			if math.Abs(adx-tc.wantADX) > tc.wantADXTol {
				t.Fatalf("ADX = %v, want %v (tol %v)", adx, tc.wantADX, tc.wantADXTol)
			}
			tc.assertDirected(t, diPlus, diMinus)
		})
	}
}

// TestADX_MixedExercisesBlending verifies the non-trivial "blending" path where both
// +DM and −DM are positive throughout the series, so DX is strictly inside (0, 100)
// and ADX must be too. Exact values are intentionally not pinned — multi-bar Wilder
// smoothing is too error-prone to compute by hand.
func TestADX_MixedExercisesBlending(t *testing.T) {
	// Alternating but net-up series: both directional indicators receive contributions,
	// so neither DI collapses to zero and neither dominates completely.
	highs := []float64{10, 12, 11, 13, 12, 14}
	lows := []float64{8, 9, 8, 10, 9, 11}
	closes := []float64{9, 11, 9, 12, 10, 13}
	period := 2

	adx, diPlus, diMinus := ADX(highs, lows, closes, period)

	// ADX must be strictly inside the open interval (0, 100) — the blending path.
	if adx <= 0 || adx >= 100 {
		t.Fatalf("ADX = %v: want strictly in (0, 100); got adx=%v diPlus=%v diMinus=%v",
			adx, adx, diPlus, diMinus)
	}

	// Both DIs must be positive: the series has both up-moves and down-moves.
	if diPlus <= 0 {
		t.Errorf("diPlus = %v, want > 0 (series has upward directional movement)", diPlus)
	}
	if diMinus <= 0 {
		t.Errorf("diMinus = %v, want > 0 (series has downward directional movement)", diMinus)
	}

	// Net-up series: +DI must dominate −DI.
	if diPlus <= diMinus {
		t.Errorf("diPlus=%v <= diMinus=%v, want diPlus > diMinus for net-up series", diPlus, diMinus)
	}
}
