package golden_x

import "math"

// percentile returns the R-7 (linear-interpolation) percentile of a sorted
// (ascending) slice. This is the default method in numpy.percentile and Excel.
// p is in [0, 100]. Empty input returns 0.
func percentile(sortedAsc []float64, p float64) float64 {
	n := len(sortedAsc)
	if n == 0 {
		return 0
	}
	if n == 1 {
		return sortedAsc[0]
	}
	rank := (p / 100.0) * float64(n-1)
	lo := int(math.Floor(rank))
	hi := int(math.Ceil(rank))
	if lo == hi {
		return sortedAsc[lo]
	}
	weight := rank - float64(lo)
	return sortedAsc[lo]*(1-weight) + sortedAsc[hi]*weight
}
