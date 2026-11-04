package core

import (
	"fmt"
	"math"
	"strings"
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
		{"body starts below zone", func(b []bar) { b[1].o = 99.5 }, .25, 1},
		{"body ends above zone", func(b []bar) { b[1].c = 102.5 }, .25, 1},
		{"body starts above lower boundary", func(b []bar) { b[1].o = 100.5 }, .25, 0},
		{"body ends below upper boundary", func(b []bar) { b[1].c = 101.5 }, .25, 0},
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

func TestFVGFormationRealGAZP20261015(t *testing.T) {
	tests := []struct {
		name  string
		hour  int
		bars  []bar
		want  int
		lower float64
		upper float64
	}{
		{
			name: "07:00 contained body",
			hour: 6,
			bars: []bar{
				{104.75, 104.75, 104.75, 104.75},
				{104.76, 105.00, 104.55, 104.88},
				{104.88, 105.35, 104.88, 105.22},
			},
			want: 0,
		},
		{
			name: "15:00 bullish impulse",
			hour: 14,
			bars: []bar{
				{104.33, 104.61, 104.29, 104.56},
				{104.56, 105.80, 104.53, 105.47},
				{105.49, 106.42, 105.35, 106.00},
			},
			want: 1, lower: 104.61, upper: 105.35,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			md := snapshot(tt.bars...)
			start := time.Date(2026, 10, 15, tt.hour, 30, 0, 0, time.FixedZone("MSK", 3*60*60))
			for i := range md.Times {
				md.Times[i] = start.Add(time.Duration(i) * step)
			}
			p := testParams()
			zones := NewWithParams("GAZP", step, p).discover(md)
			if len(zones) != tt.want {
				t.Fatalf("zones=%+v, want %d", zones, tt.want)
			}
			if tt.want == 1 && (zones[0].lower != tt.lower || zones[0].upper != tt.upper || zones[0].formed != 2) {
				t.Fatalf("zone=%+v, want [%v, %v] formed at c", zones[0], tt.lower, tt.upper)
			}
		})
	}
}

func TestDefaultParams(t *testing.T) {
	p := DefaultParams()
	if p.RequireFirstRetest != 0 || p.RequireBullishBOS != 0 || p.RequireLiquiditySweep != 0 || p.SwingK != 2 || p.StructureLookbackBars != 48 || p.MaxSweepToFVGBars != 8 || p.HTFTrendEMA != 0 || p.MaxEntryDistanceATR != 0 || p.MaxHoldBars != 0 || p.StopBufferATR != 0 {
		t.Fatalf("unexpected defaults: %+v", p)
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
		{"first retest negative", func(p *Params) { p.RequireFirstRetest = -1 }}, {"first retest large", func(p *Params) { p.RequireFirstRetest = 2 }},
		{"BOS negative", func(p *Params) { p.RequireBullishBOS = -1 }}, {"BOS large", func(p *Params) { p.RequireBullishBOS = 2 }},
		{"sweep negative", func(p *Params) { p.RequireLiquiditySweep = -1 }}, {"sweep large", func(p *Params) { p.RequireLiquiditySweep = 2 }},
		{"zero swing", func(p *Params) { p.SwingK = 0 }}, {"negative swing", func(p *Params) { p.SwingK = -1 }},
		{"zero structure lookback", func(p *Params) { p.StructureLookbackBars = 0 }}, {"negative structure lookback", func(p *Params) { p.StructureLookbackBars = -1 }},
		{"zero sweep age", func(p *Params) { p.MaxSweepToFVGBars = 0 }}, {"negative sweep age", func(p *Params) { p.MaxSweepToFVGBars = -1 }},
		{"negative EMA", func(p *Params) { p.HTFTrendEMA = -1 }}, {"negative hold", func(p *Params) { p.MaxHoldBars = -1 }},
		{"negative distance", func(p *Params) { p.MaxEntryDistanceATR = -1 }}, {"NaN distance", func(p *Params) { p.MaxEntryDistanceATR = math.NaN() }}, {"infinite distance", func(p *Params) { p.MaxEntryDistanceATR = math.Inf(1) }},
		{"negative buffer", func(p *Params) { p.StopBufferATR = -1 }}, {"NaN buffer", func(p *Params) { p.StopBufferATR = math.NaN() }}, {"infinite buffer", func(p *Params) { p.StopBufferATR = math.Inf(-1) }},
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

func TestWeekendFill(t *testing.T) {
	tests := []struct {
		name string
		day  int
		want model.SignalKind
	}{
		{"Saturday fill", 6, model.SignalNone},
		{"Sunday fill", 7, model.SignalNone},
		{"Monday fill", 8, model.SignalBuy},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			md := snapshot(append(formation(), bar{103, 104, 101, 103})...)
			// The retest closes at midnight; its next M30 open is on the following day.
			last := time.Date(2026, 11, tt.day, 23, 30, 0, 0, time.FixedZone("MSK", 3*60*60))
			for i := range md.Times {
				md.Times[i] = last.Add(time.Duration(i-len(md.Times)+1) * step)
			}
			sig := NewWithParams("TEST", step, testParams()).Decide(md)
			if sig.Kind != tt.want {
				t.Fatalf("signal=%+v, want kind %v", sig, tt.want)
			}
			if tt.want == model.SignalBuy {
				wantFill := time.Date(2026, 11, 9, 0, 0, 0, 0, last.Location())
				if !sig.FillBy.Equal(wantFill) {
					t.Fatalf("FillBy=%v, want %v", sig.FillBy, wantFill)
				}
			}
		})
	}
}

