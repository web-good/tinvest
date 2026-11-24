package paper

import (
	"testing"
	"time"
	"tinvest/internal/service/screener/density"
)

func TestEntryZoneAndFill(t *testing.T) {
	cfg := DefaultConfig()
	cfg.EntryZoneTicks = 18
	cfg.Latency = 0
	cfg.MaxEntryLots = 1
	var fills []Record
	e, err := New(cfg, []density.Instrument{{ID: "x", Ticker: "VTBR", Lot: 1, PriceStep: density.Price{Nano: 10_000_000}}}, func(r Record) error {
		if r.Kind == "entry_filled" {
			fills = append(fills, r)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	q := func(n int64) density.Price { return density.Price{Units: n / nano, Nano: int32(n % nano)} }
	b := density.Book{InstrumentID: "x", Lot: 1, Time: start, Bids: []density.Level{{Price: q(60_390_000_000), Quantity: 100}}, Asks: []density.Level{{Price: q(60_580_000_000), Quantity: 10}}}
	signal := density.Signal{InstrumentID: "x", Price: b.Bids[0].Price, Quantity: 100}
	e.OnBook(b, start)
	e.Observe(signal, start)
	if e.orders["x"] != nil {
		t.Fatal("placed entry above interest zone")
	}
	b.Time = start.Add(time.Millisecond)
	b.Asks[0].Price = q(60_460_000_000)
	received := b.Time.Add(100 * time.Millisecond)
	e.OnBook(b, received)
	e.Observe(signal, received)
	o := e.orders["x"]
	if o == nil || o.State != "open" || o.Entry != 60_460_000_000 || o.Stop != 60_370_000_000 || o.Take != 60_650_000_000 {
		t.Fatalf("wrong fill: %+v", o)
	}
	if len(fills) != 1 || fills[0].FillRubles != 60.46 || fills[0].TakeRubles != 60.65 {
		t.Fatalf("missing fill alert: %+v", fills)
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

func TestEntryZoneWaitsForLatencyAndUsesAvailableAsk(t *testing.T) {
	cfg := DefaultConfig()
	cfg.EntryZoneTicks = 20
	cfg.Latency = time.Second
	cfg.MaxEntryLots = 1
	e, err := New(cfg, []density.Instrument{{ID: "x", Ticker: "X", Lot: 1, PriceStep: density.Price{Nano: 10_000_000}}}, nil)
	if err != nil {
		t.Fatal(err)
	}
	b := book(start, 100)
	b.Bids = b.Bids[1:]
	b.Asks[0].Price = density.Price{Units: 100, Nano: 200_000_000}
	e.OnBook(b, start)
	e.Observe(density.Signal{InstrumentID: "x", Price: p(100), Quantity: 100}, start)
	if o := e.orders["x"]; o == nil || o.State != "pending" {
		t.Fatalf("latency bypassed: %+v", o)
	}
	if e.orders["x"].Take != 100_640_000_000 {
		t.Fatalf("wrong planned take: %+v", e.orders["x"])
	}
	b.Time = start.Add(time.Second)
	b.Asks[0].Price = density.Price{Units: 100, Nano: 100_000_000}
	e.OnBook(b, b.Time)
	o := e.orders["x"]
	if o.State != "open" || o.Entry != 100_100_000_000 || o.Take != 100_410_000_000 {
		t.Fatalf("wrong delayed fill: %+v", o)
	}
}

func TestTakeProfitHasMinimumReturnRoundedUpToPriceStep(t *testing.T) {
	for _, tc := range []struct {
		name                    string
		entry, stop, step, want int64
	}{
		{name: "minimum exactly on tick", entry: 100_000_000_000, stop: 99_990_000_000, step: 10_000_000, want: 100_300_000_000},
		{name: "minimum rounded up", entry: 100_010_000_000, stop: 100_000_000_000, step: 10_000_000, want: 100_320_000_000},
		{name: "larger reward risk target preserved", entry: 100_000_000_000, stop: 99_000_000_000, step: 10_000_000, want: 102_000_000_000},
		{name: "coarser price step", entry: 100_050_000_000, stop: 100_000_000_000, step: 50_000_000, want: 100_400_000_000},
	} {
		t.Run(tc.name, func(t *testing.T) {
			e := engine(t)
			i := e.instruments["x"]
			i.PriceStep = density.Price{Nano: int32(tc.step)}
			e.instruments["x"] = i
			o := &Order{InstrumentID: "x", Stop: tc.stop, Lot: 1}
			e.fillEntry(o, 1, tc.entry, start)
			if o.Take != tc.want {
				t.Fatalf("take = %d, want %d", o.Take, tc.want)
			}
		})
	}
}

func TestPlannedTakeProfitHasMinimumReturn(t *testing.T) {
	e := engine(t)
	i := e.instruments["x"]
	i.PriceStep = density.Price{Nano: 10_000_000}
	e.instruments["x"] = i
	place(t, e)
	if o := e.orders["x"]; o.Entry != 100_010_000_000 || o.Take != 100_320_000_000 {
		t.Fatalf("wrong planned target: %+v", o)
	}
}

func TestBookExecutionFillsPassiveEntryAndTakeWithoutTape(t *testing.T) {
	e := engine(t)
	e.cfg.BookExecution = true
	e.cfg.Latency = 0
	e.cfg.MaxEntryLots = 1
	place(t, e)
	o := e.orders["x"]
	if o.State != "pending" {
		t.Fatal("entry filled above its limit")
	}
	b := book(start.Add(time.Millisecond), 100)
	b.Asks[0].Price = p(101)
	b.Bids = b.Bids[1:]
	e.OnBook(b, b.Time)
	if o.State != "open" || o.Entry != 101*nano {
		t.Fatalf("no book entry: %+v", o)
	}
	b.Time = b.Time.Add(time.Millisecond)
	b.Bids = []density.Level{{Price: p(107), Quantity: 1}, {Price: p(100), Quantity: 100}}
	b.Asks[0].Price = p(108)
	e.OnBook(b, b.Time)
	if o.OpenLots != 0 || o.ExitReason != "take_profit" {
		t.Fatalf("no book take: %+v", o)
	}
}

func TestFixedLimitEntryFillsAtFortyPercent(t *testing.T) {
	cfg := DefaultConfig()
	cfg.FixedLimitEntry = true
	cfg.Latency = 0
	i := density.Instrument{ID: "x", Ticker: "T", Lot: 1, PriceStep: p(1)}
	e, err := New(cfg, []density.Instrument{i}, nil)
	if err != nil {
		t.Fatal(err)
	}
	signal := density.Signal{InstrumentID: i.ID, Price: p(100), Quantity: 100}
	b := density.Book{InstrumentID: i.ID, Time: start, Lot: 1,
		Bids: []density.Level{{Price: signal.Price, Quantity: 100}},
		Asks: []density.Level{{Price: p(102), Quantity: 10}},
	}
	e.OnBook(b, start)
	e.Observe(signal, start)
	o := e.orders[i.ID]
	if o == nil || o.State != "pending" || o.Entry != 101*nano || o.Stop != 100*nano {
		t.Fatalf("expected a resting buy at 101: %+v", o)
	}
	b.Time = start.Add(time.Millisecond)
	b.Bids[0].Quantity = 40
	b.Asks[0].Price = p(101)
	e.OnBook(b, b.Time)
	if o.State != "open" || o.Entry != 101*nano || o.Stop != 100*nano {
		t.Fatalf("expected a fill at the limit with exactly 40%% of the density: %+v", o)
	}
}

func TestScreenerStrategyOrderLifecycle(t *testing.T) {
	for _, exit := range []string{"density_lost", "breach", "no_touch", "stop", "take_profit"} {
		t.Run(exit, func(t *testing.T) {
			cfg := DefaultConfig()
			cfg.ScreenerStrategy = true
			cfg.BookExecution = true
			cfg.Latency = 0
			cfg.MaxHold = 0
			i := density.Instrument{ID: "x", Ticker: "X", Lot: 1, PriceStep: density.Price{Nano: 10_000_000}}
			e, err := New(cfg, []density.Instrument{i}, nil)
			if err != nil {
				t.Fatal(err)
			}

			wall := density.Level{Price: p(100), Quantity: 60}
			b := density.Book{InstrumentID: i.ID, Lot: 1, Time: start,
				Bids: []density.Level{wall, {Price: p(99), Quantity: 1}},
				Asks: []density.Level{{Price: p(101), Quantity: 1}},
			}
			e.OnBook(b, b.Time)
			e.Observe(density.Signal{InstrumentID: i.ID, Price: wall.Price, Quantity: wall.Quantity, MinLots: 100, HasAverageVolume: true}, b.Time)
			if e.orders[i.ID] != nil {
				t.Fatal("order placed before a trade touched the wall")
			}
			if exit != "no_touch" {
				e.OnTrade(density.Event{InstrumentID: i.ID, Price: wall.Price, Quantity: 1, Time: start.Add(time.Microsecond)}, start.Add(time.Microsecond))
			}

			b.Time = start.Add(time.Millisecond)
			b.Asks[0].Price = density.Price{Units: 100, Nano: 10_000_000}
			if exit == "density_lost" {
				b.Bids = b.Bids[1:]
			}
			e.OnBook(b, b.Time)
			if e.orders[i.ID] != nil {
				t.Fatal("book movement alone triggered entry")
			}
			rebound := b.Time.Add(time.Microsecond)
			e.OnTrade(density.Event{InstrumentID: i.ID, Price: b.Asks[0].Price, Quantity: 1, Time: rebound}, rebound)
			if exit == "breach" {
				// Same exchange timestamp as the bounce; this trade must still veto the entry.
				e.OnTrade(density.Event{InstrumentID: i.ID, Price: p(99), Quantity: 1, Time: rebound}, rebound)
				e.Observe(density.Signal{InstrumentID: i.ID, Price: wall.Price, Quantity: wall.Quantity}, rebound)
			}
			b.Time = rebound.Add(time.Millisecond)
			e.OnBook(b, b.Time)
			o := e.orders[i.ID]
			if exit == "density_lost" || exit == "breach" || exit == "no_touch" {
				if o != nil {
					t.Fatalf("unexpected entry: %+v", o)
				}
				return
			}
			if o == nil {
				t.Fatal("missing entry after rebound")
			}
			if o.State != "open" || o.Stop != 99_900_000_000 || o.Take != 100_520_000_000 {
				t.Fatalf("expected fill with stop 99.90 and take 100.52: %+v", o)
			}

			b.Time = b.Time.Add(time.Millisecond)
			b.Bids = []density.Level{{Price: p(100), Quantity: 1}}
			e.OnBook(b, b.Time)
			b.Time = b.Time.Add(time.Millisecond)
			b.Bids[0].Price = density.Price{Units: 100, Nano: 510_000_000}
			b.Asks[0].Price = p(101)
			e.OnBook(b, b.Time)
			if o.State != "open" {
				t.Fatalf("position closed before either exit level: %+v", o)
			}
			b.Time = b.Time.Add(time.Millisecond)
			if exit == "stop" {
				b.Bids[0].Price = density.Price{Units: 99, Nano: 900_000_000}
				e.OnBook(b, b.Time)
				b.Time = b.Time.Add(time.Millisecond)
			} else {
				b.Bids[0].Price = density.Price{Units: 100, Nano: 520_000_000}
			}
			e.OnBook(b, b.Time)
			if o.State != "closed" || o.ExitReason != exit {
				t.Fatalf("expected %s to close position: %+v", exit, o)
			}
		})
	}
}
