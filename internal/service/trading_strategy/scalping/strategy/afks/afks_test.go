package afks

import "testing"

func TestTickerAndDefaults(t *testing.T) {
	s := New()
	if s.Ticker() != "AFKS" {
		t.Errorf("Ticker = %q, want AFKS", s.Ticker())
	}
	p := DefaultParams()
	if p.ADXTrendLevel <= p.ADXRangeLevel {
		t.Errorf("ADXTrendLevel (%v) must exceed ADXRangeLevel (%v)", p.ADXTrendLevel, p.ADXRangeLevel)
	}
	if p.EMAPeriod <= 0 || p.ADXPeriod <= 0 || p.RSIPeriod <= 0 || p.DonchianPeriod <= 0 || p.ATRPeriod <= 0 {
		t.Errorf("all periods must be positive: %+v", p)
	}
	if p.TrendFilterPeriod <= 0 {
		t.Errorf("TrendFilterPeriod = %d, want > 0 (HTF filter on by default)", p.TrendFilterPeriod)
	}
	if p.TrailArmATR <= 0 || p.MinRR <= 0 || p.MinATRFrac <= 0 {
		t.Errorf("quality knobs must be opted in (non-zero): %+v", p)
	}
}
