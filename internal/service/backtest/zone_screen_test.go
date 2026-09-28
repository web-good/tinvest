package backtest

import (
	"math"
	"reflect"
	"testing"
	"time"

	"tinvest/internal/domain/backtest"
	zonecore "tinvest/internal/service/trading_strategy/rsi_zone/strategy/core"
)

func TestZoneGridHas24UniqueConfigsInOrder(t *testing.T) {
	grid := ZoneGrid()
	if len(grid) != 24 {
		t.Fatalf("grid size = %d, want 24 (2 RSIPeriod x 2 RSILower x 3 EMAPeriod x 2 RSIUpper)", len(grid))
	}
	seen := make(map[zonecore.Params]bool, len(grid))
	for _, p := range grid {
		if seen[p] {
			t.Fatalf("duplicate config in grid: %+v", p)
		}
		seen[p] = true
	}
	// Deterministic order: RSIPeriod -> RSILower -> EMAPeriod -> RSIUpper.
	first := zonecore.Params{RSIPeriod: 4, RSILower: 15, RSIUpper: 65, EMAPeriod: 50, DailyATRPeriod: 14, StopDailyATR: 1.0}
	second := zonecore.Params{RSIPeriod: 4, RSILower: 15, RSIUpper: 75, EMAPeriod: 50, DailyATRPeriod: 14, StopDailyATR: 1.0}
	last := zonecore.Params{RSIPeriod: 6, RSILower: 25, RSIUpper: 75, EMAPeriod: 200, DailyATRPeriod: 14, StopDailyATR: 1.0}
	if grid[0] != first || grid[1] != second || grid[23] != last {
		t.Fatalf("grid order = %+v, %+v ... %+v", grid[0], grid[1], grid[23])
	}
}

func TestZoneGridSweepsOnlyTheFourAxes(t *testing.T) {
	periods, lowers, emas, uppers := map[int]bool{}, map[float64]bool{}, map[int]bool{}, map[float64]bool{}
	for _, p := range ZoneGrid() {
		periods[p.RSIPeriod], lowers[p.RSILower], emas[p.EMAPeriod], uppers[p.RSIUpper] = true, true, true, true
		// The stop is a property of a tuned point, not of the instrument.
		if p.StopDailyATR != 1.0 || p.DailyATRPeriod != 14 {
			t.Fatalf("stop/ATR = %v/%d, want pinned 1.0/14", p.StopDailyATR, p.DailyATRPeriod)
		}
	}
	if !reflect.DeepEqual(emas, map[int]bool{50: true, 100: true, 200: true}) {
		t.Fatalf("EMA axis = %v, want {50,100,200} — EMA 50 is the lead lever on DOMRF and AFKS", emas)
	}
	if len(periods) != 2 || len(lowers) != 2 || len(uppers) != 2 {
		t.Fatalf("axes = %d/%d/%d, want 2/2/2", len(periods), len(lowers), len(uppers))
	}
}

func TestStressTradesAddsCommissionAndTwoTicks(t *testing.T) {
	trades := []backtest.Trade{{EntryPrice: 100, ExitPrice: 110, Quantity: 10, PnL: 99}}
	got := stressTrades(trades, 0.0005, 0.1)
	// extra = 0.0005*(100+110)*10 + 2*0.1*10 = 1.05 + 2 = 3.05
	if want := 99 - 3.05; math.Abs(got[0].PnL-want) > 1e-9 {
		t.Fatalf("stressed PnL = %v, want %v", got[0].PnL, want)
	}
	if trades[0].PnL != 99 {
		t.Fatalf("input mutated: PnL = %v, want 99", trades[0].PnL)
	}
}

func TestStressTradesWithoutTick(t *testing.T) {
	// Unknown tick (0) or a nonsensical negative one stresses by commission only.
	for _, tick := range []float64{0, -1} {
		got := stressTrades([]backtest.Trade{{EntryPrice: 100, ExitPrice: 100, Quantity: 1, PnL: 0}}, 0.001, tick)
		if math.Abs(got[0].PnL-(-0.2)) > 1e-9 {
			t.Fatalf("tick %v: stressed PnL = %v, want -0.2", tick, got[0].PnL)
		}
	}
	if got := stressTrades(nil, 0.001, 0.1); got != nil {
		t.Fatalf("stressTrades(nil) = %v, want nil", got)
	}
}

