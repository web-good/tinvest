package core

import (
	"math"
	"slices"
	"strings"
	"testing"
	"time"

	"tinvest/internal/service/trading_strategy/scalping/model"
	"tinvest/internal/service/trading_strategy/scalping/strategy"
	"tinvest/pkg/indicators"
)

const eps = 1e-9

type bar struct {
	t          time.Time
	o, h, l, c float64
}

// path chains 15m bars from MSK `from`: each opens at the previous close (the first at o0) and
// closes at the next value; highs and lows hug the bodies.
func path(from string, o0 float64, closes ...float64) []bar {
	var out []bar
	o := o0
	for k, c := range closes {
		out = append(out, bar{at(from).Add(time.Duration(k) * step), o, max(o, c), min(o, c), c})
		o = c
	}
	return out
}

// withLow returns day with bar i's low set to low (a lower tail).
func withLow(day []bar, i int, low float64) []bar {
	out := slices.Clone(day)
	out[i].l = low
	return out
}

// shifted returns day moved by d.
func shifted(day []bar, d time.Duration) []bar {
	out := slices.Clone(day)
	for i := range out {
		out[i].t = out[i].t.Add(d)
	}
	return out
}

// dailyBars returns `days` calendar days of completed dailies ending the day before `before`, all
// closing at 100. Weekdays span 99..101 (PDL 99, Wilder ATR exactly 2); weekend sessions span
// 99.9..100.1.
func dailyBars(before time.Time, days int) (highs, lows, closes []float64, times []time.Time) {
	b := before.In(mskLoc)
	start := time.Date(b.Year(), b.Month(), b.Day(), 0, 0, 0, 0, mskLoc).AddDate(0, 0, -days)
	for i := 0; i < days; i++ {
		d := start.AddDate(0, 0, i)
		half := 1.0
		if d.Weekday() == time.Saturday || d.Weekday() == time.Sunday {
			half = 0.1
		}
		highs = append(highs, 100+half)
		lows = append(lows, 100-half)
		closes = append(closes, 100)
		times = append(times, d.Add(10*time.Hour))
	}
	return highs, lows, closes, times
}

// warmBars is how many flat bars warm prepends: more than a weekend of 15m bars, so the volume
// baseline and both RSIs are warm (a flat RSI is 50: it neither arms nor crosses).
const warmBars = 1000

// warm prepends warmBars flat bars at the day's first open ending one bar before day[0]. Like a
// real exchange day the bars start at 06:45 MSK, so no warm date reads as a dealer date.
func warm(day []bar) []bar {
	o := day[0].o
	out := make([]bar, warmBars)
	t := day[0].t
	for k := warmBars - 1; k >= 0; {
		t = t.Add(-step)
		if minuteOf(t) < exchangeOpen {
			continue
		}
		out[k] = bar{t, o, o, o, o}
		k--
	}
	return append(out, day...)
}

// fixture builds MarketData from bars plus 40 days of dailies before the last bar (PDL 99, ATR 2).
// Every bar has volume 100; the bars from `loud` on have 200 (zero loud — none).
func fixture(bars []bar, loud time.Time) strategy.MarketData {
	var md strategy.MarketData
	for _, b := range bars {
		md.Times = append(md.Times, b.t)
		md.Opens = append(md.Opens, b.o)
		md.Highs = append(md.Highs, b.h)
		md.Lows = append(md.Lows, b.l)
		md.Closes = append(md.Closes, b.c)
		v := int64(100)
		if !loud.IsZero() && !b.t.Before(loud) {
			v = 200
		}
		md.Volumes = append(md.Volumes, v)
	}
	md.Price = md.Closes[len(md.Closes)-1]
	md.DailyHighs, md.DailyLows, md.DailyCloses, md.DailyTimes = dailyBars(md.Times[len(md.Times)-1], 40)
	return md
}

