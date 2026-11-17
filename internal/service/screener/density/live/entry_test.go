package live

import (
	"math"
	"testing"
	"time"
	"tinvest/internal/service/screener/density"
)

func TestPlanPercentageEntry(t *testing.T) {
	now := time.Now()
	i := density.Instrument{ID: "x", Ticker: "X", Lot: 1, PriceStep: density.Price{Nano: 10_000_000}}
	s := density.Signal{InstrumentID: "x", Price: density.Price{Units: 100}}
	b := density.Book{InstrumentID: "x", Lot: 1, Time: now,
		Bids: []density.Level{{Price: s.Price, Quantity: 100}},
		Asks: []density.Level{{Price: density.Price{Units: 101}, Quantity: 1}}}
	for _, tc := range []struct {
		name string
		pct  float64
		want int64
	}{
		{"ticks fallback", 0, 100_010_000_000},
		{"percentage replaces ticks", .3, 100_300_000_000},
		{"round down to step", .305, 100_300_000_000},
		{"at least one tick", .001, 100_010_000_000},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cfg := DefaultConfig()
			cfg.EntryPct = tc.pct
			ep, err := Plan(cfg, i, s, b, now)
			if err != nil {
				t.Fatal(err)
			}
			if ep.EntryPrice != tc.want {
				t.Fatalf("entry = %d, want %d", ep.EntryPrice, tc.want)
			}
			if ep.StopPrice != 99_980_000_000 {
				t.Fatalf("stop moved: %d", ep.StopPrice)
			}
			if ep.TakePrice != ep.EntryPrice+2*(ep.EntryPrice-ep.StopPrice) {
				t.Fatalf("wrong take: %d", ep.TakePrice)
			}
		})
	}
	cfg := DefaultConfig()
	cfg.EntryPct = 1
	if _, err := Plan(cfg, i, s, b, now); err == nil {
		t.Fatal("entry at ask accepted")
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
