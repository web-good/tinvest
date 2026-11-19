package density

import (
	"testing"
	"time"
)

func supportHistory() ([]SupportCandle, time.Time) {
	start := time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)
	candles := make([]SupportCandle, 30)
	for i := range candles {
		candles[i] = SupportCandle{Time: start.Add(time.Duration(i) * time.Hour), Low: Price{Units: 101}, High: Price{Units: 103}, Close: Price{Units: 102}, Complete: true}
	}
	candles[3].Low = Price{Units: 100}
	candles[15].Low = Price{Units: 100, Nano: 100000000}
	return candles, start.Add(30 * time.Hour)
}

func TestFindSupportsRequiresIndependentConfirmedBounces(t *testing.T) {
	candles, now := supportHistory()
	cfg := DefaultSupportConfig()
	got := FindSupports(candles, cfg, now)
	if len(got) != 1 || got[0].Price != (Price{Units: 100}) || got[0].Touches != 2 {
		t.Fatalf("supports: %+v", got)
	}
	// A later close through the zone invalidates both historical touches.
	candles[29].Low = Price{Units: 98}
	candles[29].Close = Price{Units: 99}
	if got = FindSupports(candles, cfg, now); len(got) != 0 {
		t.Fatalf("broken support: %+v", got)
	}
}

func TestFindSupportsRejectsUnconfirmedAndUnrelatedHistory(t *testing.T) {
	for _, tc := range []struct {
		name   string
		change func([]SupportCandle, *SupportConfig)
	}{
		{"incomplete touch", func(c []SupportCandle, _ *SupportConfig) { c[15].Complete = false }},
		{"no rebound", func(c []SupportCandle, _ *SupportConfig) {
			for i := 16; i <= 21; i++ {
				c[i].Close = Price{Units: 100, Nano: 200000000}
				c[i].Low = Price{Units: 100, Nano: 100000000}
			}
		}},
		{"not independent", func(_ []SupportCandle, c *SupportConfig) { c.MinSeparation = 24 * time.Hour }},
		{"outside lookback", func(_ []SupportCandle, c *SupportConfig) { c.LookbackDays = 1 }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c, now := supportHistory()
			cfg := DefaultSupportConfig()
			tc.change(c, &cfg)
			if got := FindSupports(c, cfg, now); len(got) != 0 {
				t.Fatalf("unexpected support %+v", got)
			}
		})
	}
	c, now := supportHistory()
	if got := FindSupports(c, DefaultSupportConfig(), c[18].Time); len(got) != 0 {
		t.Fatalf("used future confirmation: %+v", got)
	}
	// Replayed timestamps must not become independent touches.
	dup := append(append([]SupportCandle(nil), c...), c...)
	if got := FindSupports(dup, DefaultSupportConfig(), now); len(got) != 1 || got[0].Touches != 2 {
		t.Fatalf("duplicates %+v", got)
	}
}

func TestDetectorAppliesIndividualSizeAndSupport(t *testing.T) {
	now := time.Date(2026, 10, 2, 0, 0, 0, 0, time.UTC)
	for _, tc := range []struct {
		name     string
		turnover float64
		qty      int64
		support  Price
		age      time.Duration
		want     int
	}{
		{"inclusive threshold", 2e9, 400, Price{Units: 250}, 0, 1},
		{"below individual threshold", 2e9, 399, Price{Units: 250}, 0, 0},
		{"global minimum", 1e8, 199, Price{Units: 250}, 0, 0},
		{"no turnover", 0, 400, Price{Units: 250}, 0, 0},
		{"distant support", 2e9, 400, Price{Units: 240}, 0, 0},
		{"missing support", 2e9, 400, Price{}, 0, 0},
		{"expired history", 2e9, 400, Price{Units: 250}, 24 * time.Hour, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cfg := DefaultConfig()
			cfg.TurnoverFraction = .005
			cfg.Support = DefaultSupportConfig()
			cfg.Support.Enabled = true
			cfg.Persistence = 0
			d := NewDetector(cfg)
			d.SetInstruments([]Instrument{{ID: "x", Lot: 100, PriceStep: Price{Units: 1}, Turnover: tc.turnover, ProfileAt: now.Add(-tc.age), Supports: []Support{{Price: tc.support, Touches: 2}}}})
			b := Book{InstrumentID: "x", Lot: 100, Time: now, Asks: []Level{{Price: Price{Units: 251}, Quantity: 1}}, Bids: []Level{{Price: Price{Units: 250}, Quantity: tc.qty}, {Price: Price{Units: 249}, Quantity: 10}, {Price: Price{Units: 248}, Quantity: 10}, {Price: Price{Units: 247}, Quantity: 10}, {Price: Price{Units: 246}, Quantity: 10}, {Price: Price{Units: 245}, Quantity: 10}}}
			got := d.Update(b, now)
			if len(got) != tc.want {
				t.Fatalf("signals %+v", got)
			}
			if tc.want == 1 {
				if got[0].MinRubles != 1e7 || got[0].SupportPrice != (Price{Units: 250}) {
					t.Fatalf("metadata %+v", got[0])
				}
				d.Expire(now.Add(24 * time.Hour))
				if len(d.Candidates()) != 0 {
					t.Fatal("expired profile retained candidate")
				}
			}
		})
	}
}
