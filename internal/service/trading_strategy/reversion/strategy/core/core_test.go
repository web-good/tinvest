package core

import (
	"math"
	"strings"
	"testing"

	"tinvest/internal/service/trading_strategy/scalping/model"
	"tinvest/internal/service/trading_strategy/scalping/strategy"
)

// defaultParams returns valid, entry-capable params for tests.
func defaultParams() Params {
	return Params{
		FastEMA: 50, SlowEMA: 200,
		RSIPeriod: 6, RSIOversold: 40, RSIOverbought: 70,
		EntryMode:   entryConfirmed,
		VolLookback: 20, VolMultiplier: 1.2,
		UseStoch: 0, StochPeriod: 14, StochSmooth: 3, StochOversold: 20,
		StopLossPct: 0.03, MaxHoldBars: 24, ATRPeriod: 14,
	}
}

// passingInput returns a flat decideInput that clears every entry gate: uptrend,
// confirmed RSI up-cross through 40, volume OK, stochastic well oversold.
func passingInput() decideInput {
	return decideInput{
		price:      100,
		atr:        1,
		emaFast:    95,
		emaSlow:    90,
		rsiPrev:    38, // below 40
		rsiNow:     45, // above 40 -> confirmed up-cross fires
		stochK:     10,
		stochValid: true,
		volumeOK:   true,
		barLow:     100,
	}
}

// openInput returns an input with an open position above its stop (no exit triggers).
func openInput() decideInput {
	in := passingInput()
	in.pos = &strategy.Position{PurchasePrice: 100, StopLoss: 97}
	in.rsiPrev, in.rsiNow = 45, 50 // not crossing overbought
	return in
}

func TestEntryAllGatesPass(t *testing.T) {
	s := NewWithParams("TEST", defaultParams())
	sig := s.decide(passingInput())
	if sig.Kind != model.SignalBuy {
		t.Fatalf("want Buy, got kind=%v", sig.Kind)
	}
	if math.Abs(sig.StopLoss-97) > 1e-9 { // 100*(1-0.03)
		t.Fatalf("StopLoss=%v want 97", sig.StopLoss)
	}
	if !strings.Contains(sig.EntryReason, "RSI(6)") {
		t.Fatalf("EntryReason missing RSI detail: %q", sig.EntryReason)
	}
}

func TestRegimeBlocks(t *testing.T) {
	s := NewWithParams("TEST", defaultParams())

	in := passingInput()
	in.emaFast = 85 // fast below slow -> not an uptrend
	if sig := s.decide(in); sig.Kind == model.SignalBuy {
		t.Fatalf("fast<slow: want no Buy")
	}

	in = passingInput()
	in.price = 89 // below slow EMA (90)
	if sig := s.decide(in); sig.Kind == model.SignalBuy {
		t.Fatalf("price<slowEMA: want no Buy")
	}
}

func TestDipConfirmedVsKnife(t *testing.T) {
	// confirmed: up-cross fires, down-cross does not.
	s := NewWithParams("TEST", defaultParams())
	if !s.dipFired(passingInput()) {
		t.Fatalf("confirmed: up-cross should fire")
	}
	down := passingInput()
	down.rsiPrev, down.rsiNow = 45, 38 // crossing down
	if s.dipFired(down) {
		t.Fatalf("confirmed: down-cross should NOT fire")
	}

	// knife: down-cross fires, up-cross does not.
	p := defaultParams()
	p.EntryMode = entryKnife
	sk := NewWithParams("TEST", p)
	if !sk.dipFired(down) {
		t.Fatalf("knife: down-cross should fire")
	}
	if sk.dipFired(passingInput()) {
		t.Fatalf("knife: up-cross should NOT fire")
	}
}

func TestVolumeBlocks(t *testing.T) {
	s := NewWithParams("TEST", defaultParams())
	in := passingInput()
	in.volumeOK = false
	if sig := s.decide(in); sig.Kind == model.SignalBuy {
		t.Fatalf("volume low: want no Buy")
	}
}

