package strategy

import (
	"math"
	"testing"
	"time"

	"tinvest/internal/service/screener/bybitdensity/model"
)

func TestZoneReplayConfirmedBounce(t *testing.T) {
	for _, tc := range []struct {
		name         string
		sell, inside bool
		retained     float64
		invalid      int64
		interruption string
		want         bool
	}{
		{"bounce", true, true, 40, 0, "", true},
		{"no sale", false, true, 40, 0, "", false},
		{"no bid in zone", true, false, 40, 0, "", false},
		{"lost volume", true, true, 39, 0, "", false},
		{"stop touched", true, true, 40, 9990, "", false},
		{"previous high touched", true, true, 40, 9995, "", false},
		{"brief bounce", true, true, 40, 0, "bid", false},
		{"volume disappears while waiting", true, true, 40, 0, "volume", false},
		{"feed reset while waiting", true, true, 40, 0, "reset", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			inst := model.Instrument{TickSize: .01, QtyStep: .1}
			r := NewZoneReplay(inst, NewZonePolicy(inst, .01, 50), 250, .001, time.Second)
			r.SetPreviousHigh(9995)
			if tc.name == "stop touched" {
				r.SetPreviousHigh(9900)
			}
			at := time.Unix(1, 0)
			book := func(bid int64, qty float64) {
				at = at.Add(time.Millisecond)
				bids := []model.Level{{Ticks: 10000, Qty: qty}, {Ticks: 9999, Qty: 1}}
				if bid > 10000 {
					bids = append([]model.Level{{Ticks: bid, Qty: 1}}, bids...)
				}
				r.Apply(model.Event{Book: &model.Book{At: at, Bids: bids, Asks: []model.Level{{Ticks: bid + 2, Qty: .5}}}})
			}
			trade := func(ticks int64, sell bool) {
				at = at.Add(time.Millisecond)
				r.Apply(model.Event{Trades: []model.Trade{{Ticks: ticks, Sell: sell, At: at, Qty: 10}}})
			}
			book(10002, 60)
			book(10002, 100)
			if tc.inside {
				book(10000, 100)
			}
			trade(10000, tc.sell)
			if tc.invalid > 0 {
				trade(tc.invalid, true)
			}
			book(10001, tc.retained)
			if got := r.Result().OpenQuantity; got != 0 {
				t.Fatalf("entry before confirmation: open quantity %g", got)
			}
			at = at.Add(98 * time.Millisecond)
			switch tc.interruption {
			case "bid":
				book(10000, 40)
			case "volume":
				book(10001, 39)
			case "reset":
				r.Apply(model.Event{Reset: true})
			default:
				book(10001, tc.retained)
			}
			at = at.Add(899 * time.Millisecond)
			book(10001, tc.retained)
			if got := r.Result().OpenQuantity; got != 0 {
				t.Fatalf("entry after 999 ms: open quantity %g", got)
			}
			book(10002, tc.retained)
			if got := r.Result().OpenQuantity; (got > 0) != tc.want {
				t.Fatalf("open quantity %g", got)
			}
			if tc.want {
				trade(10100, false)
				trades := r.Result().Trades
				if len(trades) != 1 || trades[0].EntryPrice != inst.Price(10004) || trades[0].Quantity != .5 || trades[0].TakePrice != inst.Price(10085) || trades[0].StopPrice != inst.Price(9990) {
					t.Fatalf("trades %+v", trades)
				}
			}
		})
	}
}

func TestZoneReplayAggregatesZoneAndStopsAgainstAvailableBids(t *testing.T) {
	inst := model.Instrument{TickSize: .01, QtyStep: .1}
	r := NewZoneReplay(inst, NewZonePolicy(inst, .05, 100), 250, .001, time.Second)
	at := time.Unix(1, 0)
	book := func(bids []model.Level, ask int64) {
		at = at.Add(time.Millisecond)
		r.Apply(model.Event{Book: &model.Book{At: at, Bids: bids, Asks: []model.Level{{Ticks: ask, Qty: 5}}}})
	}
	zone := []model.Level{{Ticks: 10003, Qty: 50}, {Ticks: 10000, Qty: 50}}
	book(append([]model.Level{{Ticks: 10005, Qty: 1}}, zone...), 10006)
	book(zone, 10005)
	r.Apply(model.Event{Trades: []model.Trade{{At: at, Sell: true, Ticks: 10003, Qty: 1}}})
	book(append([]model.Level{{Ticks: 10005, Qty: 1}}, zone...), 10007)
	at = at.Add(time.Second - time.Millisecond)
	book(append([]model.Level{{Ticks: 10005, Qty: 1}}, zone...), 10007)
	if got := r.Result().OpenQuantity; math.Abs(got-2.4) > 1e-9 {
		t.Fatalf("open quantity %g", got)
	}
	book([]model.Level{{Ticks: 9990, Qty: 1}}, 9991)
	if got := r.Result(); got.OpenQuantity < 1.39 || got.OpenQuantity > 1.41 || got.Realized >= 0 {
		t.Fatalf("partial stop %+v", got)
	}
	book([]model.Level{{Ticks: 9988, Qty: 5}}, 9989)
	got := r.Result()
	if len(got.Trades) != 1 || got.Trades[0].ExitReason != "stop" || got.OpenQuantity != 0 {
		t.Fatalf("final stop %+v", got)
	}
}

