package backtest

// SimpleReturns converts a close series (oldest-first) into 1-bar simple returns.
// Returns an empty slice when fewer than two closes are supplied.
func SimpleReturns(closes []float64) []float64 {
	if len(closes) < 2 {
		return nil
	}
	out := make([]float64, 0, len(closes)-1)
	for i := 1; i < len(closes); i++ {
		if closes[i-1] == 0 {
			out = append(out, 0)
			continue
		}
		out = append(out, (closes[i]-closes[i-1])/closes[i-1])
	}
	return out
}

// popVar returns the population variance of xs (0 for fewer than two points).
func popVar(xs []float64) float64 {
	n := len(xs)
	if n < 2 {
		return 0
	}
	var sum float64
	for _, x := range xs {
		sum += x
	}
	mean := sum / float64(n)
	var ss float64
	for _, x := range xs {
		d := x - mean
		ss += d * d
	}
	return ss / float64(n)
}

// VarianceRatio is the Lo-MacKinlay variance ratio at horizon q:
// Var(q-bar overlapping returns) / (q * Var(1-bar returns)).
// VR < 1 indicates mean reversion, VR > 1 trending, VR ~ 1 a random walk.
// Returns 0 when undefined (q < 2, too few returns, or zero 1-bar variance).
func VarianceRatio(returns []float64, q int) float64 {
	if q < 2 || len(returns) < q+1 {
		return 0
	}
	var1 := popVar(returns)
	if var1 == 0 {
		return 0
	}
	qsums := make([]float64, 0, len(returns)-q+1)
	for i := 0; i+q <= len(returns); i++ {
		var s float64
		for j := i; j < i+q; j++ {
			s += returns[j]
		}
		qsums = append(qsums, s)
	}
	varq := popVar(qsums)
	return varq / (float64(q) * var1)
}

// Autocorr1 returns the lag-1 autocorrelation of returns (0 when undefined).
// A negative value indicates short-horizon mean reversion.
func Autocorr1(returns []float64) float64 {
	n := len(returns)
	if n < 3 {
		return 0
	}
	var sum float64
	for _, x := range returns {
		sum += x
	}
	mean := sum / float64(n)
	var num, den float64
	for i := 0; i < n; i++ {
		d := returns[i] - mean
		den += d * d
		if i > 0 {
			num += (returns[i-1] - mean) * d
		}
	}
	if den == 0 {
		return 0
	}
	return num / den
}

// MeanReversionVerdict classifies a 2-bar variance ratio into a label.
func MeanReversionVerdict(vr2 float64) string {
	switch {
	case vr2 > 0 && vr2 < 0.95:
		return "mean-reverting"
	case vr2 > 1.05:
		return "trending"
	default:
		return "neutral"
	}
}
