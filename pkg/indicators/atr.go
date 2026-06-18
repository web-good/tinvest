package indicators

import "math"

// ATRSeries returns Wilder's Average True Range at every bar of the input
// series. The result has length len(closes): entries before the seed bar
// (index < period) are 0, the entry at index == period is the seed
// mean(TR_1..TR_period), and later entries are Wilder-smoothed. Returns an
// all-zero slice for invalid input (period <= 0, mismatched lengths, or
// len(closes) < period+1) — the insufficient-history rule is silent.
func ATRSeries(highs, lows, closes []float64, period int) []float64 {
	n := len(closes)
	out := make([]float64, n)
	if period <= 0 || len(highs) != n || len(lows) != n || n < period+1 {
		return out
	}

	trueRange := func(i int) float64 {
		hl := highs[i] - lows[i]
		hc := math.Abs(highs[i] - closes[i-1])
		lc := math.Abs(lows[i] - closes[i-1])
		return math.Max(hl, math.Max(hc, lc))
	}

	// Seed: mean of the first `period` TR values (bars 1..period).
	sum := 0.0
	for i := 1; i <= period; i++ {
		sum += trueRange(i)
	}
	atr := sum / float64(period)
	out[period] = atr

	// Wilder smoothing for every subsequent bar.
	for i := period + 1; i < n; i++ {
		atr = (atr*float64(period-1) + trueRange(i)) / float64(period)
		out[i] = atr
	}
	return out
}

// ATR returns Wilder's Average True Range at the last bar of the input series.
// It is the final element of ATRSeries; returns 0 for empty/invalid input.
//
// Algorithm:
//   - True Range at bar i (i >= 1):
//       TR_i = max(High_i - Low_i, |High_i - Close_{i-1}|, |Low_i - Close_{i-1}|)
//   - Seed ATR_{period} = mean(TR_1 .. TR_{period}).
//   - For i > period: ATR_i = (ATR_{i-1} * (period - 1) + TR_i) / period.
func ATR(highs, lows, closes []float64, period int) float64 {
	s := ATRSeries(highs, lows, closes, period)
	if len(s) == 0 {
		return 0
	}
	return s[len(s)-1]
}
