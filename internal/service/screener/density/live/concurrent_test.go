package live

import (
	"context"
	"path/filepath"
	"testing"
	"time"
	"tinvest/internal/service/screener/density"
	"tinvest/internal/service/trading_strategy/livecore/statestore"
)

func addNeighbor(x *fixture) density.Instrument {
	i := density.Instrument{ID: "zz-sber", Ticker: "SBER", Lot: 10, PriceStep: density.Price{Units: 1}}
	x.e.instruments[i.ID] = i
	b := density.Book{InstrumentID: i.ID, Lot: i.Lot, Time: x.now, Bids: []density.Level{{Price: density.Price{Units: 100}, Quantity: 1000}}, Asks: []density.Level{{Price: density.Price{Units: 103}, Quantity: 10}}}
	x.f.SetReady(i.ID, true)
	x.f.Book(b, []density.Signal{{InstrumentID: i.ID, Price: density.Price{Units: 100}, Quantity: 1000}}, x.now)
	return i
}

func TestDensityClosePreservesCandleLotsAndStopsOnSameTicker(t *testing.T) {
	x := setup(t)
	legacy := statestore.New(filepath.Join(t.TempDir(), "candle.json"))
	if err := legacy.Save(map[string]statestore.Entry{"T": {Ticker: "T", Quantity: 50, Strategy: "rsi_pullback", StopOrderID: "candle-stop"}}); err != nil {
		t.Fatal(err)
	}
	attach(t, x, legacy)
	x.b.holdings[x.i.ID] = Holding{Units: 50}
	x.b.stops["candle-stop"] = BrokerStop{ID: "candle-stop", InstrumentID: x.i.ID, Status: "active"}
	x.step(t)
	if len(x.b.posts) != 1 || x.b.posts[0].Kind != "buy" {
		t.Fatalf("density did not buy beside candle slice: %+v", x.b.posts)
	}
	x.fill()
	x.b.holdings[x.i.ID] = Holding{Units: 60}
	x.step(t)
	x.book(499, false)
	x.step(t)
	if len(x.b.posts) != 2 || x.b.posts[1].Lots != 1 || x.b.holdings[x.i.ID].Units != 50 {
		t.Fatalf("density sold foreign inventory: posts=%+v holdings=%+v", x.b.posts, x.b.holdings)
	}
	if x.b.stops["candle-stop"].Status != "active" || x.e.State().Closed != 1 {
		t.Fatal("candle protection changed or density episode did not close")
	}
	notices := x.e.PendingNotifications()
	if len(notices) != 2 || notices[0].Side != "BUY" || notices[1].Side != "SELL" {
		t.Fatalf("real executions missing notifications: %+v", notices)
	}
}

func TestDensityPositionsKeepIndependentProtection(t *testing.T) {
	x := setup(t)
	x.f.cfg.Freshness = time.Hour
	x.e.cfg.Freshness = time.Hour
	attach(t, x, nil)
	other := addNeighbor(x)
	x.step(t)
	x.step(t)
	for _, request := range x.b.posts {
		order := x.b.orders["order-"+request.Key]
		order.Filled, order.Terminal, order.Status = 1, true, "filled"
		order.Amount, order.Fees, order.FeesKnown = 1010*Nano, Nano, true
		x.b.orders[order.ID] = order
		x.b.holdings[request.InstrumentID] = Holding{Units: 10}
	}
	x.step(t)
	if len(x.b.protections) != 4 {
		t.Fatalf("not both positions protected: %+v", x.b.protections)
	}
	x.book(499, false)
	x.step(t)
	state := x.e.State()
	if len(state.Episodes) != 1 || state.Episodes[other.ID] == nil || state.Closed != 1 {
		t.Fatalf("first exit affected neighbor: %+v", state)
	}
	for _, stop := range x.b.stops {
		if stop.InstrumentID == other.ID && stop.Status != "active" {
			t.Fatal("neighbor protection cancelled")
		}
	}
	if x.b.holdings[other.ID].Units != 10 {
		t.Fatal("neighbor inventory sold")
	}
}

