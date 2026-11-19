// Package volume defines the historical volume window used by density strategies.
package volume

import (
	"math"
	"time"
)

var moscow = time.FixedZone("MSK", 3*60*60)

const CandleInterval = 5 * time.Minute

type Average struct {
	Quantity float64
	Until    time.Time
}

func (a Average) Allows(quantity float64, now time.Time) bool {
	return now.Before(a.Until) && a.Quantity >= 0 && quantity >= a.Quantity
}

// PreviousWeekday returns the half-open Moscow session and the next change of reference day.
func PreviousWeekday(now time.Time) (from, to, until time.Time) {
	local := now.In(moscow)
	day := time.Date(local.Year(), local.Month(), local.Day(), 0, 0, 0, 0, moscow).AddDate(0, 0, -1)
	for weekend(day) {
		day = day.AddDate(0, 0, -1)
	}
	next := day.AddDate(0, 0, 1)
	for weekend(next) {
		next = next.AddDate(0, 0, 1)
	}
	return day.Add(9 * time.Hour), day.Add(18 * time.Hour), next.AddDate(0, 0, 1)
}

func weekend(day time.Time) bool {
	return day.Weekday() == time.Saturday || day.Weekday() == time.Sunday
}

// Samples keeps one volume per five-minute candle in the requested window.
type Samples struct {
	From, To time.Time
	values   map[time.Time]float64
}

func (s *Samples) Add(at time.Time, quantity float64) {
	if at.Before(s.From) || !at.Before(s.To) || at.Sub(s.From)%CandleInterval != 0 ||
		quantity < 0 || math.IsNaN(quantity) || math.IsInf(quantity, 0) {
		return
	}
	if s.values == nil {
		s.values = make(map[time.Time]float64)
	}
	s.values[at.UTC()] = quantity
}

// Mean returns -1 when the session has no usable candles.
func (s *Samples) Mean() float64 {
	if len(s.values) == 0 {
		return -1
	}
	mean := 0.0
	count := 0
	for at := s.From; at.Before(s.To); at = at.Add(CandleInterval) {
		if quantity, ok := s.values[at.UTC()]; ok {
			count++
			mean += (quantity - mean) / float64(count)
		}
	}
	return mean
}
