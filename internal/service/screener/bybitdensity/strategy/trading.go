package strategy

import (
	"math"
	"time"

	"tinvest/internal/service/screener/bybitdensity/model"
)

type setup struct {
	side            model.Side
	origin, watched int64
	baseQty         float64
	end             time.Time
}

type position struct {
	setup
	entry                   float64
	remaining, stop, target int64 // remaining is measured in quantity steps.
	forced                  string
}

// onCandle evaluates a closure exactly once, using only a currently uninterrupted episode.
func (s *Strategy) onCandle(c model.Candle, now time.Time) {
	if !c.Start.After(s.lastCandle) || !c.Start.Truncate(time.Minute).Equal(c.Start) || !c.End.Equal(c.Start.Add(time.Minute)) || now.Before(c.End) {
		return
	}
	s.lastCandle = c.Start
	if s.position != nil || s.outstanding != nil || s.book == nil || !now.Before(c.End.Add(time.Minute)) {
		return
	}
	var best model.Signal
	for _, ep := range s.episodes {
		sig := ep.signal
		if ep.confirmed.IsZero() || !ep.confirmed.Before(c.End) || (sig.Side == model.Ask && s.instrument.Key.Market != model.Linear) || !s.pinbar(c, sig) {
			continue
		}
		if ep.lastClose.Equal(c.End) {
			sig = ep.atClose
		}
		if best.Ticks == 0 || preferred(sig, best) {
			best = sig
		}
	}
	if best.Ticks != 0 {
		s.pending = &setup{side: best.Side, origin: best.Ticks, watched: best.Ticks, baseQty: best.Qty, end: c.End}
	}
}

func preferred(a, b model.Signal) bool {
	if a.Distance != b.Distance {
		return a.Distance < b.Distance
	}
	if a.Notional != b.Notional {
		return a.Notional > b.Notional
	}
	return a.Side == model.Bid && b.Side == model.Ask
}

func (s *Strategy) pinbar(c model.Candle, signal model.Signal) bool {
	bottom, top := min(c.Open, c.Close), max(c.Open, c.Close)
	if c.Low <= 0 || c.High <= c.Low || bottom < c.Low || top > c.High {
		return false
	}
	span := float64(c.High - c.Low)
	if float64(top-bottom) > span/4 {
		return false
	}
	tail, opposite := float64(bottom-c.Low), float64(c.High-bottom)
	offset := c.Low - signal.Ticks
	if signal.Side == model.Ask {
		tail, opposite = float64(c.High-top), float64(top-c.Low)
		offset = signal.Ticks - c.High
	}
	return offset >= -s.cfg.StopTicks && offset <= s.cfg.EntryZoneTicks && opposite <= span/3 && tail >= .6*span
}

// follow maintains the watched level separately from the fixed origin of the pinbar.
func (s *Strategy) follow(p *setup, beforeEntry bool) bool {
	qty := quantityAt(levelsOn(s.book, p.side), p.side, p.watched)
	if qty > 0 && (!beforeEntry || qty >= .5*p.baseQty) {
		return true
	}
	var best model.Signal
	for _, sig := range s.candidates {
		if sig.Side != p.side || sig.Qty < .5*p.baseQty || (beforeEntry && absTicks(sig.Ticks-p.origin) > 2) {
			continue
		}
		if best.Ticks == 0 || absTicks(sig.Ticks-p.watched) < absTicks(best.Ticks-p.watched) ||
			(absTicks(sig.Ticks-p.watched) == absTicks(best.Ticks-p.watched) && preferred(sig, best)) {
			best = sig
		}
	}
	if best.Ticks == 0 {
		return false
	}
	p.watched = best.Ticks
	return true
}

func (s *Strategy) entryPrices(p *setup, ticks int64) (stop, target int64) {
	sign := int64(1)
	if p.side == model.Ask {
		sign = -1
	}
	stop = p.origin - sign*s.cfg.StopTicks
	risk := sign * (ticks - stop)
	if risk <= 0 {
		return stop, 0
	}
	gain := max(float64(risk)*s.cfg.RewardRisk, float64(ticks)*s.cfg.MinTarget)
	return stop, ticks + sign*int64(math.Ceil(gain))
}

func (s *Strategy) validPrice(ticks int64) bool {
	v := s.instrument.Price(ticks)
	return ticks > 0 && v >= math.Nextafter(s.instrument.MinPrice, 0) && (s.instrument.MaxPrice == 0 || v <= math.Nextafter(s.instrument.MaxPrice, math.Inf(1)))
}

