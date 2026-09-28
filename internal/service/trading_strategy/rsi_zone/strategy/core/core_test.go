package core

import (
	"math"
	"testing"
	"time"

	"tinvest/internal/domain/ema"
	"tinvest/internal/service/trading_strategy/scalping/model"
	"tinvest/internal/service/trading_strategy/scalping/strategy"
	"tinvest/pkg/indicators"
)

// msk is the timezone every calendar rule is anchored to.
var msk = func() *time.Location {
	loc, err := time.LoadLocation("Europe/Moscow")
	if err != nil {
		return time.UTC
	}
	return loc
}()

// mondayNoon is a weekday bar time (2026-06-01 is a Monday).
var mondayNoon = time.Date(2026, 6, 1, 12, 0, 0, 0, msk)

// saturdayNoon is a weekend bar time (2026-05-30 is a Saturday).
var saturdayNoon = time.Date(2026, 5, 30, 12, 0, 0, 0, msk)

// dailyWidth is the weekday daily range of the fixture; with a flat close the daily ATR over
// weekday bars is exactly this value.
const dailyWidth = 2.0

// dailyBars builds `days` consecutive calendar days ending the day BEFORE `before` (MSK),
// oldest-first. Every bar closes at 100; a weekday bar spans `w`, a weekend bar spans `we`, so a
// test can prove weekend sessions never reach the ATR.
func dailyBars(before time.Time, days int, w, we float64) (highs, lows, closes []float64, times []time.Time) {
	b := before.In(msk)
	start := time.Date(b.Year(), b.Month(), b.Day(), 0, 0, 0, 0, msk).AddDate(0, 0, -days)
	const price = 100.0
	for i := 0; i < days; i++ {
		d := start.AddDate(0, 0, i)
		width := w
		if d.Weekday() == time.Saturday || d.Weekday() == time.Sunday {
			width = we
		}
		highs = append(highs, price+width/2)
		lows = append(lows, price-width/2)
		closes = append(closes, price)
		times = append(times, d.Add(10*time.Hour))
	}
	return highs, lows, closes, times
}

// fixtureWithDaily builds 30-minute MarketData from closes, the LAST bar stamped at `last`,
// with a 0.3% high/low envelope and 40 calendar days of completed dailies whose weekday ATR is
// exactly `width` (weekend dailies are ten times narrower).
func fixtureWithDaily(closes []float64, last time.Time, width float64) strategy.MarketData {
	n := len(closes)
	start := last.Add(-time.Duration(n-1) * 30 * time.Minute)
	md := strategy.MarketData{
		Highs:  make([]float64, n),
		Lows:   make([]float64, n),
		Closes: append([]float64(nil), closes...),
		Times:  make([]time.Time, n),
	}
	for i, c := range closes {
		md.Highs[i] = c * 1.003
		md.Lows[i] = c * 0.997
		md.Times[i] = start.Add(time.Duration(i) * 30 * time.Minute)
	}
	if n > 0 {
		md.Price = closes[n-1]
	}
	md.DailyHighs, md.DailyLows, md.DailyCloses, md.DailyTimes = dailyBars(last, 40, width, width/10)
	return md
}

func fixture(closes []float64, last time.Time) strategy.MarketData {
	return fixtureWithDaily(closes, last, dailyWidth)
}

// trendCloses is a 400-bar steady uptrend (+0.08% per bar) followed by `downBars` bars of
// -0.2%. With RSI(4): downBars=3 crosses below 25 exactly on the last bar (33.92 -> 22.58);
// downBars=4 is already inside the zone on the previous bar (22.58 -> 15.63). The close stays
// far above EMA(200) (~136.87 vs ~127.81).
func trendCloses(downBars int) []float64 {
	out := make([]float64, 0, 400+downBars)
	p := 100.0
	for i := 0; i < 400; i++ {
		p *= 1.0008
		out = append(out, p)
	}
	for i := 0; i < downBars; i++ {
		p *= 0.998
		out = append(out, p)
	}
	return out
}

// downtrendCloses is a 400-bar net decline (alternating +0.2% / -0.28%, which keeps Wilder's
// avgGain above zero so RSI has room above the band) followed by two -0.2% bars. RSI(4) crosses
// below 25 exactly on the last bar (27.42 -> 21.37) while the close (~169.55) sits below
// EMA(200) (~177.38): only the trend gate can reject it.
func downtrendCloses() []float64 {
	out := make([]float64, 0, 402)
	p := 200.0
	for i := 0; i < 400; i++ {
		if i%2 == 0 {
			p *= 1.002
		} else {
			p *= 0.9972
		}
		out = append(out, p)
	}
	for i := 0; i < 2; i++ {
		p *= 0.998
		out = append(out, p)
	}
	return out
}

