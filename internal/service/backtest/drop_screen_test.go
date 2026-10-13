package backtest

import (
	"math"
	"reflect"
	"testing"
	"time"

	"tinvest/internal/domain/backtest"
	"tinvest/internal/enum"
	dropcore "tinvest/internal/service/trading_strategy/drop_bounce/strategy/core"
)

func TestDropGridPerTimeframe(t *testing.T) {
	cases := []struct {
		interval enum.Interval
		vols     []float64
	}{
		{enum.Minutes15, []float64{0.05, 0.1, 0.2, 0.3}},
		{enum.Minutes30, []float64{0.1, 0.2, 0.3, 0.5}},
		{enum.Hour1, []float64{0.15, 0.25, 0.4, 0.6}},
	}
	for _, c := range cases {
		grid, err := DropGrid(c.interval)
		if err != nil {
			t.Fatalf("%s: DropGrid error: %v", c.interval, err)
		}
		if len(grid) != 32 {
			t.Fatalf("%s: len = %d, want 32", c.interval, len(grid))
		}
		seen := make(map[float64]bool)
		uniq := make(map[[4]float64]bool)
		for _, p := range grid {
			if p.StopDailyATR != 1.0 || p.DailyATRPeriod != 10 || p.VolAvgPeriod != 10 || p.StartHour != 7 {
				t.Fatalf("%s: pinned fields drifted: %+v", c.interval, p)
			}
			seen[p.VolDailyMult] = true
			uniq[[4]float64{p.VolDailyMult, float64(p.RSIPeriod), p.RSIExit, p.GreenVolExit}] = true
		}
		var got []float64
		for _, v := range c.vols {
			if seen[v] {
				got = append(got, v)
			}
		}
		if !reflect.DeepEqual(got, c.vols) || len(seen) != len(c.vols) {
			t.Fatalf("%s: VolDailyMult set = %v, want %v", c.interval, seen, c.vols)
		}
		if len(uniq) != 32 {
			t.Fatalf("%s: %d distinct configurations, want 32", c.interval, len(uniq))
		}
	}
}

func TestDropGridOrder(t *testing.T) {
	grid, err := DropGrid(enum.Minutes15)
	if err != nil {
		t.Fatal(err)
	}
	first, second, last := grid[0], grid[1], grid[31]
	if first.VolDailyMult != 0.05 || first.RSIPeriod != 4 || first.RSIExit != 55 || first.GreenVolExit != 0 {
		t.Fatalf("grid[0] = %+v, want vol 0.05 RSI(4)@55 GV 0", first)
	}
	if second.GreenVolExit != 1.5 || second.RSIExit != 55 {
		t.Fatalf("grid[1] = %+v, want GreenVolExit as the innermost axis", second)
	}
	if last.VolDailyMult != 0.3 || last.RSIPeriod != 6 || last.RSIExit != 75 || last.GreenVolExit != 1.5 {
		t.Fatalf("grid[31] = %+v, want vol 0.3 RSI(6)@75 GV 1.5", last)
	}
}

func TestDropGridRejectsOtherTimeframes(t *testing.T) {
	for _, iv := range []enum.Interval{enum.Minutes5, enum.Day1, enum.Hour4} {
		if _, err := DropGrid(iv); err == nil {
			t.Fatalf("DropGrid(%s) = nil error, want an error", iv)
		}
	}
}

func TestExitShare(t *testing.T) {
	trades := []backtest.Trade{{Reason: "SL"}, {Reason: "RSI"}, {Reason: "SL"}, {Reason: "GREENVOL"}}
	for reason, want := range map[string]float64{"SL": 0.5, "RSI": 0.25, "GREENVOL": 0.25, "EOD": 0} {
		if got := exitShare(trades, reason); got != want {
			t.Fatalf("exitShare(%s) = %v, want %v", reason, got, want)
		}
	}
	if got := exitShare(nil, "SL"); got != 0 {
		t.Fatalf("exitShare(nil) = %v, want 0", got)
	}
}

var dropSplit = time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)

// dropTrade is a trade held two hours at price 100 with 10 shares: the cost stress then removes
// 0.0005·200·10 = 1 of commission plus 2·tick·10 of ticks.
func dropTrade(entry time.Time, pnl float64, reason string) backtest.Trade {
	return backtest.Trade{EntryTime: entry, ExitTime: entry.Add(2 * time.Hour), EntryPrice: 100, ExitPrice: 100,
		Quantity: 10, PnL: pnl, Reason: reason}
}

func trainAt(day int) time.Time { return time.Date(2026, 2, day, 10, 0, 0, 0, time.UTC) }

func holdoutAt(day int) time.Time { return time.Date(2026, 7, day, 10, 0, 0, 0, time.UTC) }

