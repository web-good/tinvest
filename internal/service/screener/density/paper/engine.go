package paper

import (
	"fmt"
	"math"
	"sort"
	"time"
	"tinvest/internal/service/screener/bounce"
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
	entryDensityLots                                      int64
	entryUntil, profileUntil                              time.Time
	AverageVolumeLots                                     float64 `json:"average_volume_lots,omitempty"`
	HasAverageVolume                                      bool    `json:"has_average_volume,omitempty"`
	ExitReason                                            string
	EntryFees, ExitFees, GrossPnL, NetPnL                 float64
	Uncertain                                             bool
	SellLimit                                             int64 `json:"sell_limit,omitempty"`
	sellLimitActive                                       time.Time
	sellWalls                                             sellWalls
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
	book       density.Book
	received   time.Time
	consumed   time.Time
	candidates []density.Signal
}
type Engine struct {
	cfg                                     Config
	instruments                             map[string]density.Instrument
	orders                                  map[string]*Order
	reboundWatches                          map[string]map[int64]*reboundWatch
	bounces                                 map[string]*density.BounceWatch
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
	e := &Engine{cfg: cfg, instruments: make(map[string]density.Instrument), orders: make(map[string]*Order), reboundWatches: make(map[string]map[int64]*reboundWatch), bounces: make(map[string]*density.BounceWatch), books: make(map[string]marketBook), lastTrade: make(map[string]time.Time), lastBook: make(map[string]time.Time), cooldown: make(map[string]time.Time), cash: cfg.InitialCash, peak: cfg.InitialCash, complete: true, record: record}
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
	if e.cfg.ScreenerStrategy {
		e.watchRebound(s, now)
	} else if !e.cfg.CandleEntries {
		e.observe(s, now, time.Time{})
	}
}

func (e *Engine) observe(s density.Signal, now, entryUntil time.Time) bool {
	if entryUntil.IsZero() && !s.ProfileUntil.IsZero() && !now.Before(s.ProfileUntil) {
		return false
	}
	if e.err != nil || e.entryHoursBlocked(now) || !e.fresh(s.InstrumentID, now) || now.Before(e.cooldown[s.InstrumentID]) {
		return false
	}
	if o := e.orders[s.InstrumentID]; o != nil && active(o) {
		return false
	}
	count := 0
	for _, o := range e.orders {
		if active(o) {
			count++
		}
	}
	if count >= e.cfg.MaxPositions {
		return false
	}
	o := e.plan(s, now, 0, entryUntil)
	if o == nil {
		return false
	}
	e.orders[o.InstrumentID] = o
	e.emit("entry_placed", o, now, "", 0, 0)
	e.executeEntry(o, now)
	return true
}

