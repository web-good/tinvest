package core

import (
	"fmt"
	"math"
)

// setup is the bar that arms an entry: the reclaim of PDL or the breakout of PDH.
type setup struct {
	bar   int     // index of the setup bar
	name  string  // "PDL" or "PDH"
	level float64 // PDL or PDH
	kind  string  // the pattern of the PDL pattern entry ("молот", "поглощение"); "" otherwise
}

// what names the setup in reasons.
func (st setup) what() string {
	if st.name == "PDH" {
		return "пробой PDH"
	}
	if st.kind != "" {
		return st.kind + " у PDL"
	}
	return "возврат над PDL"
}

// isReclaim: bar i is green and closed above PDL, and walking back over the bars of its date a
// red bar closed below PDL before any bar closed above it.
func isReclaim(ser series, i int, pdl float64) bool {
	if ser.c[i] <= ser.o[i] || ser.c[i] <= pdl {
		return false
	}
	day := dayKey(ser.t[i])
	for j := i - 1; j >= 0 && dayKey(ser.t[j]) == day; j-- {
		if ser.c[j] > pdl {
			return false
		}
		if ser.c[j] < ser.o[j] && ser.c[j] < pdl {
			return true
		}
	}
	return false
}

// isBreakout: bar i is green and closed above PDH, the bar before it, on the same date, closed at
// or below PDH — the first close above yesterday's high.
func isBreakout(ser series, i int, pdh float64) bool {
	if i < 1 || dayKey(ser.t[i]) != dayKey(ser.t[i-1]) {
		return false
	}
	return ser.c[i] > ser.o[i] && ser.c[i] > pdh && ser.c[i-1] <= pdh
}

// trigger is the setup bar k buys on and the reason fragment of its confirmation: the held PDH
// breakout wins over the PDL reclaim on bar k itself.
// minRange is the smallest range a pattern bar of the pattern entry may have (0 — any).
func (s *Strategy) trigger(ser series, k int, pdl, pdh, minRange float64) (setup, string, bool) {
	if st, minClose, ok := s.held(ser, k, pdh); ok {
		return st, fmt.Sprintf("держится над PDH %d св. (мин. close %.4f)", k-st.bar, minClose), true
	}
	if s.p.EntryPDL <= 0 {
		return setup{}, "", false
	}
	if s.p.PDLPatternEntry > 0 {
		st, ok := s.patternEntry(ser, k, pdl, minRange)
		if !ok {
			return setup{}, "", false
		}
		why := fmt.Sprintf("первая зелёная после паттерна %s", ser.t[k].In(mskLoc).Format("15:04"))
		return s.withDipVolume(ser, k, pdl, st, why)
	}
	if !isReclaim(ser, k, pdl) {
		return setup{}, "", false
	}
	why := "покупка сразу"
	if s.patternsOn() {
		name, at, ok := s.pdlPattern(ser, k, pdl)
		if !ok {
			return setup{}, "", false
		}
		why += fmt.Sprintf(", %s %s", name, ser.t[at].In(mskLoc).Format("15:04"))
	}
	return s.withDipVolume(ser, k, pdl, setup{bar: k, name: "PDL", level: pdl}, why)
}

// withDipVolume applies the volume filter of the series under PDL to a PDL setup triggered on bar k.
func (s *Strategy) withDipVolume(ser series, k int, pdl float64, st setup, why string) (setup, string, bool) {
	if s.dipVolumeOn() {
		dip, avg, ok := s.dipVolumeOK(ser, k, pdl)
		if !ok {
			return setup{}, "", false
		}
		why += fmt.Sprintf(", объём серии %.0f > %.2f × обычный %.0f", dip, s.p.PDLDipVolMult, avg)
	}
	return st, why, true
}

