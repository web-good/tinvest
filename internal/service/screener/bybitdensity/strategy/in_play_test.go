package strategy

import (
	"testing"
	"time"

	"tinvest/internal/service/screener/bybitdensity/model"
)

func TestInPlayWindowUsesSixClosedCandlesAndPriorDayTurnover(t *testing.T) {
	end := time.Date(2026, 11, 22, 10, 30, 0, 0, time.UTC)
	filter := InPlayFilter{MinRange: .01, MinTurnoverMultiple: 2}
	for _, tc := range []struct {
		name                       string
		open, high, latestTurnover float64
		gap, want                  bool
	}{
		{name: "at both thresholds", open: 100, high: 101, latestTurnover: 200, want: true},
		{name: "decimal price at range threshold", open: .01, high: .0101, latestTurnover: 200, want: true},
		{name: "insufficient range", open: 100, high: 100.9, latestTurnover: 200},
		{name: "insufficient turnover", open: 100, high: 101, latestTurnover: 199},
		{name: "missing baseline candle", open: 100, high: 101, latestTurnover: 200, gap: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			candles := make([]model.M5Candle, 294)
			for i := range candles {
				candles[i] = model.M5Candle{End: end.Add(time.Duration(i-293) * 5 * time.Minute), Open: 100, High: 100, Low: 100, Turnover: 100}
			}
			for i := 288; i < len(candles); i++ {
				candles[i].Open, candles[i].High, candles[i].Low = tc.open, tc.high, tc.open
				candles[i].Turnover = tc.latestTurnover
			}
			if tc.gap {
				candles[100].End = candles[99].End
			}
			window := filter.Window(candles)
			if got := window.allows(end); got != tc.want {
				t.Fatalf("allows at close = %t, want %t", got, tc.want)
			}
			if window.allows(end.Add(-time.Millisecond)) || window.allows(end.Add(5*time.Minute)) {
				t.Fatal("window allowed events outside its interval")
			}
			if filter.Window(candles[1:]).allows(end) {
				t.Fatal("accepted a shortened baseline")
			}
		})
	}
}

func TestInPlayExpiryStopsLiveAlertsAndPendingEntries(t *testing.T) {
	cfg := DefaultConfig()
	cfg.RequireInPlay = true
	cfg.ZoneAlerts = true
	cfg.ZoneWidthUSDT = 2
	cfg.FixedLimitEntry = true
	cfg.SpotZoneFilter = ZoneFilter{MinNotional: 50000, NeighborMultiple: 2}
	cfg.Persistence = time.Second
	inst := model.Instrument{Key: model.Key{Market: model.Spot, Symbol: "PIEVERSEUSDT"}, TickSize: 1, QtyStep: .001}
	s := New(inst, cfg)
	start := time.Date(2026, 11, 22, 10, 30, 0, 0, time.UTC)
	update := func(at time.Time, snapshot bool, ask int64) ([]model.Signal, []model.Fill) {
		b := &model.Book{At: at, Received: at, Snapshot: snapshot, Asks: []model.Level{{Ticks: ask, Qty: 1}}}
		for tick := int64(10004); tick >= 9990; tick-- {
			qty := 1.0
			if tick == 10000 || tick == 10001 {
				qty = 6
			}
			b.Bids = append(b.Bids, model.Level{Ticks: tick, Qty: qty})
		}
		return s.Apply(model.Event{Book: b, TradeReady: true}, at)
	}
	update(start, true, 10006)
	if len(s.episodes) > 0 {
		t.Fatal("observed a zone without in-play data")
	}
	s.SetInPlayWindow(InPlayWindow{From: start, Active: true}, start)
	var signals []model.Signal
	for sec := 290; sec <= 298; sec++ {
		signals, _ = update(start.Add(time.Duration(sec)*time.Second), false, 10006)
	}
	if len(signals) != 1 || s.pending == nil {
		t.Fatalf("expected one signal and a pending buy, got %v and %v", signals, s.pending)
	}
	at := start.Add(5 * time.Minute)
	if _, ok := s.Current(signals[0], at); ok {
		t.Fatal("queued alert was deliverable after in-play expired")
	}
	s.SetInPlayWindow(InPlayWindow{From: at, Active: true}, at.Add(100*time.Millisecond))
	if s.pending != nil || len(s.episodes) > 0 {
		t.Fatal("entry observation survived publication after in-play expiry")
	}
}

func TestInPlayReplayCancelsPendingAndKeepsOpenPosition(t *testing.T) {
	inst := model.Instrument{TickSize: .01, QtyStep: .1}
	r := NewZoneReplay(inst, NewZonePolicy(inst, .01, 50), 250, .001, 0)
	start := time.Date(2026, 11, 22, 10, 30, 0, 0, time.UTC)
	book := func(at time.Time, bid int64) {
		bids := []model.Level{{Ticks: 10000, Qty: 100}, {Ticks: 9999, Qty: 1}}
		if bid < 10000 {
			bids = []model.Level{{Ticks: bid, Qty: 100}}
		}
		if bid > 10000 {
			bids = append([]model.Level{{Ticks: bid, Qty: 1}}, bids...)
		}
		r.Apply(model.Event{Book: &model.Book{At: at, Bids: bids, Asks: []model.Level{{Ticks: bid + 2, Qty: 1}}}})
	}
	r.SetInPlayWindow(InPlayWindow{From: start, Active: true})
	book(start, 10002)
	r.SetInPlayWindow(InPlayWindow{From: start.Add(5 * time.Minute)})
	book(start.Add(5*time.Minute), 10000)
	if got := r.Result(); got.Placed != 1 || got.Cancellations[ZoneCancelInPlay] != 1 {
		t.Fatalf("unexpected state outside in play: %+v", got)
	}

	active := start.Add(10 * time.Minute)
	r.SetInPlayWindow(InPlayWindow{From: active, Active: true})
	book(active, 10002)
	book(active.Add(time.Millisecond), 10000)
	r.Apply(model.Event{Trades: []model.Trade{{At: active.Add(time.Millisecond), Sell: true, Ticks: 10000, Qty: 1}}})
	book(active.Add(2*time.Millisecond), 10001)
	if r.Result().OpenQuantity == 0 {
		t.Fatal("missing entry in play")
	}
	r.SetInPlayWindow(InPlayWindow{From: active.Add(5 * time.Minute)})
	book(active.Add(5*time.Minute), 9988)
	if got := r.Result(); len(got.Trades) != 1 || got.Trades[0].ExitReason != "stop" {
		t.Fatalf("expected normal stop after in play, got %+v", got)
	}
}