func (e *Engine) plan(s density.Signal, now time.Time, released float64, entryUntil time.Time) *Order {
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
	// The screener detector already checked its configured fraction of the M5 average.
	if densityLots <= 0 || !e.cfg.ScreenerStrategy && entryUntil.IsZero() && float64(densityLots) < s.MinLots {
		return nil
	}
	entry := price(b.Asks[0].Price)
	if !e.cfg.ScreenerStrategy && e.cfg.fixedLimitEntry() {
		entry = density.FixedEntryLimit(level, step)
	} else if !e.cfg.ScreenerStrategy && entryUntil.IsZero() {
		var err error
		entry, err = density.EntryLimit(level, step, e.cfg.EntryTicks, e.cfg.EntryZoneTicks, entry)
		if err != nil {
			return nil
		}
	} else if !entryUntil.IsZero() && (!now.Add(e.cfg.Latency).Before(entryUntil) || entry <= level || !bounce.Retains(float64(s.Quantity), float64(densityLots))) {
		return nil
	}
	stop := level - e.cfg.StopTicks*step
	var take int64
	if e.cfg.ScreenerStrategy {
		stop, take = screenerExitPrices(entry, step)
		if stop <= 0 || take <= 0 {
			return nil
		}
	} else {
		if e.cfg.fixedLimitEntry() {
			stop = density.EntryStop(entry, step)
		}
		if stop <= 0 {
			return nil
		}
		if !e.cfg.IgnoreCapital || entryUntil.IsZero() && !e.cfg.fixedLimitEntry() {
			targetTicks := takeProfitTicks(entry, stop, step, e.cfg.RewardRisk)
			if targetTicks > float64(math.MaxInt64-entry)/float64(step) {
				return nil
			}
			take = entry + int64(targetTicks)*step
		} else {
			var ok bool
			take, ok = density.TakeProfit(entry, stop, step, e.cfg.RewardRisk)
			if !ok {
				return nil
			}
		}
	}
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
	o := &Order{InstrumentID: i.ID, Ticker: i.Ticker, State: "pending", Density: level, TrackedDensity: level, Entry: entry, Stop: stop, Take: take, DensityLots: densityLots, PlannedLots: lots, Lot: i.Lot, Created: now, entryUntil: entryUntil, profileUntil: s.ProfileUntil, AverageVolumeLots: s.MinLots, HasAverageVolume: s.HasAverageVolume, ActiveAt: now.Add(e.cfg.Latency)}
	if !entryUntil.IsZero() {
		o.entryDensityLots = s.Quantity
		o.profileUntil = time.Time{}
	}
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
	if previous, ok := e.books[b.InstrumentID]; ok && now.Sub(previous.book.Time) > e.cfg.Freshness {
		delete(e.bounces, b.InstrumentID)
		delete(e.reboundWatches, b.InstrumentID)
	}
	e.lastBook[b.InstrumentID] = b.Time
	e.books[b.InstrumentID] = marketBook{book: b, received: now, candidates: candidates}
	if e.cfg.ScreenerStrategy {
		defer e.checkReboundBook(b, now)
	}
	o := e.orders[b.InstrumentID]
	if o == nil || !active(o) {
		e.mark(now)
		return
	}
	if e.cancelEntryOutsideHours(o, now) {
		return
	}
	if o.State == "pending" && e.entryExpired(o, now) {
		e.cancel(o, "entry_timeout", now)
		return
	}
	reason := ""
	if o.State == "pending" || o.State == "open" && !e.cfg.ScreenerStrategy {
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
		if o.State == "pending" && e.cfg.fixedLimitEntry() && !density.RetainsEntryDensity(float64(o.DensityLots), float64(retained)) {
			e.cancel(o, "density_lost", now)
			return
		}
		if o.State == "pending" && o.entryDensityLots > 0 && !bounce.Retains(float64(o.entryDensityLots), float64(retained)) {
			e.cancel(o, "density_lost", now)
			return
		}
		if retained == 0 || (o.State == "pending" && !e.cfg.fixedLimitEntry() && o.entryDensityLots == 0 && float64(retained) < max(float64(o.DensityLots)*e.cfg.MinRemaining, o.AverageVolumeLots)) {
			moved, ok := density.Relocation(b, density.Price{Units: anchor / nano, Nano: int32(anchor % nano)}, o.DensityLots, e.cfg.MinRemaining, candidates, now)
			if !ok {
				reason = "density_lost"
				if o.OpenLots > 0 {
					reason = "density_disappeared"
				}
			} else if o.State == "pending" && o.entryUntil.IsZero() {
				released := float64(o.PlannedLots) * float64(o.Lot) * rub(o.Entry) * (1 + e.cfg.Commission)
				next := e.plan(moved, now, released, time.Time{})
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
	}
	if o.State == "open" && price(b.Bids[0].Price) <= o.Stop {
		reason = "stop"
	}
	if reason != "" {
		if o.State == "pending" {
			e.cancel(o, reason, now)
		} else {
			e.requestExit(o, reason, now)
		}
	}
	e.executeEntry(o, now)
	if (!e.cfg.CandleEntries && !e.cfg.fixedLimitEntry() || !e.cfg.IgnoreCapital) && e.err == nil && o.State == "open" && b.Time.After(o.Opened) {
		e.updateSellLimit(o, b, now)
	}
	e.executeTake(o, now)
	e.executeExit(o, now)
	e.mark(now)
}
func (e *Engine) OnTrade(t density.Event, now time.Time) {
	if e.cfg.ScreenerStrategy {
		e.trackReboundTrade(t, now)
		return // Exits still execute against the book.
	}
	if e.err != nil || !e.fresh(t.InstrumentID, now) || t.Quantity <= 0 || price(t.Price) <= 0 || t.Time.IsZero() || t.Time.After(now) || now.Sub(t.Time) > e.cfg.Freshness || !t.Time.After(e.lastTrade[t.InstrumentID]) {
		return
	}
	e.lastTrade[t.InstrumentID] = t.Time
	o := e.orders[t.InstrumentID]
	if o == nil || !active(o) {
		return
	}
	if e.cancelEntryOutsideHours(o, now) {
		return
	}
	if o.State == "open" && price(t.Price) <= o.Stop {
		e.requestExit(o, "stop", now)
	}
	if e.cfg.MaxHold > 0 && o.State == "open" && now.Sub(o.Opened) >= e.cfg.MaxHold {
		e.requestExit(o, "position_timeout", now)
	}
	if o.State == "pending" && o.entryUntil.IsZero() && !now.Before(o.ActiveAt) && !t.Time.Before(o.ActiveAt) && !e.entryExpired(o, now) && price(t.Price) < o.Entry {
		e.fillEntry(o, min(t.Quantity, o.PlannedLots), o.Entry, now)
		if price(t.Price) <= o.Stop {
			e.requestExit(o, "stop", now)
		}
	} else if o.State == "open" {
		limit, activeAt, reason := o.takeOrder()
		if !now.Before(activeAt) && !t.Time.Before(activeAt) && price(t.Price) > limit {
			e.sell(o, min(t.Quantity, o.OpenLots), limit, reason, now)
		}
	}
	e.mark(now)
}

// executeEntry models a marketable entry using visible ask liquidity.
// Passive tick-offset orders continue to require a trade through their limit.
func (e *Engine) executeEntry(o *Order, now time.Time) {
	if e.err != nil || (!e.cfg.fixedLimitEntry() && o.entryUntil.IsZero() && e.cfg.EntryZoneTicks <= 0 && !e.cfg.BookExecution) || o.State != "pending" || now.Before(o.ActiveAt) || e.entryExpired(o, now) {
		return
	}
	b := e.books[o.InstrumentID]
	if (e.cfg.Latency > 0 && b.book.Time.Before(o.ActiveAt)) || b.consumed.Equal(b.book.Time) {
		return
	}
	ask := b.book.Asks[0]
	if price(ask.Price) > o.Entry {
		if !o.entryUntil.IsZero() {
			e.cancel(o, "entry_unfilled", now)
		}
		return
	}
	e.fillEntry(o, min(ask.Quantity, o.PlannedLots), price(ask.Price), now)
	b.consumed = b.book.Time
	e.books[o.InstrumentID] = b
}

// executeTake uses visible bids for resting sell limits and book-mode takes.
func (e *Engine) executeTake(o *Order, now time.Time) {
	if e.err != nil || o.State != "open" || (o.SellLimit == 0 && !e.cfg.BookExecution) {
		return
	}
	limit, activeAt, reason := o.takeOrder()
	b := e.books[o.InstrumentID]
	if now.Before(activeAt) || b.book.Time.Before(activeAt) || b.consumed.Equal(b.book.Time) {
		return
	}
	bid := b.book.Bids[0]
	if price(bid.Price) < limit {
		return
	}
	e.sell(o, min(bid.Quantity, o.OpenLots), limit, reason, now)
	b.consumed = b.book.Time
	e.books[o.InstrumentID] = b
}

// takeProfitTicks keeps the target at least 0.3% above entry, rounded up to a tick.
func takeProfitTicks(entry, stop, step int64, rewardRisk float64) float64 {
	minProfit := (entry*3 + 999) / 1000
	minTicks := (minProfit + step - 1) / step
	return math.Max(math.Ceil(float64(entry-stop)/float64(step)*rewardRisk), float64(minTicks))
}

func screenerExitPrices(entry, step int64) (stop, take int64) {
	// Round the 0.1% stop down and the 0.5% take up to a tradable tick.
	stop = (entry - entry/1000 - min(entry%1000, 1)) / step * step
	profit := entry/200 + min(entry%200, 1)
	ticks := profit / step
	if profit%step != 0 {
		ticks++
	}
	if ticks > (math.MaxInt64-entry)/step {
		return 0, 0
	}
	return stop, entry + ticks*step
}

func (e *Engine) fillEntry(o *Order, lots, entry int64, now time.Time) {
	o.Entry = entry
	step := price(e.instruments[o.InstrumentID].PriceStep)
	if e.cfg.ScreenerStrategy {
		o.Stop, o.Take = screenerExitPrices(entry, step)
	} else {
		if e.cfg.fixedLimitEntry() {
			o.Stop = density.EntryStop(entry, step)
		}
		if !e.cfg.IgnoreCapital || o.entryUntil.IsZero() && !e.cfg.fixedLimitEntry() {
			o.Take = entry + int64(takeProfitTicks(entry, o.Stop, step, e.cfg.RewardRisk))*step
		} else if take, ok := density.TakeProfit(entry, o.Stop, step, e.cfg.RewardRisk); ok {
			o.Take = take
		}
	}
	cost := rub(entry) * float64(lots) * float64(o.Lot)
	fee := cost * e.cfg.Commission
	e.cash -= cost + fee
	e.fees += fee
	o.EntryFees = fee
	o.FilledLots = lots
	o.OpenLots = lots
	o.Opened = now
	if !e.cfg.CandleEntries || !e.cfg.IgnoreCapital {
		o.sellWalls = newSellWalls(e.books[o.InstrumentID].book.Asks, o.DensityLots)
	}
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
	e.cancelSellLimit(o, now)
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
	delete(e.bounces, id)
	delete(e.reboundWatches, id)
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
		e.cancelSellLimit(o, now)
		o.Uncertain = true
		// Data gaps suspend execution until a fresh valid book arrives.
		// Keep the position and any previously requested strategy exit intact.
		if reason == "trading_closed" && (!e.cfg.CandleEntries && !e.cfg.fixedLimitEntry() || !e.cfg.IgnoreCapital) {
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
		if e.cancelEntryOutsideHours(o, now) {
			continue
		}
		if !e.fresh(id, now) {
			e.invalidate(id, "stale_book", now)
			continue
		}
		if o.State == "pending" && e.entryExpired(o, now) {
			e.cancel(o, "entry_timeout", now)
		}
		if e.cfg.MaxHold > 0 && o.OpenLots > 0 && o.State != "exiting" && now.Sub(o.Opened) >= e.cfg.MaxHold {
			e.requestExit(o, "position_timeout", now)
		}
		e.executeExit(o, now)
	}
	e.mark(now)
}

func (e *Engine) entryExpired(o *Order, now time.Time) bool {
	return now.Sub(o.Created) >= e.cfg.EntryTTL || !o.entryUntil.IsZero() && !now.Before(o.entryUntil) ||
		!o.profileUntil.IsZero() && !now.Before(o.profileUntil)
}

func (e *Engine) bounceBook(b density.Book, signals []density.Signal, now time.Time) {
	current, ok := e.books[b.InstrumentID]
	if !ok || !current.book.Time.Equal(b.Time) || !current.received.Equal(now) {
		return
	}
	watch := e.bounces[b.InstrumentID]
	if watch == nil {
		watch = &density.BounceWatch{}
		e.bounces[b.InstrumentID] = watch
	}
	watch.Book(b, signals, e.instruments[b.InstrumentID].PriceStep)
	e.tryBounce(watch, now)
}

func (e *Engine) candle(id string, candle density.Candle, now time.Time) {
	if watch := e.bounces[id]; watch != nil && e.fresh(id, now) {
		watch.Candle(candle, now)
		e.tryBounce(watch, now)
	}
}

func (e *Engine) tryBounce(watch *density.BounceWatch, now time.Time) {
	if s, until, ok := watch.Ready(now); ok && e.observe(s, now, until) {
		watch.Consume(now)
	}
}
