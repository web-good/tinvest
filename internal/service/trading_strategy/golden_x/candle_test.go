package golden_x

import (
	"testing"
	"time"

	"tinvest/internal/domain"
)

func mustLoad(t *testing.T, name string) *time.Location {
	t.Helper()
	loc, err := time.LoadLocation(name)
	if err != nil {
		t.Fatalf("load location %s: %v", name, err)
	}
	return loc
}

func TestLastClosedWeeklyRSI(t *testing.T) {
	msk := mustLoad(t, "Europe/Moscow")

	// "now" — Thursday 2026-05-14 12:00 MSK.
	// Current week = 2026-05-11 (Mon) 00:00 .. 2026-05-18 (Mon) 00:00.
	now := time.Date(2026, 5, 14, 12, 0, 0, 0, msk)

	prevWeek := time.Date(2026, 5, 4, 0, 0, 0, 0, msk)
	twoWeeksAgo := time.Date(2026, 4, 27, 0, 0, 0, 0, msk)
	currentWeekOpen := time.Date(2026, 5, 11, 0, 0, 0, 0, msk)

	tests := []struct {
		name     string
		items    []*domain.RSIItemTechAnalyse
		wantOK   bool
		wantDate time.Time
	}{
		{
			name: "массив включает текущую открытую неделю — берём предыдущую",
			items: []*domain.RSIItemTechAnalyse{
				{Date: twoWeeksAgo},
				{Date: prevWeek},
				{Date: currentWeekOpen},
			},
			wantOK:   true,
			wantDate: prevWeek,
		},
		{
			name: "массив без текущей недели — последний элемент уже закрыт",
			items: []*domain.RSIItemTechAnalyse{
				{Date: twoWeeksAgo},
				{Date: prevWeek},
			},
			wantOK:   true,
			wantDate: prevWeek,
		},
		{
			name:   "пустой массив",
			items:  nil,
			wantOK: false,
		},
		{
			name: "только текущая неделя — нет закрытых",
			items: []*domain.RSIItemTechAnalyse{
				{Date: currentWeekOpen},
			},
			wantOK: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, ok := lastClosedWeeklyRSI(tc.items, now, msk)
			if ok != tc.wantOK {
				t.Fatalf("ok = %v, want %v", ok, tc.wantOK)
			}
			if !ok {
				return
			}
			if !got.Date.Equal(tc.wantDate) {
				t.Fatalf("date = %v, want %v", got.Date, tc.wantDate)
			}
		})
	}
}
