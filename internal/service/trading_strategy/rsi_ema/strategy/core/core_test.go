package core

import (
	"math/rand"
	"strings"
	"testing"
	"time"

	"tinvest/internal/service/trading_strategy/scalping/model"
	"tinvest/internal/service/trading_strategy/scalping/strategy"
	"tinvest/pkg/indicators"
)

// mskAt builds an MSK bar open-time for the given date and clock.
func mskAt(y int, m time.Month, d, hh, mm int) time.Time {
	return time.Date(y, m, d, hh, mm, 0, 0, mskLoc)
}

func TestDefaultParams(t *testing.T) {
	p := DefaultParams()
	if p.RSIPeriod != 12 || p.EMAFast != 10 || p.EMASlow != 50 {
		t.Fatalf("indicator defaults wrong: %+v", p)
	}
	if p.RSIMid != 50 || p.RSIUpper != 70 {
		t.Fatalf("zone defaults wrong: %+v", p)
	}
	if p.EntryCooldownBars != 1 {
		t.Fatalf("EntryCooldownBars = %d want 1", p.EntryCooldownBars)
	}
	if p.StopATR != 0 || p.ATRPeriod != 14 {
		t.Fatalf("risk defaults wrong: %+v", p)
	}
	if p.SessionStartMin != 420 || p.SessionEndMin != 1080 || p.FridayEndMin != 840 || p.DayEndMin != 1380 {
		t.Fatalf("session defaults wrong: %+v", p)
	}
}

func TestInSession(t *testing.T) {
	s := NewWithParams("TEST", DefaultParams())
	cases := []struct {
		name string
		at   time.Time
		want bool
	}{
		// 2026-07-20 Monday, 2026-07-24 Friday, 2026-07-25 Saturday.
		{"mon before open", mskAt(2026, 7, 20, 6, 55), false},
		{"mon at open", mskAt(2026, 7, 20, 7, 0), true},
		{"mon last bar", mskAt(2026, 7, 20, 17, 55), true},
		{"mon at close", mskAt(2026, 7, 20, 18, 0), false},
		{"fri before close", mskAt(2026, 7, 24, 13, 55), true},
		{"fri at close", mskAt(2026, 7, 24, 14, 0), false},
		{"saturday", mskAt(2026, 7, 25, 12, 0), false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := s.inSession(tc.at); got != tc.want {
				t.Fatalf("inSession(%v) = %v want %v", tc.at, got, tc.want)
			}
		})
	}
}

func TestInSessionZeroTimeIsPermissive(t *testing.T) {
	s := NewWithParams("TEST", DefaultParams())
	if !s.inSession(time.Time{}) {
		t.Fatalf("zero time must not block entry")
	}
}

func TestIsDayEnd(t *testing.T) {
	s := NewWithParams("TEST", DefaultParams())
	cases := []struct {
		name string
		at   time.Time
		span int
		want bool
	}{
		{"mon midday", mskAt(2026, 7, 20, 12, 0), 15, false},
		{"mon 18:05 evening held, not day-end", mskAt(2026, 7, 20, 18, 5), 15, false},
		{"mon 22:40", mskAt(2026, 7, 20, 22, 40), 15, false},
		{"mon 22:45 last 15m bar closes 23:00", mskAt(2026, 7, 20, 22, 45), 15, true},
		{"mon after 23:00", mskAt(2026, 7, 20, 23, 5), 15, true},
		{"fri day-end uniform 23:00", mskAt(2026, 7, 24, 22, 45), 15, true},
		{"fri 13:45 held into evening, not day-end", mskAt(2026, 7, 24, 13, 45), 15, false},
		{"saturday always day end", mskAt(2026, 7, 25, 10, 0), 15, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := s.isDayEnd(tc.at, tc.span); got != tc.want {
				t.Fatalf("isDayEnd(%v,%d) = %v want %v", tc.at, tc.span, got, tc.want)
			}
		})
	}
}

func TestBarSpanMinutesMedian(t *testing.T) {
	times := []time.Time{
		mskAt(2026, 7, 20, 10, 0),
		mskAt(2026, 7, 20, 10, 15),
		mskAt(2026, 7, 20, 10, 30),
		mskAt(2026, 7, 21, 10, 0), // overnight jump — must not skew the median
	}
	if got := barSpanMinutes(times); got != 15 {
		t.Fatalf("barSpanMinutes = %d want 15", got)
	}
	if got := barSpanMinutes(nil); got != defaultBarSpanMin {
		t.Fatalf("barSpanMinutes(nil) = %d want %d", got, defaultBarSpanMin)
	}
}

func TestBarTimeRequiresAlignedTimes(t *testing.T) {
	s := NewWithParams("TEST", DefaultParams())
	md := strategy.MarketData{Closes: []float64{1, 2, 3}, Times: []time.Time{mskAt(2026, 7, 20, 10, 0)}}
	if got := s.barTime(md); !got.IsZero() {
		t.Fatalf("misaligned Times must yield zero time, got %v", got)
	}
	md.Times = []time.Time{mskAt(2026, 7, 20, 10, 0), mskAt(2026, 7, 20, 10, 15), mskAt(2026, 7, 20, 10, 30)}
	if got := s.barTime(md); !got.Equal(mskAt(2026, 7, 20, 10, 30)) {
		t.Fatalf("barTime = %v want latest", got)
	}
}

