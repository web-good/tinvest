package golden_x

import "math"

// computeRSISeries returns the Wilder RSI value for each position in closes.
// Positions before index `period` are zero (no RSI defined yet). The result
// has the same length as the input.
//
// Boundary parity note: when accumulated avgLoss is exactly 0 (a strict run of
// gains across the entire warmup), this implementation matches the existing
// internal/service/instrument/rsi/calculate.go behavior of falling back to
// rs=1 → RSI=50, rather than the mathematically correct RSI=100. This keeps
// signal behavior consistent for any share that briefly hits the boundary.
func computeRSISeries(closes []float64, period int) []float64 {
	out := make([]float64, len(closes))
	if len(closes) <= period {
		return out
	}

	p := float64(period)
	var avgGain, avgLoss float64
	// Seed: SMA of the first `period` price changes (closes[1] − closes[0],
	// closes[2] − closes[1], … closes[period] − closes[period-1]).
	for i := 1; i <= period; i++ {
		ch := closes[i] - closes[i-1]
		if ch > 0 {
			avgGain += ch
		} else {
			avgLoss += -ch
		}
	}
	avgGain /= p
	avgLoss /= p
	out[period] = wilderRSI(avgGain, avgLoss)

	for i := period + 1; i < len(closes); i++ {
		ch := closes[i] - closes[i-1]
		var g, l float64
		if ch > 0 {
			g = ch
		} else {
			l = -ch
		}
		avgGain = (avgGain*(p-1) + g) / p
		avgLoss = (avgLoss*(p-1) + l) / p
		out[i] = wilderRSI(avgGain, avgLoss)
	}
	return out
}

// wilderRSI converts smoothed average gain/loss into an RSI value, rounding to
// two decimal places to match the existing calculator's quantization.
func wilderRSI(avgGain, avgLoss float64) float64 {
	rs := 1.0
	if avgLoss != 0 {
		rs = avgGain / avgLoss
	}
	rsi := 100 - 100/(1+rs)
	return math.Round(rsi*100) / 100
}
