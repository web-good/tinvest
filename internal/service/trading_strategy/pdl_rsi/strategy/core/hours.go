package core

import (
	"time"

	"tinvest/internal/service/trading_strategy/scalping/strategy"
)

// hourBar is one hour aggregated from the strategy's bars.
type hourBar struct {
	start time.Time // the hour's start
	close float64   // the close of its last bar
	first int       // index of its first bar in the window
	last  int       // index of its last bar in the window
	done  int       // index of the bar at whose close the hour is known complete
}

// hourBars aggregates the window into completed hours, oldest-first. An hour is complete at the
// close of the bar that ends it (start + step reaches the next hour), or, when that bar is missing,
// at the first bar of a later hour. The still-forming hour is left out.
func (s *Strategy) hourBars(md strategy.MarketData) []hourBar {
	n := len(md.Closes)
	out := make([]hourBar, 0, n/4+1)
	var cur *hourBar
	for k := range n {
		h := md.Times[k].Truncate(time.Hour)
		if cur != nil && !h.Equal(cur.start) {
			cur.done = k // the hour lacked its closing bar: known complete at this later bar
			out = append(out, *cur)
			cur = nil
		}
		if cur == nil {
			cur = &hourBar{start: h, first: k}
		}
		cur.close, cur.last = md.Closes[k], k
		if !md.Times[k].Add(s.step).Before(h.Add(time.Hour)) {
			cur.done = k
			out = append(out, *cur)
			cur = nil
		}
	}
	return out
}

// hourCloses is the closes of the hours, oldest-first.
func hourCloses(hs []hourBar) []float64 {
	out := make([]float64, len(hs))
	for k, h := range hs {
		out[k] = h.close
	}
	return out
}

// crossUp reports whether a series moved from below level to at or above it.
func crossUp(prev, cur, level float64) bool { return prev < level && cur >= level }
