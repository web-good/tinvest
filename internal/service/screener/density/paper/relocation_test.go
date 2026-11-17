package paper

import (
	"fmt"
	"testing"
	"time"
	"tinvest/internal/service/screener/density"
)

func relocationBook(at time.Time, level, qty int64) (density.Book, []density.Signal) {
	b := density.Book{InstrumentID: "x", Lot: 1, Time: at,
		Bids: []density.Level{{Price: p(level), Quantity: qty}},
		Asks: []density.Level{{Price: p(110), Quantity: 10}}}
	return b, []density.Signal{{InstrumentID: "x", Price: p(level), Quantity: qty, Since: at}}
}

func TestPendingEntryFollowsRelocatedDensity(t *testing.T) {
	for _, level := range []int64{101, 99, 96} {
		t.Run(fmt.Sprint(level), func(t *testing.T) {
			e := engine(t)
			place(t, e)
			b, candidates := relocationBook(start.Add(time.Millisecond), level, 100)
			e.OnBookWithCandidates(b, candidates, b.Time)
			o := e.orders["x"]
			if o.State != "pending" || o.Density != level*nano || o.Entry != (level+1)*nano || o.Stop != (level-2)*nano || !o.Created.Equal(start) {
				t.Fatalf("bad replacement %+v", o)
			}
			if !o.ActiveAt.Equal(b.Time.Add(e.cfg.Latency)) {
				t.Fatal("replacement bypassed activation latency")
			}
		})
	}
}

func TestOpenPositionKeepsProtectionWhenDensityMoves(t *testing.T) {
	e := engine(t)
	place(t, e)
	trade(e, start.Add(time.Second), 100, 1)
	o := e.orders["x"]
	entry, stop, take := o.Entry, o.Stop, o.Take
	b, candidates := relocationBook(start.Add(1100*time.Millisecond), 101, 75)
	e.OnBookWithCandidates(b, candidates, b.Time)
	if o.State != "open" || o.Entry != entry || o.Stop != stop || o.Take != take || o.Density != 100*nano {
		t.Fatalf("move changed open trade: %+v", o)
	}
	b, candidates = relocationBook(b.Time.Add(time.Millisecond), 102, 50)
	e.OnBookWithCandidates(b, candidates, b.Time)
	b.Time = b.Time.Add(time.Millisecond)
	e.OnBookWithCandidates(b, nil, b.Time)
	if o.State != "open" {
		t.Fatal("forgot relocated density")
	}
	b, _ = relocationBook(b.Time.Add(time.Millisecond), 102, 49)
	e.OnBookWithCandidates(b, nil, b.Time)
	if o.State != "exiting" || o.ExitReason != "density_lost" {
		t.Fatalf("no early exit: %+v", o)
	}
}