// run is Decide on the warmed bars up to and including day[i].
func run(p Params, day []bar, i int, loud time.Time) model.Signal {
	return NewWithParams("TEST", step, p).Decide(fixture(warm(day)[:warmBars+i+1], loud))
}

// buys lists the indexes in day whose bar Decide turns into a buy.
func buys(p Params, day []bar, loud time.Time) []int {
	var out []int
	for i := range day {
		if run(p, day, i, loud).Kind == model.SignalBuy {
			out = append(out, i)
		}
	}
	return out
}

// rsi15 is the 15m RSI of the warmed day at the day's indexes.
func rsi15(p Params, day []bar) []float64 {
	all := warm(day)
	closes := make([]float64, len(all))
	for i, b := range all {
		closes[i] = b.c
	}
	return indicators.RSISeries(closes, p.RSIPeriod)[warmBars:]
}

// hourRSI maps the day index of the bar completing an hour to that hour's RSI.
func hourRSI(p Params, day []bar) map[int]float64 {
	md := fixture(warm(day), time.Time{})
	hs := NewWithParams("TEST", step, p).hourBars(md)
	rh := indicators.RSISeries(hourCloses(hs), p.RSIPeriod)
	out := map[int]float64{}
	for h := range hs {
		if hs[h].done >= warmBars {
			out[hs[h].done-warmBars] = rh[h]
		}
	}
	return out
}

// crossesAt reports whether the 15m RSI crosses RSIEntry up on day[i].
func crossesAt(p Params, day []bar, i int) bool {
	r := rsi15(p, day)
	return crossUp(r[i-1], r[i], p.RSIEntry)
}

// inZone reports whether day[i]'s low lies in PDL 99 ± ZoneATR × ATR 2.
func inZone(p Params, day []bar, i int) bool {
	return day[i].l >= 99-2*p.ZoneATR-eps && day[i].l <= 99+2*p.ZoneATR+eps
}

// canonical is Tuesday 2026-09-15 from 10:00: a flat hour, an hour sliding to 99.2 (the hourly RSI
// under 30 at its 11:45 close arms the setup), a dip with a 98.9 tail at 12:00, the 15m RSI
// crossing 30 up on the 12:30 bar (low 99.0 at PDL), then a climb.
func canonical() []bar {
	day := path("2026-09-15 10:00", 100, 100, 100, 100, 100.1, 99.8, 99.6, 99.4, 99.2,
		99.1, 99.0, 99.3, 99.5, 99.7, 99.9, 100.1, 100.3)
	return withLow(day, 8, 98.9)
}

const (
	canonFrom  = 4  // 11:00, the arming hour's first bar
	canonArm   = 7  // 11:45, the bar completing the arming hour
	canonCross = 10 // 12:30
)

// loudFrom is the start of day[i]: the bars from it on are loud.
func loudFrom(day []bar, i int) time.Time { return day[i].t }

// checkCanonical asserts the canonical fixture does what its comment says.
func checkCanonical(t *testing.T, p Params, day []bar) {
	t.Helper()
	rh := hourRSI(p, day)
	if v, ok := rh[canonArm]; !ok || v >= p.RSIArm {
		t.Fatalf("fixture: hour completing at %d has RSI %v (ok %v), want < %v", canonArm, v, ok, p.RSIArm)
	}
	if v := rh[canonArm-4]; v < p.RSIArm {
		t.Fatalf("fixture: the 10:00 hour RSI %v must not arm", v)
	}
	r := rsi15(p, day)
	for j := canonArm + 1; j < len(day); j++ {
		if crossUp(r[j-1], r[j], p.RSIEntry) != (j == canonCross) {
			t.Fatalf("fixture: 15m RSI cross at %d = %v (rsi %v → %v)", j, j != canonCross, r[j-1], r[j])
		}
	}
	if !inZone(p, day, canonCross) {
		t.Fatalf("fixture: cross bar low %v outside the zone", day[canonCross].l)
	}
}