// TestFixturesHaveTheirIntendedShape pins the indicator values every other test relies on, so a
// fixture drift fails here with a readable message instead of as a mysterious gate rejection.
func TestFixturesHaveTheirIntendedShape(t *testing.T) {
	check := func(name string, closes []float64, wantCross, wantAboveEMA bool) {
		t.Helper()
		n := len(closes)
		rsi := indicators.RSISeries(closes, 4)
		e := ema.Compute(closes, 200)
		cross := rsi[n-2] >= 25 && rsi[n-1] < 25
		if cross != wantCross {
			t.Errorf("%s: RSI(4) %.2f -> %.2f, cross below 25 = %v, want %v", name, rsi[n-2], rsi[n-1], cross, wantCross)
		}
		if above := closes[n-1] > e[n-1]; above != wantAboveEMA {
			t.Errorf("%s: close %.4f vs EMA(200) %.4f, above = %v, want %v", name, closes[n-1], e[n-1], above, wantAboveEMA)
		}
	}
	check("trendCloses(3)", trendCloses(3), true, true)
	check("trendCloses(4)", trendCloses(4), false, true)
	check("downtrendCloses", downtrendCloses(), true, false)
}

func TestDefaultParams(t *testing.T) {
	want := Params{RSIPeriod: 4, RSILower: 25, RSIUpper: 75, EMAPeriod: 200, DailyATRPeriod: 14, StopDailyATR: 1.0}
	if got := DefaultParams(); got != want {
		t.Fatalf("DefaultParams() = %+v, want %+v", got, want)
	}
}

func TestLookback(t *testing.T) {
	cases := []struct {
		name string
		p    Params
		want int
	}{
		{"default EMA 200", DefaultParams(), 420},
		{"EMA 50 floors at minLookback", Params{RSIPeriod: 4, EMAPeriod: 50}, 120},
		{"EMA 150", Params{RSIPeriod: 4, EMAPeriod: 150}, 320},
		{"zero periods floor at minLookback", Params{}, 120},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := NewWithParams("T", c.p).Lookback(); got != c.want {
				t.Fatalf("Lookback() = %d, want %d", got, c.want)
			}
		})
	}
}

func TestCrossHelpersBoundaries(t *testing.T) {
	cases := []struct {
		name     string
		series   []float64
		period   int // RSI length series was built with; series[i-1] is warm-up iff i-1 < period
		level    float64
		wantDown bool
		wantUp   bool
	}{
		{"down through", []float64{30, 20}, 0, 25, true, false},
		{"from exactly on the level down", []float64{25, 20}, 0, 25, true, false},
		{"landing exactly on the level is not a cross", []float64{30, 25}, 0, 25, false, false},
		{"already below", []float64{20, 15}, 0, 25, false, false},
		{"warm-up zero is not a cross", []float64{0, 20}, 2, 25, false, false},
		{"up through", []float64{70, 80}, 0, 75, false, true},
		{"from exactly on the level up", []float64{75, 80}, 0, 75, false, true},
		{"already above", []float64{80, 85}, 0, 75, false, false},
		{"warm-up zero never crosses up", []float64{0, 80}, 2, 75, false, false},
		// A genuine RSI of exactly 0.00 past warm-up (i-1 >= period) is a real prior reading, not
		// an unset slot: it must still count as "at or below/above the level" for both helpers.
		// Regression for the bug where the guard rejected series[i-1] == 0 by VALUE regardless of
		// whether it was actually computed.
		{"genuine post-warm-up zero counts as a valid prior value (down)", []float64{0, -10}, 0, -5, true, false},
		{"genuine post-warm-up zero counts as a valid prior value (up)", []float64{0, 80}, 0, 75, false, true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := crossedDown(c.series, 1, c.period, c.level); got != c.wantDown {
				t.Errorf("crossedDown = %v, want %v", got, c.wantDown)
			}
			if got := crossedUp(c.series, 1, c.period, c.level); got != c.wantUp {
				t.Errorf("crossedUp = %v, want %v", got, c.wantUp)
			}
		})
	}
	if crossedDown([]float64{30}, 0, 0, 25) || crossedUp([]float64{70}, 0, 0, 75) {
		t.Fatal("index 0 has no previous bar and must never be a cross")
	}
}

