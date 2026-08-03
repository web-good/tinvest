package backtest

import (
	"math"
	"reflect"
	"testing"
	"time"

	"tinvest/internal/domain/backtest"
	"tinvest/internal/service/trading_strategy/rsi_pullback/strategy/core"
)

func TestPullbackGridHas24UniqueConfigs(t *testing.T) {
	grid := PullbackGrid()
	if len(grid) != 24 {
		t.Fatalf("grid size = %d, want 24 (2 RSIPeriod x 3 RSILower x 2 EMASlow x 2 TPDailyATR)", len(grid))
	}
	seen := make(map[core.Params]bool, len(grid))
	for _, p := range grid {
		if seen[p] {
			t.Fatalf("duplicate config in grid: %+v", p)
		}
		seen[p] = true
	}
}

func TestPullbackGridSweepsOnlyTheFourAxes(t *testing.T) {
	rsiPeriods := map[int]bool{}
	rsiLowers := map[float64]bool{}
	emaSlows := map[int]bool{}
	tps := map[float64]bool{}
	for _, p := range PullbackGrid() {
		rsiPeriods[p.RSIPeriod] = true
		rsiLowers[p.RSILower] = true
		emaSlows[p.EMASlow] = true
		tps[p.TPDailyATR] = true

		// Everything else is pinned: the screener compares tickers, not configurations.
		if p.EMAFast != 20 {
			t.Fatalf("EMAFast = %d, want pinned 20", p.EMAFast)
		}
		if p.StopDailyATR != 0.5 {
			t.Fatalf("StopDailyATR = %v, want pinned 0.5", p.StopDailyATR)
		}
		if p.DailyATRPeriod != 14 {
			t.Fatalf("DailyATRPeriod = %d, want pinned 14", p.DailyATRPeriod)
		}
		if p.UseDayATRGate != 1 || p.FreshDayATR != 0.3 || p.SpentDayATR != 0.8 {
			t.Fatalf("day gate = %d/%v/%v, want pinned 1/0.3/0.8", p.UseDayATRGate, p.FreshDayATR, p.SpentDayATR)
		}
		if p.UseVolume != 0 {
			t.Fatalf("UseVolume = %d, want pinned 0 (the volume gate starves an already thin sample)", p.UseVolume)
		}
		if p.UseRSIExit != 1 || p.RSIUpper != 60 {
			t.Fatalf("RSI exit = %d/%v, want pinned 1/60", p.UseRSIExit, p.RSIUpper)
		}
		if p.UseTrail != 0 || p.TrailDailyATR != 0 {
			t.Fatalf("trail = %d/%v, want pinned off (trailing is tuning, not ticker fitness)", p.UseTrail, p.TrailDailyATR)
		}
	}
	if len(rsiPeriods) != 2 || len(rsiLowers) != 3 || len(emaSlows) != 2 || len(tps) != 2 {
		t.Fatalf("axes = %d/%d/%d/%d, want 2/3/2/2", len(rsiPeriods), len(rsiLowers), len(emaSlows), len(tps))
	}
}

func TestPullbackGridNeverDisablesTheStop(t *testing.T) {
	// Same invariant TestRSIPullbackCalFilesValid enforces on the JSON grids: a
	// multi-day long without a stop is not a configuration the screener may pick.
	for _, p := range PullbackGrid() {
		if p.StopDailyATR <= 0 {
			t.Fatalf("StopDailyATR = %v, want > 0", p.StopDailyATR)
		}
	}
}

func TestProfitFactor(t *testing.T) {
	tests := []struct {
		name    string
		pnl     []float64
		wantPF  float64
		wantInf bool
		wantN   int
	}{
		{name: "mixed", pnl: []float64{100, -50, 50, -50}, wantPF: 1.5, wantN: 4},
		{name: "no losses is infinite, not gross profit", pnl: []float64{100, 200}, wantInf: true, wantN: 2},
		{name: "no profit", pnl: []float64{-100, -50}, wantPF: 0, wantN: 2},
		{name: "empty", pnl: nil, wantPF: 0, wantN: 0},
		{name: "zero pnl counts as a win side", pnl: []float64{0, -100}, wantPF: 0, wantN: 2},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			trades := make([]backtest.Trade, 0, len(tt.pnl))
			for _, p := range tt.pnl {
				trades = append(trades, backtest.Trade{PnL: p})
			}
			pf, n := profitFactor(trades)
			if n != tt.wantN {
				t.Fatalf("n = %d, want %d", n, tt.wantN)
			}
			if tt.wantInf {
				if !math.IsInf(pf, 1) {
					t.Fatalf("pf = %v, want +Inf", pf)
				}
				return
			}
			if math.Abs(pf-tt.wantPF) > 1e-9 {
				t.Fatalf("pf = %v, want %v", pf, tt.wantPF)
			}
		})
	}
}

