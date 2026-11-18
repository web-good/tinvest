package live

import (
	"context"
	"testing"
	"time"
	"tinvest/internal/service/screener/density"
)

func TestPinbarBuyRevalidatesRelocationBeforeSubmission(t *testing.T) {
	for _, tt := range []struct {
		name    string
		level   int64
		wantBuy bool
	}{
		{"two ticks up", 102, true},
		{"two ticks down", 98, true},
		{"three ticks away", 97, false},
	} {
		t.Run(tt.name, func(t *testing.T) {
			x := setup(t)
			x.e.cfg.PinbarEntries = true
			x.f.cfg.PinbarEntries = true
			minute := x.now.Truncate(time.Minute).Add(time.Minute)
			closeAt := minute.Add(time.Minute)
			book := func(at time.Time, level int64, confirmed bool) {
				x.now = at
				s := density.Signal{InstrumentID: x.i.ID, Price: density.Price{Units: level}, Quantity: 1000, Since: minute}
				b := density.Book{InstrumentID: x.i.ID, Lot: x.i.Lot, Time: at, Bids: []density.Level{{Price: s.Price, Quantity: s.Quantity}}, Asks: []density.Level{{Price: density.Price{Units: 110}, Quantity: 10}}}
				var signals []density.Signal
				if confirmed {
					signals = []density.Signal{s}
				}
				x.f.bookWithCandidates(b, signals, []density.Signal{s}, x.i.PriceStep, at)
			}
			book(closeAt.Add(-time.Second), 100, true)
			if err := x.e.Step(context.Background(), x.now); err != nil || len(x.b.posts) != 0 {
				t.Fatalf("entered before candle closed: posts %v, err %v", x.b.posts, err)
			}
			book(closeAt, 100, true)
			c := density.Candle{Time: minute, Low: density.Price{Units: 100}, Open: density.Price{Units: 104}, Close: density.Price{Units: 105}, High: density.Price{Units: 105}}
			x.f.candle(x.i.ID, c, x.now)
			x.b.onCanBuy = func() { book(x.now.Add(time.Millisecond), tt.level, false) }
			x.now = x.now.Add(2 * time.Second)
			if err := x.e.Step(context.Background(), x.now); err != nil {
				t.Fatal(err)
			}
			if !tt.wantBuy {
				if len(x.b.posts) != 0 {
					t.Fatalf("placed a buy after losing the signal: %v", x.b.posts)
				}
				return
			}
			if len(x.b.posts) != 1 || x.b.posts[0].Price != 110*Nano || !x.b.posts[0].Immediate {
				t.Fatalf("expected a single immediate buy at the current ask: %v", x.b.posts)
			}
			if ep := x.e.state.Episodes[x.i.ID]; ep == nil || ep.Density != tt.level*Nano {
				t.Fatalf("wrong density after relocation: %+v", ep)
			}
		})
	}
}
