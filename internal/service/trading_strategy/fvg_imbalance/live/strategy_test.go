package live

import (
	"testing"
	"tinvest/internal/config"
	"tinvest/internal/enum"
	"tinvest/internal/service/trading_strategy/fvg_imbalance/strategy/fixr"
	"tinvest/internal/service/trading_strategy/fvg_imbalance/strategy/gmkn"
	"tinvest/internal/service/trading_strategy/fvg_imbalance/strategy/irao"
	"tinvest/internal/service/trading_strategy/fvg_imbalance/strategy/magn"
	"tinvest/internal/service/trading_strategy/fvg_imbalance/strategy/mtlr"
	"tinvest/internal/service/trading_strategy/fvg_imbalance/strategy/phor"
	"tinvest/internal/service/trading_strategy/fvg_imbalance/strategy/posi"
	"tinvest/internal/service/trading_strategy/fvg_imbalance/strategy/sberp"
	"tinvest/internal/service/trading_strategy/fvg_imbalance/strategy/x5"
	"tinvest/internal/service/trading_strategy/livecore/statestore"
)

func TestStrategyRegistryAndProtection(t *testing.T) {
	s := New(config.NewFVGImbalanceConfig(), nil)
	if s.BarInterval() != enum.Minutes30 {
		t.Fatal("wrong interval")
	}
	for _, ticker := range []string{"AFKS", "ALRS", "BSPB", "FIXR", "GAZP", "GMKN", "IRAO", "MAGN", "MTLR", "PHOR", "POSI", "ROSN", "SBER", "SBERP", "SNGS", "VTBR", "YDEX"} {
		if _, ok := s.Decider(ticker); !ok {
			t.Errorf("missing %s", ticker)
		}
		if stop, reason := s.DesiredStop(ticker, statestore.Entry{EntryPrice: 100, EntryATR: 10}); stop <= 0 || reason != "SL" {
			t.Errorf("%s stop=%v reason=%q", ticker, stop, reason)
		}
	}
	if p, ok := params("MTLR"); !ok || p != mtlr.Params() {
		t.Errorf("MTLR params=%+v, registered=%v", p, ok)
	}
	if stop, reason := s.DesiredStop("MTLR", statestore.Entry{EntryPrice: 100, EntryATR: 10}); stop != 100-mtlr.Params().StopATR*10 || reason != "SL" {
		t.Errorf("MTLR stop=%v reason=%q, want stop from ticker parameters", stop, reason)
	}
	if p, ok := params("FIXR"); !ok || p != fixr.Params() {
		t.Errorf("FIXR params=%+v, registered=%v", p, ok)
	}
	if stop, reason := s.DesiredStop("FIXR", statestore.Entry{EntryPrice: 100, EntryATR: 10}); stop != 83 || reason != "SL" {
		t.Errorf("FIXR stop=%v reason=%q, want 83/SL", stop, reason)
	}
	if p, ok := params("GMKN"); !ok || p != gmkn.Params() {
		t.Errorf("GMKN params=%+v, registered=%v", p, ok)
	}
	if stop, reason := s.DesiredStop("GMKN", statestore.Entry{EntryPrice: 100, EntryATR: 10}); stop != 100-gmkn.Params().StopATR*10 || reason != "SL" {
		t.Errorf("GMKN stop=%v reason=%q, want stop from ticker parameters", stop, reason)
	}
	if p, ok := params("IRAO"); !ok || p != irao.Params() {
		t.Errorf("IRAO params=%+v, registered=%v", p, ok)
	}
	if p, ok := params("MAGN"); !ok || p != magn.Params() {
		t.Errorf("MAGN params=%+v, registered=%v", p, ok)
	}
	if stop, reason := s.DesiredStop("MAGN", statestore.Entry{EntryPrice: 100, EntryATR: 10}); stop != 95 || reason != "SL" {
		t.Errorf("MAGN stop=%v reason=%q, want 95/SL", stop, reason)
	}
	if p, ok := params("PHOR"); !ok || p != phor.Params() {
		t.Errorf("PHOR params=%+v, registered=%v", p, ok)
	}
	if stop, reason := s.DesiredStop("PHOR", statestore.Entry{EntryPrice: 100, EntryATR: 10}); stop != 95 || reason != "SL" {
		t.Errorf("PHOR stop=%v reason=%q, want 95/SL", stop, reason)
	}
	if p, ok := params("POSI"); !ok || p != posi.Params() {
		t.Errorf("POSI params=%+v, registered=%v", p, ok)
	}
	if p, ok := params("SBERP"); !ok || p != sberp.Params() {
		t.Errorf("SBERP params=%+v, registered=%v", p, ok)
	}
	if _, ok := s.Decider("UNKNOWN"); ok {
		t.Fatal("unexpected ticker")
	}
	if stop, reason := s.DesiredStop("ALRS", statestore.Entry{EntryPrice: 100, EntryATR: 10}); stop != 80 || reason != "SL" {
		t.Fatalf("ALRS stop=%v reason=%q, want 80/SL", stop, reason)
	}
	if stop, reason := s.DesiredStop("VTBR", statestore.Entry{EntryPrice: 100, EntryATR: 10}); stop != 90 || reason != "SL" {
		t.Fatalf("VTBR stop=%v reason=%q, want 90/SL", stop, reason)
	}
}

func TestX5LiveParameters(t *testing.T) {
	s := New(config.NewFVGImbalanceConfig(), nil)
	if _, ok := s.Decider("X5"); !ok {
		t.Fatal("X5 is not registered")
	}
	if p, ok := params("X5"); !ok || p != x5.Params() {
		t.Errorf("X5 params=%+v, registered=%v", p, ok)
	}
	if stop, reason := s.DesiredStop("X5", statestore.Entry{EntryPrice: 100, EntryATR: 10}); stop != 100-x5.Params().StopATR*10 || reason != "SL" {
		t.Errorf("X5 stop=%v reason=%q, want stop from ticker parameters", stop, reason)
	}
}
