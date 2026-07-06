package config

import "testing"

func TestNewReversionConfig_Defaults(t *testing.T) {
	c := NewReversionConfig()
	if len(c.Tickers) != 3 || c.Tickers[0] != "UGLD" || c.Tickers[1] != "EUTR" || c.Tickers[2] != "NVTK" {
		t.Fatalf("default Tickers = %v, want [UGLD EUTR NVTK]", c.Tickers)
	}
	if c.BuyPct != 10 {
		t.Fatalf("default BuyPct = %v, want 10", c.BuyPct)
	}
	if c.TradeEnabled {
		t.Fatalf("TradeEnabled default = true, want false (safe default)")
	}
}

func TestNewReversionConfig_TokenHasNoDefault(t *testing.T) {
	c := NewReversionConfig()
	if c.Token != "" {
		t.Fatalf("default Token = %q, want empty (must come from REVERSION_TOKEN env)", c.Token)
	}
}
