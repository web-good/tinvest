package core

import (
	"fmt"
	"math"
	"testing"
	"time"

	"tinvest/internal/service/trading_strategy/scalping/model"
	"tinvest/internal/service/trading_strategy/scalping/strategy"
)

const step = 30 * time.Minute

type bar struct{ o, h, l, c float64 }

func snapshot(bars ...bar) strategy.MarketData {
	md := strategy.MarketData{}
	start := time.Date(2026, 11, 2, 10, 0, 0, 0, time.UTC)
	for i, b := range bars {
		md.Opens = append(md.Opens, b.o)
		md.Highs = append(md.Highs, b.h)
		md.Lows = append(md.Lows, b.l)
		md.Closes = append(md.Closes, b.c)
		md.Times = append(md.Times, start.Add(time.Duration(i)*step))
	}
	if len(bars) > 0 {
		md.Price = bars[len(bars)-1].c
	}
	return md
}
func formation() []bar   { return []bar{{99, 100, 98, 99}, {100, 103, 99, 102}, {103, 104, 102, 103}} }
func testParams() Params { p := DefaultParams(); p.ATRPeriod = 1; return p }

// These cases catch loose wick/body comparisons and qualification using a later ATR.
func TestFVGFormation(t *testing.T) {
	tests := []struct {
		name   string
		change func([]bar)
		gap    float64
		want   int
	}{
		{"green body", func([]bar) {}, .25, 1},
		{"red middle", func(b []bar) { b[1].o = 102; b[1].c = 100 }, .25, 0},
		{"touching outer wicks", func(b []bar) { b[2].l = 100 }, .25, 0},
		{"body starts below zone", func(b []bar) { b[1].o = 99.5 }, .25, 0},
		{"body ends above zone", func(b []bar) { b[1].c = 102.5 }, .25, 0},
		{"gap below formation ATR threshold", func([]bar) {}, 1.01, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := formation()
			tt.change(b)
			p := testParams()
			p.MinGapATR = tt.gap
			zones := NewWithParams("TEST", step, p).discover(snapshot(b...))
			if len(zones) != tt.want {
				t.Fatalf("zones=%+v, want %d", zones, tt.want)
			}
			if tt.want > 0 && (zones[0].lower != 100 || zones[0].upper != 102 || zones[0].formed != 2 || zones[0].atr != 2) {
				t.Fatalf("zone=%+v", zones[0])
			}
		})
	}
}

func TestValidateParams(t *testing.T) {
	if err := ValidateParams(DefaultParams()); err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		name   string
		change func(*Params)
	}{
		{"zero ATR", func(p *Params) { p.ATRPeriod = 0 }}, {"negative ATR", func(p *Params) { p.ATRPeriod = -1 }},
		{"zero retest bars", func(p *Params) { p.MaxRetestBars = 0 }}, {"negative retest bars", func(p *Params) { p.MaxRetestBars = -1 }},
		{"negative gap", func(p *Params) { p.MinGapATR = -1 }}, {"negative target", func(p *Params) { p.TakeProfitR = -1 }},
		{"negative confirmation", func(p *Params) { p.RequireCloseAboveZone = -1 }}, {"large confirmation", func(p *Params) { p.RequireCloseAboveZone = 2 }},
		{"NaN gap", func(p *Params) { p.MinGapATR = math.NaN() }}, {"infinite target", func(p *Params) { p.TakeProfitR = math.Inf(1) }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := DefaultParams()
			tt.change(&p)
			if ValidateParams(p) == nil {
				t.Fatal("accepted invalid parameters")
			}
		})
	}
	p := DefaultParams()
	p.MinGapATR = 0
	p.TakeProfitR = 0
	p.RequireCloseAboveZone = 0
	if err := ValidateParams(p); err != nil {
		t.Fatal(err)
	}
}

// Compile-time assertion also pins the consumer interface.
var _ strategy.Strategy = (*Strategy)(nil)

