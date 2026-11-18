package density

import (
	"testing"
	"time"
)

func TestPinbarEntryRelocation(t *testing.T) {
	start := time.Date(2026, 11, 18, 10, 0, 0, 0, time.UTC)
	closeAt := start.Add(time.Minute)
	anchor := Signal{InstrumentID: "x", Price: Price{Units: 100}, Quantity: 100, Since: start}
	step := Price{Nano: 10_000_000}
	candle := Candle{Time: start, Low: anchor.Price, Open: Price{Units: 100, Nano: 80_000_000}, Close: Price{Units: 100, Nano: 90_000_000}, High: Price{Units: 100, Nano: 100_000_000}}
	for _, tt := range []struct {
		name  string
		level Price
		ok    bool
	}{
		{"two ticks up", Price{Units: 100, Nano: 20_000_000}, true},
		{"two ticks down", Price{Units: 99, Nano: 980_000_000}, true},
		{"three ticks up", Price{Units: 100, Nano: 30_000_000}, false},
		{"three ticks down", Price{Units: 99, Nano: 970_000_000}, false},
	} {
		t.Run(tt.name, func(t *testing.T) {
			var entry pinbarEntry
			entry.Confirm(candle, []Signal{anchor}, step, 1, closeAt)
			moved := Signal{InstrumentID: "x", Price: tt.level, Quantity: 100}
			book := Book{InstrumentID: "x", Bids: []Level{{Price: tt.level, Quantity: 100}}}
			entry.Update(book, []Signal{moved}, step, .5, closeAt)
			signal, until, ok := entry.Ready(closeAt)
			if ok != tt.ok || ok && (signal.Price != tt.level || !until.Equal(closeAt.Add(time.Minute))) {
				t.Fatalf("ready = %v until %v, ok %v", signal.Price, until, ok)
			}
			if _, _, ok := entry.Ready(closeAt.Add(time.Minute)); ok {
				t.Fatal("entry survives the next minute")
			}
		})
	}
}

func TestPinbarGeometry(t *testing.T) {
	q := func(ticks int32) Price { return Price{Units: 100, Nano: ticks * 10_000_000} }
	for _, tt := range []struct {
		name   string
		candle Candle
		want   bool
	}{
		{"upper boundary and exact body size", Candle{Low: q(1), High: q(13), Open: q(9), Close: q(12)}, true},
		{"bearish body", Candle{Low: q(0), High: q(12), Open: q(11), Close: q(8)}, true},
		{"body too large", Candle{Low: q(0), High: q(12), Open: q(8), Close: q(12)}, false},
		{"body too low", Candle{Low: q(0), High: q(12), Open: q(7), Close: q(8)}, false},
		{"above zone", Candle{Low: q(2), High: q(14), Open: q(10), Close: q(13)}, false},
		{"penetrated density", Candle{Low: Price{Units: 99, Nano: 990_000_000}, High: q(11), Open: q(7), Close: q(10)}, false},
		{"flat", Candle{Low: q(0), High: q(0), Open: q(0), Close: q(0)}, false},
	} {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.candle.matches(pinbarPrice(q(0)), 10_000_000, 1); got != tt.want {
				t.Fatalf("matches = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestPinbarWaitsForNextMinuteBook(t *testing.T) {
	minute := time.Date(2026, 11, 18, 10, 0, 0, 0, time.UTC)
	closeAt := minute.Add(time.Minute)
	s := Signal{InstrumentID: "x", Price: Price{Units: 100}, Quantity: 100, Since: minute}
	b := Book{InstrumentID: "x", Time: closeAt.Add(-time.Second), Bids: []Level{{Price: s.Price, Quantity: 100}}}
	c := Candle{Time: minute, Low: s.Price, Open: Price{Units: 104}, Close: Price{Units: 105}, High: Price{Units: 105}}
	var watch PinbarWatch
	watch.Book(b, []Signal{s}, []Signal{s}, Price{Units: 1}, .5, b.Time)
	watch.Candle(c, b, []Signal{s}, 0, .5, closeAt)
	if _, _, ok := watch.Ready(closeAt); ok {
		t.Fatal("signal could use a quote from before the candle closed")
	}

	b.Time = closeAt.Add(time.Millisecond)
	watch.Book(b, []Signal{s}, []Signal{s}, Price{Units: 1}, .5, b.Time)
	if _, _, ok := watch.Ready(b.Time); !ok {
		t.Fatal("signal disappeared before the first book of the next minute")
	}
	watch.Consume()
	watch.Candle(c, b, []Signal{s}, 0, .5, b.Time)
	if _, _, ok := watch.Ready(b.Time); ok {
		t.Fatal("replayed candle armed another buy attempt")
	}
}
