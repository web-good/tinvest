package indicators

import (
	"math"
	"sort"
)

// Percentile returns the R-7 (linear-interpolation) percentile of an
// already-ascending-sorted slice. p is in [0, 100]. Empty input returns 0.
func Percentile(sortedAsc []float64, p float64) float64 {
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

// PercentileRank returns the fraction of values strictly less than x, in [0,1].
// Empty input returns 0. Input is not mutated.
func PercentileRank(values []float64, x float64) float64 {
	if len(values) == 0 {
		return 0
	}
	sorted := append([]float64(nil), values...)
	sort.Float64s(sorted)
	count := 0
	for _, v := range sorted {
		if v < x {
			count++
		}
	}
	return float64(count) / float64(len(sorted))
}
