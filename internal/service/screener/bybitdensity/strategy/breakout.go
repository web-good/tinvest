package strategy

import (
	"math"
	"time"

	"tinvest/internal/service/screener/bybitdensity/model"
)

const (
	SOLBreakoutSymbol = "SOLUSDT"
	breakoutHold      = 10 * time.Second
	crossingWindow    = 500 * time.Millisecond
	imbalanceLevels   = 3
	minImbalance      = .33
)

type askEpisode struct {
	exchangeStart, receiptStart time.Time
	lastBook                    *model.Book
}

type askWall struct {
	ticks                      int64
	initialQty, required, buys float64
	lastQty                    float64
	armedAt, crossedAt         time.Time
	crossedReceived            time.Time
	queuedBuys                 []model.Trade
}

type breakoutSignal struct {
	ticks               int64
	quantity, imbalance float64
	expiresAt           time.Time
}

type breakout struct {
	simulations  [2]*Strategy
	book         *model.Book
	tradeReady   bool
	needSnapshot bool
	episodes     map[int64]askEpisode
	trades       tradeIDs
	wall         *askWall
	pending      *breakoutSignal
}

func newBreakout(instrument model.Instrument, cfg Config) *breakout {
	cfg.SOLBreakout = false
	cfg.CandleEntries = false
	b := &breakout{needSnapshot: true, episodes: make(map[int64]askEpisode)}
	for i, variant := range [...]model.Variant{model.Breakout, model.BreakoutImbalance} {
		s := New(instrument, cfg)
		s.variant = variant
		b.simulations[i] = s
	}
	return b
}

func (b *breakout) clearBook() {
	b.book, b.wall, b.pending = nil, nil, nil
	clear(b.episodes)
}

func (b *breakout) apply(e model.Event, now time.Time) []model.Fill {
	s := b.simulations[0]
	if e.Reset {
		b.tradeReady = false
		b.needSnapshot = true
		b.clearBook()
	}
	b.tradeReady = b.tradeReady || e.TradeReady
	if b.book != nil && !s.fresh(b.book, now) {
		b.clearBook()
	}
	var fills []model.Fill
	if book := e.Book; book != nil && (!b.needSnapshot || book.Snapshot) {
		if !s.fresh(book, now) || len(book.Bids) == 0 || len(book.Asks) == 0 || book.Bids[0].Ticks >= book.Asks[0].Ticks {
			b.clearBook()
			return nil
		}
		if book.Snapshot || b.book != nil && book.At.Sub(b.book.At) > s.cfg.Freshness {
			b.clearBook()
		}
		b.needSnapshot = false
		b.book = book
		fills = b.execute(book, now)
		if b.tradeReady {
			b.observeBook(book, now)
			if b.wall != nil {
				b.wall.countQueuedBuys(book.At)
			}
		}
	}
	if b.wall != nil {
		b.observeTrades(e.Trades, now)
		b.confirm(now)
	}
	return fills
}

func (b *breakout) observeBook(book *model.Book, now time.Time) {
	s := b.simulations[0]
	asks := book.Asks
	best := asks[0]
	if len(asks) >= 6 {
		for i, level := range asks {
			if level.Ticks-best.Ticks > maxDistanceTicks {
				break
			}
			value := s.instrument.Price(level.Ticks) * level.Qty
			if value < s.cfg.MinNotional || value < s.cfg.MinMultiple*medianNeighbors(s.instrument, asks, i) {
				continue
			}
			episode, ok := b.episodes[level.Ticks]
			if !ok {
				episode = askEpisode{exchangeStart: book.At, receiptStart: book.Received}
			}
			episode.lastBook = book
			b.episodes[level.Ticks] = episode
		}
	}
	for ticks, episode := range b.episodes {
		if episode.lastBook != book {
			delete(b.episodes, ticks)
		}
	}
	if w := b.wall; w != nil {
		switch {
		case best.Ticks < w.ticks, !w.crossedAt.IsZero() && (best.Ticks == w.ticks || now.Sub(w.crossedReceived) > crossingWindow):
			b.wall = nil
		case best.Ticks == w.ticks:
			w.required += max(0, best.Qty-w.lastQty)
			w.lastQty = best.Qty
		case w.crossedAt.IsZero():
			w.crossedAt, w.crossedReceived = book.At, book.Received
		}
	}
	if b.wall == nil {
		if episode, ok := b.episodes[best.Ticks]; ok &&
			min(book.At.Sub(episode.exchangeStart), book.Received.Sub(episode.receiptStart)) >= s.cfg.Persistence {
			b.wall = &askWall{ticks: best.Ticks, initialQty: best.Qty, required: best.Qty, lastQty: best.Qty, armedAt: book.At}
		}
	}
}