func TestTickPct(t *testing.T) {
	pct, unknown := tickPct(0.1, 50) // KMAZ-shaped: two ticks = 0.4% of price
	if unknown || math.Abs(pct-0.4) > 1e-9 {
		t.Fatalf("tickPct(0.1, 50) = %v,%v, want 0.4,false", pct, unknown)
	}
}

func TestTickPctUnknown(t *testing.T) {
	for _, c := range []struct{ tick, close float64 }{{0, 50}, {0.1, 0}, {-0.1, 50}} {
		pct, unknown := tickPct(c.tick, c.close)
		if pct != 0 || !unknown {
			t.Fatalf("tickPct(%v, %v) = %v,%v, want 0,true", c.tick, c.close, pct, unknown)
		}
	}
}

func TestHalfLabelUsesMoscowTime(t *testing.T) {
	// 21:30 UTC on June 30 is 00:30 MSK on July 1: second half, not first.
	if got := halfLabel(time.Date(2026, 6, 30, 21, 30, 0, 0, time.UTC)); got != "2026H2" {
		t.Fatalf("halfLabel = %q, want 2026H2", got)
	}
	if got := halfLabel(time.Date(2026, 6, 30, 20, 30, 0, 0, time.UTC)); got != "2026H1" {
		t.Fatalf("halfLabel = %q, want 2026H1", got)
	}
	if got := halfLabel(time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC)); got != "2025H1" {
		t.Fatalf("halfLabel = %q, want 2025H1", got)
	}
}

func TestHalfYearResultsGroupsAndSorts(t *testing.T) {
	trades := []backtest.Trade{
		{EntryTime: time.Date(2026, 2, 1, 10, 0, 0, 0, time.UTC), PnL: 5},
		{EntryTime: time.Date(2025, 8, 1, 10, 0, 0, 0, time.UTC), PnL: -2},
		{EntryTime: time.Date(2026, 3, 1, 10, 0, 0, 0, time.UTC), PnL: 1},
	}
	got := halfYearResults(trades)
	want := []HalfResult{{Label: "2025H2", PnL: -2, Trades: 1}, {Label: "2026H1", PnL: 6, Trades: 2}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("halfYearResults = %+v, want %+v", got, want)
	}
}

func TestHalvesStats(t *testing.T) {
	pos, top, ok := halvesStats([]HalfResult{{PnL: 3}, {PnL: -1}, {PnL: 1}, {PnL: 0}})
	if !ok || pos != 0.5 || top != 0.75 {
		t.Fatalf("halvesStats = %v,%v,%v, want 0.5,0.75,true", pos, top, ok)
	}
	pos, top, ok = halvesStats([]HalfResult{{PnL: -3}, {PnL: 0}})
	if ok || pos != 0 || top != 0 {
		t.Fatalf("no profitable half: halvesStats = %v,%v,%v, want 0,0,false", pos, top, ok)
	}
	if pos, top, ok = halvesStats(nil); ok || pos != 0 || top != 0 {
		t.Fatalf("empty: halvesStats = %v,%v,%v, want 0,0,false", pos, top, ok)
	}
}

func TestSLShareAndHoldDays(t *testing.T) {
	base := time.Date(2026, 3, 2, 10, 0, 0, 0, time.UTC)
	trades := []backtest.Trade{
		{Reason: "SL", EntryTime: base, ExitTime: base.Add(24 * time.Hour)},
		{Reason: "RSI", EntryTime: base, ExitTime: base.Add(72 * time.Hour)},
		{Reason: "RSI", EntryTime: base, ExitTime: base.Add(48 * time.Hour)},
		{Reason: "SL", EntryTime: base, ExitTime: base.Add(12 * time.Hour)},
	}
	if got := slShare(trades); got != 0.5 {
		t.Fatalf("slShare = %v, want 0.5", got)
	}
	// days: 1, 3, 2, 0.5 -> median 1.5
	if got := medianHoldDays(trades); got != 1.5 {
		t.Fatalf("medianHoldDays = %v, want 1.5", got)
	}
	if slShare(nil) != 0 || medianHoldDays(nil) != 0 {
		t.Fatal("empty trades must give 0 for both metrics")
	}
}

