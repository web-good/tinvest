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

// flat returns flat bars (o=h=l=c=p) every 5 minutes over [from, to], both MSK.
func flat(from, to string, p float64) []bar {
	var out []bar
	for t := at(from); !t.After(at(to)); t = t.Add(5 * time.Minute) {
		out = append(out, bar{t.Format("2006-01-02 15:04"), p, p, p, p})
	}
	return out
}

func fixture(bars []bar) strategy.MarketData {
	var md strategy.MarketData
	for _, b := range bars {
		md.Times = append(md.Times, at(b.t))
		md.Opens = append(md.Opens, b.o)
		md.Highs = append(md.Highs, b.h)
		md.Lows = append(md.Lows, b.l)
		md.Closes = append(md.Closes, b.c)
	}
	md.Price = md.Closes[len(md.Closes)-1]
	return md
}

// drop is a red 5m bar at time t that opens at 100 and closes pct percent lower.
func drop(t string, pct float64) bar {
	c := 100 * (1 - pct/100)
	return bar{t, 100, 100.1, c - 0.05, c}
}

func decide(p Params, bars []bar) model.Signal {
	return NewWithParams("TEST", p).Decide(fixture(bars))
}

// Monday 2026-09-14.
func TestEnterOnOnePercentDrop(t *testing.T) {
	sig := decide(DefaultParams(), []bar{flat("2026-09-14 10:00", "2026-09-14 10:00", 100)[0], drop("2026-09-14 10:05", 1.0)})
	if sig.Kind != model.SignalBuy {
		t.Fatalf("kind = %v, want Buy", sig.Kind)
	}
	// body = 1.0; close 99; target 99.5, stop 98.5.
	if math.Abs(sig.TakeProfit-99.5) > eps || math.Abs(sig.StopLoss-98.5) > eps {
		t.Fatalf("TP/SL = %v/%v, want 99.5/98.5", sig.TakeProfit, sig.StopLoss)
	}
	if !strings.HasPrefix(sig.EntryReason, "DROP:") {
		t.Fatalf("entry reason = %q", sig.EntryReason)
	}
}

func TestNoEntryBelowThreshold(t *testing.T) {
	sig := decide(DefaultParams(), []bar{flat("2026-09-14 10:00", "2026-09-14 10:00", 100)[0], drop("2026-09-14 10:05", 0.99)})
	if sig.Kind != model.SignalNone {
		t.Fatalf("kind = %v, want Hold for a 0.99%% drop", sig.Kind)
	}
}

func TestNoEntryOnGreenBar(t *testing.T) {
	sig := decide(DefaultParams(), []bar{
		flat("2026-09-14 10:00", "2026-09-14 10:00", 100)[0],
		{"2026-09-14 10:05", 99, 100.2, 98.9, 100.1},
	})
	if sig.Kind != model.SignalNone {
		t.Fatalf("kind = %v, want Hold", sig.Kind)
	}
}

func TestEntryWindow(t *testing.T) {
	cases := []struct {
		t    string
		want model.SignalKind
	}{
		{"2026-09-14 07:55", model.SignalNone}, // before 08:00
		{"2026-09-14 08:00", model.SignalBuy},  // first bar of the window
		{"2026-09-14 16:50", model.SignalBuy},  // closes at 16:55, fills before 17:00
		{"2026-09-14 16:55", model.SignalNone}, // its fill bar would be the 17:00 EOD bar
		{"2026-09-14 17:00", model.SignalNone},
		{"2026-09-12 10:00", model.SignalNone}, // Saturday
		{"2026-09-13 10:00", model.SignalNone}, // Sunday
	}
	for _, tc := range cases {
		prev := at(tc.t).Add(-5 * time.Minute).Format("2006-01-02 15:04")
		sig := decide(DefaultParams(), []bar{{prev, 100, 100, 100, 100}, drop(tc.t, 1.2)})
		if sig.Kind != tc.want {
			t.Errorf("%s: kind = %v, want %v", tc.t, sig.Kind, tc.want)
		}
	}
}

