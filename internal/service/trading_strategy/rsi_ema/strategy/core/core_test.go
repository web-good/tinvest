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
