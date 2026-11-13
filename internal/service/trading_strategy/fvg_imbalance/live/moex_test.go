package live

import (
	"testing"

	"tinvest/internal/config"
	"tinvest/internal/service/trading_strategy/fvg_imbalance/strategy/moex"
	"tinvest/internal/service/trading_strategy/livecore/statestore"
)

func TestMOEXLiveParameters(t *testing.T) {
	s := New(config.NewFVGImbalanceConfig(), nil)
	if _, ok := s.Decider("MOEX"); !ok {
		t.Fatal("MOEX is not registered")
	}
	if p, ok := params("MOEX"); !ok || p != moex.Params() {
		t.Errorf("MOEX params=%+v, registered=%v", p, ok)
	}
	if stop, reason := s.DesiredStop("MOEX", statestore.Entry{EntryPrice: 100, EntryATR: 10}); stop != 88 || reason != "SL" {
		t.Errorf("MOEX stop=%v reason=%q, want 88/SL", stop, reason)
	}
}
