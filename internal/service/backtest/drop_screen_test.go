package backtest

import (
	"math"
	"testing"
	"time"

	"tinvest/internal/domain/backtest"
	"tinvest/internal/enum"
	dropcore "tinvest/internal/service/trading_strategy/drop_bounce/strategy/core"
)

func TestDropGridMinutes30(t *testing.T) {
	grid, err := DropGrid(enum.Minutes30)
	if err != nil {
		t.Fatalf("DropGrid error: %v", err)
	}
	if len(grid) != 8 {
		t.Fatalf("len = %d, want 8", len(grid))
	}
	uniq := make(map[[3]float64]bool)
	for _, p := range grid {
		pinned := dropcore.DefaultParams()
		pinned.RSIPeriod, pinned.RSIExitPeriod, pinned.StopDailyATR = p.RSIPeriod, p.RSIExitPeriod, p.StopDailyATR
		if p != pinned {
			t.Fatalf("pinned fields drifted: %+v", p)
		}
		uniq[[3]float64{float64(p.RSIPeriod), float64(p.RSIExitPeriod), p.StopDailyATR}] = true
	}
	if len(uniq) != 8 {
		t.Fatalf("%d distinct configurations, want 8", len(uniq))
	}
}

func TestDropGridOrder(t *testing.T) {
	grid, err := DropGrid(enum.Minutes30)
	if err != nil {
		t.Fatal(err)
	}
	first, second, last := grid[0], grid[1], grid[7]
	if first.RSIPeriod != 4 || first.RSIExitPeriod != 8 || first.StopDailyATR != 0.5 {
		t.Fatalf("grid[0] = %+v, want RSI 4 exit 8 stop 0.5", first)
	}
	if second.StopDailyATR != 1.0 || second.RSIExitPeriod != 8 {
		t.Fatalf("grid[1] = %+v, want StopDailyATR as the innermost axis", second)
	}
	if last.RSIPeriod != 6 || last.RSIExitPeriod != 12 || last.StopDailyATR != 1.0 {
		t.Fatalf("grid[7] = %+v, want RSI 6 exit 12 stop 1.0", last)
	}
}

func TestDropGridRejectsOtherTimeframes(t *testing.T) {
	for _, iv := range []enum.Interval{enum.Minutes5, enum.Minutes15, enum.Hour1, enum.Day1, enum.Hour4} {
		if _, err := DropGrid(iv); err == nil {
			t.Fatalf("DropGrid(%s) = nil error, want an error", iv)
		}
	}
}

