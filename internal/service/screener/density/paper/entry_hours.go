package paper

import "time"

var moscowEntryClock = time.FixedZone("MSK", 3*60*60)

func (e *Engine) entryHoursBlocked(now time.Time) bool {
	if !e.cfg.RestrictEntryHours {
		return false
	}
	hour := now.In(moscowEntryClock).Hour()
	return hour < 9 || hour >= 19
}

func (e *Engine) cancelEntryOutsideHours(o *Order, now time.Time) bool {
	if o.State != "pending" || !e.entryHoursBlocked(now) {
		return false
	}
	e.cancel(o, "entry_hours", now)
	return true
}