// patternEntry: bar k is the first green bar after the latest reversal candle of an enabled kind on
// k's date with a range of at least minRange, and that candle formed after the break of PDL
// (patternUnderPDL). The setup bar is the pattern bar.
func (s *Strategy) patternEntry(ser series, k int, pdl, minRange float64) (setup, bool) {
	if ser.c[k] <= ser.o[k] {
		return setup{}, false
	}
	day := dayKey(ser.t[k])
	for j := k - 1; j >= 0 && dayKey(ser.t[j]) == day; j-- {
		name := ""
		switch {
		case s.p.PDLEngulf > 0 && isEngulf(ser, j):
			name = "поглощение"
		case s.p.PDLHammer > 0 && isHammer(ser, j):
			name = "молот"
		}
		if name != "" && ser.h[j]-ser.l[j] >= minRange {
			return setup{bar: j, name: "PDL", level: pdl, kind: name}, patternUnderPDL(ser, j, pdl)
		}
		if ser.c[j] > ser.o[j] {
			return setup{}, false // an earlier pattern already had its first green bar
		}
	}
	return setup{}, false
}

// patternUnderPDL: walking back from the pattern bar j over its date, a bar closed below PDL before
// any bar before j closed above it (j itself or the red bar of an engulfing may be the break).
func patternUnderPDL(ser series, j int, pdl float64) bool {
	day := dayKey(ser.t[j])
	for i := j; i >= 0 && dayKey(ser.t[i]) == day; i-- {
		if i < j && ser.c[i] > pdl {
			return false
		}
		if ser.c[i] < pdl {
			return true
		}
	}
	return false
}

// patternsOn: the PDL reclaim needs a reversal candle.
func (s *Strategy) patternsOn() bool { return s.p.PDLEngulf > 0 || s.p.PDLHammer > 0 }

// pdlPattern finds the latest reversal candle of an enabled kind between the break of PDL and the
// reclaim bar k, both included (from dipStart). The red bar of an engulfing may sit before the window.
func (s *Strategy) pdlPattern(ser series, k int, pdl float64) (name string, bar int, ok bool) {
	for i := k; i >= dipStart(ser, k, pdl); i-- {
		if s.p.PDLEngulf > 0 && isEngulf(ser, i) {
			return "поглощение", i, true
		}
		if s.p.PDLHammer > 0 && isHammer(ser, i) {
			return "молот", i, true
		}
	}
	return "", 0, false
}

// dipStart is the first bar of the series under PDL before the reclaim bar k: the run of closes at
// or below PDL that ends right before k on k's date.
func dipStart(ser series, k int, pdl float64) int {
	day := dayKey(ser.t[k])
	start := k
	for start > 0 && dayKey(ser.t[start-1]) == day && ser.c[start-1] <= pdl {
		start--
	}
	return start
}

// isEngulf: the bullish engulfing on bar i — bar i-1, on i's date, is red, bar i is green, its low
// is below the red bar's low and its close above the red bar's high.
func isEngulf(ser series, i int) bool {
	if i < 1 || dayKey(ser.t[i]) != dayKey(ser.t[i-1]) {
		return false
	}
	p := i - 1
	return ser.c[p] < ser.o[p] && ser.c[i] > ser.o[i] && ser.l[i] < ser.l[p] && ser.c[i] > ser.h[p]
}

// isHammer: bar i of either colour has a body of at most a third of its range lying in the upper
// third — so the lower tail is at least two thirds of the range and at least twice the body.
func isHammer(ser series, i int) bool {
	rng := ser.h[i] - ser.l[i]
	if rng <= 0 {
		return false
	}
	body := math.Abs(ser.c[i] - ser.o[i])
	return body <= rng/3 && min(ser.o[i], ser.c[i]) >= ser.l[i]+rng*2/3
}

// held: bar k completes the hold of a PDH breakout — bar k-HoldBars, on k's date, is the breakout
// (with the volume filter) and none of the bars after it up to k closed below PDH. minClose is the
// lowest close from the breakout bar to k.
func (s *Strategy) held(ser series, k int, pdh float64) (st setup, minClose float64, ok bool) {
	r := k - s.p.HoldBars
	if s.p.EntryPDH <= 0 || s.p.HoldBars < 0 || r < 0 || dayKey(ser.t[r]) != dayKey(ser.t[k]) || !isBreakout(ser, r, pdh) {
		return setup{}, 0, false
	}
	if _, _, okv := s.volumeOK(ser, r); !okv {
		return setup{}, 0, false
	}
	minClose = ser.c[r]
	for j := r + 1; j <= k; j++ {
		if ser.c[j] < pdh {
			return setup{}, 0, false
		}
		minClose = min(minClose, ser.c[j])
	}
	return setup{bar: r, name: "PDH", level: pdh}, minClose, true
}
