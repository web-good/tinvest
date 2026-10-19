package core

// The main session's 15-minute bars, MSK: from the 10:00 bar to the 18:30 one inclusive. The
// volume average skips the morning and evening sessions, the closing auction bar and weekends.
const (
	sessionFrom = 10 * 60
	sessionTo   = 18*60 + 45
	sessionBars = (sessionTo - sessionFrom) / 15
)

// volumeOn reports whether the volume filter is in play.
func (s *Strategy) volumeOn() bool { return s.p.VolumeMult > 0 && s.p.VolBars > 0 }

// volumeOK reports the volume of bar i of ser (the PDH breakout bar), the average of the VolBars
// main-session bars before it and whether the volume filter passes: the volume is above VolumeMult
// × that average. The filter is off (always passes) when volumeOn is false; without volumes it
// fails closed.
func (s *Strategy) volumeOK(ser series, i int) (vol, avg float64, ok bool) {
	if !s.volumeOn() {
		return 0, 0, true
	}
	if len(ser.v) != len(ser.c) || len(ser.t) != len(ser.c) {
		return 0, 0, false
	}
	avg, ok = averageVolume(ser, i, s.p.VolBars)
	vol = float64(ser.v[i])
	return vol, avg, ok && avg > 0 && vol > s.p.VolumeMult*avg
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

// inSession reports whether bar j of ser is a weekday main-session bar.
func inSession(ser series, j int) bool {
	if isWeekend(ser.t[j].In(mskLoc)) {
		return false
	}
	m := minuteOf(ser.t[j])
	return m >= sessionFrom && m < sessionTo
}
