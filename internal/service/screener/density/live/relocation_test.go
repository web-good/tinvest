package live

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"
	"tinvest/internal/service/screener/density"
	"tinvest/internal/service/trading_strategy/livecore/accountcoord"
)

func (x *fixture) relocatedBook(level, qty int64, candidate bool) {
	x.now = x.now.Add(time.Millisecond)
	b := density.Book{InstrumentID: x.i.ID, Lot: x.i.Lot, Time: x.now,
		Bids: []density.Level{{Price: density.Price{Units: level}, Quantity: qty}},
		Asks: []density.Level{{Price: density.Price{Units: 110}, Quantity: 10}}}
	var candidates []density.Signal
	if candidate {
		candidates = []density.Signal{{InstrumentID: x.i.ID, Price: density.Price{Units: level}, Quantity: qty, Since: x.now}}
	}
	x.f.BookWithCandidates(b, nil, candidates, x.now)
}

func runStep(t *testing.T, x *fixture) {
	t.Helper()
	if err := x.e.Step(context.Background(), x.now); err != nil {
		t.Fatal(err)
	}
}

func TestPendingEntryFollowsRelocatedDensity(t *testing.T) {
	for _, level := range []int64{101, 99, 96} {
		t.Run(fmt.Sprint(level), func(t *testing.T) {
			x := setup(t)
			runStep(t, x)
			key := x.e.State().Episodes[x.i.ID].Entry.Key
			created := x.e.State().Episodes[x.i.ID].Created
			x.relocatedBook(level, 1000, true)
			runStep(t, x)
			ep := x.e.State().Episodes[x.i.ID]
			if len(x.b.posts) != 2 || len(x.b.canceled) != 1 || ep == nil {
				t.Fatalf("posts=%+v canceled=%v state=%+v", x.b.posts, x.b.canceled, x.e.State())
			}
			if ep.EntryPrice != (level+1)*Nano || ep.StopPrice != (level-2)*Nano || ep.Entry.Key == key {
				t.Fatalf("bad replacement: %+v", ep)
			}
			if !ep.Created.Equal(created) || episodeClaim(ep).Key != key {
				t.Fatalf("replacement changed TTL or ownership: %+v", ep)
			}
			owned := x.e.State().Ownership[x.i.Ticker]
			if len(owned.OrderIDs) != 2 {
				t.Fatalf("lost order identities: %+v", owned)
			}
		})
	}
}

func TestUnconfirmedCandidatesDoNotStartFirstEntry(t *testing.T) {
	x := setup(t)
	x.relocatedBook(100, 1000, true)
	runStep(t, x)
	if len(x.b.posts) != 0 {
		t.Fatal("raw candidate bypassed initial confirmation")
	}
}

func TestReplacementUsesPercentageEntry(t *testing.T) {
	x := setup(t)
	x.i.PriceStep = density.Price{Nano: 10_000_000}
	x.e.instruments[x.i.ID] = x.i
	x.e.cfg.EntryPct, x.f.cfg.EntryPct = .3, .3
	runStep(t, x)
	x.relocatedBook(101, 1000, true)
	runStep(t, x)
	if len(x.b.posts) != 2 || x.b.posts[1].Price != 101_300_000_000 {
		t.Fatalf("wrong percentage replacement: %+v", x.b.posts)
	}
}

func TestReplacementWaitsForConfirmedCancellation(t *testing.T) {
	x := setup(t)
	runStep(t, x)
	x.b.onCancel = func(string) {} // Cancellation accepted; terminal state not visible yet.
	x.relocatedBook(101, 1000, true)
	runStep(t, x)
	if len(x.b.posts) != 1 {
		t.Fatal("new BUY sent before cancellation became terminal")
	}
	id := x.e.State().Episodes[x.i.ID].Entry.ID
	old := x.b.orders[id]
	old.Terminal, old.Status = true, "cancelled"
	x.b.orders[id] = old
	runStep(t, x)
	if len(x.b.posts) != 2 {
		t.Fatal("replacement not resumed after cancellation")
	}
}

