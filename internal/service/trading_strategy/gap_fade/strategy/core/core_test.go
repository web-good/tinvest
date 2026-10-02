package core

import (
	"math"
	"strings"
	"testing"
	"time"

	"tinvest/internal/service/trading_strategy/scalping/model"
	"tinvest/internal/service/trading_strategy/scalping/strategy"
)

const eps = 1e-9

// at parses "2006-01-02 15:04" as MSK.
func at(s string) time.Time {
	t, err := time.ParseInLocation("2006-01-02 15:04", s, mskLoc)
	if err != nil {
		panic(err)
	}
	return t
}

type bar struct {
	t          string
	o, h, l, c float64
}

// session returns flat bars (o=h=l=c=p) every 30 minutes over [from, to], both MSK.
func session(from, to string, p float64) []bar {
	var out []bar
	for t := at(from); !t.After(at(to)); t = t.Add(30 * time.Minute) {
		out = append(out, bar{t.Format("2006-01-02 15:04"), p, p, p, p})
	}
	return out
}

// dailyBars returns `days` calendar days of completed dailies ending the day before `before`,
// all closing at 100; weekday range w, weekend range we. The weekday Wilder ATR is exactly w.
func dailyBars(before time.Time, days int, w, we float64) (highs, lows, closes []float64, times []time.Time) {
	b := before.In(mskLoc)
	start := time.Date(b.Year(), b.Month(), b.Day(), 0, 0, 0, 0, mskLoc).AddDate(0, 0, -days)
	for i := 0; i < days; i++ {
		d := start.AddDate(0, 0, i)
		width := w
		if d.Weekday() == time.Saturday || d.Weekday() == time.Sunday {
			width = we
		}
		highs = append(highs, 100+width/2)
		lows = append(lows, 100-width/2)
		closes = append(closes, 100)
		times = append(times, d.Add(10*time.Hour))
	}
	return highs, lows, closes, times
}

// fixture builds MarketData from bars plus 40 days of dailies with weekday ATR = atr.
func fixture(bars []bar, atr float64) strategy.MarketData {
	var md strategy.MarketData
	for _, b := range bars {
		md.Times = append(md.Times, at(b.t))
		md.Opens = append(md.Opens, b.o)
		md.Highs = append(md.Highs, b.h)
		md.Lows = append(md.Lows, b.l)
		md.Closes = append(md.Closes, b.c)
	}
	md.Price = md.Closes[len(md.Closes)-1]
	md.DailyHighs, md.DailyLows, md.DailyCloses, md.DailyTimes = dailyBars(md.Times[len(md.Times)-1], 40, atr, atr/10)
	return md
}

// friday is the tail of Friday 2026-09-11's evening session, flat at 100.
func friday() []bar { return session("2026-09-11 19:00", "2026-09-11 23:30", 100) }

// mondayFirst opens 0.75 ATR (ATR 2) below Friday's close and is still below it at the close.
var mondayFirst = bar{"2026-09-14 07:00", 98.5, 99.0, 98.2, 98.8}

func decide(p Params, bars []bar) model.Signal {
	return NewWithParams("TEST", p).Decide(fixture(bars, 2.0))
}

func TestEnterOnGapDownWithFrozenLevels(t *testing.T) {
	sig := decide(DefaultParams(), append(friday(), mondayFirst))
	if sig.Kind != model.SignalBuy {
		t.Fatalf("Kind = %v, want Buy", sig.Kind)
	}
	if math.Abs(sig.TakeProfit-100) > eps || math.Abs(sig.StopLoss-97.8) > eps || math.Abs(sig.ATR-2) > eps {
		t.Fatalf("TP/SL/ATR = %v/%v/%v, want 100/97.8/2", sig.TakeProfit, sig.StopLoss, sig.ATR)
	}
	if sig.EntryReason == "" {
		t.Fatal("EntryReason is empty")
	}
}

func TestTargetFillScalesTheTarget(t *testing.T) {
	p := DefaultParams()
	p.TargetFill = 0.5
	sig := decide(p, append(friday(), mondayFirst))
	if math.Abs(sig.TakeProfit-99.4) > eps {
		t.Fatalf("TP = %v, want 99.4 (half of the 1.2 left to Friday's close)", sig.TakeProfit)
	}
}

