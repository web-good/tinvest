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

func TestConfigRejectsInvalidEntryPct(t *testing.T) {
	for _, pct := range []float64{-1, math.NaN(), math.Inf(1), 101} {
		cfg := DefaultConfig()
		cfg.EntryPct = pct
		if cfg.Validate() == nil {
			t.Fatalf("accepted entry percentage %v", pct)
		}
	}
}

func TestPercentageEntryWaitsForLatencyAndUsesAvailableAsk(t *testing.T) {
	cfg := DefaultConfig()
	cfg.EntryPct = .3
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
