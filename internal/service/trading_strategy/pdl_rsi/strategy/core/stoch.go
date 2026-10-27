package core

import "tinvest/pkg/indicators"

// stochZone is the oversold zone of the stochastic gate: %K on the signal hour must be under it.
const stochZone = 20

// stochOn reports whether the stochastic gate is on.
func (s *Strategy) stochOn() bool { return s.p.StochK > 0 && s.p.StochSmooth > 0 }

// stochK is the slow stochastic %K aligned to the bars of ser: the raw StochK-bar %K smoothed over
// StochSmooth bars. first is the first bar with a value; the values before it are zero. Nil when
// the bars are too few.
func (s *Strategy) stochK(ser series) (k []float64, first int) {
	_, slow := indicators.StochasticSeries(ser.h, ser.l, ser.c, s.p.StochK, s.p.StochSmooth)
	if slow == nil {
		return nil, 0
	}
	first = s.p.StochK - 1 + s.p.StochSmooth - 1
	k = make([]float64, len(ser.c))
	copy(k[first:], slow)
	return k, first
}

// stochD is %D at bar i, the StochD-bar average of %K; 0 while %D is not warm (kFirst is the first
// bar of k with a value) or StochD is not positive. It only goes into the entry reason.
func (s *Strategy) stochD(k []float64, kFirst, i int) float64 {
	if s.p.StochD <= 0 || i-s.p.StochD+1 < kFirst {
		return 0
	}
	var sum float64
	for m := i - s.p.StochD + 1; m <= i; m++ {
		sum += k[m]
	}
	return sum / float64(s.p.StochD)
}

// stochHit reports whether bar i passes the gate: %K under stochZone.
func stochHit(k []float64, i int) bool { return k[i] < stochZone }
