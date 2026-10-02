package backtest

import (
	"math"
	"reflect"
	"strings"
	"testing"
	"time"

	"tinvest/internal/domain/backtest"
)

func gsDate(s string) time.Time {
	t, err := time.ParseInLocation("2006-01-02", s, gapLoc)
	if err != nil {
		panic(err)
	}
	return t
}

// gsBar is a flat 30m bar starting at hh:mm MSK on day.
func gsBar(day string, hh, mm int, price float64, vol int64) backtest.Candle {
	t := gsDate(day).Add(time.Duration(hh)*time.Hour + time.Duration(mm)*time.Minute)
	return backtest.Candle{Time: t, Open: price, High: price, Low: price, Close: price, Volume: vol}
}

func TestGapMorningStats(t *testing.T) {
	bars := []backtest.Candle{
		// Mon 2026-06-01: auction bar 06:30 is ignored, the 07:00 bar is 1000 lots·10·100 = 1 млн ₽.
		gsBar("2026-06-01", 6, 30, 100, 99999),
		gsBar("2026-06-01", 7, 0, 100, 1000),
		gsBar("2026-06-01", 7, 0, 100, 1000), // duplicate 07:00 bar from a cache overlap: counted once
		// Tue: session starts at 07:30 — a weekday without a morning bar.
		gsBar("2026-06-02", 7, 30, 100, 500),
		// Wed: 07:00 bar of 3 млн ₽.
		gsBar("2026-06-03", 7, 0, 100, 3000),
		// Sat: a weekend session is neither a weekday nor a morning bar.
		gsBar("2026-06-06", 7, 0, 100, 777777),
	}
	days, turnover := GapMorningStats(bars, 10)
	if math.Abs(days-2.0/3.0) > 1e-9 {
		t.Errorf("days = %v, want 2/3", days)
	}
	if math.Abs(turnover-2.0) > 1e-9 {
		t.Errorf("turnover = %v, want median(1, 3) = 2", turnover)
	}
	if d, m := GapMorningStats(nil, 10); d != 0 || m != 0 {
		t.Errorf("empty: %v %v", d, m)
	}
}

func passingRow() GapScreenRow {
	return GapScreenRow{Ticker: "AAA", MorningDays: 0.95, MorningTurnoverM: 6, TurnoverM: 60, TickPct: 0.1}
}

func TestApplyGapGatesPassAndBoundaries(t *testing.T) {
	opts := DefaultGapScreenOptions()
	if got := ApplyGapGates(passingRow(), opts); !got.Pass || len(got.Reasons) != 0 {
		t.Fatalf("passing row rejected: %+v", got)
	}
	edge := GapScreenRow{Ticker: "EDGE", MorningDays: 0.9, MorningTurnoverM: 5, TurnoverM: 50, TickPct: 0.2}
	if got := ApplyGapGates(edge, opts); !got.Pass {
		t.Fatalf("equality must pass: %+v", got.Reasons)
	}
	unknownTick := passingRow()
	unknownTick.TickPct = 0
	if got := ApplyGapGates(unknownTick, opts); !got.Pass {
		t.Fatalf("unknown tick must not fail: %+v", got.Reasons)
	}
	off := passingRow()
	off.TickPct = 5
	opts.MaxTickPct = 0
	if got := ApplyGapGates(off, opts); !got.Pass {
		t.Fatalf("-max-tick-pct 0 must disable the gate: %+v", got.Reasons)
	}
}

func TestApplyGapGatesEachFailsAlone(t *testing.T) {
	cases := map[string]struct {
		mut  func(*GapScreenRow)
		want string
	}{
		"morning days":     {func(r *GapScreenRow) { r.MorningDays = 0.5 }, "бар 07:00 в 50% будних дней < 90%"},
		"morning turnover": {func(r *GapScreenRow) { r.MorningTurnoverM = 1.24 }, "утренний оборот 1.2 млн ₽ < 5.0"},
		"tick":             {func(r *GapScreenRow) { r.TickPct = 0.31 }, "два шага 0.31% > 0.20%"},
		"daily turnover":   {func(r *GapScreenRow) { r.TurnoverM = 12 }, "дневной оборот 12 млн ₽ < 50"},
	}
	for name, c := range cases {
		r := passingRow()
		c.mut(&r)
		got := ApplyGapGates(r, DefaultGapScreenOptions())
		if got.Pass || !reflect.DeepEqual(got.Reasons, []string{c.want}) {
			t.Errorf("%s: pass=%v reasons=%q, want [%q]", name, got.Pass, got.Reasons, c.want)
		}
	}
}

