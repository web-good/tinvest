package core

import (
	"testing"
	"tinvest/internal/service/trading_strategy/scalping/model"
	"tinvest/internal/service/trading_strategy/scalping/strategy"
)

func TestStochasticCrossAfterFVG(t *testing.T) {
	p := testParams()
	bars := make([]bar, 14)
	for i := range bars {
		bars[i] = bar{99, 100, 98, 99}
	}
	bars = append(bars, formation()...)
	bars = append(bars, bar{103, 104, 98, 98})
	sig := NewWithParams("TEST", step, p).Decide(snapshot(bars...))
	if sig.Kind != model.SignalBuy {
		t.Fatalf("cross below 20 after FVG: got %+v", sig)
	}
}

func TestFVGCanTriggerOnlyOnFirstStochasticCross(t *testing.T) {
	p := testParams()
	p.StochPeriod = 2
	bars := make([]bar, 14)
	for i := range bars {
		bars[i] = bar{99, 100, 98, 99}
	}
	bars = append(bars, formation()...)
	bars = append(bars, bar{103, 104, 102, 103}, bar{103, 104, 98, 98})
	first := NewWithParams("TEST", step, p).Decide(snapshot(bars...))
	if first.Kind != model.SignalBuy {
		t.Fatalf("first stochastic cross should enter: %+v", first)
	}

	bars = append(bars, bar{100, 104, 98, 103}, bar{103, 104, 98, 98})
	second := NewWithParams("TEST", step, p).Decide(snapshot(bars...))
	if second.Kind == model.SignalBuy {
		t.Fatalf("same FVG must not trigger on a later stochastic cross: %+v", second)
	}
}

func TestFVGStaysConsumedWhenFirstTriggerCannotEnter(t *testing.T) {
	p := testParams()
	p.StochPeriod = 2
	bars := make([]bar, 14)
	for i := range bars {
		bars[i] = bar{99, 100, 98, 99}
	}
	bars = append(bars, formation()...)
	bars = append(bars, bar{103, 104, 102, 103}, bar{103, 104, 98, 98})
	occupied := snapshot(bars...)
	occupied.Position = &strategy.Position{StopLoss: 1}
	if got := NewWithParams("TEST", step, p).Decide(occupied); got.Kind == model.SignalBuy {
		t.Fatalf("cannot enter while a position is open: %+v", got)
	}

	bars = append(bars, bar{100, 104, 98, 103}, bar{103, 104, 98, 98})
	if got := NewWithParams("TEST", step, p).Decide(snapshot(bars...)); got.Kind == model.SignalBuy {
		t.Fatalf("FVG stays consumed after its first trigger even without an entry: %+v", got)
	}
}

func TestStochasticPositionStopWins(t *testing.T) {
	p := testParams()
	md := snapshot(bar{100, 110, 90, 105})
	md.Position = &strategy.Position{PurchasePrice: 100, StopLoss: 95, TakeProfit: 105}
	sig := NewWithParams("TEST", step, p).Decide(md)
	if sig.Reason != "SL" || sig.StopLoss != 95 {
		t.Fatalf("want stop before target: %+v", sig)
	}
}

func TestStochasticExitAndTarget(t *testing.T) {
	p := testParams()
	p.StochPeriod = 2
	for _, tc := range []struct {
		name   string
		bars   []bar
		target float64
		reason string
	}{
		{"upward crossing", []bar{{100, 110, 90, 95}, {100, 110, 90, 100}, {100, 110, 90, 109}}, 0, "STOCH_EXIT"},
		{"target", []bar{{100, 110, 90, 95}, {100, 110, 90, 100}, {100, 110, 90, 109}}, 108, "TP"},
		{"no cross", []bar{{100, 110, 90, 108}, {100, 110, 90, 109}, {100, 110, 90, 109}}, 0, ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			md := snapshot(tc.bars...)
			md.Position = &strategy.Position{StopLoss: 80, TakeProfit: tc.target}
			sig := NewWithParams("TEST", step, p).Decide(md)
			if sig.Reason != tc.reason {
				t.Fatalf("signal: %+v", sig)
			}
		})
	}
}

func TestStochasticExitCanBeDisabled(t *testing.T) {
	p := testParams()
	p.StochPeriod = 2
	p.UseStochExit = 0
	md := snapshot(bar{100, 110, 90, 95}, bar{100, 110, 90, 100}, bar{100, 110, 90, 109})
	md.Position = &strategy.Position{StopLoss: 80}
	if got := NewWithParams("TEST", step, p).Decide(md); got.Kind != model.SignalNone {
		t.Fatalf("stochastic exit should be disabled: %+v", got)
	}
	md.Position.StopLoss = 95
	if got := NewWithParams("TEST", step, p).Decide(md); got.Reason != "SL" {
		t.Fatalf("stop-loss should remain active: %+v", got)
	}
	md.Position.StopLoss = 80
	md.Position.TakeProfit = 108
	if got := NewWithParams("TEST", step, p).Decide(md); got.Reason != "TP" {
		t.Fatalf("take-profit should remain active: %+v", got)
	}
}