func TestLookbackWarmsSlowEMA(t *testing.T) {
	s := NewWithParams("TEST", DefaultParams())
	if got := s.Lookback(); got < DefaultParams().EMASlow+20 {
		t.Fatalf("Lookback = %d too small to warm EMASlow", got)
	}
}

func TestTickerRoundTrip(t *testing.T) {
	if got := NewWithParams("SBER", DefaultParams()).Ticker(); got != "SBER" {
		t.Fatalf("Ticker = %q want SBER", got)
	}
}

// driftWalk builds a deterministic pseudo-random up-drift price walk. Over enough bars every
// RSI/EMA cross geometry occurs many times, so a bar where RSI crosses 50 up WHILE EMAFast is
// above EMASlow is guaranteed to exist. Fixed seed → reproducible.
func driftWalk(n int, seed int64) (closes, highs, lows []float64) {
	r := rand.New(rand.NewSource(seed))
	c := 100.0
	for i := 0; i < n; i++ {
		c += 0.05 + r.NormFloat64()*0.9
		if c < 5 {
			c = 5
		}
		closes = append(closes, c)
		highs = append(highs, c+0.3)
		lows = append(lows, c-0.3)
	}
	return closes, highs, lows
}

// mdEndingAt builds MarketData over closes[:k+1] with 15m MSK bars whose LAST bar opens at
// `end` (so barTime is controllable mid-session). pos may be nil.
func mdEndingAt(closes, highs, lows []float64, k int, end time.Time, pos *strategy.Position) strategy.MarketData {
	times := make([]time.Time, k+1)
	for i := 0; i <= k; i++ {
		times[i] = end.Add(time.Duration((i-k)*15) * time.Minute)
	}
	return strategy.MarketData{
		Price:    closes[k],
		Closes:   append([]float64(nil), closes[:k+1]...),
		Highs:    append([]float64(nil), highs[:k+1]...),
		Lows:     append([]float64(nil), lows[:k+1]...),
		Times:    times,
		Position: pos,
	}
}

// firstBuyBar scans Decide over a flat series and returns the first bar it marks as a Buy.
func firstBuyBar(t *testing.T, s *Strategy, closes, highs, lows []float64) int {
	t.Helper()
	for k := 60; k < len(closes); k++ {
		md := mdEndingAt(closes, highs, lows, k, mskAt(2026, 7, 20, 12, 0), nil)
		if s.Decide(md).Kind == model.SignalBuy {
			return k
		}
	}
	t.Fatalf("no entry bar found in series")
	return -1
}

func TestCrossedUp(t *testing.T) {
	s := []float64{0, 40, 55} // warm-up 0, then 40<50, then 55>50
	if !crossedUp(s, 2, 50) {
		t.Fatalf("40->55 must cross up through 50")
	}
	if crossedUp(s, 1, 50) {
		t.Fatalf("warm-up 0 must not read as below the level")
	}
	if crossedUp([]float64{55, 60}, 1, 50) {
		t.Fatalf("already above the level is not a fresh cross")
	}
}

func TestEnterBuysOnRSICrossWithEMAUp(t *testing.T) {
	s := NewWithParams("TEST", DefaultParams())
	closes, highs, lows := driftWalk(800, 1)
	k := firstBuyBar(t, s, closes, highs, lows)
	sig := s.Decide(mdEndingAt(closes, highs, lows, k, mskAt(2026, 7, 20, 12, 0), nil))
	if sig.Kind != model.SignalBuy {
		t.Fatalf("Kind = %v want Buy at bar %d", sig.Kind, k)
	}
	if sig.StopLoss != 0 {
		t.Fatalf("StopATR=0 must leave StopLoss zero, got %v", sig.StopLoss)
	}
	if sig.EntryReason == "" {
		t.Fatalf("EntryReason must be set on Buy")
	}
}

func TestEnterSetsATRStopWhenEnabled(t *testing.T) {
	p := DefaultParams()
	p.StopATR = 1.5
	s := NewWithParams("TEST", p)
	closes, highs, lows := driftWalk(800, 1)
	k := firstBuyBar(t, s, closes, highs, lows)
	sig := s.Decide(mdEndingAt(closes, highs, lows, k, mskAt(2026, 7, 20, 12, 0), nil))
	if sig.Kind != model.SignalBuy {
		t.Fatalf("want Buy, got %v", sig.Kind)
	}
	if sig.StopLoss <= 0 || sig.StopLoss >= closes[k] {
		t.Fatalf("StopLoss = %v must be positive and below entry %v", sig.StopLoss, closes[k])
	}
}

func TestEnterRejectsWhenEMANotWarmed(t *testing.T) {
	// EMASlow larger than the whole series never warms → EMA guard fails → never buys.
	p := DefaultParams()
	p.EMASlow = 100000
	s := NewWithParams("TEST", p)
	closes, highs, lows := driftWalk(400, 1)
	for k := 60; k < len(closes); k++ {
		if s.Decide(mdEndingAt(closes, highs, lows, k, mskAt(2026, 7, 20, 12, 0), nil)).Kind == model.SignalBuy {
			t.Fatalf("unwarmed EMASlow must never buy (bar %d)", k)
		}
	}
}

