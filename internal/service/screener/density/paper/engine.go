package paper

import (
	"fmt"
	"math"
	"sort"
	"time"
	"tinvest/internal/service/screener/density"
)

const nano int64 = 1000000000

func price(p density.Price) int64 {
	if p.Units < 0 || p.Units > 100000000 || p.Nano < 0 || p.Nano >= int32(nano) {
		return 0
	}
	return p.Units*nano + int64(p.Nano)
}
func rub(p int64) float64 { return float64(p) / float64(nano) }

type Order struct {
	TrackedDensity                                        int64  // Monitoring anchor; fixed trade levels stay unchanged after fill.
	InstrumentID                                          string `json:"instrument_id"`
	Ticker                                                string `json:"ticker"`
	State                                                 string `json:"state"`
	Density, Entry, Stop, Take                            int64  // Prices in nanorubles; records also provide RUB prices.
	DensityLots, PlannedLots, FilledLots, OpenLots        int64
	Lot                                                   int64
	Created, ActiveAt, Opened, TakeActive, ExitAt, Closed time.Time
	ExitReason                                            string
	EntryFees, ExitFees, GrossPnL, NetPnL                 float64
	Uncertain                                             bool
}
type Record struct {
	At                                  time.Time `json:"at"`
	Kind                                string    `json:"kind"`
	Detail                              string    `json:"detail,omitempty"`
	Order                               *Order    `json:"order,omitempty"`
	EntryRubles, StopRubles, TakeRubles float64
	FillLots                            int64   `json:"fill_lots,omitempty"`
	FillRubles                          float64 `json:"fill_rubles,omitempty"`
}
type marketBook struct {
	book     density.Book
	received time.Time
	consumed time.Time
}
type Engine struct {
	cfg                                     Config
	instruments                             map[string]density.Instrument
	orders                                  map[string]*Order
	books                                   map[string]marketBook
	lastTrade                               map[string]time.Time
	lastBook                                map[string]time.Time
	cooldown                                map[string]time.Time
	cash, realized, fees, peak, maxDrawdown float64
	closed, wins, canceled, gaps            int
	complete                                bool
	record                                  func(Record) error
	err                                     error
}

func New(cfg Config, instruments []density.Instrument, record func(Record) error) (*Engine, error) {
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	e := &Engine{cfg: cfg, instruments: make(map[string]density.Instrument), orders: make(map[string]*Order), books: make(map[string]marketBook), lastTrade: make(map[string]time.Time), lastBook: make(map[string]time.Time), cooldown: make(map[string]time.Time), cash: cfg.InitialCash, peak: cfg.InitialCash, complete: true, record: record}
	for _, i := range instruments {
		if i.ID == "" || i.Ticker == "" || i.Lot <= 0 || price(i.PriceStep) <= 0 {
			return nil, fmt.Errorf("invalid instrument metadata for %s", i.Ticker)
		}
		if _, ok := e.instruments[i.ID]; ok {
			return nil, fmt.Errorf("duplicate instrument %s", i.ID)
		}
		e.instruments[i.ID] = i
	}
	return e, nil
}
func (e *Engine) Err() error { return e.err }
func (e *Engine) emit(kind string, o *Order, at time.Time, detail string, lots int64, fill float64) {
	if e.err != nil || e.record == nil {
		return
	}
	r := Record{At: at, Kind: kind, Detail: detail, FillLots: lots, FillRubles: fill}
	if o != nil {
		copy := *o
		r.Order = &copy
		r.EntryRubles = rub(o.Entry)
		r.StopRubles = rub(o.Stop)
		r.TakeRubles = rub(o.Take)
	}
	if err := e.record(r); err != nil {
		e.err = fmt.Errorf("paper journal: %w", err)
	}
}
func active(o *Order) bool { return o.State == "pending" || o.OpenLots > 0 }
func (e *Engine) reserved() float64 {
	total := 0.
	for _, o := range e.orders {
		if o.State == "pending" {
			total += float64(o.PlannedLots) * float64(o.Lot) * rub(o.Entry) * (1 + e.cfg.Commission)
		}
	}
	return total
}
func validBook(b density.Book, now time.Time, fresh time.Duration) bool {
	if b.Time.IsZero() || now.Sub(b.Time) > fresh || b.Time.After(now) || b.Lot <= 0 || len(b.Bids) == 0 || len(b.Asks) == 0 {
		return false
	}
	for side, levels := range [][]density.Level{b.Bids, b.Asks} {
		for i, l := range levels {
			p := price(l.Price)
			if p <= 0 || l.Quantity <= 0 {
				return false
			}
			if i > 0 {
				prev := price(levels[i-1].Price)
				if side == 0 && prev <= p || side == 1 && prev >= p {
					return false
				}
			}
		}
	}
	return price(b.Bids[0].Price) < price(b.Asks[0].Price)
}
func (e *Engine) fresh(id string, now time.Time) bool {
	b, ok := e.books[id]
	return ok && now.Sub(b.received) <= e.cfg.Freshness && now.Sub(b.book.Time) <= e.cfg.Freshness
}
func (e *Engine) Observe(s density.Signal, now time.Time) {
	if !s.ProfileUntil.IsZero() && !now.Before(s.ProfileUntil) {
		return
	}
	if e.err != nil || !e.fresh(s.InstrumentID, now) || now.Before(e.cooldown[s.InstrumentID]) {
		return
	}
	if o := e.orders[s.InstrumentID]; o != nil && active(o) {
		return
	}
	count := 0
	for _, o := range e.orders {
		if active(o) {
			count++
		}
	}
	if count >= e.cfg.MaxPositions {
		return
	}
	o := e.plan(s, now, 0)
	if o == nil {
		return
	}
	e.orders[o.InstrumentID] = o
	e.emit("entry_placed", o, now, "", 0, 0)
	e.executeEntry(o, now)
}

