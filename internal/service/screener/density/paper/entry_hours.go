package paper

import "time"

var moscowEntryClock = time.FixedZone("MSK", 3*60*60)

func (e *Engine) entryHoursBlocked(now time.Time) bool {
	if !e.cfg.RestrictEntryHours {
		return false
	}
	hour := now.In(moscowEntryClock).Hour()
	inEntryWindow := (hour >= 7 && hour < 12) || (hour >= 15 && hour < 18)
	return !inEntryWindow
}

func (e *Engine) cancelEntryOutsideHours(o *Order, now time.Time) bool {
	if o.State != "pending" || !e.entryHoursBlocked(now) {
		return false
	}
	e.cancel(o, "entry_hours", now)
	return true
}
