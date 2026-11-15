package live

import (
	"testing"

	"tinvest/internal/config"
	"tinvest/internal/service/trading_strategy/fvg_imbalance/strategy/rtkm"
	"tinvest/internal/service/trading_strategy/livecore/statestore"
)

func TestRTKMLiveParameters(t *testing.T) {
	s := New(config.NewFVGImbalanceConfig(), nil)
	if _, ok := s.Decider("RTKM"); !ok {
		t.Fatal("RTKM is not registered")
	}
	if p, ok := params("RTKM"); !ok || p != rtkm.Params() {
		t.Errorf("RTKM params=%+v, registered=%v", p, ok)
	}
	if stop, reason := s.DesiredStop("RTKM", statestore.Entry{EntryPrice: 100, EntryATR: 10}); stop != 100-rtkm.Params().StopATR*10 || reason != "SL" {
		t.Errorf("RTKM stop=%v reason=%q, want stop from ticker parameters", stop, reason)
	}
}