func (e *Engine) plan(s density.Signal, now time.Time, released float64) *Order {
	i, ok := e.instruments[s.InstrumentID]
	if !ok || s.Quantity <= 0 {
		return nil
	}
	level := price(s.Price)
	step := price(i.PriceStep)
	if level <= 0 || level%step != 0 {
		return nil
	}
	b := e.books[i.ID].book
	var densityLots int64
	for _, l := range b.Bids {
		if price(l.Price) == level {
			densityLots = l.Quantity
			break
		}
	}
	if densityLots <= 0 {
		return nil
	}
	entry, err := density.EntryLimit(level, step, e.cfg.EntryTicks, e.cfg.EntryPct, price(b.Asks[0].Price))
	if err != nil {
		return nil
	}
	stop := level - e.cfg.StopTicks*step
	if stop <= 0 {
		return nil
	}
	targetTicks := math.Ceil(float64(entry-stop) / float64(step) * e.cfg.RewardRisk)
	if targetTicks > float64(math.MaxInt64-entry)/float64(step) {
		return nil
	}
	take := entry + int64(targetTicks)*step
	available := math.Min(e.cfg.PositionRubles, e.cash-e.reserved()+released)
	lots := int64(math.Floor(available / (rub(entry) * float64(i.Lot) * (1 + e.cfg.Commission))))
	if e.cfg.IgnoreCapital {
		lots = e.cfg.MaxEntryLots
	}
	if lots < 1 {
		return nil
	}
	if e.cfg.MaxEntryLots > 0 {
		lots = min(lots, e.cfg.MaxEntryLots)
	}
	o := &Order{InstrumentID: i.ID, Ticker: i.Ticker, State: "pending", Density: level, TrackedDensity: level, Entry: entry, Stop: stop, Take: take, DensityLots: densityLots, PlannedLots: lots, Lot: i.Lot, Created: now, ActiveAt: now.Add(e.cfg.Latency)}
	return o
}
func (e *Engine) OnBook(b density.Book, now time.Time) {
	e.OnBookWithCandidates(b, nil, now)
}

