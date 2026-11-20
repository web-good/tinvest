package live

import (
	"math"
	"testing"
	"time"
	"tinvest/internal/service/screener/density"
)

func TestPlanEntryZone(t *testing.T) {
	now := time.Now()
	i := density.Instrument{ID: "x", Ticker: "VTBR", Lot: 1, PriceStep: density.Price{Nano: 10_000_000}}
	signal := density.Signal{InstrumentID: "x", Price: priceFromNano(60_390_000_000)}
	for _, tc := range []struct {
		name     string
		ask      int64
		accepted bool
	}{
		{"inside zone", 60_460_000_000, true},
		{"at boundary", 60_570_000_000, true},
		{"above boundary", 60_580_000_000, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cfg := DefaultConfig()
			cfg.EntryZoneTicks = 18
			cfg.RewardRisk = 1.5
			b := density.Book{InstrumentID: "x", Lot: 1, Time: now, Bids: []density.Level{{Price: signal.Price, Quantity: 100}}, Asks: []density.Level{{Price: priceFromNano(tc.ask), Quantity: 10}}}
			ep, err := Plan(cfg, i, signal, b, now)
			if !tc.accepted {
				if err == nil {
					t.Fatal("entry outside zone accepted")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if ep.EntryPrice != tc.ask || ep.StopPrice != 60_370_000_000 {
				t.Fatalf("wrong entry/protection: %+v", ep)
			}
			wantTake := tc.ask + int64(math.Ceil(float64(tc.ask-ep.StopPrice)/10_000_000*1.5))*10_000_000
			if ep.TakePrice != wantTake {
				t.Fatalf("take=%d want %d", ep.TakePrice, wantTake)
			}
		})
	}
}

func TestConfigRejectsInvalidEntryZoneTicks(t *testing.T) {
	for _, ticks := range []int64{-1, 1000001} {
		cfg := DefaultConfig()
		cfg.EntryZoneTicks = ticks
		if cfg.Validate() == nil {
			t.Fatalf("accepted entry zone ticks %d", ticks)
		}
	}
}

func TestFixedLimitEntry(t *testing.T) {
	now := time.Now()
	cfg := DefaultConfig()
	cfg.FixedLimitEntry = true
	i := density.Instrument{ID: "x", Ticker: "T", Lot: 1, PriceStep: density.Price{Nano: 10_000_000}}
	s := density.Signal{InstrumentID: i.ID, Price: density.Price{Units: 100}, Quantity: 100}
	b := density.Book{InstrumentID: i.ID, Lot: 1, Time: now,
		Bids: []density.Level{{Price: s.Price, Quantity: 100}},
		Asks: []density.Level{{Price: priceFromNano(100_010_000_000), Quantity: 10}},
	}
	ep, err := Plan(cfg, i, s, b, now)
	if err != nil {
		t.Fatal(err)
	}
	if ep.EntryPrice != 100_010_000_000 || ep.StopPrice != 99_800_000_000 {
		t.Fatalf("unexpected entry and stop: %d, %d", ep.EntryPrice, ep.StopPrice)
	}
	f := NewFeed(cfg)
	f.SetReady(i.ID, true)
	f.bookWithCandidates(b, nil, nil, i.PriceStep, now)
	f.Watch(ep, now)
	for _, qty := range []int64{40, 39} {
		now = now.Add(time.Millisecond)
		b.Time, b.Bids[0].Quantity = now, qty
		f.bookWithCandidates(b, nil, nil, i.PriceStep, now)
		want := ""
		if qty == 39 {
			want = "density_lost"
		}
		if got := f.Reason(now); got != want {
			t.Fatalf("volume %d: reason %q, want %q", qty, got, want)
		}
	}
}