func TestReplacementCancelsOutsideAccountLeaseAndReservesSameClaim(t *testing.T) {
	x := setup(t)
	c := attach(t, x, nil)
	runStep(t, x)
	claim := episodeClaim(x.e.State().Episodes[x.i.ID])
	lease, err := c.Lock(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	x.relocatedBook(101, 1000, true)
	runStep(t, x)
	if len(x.b.canceled) != 1 || len(x.b.posts) != 1 {
		t.Fatal("lease blocked cancellation or allowed replacement")
	}
	if err = lease.Close(); err != nil {
		t.Fatal(err)
	}
	x.b.onPost = func() {
		if err := c.Verify(context.Background(), claim); err != nil {
			t.Error(err)
		}
		unexpected, err := c.TryLock()
		if !errors.Is(err, accountcoord.ErrBusy) {
			if unexpected != nil {
				unexpected.Close()
			}
			t.Error("replacement BUY sent without account lease", err)
		}
	}
	runStep(t, x)
	if len(x.b.posts) != 2 || episodeClaim(x.e.State().Episodes[x.i.ID]) != claim {
		t.Fatal("claim changed during replacement")
	}
}

func TestMoveDuringReplacementAllocationIsRevalidated(t *testing.T) {
	x := setup(t)
	runStep(t, x)
	x.relocatedBook(101, 1000, true)
	x.b.onCanBuy = func() { x.relocatedBook(102, 1000, true) }
	runStep(t, x)
	if len(x.b.posts) != 1 {
		t.Fatal("submitted outdated replacement")
	}
	x.b.onCanBuy = nil
	runStep(t, x)
	if len(x.b.posts) != 2 || x.b.posts[1].Price != 103*Nano {
		t.Fatalf("missing current replacement: %+v", x.b.posts)
	}
}

func TestRelocationDoesNotExtendEntryTTL(t *testing.T) {
	x := setup(t)
	x.e.cfg.EntryTTL = time.Millisecond
	runStep(t, x)
	x.relocatedBook(101, 1000, true)
	runStep(t, x)
	if len(x.b.posts) != 1 || x.e.State().Episodes[x.i.ID] != nil {
		t.Fatal("relocation prolonged expired entry")
	}
}

func TestReplacementOwnershipSurvivesRestart(t *testing.T) {
	x := setup(t)
	c := attach(t, x, nil)
	runStep(t, x)
	claim := episodeClaim(x.e.State().Episodes[x.i.ID])
	x.relocatedBook(101, 1000, true)
	runStep(t, x)
	feed := NewFeed(DefaultConfig())
	restarted, err := NewEngine(DefaultConfig(), x.b, x.s, feed, []density.Instrument{x.i})
	if err != nil {
		t.Fatal(err)
	}
	if err = restarted.AttachCoordinator(context.Background(), c); err != nil {
		t.Fatal(err)
	}
	ep := restarted.State().Episodes[x.i.ID]
	if episodeClaim(ep) != claim || ep.Entry.Key == claim.Key || len(ep.RetiredEntries) != 1 || len(restarted.State().Ownership[x.i.Ticker].OrderIDs) != 2 {
		t.Fatalf("lost replacement ownership: %+v", ep)
	}
}

func TestOpenPositionKeepsProtectionWhenDensityMoves(t *testing.T) {
	x := setup(t)
	runStep(t, x)
	x.fill()
	x.now = x.now.Add(2 * time.Second)
	x.book(1000, false)
	runStep(t, x)
	ep := x.e.State().Episodes[x.i.ID]
	entry, stop, take := ep.EntryPrice, ep.StopPrice, ep.TakePrice
	x.relocatedBook(101, 750, true)
	runStep(t, x)
	if ep.Reason != "" || ep.EntryPrice != entry || ep.StopPrice != stop || ep.TakePrice != take || len(x.b.posts) != 1 || len(x.b.canceled) != 0 {
		t.Fatalf("move changed open trade: %+v posts=%v", ep, x.b.posts)
	}
	x.relocatedBook(102, 500, true)
	runStep(t, x)
	x.relocatedBook(102, 500, false)
	if reason := x.f.Reason(x.now); reason != "" {
		t.Fatalf("forgot new density location: %s", reason)
	}
	x.relocatedBook(102, 499, false)
	runStep(t, x)
	if len(x.b.posts) != 2 || x.b.posts[1].Kind != "exit" {
		t.Fatalf("missing early exit: %+v", x.b.posts)
	}
}

type replacementTimeoutBroker struct {
	Broker
	posts int
}

func (b *replacementTimeoutBroker) PostOrder(ctx context.Context, r OrderRequest) (Order, error) {
	b.posts++
	got, err := b.Broker.PostOrder(ctx, r)
	if err == nil && b.posts == 2 {
		return Order{}, errors.New("replacement response lost")
	}
	return got, err
}

func (b *replacementTimeoutBroker) CancelOrder(ctx context.Context, id string, request bool) error {
	// The basic fixture cancels by exchange ID only; model the real adapter's
	// ability to resolve a request UUID when the placement response was lost.
	if request {
		got, err := b.Order(ctx, id, true)
		if err != nil {
			return err
		}
		id = got.ID
	}
	return b.Broker.CancelOrder(ctx, id, false)
}

func TestUnknownReplacementPlacementIsReconciledAfterRestart(t *testing.T) {
	x := setup(t)
	c := attach(t, x, nil)
	b := &replacementTimeoutBroker{Broker: x.b}
	x.e.broker = b
	runStep(t, x)
	x.relocatedBook(101, 1000, true)
	if err := x.e.Step(context.Background(), x.now); err == nil {
		t.Fatal("missing uncertain-placement error")
	}
	ep := x.e.State().Episodes[x.i.ID]
	key := ep.Entry.Key
	if ep.Entry.ID != "" || !ep.Entry.Sent || len(x.b.posts) != 2 {
		t.Fatal("uncertain request identity lost")
	}
	f := NewFeed(DefaultConfig())
	restarted, err := NewEngine(DefaultConfig(), b, x.s, f, []density.Instrument{x.i})
	if err != nil {
		t.Fatal(err)
	}
	if err = restarted.AttachCoordinator(context.Background(), c); err != nil {
		t.Fatal(err)
	}
	restarted.Now = func() time.Time { return x.now }
	if err = restarted.Step(context.Background(), x.now); err != nil {
		t.Fatal(err)
	}
	if len(x.b.posts) != 2 || restarted.State().Episodes[x.i.ID] != nil || x.b.orders["order-"+key].Status != "cancelled" {
		t.Fatal("restart duplicated or abandoned uncertain replacement")
	}
}

func TestFillDuringRelocationCancellationDoesNotBuyAgain(t *testing.T) {
	x := setup(t)
	runStep(t, x)
	x.b.onCancel = func(string) { x.fill() }
	x.relocatedBook(101, 1000, true)
	runStep(t, x)
	ep := x.e.State().Episodes[x.i.ID]
	if ep == nil || ep.Entry.Filled != 1 || ep.Reason != "" || ep.StopPrice != 98*Nano || len(x.b.posts) != 1 || len(x.b.protections) != 2 {
		t.Fatalf("fill race mishandled: ep=%+v posts=%+v protections=%+v", ep, x.b.posts, x.b.protections)
	}
}
