package core

import (
	"math"
	"strings"
	"testing"

	"tinvest/internal/service/trading_strategy/scalping/model"
	"tinvest/internal/service/trading_strategy/scalping/strategy"
)

// defaultParams returns valid, entry-capable params: trend on, RSI/Stoch oversold 20,
// overbought 70/80, ATR stop = 1x ATR.
func defaultParams() Params {
	return Params{
		UseTrend: 1, FastEMA: 50, SlowEMA: 200,
		RSIPeriod: 14, RSIOversold: 20, RSIOverbought: 70,
		StochKPeriod: 14, StochDSmooth: 3, StochOversold: 20, StochOverbought: 80,
		ATRPeriod: 14, ATRMult: 1.0,
	}
}

// passingInput clears every entry gate: uptrend; RSI crosses DOWN into oversold while
// Stoch %D is already in the oversold zone; ATR positive.
func passingInput() decideInput {
	return decideInput{
		price: 100, atr: 2, emaFast: 95, emaSlow: 90,
		rsiPrev: 25, rsiNow: 15, rsiOK: true, // crossDown through 20 (RSI enters oversold)
		stochPrev: 10, stochNow: 8, stochOK: true, // already < 20 (Stoch already in zone)
		barLow: 100,
	}
}

// openInput is an open position above its stop with neutral oscillators (no exit).
func openInput() decideInput {
	in := passingInput()
	in.pos = &strategy.Position{PurchasePrice: 100, StopLoss: 98}
	in.rsiPrev, in.rsiNow = 50, 55
	in.stochPrev, in.stochNow = 50, 55
	return in
}

func TestEntryBlockedWhenOscillatorInvalid(t *testing.T) {
	s := NewWithParams("TEST", defaultParams())

	// Stoch warm-up: no valid %D reading. The sentinel 0 < oversold(20) would otherwise
	// masquerade as "already deep in zone" and degrade the dual gate to RSI-only.
	in := passingInput()
	in.stochOK = false
	in.stochNow, in.stochPrev = 0, 0
	if sig := s.decide(in); sig.Kind == model.SignalBuy {
		t.Fatalf("invalid stoch reading: want no Buy (dual confirmation must require a real reading)")
	}

	// Symmetric: RSI warm-up also blocks.
	in = passingInput()
	in.rsiOK = false
	in.rsiNow, in.rsiPrev = 0, 0
	if sig := s.decide(in); sig.Kind == model.SignalBuy {
		t.Fatalf("invalid rsi reading: want no Buy")
	}
}

func TestEntryAllGatesPass(t *testing.T) {
	s := NewWithParams("TEST", defaultParams())
	sig := s.decide(passingInput())
	if sig.Kind != model.SignalBuy {
		t.Fatalf("want Buy, got kind=%v", sig.Kind)
	}
	if math.Abs(sig.StopLoss-98) > 1e-9 { // 100 - 1.0*2
		t.Fatalf("StopLoss=%v want 98", sig.StopLoss)
	}
	if !strings.Contains(sig.EntryReason, "RSI(14)") || !strings.Contains(sig.EntryReason, "Stoch") {
		t.Fatalf("EntryReason missing dual detail: %q", sig.EntryReason)
	}
}

func TestEntryRequiresBothIndicators(t *testing.T) {
	s := NewWithParams("TEST", defaultParams())

	// RSI crosses in but Stoch NOT in zone -> no buy.
	in := passingInput()
	in.stochPrev, in.stochNow = 50, 55
	if sig := s.decide(in); sig.Kind == model.SignalBuy {
		t.Fatalf("RSI cross but Stoch out of zone: want no Buy")
	}

	// Stoch crosses in while RSI already in zone -> buy (the mirror branch).
	in = passingInput()
	in.rsiPrev, in.rsiNow = 15, 12     // already < 20, no fresh cross
	in.stochPrev, in.stochNow = 25, 15 // crossDown through 20
	if sig := s.decide(in); sig.Kind != model.SignalBuy {
		t.Fatalf("Stoch cross + RSI already in: want Buy, got %v", sig.Kind)
	}
}

func TestSimultaneousEntry(t *testing.T) {
	s := NewWithParams("TEST", defaultParams())
	in := passingInput()
	in.rsiPrev, in.rsiNow = 25, 15     // RSI crosses in
	in.stochPrev, in.stochNow = 25, 15 // Stoch crosses in same bar
	if sig := s.decide(in); sig.Kind != model.SignalBuy {
		t.Fatalf("both cross into zone same bar: want Buy, got %v", sig.Kind)
	}
}

func TestTrendFilterToggles(t *testing.T) {
	s := NewWithParams("TEST", defaultParams())
	in := passingInput()
	in.emaFast = 85 // fast < slow
	if sig := s.decide(in); sig.Kind == model.SignalBuy {
		t.Fatalf("UseTrend=1, fast<slow: want no Buy")
	}
	in = passingInput()
	in.price = 89 // below slow EMA (90)
	if sig := s.decide(in); sig.Kind == model.SignalBuy {
		t.Fatalf("UseTrend=1, price<slowEMA: want no Buy")
	}

	p := defaultParams()
	p.UseTrend = 0
	s0 := NewWithParams("TEST", p)
	in = passingInput()
	in.emaFast = 85
	if sig := s0.decide(in); sig.Kind != model.SignalBuy {
		t.Fatalf("UseTrend=0: want Buy regardless of trend, got %v", sig.Kind)
	}
}

