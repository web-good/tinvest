package density

import "time"

type Candle struct {
	Time                   time.Time
	Open, High, Low, Close Price
}

// pinbarEntry tracks a single entry opportunity for one instrument.
type pinbarEntry struct {
	lastCandle, expires time.Time
	origin              Price
	baseLots            int64
	signal              Signal
}

func pinbarPrice(p Price) int64 {
	if !valid(p) || p.Units > 100_000_000 {
		return 0
	}
	return p.Units*1e9 + int64(p.Nano)
}

func (c Candle) matches(level, step, zoneTicks, stopTicks int64) bool {
	low, high, open, closePrice := pinbarPrice(c.Low), pinbarPrice(c.High), pinbarPrice(c.Open), pinbarPrice(c.Close)
	if low == 0 || high <= low || min(open, closePrice) < low || max(open, closePrice) > high ||
		level <= 0 || step <= 0 || zoneTicks < 0 || stopTicks < 0 {
		return false
	}
	distance := low - level
	ticks := distance / step
	if ticks < -stopTicks || ticks > zoneTicks || distance%step != 0 {
		return false
	}
	span, bodyBottom := high-low, min(open, closePrice)
	return 4*(max(open, closePrice)-bodyBottom) <= span &&
		3*(high-bodyBottom) <= span && 5*(bodyBottom-low) >= 3*span
}

func (p *pinbarEntry) Confirm(c Candle, signals []Signal, step Price, zoneTicks, stopTicks int64, now time.Time) {
	closeAt := c.Time.Add(time.Minute)
	if c.Time.IsZero() || !c.Time.Equal(c.Time.Truncate(time.Minute)) || !c.Time.After(p.lastCandle) ||
		now.Before(closeAt) || !now.Before(closeAt.Add(time.Minute)) {
		return
	}
	p.lastCandle = c.Time
	p.Consume()
	for _, s := range signals {
		if s.Quantity <= 0 || !s.Since.Before(closeAt) || (!s.ProfileUntil.IsZero() && !now.Before(s.ProfileUntil)) ||
			!c.matches(pinbarPrice(s.Price), pinbarPrice(step), zoneTicks, stopTicks) {
			continue
		}
		if p.expires.IsZero() || comparePrice(s.Price, p.origin) > 0 {
			p.origin, p.baseLots, p.signal, p.expires = s.Price, s.Quantity, s, closeAt.Add(time.Minute)
		}
	}
}

func (p *pinbarEntry) Update(b Book, candidates []Signal, step Price, minRemaining float64, now time.Time) {
	if _, _, ok := p.Ready(now); !ok {
		p.Consume()
		return
	}
	for _, level := range b.Bids {
		if level.Price == p.signal.Price && float64(level.Quantity) >= float64(p.baseLots)*minRemaining {
			return
		}
	}
	moved, ok := Relocation(b, p.signal.Price, p.baseLots, minRemaining, candidates, now)
	delta := pinbarPrice(moved.Price) - pinbarPrice(p.origin)
	if !ok || delta < -2*pinbarPrice(step) || delta > 2*pinbarPrice(step) {
		p.Consume()
		return
	}
	p.signal = moved
}

func (p *pinbarEntry) Ready(now time.Time) (Signal, time.Time, bool) {
	return p.signal, p.expires, now.Before(p.expires) &&
		(p.signal.ProfileUntil.IsZero() || now.Before(p.signal.ProfileUntil))
}

func (p *pinbarEntry) Consume() { p.expires = time.Time{} }

// PinbarWatch retains the last confirmed levels from the candle's minute even
// if the next minute's book arrives before the closed candle.
type PinbarWatch struct {
	pinbarEntry
	step              Price
	previous, current pinbarSnapshot
}

type pinbarSnapshot struct {
	at      time.Time
	signals []Signal
}

func (p *PinbarWatch) Book(b Book, signals, candidates []Signal, step Price, fraction float64, now time.Time) {
	if !p.current.at.IsZero() && !b.Time.Truncate(time.Minute).Equal(p.current.at.Truncate(time.Minute)) {
		p.previous = p.current
		p.current = pinbarSnapshot{}
	}
	p.current.at = b.Time
	p.current.signals = append(p.current.signals[:0], signals...)
	p.step = step
	p.Update(b, candidates, step, fraction, now)
}

func (p *PinbarWatch) Candle(c Candle, b Book, candidates []Signal, zoneTicks, stopTicks int64, fraction float64, now time.Time) {
	for _, snap := range []pinbarSnapshot{p.current, p.previous} {
		if snap.at.Truncate(time.Minute).Equal(c.Time) {
			p.Confirm(c, snap.signals, p.step, zoneTicks, stopTicks, now)
			p.Update(b, candidates, p.step, fraction, now)
			return
		}
	}
}

// Ready exposes the entry only after a book from the execution minute arrives.
func (p *PinbarWatch) Ready(now time.Time) (Signal, time.Time, bool) {
	signal, until, ok := p.pinbarEntry.Ready(now)
	return signal, until, ok && !p.current.at.Before(until.Add(-time.Minute))
}
