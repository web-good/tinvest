package strategy_test

import (
	"math"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"tinvest/internal/service/screener/bybitdensity/model"
	"tinvest/internal/service/screener/bybitdensity/strategy"
	"tinvest/internal/service/screener/volume"
)

var minute = time.Date(2026, 11, 19, 12, 0, 0, 0, time.UTC)

func at(seconds int) time.Time { return minute.Add(time.Duration(seconds) * time.Second) }

func engine(market model.Market) *strategy.Strategy {
	cfg := strategy.DefaultConfig()
	cfg.MinNotional = 500
	cfg.Persistence = time.Second
	return strategy.New(model.Instrument{Key: model.Key{Market: market, Symbol: "TESTUSDT"}, TickSize: .1, QtyStep: .1}, cfg)
}

func book(second int, side model.Side) *model.Book {
	b := &model.Book{At: at(second), Received: at(second)}
	for i := int64(0); i < 8; i++ {
		b.Bids = append(b.Bids, model.Level{Ticks: 1000 - i, Qty: 1})
		b.Asks = append(b.Asks, model.Level{Ticks: 1001 + i, Qty: 1})
	}
	if side == model.Bid {
		b.Bids[0].Qty = 10
		b.Asks[0].Qty = 1.5
	} else {
		b.Asks[0].Qty = 10
		b.Bids[0].Qty = 1.5
	}
	return b
}

func wideBook(second int, side model.Side) *model.Book {
	b := book(second, side)
	if side == model.Bid {
		for i := range b.Asks {
			b.Asks[i].Ticks++
		}
	} else {
		for i := range b.Bids {
			b.Bids[i].Ticks--
		}
	}
	return b
}

func start(t *testing.T, s *strategy.Strategy, side model.Side) model.Signal {
	t.Helper()
	b := wideBook(58, side)
	b.Snapshot = true
	signals, fills := s.Apply(model.Event{Book: b, TradeReady: true}, at(58))
	require.Empty(t, signals)
	require.Empty(t, fills)
	signals, fills = s.Apply(model.Event{Book: wideBook(59, side)}, at(59))
	require.Empty(t, fills)
	require.Len(t, signals, 1)
	return signals[0]
}

func TestScannerCoverageFreshnessAndCooldown(t *testing.T) {
	s := engine(model.Linear)
	sig := start(t, s, model.Bid)
	require.Equal(t, 1000., sig.Notional)
	require.InDelta(t, 10.04, sig.Multiple, .01)
	require.Equal(t, time.Second, sig.Observed)
	tr := model.Trade{ID: "match", Ticks: 1000, Qty: .35, At: at(59)}
	s.Apply(model.Event{Trades: []model.Trade{tr, tr}}, at(59))
	got, ok := s.Current(sig, at(59))
	require.True(t, ok)
	require.Equal(t, .35, got.TradeQty)
	require.Equal(t, time.Second, got.TradeCoverage)
	s.MarkSent(sig, at(59))
	signals, _ := s.Apply(model.Event{Book: book(60, model.Bid)}, at(60))
	require.Empty(t, signals)
	s.Expire(at(64))
	_, ok = s.Current(sig, at(64))
	require.False(t, ok)
	signals, _ = s.Apply(model.Event{Book: book(65, model.Bid)}, at(65))
	require.Empty(t, signals)
	signals, _ = s.Apply(model.Event{Book: book(66, model.Bid)}, at(66))
	require.Empty(t, signals)
	s.Apply(model.Event{Book: book(1859, model.Bid)}, at(1859))
	signals, _ = s.Apply(model.Event{Book: book(1860, model.Bid)}, at(1860))
	require.Len(t, signals, 1)
	require.NotEqual(t, sig.Since, signals[0].Since)
}