// On 15m bars the fill bar starts 15 minutes after the signal bar, so the last signal bar before a
// 17:00 EOD is 16:30.
func TestEntryWindow15m(t *testing.T) {
	cases := []struct {
		t    string
		want model.SignalKind
	}{
		{"2026-09-14 08:00", model.SignalBuy},
		{"2026-09-14 16:30", model.SignalBuy},  // fills at 16:45
		{"2026-09-14 16:45", model.SignalNone}, // its fill bar would be the 17:00 EOD bar
	}
	for _, tc := range cases {
		prev := at(tc.t).Add(-15 * time.Minute).Format("2006-01-02 15:04")
		md := fixture([]bar{{prev, 100, 100, 100, 100}, drop(tc.t, 1.2)})
		if sig := NewWithStep("TEST", 15*time.Minute, DefaultParams()).Decide(md); sig.Kind != tc.want {
			t.Errorf("%s: kind = %v, want %v", tc.t, sig.Kind, tc.want)
		}
	}
}

func TestNewWithStepFallsBackToDefault(t *testing.T) {
	if s := NewWithStep("TEST", 0, DefaultParams()); s.step != DefaultBarStep {
		t.Fatalf("step = %v, want %v", s.step, DefaultBarStep)
	}
}

func TestMaxDropCap(t *testing.T) {
	p := DefaultParams()
	p.MaxDropPct = 2.0
	sig := decide(p, []bar{{"2026-09-14 10:00", 100, 100, 100, 100}, drop("2026-09-14 10:05", 2.5)})
	if sig.Kind != model.SignalNone {
		t.Fatalf("kind = %v, want Hold above MaxDropPct", sig.Kind)
	}
}

func position(entry, tp, sl float64, when string) *strategy.Position {
	return &strategy.Position{PurchasePrice: entry, TakeProfit: tp, StopLoss: sl, EntryTime: at(when)}
}

func manage(bars []bar, pos *strategy.Position) model.Signal {
	md := fixture(bars)
	md.Position = pos
	return NewWithParams("TEST", DefaultParams()).Decide(md)
}

func TestManageStopBeforeTarget(t *testing.T) {
	// One bar touches both levels: the stop wins (conservative).
	sig := manage([]bar{
		{"2026-09-14 10:10", 99, 99, 99, 99},
		{"2026-09-14 10:15", 99, 99.6, 98.4, 99},
	}, position(99, 99.5, 98.5, "2026-09-14 10:10"))
	if sig.Kind != model.SignalSell || sig.Reason != "SL" || sig.StopLoss != 98.5 {
		t.Fatalf("got %v %q stop %v, want Sell SL 98.5", sig.Kind, sig.Reason, sig.StopLoss)
	}
}

func TestManageTarget(t *testing.T) {
	sig := manage([]bar{
		{"2026-09-14 10:10", 99, 99, 99, 99},
		{"2026-09-14 10:15", 99, 99.6, 98.9, 99.4},
	}, position(99, 99.5, 98.5, "2026-09-14 10:10"))
	if sig.Kind != model.SignalSell || sig.Reason != "TP" || sig.TakeProfit != 99.5 {
		t.Fatalf("got %v %q, want Sell TP", sig.Kind, sig.Reason)
	}
}

func TestManageHoldInside(t *testing.T) {
	sig := manage([]bar{
		{"2026-09-14 10:10", 99, 99, 99, 99},
		{"2026-09-14 10:15", 99, 99.2, 98.8, 99.1},
	}, position(99, 99.5, 98.5, "2026-09-14 10:10"))
	if sig.Kind != model.SignalNone {
		t.Fatalf("kind = %v, want Hold", sig.Kind)
	}
}

func TestManageEODAtFive(t *testing.T) {
	sig := manage([]bar{
		{"2026-09-14 16:55", 99, 99, 99, 99},
		{"2026-09-14 17:00", 99, 99.1, 98.9, 99},
	}, position(99, 99.5, 98.5, "2026-09-14 16:00"))
	if sig.Kind != model.SignalSell || sig.Reason != "EOD" {
		t.Fatalf("got %v %q, want Sell EOD", sig.Kind, sig.Reason)
	}
}

