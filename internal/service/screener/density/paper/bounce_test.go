package paper

import (
	"testing"
	"time"
	"tinvest/internal/service/screener/density"
)

func TestBounceSimulationIgnoresSellWall(t *testing.T) {
	cfg := DefaultConfig()
	cfg.CandleEntries = true
	cfg.IgnoreCapital = true
	cfg.Freshness = 65 * time.Second
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
	s := density.Signal{InstrumentID: "x", Price: p(100), Quantity: 100, MinLots: 75, Since: minute}
	b := density.Book{InstrumentID: "x", Lot: 1, Time: minute, Bids: []density.Level{{Price: p(100), Quantity: 100}}, Asks: []density.Level{{Price: p(110), Quantity: 10}}}
	e.OnBookWithCandidates(b, []density.Signal{s}, b.Time)
	e.bounceBook(b, []density.Signal{s}, b.Time)
	e.Observe(s, b.Time)
	if e.orders["x"] != nil {
		t.Fatal("buy placed before touch candle close")
	}

	b.Time = closeAt
	e.OnBookWithCandidates(b, []density.Signal{s}, b.Time)
	e.bounceBook(b, []density.Signal{s}, b.Time)
	e.candle("x", density.Candle{Time: minute, Low: p(95), Close: p(100)}, closeAt)
	if e.orders["x"] != nil {
		t.Fatal("buy placed before confirmation")
	}

	b.Time = closeAt.Add(time.Minute)
	e.candle("x", density.Candle{Time: closeAt, Low: p(100), Close: p(103)}, b.Time)
	if e.orders["x"] != nil {
		t.Fatal("buy used a quote from before confirmation")
	}
	b.Bids[0].Quantity = 30
	e.OnBookWithCandidates(b, nil, b.Time)
	e.bounceBook(b, nil, b.Time)
	o := e.orders["x"]
	if o == nil || o.State != "open" || o.Entry != 110*nano || o.Density != 100*nano {
		t.Fatalf("expected fill at current ask, got %+v", o)
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
