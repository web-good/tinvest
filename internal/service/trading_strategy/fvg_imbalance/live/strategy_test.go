package live

import (
	"testing"
	"tinvest/internal/config"
	"tinvest/internal/enum"
	"tinvest/internal/service/trading_strategy/livecore/statestore"
)

func TestStrategyRegistryAndProtection(t *testing.T) {
	s := New(config.NewFVGImbalanceConfig(), nil)
	if s.BarInterval() != enum.Minutes30 {
		t.Fatal("wrong interval")
	}
	for _, ticker := range []string{"BSPB", "SNGS"} {
		if _, ok := s.Decider(ticker); !ok {
			t.Errorf("missing %s", ticker)
		}
		if stop, reason := s.DesiredStop(ticker, statestore.Entry{EntryPrice: 100, EntryATR: 10}); stop <= 0 || reason != "SL" {
			t.Errorf("%s stop=%v reason=%q", ticker, stop, reason)
		}
	}
	if _, ok := s.Decider("UNKNOWN"); ok {
		t.Fatal("unexpected ticker")
	}
}
