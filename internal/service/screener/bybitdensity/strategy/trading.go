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
	limit           int64
}

type position struct {
	setup
	entry                   float64
	remaining, stop, target int64 // remaining is measured in quantity steps.
	forced                  string
}

// prepareEntry places one simulated limit per confirmed density episode.
func (s *Strategy) prepareEntry() {
	var best *episode
	for _, ep := range s.episodes {
		sig := ep.signal
		if ep.confirmed.IsZero() || ep.entryPlaced || (sig.Side == model.Ask && s.instrument.Key.Market != model.Linear) {
			continue
		}
		if best == nil || preferred(sig, best.signal) {
			best = ep
		}
	}
	if best == nil {
		return
	}
	best.entryPlaced = true
	sig := best.signal
	limit := sig.Ticks + direction(sig.Side)*s.cfg.EntryTicks
	stop, target := s.entryPrices(sig.Side, sig.Ticks, limit)
	if s.validPrice(limit) && s.validPrice(stop) && s.validPrice(target) {
		s.pending = &setup{side: sig.Side, origin: sig.Ticks, watched: sig.Ticks, baseQty: sig.Qty, limit: limit}
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

// follow keeps tracking support for an open position if the wall is relocated.
func (s *Strategy) follow(p *setup) bool {
	if quantityAt(levelsOn(s.book, p.side), p.side, p.watched) > 0 {
		return true
	}
	var best model.Signal
	for _, sig := range s.candidates {
		if sig.Side != p.side || sig.Qty < .5*p.baseQty {
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

func direction(side model.Side) int64 {
	if side == model.Ask {
		return -1
	}
	return 1
}

func (s *Strategy) entryPrices(side model.Side, origin, ticks int64) (stop, target int64) {
	sign := direction(side)
	stop = origin - sign*s.cfg.StopTicks
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
	if quantityAt(levelsOn(s.book, p.side), p.side, p.origin) < .5*p.baseQty {
		s.pending = nil
		return Order{}, false
	}
	level := levelsOn(s.book, oppositeSide(p.side))[0]
	if direction(p.side)*(level.Ticks-p.limit) > 0 {
		return Order{}, false
	}
	price := s.instrument.Price(max(p.limit, level.Ticks))
	qty := s.cfg.PaperNotional / price
	if s.instrument.MaxQty > 0 {
		qty = min(qty, s.instrument.MaxQty)
	}
	steps := quantitySteps(qty, s.instrument.QtyStep)
	qty = float64(steps) * s.instrument.QtyStep
	if steps <= 0 || !meetsMinimum(s.instrument, qty, price) {
		return Order{}, false
	}
	return Order{Entry: true, Reason: "entry", Direction: p.side, Steps: steps, LimitTicks: p.limit}, true
}

func (s *Strategy) exitIntent(p *position) (Order, bool) {
	best := levelsOn(s.book, p.side)[0].Ticks
	if p.forced == "" {
		switch {
		case (p.side == model.Bid && best <= p.stop) || (p.side == model.Ask && best >= p.stop):
			p.forced = "stop"
		case !s.follow(&p.setup):
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
		s.pending = nil
		stop, target := s.entryPrices(pending.side, pending.origin, s.instrument.Ticks(f.Price))
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
