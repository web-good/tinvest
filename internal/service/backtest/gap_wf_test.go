package backtest

import (
	"math"
	"reflect"
	"testing"
	"time"

	"tinvest/internal/domain/backtest"
	"tinvest/internal/service/trading_strategy/gap_fade/strategy/core"
)

func gapAt(s string) time.Time {
	t, err := time.ParseInLocation("2006-01-02 15:04", s, gapLoc)
	if err != nil {
		panic(err)
	}
	return t
}

func gt(ticker, entry string, pct float64) GapTrade {
	e := gapAt(entry)
	return GapTrade{Ticker: ticker, Trade: backtest.Trade{EntryTime: e, ExitTime: e.Add(time.Hour), PnLPct: pct, Reason: "TP"}}
}

func TestGapPF(t *testing.T) {
	if got := GapPF([]GapTrade{gt("A", "2025-01-06 07:00", 0.02), gt("A", "2025-01-07 07:00", -0.01)}); math.Abs(got-2) > 1e-12 {
		t.Fatalf("PF = %v, want 2", got)
	}
	if got := GapPF([]GapTrade{gt("A", "2025-01-06 07:00", 0.02)}); !math.IsInf(got, 1) {
		t.Fatalf("PF without losses = %v, want +Inf", got)
	}
	if got := GapPF(nil); got != 0 {
		t.Fatalf("PF of nothing = %v, want 0", got)
	}
}

func TestDividendExDays(t *testing.T) {
	got := DividendExDays([]time.Time{
		time.Date(2025, 7, 16, 0, 0, 0, 0, time.UTC),  // Wed -> Thu 2025-07-17
		time.Date(2025, 7, 17, 21, 0, 0, 0, time.UTC), // Fri 00:00 MSK -> Mon 2025-07-21
		{}, // missing last buy date -> skipped
	})
	want := map[string]bool{"2025-07-17": true, "2025-07-21": true}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("ex-days = %v, want %v", got, want)
	}
}

func TestDropExDayTrades(t *testing.T) {
	trades := []GapTrade{gt("A", "2025-07-17 07:00", 0.01), gt("A", "2025-07-18 07:00", 0.01), gt("A", "2025-07-17 09:00", -0.01)}
	kept, days := DropExDayTrades(trades, map[string]bool{"2025-07-17": true})
	if len(kept) != 1 || !kept[0].EntryTime.Equal(gapAt("2025-07-18 07:00")) {
		t.Fatalf("kept = %+v", kept)
	}
	if !reflect.DeepEqual(days, []string{"2025-07-17"}) {
		t.Fatalf("dropped days = %v, want one distinct day", days)
	}
}

// Window 2024-01-01..2025-01-01, train 6 / test 3: folds test Jul–Oct and Oct–Jan.
func wfRuns() []GapComboRun {
	return []GapComboRun{
		{Params: "c0", Trades: []GapTrade{
			gt("A", "2024-02-05 07:00", 0.02), gt("B", "2024-03-05 07:00", -0.01), // train fold 1: PF 2, 2 trades
			gt("A", "2024-08-05 07:00", 0.03),  // test fold 1
			gt("A", "2024-11-05 07:00", -0.02), // test fold 2
		}},
		{Params: "c1", Trades: []GapTrade{
			gt("A", "2024-02-06 07:00", 0.05), // train fold 1: PF +Inf but 1 trade < minTrades
			gt("A", "2024-08-06 07:00", -0.04),
			gt("B", "2024-07-08 07:00", 0.06), gt("B", "2024-09-09 07:00", -0.01), // with Aug: fold 2 train PF 1.2 on 3 trades
			gt("B", "2024-11-06 07:00", 0.01), // test fold 2
		}},
	}
}

