package core

import (
	"testing"
	"time"

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
	md.DailyTimes = []time.Time{start.AddDate(0, 0, -6), start.AddDate(0, 0, -5), start.AddDate(0, 0, -4), start.AddDate(0, 0, -3)}
	md.DailyHighs = []float64{101, 101, 101, 101}
	md.DailyLows = []float64{99, 99, 99, 99}
	md.DailyCloses = []float64{100, 100, 100, 100}
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
		{"third high equals second high", func(b []bar) { b[2].h = b[1].h }, .25, 1},
		{"third high below second high", func(b []bar) { b[1].h = 104; b[2].h = 103.5 }, .25, 1},
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

// This fails if FVG qualification falls back to the M30 range instead of the
// completed daily ATR available when the zone forms.
func TestFVGFormationUsesCompletedDailyATR(t *testing.T) {
	p := testParams()
	p.MinGapATR = 1
	md := snapshot(formation()...)
	md.DailyTimes = []time.Time{
		time.Date(2026, 10, 29, 0, 0, 0, 0, time.UTC),
		time.Date(2026, 10, 30, 0, 0, 0, 0, time.UTC),
	}
	md.DailyHighs = []float64{110, 110}
	md.DailyLows = []float64{100, 100}
	md.DailyCloses = []float64{105, 105}

	if zones := NewWithParams("TEST", step, p).discover(md); len(zones) != 0 {
		t.Fatalf("zones=%+v, want none: daily ATR 10 makes gap 2 too small", zones)
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
			md.DailyTimes = []time.Time{start.AddDate(0, 0, -6), start.AddDate(0, 0, -5), start.AddDate(0, 0, -4), start.AddDate(0, 0, -3)}
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