func TestPositionWeekendStop(t *testing.T) {
	md := snapshot(bar{104, 120, 99, 104})
	md.Times[0] = time.Date(2026, 11, 7, 12, 0, 0, 0, time.UTC)
	md.Position = &strategy.Position{PurchasePrice: 104, StopLoss: 100}
	sig := NewWithParams("TEST", step, testParams()).Decide(md)
	if sig.Kind != model.SignalSell || sig.Reason != "SL" || sig.StopLoss != 100 {
		t.Fatalf("weekend stop: %+v", sig)
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

// A zone must retain the same formation ATR and qualification when the engine
// drops the oldest candle. Period 2 exercises Wilder smoothing, unlike period 1.
func TestRetestFormationATRStableAcrossSlidingWindows(t *testing.T) {
	bars := make([]bar, 8)
	for i := range bars {
		bars[i] = bar{99, 100, 98, 99}
	}
	bars[1].h = 1124
	bars = append(bars, formation()...)
	bars = append(bars, bar{103, 104, 101, 103}, bar{103, 104, 101, 103}, bar{103, 104, 101, 103})
	full := snapshot(bars...)
	for _, tc := range []struct {
		name   string
		minGap float64
		want   model.SignalKind
	}{
		{"qualified stays qualified", .5, model.SignalBuy},
		{"rejected stays rejected", .9, model.SignalNone},
	} {
		t.Run(tc.name, func(t *testing.T) {
			p := testParams()
			p.ATRPeriod = 2
			p.MinGapATR = tc.minGap
			// Same zone formed at absolute index 10; the windows end on later retests.
			for offset := 0; offset < 2; offset++ {
				md := snapshot(bars[offset : offset+13]...)
				md.Times = full.Times[offset : offset+13]
				sig := NewWithParams("TEST", step, p).Decide(md)
				if sig.Kind != tc.want {
					t.Fatalf("window [%d:%d]: kind %v, want %v; signal %+v", offset, offset+13, sig.Kind, tc.want, sig)
				}
				// The fixed four-bar formation history has TRs 2,4,2: seed 3,
				// then Wilder-smoothed (3+2)/2 = 2.5, independent of later bars.
				if tc.want == model.SignalBuy && sig.ATR != 2.5 {
					t.Fatalf("window [%d:%d]: formation ATR %v, want 2.5", offset, offset+13, sig.ATR)
				}
			}
		})
	}
}

func TestRetestFormationATRRetainedThroughExpiry(t *testing.T) {
	p := testParams()
	p.ATRPeriod = 2
	p.MinGapATR = .5
	s := NewWithParams("TEST", step, p)
	bars := make([]bar, 8)
	for i := range bars {
		bars[i] = bar{99, 100, 98, 99}
	}
	bars[1].h = 1124
	bars = append(bars, formation()...)
	for age := 1; age <= p.MaxRetestBars; age++ {
		bars = append(bars, bar{103, 104, 101, 103})
		// This is the same fixed-size truncation performed by backtest.Run.
		md := snapshot(bars[len(bars)-s.Lookback():]...)
		sig := s.Decide(md)
		if sig.Kind != model.SignalBuy || sig.ATR != 2.5 {
			t.Fatalf("age %d, lookback %d: want buy with formation ATR 2.5, got %+v", age, s.Lookback(), sig)
		}
	}
}

func TestSwing(t *testing.T) {
	for _, tc := range []struct {
		name       string
		values     []float64
		i, k       int
		high, want bool
	}{
		{"high", []float64{1, 3, 5, 4, 2}, 2, 2, true, true},
		{"low", []float64{5, 3, 1, 2, 4}, 2, 2, false, true},
		{"equal high", []float64{1, 5, 5, 4, 2}, 2, 2, true, false},
		{"equal low", []float64{5, 1, 1, 2, 4}, 2, 2, false, false},
		{"missing left", []float64{5, 4, 3}, 0, 1, true, false},
		{"missing right confirmation", []float64{1, 5}, 1, 1, true, false},
		{"invalid k", []float64{1, 5, 1}, 1, 0, true, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var got bool
			if tc.high {
				got = swingHighAt(tc.values, tc.i, tc.k)
			} else {
				got = swingLowAt(tc.values, tc.i, tc.k)
			}
			if got != tc.want {
				t.Fatalf("got %v want %v", got, tc.want)
			}
		})
	}
}

func TestLookback(t *testing.T) {
	p := DefaultParams()
	p.ATRPeriod = 2
	p.MaxRetestBars = 8
	p.StructureLookbackBars = 10
	p.SwingK = 2
	s := NewWithParams("TEST", step, p)
	if got := s.Lookback(); got != 12 {
		t.Fatalf("legacy %d", got)
	}
	p.RequireBullishBOS = 1
	s = NewWithParams("TEST", step, p)
	if got := s.Lookback(); got != 25 {
		t.Fatalf("BOS %d", got)
	}
	p.RequireBullishBOS = 0
	p.RequireLiquiditySweep = 1
	s = NewWithParams("TEST", step, p)
	if got := s.Lookback(); got != 25 {
		t.Fatalf("sweep %d", got)
	}
}

func TestStructure(t *testing.T) {
	base := []bar{{10, 10, 10, 10}, {10, 11, 9, 10}, {10, 12, 8, 10}, {10, 11, 9, 10}, {10, 10, 10, 10}, {10, 11, 7, 10}, {10, 14, 10, 13}, {10, 11, 10, 10}, {10, 11, 10, 10}}
	for _, tc := range []struct {
		name       string
		sweep, bos int
		change     func([]bar, *Params)
		want       bool
	}{
		{"both ordered", 1, 1, func([]bar, *Params) {}, true},
		{"standalone sweep", 1, 0, func(b []bar, _ *Params) { b[6].c = 10 }, true},
		{"standalone BOS", 0, 1, func(b []bar, _ *Params) { b[5].l = 9; b[7].c = 15 }, true},
		{"wick without reclaim", 1, 0, func(b []bar, _ *Params) { b[5].c = 7 }, false},
		{"reclaim equality", 1, 0, func(b []bar, _ *Params) { b[5].c = 8 }, false},
		{"sweep equality", 1, 0, func(b []bar, _ *Params) { b[5].l = 8 }, false},
		{"expired sweep", 1, 0, func(_ []bar, p *Params) { p.MaxSweepToFVGBars = 2 }, false},
		{"age boundary", 1, 0, func(_ []bar, p *Params) { p.MaxSweepToFVGBars = 3 }, true},
		{"wick only BOS", 1, 1, func(b []bar, _ *Params) { b[6].c = 12 }, false},
		{"BOS before sweep", 1, 1, func(b []bar, _ *Params) { b[4].c = 13; b[6].c = 10 }, false},
		{"sweep and BOS same bar", 1, 1, func(b []bar, _ *Params) { b[5].h = 14; b[5].c = 13; b[6].c = 10 }, true},
		{"late confirmation unavailable", 1, 0, func(b []bar, _ *Params) { b[2].l = 9; b[4].l = 8; b[5].l = 7 }, false},
		{"standalone BOS on b", 0, 1, func(b []bar, _ *Params) { b[6].c = 10; b[7].c = 13 }, true},
		{"standalone BOS before b", 0, 1, func(b []bar, _ *Params) { b[6].c = 13; b[7].c = 10 }, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			b := append([]bar(nil), base...)
			p := testParams()
			p.SwingK = 1
			p.RequireLiquiditySweep = tc.sweep
			p.RequireBullishBOS = tc.bos
			tc.change(b, &p)
			got := NewWithParams("TEST", step, p).structurePasses(snapshot(b...), zone{formed: 8})
			if got != tc.want {
				t.Fatalf("got %v want %v", got, tc.want)
			}
		})
	}
}