func TestEnterRejectsOutsideSession(t *testing.T) {
	s := NewWithParams("TEST", DefaultParams())
	closes, highs, lows := driftWalk(800, 1)
	k := firstBuyBar(t, s, closes, highs, lows)
	md := mdEndingAt(closes, highs, lows, k, mskAt(2026, 7, 20, 19, 0), nil) // after close
	if s.Decide(md).Kind == model.SignalBuy {
		t.Fatalf("entry outside session must be rejected")
	}
}

func TestEnterRejectsOnDayEndBar(t *testing.T) {
	s := NewWithParams("TEST", DefaultParams())
	closes, highs, lows := driftWalk(800, 1)
	k := firstBuyBar(t, s, closes, highs, lows)
	md := mdEndingAt(closes, highs, lows, k, mskAt(2026, 7, 20, 22, 50), nil) // last 15m bar of the day (23:00 close)
	if s.Decide(md).Kind == model.SignalBuy {
		t.Fatalf("entry on the day-end bar must be rejected")
	}
}

// TestEnterRejectedInEveningHoldWindow verifies the entry cutoff (18:00) is independent of the
// day-end boundary (23:00): after 18:00 no new longs open, even though positions may still be
// held and exited in that window.
func TestEnterRejectedInEveningHoldWindow(t *testing.T) {
	s := NewWithParams("TEST", DefaultParams())
	closes, highs, lows := driftWalk(800, 1)
	k := firstBuyBar(t, s, closes, highs, lows)
	md := mdEndingAt(closes, highs, lows, k, mskAt(2026, 7, 20, 20, 0), nil) // 20:00 — past entry cutoff, before day-end
	if s.Decide(md).Kind == model.SignalBuy {
		t.Fatalf("entry after the 18:00 cutoff must be rejected")
	}
}

// openPos builds an open long entered `entryBarsAgo` 15m bars before `end`.
func openPos(entry float64, end time.Time, entryBarsAgo int) *strategy.Position {
	return &strategy.Position{
		PurchasePrice: entry,
		Quantity:      1,
		EntryTime:     end.Add(time.Duration(-entryBarsAgo*15) * time.Minute),
	}
}

// firstExitBar scans Decide (with an open position entered entryBarsAgo bars ago) and returns
// the first bar whose exit Reason equals want. Because it reads Decide directly, the returned
// bar is one where `want` genuinely wins the exit precedence.
func firstExitBar(t *testing.T, s *Strategy, closes, highs, lows []float64, want string, entryBarsAgo int) int {
	t.Helper()
	end := mskAt(2026, 7, 20, 12, 0)
	for k := 60; k < len(closes); k++ {
		pos := openPos(closes[k], end, entryBarsAgo)
		if s.Decide(mdEndingAt(closes, highs, lows, k, end, pos)).Reason == want {
			return k
		}
	}
	t.Fatalf("no %s exit found in series", want)
	return -1
}

func TestCrossedDown(t *testing.T) {
	s := []float64{0, 60, 45} // warm-up 0, then 60>=50, then 45<50
	if !crossedDown(s, 2, 50) {
		t.Fatalf("60->45 must cross down through 50")
	}
	if crossedDown(s, 1, 50) {
		t.Fatalf("warm-up 0 must not count")
	}
}

func TestEmaCrossDown(t *testing.T) {
	fast := []float64{10, 11, 9}
	slow := []float64{10, 10, 10}
	if !emaCrossDown(fast, slow, 2) {
		t.Fatalf("fast 11>=10 then 9<10 must be a cross down")
	}
	if emaCrossDown([]float64{0, 9}, []float64{0, 10}, 1) {
		t.Fatalf("unwarmed (prev 0) must not count")
	}
}

func TestBarsSinceEntry(t *testing.T) {
	s := NewWithParams("TEST", DefaultParams())
	end := mskAt(2026, 7, 20, 12, 0)
	md := mdEndingAt([]float64{1, 2, 3}, []float64{1, 2, 3}, []float64{1, 2, 3}, 2, end, openPos(2, end, 3))
	if got := s.barsSinceEntry(md); got != 3 {
		t.Fatalf("barsSinceEntry = %d want 3", got)
	}
}

func TestBarsSinceEntryZeroEntryTimeDegrades(t *testing.T) {
	s := NewWithParams("TEST", DefaultParams())
	md := strategy.MarketData{
		Closes:   []float64{100, 100},
		Highs:    []float64{101, 101},
		Lows:     []float64{99, 99},
		Times:    []time.Time{mskAt(2026, 7, 20, 11, 45), mskAt(2026, 7, 20, 12, 0)},
		Position: &strategy.Position{PurchasePrice: 100}, // zero EntryTime
	}
	if got := s.barsSinceEntry(md); got < 1_000_000 {
		t.Fatalf("zero EntryTime must degrade to a large barsSinceEntry, got %d", got)
	}
}

func TestManageSLFillsWhenStopBreached(t *testing.T) {
	s := NewWithParams("TEST", DefaultParams())
	pos := openPos(100, mskAt(2026, 7, 20, 12, 0), 5)
	pos.StopLoss = 95
	md := mdEndingAt([]float64{100, 100, 100}, []float64{101, 101, 101}, []float64{99, 99, 94}, 2, mskAt(2026, 7, 20, 12, 0), pos)
	sig := s.Decide(md)
	if sig.Kind != model.SignalSell || sig.Reason != "SL" {
		t.Fatalf("want SL sell, got kind=%v reason=%q", sig.Kind, sig.Reason)
	}
}