// wantStop is the min low of window bars from..k-1 with 15m RSI < RSIEntry and of bar k.
func wantStop(p Params, day []bar, from, k int) float64 {
	r := rsi15(p, day)
	low := day[k].l
	for j := from; j < k; j++ {
		if r[j] < p.RSIEntry {
			low = min(low, day[j].l)
		}
	}
	return low
}

func TestEntryOnFifteenCrossInZone(t *testing.T) {
	p, day := DefaultParams(), canonical()
	checkCanonical(t, p, day)
	loud := loudFrom(day, canonFrom)
	if got := buys(p, day, loud); !slices.Equal(got, []int{canonCross}) {
		t.Fatalf("buys = %v, want [%d]", got, canonCross)
	}
	sig := run(p, day, canonCross, loud)
	if want := wantStop(p, day, canonFrom, canonCross); math.Abs(sig.StopLoss-want) > eps || math.Abs(want-98.9) > eps {
		t.Fatalf("StopLoss = %v, want %v (the 12:00 tail 98.9)", sig.StopLoss, want)
	}
	if math.Abs(sig.Level-99) > eps || sig.TakeProfit != 0 || math.Abs(sig.ATR-2) > eps {
		t.Fatalf("Level/TakeProfit/ATR = %v/%v/%v, want 99/0/2", sig.Level, sig.TakeProfit, sig.ATR)
	}
	if !sig.FillBy.Equal(day[canonCross].t.Add(step)) {
		t.Fatalf("FillBy = %v, want the next bar's open", sig.FillBy)
	}
	for _, s := range []string{"11:00", "12:30", "PDL", "стоп", "объём", "запас хода"} {
		if !strings.Contains(sig.EntryReason, s) {
			t.Errorf("EntryReason %q lacks %q", sig.EntryReason, s)
		}
	}
}

func TestStopIncludesSignalBarLow(t *testing.T) {
	p, day := DefaultParams(), withLow(canonical(), canonCross, 98.8)
	checkCanonical(t, p, day)
	sig := run(p, day, canonCross, loudFrom(day, canonFrom))
	if sig.Kind != model.SignalBuy || math.Abs(sig.StopLoss-98.8) > eps {
		t.Fatalf("Kind %v StopLoss = %v, want a buy with the cross bar's low 98.8", sig.Kind, sig.StopLoss)
	}
}

func TestStopOf(t *testing.T) {
	ser := series{l: []float64{5, 3, 1, 4, 2}}
	r15 := []float64{10, 20, 40, 25, 35}
	// bars 0..3: RSI < 30 at 0, 1, 3 (the 1.0 low has RSI 40 and is skipped); signal bar 4 low 2.
	if got := stopOf(ser, r15, setup{from: 0, arm: 0, signal: 4}, 30); got != 2 {
		t.Fatalf("stopOf = %v, want 2", got)
	}
	if got := stopOf(ser, r15, setup{from: 3, arm: 3, signal: 4}, 30); got != 2 {
		t.Fatalf("stopOf from 3 = %v, want 2", got)
	}
}

func TestNoArmNoEntry(t *testing.T) {
	p, day := DefaultParams(), canonical()
	p.RSIArm = 5
	for j, v := range hourRSI(p, day) {
		if v < p.RSIArm {
			t.Fatalf("fixture: hour completing at %d has RSI %v < 5", j, v)
		}
	}
	if got := buys(p, day, loudFrom(day, canonFrom)); len(got) != 0 {
		t.Fatalf("buys = %v, want none", got)
	}
}