func TestFirstRetest(t *testing.T) {
	for _, tc := range []struct {
		name    string
		later   []bar
		enabled int
		want    model.SignalKind
	}{
		{"first touch", []bar{{103, 104, 101, 103}}, 1, model.SignalBuy},
		{"second protected touch", []bar{{103, 104, 101, 103}, {103, 104, 101, 103}}, 1, model.SignalNone},
		{"first touch fails close confirmation", []bar{{103, 104, 101, 101.5}, {103, 104, 101, 103}}, 1, model.SignalNone},
		{"disabled allows second touch", []bar{{103, 104, 101, 103}, {103, 104, 101, 103}}, 0, model.SignalBuy},
	} {
		t.Run(tc.name, func(t *testing.T) {
			p := testParams()
			p.RequireFirstRetest = tc.enabled
			sig := NewWithParams("TEST", step, p).Decide(snapshot(append(formation(), tc.later...)...))
			if sig.Kind != tc.want {
				t.Fatalf("signal=%+v, want %v", sig, tc.want)
			}
		})
	}
	t.Run("weekend or missing fill slot consumes first touch", func(t *testing.T) {
		p := testParams()
		p.RequireFirstRetest = 1
		md := snapshot(append(formation(), bar{103, 104, 101, 103}, bar{103, 104, 101, 103})...)
		md.Times[3] = time.Date(2026, 11, 6, 23, 30, 0, 0, time.UTC)
		md.Times[4] = time.Date(2026, 11, 9, 10, 0, 0, 0, time.UTC)
		s := NewWithParams("TEST", step, p)
		first := md
		first.Closes = md.Closes[:4]
		first.Opens = md.Opens[:4]
		first.Highs = md.Highs[:4]
		first.Lows = md.Lows[:4]
		first.Times = md.Times[:4]
		if sig := s.Decide(first); sig.Kind != model.SignalNone {
			t.Fatalf("weekend signal=%+v", sig)
		}
		if sig := s.Decide(md); sig.Kind != model.SignalNone {
			t.Fatalf("later signal=%+v", sig)
		}
	})
	t.Run("missing next slot consumes first touch", func(t *testing.T) {
		p := testParams()
		p.RequireFirstRetest = 1
		md := snapshot(append(formation(), bar{103, 104, 101, 103}, bar{103, 104, 101, 103})...)
		md.Times[4] = md.Times[4].Add(step)
		if sig := NewWithParams("TEST", step, p).Decide(md); sig.Kind != model.SignalNone {
			t.Fatalf("later signal=%+v", sig)
		}
	})
	t.Run("first touch failing H4 still consumes", func(t *testing.T) {
		p := testParams()
		p.RequireFirstRetest, p.HTFTrendEMA = 1, 3
		md := snapshot(append(formation(), bar{103, 104, 101, 103}, bar{103, 104, 101, 103})...)
		first := md
		first.Closes, first.Opens, first.Highs, first.Lows, first.Times = md.Closes[:4], md.Opens[:4], md.Highs[:4], md.Lows[:4], md.Times[:4]
		first.HTFCloses = []float64{3, 2, 1}
		md.HTFCloses = []float64{1, 2, 3}
		s := NewWithParams("TEST", step, p)
		if sig := s.Decide(first); sig.Kind != model.SignalNone {
			t.Fatalf("first signal=%+v", sig)
		}
		if sig := s.Decide(md); sig.Kind != model.SignalNone {
			t.Fatalf("later signal=%+v", sig)
		}
	})
	t.Run("consumed newer zone does not hide older", func(t *testing.T) {
		// Synthetic OHLC allows overlapping reconstructed zones while keeping the older untouched.
		b := append(formation(), bar{101, 101, 102.5, 101}, bar{101, 104, 102.5, 103}, bar{104, 105, 103, 104}, bar{104, 105, 102.5, 104}, bar{103, 105, 101.5, 104})
		p := testParams()
		p.RequireFirstRetest = 1
		s := NewWithParams("TEST", step, p)
		md := snapshot(b...)
		zones := s.discover(md)
		if len(zones) < 2 || zones[len(zones)-1].lower != 101 {
			t.Fatalf("expected newer zone: %+v", zones)
		}
		legacy := p
		legacy.RequireFirstRetest = 0
		if sig := NewWithParams("TEST", step, legacy).Decide(md); sig.Kind != model.SignalBuy || sig.StopLoss != 101 {
			t.Fatalf("newer zone should otherwise qualify: %+v", sig)
		}
		sig := s.Decide(md)
		if sig.Kind != model.SignalBuy || sig.StopLoss != 100 {
			t.Fatalf("older zone signal=%+v", sig)
		}
	})
}

