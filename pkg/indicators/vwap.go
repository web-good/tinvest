package indicators

import (
	"math"
	"time"
)

// SessionVWAP returns the session-anchored VWAP, the volume-weighted standard deviation of
// typical price ((H+L+C)/3) around it, and each bar's zero-based index within its session —
// all index-aligned to the input bars.
//
// Sessions are delimited by a change of calendar date in loc, so weekends and holidays split
// sessions without a separate rule. On bar i the sums run from that session's first bar
// through i inclusive.
//
// The FIRST session present in the window is always reported as incomplete (barsFromOpen = -1
// on every one of its bars), even when the window happens to start exactly at a session open.
// A rolling window usually starts mid-day, where the anchor would be arbitrary; keeping the
// rule unconditional makes every returned value independent of where the window was cut,
// which is what lets callers stay free of look-ahead.
//
// Returns nil, nil, nil when the inputs are empty or not index-aligned. Bars with
// non-positive volume contribute zero weight; a session whose total volume is zero keeps
// vwap = 0 and sigma = 0 rather than producing NaN. A nil loc degrades to UTC.
func SessionVWAP(highs, lows, closes []float64, volumes []int64,
	times []time.Time, loc *time.Location) (vwap, sigma []float64, barsFromOpen []int) {
	n := len(closes)
	if n == 0 || len(highs) != n || len(lows) != n || len(volumes) != n || len(times) != n {
		return nil, nil, nil
	}
	if loc == nil {
		loc = time.UTC
	}

	vwap = make([]float64, n)
	sigma = make([]float64, n)
	barsFromOpen = make([]int, n)

	// West's weighted incremental mean/variance: numerically stable for prices in the
	// hundreds carrying a variance in the fractions.
	var sumV, mean, m2 float64
	idx := 0
	firstSession := true
	var py, pd int
	var pm time.Month

	for i := 0; i < n; i++ {
		y, m, d := times[i].In(loc).Date()
		switch {
		case i == 0:
			// first bar opens the first (incomplete-by-rule) session
		case y != py || m != pm || d != pd:
			firstSession = false
			sumV, mean, m2 = 0, 0, 0
			idx = 0
		default:
			idx++
		}
		py, pm, pd = y, m, d

		if v := float64(volumes[i]); v > 0 {
			tp := (highs[i] + lows[i] + closes[i]) / 3
			sumV += v
			delta := tp - mean
			mean += delta * v / sumV
			m2 += v * delta * (tp - mean)
		}
		if sumV > 0 {
			vwap[i] = mean
			if variance := m2 / sumV; variance > 0 {
				sigma[i] = math.Sqrt(variance)
			}
		}
		if firstSession {
			barsFromOpen[i] = -1
		} else {
			barsFromOpen[i] = idx
		}
	}
	return vwap, sigma, barsFromOpen
}
