package indicators

// StochasticSeries returns the right-aligned %K and %D series of the stochastic
// oscillator. %K[t] = 100 * (close - lowestLow) / (highestHigh - lowestLow) over the
// trailing kPeriod bars; a window whose high/low range collapses to zero yields 0. %D
// is the simple moving average of %K over dSmooth values, so len(ds) = len(ks)-dSmooth+1
// (every %D value is fully smoothed — there are no warm-up zeros). Both are nil when the
// inputs are malformed or history is shorter than kPeriod (ds also nil when fewer than
// dSmooth %K values exist).
func StochasticSeries(highs, lows, closes []float64, kPeriod, dSmooth int) (ks, ds []float64) {
	n := len(closes)
	if kPeriod <= 0 || dSmooth <= 0 || n < kPeriod || len(highs) < n || len(lows) < n {
		return nil, nil
	}

	ks = make([]float64, 0, n-kPeriod+1)
	for end := kPeriod; end <= n; end++ {
		hi, lo := highs[end-kPeriod], lows[end-kPeriod]
		for i := end - kPeriod + 1; i < end; i++ {
			if highs[i] > hi {
				hi = highs[i]
			}
			if lows[i] < lo {
				lo = lows[i]
			}
		}
		rng := hi - lo
		if rng == 0 {
			ks = append(ks, 0)
			continue
		}
		ks = append(ks, 100*(closes[end-1]-lo)/rng)
	}

	if len(ks) < dSmooth {
		return ks, nil
	}
	ds = make([]float64, 0, len(ks)-dSmooth+1)
	for j := dSmooth - 1; j < len(ks); j++ {
		var sum float64
		for i := j - dSmooth + 1; i <= j; i++ {
			sum += ks[i]
		}
		ds = append(ds, sum/float64(dSmooth))
	}
	return ks, ds
}

// Stochastic returns the latest %K and %D of the stochastic oscillator. %D is 0 when
// history is insufficient (fewer than dSmooth %K values); both are 0 when there is no %K
// at all. Thin wrapper over StochasticSeries.
func Stochastic(highs, lows, closes []float64, kPeriod, dSmooth int) (k, d float64) {
	ks, ds := StochasticSeries(highs, lows, closes, kPeriod, dSmooth)
	if len(ks) == 0 {
		return 0, 0
	}
	k = ks[len(ks)-1]
	if len(ds) == 0 {
		return k, 0
	}
	return k, ds[len(ds)-1]
}
