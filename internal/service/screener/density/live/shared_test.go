package live

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"tinvest/internal/service/screener/density"
	"tinvest/internal/service/trading_strategy/livecore/accountcoord"
	"tinvest/internal/service/trading_strategy/livecore/statestore"
)

func attach(t *testing.T, x *fixture, store statestore.Store) *accountcoord.Coordinator {
	t.Helper()
	c := accountcoord.New(t.TempDir(), x.account, store)
	if err := x.e.AttachCoordinator(context.Background(), c); err != nil {
		t.Fatal(err)
	}
	return c
}
func TestSharedEntryReservesBeforeBrokerCall(t *testing.T) {
	x := setup(t)
	c := attach(t, x, nil)
	x.b.onPost = func() {
		claims, err := c.Claims(context.Background())
		if err != nil || claims["T"].Owner != "density" || claims["T"].Key != x.e.State().Episodes["uid"].Entry.Key {
			t.Error("BUY sent before durable ownership", claims, err)
		}
	}
	x.step(t)
	claims, _ := c.Claims(context.Background())
	if claims["T"].Status != "active" {
		t.Fatal(claims)
	}
}
func TestSharedBusyTickerDoesNotBlockFreeNeighbor(t *testing.T) {
	x := setup(t)
	store := statestore.New(filepath.Join(t.TempDir(), "rsi.json"))
	store.Save(map[string]statestore.Entry{"T": {Ticker: "T", Quantity: 10, Strategy: "rsi_pullback"}})
	attach(t, x, store)
	x.b.holdings["uid"] = Holding{Units: 10}
	other := density.Instrument{ID: "sber", Ticker: "SBER", Lot: 10, PriceStep: density.Price{Units: 1}}
	x.e.instruments[other.ID] = other
	b := x.f.Candidates(x.now)[0].Book
	b.InstrumentID = other.ID
	b.Time = x.now
	x.f.SetReady(other.ID, true)
	x.f.Book(b, []density.Signal{{InstrumentID: other.ID, Price: density.Price{Units: 100}, Quantity: 1000}}, x.now)
	x.step(t)
	if len(x.b.posts) != 1 || x.b.posts[0].InstrumentID != "sber" {
		t.Fatalf("busy T blocked SBER: %+v", x.b.posts)
	}
}
func TestEntryLeaseDoesNotDelayDensityLossCancellationOrExit(t *testing.T) {
	x := setup(t)
	c := attach(t, x, nil)
	x.step(t)
	x.fill()
	x.step(t)
	lease, err := c.Lock(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	defer lease.Close()
	x.book(499, false)
	x.step(t)
	if x.e.State().Closed != 1 {
		t.Fatal("candle entry lock blocked safety exit")
	}
	claims, _ := c.Claims(context.Background())
	if claims["T"].Status != "closed" {
		t.Fatal("closed ownership not retained as tombstone")
	}
}
func TestSharedNewEntryWaitsForCandlePass(t *testing.T) {
	x := setup(t)
	c := attach(t, x, nil)
	lease, err := c.Lock(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	defer lease.Close()
	if err = x.e.Step(context.Background(), x.now); err != nil {
		t.Fatal(err)
	}
	if len(x.b.posts) != 0 {
		t.Fatal("density entered during candle pass")
	}
}
func TestSharedSnapshotCannotTradeWithoutCoordinator(t *testing.T) {
	x := setup(t)
	attach(t, x, nil)
	e, err := NewEngine(DefaultConfig(), x.b, x.s, x.f, []density.Instrument{x.i})
	if err != nil {
		t.Fatal(err)
	}
	if err = e.Step(context.Background(), x.now); err == nil {
		t.Fatal("shared state traded as standalone")
	}
	if len(x.b.posts) != 0 {
		t.Fatal("uncoordinated BUY")
	}
}
func TestUnknownSharedBuyKeepsOwnershipAfterRestart(t *testing.T) {
	x := setup(t)
	c := attach(t, x, nil)
	x.step(t)
	claim := x.e.State().Episodes["uid"].Entry.Key
	fresh, err := NewEngine(DefaultConfig(), x.b, x.s, x.f, []density.Instrument{x.i})
	if err != nil {
		t.Fatal(err)
	}
	if err = fresh.AttachCoordinator(context.Background(), c); err != nil {
		t.Fatal(err)
	}
	claims, _ := c.Claims(context.Background())
	if claims["T"].Key != claim {
		t.Fatal("restart replaced owner")
	}
	other := accountcoord.New(t.TempDir(), x.account, nil)
	if err = fresh.AttachCoordinator(context.Background(), other); err == nil {
		t.Fatal("another registry bypassed ownership")
	}
	if errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
}

func TestOrphanActiveOwnershipBlocksEveryNewDensityEntry(t *testing.T) {
	x := setup(t)
	c := accountcoord.New(t.TempDir(), x.account, nil)
	lease, err := c.Lock(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if err = c.Reserve(lease, accountcoord.Claim{Ticker: "OTHER", InstrumentID: "other", Owner: "density", Key: "unresolved"}); err != nil {
		t.Fatal(err)
	}
	lease.Close()
	if err = x.e.AttachCoordinator(context.Background(), c); err == nil {
		t.Fatal("orphan density episode allowed another position")
	}
	if len(x.b.posts) != 0 {
		t.Fatal("unexpected trade")
	}
}

func TestPendingOwnershipReleaseReplaysWithoutDoubleAccounting(t *testing.T) {
	x := setup(t)
	c := attach(t, x, nil)
	x.step(t)
	x.fill()
	x.step(t)
	path := filepath.Join(c.Root(), "owners.json")
	backup, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	x.b.onPost = func() {
		if len(x.b.posts) > 1 {
			if err := os.WriteFile(path, []byte("broken registry"), 0600); err != nil {
				t.Error(err)
			}
		}
	}
	x.book(499, false)
	if err = x.e.Step(context.Background(), x.now); err == nil {
		t.Fatal("release failure ignored")
	}
	if x.e.State().Episodes["uid"] != nil || len(x.e.State().Releases) == 0 || x.e.State().Closed != 1 {
		t.Fatal("closure and release intent were not atomic")
	}
	if err = os.WriteFile(path, backup, 0600); err != nil {
		t.Fatal(err)
	}
	restored, err := NewEngine(DefaultConfig(), x.b, x.s, x.f, []density.Instrument{x.i})
	if err != nil {
		t.Fatal(err)
	}
	if err = restored.AttachCoordinator(context.Background(), c); err != nil {
		t.Fatal(err)
	}
	if len(restored.State().Releases) != 0 || restored.State().Closed != 1 || len(x.b.posts) != 2 {
		t.Fatal("recovery duplicated trade/accounting")
	}
}

func TestBrokerBusyFirstCandidateStillTriesFreeNeighbor(t *testing.T) {
	x := setup(t)
	attach(t, x, nil)
	x.b.holdings[x.i.ID] = Holding{Units: 10}
	other := density.Instrument{ID: "zz-sber", Ticker: "SBER", Lot: 10, PriceStep: density.Price{Units: 1}}
	x.e.instruments[other.ID] = other
	b := x.f.Candidates(x.now)[0].Book
	b.InstrumentID = other.ID
	x.f.SetReady(other.ID, true)
	x.f.Book(b, []density.Signal{{InstrumentID: other.ID, Price: density.Price{Units: 100}, Quantity: 1000}}, x.now)
	if err := x.e.Step(context.Background(), x.now); err != nil {
		t.Fatal(err)
	}
	if len(x.b.posts) != 1 || x.b.posts[0].InstrumentID != other.ID {
		t.Fatal("broker-held first candidate blocked free second candidate")
	}
}
