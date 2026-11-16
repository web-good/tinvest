package live

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"tinvest/internal/service/screener/density"
)

func TestLegacyEpisodeMigratesWithoutResubmittingBuy(t *testing.T) {
	x := setup(t)
	c := attach(t, x, nil)
	x.step(t)
	state := x.e.State()
	key := state.Episodes[x.i.ID].Entry.Key
	state.Version = 1
	state.Episode = state.Episodes[x.i.ID]
	state.Episodes, state.Releases = nil, nil
	data, err := json.Marshal(journalRecord{Event: "entry_intent", State: state})
	if err != nil {
		t.Fatal(err)
	}
	if err = x.s.Close(); err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(filepath.Join(x.s.Dir, "events.jsonl"), append(data, '\n'), 0600); err != nil {
		t.Fatal(err)
	}
	store, err := OpenStore(filepath.Dir(x.s.Dir), x.account)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	migrated := store.State()
	if migrated.Version != 2 || migrated.Episode != nil || migrated.Episodes[x.i.ID].Entry.Key != key {
		t.Fatalf("migration lost legacy request identity: %+v", migrated)
	}
	e, err := NewEngine(DefaultConfig(), x.b, store, NewFeed(DefaultConfig()), []density.Instrument{x.i})
	if err != nil {
		t.Fatal(err)
	}
	if err = e.AttachCoordinator(context.Background(), c); err != nil {
		t.Fatal(err)
	}
	if err = e.Shutdown(context.Background()); err != nil {
		t.Fatal(err)
	}
	if len(x.b.posts) != 1 || e.State().Canceled != 1 {
		t.Fatal("migration repeated broker request or accounting")
	}
}

func TestRecoverFlushesNotificationBeforeLookingForNewInstruments(t *testing.T) {
	x := setup(t)
	c := attach(t, x, nil)
	state := x.s.State()
	state.Notifications = []Notification{{ID: "closed:sell", Side: "SELL", Ticker: "T"}}
	if err := x.s.Save(state, "episode_closed"); err != nil {
		t.Fatal(err)
	}
	sent := 0
	p := Participant{Coordinator: c, Notify: func(_ context.Context, n Notification) error {
		sent++
		if n.ID != "closed:sell" {
			t.Fatal(n)
		}
		return nil
	}}
	// No market transport exists: delivery must not depend on a new universe.
	if err := p.recover(context.Background(), x.s); err != nil {
		t.Fatal(err)
	}
	if sent != 1 || len(x.s.State().Notifications) != 0 {
		t.Fatal("recovery did not deliver and persist acknowledgement")
	}
}
