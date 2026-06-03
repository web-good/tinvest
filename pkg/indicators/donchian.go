package indicators

// Donchian returns the highest high and lowest low over the last `period` bars.
//
// Returns (0, 0) when period <= 0, when highs and lows differ in length, or when
// fewer than `period` bars are available — the insufficient-history rule is silent
// (no error), mirroring ATR. The channel midpoint is the caller's (upper+lower)/2.
func Donchian(highs, lows []float64, period int) (upper, lower float64) {
	if period <= 0 {
		return 0, 0
	}
	n := len(highs)
	if len(lows) != n || n < period {
		return 0, 0
	}
	upper = highs[n-period]
	lower = lows[n-period]
	for i := n - period + 1; i < n; i++ {
		if highs[i] > upper {
			upper = highs[i]
		}
		if lows[i] < lower {
			lower = lows[i]
		}
	}
	return upper, lower
}