func TestDefaultParamsEnableStochasticExit(t *testing.T) {
	if got := DefaultParams().UseStochExit; got != 1 {
		t.Fatalf("UseStochExit default = %d, want 1", got)
	}
}

func TestStochasticEntryWindowAndHistory(t *testing.T) {
	p := testParams()
	p.MaxWaitBars = 2
	warm := make([]bar, 14)
	for i := range warm {
		warm[i] = bar{99, 100, 98, 99}
	}
	base := append(warm, formation()...)
	cases := []struct {
		name string
		bars []bar
		want model.SignalKind
	}{
		{"last eligible bar", append(append([]bar{}, base...), bar{103, 104, 102, 103}, bar{103, 104, 98, 98}), model.SignalBuy},
		{"expired", append(append([]bar{}, base...), bar{103, 104, 102, 103}, bar{103, 104, 102, 103}, bar{103, 104, 98, 98}), model.SignalNone},
		{"insufficient stochastic history", append(formation(), bar{103, 104, 98, 98}), model.SignalNone},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			sig := NewWithParams("TEST", step, p).Decide(snapshot(tc.bars...))
			if sig.Kind != tc.want {
				t.Fatalf("got %+v, want %v", sig, tc.want)
			}
		})
	}
}

func TestStochasticHTFFilter(t *testing.T) {
	p := testParams()
	p.HTFTrendEMA = 3
	bars := make([]bar, 14)
	for i := range bars {
		bars[i] = bar{99, 100, 98, 99}
	}
	bars = append(bars, formation()...)
	bars = append(bars, bar{103, 104, 98, 98})
	for _, tc := range []struct {
		name  string
		daily []float64
		htf   []float64
		want  model.SignalKind
	}{{"insufficient daily history", []float64{1, 2}, []float64{1, 2, 3}, model.SignalNone},
		{"daily down despite H4 up", []float64{3, 2, 1}, []float64{1, 2, 3}, model.SignalNone},
		{"daily up despite H4 down", []float64{1, 2, 3}, []float64{3, 2, 1}, model.SignalBuy}} {
		t.Run(tc.name, func(t *testing.T) {
			md := snapshot(bars...)
			md.DailyCloses = tc.daily
			md.DailyHighs = make([]float64, len(tc.daily))
			md.DailyLows = make([]float64, len(tc.daily))
			md.DailyTimes = md.DailyTimes[:len(tc.daily)]
			for i, close := range tc.daily {
				md.DailyHighs[i] = close + .2
				md.DailyLows[i] = close - .2
			}
			md.HTFCloses = tc.htf
			got := NewWithParams("TEST", step, p).Decide(md)
			if got.Kind != tc.want {
				t.Fatalf("signal: %+v", got)
			}
		})
	}
}

func TestStochasticInvalidConfiguration(t *testing.T) {
	base := DefaultParams()
	for _, edit := range []func(*Params){func(p *Params) { p.StopATR = 0 }, func(p *Params) { p.TakeProfitATR = -1 }, func(p *Params) { p.StochPeriod = 0 }, func(p *Params) { p.MaxWaitBars = 0 }, func(p *Params) { p.ATRPeriod = 0 }, func(p *Params) { p.MinGapATR = -1 }, func(p *Params) { p.UseStochExit = 2 }} {
		p := base
		edit(&p)
		if ValidateParams(p) == nil {
			t.Errorf("accepted %+v", p)
		}
	}
}

func TestStochasticExitBoundary(t *testing.T) {
	p := testParams()
	p.StochPeriod = 2
	for _, tc := range []struct {
		name              string
		previous, current float64
		want              string
	}{{"at 80 does not cross", 70, 80, ""}, {"from 80 to above crosses", 80, 81, "STOCH_EXIT"}, {"already above", 81, 82, ""}} {
		t.Run(tc.name, func(t *testing.T) {
			md := snapshot(bar{50, 100, 0, 50}, bar{50, 100, 0, tc.previous}, bar{50, 100, 0, tc.current})
			md.Position = &strategy.Position{}
			got := NewWithParams("TEST", step, p).Decide(md)
			if got.Reason != tc.want {
				t.Fatalf("signal %+v", got)
			}
		})
	}
}