func TestGapWalkForwardSelectsByPooledTrainPF(t *testing.T) {
	folds, err := RunGapWalkForward(wfRuns(), gapAt("2024-01-01 00:00"), gapAt("2025-01-01 00:00"), 6, 3, 2)
	if err != nil {
		t.Fatal(err)
	}
	if len(folds) != 2 {
		t.Fatalf("folds = %d, want 2", len(folds))
	}
	// Fold 1 trains Jan–Jul: c0 has 2 trades (PF 2); c1 has 1 trade, below min-trades.
	if folds[0].Combo != 0 || folds[0].TrainTrades != 2 || math.Abs(folds[0].TrainPF-2) > 1e-12 {
		t.Fatalf("fold 1 = %+v, want combo 0 with PF 2 on 2 trades", folds[0])
	}
	if len(folds[0].OOS) != 1 || folds[0].OOS[0].PnLPct != 0.03 {
		t.Fatalf("fold 1 OOS = %+v, want c0's Aug trade", folds[0].OOS)
	}
	// Fold 2 trains Apr–Oct: c0 has 1 trade (Aug) — below min-trades; c1 has Aug -0.04,
	// Jul +0.06, Sep -0.01 -> PF 0.06/0.05 = 1.2 on 3 trades.
	if folds[1].Combo != 1 || folds[1].TrainTrades != 3 || math.Abs(folds[1].TrainPF-1.2) > 1e-9 {
		t.Fatalf("fold 2 = %+v, want combo 1 with PF 1.2 on 3 trades", folds[1])
	}
	if len(folds[1].OOS) != 1 || folds[1].OOS[0].Ticker != "B" {
		t.Fatalf("fold 2 OOS = %+v, want c1's Nov trade only", folds[1].OOS)
	}
	if got := len(PooledGapOOS(folds)); got != 2 {
		t.Fatalf("pooled OOS = %d trades, want 2", got)
	}
	if got := SelectedGapCombos(folds); !reflect.DeepEqual(got, map[int]bool{0: true, 1: true}) {
		t.Fatalf("selected = %v", got)
	}
}

func TestGapWalkForwardTieGoesToMoreTradesThenGridOrder(t *testing.T) {
	runs := []GapComboRun{
		{Params: "a", Trades: []GapTrade{gt("A", "2024-02-05 07:00", 0.02), gt("A", "2024-02-06 07:00", -0.01)}},
		{Params: "b", Trades: []GapTrade{gt("A", "2024-02-05 07:00", 0.02), gt("A", "2024-02-06 07:00", -0.01)}},
		{Params: "c", Trades: []GapTrade{gt("A", "2024-02-05 07:00", 0.02), gt("A", "2024-02-06 07:00", 0.02), gt("A", "2024-02-07 07:00", -0.02)}},
	}
	folds, err := RunGapWalkForward(runs, gapAt("2024-01-01 00:00"), gapAt("2024-10-01 00:00"), 6, 3, 2)
	if err != nil {
		t.Fatal(err)
	}
	if folds[0].Combo != 2 {
		t.Fatalf("combo = %d, want 2 (same PF 2, more trades)", folds[0].Combo)
	}
	folds, _ = RunGapWalkForward(runs[:2], gapAt("2024-01-01 00:00"), gapAt("2024-10-01 00:00"), 6, 3, 2)
	if folds[0].Combo != 0 {
		t.Fatalf("combo = %d, want 0 (full tie keeps grid order)", folds[0].Combo)
	}
}

func TestGapWalkForwardFoldWithoutQualifiedCombo(t *testing.T) {
	runs := []GapComboRun{{Params: "a", Trades: []GapTrade{gt("A", "2024-02-05 07:00", 0.02), gt("A", "2024-08-05 07:00", 0.02)}}}
	folds, err := RunGapWalkForward(runs, gapAt("2024-01-01 00:00"), gapAt("2024-10-01 00:00"), 6, 3, 5)
	if err != nil {
		t.Fatal(err)
	}
	if folds[0].Combo != -1 || len(folds[0].OOS) != 0 {
		t.Fatalf("fold = %+v, want Combo -1 and no OOS", folds[0])
	}
	if got, err := ReplayGapFolds(folds, runs); err != nil || len(got) != 0 {
		t.Fatalf("replay = %+v, %v; want none", got, err)
	}
}

func TestReplayGapFoldsUsesTheSameCombosAndWindows(t *testing.T) {
	folds, _ := RunGapWalkForward(wfRuns(), gapAt("2024-01-01 00:00"), gapAt("2025-01-01 00:00"), 6, 3, 2)
	alt := []GapComboRun{
		{Trades: []GapTrade{gt("A", "2024-08-05 07:00", 0.025), gt("A", "2024-11-05 07:00", 0.5)}},
		{Trades: []GapTrade{gt("B", "2024-11-06 07:00", 0.005), gt("B", "2024-08-01 07:00", 0.5)}},
	}
	got, err := ReplayGapFolds(folds, alt)
	if err != nil || len(got) != 2 || got[0].PnLPct != 0.025 || got[1].PnLPct != 0.005 {
		t.Fatalf("replay = %+v, want fold 1 from combo 0 and fold 2 from combo 1, test windows only", got)
	}
}

