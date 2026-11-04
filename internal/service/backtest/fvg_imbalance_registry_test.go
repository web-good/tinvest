package backtest

import (
	"testing"
	"time"

	"tinvest/internal/enum"
	"tinvest/internal/service/trading_strategy/fvg_imbalance/strategy/core"
	"tinvest/internal/service/trading_strategy/scalping/model"
	"tinvest/internal/service/trading_strategy/scalping/strategy"
)

func TestFVGImbalanceDefaults(t *testing.T) {
	for ticker, want := range map[string]core.Params{"GAZP": {ATRPeriod: 14, MinGapATR: .8, MaxRetestBars: 8, TakeProfitR: 1.5, SwingK: 2, StructureLookbackBars: 48, MaxSweepToFVGBars: 8}, "ZZZZ": {ATRPeriod: 14, MinGapATR: .25, MaxRetestBars: 8, RequireCloseAboveZone: 1, TakeProfitR: 1.5, SwingK: 2, StructureLookbackBars: 48, MaxSweepToFVGBars: 8}} {
		t.Run(ticker, func(t *testing.T) {
			b, err := FVGImbalanceLookupOrGeneric(ticker, enum.Minutes30)
			if err != nil {
				t.Fatal(err)
			}
			if got := b.DefaultParams(); got != want {
				t.Fatalf("defaults = %+v, want %+v", got, want)
			}
			if got := b.Build(b.DefaultParams()).Ticker(); got != ticker {
				t.Fatalf("ticker = %q, want %q", got, ticker)
			}
		})
	}
}

func TestFVGImbalanceRejectsNonM30(t *testing.T) {
	for _, ticker := range []string{"GAZP", "ZZZZ"} {
		for _, interval := range []enum.Interval{enum.Minutes5, enum.Minutes15, enum.Hour1, enum.Hour4, enum.Day1, enum.Week1, 0, -1} {
			if _, err := FVGImbalanceLookupOrGeneric(ticker, interval); err == nil {
				t.Errorf("%s %s: expected rejection", ticker, interval)
			}
		}
	}
}

func TestFVGImbalanceParseParams(t *testing.T) {
	for _, ticker := range []string{"GAZP", "ZZZZ"} {
		t.Run(ticker, func(t *testing.T) {
			b, err := FVGImbalanceLookupOrGeneric(ticker, enum.Minutes30)
			if err != nil {
				t.Fatal(err)
			}
			defaults := b.DefaultParams()
			for _, raw := range []string{`{}`, `{"TakeProfitR": 2}`, `{"MinGapATR":0,"TakeProfitR":0,"RequireCloseAboveZone":0}`} {
				want := b.DefaultParams().(core.Params)
				switch raw {
				case `{"TakeProfitR": 2}`:
					want.TakeProfitR = 2
				case `{"MinGapATR":0,"TakeProfitR":0,"RequireCloseAboveZone":0}`:
					want.MinGapATR, want.TakeProfitR, want.RequireCloseAboveZone = 0, 0, 0
				}
				got, err := b.ParseParams([]byte(raw))
				if err != nil || got != want {
					t.Errorf("ParseParams(%s) = %+v, %v; want %+v", raw, got, err, want)
				}
			}
			for _, raw := range []string{
				`{`, `[]`, `{"ATRPeriod":"14"}`, `{"ATRPeriod":0}`, `{"ATRPeriod":-1}`,
				`{"MaxRetestBars":0}`, `{"MaxRetestBars":-1}`, `{"MinGapATR":-0.1}`,
				`{"TakeProfitR":-1}`, `{"RequireCloseAboveZone":-1}`, `{"RequireCloseAboveZone":2}`,
				`{"MinGapATR":1e999}`, `{"TakeProfitR":NaN}`,
				`{"RequireFirstRetest":-1}`, `{"RequireFirstRetest":2}`,
				`{"RequireBullishBOS":-1}`, `{"RequireBullishBOS":2}`,
				`{"RequireLiquiditySweep":-1}`, `{"RequireLiquiditySweep":2}`,
				`{"SwingK":0}`, `{"SwingK":-1}`, `{"StructureLookbackBars":0}`, `{"StructureLookbackBars":-1}`,
				`{"MaxSweepToFVGBars":0}`, `{"MaxSweepToFVGBars":-1}`,
				`{"HTFTrendEMA":-1}`, `{"MaxHoldBars":-1}`,
				`{"MaxEntryDistanceATR":-1}`, `{"MaxEntryDistanceATR":1e999}`, `{"MaxEntryDistanceATR":NaN}`,
				`{"StopBufferATR":-1}`, `{"StopBufferATR":1e999}`, `{"StopBufferATR":NaN}`,
			} {
				if got, err := b.ParseParams([]byte(raw)); err == nil {
					t.Errorf("ParseParams(%s) accepted invalid params: %+v", raw, got)
				}
			}

			raw := `{"RequireFirstRetest":1,"RequireBullishBOS":1,"RequireLiquiditySweep":1,"SwingK":3,"StructureLookbackBars":24,"MaxSweepToFVGBars":4,"HTFTrendEMA":20,"MaxEntryDistanceATR":0.5,"MaxHoldBars":8,"StopBufferATR":0.1}`
			want := b.DefaultParams().(core.Params)
			want.RequireFirstRetest, want.RequireBullishBOS, want.RequireLiquiditySweep = 1, 1, 1
			want.SwingK, want.StructureLookbackBars, want.MaxSweepToFVGBars = 3, 24, 4
			want.HTFTrendEMA, want.MaxEntryDistanceATR, want.MaxHoldBars, want.StopBufferATR = 20, .5, 8, .1
			got, err := b.ParseParams([]byte(raw))
			if err != nil || got != want {
				t.Errorf("overlay = %+v, %v; want %+v", got, err, want)
			}

			if got := b.DefaultParams(); got != defaults {
				t.Fatalf("parsing mutated defaults: %+v", got)
			}
		})
	}
}

