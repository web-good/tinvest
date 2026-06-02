package universe

import "sort"

// Scored is one instrument with its volatility score (ATR%).
type Scored struct {
	InstrumentID   string
	InstrumentName string
	Ticker         string
	ATRPercent     float64
}

// TopN returns the n most volatile instruments, highest ATR% first.
// Ties are broken by InstrumentID ascending for deterministic output.
// The input slice is not mutated.
func TopN(scored []Scored, n int) []Scored {
	sorted := make([]Scored, len(scored))
	copy(sorted, scored)
	sort.Slice(sorted, func(i, j int) bool {
		if sorted[i].ATRPercent != sorted[j].ATRPercent {
			return sorted[i].ATRPercent > sorted[j].ATRPercent
		}
		return sorted[i].InstrumentID < sorted[j].InstrumentID
	})
	if n < len(sorted) {
		sorted = sorted[:n]
	}
	return sorted
}
