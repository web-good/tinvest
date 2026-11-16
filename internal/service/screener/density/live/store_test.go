package live

import (
	"os"
	"path/filepath"
	"testing"
	"time"
	"tinvest/internal/service/screener/density"

	"github.com/google/uuid"
)

func TestStoreRecoversAndExcludesAnotherProcess(t *testing.T) {
	root, account := t.TempDir(), uuid.NewString()
	s, err := OpenStore(root, account)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = OpenStore(t.TempDir(), account); err == nil {
		t.Fatal("second directory bypassed account lock")
	}
	state := s.State()
	state.Episodes["uid"] = &Episode{Instrument: density.Instrument{ID: "uid", Ticker: "T", Lot: 10}, Entry: Order{Key: "stable-key", Sent: true}}
	if err = s.Save(state, "entry_intent"); err != nil {
		t.Fatal(err)
	}
	if err = s.Close(); err != nil {
		t.Fatal(err)
	}
	s, err = OpenStore(root, account)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	if s.State().Episodes["uid"].Entry.Key != "stable-key" || !s.State().Episodes["uid"].Entry.Sent {
		t.Fatal("lost uncertain intent")
	}
	if err = os.WriteFile(filepath.Join(s.Dir, "events.jsonl"), []byte("{broken"), 0600); err != nil {
		t.Fatal(err)
	}
	s.Close()
	if _, err = OpenStore(root, account); err == nil {
		t.Fatal("corrupt journal silently discarded")
	}
}

func TestFeedRemembersDensityLossDuringRPC(t *testing.T) {
	now := time.Now()
	f := NewFeed(DefaultConfig())
	i := density.Instrument{ID: "uid", Ticker: "T", Lot: 10, PriceStep: density.Price{Units: 1}}
	book := density.Book{InstrumentID: i.ID, Lot: i.Lot, Time: now, Bids: []density.Level{{Price: density.Price{Units: 100}, Quantity: 1000}}, Asks: []density.Level{{Price: density.Price{Units: 103}, Quantity: 10}}}
	f.SetReady(i.ID, true)
	f.Book(book, nil, now)
	f.Watch(&Episode{Instrument: i, Density: 100e9, DensityLots: 1000, StopPrice: 98e9}, now)
	book.Time = now.Add(time.Millisecond)
	book.Bids[0].Quantity = 499
	f.Book(book, nil, book.Time)
	book.Time = now.Add(2 * time.Millisecond)
	book.Bids[0].Quantity = 1000
	f.Book(book, nil, book.Time)
	if got := f.Reason(now.Add(3 * time.Millisecond)); got != "density_lost" {
		t.Fatalf("loss was overwritten: %q", got)
	}
}

func TestPlanUsesOneLotAndPaperPrices(t *testing.T) {
	now := time.Now()
	cfg := DefaultConfig()
	i := density.Instrument{ID: "uid", Ticker: "SBER", Lot: 10, PriceStep: density.Price{Nano: 10000000}}
	b := density.Book{InstrumentID: i.ID, Lot: 10, Time: now, Bids: []density.Level{{Price: density.Price{Units: 100}, Quantity: 1000}}, Asks: []density.Level{{Price: density.Price{Units: 101}, Quantity: 10}}}
	ep, err := Plan(cfg, i, density.Signal{InstrumentID: i.ID, Price: density.Price{Units: 100}, Quantity: 1000}, b, now)
	if err != nil {
		t.Fatal(err)
	}
	if ep.Entry.Lots != 1 || ep.Instrument.Lot != 10 || ep.EntryPrice != 100010000000 || ep.StopPrice != 99980000000 || ep.TakePrice != 100070000000 {
		t.Fatalf("wrong plan: %+v", ep)
	}
}

func TestFeedLatchesStaleIntervalBeforeReplacingBook(t *testing.T) {
	x := setup(t)
	x.step(t)
	x.now = x.now.Add(5 * time.Second)
	x.book(1000, false)
	if x.f.Reason(x.now) == "" {
		t.Fatal("fresh replacement erased the stale interval during RPC")
	}
}
func TestDirectorySyncFailsForNonDirectory(t *testing.T) {
	path := filepath.Join(t.TempDir(), "file")
	if err := os.WriteFile(path, []byte("x"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := syncDirectoryChain(path); err == nil {
		t.Fatal("non-directory accepted for durability sync")
	}
}

func TestPlanRejectsOverflowingTickOffset(t *testing.T) {
	now := time.Now()
	c := DefaultConfig()
	c.EntryTicks = 10000
	c.StopTicks = 1
	i := density.Instrument{ID: "large", Ticker: "LARGE", Lot: 1, PriceStep: density.Price{Units: 1000000}}
	b := density.Book{InstrumentID: i.ID, Lot: 1, Time: now, Bids: []density.Level{{Price: density.Price{Units: 90000000}, Quantity: 1000}}, Asks: []density.Level{{Price: density.Price{Units: 91000000}, Quantity: 1}}}
	if ep, err := Plan(c, i, density.Signal{InstrumentID: i.ID, Price: b.Bids[0].Price, Quantity: 1000}, b, now); err == nil {
		t.Fatalf("overflow yielded a tradable plan: %+v", ep)
	}
}