func TestScannerFilters(t *testing.T) {
	for _, tc := range []struct {
		name   string
		change func(*model.Book)
	}{
		{"fewer than five neighbors", func(b *model.Book) { b.Bids = b.Bids[:5] }},
		{"under minimum", func(b *model.Book) { b.Bids[0].Qty = 4.99 }},
		{"below median multiple", func(b *model.Book) {
			for j := 1; j < len(b.Bids); j++ {
				b.Bids[j].Qty = 3
			}
		}},
		{"outside distance", func(b *model.Book) { b.Bids[0].Qty = 1; b.Bids[3].Qty = 10 }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s := engine(model.Spot)
			for i := 0; i < 3; i++ {
				b := book(i, model.Bid)
				b.Snapshot = i == 0
				tc.change(b)
				signals, _ := s.Apply(model.Event{Book: b, TradeReady: true}, at(i))
				require.Empty(t, signals)
			}
		})
	}
}

func TestLongLimitEntryAndTake(t *testing.T) {
	for _, market := range []model.Market{model.Spot, model.Linear} {
		t.Run(string(market), func(t *testing.T) {
			s := engine(market)
			sig := start(t, s, model.Bid)
			s.MarkSent(sig, at(59))
			_, fills := s.Apply(model.Event{Book: wideBook(60, model.Bid)}, at(60))
			require.Empty(t, fills)
			_, fills = s.Apply(model.Event{Book: book(61, model.Bid)}, at(61))
			require.Len(t, fills, 1)
			entry := fills[0]
			require.True(t, entry.Entry)
			require.Equal(t, model.Bid, entry.Side)
			require.Equal(t, 1.5, entry.Qty)
			require.InDelta(t, float64(sig.Ticks+1)*.1, entry.Price, 1e-10)
			b := book(62, model.Bid)
			b.Bids = append([]model.Level{{Ticks: 1007, Qty: 1.5}}, b.Bids...)
			for i := range b.Asks {
				b.Asks[i].Ticks += 9
			}
			_, fills = s.Apply(model.Event{Book: b}, at(62))
			require.Len(t, fills, 1)
			exit := fills[0]
			require.Equal(t, "take", exit.Reason)
			require.False(t, exit.Partial)
			cfg := strategy.DefaultConfig()
			fee := cfg.SpotFee
			if market == model.Linear {
				fee = cfg.LinearFee
			}
			require.InDelta(t, .9-1.5*(entry.Price+exit.Price)*fee, exit.NetPnL, 1e-10)
			require.InDelta(t, exit.Price*1.5*fee, exit.Fee, 1e-10)
		})
	}
}

func TestEntryAtConfiguredOffsetOnConfirmation(t *testing.T) {
	cfg := strategy.DefaultConfig()
	cfg.MinNotional = 500
	cfg.Persistence = time.Second
	cfg.EntryTicks = 2
	s := strategy.New(model.Instrument{Key: model.Key{Market: model.Spot, Symbol: "TESTUSDT"}, TickSize: .1, QtyStep: .1}, cfg)
	b := wideBook(58, model.Bid)
	b.Snapshot = true
	_, fills := s.Apply(model.Event{Book: b, TradeReady: true}, at(58))
	require.Empty(t, fills)
	signals, fills := s.Apply(model.Event{Book: wideBook(59, model.Bid)}, at(59))
	require.Len(t, signals, 1)
	require.Len(t, fills, 1)
	require.InDelta(t, 100.2, fills[0].Price, 1e-10)
}

func TestScannerIgnoresAskDensity(t *testing.T) {
	for _, market := range []model.Market{model.Spot, model.Linear} {
		t.Run(string(market), func(t *testing.T) {
			s := engine(market)
			b := book(58, model.Ask)
			b.Snapshot = true
			signals, fills := s.Apply(model.Event{Book: b, TradeReady: true}, at(58))
			require.Empty(t, signals)
			require.Empty(t, fills)

			signals, fills = s.Apply(model.Event{Book: book(59, model.Ask)}, at(59))
			require.Empty(t, signals)
			require.Empty(t, fills)
		})
	}
}