func TestGapExactlyAtThresholdEnters(t *testing.T) {
	first := bar{"2026-09-14 07:00", 99.0, 99.2, 98.8, 99.1} // (99-100)/2 = -0.5
	if sig := decide(DefaultParams(), append(friday(), first)); sig.Kind != model.SignalBuy {
		t.Fatalf("Kind = %v, want Buy at gap == -GapATR", sig.Kind)
	}
}

func TestNoEntryOnShallowGap(t *testing.T) {
	first := bar{"2026-09-14 07:00", 99.2, 99.4, 99.0, 99.3} // -0.4 ATR
	if sig := decide(DefaultParams(), append(friday(), first)); sig.Kind != model.SignalNone {
		t.Fatalf("Kind = %v, want None", sig.Kind)
	}
}

func TestNoEntryWhenGapAlreadyClosed(t *testing.T) {
	for _, c := range []float64{100, 100.1} {
		first := bar{"2026-09-14 07:00", 98.5, 100.2, 98.4, c}
		if sig := decide(DefaultParams(), append(friday(), first)); sig.Kind != model.SignalNone {
			t.Fatalf("close %v: Kind = %v, want None (gap closed by the entry close)", c, sig.Kind)
		}
	}
}

func TestMaxGapATRSkipsEventGaps(t *testing.T) {
	first := bar{"2026-09-14 07:00", 93.0, 93.8, 92.9, 93.5} // -3.5 ATR
	if sig := decide(DefaultParams(), append(friday(), first)); sig.Kind != model.SignalNone {
		t.Fatalf("Kind = %v, want None beyond MaxGapATR", sig.Kind)
	}
	p := DefaultParams()
	p.MaxGapATR = 0
	if sig := decide(p, append(friday(), first)); sig.Kind != model.SignalBuy {
		t.Fatalf("MaxGapATR=0: Kind = %v, want Buy", sig.Kind)
	}
}

func TestEntryBarOneWaitsForTheSecondBar(t *testing.T) {
	p := DefaultParams()
	p.EntryBar = 1
	if sig := decide(p, append(friday(), mondayFirst)); sig.Kind != model.SignalNone {
		t.Fatalf("first bar: Kind = %v, want None", sig.Kind)
	}
	second := bar{"2026-09-14 07:30", 98.8, 99.1, 98.6, 99.0}
	sig := decide(p, append(friday(), mondayFirst, second))
	if sig.Kind != model.SignalBuy {
		t.Fatalf("second bar: Kind = %v, want Buy", sig.Kind)
	}
	if math.Abs(sig.TakeProfit-100) > eps || math.Abs(sig.StopLoss-98.0) > eps {
		t.Fatalf("TP/SL = %v/%v, want 100/98 (prevClose stays Friday's)", sig.TakeProfit, sig.StopLoss)
	}
	third := bar{"2026-09-14 08:00", 99.0, 99.1, 98.6, 98.9}
	if sig := decide(p, append(friday(), mondayFirst, second, third)); sig.Kind != model.SignalNone {
		t.Fatalf("third bar: Kind = %v, want None", sig.Kind)
	}
}

func TestNoEntryAtOrAfterEOD(t *testing.T) {
	p := DefaultParams()
	p.EODHour, p.EODMinute = 7, 0
	if sig := decide(p, append(friday(), mondayFirst)); sig.Kind != model.SignalNone {
		t.Fatalf("07:00 bar with EOD 07:00: Kind = %v, want None", sig.Kind)
	}
	early := bar{"2026-09-14 06:30", 98.5, 99.0, 98.2, 98.8}
	if sig := decide(p, append(friday(), early)); sig.Kind != model.SignalBuy {
		t.Fatalf("06:30 bar with EOD 07:00: Kind = %v, want Buy", sig.Kind)
	}
}

func TestNoEntryOnWeekendSession(t *testing.T) {
	sat := bar{"2026-09-12 10:00", 98.0, 98.5, 97.8, 98.2}
	if sig := decide(DefaultParams(), append(friday(), sat)); sig.Kind != model.SignalNone {
		t.Fatalf("Kind = %v, want None on Saturday", sig.Kind)
	}
}

