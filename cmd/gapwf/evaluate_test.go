package main

import (
	"math"
	"strings"
	"testing"
	"time"

	domain "tinvest/internal/domain/backtest"
	svc "tinvest/internal/service/backtest"
	"tinvest/internal/service/trading_strategy/gap_fade/strategy/core"
)

var msk = func() *time.Location {
	loc, err := time.LoadLocation("Europe/Moscow")
	if err != nil {
		return time.UTC
	}
	return loc
}()

func mskDate(s string) time.Time {
	t, err := time.ParseInLocation("2006-01-02", s, msk)
	if err != nil {
		panic(err)
	}
	return t
}

func TestWindowStartGivesFullFoldGrid(t *testing.T) {
	for _, d := range []string{"2026-05-31", "2026-07-31", "2026-12-31", "2026-02-28", "2026-10-15"} {
		to := mskDate(d).Add(15*time.Hour + 4*time.Minute)
		from := windowStart(to, 36)
		folds, err := svc.RunGapWalkForward(nil, from, to, 12, 3, 30)
		if err != nil {
			t.Fatalf("%s: %v", d, err)
		}
		if len(folds) != 8 {
			t.Errorf("%s: folds = %d, want (36-12)/3 = 8 (from %v)", d, len(folds), from)
			continue
		}
		last := folds[len(folds)-1].TestTo
		if lag := to.Sub(last); lag < 0 || lag > maxFoldLag {
			t.Errorf("%s: last TestTo %v is %v before to", d, last, lag)
		}
		if err := checkFoldsReachTo(folds, to); err != nil {
			t.Errorf("%s: %v", d, err)
		}
	}
}

func TestCheckFoldsReachToRejectsStaleGrid(t *testing.T) {
	to := mskDate("2026-05-31")
	folds := []svc.GapFold{{TestTo: mskDate("2026-03-03")}}
	err := checkFoldsReachTo(folds, to)
	if err == nil || !strings.Contains(err.Error(), "2026-03-03") || !strings.Contains(err.Error(), "2026-05-31") {
		t.Fatalf("err = %v, want both dates named", err)
	}
}

// --- synthetic market: flat 30m sessions at 100 with gap-down Wednesdays ---

func flatBars(day time.Time, from int, p float64) []domain.Candle {
	var out []domain.Candle
	for t := day.Add(time.Duration(from) * 30 * time.Minute); t.Before(day.Add(24 * time.Hour)); t = t.Add(30 * time.Minute) {
		out = append(out, domain.Candle{Time: t, Open: p, High: p, Low: p, Close: p})
	}
	return out
}

// marketBars covers every weekday in [from, to) with a 07:00–23:30 session flat at 100. On a gap
// day the first bar opens and closes at 99.7 (0.6 ATR below 100 with ATR 0.5) and the second
// bar touches 100 — a TP of +0.3% gross: positive at 0.05% and 0.1% per side, negative at 0.2%.
func marketBars(from, to time.Time, gapDays map[string]bool) []domain.Candle {
	var out []domain.Candle
	for d := from; d.Before(to); d = d.AddDate(0, 0, 1) {
		if d.Weekday() == time.Saturday || d.Weekday() == time.Sunday {
			continue
		}
		session := flatBars(d, 14, 100) // 07:00 onwards
		if gapDays[d.Format("2006-01-02")] {
			session[0] = domain.Candle{Time: session[0].Time, Open: 99.7, High: 99.7, Low: 99.7, Close: 99.7}
			session[1] = domain.Candle{Time: session[1].Time, Open: 99.8, High: 100, Low: 99.7, Close: 100}
		}
		out = append(out, session...)
	}
	return out
}

// marketDaily is weekday dailies with range 0.5 around 100: a Wilder ATR of exactly 0.5.
func marketDaily(from, to time.Time) []domain.Candle {
	var out []domain.Candle
	for d := from; d.Before(to); d = d.AddDate(0, 0, 1) {
		if d.Weekday() == time.Saturday || d.Weekday() == time.Sunday {
			continue
		}
		out = append(out, domain.Candle{Time: d.Add(10 * time.Hour), Open: 100, High: 100.25, Low: 99.75, Close: 100})
	}
	return out
}

func syntheticTicker() tickerData {
	gaps := map[string]bool{"2026-06-10": true, "2026-07-15": true, "2026-08-12": true, "2026-08-19": true}
	return tickerData{
		ticker: "SYN", lot: 1,
		bars:   marketBars(mskDate("2026-05-18"), mskDate("2026-09-01"), gaps),
		daily:  marketDaily(mskDate("2026-03-02"), mskDate("2026-09-01")),
		exDays: map[string]bool{"2026-08-19": true},
	}
}