func TestForcedExitPreservesCauseAndConsumedLiquidity(t *testing.T) {
	s := engine(model.Spot)
	start(t, s, model.Bid)
	_, fills := s.Apply(model.Event{Book: book(60, model.Bid)}, at(60))
	require.Len(t, fills, 1)
	entry := fills[0]
	b := book(61, model.Bid)
	b.Bids = []model.Level{{Ticks: 998, Qty: .4}, {Ticks: 997, Qty: .2}}
	_, fills = s.Apply(model.Event{Book: b}, at(61))
	require.NotEmpty(t, fills)
	total := 0.
	net := 0.
	for _, f := range fills {
		require.Equal(t, "stop", f.Reason)
		require.True(t, f.Partial)
		total += f.Qty
		net += f.NetPnL
	}
	require.InDelta(t, .6, total, 1e-10)
	require.InDelta(t, (99.8-entry.Price)*.4+(99.7-entry.Price)*.2-(99.8*.4+99.7*.2+entry.Price*.6)*.001, net, 1e-9)
	for _, price := range []int{62, 63} {
		b.At = at(price)
		b.Received = b.At
		b.Asks[3].Qty++
		_, fills = s.Apply(model.Event{Book: b}, at(price))
		require.Empty(t, fills)
	}
	b.At = at(64)
	b.Received = b.At
	b.Bids[0].Qty = .7
	_, fills = s.Apply(model.Event{Book: b}, at(64))
	require.Len(t, fills, 1)
	require.InDelta(t, .3, fills[0].Qty, 1e-10)
	s.Apply(model.Event{Reset: true}, at(65))
	b.At = at(66)
	b.Received = b.At
	b.Bids = []model.Level{{Ticks: 1000, Qty: 2}}
	_, fills = s.Apply(model.Event{Book: b}, at(66))
	require.Empty(t, fills)
	b.At = at(67)
	b.Received = b.At
	b.Snapshot = true
	_, fills = s.Apply(model.Event{Book: b, TradeReady: true}, at(67))
	require.Len(t, fills, 1)
	require.Equal(t, "stop", fills[0].Reason)
	require.False(t, fills[0].Partial)
	require.Less(t, math.Abs(fills[0].Qty-.6), 1e-9)
}

func TestPendingEntryCancelsWhenDensityLeaves(t *testing.T) {
	for _, qty := range []float64{0, 4.9} {
		s := engine(model.Spot)
		start(t, s, model.Bid)
		b := book(60, model.Bid)
		b.Bids[0].Qty = qty
		b.Bids[1].Qty = 9 // A nearby wall does not replace the entry's density.
		if qty == 0 {
			b.Bids = b.Bids[1:]
		}
		_, fills := s.Apply(model.Event{Book: b}, at(60))
		require.Empty(t, fills)
	}
}

func TestPartialTakeThenLoss(t *testing.T) {
	s := engine(model.Spot)
	start(t, s, model.Bid)
	_, fills := s.Apply(model.Event{Book: book(60, model.Bid)}, at(60))
	require.Len(t, fills, 1)
	entry := fills[0]
	require.Nil(t, entry.Closed)
	b := book(61, model.Bid)
	b.Bids[0].Qty = .1
	b.Bids = append([]model.Level{{Ticks: 1007, Qty: .3}}, b.Bids...)
	for i := range b.Asks {
		b.Asks[i].Ticks += 10
	}
	_, fills = s.Apply(model.Event{Book: b}, at(61))
	require.Len(t, fills, 1)
	require.Equal(t, "take", fills[0].Reason)
	require.True(t, fills[0].Partial)
	take := fills[0]
	require.Nil(t, take.Closed)
	b = book(62, model.Bid)
	b.Bids = b.Bids[1:]
	b.Bids[4].Qty = 6 // 995 is five ticks from the origin.
	_, fills = s.Apply(model.Event{Book: b}, at(62))
	require.Empty(t, fills)
	b = book(63, model.Bid)
	b.Bids = b.Bids[1:]
	b.Bids = b.Bids[:3] // Original and replacement have disappeared.
	_, fills = s.Apply(model.Event{Book: b}, at(63))
	require.Len(t, fills, 1)
	require.Equal(t, "density_lost", fills[0].Reason)
	require.InDelta(t, 1.2, fills[0].Qty, 1e-12)
	require.False(t, fills[0].Partial)
	exit := fills[0]
	report := exit.Closed
	require.NotNil(t, report)
	require.Equal(t, at(60), report.EntryAt)
	require.Equal(t, at(63), exit.At)
	require.Equal(t, entry.Price, report.EntryPrice)
	require.InDelta(t, (take.Price*take.Qty+exit.Price*exit.Qty)/entry.Qty, report.ExitPrice, 1e-10)
	require.InDelta(t, 100, report.DensityPrice, 1e-10)
	require.InDelta(t, 10, report.DensityQty, 1e-10)
	require.InDelta(t, 99.8, report.StopPrice, 1e-10)
	require.InDelta(t, 100.7, report.TakePrice, 1e-10)
	require.InDelta(t, take.NetPnL+exit.NetPnL, report.NetPnL, 1e-10)
	require.InDelta(t, report.NetPnL/(entry.Price*entry.Qty), report.NetReturn, 1e-10)
	require.Equal(t, []string{"take", "density_lost"}, report.ExitReasons)
}

