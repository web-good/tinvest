package live

import (
	"testing"
	"time"
)

func TestEntryBarTimeKeepsZero(t *testing.T) {
	if got := entryBarTime(time.Time{}); !got.IsZero() {
		t.Fatalf("entryBarTime(zero) = %v, want zero: без времени заявки ядро должно молчать", got)
	}
}
