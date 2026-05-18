package indicators

import "math"

// ATR returns Wilder's Average True Range over the input bar series.
//
// Inputs must be aligned (highs[i], lows[i], closes[i] all describe the same
// bar). Returns 0 when period <= 0, when the three slices are not the same
// length, or when len(closes) < period+1 — the insufficient-history rule is
// silent (no error), mirroring VolumeConfirmed.
//
// Algorithm:
//   - True Range at bar i (i >= 1):
//       TR_i = max(High_i - Low_i, |High_i - Close_{i-1}|, |Low_i - Close_{i-1}|)
//   - Seed ATR_{period} = mean(TR_1 .. TR_{period}).
//   - For i > period: ATR_i = (ATR_{i-1} * (period - 1) + TR_i) / period.
//   - Returns ATR at the last index.
func ATR(highs, lows, closes []float64, period int) float64 {
	if period <= 0 {
		return 0
	}
	n := len(closes)
	if len(highs) != n || len(lows) != n {
		return 0
	}
	if n < period+1 {
		return 0
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

	// Wilder smoothing for every subsequent bar.
	for i := period + 1; i < n; i++ {
		atr = (atr*float64(period-1) + trueRange(i)) / float64(period)
	}
	return atr
}