// zoneTrade is a one-lot trade entered at `entry` with the given PnL and exit reason.
func zoneTrade(entry time.Time, pnl float64, reason string) backtest.Trade {
	return backtest.Trade{
		EntryTime: entry, ExitTime: entry.Add(48 * time.Hour),
		EntryPrice: 100, ExitPrice: 100 + pnl, Quantity: 1, PnL: pnl, Reason: reason,
	}
}

func TestAggregateZoneRanksByStressedMedian(t *testing.T) {
	// Two configurations with the same raw PF 2.0: the stress (commission 0.01 per
	// side, tick 1) knocks ~4 off every 1-share trade at price ~100, turning +4/-2 into
	// -0.04/-5.98 -> no gross profit, stressed PF 0. PFMed stays 2.
	d := time.Date(2025, 3, 3, 10, 0, 0, 0, time.UTC)
	trades := []backtest.Trade{zoneTrade(d, 4, "RSI"), zoneTrade(d.AddDate(0, 0, 7), -2, "SL")}
	grid := ZoneGrid()[:2]
	results := []ZoneConfigResult{{Params: grid[0], Trades: trades}, {Params: grid[1], Trades: trades}}
	opts := DefaultScreenOpts()
	opts.Commission = 0.01

	row := AggregateZone("XXXX", "Test", results, time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC), 1, opts)

	if row.PFMed != 2 {
		t.Fatalf("PFMed = %v, want 2", row.PFMed)
	}
	if row.PFMedStress != 0 {
		t.Fatalf("PFMedStress = %v, want 0 (stress wipes the edge)", row.PFMedStress)
	}
}

func TestAggregateZoneDetailMetrics(t *testing.T) {
	// Config A trades in 2025H1 (+3, +1) and 2025H2 (-1); config B is silent.
	d1 := time.Date(2025, 2, 3, 10, 0, 0, 0, time.UTC)
	d2 := time.Date(2025, 9, 1, 10, 0, 0, 0, time.UTC)
	grid := ZoneGrid()
	results := []ZoneConfigResult{
		{Params: grid[0], Trades: []backtest.Trade{zoneTrade(d1, 3, "RSI"), zoneTrade(d1.AddDate(0, 0, 5), 1, "RSI"), zoneTrade(d2, -1, "SL")}},
		{Params: grid[1]},
	}
	row := AggregateZone("XXXX", "Test", results, time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC), 0, DefaultScreenOpts())

	if row.SilentCfg != 1 {
		t.Fatalf("SilentCfg = %d, want 1", row.SilentCfg)
	}
	// Silent configs stay out of the detail medians: they have no halves to judge.
	if row.HalvesPos != 0.5 {
		t.Fatalf("HalvesPos = %v, want 0.5 (one of two halves profitable)", row.HalvesPos)
	}
	if row.TopHalf != 1 {
		t.Fatalf("TopHalf = %v, want 1 (all profit from 2025H1)", row.TopHalf)
	}
	if math.Abs(row.SLShare-1.0/3) > 1e-9 {
		t.Fatalf("SLShare = %v, want 1/3", row.SLShare)
	}
	if row.HoldDays != 2 {
		t.Fatalf("HoldDays = %v, want 2", row.HoldDays)
	}
	if row.Best != grid[0] {
		t.Fatalf("Best = %+v, want grid[0]", row.Best)
	}
	want := []HalfResult{{Label: "2025H1", PnL: 4, Trades: 2}, {Label: "2025H2", PnL: -1, Trades: 1}}
	if !reflect.DeepEqual(row.BestHalves, want) {
		t.Fatalf("BestHalves = %+v, want %+v", row.BestHalves, want)
	}
}