func TestForeignInventoryMismatchPreservesDensityProtection(t *testing.T) {
	x := setup(t)
	legacy := statestore.New(filepath.Join(t.TempDir(), "candle.json"))
	attach(t, x, legacy)
	x.step(t)
	x.fill()
	x.step(t)
	// A candle fill has appeared at the broker before its durable state catches
	// up. Keep existing protection while that discrepancy is reconciled.
	x.b.holdings[x.i.ID] = Holding{Units: 60}
	x.book(499, false)
	if err := x.e.Step(context.Background(), x.now); err == nil {
		t.Fatal("unknown foreign inventory accepted")
	}
	if len(x.b.posts) != 1 {
		t.Fatal("sold during ownership discrepancy")
	}
	for _, stop := range x.b.stops {
		if stop.Status != "active" {
			t.Fatal("removed own protection before foreign inventory reconciled")
		}
	}
}

func TestForeignFillDuringStopCancellationRestoresDensityProtection(t *testing.T) {
	x := setup(t)
	legacy := statestore.New(filepath.Join(t.TempDir(), "candle.json"))
	if err := legacy.Save(map[string]statestore.Entry{"T": {Ticker: "T", Quantity: 50}}); err != nil {
		t.Fatal(err)
	}
	attach(t, x, legacy)
	x.b.holdings[x.i.ID] = Holding{Units: 50}
	x.step(t)
	x.fill()
	x.b.holdings[x.i.ID] = Holding{Units: 60}
	x.step(t)
	x.b.onCancelStop = func(string) { x.b.holdings[x.i.ID] = Holding{Units: 50} }
	x.book(499, false)
	if err := x.e.Step(context.Background(), x.now); err == nil {
		t.Fatal("foreign execution race was ignored")
	}
	active := 0
	for _, stop := range x.b.stops {
		if stop.Status == "active" {
			active++
		}
	}
	if active == 0 || len(x.b.posts) != 1 {
		t.Fatal("density left naked after cancelling protection during a foreign fill")
	}
}

func TestDensityOpensNeighborWhileFirstEntryIsPending(t *testing.T) {
	x := setup(t)
	attach(t, x, nil)
	other := addNeighbor(x)
	x.step(t)
	x.step(t)
	if len(x.b.posts) != 2 || x.b.posts[0].InstrumentID != x.i.ID || x.b.posts[1].InstrumentID != other.ID {
		t.Fatalf("expected independent pending entries, got %+v", x.b.posts)
	}
}

func TestFeedKeepsLossForEachInstrument(t *testing.T) {
	x := setup(t)
	other := addNeighbor(x)
	a, err := x.f.Prepare(x.i, density.Signal{InstrumentID: x.i.ID, Price: density.Price{Units: 100}}, x.now)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = x.f.Prepare(other, density.Signal{InstrumentID: other.ID, Price: density.Price{Units: 100}}, x.now); err != nil {
		t.Fatal(err)
	}
	x.book(499, false)
	// Re-selecting a watcher must preserve a loss observed while another ticker
	// was being processed, even when the original density returns.
	x.book(1000, false)
	x.f.Watch(a, x.now)
	if got := x.f.Reason(x.now); got != "density_lost" {
		t.Fatalf("first instrument's loss was erased: %q", got)
	}
}

func TestBrokenEpisodeDoesNotStopNeighborCancellation(t *testing.T) {
	x := setup(t)
	attach(t, x, nil)
	other := addNeighbor(x)
	x.step(t)
	x.step(t)
	if len(x.b.posts) != 2 {
		t.Fatalf("need two entries, got %+v", x.b.posts)
	}
	// Make the first broker order unavailable; the second must still cancel
	// after a stream gap instead of waiting for the first RPC to recover.
	delete(x.b.orders, "order-"+x.b.posts[0].Key)
	x.f.Gap(x.now)
	_ = x.e.Step(context.Background(), x.now.Add(2*time.Second))
	second := "order-" + x.b.posts[1].Key
	found := false
	for _, id := range x.b.canceled {
		found = found || id == second
	}
	if !found {
		t.Fatalf("broken %s prevented cancellation for %s", x.i.ID, other.ID)
	}
}
