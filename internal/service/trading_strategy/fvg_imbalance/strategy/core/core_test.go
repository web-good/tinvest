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
		{"third high equals second high", func(b []bar) { b[2].h = b[1].h }, .25, 0},
		{"third high below second high", func(b []bar) { b[1].h = 104; b[2].h = 103.5 }, .25, 0},
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

func TestFVGFormationHigherHighIsOptional(t *testing.T) {
	b := formation()
	b[2].h = b[1].h
	p := testParams()
	p.RequireThirdCandleHigherHigh = 0
	zones := NewWithParams("TEST", step, p).discover(snapshot(b...))
	if len(zones) != 1 {
		t.Fatalf("zones=%+v, want one zone when third-candle higher-high filter is disabled", zones)
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
	if p.MinRetestDepthPct != 50 || p.RequireThirdCandleHigherHigh != 1 || p.RetestDepthByBody != 1 || p.HTFTrendEMA != 0 || p.StopBufferATR != 0 || p.UseStochFilter != 0 {
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
		{"negative EMA", func(p *Params) { p.HTFTrendEMA = -1 }},
		{"negative higher-high switch", func(p *Params) { p.RequireThirdCandleHigherHigh = -1 }}, {"large higher-high switch", func(p *Params) { p.RequireThirdCandleHigherHigh = 2 }},
		{"negative retest-depth mode", func(p *Params) { p.RetestDepthByBody = -1 }}, {"large retest-depth mode", func(p *Params) { p.RetestDepthByBody = 2 }},
		{"negative stochastic switch", func(p *Params) { p.UseStochFilter = -1 }}, {"large stochastic switch", func(p *Params) { p.UseStochFilter = 2 }},
		{"negative buffer", func(p *Params) { p.StopBufferATR = -1 }}, {"NaN buffer", func(p *Params) { p.StopBufferATR = math.NaN() }}, {"infinite buffer", func(p *Params) { p.StopBufferATR = math.Inf(-1) }},
		{"zero ATR", func(p *Params) { p.ATRPeriod = 0 }}, {"negative ATR", func(p *Params) { p.ATRPeriod = -1 }},
		{"zero retest bars", func(p *Params) { p.MaxRetestBars = 0 }}, {"negative retest bars", func(p *Params) { p.MaxRetestBars = -1 }},
		{"negative gap", func(p *Params) { p.MinGapATR = -1 }}, {"negative target", func(p *Params) { p.TakeProfitR = -1 }},
		{"negative depth", func(p *Params) { p.MinRetestDepthPct = -1 }}, {"large depth", func(p *Params) { p.MinRetestDepthPct = 101 }},
		{"NaN depth", func(p *Params) { p.MinRetestDepthPct = math.NaN() }}, {"infinite depth", func(p *Params) { p.MinRetestDepthPct = math.Inf(1) }},
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
	p.MinRetestDepthPct = 0
	p.UseStochFilter = 1
	if err := ValidateParams(p); err != nil {
		t.Fatal(err)
	}
}

// The high near the middle of the warm-up keeps the latest 14-bar range at
// [98, 130]. Thus closes 107 and 103 produce %K above and below 20.
func stochasticFormation() []bar {
	b := make([]bar, 11)
	for i := range b {
		b[i] = bar{110, 120, 108, 110}
	}
	b[5].h = 130
	return append(b, formation()...)
}

func TestStochasticSameCandleConfirmation(t *testing.T) {
	p := testParams()
	p.UseStochFilter = 1
	md := snapshot(append(stochasticFormation(), bar{103, 104, 100.5, 101})...)
	if sig := NewWithParams("TEST", step, p).Decide(md); sig.Kind != model.SignalBuy {
		t.Fatalf("depth candle with %%K below 20: %+v", sig)
	}
}

func TestStochasticWaitsAfterDepthWithoutNewTouch(t *testing.T) {
	p := testParams()
	p.UseStochFilter = 1
	s := NewWithParams("TEST", step, p)
	b := append(stochasticFormation(), bar{101, 108, 100.5, 107})
	if sig := s.Decide(snapshot(b...)); sig.Kind != model.SignalNone {
		t.Fatalf("depth reached with %%K above 20: %+v", sig)
	}
	b = append(b, bar{104, 105, 103, 103})
	md := snapshot(b...)
	if sig := s.Decide(md); sig.Kind != model.SignalBuy || !sig.FillBy.Equal(md.Times[len(md.Times)-1].Add(step)) {
		t.Fatalf("later %%K below 20 without new touch: %+v", sig)
	}
}

func TestStochasticBreachAfterDepthInvalidates(t *testing.T) {
	p := testParams()
	p.UseStochFilter = 1
	b := append(stochasticFormation(), bar{101, 108, 100.5, 107}, bar{101, 105, 99, 103}, bar{104, 105, 103, 103})
	if sig := NewWithParams("TEST", step, p).Decide(snapshot(b...)); sig.Kind != model.SignalNone {
		t.Fatalf("breached zone must stay invalid: %+v", sig)
	}
}

func TestStochasticInsufficientHistory(t *testing.T) {
	p := testParams()
	p.UseStochFilter = 1
	md := snapshot(append(formation(), bar{103, 104, 100.5, 101})...)
	if sig := NewWithParams("TEST", step, p).Decide(md); sig.Kind != model.SignalNone {
		t.Fatalf("four bars cannot supply 14-bar %%K: %+v", sig)
	}
}

func TestStochasticDisabledEntersImmediately(t *testing.T) {
	p := testParams()
	p.UseStochFilter = 0
	b := append(stochasticFormation(), bar{101, 108, 100.5, 107})
	if sig := NewWithParams("TEST", step, p).Decide(snapshot(b...)); sig.Kind != model.SignalBuy {
		t.Fatalf("disabled filter must enter on retest: %+v", sig)
	}
}

func TestStochasticExpiryWhileWaiting(t *testing.T) {
	p := testParams()
	p.UseStochFilter = 1
	p.MaxRetestBars = 1
	b := append(stochasticFormation(), bar{101, 108, 100.5, 107}, bar{104, 105, 103, 103})
	if sig := NewWithParams("TEST", step, p).Decide(snapshot(b...)); sig.Kind != model.SignalNone {
		t.Fatalf("expired zone must not enter: %+v", sig)
	}
}

func TestStochasticLookback(t *testing.T) {
	p := testParams()
	p.UseStochFilter = 1
	if got := NewWithParams("TEST", step, p).Lookback(); got != 22 {
		t.Fatalf("lookback %d, want 22", got)
	}
}

func TestRetestDepthCanUseWick(t *testing.T) {
	for _, tc := range []struct {
		name             string
		open, low, close float64
		want             model.SignalKind
	}{
		{"wick reaches midpoint while body stays above", 103, 101, 103, model.SignalBuy},
		{"wick stays above midpoint", 103, 101.01, 103, model.SignalNone},
		{"lower boundary inclusive", 103, 100, 103, model.SignalBuy},
		{"lower boundary breach invalidates", 103, 99.99, 103, model.SignalNone},
	} {
		t.Run(tc.name, func(t *testing.T) {
			p := testParams()
			p.RetestDepthByBody = 0
			md := snapshot(append(formation(), bar{tc.open, 104, tc.low, tc.close})...)
			if sig := NewWithParams("TEST", step, p).Decide(md); sig.Kind != tc.want {
				t.Fatalf("signal=%+v, want %v", sig, tc.want)
			}
		})
	}
}

// Compile-time assertion also pins the consumer interface.
var _ strategy.Strategy = (*Strategy)(nil)

// Require the candle body to reach the configured depth while preserving
// inclusive boundaries and lower-wick invalidation.
func TestRetestDepthUsesBody(t *testing.T) {
	for _, tc := range []struct {
		name                    string
		depth, open, low, close float64
		want                    model.SignalKind
	}{
		{"wick reaches midpoint but body stays above", 50, 103, 101, 103, model.SignalNone},
		{"bearish body reaches midpoint", 50, 103, 100.5, 101, model.SignalBuy},
		{"bullish body starts at midpoint", 50, 101, 100.5, 103, model.SignalBuy},
		{"body stays above midpoint", 50, 103, 101, 101.01, model.SignalNone},
		{"lower body boundary inclusive", 50, 103, 100, 100, model.SignalBuy},
		{"lower wick breach invalidates", 50, 103, 99.99, 100, model.SignalNone},
		{"full depth lower body inclusive", 100, 103, 100, 100, model.SignalBuy},
		{"full depth body above lower", 100, 103, 100, 100.01, model.SignalNone},
		{"fractional depth body inclusive", 37.5, 103, 101, 101.25, model.SignalBuy},
		{"fractional depth body shallow", 37.5, 103, 101, 101.26, model.SignalNone},
		{"zero depth upper body inclusive", 0, 103, 102, 102, model.SignalBuy},
		{"zero depth wick only", 0, 103, 102, 103, model.SignalNone},
	} {
		t.Run(tc.name, func(t *testing.T) {
			p := testParams()
			p.MinRetestDepthPct = tc.depth
			md := snapshot(append(formation(), bar{tc.open, 104, tc.low, tc.close})...)
			sig := NewWithParams("TEST", step, p).Decide(md)
			if sig.Kind != tc.want {
				t.Fatalf("signal=%+v, want %v", sig, tc.want)
			}
			if sig.Kind == model.SignalBuy && (sig.StopLoss != 100 || !sig.FillBy.Equal(md.Times[len(md.Times)-1].Add(step)) || sig.TakeProfit != md.Price+1.5*(md.Price-100)) {
				t.Fatalf("wrong entry levels: %+v", sig)
			}
		})
	}
}

func TestRetest(t *testing.T) {
	for _, tc := range []struct {
		name  string
		later []bar
		want  model.SignalKind
	}{
		{"formation cannot retest itself", nil, model.SignalNone},
		{"lower breach invalidates", []bar{{103, 104, 99, 101}, {103, 104, 101, 101}}, model.SignalNone},
		{"second protected retest remains valid", []bar{{103, 104, 100, 102}, {103, 104, 101, 101}}, model.SignalBuy},
		{"no touch", []bar{{103, 104, 102.5, 101}}, model.SignalNone},
	} {
		t.Run(tc.name, func(t *testing.T) {
			sig := NewWithParams("TEST", step, testParams()).Decide(snapshot(append(formation(), tc.later...)...))
			if sig.Kind != tc.want {
				t.Fatalf("signal=%+v, want %v", sig, tc.want)
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
			md := snapshot(append(formation(), bar{103, 104, 101, 101})...)
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
	if sig.Kind != model.SignalSell || sig.Reason != "SL_EMERGENCY" || sig.StopLoss != 100 {
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
			b = append(b, bar{103, 104, 101, 101})
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
	b := append(formation(), bar{101, 101, 100.5, 101}, bar{101, 103, 100.5, 102}, bar{103, 104, 102, 103}, bar{103, 104, 101.5, 101.5})
	sig := NewWithParams("TEST", step, testParams()).Decide(snapshot(b...))
	if sig.Kind != model.SignalBuy || sig.StopLoss != 101 {
		t.Fatalf("want newest stop 101: %+v", sig)
	}
}
func TestRetestFrozenATRAndDisabledTarget(t *testing.T) {
	p := testParams()
	p.MinGapATR = .9
	p.TakeProfitR = 0
	b := append(formation(), bar{103, 200, 102.5, 103}, bar{103, 104, 101, 101})
	sig := NewWithParams("TEST", step, p).Decide(snapshot(b...))
	if sig.Kind != model.SignalBuy || sig.ATR != 2 || sig.TakeProfit != 0 {
		t.Fatalf("formation ATR and disabled target: %+v", sig)
	}
}
func TestRetestUnsupportedStepAndMalformedData(t *testing.T) {
	md := snapshot(append(formation(), bar{103, 104, 101, 101})...)
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
		{"stop wins", 104, 99, 120, 1.5, "SL_EMERGENCY", 0},
		{"fill below stop with emergency breach", 99, 98, 110, 1.5, "SL_EMERGENCY", 0},
		{"fill at stop with emergency equality", 100, 100, 110, 1.5, "SL_EMERGENCY", 0},
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
	bars = append(bars, bar{103, 104, 101, 101}, bar{103, 104, 101, 101}, bar{103, 104, 101, 101})
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
		bars = append(bars, bar{103, 104, 101, 101})
		// This is the same fixed-size truncation performed by backtest.Run.
		md := snapshot(bars[len(bars)-s.Lookback():]...)
		sig := s.Decide(md)
		if sig.Kind != model.SignalBuy || sig.ATR != 2.5 {
			t.Fatalf("age %d, lookback %d: want buy with formation ATR 2.5, got %+v", age, s.Lookback(), sig)
		}
	}
}

func TestLookback(t *testing.T) {
	p := DefaultParams()
	p.ATRPeriod = 2
	p.MaxRetestBars = 8
	s := NewWithParams("TEST", step, p)
	if got := s.Lookback(); got != 12 {
		t.Fatalf("lookback %d, want 12", got)
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
			md := snapshot(append(formation(), bar{103, 104, 101, 101})...)
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
		if sig.Kind != model.SignalSell || sig.Reason != "SL_EMERGENCY" {
			t.Fatalf("signal=%+v", sig)
		}
	})
}

func TestStopBuffer(t *testing.T) {
	p := testParams()
	p.StopBufferATR = .5
	md := snapshot(append(formation(), bar{103, 200, 102.5, 103}, bar{103, 104, 101, 101})...)
	s := NewWithParams("TEST", step, p)
	sig := s.Decide(md)
	if sig.Kind != model.SignalBuy || sig.StopLoss != 99 || sig.TakeProfit != 104 || !strings.Contains(sig.EntryReason, "stop 99.0000") {
		t.Fatalf("buffered entry: %+v", sig)
	}
	md.Position = &strategy.Position{PurchasePrice: 104, StopLoss: sig.StopLoss}
	md.Highs[len(md.Highs)-1] = 111.5
	sig = s.Decide(md)
	if sig.Kind != model.SignalSell || sig.Reason != "TP" || sig.TakeProfit != 111.5 {
		t.Fatalf("actual-fill target: %+v", sig)
	}
}

func TestStopBufferRequiresPositiveStop(t *testing.T) {
	for _, tc := range []struct {
		name   string
		buffer float64
		want   model.SignalKind
		stop   float64
	}{
		{"positive stop", 49.5, model.SignalBuy, 1},
		{"zero stop", 50, model.SignalNone, 0},
		{"negative stop", 51, model.SignalNone, 0},
		{"overflowing buffer", math.MaxFloat64, model.SignalNone, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			p := testParams()
			p.StopBufferATR = tc.buffer
			md := snapshot(append(formation(), bar{103, 104, 101, 101})...)
			sig := NewWithParams("TEST", step, p).Decide(md)
			if sig.Kind != tc.want || sig.StopLoss != tc.stop {
				t.Fatalf("signal=%+v, want kind %v and stop %v", sig, tc.want, tc.stop)
			}
		})
	}
}

// Catch wick-only primary exits and incorrect stop/target priority.
func TestPositionDualStops(t *testing.T) {
	for _, tc := range []struct {
		name                     string
		buffer, low, close, high float64
		want                     model.SignalKind
		reason                   string
		stop                     float64
	}{
		{"primary wick only", .5, 99, 101, 105, model.SignalNone, "", 0},
		{"primary close equality", .5, 101, 100, 105, model.SignalSell, "SL_CLOSE", 100},
		{"primary close below", .5, 99, 99, 105, model.SignalSell, "SL_CLOSE", 100},
		{"emergency equality", .5, 98, 101, 105, model.SignalSell, "SL_EMERGENCY", 98},
		{"emergency below", .5, 97, 101, 105, model.SignalSell, "SL_EMERGENCY", 98},
		{"zero buffer equality", 0, 100, 101, 105, model.SignalSell, "SL_EMERGENCY", 100},
		{"primary before target", .5, 99, 100, 120, model.SignalSell, "SL_CLOSE", 100},
		{"emergency before primary and target", .5, 98, 99, 120, model.SignalSell, "SL_EMERGENCY", 98},
		{"wick allows target", .5, 99, 101, 120, model.SignalSell, "TP", 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			p := testParams()
			p.StopBufferATR = tc.buffer
			md := snapshot(bar{104, tc.high, tc.low, tc.close})
			md.Position = &strategy.Position{PurchasePrice: 104, StopLoss: 100, EntryATR: 2}
			sig := NewWithParams("TEST", step, p).Decide(md)
			if sig.Kind != tc.want || sig.Reason != tc.reason || sig.StopLoss != tc.stop {
				t.Fatalf("signal=%+v, want %v / %s / %v", sig, tc.want, tc.reason, tc.stop)
			}
			if tc.reason == "TP" && sig.TakeProfit != 110 {
				t.Fatalf("target must use primary stop: %+v", sig)
			}
		})
	}
}
