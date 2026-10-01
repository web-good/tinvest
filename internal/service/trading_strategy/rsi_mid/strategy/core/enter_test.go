package core

import (
	"math"
	"strings"
	"testing"

	"tinvest/internal/domain/ema"
	"tinvest/internal/service/trading_strategy/scalping/model"
	"tinvest/pkg/indicators"
)

// TestFixturesHaveTheirIntendedShape pins the preconditions every entry test relies on, so a
// failing entry test points at the strategy and not at a fixture that drifted.
func TestFixturesHaveTheirIntendedShape(t *testing.T) {
	for name, tc := range map[string]struct {
		closes    []float64
		wantAbove bool
	}{
		"upCross":   {upCross(), true},
		"downCross": {downCross(), false},
	} {
		n := len(tc.closes)
		rsi := indicators.RSISeries(tc.closes, 14)
		if !(rsi[n-2] <= midLine && rsi[n-1] > midLine) {
			t.Fatalf("%s: RSI %v -> %v, want a cross up through 50 on the last bar", name, rsi[n-2], rsi[n-1])
		}
		e := ema.Compute(tc.closes, 100)
		if above := tc.closes[n-1] > e[n-1]; above != tc.wantAbove {
			t.Fatalf("%s: close %v vs EMA(100) %v, want above=%v", name, tc.closes[n-1], e[n-1], tc.wantAbove)
		}
	}
}

func TestDefaultParams(t *testing.T) {
	want := Params{RSIPeriod: 14, RSIUpper: 70, BelowMidBars: 2, EMAPeriod: 100, DailyATRPeriod: 14, StopDailyATR: 0.5}
	if got := DefaultParams(); got != want {
		t.Fatalf("DefaultParams() = %+v, want %+v", got, want)
	}
}

func TestLookback(t *testing.T) {
	for _, tc := range []struct {
		rsi, ema, want int
	}{
		{14, 100, 220}, // default
		{21, 20, 120},  // floor
		{14, 200, 420},
		{5, 0, 120},
	} {
		p := DefaultParams()
		p.RSIPeriod, p.EMAPeriod = tc.rsi, tc.ema
		if got := NewWithParams("T", p).Lookback(); got != tc.want {
			t.Errorf("Lookback(RSI %d, EMA %d) = %d, want %d", tc.rsi, tc.ema, got, tc.want)
		}
	}
}

func TestCrossedUp(t *testing.T) {
	for _, tc := range []struct {
		name   string
		series []float64
		i, per int
		want   bool
	}{
		{"from below", []float64{0, 0, 49, 51}, 3, 1, true},
		{"from exactly 50", []float64{0, 0, 50, 51}, 3, 1, true},
		{"to exactly 50 is not above", []float64{0, 0, 49, 50}, 3, 1, false},
		{"already above", []float64{0, 0, 51, 52}, 3, 1, false},
		{"falling", []float64{0, 0, 51, 49}, 3, 1, false},
		{"previous bar in warm-up", []float64{0, 0, 49, 51}, 3, 2, true},
		{"previous bar before warm-up", []float64{0, 0, 49, 51}, 3, 3, false},
		{"index out of range", []float64{49, 51}, 2, 0, false},
		{"first bar", []float64{49, 51}, 0, 0, false},
	} {
		if got := crossedUp(tc.series, tc.i, tc.per, midLine); got != tc.want {
			t.Errorf("%s: crossedUp = %v, want %v", tc.name, got, tc.want)
		}
	}
}

func TestEnterBuysOnMidCrossAboveTrend(t *testing.T) {
	closes := upCross()
	sig := NewWithParams("T", DefaultParams()).Decide(fixture(closes, mondayNoon))
	if sig.Kind != model.SignalBuy {
		t.Fatalf("Kind = %v, want Buy", sig.Kind)
	}
	entry := closes[len(closes)-1]
	if math.Abs(sig.ATR-dailyWidth) > 1e-9 {
		t.Fatalf("ATR = %v, want %v: weekend dailies must not reach the ATR", sig.ATR, dailyWidth)
	}
	if want := entry - 0.5*dailyWidth; math.Abs(sig.StopLoss-want) > 1e-9 {
		t.Fatalf("StopLoss = %v, want %v", sig.StopLoss, want)
	}
	if sig.RSI <= midLine {
		t.Fatalf("RSI = %v, want > 50", sig.RSI)
	}
	if !strings.Contains(sig.EntryReason, "RSI(14)") || !strings.Contains(sig.EntryReason, "EMA(100)") {
		t.Fatalf("EntryReason = %q, want RSI(14) and EMA(100) mentioned", sig.EntryReason)
	}
}

func TestEnterRefuses(t *testing.T) {
	up := upCross()
	s := func(mut func(*Params)) *Strategy {
		p := DefaultParams()
		if mut != nil {
			mut(&p)
		}
		return NewWithParams("T", p)
	}

	t.Run("no cross: RSI already above 50 on the previous bar", func(t *testing.T) {
		closes := append(append([]float64(nil), up...), up[len(up)-1]*1.003)
		if sig := s(nil).Decide(fixture(closes, mondayNoon)); sig.Kind == model.SignalBuy {
			t.Fatal("bought without a fresh cross")
		}
	})
	t.Run("weekend bar", func(t *testing.T) {
		if sig := s(nil).Decide(fixture(up, saturdayNoon)); sig.Kind == model.SignalBuy {
			t.Fatal("bought on a Saturday")
		}
	})
	t.Run("close below EMA", func(t *testing.T) {
		if sig := s(nil).Decide(fixture(downCross(), mondayNoon)); sig.Kind == model.SignalBuy {
			t.Fatal("bought below the trend EMA")
		}
	})
	t.Run("EMA not warmed", func(t *testing.T) {
		if sig := s(func(p *Params) { p.EMAPeriod = len(up) + 1 }).Decide(fixture(up, mondayNoon)); sig.Kind == model.SignalBuy {
			t.Fatal("bought against an all-zero EMA")
		}
	})
	t.Run("no daily data", func(t *testing.T) {
		md := fixture(up, mondayNoon)
		md.DailyHighs, md.DailyLows, md.DailyCloses, md.DailyTimes = nil, nil, nil, nil
		if sig := s(nil).Decide(md); sig.Kind == model.SignalBuy {
			t.Fatal("bought without a daily ATR")
		}
	})
	t.Run("stop at or below zero", func(t *testing.T) {
		if sig := s(func(p *Params) { p.StopDailyATR = 1000 }).Decide(fixture(up, mondayNoon)); sig.Kind == model.SignalBuy {
			t.Fatal("bought with a stop below zero")
		}
	})
	for _, bad := range []struct {
		name string
		mut  func(*Params)
	}{
		{"RSIPeriod 0", func(p *Params) { p.RSIPeriod = 0 }},
		{"RSIPeriod negative", func(p *Params) { p.RSIPeriod = -3 }},
		{"EMAPeriod 0", func(p *Params) { p.EMAPeriod = 0 }},
		{"EMAPeriod negative", func(p *Params) { p.EMAPeriod = -5 }},
	} {
		t.Run(bad.name, func(t *testing.T) {
			if sig := s(bad.mut).Decide(fixture(up, mondayNoon)); sig.Kind == model.SignalBuy {
				t.Fatal("bought with a non-positive period")
			}
		})
	}
	t.Run("too few bars", func(t *testing.T) {
		if sig := s(nil).Decide(fixture(up[:1], mondayNoon)); sig.Kind == model.SignalBuy {
			t.Fatal("bought on one bar")
		}
	})
}
