package core

import (
	"strings"
	"testing"

	"tinvest/internal/service/trading_strategy/scalping/model"
	"tinvest/internal/service/trading_strategy/scalping/strategy"
)

// defaultParams returns valid, entry-capable params: trend on, RSI/Stoch oversold 20.
func defaultParams() Params {
	return Params{
		UseTrend: 1, FastEMA: 50, SlowEMA: 200,
		RSIPeriod: 14, RSIOversold: 20,
		StochKPeriod: 14, StochDSmooth: 3, StochOversold: 20,
	}
}

// passingInput clears every entry gate: uptrend; RSI crosses DOWN into oversold while
// Stoch %D is already in the oversold zone. EMA prev == now so no spurious cross.
func passingInput() decideInput {
	return decideInput{
		price: 100,
		emaFast: 95, emaFastPrev: 95, emaSlow: 90, emaSlowPrev: 90,
		rsiPrev: 25, rsiNow: 15, rsiOK: true, // crossDown through 20 (RSI enters oversold)
		stochPrev: 10, stochNow: 8, stochOK: true, // already < 20 (Stoch already in zone)
	}
}

// openInput is an open position holding, with neutral signals (no exit): RSI above 50
// and rising, fast EMA above slow on both bars.
func openInput() decideInput {
	in := passingInput()
	in.pos = &strategy.Position{PurchasePrice: 100}
	in.rsiPrev, in.rsiNow = 60, 62
	in.stochPrev, in.stochNow = 50, 55
	in.emaFast, in.emaFastPrev = 95, 95
	in.emaSlow, in.emaSlowPrev = 90, 90
	in.emaOK = true
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

func TestExitRSI50(t *testing.T) {
	s := NewWithParams("TEST", defaultParams())

	// RSI crosses 50 downward -> sell RSI50.
	in := openInput()
	in.rsiPrev, in.rsiNow = 55, 45
	if sig := s.decide(in); sig.Kind != model.SignalSell || sig.Reason != "RSI50" {
		t.Fatalf("RSI down-cross 50: want RSI50 sell, got kind=%v reason=%q", sig.Kind, sig.Reason)
	}

	// RSI stays above 50 -> no sell.
	in = openInput()
	in.rsiPrev, in.rsiNow = 55, 52
	if sig := s.decide(in); sig.Kind == model.SignalSell {
		t.Fatalf("RSI above 50: should NOT sell")
	}
}

func TestExitEMACross(t *testing.T) {
	s := NewWithParams("TEST", defaultParams())

	// Fast EMA drops below slow EMA, RSI neutral (no 50 cross) -> sell EMAX.
	in := openInput()
	in.rsiPrev, in.rsiNow = 60, 58 // no 50 cross
	in.emaFastPrev, in.emaSlowPrev = 95, 90
	in.emaFast, in.emaSlow = 88, 90 // fast now below slow
	if sig := s.decide(in); sig.Kind != model.SignalSell || sig.Reason != "EMAX" {
		t.Fatalf("bearish EMA cross: want EMAX sell, got kind=%v reason=%q", sig.Kind, sig.Reason)
	}

	// Fast stays above slow -> no EMA exit.
	in = openInput()
	in.rsiPrev, in.rsiNow = 60, 58
	in.emaFastPrev, in.emaSlowPrev = 95, 90
	in.emaFast, in.emaSlow = 94, 90
	if sig := s.decide(in); sig.Kind == model.SignalSell {
		t.Fatalf("fast above slow: should NOT sell")
	}
}

func TestExitPrecedenceRSIWhenBoth(t *testing.T) {
	s := NewWithParams("TEST", defaultParams())
	in := openInput()
	in.rsiPrev, in.rsiNow = 55, 45     // RSI50 fires
	in.emaFastPrev, in.emaSlowPrev = 95, 90
	in.emaFast, in.emaSlow = 88, 90 // EMAX also fires
	if sig := s.decide(in); sig.Reason != "RSI50" {
		t.Fatalf("both fire: want RSI50 precedence, got %q", sig.Reason)
	}
}

func TestNoExitWhenHolding(t *testing.T) {
	s := NewWithParams("TEST", defaultParams())
	if sig := s.decide(openInput()); sig.Kind == model.SignalSell {
		t.Fatalf("neutral signals: should hold, got sell %q", sig.Reason)
	}
}

func TestExitNoFalseEMACrossAtWarmup(t *testing.T) {
	p := defaultParams()
	p.UseTrend = 0
	s := NewWithParams("TEST", p)

	// Exactly SlowEMA(200) closes: the slow EMA is valid only on the last bar, so
	// emaSlowPrev is the warm-up zero sentinel. A monotone decline would make the
	// fast-minus-slow diff cross below zero purely from that sentinel; the emaOK guard
	// must suppress the spurious EMAX exit.
	n := 200
	closes := make([]float64, n)
	for i := 0; i < n; i++ {
		closes[i] = float64(300 - i) // strictly declining
	}
	md := strategy.MarketData{
		Price:    closes[n-1],
		Highs:    closes,
		Lows:     closes,
		Closes:   closes,
		Volumes:  make([]int64, n),
		Position: &strategy.Position{PurchasePrice: closes[n-1]},
	}
	if sig := s.Decide(md); sig.Kind == model.SignalSell && sig.Reason == "EMAX" {
		t.Fatalf("warm-up slow EMA sentinel produced a false EMAX exit")
	}
}

func TestExitRSIOversoldBreakdown(t *testing.T) {
	s := NewWithParams("T", Params{FastEMA: 50, SlowEMA: 200, RSIPeriod: 14, RSIOversold: 30, StochKPeriod: 14, StochDSmooth: 3, StochOversold: 20})
	in := openInput()
	in.rsiOK = true
	in.rsiPrev = 32 // above the oversold zone
	in.rsiNow = 28  // crossed down through 30
	sig := s.decide(in)
	if sig.Kind != model.SignalSell || sig.Reason != "RSIOS" {
		t.Fatalf("expected RSIOS sell, got kind=%v reason=%q", sig.Kind, sig.Reason)
	}
}

func TestNoRSIOSExitJustAfterEntry(t *testing.T) {
	// On/after the entry bar RSI is already below the oversold zone, so prev < level
	// and crossDown must not fire.
	s := NewWithParams("T", Params{FastEMA: 50, SlowEMA: 200, RSIPeriod: 14, RSIOversold: 30, StochKPeriod: 14, StochDSmooth: 3, StochOversold: 20})
	in := openInput()
	in.rsiOK = true
	in.rsiPrev = 25 // already inside the zone
	in.rsiNow = 22  // still falling, but no fresh down-cross of 30
	sig := s.decide(in)
	if sig.Kind == model.SignalSell && sig.Reason == "RSIOS" {
		t.Fatalf("RSIOS must not fire when prev already below oversold (prev=%.0f now=%.0f)", in.rsiPrev, in.rsiNow)
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
		Position: &strategy.Position{PurchasePrice: 100},
	}
	if out := s.Explain(md); !strings.Contains(out, "позиция уже открыта") {
		t.Fatalf("Explain with open position: %q", out)
	}
}
