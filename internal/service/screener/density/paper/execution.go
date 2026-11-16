package paper

import "time"

func (e *Engine) sell(o *Order, lots, at int64, reason string, now time.Time) {
	if lots <= 0 {
		return
	}
	value := rub(at) * float64(lots) * float64(o.Lot)
	fee := value * e.cfg.Commission
	entryCost := rub(o.Entry) * float64(lots) * float64(o.Lot)
	gross := value - entryCost
	// Realized PnL allocates entry fees only to shares actually sold.
	net := gross - fee - o.EntryFees*float64(lots)/float64(o.FilledLots)
	e.cash += value - fee
	e.fees += fee
	e.realized += net
	o.ExitFees += fee
	o.GrossPnL += gross
	o.NetPnL += net
	o.OpenLots -= lots
	if o.OpenLots == 0 {
		o.State = "closed"
		o.Closed = now
		o.ExitReason = reason
		e.closed++
		if o.NetPnL > 0 {
			e.wins++
		}
		e.cooldown[o.InstrumentID] = now.Add(e.cfg.Cooldown)
	}
	e.emit("exit_filled", o, now, reason, lots, rub(at))
}
func (e *Engine) executeExit(o *Order, now time.Time) {
	if e.err != nil || o.State != "exiting" || now.Before(o.ExitAt) || !e.fresh(o.InstrumentID, now) {
		return
	}
	b := e.books[o.InstrumentID]
	// Require a new snapshot at/after activation; never sweep the trigger snapshot.
	if b.book.Time.Before(o.ExitAt) || !b.book.Time.After(b.consumed) {
		return
	}
	b.consumed = b.book.Time
	e.books[o.InstrumentID] = b
	for _, l := range b.book.Bids {
		e.sell(o, min(l.Quantity, o.OpenLots), price(l.Price), o.ExitReason, now)
		if e.err != nil || o.OpenLots == 0 {
			break
		}
	}
}
