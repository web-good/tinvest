package percentile

import (
	"math"
	"sort"

	"tinvest/internal/service/trading_strategy/golden_x/model"
)

// r7 returns the R-7 (linear-interpolation) percentile of a sorted
// (ascending) slice. This is the default method in numpy.percentile and Excel.
// p is in [0, 100]. Empty input returns 0.
func r7(sortedAsc []float64, p float64) float64 {
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

// TierFromAdaptive maps the last closed-week RSI to a Golden X buy tier using
// the share's own historical percentiles. Comparisons are strict — equality
// at the boundary falls into the looser tier (Yellow at p5, None at p15).
func TierFromAdaptive(rsi, p5, p15 float64) model.AlertTier {
	switch {
	case rsi < p5:
		return model.TierGreen
	case rsi < p15:
		return model.TierYellow
	default:
		return model.TierNone
	}
}

// AdaptiveThresholds computes the two buy-tier percentiles over an unordered
// slice of historical RSI values. pGreen and pYellow are percentile points
// in [0, 100] (typically 5 and 15). The input is not mutated; a sorted copy
// is taken internally.
func AdaptiveThresholds(rsiSeries []float64, pGreen, pYellow float64) model.Thresholds {
	sorted := append([]float64(nil), rsiSeries...)
	sort.Float64s(sorted)
	return model.Thresholds{
		P5:  r7(sorted, pGreen),
		P15: r7(sorted, pYellow),
	}
}

// AdaptiveSellThresholds computes the three sell-tier percentiles over an
// unordered slice of historical RSI values. pYellow/pOrange/pRed are
// percentile points in [0, 100] (typically 80/90/95). The input is not
// mutated; a sorted copy is taken internally.
func AdaptiveSellThresholds(rsiSeries []float64, pYellow, pOrange, pRed float64) model.SellThresholds {
	sorted := append([]float64(nil), rsiSeries...)
	sort.Float64s(sorted)
	return model.SellThresholds{
		P80: r7(sorted, pYellow),
		P90: r7(sorted, pOrange),
		P95: r7(sorted, pRed),
	}
}

// SellTierFromAdaptive maps the last closed-week RSI to a Golden X sell tier
// using the share's own historical upper percentiles. Comparisons are strict
// (`>`) — equality at a boundary falls into the looser tier. Behavior depends
// on kind:
//
//   - Dividend (Gold): three tiers — SellYellow at p80, SellOrange at p90,
//     SellRed at p95.
//   - Growth: single tier — SellOrange at p90. The sharp-exit semantics from
//     the original Golden X spec.
//   - Unknown: always TierNone (defensive default).
func SellTierFromAdaptive(rsi float64, st model.SellThresholds, kind model.StrategyKind) model.AlertTier {
	switch kind {
	case model.StrategyKindDividend:
		switch {
		case rsi > st.P95:
			return model.TierSellRed
		case rsi > st.P90:
			return model.TierSellOrange
		case rsi > st.P80:
			return model.TierSellYellow
		default:
			return model.TierNone
		}
	case model.StrategyKindGrowth:
		if rsi > st.P90 {
			return model.TierSellOrange
		}
		return model.TierNone
	default:
		return model.TierNone
	}
}