func TestManageNewDateCloses(t *testing.T) {
	sig := manage([]bar{
		{"2026-09-14 15:00", 99, 99, 99, 99},
		{"2026-09-15 07:00", 99, 99.1, 98.9, 99},
	}, position(99, 99.5, 98.5, "2026-09-14 15:00"))
	if sig.Kind != model.SignalSell || sig.Reason != "EOD" {
		t.Fatalf("got %v %q, want Sell EOD on a new date", sig.Kind, sig.Reason)
	}
}

func manageWith(p Params, bars []bar, pos *strategy.Position) model.Signal {
	md := fixture(bars)
	md.Position = pos
	return NewWithParams("TEST", p).Decide(md)
}

func timeExitParams() Params {
	p := DefaultParams()
	p.TimeExit = 1
	return p
}

// Entry filled at the 10:05 bar's open: 10:05 is bar 1, 10:10 bar 2, 10:15 bar 3.
func TestTimeExitAfterNBars(t *testing.T) {
	bars := []bar{
		{"2026-09-14 10:00", 99, 99, 99, 99},
		{"2026-09-14 10:05", 99, 99.1, 98.9, 99},
		{"2026-09-14 10:10", 99, 99.1, 98.9, 99},
		{"2026-09-14 10:15", 99, 99.1, 98.9, 99.05},
	}
	pos := position(99, 99.5, 98.5, "2026-09-14 10:05")
	sig := manageWith(timeExitParams(), bars, pos)
	if sig.Kind != model.SignalSell || sig.Reason != "TIME" {
		t.Fatalf("got %v %q, want Sell TIME on bar 3", sig.Kind, sig.Reason)
	}
	if sig := manageWith(timeExitParams(), bars[:3], pos); sig.Kind != model.SignalNone {
		t.Fatalf("bar 2: got %v %q, want hold", sig.Kind, sig.Reason)
	}
}

func TestTimeExitOffByDefault(t *testing.T) {
	if p := DefaultParams(); p.TimeExit != 0 || p.TimeExitBars != 3 {
		t.Fatalf("defaults TimeExit/TimeExitBars = %d/%d, want 0/3", p.TimeExit, p.TimeExitBars)
	}
	sig := manageWith(DefaultParams(), []bar{
		{"2026-09-14 10:00", 99, 99, 99, 99},
		{"2026-09-14 10:05", 99, 99.1, 98.9, 99},
		{"2026-09-14 10:10", 99, 99.1, 98.9, 99},
		{"2026-09-14 10:15", 99, 99.1, 98.9, 99},
		{"2026-09-14 10:20", 99, 99.1, 98.9, 99},
	}, position(99, 99.5, 98.5, "2026-09-14 10:05"))
	if sig.Kind != model.SignalNone {
		t.Fatalf("got %v %q, want hold with TimeExit off", sig.Kind, sig.Reason)
	}
}

func TestTargetBeatsTimeExit(t *testing.T) {
	sig := manageWith(timeExitParams(), []bar{
		{"2026-09-14 10:05", 99, 99.1, 98.9, 99},
		{"2026-09-14 10:10", 99, 99.1, 98.9, 99},
		{"2026-09-14 10:15", 99, 99.6, 98.9, 99.4},
	}, position(99, 99.5, 98.5, "2026-09-14 10:05"))
	if sig.Reason != "TP" {
		t.Fatalf("reason = %q, want TP before TIME", sig.Reason)
	}
}

func TestLookbackCoversTimeExit(t *testing.T) {
	p := timeExitParams()
	p.TimeExitBars = 10
	if got := NewWithParams("TEST", p).Lookback(); got < 11 {
		t.Fatalf("Lookback = %d, want ≥ 11", got)
	}
}

