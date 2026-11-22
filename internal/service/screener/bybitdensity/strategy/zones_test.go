package strategy

import (
	"testing"
	"time"

	"tinvest/internal/service/screener/bybitdensity/model"
)

func TestZoneApproachAfterConfirmation(t *testing.T) {
	cfg := DefaultConfig()
	cfg.ZoneAlerts = true
	cfg.ZoneWidthUSDT = 2
	cfg.RequireAverageVolume = true // Zone alerts work without a candle reference.
	inst := model.Instrument{Key: model.Key{Market: model.Spot, Symbol: "BTCUSDT"}, TickSize: 1, QtyStep: .001}
	s := New(inst, cfg)
	start := time.Now()
	update := func(second int, best int64, qty float64) []model.Signal {
		at := start.Add(time.Duration(second) * time.Second)
		b := &model.Book{At: at, Received: at, Snapshot: second == 0, Asks: []model.Level{{Ticks: best + 2, Qty: 1}}}
		for tick := best; tick >= 9960; tick-- {
			size := 3.0 // The target is only twice as large as its neighbors.
			if tick == 9980 || tick == 9981 {
				size = qty
			}
			b.Bids = append(b.Bids, model.Level{Ticks: tick, Qty: size})
		}
		sig, _ := s.Apply(model.Event{Book: b, TradeReady: true}, at)
		return sig
	}
	for i := 0; i <= 8; i++ {
		if sig := update(i, 10000, 6); len(sig) > 0 {
			t.Fatal("distant zone alerted", sig)
		}
	}
	tradeAt := start.Add(8500 * time.Millisecond)
	s.Apply(model.Event{Trades: []model.Trade{
		{ID: "sale", Sell: true, Ticks: 9981, Qty: 1, At: tradeAt},
		{ID: "buy", Ticks: 9980, Qty: 2, At: tradeAt},
	}}, tradeAt)
	sig := update(9, 9985, 3.6)
	if len(sig) != 1 || sig[0].Qty != 7.2 || sig[0].Ticks != 9980 || sig[0].ZoneHigh != 9981 || sig[0].TradeQty != 1 || !sig[0].HeldAfterSales {
		t.Fatal("missing supported zone at approach", sig)
	}
	if _, ok := s.Current(sig[0], start.Add(9*time.Second)); !ok {
		t.Fatal("fresh zone rejected before delivery")
	}
	update(10, 10000, 3.6)
	if _, ok := s.Current(sig[0], start.Add(10*time.Second)); ok {
		t.Fatal("zone still deliverable after retreat")
	}
	update(11, 9985, 3.5)
	if _, ok := s.Current(sig[0], start.Add(11*time.Second)); ok {
		t.Fatal("depleted zone still deliverable")
	}
}