func TestAggregateDropMediansExitsAndSilence(t *testing.T) {
	grid, _ := DropGrid(enum.Minutes15)
	results := []DropConfigResult{
		{Params: grid[0], Trades: []backtest.Trade{
			dropTrade(trainAt(2), 200, "RSI"), dropTrade(trainAt(3), -100, "SL"), dropTrade(holdoutAt(1), 50, "GREENVOL"),
		}},
		{Params: grid[1], Trades: []backtest.Trade{
			dropTrade(trainAt(4), 300, "GREENVOL"), dropTrade(trainAt(5), -100, "SL"), dropTrade(trainAt(6), -100, "SL"),
		}},
		{Params: grid[2]},
	}
	row := AggregateDrop("GAZP", "Газпром", results, dropSplit, 0, DefaultScreenOpts())

	near := func(name string, got, want float64) {
		t.Helper()
		if math.Abs(got-want) > 1e-9 {
			t.Fatalf("%s = %v, want %v", name, got, want)
		}
	}
	near("PFMed", row.PFMed, 1.5)       // PFs 2, 1.5, 0
	near("TradesMed", row.TradesMed, 2) // 2, 3, 0
	near("SLShare", row.SLShare, (0.5+2.0/3)/2)
	near("RSIShare", row.RSIShare, 0.25)            // 0.5 and 0 over trading configs
	near("GreenVolShare", row.GreenVolShare, 1.0/6) // 0 and 1/3
	near("HoldDays", row.HoldDays, 2.0/24)
	near("PFMedHO", row.PFMedHO, 0) // holdout PFs: +Inf→10, 0, 0
	near("TradesMedHO", row.TradesMedHO, 0)
	if row.SilentCfg != 1 || row.NoSignals {
		t.Fatalf("SilentCfg/NoSignals = %d/%v, want 1/false", row.SilentCfg, row.NoSignals)
	}
	if row.Best != grid[0] || row.BestPF != 2 {
		t.Fatalf("Best/BestPF = %+v/%v, want grid[0]/2", row.Best, row.BestPF)
	}
	if row.Capped != 0 {
		t.Fatalf("Capped = %d, want 0: only train PFs count", row.Capped)
	}
}

func TestAggregateDropStressIgnoresHoldout(t *testing.T) {
	grid, _ := DropGrid(enum.Minutes15)
	train := []backtest.Trade{dropTrade(trainAt(2), 200, "RSI"), dropTrade(trainAt(3), -100, "SL")}
	opts := DefaultScreenOpts()

	row := AggregateDrop("T", "Т", []DropConfigResult{{Params: grid[0], Trades: train}}, dropSplit, 1, opts)
	if row.PFMed != 2 {
		t.Fatalf("PFMed = %v, want 2", row.PFMed)
	}
	want := 179.0 / 121.0 // each trade loses 1 commission + 20 ticks
	if math.Abs(row.PFMedStress-want) > 1e-9 {
		t.Fatalf("PFMedStress = %v, want %v", row.PFMedStress, want)
	}

	withHoldout := append(append([]backtest.Trade(nil), train...), dropTrade(holdoutAt(1), -1000, "SL"))
	row2 := AggregateDrop("T", "Т", []DropConfigResult{{Params: grid[0], Trades: withHoldout}}, dropSplit, 1, opts)
	if row2.PFMedStress != row.PFMedStress || row2.TradesMed != row.TradesMed {
		t.Fatalf("holdout moved the ranking key: %v/%v vs %v/%v", row2.PFMedStress, row2.TradesMed, row.PFMedStress, row.TradesMed)
	}
	if row2.PFMedStressHO != 0 || row2.TradesMedHO != 1 {
		t.Fatalf("PFMedStressHO/TradesMedHO = %v/%v, want 0/1", row2.PFMedStressHO, row2.TradesMedHO)
	}
}

func TestAggregateDropAllSilent(t *testing.T) {
	grid, _ := DropGrid(enum.Minutes15)
	row := AggregateDrop("XXXX", "", []DropConfigResult{{Params: grid[3]}, {Params: grid[4]}}, dropSplit, 0, DefaultScreenOpts())
	if !row.NoSignals || row.SilentCfg != 2 || row.TradesMed != 0 {
		t.Fatalf("NoSignals/SilentCfg/TradesMed = %v/%d/%v, want true/2/0", row.NoSignals, row.SilentCfg, row.TradesMed)
	}
	if row.Best != grid[3] {
		t.Fatalf("Best = %+v, want the first configuration when nothing trades", row.Best)
	}
}

func TestAggregateDropCapsWinnersOnly(t *testing.T) {
	grid, _ := DropGrid(enum.Minutes15)
	opts := DefaultScreenOpts()
	row := AggregateDrop("GAZP", "", []DropConfigResult{
		{Params: grid[0], Trades: []backtest.Trade{dropTrade(trainAt(2), 100, "RSI")}},
	}, dropSplit, 0, opts)
	if row.PFMed != opts.PFCap || row.Capped != 1 || math.IsInf(row.PFMedStress, 0) {
		t.Fatalf("PFMed/Capped/PFMedStress = %v/%d/%v, want %v/1/finite", row.PFMed, row.Capped, row.PFMedStress, opts.PFCap)
	}
}