func TestAggregateZoneAllSilentStillPicksRealBest(t *testing.T) {
	grid := ZoneGrid()
	results := []ZoneConfigResult{{Params: grid[0]}, {Params: grid[1]}}
	row := AggregateZone("XXXX", "Test", results, time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC), 0.1, DefaultScreenOpts())

	if !row.NoSignals || row.SilentCfg != 2 {
		t.Fatalf("NoSignals/SilentCfg = %v/%d, want true/2", row.NoSignals, row.SilentCfg)
	}
	if row.HalvesPos != 0 || row.TopHalf != 0 || row.SLShare != 0 || row.HoldDays != 0 {
		t.Fatalf("detail metrics = %v/%v/%v/%v, want all 0", row.HalvesPos, row.TopHalf, row.SLShare, row.HoldDays)
	}
	if row.Best != grid[0] {
		t.Fatalf("Best = %+v, want grid[0] — a zero Params would render as a fake configuration", row.Best)
	}
}

func TestAggregateZoneHoldoutOnlyTradesAreSignals(t *testing.T) {
	split := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	grid := ZoneGrid()
	results := []ZoneConfigResult{{Params: grid[0], Trades: []backtest.Trade{
		zoneTrade(split.AddDate(0, 1, 0), 2, "RSI"), zoneTrade(split.AddDate(0, 2, 0), -1, "SL"),
	}}}
	row := AggregateZone("XXXX", "Test", results, split, 0, DefaultScreenOpts())

	if row.NoSignals {
		t.Fatal("NoSignals = true, want false: the holdout traded")
	}
	if row.PFMedHO != 2 || row.TradesMedHO != 2 {
		t.Fatalf("holdout PF/trades = %v/%v, want 2/2", row.PFMedHO, row.TradesMedHO)
	}
	if row.PFMedStressHO >= row.PFMedHO {
		t.Fatalf("PFMedStressHO = %v, want below raw holdout PF %v", row.PFMedStressHO, row.PFMedHO)
	}
}

// zoneFixtureCandles builds n weekday MSK 30-minute bars the rsi_zone grid trades: a
// +0.3%/bar uptrend (close stays above EMA 50) broken every 60 bars by three -1% bars,
// which drive RSI(4) from ~100 to ~18 — a cross below 25 — before the trend resumes and
// RSI crosses back above 65/75.
func zoneFixtureCandles(n int) []backtest.Candle {
	const (
		driftPct = 0.003
		dipPct   = -0.01
		dipEvery = 60
		dipLen   = 3
	)
	t := time.Date(2026, 2, 2, 0, 0, 0, 0, screenMSK) // Monday
	out := make([]backtest.Candle, 0, n)
	price := 100.0
	for i := 0; len(out) < n; {
		if t.Weekday() == time.Saturday || t.Weekday() == time.Sunday {
			t = t.Add(30 * time.Minute)
			continue
		}
		pct := driftPct
		if i%(dipEvery+dipLen) >= dipEvery {
			pct = dipPct
		}
		open, closeP := price, price*(1+pct)
		out = append(out, backtest.Candle{
			Time: t, Open: open, High: max(open, closeP), Low: min(open, closeP), Close: closeP, Volume: 10000,
		})
		price = closeP
		t = t.Add(30 * time.Minute)
		i++
	}
	return out
}

