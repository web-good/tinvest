package core

import (
	"fmt"
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

// testParams is DefaultParams with the trend filter, the RSI floor and the stochastic gate off: the
// fixtures dip from a flat warm-up, so every signal closes under the EMA and the RSI drops to about
// 0 (TestTrendEMA, TestRSIFloor and the stochastic tests cover them).
func testParams() Params {
	p := DefaultParams()
	p.TrendEMA, p.RSIFloor, p.StochK = 0, 0, 0
	return p
}

type bar struct {
	t          time.Time
	o, h, l, c float64
}

// path chains hourly bars from MSK `from`: each opens at the previous close (the first at o0) and
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
// closing at 100. Weekdays span 99..101 (Wilder ATR exactly 2); weekend sessions span
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

// warmBars is how many flat hourly bars warm prepends: more than twice EMA(200), so the volume
// baseline, the RSI and the trend EMA are warm (a flat RSI is 50: the first falling hour drops it
// to 0, crossing 30 down).
const warmBars = 500

// warm prepends warmBars flat bars at the day's first open ending one bar before day[0]. Like a
// real exchange day the bars start at 06:00 MSK, so no warm date reads as a dealer date.
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

// fixture builds MarketData from bars plus 40 days of dailies before the last bar (ATR 2).
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

// rsiOf is the hourly RSI of the warmed day at the day's indexes.
func rsiOf(p Params, day []bar) []float64 {
	all := warm(day)
	closes := make([]float64, len(all))
	for i, b := range all {
		closes[i] = b.c
	}
	return indicators.RSISeries(closes, p.HourRSIPeriod)[warmBars:]
}

// canonical is Tuesday 2026-09-15 by hours from 10:00: flat at 100 (RSI 50), then the 12:00 hour
// falling to 99.0 with the RSI crossing 30 down to 0; then a climb that never crosses 30 down again.
func canonical() []bar {
	return path("2026-09-15 10:00", 100, 100, 100, 99.0, 99.5, 100.0, 100.5)
}

const canonSignal = 2 // 12:00: the RSI crossing 30 down

// loudFrom is the start of day[i]: the bars from it on are loud.
func loudFrom(day []bar, i int) time.Time { return day[i].t }

// checkCanonical asserts the canonical fixture does what its comment says.
func checkCanonical(t *testing.T, p Params, day []bar) {
	t.Helper()
	r := rsiOf(p, day)
	for j := 1; j < len(day); j++ {
		if crossDown(r[j-1], r[j], p.RSIArm) != (j == canonSignal) {
			t.Fatalf("fixture: hour %d RSI %v → %v", j, r[j-1], r[j])
		}
	}
}

func TestEntryOnRSISignal(t *testing.T) {
	p, day := testParams(), canonical()
	checkCanonical(t, p, day)
	if got := buys(p, day, time.Time{}); !slices.Equal(got, []int{canonSignal}) {
		t.Fatalf("buys = %v, want [%d]", got, canonSignal)
	}
	sig := run(p, day, canonSignal, time.Time{})
	// The stop sits StopATR (0.5) daily ATRs (2) under the 99.0 close, the target 2 R above it.
	if math.Abs(sig.StopLoss-98.0) > eps || math.Abs(sig.TakeProfit-101.0) > eps || math.Abs(sig.ATR-2) > eps {
		t.Fatalf("StopLoss/TakeProfit/ATR = %v/%v/%v, want 98/101/2", sig.StopLoss, sig.TakeProfit, sig.ATR)
	}
	if !sig.FillBy.Equal(day[canonSignal].t.Add(step)) {
		t.Fatalf("FillBy = %v, want the next hour's open", sig.FillBy)
	}
	for _, s := range []string{"вниз на часе 12:00", "close − 0.50 ATR", "цель"} {
		if !strings.Contains(sig.EntryReason, s) {
			t.Errorf("EntryReason %q lacks %q", sig.EntryReason, s)
		}
	}
}

func TestRSIFloor(t *testing.T) {
	// A choppy morning, then the 14:00 hour falls to 98.8 with the RSI crossing 30 down to about 28.
	// The canonical dip drops the RSI to 0.
	p := testParams()
	day := path("2026-09-15 06:00", 100, 100.5, 100, 100.5, 100, 100.5, 100, 100.5, 100, 98.8, 99.1)
	const signal = 8
	r := rsiOf(p, day)
	if !crossDown(r[signal-1], r[signal], p.RSIArm) || r[signal] < 22 {
		t.Fatalf("fixture: RSI %v", r)
	}
	p.RSIFloor = 22
	if got := buys(p, day, time.Time{}); !slices.Equal(got, []int{signal}) {
		t.Fatalf("RSIFloor 22, RSI %v: buys = %v, want [%d]", r[signal], got, signal)
	}
	if sig := run(p, day, signal, time.Time{}); !strings.Contains(sig.EntryReason, "≥ пола 22") {
		t.Fatalf("reason %q lacks the RSI floor", sig.EntryReason)
	}
	p.RSIFloor = r[signal] + 0.01
	if got := buys(p, day, time.Time{}); len(got) != 0 {
		t.Fatalf("RSIFloor over the signal RSI %v: buys = %v, want none", r[signal], got)
	}
	p.RSIFloor = 22
	if got := buys(p, canonical(), time.Time{}); len(got) != 0 {
		t.Fatalf("canonical dip to RSI 0 under the floor 22: buys = %v, want none", got)
	}
}

func TestStopATR(t *testing.T) {
	p, day := testParams(), canonical()
	p.StopATR = 0.3
	sig := run(p, day, canonSignal, time.Time{})
	if sig.Kind != model.SignalBuy || math.Abs(sig.StopLoss-(99.0-0.6)) > eps {
		t.Fatalf("StopATR 0.3: Kind %v StopLoss %v, want 99.0 − 0.3 × 2", sig.Kind, sig.StopLoss)
	}
	p.StopATR = 0
	if got := buys(p, day, time.Time{}); len(got) != 0 {
		t.Fatalf("StopATR 0: buys = %v, want none", got)
	}
}

func TestMaxStopPct(t *testing.T) {
	// 0.5 daily ATRs (2) is 1.0 under the 99.0 close; a 0.5% cap cuts it to 0.495.
	p, day := testParams(), canonical()
	p.MaxStopPct = 0.5
	sig := run(p, day, canonSignal, time.Time{})
	want := 99.0 - 0.005*99.0
	if sig.Kind != model.SignalBuy || math.Abs(sig.StopLoss-want) > eps ||
		math.Abs(sig.TakeProfit-(99.0+2*0.005*99.0)) > eps || !strings.Contains(sig.EntryReason, "урезан до 0.50%") {
		t.Fatalf("MaxStopPct 0.5: Kind %v stop/target %v/%v (%q), want %v and 2 R of the capped stop",
			sig.Kind, sig.StopLoss, sig.TakeProfit, sig.EntryReason, want)
	}
	p.MaxStopPct = 2 // 1.98 over the 1.0 ATR distance: no cap
	if sig := run(p, day, canonSignal, time.Time{}); math.Abs(sig.StopLoss-98.0) > eps || strings.Contains(sig.EntryReason, "урезан") {
		t.Fatalf("MaxStopPct 2: stop %v (%q), want 98.0 uncapped", sig.StopLoss, sig.EntryReason)
	}
}

func TestNoCrossNoEntry(t *testing.T) {
	p, day := testParams(), canonical()
	p.RSIArm = -1 // the RSI never drops under it
	if got := buys(p, day, time.Time{}); len(got) != 0 {
		t.Fatalf("buys = %v, want none", got)
	}
}

func TestCrossAfterPreviousDateIgnored(t *testing.T) {
	// Tuesday ends flat at 23:00; Wednesday's 06:00 hour falls to 99.0, crossing 30 down. The hour
	// before the RSI cross belongs to Tuesday: no signal.
	p := testParams()
	eve := path("2026-09-15 20:00", 100, 100, 100, 100, 100)
	morning := path("2026-09-16 06:00", 100, 99.0, 99.5)
	day := append(eve, morning...)
	r := rsiOf(p, day)
	if !crossDown(r[3], r[4], p.RSIArm) {
		t.Fatalf("fixture: RSI %v", r)
	}
	if got := buys(p, day, time.Time{}); len(got) != 0 {
		t.Fatalf("buys = %v, want none", got)
	}
}

func TestVolumeFilter(t *testing.T) {
	day := canonical()
	evening := shifted(day, 8*time.Hour) // the signal hour at 20:00: not a main-session hour
	cases := []struct {
		name string
		day  []bar
		mult float64
		loud time.Time
		want bool
	}{
		{"loud signal hour passes", day, 1.5, loudFrom(day, canonSignal), true},
		{"quiet signal hour fails", day, 1.5, time.Time{}, false},
		{"filter off buys on a quiet hour", day, 0, time.Time{}, true},
		{"evening signal hour is out of the main session", evening, 1.5, loudFrom(evening, canonSignal), false},
		{"evening signal hour, filter off", evening, 0, time.Time{}, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			p := testParams()
			p.VolumeMult = tc.mult
			if got := run(p, tc.day, canonSignal, tc.loud).Kind == model.SignalBuy; got != tc.want {
				t.Fatalf("buy = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestNoVolumesNoEntry(t *testing.T) {
	day := canonical()
	md := fixture(warm(day)[:warmBars+canonSignal+1], loudFrom(day, canonSignal))
	md.Volumes = nil
	p := testParams()
	p.VolumeMult = 1.5
	if sig := NewWithParams("TEST", step, p).Decide(md); sig.Kind == model.SignalBuy {
		t.Fatal("bought without volumes")
	}
}

func TestMinRoomATR(t *testing.T) {
	// The day's low 99.0 + ATR 2 − the 99.0 signal close = 1.0 ATR of room.
	day := canonical()
	md := fixture(warm(day)[:warmBars+canonSignal+1], time.Time{})
	if room := roomLeft(barsOf(md), warmBars+canonSignal, 2); math.Abs(room-1.0) > 1e-6 {
		t.Fatalf("fixture: room %v, want 1.0", room)
	}
	p := testParams()
	p.MinRoomATR = 1.1
	if got := buys(p, day, time.Time{}); len(got) != 0 {
		t.Fatalf("MinRoomATR 1.1: buys = %v, want none", got)
	}
	p.MinRoomATR = 0.9
	if got := buys(p, day, time.Time{}); !slices.Equal(got, []int{canonSignal}) {
		t.Fatalf("MinRoomATR 0.9: buys = %v, want [%d]", got, canonSignal)
	}
}

// risingWarm is warm(day) with the warm-up climbing linearly from lo to day[0].o instead of flat:
// the EMA(200) lags (day[0].o − lo) / 5 under the day.
func risingWarm(day []bar, lo float64) []bar {
	out := warm(day)
	o := day[0].o
	for k := range warmBars {
		c := lo + (o-lo)*float64(k+1)/warmBars
		prev := lo + (o-lo)*float64(k)/warmBars
		out[k].o, out[k].h, out[k].l, out[k].c = prev, c, prev, c
	}
	return out
}

func TestTrendEMA(t *testing.T) {
	p := testParams()
	p.TrendEMA = 200
	// Nine flat hours from 06:00, then the canonical dip at 15:00.
	day := path("2026-09-15 06:00", 100, append(slices.Repeat([]float64{100}, 9), 99.0, 99.5, 100.0)...)
	const signal = 9
	// Flat warm-up: the EMA(200) is about 100, the 99.0 signal close sits under it.
	if got := buys(p, day, time.Time{}); len(got) != 0 {
		t.Fatalf("close under EMA: buys = %v, want none", got)
	}
	// Warm-up rising 90 → 100: the EMA lags about 2 points, under the 99.0 close.
	bars := risingWarm(day, 90)
	s := NewWithParams("TEST", step, p)
	var got []int
	var sig model.Signal
	for i := range day {
		if g := s.Decide(fixture(bars[:warmBars+i+1], time.Time{})); g.Kind == model.SignalBuy {
			got, sig = append(got, i), g
		}
	}
	if !slices.Equal(got, []int{signal}) || !strings.Contains(sig.EntryReason, "над EMA(200)") {
		t.Fatalf("close over EMA: buys = %v, reason %q, want [%d]", got, sig.EntryReason, signal)
	}
	p.TrendEMA = 1000 // more bars than the window holds: the EMA never warms
	if g := NewWithParams("TEST", step, p).Decide(fixture(bars[:warmBars+signal+1], time.Time{})); g.Kind == model.SignalBuy {
		t.Fatal("cold EMA: bought")
	}
}

func TestWeekendAndLastBarOfDate(t *testing.T) {
	p := testParams()
	saturday := shifted(canonical(), 4*24*time.Hour)
	if got := buys(p, saturday, time.Time{}); len(got) != 0 {
		t.Fatalf("Saturday: buys = %v, want none", got)
	}
	last := shifted(canonical()[:canonSignal+1], 11*time.Hour) // the signal hour at 23:00
	if got := last[canonSignal].t; !got.Equal(at("2026-09-15 23:00")) {
		t.Fatalf("fixture: signal hour at %v", got)
	}
	if got := buys(p, last, time.Time{}); len(got) != 0 {
		t.Fatalf("23:00 signal: buys = %v, want none", got)
	}
}

func TestWarmupZerosNeverSignal(t *testing.T) {
	// Only the day's 7 hours: the RSI warm-up zeros are no cross.
	p := testParams()
	day := canonical()
	s := NewWithParams("TEST", step, p)
	for i := range day {
		if sig := s.Decide(fixture(day[:i+1], time.Time{})); sig.Kind == model.SignalBuy {
			t.Fatalf("bought at %d on warm-up zeros", i)
		}
	}
}

// stochOf is the slow stochastic %K of the warmed day at the day's indexes.
func stochOf(p Params, day []bar) []float64 {
	md := fixture(warm(day), time.Time{})
	k, _ := NewWithParams("TEST", step, p).stochK(barsOf(md))
	return k[warmBars:]
}

// stochDay is a choppy Tuesday from 06:00, then the 14:00 hour falls to 98.8 with a tail to 98.0,
// crossing the RSI 30 down; the 15:00 and 16:00 hours keep falling to 98.2 and 98.1 over the same
// 98.0 tail. With StochK 5, StochSmooth 1 the %K is about 32 at 14:00 and under 10 after it.
func stochDay() []bar {
	day := path("2026-09-15 06:00", 100, 100.5, 100, 100.5, 100, 100.5, 100, 100.5, 100, 98.8, 98.2, 98.1)
	for _, i := range []int{8, 9, 10} {
		day = withLow(day, i, 98.0)
	}
	return day
}

const stochSignal = 8 // 14:00: the RSI crossing 30 down

func stochParams() Params {
	p := testParams()
	p.StochK, p.StochSmooth = 5, 1
	return p
}

func checkStochDay(t *testing.T, p Params, day []bar) (r, k []float64) {
	t.Helper()
	r, k = rsiOf(p, day), stochOf(p, day)
	if !crossDown(r[stochSignal-1], r[stochSignal], p.RSIArm) || r[stochSignal+1] >= r[stochSignal] || r[stochSignal+2] >= p.RSIArm {
		t.Fatalf("fixture: RSI %v", r)
	}
	if k[stochSignal] < 25 || k[stochSignal] > 40 || k[stochSignal+1] >= 20 || k[stochSignal+2] >= 20 {
		t.Fatalf("fixture: %%K %v", k)
	}
	return r, k
}

// buys decides each hour without a position, so every hour with both the RSI and the stochastic low
// shows up; in the backtest the first one the filters pass buys and the open position skips the rest.
func TestStochLevelBothLow(t *testing.T) {
	p, day := stochParams(), stochDay()
	checkStochDay(t, p, day)
	// %K about 32 on the RSI-cross hour: not yet. From 15:00 the RSI (under 30) and the %K (under 20)
	// are both low: each such hour is a signal.
	if got := buys(p, day, time.Time{}); !slices.Equal(got, []int{stochSignal + 1, stochSignal + 2}) {
		t.Fatalf("buys = %v, want [%d %d]", got, stochSignal+1, stochSignal+2)
	}
	sig := run(p, day, stochSignal+1, time.Time{})
	if want := "на часе 15:00 часовой RSI(6)"; !strings.Contains(sig.EntryReason, want) || !strings.Contains(sig.EntryReason, "< 20") {
		t.Errorf("EntryReason %q lacks %q and the %%K level", sig.EntryReason, want)
	}
	// The stop and the target come off the signal close 98.2.
	if math.Abs(sig.StopLoss-97.2) > eps || math.Abs(sig.TakeProfit-100.2) > eps {
		t.Fatalf("stop/target %v/%v, want 97.2/100.2", sig.StopLoss, sig.TakeProfit)
	}
}

func TestStochLevelNeedsRSILow(t *testing.T) {
	// The %K is under 20 from 15:00, but with RSIArm under every RSI of the day the RSI is never low.
	p, day := stochParams(), stochDay()
	r, _ := checkStochDay(t, p, day)
	p.RSIArm = slices.Min(r) - 0.01
	if got := buys(p, day, time.Time{}); len(got) != 0 {
		t.Fatalf("RSIArm %v: buys = %v, want none", p.RSIArm, got)
	}
}

func TestStochLevelNeedsStochLow(t *testing.T) {
	// The RSI crosses 30 down at 14:00 with %K about 32: the gate (under 20) skips that hour, while
	// with the gate off (StochK 0) the RSI cross itself is the only signal.
	p, day := stochParams(), stochDay()
	checkStochDay(t, p, day)
	if got := buys(p, day, time.Time{}); slices.Contains(got, stochSignal) {
		t.Fatalf("buys = %v, want no %d (%%K over 20)", got, stochSignal)
	}
	p.StochK = 0
	if got := buys(p, day, time.Time{}); !slices.Equal(got, []int{stochSignal}) {
		t.Fatalf("gate off: buys = %v, want [%d]", got, stochSignal)
	}
}

func TestStochDOnlyInReason(t *testing.T) {
	// %D never gates: every StochD buys the same hours; the reason shows %D, the average of the last
	// StochD %K values.
	p, day := stochParams(), stochDay()
	_, k := checkStochDay(t, p, day)
	want := buys(p, day, time.Time{})
	for d := 1; d <= 5; d++ {
		p.StochD = d
		if got := buys(p, day, time.Time{}); !slices.Equal(got, want) {
			t.Fatalf("StochD %d: buys = %v, want %v", d, got, want)
		}
	}
	p.StochD = 3
	i := stochSignal + 1
	sig := run(p, day, i, time.Time{})
	if want := fmt.Sprintf("(%%D %.2f)", (k[i-2]+k[i-1]+k[i])/3); !strings.Contains(sig.EntryReason, want) {
		t.Errorf("EntryReason %q lacks %q", sig.EntryReason, want)
	}
}

func TestSignalHourRSIFloor(t *testing.T) {
	// The RSI keeps falling after 14:00: the floor reads the signal hour's own RSI.
	p, day := stochParams(), stochDay()
	r, _ := checkStochDay(t, p, day)
	p.RSIFloor = (r[stochSignal+1] + r[stochSignal+2]) / 2 // 15:00 over it, 16:00 under it
	if r[stochSignal+2] >= r[stochSignal+1] {
		t.Fatalf("fixture: RSI %v", r)
	}
	if got := buys(p, day, time.Time{}); !slices.Equal(got, []int{stochSignal + 1}) {
		t.Fatalf("RSIFloor %v: buys = %v, want [%d]", p.RSIFloor, got, stochSignal+1)
	}
}
