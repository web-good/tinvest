package backtest

// PullbackStats measures "buy-the-dip in an uptrend" opportunities on a single
// series. closes/ema/rsi must be equal-length, oldest-first, index-aligned
// (rsi is the RSI series, ema the trend EMA — e.g. EMA200). A pullback event is
// a bar i where the close is above its EMA (uptrend) and RSI crosses DOWN into
// the oversold zone (rsi[i-1] >= oversold && rsi[i] < oversold). An event is
// "recovered" if RSI exceeds recoverRSI within the next recoverBars bars.
//
// Warm-up sentinels (ema<=0 or rsi<=0) are skipped so leading-zero series entries
// cannot fake a cross. Events without a full recoverBars lookahead window are
// incomplete and excluded entirely. recoveryRate is recovered/events (0 when no
// events); eventFreq is events per 1000 bars over the whole series.
func PullbackStats(closes, ema, rsi []float64, oversold, recoverRSI float64, recoverBars int) (recoveryRate, eventFreq float64, events int) {
	n := len(closes)
	if n == 0 || len(ema) != n || len(rsi) != n || recoverBars <= 0 {
		return 0, 0, 0
	}
	recovered := 0
	for i := 1; i < n; i++ {
		if i+recoverBars >= n { // no full lookahead window left → all later events incomplete
			break
		}
		if ema[i] <= 0 || rsi[i] <= 0 || rsi[i-1] <= 0 { // warm-up sentinel
			continue
		}
		if closes[i] <= ema[i] { // not in an uptrend
			continue
		}
		if !(rsi[i-1] >= oversold && rsi[i] < oversold) { // not a fresh cross-down into oversold
			continue
		}
		events++
		for j := i + 1; j <= i+recoverBars; j++ {
			if rsi[j] > recoverRSI {
				recovered++
				break
			}
		}
	}
	if events == 0 {
		return 0, 0, 0
	}
	recoveryRate = float64(recovered) / float64(events)
	eventFreq = float64(events) / float64(n) * 1000
	return recoveryRate, eventFreq, events
}

// MeanDailyTurnoverM returns the mean daily turnover (in millions of currency
// units) from a series of candles, grouping by calendar date (UTC) of each
// candle's Time. Turnover per candle is Volume*lot*Close; per-day turnovers are
// summed, then averaged across distinct days. Returns 0 for an empty series.
func MeanDailyTurnoverM(candles []Candle, lot int32) float64 {
	if len(candles) == 0 {
		return 0
	}
	type dayKey struct{ y, m, d int }
	daySum := make(map[dayKey]float64)
	for _, c := range candles {
		k := dayKey{c.Time.Year(), int(c.Time.Month()), c.Time.Day()}
		daySum[k] += float64(c.Volume) * float64(lot) * c.Close
	}
	if len(daySum) == 0 {
		return 0
	}
	var total float64
	for _, v := range daySum {
		total += v
	}
	return total / float64(len(daySum)) / 1e6
}
