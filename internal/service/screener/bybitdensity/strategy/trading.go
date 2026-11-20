package strategy

import (
	"math"
	"time"

	"tinvest/internal/service/screener/bounce"
	"tinvest/internal/service/screener/bybitdensity/model"
	"tinvest/internal/service/screener/density"
)

type setup struct {
	side            model.Side
	origin, watched int64
	baseQty         float64
	averageVolume   *float64
	limit           int64
}

type position struct {
	setup
	entry                   model.Fill
	exitQuote, netPnL       float64
	exitReasons             []string
	remaining, stop, target int64 // remaining is measured in quantity steps.
	forced                  string
}

// prepareEntry places one simulated buy limit per confirmed bid density episode.
func (s *Strategy) prepareEntry() {
	var best *episode
	for _, ep := range s.episodes {
		sig := ep.signal
		if ep.confirmed.IsZero() || ep.entryPlaced || sig.Side != model.Bid {
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
	offset := s.cfg.EntryTicks
	if s.cfg.FixedLimitEntry {
		offset = 1
	}
	limit := sig.Ticks + direction(sig.Side)*offset
	stop, target := s.entryPrices(sig.Side, sig.Ticks, limit)
	if s.validPrice(limit) && s.validPrice(stop) && s.validPrice(target) {
		s.pending = &setup{side: sig.Side, origin: sig.Ticks, watched: sig.Ticks, baseQty: sig.Qty, limit: limit}
		if sig.HasAverageVolume {
			s.pending.averageVolume = &sig.AverageVolume
		}
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
	if s.cfg.FixedLimitEntry {
		stop = percentStop(side, float64(ticks))
	}
	return stop, s.targetPrice(side, float64(ticks), stop)
}

func percentStop(side model.Side, entryTicks float64) int64 {
	if side == model.Ask {
		return int64(math.Ceil(math.Nextafter(entryTicks*1.002, 0)))
	}
	return int64(math.Floor(math.Nextafter(entryTicks*0.998, math.Inf(1))))
}

func (s *Strategy) targetPrice(side model.Side, ticks float64, stop int64) int64 {
	sign := direction(side)
	risk := float64(sign) * (ticks - float64(stop))
	if risk <= 0 {
		return 0
	}
	gain := max(risk*s.cfg.RewardRisk, ticks*s.cfg.MinTarget)
	return sign * int64(math.Ceil(float64(sign)*ticks+gain))
}

func (s *Strategy) validPrice(ticks int64) bool {
	v := s.instrument.Price(ticks)
	return ticks > 0 && v >= math.Nextafter(s.instrument.MinPrice, 0) && (s.instrument.MaxPrice == 0 || v <= math.Nextafter(s.instrument.MaxPrice, math.Inf(1)))
}

func (s *Strategy) entryIntent(p *setup, now time.Time) (Order, bool) {
	qtyAtDensity := quantityAt(levelsOn(s.book, p.side), p.side, p.origin)
	retained := bounce.Retains(p.baseQty, qtyAtDensity)
	if s.cfg.FixedLimitEntry {
		retained = density.RetainsEntryDensity(p.baseQty, qtyAtDensity)
	} else if !s.cfg.CandleEntries {
		retained = qtyAtDensity >= .5*p.baseQty && s.volumeAllows(qtyAtDensity, now)
	}
	if !retained {
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
		order, ok = s.entryIntent(pending, now)
	}
	if s.cfg.CandleEntries && pending != nil {
		s.pending = nil
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
		if s.cfg.FixedLimitEntry {
			entryTicks := f.Price / s.instrument.TickSize
			stop = percentStop(pending.side, entryTicks)
			target = s.targetPrice(pending.side, entryTicks, stop)
		}
		s.position = &position{setup: *pending, entry: f, remaining: m.Steps, stop: stop, target: target}
		return f, true
	}
	p := s.position
	cost := p.entry.Price * qty
	gross := m.Quote - cost
	if p.side == model.Ask {
		gross = -gross
	}
	f.NetPnL = gross - f.Fee - cost*feeRate
	f.NetReturn = f.NetPnL / cost
	p.exitQuote += m.Quote
	p.netPnL += f.NetPnL
	if n := len(p.exitReasons); n == 0 || p.exitReasons[n-1] != f.Reason {
		p.exitReasons = append(p.exitReasons, f.Reason)
	}
	p.remaining -= m.Steps
	f.Partial = p.remaining > 0
	if !f.Partial {
		f.Closed = &model.TradeSummary{
			EntryAt: p.entry.At, EntryPrice: p.entry.Price, ExitPrice: p.exitQuote / p.entry.Qty,
			DensityPrice: s.instrument.Price(p.origin), DensityQty: p.baseQty, AverageVolume: p.averageVolume,
			StopPrice: s.instrument.Price(p.stop), TakePrice: s.instrument.Price(p.target),
			NetPnL: p.netPnL, NetReturn: p.netPnL / (p.entry.Price * p.entry.Qty), ExitReasons: p.exitReasons,
		}
		s.position = nil
	}
	return f, true
}
