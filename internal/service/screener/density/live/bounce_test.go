package live

import (
	"context"
	"testing"
	"time"

	"tinvest/internal/service/screener/density"
)

func TestBounceBuyRevalidatesWallBeforeSubmission(t *testing.T) {
	for _, tc := range []struct {
		name     string
		bid, qty int64
		wantBuy  bool
	}{
		{"retained at 30% below M5 average", 100, 300, true},
		{"below 30%", 100, 299, false},
		{"relocated", 101, 1000, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			x := setup(t)
			x.e.cfg.CandleEntries = true
			x.f.cfg.CandleEntries = true
			x.f.cfg.Freshness = 65 * time.Second
			x.e.cfg.Freshness = x.f.cfg.Freshness
			minute := x.now.Truncate(time.Minute).Add(time.Minute)
			s := density.Signal{InstrumentID: x.i.ID, Price: density.Price{Units: 100}, Quantity: 1000, MinLots: 750, Since: minute}
			book := func(at time.Time, bid, qty int64) {
				x.now = at
				b := density.Book{InstrumentID: x.i.ID, Lot: x.i.Lot, Time: at,
					Bids: []density.Level{{Price: density.Price{Units: bid}, Quantity: qty}},
					Asks: []density.Level{{Price: density.Price{Units: 110}, Quantity: 10}}}
				var signals []density.Signal
				if bid == 100 && qty >= 750 {
					signals = []density.Signal{s}
				}
				x.f.bookWithCandidates(b, signals, signals, x.i.PriceStep, at)
			}
			checkNoBuy := func(now time.Time) {
				t.Helper()
				x.now = now
				if err := x.e.Step(context.Background(), now); err != nil || len(x.b.posts) != 0 {
					t.Fatalf("unexpected buy %v, err %v", x.b.posts, err)
				}
			}

			book(minute, 100, 1000)
			checkNoBuy(minute)
			second := minute.Add(time.Minute)
			book(second, 100, 1000)
			x.f.candle(x.i.ID, density.Candle{Time: minute, Low: density.Price{Units: 95}, Close: density.Price{Units: 100}}, second)
			checkNoBuy(second)
			third := second.Add(time.Minute)
			x.f.candle(x.i.ID, density.Candle{Time: second, Low: density.Price{Units: 101}, Close: density.Price{Units: 103}}, third)
			checkNoBuy(third)
			book(third.Add(x.e.cfg.PollInterval), 100, 1000)
			x.b.onCanBuy = func() { book(x.now.Add(time.Millisecond), tc.bid, tc.qty) }
			if err := x.e.Step(context.Background(), x.now); err != nil {
				t.Fatal(err)
			}
			if got := len(x.b.posts); (got == 1) != tc.wantBuy {
				t.Fatalf("posted %d, want buy %v", got, tc.wantBuy)
			}
			if tc.wantBuy && (x.b.posts[0].Price != 110*Nano || !x.b.posts[0].Immediate) {
				t.Fatalf("expected immediate buy at current ask: %v", x.b.posts)
			}
		})
	}
}
