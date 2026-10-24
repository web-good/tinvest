package core

import (
	"fmt"

	"tinvest/pkg/indicators"
)

// stochKD is the slow stochastic aligned to the bars of ser: %K is the raw StochK-bar %K smoothed
// over StochSmooth bars, %D the StochD-bar average of %K. first is the first bar with both %K and
// %D; the values before it are zero. All nil when a period is not positive or the bars are too few.
func (s *Strategy) stochKD(ser series) (k, d []float64, first int) {
	if s.p.StochK <= 0 || s.p.StochSmooth <= 0 || s.p.StochD <= 0 {
		return nil, nil, 0
	}
	_, slow := indicators.StochasticSeries(ser.h, ser.l, ser.c, s.p.StochK, s.p.StochSmooth)
	if len(slow) < s.p.StochD {
		return nil, nil, 0
	}
	n := len(ser.c)
	off := s.p.StochK - 1 + s.p.StochSmooth - 1
	k = make([]float64, n)
	copy(k[off:], slow)
	d = make([]float64, n)
	first = off + s.p.StochD - 1
	for i := first; i < n; i++ {
		var sum float64
		for m := i - s.p.StochD + 1; m <= i; m++ {
			sum += k[m]
		}
		d[i] = sum / float64(s.p.StochD)
	}
	return k, d, first
}

// crossOver reports whether %K crossed %D upward on bar i: at or under it on i−1, above it on i.
func crossOver(k, d []float64, i int) bool { return k[i-1] <= d[i-1] && k[i] > d[i] }

// crossUnder reports whether %K crossed %D downward on bar i: at or over it on i−1, under it on i.
func crossUnder(k, d []float64, i int) bool { return k[i-1] >= d[i-1] && k[i] < d[i] }

// stochTrigger reports whether bar i is a trigger cross: %K crossed %D upward with %K below
// StochCrossMax (any level when it is 0).
func (s *Strategy) stochTrigger(k, d []float64, i int) bool {
	return crossOver(k, d, i) && (s.p.StochCrossMax <= 0 || k[i] < s.p.StochCrossMax)
}

// stochWhy describes the StochCrossMax rule for the entry reason.
func (s *Strategy) stochWhy() string {
	if s.p.StochCrossMax <= 0 {
		return ""
	}
	return fmt.Sprintf(" ниже %.0f", s.p.StochCrossMax)
}