func seriesParams() Params {
	p := DefaultParams()
	p.DropSeries = 1
	return p
}

// red3 is three red bars 10:00–10:10 falling 100 → 99.6 → 99.3 → 98.9 (−1.1% in total), each
// below MinDropPct on its own.
func red3() []bar {
	return []bar{
		{"2026-09-14 09:55", 100, 100, 100, 100},
		{"2026-09-14 10:00", 100, 100.05, 99.55, 99.6},
		{"2026-09-14 10:05", 99.6, 99.65, 99.25, 99.3},
		{"2026-09-14 10:10", 99.3, 99.35, 98.85, 98.9},
	}
}

func TestSeriesOffByDefault(t *testing.T) {
	if p := DefaultParams(); p.DropSeries != 0 || p.DropSeriesBars != 3 {
		t.Fatalf("defaults DropSeries/DropSeriesBars = %d/%d, want 0/3", p.DropSeries, p.DropSeriesBars)
	}
	if sig := decide(DefaultParams(), red3()); sig.Kind != model.SignalNone {
		t.Fatalf("kind = %v, want no entry with the series off", sig.Kind)
	}
}

func TestSeriesEntry(t *testing.T) {
	sig := decide(seriesParams(), red3())
	if sig.Kind != model.SignalBuy {
		t.Fatalf("kind = %v, want Buy on a 3-bar −1.1%% series", sig.Kind)
	}
	// body = 100 − 98.9 = 1.1; target 98.9 + 0.55, stop 98.9 − 0.55.
	if math.Abs(sig.TakeProfit-99.45) > eps || math.Abs(sig.StopLoss-98.35) > eps {
		t.Fatalf("TP/SL = %v/%v, want 99.45/98.35", sig.TakeProfit, sig.StopLoss)
	}
	if !strings.HasPrefix(sig.EntryReason, "DROP3:") {
		t.Fatalf("entry reason = %q", sig.EntryReason)
	}
}

func TestSeriesNeedsAllRed(t *testing.T) {
	bars := red3()
	bars[2] = bar{"2026-09-14 10:05", 99.3, 99.65, 99.25, 99.3} // doji: open == close
	if sig := decide(seriesParams(), bars); sig.Kind != model.SignalNone {
		t.Fatalf("kind = %v, want no entry when a bar is not red", sig.Kind)
	}
}

func TestSeriesBelowThreshold(t *testing.T) {
	bars := red3()
	bars[3] = bar{"2026-09-14 10:10", 99.3, 99.35, 99.15, 99.2} // total −0.8%
	if sig := decide(seriesParams(), bars); sig.Kind != model.SignalNone {
		t.Fatalf("kind = %v, want no entry for a −0.8%% series", sig.Kind)
	}
}

func TestSeriesRespectsMaxDrop(t *testing.T) {
	p := seriesParams()
	p.MaxDropPct = 1.0
	if sig := decide(p, red3()); sig.Kind != model.SignalNone {
		t.Fatalf("kind = %v, want no entry above MaxDropPct", sig.Kind)
	}
}

func TestSeriesStaysWithinOneDay(t *testing.T) {
	bars := []bar{
		{"2026-09-11 16:00", 100, 100, 100, 100},
		{"2026-09-11 16:05", 100, 100.05, 99.55, 99.6},
		{"2026-09-14 10:00", 99.6, 99.65, 99.25, 99.3},
		{"2026-09-14 10:05", 99.3, 99.35, 98.85, 98.9},
	}
	if sig := decide(seriesParams(), bars); sig.Kind != model.SignalNone {
		t.Fatalf("kind = %v, want no entry for a series across dates", sig.Kind)
	}
}

func TestSingleBarStillWorksWithSeriesOn(t *testing.T) {
	sig := decide(seriesParams(), []bar{{"2026-09-14 10:00", 100, 100, 100, 100}, drop("2026-09-14 10:05", 1.0)})
	if sig.Kind != model.SignalBuy || !strings.HasPrefix(sig.EntryReason, "DROP:") {
		t.Fatalf("got %v %q, want single-bar Buy", sig.Kind, sig.EntryReason)
	}
}