func TestFreshSnapshotAfterDisconnectCanRelocateOpenPositionBeforeTradeAck(t *testing.T) {
	s := engine(model.Spot)
	start(t, s, model.Bid)
	_, fills := s.Apply(model.Event{Book: book(60, model.Bid)}, at(60))
	require.Len(t, fills, 1)
	s.Apply(model.Event{Reset: true}, at(61))
	b := book(62, model.Bid)
	b.Snapshot = true
	b.Bids = b.Bids[1:]
	b.Bids[4].Qty = 6
	signals, fills := s.Apply(model.Event{Book: b}, at(62))
	require.Empty(t, signals)
	require.Empty(t, fills)
}

func TestBookFreshnessUsesBothClocksAndRequiresTape(t *testing.T) {
	s := engine(model.Linear)
	for i := 58; i <= 60; i++ {
		b := book(i, model.Bid)
		b.Snapshot = i == 58
		signals, _ := s.Apply(model.Event{Book: b}, at(i))
		require.Empty(t, signals)
	}
	s.Apply(model.Event{TradeReady: true}, at(60))
	s.Apply(model.Event{Book: book(61, model.Bid)}, at(61))
	signals, _ := s.Apply(model.Event{Book: book(62, model.Bid)}, at(62))
	require.Len(t, signals, 1)
	for _, stale := range []string{"exchange", "received"} {
		b := book(63, model.Bid)
		if stale == "exchange" {
			b.At = at(59)
		} else {
			b.Received = at(59)
		}
		got, fills := s.Apply(model.Event{Book: b}, at(63))
		require.Empty(t, got)
		require.Empty(t, fills)
		_, ok := s.Current(signals[0], at(63))
		require.False(t, ok)
	}
}

func TestEntryUsesLotSizeAndExchangeBounds(t *testing.T) {
	for _, tc := range []struct {
		name                                                     string
		minQty, maxQty, minNotional, minPrice, maxPrice, wantQty float64
	}{
		{"rounded max", 0, 1.29, 110, 98, 102, 1.2},
		{"below min qty", 1.6, 0, 0, 0, 0, 0},
		{"below min notional", 0, 1.29, 121, 0, 0, 0},
		{"unavailable target price", 0, 0, 0, 0, 100.6, 0},
		{"unavailable stop price", 0, 0, 0, 99.9, 0, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cfg := strategy.DefaultConfig()
			cfg.MinNotional = 500
			cfg.Persistence = time.Second
			s := strategy.New(model.Instrument{Key: model.Key{Market: model.Spot, Symbol: "TESTUSDT"}, TickSize: .1, QtyStep: .1, MinQty: tc.minQty, MaxQty: tc.maxQty, MinNotional: tc.minNotional, MinPrice: tc.minPrice, MaxPrice: tc.maxPrice}, cfg)
			start(t, s, model.Bid)
			_, fills := s.Apply(model.Event{Book: book(60, model.Bid)}, at(60))
			if tc.wantQty == 0 {
				require.Empty(t, fills)
				return
			}
			require.Len(t, fills, 1)
			require.InDelta(t, tc.wantQty, fills[0].Qty, 1e-10)
		})
	}
}

