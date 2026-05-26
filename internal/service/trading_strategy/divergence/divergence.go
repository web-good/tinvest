package divergence

// FindRecentPivotLow returns the index of the most recent confirmed fractal
// pivot low in `lows`, or -1 if none exists. An index i is a confirmed pivot
// low iff lows[i] is strictly less than each of its k neighbors on both
// sides. Indices in [0, k) and (len-1-k, len-1] cannot be pivots (they lack
// k confirming neighbors on at least one side) — the search is restricted to
// [k, len-1-k]. The search proceeds from the right so the first match found
// is the most recent.
func FindRecentPivotLow(lows []float64, k int) int {
	n := len(lows)
	if k < 1 || n < 2*k+1 {
		return -1
	}
	for i := n - 1 - k; i >= k; i-- {
		if isPivotLow(lows, i, k) {
			return i
		}
	}
	return -1
}

func isPivotLow(lows []float64, i, k int) bool {
	v := lows[i]
	for off := 1; off <= k; off++ {
		if !(v < lows[i-off]) || !(v < lows[i+off]) {
			return false
		}
	}
	return true
}

// Bullish reports whether the last entry of `lows`/`rsi` forms a
// bullish RSI divergence against the most recent confirmed fractal pivot low
// (window k) found earlier in the same slices. Both strict: price strictly
// lower than the pivot's low AND RSI strictly higher than the pivot's RSI.
// Returns false if no pivot exists, if the slices are length-mismatched,
// or if either inequality fails. The slices are assumed aligned 1-to-1 by
// the caller (same candle index → same array index).
func Bullish(lows, rsi []float64, k int) bool {
	if len(lows) != len(rsi) {
		return false
	}
	p := FindRecentPivotLow(lows, k)
	if p < 0 {
		return false
	}
	last := len(lows) - 1
	return lows[last] < lows[p] && rsi[last] > rsi[p]
}