func (s *Strategy) entryIntent(p *setup) (Order, bool) {
	if !s.follow(p, true) {
		s.pending = nil
		return Order{}, false
	}
	if s.book.At.Before(p.end) {
		return Order{}, false
	}
	// One IOC attempt per setup, including when available quantity misses the minimum.
	s.pending = nil
	if !s.book.At.Before(p.end.Add(time.Minute)) {
		return Order{}, false
	}
	level := levelsOn(s.book, oppositeSide(p.side))[0]
	stop, target := s.entryPrices(p, level.Ticks)
	if !s.validPrice(level.Ticks) || !s.validPrice(stop) || !s.validPrice(target) {
		return Order{}, false
	}
	price := s.instrument.Price(level.Ticks)
	qty := s.cfg.PaperNotional / price
	if s.instrument.MaxQty > 0 {
		qty = min(qty, s.instrument.MaxQty)
	}
	steps := quantitySteps(qty, s.instrument.QtyStep)
	qty = float64(steps) * s.instrument.QtyStep
	if steps <= 0 || !meetsMinimum(s.instrument, qty, price) {
		return Order{}, false
	}
	return Order{Entry: true, Reason: "entry", Direction: p.side, Steps: steps, LimitTicks: level.Ticks}, true
}

func (s *Strategy) exitIntent(p *position) (Order, bool) {
	best := levelsOn(s.book, p.side)[0].Ticks
	if p.forced == "" {
		switch {
		case (p.side == model.Bid && best <= p.stop) || (p.side == model.Ask && best >= p.stop):
			p.forced = "stop"
		case !s.follow(&p.setup, false):
			p.forced = "density_lost"
		}
	}
	if p.forced != "" {
		return Order{Direction: p.side, Reason: p.forced, Steps: p.remaining}, true
	}
	reached := (p.side == model.Bid && best >= p.target) || (p.side == model.Ask && best <= p.target)
	return Order{Direction: p.side, Reason: "take", Steps: p.remaining, LimitTicks: p.target}, reached
}

func oppositeSide(side model.Side) model.Side {
	if side == model.Bid {
		return model.Ask
	}
	return model.Bid
}

func (s *Strategy) execute(now time.Time) []model.Fill {
	var order Order
	var ok bool
	pending := s.pending
	switch {
	case s.position != nil:
		// Keep watching risk while the executor reconciles an earlier IOC.
		order, ok = s.exitIntent(s.position)
	case pending != nil:
		order, ok = s.entryIntent(pending)
	}
	if !ok || s.outstanding != nil {
		return nil
	}
	s.orderID++
	order.Key, order.ID = s.instrument.Key, s.orderID
	s.outstanding = &submittedOrder{order: order, setup: pending}
	executed := s.executor.Submit(order, s.book)
	if fill, filled := s.ConfirmExecution(executed, now); filled {
		return []model.Fill{fill}
	}
	return nil
}

// ConfirmExecution reconciles the final cumulative result of one IOC. Repeated or
// mismatched IDs are ignored. Serialize this callback with Apply and Expire.
func (s *Strategy) ConfirmExecution(m Execution, now time.Time) (model.Fill, bool) {
	submitted := s.outstanding
	if submitted == nil || m.OrderID != submitted.order.ID || m.Steps < 0 || m.Steps > submitted.order.Steps || math.IsNaN(m.Quote) || math.IsInf(m.Quote, 0) || (m.Steps > 0 && m.Quote <= 0) || (m.Steps == 0 && m.Quote != 0) {
		return model.Fill{}, false
	}
	s.outstanding = nil
	if m.Steps == 0 {
		return model.Fill{}, false
	}
	order := submitted.order
	pending := submitted.setup
	qty := float64(m.Steps) * s.instrument.QtyStep
	feeRate := s.cfg.SpotFee
	if s.instrument.Key.Market == model.Linear {
		feeRate = s.cfg.LinearFee
	}
	f := model.Fill{Key: s.instrument.Key, Side: order.Direction, Entry: order.Entry, Reason: order.Reason, Price: m.Quote / qty, Qty: qty, Fee: m.Quote * feeRate, At: now}
	if order.Entry {
		stop, target := s.entryPrices(pending, s.instrument.Ticks(f.Price))
		s.position = &position{setup: *pending, entry: f.Price, remaining: m.Steps, stop: stop, target: target}
		return f, true
	}
	p := s.position
	cost := p.entry * qty
	gross := m.Quote - cost
	if p.side == model.Ask {
		gross = -gross
	}
	f.NetPnL = gross - f.Fee - cost*feeRate
	f.NetReturn = f.NetPnL / cost
	p.remaining -= m.Steps
	f.Partial = p.remaining > 0
	if p.remaining == 0 {
		s.position = nil
	}
	return f, true
}
