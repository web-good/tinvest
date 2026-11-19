package density

import (
	"testing"
	"time"
)

func TestDetectorPersistenceAndExpiry(t *testing.T) {
	cfg := DefaultConfig()
	d := NewDetector(cfg)
	d.SetInstruments([]Instrument{{ID: "x", PriceStep: Price{Nano: 250_000_000}}})
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	book := Book{InstrumentID: "x", Lot: 100, Asks: []Level{{Price: Price{Units: 251}, Quantity: 10}}, Bids: []Level{{Price: Price{Units: 250}, Quantity: 100}, {Price: Price{Units: 247, Nano: 500000000}, Quantity: 203}, {Price: Price{Units: 247}, Quantity: 10}, {Price: Price{Units: 246}, Quantity: 10}, {Price: Price{Units: 245}, Quantity: 10}, {Price: Price{Units: 244}, Quantity: 10}, {Price: Price{Units: 243}, Quantity: 10}}}
	for i := 0; i <= 60; i++ {
		book.Time = now.Add(time.Duration(i) * time.Second)
		got := d.Update(book, book.Time)
		if i < 60 && len(got) != 0 {
			t.Fatal("premature signal")
		}
		if i == 60 && (len(got) != 1 || got[0].Price != book.Bids[1].Price) {
			t.Fatalf("boundary signal: %+v", got)
		}
	}
	d.Expire(now.Add(64 * time.Second))
	book.Time = now.Add(64 * time.Second)
	if got := d.Update(book, book.Time); len(got) != 0 {
		t.Fatal("expired observation survived")
	}
}

func TestDetectorRejectsInvalidBooksAndResets(t *testing.T) {
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	book := Book{InstrumentID: "x", Lot: 100, Asks: []Level{{Price: Price{Units: 251}, Quantity: 1}}, Bids: []Level{{Price: Price{Units: 250}, Quantity: 300}, {Price: Price{Units: 249}, Quantity: 10}, {Price: Price{Units: 248}, Quantity: 10}, {Price: Price{Units: 247}, Quantity: 10}, {Price: Price{Units: 246}, Quantity: 10}, {Price: Price{Units: 245}, Quantity: 10}}}
	for _, tc := range []struct {
		name   string
		change func(*Book)
	}{
		{"crossed", func(b *Book) { b.Asks[0].Price = Price{Units: 250} }},
		{"insufficient neighbors", func(b *Book) { b.Bids = b.Bids[:5] }},
		{"below threshold", func(b *Book) { b.Bids[0].Quantity = 1 }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			d := NewDetector(DefaultConfig())
			d.SetInstruments([]Instrument{{ID: "x", PriceStep: Price{Nano: 250_000_000}}})
			b := book
			b.Bids = append([]Level(nil), book.Bids...)
			b.Asks = append([]Level(nil), book.Asks...)
			b.Time = now
			d.Update(b, now)
			b.Time = now.Add(time.Second)
			tc.change(&b)
			d.Update(b, b.Time)
			if len(d.observations) != 0 {
				t.Fatal("invalid book retained observation")
			}
		})
	}
}

func TestDetectorIgnoresReorderedTimestamp(t *testing.T) {
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	d := NewDetector(DefaultConfig())
	d.SetInstruments([]Instrument{{ID: "x", PriceStep: Price{Nano: 250_000_000}}})
	b := Book{InstrumentID: "x", Lot: 100, Time: now, Asks: []Level{{Price: Price{Units: 251}, Quantity: 1}}, Bids: []Level{{Price: Price{Units: 250}, Quantity: 300}, {Price: Price{Units: 249}, Quantity: 10}, {Price: Price{Units: 248}, Quantity: 10}, {Price: Price{Units: 247}, Quantity: 10}, {Price: Price{Units: 246}, Quantity: 10}, {Price: Price{Units: 245}, Quantity: 10}}}
	d.Update(b, now)
	d.Update(b, now.Add(2*time.Second))
	for _, o := range d.observations {
		if !o.last.Equal(now) {
			t.Fatal("duplicate timestamp advanced freshness")
		}
	}
	d.Expire(now.Add(4 * time.Second))
	if len(d.observations) != 0 {
		t.Fatal("duplicate kept observation alive")
	}
}

func TestExactPriceOrdering(t *testing.T) {
	a := Price{Units: 1000000000, Nano: 1}
	b := Price{Units: 1000000000, Nano: 2}
	if comparePrice(a, b) >= 0 || comparePrice(b, a) <= 0 || comparePrice(a, a) != 0 {
		t.Fatal("price comparison lost nanounit precision")
	}
}

func TestDetectorExposesCandidatesBeforeMaturity(t *testing.T) {
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	b := Book{InstrumentID: "x", Lot: 100, Time: now, Asks: []Level{{Price: Price{Units: 251}, Quantity: 1}}, Bids: []Level{{Price: Price{Units: 250}, Quantity: 200}, {Price: Price{Units: 249}, Quantity: 10}, {Price: Price{Units: 248}, Quantity: 10}, {Price: Price{Units: 247}, Quantity: 10}, {Price: Price{Units: 246}, Quantity: 10}, {Price: Price{Units: 245}, Quantity: 10}}}
	d := NewDetector(DefaultConfig())
	d.SetInstruments([]Instrument{{ID: "x", PriceStep: Price{Nano: 250_000_000}}})
	if len(d.Update(b, now)) != 0 {
		t.Fatal("premature")
	}
	got := d.Candidates()
	if len(got) != 1 || got[0].Rubles != 5_000_000 {
		t.Fatalf("candidate %+v", got)
	}
	d.Expire(now.Add(4 * time.Second))
	if len(d.Candidates()) != 0 {
		t.Fatal("stale candidate")
	}
}

