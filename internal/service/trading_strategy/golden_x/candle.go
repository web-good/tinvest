package golden_x

import (
	"time"
)

// startOfWeek returns Monday 00:00 in loc for the week containing t.
func startOfWeek(t time.Time, loc *time.Location) time.Time {
	t = t.In(loc)
	// time.Weekday: Sunday=0..Saturday=6. Shift so Monday=0.
	weekday := (int(t.Weekday()) + 6) % 7
	year, month, day := t.Date()
	return time.Date(year, month, day-weekday, 0, 0, 0, 0, loc)
}