func TestLookbackCoversSeries(t *testing.T) {
	p := seriesParams()
	p.DropSeriesBars = 6
	if got := NewWithParams("TEST", p).Lookback(); got < 6 {
		t.Fatalf("Lookback = %d, want ≥ 6", got)
	}
}

// TestTargetPctOverridesBody pins the percent target: close 99 × (1 + 0.4%) = 99.396, the stop
// still StopBody of the body.
func TestTargetPctOverridesBody(t *testing.T) {
	p := DefaultParams()
	p.TargetPct = 0.4
	sig := decide(p, []bar{flat("2026-09-14 10:00", "2026-09-14 10:00", 100)[0], drop("2026-09-14 10:05", 1.0)})
	if sig.Kind != model.SignalBuy {
		t.Fatalf("kind = %v, want Buy", sig.Kind)
	}
	if math.Abs(sig.TakeProfit-99.396) > eps || math.Abs(sig.StopLoss-98.5) > eps {
		t.Fatalf("TP/SL = %v/%v, want 99.396/98.5", sig.TakeProfit, sig.StopLoss)
	}
}

// TestTargetPctWorksWithoutTargetBody pins that a percent target needs no TargetBody.
func TestTargetPctWorksWithoutTargetBody(t *testing.T) {
	p := DefaultParams()
	p.TargetBody, p.TargetPct = 0, 0.4
	if sig := decide(p, []bar{flat("2026-09-14 10:00", "2026-09-14 10:00", 100)[0], drop("2026-09-14 10:05", 1.0)}); sig.Kind != model.SignalBuy {
		t.Fatalf("kind = %v, want Buy", sig.Kind)
	}
}

// TestMinTargetPctSkipsShortTarget pins the cost filter: a 1% bar with TargetBody 0.5 aims
// 0.5/99 ≈ 0.505% above the close, so a 0.5% floor passes and a 0.51% floor skips.
func TestMinTargetPctSkipsShortTarget(t *testing.T) {
	bars := []bar{flat("2026-09-14 10:00", "2026-09-14 10:00", 100)[0], drop("2026-09-14 10:05", 1.0)}
	p := DefaultParams()
	p.MinTargetPct = 0.5
	if sig := decide(p, bars); sig.Kind != model.SignalBuy {
		t.Fatalf("floor 0.5: kind = %v, want Buy", sig.Kind)
	}
	p.MinTargetPct = 0.51
	if sig := decide(p, bars); sig.Kind == model.SignalBuy {
		t.Fatalf("floor 0.51: got Buy, want no signal")
	}
}

func confirmParams() Params {
	p := DefaultParams()
	p.ConfirmBar = 1
	return p
}

// dropThen is a flat bar, a −1% drop at 10:05 (close 99, body 1) and the bar after it at 10:10.
func dropThen(next bar) []bar {
	return []bar{{"2026-09-14 10:00", 100, 100, 100, 100}, drop("2026-09-14 10:05", 1.0), next}
}

func TestConfirmOffByDefault(t *testing.T) {
	if p := DefaultParams(); p.ConfirmBar != 0 {
		t.Fatalf("default ConfirmBar = %d, want 0", p.ConfirmBar)
	}
}

// TestConfirmSkipsTheDropBar pins that with confirmation on the drop bar itself does not buy.
func TestConfirmSkipsTheDropBar(t *testing.T) {
	bars := []bar{{"2026-09-14 10:00", 100, 100, 100, 100}, drop("2026-09-14 10:05", 1.0)}
	if sig := decide(confirmParams(), bars); sig.Kind != model.SignalNone {
		t.Fatalf("kind = %v, want no entry on the drop bar", sig.Kind)
	}
}