func (b *breakout) observeTrades(trades []model.Trade, now time.Time) {
	w := b.wall
	freshness := b.simulations[0].cfg.Freshness
	for _, trade := range trades {
		if trade.TakerSide == model.Bid && !trade.OffBook && trade.Ticks == w.ticks &&
			trade.Qty > 0 && !math.IsInf(trade.Qty, 0) &&
			!trade.At.Before(w.armedAt) && now.Sub(trade.At).Abs() <= freshness &&
			(w.crossedAt.IsZero() || !trade.At.After(w.crossedAt)) && b.trades.add(trade.ID) {
			if trade.At.After(b.book.At) {
				w.queuedBuys = append(w.queuedBuys, trade)
			} else {
				w.buys += trade.Qty
			}
		}
	}
}

func (w *askWall) countQueuedBuys(cutoff time.Time) {
	if !w.crossedAt.IsZero() {
		cutoff = w.crossedAt
	}
	// The trade topic can run ahead of the book topic. Count a buy
	// only after the book timestamp has reached it.
	remaining := w.queuedBuys[:0]
	for _, trade := range w.queuedBuys {
		if !trade.At.After(cutoff) {
			w.buys += trade.Qty
		} else if w.crossedAt.IsZero() {
			remaining = append(remaining, trade)
		}
	}
	clear(w.queuedBuys[len(remaining):])
	w.queuedBuys = remaining
}

func (b *breakout) confirm(now time.Time) {
	w := b.wall
	if w.crossedAt.IsZero() {
		return
	}
	if now.Sub(w.crossedReceived) > crossingWindow {
		b.wall = nil
		return
	}
	book := b.book
	if w.buys+.5*b.simulations[0].instrument.QtyStep < w.required || book.Bids[0].Ticks < w.ticks || len(book.Bids) < imbalanceLevels || len(book.Asks) < imbalanceLevels {
		return
	}
	var bids, asks float64
	for i := range imbalanceLevels {
		bids += book.Bids[i].Qty
		asks += book.Asks[i].Qty
	}
	if b.simulations[0].position == nil && b.simulations[1].position == nil {
		b.pending = &breakoutSignal{ticks: w.ticks, quantity: w.initialQty, imbalance: (bids - asks) / (bids + asks), expiresAt: w.crossedReceived.Add(crossingWindow)}
	}
	b.wall = nil
}

func (b *breakout) execute(book *model.Book, now time.Time) []model.Fill {
	signal := b.pending
	b.pending = nil
	var fills []model.Fill
	for _, s := range b.simulations {
		s.book = book
		s.executor.ObserveBook(book)
		if s.position == nil && signal != nil && !now.After(signal.expiresAt) &&
			book.Bids[0].Ticks >= signal.ticks &&
			(s.variant == model.Breakout || signal.imbalance > minImbalance) {
			limit := book.Asks[0].Ticks
			if s.validPrice(limit) {
				s.pending = &setup{side: model.Bid, origin: signal.ticks,
					baseQty: signal.quantity, limit: limit, imbalance: &signal.imbalance}
			}
		}
		fills = append(fills, s.execute(now)...)
		s.pending = nil // An IOC entry is attempted once on the next book.
	}
	return fills
}

func breakoutExit(p *position, best int64, now time.Time) (Order, bool) {
	if p.forced == "" {
		switch {
		case best < p.origin:
			p.forced = "breakout_failed"
		case !now.Before(p.entry.At.Add(breakoutHold)):
			p.forced = "timeout"
		}
	}
	return Order{Direction: p.side, Reason: p.forced, Steps: p.remaining}, p.forced != ""
}
