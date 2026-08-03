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
