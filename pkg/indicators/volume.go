package indicators

// VolumeConfirmed reports whether the last value of volumes is strictly greater
// than multiplier × SMA of the preceding lookback values. Returns false when
// lookback <= 0 or len(volumes) < lookback+1 — the insufficient-history rule
// is silent (no error).
//
// Used by Golden X Stage C4 as a buy-side annotation: a true result says the
// last bar's volume is meaningfully above its recent baseline by the given
// multiplier ratio.
func VolumeConfirmed(volumes []int64, lookback int, multiplier float64) bool {
	if lookback <= 0 || len(volumes) < lookback+1 {
		return false
	}
	last := volumes[len(volumes)-1]
	window := volumes[len(volumes)-1-lookback : len(volumes)-1]
	var sum int64
	for _, v := range window {
		sum += v
	}
	sma := float64(sum) / float64(lookback)
	return float64(last) > multiplier*sma
}