func TestTrendUp(t *testing.T) {
	if !trendUp(101, 100) {
		t.Error("close above EMA must be an uptrend")
	}
	if trendUp(100, 100) {
		t.Error("close equal to EMA is not an uptrend")
	}
	if trendUp(99, 100) {
		t.Error("close below EMA is not an uptrend")
	}
	if trendUp(101, 0) {
		t.Error("an unwarmed (zero) EMA is not an uptrend")
	}
}

func TestStopLevel(t *testing.T) {
	cases := []struct {
		name          string
		k, entry, atr float64
		want          float64
	}{
		{"one ATR below entry", 1.0, 100, 2, 98},
		{"half ATR", 0.5, 100, 2, 99},
		{"stop disabled", 0, 100, 2, 0},
		{"no entry ATR", 1.0, 100, 0, 0},
		{"negative entry ATR", 1.0, 100, -1, 0},
		{"level at zero is no stop", 1.0, 2, 2, 0},
		{"level below zero is no stop", 1.0, 1, 2, 0},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			p := DefaultParams()
			p.StopDailyATR = c.k
			if got := StopLevel(p, c.entry, c.atr); math.Abs(got-c.want) > 1e-9 {
				t.Fatalf("StopLevel = %v, want %v", got, c.want)
			}
		})
	}
}

func TestEnterBuysOnFreshCrossInUptrend(t *testing.T) {
	md := fixture(trendCloses(3), mondayNoon)
	sig := NewWithParams("TEST", DefaultParams()).Decide(md)
	if sig.Kind != model.SignalBuy {
		t.Fatalf("Kind = %v, want SignalBuy", sig.Kind)
	}
	entry := md.Closes[len(md.Closes)-1]
	// ATR == dailyWidth proves the weekend dailies (width/10) were dropped: with them the
	// Wilder average would come out strictly below dailyWidth.
	if math.Abs(sig.ATR-dailyWidth) > 1e-9 {
		t.Errorf("ATR = %v, want %v (weekday-only daily ATR)", sig.ATR, dailyWidth)
	}
	if math.Abs(sig.StopLoss-(entry-dailyWidth)) > 1e-9 {
		t.Errorf("StopLoss = %v, want %v (entry - 1.0 x daily ATR)", sig.StopLoss, entry-dailyWidth)
	}
	if sig.TakeProfit != 0 {
		t.Errorf("TakeProfit = %v, want 0 (no target)", sig.TakeProfit)
	}
	if sig.RSI >= 25 || sig.RSI <= 0 {
		t.Errorf("RSI = %v, want inside (0, 25)", sig.RSI)
	}
	if sig.Ticker != "TEST" || sig.Price != entry {
		t.Errorf("Ticker/Price = %q/%v, want TEST/%v", sig.Ticker, sig.Price, entry)
	}
	if sig.EntryReason == "" {
		t.Error("EntryReason must explain the entry")
	}
}

func TestEnterWithStopDisabledCarriesNoStop(t *testing.T) {
	p := DefaultParams()
	p.StopDailyATR = 0
	sig := NewWithParams("TEST", p).Decide(fixture(trendCloses(3), mondayNoon))
	if sig.Kind != model.SignalBuy || sig.StopLoss != 0 {
		t.Fatalf("Kind/StopLoss = %v/%v, want SignalBuy/0", sig.Kind, sig.StopLoss)
	}
}

