package backtest

import (
	"reflect"
	"testing"

	"tinvest/internal/domain/backtest"
	"tinvest/internal/enum"
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