func TestRetest(t *testing.T) {
	tests := []struct {
		name    string
		later   []bar
		confirm int
		want    model.SignalKind
	}{
		{"protected retest", []bar{{103, 104, 101, 103}}, 1, model.SignalBuy},
		{"formation cannot retest itself", nil, 1, model.SignalNone},
		{"inside close with confirmation", []bar{{103, 104, 101, 101.5}}, 1, model.SignalNone},
		{"close at upper with confirmation", []bar{{103, 104, 101, 102}}, 1, model.SignalNone},
		{"inside close without confirmation", []bar{{103, 104, 101, 101.5}}, 0, model.SignalBuy},
		{"lower touch invalidates", []bar{{103, 104, 100, 103}, {103, 104, 101, 103}}, 1, model.SignalNone},
		{"lower breach invalidates", []bar{{103, 104, 99, 103}, {103, 104, 101, 103}}, 1, model.SignalNone},
		{"no touch", []bar{{103, 104, 102.1, 103}}, 1, model.SignalNone},
		{"touch upper", []bar{{103, 104, 102, 103}}, 1, model.SignalBuy},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := testParams()
			p.RequireCloseAboveZone = tt.confirm
			md := snapshot(append(formation(), tt.later...)...)
			sig := NewWithParams("TEST", step, p).Decide(md)
			if sig.Kind != tt.want {
				t.Fatalf("signal=%+v, want kind %v", sig, tt.want)
			}
			if sig.Kind == model.SignalBuy {
				if sig.StopLoss != 100 || !sig.FillBy.Equal(md.Times[len(md.Times)-1].Add(step)) || sig.TakeProfit != md.Price+1.5*(md.Price-100) {
					t.Fatalf("wrong entry levels: %+v", sig)
				}
			}
		})
	}
}

func TestRetestExpiry(t *testing.T) {
	for _, age := range []int{8, 9} {
		t.Run(fmt.Sprint(age), func(t *testing.T) {
			b := formation()
			for i := 1; i < age; i++ {
				b = append(b, bar{103, 104, 102.5, 103})
			}
			b = append(b, bar{103, 104, 101, 103})
			sig := NewWithParams("TEST", step, testParams()).Decide(snapshot(b...))
			want := model.SignalNone
			if age == 8 {
				want = model.SignalBuy
			}
			if sig.Kind != want {
				t.Fatalf("age %d: %+v", age, sig)
			}
		})
	}
}
func TestNewestZone(t *testing.T) {
	// Second zone [101,102] lies inside the first [100,102].
	b := append(formation(), bar{101, 101, 100.5, 101}, bar{101, 103, 100.5, 102}, bar{103, 104, 102, 103}, bar{103, 104, 101.5, 103})
	sig := NewWithParams("TEST", step, testParams()).Decide(snapshot(b...))
	if sig.Kind != model.SignalBuy || sig.StopLoss != 101 {
		t.Fatalf("want newest stop 101: %+v", sig)
	}
}
func TestRetestFrozenATRAndDisabledTarget(t *testing.T) {
	p := testParams()
	p.MinGapATR = .9
	p.TakeProfitR = 0
	b := append(formation(), bar{103, 200, 102.5, 103}, bar{103, 104, 101, 103})
	sig := NewWithParams("TEST", step, p).Decide(snapshot(b...))
	if sig.Kind != model.SignalBuy || sig.ATR != 2 || sig.TakeProfit != 0 {
		t.Fatalf("formation ATR and disabled target: %+v", sig)
	}
}
func TestRetestUnsupportedStepAndMalformedData(t *testing.T) {
	md := snapshot(append(formation(), bar{103, 104, 101, 103})...)
	if sig := NewWithParams("TEST", 15*time.Minute, testParams()).Decide(md); sig.Kind != model.SignalNone {
		t.Fatal(sig)
	}
	md.Opens = nil
	if sig := NewWithParams("TEST", step, testParams()).Decide(md); sig.Kind != model.SignalNone {
		t.Fatal(sig)
	}
}
func TestPosition(t *testing.T) {
	tests := []struct {
		name               string
		fill, low, high, r float64
		reason             string
		target             float64
	}{
		{"stop wins", 104, 99, 120, 1.5, "SL", 0},
		{"fill below stop remains stopped", 99, 101, 110, 1.5, "SL", 0},
		{"fill at stop remains stopped", 100, 101, 110, 1.5, "SL", 0},
		{"actual fill target", 104, 101, 110, 1.5, "TP", 110},
		{"signal target ignored", 104, 101, 108, 1.5, "", 0},
		{"disabled target", 104, 101, 120, 0, "", 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := testParams()
			p.TakeProfitR = tt.r
			md := snapshot(bar{104, tt.high, tt.low, 104})
			md.Position = &strategy.Position{PurchasePrice: tt.fill, StopLoss: 100, TakeProfit: 107.5}
			sig := NewWithParams("TEST", step, p).Decide(md)
			if sig.Reason != tt.reason || sig.TakeProfit != tt.target {
				t.Fatalf("signal=%+v, want %s target %v", sig, tt.reason, tt.target)
			}
			if tt.reason != "" && sig.Kind != model.SignalSell {
				t.Fatal(sig)
			}
		})
	}
}