func TestEnterGates(t *testing.T) {
	cases := []struct {
		name   string
		md     func() strategy.MarketData
		params func(p *Params)
	}{
		{"RSI already inside the zone, no fresh cross", func() strategy.MarketData { return fixture(trendCloses(4), mondayNoon) }, nil},
		{"weekend bar", func() strategy.MarketData { return fixture(trendCloses(3), saturdayNoon) }, nil},
		{"close below EMA", func() strategy.MarketData { return fixture(downtrendCloses(), mondayNoon) }, nil},
		{"EMA not warmed", func() strategy.MarketData { return fixture(trendCloses(3), mondayNoon) }, func(p *Params) { p.EMAPeriod = 1000 }},
		{"EMAPeriod 0", func() strategy.MarketData { return fixture(trendCloses(3), mondayNoon) }, func(p *Params) { p.EMAPeriod = 0 }},
		{"RSIPeriod 0", func() strategy.MarketData { return fixture(trendCloses(3), mondayNoon) }, func(p *Params) { p.RSIPeriod = 0 }},
		{"RSI period longer than the window", func() strategy.MarketData { return fixture(trendCloses(3), mondayNoon) }, func(p *Params) { p.RSIPeriod = 5000 }},
		{"no daily data", func() strategy.MarketData {
			md := fixture(trendCloses(3), mondayNoon)
			md.DailyHighs, md.DailyLows, md.DailyCloses, md.DailyTimes = nil, nil, nil, nil
			return md
		}, nil},
		{"too few weekday dailies", func() strategy.MarketData {
			md := fixture(trendCloses(3), mondayNoon)
			k := len(md.DailyCloses) - 10
			md.DailyHighs, md.DailyLows, md.DailyCloses, md.DailyTimes = md.DailyHighs[k:], md.DailyLows[k:], md.DailyCloses[k:], md.DailyTimes[k:]
			return md
		}, nil},
		{"DailyATRPeriod 0", func() strategy.MarketData { return fixture(trendCloses(3), mondayNoon) }, func(p *Params) { p.DailyATRPeriod = 0 }},
		{"stop at or below zero", func() strategy.MarketData { return fixture(trendCloses(3), mondayNoon) }, func(p *Params) { p.StopDailyATR = 100 }},
		{"misaligned highs", func() strategy.MarketData {
			md := fixture(trendCloses(3), mondayNoon)
			md.Highs = md.Highs[:len(md.Highs)-1]
			return md
		}, nil},
		{"single bar", func() strategy.MarketData { return fixture(trendCloses(3)[:1], mondayNoon) }, nil},
		{"empty window", func() strategy.MarketData { return strategy.MarketData{} }, nil},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			p := DefaultParams()
			if c.params != nil {
				c.params(&p)
			}
			if sig := NewWithParams("TEST", p).Decide(c.md()); sig.Kind != model.SignalNone {
				t.Fatalf("Kind = %v, want SignalNone", sig.Kind)
			}
		})
	}
}

func TestEnterSkipsWeekdayGateWithoutTimes(t *testing.T) {
	md := fixture(trendCloses(3), saturdayNoon)
	md.Times = nil
	if sig := NewWithParams("TEST", DefaultParams()).Decide(md); sig.Kind != model.SignalBuy {
		t.Fatalf("Kind = %v, want SignalBuy: missing bar times must skip the weekday gate, not block", sig.Kind)
	}
}

func TestEnterUsesUnfilteredDailiesWithoutDailyTimes(t *testing.T) {
	md := fixture(trendCloses(3), mondayNoon)
	md.DailyTimes = nil
	sig := NewWithParams("TEST", DefaultParams()).Decide(md)
	if sig.Kind != model.SignalBuy {
		t.Fatalf("Kind = %v, want SignalBuy: missing daily times must not refuse the entry", sig.Kind)
	}
	if sig.ATR <= 0 || sig.ATR >= dailyWidth {
		t.Fatalf("ATR = %v, want in (0, %v): unfiltered dailies include the narrow weekend bars", sig.ATR, dailyWidth)
	}
}

func TestOpenPositionIsNeverReentered(t *testing.T) {
	md := fixture(trendCloses(3), mondayNoon)
	entry := md.Closes[len(md.Closes)-1]
	md.Position = &strategy.Position{PurchasePrice: entry, Quantity: 1, EntryATR: dailyWidth}
	if sig := NewWithParams("TEST", DefaultParams()).Decide(md); sig.Kind == model.SignalBuy {
		t.Fatal("an open position must never produce another Buy")
	}
}

// recoveryCloses is trendCloses(3) (the entry bar at index 402) followed by `upBars` bars of
// +0.3%. With RSI(4): upBars=3 crosses above 75 exactly on the last bar (69.75 -> 79.37);
// upBars=4 is already above it on the previous bar (79.37 -> 85.52).
func recoveryCloses(upBars int) []float64 {
	out := trendCloses(3)
	p := out[len(out)-1]
	for i := 0; i < upBars; i++ {
		p *= 1.003
		out = append(out, p)
	}
	return out
}

