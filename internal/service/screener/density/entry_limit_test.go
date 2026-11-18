package density

import "testing"

func TestEntryLimitUsesInstrumentPriceStep(t *testing.T) {
	const level = 100_000_000_000
	const ask = 101_000_000_000
	for _, tc := range []struct {
		name     string
		step     int64
		accepted bool
	}{
		{"one tick above density", 1_000_000_000, true},
		{"two ticks above density", 500_000_000, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			entry, err := EntryLimit(level, tc.step, 1, 1, ask)
			if !tc.accepted {
				if err == nil {
					t.Fatalf("accepted ask %d two ticks above density", entry)
				}
				return
			}
			if err != nil || entry != ask {
				t.Fatalf("entry = %d, err = %v; want %d", entry, err, ask)
			}
		})
	}
}