func TestSplitTradesByEntryTime(t *testing.T) {
	split := time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC)
	trades := []backtest.Trade{
		{EntryTime: split.AddDate(0, 0, -10), ExitTime: split.AddDate(0, 0, -9), PnL: 1},  // train
		{EntryTime: split.AddDate(0, 0, -1), ExitTime: split.AddDate(0, 0, 3), PnL: 2},    // train: straddles the split, classified by ENTRY
		{EntryTime: split, ExitTime: split.AddDate(0, 0, 1), PnL: 3},                      // holdout: exactly on the boundary
		{EntryTime: split.AddDate(0, 0, 5), ExitTime: split.AddDate(0, 0, 6), PnL: 4},     // holdout
	}
	train, holdout := splitTrades(trades, split)
	if len(train) != 2 || train[0].PnL != 1 || train[1].PnL != 2 {
		t.Fatalf("train = %+v, want the two trades entered before the split", train)
	}
	if len(holdout) != 2 || holdout[0].PnL != 3 || holdout[1].PnL != 4 {
		t.Fatalf("holdout = %+v, want the boundary trade and the later one", holdout)
	}
}

func TestSplitTradesEmptySides(t *testing.T) {
	split := time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC)
	all := []backtest.Trade{{EntryTime: split.AddDate(0, 0, 1)}}
	train, holdout := splitTrades(all, split)
	if len(train) != 0 || len(holdout) != 1 {
		t.Fatalf("train/holdout = %d/%d, want 0/1", len(train), len(holdout))
	}
	train, holdout = splitTrades(nil, split)
	if len(train) != 0 || len(holdout) != 0 {
		t.Fatalf("train/holdout = %d/%d on empty input, want 0/0", len(train), len(holdout))
	}
}

func TestMedianF(t *testing.T) {
	tests := []struct {
		name string
		in   []float64
		want float64
	}{
		{name: "odd", in: []float64{3, 1, 2}, want: 2},
		{name: "even averages the middles", in: []float64{4, 1, 3, 2}, want: 2.5},
		{name: "all zero", in: []float64{0, 0, 0}, want: 0},
		{name: "empty", in: nil, want: 0},
		{name: "single", in: []float64{7}, want: 7},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			in := append([]float64(nil), tt.in...)
			if got := medianF(in); math.Abs(got-tt.want) > 1e-9 {
				t.Fatalf("medianF = %v, want %v", got, tt.want)
			}
			if !reflect.DeepEqual(in, tt.in) {
				t.Fatalf("medianF mutated its input: %v != %v", in, tt.in)
			}
		})
	}
}

func TestClampPF(t *testing.T) {
	if got, capped := clampPF(3, 10); got != 3 || capped {
		t.Fatalf("clampPF(3,10) = %v,%v, want 3,false", got, capped)
	}
	if got, capped := clampPF(math.Inf(1), 10); got != 10 || !capped {
		t.Fatalf("clampPF(+Inf,10) = %v,%v, want 10,true", got, capped)
	}
	if got, capped := clampPF(25, 10); got != 10 || !capped {
		t.Fatalf("clampPF(25,10) = %v,%v, want 10,true", got, capped)
	}
	if got, capped := clampPF(math.Inf(1), 0); !math.IsInf(got, 1) || capped {
		t.Fatalf("clampPF with cap<=0 must pass through, got %v,%v", got, capped)
	}
}

// screenTrade builds one trade entered `dayOffset` days from `base` with the given PnL.
func screenTrade(base time.Time, dayOffset int, pnl float64) backtest.Trade {
	entry := base.AddDate(0, 0, dayOffset)
	return backtest.Trade{EntryTime: entry, ExitTime: entry.AddDate(0, 0, 1), PnL: pnl}
}

func TestAggregateRanksByMedianNotBest(t *testing.T) {
	base := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	split := base.AddDate(0, 6, 0)
	grid := PullbackGrid()

	// 23 mediocre configurations (PF 1.0) and one lucky outlier (PF 5.0).
	results := make([]ConfigResult, 0, len(grid))
	for i, p := range grid {
		trades := []backtest.Trade{screenTrade(base, 1, 100), screenTrade(base, 2, -100)}
		if i == 0 {
			trades = []backtest.Trade{screenTrade(base, 1, 500), screenTrade(base, 2, -100)}
		}
		results = append(results, ConfigResult{Params: p, Trades: trades})
	}
	row := Aggregate("XXXX", "Test", results, split, DefaultScreenOpts())

	if math.Abs(row.PFMed-1.0) > 1e-9 {
		t.Fatalf("PFMed = %v, want 1.0 — the median must ignore the single lucky config", row.PFMed)
	}
	if math.Abs(row.BestPF-5.0) > 1e-9 {
		t.Fatalf("BestPF = %v, want 5.0", row.BestPF)
	}
	if row.Best != grid[0] {
		t.Fatalf("Best = %+v, want the outlier config %+v", row.Best, grid[0])
	}
}

