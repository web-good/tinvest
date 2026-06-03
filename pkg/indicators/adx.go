package indicators

import "math"

// ADX returns Wilder's Average Directional Index together with the directional
// indicators +DI and -DI over the input bar series.
//
// Inputs must be aligned (highs[i], lows[i], closes[i] all describe the same bar).
// Returns (0, 0, 0) when period <= 0, when the slices are not the same length, or
// when len(closes) < 2*period+1 — the insufficient-history rule is silent (no error),
// mirroring ATR. The +2*period+1 minimum covers the double smoothing: DI/DX seed over
// the first `period` increments, then ADX seeds over the next `period` DX values.
//
// Returned values are the indicators at the last bar.
func ADX(highs, lows, closes []float64, period int) (adx, diPlus, diMinus float64) {
	if period <= 0 {
		return 0, 0, 0
	}
	n := len(closes)
	if len(highs) != n || len(lows) != n {
		return 0, 0, 0
	}
	if n < 2*period+1 {
		return 0, 0, 0
	}

	plusDM := make([]float64, n)
	minusDM := make([]float64, n)
	tr := make([]float64, n)
	for i := 1; i < n; i++ {
		up := highs[i] - highs[i-1]
		down := lows[i-1] - lows[i]
		if up > down && up > 0 {
			plusDM[i] = up
		}
		if down > up && down > 0 {
			minusDM[i] = down
		}
		hl := highs[i] - lows[i]
		hc := math.Abs(highs[i] - closes[i-1])
		lc := math.Abs(lows[i] - closes[i-1])
		tr[i] = math.Max(hl, math.Max(hc, lc))
	}

	// Wilder seeds over the first `period` increments (indices 1..period).
	var sTR, sPlus, sMinus float64
	for i := 1; i <= period; i++ {
		sTR += tr[i]
		sPlus += plusDM[i]
		sMinus += minusDM[i]
	}

	p := float64(period)
	dx := func() float64 {
		var pdi, mdi float64
		if sTR != 0 {
			pdi = 100 * sPlus / sTR
			mdi = 100 * sMinus / sTR
		}
		denom := pdi + mdi
		if denom == 0 {
			return 0
		}
		return 100 * math.Abs(pdi-mdi) / denom
	}

	// Seed ADX as the average of the first `period` DX values (indices period..2*period-1).
	dxSum := dx()
	for i := period + 1; i <= 2*period-1; i++ {
		sTR += tr[i] - sTR/p
		sPlus += plusDM[i] - sPlus/p
		sMinus += minusDM[i] - sMinus/p
		dxSum += dx()
	}
	adx = dxSum / p

	// Wilder-smooth ADX (and the underlying DI) to the last bar.
	for i := 2 * period; i < n; i++ {
		sTR += tr[i] - sTR/p
		sPlus += plusDM[i] - sPlus/p
		sMinus += minusDM[i] - sMinus/p
		adx = (adx*(p-1) + dx()) / p
	}

	if sTR != 0 {
		diPlus = 100 * sPlus / sTR
		diMinus = 100 * sMinus / sTR
	}
	return adx, diPlus, diMinus
}