func TestEntryStructureGates(t *testing.T) {
	base := []bar{{10, 10, 10, 10}, {10, 11, 9, 10}, {10, 12, 8, 10}, {10, 11, 9, 10}, {10, 10, 10, 10}, {10, 11, 7, 10}, {10, 14, 10, 13}, {14, 17, 14, 16}, {16, 17, 16, 16.5}, {16, 17, 15, 16.5}}
	for _, tc := range []struct {
		name       string
		sweep, bos int
		change     func([]bar)
		want       model.SignalKind
	}{
		{"disabled", 0, 0, func([]bar) {}, model.SignalBuy},
		{"sweep passes", 1, 0, func([]bar) {}, model.SignalBuy},
		{"sweep fails", 1, 0, func(b []bar) { b[5].l = 8 }, model.SignalNone},
		{"BOS passes without sweep", 0, 1, func(b []bar) { b[5].l = 9 }, model.SignalBuy},
		{"BOS fails", 0, 1, func(b []bar) { b[2].h = 18 }, model.SignalNone},
		{"combined passes", 1, 1, func([]bar) {}, model.SignalBuy},
		{"combined fails sweep", 1, 1, func(b []bar) { b[5].l = 8 }, model.SignalNone},
		{"combined fails BOS", 1, 1, func(b []bar) { b[2].h = 18 }, model.SignalNone},
	} {
		t.Run(tc.name, func(t *testing.T) {
			b := append([]bar(nil), base...)
			tc.change(b)
			p := testParams()
			p.SwingK = 1
			p.RequireLiquiditySweep = tc.sweep
			p.RequireBullishBOS = tc.bos
			sig := NewWithParams("TEST", step, p).Decide(snapshot(b...))
			if sig.Kind != tc.want {
				t.Fatalf("signal=%+v, want %v", sig, tc.want)
			}
		})
	}
}