func TestUnexpectedFullSnapshotRestartsObservationAndCancelsSetup(t *testing.T) {
	s := engine(model.Spot)
	old := start(t, s, model.Bid)
	b := book(60, model.Bid)
	b.Snapshot = true
	signals, fills := s.Apply(model.Event{Book: b}, at(60))
	require.Empty(t, signals)
	require.Empty(t, fills)
	_, ok := s.Current(old, at(60))
	require.False(t, ok)
	signals, fills = s.Apply(model.Event{Book: book(61, model.Bid)}, at(61))
	require.Len(t, signals, 1)
	require.Len(t, fills, 1)
}

type delayedExecutor struct {
	orders   []strategy.Order
	lastBook *model.Book
}

func (e *delayedExecutor) ObserveBook(b *model.Book) { e.lastBook = b }
func (e *delayedExecutor) Submit(order strategy.Order, _ *model.Book) strategy.Execution {
	e.orders = append(e.orders, order)
	return strategy.Execution{}
}

func TestExternalExecutionConfirmsAndSizesPosition(t *testing.T) {
	cfg := strategy.DefaultConfig()
	cfg.MinNotional = 500
	cfg.Persistence = time.Second
	venue := new(delayedExecutor)
	s := strategy.NewWithExecutor(model.Instrument{Key: model.Key{Market: model.Linear, Symbol: "TESTUSDT"}, TickSize: .1, QtyStep: .1}, cfg, venue)
	start(t, s, model.Bid)
	_, fills := s.Apply(model.Event{Book: book(60, model.Bid)}, at(60))
	require.Empty(t, fills)
	require.Len(t, venue.orders, 1)
	entry := venue.orders[0]
	// IOC requests 2.4 coins, even though the paper book shows only 1.5 on offer.
	require.Equal(t, int64(24), entry.Steps)
	require.Equal(t, int64(1001), entry.LimitTicks)
	s.Apply(model.Event{Book: book(61, model.Bid)}, at(61))
	require.Len(t, venue.orders, 1)
	exec := strategy.Execution{OrderID: entry.ID, Steps: 7, Quote: 70.07}
	fill, ok := s.ConfirmExecution(exec, at(61))
	require.True(t, ok)
	require.True(t, fill.Entry)
	require.InDelta(t, .7, fill.Qty, 1e-12)
	_, ok = s.ConfirmExecution(exec, at(61))
	require.False(t, ok)
	s.Apply(model.Event{Book: book(62, model.Bid)}, at(62))
	require.Len(t, venue.orders, 1)
	b := book(63, model.Bid)
	b.Bids = b.Bids[2:]
	s.Apply(model.Event{Book: b}, at(63))
	require.Len(t, venue.orders, 2)
	exit := venue.orders[1]
	require.Equal(t, int64(7), exit.Steps)
	require.Equal(t, "stop", exit.Reason)
	s.Apply(model.Event{Book: b}, at(63))
	require.Len(t, venue.orders, 2)
	fill, ok = s.ConfirmExecution(strategy.Execution{OrderID: exit.ID, Steps: 7, Quote: 69.86}, at(63))
	require.True(t, ok)
	require.False(t, fill.Partial)
	require.InDelta(t, -.21-(70.07+69.86)*.00055, fill.NetPnL, 1e-12)
	require.Same(t, b, venue.lastBook)
}

func TestPendingEntryExpiresWithStaleBook(t *testing.T) {
	s := engine(model.Spot)
	start(t, s, model.Bid)
	s.Expire(at(63))
	_, fills := s.Apply(model.Event{Book: book(63, model.Bid)}, at(63))
	require.Empty(t, fills)
}

