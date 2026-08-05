package config

import "testing"

func TestNewRSIPullbackConfig_Defaults(t *testing.T) {
	c := NewRSIPullbackConfig()
	want := []string{"UGLD", "T", "GAZP"}
	if len(c.Tickers) != len(want) {
		t.Fatalf("default Tickers = %v, want %v", c.Tickers, want)
	}
	for i, w := range want {
		if c.Tickers[i] != w {
			t.Fatalf("default Tickers = %v, want %v", c.Tickers, want)
		}
	}
	if c.BuyPct != 5 {
		t.Fatalf("default BuyPct = %v, want 5", c.BuyPct)
	}
	if c.Schedule != "1,31 6-23 * * *" {
		t.Fatalf("default Schedule = %q, want \"1,31 6-23 * * *\"", c.Schedule)
	}
}

// Забытый флаг никогда не должен выставить реальный ордер.
func TestNewRSIPullbackConfig_TradeDisabledByDefault(t *testing.T) {
	if NewRSIPullbackConfig().TradeEnabled {
		t.Fatal("TradeEnabled default = true, want false")
	}
}

func TestNewRSIPullbackConfig_TokenHasNoDefault(t *testing.T) {
	if tok := NewRSIPullbackConfig().Token; tok != "" {
		t.Fatalf("default Token = %q, want empty (must come from RSI_PULLBACK_TOKEN env)", tok)
	}
}