func TestManageEODClosesOnLastBar(t *testing.T) {
	s := NewWithParams("TEST", DefaultParams())
	pos := openPos(100, mskAt(2026, 7, 20, 22, 50), 5) // day-end bar (23:00 close)
	md := mdEndingAt([]float64{100, 100, 100}, []float64{101, 101, 101}, []float64{99, 99, 99}, 2, mskAt(2026, 7, 20, 22, 50), pos)
	sig := s.Decide(md)
	if sig.Kind != model.SignalSell || sig.Reason != "EOD" {
		t.Fatalf("want EOD sell, got kind=%v reason=%q", sig.Kind, sig.Reason)
	}
}

// TestManageHoldsThroughEntryCutoff verifies a position opened inside the entry window is NOT
// force-closed at the old 18:00 boundary: with no exit signal on an 18:05 evening bar, it stays
// open until the 23:00 day-end.
func TestManageHoldsThroughEntryCutoff(t *testing.T) {
	s := NewWithParams("TEST", DefaultParams())
	pos := openPos(100, mskAt(2026, 7, 20, 18, 5), 5) // evening bar, past the 18:00 cutoff
	md := mdEndingAt([]float64{100, 100, 100}, []float64{101, 101, 101}, []float64{99, 99, 99}, 2, mskAt(2026, 7, 20, 18, 5), pos)
	if sig := s.Decide(md); sig.Kind == model.SignalSell {
		t.Fatalf("position must be held past the 18:00 cutoff, got sell reason=%q", sig.Reason)
	}
}

func TestManageEMAExitFires(t *testing.T) {
	s := NewWithParams("TEST", DefaultParams())
	closes, highs, lows := driftWalk(800, 2)
	k := firstExitBar(t, s, closes, highs, lows, "EMAX", 5)
	if s.Decide(mdEndingAt(closes, highs, lows, k, mskAt(2026, 7, 20, 12, 0), openPos(closes[k], mskAt(2026, 7, 20, 12, 0), 5))).Reason != "EMAX" {
		t.Fatalf("EMAX exit must reproduce at bar %d", k)
	}
}

func TestManageRSI70ExitUpwardIgnoresCooldown(t *testing.T) {
	p := DefaultParams()
	p.EntryCooldownBars = 100 // would suppress RSI50, must NOT affect RSI70
	s := NewWithParams("TEST", p)
	closes, highs, lows := driftWalk(800, 3)
	// entered 1 bar ago (inside the huge cooldown); RSI70 must still fire.
	k := firstExitBar(t, s, closes, highs, lows, "RSI70", 1)
	if k < 0 {
		t.Fatalf("expected an RSI70 exit despite cooldown")
	}
}

func TestManageRSI50ExitGatedByCooldown(t *testing.T) {
	p := DefaultParams() // cooldown 1
	s := NewWithParams("TEST", p)
	closes, highs, lows := driftWalk(800, 4)
	// find a bar where RSI50 wins with the cooldown satisfied (entered 50 bars ago).
	k := firstExitBar(t, s, closes, highs, lows, "RSI50", 50)

	// same bar, but entered 1 bar ago with a huge cooldown → RSI50 suppressed.
	p2 := p
	p2.EntryCooldownBars = 100
	s2 := NewWithParams("TEST", p2)
	end := mskAt(2026, 7, 20, 12, 0)
	md := mdEndingAt(closes, highs, lows, k, end, openPos(closes[k], end, 1))
	if got := s2.Decide(md).Reason; got == "RSI50" {
		t.Fatalf("RSI50 must be suppressed within cooldown, got %q", got)
	}
}

func TestExplainReportsGates(t *testing.T) {
	s := NewWithParams("TEST", DefaultParams())
	closes, highs, lows := driftWalk(800, 1)
	k := firstBuyBar(t, s, closes, highs, lows)
	out := s.Explain(mdEndingAt(closes, highs, lows, k, mskAt(2026, 7, 20, 12, 0), nil))
	for _, want := range []string{"сессия", "RSI", "EMA", "фильтр свежести"} {
		if !strings.Contains(out, want) {
			t.Fatalf("Explain output missing %q:\n%s", want, out)
		}
	}
}

func TestExplainShortSeriesDoesNotPanic(t *testing.T) {
	s := NewWithParams("TEST", DefaultParams())
	_ = s.Explain(strategy.MarketData{Closes: []float64{1}})
}

// freshParams returns the defaults with the bars-above-mid sub-filter explicitly ON.
// DefaultParams ships every entry quality filter off, so tests that exercise the filter must
// switch it on themselves.
func freshParams() Params {
	p := DefaultParams()
	p.EntryAboveMidLimit = 3
	return p
}

func TestDefaultParamsFreshEntryFilter(t *testing.T) {
	p := DefaultParams()
	if p.EntryLookbackBars != 5 {
		t.Fatalf("EntryLookbackBars = %d want 5", p.EntryLookbackBars)
	}
	if p.EntryAboveMidLimit != 0 {
		t.Fatalf("EntryAboveMidLimit = %d want 0 (quality filters are off by default)", p.EntryAboveMidLimit)
	}
}

