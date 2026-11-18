package live

import (
	"fmt"
	"sync"
	"time"
	"tinvest/internal/service/screener/density"
)

// Prepare atomically revalidates the uninterrupted detector observation and
// installs the loss watcher after potentially slow account/allocation RPCs.
func (f *Feed) Prepare(i density.Instrument, original density.Signal, now time.Time) (*Episode, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if !f.ready[i.ID] {
		return nil, fmt.Errorf("market no longer ready")
	}
	for _, s := range f.signals[i.ID] {
		if s.Price != original.Price || !s.Since.Equal(original.Since) {
			continue
		}
		ep, err := Plan(f.cfg, i, s, f.books[i.ID], now)
		if err != nil {
			return nil, err
		}
		copy := *ep
		f.selected = i.ID
		f.watches[i.ID] = &copy
		f.anchors[i.ID] = s
		delete(f.reasons, i.ID)
		f.check(now)
		if f.reasons[i.ID] != "" {
			delete(f.watches, i.ID)
			return nil, fmt.Errorf("density no longer valid")
		}
		return ep, nil
	}
	return nil, fmt.Errorf("density confirmation expired during allocation check")
}

type Feed struct {
	mu         sync.Mutex
	cfg        Config
	books      map[string]density.Book
	signals    map[string][]density.Signal
	candidates map[string][]density.Signal
	anchors    map[string]density.Signal
	ready      map[string]bool
	watermarks map[string]time.Time
	trades     map[string]time.Time
	watches    map[string]*Episode
	reasons    map[string]string
	selected   string
}
type Candidate struct {
	Signal density.Signal
	Book   density.Book
}

// ClearSignals resets entry confirmations without interrupting position watchers.
func (f *Feed) ClearSignals() {
	f.mu.Lock()
	defer f.mu.Unlock()
	clear(f.signals)
	clear(f.candidates)
}

func NewFeed(c Config) *Feed {
	return &Feed{cfg: c, books: map[string]density.Book{}, signals: map[string][]density.Signal{}, candidates: map[string][]density.Signal{}, anchors: map[string]density.Signal{}, ready: map[string]bool{}, watermarks: map[string]time.Time{}, trades: map[string]time.Time{}, watches: map[string]*Episode{}, reasons: map[string]string{}}
}

func (f *Feed) Trade(event density.Event, now time.Time) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if event.Time.IsZero() || event.Time.After(now) || now.Sub(event.Time) > f.cfg.Freshness || !event.Time.After(f.trades[event.InstrumentID]) || Price(event.Price) <= 0 || event.Quantity <= 0 {
		return
	}
	f.trades[event.InstrumentID] = event.Time
	if ep := f.watches[event.InstrumentID]; ep != nil && (ep.Entry.Filled > 0 || Price(f.anchors[event.InstrumentID].Price) == ep.Density) && f.reasons[event.InstrumentID] == "" && Price(event.Price) <= ep.StopPrice {
		f.reasons[event.InstrumentID] = "stop_trade"
	}
}
func (f *Feed) Watch(ep *Episode, now time.Time) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if ep == nil {
		delete(f.watches, f.selected)
		delete(f.reasons, f.selected)
		return
	}
	f.selected = ep.Instrument.ID
	if f.watches[f.selected] == nil {
		copy := *ep
		f.watches[f.selected] = &copy
		f.anchors[f.selected] = density.Signal{InstrumentID: f.selected, Price: priceFromNano(ep.Density), Quantity: ep.DensityLots}
		delete(f.reasons, f.selected)
	}
	f.check(now)
}
func (f *Feed) Select(id string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.selected = id
}
func (f *Feed) Unwatch(id string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	delete(f.watches, id)
	delete(f.anchors, id)
	delete(f.reasons, id)
}
func (f *Feed) SetReady(id string, ready bool) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.ready[id] = ready
	if !ready {
		delete(f.signals, id)
		delete(f.candidates, id)
		if f.watches[id] != nil && f.reasons[id] == "" {
			f.reasons[id] = "data_unavailable"
		}
	}
}
func (f *Feed) Gap(now time.Time) {
	f.mu.Lock()
	defer f.mu.Unlock()
	for id := range f.ready {
		f.ready[id] = false
		f.watermarks[id] = now
	}
	clear(f.signals)
	clear(f.candidates)
	for id := range f.watches {
		if f.reasons[id] == "" {
			f.reasons[id] = "data_gap"
		}
	}
}
func (f *Feed) Book(b density.Book, signals []density.Signal, now time.Time) {
	f.BookWithCandidates(b, signals, signals, now)
}

