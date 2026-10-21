package core

import (
	"fmt"

	"tinvest/internal/service/trading_strategy/scalping/strategy"
)

func (s *Strategy) trendOn() bool { return s.p.H1TrendEMA > 0 }

// trendOK reports whether the ticker's hourly trend lets the current bar enter: off, or the close
// of the last completed hour (md.HTFCloses, the engine's Hour1 series) above the EMA(H1TrendEMA)
// of the completed hourly closes. Fewer than H1TrendEMA hours block the entry. The reason is for
// EntryReason and Explain.
func (s *Strategy) trendOK(md strategy.MarketData) (string, bool) {
	if !s.trendOn() {
		return "", true
	}
	c, period := md.HTFCloses, s.p.H1TrendEMA
	n := len(c)
	if n < period {
		return fmt.Sprintf("EMA(%d) часа не посчитана: часовых свечей %d", period, n), false
	}
	e := emaSeries(c, period)[n-1]
	return fmt.Sprintf("close часа %.4f %s EMA(%d) часа %.4f", c[n-1], cmpSign(c[n-1], e), period, e), c[n-1] > e
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