func TestNeighborZoneSelection(t *testing.T) {
	for _, tc := range []struct {
		name          string
		qty, neighbor float64
		lastTick      int64
		want          bool
	}{
		{"minimum equality", 10, 4, 9980, true},
		{"neighbor equality", 12, 6, 9980, true},
		{"below minimum", 9, 4, 9980, false},
		{"below twice larger neighbor", 11, 6, 9980, false},
		{"incomplete lower neighbor", 12, 6, 9981, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			inst := model.Instrument{Key: model.Key{Market: model.Spot}, TickSize: .01}
			at := time.Unix(100, 0)
			book := &model.Book{At: at, Received: at, Bids: []model.Level{
				{Ticks: 10020, Qty: 400.0 / 100.2}, {Ticks: 10000, Qty: tc.qty},
				{Ticks: tc.lastTick, Qty: tc.neighbor * 100 / 99.8},
			}, Asks: []model.Level{{Ticks: 10021, Qty: 1}}}
			filter := ZoneFilter{MinNotional: 1000, NeighborMultiple: 2}
			policy := NewNeighborZonePolicy(inst, .1, filter)
			_, replay := policy.Entry(book)
			cfg := Config{ZoneWidthUSDT: .1, ZoneAlerts: true, SpotZoneFilter: filter}
			s := New(inst, cfg)
			s.findCandidates(book, at)
			s.observeZones(book, at)
			_, alert := s.alertEpisodes[priceLevel{model.Bid, 10000}]
			if replay != tc.want || (len(s.candidates) == 1) != tc.want || alert != tc.want {
				t.Fatalf("replay=%t simulator candidates=%d alert=%t, want %t", replay, len(s.candidates), alert, tc.want)
			}
		})
	}
}

func TestZoneReplayChoosesNearestSimultaneousBounce(t *testing.T) {
	inst := model.Instrument{TickSize: .01, QtyStep: .1}
	r := NewZoneReplay(inst, NewZonePolicy(inst, .01, 50), 250, .001, time.Second)
	start := time.Unix(1, 0)
	book := func(ms int, best int64, zones ...model.Level) {
		bids := zones
		if len(zones) == 0 || zones[0].Ticks < best {
			bids = append([]model.Level{{Ticks: best, Qty: 1}}, zones...)
		}
		r.Apply(model.Event{Book: &model.Book{
			At:   start.Add(time.Duration(ms) * time.Millisecond),
			Bids: bids,
			Asks: []model.Level{{Ticks: best + 1, Qty: 5}},
		}})
	}
	sell := func(ms int, ticks int64) {
		r.Apply(model.Event{Trades: []model.Trade{{At: start.Add(time.Duration(ms) * time.Millisecond), Sell: true, Ticks: ticks, Qty: 1}}})
	}
	far := model.Level{Ticks: 10000, Qty: 60}
	near := model.Level{Ticks: 10003, Qty: 60}
	book(0, 10005, far)
	sell(1, far.Ticks)
	book(2, far.Ticks, far)
	book(3, 10005, near, far)
	sell(4, near.Ticks)
	book(5, near.Ticks, near, far)
	book(6, 10005, near, far)
	book(1100, 10005, near, far)
	if r.position == nil || r.position.trade.ZoneLow != inst.Price(near.Ticks) {
		t.Fatalf("entry position = %+v, want nearer zone", r.position)
	}
	if got := r.Result(); got.Placed != 2 || got.Cancelled != 1 || got.Pending {
		t.Fatalf("diagnostics = %+v", got)
	}
}

func TestZoneReplayDropsZoneWhenBidMovesOutOfRange(t *testing.T) {
	inst := model.Instrument{TickSize: .01, QtyStep: .1}
	r := NewZoneReplay(inst, NewZonePolicy(inst, .01, 50), 250, .001, time.Second)
	at := time.Unix(1, 0)
	book := func(best int64) {
		at = at.Add(time.Millisecond)
		r.Apply(model.Event{Book: &model.Book{
			At: at, Bids: []model.Level{{Ticks: best, Qty: 1}, {Ticks: 10000, Qty: 60}},
			Asks: []model.Level{{Ticks: best + 1, Qty: 5}},
		}})
	}
	book(10020)
	if result := r.Result(); !result.Pending || result.Placed != 1 {
		t.Fatalf("near zone was not observed: %+v", result)
	}
	book(10040)
	if result := r.Result(); result.Pending || result.Placed != 1 || result.Cancellations[ZoneCancelDistance] != 1 {
		t.Fatalf("distant zone was not dropped: %+v", result)
	}
}
