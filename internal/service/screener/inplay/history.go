package inplay

import "time"

const (
	baselineDays        = 10
	historyLookbackDays = 35
	barsPerHalfHour     = 6
)

type historyKey struct {
	id  string
	day time.Time
}

type dayVolumes struct {
	totals [48]float64
	valid  [48]bool
}

type volumeHistory map[historyKey]dayVolumes

func (v dayVolumes) window(slot, n int) (float64, bool) {
	if n == barsPerHalfHour {
		return v.totals[slot], v.valid[slot]
	}
	return v.totals[slot] + v.totals[slot+1], v.valid[slot] && v.valid[slot+1]
}

func (h volumeHistory) baseline(fetch func(string, time.Time, time.Time) ([]bar, error), id string, end time.Time, n int) (float64, error) {
	start := end.UTC().Add(-time.Duration(n) * step)
	day := start.Truncate(24 * time.Hour)
	slot := int(start.Sub(day) / (30 * time.Minute))
	var total float64
	days := 0
	for offset := 1; offset <= historyLookbackDays; offset++ {
		past := day.AddDate(0, 0, -offset)
		key := historyKey{id, past}
		volumes, cached := h[key]
		if !cached {
			bars, err := fetch(id, past, past.AddDate(0, 0, 1))
			if err != nil {
				return 0, err
			}
			for i := range volumes.totals {
				slotEnd := past.Add(time.Duration(i+1) * 30 * time.Minute)
				r, ok := summarize(bars, slotEnd, barsPerHalfHour, 0)
				volumes.totals[i], volumes.valid[i] = r.volume, ok
			}
			h[key] = volumes
		}
		if volume, ok := volumes.window(slot, n); ok {
			total += volume
			days++
			if days == baselineDays {
				return total / baselineDays, nil
			}
		}
	}
	return 0, nil
}