func TestGapGroupsAndNextDayEOD(t *testing.T) {
	a := gt("A", "2025-03-03 07:00", 0.02)
	b := gt("B", "2025-04-01 07:00", -0.01)
	b.Reason = "EOD"
	b.ExitTime = gapAt("2025-04-02 07:00")
	c := gt("A", "2026-01-05 07:00", -0.01)
	c.Reason = "SL"
	trades := []GapTrade{a, b, c}

	byTicker := GapByTicker(trades)
	if len(byTicker) != 2 || byTicker[0].Key != "A" || byTicker[0].Trades != 2 || math.Abs(byTicker[0].PF-2) > 1e-12 {
		t.Fatalf("by ticker = %+v", byTicker)
	}
	if q := GapByQuarter(trades); len(q) != 3 || q[0].Key != "2025Q1" || q[1].Key != "2025Q2" || q[2].Key != "2026Q1" {
		t.Fatalf("by quarter = %+v", q)
	}
	if y := GapByYear(trades); len(y) != 2 || y[0].Key != "2025" || y[0].Trades != 2 {
		t.Fatalf("by year = %+v", y)
	}
	if r := GapByReason(trades); len(r) != 3 || r[0].Key != "EOD" {
		t.Fatalf("by reason = %+v (sorted by key)", r)
	}
	if n := GapNextDayEOD(trades); n != 1 {
		t.Fatalf("next-day EOD = %d, want 1", n)
	}
	if got := GapTradesFrom(trades, gapAt("2025-04-01 07:00")); len(got) != 2 {
		t.Fatalf("from = %d trades, want 2 (inclusive bound)", len(got))
	}
}

func TestRankGapCombos(t *testing.T) {
	runs := []GapComboRun{
		{Trades: []GapTrade{gt("A", "2025-01-06 07:00", 0.02), gt("B", "2025-01-07 07:00", -0.01)}}, // PF 2
		{Trades: []GapTrade{gt("A", "2025-01-06 07:00", 0.03), gt("B", "2025-01-07 07:00", -0.01)}}, // PF 3
		{Trades: []GapTrade{gt("A", "2025-01-06 07:00", 0.09)}},                                     // PF +Inf, 1 trade
	}
	got := RankGapCombos(runs, 2, 10)
	if len(got) != 2 || got[0].Combo != 1 || got[1].Combo != 0 {
		t.Fatalf("rank = %+v, want combos 1 then 0, combo 2 below min-trades", got)
	}
	if got[0].ProfitableTickers != 1 || got[0].TradedTickers != 2 {
		t.Fatalf("tickers = %d/%d, want 1/2", got[0].ProfitableTickers, got[0].TradedTickers)
	}
	if got := RankGapCombos(runs, 1, 1); len(got) != 1 || got[0].Combo != 2 {
		t.Fatalf("top 1 = %+v, want combo 2 (+Inf)", got)
	}
}

// gateFixture passes all six checks exactly at their thresholds when judged with
// minPooled 7 and minLast12 2: pooled PF 1.5 (plus 0.013+0.01+0.01+0.012 = 0.045, minus 0.03),
// last-12 PF 1.2, 3 of 5 tickers profitable, stressed PF 1.0, 7 pooled and 2 last-12 trades.
func gateFixture() (oos, stressed []GapTrade) {
	oos = []GapTrade{
		gt("A", "2025-01-06 07:00", 0.013), gt("B", "2025-01-07 07:00", 0.01), gt("C", "2025-02-03 07:00", 0.01),
		gt("D", "2025-02-04 07:00", -0.01), gt("E", "2025-02-05 07:00", -0.01),
		gt("A", "2025-10-06 07:00", 0.012), gt("A", "2025-10-07 07:00", -0.01),
	}
	stressed = []GapTrade{gt("A", "2025-01-06 07:00", 0.01), gt("B", "2025-01-07 07:00", -0.01)}
	return oos, stressed
}

func TestGapGateBoundaries(t *testing.T) {
	last12 := gapAt("2025-10-01 00:00")
	oos, stressed := gateFixture()
	v := GapGate(oos, stressed, last12, 7, 2)
	if !v.Pass || len(v.Checks) != 6 {
		t.Fatalf("verdict = %+v, want all six passing at the exact thresholds", v)
	}
	for _, i := range []int{4, 5} {
		if !v.Checks[i].Count {
			t.Fatalf("check %d (%s) must be a count check", i+1, v.Checks[i].Name)
		}
	}
	if v.Checks[4].Value != 7 || v.Checks[4].Threshold != 7 || v.Checks[5].Value != 2 || v.Checks[5].Threshold != 2 {
		t.Fatalf("count checks = %+v / %+v, want 7≥7 and 2≥2", v.Checks[4], v.Checks[5])
	}
	if v := GapGate(nil, nil, last12, 1, 1); v.Pass {
		t.Fatalf("empty OOS must fail: %+v", v)
	}
}