func TestHTFTrend(t *testing.T) {
	for _, tc := range []struct {
		name   string
		period int
		closes []float64
		want   model.SignalKind
	}{
		{"disabled nil", 0, nil, model.SignalBuy},
		{"disabled nonfinite", 0, []float64{math.NaN()}, model.SignalBuy},
		{"enabled nil", 3, nil, model.SignalNone},
		{"enabled short", 3, []float64{1, 2}, model.SignalNone},
		{"enabled NaN", 3, []float64{1, math.NaN(), 3}, model.SignalNone},
		{"enabled infinite", 3, []float64{1, 2, math.Inf(1)}, model.SignalNone},
		{"equality", 3, []float64{2, 2, 2}, model.SignalNone},
		{"downtrend", 3, []float64{3, 2, 1}, model.SignalNone},
		{"strict uptrend", 3, []float64{1, 2, 3}, model.SignalBuy},
	} {
		t.Run(tc.name, func(t *testing.T) {
			p := testParams()
			p.HTFTrendEMA = tc.period
			md := snapshot(append(formation(), bar{103, 104, 101, 103})...)
			md.HTFCloses = tc.closes
			sig := NewWithParams("TEST", step, p).Decide(md)
			if sig.Kind != tc.want {
				t.Fatalf("signal=%+v, want %v", sig, tc.want)
			}
			if tc.period > 0 && sig.Kind == model.SignalBuy && !strings.Contains(sig.EntryReason, "H4") {
				t.Fatalf("missing H4 verdict: %+v", sig)
			}
		})
	}
	t.Run("position exit ignores missing H4", func(t *testing.T) {
		p := testParams()
		p.HTFTrendEMA = 3
		md := snapshot(bar{104, 105, 99, 104})
		md.Position = &strategy.Position{PurchasePrice: 104, StopLoss: 100}
		sig := NewWithParams("TEST", step, p).Decide(md)
		if sig.Kind != model.SignalSell || sig.Reason != "SL" {
			t.Fatalf("signal=%+v", sig)
		}
	})
}