func twoCombos() []any {
	p := core.DefaultParams()
	q := core.DefaultParams()
	q.GapATR = 0.7 // the 0.6-ATR gaps are too shallow: no trades
	return []any{p, q}
}

func TestEvaluateOnSyntheticMarket(t *testing.T) {
	cfg := runCfg{trainMonths: 1, testMonths: 1, minTrades: 1, workers: 2, commission: 0.0005}
	from, to := mskDate("2026-06-01"), mskDate("2026-09-01")
	combos := twoCombos()
	res, err := evaluate([]tickerData{syntheticTicker()}, combos, cfg, from, to)
	if err != nil {
		t.Fatal(err)
	}

	// Runs are index-aligned with combos; the ex-day trade (08-19) is dropped.
	if len(res.runs) != 2 || res.runs[0].Params != combos[0] || res.runs[1].Params != combos[1] {
		t.Fatalf("runs not aligned with combos: %+v", res.runs)
	}
	if n := len(res.runs[0].Trades); n != 3 {
		t.Fatalf("combo 0 trades = %d, want 3 (four gap days minus the ex-day)", n)
	}
	for _, tr := range res.runs[0].Trades {
		if tr.EntryTime.In(msk).Format("2006-01-02") == "2026-08-19" {
			t.Fatalf("ex-day trade kept: %+v", tr)
		}
	}
	if len(res.runs[1].Trades) != 0 {
		t.Fatalf("combo 1 trades = %d, want 0", len(res.runs[1].Trades))
	}
	if res.exDays["SYN"] != 1 {
		t.Fatalf("exDays = %v, want SYN: 1", res.exDays)
	}

	// Two folds (Jul and Aug tests), both choose combo 0; pooled OOS = Jul + Aug-12 trades.
	if len(res.folds) != 2 || res.folds[0].Combo != 0 || res.folds[1].Combo != 0 {
		t.Fatalf("folds = %+v", res.folds)
	}
	if !res.last12From.Equal(res.folds[1].TestTo.AddDate(0, -12, 0)) {
		t.Fatalf("last12From = %v, want last TestTo minus 12 months", res.last12From)
	}

	// Sensitivity: base row, then the stress commissions replayed on the selected combo.
	if len(res.sens) != 3 || res.sens[0].Commission != 0.0005 || res.sens[1].Commission != 0.001 || res.sens[2].Commission != 0.002 {
		t.Fatalf("sensitivity rows = %+v", res.sens)
	}
	for _, row := range res.sens {
		if len(row.Trades) != 2 {
			t.Fatalf("row %v: %d trades, want the 2 OOS trades of the selected combo", row.Commission, len(row.Trades))
		}
	}
	if pf := svc.GapPF(res.sens[1].Trades); !math.IsInf(pf, 1) {
		t.Fatalf("0.1%% row PF = %v, want +Inf (still profitable)", pf)
	}

	// Gate 4 is fed by the 0.2% row (PF 0), not the 0.1% row (PF +Inf).
	g4 := res.verdict.Checks[3]
	if g4.Pass || g4.Value != svc.GapPF(res.sens[2].Trades) {
		t.Fatalf("gate 4 = %+v, want the failing 0.2%% row", g4)
	}
	if !res.verdict.Checks[0].Pass || res.verdict.Checks[4].Value != 2 || res.verdict.Checks[5].Pass {
		t.Fatalf("verdict = %+v", res.verdict)
	}
}

func TestRunCombosOnlyRunsSelected(t *testing.T) {
	data := []tickerData{syntheticTicker()}
	combos := []any{core.DefaultParams(), core.DefaultParams()}
	all, _ := runCombos(data, combos, nil, 0.0005, 1)
	if len(all[0].Trades) == 0 || len(all[1].Trades) == 0 {
		t.Fatalf("both combos must trade without a filter: %d/%d", len(all[0].Trades), len(all[1].Trades))
	}
	only, _ := runCombos(data, combos, map[int]bool{1: true}, 0.0005, 1)
	if len(only[0].Trades) != 0 || len(only[1].Trades) != len(all[1].Trades) {
		t.Fatalf("only {1}: %d/%d trades, want 0/%d", len(only[0].Trades), len(only[1].Trades), len(all[1].Trades))
	}
}