// Each fixture breaks exactly one condition; the verdict must fail on that check alone.
func TestGapGateEachCheckFailsAlone(t *testing.T) {
	last12 := gapAt("2025-10-01 00:00")
	cases := []struct {
		name                 string
		mutate               func(oos, stressed []GapTrade) []GapTrade
		minPooled, minLast12 int
		failing              int
	}{
		{"pooled PF", func(oos, _ []GapTrade) []GapTrade { oos[0].PnLPct = 0.012; return oos }, 7, 2, 0},
		{"last-12 PF", func(oos, _ []GapTrade) []GapTrade {
			// Full window stays at 1.5 (0.014+0.01+0.01+0.011), last 12 months drop to 1.1.
			oos[0].PnLPct, oos[5].PnLPct = 0.014, 0.011
			return oos
		}, 7, 2, 1},
		{"ticker share", func(oos, _ []GapTrade) []GapTrade { oos[2].Ticker = "D"; return oos }, 7, 2, 2},
		{"stressed PF", func(oos, stressed []GapTrade) []GapTrade { stressed[0].PnLPct = 0.0099; return oos }, 7, 2, 3},
		{"pooled trades", func(oos, _ []GapTrade) []GapTrade { return oos }, 8, 2, 4},
		{"last-12 trades", func(oos, _ []GapTrade) []GapTrade { return oos }, 7, 3, 5},
	}
	for _, c := range cases {
		oos, stressed := gateFixture()
		oos = c.mutate(oos, stressed)
		v := GapGate(oos, stressed, last12, c.minPooled, c.minLast12)
		if v.Pass {
			t.Errorf("%s: verdict passed, want fail", c.name)
		}
		for i, ch := range v.Checks {
			if ch.Pass == (i == c.failing) {
				t.Errorf("%s: check %d (%s) Pass = %v, want only check %d failing", c.name, i+1, ch.Name, ch.Pass, c.failing+1)
			}
		}
	}
}

func TestReplayGapFoldsRejectsComboOutsideAlt(t *testing.T) {
	folds := []GapFold{{TestFrom: gapAt("2024-07-01 00:00"), TestTo: gapAt("2024-10-01 00:00"), Combo: 2}}
	if _, err := ReplayGapFolds(folds, []GapComboRun{{}}); err == nil {
		t.Fatal("combo index beyond alt must be an error, not a silent skip")
	}
}

func TestExpandGapGrid(t *testing.T) {
	combos, err := ExpandGapGrid([]byte(`{"phases":[{"name":"all","grid":{"GapATR":[0.3,0.5],"EntryBar":[0,1]}}]}`))
	if err != nil || len(combos) != 4 {
		t.Fatalf("combos = %d, err %v; want 4", len(combos), err)
	}
	if _, ok := combos[0].(core.Params); !ok {
		t.Fatalf("combo type = %T, want core.Params", combos[0])
	}
	if _, err := ExpandGapGrid([]byte(`{"phases":[{"grid":{"GapATR":[0.3]}},{"grid":{"EntryBar":[1]}}]}`)); err == nil {
		t.Fatal("two phases must be rejected: the pooled walk-forward sweeps one cartesian grid")
	}
	if _, err := ExpandGapGrid([]byte(`{"phases":[{"grid":{"GapATR":[]}}]}`)); err == nil {
		t.Fatal("a grid that expands to zero combinations must be rejected")
	}
}

func TestDividendExDaysFromBars(t *testing.T) {
	bars := []time.Time{gapAt("2025-06-11 07:00"), gapAt("2025-06-11 23:30"), gapAt("2025-06-13 07:00"), gapAt("2025-06-13 07:30")}
	got := DividendExDaysFromBars([]time.Time{
		time.Date(2025, 6, 11, 0, 0, 0, 0, time.UTC), // 2025-06-12 is a holiday: the ex-day is the next bar's date
		time.Date(2025, 6, 13, 0, 0, 0, 0, time.UTC), // no bar after it: falls back to the next weekday
		{}, // missing last buy date -> skipped
	}, bars)
	want := map[string]bool{"2025-06-13": true, "2025-06-16": true}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("ex-days = %v, want %v", got, want)
	}
}