func TestScreenGapWiresMetrics(t *testing.T) {
	var bars []backtest.Candle
	for _, d := range []string{"2026-06-01", "2026-06-02"} {
		bars = append(bars, gsBar(d, 7, 0, 200, 5000), gsBar(d, 7, 30, 200, 15000))
	}
	row := ScreenGap(GapScreenInput{
		Ticker: "BBB", Name: "Бэ", Bars: bars, Lot: 10, MinPriceIncrement: 0.1, SignalsPerYear: 3.5,
	}, DefaultGapScreenOptions())
	if row.Ticker != "BBB" || row.Name != "Бэ" || row.SignalsPerYear != 3.5 {
		t.Fatalf("identity not carried: %+v", row)
	}
	if row.MorningDays != 1 || math.Abs(row.MorningTurnoverM-10) > 1e-9 {
		t.Errorf("morning = %v / %v, want 1 / 10", row.MorningDays, row.MorningTurnoverM)
	}
	if math.Abs(row.TurnoverM-40) > 1e-9 {
		t.Errorf("turnover = %v, want 40", row.TurnoverM)
	}
	if math.Abs(row.TickPct-0.1) > 1e-9 {
		t.Errorf("tick = %v, want 2·0.1/200·100 = 0.1", row.TickPct)
	}
	if row.Pass || len(row.Reasons) != 1 || !strings.HasPrefix(row.Reasons[0], "дневной оборот 40") {
		t.Errorf("want only the daily-turnover gate to fail, got %q", row.Reasons)
	}
	if got := ScreenGap(GapScreenInput{Ticker: "NOTICK", Bars: bars, Lot: 10}, DefaultGapScreenOptions()); got.TickPct != 0 {
		t.Errorf("unknown tick: TickPct = %v, want 0", got.TickPct)
	}
}

// gsMarket covers every weekday in [from, to) with a 07:00–23:30 session flat at 100; on a gap
// day the 07:00 bar sits at 99.7 (0.6 ATR below 100 with ATR 0.5) and the 07:30 bar touches 100.
func gsMarket(from, to time.Time, gapDays map[string]bool) (bars, daily []backtest.Candle) {
	for d := from.AddDate(0, -3, 0); d.Before(to); d = d.AddDate(0, 0, 1) {
		if d.Weekday() == time.Saturday || d.Weekday() == time.Sunday {
			continue
		}
		daily = append(daily, backtest.Candle{Time: d.Add(10 * time.Hour), Open: 100, High: 100.25, Low: 99.75, Close: 100})
		if d.Before(from) {
			continue
		}
		for t := d.Add(7 * time.Hour); t.Before(d.Add(24 * time.Hour)); t = t.Add(30 * time.Minute) {
			bars = append(bars, backtest.Candle{Time: t, Open: 100, High: 100, Low: 100, Close: 100})
		}
		if gapDays[d.Format("2006-01-02")] {
			i := len(bars) - 34 // 34 bars per day; the day's first bar
			bars[i] = backtest.Candle{Time: bars[i].Time, Open: 99.7, High: 99.7, Low: 99.7, Close: 99.7}
			bars[i+1] = backtest.Candle{Time: bars[i+1].Time, Open: 99.8, High: 100, Low: 99.7, Close: 100}
		}
	}
	return bars, daily
}

func TestGapSignalsPerYearCountsEngineTrades(t *testing.T) {
	bars, daily := gsMarket(gsDate("2026-05-18"), gsDate("2026-09-01"),
		map[string]bool{"2026-06-10": true, "2026-07-15": true})
	if got := GapSignalsPerYear("SYN", bars, daily, 1, 6); math.Abs(got-4) > 1e-9 {
		t.Fatalf("signals per year = %v, want 2 trades / 0.5 year = 4", got)
	}
	if got := GapSignalsPerYear("SYN", bars, daily, 1, 0); got != 0 {
		t.Fatalf("months 0: %v, want 0", got)
	}
}