func TestStochGate(t *testing.T) {
	p := defaultParams()
	p.UseStoch = 1
	s := NewWithParams("TEST", p)

	in := passingInput()
	in.stochK = 50 // above oversold 20 -> blocked
	if sig := s.decide(in); sig.Kind == model.SignalBuy {
		t.Fatalf("stoch high: want no Buy")
	}

	in = passingInput()
	in.stochK = 10 // below 20 -> passes
	if sig := s.decide(in); sig.Kind != model.SignalBuy {
		t.Fatalf("stoch oversold: want Buy, got %v", sig.Kind)
	}
}

func TestStopSanityBlocksWhenNoStop(t *testing.T) {
	p := defaultParams()
	p.StopLossPct = 0
	s := NewWithParams("TEST", p)
	if sig := s.decide(passingInput()); sig.Kind == model.SignalBuy {
		t.Fatalf("StopLossPct=0: want no Buy (safety mandatory)")
	}
}

func TestExitSL(t *testing.T) {
	s := NewWithParams("TEST", defaultParams())
	in := openInput()
	in.barLow = 96 // <= 97
	sig := s.decide(in)
	if sig.Kind != model.SignalSell || sig.Reason != "SL" {
		t.Fatalf("want SL sell, got kind=%v reason=%q", sig.Kind, sig.Reason)
	}
}

func TestExitTime(t *testing.T) {
	s := NewWithParams("TEST", defaultParams())
	in := openInput()
	in.barsInPos = 24 // >= MaxHoldBars
	sig := s.decide(in)
	if sig.Kind != model.SignalSell || sig.Reason != "TIME" {
		t.Fatalf("want TIME sell, got kind=%v reason=%q", sig.Kind, sig.Reason)
	}
}

func TestExitRSI(t *testing.T) {
	s := NewWithParams("TEST", defaultParams())
	in := openInput()
	in.rsiPrev, in.rsiNow = 65, 72 // crosses up through 70
	sig := s.decide(in)
	if sig.Kind != model.SignalSell || sig.Reason != "RSI" {
		t.Fatalf("want RSI sell, got kind=%v reason=%q", sig.Kind, sig.Reason)
	}
}

func TestProtectiveStopWinsTie(t *testing.T) {
	s := NewWithParams("TEST", defaultParams())
	in := openInput()
	in.barLow = 96                 // SL hit
	in.rsiPrev, in.rsiNow = 65, 72 // RSI overbought too
	sig := s.decide(in)
	if sig.Reason != "SL" {
		t.Fatalf("protective first: want SL, got %q", sig.Reason)
	}
}

func TestStochGateBlocksDuringWarmup(t *testing.T) {
	p := defaultParams()
	p.UseStoch = 1
	s := NewWithParams("TEST", p)
	in := passingInput()
	in.stochK, in.stochValid = 0, false // insufficient history -> not a real oversold reading
	if sig := s.decide(in); sig.Kind == model.SignalBuy {
		t.Fatalf("stoch warm-up (invalid): want no Buy")
	}
}

func TestDecideTimeStopCounter(t *testing.T) {
	p := defaultParams()
	p.MaxHoldBars = 2
	s := NewWithParams("TEST", p)
	pos := &strategy.Position{PurchasePrice: 100, StopLoss: 90}
	md := strategy.MarketData{
		Price:    100,
		Highs:    []float64{100, 100, 100},
		Lows:     []float64{99, 99, 99}, // above SL 90, no SL exit
		Closes:   []float64{100, 100, 100},
		Volumes:  []int64{1, 1, 1},
		Position: pos,
	}
	if sig := s.Decide(md); sig.Reason == "TIME" { // barsInPosition=1
		t.Fatalf("bar1: unexpected TIME exit")
	}
	sig := s.Decide(md) // barsInPosition=2 >= 2
	if sig.Kind != model.SignalSell || sig.Reason != "TIME" {
		t.Fatalf("bar2: want TIME sell, got kind=%v reason=%q", sig.Kind, sig.Reason)
	}
}