// TestConfirmBuysOnGreenBarAboveDropClose pins the entry on the confirmation bar with the levels
// of the drop bar: close 99, body 1 → target 99.5, stop 98.5.
func TestConfirmBuysOnGreenBarAboveDropClose(t *testing.T) {
	sig := decide(confirmParams(), dropThen(bar{"2026-09-14 10:10", 99.0, 99.2, 98.95, 99.1}))
	if sig.Kind != model.SignalBuy {
		t.Fatalf("kind = %v, want Buy", sig.Kind)
	}
	if math.Abs(sig.TakeProfit-99.5) > eps || math.Abs(sig.StopLoss-98.5) > eps {
		t.Fatalf("TP/SL = %v/%v, want 99.5/98.5", sig.TakeProfit, sig.StopLoss)
	}
	if !strings.HasPrefix(sig.EntryReason, "DROP+CONF:") {
		t.Fatalf("entry reason = %q", sig.EntryReason)
	}
}

func TestConfirmNeedsGreenBar(t *testing.T) {
	// opens above the drop close and closes lower: red, though still above 99.
	if sig := decide(confirmParams(), dropThen(bar{"2026-09-14 10:10", 99.3, 99.35, 99.1, 99.15})); sig.Kind != model.SignalNone {
		t.Fatalf("kind = %v, want no entry on a red bar", sig.Kind)
	}
}

func TestConfirmNeedsCloseAboveDropClose(t *testing.T) {
	// green, but opened lower and closed at 98.95 < 99.
	if sig := decide(confirmParams(), dropThen(bar{"2026-09-14 10:10", 98.8, 99.0, 98.75, 98.95})); sig.Kind != model.SignalNone {
		t.Fatalf("kind = %v, want no entry below the drop close", sig.Kind)
	}
}

func TestConfirmStaysWithinOneDay(t *testing.T) {
	bars := []bar{{"2026-09-11 16:00", 100, 100, 100, 100}, drop("2026-09-11 16:05", 1.0), {"2026-09-14 10:00", 99.0, 99.2, 98.95, 99.1}}
	if sig := decide(confirmParams(), bars); sig.Kind != model.SignalNone {
		t.Fatalf("kind = %v, want no entry when the confirmation is on the next date", sig.Kind)
	}
}

// TestConfirmMinTargetFromConfirmClose pins that the cost floor is measured from the confirmation
// close, the price the order will get: target 99.5 is 0.5/99.1 ≈ 0.4036% above 99.1.
func TestConfirmMinTargetFromConfirmClose(t *testing.T) {
	bars := dropThen(bar{"2026-09-14 10:10", 99.0, 99.2, 98.95, 99.1})
	p := confirmParams()
	p.MinTargetPct = 0.4
	if sig := decide(p, bars); sig.Kind != model.SignalBuy {
		t.Fatalf("floor 0.4: kind = %v, want Buy", sig.Kind)
	}
	p.MinTargetPct = 0.41
	if sig := decide(p, bars); sig.Kind == model.SignalBuy {
		t.Fatal("floor 0.41: got Buy, want no signal")
	}
}

func TestConfirmWorksWithSeries(t *testing.T) {
	p := seriesParams()
	p.ConfirmBar = 1
	bars := append(red3(), bar{"2026-09-14 10:15", 98.9, 99.1, 98.85, 99.0})
	sig := decide(p, bars)
	if sig.Kind != model.SignalBuy || !strings.HasPrefix(sig.EntryReason, "DROP3+CONF:") {
		t.Fatalf("got %v %q, want series Buy after confirmation", sig.Kind, sig.EntryReason)
	}
	if math.Abs(sig.TakeProfit-99.45) > eps {
		t.Fatalf("TP = %v, want 99.45 (series levels)", sig.TakeProfit)
	}
}

func TestLookbackCoversConfirm(t *testing.T) {
	p := seriesParams()
	p.ConfirmBar = 1
	p.DropSeriesBars = 6
	if got := NewWithParams("TEST", p).Lookback(); got < 7 {
		t.Fatalf("Lookback = %d, want ≥ 7", got)
	}
}

