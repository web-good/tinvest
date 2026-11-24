package strategy

import (
	"maps"
	"math"
	"time"

	"tinvest/internal/service/screener/bybitdensity/model"
)

type ZoneTrade struct {
	SignalAt, EntryAt, ExitAt       time.Time
	ZoneLow, ZoneNotional           float64
	EntryPrice, ExitPrice, Quantity float64
	StopPrice, TakePrice            float64
	Fees, NetPnL                    float64
	ExitReason                      string
}

type ZoneCancelReason string

const (
	ZoneCancelInPlay       ZoneCancelReason = "in_play"
	ZoneCancelPreviousHigh ZoneCancelReason = "previous_high"
	ZoneCancelStop         ZoneCancelReason = "stop"
	ZoneCancelVolume       ZoneCancelReason = "volume"
	ZoneCancelReturn       ZoneCancelReason = "returned_to_zone"
	ZoneCancelFeed         ZoneCancelReason = "feed_reset"
	ZoneCancelReference    ZoneCancelReason = "reference_changed"
	ZoneCancelUnfilled     ZoneCancelReason = "unfilled"
	ZoneCancelDistance     ZoneCancelReason = "out_of_range"
	ZoneCancelOtherEntry   ZoneCancelReason = "other_entry"
)

type ZoneResult struct {
	Trades                                                                   []ZoneTrade
	InPlayActiveWindows, InPlayWindows                                       int
	Placed, Cancelled                                                        int
	Pending                                                                  bool
	Cancellations                                                            map[ZoneCancelReason]int
	CancelledBeforeTouch, CancelledBeforeBounce, CancelledDuringConfirmation int
	MaxPendingZones                                                          int
	OpenQuantity                                                             float64
	Realized, Unrealized                                                     float64
	MaxDrawdown                                                              float64
}

type zonePending struct {
	order        ZoneOrder
	at           time.Time
	touched      bool
	lastQuantity float64
	bounceAt     time.Time
}

type zonePosition struct {
	trade      ZoneTrade
	remaining  int64
	stop, take int64
	stopping   bool
	exitQuote  float64
}

// ZoneReplay watches nearby bid zones and models at most one long position.
type ZoneReplay struct {
	inPlay             *InPlayWindow
	instrument         model.Instrument
	policy             ZonePolicy
	notional           float64
	fee                float64
	executor           paperExecutor
	pending            map[int64]*zonePending
	position           *zonePosition
	previousHigh       int64
	lastBid            int64
	bounceConfirmation time.Duration
	peakEquity         float64
	result             ZoneResult
}

func NewZoneReplay(instrument model.Instrument, policy ZonePolicy, notional, fee float64, bounceConfirmation time.Duration) *ZoneReplay {
	return &ZoneReplay{
		instrument: instrument, policy: policy, notional: notional, fee: fee, bounceConfirmation: bounceConfirmation,
		pending: make(map[int64]*zonePending), executor: paperExecutor{instrument: instrument, spent: make(map[priceLevel]float64)},
	}
}

func (r *ZoneReplay) SetPreviousHigh(ticks int64) {
	r.cancelAll(ZoneCancelReference)
	r.previousHigh = ticks
}

func (r *ZoneReplay) SetInPlayWindow(window InPlayWindow) {
	if !window.Active || r.inPlay != nil && window.From.After(r.inPlay.From.Add(InPlayInterval)) {
		r.cancelAll(ZoneCancelInPlay)
	}
	r.inPlay = &window
}

// SetMinZoneQuantity replaces the volume threshold when the reference day changes.
func (r *ZoneReplay) SetMinZoneQuantity(quantity float64) {
	r.cancelAll(ZoneCancelReference)
	r.policy.minQuantity = quantity
}

func (r *ZoneReplay) Apply(event model.Event) {
	if event.Reset {
		r.cancelAll(ZoneCancelFeed)
		clear(r.executor.spent)
	}
	for _, trade := range event.Trades {
		r.onTrade(trade)
	}
	if event.Book != nil && len(event.Book.Bids) > 0 && len(event.Book.Asks) > 0 {
		r.onBook(event.Book)
	}
}

