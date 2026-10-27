package core

import (
	"fmt"
	"sort"
	"time"
)

// indexSlopeDays is how many index dailies back the EMA slope is measured over: the index trend
// is down when today's EMA is below the EMA indexSlopeDays completed dailies earlier.
const indexSlopeDays = 5

// index is the daily closes of the market index (IMOEX) the strategy was built with, oldest-first,
// weekday sessions only. The engine hands a strategy its own dailies only, so the index series
// comes in at construction; indexTrend reads only the dailies of dates before the current bar's,
// the same no-lookahead rule the engine applies to the ticker's dailies.
type index struct {
	days   []int // MSK date keys (yyyymmdd), ascending
	closes []float64
}

// WithIndex sets the market index dailies (open-times and closes, oldest-first) the IndexTrendEMA
// filter reads; weekend sessions are dropped. Misaligned slices leave the strategy without an
// index, and the filter then blocks every entry.
func (s *Strategy) WithIndex(times []time.Time, closes []float64) *Strategy {
	s.idx = index{}
	if len(times) != len(closes) {
		return s
	}
	for i, t := range times {
		if isWeekend(t.In(mskLoc)) {
			continue
		}
		s.idx.days = append(s.idx.days, dayKey(t))
		s.idx.closes = append(s.idx.closes, closes[i])
	}
	return s
}

func (s *Strategy) indexOn() bool { return s.p.IndexTrendEMA > 0 }

// indexTrend is the EMA(IndexTrendEMA) of the index closes completed before the date of t, now
// and indexSlopeDays dailies earlier; ok is false when there are fewer than IndexTrendEMA +
// indexSlopeDays such dailies.
func (s *Strategy) indexTrend(t time.Time) (now, before float64, ok bool) {
	n := sort.SearchInts(s.idx.days, dayKey(t))
	period := s.p.IndexTrendEMA
	if period <= 0 || n < period+indexSlopeDays {
		return 0, 0, false
	}
	e := emaSeries(s.idx.closes[:n], period)
	return e[n-1], e[n-1-indexSlopeDays], true
}

// indexOK reports whether the index trend lets the bar at t enter: off, or the index EMA falling
// over indexSlopeDays dailies. The reason is for EntryReason and Explain.
func (s *Strategy) indexOK(t time.Time) (string, bool) {
	if !s.indexOn() {
		return "", true
	}
	now, before, ok := s.indexTrend(t)
	if !ok {
		return fmt.Sprintf("EMA(%d) индекса не посчитана: мало дневок индекса", s.p.IndexTrendEMA), false
	}
	return fmt.Sprintf("EMA(%d) индекса %.2f %s %.2f %d дн. назад", s.p.IndexTrendEMA, now, cmpSign(now, before), before, indexSlopeDays),
		now < before
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
