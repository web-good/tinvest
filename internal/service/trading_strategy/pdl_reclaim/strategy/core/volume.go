package core

// tfEnabled reports whether entries on signals of timeframe tf are switched on (Entry5m, Entry30m).
func (s *Strategy) tfEnabled(tf string) bool {
	if tf == "30m" {
		return s.p.Entry30m > 0
	}
	return s.p.Entry5m > 0
}

// volumeOn reports whether the volume filter is in play.
func (s *Strategy) volumeOn() bool { return s.p.VolumeMult > 0 && s.p.VolDays > 0 }

// volumeOK reports the volume of bar i of ser, the average of its slot and whether the volume
// filter passes: the volume is above VolumeMult × the slot average. The filter is off (always
// passes) when volumeOn is false; without volumes it fails closed.
func (s *Strategy) volumeOK(ser series, i int) (vol, avg float64, ok bool) {
	if !s.volumeOn() {
		return 0, 0, true
	}
	if len(ser.v) != len(ser.c) {
		return 0, 0, false
	}
	avg, ok = slotAverage(ser, i, s.p.VolDays)
	vol = float64(ser.v[i])
	return vol, avg, ok && avg > 0 && vol > s.p.VolumeMult*avg
}

// slotAverage is the average volume of the bars of ser starting at the same MSK minute as bar i on
// the days previous weekdays before its date (one bar per date, weekend sessions skipped), and
// whether the window held that many. Only bars before i are read.
func slotAverage(ser series, i, days int) (float64, bool) {
	m, day := minuteOf(ser.t[i]), dayKey(ser.t[i])
	var sum int64
	found, last := 0, day
	for j := i - 1; j >= 0 && found < days; j-- {
		d := dayKey(ser.t[j])
		if d == last || minuteOf(ser.t[j]) != m || isWeekend(ser.t[j].In(mskLoc)) {
			continue
		}
		sum += ser.v[j]
		found++
		last = d
	}
	if found < days {
		return 0, false
	}
	return float64(sum) / float64(days), true
}
