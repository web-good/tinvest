package golden_x

import (
	"math"
	"testing"
)

func TestComputeRSISeries_AlternatingFixture(t *testing.T) {
	// Closes alternating up/down by 1: [10, 11, 10, 11, 10, 11, 10, 11, 10, 11].
	// Period = 3. Hand-traced expected values:
	//
	// Changes: [+1, -1, +1, -1, +1, -1, +1, -1, +1]
	//          (9 changes, indexes 1..9 of the closes array)
	//
	// Seed at index 3 (after 3 changes: +1, -1, +1):
	//   avgGain = (1+0+1)/3 = 0.6667
	//   avgLoss = (0+1+0)/3 = 0.3333
	//   RS = 2.0  → RSI = 100 - 100/3 ≈ 66.67
	//
	// Index 4 (change = -1):  g=0, l=1
	//   avgGain = (0.6667*2 + 0)/3 ≈ 0.4444
	//   avgLoss = (0.3333*2 + 1)/3 ≈ 0.5556
	//   RS ≈ 0.8 → RSI ≈ 44.44
	//
	// Index 5 (change = +1):  g=1, l=0
	//   avgGain = (0.4444*2 + 1)/3 ≈ 0.6296
	//   avgLoss = (0.5556*2 + 0)/3 ≈ 0.3704
	//   RS ≈ 1.7 → RSI ≈ 62.96
	closes := []float64{10, 11, 10, 11, 10, 11, 10, 11, 10, 11}
	got := computeRSISeries(closes, 3)

	if len(got) != len(closes) {
		t.Fatalf("len = %d, want %d", len(got), len(closes))
	}
	// Positions before period have no RSI.
	for i := 0; i < 3; i++ {
		if got[i] != 0 {
			t.Errorf("pos %d: got %v, want 0", i, got[i])
		}
	}
	// Check the seed and the next two Wilder steps within 0.05 (rounding-friendly).
	checkClose := func(t *testing.T, pos int, want float64) {
		t.Helper()
		if math.Abs(got[pos]-want) > 0.05 {
			t.Errorf("pos %d: got %v, want %v ±0.05", pos, got[pos], want)
		}
	}
	checkClose(t, 3, 66.67)
	checkClose(t, 4, 44.44)
	checkClose(t, 5, 62.96)
}

func TestComputeRSISeries_TooFewClosesReturnsZeroes(t *testing.T) {
	got := computeRSISeries([]float64{1, 2, 3}, 3)
	if len(got) != 3 {
		t.Fatalf("len = %d, want 3", len(got))
	}
	for i, v := range got {
		if v != 0 {
			t.Errorf("pos %d: got %v, want 0", i, v)
		}
	}
}

func TestComputeRSISeries_NoLossesBoundary(t *testing.T) {
	// Strictly increasing input: avgLoss stays at 0. Match the existing
	// instrument/rsi calculator's edge-case behavior (rs=1 → RSI=50) for
	// behavioral parity. See internal/service/instrument/rsi/calculate.go:83.
	closes := []float64{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}
	got := computeRSISeries(closes, 3)
	if math.Abs(got[3]-50.0) > 0.01 {
		t.Errorf("pos 3: got %v, want 50.00 (boundary parity with instrument/rsi)", got[3])
	}
}
