package config

import (
	"context"
	"testing"

	"github.com/heetch/confita"
	"github.com/heetch/confita/backend/env"
)

func TestFVGImbalanceDefaultsAndEnabled(t *testing.T) {
	c := NewFVGImbalanceConfig()
	if c.Enabled() || c.TradeEnabled || c.BuyPct != 5 {
		t.Fatalf("unsafe defaults: %+v", c)
	}
	c.Tickers = []string{" ", "BSPB"}
	if !c.Enabled() {
		t.Fatal("configured ticker must enable strategy")
	}
	c.Tickers = []string{"", "  "}
	if c.Enabled() {
		t.Fatal("blank tickers must not enable strategy")
	}
}

func TestFVGImbalanceLoadsEnvironment(t *testing.T) {
	t.Setenv("FVG_IMBALANCE_TICKERS", "BSPB,SNGS")
	t.Setenv("FVG_IMBALANCE_BUY_PCT", "5")
	t.Setenv("FVG_IMBALANCE_TRADE_ENABLED", "false")
	t.Setenv("FVG_IMBALANCE_NOTIFY_ENABLED", "true")
	c := NewFVGImbalanceConfig()
	if err := confita.NewLoader(env.NewBackend()).Load(context.Background(), c); err != nil {
		t.Fatal(err)
	}
	if len(c.Tickers) != 2 || c.Tickers[0] != "BSPB" || c.Tickers[1] != "SNGS" || c.TradeEnabled || !c.NotifyEnabled {
		t.Fatalf("unexpected config: %+v", c)
	}
}
