package golden_x

import (
	"math"
	"sort"

	"tinvest/internal/service/trading_strategy/golden_x/dto"
)

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

// tierFromAdaptive maps the last closed-week RSI to a Golden X buy tier using
// the share's own historical percentiles. Comparisons are strict — equality
// at the boundary falls into the looser tier (Yellow at p5, None at p15).
func tierFromAdaptive(rsi, p5, p15 float64) alertTier {
	switch {
	case rsi < p5:
		return tierGreen
	case rsi < p15:
		return tierYellow
	default:
		return tierNone
	}
}

// adaptiveThresholds computes P5 and P15 over an unordered slice of historical
// RSI values. The input is not mutated; a sorted copy is taken internally.
func adaptiveThresholds(rsiSeries []float64) dto.Thresholds {
	sorted := append([]float64(nil), rsiSeries...)
	sort.Float64s(sorted)
	return dto.Thresholds{
		P5:  percentile(sorted, 5),
		P15: percentile(sorted, 15),
	}
}

// adaptiveSellThresholds computes P80, P90 and P95 over an unordered slice of
// historical RSI values. The input is not mutated; a sorted copy is taken
// internally. Mirrors adaptiveThresholds but for the upper tail.
func adaptiveSellThresholds(rsiSeries []float64) dto.SellThresholds {
	sorted := append([]float64(nil), rsiSeries...)
	sort.Float64s(sorted)
	return dto.SellThresholds{
		P80: percentile(sorted, 80),
		P90: percentile(sorted, 90),
		P95: percentile(sorted, 95),
	}
}
