package live

import (
	"testing"

	"tinvest/internal/config"
	"tinvest/internal/service/trading_strategy/fvg_imbalance/strategy/mrkc"
	"tinvest/internal/service/trading_strategy/livecore/statestore"
)

func TestMRKCLiveParameters(t *testing.T) {
	s := New(config.NewFVGImbalanceConfig(), nil)
	if _, ok := s.Decider("MRKC"); !ok {
		t.Fatal("MRKC is not registered")
	}
	if p, ok := params("MRKC"); !ok || p != mrkc.Params() {
		t.Errorf("MRKC params=%+v, registered=%v", p, ok)
	}
	if stop, reason := s.DesiredStop("MRKC", statestore.Entry{EntryPrice: 100, EntryATR: 10}); stop != 100-mrkc.Params().StopATR*10 || reason != "SL" {
		t.Errorf("MRKC stop=%v reason=%q, want stop from ticker parameters", stop, reason)
	}
}
