package paper

import (
	"time"
	"tinvest/internal/service/screener/density"
)

func (e *Engine) updateSellLimit(o *Order, b density.Book, now time.Time) {
	limit := o.sellWalls.limit(b.Asks, price(e.instruments[o.InstrumentID].PriceStep), o.Take)
	if limit == o.SellLimit || (o.SellLimit == 0 && price(b.Bids[0].Price) >= o.Take) {
		return
	}
	e.cancelSellLimit(o, now)
	if limit > 0 {
		o.SellLimit = limit
		o.sellLimitActive = now.Add(e.cfg.Latency)
		if e.cfg.Latency == 0 {
			o.sellLimitActive = b.Time
		}
		e.emit("sell_limit_placed", o, now, density.ExitSellWall, 0, 0)
	}
}

func (e *Engine) cancelSellLimit(o *Order, now time.Time) {
	if o.SellLimit > 0 {
		e.emit("sell_limit_canceled", o, now, density.ExitSellWall, 0, 0)
		o.SellLimit = 0
		o.sellLimitActive = time.Time{}
	}
}

// sellWalls tracks asks that first became large after the buy fill.
type sellWalls struct {
	minLots int64
	atEntry map[density.Price]bool
}

func newSellWalls(asks []density.Level, buyLots int64) sellWalls {
	walls := sellWalls{minLots: buyLots/2 + buyLots%2, atEntry: make(map[density.Price]bool)}
	for _, ask := range asks {
		if ask.Quantity >= walls.minLots {
			walls.atEntry[ask.Price] = true
		}
	}
	return walls
}

// limit returns the price two ticks below the closest new sell wall, provided
// it is closer than the existing take. Call with a validated, ascending ask book.
func (w *sellWalls) limit(asks []density.Level, step, take int64) int64 {
	if w.minLots <= 0 || step <= 0 {
		return 0
	}
	for price := range w.atEntry {
		w.atEntry[price] = false
	}
	var limit int64
	for _, ask := range asks {
		if ask.Quantity < w.minLots {
			continue
		}
		if _, existed := w.atEntry[ask.Price]; existed {
			w.atEntry[ask.Price] = true
			continue
		}
		target := price(ask.Price) - 2*step
		if limit == 0 && target > 0 && target < take && target%step == 0 {
			limit = target
		}
	}
	for price, seen := range w.atEntry {
		if !seen {
			delete(w.atEntry, price)
		}
	}
	return limit
}

func (o *Order) takeOrder() (int64, time.Time, string) {
	if o.SellLimit > 0 {
		return o.SellLimit, o.sellLimitActive, density.ExitSellWall
	}
	return o.Take, o.TakeActive, "take_profit"
}
