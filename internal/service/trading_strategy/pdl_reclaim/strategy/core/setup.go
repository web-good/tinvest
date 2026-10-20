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
}

// what names the setup in reasons.
func (st setup) what() string {
	if st.name == "PDH" {
		return "пробой PDH"
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
func (s *Strategy) trigger(ser series, k int, pdl, pdh float64) (setup, string, bool) {
	if st, minClose, ok := s.held(ser, k, pdh); ok {
		return st, fmt.Sprintf("держится над PDH %d св. (мин. close %.4f)", k-st.bar, minClose), true
	}
	if s.p.EntryPDL <= 0 || !isReclaim(ser, k, pdl) {
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
	return setup{bar: k, name: "PDL", level: pdl}, why, true
}

// patternsOn: the PDL reclaim needs a reversal candle.
func (s *Strategy) patternsOn() bool { return s.p.PDLEngulf > 0 || s.p.PDLHammer > 0 }

// pdlPattern finds the latest reversal candle of an enabled kind between the break of PDL and the
// reclaim bar k, both included: the window starts at the first bar of a run of closes at or below
// PDL that ends right before k on k's date. The red bar of an engulfing may sit before the window.
func (s *Strategy) pdlPattern(ser series, k int, pdl float64) (name string, bar int, ok bool) {
	day := dayKey(ser.t[k])
	start := k
	for start > 0 && dayKey(ser.t[start-1]) == day && ser.c[start-1] <= pdl {
		start--
	}
	for i := k; i >= start; i-- {
		if s.p.PDLEngulf > 0 && isEngulf(ser, i) {
			return "поглощение", i, true
		}
		if s.p.PDLHammer > 0 && isHammer(ser, i) {
			return "молот", i, true
		}
	}
	return "", 0, false
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
