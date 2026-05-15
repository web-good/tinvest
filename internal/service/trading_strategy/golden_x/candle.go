package golden_x

import (
	"time"

	"tinvest/internal/domain"
)

// startOfWeek returns Monday 00:00 in loc for the week containing t.
func startOfWeek(t time.Time, loc *time.Location) time.Time {
	t = t.In(loc)
	// time.Weekday: Sunday=0..Saturday=6. Shift so Monday=0.
	weekday := (int(t.Weekday()) + 6) % 7
	year, month, day := t.Date()
	return time.Date(year, month, day-weekday, 0, 0, 0, 0, loc)
}

// lastClosedWeeklyRSI returns the latest RSI point whose Date is strictly
// before the start of the current week (Monday 00:00 in loc).
// Returns (nil, false) if no such point exists.
func lastClosedWeeklyRSI(items []*domain.RSIItemTechAnalyse, now time.Time, loc *time.Location) (*domain.RSIItemTechAnalyse, bool) {
	currentWeekStart := startOfWeek(now, loc)
	for i := len(items) - 1; i >= 0; i-- {
		if items[i] == nil {
			continue
		}
		if items[i].Date.Before(currentWeekStart) {
			return items[i], true
		}
	}
	return nil, false
}
