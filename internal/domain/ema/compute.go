package ema

// Compute returns a sliding EMA of the given period over closes.
// Result has the same length as closes; positions before period-1 are zero.
// The seed is an SMA of the first `period` values.
func Compute(closes []float64, period int) []float64 {
	out := make([]float64, len(closes))
	if len(closes) < period {
		return out
	}

	var sum float64
	for i := 0; i < period; i++ {
		sum += closes[i]
	}
	out[period-1] = sum / float64(period)
	multiplier := 2.0 / float64(period+1)

	for k := period; k < len(closes); k++ {
		out[k] = (closes[k]-out[k-1])*multiplier + out[k-1]
	}
	return out
}
