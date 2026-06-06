package indicators

// VolumeProfile holds the result of a horizontal volume analysis over a window
// of price bars. It partitions the price range into equal-width bins and
// attributes each candle's volume proportionally across the bins its [low,high]
// range overlaps.
type VolumeProfile struct {
	BinCenter []float64 // center price of each bin, ascending (low price → high price)
	BinVol    []float64 // volume attributed to each bin, aligned with BinCenter
	POC       float64   // price (bin center) of the bin holding the most volume
	VAHigh    float64   // center price of the highest bin in the value area
	VALow     float64   // center price of the lowest bin in the value area
}

// ComputeVolumeProfile builds a volume profile from a window of OHLCV bars.
//
// Price range: [min(lows), max(highs)] divided into `bins` equal-width price
// bins. BinCenter[i] is the midpoint of bin i, ascending.
//
// Volume attribution: each candle spreads its volume uniformly across the bins
// its [low, high] range overlaps, weighted by the fraction of overlap. A candle
// with low == high (zero range) places its full volume into the single bin that
// contains that price.
//
// POC (Point of Control): bin center with the highest accumulated volume.
//
// Value area: starting from the POC bin, greedily add whichever neighbouring
// bin (above or below) has more volume, expanding outward until accumulated
// volume reaches >= 70% of total volume. On a tie between the above and below
// neighbours, the above bin is chosen (standard Market Profile convention).
// VAHigh and VALow are the highest and lowest bin centers included.
//
// Returns a zero-value VolumeProfile (empty slices, zero scalars) for any
// invalid or degenerate input: bins <= 0, empty slices, mismatched slice
// lengths, no price range (max(highs) <= min(lows)), zero total volume, or
// any negative volume entry (treated as invalid input).
func ComputeVolumeProfile(highs, lows []float64, volumes []int64, bins int) VolumeProfile {
	// Validate inputs.
	if bins <= 0 {
		return VolumeProfile{}
	}
	n := len(highs)
	if n == 0 || len(lows) != n || len(volumes) != n {
		return VolumeProfile{}
	}

	// Find price range.
	priceMin := lows[0]
	priceMax := highs[0]
	for i := 1; i < n; i++ {
		if lows[i] < priceMin {
			priceMin = lows[i]
		}
		if highs[i] > priceMax {
			priceMax = highs[i]
		}
	}
	if priceMax <= priceMin {
		return VolumeProfile{}
	}

	// Check total volume; reject negative entries.
	var totalVol int64
	for _, v := range volumes {
		if v < 0 {
			return VolumeProfile{}
		}
		totalVol += v
	}
	if totalVol == 0 {
		return VolumeProfile{}
	}

	// Build bin centers and accumulate volume.
	binWidth := (priceMax - priceMin) / float64(bins)
	centers := make([]float64, bins)
	binVol := make([]float64, bins)
	for i := 0; i < bins; i++ {
		centers[i] = priceMin + (float64(i)+0.5)*binWidth
	}

	for i := 0; i < n; i++ {
		lo := lows[i]
		hi := highs[i]
		vol := float64(volumes[i])

		if lo == hi {
			// Zero-range candle: dump all volume into the single containing bin.
			b := priceToBin(lo, priceMin, binWidth, bins)
			binVol[b] += vol
			continue
		}

		candleRange := hi - lo
		for b := 0; b < bins; b++ {
			binLo := priceMin + float64(b)*binWidth
			binHi := priceMin + float64(b+1)*binWidth

			overlap := overlapLen(lo, hi, binLo, binHi)
			if overlap <= 0 {
				continue
			}
			fraction := overlap / candleRange
			binVol[b] += vol * fraction
		}
	}

	// Find POC: first bin with maximum volume.
	pocIdx := 0
	for b := 1; b < bins; b++ {
		if binVol[b] > binVol[pocIdx] {
			pocIdx = b
		}
	}
	poc := centers[pocIdx]

	// Value area: expand from POC until accumulated >= 70% of total.
	threshold := float64(totalVol) * 0.70
	accumulated := binVol[pocIdx]
	loIdx := pocIdx
	hiIdx := pocIdx

	for accumulated < threshold {
		canExpandUp := hiIdx+1 < bins
		canExpandDown := loIdx-1 >= 0

		if !canExpandUp && !canExpandDown {
			break
		}

		var upVol, downVol float64
		if canExpandUp {
			upVol = binVol[hiIdx+1]
		}
		if canExpandDown {
			downVol = binVol[loIdx-1]
		}

		// Add above when tied (>= rather than >) — standard Market Profile convention.
		if canExpandUp && (!canExpandDown || upVol >= downVol) {
			hiIdx++
			accumulated += upVol
		} else {
			loIdx--
			accumulated += downVol
		}
	}

	return VolumeProfile{
		BinCenter: centers,
		BinVol:    binVol,
		POC:       poc,
		VAHigh:    centers[hiIdx],
		VALow:     centers[loIdx],
	}
}

// priceToBin returns the bin index for a given price. Clamps to [0, bins-1].
func priceToBin(price, priceMin, binWidth float64, bins int) int {
	b := int((price - priceMin) / binWidth)
	if b < 0 {
		return 0
	}
	if b >= bins {
		return bins - 1
	}
	return b
}

// overlapLen returns the length of the intersection of two closed/half-open
// intervals [aLo, aHi) and [bLo, bHi). Returns 0 when there is no overlap.
// The last bin is effectively closed at its upper edge (priceMax).
func overlapLen(aLo, aHi, bLo, bHi float64) float64 {
	lo := aLo
	if bLo > lo {
		lo = bLo
	}
	hi := aHi
	if bHi < hi {
		hi = bHi
	}
	if hi <= lo {
		return 0
	}
	return hi - lo
}
