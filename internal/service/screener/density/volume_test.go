package density

import (
	"testing"
	"time"

	"tinvest/internal/service/screener/volume"
)

func TestDetectorRequiresCurrentAverageVolume(t *testing.T) {
	cfg := DefaultConfig()
	cfg.RequireAverageVolume = true
	cfg.MinRubles = 1
	cfg.Persistence = 0
	now := time.Date(2026, 11, 19, 10, 0, 0, 0, time.UTC)
	for _, tc := range []struct {
		average                float64
		until                  time.Time
		wantTrading, wantAlert int
	}{
		{0, time.Time{}, 0, 0},
		{-1, now.Add(time.Hour), 0, 0},
		{201, now.Add(time.Hour), 0, 0},
		{200, now.Add(time.Hour), 0, 1},
		{100, now.Add(time.Hour), 1, 1},
		{99, now.Add(time.Hour), 1, 1},
		{99, now, 0, 0},
	} {
		instruments := []Instrument{{ID: "test", PriceStep: Price{Units: 1}, AverageVolume: volume.Average{Quantity: tc.average, Until: tc.until}}}
		b := Book{InstrumentID: "test", Lot: 10, Time: now, Asks: []Level{{Price: Price{Units: 1001}, Quantity: 1}}}
		for offset := range int64(6) {
			b.Bids = append(b.Bids, Level{Price: Price{Units: 1000 - offset}, Quantity: 1})
		}
		b.Bids[0].Quantity = 100
		trading := NewDetector(cfg)
		trading.SetInstruments(instruments)
		if got := len(trading.Update(b, now)); got != tc.wantTrading {
			t.Errorf("average %v, until %v: got %d trading signals, want %d", tc.average, tc.until, got, tc.wantTrading)
		}

		screener := newScreenerDetector(cfg)
		screener.SetInstruments(instruments)
		signals := screener.Update(b, now)
		if len(signals) != tc.wantAlert {
			t.Errorf("average %v, until %v: got %d alerts, want %d", tc.average, tc.until, len(signals), tc.wantAlert)
		}
		if len(signals) > 0 && signals[0].MinLots != tc.average {
			t.Errorf("got reported M5 volume %v, want %v", signals[0].MinLots, tc.average)
		}
	}
}