// dropFixture is 15m bars over Monday 2026-03-09 and Tuesday 2026-03-10, 07:00–23:45 MSK, flat at
// 100 with volume 10, except a red bar at Tuesday 10:00 (100 → 99, volume 100 = 0.1 of the 1000
// daily volume) followed by a steady climb that pushes RSI back up. Monday gives the engine the 64
// bars of lookback the strategy asks for before the signal.
func dropFixture() []backtest.Candle {
	msk, err := time.LoadLocation("Europe/Moscow")
	if err != nil {
		msk = time.UTC
	}
	var out []backtest.Candle
	signal := time.Date(2026, 3, 10, 10, 0, 0, 0, msk)
	for _, day := range []int{9, 10} {
		for m := 7 * 60; m < 24*60; m += 15 {
			ts := time.Date(2026, 3, day, 0, 0, 0, 0, msk).Add(time.Duration(m) * time.Minute)
			c := backtest.Candle{Time: ts, Open: 100, High: 100.1, Low: 99.9, Close: 100, Volume: 10}
			switch {
			case ts.Equal(signal):
				c = backtest.Candle{Time: ts, Open: 100, High: 100.05, Low: 98.95, Close: 99, Volume: 100}
			case ts.After(signal):
				k := float64(ts.Sub(signal) / (15 * time.Minute))
				open := 99 + 0.2*(k-1)
				c = backtest.Candle{Time: ts, Open: open, High: open + 0.25, Low: open - 0.05, Close: open + 0.2, Volume: 10}
			}
			out = append(out, c)
		}
	}
	return out
}

func TestScreenDropTickerRunsTheGridParamsVerbatim(t *testing.T) {
	// GAZP is a REGISTERED drop_bounce ticker with calibrated literals; the screener must grade it
	// on the grid config like every other ticker, so one grid config through ScreenDropTicker must
	// reproduce a direct engine run with that config.
	bars := dropFixture()
	daily := dailyCandlesMSK(60, 100, 2)
	opts := DefaultScreenOpts()
	grid, _ := DropGrid(enum.Minutes15)
	cfg := grid[0] // vol 0.05, RSI(4)@55, GREENVOL off

	want := backtest.Run(dropcore.NewWithParams("GAZP", 15*time.Minute, cfg), bars, daily, nil,
		backtest.Config{InitialCash: opts.Cash, Fraction: opts.Fraction, Commission: opts.Commission, Lot: 1, EntryAtNextOpen: true})
	train, _ := splitTrades(want.Trades, dropSplit)
	wantPF, wantN := profitFactor(train)
	if wantN == 0 {
		t.Fatal("fixture produces no train trades: the comparison would be 0 vs 0")
	}
	wantPF, _ = clampPF(wantPF, opts.PFCap)

	row := ScreenDropTicker(DropTickerInput{
		Ticker: "GAZP", Name: "Газпром", Interval: enum.Minutes15, Bars: bars, Daily: daily, Lot: 1, MinPriceIncrement: 0.01,
	}, []dropcore.Params{cfg}, dropSplit, opts)

	if row.PFMed != wantPF || row.TradesMed != float64(wantN) {
		t.Fatalf("PFMed/TradesMed = %v/%v, want %v/%d — ScreenDropTicker must run the grid config", row.PFMed, row.TradesMed, wantPF, wantN)
	}
	if row.Bars != len(bars) || row.TurnoverM <= 0 || row.DailyATRPct <= 0 {
		t.Fatalf("Bars/TurnoverM/DailyATRPct = %d/%v/%v, want %d/>0/>0", row.Bars, row.TurnoverM, row.DailyATRPct, len(bars))
	}
	if row.TickUnknown || row.TickPct <= 0 {
		t.Fatalf("TickPct/TickUnknown = %v/%v, want >0/false", row.TickPct, row.TickUnknown)
	}
}

func TestScreenDropTickerFlatSeriesProducesNoSignals(t *testing.T) {
	bars := dropFixture()
	for i := range bars {
		bars[i].Open, bars[i].High, bars[i].Low, bars[i].Close = 100, 100, 100, 100
	}
	grid, _ := DropGrid(enum.Minutes15)
	row := ScreenDropTicker(DropTickerInput{
		Ticker: "XXXX", Interval: enum.Minutes15, Bars: bars, Daily: dailyCandlesMSK(60, 100, 2), Lot: 1,
	}, grid, dropSplit, DefaultScreenOpts())
	if !row.NoSignals {
		t.Fatalf("NoSignals = false on a flat series (no red bar), row = %+v", row)
	}
	if !row.TickUnknown {
		t.Fatal("TickUnknown = false with MinPriceIncrement 0, want true")
	}
}
