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
}

func NewFeed(c Config) *Feed {
	return &Feed{cfg: c, books: map[string]density.Book{}, signals: map[string][]density.Signal{}, ready: map[string]bool{}, watermarks: map[string]time.Time{}, trades: map[string]time.Time{}, watches: map[string]*Episode{}, reasons: map[string]string{}}
}

func (f *Feed) Trade(event density.Event, now time.Time) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if event.Time.IsZero() || event.Time.After(now) || now.Sub(event.Time) > f.cfg.Freshness || !event.Time.After(f.trades[event.InstrumentID]) || Price(event.Price) <= 0 || event.Quantity <= 0 {
		return
	}
	f.trades[event.InstrumentID] = event.Time
	if ep := f.watches[event.InstrumentID]; ep != nil && f.reasons[event.InstrumentID] == "" && Price(event.Price) <= ep.StopPrice {
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
	delete(f.reasons, id)
}
func (f *Feed) SetReady(id string, ready bool) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.ready[id] = ready
	if !ready {
		delete(f.signals, id)
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
	for id := range f.watches {
		if f.reasons[id] == "" {
			f.reasons[id] = "data_gap"
		}
	}
}
func (f *Feed) Book(b density.Book, signals []density.Signal, now time.Time) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if !b.Time.After(f.watermarks[b.InstrumentID]) {
		return
	}
	if !validBook(b, now, f.cfg.Freshness) {
		delete(f.signals, b.InstrumentID)
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
	} else {
		delete(f.signals, b.InstrumentID)
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
		for _, l := range b.Bids {
			if Price(l.Price) == ep.Density {
				qty = l.Quantity
				break
			}
		}
		if float64(qty) < float64(ep.DensityLots)*f.cfg.MinRemaining {
			f.reasons[id] = "density_lost"
			continue
		}
		if Price(b.Bids[0].Price) <= ep.StopPrice {
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