func (r *ZoneReplay) cancelAll(reason ZoneCancelReason) {
	for _, p := range r.pending {
		r.cancel(p, reason)
	}
}

func (r *ZoneReplay) cancel(p *zonePending, reason ZoneCancelReason) {
	delete(r.pending, p.order.LowTicks)
	r.result.Cancelled++
	if r.result.Cancellations == nil {
		r.result.Cancellations = make(map[ZoneCancelReason]int)
	}
	r.result.Cancellations[reason]++
	switch {
	case !p.touched:
		r.result.CancelledBeforeTouch++
	case p.bounceAt.IsZero():
		r.result.CancelledBeforeBounce++
	default:
		r.result.CancelledDuringConfirmation++
	}
}

func (r *ZoneReplay) onBook(book *model.Book) {
	previousBid := r.lastBid
	r.lastBid = book.Bids[0].Ticks
	if r.lastBid <= r.previousHigh {
		r.cancelAll(ZoneCancelPreviousHigh)
	}
	r.executor.ObserveBook(book)
	defer r.updateEquity(r.lastBid)
	if p := r.position; p != nil {
		if !book.At.Before(p.trade.EntryAt) && (p.stopping || r.lastBid <= p.stop) {
			p.stopping = true
			fill := r.executor.Submit(Order{Direction: model.Bid, Steps: p.remaining}, book)
			r.exit(fill.Steps, fill.Quote, book.At, "stop")
		}
		return
	}
	if r.lastBid <= r.previousHigh {
		return
	}
	if r.inPlay != nil && !r.inPlay.allows(book.At) {
		r.cancelAll(ZoneCancelInPlay)
		return
	}
	var nearest *zonePending
	for _, p := range r.pending {
		if r.confirmBounce(book, p, previousBid) && (nearest == nil || p.order.HighTicks > nearest.order.HighTicks) {
			nearest = p
		}
	}
	if nearest != nil {
		r.buyOnBounce(book, nearest)
		return
	}
	r.policy.visitEntries(book, searchDistance, func(order ZoneOrder) bool {
		if order.HighTicks <= r.previousHigh {
			return false
		}
		if r.pending[order.LowTicks] == nil {
			r.pending[order.LowTicks] = &zonePending{order: order, at: book.At, lastQuantity: order.Quantity}
			r.result.Placed++
		}
		return true
	})
	r.result.MaxPendingZones = max(r.result.MaxPendingZones, len(r.pending))
}

func (r *ZoneReplay) confirmBounce(book *model.Book, p *zonePending, previousBid int64) bool {
	switch {
	case r.lastBid <= zoneStopTicks(p.order.LowTicks):
		r.cancel(p, ZoneCancelStop)
		return false
	case zoneDistance(r.lastBid, p.order.HighTicks) > searchDistance:
		r.cancel(p, ZoneCancelDistance)
		return false
	case !r.policy.Keep(book, p.order):
		r.cancel(p, ZoneCancelVolume)
		return false
	}
	if !p.touched {
		p.lastQuantity = bidZoneQuantity(book.Bids, p.order.LowTicks, r.policy.width)
		return false
	}
	if p.bounceAt.IsZero() {
		if previousBid < p.order.LowTicks || previousBid > p.order.HighTicks || r.lastBid <= p.order.HighTicks {
			return false
		}
		p.bounceAt = book.At
	}
	if r.lastBid <= p.order.HighTicks {
		r.cancel(p, ZoneCancelReturn)
		return false
	}
	return book.At.Sub(p.bounceAt) >= r.bounceConfirmation
}

