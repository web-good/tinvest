package main

import (
	"fmt"
	"time"
	"tinvest/internal/service/screener/bybitdensity/feed"
)

const (
	minBreakoutBodyFraction = 0.65
	stopBuffer              = 0.0001 // 0.01% below the pre-breakout low.
	maxStopDistance         = 0.002  // Fraction of the entry price.
)

type bar = feed.SpotBar
type peak struct {
	index int
	price float64
}
type line struct {
	anchor peak
	slope  float64
	born   int
}

func (l line) level(i int) float64 { return l.anchor.price + l.slope*float64(i-l.anchor.index) }

type detector struct {
	width    int
	previous *peak
	active   *line
}

func pivot(b []bar, width int, high bool) bool {
	i := len(b) - 1 - width
	if i < width {
		return false
	}
	for j := i - width; j <= i+width; j++ {
		if j == i {
			continue
		}
		if high && b[j].High >= b[i].High || !high && b[j].Low <= b[i].Low {
			return false
		}
	}
	return true
}
func (d *detector) step(b []bar) bool {
	i := len(b) - 1
	hit := false
	if l := d.active; l != nil {
		c := b[i]
		level := l.level(i)
		crossed := c.Close > level && b[i-1].Close <= l.level(i-1)
		if crossed {
			hit = c.Open < level && c.Close-c.Open >= minBreakoutBodyFraction*(c.High-c.Low)
		}
		if crossed || i-l.born >= 200 {
			d.active = nil
		}
	}
	if !pivot(b, d.width, true) {
		return hit
	}
	k := i - d.width
	p := peak{k, b[k].High}
	old := d.previous
	d.previous = &p
	if old == nil || p.price >= old.price || k-old.index > 400 {
		return hit
	}
	l := line{anchor: p, slope: (p.price - old.price) / float64(k-old.index), born: i}
	for j := old.index + 1; j <= i; j++ {
		if b[j].Close > l.level(j) {
			return hit
		}
	}
	d.active = &l
	return hit
}

type trade struct {
	signal, opened, closed         time.Time
	scale, reason                  string
	stop, entry, target, exit, net float64
}
type engine struct {
	bars                   []bar
	fast, slow             detector
	pending, position      *trade
	done                   []trade
	entryFloor             float64
	fee, slip, takeProfitR float64
}

func newEngine(fee, slip, takeProfitR float64) *engine {
	return &engine{fast: detector{width: 3}, slow: detector{width: 15}, fee: fee, slip: slip, takeProfitR: takeProfitR}
}
func (e *engine) step(c bar, interval time.Duration, entryAllowed, signalAllowed bool) []string {
	var events []string
	if t := e.pending; t != nil {
		e.pending = nil
		entry := c.Open * (1 + e.slip)
		if !entryAllowed {
			events = append(events, "Вход отменён: монета вне inplay.")
		} else if c.Open < e.entryFloor {
			events = append(events, "Вход отменён: открытие ниже максимума вчерашнего дня.")
		} else if t.stop < entry*(1-maxStopDistance) {
			events = append(events, "Вход отменён: расстояние до стопа больше 0,2%.")
		} else if c.Open > t.stop {
			t.entry = entry
			t.target = entry + e.takeProfitR*(entry-t.stop)
			t.opened = c.End.Add(-interval)
			e.position = t
			events = append(events, fmt.Sprintf("Бумажный вход %.8g · стоп %.8g · цель %.8g", t.entry, t.stop, t.target))
		} else {
			events = append(events, "Вход отменён: открытие ниже стопа или на нём.")
		}
	}
	if t := e.position; t != nil {
		switch {
		case c.Open <= t.stop:
			t.exit = c.Open
			t.reason = "стоп с гэпом"
		case c.Open >= t.target:
			t.exit = c.Open
			t.reason = "цель на открытии"
		case c.Low <= t.stop:
			t.exit = t.stop
			t.reason = "стоп"
		case c.High >= t.target:
			t.exit = t.target
			t.reason = "цель"
		}
		if t.exit > 0 {
			t.exit *= 1 - e.slip
			t.closed = c.End
			t.net = t.exit*(1-e.fee)/(t.entry*(1+e.fee)) - 1
			e.done = append(e.done, *t)
			e.position = nil
			events = append(events, fmt.Sprintf("Выход %.8g · %s · после комиссии %+.2f%%", t.exit, t.reason, 100*t.net))
		}
	}
	e.bars = append(e.bars, c)
	b := e.bars
	i := len(b) - 1
	fast, slow := e.fast.step(b), e.slow.step(b)
	if !signalAllowed || (!fast && !slow) || e.position != nil || i < 1 {
		return events
	}
	stop := b[i-1].Low * (1 - stopBuffer)
	if stop <= 0 || stop >= c.Close {
		return events
	}
	scale := "короткая"
	if slow {
		scale = "длинная"
		if fast {
			scale = "обе"
		}
	}
	e.pending = &trade{signal: c.End, scale: scale, stop: stop}
	return append(events, fmt.Sprintf("Пробой · %s · закрытие %.8g · стоп %.8g · бумажный вход на следующем открытии", scale, c.Close, stop))
}