func TestFreshEntryFilter(t *testing.T) {
	s := NewWithParams("TEST", freshParams()) // window 5, limit 3
	cases := []struct {
		name string
		rsi  []float64 // indices 0..i-1 form the window when i = len(rsi)
		want bool      // true = entry allowed
	}{
		// 5-bar window ending just before the cross bar (i = len(rsi)).
		{"chop: 4 of 5 above -> reject", []float64{55, 55, 55, 55, 47}, false},
		{"fresh: 1 of 5 above -> allow", []float64{55, 47, 47, 47, 47}, true},
		{"boundary: exactly limit-1 (2) above -> allow", []float64{55, 55, 47, 47, 47}, true},
		{"boundary: exactly limit (3) above -> reject", []float64{55, 55, 55, 47, 47}, false},
		{"short history truncates, no panic", []float64{55, 47}, true}, // only 2 bars, 1 above < 3
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := s.freshEntry(tc.rsi, len(tc.rsi)); got != tc.want {
				t.Fatalf("freshEntry=%v want %v (above=%d)", got, tc.want, s.barsAboveMid(tc.rsi, len(tc.rsi)))
			}
		})
	}
}

// TestBarsAboveMidExcludesEntryBar pins that the entry bar (index i) and beyond lie OUTSIDE the
// window and must never be counted. Every case in TestFreshEntryFilter calls barsAboveMid/
// freshEntry with i == len(rsi), so the `j < i && j < len(rsi)` guard is indistinguishable from
// `j <= i` there — this case puts values ABOVE RSIMid at i and beyond while the window itself
// sits entirely at/below RSIMid, so the two are distinguishable.
func TestBarsAboveMidExcludesEntryBar(t *testing.T) {
	s := NewWithParams("TEST", DefaultParams()) // window 5, RSIMid 50
	// Window (indices 0..4): 40,40,40,40,40 — none above the mid. Index 5 (the entry bar) and
	// index 6 (beyond) sit above the mid but must not be counted.
	if got := s.barsAboveMid([]float64{40, 40, 40, 40, 40, 60, 60}, 5); got != 0 {
		t.Fatalf("entry bar and beyond must not be counted: barsAboveMid = %d want 0", got)
	}
}

func TestFreshEntryFilterDisabled(t *testing.T) {
	p := freshParams()
	p.EntryAboveMidLimit = 0 // off
	s := NewWithParams("TEST", p)
	if !s.freshEntry([]float64{55, 55, 55, 55, 47}, 5) {
		t.Fatalf("EntryAboveMidLimit<=0 must disable the filter")
	}
	p2 := freshParams()
	p2.EntryLookbackBars = 0 // off
	s2 := NewWithParams("TEST", p2)
	if !s2.freshEntry([]float64{55, 55, 55, 55, 47}, 5) {
		t.Fatalf("EntryLookbackBars<=0 must disable the filter")
	}
}

// TestEnterFilterRejectsChopReentry finds a bar that the filter-OFF strategy buys but where the
// preceding window holds >= EntryAboveMidLimit bars above the mid, and asserts the default
// (filter-ON) strategy rejects that exact bar.
func TestEnterFilterRejectsChopReentry(t *testing.T) {
	sOff := NewWithParams("TEST", DefaultParams()) // defaults: every quality filter off
	on := NewWithParams("TEST", freshParams())     // filter on (window 5, limit 3)
	closes, highs, lows := driftWalk(1500, 7)
	end := mskAt(2026, 7, 20, 12, 0)
	for k := 60; k < len(closes); k++ {
		md := mdEndingAt(closes, highs, lows, k, end, nil)
		if sOff.Decide(md).Kind != model.SignalBuy {
			continue
		}
		rsi := indicators.RSISeries(md.Closes, DefaultParams().RSIPeriod)
		if on.barsAboveMid(rsi, k) >= freshParams().EntryAboveMidLimit {
			if on.Decide(md).Kind == model.SignalBuy {
				t.Fatalf("chop re-entry at bar %d must be filtered out", k)
			}
			return // found and verified a chop candidate
		}
	}
	t.Fatalf("no chop re-entry candidate found in series (adjust the driftWalk seed)")
}

func TestDefaultParamsChopFilterOff(t *testing.T) {
	if got := DefaultParams().EntryMaxMidCrossings; got != 0 {
		t.Fatalf("EntryMaxMidCrossings = %d want 0 (filter off by default)", got)
	}
}

func TestMidCrossings(t *testing.T) {
	s := NewWithParams("TEST", DefaultParams()) // window 5, RSIMid 50
	cases := []struct {
		name string
		rsi  []float64 // indices 0..i-1 form the window when i = len(rsi)
		want int
	}{
		{"saw crosses three times", []float64{52, 48, 51, 49, 49}, 3},
		{"quiet below the mid", []float64{48, 49, 49, 49, 49}, 0},
		{"single cross down", []float64{55, 55, 55, 55, 47}, 1},
		{"warm-up zeros do not count", []float64{0, 0, 0, 55, 47}, 1},
		{"short history truncates, no panic", []float64{55, 47}, 1},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := s.midCrossings(tc.rsi, len(tc.rsi)); got != tc.want {
				t.Fatalf("midCrossings = %d want %d", got, tc.want)
			}
		})
	}
}

