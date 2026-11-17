package paper

import (
	"testing"
	"time"
	"tinvest/internal/service/screener/density"
)

func TestMoscowEntryHours(t *testing.T) {
	for _, tc := range []struct {
		utc     string
		blocked bool
	}{
		{"03:59:59", false}, {"04:00:00", true},
		{"08:59:59", true}, {"09:00:00", false},
		{"11:59:59", false}, {"12:00:00", true},
		{"14:59:59", true}, {"15:00:00", false},
	} {
		t.Run(tc.utc, func(t *testing.T) {
			at, err := time.Parse("2006-01-02 15:04:05", "2026-11-17 "+tc.utc)
			if err != nil {
				t.Fatal(err)
			}
			e := engine(t)
			e.cfg.RestrictEntryHours = true
			e.OnBook(book(at, 100), at)
			e.Observe(density.Signal{InstrumentID: "x", Price: p(100), Quantity: 100}, at)
			if (e.orders["x"] == nil) != tc.blocked {
				t.Fatalf("entry blocked = %v, want %v", e.orders["x"] == nil, tc.blocked)
			}
		})
	}
}

func TestMoscowEntryHoursCancelPendingBuyAndKeepOpenPosition(t *testing.T) {
	boundary := time.Date(2026, 11, 17, 12, 0, 0, 0, time.UTC) // 15:00 МСК
	before := boundary.Add(-2 * time.Second)
	for _, event := range []string{"book", "trade", "tick", "take"} {
		t.Run(event, func(t *testing.T) {
			e := engine(t)
			e.cfg.RestrictEntryHours = true
			e.OnBook(book(before, 100), before)
			e.Observe(density.Signal{InstrumentID: "x", Price: p(100), Quantity: 100}, before)
			o := e.orders["x"]
			if o == nil {
				t.Fatal("entry should be permitted before the window")
			}
			switch event {
			case "book":
				e.cfg.BookExecution = true
				b := book(boundary, 100)
				b.Bids = b.Bids[1:]
				b.Asks[0].Price = p(101)
				e.OnBook(b, boundary)
			case "trade":
				trade(e, boundary, 100, 1)
			case "tick":
				e.Tick(boundary)
			case "take":
				trade(e, before.Add(time.Second), 100, 1)
				e.Tick(boundary)
				if o.State != "open" {
					t.Fatalf("window affected open position: %+v", o)
				}
				trade(e, boundary.Add(time.Second), 108, 1)
				if o.State != "closed" || o.ExitReason != "take_profit" {
					t.Fatalf("take did not close position: %+v", o)
				}
				return
			}
			if o.State != "canceled" || o.FilledLots != 0 || o.ExitReason != "entry_hours" {
				t.Fatalf("pending entry not canceled: %+v", o)
			}
		})
	}
}
