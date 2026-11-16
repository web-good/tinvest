package live

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"tinvest/internal/service/trading_strategy/livecore/accountcoord"
)

func TestCandleGuardRejectsDifferentCoordinatorBeforeDecisions(t *testing.T) {
	x := setup(t)
	old := attach(t, x, nil)
	x.step(t)
	fresh := accountcoord.New(t.TempDir(), x.account, nil)
	fresh.SetGuard(func() error { return CheckSharedBinding(x.s.Dir, x.account, fresh.Root()) })
	if lease, err := fresh.Lock(context.Background()); err == nil {
		lease.Close()
		t.Fatal("new candle root ignored old density position")
	}
	if err := CheckSharedBinding(x.s.Dir, x.account, old.Root()); err != nil {
		t.Fatal(err)
	}
}

func TestReadOnlyGuardAcceptsLargeCompleteJournalFrames(t *testing.T) {
	x := setup(t)
	c := attach(t, x, nil)
	state := x.s.State()
	state.Notifications = []Notification{{ID: "offline", Reason: strings.Repeat("x", 1<<20)}}
	if err := x.s.Save(state, "large_pending_notification"); err != nil {
		t.Fatal(err)
	}
	// A concurrent append may leave an incomplete next frame. Read the last
	// complete state even when its start lies beyond the initial tail window.
	f, err := os.OpenFile(filepath.Join(x.s.Dir, "events.jsonl"), os.O_APPEND|os.O_WRONLY, 0600)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = f.WriteString(`{"partial":`); err != nil {
		t.Fatal(err)
	}
	if err = f.Close(); err != nil {
		t.Fatal(err)
	}
	if err := CheckSharedBinding(x.s.Dir, x.account, c.Root()); err != nil {
		t.Fatal(err)
	}
}