func TestMondayGapIsMeasuredFromWeekendSession(t *testing.T) {
	weekend := append(friday(), session("2026-09-12 10:00", "2026-09-12 18:30", 99)...)
	shallow := bar{"2026-09-14 07:00", 98.5, 98.9, 98.3, 98.6} // -0.25 vs Saturday's 99, -0.75 vs Friday
	if sig := decide(DefaultParams(), append(weekend, shallow)); sig.Kind != model.SignalNone {
		t.Fatalf("Kind = %v, want None: the gap is measured from Saturday's close", sig.Kind)
	}
	deep := bar{"2026-09-14 07:00", 97.9, 98.3, 97.7, 98.1} // -0.55 vs 99
	sig := decide(DefaultParams(), append(weekend, deep))
	if sig.Kind != model.SignalBuy || math.Abs(sig.TakeProfit-99) > eps {
		t.Fatalf("Kind/TP = %v/%v, want Buy with target 99", sig.Kind, sig.TakeProfit)
	}
}

func TestNoEntryWithoutWarmDailyATR(t *testing.T) {
	md := fixture(append(friday(), mondayFirst), 2.0)
	md.DailyHighs, md.DailyLows, md.DailyCloses, md.DailyTimes = dailyBars(at("2026-09-14 07:00"), 5, 2, 0.2)
	if sig := NewWithParams("TEST", DefaultParams()).Decide(md); sig.Kind != model.SignalNone {
		t.Fatalf("Kind = %v, want None without a warm ATR", sig.Kind)
	}
}

func TestNoEntryWithoutOpens(t *testing.T) {
	md := fixture(append(friday(), mondayFirst), 2.0)
	md.Opens = nil
	if sig := NewWithParams("TEST", DefaultParams()).Decide(md); sig.Kind != model.SignalNone {
		t.Fatalf("Kind = %v, want None without Opens", sig.Kind)
	}
}

func TestNoEntryWhenDayStartNotInWindow(t *testing.T) {
	p := DefaultParams()
	p.EntryBar = 1
	second := bar{"2026-09-14 07:30", 98.8, 99.1, 98.6, 99.0}
	if sig := decide(p, []bar{mondayFirst, second}); sig.Kind != model.SignalNone {
		t.Fatalf("Kind = %v, want None: no bar before the day start", sig.Kind)
	}
}

func TestNoEntryWithoutStop(t *testing.T) {
	p := DefaultParams()
	p.StopDailyATR = 0
	if sig := decide(p, append(friday(), mondayFirst)); sig.Kind != model.SignalNone {
		t.Fatalf("Kind = %v, want None with StopDailyATR=0", sig.Kind)
	}
}

func openPos() *strategy.Position {
	return &strategy.Position{PurchasePrice: 98.8, StopLoss: 97.8, TakeProfit: 100, EntryTime: at("2026-09-14 07:00")}
}

func manageOn(b bar) model.Signal {
	md := fixture(append(friday(), mondayFirst, b), 2.0)
	md.Position = openPos()
	return NewWithParams("TEST", DefaultParams()).Decide(md)
}

func TestManageStopLoss(t *testing.T) {
	sig := manageOn(bar{"2026-09-14 07:30", 98.5, 99.0, 97.7, 97.9})
	if sig.Kind != model.SignalSell || sig.Reason != "SL" || sig.StopLoss != 97.8 {
		t.Fatalf("got %v/%q/SL %v, want Sell/SL/97.8", sig.Kind, sig.Reason, sig.StopLoss)
	}
}

// Позиция без замороженного стопа (рукописный стейт) закрывается по стопу от цены входа:
// PurchasePrice − StopDailyATR·EntryATR (DefaultParams: 0.5·EntryATR).
func TestManageStopFallsBackToEntryATR(t *testing.T) {
	md := fixture(append(friday(), mondayFirst, bar{"2026-09-14 07:30", 98.5, 99.0, 97.7, 97.9}), 2.0)
	md.Position = openPos()
	md.Position.StopLoss, md.Position.EntryATR = 0, 2.0 // стоп 98.8 − 0.5·2 = 97.8
	sig := NewWithParams("TEST", DefaultParams()).Decide(md)
	if sig.Kind != model.SignalSell || sig.Reason != "SL" || math.Abs(sig.StopLoss-97.8) > eps {
		t.Fatalf("got %v/%q/SL %v, want Sell/SL/97.8", sig.Kind, sig.Reason, sig.StopLoss)
	}
}