func (f *Feed) BookWithCandidates(b density.Book, signals, candidates []density.Signal, now time.Time) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if !b.Time.After(f.watermarks[b.InstrumentID]) {
		return
	}
	if !validBook(b, now, f.cfg.Freshness) {
		delete(f.signals, b.InstrumentID)
		delete(f.candidates, b.InstrumentID)
		if f.watches[b.InstrumentID] != nil && f.reasons[b.InstrumentID] == "" {
			f.reasons[b.InstrumentID] = "invalid_book"
		}
		return
	}
	// Preserve a stale interval even when data resumes during a blocking broker RPC.
	f.check(now)
	b.Bids = append([]density.Level(nil), b.Bids...)
	b.Asks = append([]density.Level(nil), b.Asks...)
	f.books[b.InstrumentID] = b
	f.watermarks[b.InstrumentID] = b.Time
	if f.ready[b.InstrumentID] {
		f.signals[b.InstrumentID] = append([]density.Signal(nil), signals...)
		f.candidates[b.InstrumentID] = append([]density.Signal(nil), candidates...)
	} else {
		delete(f.signals, b.InstrumentID)
		delete(f.candidates, b.InstrumentID)
	}
	f.check(now)
}
func (f *Feed) check(now time.Time) {
	for id, ep := range f.watches {
		if f.reasons[id] != "" {
			continue
		}
		b, ok := f.books[id]
		if !ok || !f.ready[id] || !validBook(b, now, f.cfg.Freshness) {
			f.reasons[id] = "data_unavailable"
			continue
		}
		var qty int64
		anchor := f.anchors[id]
		for _, l := range b.Bids {
			if l.Price == anchor.Price {
				qty = l.Quantity
				break
			}
		}
		if float64(qty) < float64(ep.DensityLots)*f.cfg.MinRemaining {
			moved, ok := density.Relocation(b, anchor.Price, ep.DensityLots, f.cfg.MinRemaining, f.candidates[id], now)
			if !ok {
				f.reasons[id] = "density_lost"
				continue
			}
			f.anchors[id] = moved
		}
		if (ep.Entry.Filled > 0 || Price(f.anchors[id].Price) == ep.Density) && Price(b.Bids[0].Price) <= ep.StopPrice {
			f.reasons[id] = "stop_price"
		}
	}
}
func (f *Feed) Reason(now time.Time) string {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.check(now)
	return f.reasons[f.selected]
}
func (f *Feed) exitBid(id string, now time.Time) int64 {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.check(now)
	b := f.books[id]
	if !f.ready[id] || f.reasons[id] != "" || !validBook(b, now, f.cfg.Freshness) {
		return 0
	}
	return Price(b.Bids[0].Price)
}

func (f *Feed) Candidates(now time.Time) []Candidate {
	f.mu.Lock()
	defer f.mu.Unlock()
	var result []Candidate
	for id, signals := range f.signals {
		b := f.books[id]
		if f.ready[id] && validBook(b, now, f.cfg.Freshness) {
			for _, s := range signals {
				if !s.ProfileUntil.IsZero() && !now.Before(s.ProfileUntil) {
					continue
				}
				result = append(result, Candidate{Signal: s, Book: b})
			}
		}
	}
	return result
}

func priceFromNano(p int64) density.Price {
	return density.Price{Units: p / Nano, Nano: int32(p % Nano)}
}

func (f *Feed) moved(ep *Episode, now time.Time) bool {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.check(now)
	return f.reasons[ep.Instrument.ID] == "" && Price(f.anchors[ep.Instrument.ID].Price) != ep.Density
}

func (f *Feed) filled(ep *Episode, now time.Time) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if watch := f.watches[ep.Instrument.ID]; watch != nil {
		watch.Entry.Filled = ep.Entry.Filled
	}
	f.check(now)
}

func (f *Feed) replacement(ep *Episode, now time.Time) (*Episode, density.Signal, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.check(now)
	id := ep.Instrument.ID
	if f.reasons[id] != "" || !f.ready[id] {
		return nil, density.Signal{}, fmt.Errorf("replacement market unavailable")
	}
	for _, s := range f.candidates[id] {
		if s.InstrumentID != id || s.Price != f.anchors[id].Price {
			continue
		}
		plan, err := Plan(f.cfg, ep.Instrument, s, f.books[id], now)
		return plan, s, err
	}
	return nil, density.Signal{}, fmt.Errorf("replacement no longer qualifies")
}

// rebase installs new pending protection only while the selected candidate is
// still current. It never clears a latched data gap or actual density loss.
func (f *Feed) rebase(ep *Episode, s density.Signal, now time.Time) bool {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.check(now)
	id := ep.Instrument.ID
	if f.reasons[id] != "" || f.anchors[id].Price != s.Price {
		return false
	}
	for _, current := range f.candidates[id] {
		if current.InstrumentID == id && current.Price == s.Price && current.Since.Equal(s.Since) {
			copy := *ep
			f.watches[id] = &copy
			f.anchors[id] = current
			return true
		}
	}
	return false
}