func TestAverageVolumeGatesAlertAndPendingEntry(t *testing.T) {
	for _, tc := range []struct {
		name           string
		average        float64
		until          time.Time
		lastQty        float64
		signals, fills int
	}{
		{"missing", 0, time.Time{}, 10, 0, 0},
		{"below alert threshold", 21, at(61), 10, 0, 0},
		{"at alert threshold only", 20, at(61), 10, 1, 0},
		{"equal", 10, at(61), 10, 1, 1},
		{"shrunk", 10, at(61), 9, 1, 0},
		{"expired", 10, at(60), 10, 1, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cfg := strategy.DefaultConfig()
			cfg.MinNotional = 500
			cfg.Persistence = time.Second
			cfg.RequireAverageVolume = true
			s := strategy.New(model.Instrument{Key: model.Key{Market: model.Spot, Symbol: "TESTUSDT"}, TickSize: .1, QtyStep: .1}, cfg)
			s.SetAverageVolume(volume.Average{Quantity: tc.average, Until: tc.until})
			first := wideBook(58, model.Bid)
			first.Snapshot = true
			s.Apply(model.Event{Book: first, TradeReady: true}, at(58))
			signals, fills := s.Apply(model.Event{Book: wideBook(59, model.Bid)}, at(59))
			require.Len(t, signals, tc.signals)
			if len(signals) > 0 {
				current, ok := s.Current(signals[0], at(59))
				require.True(t, ok)
				require.Equal(t, tc.average, current.AverageVolume)
			}
			require.Empty(t, fills)
			last := book(60, model.Bid)
			last.Bids[0].Qty = tc.lastQty
			_, fills = s.Apply(model.Event{Book: last}, at(60))
			require.Len(t, fills, tc.fills)
		})
	}
}

func TestClosedTradeKeepsEntryAverageVolumeAfterRefresh(t *testing.T) {
	cfg := strategy.DefaultConfig()
	cfg.MinNotional = 500
	cfg.Persistence = time.Second
	cfg.RequireAverageVolume = true
	s := strategy.New(model.Instrument{Key: model.Key{Market: model.Spot, Symbol: "TESTUSDT"}, TickSize: .1, QtyStep: .1}, cfg)
	s.SetAverageVolume(volume.Average{Quantity: 0, Until: at(61)})
	require.True(t, start(t, s, model.Bid).HasAverageVolume)
	_, fills := s.Apply(model.Event{Book: book(60, model.Bid)}, at(60))
	require.Len(t, fills, 1)
	require.True(t, fills[0].Entry)

	s.SetAverageVolume(volume.Average{Quantity: 100, Until: at(120)})
	take := book(61, model.Bid)
	take.Bids = append([]model.Level{{Ticks: 1007, Qty: 1.5}}, take.Bids...)
	for i := range take.Asks {
		take.Asks[i].Ticks += 9
	}
	_, fills = s.Apply(model.Event{Book: take}, at(61))
	require.Len(t, fills, 1)
	require.NotNil(t, fills[0].Closed)
	require.NotNil(t, fills[0].Closed.AverageVolume)
	require.Zero(t, *fills[0].Closed.AverageVolume)
}

