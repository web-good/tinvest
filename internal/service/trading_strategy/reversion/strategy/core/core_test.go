package core

import (
	"math"
	"strings"
	"testing"

	"tinvest/internal/service/trading_strategy/scalping/model"
	"tinvest/internal/service/trading_strategy/scalping/strategy"
)

// defaultParams returns valid, entry-capable params for tests: trend on,
// EntryMode "exit oversold zone" (up-cross through 40), ExitMode "enter
// overbought zone" (up-cross through 70), ATR stop = 1x ATR.
func defaultParams() Params {
	return Params{
		UseTrend: 1, FastEMA: 50, SlowEMA: 200,
		RSIPeriod: 6, RSIOversold: 40, RSIOverbought: 70,
		EntryMode: triggerExitZone, ExitMode: triggerEnterZone,
		ATRPeriod: 14, ATRMult: 1.0,
	}
}

// passingInput returns a flat decideInput that clears every entry gate: uptrend,
// RSI up-cross through 40 (exit of oversold zone), ATR positive.
func passingInput() decideInput {
	return decideInput{
		price:   100,
		atr:     2,
		emaFast: 95,
		emaSlow: 90,
		rsiPrev: 38, // below 40
		rsiNow:  45, // above 40 -> up-cross (exit oversold) fires
		barLow:  100,
	}
}

// openInput returns an input with an open position above its stop (no exit triggers).
func openInput() decideInput {
	in := passingInput()
	in.pos = &strategy.Position{PurchasePrice: 100, StopLoss: 98}
	in.rsiPrev, in.rsiNow = 45, 50 // not crossing overbought
	return in
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
	if !strings.Contains(sig.EntryReason, "RSI(6)") {
		t.Fatalf("EntryReason missing RSI detail: %q", sig.EntryReason)
	}
}

func TestTrendFilterToggles(t *testing.T) {
	// UseTrend=1: fast below slow -> blocked.
	s := NewWithParams("TEST", defaultParams())
	in := passingInput()
	in.emaFast = 85
	if sig := s.decide(in); sig.Kind == model.SignalBuy {
		t.Fatalf("UseTrend=1, fast<slow: want no Buy")
	}
	in = passingInput()
	in.price = 89 // below slow EMA (90)
	if sig := s.decide(in); sig.Kind == model.SignalBuy {
		t.Fatalf("UseTrend=1, price<slowEMA: want no Buy")
	}

	// UseTrend=0: same broken-trend input now passes (trend ignored).
	p := defaultParams()
	p.UseTrend = 0
	s0 := NewWithParams("TEST", p)
	in = passingInput()
	in.emaFast = 85
	if sig := s0.decide(in); sig.Kind != model.SignalBuy {
		t.Fatalf("UseTrend=0: want Buy regardless of trend, got %v", sig.Kind)
	}
}

func TestEntryModeEnterVsExitZone(t *testing.T) {
	up := passingInput()               // 38 -> 45: up-cross (exit oversold)
	down := passingInput()             // build a down-cross into oversold
	down.rsiPrev, down.rsiNow = 45, 38 // 45 -> 38: down-cross (enter oversold)

	// EntryMode = exit zone (default): up-cross fires, down-cross does not.
	s := NewWithParams("TEST", defaultParams())
	if !s.entryFired(up) {
		t.Fatalf("exit-zone: up-cross should fire")
	}
	if s.entryFired(down) {
		t.Fatalf("exit-zone: down-cross should NOT fire")
	}

	// EntryMode = enter zone: down-cross fires, up-cross does not.
	p := defaultParams()
	p.EntryMode = triggerEnterZone
	se := NewWithParams("TEST", p)
	if !se.entryFired(down) {
		t.Fatalf("enter-zone: down-cross should fire")
	}
	if se.entryFired(up) {
		t.Fatalf("enter-zone: up-cross should NOT fire")
	}
}

func TestExitModeEnterVsExitZone(t *testing.T) {
	// ExitMode = enter overbought zone (default): up-cross through 70 sells.
	s := NewWithParams("TEST", defaultParams())
	in := openInput()
	in.rsiPrev, in.rsiNow = 65, 72 // up-cross through 70
	if sig := s.decide(in); sig.Kind != model.SignalSell || sig.Reason != "RSI" {
		t.Fatalf("enter-zone exit: want RSI sell, got kind=%v reason=%q", sig.Kind, sig.Reason)
	}
	// Down-cross through 70 must NOT sell in enter-zone mode.
	in = openInput()
	in.rsiPrev, in.rsiNow = 72, 65
	if sig := s.decide(in); sig.Kind == model.SignalSell {
		t.Fatalf("enter-zone exit: down-cross should NOT sell")
	}

	// ExitMode = exit overbought zone: down-cross through 70 sells.
	p := defaultParams()
	p.ExitMode = triggerExitZone
	se := NewWithParams("TEST", p)
	in = openInput()
	in.rsiPrev, in.rsiNow = 72, 65 // down-cross through 70
	if sig := se.decide(in); sig.Kind != model.SignalSell || sig.Reason != "RSI" {
		t.Fatalf("exit-zone exit: want RSI sell, got kind=%v reason=%q", sig.Kind, sig.Reason)
	}
}

func TestStopSanityBlocksWhenNoStop(t *testing.T) {
	// ATRMult <= 0 -> no protective stop -> no entry.
	p := defaultParams()
	p.ATRMult = 0
	s := NewWithParams("TEST", p)
	if sig := s.decide(passingInput()); sig.Kind == model.SignalBuy {
		t.Fatalf("ATRMult=0: want no Buy (safety mandatory)")
	}
	// atr <= 0 -> cannot size a stop -> no entry.
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
	in.rsiPrev, in.rsiNow = 65, 72 // RSI overbought too
	sig := s.decide(in)
	if sig.Reason != "SL" {
		t.Fatalf("protective first: want SL, got %q", sig.Reason)
	}
}

func TestExplainBlocksOnTrend(t *testing.T) {
	s := NewWithParams("TEST", defaultParams())
	md := strategy.MarketData{
		Price:   1,
		Highs:   []float64{1},
		Lows:    []float64{1},
		Closes:  []float64{1}, // no EMA history -> emaSlow 0 -> not uptrend
		Volumes: []int64{1},
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
		Price:   1,
		Highs:   []float64{1},
		Lows:    []float64{1},
		Closes:  []float64{1},
		Volumes: []int64{1},
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
