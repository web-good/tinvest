package indicators

// Stochastic returns the latest %K and %D of the stochastic oscillator.
// %K = 100 * (close - lowestLow) / (highestHigh - lowestLow) over the last kPeriod
// bars; %D = simple moving average of %K over the last dSmooth %K values. Both are 0
// when history is insufficient (fewer than kPeriod bars, or fewer than dSmooth %K
// values for %D) or when the high/low range collapses to zero.
func Stochastic(highs, lows, closes []float64, kPeriod, dSmooth int) (k, d float64) {
	n := len(closes)
	if kPeriod <= 0 || dSmooth <= 0 || n < kPeriod || len(highs) < n || len(lows) < n {
		return 0, 0
	}

	ks := make([]float64, 0, n-kPeriod+1)
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

	k = ks[len(ks)-1]
	if len(ks) < dSmooth {
		return k, 0
	}
	var sum float64
	for i := len(ks) - dSmooth; i < len(ks); i++ {
		sum += ks[i]
	}
	return k, sum / float64(dSmooth)
}
