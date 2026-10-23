package core

// The main session, MSK minutes: from the 10:00 bar to the last one starting before 18:45. The
// volume averages skip the morning and evening sessions, the closing auction, weekends and
// weekday exchange holidays (dealer dates).
const (
	sessionFrom = 10 * 60
	sessionTo   = 18*60 + 45
	sessionBars = (sessionTo - sessionFrom) / 15 // 15-minute bars
)

// volumeOn reports whether the volume filter is in play.
func (s *Strategy) volumeOn() bool { return s.p.VolumeMult > 0 && s.p.VolBars > 0 }

// volumeOK reports the average main-session volume of bars from..k, the average of the VolBars
// main-session bars before from and whether the window is above VolumeMult × that average. Off —
// always passes; without volumes or main-session bars in the window it fails closed.
func (s *Strategy) volumeOK(ser series, from, k int) (win, base float64, ok bool) {
	if !s.volumeOn() {
		return 0, 0, true
	}
	if len(ser.v) != len(ser.c) || len(ser.t) != len(ser.c) {
		return 0, 0, false
	}
	var sum int64
	got := 0
	for j := from; j <= k; j++ {
		if inSession(ser, j) {
			sum += ser.v[j]
			got++
		}
	}
	if got == 0 {
		return 0, 0, false
	}
	win = float64(sum) / float64(got)
	base, ok = averageVolume(ser, from, s.p.VolBars)
	return win, base, ok && base > 0 && win > s.p.VolumeMult*base
}

// averageVolume is the average volume of the n main-session bars of ser before bar i, and whether
// there are that many.
func averageVolume(ser series, i, n int) (float64, bool) {
	var sum int64
	got := 0
	for j := i - 1; j >= 0 && got < n; j-- {
		if !inSession(ser, j) {
			continue
		}
		sum += ser.v[j]
		got++
	}
	if got < n {
		return 0, false
	}
	return float64(sum) / float64(n), true
}

// inSession reports whether bar j of ser is a main-session bar of a weekday exchange date: weekend
// and dealer-date (dealerDate) bars never count.
func inSession(ser series, j int) bool {
	if isWeekend(ser.t[j].In(mskLoc)) {
		return false
	}
	m := minuteOf(ser.t[j])
	return m >= sessionFrom && m < sessionTo && !dealerDate(ser.t, j)
}
