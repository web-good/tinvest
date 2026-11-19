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
	d := NewDetector(cfg)
	for _, tc := range []struct {
		average float64
		until   time.Time
		want    int
	}{
		{0, time.Time{}, 0},
		{101, now.Add(time.Hour), 0},
		{100, now.Add(time.Hour), 1},
		{99, now.Add(time.Hour), 1},
		{99, now, 0},
	} {
		d.Reset("test")
		d.SetInstruments([]Instrument{{ID: "test", PriceStep: Price{Units: 1}, AverageVolume: volume.Average{Quantity: tc.average, Until: tc.until}}})
		b := Book{InstrumentID: "test", Lot: 10, Time: now, Asks: []Level{{Price: Price{Units: 1001}, Quantity: 1}}}
		for offset := range int64(6) {
			b.Bids = append(b.Bids, Level{Price: Price{Units: 1000 - offset}, Quantity: 1})
		}
		b.Bids[0].Quantity = 100
		if got := len(d.Update(b, now)); got != tc.want {
			t.Errorf("average %v, until %v: got %d signals, want %d", tc.average, tc.until, got, tc.want)
		}
	}
}
