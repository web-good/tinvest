// Package bounce tracks consecutive closed minute candles against a resting bid.
package bounce

import (
	"math"
	"time"
)

const (
	maxBreachTicks      = 5
	minRetainedFraction = 0.30
)

// Retains reports whether the original price still displays at least 30% of its initial volume.
func Retains(initial, current float64) bool {
	return initial > 0 && current > 0 && current >= math.Nextafter(initial*minRetainedFraction, 0)
}

type level[T any] struct {
	signal                 T
	observed, touch, entry time.Time
}

// Watch follows confirmed levels on one instrument. Prices are exchange ticks.
type Watch[T any] struct {
	levels     map[int64]level[T]
	lastBook   time.Time
	lastCandle time.Time
}

// Book discards any setup whose original level fails the current book check.
func (w *Watch[T]) Book(at time.Time, retained func(T) bool) {
	w.lastBook = at
	for tick, l := range w.levels {
		if !retained(l.signal) {
			delete(w.levels, tick)
		}
	}
}

// Track starts observing a confirmed level at the time of the current book.
func (w *Watch[T]) Track(tick int64, signal T) {
	if tick <= 0 || w.lastBook.IsZero() {
		return
	}
	if _, ok := w.levels[tick]; ok {
		return
	}
	if w.levels == nil {
		w.levels = make(map[int64]level[T])
	}
	w.levels[tick] = level[T]{signal: signal, observed: w.lastBook}
}

// Candle consumes an exchange-closed M1 bar. The entry minute starts at the
// close of the bar immediately following a valid touch.
func (w *Watch[T]) Candle(start time.Time, low, closePrice int64, now time.Time) {
	end := start.Add(time.Minute)
	if start.IsZero() || !start.Equal(start.Truncate(time.Minute)) || !start.After(w.lastCandle) ||
		now.Before(end) || !now.Before(end.Add(2*time.Minute)) || low <= 0 || closePrice < low {
		return
	}
	w.lastCandle = start
	for tick, l := range w.levels {
		if start.Before(l.observed) || !l.entry.IsZero() && start.Before(l.entry) {
			continue
		}
		if !l.touch.IsZero() && start.Equal(l.touch.Add(time.Minute)) && closePrice > tick {
			l.touch, l.entry = time.Time{}, end
		} else {
			l.touch, l.entry = time.Time{}, time.Time{}
			if low <= tick && tick-low <= maxBreachTicks {
				l.touch = start
			}
		}
		w.levels[tick] = l
	}
}

func (w *Watch[T]) ready(now time.Time) (int64, level[T]) {
	var tick int64
	var best level[T]
	for price, l := range w.levels {
		if price > tick && !l.entry.IsZero() && !w.lastBook.Before(l.entry) && w.lastBook.Before(l.entry.Add(time.Minute)) &&
			!now.Before(l.entry) && now.Before(l.entry.Add(time.Minute)) {
			tick, best = price, l
		}
	}
	return tick, best
}

// Ready returns the highest eligible bid level and the end of the entry minute.
func (w *Watch[T]) Ready(now time.Time) (T, time.Time, bool) {
	tick, best := w.ready(now)
	if tick == 0 {
		return best.signal, time.Time{}, false
	}
	return best.signal, best.entry.Add(time.Minute), true
}

// Consume removes the chosen setup once its entry attempt is committed.
func (w *Watch[T]) Consume(now time.Time) {
	tick, _ := w.ready(now)
	delete(w.levels, tick)
}
