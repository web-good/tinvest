package density

import (
	"fmt"
	"math"
)

// EntryLimit selects a buy limit using the instrument's price step.
// A positive zone buys at the best ask within that many ticks above the density.
// With a zero zone, the tick offset places a passive limit below the best ask.
func EntryLimit(level, step, ticks, zoneTicks, ask int64) (int64, error) {
	if level <= 0 || step <= 0 || level%step != 0 || ticks < 1 || zoneTicks < 0 {
		return 0, fmt.Errorf("invalid density entry offset")
	}
	if zoneTicks > 0 {
		ticks = zoneTicks
	}
	if ticks > (math.MaxInt64-level)/step {
		return 0, fmt.Errorf("density entry price overflows")
	}
	boundary := level + ticks*step
	if ask <= level || ask%step != 0 {
		return 0, fmt.Errorf("invalid entry ask")
	}
	if zoneTicks > 0 {
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

// RetainsEntryDensity checks the wall against its size when the entry was planned.
func RetainsEntryDensity(initial, current float64) bool {
	return initial > 0 && current >= math.Nextafter(initial*0.4, 0)
}

// FixedEntryLimit quotes one tick above the wall.
func FixedEntryLimit(level, step int64) int64 {
	if level <= 0 || step <= 0 || level > math.MaxInt64-step {
		return 0
	}
	return level + step
}

// EntryStop rounds 0.2% below the fill down to an exchange tick.
func EntryStop(entry, step int64) int64 {
	return ((entry - entry/500 - min(entry%500, 1)) / step) * step
}

// TakeProfit rounds the reward target up to an exchange tick.
func TakeProfit(entry, stop, step int64, rewardRisk float64) (int64, bool) {
	if stop <= 0 || entry <= stop || step <= 0 {
		return 0, false
	}
	base := entry - entry%step
	ticks := math.Ceil(float64(entry%step)/float64(step) + float64(entry-stop)/float64(step)*rewardRisk)
	if !(ticks >= 1 && ticks < float64(math.MaxInt64)) || int64(ticks) > (math.MaxInt64-base)/step {
		return 0, false
	}
	return base + int64(ticks)*step, true
}
