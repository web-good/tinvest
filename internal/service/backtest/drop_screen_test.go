package backtest

import (
	"math"
	"testing"
	"time"

	"tinvest/internal/domain/backtest"
	"tinvest/internal/enum"
	dropcore "tinvest/internal/service/trading_strategy/drop_bounce/strategy/core"
)

func TestDropGridHour1(t *testing.T) {
	grid, err := DropGrid(enum.Hour1)
	if err != nil {
		t.Fatalf("DropGrid error: %v", err)
	}
	if len(grid) != 16 {
		t.Fatalf("len = %d, want 16", len(grid))
	}
	uniq := make(map[[4]float64]bool)
	for _, p := range grid {
		if p.RSIPeriod != 6 || p.EMAPeriod != 200 || p.StopDailyATR != 1.0 || p.DailyATRPeriod != 14 || p.StartHour != 7 {
			t.Fatalf("pinned fields drifted: %+v", p)
		}
		uniq[[4]float64{p.RSIZone, p.RSICap, p.DaySpentATR, p.RSIExit}] = true
	}
	if len(uniq) != 16 {
		t.Fatalf("%d distinct configurations, want 16", len(uniq))
	}
}

func TestDropGridOrder(t *testing.T) {
	grid, err := DropGrid(enum.Hour1)
	if err != nil {
		t.Fatal(err)
	}
	first, second, last := grid[0], grid[1], grid[15]
	if first.RSIZone != 25 || first.RSICap != 40 || first.DaySpentATR != 0.5 || first.RSIExit != 60 {
		t.Fatalf("grid[0] = %+v, want zone 25 cap 40 day 0.5 exit 60", first)
	}
	if second.RSIExit != 75 || second.DaySpentATR != 0.5 {
		t.Fatalf("grid[1] = %+v, want RSIExit as the innermost axis", second)
	}
	if last.RSIZone != 35 || last.RSICap != 60 || last.DaySpentATR != 1.0 || last.RSIExit != 75 {
		t.Fatalf("grid[15] = %+v, want zone 35 cap 60 day 1.0 exit 75", last)
	}
}

func TestDropGridRejectsOtherTimeframes(t *testing.T) {
	for _, iv := range []enum.Interval{enum.Minutes5, enum.Minutes15, enum.Minutes30, enum.Day1, enum.Hour4} {
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
	grid, _ := DropGrid(enum.Hour1)
	results := []DropConfigResult{
		{Params: grid[0], Trades: []backtest.Trade{
			dropTrade(trainAt(2), 200, "RSI"), dropTrade(trainAt(3), -100, "SL"), dropTrade(holdoutAt(1), 50, "RSI"),
		}},
		{Params: grid[1], Trades: []backtest.Trade{
			dropTrade(trainAt(4), 300, "EOD"), dropTrade(trainAt(5), -100, "SL"), dropTrade(trainAt(6), -100, "SL"),
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
	grid, _ := DropGrid(enum.Hour1)
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
	grid, _ := DropGrid(enum.Hour1)
	row := AggregateDrop("XXXX", "", []DropConfigResult{{Params: grid[3]}, {Params: grid[4]}}, dropSplit, 0, DefaultScreenOpts())
	if !row.NoSignals || row.SilentCfg != 2 || row.TradesMed != 0 {
		t.Fatalf("NoSignals/SilentCfg/TradesMed = %v/%d/%v, want true/2/0", row.NoSignals, row.SilentCfg, row.TradesMed)
	}
	if row.Best != grid[3] {
		t.Fatalf("Best = %+v, want the first configuration when nothing trades", row.Best)
	}
}

func TestAggregateDropCapsWinnersOnly(t *testing.T) {
	grid, _ := DropGrid(enum.Hour1)
	opts := DefaultScreenOpts()
	row := AggregateDrop("GAZP", "", []DropConfigResult{
		{Params: grid[0], Trades: []backtest.Trade{dropTrade(trainAt(2), 100, "RSI")}},
	}, dropSplit, 0, opts)
	if row.PFMed != opts.PFCap || row.Capped != 1 || math.IsInf(row.PFMedStress, 0) {
		t.Fatalf("PFMed/Capped/PFMedStress = %v/%d/%v, want %v/1/finite", row.PFMed, row.Capped, row.PFMedStress, opts.PFCap)
	}
}

// dropFixture is 1h bars 07:00–22:00 MSK on every weekday from 2026-01-12 to 2026-03-31: a slow
// uptrend (+0.03% a bar) with a dip at 10:00–12:00 (−0.25% each) and a rebound at 13:00–15:00
// (+0.3% each). The 12:00 bar has RSI(6) ≈ 20 above EMA(200) with a day range ≈ 0.8 — under the
// gate 0.5 · daily ATR 2 = 1.0 of grid[0] — and the rebound crosses RSI 60, so grid[0] trades
// once a day after the EMA(200) lookback (600 bars) is filled.
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
		for h := 7; h <= 22; h++ {
			o := p
			switch {
			case h >= 10 && h <= 12:
				p *= 0.9975
			case h >= 13 && h <= 15:
				p *= 1.003
			default:
				p *= 1.0003
			}
			out = append(out, backtest.Candle{Time: d.Add(time.Duration(h) * time.Hour),
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
	grid, _ := DropGrid(enum.Hour1)
	cfg := grid[0] // zone 25, cap 40, day 0.5, exit 60

	want := backtest.Run(dropcore.NewWithParams("GAZP", time.Hour, cfg), bars, daily, nil,
		backtest.Config{InitialCash: opts.Cash, Fraction: opts.Fraction, Commission: opts.Commission, Lot: 1, EntryAtNextOpen: true})
	train, _ := splitTrades(want.Trades, dropSplit)
	wantPF, wantN := profitFactor(train)
	if wantN == 0 {
		t.Fatal("fixture produces no train trades: the comparison would be 0 vs 0")
	}
	wantPF, _ = clampPF(wantPF, opts.PFCap)

	row := ScreenDropTicker(DropTickerInput{
		Ticker: "GAZP", Name: "Газпром", Interval: enum.Hour1, Bars: bars, Daily: daily, Lot: 1, MinPriceIncrement: 0.01,
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
	grid, _ := DropGrid(enum.Hour1)
	row := ScreenDropTicker(DropTickerInput{
		Ticker: "XXXX", Interval: enum.Hour1, Bars: bars, Daily: dailyCandlesMSK(100, 100, 2), Lot: 1,
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
