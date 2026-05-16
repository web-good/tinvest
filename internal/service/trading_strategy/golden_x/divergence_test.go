package golden_x

import "testing"

func TestFindRecentPivotLow_ClassicV(t *testing.T) {
	// indices:  0  1  2  3  4  5  6
	// lows:    10 9  8  6  7  9 10 — pivot at index 3 (6 < 9,8,7,10 within ±2)
	// k=2: at i=3 check lows[1]=9, lows[2]=8 (both>6), lows[4]=7, lows[5]=9 (both>6). OK.
	lows := []float64{10, 9, 8, 6, 7, 9, 10}
	got := findRecentPivotLow(lows, 2)
	if got != 3 {
		t.Fatalf("findRecentPivotLow = %d, want 3", got)
	}
}

func TestFindRecentPivotLow_NoPivot_Monotonic(t *testing.T) {
	lows := []float64{10, 9, 8, 7, 6, 5, 4}
	if got := findRecentPivotLow(lows, 2); got != -1 {
		t.Fatalf("findRecentPivotLow on monotonic = %d, want -1", got)
	}
}

func TestFindRecentPivotLow_PicksMostRecent(t *testing.T) {
	// Two pivots: i=2 (low=5) and i=6 (low=4). Want i=6.
	lows := []float64{10, 8, 5, 9, 12, 10, 4, 7, 9}
	got := findRecentPivotLow(lows, 2)
	if got != 6 {
		t.Fatalf("findRecentPivotLow = %d, want 6 (more recent)", got)
	}
}

func TestFindRecentPivotLow_EdgeExclusion(t *testing.T) {
	// A would-be pivot at index len-1 cannot be confirmed: lacks k=2 future
	// candles. Same for len-2. With k=2 the last 2 indices are excluded.
	lows := []float64{10, 9, 8, 9, 10, 5, 4} // i=6 has no future
	if got := findRecentPivotLow(lows, 2); got == 5 || got == 6 {
		t.Fatalf("findRecentPivotLow leaked unconfirmed pivot: got %d", got)
	}
}

func TestFindRecentPivotLow_EqualNeighborNotPivot(t *testing.T) {
	// strict <: equal neighbor disqualifies.
	lows := []float64{10, 8, 8, 10, 12, 11, 13}
	if got := findRecentPivotLow(lows, 2); got != -1 {
		t.Fatalf("strict-< should reject equal neighbor, got %d", got)
	}
}

func TestFindRecentPivotLow_TooShort(t *testing.T) {
	if got := findRecentPivotLow([]float64{1, 2, 3}, 2); got != -1 {
		t.Fatalf("len<2k+1 should return -1, got %d", got)
	}
}

func TestBullishDivergence_Classic(t *testing.T) {
	// Pivot low at i=2 (price=5, rsi=25). Current (last) makes a lower low
	// (price=4) with higher RSI (35). → true.
	lows := []float64{10, 8, 5, 9, 12, 10, 7, 6, 4}
	rsi := []float64{45, 35, 25, 40, 55, 50, 33, 30, 35}
	if !bullishDivergence(lows, rsi, 2) {
		t.Fatal("expected bullish divergence")
	}
}

func TestBullishDivergence_PriceLLButRSILL_NoDivergence(t *testing.T) {
	// Lower low in price, lower low in RSI too — momentum confirms, no div.
	lows := []float64{10, 8, 5, 9, 12, 10, 7, 6, 4}
	rsi := []float64{45, 35, 25, 40, 55, 50, 33, 30, 20}
	if bullishDivergence(lows, rsi, 2) {
		t.Fatal("price LL + RSI LL must not be divergence")
	}
}

func TestBullishDivergence_HigherLow_NoDivergence(t *testing.T) {
	// Current price is not a lower low → no divergence regardless of RSI.
	lows := []float64{10, 8, 5, 9, 12, 10, 7, 6, 6}
	rsi := []float64{45, 35, 25, 40, 55, 50, 33, 30, 40}
	if bullishDivergence(lows, rsi, 2) {
		t.Fatal("price not making a lower low must not be divergence")
	}
}

func TestBullishDivergence_NoPivot_NoDivergence(t *testing.T) {
	lows := []float64{10, 9, 8, 7, 6, 5, 4, 3, 2}
	rsi := []float64{50, 48, 46, 44, 42, 40, 38, 36, 50}
	if bullishDivergence(lows, rsi, 2) {
		t.Fatal("monotonic series has no pivot — no divergence")
	}
}

func TestBullishDivergence_EqualRSI_StrictBoundary(t *testing.T) {
	// strict >: equal RSI at current vs pivot does not count as HL.
	lows := []float64{10, 8, 5, 9, 12, 10, 7, 6, 4}
	rsi := []float64{45, 35, 25, 40, 55, 50, 33, 30, 25}
	if bullishDivergence(lows, rsi, 2) {
		t.Fatal("equal RSI must not count as HL (strict >)")
	}
}

func TestBullishDivergence_LengthMismatch_NoCrash(t *testing.T) {
	lows := []float64{10, 9, 8}
	rsi := []float64{50, 48}
	if bullishDivergence(lows, rsi, 2) {
		t.Fatal("length mismatch / too short must return false (no crash)")
	}
}
