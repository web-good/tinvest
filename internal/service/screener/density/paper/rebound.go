package paper

import (
	"time"
	"tinvest/internal/service/screener/density"
)

// A watch belongs to one continuously observed wall at one price.
type reboundWatch struct {
	signal                             density.Signal
	armed, touched, rebound, lastTrade time.Time
	blocked                            bool
}

func (e *Engine) watchRebound(s density.Signal, now time.Time) {
	id := s.InstrumentID
	if e.entryHoursBlocked(now) || !e.fresh(id, now) || now.Before(e.cooldown[id]) {
		return
	}
	if o := e.orders[id]; o != nil && active(o) {
		return
	}
	watches := e.reboundWatches[id]
	if watches == nil {
		watches = make(map[int64]*reboundWatch)
		e.reboundWatches[id] = watches
	}
	level := price(s.Price)
	if old := watches[level]; old != nil && old.signal.Since.Equal(s.Since) {
		return
	}
	watches[level] = &reboundWatch{signal: s, armed: e.books[id].book.Time}
}

func (e *Engine) trackReboundTrade(t density.Event, now time.Time) {
	id := t.InstrumentID
	if len(e.reboundWatches[id]) == 0 {
		return
	}
	if !e.fresh(id, now) || e.entryHoursBlocked(now) {
		delete(e.reboundWatches, id)
		return
	}
	p := price(t.Price)
	for level, w := range e.reboundWatches[id] {
		if w.blocked {
			continue
		}
		if t.Time.IsZero() || t.Time.After(now) || t.Quantity <= 0 || p <= 0 || p%price(e.instruments[id].PriceStep) != 0 {
			w.blocked = true
			continue
		}
		if t.Time.Before(w.armed) {
			continue
		}
		// Even a delayed or same-timestamp print below support invalidates this episode.
		if p < level {
			w.blocked = true
			continue
		}
		if now.Sub(t.Time) > e.cfg.Freshness || t.Time.Before(w.lastTrade) {
			continue
		}
		if p == level {
			w.touched = t.Time
			w.rebound = time.Time{}
		} else if !w.touched.IsZero() && t.Time.After(w.touched) {
			w.rebound = t.Time
		}
		w.lastTrade = t.Time
	}
}

func (e *Engine) checkReboundBook(b density.Book, now time.Time) {
	watches := e.reboundWatches[b.InstrumentID]
	if len(watches) == 0 {
		return
	}
	if e.entryHoursBlocked(now) {
		delete(e.reboundWatches, b.InstrumentID)
		return
	}
	// Prefer the highest qualifying bid if several levels rebound together.
	var candidate *reboundWatch
	for level, w := range watches {
		var lots int64
		for _, bid := range b.Bids {
			if price(bid.Price) == level {
				lots = bid.Quantity
				break
			}
		}
		if lots == 0 {
			delete(watches, level)
			continue
		}
		if w.blocked {
			continue
		}
		if !density.RetainsEntryDensity(float64(w.signal.Quantity), float64(lots)) {
			w.blocked = true
			continue
		}
		if !w.rebound.IsZero() && !b.Time.Before(w.rebound) && now.Sub(w.rebound) <= e.cfg.Freshness &&
			(candidate == nil || level > price(candidate.signal.Price)) {
			candidate = w
		}
	}
	if candidate != nil {
		candidate.blocked = true
		if e.observe(candidate.signal, now, time.Time{}) {
			for _, w := range watches {
				w.blocked = true
			}
		}
	}
}