// TestMidCrossingsExcludesEntryCross pins that the entry cross (i-1 -> i) lies OUTSIDE the
// window and must never be counted. Every case above calls midCrossings with i == len(rsi), so
// the `end > len(rsi)` clamp makes `end = i` and a regressed `end = i + 1` behave identically —
// this case uses data PAST index i so the two are distinguishable.
func TestMidCrossingsExcludesEntryCross(t *testing.T) {
	s := NewWithParams("TEST", DefaultParams()) // window 5, RSIMid 50
	// Window (indices 0..4): 48, 49, 49, 49, 48 — all below the mid, 0 internal crossings.
	// The entry cross 48(i-1=4) -> 55(i=5) is the cross itself, not part of the window.
	if got := s.midCrossings([]float64{48, 49, 49, 49, 48, 55}, 5); got != 0 {
		t.Fatalf("entry cross must not be counted: midCrossings = %d want 0", got)
	}
}

func TestChopFilterRejectsSaw(t *testing.T) {
	p := DefaultParams()
	p.EntryMaxMidCrossings = 3
	s := NewWithParams("TEST", p)
	// saw crosses the mid line 3 times, twoCross only twice.
	saw := []float64{52, 48, 51, 49, 49}
	twoCross := []float64{52, 48, 51, 51, 51}
	if s.freshEntry(saw, len(saw)) {
		t.Fatalf("saw with 3 >= 3 crossings must be rejected")
	}
	if !s.freshEntry(twoCross, len(twoCross)) {
		t.Fatalf("2 < 3 crossings must be allowed")
	}
}

func TestChopFilterDisabled(t *testing.T) {
	saw := []float64{52, 48, 51, 49, 49}
	s := NewWithParams("TEST", DefaultParams()) // EntryMaxMidCrossings = 0
	if !s.freshEntry(saw, len(saw)) {
		t.Fatalf("EntryMaxMidCrossings<=0 must disable the chop filter")
	}
	p := DefaultParams()
	p.EntryMaxMidCrossings = 3
	p.EntryLookbackBars = 0 // the shared window switch disables both sub-filters
	if !NewWithParams("TEST", p).freshEntry(saw, len(saw)) {
		t.Fatalf("EntryLookbackBars<=0 must disable both sub-filters")
	}
}

// TestEntrySubFiltersAreIndependent pins that each sub-filter cuts only its own pattern: the
// chop filter rejects the saw and passes the dip-after-a-run, the bars-above-mid filter does
// the opposite.
func TestEntrySubFiltersAreIndependent(t *testing.T) {
	saw := []float64{52, 48, 51, 49, 49} // 3 crossings, 2 bars above
	dip := []float64{55, 55, 55, 55, 47} // 1 crossing, 4 bars above

	chopOnly := DefaultParams()
	chopOnly.EntryMaxMidCrossings = 3
	sChop := NewWithParams("TEST", chopOnly)
	if sChop.freshEntry(saw, len(saw)) {
		t.Fatalf("chop filter must reject the saw")
	}
	if !sChop.freshEntry(dip, len(dip)) {
		t.Fatalf("chop filter must not reject the dip (only 1 crossing)")
	}

	aboveOnly := freshParams() // EntryAboveMidLimit = 3, EntryMaxMidCrossings = 0
	sAbove := NewWithParams("TEST", aboveOnly)
	if sAbove.freshEntry(dip, len(dip)) {
		t.Fatalf("bars-above-mid filter must reject the dip")
	}
	if !sAbove.freshEntry(saw, len(saw)) {
		t.Fatalf("bars-above-mid filter must not reject the saw (only 2 bars above)")
	}
}

// TestEnterFilterRejectsSawEntry finds a bar the defaults (all filters off) buy but whose
// preceding window holds >= EntryMaxMidCrossings mid-line crossings, and asserts the chop-ON
// strategy rejects that exact bar.
func TestEnterFilterRejectsSawEntry(t *testing.T) {
	off := NewWithParams("TEST", DefaultParams()) // every quality filter off
	chop := DefaultParams()
	chop.EntryMaxMidCrossings = 3
	on := NewWithParams("TEST", chop)
	closes, highs, lows := driftWalk(1500, 7)
	end := mskAt(2026, 7, 20, 12, 0)
	for k := 60; k < len(closes); k++ {
		md := mdEndingAt(closes, highs, lows, k, end, nil)
		if off.Decide(md).Kind != model.SignalBuy {
			continue
		}
		rsi := indicators.RSISeries(md.Closes, DefaultParams().RSIPeriod)
		if on.midCrossings(rsi, k) >= chop.EntryMaxMidCrossings {
			if on.Decide(md).Kind == model.SignalBuy {
				t.Fatalf("saw entry at bar %d must be filtered out", k)
			}
			return // found and verified a saw candidate
		}
	}
	t.Fatalf("no saw candidate found in series (adjust the driftWalk seed)")
}

