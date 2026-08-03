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
		{EntryTime: split.AddDate(0, 0, -10), ExitTime: split.AddDate(0, 0, -9), PnL: 1}, // train
		{EntryTime: split.AddDate(0, 0, -1), ExitTime: split.AddDate(0, 0, 3), PnL: 2},   // train: straddles the split, classified by ENTRY
		{EntryTime: split, ExitTime: split.AddDate(0, 0, 1), PnL: 3},                     // holdout: exactly on the boundary
		{EntryTime: split.AddDate(0, 0, 5), ExitTime: split.AddDate(0, 0, 6), PnL: 4},    // holdout
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
			screenTrade(base, 1, 200),    // train: PF 2.0
			screenTrade(base, 2, -100),   // train
			screenTrade(base, 150, 50),   // holdout: PF 0.5
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

func TestAggregateSelectsBestByRawPF(t *testing.T) {
	base := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	split := base.AddDate(0, 6, 0)
	grid := PullbackGrid()
	results := make([]ConfigResult, 0, len(grid))

	// Two configurations with raw PF above PFCap (10), distinct values.
	// Config 0: raw PF 12 (clamped to 10)
	// Config 1: raw PF 50 (clamped to 10)
	// If Best is selected by clamped value, both would look equal (10).
	// If Best is selected by raw value, config 1 wins (50 > 12).
	for i, p := range grid {
		var trades []backtest.Trade
		switch i {
		case 0:
			// PF = 1200/100 = 12
			for j := 0; j < 12; j++ {
				trades = append(trades, screenTrade(base, j, 100))
			}
			trades = append(trades, screenTrade(base, 12, -100))
		case 1:
			// PF = 5000/100 = 50
			for j := 0; j < 50; j++ {
				trades = append(trades, screenTrade(base, j, 100))
			}
			trades = append(trades, screenTrade(base, 50, -100))
		default:
			// Other configs have no trades
		}
		results = append(results, ConfigResult{Params: p, Trades: trades})
	}
	row := Aggregate("XXXX", "Test", results, split, DefaultScreenOpts())

	// Both raw PF values exceed PFCap of 10 and are clamped.
	if math.Abs(row.BestPF-10) > 1e-9 {
		t.Fatalf("BestPF = %v, want 10 (clamped)", row.BestPF)
	}
	// Best must be config 1, not config 0, proving raw PF (50 vs 12) was used.
	if row.Best != grid[1] {
		t.Fatalf("Best = %+v, want grid[1] (raw PF 50 > 12), proving selection by raw PF", row.Best)
	}
}

// dailyCandlesMSK builds `n` consecutive daily candles starting at 2026-01-05 (a
// Monday) with a constant true range of `rangePct` percent around `close`, in MSK.
func dailyCandlesMSK(n int, closePrice, rangePct float64) []backtest.Candle {
	msk, err := time.LoadLocation("Europe/Moscow")
	if err != nil {
		msk = time.UTC
	}
	base := time.Date(2026, 1, 5, 10, 0, 0, 0, msk) // Monday
	out := make([]backtest.Candle, 0, n)
	half := closePrice * rangePct / 100 / 2
	for i := 0; i < n; i++ {
		out = append(out, backtest.Candle{
			Time:   base.AddDate(0, 0, i),
			Open:   closePrice,
			High:   closePrice + half,
			Low:    closePrice - half,
			Close:  closePrice,
			Volume: 1000,
		})
	}
	return out
}

func TestMeanDailyATRPct(t *testing.T) {
	// 40 days of a constant 2% range: ATR% must converge on 2%.
	got := MeanDailyATRPct(dailyCandlesMSK(40, 100, 2), 14)
	if math.Abs(got-2.0) > 0.2 {
		t.Fatalf("MeanDailyATRPct = %v, want ~2.0", got)
	}
}

func TestMeanDailyATRPctDropsWeekends(t *testing.T) {
	// Weekend bars are 10x narrower. Leaving them in drags the mean down; the
	// strategy itself filters them out (docs/rsi_pullback/strategy.md, section 5)
	// and the screener must measure the same ruler.
	full := dailyCandlesMSK(60, 100, 2)
	msk := full[0].Time.Location()
	for i := range full {
		wd := full[i].Time.In(msk).Weekday()
		if wd == time.Saturday || wd == time.Sunday {
			full[i].High = 100.1
			full[i].Low = 99.9
		}
	}
	got := MeanDailyATRPct(full, 14)
	if math.Abs(got-2.0) > 0.3 {
		t.Fatalf("MeanDailyATRPct = %v, want ~2.0 — weekend bars must be dropped before the ATR", got)
	}
}

