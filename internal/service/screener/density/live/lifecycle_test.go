package live

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestBrokerStopAndTakeExecutionCannotCauseSecondSale(t *testing.T) {
	for _, kind := range []string{"stop", "take"} {
		t.Run(kind, func(t *testing.T) {
			x := setup(t)
			x.step(t)
			x.fill()
			x.step(t)
			ep := x.e.State().Episodes["uid"]
			stop := ep.Stop
			if kind == "take" {
				stop = ep.Take
			}
			child := "child-" + kind
			x.b.stops[stop.ID] = BrokerStop{ID: stop.ID, Status: "executed", ChildID: child, Activated: true}
			x.b.orders[child] = Order{ID: child, Kind: "sell", Lots: 1, Filled: 1, Terminal: true, Status: "filled", Amount: 1070 * Nano, Fees: Nano}
			x.b.holdings["uid"] = Holding{}
			x.step(t)
			if len(x.b.posts) != 1 || x.e.State().Closed != 1 || x.e.State().Episodes["uid"] != nil || x.e.State().Realized != 58*Nano {
				t.Fatalf("duplicate sale or wrong result: %+v posts=%+v", x.e.State(), x.b.posts)
			}
			want := "stop"
			if kind == "take" {
				want = "take_profit"
			}
			pending := x.e.PendingNotifications()
			if len(pending) != 2 || pending[1].Reason != want {
				t.Fatalf("wrong broker exit reason: %+v", pending)
			}
		})
	}
}
func TestActivatedTakeRemainsPendingUntilActualFill(t *testing.T) {
	x := setup(t)
	x.step(t)
	x.fill()
	x.step(t)
	id := x.e.State().Episodes["uid"].Take.ID
	x.b.stops[id] = BrokerStop{ID: id, Status: "executed", ChildID: "take-child", Activated: true}
	x.b.orders["take-child"] = Order{ID: "take-child", Kind: "sell", Lots: 1, Status: "new"}
	x.step(t)
	if x.e.State().Episodes["uid"] == nil || x.e.State().Episodes["uid"].Remaining() != 1 || x.e.State().Closed != 0 {
		t.Fatal("activation mistaken for fill")
	}
	x.book(499, false)
	x.step(t)
	if len(x.b.posts) != 2 || x.e.State().Closed != 1 {
		t.Fatal("pending take not canceled before market exit")
	}
}
func TestRejectedProtectionClosesConfirmedInventory(t *testing.T) {
	x := setup(t)
	x.step(t)
	x.fill()
	x.b.stopErr = errors.Join(ErrRejected, errors.New("bad stop price"))
	x.step(t)
	if x.e.State().Closed != 1 || len(x.b.posts) != 2 {
		t.Fatal("rejected protection left silently open")
	}
}
func TestJournalFailureForbidsNextOrder(t *testing.T) {
	x := setup(t)
	x.s.Close()
	if err := x.e.Step(context.Background(), x.now); err == nil {
		t.Fatal("closed journal ignored")
	}
	if len(x.b.posts) != 0 {
		t.Fatal("order sent without durable intent")
	}
}
func TestMarketExitUnknownResultIsQueriedNotReposted(t *testing.T) {
	x := setup(t)
	x.step(t)
	x.fill()
	x.step(t)
	ep := x.e.State().Episodes["uid"]
	ep.Reason = "shutdown"
	ep.Exit = &Order{Key: "exit-key", Kind: "exit", Lots: 1, Sent: true}
	for _, s := range []*Stop{&ep.Stop, &ep.Take} {
		v := x.b.stops[s.ID]
		v.Status = "cancelled"
		x.b.stops[s.ID] = v
	}
	x.b.orders["already-sold"] = Order{ID: "already-sold", Key: "exit-key", Kind: "sell", Lots: 1, Filled: 1, Terminal: true, Status: "filled", Amount: 990 * Nano, Fees: Nano}
	x.b.holdings["uid"] = Holding{}
	x.step(t)
	if x.e.State().Closed != 1 || len(x.b.posts) != 1 {
		t.Fatal("uncertain market order duplicated")
	}
}
func TestShutdownCancelsPendingOrder(t *testing.T) {
	x := setup(t)
	x.step(t)
	x.e.Now = time.Now
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := x.e.Shutdown(ctx); err != nil {
		t.Fatal(err)
	}
	if x.e.State().Episodes["uid"] != nil || len(x.b.canceled) != 1 {
		t.Fatal("pending left open on shutdown")
	}
}
func TestRejectedActivatedTakeRequestsExit(t *testing.T) {
	x := setup(t)
	x.step(t)
	x.fill()
	x.step(t)
	ep := x.e.State().Episodes["uid"]
	id := ep.Take.ID
	x.b.stops[id] = BrokerStop{ID: id, Status: "executed", ChildID: "rejected-take", Activated: true}
	x.b.orders["rejected-take"] = Order{ID: "rejected-take", Lots: 1, Kind: "sell", Terminal: true, Status: "rejected"}
	x.step(t)
	if x.e.State().Closed != 1 {
		t.Fatal("failed take left silently without a working take")
	}
}

func TestFinishRefreshesFinalEntryFees(t *testing.T) {
	x := setup(t)
	x.step(t)
	x.fill()
	x.step(t)
	id := x.e.State().Episodes["uid"].Entry.ID
	o := x.b.orders[id]
	o.Fees = 2 * Nano
	x.b.orders[id] = o
	x.book(499, false)
	x.step(t)
	if x.e.State().Realized != -23*Nano {
		t.Fatalf("stale commission used: %+v", x.e.State())
	}
}
func TestMissingCommissionDoesNotClaimCompletePnL(t *testing.T) {
	x := setup(t)
	x.step(t)
	x.fill()
	id := x.b.posts[0].Key
	for k, o := range x.b.orders {
		if o.Key == id {
			o.Fees = 0
			x.b.orders[k] = o
		}
	}
	x.step(t)
	x.book(499, false)
	x.step(t)
	if x.e.State().PnLComplete {
		t.Fatal("absent execution commission counted as known zero")
	}
}

func TestExpiredConfirmationDuringAllocationCheckCannotEnter(t *testing.T) {
	x := setup(t)
	x.b.onCanBuy = func() { x.book(499, false); x.book(1000, false) }
	if err := x.e.Step(context.Background(), x.now); err != nil {
		t.Fatal(err)
	}
	if len(x.b.posts) != 0 || x.e.State().Episodes["uid"] != nil {
		t.Fatal("used old density confirmation after its disappearance during preflight")
	}
}