func TestExitDualOverbought(t *testing.T) {
	s := NewWithParams("TEST", defaultParams())

	// RSI crosses UP through 70 while Stoch already > 80 -> sell.
	in := openInput()
	in.rsiPrev, in.rsiNow = 65, 75
	in.stochPrev, in.stochNow = 85, 85
	if sig := s.decide(in); sig.Kind != model.SignalSell || sig.Reason != "XOVER" {
		t.Fatalf("dual overbought: want XOVER sell, got kind=%v reason=%q", sig.Kind, sig.Reason)
	}

	// RSI crosses up but Stoch not high -> no sell.
	in = openInput()
	in.rsiPrev, in.rsiNow = 65, 75
	in.stochPrev, in.stochNow = 50, 55
	if sig := s.decide(in); sig.Kind == model.SignalSell {
		t.Fatalf("RSI high but Stoch low: should NOT sell")
	}

	// Stoch crosses up while RSI already high -> sell (mirror branch).
	in = openInput()
	in.rsiPrev, in.rsiNow = 75, 75     // already > 70, no cross
	in.stochPrev, in.stochNow = 75, 85 // crossUp through 80
	if sig := s.decide(in); sig.Kind != model.SignalSell || sig.Reason != "XOVER" {
		t.Fatalf("Stoch cross + RSI already high: want XOVER sell, got %v/%q", sig.Kind, sig.Reason)
	}
}

func TestStopSanityBlocksWhenNoStop(t *testing.T) {
	p := defaultParams()
	p.ATRMult = 0
	s := NewWithParams("TEST", p)
	if sig := s.decide(passingInput()); sig.Kind == model.SignalBuy {
		t.Fatalf("ATRMult=0: want no Buy (safety mandatory)")
	}
	s2 := NewWithParams("TEST", defaultParams())
	in := passingInput()
	in.atr = 0
	if sig := s2.decide(in); sig.Kind == model.SignalBuy {
		t.Fatalf("atr=0: want no Buy")
	}
}

func TestStopLevelUsesATRMult(t *testing.T) {
	p := defaultParams()
	p.ATRMult = 1.5
	s := NewWithParams("TEST", p)
	in := passingInput()
	in.atr = 2 // stop = 100 - 1.5*2 = 97
	sig := s.decide(in)
	if sig.Kind != model.SignalBuy {
		t.Fatalf("want Buy, got %v", sig.Kind)
	}
	if math.Abs(sig.StopLoss-97) > 1e-9 {
		t.Fatalf("StopLoss=%v want 97", sig.StopLoss)
	}
}

func TestExitSL(t *testing.T) {
	s := NewWithParams("TEST", defaultParams())
	in := openInput()
	in.barLow = 97 // <= stop 98
	sig := s.decide(in)
	if sig.Kind != model.SignalSell || sig.Reason != "SL" {
		t.Fatalf("want SL sell, got kind=%v reason=%q", sig.Kind, sig.Reason)
	}
}

func TestProtectiveStopWinsTie(t *testing.T) {
	s := NewWithParams("TEST", defaultParams())
	in := openInput()
	in.barLow = 97                 // SL hit (stop 98)
	in.rsiPrev, in.rsiNow = 65, 75 // overbought cross too
	in.stochPrev, in.stochNow = 85, 85
	sig := s.decide(in)
	if sig.Reason != "SL" {
		t.Fatalf("protective first: want SL, got %q", sig.Reason)
	}
}

func TestExplainBlocksOnTrend(t *testing.T) {
	s := NewWithParams("TEST", defaultParams())
	md := strategy.MarketData{
		Price: 1, Highs: []float64{1}, Lows: []float64{1},
		Closes: []float64{1}, Volumes: []int64{1}, // no EMA history -> emaSlow 0 -> not uptrend
	}
	out := s.Explain(md)
	if !strings.Contains(out, "Тренд") || !strings.Contains(out, "ВХОДА НЕТ") {
		t.Fatalf("Explain should block on trend: %q", out)
	}
}

func TestExplainTrendOffSkipsGate(t *testing.T) {
	p := defaultParams()
	p.UseTrend = 0
	s := NewWithParams("TEST", p)
	md := strategy.MarketData{
		Price: 1, Highs: []float64{1}, Lows: []float64{1},
		Closes: []float64{1}, Volumes: []int64{1},
	}
	if out := s.Explain(md); strings.Contains(out, "Тренд:") {
		t.Fatalf("UseTrend=0: Explain should not show trend gate: %q", out)
	}
}

func TestExplainPositionOpen(t *testing.T) {
	s := NewWithParams("TEST", defaultParams())
	md := strategy.MarketData{
		Price: 100, Highs: []float64{100}, Lows: []float64{100},
		Closes: []float64{100}, Volumes: []int64{1},
		Position: &strategy.Position{PurchasePrice: 100, StopLoss: 98},
	}
	if out := s.Explain(md); !strings.Contains(out, "позиция уже открыта") {
		t.Fatalf("Explain with open position: %q", out)
	}
}