func TestExplainReportsChopFilter(t *testing.T) {
	p := DefaultParams()
	p.EntryMaxMidCrossings = 3
	s := NewWithParams("TEST", p)
	closes, highs, lows := driftWalk(800, 1)
	out := s.Explain(mdEndingAt(closes, highs, lows, 200, mskAt(2026, 7, 20, 12, 0), nil))
	if !strings.Contains(out, "фильтр пилы") {
		t.Fatalf("Explain must report the chop filter:\n%s", out)
	}
	sOff := NewWithParams("TEST", DefaultParams())
	outOff := sOff.Explain(mdEndingAt(closes, highs, lows, 200, mskAt(2026, 7, 20, 12, 0), nil))
	// The generic "выключен" fragment also appears unconditionally in the volume-gate and
	// stop lines at defaults, so it cannot discriminate whether the chop filter's own
	// disabled-label path ran. Pin the chop filter's specific line instead.
	var chopLine string
	for _, line := range strings.Split(outOff, "\n") {
		if strings.Contains(line, "фильтр пилы") {
			chopLine = line
			break
		}
	}
	if chopLine == "" {
		t.Fatalf("Explain must report the chop filter line when off:\n%s", outOff)
	}
	if !strings.Contains(chopLine, "50 в окне 5") || !strings.Contains(chopLine, "(лимит выключен)") {
		t.Fatalf("chop filter disabled-label line wrong: %q\nfull output:\n%s", chopLine, outOff)
	}
}

func TestDefaultParamsVolumeFilterOff(t *testing.T) {
	p := DefaultParams()
	if p.UseVolume != 0 {
		t.Fatalf("UseVolume = %d want 0 (filter off by default)", p.UseVolume)
	}
	if p.VolShortPeriod != 10 || p.VolLongPeriod != 50 || p.VolMult != 1.0 {
		t.Fatalf("volume defaults wrong: %+v", p)
	}
}

func TestAvgVolumeLastN(t *testing.T) {
	vols := []int64{100, 200, 300, 400}
	if avg, ok := avgVolumeLastN(vols, nil, 2); !ok || avg != 350 {
		t.Fatalf("avg = %v ok = %v want 350 true", avg, ok)
	}
	if avg, ok := avgVolumeLastN(vols, nil, 10); !ok || avg != 250 {
		t.Fatalf("window longer than the series must truncate: avg = %v ok = %v want 250 true", avg, ok)
	}
	if avg, ok := avgVolumeLastN([]int64{0, 0, 300}, nil, 3); !ok || avg != 300 {
		t.Fatalf("non-positive volumes must be ignored: avg = %v ok = %v want 300 true", avg, ok)
	}
	if _, ok := avgVolumeLastN(nil, nil, 5); ok {
		t.Fatalf("empty series must report ok=false")
	}
	if _, ok := avgVolumeLastN([]int64{0, 0}, nil, 2); ok {
		t.Fatalf("no positive sample must report ok=false")
	}
	if _, ok := avgVolumeLastN(vols, nil, 0); ok {
		t.Fatalf("non-positive window must report ok=false")
	}
}

func TestAvgVolumeLastNExcludesWeekends(t *testing.T) {
	// 2026-07-25 is a Saturday; its bar must be dropped when Times is aligned.
	times := []time.Time{
		mskAt(2026, 7, 24, 12, 0),
		mskAt(2026, 7, 25, 12, 0), // Saturday
		mskAt(2026, 7, 27, 12, 0),
	}
	vols := []int64{100, 9000, 300}
	avg, ok := avgVolumeLastN(vols, times, 3)
	if !ok || avg != 200 {
		t.Fatalf("weekend bar must be excluded: avg = %v ok = %v want 200 true", avg, ok)
	}
	// Misaligned Times → weekend exclusion is skipped, all bars count.
	avg2, ok2 := avgVolumeLastN(vols, times[:2], 3)
	if !ok2 || avg2 != (100+9000+300)/3.0 {
		t.Fatalf("misaligned Times must skip weekend exclusion: avg = %v ok = %v", avg2, ok2)
	}
}

// withVolumes attaches a volume series to md: the last `short` bars carry `recent`, every
// earlier bar carries `background`.
func withVolumes(md strategy.MarketData, background, recent int64, short int) strategy.MarketData {
	n := len(md.Closes)
	vols := make([]int64, n)
	for i := range vols {
		vols[i] = background
		if i >= n-short {
			vols[i] = recent
		}
	}
	md.Volumes = vols
	return md
}

func TestVolumeRegimeGate(t *testing.T) {
	closes, highs, lows := driftWalk(300, 1)
	base := mdEndingAt(closes, highs, lows, 250, mskAt(2026, 7, 20, 12, 0), nil)

	on := DefaultParams()
	on.UseVolume = 1
	sOn := NewWithParams("TEST", on)
	// Recent 10 bars at 12000 over an 8000 background: the long window mixes both, so
	// shortAvg (12000) clears longAvg (~8800) comfortably.
	if !sOn.volumeRegimeOK(withVolumes(base, 8000, 12000, 10)) {
		t.Fatalf("a live tape must pass the volume gate")
	}
	// Recent 10 bars at 5000: shortAvg falls well under the ~7400 background.
	if sOn.volumeRegimeOK(withVolumes(base, 8000, 5000, 10)) {
		t.Fatalf("a fading tape must be rejected")
	}
	// VolMult 1.2 with a barely-elevated tape (9000 vs a ~8200 long average): 9000 clears the
	// plain average but not the 1.2x threshold.
	mult := on
	mult.VolMult = 1.2
	if NewWithParams("TEST", mult).volumeRegimeOK(withVolumes(base, 8000, 9000, 10)) {
		t.Fatalf("VolMult 1.2 must reject a merely-flat tape")
	}
}

