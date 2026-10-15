package core

import (
	"time"

	"tinvest/internal/service/trading_strategy/scalping/strategy"
)

// halfHour is the timeframe the strategy trades on from Switch on.
const halfHour = 30 * time.Minute

// series is one timeframe's bars, oldest-first: opens, highs, lows, closes and start times.
type series struct {
	o, h, l, c []float64
	t          []time.Time
}

// fiveMin is the engine's own 5m bars as a series (no copy).
func fiveMin(md strategy.MarketData) series {
	return series{o: md.Opens, h: md.Highs, l: md.Lows, c: md.Closes, t: md.Times}
}

// halfStart is the start of the MSK half-hour holding t. MSK is UTC+3 in whole hours, so
// truncating the absolute time lands on MSK :00 and :30.
func halfStart(t time.Time) time.Time { return t.Truncate(halfHour) }

// lastOfHalf reports whether the bar starting at t is the last 5m slot of its half-hour (:25/:55).
func (s *Strategy) lastOfHalf(t time.Time) bool {
	return t.Add(s.step).Equal(halfStart(t).Add(halfHour))
}

// switchMinute is the MSK minute of the day from which signals are read on half-hours.
func (s *Strategy) switchMinute() int { return s.p.SwitchHour*60 + s.p.SwitchMinute }

// switchOn reports whether the half-hour timeframe is in play at all (SwitchHour 24 — never).
func (s *Strategy) switchOn() bool { return s.switchMinute() < 24*60 }

// halves builds the completed half-hours from md's 5m bars. done[k] is the index of the 5m bar on
// whose close half-hour k became known: its :25/:55 bar, or — when that bar is missing — the
// first bar of a later half-hour. The half-hour still forming is not returned.
func (s *Strategy) halves(md strategy.MarketData) (series, []int) {
	var hs series
	var done []int
	for j := range md.Closes {
		st := halfStart(md.Times[j])
		k := len(hs.t) - 1
		if k < 0 || !hs.t[k].Equal(st) {
			if k >= 0 && done[k] < 0 {
				done[k] = j
			}
			hs.t = append(hs.t, st)
			hs.o = append(hs.o, md.Opens[j])
			hs.h = append(hs.h, md.Highs[j])
			hs.l = append(hs.l, md.Lows[j])
			hs.c = append(hs.c, md.Closes[j])
			done = append(done, -1)
		} else {
			hs.h[k] = max(hs.h[k], md.Highs[j])
			hs.l[k] = min(hs.l[k], md.Lows[j])
			hs.c[k] = md.Closes[j]
		}
		if s.lastOfHalf(md.Times[j]) {
			done[len(done)-1] = j
		}
	}
	if k := len(done) - 1; k >= 0 && done[k] < 0 {
		hs = series{o: hs.o[:k], h: hs.h[:k], l: hs.l[:k], c: hs.c[:k], t: hs.t[:k]}
		done = done[:k]
	}
	return hs, done
}

// closedHalf returns the half-hours and the index of the one that closed on the current bar, or
// ok=false when no half-hour closed on it. The cheap time test runs first, so the half-hours are
// built about once per half-hour instead of on every 5m bar.
func (s *Strategy) closedHalf(md strategy.MarketData) (series, int, bool) {
	n := len(md.Closes)
	if n == 0 || len(md.Times) != n || len(md.Opens) != n || len(md.Highs) != n || len(md.Lows) != n {
		return series{}, -1, false
	}
	t := md.Times[n-1]
	gap := n > 1 && !halfStart(t).Equal(halfStart(md.Times[n-2])) && !s.lastOfHalf(md.Times[n-2])
	if !s.lastOfHalf(t) && !gap {
		return series{}, -1, false
	}
	hs, done := s.halves(md)
	k := len(done) - 1
	if k < 0 || done[k] != n-1 {
		return series{}, -1, false
	}
	return hs, k, true
}

// signalBar picks the bar the entry rules read on the current 5m bar: the 5m bar itself before
// Switch, the half-hour that just closed from Switch on (ok=false when none closed).
func (s *Strategy) signalBar(md strategy.MarketData) (ser series, k int, tf string, ok bool) {
	n := len(md.Closes)
	if !s.switchOn() || minuteOf(md.Times[n-1]) < s.switchMinute() {
		return fiveMin(md), n - 1, "5m", true
	}
	hs, k, ok := s.closedHalf(md)
	return hs, k, "30m", ok
}

// on30m reports whether a position lives on half-hours: it filled at or after Switch + 30m (the
// earliest half-hour fill). A 5m position keeps 5m exits for its whole life; no entry time — 5m.
func (s *Strategy) on30m(pos *strategy.Position) bool {
	return s.switchOn() && !pos.EntryTime.IsZero() && minuteOf(pos.EntryTime) >= s.switchMinute()+30
}

// halfIndex is the index of the half-hour holding entry, or -1 when entry is unset or not in times.
func halfIndex(times []time.Time, entry time.Time) int {
	if entry.IsZero() {
		return -1
	}
	for j := len(times) - 1; j >= 0; j-- {
		if !times[j].After(entry) {
			if entry.Before(times[j].Add(halfHour)) {
				return j
			}
			return -1
		}
	}
	return -1
}

// exitSeries returns the closes the RSI and MACD exits read for a held position, the current
// index on them and the entry's index: 5m bars for a 5m position, half-hours for a 30m one
// (ok=false when no half-hour closed on this bar).
func (s *Strategy) exitSeries(md strategy.MarketData, pos *strategy.Position) (closes []float64, i, e int, tf string, ok bool) {
	if s.on30m(pos) {
		hs, k, ok := s.closedHalf(md)
		if !ok {
			return nil, -1, -1, "30m", false
		}
		return hs.c, k, halfIndex(hs.t, pos.EntryTime), "30m", true
	}
	n := len(md.Closes)
	if len(md.Times) != n {
		return md.Closes, n - 1, -1, "5m", true
	}
	return md.Closes, n - 1, entryIndex(md.Times, pos.EntryTime), "5m", true
}
