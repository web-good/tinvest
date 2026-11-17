package paper

import (
	"math"
	"testing"
	"time"
	"tinvest/internal/service/screener/density"
)

func TestPercentageEntryAndFill(t *testing.T) {
	cfg := DefaultConfig()
	cfg.EntryPct = .3
	cfg.Latency = 0
	e, err := New(cfg, []density.Instrument{{ID: "x", Ticker: "X", Lot: 1, PriceStep: density.Price{Nano: 10_000_000}}}, nil)
	if err != nil {
		t.Fatal(err)
	}
	e.OnBook(book(start, 100), start)
	e.Observe(density.Signal{InstrumentID: "x", Price: p(100), Quantity: 100}, start)
	o := e.orders["x"]
	if o == nil {
		t.Fatal("no entry")
	}
	if o.Entry != 100_300_000_000 {
		t.Fatalf("entry = %d, want 100.30 RUB", o.Entry)
	}
	at := start.Add(time.Millisecond)
	e.OnTrade(density.Event{InstrumentID: "x", Price: density.Price{Units: 100, Nano: 300_000_000}, Quantity: 1, Time: at}, at)
	if o.State != "pending" {
		t.Fatal("touch filled entry")
	}
	at = at.Add(time.Millisecond)
	e.OnTrade(density.Event{InstrumentID: "x", Price: density.Price{Units: 100, Nano: 290_000_000}, Quantity: 1, Time: at}, at)
	if o.State != "open" || o.Entry != 100_300_000_000 {
		t.Fatalf("percentage entry not filled: %+v", o)
	}
}

func TestConfigRejectsInvalidEntryPct(t *testing.T) {
	for _, pct := range []float64{-1, math.NaN(), math.Inf(1), 101} {
		cfg := DefaultConfig()
		cfg.EntryPct = pct
		if cfg.Validate() == nil {
			t.Fatalf("accepted entry percentage %v", pct)
		}
	}
}