func TestVolumeRegimeGateDegrades(t *testing.T) {
	closes, highs, lows := driftWalk(300, 1)
	base := mdEndingAt(closes, highs, lows, 250, mskAt(2026, 7, 20, 12, 0), nil)
	fading := withVolumes(base, 8000, 5000, 10) // would be rejected when armed

	if !NewWithParams("TEST", DefaultParams()).volumeRegimeOK(fading) {
		t.Fatalf("UseVolume=0 must disable the gate")
	}

	on := DefaultParams()
	on.UseVolume = 1
	if !NewWithParams("TEST", on).volumeRegimeOK(base) {
		t.Fatalf("missing Volumes must never block an entry")
	}

	bad := on
	bad.VolLongPeriod = bad.VolShortPeriod // long window must exceed short
	if !NewWithParams("TEST", bad).volumeRegimeOK(fading) {
		t.Fatalf("VolLongPeriod <= VolShortPeriod must disable the gate")
	}

	bad2 := on
	bad2.VolShortPeriod = 0 // VolLongPeriod (50) stays > VolShortPeriod, so only this disjunct fires
	if !NewWithParams("TEST", bad2).volumeRegimeOK(fading) {
		t.Fatalf("VolShortPeriod <= 0 must disable the gate")
	}

	zeroVols := base
	zeroVols.Volumes = make([]int64, len(base.Closes)) // all zero → no usable sample
	if !NewWithParams("TEST", on).volumeRegimeOK(zeroVols) {
		t.Fatalf("an all-zero volume series must never block an entry")
	}
}

// TestEnterVolumeGateWiredOnEntryPath proves the gate sits on the real entry path: the same bar
// that the defaults buy is rejected once the gate is armed against a fading tape, and still
// bought when the tape is alive.
func TestEnterVolumeGateWiredOnEntryPath(t *testing.T) {
	closes, highs, lows := driftWalk(800, 1)
	sDef := NewWithParams("TEST", DefaultParams())
	k := firstBuyBar(t, sDef, closes, highs, lows)
	md := mdEndingAt(closes, highs, lows, k, mskAt(2026, 7, 20, 12, 0), nil)

	if sDef.Decide(withVolumes(md, 8000, 5000, 10)).Kind != model.SignalBuy {
		t.Fatalf("defaults must ignore the volume background")
	}

	on := DefaultParams()
	on.UseVolume = 1
	sOn := NewWithParams("TEST", on)
	if sOn.Decide(withVolumes(md, 8000, 5000, 10)).Kind == model.SignalBuy {
		t.Fatalf("armed gate must reject an entry into a fading tape")
	}
	if sOn.Decide(withVolumes(md, 8000, 12000, 10)).Kind != model.SignalBuy {
		t.Fatalf("armed gate must allow an entry on a live tape")
	}
}

func TestExplainReportsVolumeGate(t *testing.T) {
	closes, highs, lows := driftWalk(300, 1)
	base := mdEndingAt(closes, highs, lows, 250, mskAt(2026, 7, 20, 12, 0), nil)

	if out := NewWithParams("TEST", DefaultParams()).Explain(base); !strings.Contains(out, "фон объёмов: выключен") {
		t.Fatalf("Explain must mark the volume gate as off:\n%s", out)
	}

	on := DefaultParams()
	on.UseVolume = 1
	sOn := NewWithParams("TEST", on)
	if out := sOn.Explain(withVolumes(base, 8000, 12000, 10)); !strings.Contains(out, "отношение") {
		t.Fatalf("Explain must report the volume ratio:\n%s", out)
	}
	if out := sOn.Explain(base); !strings.Contains(out, "нет данных") {
		t.Fatalf("Explain must report missing volume data:\n%s", out)
	}
}

// TestExplainReportsVolumeMisconfiguration pins that Explain reports a distinct "некорректные
// окна" line — not the generic "нет данных" (data problem) — for both ways the volume gate can
// be misconfigured, mirroring volumeRegimeOK's own disable predicate.
func TestExplainReportsVolumeMisconfiguration(t *testing.T) {
	closes, highs, lows := driftWalk(300, 1)
	base := mdEndingAt(closes, highs, lows, 250, mskAt(2026, 7, 20, 12, 0), nil)

	longNotExceedingShort := DefaultParams()
	longNotExceedingShort.UseVolume = 1
	longNotExceedingShort.VolLongPeriod = longNotExceedingShort.VolShortPeriod
	out := NewWithParams("TEST", longNotExceedingShort).Explain(withVolumes(base, 8000, 12000, 10))
	if !strings.Contains(out, "некорректные окна") {
		t.Fatalf("Explain must flag VolLongPeriod<=VolShortPeriod as misconfigured:\n%s", out)
	}
	if strings.Contains(out, "нет данных") || strings.Contains(out, "отношение") {
		t.Fatalf("misconfiguration must not be reported as missing data or a real ratio verdict:\n%s", out)
	}

	nonPositiveShort := DefaultParams()
	nonPositiveShort.UseVolume = 1
	nonPositiveShort.VolShortPeriod = 0
	out2 := NewWithParams("TEST", nonPositiveShort).Explain(withVolumes(base, 8000, 12000, 10))
	if !strings.Contains(out2, "некорректные окна") {
		t.Fatalf("Explain must flag VolShortPeriod<=0 as misconfigured:\n%s", out2)
	}
}
