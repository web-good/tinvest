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