func TestBounceBuysAtThirdMinuteAsk(t *testing.T) {
	cfg := strategy.DefaultConfig()
	cfg.MinNotional = 500
	cfg.Persistence = time.Second
	cfg.Freshness = 70 * time.Second
	cfg.CandleEntries = true
	cfg.RequireAverageVolume = true
	s := strategy.New(model.Instrument{Key: model.Key{Market: model.Spot, Symbol: "TESTUSDT"}, TickSize: .1, QtyStep: .1}, cfg)

	s.SetAverageVolume(volume.Average{Quantity: 9.25, Until: at(180)})
	first := book(-1, model.Bid)
	first.Snapshot = true
	_, fills := s.Apply(model.Event{TradeReady: true, Book: first}, at(-1))
	require.Empty(t, fills)
	_, fills = s.Apply(model.Event{Book: book(0, model.Bid)}, at(0))
	require.Empty(t, fills, "density alone placed an entry")
	_, fills = s.Apply(model.Event{Candle: &model.Candle{Start: at(0), End: at(60), Low: 995, Close: 1000}}, at(60))
	require.Empty(t, fills, "touch placed an entry")
	_, fills = s.Apply(model.Event{Book: book(60, model.Bid)}, at(60))
	require.Empty(t, fills, "second minute placed an entry")
	_, fills = s.Apply(model.Event{Candle: &model.Candle{Start: at(60), End: at(120), Low: 1000, Close: 1002}}, at(120))
	require.Empty(t, fills, "entry used an old quote")

	third := book(120, model.Bid)
	third.Bids[0].Qty = 3
	for i := range third.Asks {
		third.Asks[i].Ticks += 3
	}
	_, fills = s.Apply(model.Event{Book: third}, at(120))
	require.Len(t, fills, 1)
	require.True(t, fills[0].Entry)
	require.InDelta(t, 100.4, fills[0].Price, 1e-8)
}

func TestAlertWaitsForBidWithinTwoTicks(t *testing.T) {
	for _, price := range []int64{500, 101_000} {
		cfg := strategy.DefaultConfig()
		cfg.MinNotional = 100
		cfg.Persistence = time.Second
		cfg.CandleEntries = true
		s := strategy.New(model.Instrument{TickSize: 1}, cfg)

		update := func(second int, distance int64) []model.Signal {
			b := wideBook(second, model.Bid)
			b.Snapshot = second == 0
			for i := range b.Bids {
				b.Bids[i].Ticks += price - 1000
			}
			bestBid := price + distance
			b.Bids = append([]model.Level{{Ticks: bestBid, Qty: 1}}, b.Bids...)
			b.Asks = []model.Level{{Ticks: bestBid + 1, Qty: 1}}
			signals, _ := s.Apply(model.Event{Book: b, TradeReady: true}, at(second))
			return signals
		}

		require.Empty(t, update(0, 3))
		require.Empty(t, update(1, 3))
		require.Empty(t, update(2, 2))
		signals := update(3, 2)
		require.Len(t, signals, 1)
		require.Equal(t, at(2), signals[0].Since)
		_, ok := s.Current(signals[0], at(3))
		require.True(t, ok)

		require.Empty(t, update(4, 3))
		_, ok = s.Current(signals[0], at(4))
		require.False(t, ok)
	}
}

func TestFixedLimitEntryRetainsFortyPercent(t *testing.T) {
	for _, tc := range []struct {
		name string
		qty  float64
		fill bool
	}{
		{"retains at forty percent", 4, true},
		{"cancels below forty percent", 3.9, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cfg := strategy.DefaultConfig()
			cfg.FixedLimitEntry = true
			cfg.MinNotional = 500
			cfg.Persistence = time.Second
			cfg.EntryTicks = 3
			cfg.StopTicks = 20
			s := strategy.New(model.Instrument{Key: model.Key{Market: model.Spot, Symbol: "TESTUSDT"}, TickSize: .1, QtyStep: .1}, cfg)
			start(t, s, model.Bid)
			b := book(60, model.Bid)
			b.Bids[0].Qty = tc.qty
			_, fills := s.Apply(model.Event{Book: b}, at(60))
			if !tc.fill {
				require.Empty(t, fills)
				return
			}
			require.Len(t, fills, 1)
			require.InDelta(t, 100.1, fills[0].Price, 1e-10)
			b = book(61, model.Bid)
			b.Bids = b.Bids[2:]
			b.Bids[0].Qty = 2
			_, fills = s.Apply(model.Event{Book: b}, at(61))
			require.Len(t, fills, 1)
			require.Equal(t, "stop", fills[0].Reason)
			require.NotNil(t, fills[0].Closed)
			require.InDelta(t, 99.8, fills[0].Closed.StopPrice, 1e-10)
		})
	}
}