// openPosition is the long opened on trendCloses(3)'s last bar with the daily ATR frozen at
// entryATR.
func openPosition(entryATR float64) *strategy.Position {
	entry := trendCloses(3)[402]
	return &strategy.Position{
		PurchasePrice:         entry,
		Quantity:              1,
		StopLoss:              entry - entryATR,
		EntryATR:              entryATR,
		MaxFavorablePrice:     entry,
		PrevMaxFavorablePrice: entry,
	}
}

func TestRecoveryFixtureShape(t *testing.T) {
	for _, c := range []struct {
		up        int
		wantCross bool
	}{{3, true}, {4, false}} {
		closes := recoveryCloses(c.up)
		n := len(closes)
		rsi := indicators.RSISeries(closes, 4)
		if cross := rsi[n-2] <= 75 && rsi[n-1] > 75; cross != c.wantCross {
			t.Errorf("recoveryCloses(%d): RSI(4) %.2f -> %.2f, cross above 75 = %v, want %v", c.up, rsi[n-2], rsi[n-1], cross, c.wantCross)
		}
	}
}

func TestExitOnRSICrossUp(t *testing.T) {
	md := fixture(recoveryCloses(3), mondayNoon)
	md.Position = openPosition(dailyWidth)
	sig := NewWithParams("TEST", DefaultParams()).Decide(md)
	if sig.Kind != model.SignalSell || sig.Reason != "RSI" {
		t.Fatalf("Kind/Reason = %v/%q, want SignalSell/RSI", sig.Kind, sig.Reason)
	}
	if sig.RSI <= 75 {
		t.Errorf("RSI = %v, want > 75", sig.RSI)
	}
	if sig.ExitReason == "" {
		t.Error("ExitReason must explain the exit")
	}
}

func TestNoExitWhileRSIStaysAboveTheBand(t *testing.T) {
	md := fixture(recoveryCloses(4), mondayNoon)
	md.Position = openPosition(dailyWidth)
	if sig := NewWithParams("TEST", DefaultParams()).Decide(md); sig.Kind != model.SignalNone {
		t.Fatalf("Kind = %v, want SignalNone: RSI already above 75 on the previous bar is not a new cross", sig.Kind)
	}
}

func TestExitOnStopLoss(t *testing.T) {
	md := fixture(recoveryCloses(4), mondayNoon)
	pos := openPosition(dailyWidth)
	md.Position = pos
	level := pos.PurchasePrice - dailyWidth
	md.Lows[len(md.Lows)-1] = level - 0.01
	sig := NewWithParams("TEST", DefaultParams()).Decide(md)
	if sig.Kind != model.SignalSell || sig.Reason != "SL" {
		t.Fatalf("Kind/Reason = %v/%q, want SignalSell/SL", sig.Kind, sig.Reason)
	}
	if math.Abs(sig.StopLoss-level) > 1e-9 {
		t.Errorf("StopLoss = %v, want %v", sig.StopLoss, level)
	}
	if !model.IsStopReason(sig.Reason) {
		t.Error("SL must be a stop reason so the engine fills it at the stop level")
	}
	if sig.ExitReason == "" {
		t.Error("ExitReason must explain the exit")
	}
}

func TestStopWinsOverRSIExitOnTheSameBar(t *testing.T) {
	md := fixture(recoveryCloses(3), mondayNoon)
	pos := openPosition(dailyWidth)
	md.Position = pos
	md.Lows[len(md.Lows)-1] = pos.PurchasePrice - dailyWidth // touching the level counts
	if sig := NewWithParams("TEST", DefaultParams()).Decide(md); sig.Reason != "SL" {
		t.Fatalf("Reason = %q, want SL: the worse outcome wins a same-bar tie", sig.Reason)
	}
}

func TestStopUsesEntryATRNotTheCurrentOne(t *testing.T) {
	// Current daily ATR is 0.5 (level would be entry-0.5), frozen EntryATR is 2 (level entry-2).
	// A low between the two must NOT stop the trade.
	md := fixtureWithDaily(recoveryCloses(4), mondayNoon, 0.5)
	pos := openPosition(dailyWidth)
	md.Position = pos
	md.Lows[len(md.Lows)-1] = pos.PurchasePrice - 1.0
	if sig := NewWithParams("TEST", DefaultParams()).Decide(md); sig.Kind != model.SignalNone {
		t.Fatalf("Kind/Reason = %v/%q, want SignalNone: the stop is frozen at entry", sig.Kind, sig.Reason)
	}
}

