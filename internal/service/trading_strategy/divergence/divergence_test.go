package divergence

import "testing"

func TestFindRecentPivotLow_ClassicV(t *testing.T) {
	lows := []float64{10, 9, 8, 6, 7, 9, 10}
	got := FindRecentPivotLow(lows, 2)
	if got != 3 {
		t.Fatalf("FindRecentPivotLow = %d, want 3", got)
	}
}

func TestFindRecentPivotLow_NoPivot_Monotonic(t *testing.T) {
	lows := []float64{10, 9, 8, 7, 6, 5, 4}
	if got := FindRecentPivotLow(lows, 2); got != -1 {
		t.Fatalf("FindRecentPivotLow on monotonic = %d, want -1", got)
	}
}

func TestFindRecentPivotLow_PicksMostRecent(t *testing.T) {
	lows := []float64{10, 8, 5, 9, 12, 10, 4, 7, 9}
	got := FindRecentPivotLow(lows, 2)
	if got != 6 {
		t.Fatalf("FindRecentPivotLow = %d, want 6 (more recent)", got)
	}
}

func TestFindRecentPivotLow_EdgeExclusion(t *testing.T) {
	lows := []float64{10, 9, 8, 9, 10, 5, 4}
	if got := FindRecentPivotLow(lows, 2); got == 5 || got == 6 {
		t.Fatalf("FindRecentPivotLow leaked unconfirmed pivot: got %d", got)
	}
}

func TestFindRecentPivotLow_EqualNeighborNotPivot(t *testing.T) {
	lows := []float64{10, 8, 8, 10, 12, 11, 13}
	if got := FindRecentPivotLow(lows, 2); got != -1 {
		t.Fatalf("strict-< should reject equal neighbor, got %d", got)
	}
}

func TestFindRecentPivotLow_TooShort(t *testing.T) {
	if got := FindRecentPivotLow([]float64{1, 2, 3}, 2); got != -1 {
		t.Fatalf("len<2k+1 should return -1, got %d", got)
	}
}

func TestBullish_Classic(t *testing.T) {
	lows := []float64{10, 8, 5, 9, 12, 10, 7, 6, 4}
	rsi := []float64{45, 35, 25, 40, 55, 50, 33, 30, 35}
	if !Bullish(lows, rsi, 2) {
		t.Fatal("expected bullish divergence")
	}
}

func TestBullish_PriceLLButRSILL_NoDivergence(t *testing.T) {
	lows := []float64{10, 8, 5, 9, 12, 10, 7, 6, 4}
	rsi := []float64{45, 35, 25, 40, 55, 50, 33, 30, 20}
	if Bullish(lows, rsi, 2) {
		t.Fatal("price LL + RSI LL must not be divergence")
	}
}

func TestBullish_HigherLow_NoDivergence(t *testing.T) {
	lows := []float64{10, 8, 5, 9, 12, 10, 7, 6, 6}
	rsi := []float64{45, 35, 25, 40, 55, 50, 33, 30, 40}
	if Bullish(lows, rsi, 2) {
		t.Fatal("price not making a lower low must not be divergence")
	}
}

func TestBullish_NoPivot_NoDivergence(t *testing.T) {
	lows := []float64{10, 9, 8, 7, 6, 5, 4, 3, 2}
	rsi := []float64{50, 48, 46, 44, 42, 40, 38, 36, 50}
	if Bullish(lows, rsi, 2) {
		t.Fatal("monotonic series has no pivot — no divergence")
	}
}

func TestBullish_EqualRSI_StrictBoundary(t *testing.T) {
	lows := []float64{10, 8, 5, 9, 12, 10, 7, 6, 4}
	rsi := []float64{45, 35, 25, 40, 55, 50, 33, 30, 25}
	if Bullish(lows, rsi, 2) {
		t.Fatal("equal RSI must not count as HL (strict >)")
	}
}

func TestBullish_LengthMismatch_NoCrash(t *testing.T) {
	lows := []float64{10, 9, 8}
	rsi := []float64{50, 48}
	if Bullish(lows, rsi, 2) {
		t.Fatal("length mismatch / too short must return false (no crash)")
	}
}