func (r *ZoneReplay) buyOnBounce(book *model.Book, p *zonePending) {
	ask := book.Asks[0].Ticks
	price := r.instrument.Price(ask)
	qty := r.notional / price
	if r.instrument.MaxQty > 0 {
		qty = min(qty, r.instrument.MaxQty)
	}
	fill := r.executor.Submit(Order{Entry: true, Direction: model.Bid, LimitTicks: ask, Steps: quantitySteps(qty, r.instrument.QtyStep)}, book)
	if fill.Steps == 0 {
		r.cancel(p, ZoneCancelUnfilled)
		return
	}
	stop, take := r.policy.ExitTicks(p.order, ask)
	r.position = &zonePosition{remaining: fill.Steps, stop: stop, take: take, trade: ZoneTrade{
		SignalAt: p.at, EntryAt: book.At, ZoneLow: r.instrument.Price(p.order.LowTicks), ZoneNotional: p.order.Notional, EntryPrice: price,
		Quantity: float64(fill.Steps) * r.instrument.QtyStep, StopPrice: r.instrument.Price(stop), TakePrice: r.instrument.Price(take),
	}}
	delete(r.pending, p.order.LowTicks)
	r.cancelAll(ZoneCancelOtherEntry)
}

func (r *ZoneReplay) onTrade(trade model.Trade) {
	if r.inPlay != nil && !r.inPlay.allows(trade.At) {
		r.cancelAll(ZoneCancelInPlay)
	}
	if trade.Ticks <= r.previousHigh {
		r.cancelAll(ZoneCancelPreviousHigh)
	} else {
		for _, p := range r.pending {
			if trade.Ticks <= zoneStopTicks(p.order.LowTicks) {
				r.cancel(p, ZoneCancelStop)
			}
		}
	}
	if !trade.Sell {
		p := r.position
		if p != nil && !p.stopping && trade.Ticks > p.take && trade.At.After(p.trade.EntryAt) {
			steps := min(p.remaining, quantitySteps(trade.Qty, r.instrument.QtyStep))
			if steps > 0 {
				r.exit(steps, float64(steps)*r.instrument.QtyStep*p.trade.TakePrice, trade.At, "take")
				r.updateEquity(r.lastBid)
			}
		}
		return
	}
	low := trade.Ticks - trade.Ticks%r.policy.width
	if p := r.pending[low]; p != nil && !p.touched && !trade.At.Before(p.at) {
		p.touched = true
		p.order.Quantity = p.lastQuantity
	}
}

func (r *ZoneReplay) exit(steps int64, quote float64, at time.Time, reason string) {
	if steps == 0 {
		return
	}
	p := r.position
	p.remaining -= steps
	p.exitQuote += quote
	_, net := r.pnl(p.trade.EntryPrice, float64(steps)*r.instrument.QtyStep, quote)
	r.result.Realized += net
	if p.trade.ExitReason != "" && p.trade.ExitReason != reason {
		p.trade.ExitReason = "take+stop"
	} else {
		p.trade.ExitReason = reason
	}
	if p.remaining != 0 {
		return
	}
	p.trade.ExitAt = at
	p.trade.ExitPrice = p.exitQuote / p.trade.Quantity
	p.trade.Fees, p.trade.NetPnL = r.pnl(p.trade.EntryPrice, p.trade.Quantity, p.exitQuote)
	r.result.Trades = append(r.result.Trades, p.trade)
	r.position = nil
}

func (r *ZoneReplay) pnl(entryPrice, quantity, exitQuote float64) (fee, net float64) {
	entryQuote := entryPrice * quantity
	fee = (entryQuote + exitQuote) * r.fee
	return fee, exitQuote - entryQuote - fee
}

func (r *ZoneReplay) updateEquity(markTicks int64) {
	_, openPnL := r.markPosition(markTicks)
	equity := r.result.Realized + openPnL
	r.peakEquity = math.Max(r.peakEquity, equity)
	r.result.MaxDrawdown = math.Max(r.result.MaxDrawdown, r.peakEquity-equity)
}

func (r *ZoneReplay) Result() ZoneResult {
	result := r.result
	result.Cancellations = maps.Clone(result.Cancellations)
	result.Pending = len(r.pending) > 0
	result.OpenQuantity, result.Unrealized = r.markPosition(r.lastBid)
	return result
}

func (r *ZoneReplay) markPosition(ticks int64) (quantity, net float64) {
	p := r.position
	if p == nil {
		return 0, 0
	}
	quantity = float64(p.remaining) * r.instrument.QtyStep
	_, net = r.pnl(p.trade.EntryPrice, quantity, quantity*r.instrument.Price(ticks))
	return quantity, net
}