func TestFVGImbalancePartialQualityOverlays(t *testing.T) {
	b, err := FVGImbalanceLookupOrGeneric("GAZP", enum.Minutes30)
	if err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		raw   string
		apply func(*core.Params)
	}{
		{`{"RequireFirstRetest":1}`, func(p *core.Params) { p.RequireFirstRetest = 1 }},
		{`{"RequireBullishBOS":1}`, func(p *core.Params) { p.RequireBullishBOS = 1 }},
		{`{"RequireLiquiditySweep":1}`, func(p *core.Params) { p.RequireLiquiditySweep = 1 }},
		{`{"SwingK":3}`, func(p *core.Params) { p.SwingK = 3 }},
		{`{"StructureLookbackBars":24}`, func(p *core.Params) { p.StructureLookbackBars = 24 }},
		{`{"MaxSweepToFVGBars":4}`, func(p *core.Params) { p.MaxSweepToFVGBars = 4 }},
		{`{"HTFTrendEMA":20}`, func(p *core.Params) { p.HTFTrendEMA = 20 }},
		{`{"MaxEntryDistanceATR":0.5}`, func(p *core.Params) { p.MaxEntryDistanceATR = .5 }},
		{`{"MaxHoldBars":8}`, func(p *core.Params) { p.MaxHoldBars = 8 }},
		{`{"StopBufferATR":0.1}`, func(p *core.Params) { p.StopBufferATR = .1 }},
	}
	for _, tc := range tests {
		t.Run(tc.raw, func(t *testing.T) {
			want := b.DefaultParams().(core.Params)
			tc.apply(&want)
			got, err := b.ParseParams([]byte(tc.raw))
			if err != nil || got != want {
				t.Fatalf("overlay = %+v, %v; want %+v", got, err, want)
			}
		})
	}
}

func TestFVGImbalanceBuildUsesM30AndOverrides(t *testing.T) {
	b, err := FVGImbalanceLookupOrGeneric("ZZZZ", enum.Minutes30)
	if err != nil {
		t.Fatal(err)
	}
	p, err := b.ParseParams([]byte(`{"ATRPeriod":1,"TakeProfitR":2}`))
	if err != nil {
		t.Fatal(err)
	}
	start := time.Date(2026, 11, 2, 10, 0, 0, 0, time.UTC)
	md := strategy.MarketData{
		Opens: []float64{99, 100, 103, 103}, Highs: []float64{100, 103, 104, 104},
		Lows: []float64{98, 99, 102, 101}, Closes: []float64{99, 102, 103, 103}, Price: 103,
		Times: []time.Time{start, start.Add(30 * time.Minute), start.Add(time.Hour), start.Add(90 * time.Minute)},
	}
	sig := b.Build(p).Decide(md)
	if sig.Kind != model.SignalBuy || sig.Ticker != "ZZZZ" || sig.StopLoss != 100 || sig.TakeProfit != 109 || !sig.FillBy.Equal(start.Add(2*time.Hour)) {
		t.Fatalf("M30 retest signal = %+v, want ZZZZ buy with stop 100, target 109 and next M30 fill", sig)
	}
}