func TestExitShare(t *testing.T) {
	trades := []backtest.Trade{{Reason: "SL"}, {Reason: "RSI"}, {Reason: "SL"}, {Reason: "EOD"}}
	for reason, want := range map[string]float64{"SL": 0.5, "RSI": 0.25, "EOD": 0.25, "TP": 0} {
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
	grid, _ := DropGrid(enum.Minutes30)
	results := []DropConfigResult{
		{Params: grid[0], Trades: []backtest.Trade{
			dropTrade(trainAt(2), 200, "RSI"), dropTrade(trainAt(3), -100, "SL"), dropTrade(holdoutAt(1), 50, "RSI"),
		}},
		{Params: grid[1], Trades: []backtest.Trade{
			dropTrade(trainAt(4), 300, "EOW"), dropTrade(trainAt(5), -100, "SL"), dropTrade(trainAt(6), -100, "SL"),
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
	near("RSIShare", row.RSIShare, 0.25) // 0.5 and 0 over trading configs
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
	grid, _ := DropGrid(enum.Minutes30)
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
	grid, _ := DropGrid(enum.Minutes30)
	row := AggregateDrop("XXXX", "", []DropConfigResult{{Params: grid[3]}, {Params: grid[4]}}, dropSplit, 0, DefaultScreenOpts())
	if !row.NoSignals || row.SilentCfg != 2 || row.TradesMed != 0 {
		t.Fatalf("NoSignals/SilentCfg/TradesMed = %v/%d/%v, want true/2/0", row.NoSignals, row.SilentCfg, row.TradesMed)
	}
	if row.Best != grid[3] {
		t.Fatalf("Best = %+v, want the first configuration when nothing trades", row.Best)
	}
}

func TestAggregateDropCapsWinnersOnly(t *testing.T) {
	grid, _ := DropGrid(enum.Minutes30)
	opts := DefaultScreenOpts()
	row := AggregateDrop("GAZP", "", []DropConfigResult{
		{Params: grid[0], Trades: []backtest.Trade{dropTrade(trainAt(2), 100, "RSI")}},
	}, dropSplit, 0, opts)
	if row.PFMed != opts.PFCap || row.Capped != 1 || math.IsInf(row.PFMedStress, 0) {
		t.Fatalf("PFMed/Capped/PFMedStress = %v/%d/%v, want %v/1/finite", row.PFMed, row.Capped, row.PFMedStress, opts.PFCap)
	}
}

// dropFixture is 30m bars 07:00–23:30 MSK on every weekday from 2026-01-12 to 2026-03-31: a slow
// uptrend (+0.03% a bar) with a dip at 10:00–12:30 (−0.4% each: three hours of −0.8% put the hourly
// RSI(4) deep in the zone above the hourly EMA(200)) and a rebound at 13:00–14:30 (+2% each),
// which turns the 30m MACD histogram green, so grid[0] trades about once a day after the lookback
// (1202 bars) is filled.
func dropFixture() []backtest.Candle {
	msk, err := time.LoadLocation("Europe/Moscow")
	if err != nil {
		msk = time.UTC
	}
	var out []backtest.Candle
	p := 100.0
	for d := time.Date(2026, 1, 12, 0, 0, 0, 0, msk); d.Before(time.Date(2026, 4, 1, 0, 0, 0, 0, msk)); d = d.AddDate(0, 0, 1) {
		if wd := d.Weekday(); wd == time.Saturday || wd == time.Sunday {
			continue
		}
		for m := 7 * 60; m < 24*60; m += 30 {
			o := p
			switch {
			case m >= 10*60 && m < 13*60:
				p *= 0.996
			case m >= 13*60 && m < 15*60:
				p *= 1.02
			default:
				p *= 1.0003
			}
			out = append(out, backtest.Candle{Time: d.Add(time.Duration(m) * time.Minute),
				Open: o, High: max(o, p), Low: min(o, p), Close: p, Volume: 10})
		}
	}
	return out
}

func TestScreenDropTickerRunsTheGridParamsVerbatim(t *testing.T) {
	// The screener must grade every ticker on the grid config, registered or not, so one grid
	// config through ScreenDropTicker must reproduce a direct engine run with that config.
	bars := dropFixture()
	daily := dailyCandlesMSK(100, 100, 2)
	opts := DefaultScreenOpts()
	grid, _ := DropGrid(enum.Minutes30)
	cfg := grid[0] // RSI 4, exit RSI 8, stop 0.5

	want := backtest.Run(dropcore.NewWithParams("GAZP", 30*time.Minute, cfg), bars, daily, nil,
		backtest.Config{InitialCash: opts.Cash, Fraction: opts.Fraction, Commission: opts.Commission, Lot: 1, EntryAtNextOpen: true})
	train, _ := splitTrades(want.Trades, dropSplit)
	wantPF, wantN := profitFactor(train)
	if wantN == 0 {
		t.Fatal("fixture produces no train trades: the comparison would be 0 vs 0")
	}
	wantPF, _ = clampPF(wantPF, opts.PFCap)

	row := ScreenDropTicker(DropTickerInput{
		Ticker: "GAZP", Name: "Газпром", Interval: enum.Minutes30, Bars: bars, Daily: daily, Lot: 1, MinPriceIncrement: 0.01,
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
	grid, _ := DropGrid(enum.Minutes30)
	row := ScreenDropTicker(DropTickerInput{
		Ticker: "XXXX", Interval: enum.Minutes30, Bars: bars, Daily: dailyCandlesMSK(100, 100, 2), Lot: 1,
	}, grid, dropSplit, DefaultScreenOpts())
	if !row.NoSignals {
		t.Fatalf("NoSignals = false on a flat series (RSI pinned at 50, close on the EMA), row = %+v", row)
	}
	if !row.TickUnknown {
		t.Fatal("TickUnknown = false with MinPriceIncrement 0, want true")
	}
}

func TestDropPregateRow(t *testing.T) {
	from := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2025, 1, 10, 0, 0, 0, 0, time.UTC)
	var daily []backtest.Candle
	// Lead-in days before from carry a huge volume; they must not count toward turnover.
	for d := 0; d < 3; d++ {
		daily = append(daily, backtest.Candle{Time: from.AddDate(0, 0, -3+d), Open: 100, High: 101, Low: 99, Close: 100, Volume: 1_000_000_000})
	}
	// In-window days (edges included): 2 lots * 10 per lot... volume 10, lot 1, close 100 -> 1000 RUB/day.
	for _, day := range []time.Time{from, from.AddDate(0, 0, 3), to} {
		daily = append(daily, backtest.Candle{Time: day, Open: 100, High: 101, Low: 99, Close: 100, Volume: 10})
	}
	row := DropPregateRow("ABCD", "Abcd", daily, from, to, 1, 0)
	if !row.Pregated {
		t.Fatal("Pregated = false")
	}
	if want := 1000.0 / 1e6; math.Abs(row.TurnoverM-want) > 1e-12 {
		t.Fatalf("TurnoverM = %v, want %v (lead-in must be excluded)", row.TurnoverM, want)
	}
	if !row.TickUnknown {
		t.Fatal("TickUnknown = false for increment 0")
	}
	if row.Ticker != "ABCD" || row.Name != "Abcd" || row.PFMed != 0 || row.TradesMed != 0 {
		t.Fatalf("unexpected row %+v", row)
	}
	row = DropPregateRow("ABCD", "Abcd", daily, from, to, 1, 0.5)
	if row.TickUnknown || math.Abs(row.TickPct-1.0) > 1e-9 {
		t.Fatalf("TickPct = %v unknown=%v, want 1.0 known", row.TickPct, row.TickUnknown)
	}
}
