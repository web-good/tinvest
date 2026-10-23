package core

// setup is a setup that produced a signal: from is the first 15m bar of the arming hour (the
// search window's start), arm the bar that completed the arming hour, signal the signal bar.
type setup struct{ from, arm, signal int }

// scan replays the setup automaton over the bars of bar k's date and returns the setup whose
// signal is bar k. Per bar j, in order:
//   - each hour of that date completed at j (from RSIPeriod hours of history on, so the warm-up
//     zeros never arm): an armed setup whose hourly RSI crosses RSIGate up burns; an idle one
//     arms when the hourly RSI is below RSIArm (a further hour below RSIArm while armed changes
//     nothing);
//   - an armed setup after its arming bar sees the 15m RSI cross RSIEntry up: a low outside
//     [zoneLo, zoneHi] keeps it armed, a low inside is the signal and burns it, whatever the
//     filters say later.
//
// rh is the RSI of hs's closes, r15 the RSI of ser's closes.
func (s *Strategy) scan(ser series, hs []hourBar, rh, r15 []float64, k int, zoneLo, zoneHi float64) (setup, bool) {
	date := dayKey(ser.t[k])
	start := k
	for start > 0 && dayKey(ser.t[start-1]) == date {
		start--
	}
	h := 0
	for h < len(hs) && hs[h].done < start {
		h++
	}
	armed := false
	var cur setup
	for j := start; j <= k; j++ {
		for ; h < len(hs) && hs[h].done == j; h++ {
			if h < s.p.RSIPeriod || dayKey(hs[h].start) != date {
				continue
			}
			if armed && s.p.RSIGate > 0 && crossUp(rh[h-1], rh[h], s.p.RSIGate) {
				armed = false
				continue
			}
			if !armed && rh[h] < s.p.RSIArm {
				armed, cur = true, setup{from: hs[h].first, arm: j}
			}
		}
		if !armed || j <= cur.arm || j < s.p.RSIPeriod+1 || !crossUp(r15[j-1], r15[j], s.p.RSIEntry) {
			continue
		}
		if ser.l[j] < zoneLo || ser.l[j] > zoneHi {
			continue
		}
		armed = false
		if j == k {
			cur.signal = j
			return cur, true
		}
	}
	return setup{}, false
}

// stopOf is the lowest low of the window bars from..signal-1 whose 15m RSI is below entryLevel
// and of the signal bar itself. The window lies past the RSI warm-up (the arming hour needs
// RSIPeriod hours of history), so r15 there holds real values.
func stopOf(ser series, r15 []float64, st setup, entryLevel float64) float64 {
	return ser.l[stopBar(ser, r15, st, entryLevel)]
}

// stopBar is the bar whose low is stopOf's level: the earliest such window bar, the signal bar
// when nothing earlier is strictly lower.
func stopBar(ser series, r15 []float64, st setup, entryLevel float64) int {
	at := st.signal
	for j := st.from; j < st.signal; j++ {
		if r15[j] < entryLevel && ser.l[j] < ser.l[at] {
			at = j
		}
	}
	return at
}