func TestNoStopWithoutEntryATR(t *testing.T) {
	md := fixture(recoveryCloses(4), mondayNoon)
	md.Position = openPosition(0)
	md.Lows[len(md.Lows)-1] = 1
	if sig := NewWithParams("TEST", DefaultParams()).Decide(md); sig.Kind != model.SignalNone {
		t.Fatalf("Kind/Reason = %v/%q, want SignalNone: no EntryATR means no stop", sig.Kind, sig.Reason)
	}
	md = fixture(recoveryCloses(3), mondayNoon)
	md.Position = openPosition(0)
	if sig := NewWithParams("TEST", DefaultParams()).Decide(md); sig.Reason != "RSI" {
		t.Fatalf("Reason = %q, want RSI: the RSI exit still works without a stop", sig.Reason)
	}
}

func TestNoStopWhenDisabled(t *testing.T) {
	p := DefaultParams()
	p.StopDailyATR = 0
	md := fixture(recoveryCloses(4), mondayNoon)
	md.Position = openPosition(dailyWidth)
	md.Lows[len(md.Lows)-1] = 1
	if sig := NewWithParams("TEST", p).Decide(md); sig.Kind != model.SignalNone {
		t.Fatalf("Kind/Reason = %v/%q, want SignalNone", sig.Kind, sig.Reason)
	}
}

// zeroRSICloses is built for RSIPeriod=4: a long pure downtrend (constant -1 per bar) makes
// Wilder's avgGain seed at exactly zero and stay there, since every subsequent bar is also a
// loss (0*(p-1)/p + 0/p == 0) — the RSI reads exactly 0.00 for many consecutive bars, a genuine
// computed value, not warm-up. The final bar is a single large jump up, producing a real cross
// above 75 straight out of that zero: exactly the case crossedUp's old value-based guard
// misclassified as "still unset" (see TestExitOnRSICrossUpFromGenuineZero).
func zeroRSICloses() []float64 {
	closes := []float64{100}
	p := 100.0
	for i := 0; i < 30; i++ {
		p -= 1.0
		closes = append(closes, p)
	}
	closes = append(closes, p+15)
	return closes
}

func TestExitOnRSICrossUpFromGenuineZero(t *testing.T) {
	closes := zeroRSICloses()
	n := len(closes)
	rsi := indicators.RSISeries(closes, 4)
	if rsi[n-2] != 0 {
		t.Fatalf("fixture drift: rsi[n-2] = %v, want exactly 0 (a genuine post-warm-up reading)", rsi[n-2])
	}
	if rsi[n-1] <= 75 {
		t.Fatalf("fixture drift: rsi[n-1] = %v, want > 75 (a real cross above the upper band)", rsi[n-1])
	}

	md := fixture(closes, mondayNoon)
	// EntryATR: 0 disables the stop entirely (StopLevel returns 0 whenever entryATR <= 0), which
	// isolates the RSI branch from the SL branch regardless of how the fixture's lows are shaped.
	md.Position = &strategy.Position{PurchasePrice: closes[n-2], Quantity: 1, EntryATR: 0}
	sig := NewWithParams("TEST", DefaultParams()).Decide(md)
	if sig.Kind != model.SignalSell || sig.Reason != "RSI" {
		t.Fatalf("Kind/Reason = %v/%q, want SignalSell/RSI: a genuine RSI of exactly 0 after warm-up "+
			"is a real prior reading, and the next bar's cross above the upper band must fire the RSI exit",
			sig.Kind, sig.Reason)
	}
}

func TestManageDegrades(t *testing.T) {
	cases := []struct {
		name   string
		md     func() strategy.MarketData
		params func(p *Params)
	}{
		{"misaligned lows", func() strategy.MarketData {
			md := fixture(recoveryCloses(3), mondayNoon)
			md.Lows = md.Lows[:len(md.Lows)-1]
			return md
		}, nil},
		{"single bar", func() strategy.MarketData { return fixture(recoveryCloses(3)[:1], mondayNoon) }, nil},
		{"RSIPeriod 0", func() strategy.MarketData { return fixture(recoveryCloses(3), mondayNoon) }, func(p *Params) { p.RSIPeriod = 0 }},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			p := DefaultParams()
			if c.params != nil {
				c.params(&p)
			}
			md := c.md()
			md.Position = openPosition(dailyWidth)
			if sig := NewWithParams("TEST", p).Decide(md); sig.Kind != model.SignalNone {
				t.Fatalf("Kind/Reason = %v/%q, want SignalNone", sig.Kind, sig.Reason)
			}
		})
	}
}