func TestSignalBeforeArmingHourCompletesIgnored(t *testing.T) {
	// The 15m RSI dips and crosses 30 on the 11:30 bar, inside the hour whose close arms the setup.
	p := DefaultParams()
	day := path("2026-09-15 10:00", 100, 100, 100, 100, 100.1, 99.6, 99.2, 99.6, 99.0,
		99.4, 99.6, 99.8, 100.0)
	const early = 6
	if !crossesAt(p, day, early) || !inZone(p, day, early) {
		t.Fatalf("fixture: no in-zone cross at 11:30 (rsi %v)", rsi15(p, day)[early-1:early+1])
	}
	if v := hourRSI(p, day)[7]; v >= p.RSIArm {
		t.Fatalf("fixture: the 11:00 hour RSI %v must arm", v)
	}
	p.MinRoomATR = 0
	if got := buys(p, day, loudFrom(day, 4)); slices.Contains(got, early) {
		t.Fatalf("buys = %v: bought the 11:30 cross before the arming hour completed", got)
	}
}

func TestCrossOnArmingBarIgnored(t *testing.T) {
	// The 11:45 bar both completes the arming hour and crosses 30 up on 15m: the hourly RSI is
	// known only at its close, so the cross is not a signal.
	p := DefaultParams()
	p.MinRoomATR = 0
	day := path("2026-09-15 10:00", 100, 100, 100, 100, 100.1, 99.6, 99.2, 98.9, 99.5,
		99.7, 99.9)
	const arm = 7
	if v := hourRSI(p, day)[arm]; v >= p.RSIArm {
		t.Fatalf("fixture: the 11:00 hour RSI %v must arm", v)
	}
	if !crossesAt(p, day, arm) || !inZone(p, day, arm) {
		t.Fatalf("fixture: want an in-zone cross at %d (rsi %v)", arm, rsi15(p, day))
	}
	if got := buys(p, day, loudFrom(day, 4)); len(got) != 0 {
		t.Fatalf("buys = %v, want none", got)
	}
}

func TestHourOfPreviousDateDoesNotArm(t *testing.T) {
	// Tuesday's 23:00 hour lacks its 23:45 bar, so it completes on Wednesday's 07:00 bar; it
	// belongs to Tuesday and must not arm Wednesday's setup.
	p := DefaultParams()
	p.VolumeMult, p.MinRoomATR = 0, 0
	eve := path("2026-09-15 22:00", 100, 100, 100, 100, 100.1, 99.8, 99.5, 99.2)
	morning := path("2026-09-16 07:00", 99.2, 99.0, 98.9, 99.3, 99.5)
	day := append(eve, morning...)
	const done, cross = 7, 9
	if v, ok := hourRSI(p, day)[done]; !ok || v >= p.RSIArm {
		t.Fatalf("fixture: the 23:00 hour RSI %v (ok %v) must be under 30 at %d", v, ok, done)
	}
	if !crossesAt(p, day, cross) || !inZone(p, day, cross) {
		t.Fatalf("fixture: want an in-zone cross at %d (rsi %v)", cross, rsi15(p, day))
	}
	if got := buys(p, day, time.Time{}); len(got) != 0 {
		t.Fatalf("buys = %v, want none", got)
	}
}

func TestCrossOutsideZoneKeepsSetup(t *testing.T) {
	p := DefaultParams()
	day := path("2026-09-15 10:00", 101, 101, 101, 101, 101.1, 100.9, 100.7, 100.5, 100.3,
		100.2, 100.5, 100.6, 100.4, 100.0, 99.6, 99.3, 99.6, 99.8, 100.0)
	const outside, second = 9, 15
	if !crossesAt(p, day, outside) || inZone(p, day, outside) {
		t.Fatalf("fixture: want an out-of-zone cross at %d", outside)
	}
	if !crossesAt(p, day, second) || !inZone(p, day, second) {
		t.Fatalf("fixture: want an in-zone cross at %d (rsi %v)", second, rsi15(p, day))
	}
	if got := buys(p, day, loudFrom(day, 4)); !slices.Equal(got, []int{second}) {
		t.Fatalf("buys = %v, want [%d]", got, second)
	}
}

