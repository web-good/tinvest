package density

import "testing"

func TestEntryLimitNeverExceedsPercentageZoneOnCoarseTick(t *testing.T) {
	// A one-ruble tick is 1% at this level. There is no ask tick inside 0.3%.
	if entry, err := EntryLimit(100_000_000_000, 1_000_000_000, 1, .3, 101_000_000_000); err == nil {
		t.Fatalf("bought at %d beyond 100.30 RUB", entry)
	}
}
