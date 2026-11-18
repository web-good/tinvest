package paper

import (
	"testing"
	"time"
	"tinvest/internal/service/screener/density"
)

func TestPinbarSimulationIgnoresSellWall(t *testing.T) {
	cfg := DefaultConfig()
	cfg.PinbarEntries = true
	cfg.EntryZoneTicks = 1
	cfg.BookExecution = true
	cfg.Latency = 0
	cfg.MaxHold = 0
	cfg.MaxEntryLots = 1
	e, err := New(cfg, []density.Instrument{{ID: "x", Ticker: "X", Lot: 1, PriceStep: p(1)}}, nil)
	if err != nil {
		t.Fatal(err)
	}
	minute := start.Truncate(time.Minute)
	closeAt := minute.Add(time.Minute)
	s := density.Signal{InstrumentID: "x", Price: p(100), Quantity: 100, Since: minute}
	b := density.Book{InstrumentID: "x", Lot: 1, Time: closeAt.Add(-time.Second), Bids: []density.Level{{Price: p(100), Quantity: 100}}, Asks: []density.Level{{Price: p(110), Quantity: 10}}}
	e.OnBookWithCandidates(b, []density.Signal{s}, b.Time)
	e.pinbarBook(b, []density.Signal{s}, b.Time)
	e.Observe(s, b.Time)
	if e.orders["x"] != nil {
		t.Fatal("buy placed before pinbar close")
	}

	// The next minute's book shows a relocation before the candle arrives.
	s.Price = p(98)
	b.Bids[0].Price = s.Price
	b.Time = closeAt
	e.OnBookWithCandidates(b, []density.Signal{s}, b.Time)
	e.pinbarBook(b, nil, b.Time)
	e.candle("x", density.Candle{Time: minute, Low: p(100), Open: p(104), Close: p(105), High: p(105)}, closeAt)
	o := e.orders["x"]
	if o == nil || o.State != "open" || o.Entry != 110*nano || o.Density != 98*nano {
		t.Fatalf("expected fill at current ask after two-tick relocation, got %+v", o)
	}

	b.Time = b.Time.Add(time.Second)
	b.Bids = append([]density.Level{{Price: p(115), Quantity: 10}}, b.Bids...)
	b.Asks = []density.Level{{Price: p(117), Quantity: 100}}
	e.OnBookWithCandidates(b, []density.Signal{s}, b.Time)
	if o.State != "open" || o.SellLimit != 0 {
		t.Fatalf("sell wall affected open position: %+v", o)
	}

	b.Time = b.Time.Add(time.Second)
	b.Bids[0].Price = p(138)
	b.Asks[0].Price = p(139)
	e.OnBookWithCandidates(b, []density.Signal{s}, b.Time)
	if o.State != "closed" || o.ExitReason != "take_profit" {
		t.Fatalf("expected configured take profit to close position, got %+v", o)
	}
}
