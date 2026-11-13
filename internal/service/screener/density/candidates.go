package density

import (
	"math"
	"sort"
	"time"
)

// Candidates tracks the active trade subscriptions for a single stream generation.
type Candidates struct {
	limit      int
	delay      time.Duration
	generation uint64
	selected   map[string]bool
	readySince map[string]time.Time
	lastSelect time.Time
	ids        []string
	trades     map[key]int64
}

func NewCandidates(limit int, delay time.Duration) *Candidates {
	return &Candidates{limit: limit, delay: delay, generation: 1, selected: make(map[string]bool), readySince: make(map[string]time.Time), trades: make(map[key]int64)}
}
func (c *Candidates) Select(signals []Signal, now time.Time) []string {
	if !c.lastSelect.IsZero() && now.Sub(c.lastSelect) < 15*time.Second {
		return append([]string(nil), c.ids...)
	}
	c.lastSelect = now
	best := make(map[string]Signal)
	for _, s := range signals {
		old, ok := best[s.InstrumentID]
		if !ok || better(s, old) {
			best[s.InstrumentID] = s
		}
	}
	list := make([]Signal, 0, len(best))
	for _, s := range best {
		list = append(list, s)
	}
	sort.Slice(list, func(i, j int) bool {
		a, b := list[i], list[j]
		if a.Multiple == b.Multiple && a.Rubles == b.Rubles && c.selected[a.InstrumentID] != c.selected[b.InstrumentID] {
			return c.selected[a.InstrumentID]
		}
		return better(a, b)
	})
	if len(list) > c.limit {
		list = list[:c.limit]
	}
	selected := make(map[string]bool, len(list))
	ids := make([]string, 0, len(list))
	for _, s := range list {
		selected[s.InstrumentID] = true
		ids = append(ids, s.InstrumentID)
	}
	for id := range c.readySince {
		if !selected[id] {
			delete(c.readySince, id)
			for k := range c.trades {
				if k.id == id {
					delete(c.trades, k)
				}
			}
		}
	}
	c.selected = selected
	c.ids = append([]string(nil), ids...)
	return ids
}
func better(a, b Signal) bool {
	if a.Multiple != b.Multiple {
		return a.Multiple > b.Multiple
	}
	if a.Rubles != b.Rubles {
		return a.Rubles > b.Rubles
	}
	if a.InstrumentID != b.InstrumentID {
		return a.InstrumentID < b.InstrumentID
	}
	return comparePrice(a.Price, b.Price) < 0
}
func (c *Candidates) Acknowledge(id string, generation uint64, now time.Time) {
	if generation == c.generation && c.selected[id] {
		if _, ok := c.readySince[id]; !ok {
			c.readySince[id] = now
		}
	}
}
func (c *Candidates) Ready(id string, now time.Time) bool {
	since, ok := c.readySince[id]
	return ok && now.Sub(since) >= c.delay
}
func (c *Candidates) Reconnect(generation uint64) {
	c.generation = generation
	clear(c.selected)
	clear(c.readySince)
	clear(c.trades)
	c.lastSelect = time.Time{}
	c.ids = nil
}

func (c *Candidates) RecordTrade(id string, price Price, quantity int64, generation uint64) {
	if generation != c.generation || !c.selected[id] || quantity <= 0 {
		return
	}
	if _, ok := c.readySince[id]; !ok {
		return
	}
	k := key{id, price}
	if c.trades[k] > math.MaxInt64-quantity {
		c.trades[k] = math.MaxInt64
	} else {
		c.trades[k] += quantity
	}
}
func (c *Candidates) Trades(id string, price Price, now time.Time) (int64, bool) {
	if !c.Ready(id, now) {
		return 0, false
	}
	return c.trades[key{id, price}], true
}
