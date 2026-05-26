package indicators

import (
	"math"
	"testing"
)

func TestRSISeries_AlternatingFixture(t *testing.T) {
	closes := []float64{10, 11, 10, 11, 10, 11, 10, 11, 10, 11}
	got := RSISeries(closes, 3)

	if len(got) != len(closes) {
		t.Fatalf("len = %d, want %d", len(got), len(closes))
	}
	for i := 0; i < 3; i++ {
		if got[i] != 0 {
			t.Errorf("pos %d: got %v, want 0", i, got[i])
		}
	}
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

func TestRSISeries_TooFewClosesReturnsZeroes(t *testing.T) {
	got := RSISeries([]float64{1, 2, 3}, 3)
	if len(got) != 3 {
		t.Fatalf("len = %d, want 3", len(got))
	}
	for i, v := range got {
		if v != 0 {
			t.Errorf("pos %d: got %v, want 0", i, v)
		}
	}
}

func TestRSISeries_NoLossesBoundary(t *testing.T) {
	closes := []float64{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}
	got := RSISeries(closes, 3)
	if math.Abs(got[3]-50.0) > 0.01 {
		t.Errorf("pos 3: got %v, want 50.00 (boundary parity with instrument/rsi)", got[3])
	}
}
