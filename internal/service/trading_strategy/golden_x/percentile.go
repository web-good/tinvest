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

// adaptiveThresholds computes the two buy-tier percentiles over an unordered
// slice of historical RSI values. pGreen and pYellow are percentile points
// in [0, 100] (typically 5 and 15). The input is not mutated; a sorted copy
// is taken internally.
func adaptiveThresholds(rsiSeries []float64, pGreen, pYellow float64) dto.Thresholds {
	sorted := append([]float64(nil), rsiSeries...)
	sort.Float64s(sorted)
	return dto.Thresholds{
		P5:  percentile(sorted, pGreen),
		P15: percentile(sorted, pYellow),
	}
}

// adaptiveSellThresholds computes the three sell-tier percentiles over an
// unordered slice of historical RSI values. pYellow/pOrange/pRed are
// percentile points in [0, 100] (typically 80/90/95). The input is not
// mutated; a sorted copy is taken internally.
func adaptiveSellThresholds(rsiSeries []float64, pYellow, pOrange, pRed float64) dto.SellThresholds {
	sorted := append([]float64(nil), rsiSeries...)
	sort.Float64s(sorted)
	return dto.SellThresholds{
		P80: percentile(sorted, pYellow),
		P90: percentile(sorted, pOrange),
		P95: percentile(sorted, pRed),
	}
}

// sellTierFromAdaptive maps the last closed-week RSI to a Golden X sell tier
// using the share's own historical upper percentiles. Comparisons are strict
// (`>`) — equality at a boundary falls into the looser tier. Behavior depends
// on kind:
//
//   - Dividend (Gold): three tiers — SellYellow at p80, SellOrange at p90,
//     SellRed at p95.
//   - Growth: single tier — SellOrange at p90. The sharp-exit semantics from
//     the original Golden X spec.
//   - Unknown: always tierNone (defensive default).
func sellTierFromAdaptive(rsi float64, st dto.SellThresholds, kind dto.StrategyKind) alertTier {
	switch kind {
	case dto.StrategyKindDividend:
		switch {
		case rsi > st.P95:
			return tierSellRed
		case rsi > st.P90:
			return tierSellOrange
		case rsi > st.P80:
			return tierSellYellow
		default:
			return tierNone
		}
	case dto.StrategyKindGrowth:
		if rsi > st.P90 {
			return tierSellOrange
		}
		return tierNone
	default:
		return tierNone
	}
}
