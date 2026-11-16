package live

import (
	"context"
	"errors"
	"testing"
	"tinvest/internal/service/screener/density"
)

func TestDisabledTradingParticipantDoesNotTouchMarketOrBroker(t *testing.T) {
	p := &Participant{TradeEnabled: false, NewBroker: func([]density.Instrument) Broker { t.Fatal("disabled participant created broker"); return nil }}
	if err := p.Run(context.Background()); err != nil {
		t.Fatal(err)
	}
}

func TestSimulationParticipantNeverCreatesRealBroker(t *testing.T) {
	want := errors.New("paper run")
	p := &Participant{RunPaper: func(context.Context) error { return want }, NewBroker: func([]density.Instrument) Broker { t.Fatal("simulation created broker"); return nil }}
	if err := p.Run(context.Background()); !errors.Is(err, want) {
		t.Fatalf("simulation routing: %v", err)
	}
}
