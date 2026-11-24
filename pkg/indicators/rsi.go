package indicators

import "math"

func RSISeries(closes []float64, period int) []float64 {
	return rsiSeries(closes, period, wilderRSI)
}

// WilderRSISeries returns unrounded RSI, with 100 for gains without losses and
// 50 for a flat market. Entries before the first complete period are zero.
func WilderRSISeries(closes []float64, period int) []float64 {
	return rsiSeries(closes, period, func(gain, loss float64) float64 {
		if gain == 0 && loss == 0 {
			return 50
		}
		return 100 * gain / (gain + loss)
	})
}

func rsiSeries(closes []float64, period int, score func(float64, float64) float64) []float64 {
	out := make([]float64, len(closes))
	if len(closes) <= period {
		return out
	}

	p := float64(period)
	var avgGain, avgLoss float64
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
	out[period] = score(avgGain, avgLoss)

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
		out[i] = score(avgGain, avgLoss)
	}
	return out
}

func wilderRSI(avgGain, avgLoss float64) float64 {
	rs := 1.0
	if avgLoss != 0 {
		rs = avgGain / avgLoss
	}
	rsi := 100 - 100/(1+rs)
	return math.Round(rsi*100) / 100
}