func TestAggregatePlateauShare(t *testing.T) {
	base := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	split := base.AddDate(0, 6, 0)
	grid := PullbackGrid()
	opts := DefaultScreenOpts() // PlateauPF 1.3, PlateauTrades 10

	results := make([]ConfigResult, 0, len(grid))
	for i, p := range grid {
		var trades []backtest.Trade
		switch {
		case i < 12: // PF 2.0 on 12 trades -> counts toward the plateau
			for d := 0; d < 8; d++ {
				trades = append(trades, screenTrade(base, d, 100))
			}
			for d := 8; d < 12; d++ {
				trades = append(trades, screenTrade(base, d, -100))
			}
		case i < 18: // PF 2.0 but only 3 trades -> too thin to count
			trades = []backtest.Trade{screenTrade(base, 0, 100), screenTrade(base, 1, 100), screenTrade(base, 2, -100)}
		default: // PF 0.5 on 12 trades -> below the PF bar
			for d := 0; d < 4; d++ {
				trades = append(trades, screenTrade(base, d, 100))
			}
			for d := 4; d < 12; d++ {
				trades = append(trades, screenTrade(base, d, -100))
			}
		}
		results = append(results, ConfigResult{Params: p, Trades: trades})
	}
	row := Aggregate("XXXX", "Test", results, split, opts)

	if math.Abs(row.Plateau-0.5) > 1e-9 {
		t.Fatalf("Plateau = %v, want 0.5 (12 of 24 configs clear both PF>=1.3 and trades>=10)", row.Plateau)
	}
}

func TestAggregateClampsUnboundedPF(t *testing.T) {
	base := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	split := base.AddDate(0, 6, 0)
	results := make([]ConfigResult, 0, len(PullbackGrid()))
	for _, p := range PullbackGrid() {
		// Every configuration wins twice and never loses: PF is +Inf before clamping.
		results = append(results, ConfigResult{Params: p, Trades: []backtest.Trade{
			screenTrade(base, 1, 1000), screenTrade(base, 2, 2000),
		}})
	}
	row := Aggregate("XXXX", "Test", results, split, DefaultScreenOpts())

	if row.PFMed != 10 {
		t.Fatalf("PFMed = %v, want the 10.0 cap", row.PFMed)
	}
	if row.Capped != 24 {
		t.Fatalf("Capped = %d, want 24", row.Capped)
	}
}

func TestAggregateSplitsTrainAndHoldout(t *testing.T) {
	base := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	split := base.AddDate(0, 0, 100)
	results := make([]ConfigResult, 0, len(PullbackGrid()))
	for _, p := range PullbackGrid() {
		results = append(results, ConfigResult{Params: p, Trades: []backtest.Trade{
			screenTrade(base, 1, 200),   // train: PF 2.0
			screenTrade(base, 2, -100),  // train
			screenTrade(base, 150, 50),  // holdout: PF 0.5
			screenTrade(base, 151, -100), // holdout
		}})
	}
	row := Aggregate("XXXX", "Test", results, split, DefaultScreenOpts())

	if math.Abs(row.PFMed-2.0) > 1e-9 {
		t.Fatalf("PFMed(train) = %v, want 2.0", row.PFMed)
	}
	if math.Abs(row.PFMedHO-0.5) > 1e-9 {
		t.Fatalf("PFMed(holdout) = %v, want 0.5", row.PFMedHO)
	}
	if row.TradesMed != 2 || row.TradesMedHO != 2 {
		t.Fatalf("TradesMed/HO = %v/%v, want 2/2", row.TradesMed, row.TradesMedHO)
	}
}

func TestAggregateNoSignals(t *testing.T) {
	split := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)
	results := make([]ConfigResult, 0, len(PullbackGrid()))
	for _, p := range PullbackGrid() {
		results = append(results, ConfigResult{Params: p, Trades: nil})
	}
	row := Aggregate("XXXX", "Test", results, split, DefaultScreenOpts())
	if !row.NoSignals {
		t.Fatal("NoSignals = false, want true when every configuration produced zero trades")
	}
	if row.PFMed != 0 || row.Plateau != 0 {
		t.Fatalf("PFMed/Plateau = %v/%v, want 0/0", row.PFMed, row.Plateau)
	}
}

func TestAggregateOneTradingConfigIsNotNoSignals(t *testing.T) {
	base := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	split := base.AddDate(0, 6, 0)
	results := make([]ConfigResult, 0, len(PullbackGrid()))
	for i, p := range PullbackGrid() {
		var trades []backtest.Trade
		if i == 3 {
			trades = []backtest.Trade{screenTrade(base, 1, 100), screenTrade(base, 2, -50)}
		}
		results = append(results, ConfigResult{Params: p, Trades: trades})
	}
	row := Aggregate("XXXX", "Test", results, split, DefaultScreenOpts())
	if row.NoSignals {
		t.Fatal("NoSignals = true, want false when at least one configuration traded")
	}
}
