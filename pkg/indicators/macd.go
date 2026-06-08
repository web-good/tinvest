package indicators

// MACD returns the MACD line and signal line over closes (oldest-first, aligned
// to closes). macdLine = EMA(closes, fast) - EMA(closes, slow);
// signalLine = EMA(macdLine, signalPeriod).
//
// Returns nil, nil when any period is non-positive, when fast >= slow, or when
// there is not enough history (len(closes) < slow) — the insufficient-history
// rule is silent, mirroring ATR/VolumeConfirmed. The lines are only meaningful
// once there is ample history (callers size their lookback to slow+signalPeriod
// plus margin); early positions carry seeding bias and should not be inspected.
func MACD(closes []float64, fast, slow, signalPeriod int) (macdLine, signalLine []float64) {
	if fast <= 0 || slow <= 0 || signalPeriod <= 0 || fast >= slow || len(closes) < slow {
		return nil, nil
	}
	emaFast := ema(closes, fast)
	emaSlow := ema(closes, slow)
	macdLine = make([]float64, len(closes))
	for i := range closes {
		macdLine[i] = emaFast[i] - emaSlow[i]
	}
	signalLine = ema(macdLine, signalPeriod)
	return macdLine, signalLine
}

// ema is a sliding EMA seeded by the SMA of the first period values; positions
// before period-1 are 0. Local to keep pkg/indicators free of internal deps
// (mirrors internal/domain/ema.Compute).
func ema(values []float64, period int) []float64 {
	out := make([]float64, len(values))
	if period <= 0 || len(values) < period {
		return out
	}
	var sum float64
	for i := 0; i < period; i++ {
		sum += values[i]
	}
	out[period-1] = sum / float64(period)
	k := 2.0 / float64(period+1)
	for i := period; i < len(values); i++ {
		out[i] = (values[i]-out[i-1])*k + out[i-1]
	}
	return out
}