// gateDay arms at 11:45, rallies through hour 12 (the hourly RSI crosses 50 at 12:45; the 15m
// cross at 12:00 lies above the zone), then drops into the zone and crosses on the 13:45 bar.
func gateDay() []bar {
	return path("2026-09-15 10:00", 101, 101, 101, 101, 101.1, 100.9, 100.7, 100.5, 100.3,
		100.5, 100.7, 100.85, 101.0, 100.4, 99.8, 99.4, 99.7, 99.9, 100.1)
}

func TestGateBurnsSetup(t *testing.T) {
	p, day := DefaultParams(), gateDay()
	const second = 15
	rh := hourRSI(p, day)
	if !crossUp(rh[7], rh[11], p.RSIGate) || rh[7] >= p.RSIArm {
		t.Fatalf("fixture: hour RSI 11:45 %v, 12:45 %v: want arm then a cross of 50", rh[7], rh[11])
	}
	if !crossesAt(p, day, second) || !inZone(p, day, second) {
		t.Fatalf("fixture: want an in-zone cross at %d (rsi %v)", second, rsi15(p, day))
	}
	if got := buys(p, day, loudFrom(day, 4)); len(got) != 0 {
		t.Fatalf("buys = %v, want none (the gate burnt the setup)", got)
	}
	p.RSIGate = 0
	if got := buys(p, day, loudFrom(day, 4)); !slices.Equal(got, []int{second}) {
		t.Fatalf("RSIGate 0: buys = %v, want [%d]", got, second)
	}
}

func TestGateOnSignalBarWins(t *testing.T) {
	p := DefaultParams()
	p.MinRoomATR = 0
	day := path("2026-09-15 10:00", 100, 100, 100, 100, 100.1, 99.9, 99.8, 99.7, 99.6,
		99.4, 99.2, 99.0, 100.0, 100.2, 100.4)
	const cross = 11 // 12:45: completes the hour whose RSI crosses 50
	rh := hourRSI(p, day)
	if rh[7] >= p.RSIArm || !crossUp(rh[7], rh[cross], p.RSIGate) {
		t.Fatalf("fixture: hour RSI %v → %v", rh[7], rh[cross])
	}
	if !crossesAt(p, day, cross) || !inZone(p, day, cross) {
		t.Fatalf("fixture: want an in-zone 15m cross at %d", cross)
	}
	if got := buys(p, day, loudFrom(day, 4)); len(got) != 0 {
		t.Fatalf("buys = %v, want none", got)
	}
	p.RSIGate = 0
	if got := buys(p, day, loudFrom(day, 4)); !slices.Equal(got, []int{cross}) {
		t.Fatalf("RSIGate 0: buys = %v, want [%d]", got, cross)
	}
}

// twoCrossDay arms at 11:45 and crosses in zone at 12:00; then the 15m RSI dips and crosses again
// at 12:30, the 12:45 hour staying at or above 30.
func twoCrossDay() []bar {
	return path("2026-09-15 10:00", 100, 100, 100, 100, 100.1, 99.8, 99.6, 99.4, 99.2,
		99.5, 98.9, 99.2, 99.5, 99.7, 99.9)
}

func TestOneSignalPerSetup(t *testing.T) {
	p := DefaultParams()
	day := twoCrossDay()
	for _, j := range []int{8, 10} {
		if !crossesAt(p, day, j) || !inZone(p, day, j) {
			t.Fatalf("fixture: want an in-zone cross at %d (rsi %v)", j, rsi15(p, day))
		}
	}
	if v := hourRSI(p, day)[11]; v < p.RSIArm {
		t.Fatalf("fixture: the 12:00 hour RSI %v re-arms", v)
	}
	if got := buys(p, day, loudFrom(day, 4)); !slices.Equal(got, []int{8}) {
		t.Fatalf("no re-arm: buys = %v, want [8]", got)
	}

	// The 12:00 hour slides to 98.9 (RSI < 30 at 12:45): a new setup buys the 13:00 cross.
	day = path("2026-09-15 10:00", 100, 100, 100, 100, 100.1, 99.8, 99.6, 99.4, 99.2,
		99.5, 99.1, 99.0, 98.9, 99.2, 99.4, 99.6)
	if v := hourRSI(p, day)[11]; v >= p.RSIArm {
		t.Fatalf("fixture: the 12:00 hour RSI %v must re-arm", v)
	}
	if got := buys(p, day, loudFrom(day, 4)); !slices.Equal(got, []int{8, 12}) {
		t.Fatalf("re-arm: buys = %v, want [8 12]", got)
	}
}

