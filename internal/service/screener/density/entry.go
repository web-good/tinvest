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