func TestEffectiveStop(t *testing.T) {
	p := DefaultParams()
	cases := []struct{ stop, price, atr, want float64 }{
		{78, 90, 4, 78}, // замороженный стоп главнее
		{0, 90, 4, 88},  // фолбэк: 90 − 0.5·4
		{0, 90, 0, 0},   // нечем считать
		{-1, 90, 4, 88}, // отрицательный = нет
	}
	for _, c := range cases {
		if got := p.EffectiveStop(c.stop, c.price, c.atr); math.Abs(got-c.want) > eps {
			t.Errorf("EffectiveStop(%v,%v,%v) = %v, want %v", c.stop, c.price, c.atr, got, c.want)
		}
	}
}

func TestManageTakeProfit(t *testing.T) {
	sig := manageOn(bar{"2026-09-14 07:30", 99.0, 100.2, 98.9, 100.1})
	if sig.Kind != model.SignalSell || sig.Reason != "TP" || sig.TakeProfit != 100 {
		t.Fatalf("got %v/%q/TP %v, want Sell/TP/100", sig.Kind, sig.Reason, sig.TakeProfit)
	}
}

func TestManageStopBeatsTargetOnOneBar(t *testing.T) {
	sig := manageOn(bar{"2026-09-14 07:30", 99.0, 100.2, 97.7, 99.0})
	if sig.Reason != "SL" {
		t.Fatalf("Reason = %q, want SL when both levels are touched", sig.Reason)
	}
}

func TestManageEODAtMainSessionEnd(t *testing.T) {
	if sig := manageOn(bar{"2026-09-14 18:00", 99, 99.1, 98.9, 99}); sig.Kind != model.SignalNone {
		t.Fatalf("18:00: Kind = %v, want None", sig.Kind)
	}
	sig := manageOn(bar{"2026-09-14 18:30", 99, 99.1, 98.9, 99})
	if sig.Kind != model.SignalSell || sig.Reason != "EOD" || sig.ExitReason == "" {
		t.Fatalf("18:30: got %v/%q, want Sell/EOD with a reason", sig.Kind, sig.Reason)
	}
}

func TestManageExitsOnNextDayBar(t *testing.T) {
	sig := manageOn(bar{"2026-09-15 07:00", 99, 99.1, 98.9, 99})
	if sig.Kind != model.SignalSell || sig.Reason != "EOD" {
		t.Fatalf("got %v/%q, want Sell/EOD on the next date", sig.Kind, sig.Reason)
	}
}

// The overnight guard must not depend on EntryTime: a position whose entry time is unknown is
// still closed on the first bar of a new date.
func TestManageExitsOnNewDateFirstBarWithoutEntryTime(t *testing.T) {
	md := fixture(append(friday(), mondayFirst, bar{"2026-09-14 07:30", 99, 99.1, 98.9, 99}, bar{"2026-09-15 07:00", 99, 99.1, 98.9, 99}), 2.0)
	md.Position = openPos()
	md.Position.EntryTime = time.Time{}
	sig := NewWithParams("TEST", DefaultParams()).Decide(md)
	if sig.Kind != model.SignalSell || sig.Reason != "EOD" {
		t.Fatalf("got %v/%q, want Sell/EOD on the new date's first bar", sig.Kind, sig.Reason)
	}
}

// Without bar times the core cannot tell whether the night has passed, so it fails closed.
func TestManageExitsWhenTimesMissing(t *testing.T) {
	for name, mut := range map[string]func(*strategy.MarketData){
		"nil":        func(md *strategy.MarketData) { md.Times = nil },
		"misaligned": func(md *strategy.MarketData) { md.Times = md.Times[1:] },
	} {
		md := fixture(append(friday(), mondayFirst, bar{"2026-09-14 07:30", 99, 99.1, 98.9, 99}), 2.0)
		md.Position = openPos()
		mut(&md)
		sig := NewWithParams("TEST", DefaultParams()).Decide(md)
		if sig.Kind != model.SignalSell || sig.Reason != "EOD" || !strings.Contains(sig.ExitReason, "нет времени баров") {
			t.Fatalf("%s: got %v/%q/%q, want Sell/EOD for missing times", name, sig.Kind, sig.Reason, sig.ExitReason)
		}
	}
}

func TestLookbackCoversEntryBar(t *testing.T) {
	p := DefaultParams()
	if got := NewWithParams("T", p).Lookback(); got != 64 {
		t.Fatalf("Lookback = %d, want 64", got)
	}
	p.EntryBar = 70
	if got := NewWithParams("T", p).Lookback(); got != 72 {
		t.Fatalf("Lookback = %d, want 72", got)
	}
}