func TestEntryDistance(t *testing.T) {
	for _, tc := range []struct{ distance, want float64 }{{0, 0}, {.5, 103}} {
		p := testParams()
		p.MaxEntryDistanceATR = tc.distance
		md := snapshot(append(formation(), bar{103, 200, 102.5, 103}, bar{103, 104, 101, 103})...)
		sig := NewWithParams("TEST", step, p).Decide(md)
		if sig.Kind != model.SignalBuy || sig.MaxEntryPrice != tc.want || sig.ATR != 2 {
			t.Fatalf("distance %v: %+v", tc.distance, sig)
		}
		if tc.distance > 0 && !strings.Contains(sig.EntryReason, "max entry") {
			t.Fatalf("missing limit: %+v", sig)
		}
	}
}

func TestStopBuffer(t *testing.T) {
	p := testParams()
	p.StopBufferATR = .5
	md := snapshot(append(formation(), bar{103, 200, 102.5, 103}, bar{103, 104, 101, 103})...)
	s := NewWithParams("TEST", step, p)
	sig := s.Decide(md)
	if sig.Kind != model.SignalBuy || sig.StopLoss != 99 || sig.TakeProfit != 109 || !strings.Contains(sig.EntryReason, "stop 99.0000") {
		t.Fatalf("buffered entry: %+v", sig)
	}
	md.Position = &strategy.Position{PurchasePrice: 104, StopLoss: sig.StopLoss}
	md.Highs[len(md.Highs)-1] = 111.5
	sig = s.Decide(md)
	if sig.Kind != model.SignalSell || sig.Reason != "TP" || sig.TakeProfit != 111.5 {
		t.Fatalf("actual-fill target: %+v", sig)
	}
}