func (e *Engine) OnBookWithCandidates(b density.Book, candidates []density.Signal, now time.Time) {
	i, known := e.instruments[b.InstrumentID]
	if !known {
		return
	}
	if !validBook(b, now, e.cfg.Freshness) || b.Lot != i.Lot {
		e.invalidate(b.InstrumentID, "invalid_book", now)
		return
	}
	if !b.Time.After(e.lastBook[b.InstrumentID]) {
		return
	}
	for _, levels := range [][]density.Level{b.Bids, b.Asks} {
		for _, l := range levels {
			if price(l.Price)%price(i.PriceStep) != 0 {
				e.invalidate(b.InstrumentID, "off_tick_book", now)
				return
			}
		}
	}
	b.Bids = append([]density.Level(nil), b.Bids...)
	b.Asks = append([]density.Level(nil), b.Asks...)
	e.lastBook[b.InstrumentID] = b.Time
	e.books[b.InstrumentID] = marketBook{book: b, received: now}
	o := e.orders[b.InstrumentID]
	if o == nil || !active(o) {
		e.mark(now)
		return
	}
	if o.State == "pending" && now.Sub(o.Created) >= e.cfg.EntryTTL {
		e.cancel(o, "entry_timeout", now)
		return
	}
	if o.State != "exiting" {
		anchor := o.TrackedDensity
		if anchor == 0 {
			anchor = o.Density
		}
		var retained int64
		for _, l := range b.Bids {
			if price(l.Price) == anchor {
				retained = l.Quantity
				break
			}
		}
		reason := ""
		if retained == 0 || (o.State == "pending" && float64(retained) < float64(o.DensityLots)*e.cfg.MinRemaining) {
			moved, ok := density.Relocation(b, density.Price{Units: anchor / nano, Nano: int32(anchor % nano)}, o.DensityLots, e.cfg.MinRemaining, candidates, now)
			if !ok {
				reason = "density_lost"
				if o.OpenLots > 0 {
					reason = "density_disappeared"
				}
			} else if o.State == "pending" {
				released := float64(o.PlannedLots) * float64(o.Lot) * rub(o.Entry) * (1 + e.cfg.Commission)
				next := e.plan(moved, now, released)
				if next == nil {
					reason = "replacement_unavailable"
				} else {
					next.Created = o.Created
					*o = *next
					e.emit("entry_replaced", o, now, "density_relocated", 0, 0)
				}
			} else {
				o.TrackedDensity = price(moved.Price)
			}
		}
		if o.OpenLots > 0 && price(b.Bids[0].Price) <= o.Stop {
			reason = "stop"
		}
		if reason != "" {
			if o.State == "pending" {
				e.cancel(o, reason, now)
			} else {
				e.requestExit(o, reason, now)
			}
		}
	}
	e.executeEntry(o, now)
	e.executeTake(o, now)
	e.executeExit(o, now)
	e.mark(now)
}
func (e *Engine) OnTrade(t density.Event, now time.Time) {
	if e.err != nil || !e.fresh(t.InstrumentID, now) || t.Quantity <= 0 || price(t.Price) <= 0 || t.Time.IsZero() || t.Time.After(now) || now.Sub(t.Time) > e.cfg.Freshness || !t.Time.After(e.lastTrade[t.InstrumentID]) {
		return
	}
	e.lastTrade[t.InstrumentID] = t.Time
	o := e.orders[t.InstrumentID]
	if o == nil || !active(o) {
		return
	}
	if o.State == "open" && price(t.Price) <= o.Stop {
		e.requestExit(o, "stop", now)
	}
	if e.cfg.MaxHold > 0 && o.State == "open" && now.Sub(o.Opened) >= e.cfg.MaxHold {
		e.requestExit(o, "position_timeout", now)
	}
	if o.State == "pending" && !now.Before(o.ActiveAt) && !t.Time.Before(o.ActiveAt) && now.Sub(o.Created) < e.cfg.EntryTTL && price(t.Price) < o.Entry {
		e.fillEntry(o, min(t.Quantity, o.PlannedLots), o.Entry, now)
		if price(t.Price) <= o.Stop {
			e.requestExit(o, "stop", now)
		}
	} else if o.State == "open" && !now.Before(o.TakeActive) && !t.Time.Before(o.TakeActive) && price(t.Price) > o.Take {
		e.sell(o, min(t.Quantity, o.OpenLots), o.Take, "take_profit", now)
	}
	e.mark(now)
}

// executeEntry models a marketable percentage entry using visible ask liquidity.
// Passive tick-offset orders continue to require a trade through their limit.
func (e *Engine) executeEntry(o *Order, now time.Time) {
	if e.err != nil || (e.cfg.EntryPct <= 0 && !e.cfg.BookExecution) || o.State != "pending" || now.Before(o.ActiveAt) || now.Sub(o.Created) >= e.cfg.EntryTTL {
		return
	}
	b := e.books[o.InstrumentID]
	if (e.cfg.Latency > 0 && b.book.Time.Before(o.ActiveAt)) || b.consumed.Equal(b.book.Time) {
		return
	}
	ask := b.book.Asks[0]
	if price(ask.Price) > o.Entry {
		return
	}
	e.fillEntry(o, min(ask.Quantity, o.PlannedLots), price(ask.Price), now)
	b.consumed = b.book.Time
	e.books[o.InstrumentID] = b
}