func TestMeanDailyATRPctInsufficientData(t *testing.T) {
	if got := MeanDailyATRPct(dailyCandlesMSK(5, 100, 2), 14); got != 0 {
		t.Fatalf("MeanDailyATRPct = %v on 5 bars with period 14, want 0", got)
	}
	if got := MeanDailyATRPct(nil, 14); got != 0 {
		t.Fatalf("MeanDailyATRPct = %v on empty input, want 0", got)
	}
}

func TestScreenTickerFlatSeriesProducesNoSignals(t *testing.T) {
	// A dead-flat series never crosses RSI down through the lower band, so no
	// configuration can enter: the row must land in the "no signals" bucket rather
	// than in the ranking with a fabricated profit factor.
	bars := tinyCandles(600)
	for i := range bars {
		bars[i].Open, bars[i].High, bars[i].Low, bars[i].Close = 100, 100, 100, 100
	}
	daily := dailyCandlesMSK(60, 100, 2)
	split := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)

	row := ScreenTicker("XXXX", "Test", bars, daily, 1, PullbackGrid(), split, DefaultScreenOpts())

	if !row.NoSignals {
		t.Fatalf("NoSignals = false on a flat series, row = %+v", row)
	}
	if row.Bars != len(bars) {
		t.Fatalf("Bars = %d, want %d", row.Bars, len(bars))
	}
	if row.DailyATRPct <= 0 {
		t.Fatalf("DailyATRPct = %v, want the daily series to be measured even with no trades", row.DailyATRPct)
	}
	if row.TurnoverM <= 0 {
		t.Fatalf("TurnoverM = %v, want > 0", row.TurnoverM)
	}
}

func TestScreenTickerRunsTheGridParamsVerbatim(t *testing.T) {
	// The screener compares tickers on ONE grid. UGLD is a REGISTERED ticker whose
	// package carries a calibrated literal (RSIPeriod 6, EMASlow 150, UseVolume 1,
	// UseTrail 1); grading it on that literal instead of the grid config would make
	// its row incomparable with the other 268. This pins the equivalence: one grid
	// config through ScreenTicker must reproduce a direct engine run with that same
	// config, bit for bit.
	bars := tinyCandles(600)
	daily := dailyCandlesMSK(60, 100, 2)
	split := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)
	opts := DefaultScreenOpts()

	cfg := PullbackGrid()[0]
	want := backtest.Run(
		core.NewWithParams("UGLD", cfg),
		bars, daily, nil,
		backtest.Config{InitialCash: opts.Cash, Fraction: opts.Fraction, Commission: opts.Commission, Lot: 1},
	)
	wantPF, wantN := profitFactor(want.Trades)

	row := ScreenTicker("UGLD", "calibrated", bars, daily, 1, []core.Params{cfg}, split, opts)

	gotPF, _ := clampPF(wantPF, opts.PFCap)
	trainWant, _ := splitTrades(want.Trades, split)
	trainPF, trainN := profitFactor(trainWant)
	trainPF, _ = clampPF(trainPF, opts.PFCap)

	if row.PFMed != trainPF {
		t.Fatalf("PFMed = %v, want %v — ScreenTicker must run the grid config, not the ticker's calibrated literal (whole-window PF was %v on %d trades)",
			row.PFMed, trainPF, gotPF, wantN)
	}
	if row.TradesMed != float64(trainN) {
		t.Fatalf("TradesMed = %v, want %v", row.TradesMed, float64(trainN))
	}
}

func TestAggregateHoldoutSignalsSetsNoSignalsFalse(t *testing.T) {
	base := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	split := base.AddDate(0, 0, 100)
	results := make([]ConfigResult, 0, len(PullbackGrid()))

	// All configurations have trades ONLY in holdout (after split).
	// NoSignals should be false because at least one config produced trades.
	for _, p := range PullbackGrid() {
		results = append(results, ConfigResult{Params: p, Trades: []backtest.Trade{
			screenTrade(base, 150, 100),  // holdout only
			screenTrade(base, 151, -100), // holdout only
		}})
	}
	row := Aggregate("XXXX", "Test", results, split, DefaultScreenOpts())

	if row.NoSignals {
		t.Fatal("NoSignals = true, want false when holdout has trades")
	}
	// Train should be empty, so PFMed should be 0.
	if row.PFMed != 0 {
		t.Fatalf("PFMed(train) = %v, want 0 (no train trades)", row.PFMed)
	}
	// Holdout should have PF 1.0 (100 / 100).
	if math.Abs(row.PFMedHO-1.0) > 1e-9 {
		t.Fatalf("PFMedHO = %v, want 1.0", row.PFMedHO)
	}
}
