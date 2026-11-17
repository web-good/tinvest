package density

import "time"

// Relocation chooses the nearest qualifying alternative in the same snapshot.
// Candidates have passed detector filters, but need not have accumulated age.
// A move is a heuristic: aggregated books do not identify the order owner.
func Relocation(b Book, previous Price, baseLots int64, minRemaining float64, candidates []Signal, now time.Time) (Signal, bool) {
	var best Signal
	var nearest int64
	found := false
	for _, s := range candidates {
		if s.InstrumentID != b.InstrumentID || s.Price == previous || !valid(s.Price) || s.Price.Units > 100000000 || previous.Units > 100000000 || (!s.ProfileUntil.IsZero() && !now.Before(s.ProfileUntil)) {
			continue
		}
		var qty int64
		for _, l := range b.Bids {
			if l.Price == s.Price {
				qty = l.Quantity
				break
			}
		}
		if qty <= 0 || float64(qty) < float64(baseLots)*minRemaining {
			continue
		}
		delta := (s.Price.Units-previous.Units)*1e9 + int64(s.Price.Nano) - int64(previous.Nano)
		if delta < 0 {
			delta = -delta
		}
		if !found || delta < nearest || delta == nearest && comparePrice(s.Price, best.Price) > 0 {
			s.Quantity = qty
			best, nearest, found = s, delta, true
		}
	}
	return best, found
}