// trendBars returns n flat 5m bars at price p ending at 10:00 on Monday 2026-09-14, then a 1.2%
// drop from 100 at 10:05.
func trendBars(n int, p float64) []bar {
	from := at("2026-09-14 10:00").Add(-time.Duration(n-1) * 5 * time.Minute)
	bars := flat(from.Format("2006-01-02 15:04"), "2026-09-14 10:00", p)
	return append(bars, drop("2026-09-14 10:05", 1.2))
}

func TestTrendEMAOffByDefault(t *testing.T) {
	if sig := decide(DefaultParams(), trendBars(5, 90)); sig.Kind != model.SignalBuy {
		t.Fatalf("kind = %v, want Buy with the filter off", sig.Kind)
	}
}

func TestTrendEMAPassesAboveEMA(t *testing.T) {
	p := DefaultParams()
	p.TrendEMA = 10
	bars := trendBars(20, 99)
	bars[len(bars)-2].c = 100 // pre-drop close above an EMA of ~99
	if sig := decide(p, bars); sig.Kind != model.SignalBuy {
		t.Fatalf("kind = %v, want Buy above the EMA", sig.Kind)
	}
}

func TestTrendEMABlocksBelowEMA(t *testing.T) {
	p := DefaultParams()
	p.TrendEMA = 10
	bars := trendBars(20, 101)
	bars[len(bars)-2].c = 100 // pre-drop close below an EMA of ~101
	if sig := decide(p, bars); sig.Kind != model.SignalNone {
		t.Fatalf("kind = %v, want Hold below the EMA", sig.Kind)
	}
}

func TestTrendEMAFailsWithoutHistory(t *testing.T) {
	p := DefaultParams()
	p.TrendEMA = 10
	if sig := decide(p, trendBars(5, 99)); sig.Kind != model.SignalNone {
		t.Fatalf("kind = %v, want Hold without enough bars for the EMA", sig.Kind)
	}
}

func TestTrendDailyEMA(t *testing.T) {
	p := DefaultParams()
	p.TrendDailyEMA = 3
	cases := []struct {
		name  string
		daily []float64
		want  model.SignalKind
	}{
		{"above", []float64{90, 91, 92, 93, 95}, model.SignalBuy},
		{"below", []float64{95, 94, 93, 92, 90}, model.SignalNone},
		{"short history", []float64{90, 95}, model.SignalNone},
	}
	for _, tc := range cases {
		md := fixture(trendBars(2, 100))
		md.DailyCloses = tc.daily
		if sig := NewWithParams("TEST", p).Decide(md); sig.Kind != tc.want {
			t.Errorf("%s: kind = %v, want %v", tc.name, sig.Kind, tc.want)
		}
	}
}

func TestLookbackCoversTrendEMA(t *testing.T) {
	p := DefaultParams()
	p.TrendEMA = 200
	if got := NewWithParams("TEST", p).Lookback(); got < 401 {
		t.Fatalf("Lookback = %d, want ≥ 401", got)
	}
}

func TestSkipHours(t *testing.T) {
	p := DefaultParams()
	p.SkipFromHour, p.SkipToHour = 10, 14
	cases := []struct {
		t    string
		want model.SignalKind
	}{
		{"2026-09-14 09:55", model.SignalBuy},
		{"2026-09-14 10:00", model.SignalNone},
		{"2026-09-14 13:55", model.SignalNone},
		{"2026-09-14 14:00", model.SignalBuy},
	}
	for _, tc := range cases {
		prev := at(tc.t).Add(-5 * time.Minute).Format("2006-01-02 15:04")
		if sig := decide(p, []bar{{prev, 100, 100, 100, 100}, drop(tc.t, 1.2)}); sig.Kind != tc.want {
			t.Errorf("%s: kind = %v, want %v", tc.t, sig.Kind, tc.want)
		}
	}
}

func TestSkipHoursOffWhenReversed(t *testing.T) {
	p := DefaultParams()
	p.SkipFromHour, p.SkipToHour = 14, 10
	if sig := decide(p, []bar{{"2026-09-14 11:55", 100, 100, 100, 100}, drop("2026-09-14 12:00", 1.2)}); sig.Kind != model.SignalBuy {
		t.Fatalf("kind = %v, want Buy with a reversed (off) pause", sig.Kind)
	}
}