func TestDetectorPriceChangeStartsNewEpisode(t *testing.T) {
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	cfg := DefaultConfig()
	cfg.Persistence = time.Second
	b := Book{InstrumentID: "x", Lot: 100, Time: now, Asks: []Level{{Price: Price{Units: 251}, Quantity: 1}}, Bids: []Level{{Price: Price{Units: 250}, Quantity: 250}, {Price: Price{Units: 249}, Quantity: 10}, {Price: Price{Units: 248}, Quantity: 10}, {Price: Price{Units: 247}, Quantity: 10}, {Price: Price{Units: 246}, Quantity: 10}, {Price: Price{Units: 245}, Quantity: 10}}}
	d := NewDetector(cfg)
	d.SetInstruments([]Instrument{{ID: "x", PriceStep: Price{Nano: 250_000_000}}})
	d.Update(b, now)
	b.Time = now.Add(time.Second)
	if len(d.Update(b, b.Time)) != 1 {
		t.Fatal("expected mature observation")
	}
	b.Bids[0].Price.Nano = 1
	b.Time = now.Add(2 * time.Second)
	if len(d.Update(b, b.Time)) != 0 {
		t.Fatal("new price inherited old age")
	}
	if len(d.Candidates()) != 1 || d.Candidates()[0].Price.Nano != 1 {
		t.Fatal("new price not tracked exactly")
	}
}

func TestDetectorInclusiveVolumeAndMultiple(t *testing.T) {
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	cfg := DefaultConfig()
	cfg.Persistence = time.Nanosecond
	b := Book{InstrumentID: "x", Lot: 100, Time: now, Asks: []Level{{Price: Price{Units: 251}, Quantity: 1}}, Bids: []Level{{Price: Price{Units: 250}, Quantity: 200}, {Price: Price{Units: 200}, Quantity: 50}, {Price: Price{Units: 125}, Quantity: 80}, {Price: Price{Units: 100}, Quantity: 100}, {Price: Price{Units: 80}, Quantity: 125}, {Price: Price{Units: 50}, Quantity: 200}}}
	d := NewDetector(cfg)
	d.SetInstruments([]Instrument{{ID: "x", PriceStep: Price{Nano: 250_000_000}}})
	d.Update(b, now)
	b.Time = now.Add(time.Second)
	sig := d.Update(b, b.Time)
	if len(sig) != 1 || sig[0].Rubles != 5_000_000 || sig[0].Multiple != 5 {
		t.Fatalf("inclusive thresholds: %+v", sig)
	}
}

func TestDetectorExpiresByExchangeTimestamp(t *testing.T) {
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	b := Book{InstrumentID: "x", Lot: 100, Time: now.Add(-2 * time.Second), Asks: []Level{{Price: Price{Units: 251}, Quantity: 1}}, Bids: []Level{{Price: Price{Units: 250}, Quantity: 200}, {Price: Price{Units: 249}, Quantity: 10}, {Price: Price{Units: 248}, Quantity: 10}, {Price: Price{Units: 247}, Quantity: 10}, {Price: Price{Units: 246}, Quantity: 10}, {Price: Price{Units: 245}, Quantity: 10}}}
	d := NewDetector(DefaultConfig())
	d.SetInstruments([]Instrument{{ID: "x", PriceStep: Price{Nano: 250_000_000}}})
	d.Update(b, now)
	if len(d.Candidates()) != 1 {
		t.Fatal("fresh book rejected")
	}
	d.Expire(now.Add(2 * time.Second))
	if len(d.Candidates()) != 0 {
		t.Fatal("exchange timestamp expired but observation remains")
	}
}

func TestDetectorUsesTwentyFiveTicksFromBestBid(t *testing.T) {
	for _, tc := range []struct {
		name  string
		step  Price
		bid25 Price
		bid26 Price
	}{
		{"fractional step", Price{Nano: 30_000_000}, Price{Units: 251, Nano: 650_000_000}, Price{Units: 251, Nano: 680_000_000}},
		{"whole step", Price{Units: 1}, Price{Units: 275, Nano: 900_000_000}, Price{Units: 276, Nano: 900_000_000}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cfg := DefaultConfig()
			cfg.Persistence = time.Second
			d := NewDetector(cfg)
			d.SetInstruments([]Instrument{{ID: "x", PriceStep: tc.step}})
			now := time.Date(2026, 11, 19, 10, 0, 0, 0, time.UTC)
			level := Price{Units: 250, Nano: 900_000_000}
			book := Book{
				InstrumentID: "x", Lot: 100,
				Asks: []Level{{Price: Price{Units: 278}, Quantity: 1}},
				Bids: []Level{{Quantity: 10}, {Price: level, Quantity: 250}},
			}
			for offset := range int64(5) {
				book.Bids = append(book.Bids, Level{Price: Price{Units: 250 - offset}, Quantity: 10})
			}
			update := func(seconds int, bid Price) []Signal {
				book.Time = now.Add(time.Duration(seconds) * time.Second)
				book.Bids[0].Price = bid
				return d.Update(book, book.Time)
			}
			update(0, tc.bid26)
			if got := d.Candidates(); len(got) != 0 {
				t.Fatalf("26-tick level tracked: %+v", got)
			}
			if got := update(1, tc.bid25); len(got) != 0 {
				t.Fatalf("25-tick level confirmed before persistence: %+v", got)
			}
			if got := update(2, tc.bid25); len(got) != 1 || got[0].Price != level {
				t.Fatalf("expected 25-tick level, got %+v", got)
			}
			update(3, tc.bid26)
			if got := d.Candidates(); len(got) != 0 {
				t.Fatalf("26-tick level remained eligible for delivery: %+v", got)
			}
		})
	}
}
