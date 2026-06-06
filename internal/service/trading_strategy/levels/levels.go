package levels

import "tinvest/pkg/indicators"

// Level is a strong price level extracted from a volume profile.
type Level struct {
	Price    float64 // volume-weighted center price of the HVN cluster
	Strength float64 // cluster volume as a fraction of total profile volume (0..1)
}

// ExtractLevels returns the strong levels (high-volume-node clusters) of vp,
// ordered ascending by Price. A bin is a high-volume node when its volume is
// >= hvnFactor * mean(BinVol). Runs of adjacent HVN bins are merged into one
// level whose Price is the volume-weighted average of the member bin centers
// and whose Strength is the cluster's volume divided by the profile's total volume.
// Returns nil for a degenerate profile (no bins or all-zero volumes) or non-positive hvnFactor.
func ExtractLevels(vp indicators.VolumeProfile, hvnFactor float64) []Level {
	n := len(vp.BinCenter)
	if n == 0 || hvnFactor <= 0 {
		return nil
	}

	// Compute mean bin volume and total volume.
	var totalVol float64
	for _, v := range vp.BinVol {
		totalVol += v
	}
	if totalVol == 0 {
		return nil
	}
	mean := totalVol / float64(n)
	threshold := hvnFactor * mean

	// Scan for maximal runs of HVN bins and merge each run into a level.
	var result []Level
	i := 0
	for i < n {
		if vp.BinVol[i] < threshold {
			i++
			continue
		}
		// Start of a HVN run.
		var clusterVol, weightedSum float64
		for i < n && vp.BinVol[i] >= threshold {
			clusterVol += vp.BinVol[i]
			weightedSum += vp.BinCenter[i] * vp.BinVol[i]
			i++
		}
		price := weightedSum / clusterVol
		result = append(result, Level{
			Price:    price,
			Strength: clusterVol / totalVol,
		})
	}

	return result
}

// NearestSupportBelow returns the highest level strictly below price.
// ok is false when no such level exists. Assumes levels is ascending by Price
// (as ExtractLevels returns).
func NearestSupportBelow(levels []Level, price float64) (Level, bool) {
	// Walk from the highest level downward to find the first one strictly below price.
	for i := len(levels) - 1; i >= 0; i-- {
		if levels[i].Price < price {
			return levels[i], true
		}
	}
	return Level{}, false
}

// NearestResistanceAbove returns the lowest level strictly above price.
// ok is false when no such level exists. Assumes levels is ascending by Price.
func NearestResistanceAbove(levels []Level, price float64) (Level, bool) {
	// Walk from the lowest level upward to find the first one strictly above price.
	for i := 0; i < len(levels); i++ {
		if levels[i].Price > price {
			return levels[i], true
		}
	}
	return Level{}, false
}