// With confirmation on, the pause applies to the drop bar, not to the confirmation bar.
func TestSkipHoursUsesDropBarWithConfirm(t *testing.T) {
	p := confirmParams()
	p.SkipFromHour, p.SkipToHour = 10, 11
	bars := []bar{{"2026-09-14 09:50", 100, 100, 100, 100}, drop("2026-09-14 09:55", 1.0), {"2026-09-14 10:00", 99.0, 99.2, 98.95, 99.1}}
	if sig := decide(p, bars); sig.Kind != model.SignalBuy {
		t.Fatalf("kind = %v, want Buy: the drop bar 09:55 is before the pause", sig.Kind)
	}
}

// pumpBars returns 25 flat 5m bars at 100 ending at 10:00 on Monday 2026-09-14, the back-th of
// them from the end (1 = 10:00) replaced by a green bar rising pct percent, then a 1.2% drop from
// 100 at 10:05 (body 1.2).
func pumpBars(back int, pct float64) []bar {
	bars := trendBars(25, 100)
	j := len(bars) - 1 - back
	c := 100 * (1 + pct/100)
	bars[j] = bar{bars[j].t, 100, c, 100, c}
	return bars
}

func pumpParams() Params {
	p := DefaultParams()
	p.PumpBars = 20
	return p
}

func TestPumpOffByDefault(t *testing.T) {
	if sig := decide(DefaultParams(), pumpBars(5, 3)); sig.Kind != model.SignalBuy {
		t.Fatalf("kind = %v, want Buy with the filter off", sig.Kind)
	}
}

func TestPumpOffWithoutThresholds(t *testing.T) {
	if sig := decide(pumpParams(), pumpBars(5, 3)); sig.Kind != model.SignalBuy {
		t.Fatalf("kind = %v, want Buy: PumpBars without PumpPct/PumpRatio is off", sig.Kind)
	}
}

func TestPumpPct(t *testing.T) {
	p := pumpParams()
	p.PumpPct = 1.0
	cases := []struct {
		back int
		pct  float64
		want model.SignalKind
	}{
		{1, 1.0, model.SignalNone},  // the bar right before the drop
		{20, 1.0, model.SignalNone}, // the oldest bar of the window
		{21, 1.0, model.SignalBuy},  // outside the window
		{5, 0.9, model.SignalBuy},   // below the threshold
	}
	for _, tc := range cases {
		if sig := decide(p, pumpBars(tc.back, tc.pct)); sig.Kind != tc.want {
			t.Errorf("back %d, +%.1f%%: kind = %v, want %v", tc.back, tc.pct, sig.Kind, tc.want)
		}
	}
}

func TestPumpRatio(t *testing.T) {
	p := pumpParams()
	p.PumpRatio = 1.0
	if sig := decide(p, pumpBars(5, 1.2)); sig.Kind != model.SignalNone {
		t.Fatalf("green body equal to the drop body: kind = %v, want None", sig.Kind)
	}
	if sig := decide(p, pumpBars(5, 1.1)); sig.Kind != model.SignalBuy {
		t.Fatalf("green body below the drop body: kind = %v, want Buy", sig.Kind)
	}
}

func TestPumpChecksAvailableHistoryOnly(t *testing.T) {
	p := pumpParams()
	p.PumpPct = 1.0
	if sig := decide(p, trendBars(3, 100)); sig.Kind != model.SignalBuy {
		t.Fatalf("kind = %v, want Buy: short history without a pump passes", sig.Kind)
	}
}

func TestLookbackCoversPumpBars(t *testing.T) {
	p := pumpParams()
	p.PumpPct = 1.0
	if got := NewWithParams("TEST", p).Lookback(); got < 21 {
		t.Fatalf("Lookback = %d, want ≥ 21", got)
	}
}