func TestFilterFailBurnsSetup(t *testing.T) {
	p := DefaultParams()
	day := twoCrossDay()
	if got := buys(p, day, time.Time{}); len(got) != 0 {
		t.Fatalf("quiet: buys = %v, want none", got)
	}
	// Loud from 12:15: the 12:00 window (11:00–12:00) is quiet and fails 1.1×; the 12:30 window
	// (11:00–12:30, two loud bars) would pass, but the setup is already burnt.
	p.VolumeMult = 1.1
	loud := loudFrom(day, 9)
	md := fixture(warm(day)[:warmBars+11], loud)
	if _, _, ok := NewWithParams("TEST", step, p).volumeOK(barsOf(md), warmBars+4, warmBars+10); !ok {
		t.Fatal("fixture: the second window must pass the volume filter")
	}
	if got := buys(p, day, loud); len(got) != 0 {
		t.Fatalf("buys = %v, want none", got)
	}
	p.VolumeMult = 0
	if got := buys(p, day, loud); !slices.Equal(got, []int{8}) {
		t.Fatalf("volume off: buys = %v, want [8]", got)
	}
}

func TestVolumeFilter(t *testing.T) {
	day := canonical()
	cases := []struct {
		name string
		day  []bar
		mult float64
		loud time.Time
		want bool
	}{
		{"loud window passes", day, 1.5, loudFrom(day, canonFrom), true},
		{"quiet window fails", day, 1.5, time.Time{}, false},
		{"filter off buys on a quiet window", day, 0, time.Time{}, true},
		{"evening window has no main-session bars", shifted(day, 9*time.Hour), 1.5, at("2026-09-15 18:00"), false},
		{"evening window, filter off", shifted(day, 9*time.Hour), 0, time.Time{}, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			p := DefaultParams()
			p.VolumeMult = tc.mult
			if got := run(p, tc.day, canonCross, tc.loud).Kind == model.SignalBuy; got != tc.want {
				t.Fatalf("buy = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestNoVolumesNoEntry(t *testing.T) {
	day := canonical()
	md := fixture(warm(day)[:warmBars+canonCross+1], loudFrom(day, canonFrom))
	md.Volumes = nil
	if sig := NewWithParams("TEST", step, DefaultParams()).Decide(md); sig.Kind == model.SignalBuy {
		t.Fatal("bought without volumes")
	}
}

func TestMinRoomATR(t *testing.T) {
	// The cross bar jumps from 99.0 (the day's low) to 100.0: room (99 + 2 − 100) / 2 = 0.5 ATR.
	day := path("2026-09-15 10:00", 100, 100, 100, 100, 100.1, 99.8, 99.6, 99.4, 99.2,
		99.1, 99.0, 100.0, 100.2, 100.4)
	md := fixture(warm(day)[:warmBars+canonCross+1], time.Time{})
	if room := roomLeft(barsOf(md), warmBars+canonCross, 2); math.Abs(room-0.5) > eps {
		t.Fatalf("fixture: room %v, want 0.5", room)
	}
	p := DefaultParams()
	p.MinRoomATR = 0.6
	if got := buys(p, day, loudFrom(day, canonFrom)); len(got) != 0 {
		t.Fatalf("MinRoomATR 0.6: buys = %v, want none", got)
	}
	p.MinRoomATR = 0
	if got := buys(p, day, loudFrom(day, canonFrom)); !slices.Equal(got, []int{canonCross}) {
		t.Fatalf("MinRoomATR 0: buys = %v, want [%d]", got, canonCross)
	}
}

func TestSetupDoesNotOutliveDate(t *testing.T) {
	// The 23:00 hour arms at 23:45 Tuesday; the 15m RSI crosses on Wednesday's 07:15 bar.
	p := DefaultParams()
	p.VolumeMult, p.MinRoomATR = 0, 0
	eve := path("2026-09-15 22:00", 100, 100, 100, 100, 100.1, 99.8, 99.6, 99.4, 99.2)
	morning := path("2026-09-16 07:00", 99.2, 99.0, 99.3, 99.5, 99.7)
	day := append(eve, morning...)
	const cross = 9
	if v := hourRSI(p, day)[7]; v >= p.RSIArm {
		t.Fatalf("fixture: the 23:00 hour RSI %v must arm", v)
	}
	if !crossesAt(p, day, cross) || !inZone(p, day, cross) {
		t.Fatalf("fixture: want an in-zone cross at %d", cross)
	}
	if got := buys(p, day, time.Time{}); len(got) != 0 {
		t.Fatalf("buys = %v, want none", got)
	}
}

// lateDay is canonical with the cross on the 12:45 bar, which completes an hour under 50.
func lateDay() []bar {
	return path("2026-09-15 10:00", 100, 100, 100, 100, 100.1, 99.8, 99.6, 99.4, 99.2,
		99.1, 99.05, 99.0, 99.3, 99.5, 99.7)
}

func TestWeekendAndLastBarOfDate(t *testing.T) {
	p := DefaultParams()
	p.VolumeMult = 0
	saturday := shifted(canonical(), 4*24*time.Hour)
	if got := buys(p, saturday, time.Time{}); len(got) != 0 {
		t.Fatalf("Saturday: buys = %v, want none", got)
	}
	const cross = 11
	if got := buys(p, lateDay(), time.Time{}); !slices.Equal(got, []int{cross}) {
		t.Fatalf("fixture: lateDay buys = %v, want [%d]", got, cross)
	}
	last := shifted(lateDay(), 11*time.Hour) // the cross at 23:45
	if got := buys(p, last, time.Time{}); len(got) != 0 {
		t.Fatalf("23:45 cross: buys = %v, want none", got)
	}
}

func TestWarmupZerosNeverArm(t *testing.T) {
	// Only the day's 16 bars (4 hours): the hourly RSI is all warm-up zeros and must not arm.
	p := DefaultParams()
	p.VolumeMult = 0
	day := canonical()
	s := NewWithParams("TEST", step, p)
	for i := range day {
		if sig := s.Decide(fixture(day[:i+1], time.Time{})); sig.Kind == model.SignalBuy {
			t.Fatalf("bought at %d on warm-up zeros", i)
		}
	}
}

func TestHourMissingCloseBarArmsOnce(t *testing.T) {
	// canonical without the 11:45 bar: the 11:00 hour completes at the 12:00 bar.
	p := DefaultParams()
	day := slices.Delete(canonical(), canonArm, canonArm+1)
	const arm, cross = 7, 9 // 12:00, 12:30
	if v, ok := hourRSI(p, day)[arm]; !ok || v >= p.RSIArm {
		t.Fatalf("fixture: hour completing at %d RSI %v (ok %v)", arm, v, ok)
	}
	if !crossesAt(p, day, cross) || !inZone(p, day, cross) {
		t.Fatalf("fixture: want an in-zone cross at %d", cross)
	}
	if got := buys(p, day, loudFrom(day, canonFrom)); !slices.Equal(got, []int{cross}) {
		t.Fatalf("buys = %v, want [%d]", got, cross)
	}
}