// executeTake uses visible bid liquidity only in book execution mode.
func (e *Engine) executeTake(o *Order, now time.Time) {
	if e.err != nil || !e.cfg.BookExecution || o.State != "open" || now.Before(o.TakeActive) {
		return
	}
	b := e.books[o.InstrumentID]
	if b.book.Time.Before(o.TakeActive) || b.consumed.Equal(b.book.Time) {
		return
	}
	bid := b.book.Bids[0]
	if price(bid.Price) < o.Take {
		return
	}
	e.sell(o, min(bid.Quantity, o.OpenLots), o.Take, "take_profit", now)
	b.consumed = b.book.Time
	e.books[o.InstrumentID] = b
}

func (e *Engine) fillEntry(o *Order, lots, entry int64, now time.Time) {
	o.Entry = entry
	step := price(e.instruments[o.InstrumentID].PriceStep)
	o.Take = entry + int64(math.Ceil(float64(entry-o.Stop)/float64(step)*e.cfg.RewardRisk))*step
	cost := rub(entry) * float64(lots) * float64(o.Lot)
	fee := cost * e.cfg.Commission
	e.cash -= cost + fee
	e.fees += fee
	o.EntryFees = fee
	o.FilledLots = lots
	o.OpenLots = lots
	o.Opened = now
	o.TakeActive = now.Add(e.cfg.Latency)
	o.State = "open"
	e.emit("entry_filled", o, now, "unfilled entry remainder canceled", lots, rub(entry))
}

func (e *Engine) cancel(o *Order, reason string, now time.Time) {
	o.State = "canceled"
	o.Closed = now
	o.ExitReason = reason
	e.canceled++
	e.cooldown[o.InstrumentID] = now.Add(e.cfg.Cooldown)
	e.emit("entry_canceled", o, now, reason, 0, 0)
}
func (e *Engine) requestExit(o *Order, reason string, now time.Time) {
	if o.State == "exiting" {
		return
	}
	o.State = "exiting"
	o.ExitReason = reason
	o.ExitAt = now.Add(e.cfg.Latency)
	if b, ok := e.books[o.InstrumentID]; ok {
		b.consumed = b.book.Time
		e.books[o.InstrumentID] = b
	}
	e.emit("exit_requested", o, now, reason, 0, 0)
}
func (e *Engine) invalidate(id, reason string, now time.Time) {
	_, hadBook := e.books[id]
	delete(e.books, id)
	if now.After(e.lastBook[id]) {
		e.lastBook[id] = now
	}
	o := e.orders[id]
	if o == nil || !active(o) || !hadBook && o.Uncertain {
		return
	}
	e.complete = false
	e.gaps++
	if o.State == "pending" {
		e.cancel(o, reason, now)
	} else {
		o.Uncertain = true
		// Data gaps suspend execution until a fresh valid book arrives.
		// Keep the position and any previously requested strategy exit intact.
		if reason == "trading_closed" {
			e.requestExit(o, reason, now)
		}
		e.emit("data_gap", o, now, reason, 0, 0)
	}
}
func (e *Engine) Gap(reason string, now time.Time) {
	e.complete = false
	e.gaps++
	e.emit("stream_gap", nil, now, reason, 0, 0)
	ids := make([]string, 0, len(e.instruments))
	for id := range e.instruments {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	for _, id := range ids {
		e.invalidate(id, reason, now)
	}
}
func (e *Engine) Tick(now time.Time) {
	ids := make([]string, 0, len(e.orders))
	for id := range e.orders {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	for _, id := range ids {
		o := e.orders[id]
		if !active(o) {
			continue
		}
		if !e.fresh(id, now) {
			e.invalidate(id, "stale_book", now)
			continue
		}
		if o.State == "pending" && now.Sub(o.Created) >= e.cfg.EntryTTL {
			e.cancel(o, "entry_timeout", now)
		}
		if e.cfg.MaxHold > 0 && o.OpenLots > 0 && o.State != "exiting" && now.Sub(o.Opened) >= e.cfg.MaxHold {
			e.requestExit(o, "position_timeout", now)
		}
		e.executeExit(o, now)
	}
	e.mark(now)
}
