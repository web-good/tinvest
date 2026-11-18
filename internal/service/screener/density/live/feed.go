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
	return f.prepare(i, Candidate{Signal: original}, now)
}

func (f *Feed) prepare(i density.Instrument, c Candidate, now time.Time) (*Episode, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if !f.ready[i.ID] {
		return nil, fmt.Errorf("market no longer ready")
	}
	signals := f.signals[i.ID]
	if !c.EntryUntil.IsZero() {
		watch := f.pinbars[i.ID]
		if watch == nil {
			return nil, fmt.Errorf("pinbar confirmation lost")
		}
		s, until, ok := watch.Ready(now)
		if !ok || !until.Equal(c.EntryUntil) {
			return nil, fmt.Errorf("pinbar confirmation expired")
		}
		signals = []density.Signal{s}
	}
	for _, s := range signals {
		if c.EntryUntil.IsZero() && (s.Price != c.Signal.Price || !s.Since.Equal(c.Signal.Since)) {
			continue
		}
		ep, err := plan(f.cfg, i, s, f.books[i.ID], now, c.EntryUntil)
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
	pinbars    map[string]*density.PinbarWatch
}
type Candidate struct {
	Signal     density.Signal
	Book       density.Book
	EntryUntil time.Time
}

// ClearSignals resets entry confirmations without interrupting position watchers.
func (f *Feed) ClearSignals() {
	f.mu.Lock()
	defer f.mu.Unlock()
	clear(f.signals)
	clear(f.candidates)
	clear(f.pinbars)
}

func NewFeed(c Config) *Feed {
	return &Feed{cfg: c, books: map[string]density.Book{}, signals: map[string][]density.Signal{}, candidates: map[string][]density.Signal{}, anchors: map[string]density.Signal{}, ready: map[string]bool{}, watermarks: map[string]time.Time{}, trades: map[string]time.Time{}, watches: map[string]*Episode{}, reasons: map[string]string{}, pinbars: map[string]*density.PinbarWatch{}}
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
		delete(f.pinbars, id)
		f.dataLoss(id, "data_unavailable")
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
	clear(f.pinbars)
	for id := range f.watches {
		f.dataLoss(id, "data_gap")
	}
}
func (f *Feed) Book(b density.Book, signals []density.Signal, now time.Time) {
	f.BookWithCandidates(b, signals, signals, now)
}

func (f *Feed) BookWithCandidates(b density.Book, signals, candidates []density.Signal, now time.Time) {
	f.bookWithCandidates(b, signals, candidates, density.Price{}, now)
}

func (f *Feed) bookWithCandidates(b density.Book, signals, candidates []density.Signal, step density.Price, now time.Time) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if !b.Time.After(f.watermarks[b.InstrumentID]) {
		return
	}
	if !validBook(b, now, f.cfg.Freshness) {
		delete(f.signals, b.InstrumentID)
		delete(f.candidates, b.InstrumentID)
		delete(f.pinbars, b.InstrumentID)
		f.dataLoss(b.InstrumentID, "invalid_book")
		return
	}
	// Preserve a stale interval even when data resumes during a blocking broker RPC.
	f.check(now)
	if previous, ok := f.books[b.InstrumentID]; ok && now.Sub(previous.Time) > f.cfg.Freshness {
		delete(f.pinbars, b.InstrumentID)
	}
	b.Bids = append([]density.Level(nil), b.Bids...)
	b.Asks = append([]density.Level(nil), b.Asks...)
	f.books[b.InstrumentID] = b
	f.watermarks[b.InstrumentID] = b.Time
	if f.ready[b.InstrumentID] {
		f.signals[b.InstrumentID] = append([]density.Signal(nil), signals...)
		f.candidates[b.InstrumentID] = append([]density.Signal(nil), candidates...)
		if f.cfg.PinbarEntries {
			watch := f.pinbars[b.InstrumentID]
			if watch == nil {
				watch = &density.PinbarWatch{}
				f.pinbars[b.InstrumentID] = watch
			}
			watch.Book(b, signals, candidates, step, f.cfg.MinRemaining, now)
		}
	} else {
		delete(f.signals, b.InstrumentID)
		delete(f.candidates, b.InstrumentID)
		delete(f.pinbars, b.InstrumentID)
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
			f.dataLoss(id, "data_unavailable")
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
func (f *Feed) Candidates(now time.Time) []Candidate {
	f.mu.Lock()
	defer f.mu.Unlock()
	var result []Candidate
	for id, signals := range f.signals {
		b := f.books[id]
		if f.ready[id] && validBook(b, now, f.cfg.Freshness) {
			if f.cfg.PinbarEntries {
				if watch := f.pinbars[id]; watch != nil {
					if s, until, ok := watch.Ready(now); ok {
						result = append(result, Candidate{Signal: s, Book: b, EntryUntil: until})
					}
				}
				continue
			}
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

func (f *Feed) dataLoss(id, reason string) {
	if ep := f.watches[id]; ep != nil && f.reasons[id] == "" && (!f.cfg.PinbarEntries || ep.Entry.Filled == 0) {
		f.reasons[id] = reason
	}
}

func (f *Feed) candle(id string, candle density.Candle, now time.Time) {
	f.mu.Lock()
	defer f.mu.Unlock()
	b := f.books[id]
	if watch := f.pinbars[id]; watch != nil && f.ready[id] && validBook(b, now, f.cfg.Freshness) {
		watch.Candle(candle, b, f.candidates[id], f.cfg.EntryZoneTicks, f.cfg.MinRemaining, now)
	}
}

func (f *Feed) claimPinbar(id string, until, now time.Time) bool {
	f.mu.Lock()
	defer f.mu.Unlock()
	watch := f.pinbars[id]
	if watch == nil || f.reasons[id] != "" || !f.ready[id] || !validBook(f.books[id], now, f.cfg.Freshness) {
		return false
	}
	_, expires, ready := watch.Ready(now)
	if !ready || !expires.Equal(until) {
		return false
	}
	watch.Consume()
	return true
}
