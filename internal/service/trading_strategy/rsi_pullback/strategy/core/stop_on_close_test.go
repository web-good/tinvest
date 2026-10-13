package core

import (
	"math"
	"testing"
	"time"

	"tinvest/internal/service/trading_strategy/scalping/model"
	"tinvest/internal/service/trading_strategy/scalping/strategy"
)

// closeStopParams arms the close-based stop: entry 100 and EntryATR 10 put the protective level
// at 95 (StopDailyATR 0.5) and the catastrophe stop at 90 (2 × the protective distance).
func closeStopParams() Params {
	p := DefaultParams()
	p.StopDailyATR = 0.5
	p.StopOnClose = 1
	p.CatStopMult = 2
	return p
}

// closeStopBar builds a flat series whose LAST bar has the given low, high and close, with an
// open long entered at 100 on EntryATR 10.
func closeStopBar(low, high, closeP float64) strategy.MarketData {
	start := time.Date(2026, 3, 2, 7, 0, 0, 0, msk)
	md := barSeries([]float64{100, 100, 100, 100}, start)
	i := len(md.Closes) - 1
	md.Lows[i], md.Highs[i], md.Closes[i] = low, high, closeP
	md.Price = closeP
	md.Position = &strategy.Position{
		PurchasePrice: 100, Quantity: 1,
		EntryATR: 10, MaxFavorablePrice: 100, PrevMaxFavorablePrice: 100,
		EntryTime: md.Times[0],
	}
	return md
}

func TestStopOnCloseIgnoresAWickThroughTheLevel(t *testing.T) {
	s := NewWithParams("T", closeStopParams())
	// Тень пробила 95, но бар закрылся выше: при стопе по закрытию это не выход.
	if got := s.Decide(closeStopBar(94, 100.5, 96)); got.Kind != model.SignalNone {
		t.Fatalf("Kind/Reason = %v/%q, want no exit: only the close may trigger the stop", got.Kind, got.Reason)
	}
}

func TestStopOnCloseExitsAtTheClose(t *testing.T) {
	s := NewWithParams("T", closeStopParams())
	tests := []struct {
		name   string
		closeP float64
	}{
		{"close below the level", 94},
		{"close exactly on the level", 95},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := s.Decide(closeStopBar(93, 100.5, tc.closeP))
			if got.Kind != model.SignalSell || got.Reason != "SLC" {
				t.Fatalf("Kind/Reason = %v/%q, want Sell/SLC", got.Kind, got.Reason)
			}
			// Движок исполняет по цене стопа только стоповые причины; SLC обязан уйти по закрытию.
			if model.IsStopReason(got.Reason) {
				t.Fatal("SLC must not be a stop-style reason: it fills at the bar close, not at the level")
			}
		})
	}
}

func TestStopOnCloseCatastropheStopFiresIntrabar(t *testing.T) {
	s := NewWithParams("T", closeStopParams())
	tests := []struct {
		name string
		low  float64
	}{
		{"low pierces the catastrophe level", 89},
		{"low touches the catastrophe level exactly", 90},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := s.Decide(closeStopBar(tc.low, 100.5, 96))
			if got.Kind != model.SignalSell || got.Reason != "CATSL" {
				t.Fatalf("Kind/Reason = %v/%q, want Sell/CATSL", got.Kind, got.Reason)
			}
			if math.Abs(got.StopLoss-90) > 1e-9 {
				t.Fatalf("StopLoss = %v, want 90", got.StopLoss)
			}
			if !model.IsStopReason(got.Reason) {
				t.Fatal("CATSL must be a stop-style reason so the engine fills it at min(level, open)")
			}
		})
	}
}

func TestStopOnCloseWithoutCatastropheStopHoldsThroughAnyWick(t *testing.T) {
	p := closeStopParams()
	p.CatStopMult = 0
	s := NewWithParams("T", p)
	if got := s.Decide(closeStopBar(50, 100.5, 96)); got.Kind != model.SignalNone {
		t.Fatalf("Kind/Reason = %v/%q, want no exit with CatStopMult=0", got.Kind, got.Reason)
	}
}

func TestStopOnCloseTrailChecksTheClose(t *testing.T) {
	p := closeStopParams()
	p.UseTrail, p.TrailDailyATR = 1, 0.3
	s := NewWithParams("T", p)
	// Трейл от максимума 110: 110 − 0.3×10 = 107, теснее SL 95.
	md := closeStopBar(105, 108, 108)
	md.Position.MaxFavorablePrice, md.Position.PrevMaxFavorablePrice = 110, 110
	if got := s.Decide(md); got.Kind != model.SignalNone {
		t.Fatalf("close 108 above the trail 107: Kind/Reason = %v/%q, want no exit", got.Kind, got.Reason)
	}
	md.Closes[len(md.Closes)-1] = 106
	if got := s.Decide(md); got.Kind != model.SignalSell || got.Reason != "TRAILC" {
		t.Fatalf("close 106 below the trail 107: Kind/Reason = %v/%q, want Sell/TRAILC", got.Kind, got.Reason)
	}
}

// TestStopOnCloseTakeProfitBeatsTheCloseStop: the target is a resting order hit during the bar,
// the close-based stop is decided only when the bar is over — so a bar that tagged the target and
// then closed below the stop level exits at the target.
func TestStopOnCloseTakeProfitBeatsTheCloseStop(t *testing.T) {
	s := NewWithParams("T", closeStopParams())
	md := closeStopBar(93, 106, 94)
	md.Position.TakeProfit = 105
	if got := s.Decide(md); got.Reason != "TP" {
		t.Fatalf("Reason = %q, want TP: the target fills before the close is known", got.Reason)
	}
}

// TestStopOnCloseCatastropheBeatsTakeProfit: two resting orders touched in one bar — the
// intrabar order is unknowable, so the worse outcome wins, exactly as with the intrabar SL.
func TestStopOnCloseCatastropheBeatsTakeProfit(t *testing.T) {
	s := NewWithParams("T", closeStopParams())
	md := closeStopBar(89, 106, 100)
	md.Position.TakeProfit = 105
	if got := s.Decide(md); got.Reason != "CATSL" {
		t.Fatalf("Reason = %q, want CATSL to take precedence over TP", got.Reason)
	}
}

func TestCatastropheStopLevel(t *testing.T) {
	tests := []struct {
		name string
		p    Params
		atr  float64
		want float64
	}{
		{"armed", closeStopParams(), 10, 90},
		{"intrabar stop mode has no catastrophe stop", func() Params { p := closeStopParams(); p.StopOnClose = 0; return p }(), 10, 0},
		{"zero multiplier disables it", func() Params { p := closeStopParams(); p.CatStopMult = 0; return p }(), 10, 0},
		{"no fixed stop, nothing to scale", func() Params { p := closeStopParams(); p.StopDailyATR = 0; return p }(), 10, 0},
		{"no ATR", closeStopParams(), 0, 0},
		{"level at or below zero is no stop", closeStopParams(), 100, 0},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := CatastropheStop(tc.p, 100, tc.atr); math.Abs(got-tc.want) > 1e-9 {
				t.Fatalf("CatastropheStop = %v, want %v", got, tc.want)
			}
		})
	}
}