func TestScreenZoneTickerRunsTheGridParamsVerbatim(t *testing.T) {
	// AFKS is a REGISTERED rsi_zone ticker with a calibrated literal (RSI 4, 25/75,
	// EMA 50, stop 1.0). The screener must grade it on the grid config like every other
	// ticker. One grid config through ScreenZoneTicker must reproduce a direct engine
	// run with that config; the fixture is built to trade so the comparison is not
	// 0 vs 0. On this fixture every trade may win, which caps PF at -pf-cap on BOTH
	// paths — so the test also compares the summed PnL (via BestHalves), which the
	// RSIUpper 65 vs 75 difference does move.
	bars := zoneFixtureCandles(1200)
	daily := dailyCandlesMSK(150, 100, 2)
	split := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)
	opts := DefaultScreenOpts()
	cfg := ZoneGrid()[10] // RSI 4, 25/65, EMA 200: RSI(4) crosses 25 on the fixture; exit 65 differs from the literal's 75
	if cfg.RSIPeriod != 4 || cfg.RSILower != 25 || cfg.RSIUpper != 65 || cfg.EMAPeriod != 200 {
		t.Fatalf("ZoneGrid()[10] = %+v, want RSI 4 25/65 EMA 200 — grid order changed", cfg)
	}

	want := backtest.Run(zonecore.NewWithParams("AFKS", cfg), bars, daily, nil,
		backtest.Config{InitialCash: opts.Cash, Fraction: opts.Fraction, Commission: opts.Commission, Lot: 1})
	train, _ := splitTrades(want.Trades, split)
	wantPF, wantN := profitFactor(train)
	if wantN == 0 {
		t.Fatal("fixture produces no train trades: the test cannot tell the grid config from the registered literal")
	}
	wantPF, _ = clampPF(wantPF, opts.PFCap)

	row := ScreenZoneTicker(ZoneTickerInput{
		Ticker: "AFKS", Name: "calibrated", Bars: bars, Daily: daily, Lot: 1, MinPriceIncrement: 0.001,
	}, []zonecore.Params{cfg}, split, opts)

	if row.PFMed != wantPF || row.TradesMed != float64(wantN) {
		t.Fatalf("PFMed/TradesMed = %v/%v, want %v/%d — ScreenZoneTicker must run the grid config", row.PFMed, row.TradesMed, wantPF, wantN)
	}
	if want := halfYearResults(train); !reflect.DeepEqual(row.BestHalves, want) {
		t.Fatalf("BestHalves = %+v, want %+v — ScreenZoneTicker must run the grid config, not the registered literal", row.BestHalves, want)
	}
	if row.Bars != len(bars) || row.TurnoverM <= 0 || row.DailyATRPct <= 0 {
		t.Fatalf("Bars/TurnoverM/DailyATRPct = %d/%v/%v, want %d/>0/>0", row.Bars, row.TurnoverM, row.DailyATRPct, len(bars))
	}
	if row.TickUnknown || row.TickPct <= 0 {
		t.Fatalf("TickPct/TickUnknown = %v/%v, want >0/false", row.TickPct, row.TickUnknown)
	}
	if row.AboveEMA50 <= 0.5 {
		t.Fatalf("AboveEMA50 = %v, want most of an uptrend above EMA 50", row.AboveEMA50)
	}
}

func TestScreenZoneTickerFlatSeriesProducesNoSignals(t *testing.T) {
	bars := tinyCandles(600)
	for i := range bars {
		bars[i].Open, bars[i].High, bars[i].Low, bars[i].Close = 100, 100, 100, 100
	}
	row := ScreenZoneTicker(ZoneTickerInput{
		Ticker: "XXXX", Name: "Test", Bars: bars, Daily: dailyCandlesMSK(60, 100, 2), Lot: 1,
	}, ZoneGrid(), time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC), DefaultScreenOpts())

	if !row.NoSignals {
		t.Fatalf("NoSignals = false on a flat series, row = %+v", row)
	}
	if !row.TickUnknown {
		t.Fatal("TickUnknown = false with MinPriceIncrement 0, want true")
	}
}

func TestAboveEMAShareSkipsWarmup(t *testing.T) {
	// Period 3 on closes 1..6: EMA is zero on bars 0 and 1 (warm-up) and excluded.
	// Warmed EMA = 2, 3, 4, 5 on bars 2..5 against closes 3..6: every warmed bar is
	// above its EMA, so 4/4 — and the two warm-up bars must not dilute that to 4/6.
	bars := make([]backtest.Candle, 6)
	for i := range bars {
		bars[i].Close = float64(i + 1)
	}
	if got := aboveEMAShare(bars, 3); got != 1 {
		t.Fatalf("aboveEMAShare(rising) = %v, want 1", got)
	}
	// Falling series: every warmed bar is below its EMA.
	for i := range bars {
		bars[i].Close = float64(10 - i)
	}
	if got := aboveEMAShare(bars, 3); got != 0 {
		t.Fatalf("aboveEMAShare(falling) = %v, want 0", got)
	}
	if aboveEMAShare(bars[:2], 3) != 0 || aboveEMAShare(bars, 0) != 0 {
		t.Fatal("too-short series or non-positive period must give 0")
	}
}
