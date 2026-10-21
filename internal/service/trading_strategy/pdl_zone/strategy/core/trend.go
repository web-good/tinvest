package core

import (
	"fmt"

	"tinvest/internal/service/trading_strategy/scalping/strategy"
)

// trendOK reports whether the higher-timeframe trends let the current bar enter, and the reasons:
//   - H1TrendEMA > 0: the close of the last completed hour (md.HTFCloses, the engine's Hour1
//     series) above the EMA(H1TrendEMA) of the completed hourly closes;
//   - D1TrendEMA > 0: the close of the last completed weekday daily above the EMA(D1TrendEMA) of
//     the completed weekday daily closes.
//
// Fewer bars than the period block the entry; both off — always passes.
func (s *Strategy) trendOK(md strategy.MarketData) (string, bool) {
	why := ""
	if s.p.H1TrendEMA > 0 {
		w, ok := aboveEMA("часа", md.HTFCloses, s.p.H1TrendEMA)
		if !ok {
			return w, false
		}
		why = w
	}
	if s.p.D1TrendEMA > 0 {
		_, _, c := weekdayDaily(md.DailyHighs, md.DailyLows, md.DailyCloses, md.DailyTimes)
		w, ok := aboveEMA("дня", c, s.p.D1TrendEMA)
		if !ok {
			return w, false
		}
		if why != "" {
			why += ", "
		}
		why += w
	}
	return why, true
}

// aboveEMA reports whether the last of closes is above their EMA(period), with the reason; tf
// names the timeframe in it.
func aboveEMA(tf string, closes []float64, period int) (string, bool) {
	n := len(closes)
	if n < period {
		return fmt.Sprintf("EMA(%d) %s не посчитана: свечей %d", period, tf, n), false
	}
	e := emaSeries(closes, period)[n-1]
	return fmt.Sprintf("close %s %.4f %s EMA(%d) %s %.4f", tf, closes[n-1], cmpSign(closes[n-1], e), period, tf, e), closes[n-1] > e
}

// cmpSign is the relation of a to b as a sign for the reasons.
func cmpSign(a, b float64) string {
	switch {
	case a < b:
		return "<"
	case a > b:
		return ">"
	}
	return "="
}

// emaSeries is the EMA of v with the given period, seeded with the SMA of the first period values;
// the elements before the seed hold the running SMA.
func emaSeries(v []float64, period int) []float64 {
	out := make([]float64, len(v))
	k := 2 / float64(period+1)
	sum := 0.0
	for i, x := range v {
		if i < period {
			sum += x
			out[i] = sum / float64(i+1)
			continue
		}
		out[i] = x*k + out[i-1]*(1-k)
	}
	return out
}
