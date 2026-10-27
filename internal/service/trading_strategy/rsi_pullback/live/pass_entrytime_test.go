package live

import (
	"testing"
	"time"

	"tinvest/internal/enum"
)

func TestEntryBarTimeKeepsZero(t *testing.T) {
	if got := entryBarTime(time.Time{}, enum.Minutes30); !got.IsZero() {
		t.Fatalf("entryBarTime(zero) = %v, want zero: без времени заявки ядро должно молчать", got)
	}
}
