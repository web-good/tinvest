package density

import (
	"fmt"
	"math"
	"math/big"
	"strconv"
)

// EntryPrice returns nanorubles, rounding the percentage offset down to whole
// ticks (at least one). A zero percentage uses the explicit tick offset.
func EntryPrice(level, step, ticks int64, pct float64) (int64, error) {
	if level <= 0 || step <= 0 || level%step != 0 || ticks < 1 || math.IsNaN(pct) || math.IsInf(pct, 0) || pct < 0 || pct > 100 {
		return 0, fmt.Errorf("invalid density entry offset")
	}
	if pct > 0 {
		// Parse the configured decimal exactly so a whole tick is not lost to
		// binary floating point at boundaries such as 100 RUB + 0.3%.
		ratio, ok := new(big.Rat).SetString(strconv.FormatFloat(pct, 'f', -1, 64))
		if !ok {
			return 0, fmt.Errorf("invalid density entry percentage")
		}
		ratio.Mul(ratio, new(big.Rat).SetInt64(level/step))
		ratio.Quo(ratio, big.NewRat(100, 1))
		whole := new(big.Int).Quo(ratio.Num(), ratio.Denom())
		if !whole.IsInt64() {
			return 0, fmt.Errorf("density entry offset overflows")
		}
		ticks = max(1, whole.Int64())
	}
	if ticks > (math.MaxInt64-level)/step {
		return 0, fmt.Errorf("density entry price overflows")
	}
	return level + ticks*step, nil
}
