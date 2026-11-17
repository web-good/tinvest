package density

import (
	"testing"
	"time"
)

func TestRelocationChoosesNearestQualifiedLevelFromCurrentBook(t *testing.T) {
	now := time.Now()
	b := Book{InstrumentID: "x", Bids: []Level{{Price: Price{Units: 102}, Quantity: 2000}, {Price: Price{Units: 101}, Quantity: 500}, {Price: Price{Units: 99}, Quantity: 500}, {Price: Price{Units: 98}, Quantity: 2000}}}
	candidates := []Signal{{InstrumentID: "x", Price: Price{Units: 98}}, {InstrumentID: "x", Price: Price{Units: 99}}, {InstrumentID: "x", Price: Price{Units: 101}}, {InstrumentID: "x", Price: Price{Units: 102}, ProfileUntil: now}}
	s, ok := Relocation(b, Price{Units: 100}, 1000, .5, candidates, now)
	if !ok || s.Price.Units != 101 || s.Quantity != 500 {
		t.Fatalf("wrong nearest/tie result: %+v %v", s, ok)
	}
	for _, bad := range []Signal{{InstrumentID: "other", Price: Price{Units: 101}}, {InstrumentID: "x", Price: Price{Units: 100}}, {InstrumentID: "x", Price: Price{Units: 103}}, {InstrumentID: "x", Price: Price{Units: 102}, ProfileUntil: now}} {
		if _, ok := Relocation(b, Price{Units: 100}, 1000, .5, []Signal{bad}, now); ok {
			t.Fatalf("accepted invalid relocation: %+v", bad)
		}
	}
	b.Bids[1].Quantity = 499
	if _, ok := Relocation(b, Price{Units: 100}, 1000, .5, candidates[2:3], now); ok {
		t.Fatal("accepted replacement below volume threshold")
	}
}
