package bonds

import (
	"testing"
	"time"
)

func TestDefaultLadder(t *testing.T) {
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	rungs := DefaultLadder(now)

	if len(rungs) != 4 {
		t.Fatalf("ожидалось 4 ступени, получено %d", len(rungs))
	}
	// Три ОФЗ-ступени + одна корпоративная.
	ofz := 0
	for _, r := range rungs {
		if r.IsOfz {
			ofz++
		}
		if !r.To.After(r.From) {
			t.Fatalf("ступень с некорректными границами: %+v", r)
		}
	}
	if ofz != 3 {
		t.Fatalf("ожидалось 3 ОФЗ-ступени, получено %d", ofz)
	}
	// Первая ОФЗ-ступень: 180д..2г.
	if !rungs[0].From.Equal(now.AddDate(0, 0, 180)) || !rungs[0].To.Equal(now.AddDate(2, 0, 0)) {
		t.Fatalf("границы первой ступени неверны: %+v", rungs[0])
	}
}
