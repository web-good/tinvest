package strategy_test

import (
	"testing"
	"time"

	"tinvest/internal/service/screener/bybitdensity/model"
	"tinvest/internal/service/screener/bybitdensity/strategy"

	"github.com/stretchr/testify/require"
)

func TestBreakoutRequiresBuyingAtTheWall(t *testing.T) {
	for _, tc := range []struct {
		name       string
		taker      model.Side
		tradeQty   float64
		bidQty     float64
		offBook    bool
		refilled   bool
		reconnect  bool
		failedBid  bool
		afterCross bool
		wantFills  int
	}{
		{name: "cancellation", bidQty: 50},
		{name: "sell trade", taker: model.Ask, tradeQty: 100, bidQty: 50},
		{name: "off book trade", taker: model.Bid, tradeQty: 100, bidQty: 50, offBook: true},
		{name: "visible refill", taker: model.Bid, tradeQty: 100, bidQty: 50, refilled: true},
		{name: "reset cancels entry", taker: model.Bid, tradeQty: 100, bidQty: 50, reconnect: true},
		{name: "bid falls after entry", taker: model.Bid, tradeQty: 100, bidQty: 50, failedBid: true, wantFills: 2},
		{name: "trade matched after crossing", taker: model.Bid, bidQty: 50, afterCross: true},
		{name: "partial absorption", taker: model.Bid, tradeQty: 50, bidQty: 50},
		{name: "both variants", taker: model.Bid, tradeQty: 100, bidQty: 50, wantFills: 2},
		{name: "baseline only", taker: model.Bid, tradeQty: 100, bidQty: 10, wantFills: 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cfg := strategy.DefaultConfig()
			cfg.SOLBreakout = true
			cfg.MinNotional, cfg.Persistence = 1000, time.Second
			instrument := model.Instrument{Key: model.Key{Market: model.Spot, Symbol: "SOLUSDT"}, TickSize: 1, QtyStep: .01}
			s := strategy.New(instrument, cfg)
			start := time.Unix(1800000000, 0)
			book := func(offset time.Duration, bid, ask int64, bidQty, askQty float64, snapshot bool) []model.Fill {
				now := start.Add(offset)
				b := &model.Book{At: now, Received: now, Snapshot: snapshot}
				for i := int64(0); i < 6; i++ {
					b.Bids = append(b.Bids, model.Level{Ticks: bid - i, Qty: bidQty})
					b.Asks = append(b.Asks, model.Level{Ticks: ask + i, Qty: 10})
				}
				b.Asks[0].Qty = askQty
				_, fills := s.Apply(model.Event{Key: instrument.Key, Book: b, TradeReady: true}, now)
				return fills
			}

			require.Empty(t, book(0, 99, 100, 10, 100, true))
			require.Empty(t, book(time.Second, 99, 100, 10, 100, false))
			if tc.refilled {
				require.Empty(t, book(1050*time.Millisecond, 99, 100, 10, 120, false))
			}
			if tc.afterCross {
				trade := model.Trade{ID: "early", TakerSide: model.Bid, Ticks: 100, Qty: 100, At: start.Add(1210 * time.Millisecond)}
				_, fills := s.Apply(model.Event{Key: instrument.Key, Trades: []model.Trade{trade}}, start.Add(1190*time.Millisecond))
				require.Empty(t, fills)
			}
			require.Empty(t, book(1200*time.Millisecond, 100, 101, tc.bidQty, 10, false))
			if tc.tradeQty > 0 {
				at := start.Add(1100 * time.Millisecond)
				trade := model.Trade{ID: "1", TakerSide: tc.taker, OffBook: tc.offBook, Ticks: 100, Qty: tc.tradeQty, At: at}
				_, fills := s.Apply(model.Event{Key: instrument.Key, Trades: []model.Trade{trade, trade}}, start.Add(1250*time.Millisecond))
				require.Empty(t, fills)
			}
			if tc.reconnect {
				_, fills := s.Apply(model.Event{Key: instrument.Key, Reset: true}, start.Add(1275*time.Millisecond))
				require.Empty(t, fills)
			}
			entries := book(1300*time.Millisecond, 100, 101, tc.bidQty, 10, tc.reconnect)
			require.Len(t, entries, tc.wantFills)
			if len(entries) == 0 {
				return
			}
			require.Equal(t, model.Breakout, entries[0].Variant)
			require.Equal(t, 101.0, entries[0].Price)
			if len(entries) == 2 {
				require.Equal(t, model.BreakoutImbalance, entries[1].Variant)
			}

			if tc.failedBid {
				exits := book(1400*time.Millisecond, 99, 100, tc.bidQty, 10, false)
				require.Len(t, exits, 2)
				require.Equal(t, "breakout_failed", exits[0].Reason)
				return
			}
			require.Empty(t, book(10*time.Second, 100, 101, tc.bidQty, 10, false))
			exits := book(11300*time.Millisecond, 100, 101, tc.bidQty, 10, false)
			require.Len(t, exits, len(entries))
			require.Equal(t, "timeout", exits[0].Reason)
			require.Equal(t, 100.0, exits[0].Price)
			require.InDelta(t, -2.96647, exits[0].Closed.NetPnL, .00001)
			require.NotNil(t, exits[0].Closed.Imbalance)
		})
	}
}
