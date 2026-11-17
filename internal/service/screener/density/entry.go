package density

import (
	"fmt"
	"math"
	"math/big"
	"strconv"
)

// EntryPrice returns the upper entry boundary in nanorubles, rounding the
// percentage offset down to whole ticks. A zero percentage uses the tick offset.
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
		ticks = whole.Int64()
	}
	if ticks > (math.MaxInt64-level)/step {
		return 0, fmt.Errorf("density entry price overflows")
	}
	return level + ticks*step, nil
}

// EntryLimit selects a buy limit from the current best ask. A positive percentage
// is an interest zone: wait outside it and buy at ask inside it. Zero preserves
// the passive tick-offset entry.
func EntryLimit(level, step, ticks int64, pct float64, ask int64) (int64, error) {
	boundary, err := EntryPrice(level, step, ticks, pct)
	if err != nil {
		return 0, err
	}
	if ask <= level || ask%step != 0 {
		return 0, fmt.Errorf("invalid entry ask")
	}
	if pct > 0 {
		if ask > boundary {
			return 0, fmt.Errorf("ask outside density interest zone")
		}
		return ask, nil
	}
	if boundary >= ask {
		return 0, fmt.Errorf("entry crosses ask")
	}
	return boundary, nil
}
