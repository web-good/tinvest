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

// testParams is DefaultParams with the trend filter and the RSI floor off: the fixtures dip from a
// flat warm-up, so every signal closes under the EMA and the RSI drops to about 0 (TestTrendEMA
// and TestRSIFloor cover them).
func testParams() Params {
	p := DefaultParams()
	p.TrendEMA, p.RSIFloor = 0, 0
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

// stochOf is the slow stochastic %K and %D of the warmed day at the day's indexes.
func stochOf(p Params, day []bar) (k, d []float64) {
	md := fixture(warm(day), time.Time{})
	sk, sd, _ := NewWithParams("TEST", step, p).stochKD(barsOf(md))
	return sk[warmBars:], sd[warmBars:]
}

// canonical is Tuesday 2026-09-15 by hours from 10:00: flat at 100 (RSI 50, %K = %D = 0), then
// the 12:00 hour falling to 99.0 with the RSI crossing 30 down; the 13:00
// hour back to 99.5 turns the stochastic %K over %D with the RSI under 50; then a climb that never
// crosses 30 down again nor turns the stochastic over again.
func canonical() []bar {
	return path("2026-09-15 10:00", 100, 100, 100, 99.0, 99.5, 100.0, 100.5)
}

const (
	canonRSI     = 2 // 12:00: the RSI crossing 30 down
	canonTrigger = 3 // 13:00: the stochastic crossing up
)

// loudFrom is the start of day[i]: the bars from it on are loud.
func loudFrom(day []bar, i int) time.Time { return day[i].t }

// checkCanonical asserts the canonical fixture does what its comment says.
func checkCanonical(t *testing.T, p Params, day []bar) {
	t.Helper()
	r := rsiOf(p, day)
	k, d := stochOf(p, day)
	for j := 1; j < len(day); j++ {
		if crossDown(r[j-1], r[j], p.RSIArm) != (j == canonRSI) {
			t.Fatalf("fixture: hour %d RSI %v → %v", j, r[j-1], r[j])
		}
		if crossOver(k, d, j) != (j == canonTrigger) {
			t.Fatalf("fixture: hour %d %%K %v → %v, %%D %v → %v", j, k[j-1], k[j], d[j-1], d[j])
		}
	}
	if r[canonTrigger] > 50 {
		t.Fatalf("fixture: trigger RSI %v over 50", r[canonTrigger])
	}
}

func TestEntryOnStochAfterRSI(t *testing.T) {
	p, day := testParams(), canonical()
	checkCanonical(t, p, day)
	loud := loudFrom(day, canonRSI)
	if got := buys(p, day, loud); !slices.Equal(got, []int{canonTrigger}) {
		t.Fatalf("buys = %v, want [%d]", got, canonTrigger)
	}
	sig := run(p, day, canonTrigger, loud)
	// The stop sits StopATR (0.5) daily ATRs (2) under the 99.5 close, the target 2 R above it.
	if math.Abs(sig.StopLoss-98.5) > eps || math.Abs(sig.TakeProfit-101.5) > eps || math.Abs(sig.ATR-2) > eps {
		t.Fatalf("StopLoss/TakeProfit/ATR = %v/%v/%v, want 98.5/101.5/2", sig.StopLoss, sig.TakeProfit, sig.ATR)
	}
	if !sig.FillBy.Equal(day[canonTrigger].t.Add(step)) {
		t.Fatalf("FillBy = %v, want the next hour's open", sig.FillBy)
	}
	for _, s := range []string{"вниз на часе 12:00", "вверх на часе 13:00", "close − 0.50 ATR", "цель", "RSI ≤ 50", "ниже 40"} {
		if !strings.Contains(sig.EntryReason, s) {
			t.Errorf("EntryReason %q lacks %q", sig.EntryReason, s)
		}
	}
}

func TestStochWindow(t *testing.T) {
	// After the 12:00 dip the price sits at 99.0 for five hours: the 18:00 recovery is the first
	// stochastic cross, six hours after the RSI signal.
	p := testParams()
	day := path("2026-09-15 10:00", 100, 100, 100, 99.0, 99.0, 99.0, 99.0, 99.0, 99.0, 99.5, 100.0)
	k, d := stochOf(p, day)
	for j := 1; j < len(day); j++ {
		if crossOver(k, d, j) != (j == 8) {
			t.Fatalf("fixture: hour %d %%K %v → %v, %%D %v → %v", j, k[j-1], k[j], d[j-1], d[j])
		}
	}
	p.RSITriggerMax = 0 // the recovery after the long flat stretch lifts the RSI over 50
	if got := buys(p, day, time.Time{}); len(got) != 0 {
		t.Fatalf("StochBars 4: buys = %v, want none", got)
	}
	p.StochBars = 6
	if got := buys(p, day, time.Time{}); !slices.Equal(got, []int{8}) {
		t.Fatalf("StochBars 6: buys = %v, want [8]", got)
	}
	p.StochBars = 0
	if got := buys(p, canonical(), time.Time{}); len(got) != 0 {
		t.Fatalf("StochBars 0: buys = %v, want none", got)
	}
}

func TestStochCrossMax(t *testing.T) {
	p, day := testParams(), canonical()
	k, _ := stochOf(p, day)
	if k[canonTrigger] >= p.StochCrossMax {
		t.Fatalf("fixture: trigger %%K %v not under %v", k[canonTrigger], p.StochCrossMax)
	}
	p.StochCrossMax = k[canonTrigger]
	if got := buys(p, day, time.Time{}); len(got) != 0 {
		t.Fatalf("StochCrossMax at the trigger %%K %v: buys = %v, want none", k[canonTrigger], got)
	}
	p.StochCrossMax = 0
	if got := buys(p, day, time.Time{}); !slices.Equal(got, []int{canonTrigger}) {
		t.Fatalf("StochCrossMax 0: buys = %v, want [%d]", got, canonTrigger)
	}
}

func TestRSIFloor(t *testing.T) {
	// A choppy morning, then the 14:00 hour falls to 98.8 with the RSI crossing 30 down to about 28;
	// the 16:00 hour turns the stochastic up. The canonical dip drops the RSI to 0.
	p := testParams()
	day := path("2026-09-15 06:00", 100, 100.5, 100, 100.5, 100, 100.5, 100, 100.5, 100, 98.8, 99.1, 99.6)
	const signal, trigger = 8, 10
	r := rsiOf(p, day)
	low := min(r[signal], r[signal+1], r[trigger])
	if !crossDown(r[signal-1], r[signal], p.RSIArm) || low < 22 || low >= 30 {
		t.Fatalf("fixture: RSI %v", r)
	}
	p.RSIFloor = 22
	if got := buys(p, day, time.Time{}); !slices.Equal(got, []int{trigger}) {
		t.Fatalf("RSIFloor 22, low %v: buys = %v, want [%d]", low, got, trigger)
	}
	if sig := run(p, day, trigger, time.Time{}); !strings.Contains(sig.EntryReason, "мин. RSI от сигнала") {
		t.Fatalf("reason %q lacks the RSI floor", sig.EntryReason)
	}
	p.RSIFloor = low + 0.01
	if got := buys(p, day, time.Time{}); len(got) != 0 {
		t.Fatalf("RSIFloor over the low %v: buys = %v, want none", low, got)
	}
	p.RSIFloor = 22
	if got := buys(p, canonical(), time.Time{}); len(got) != 0 {
		t.Fatalf("canonical dip to RSI 0 under the floor 22: buys = %v, want none", got)
	}
}

func TestRSITriggerMax(t *testing.T) {
	p, day := testParams(), canonical()
	r := rsiOf(p, day)
	p.RSITriggerMax = r[canonTrigger] - 0.01
	if got := buys(p, day, time.Time{}); len(got) != 0 {
		t.Fatalf("RSITriggerMax under the trigger RSI %v: buys = %v, want none", r[canonTrigger], got)
	}
	p.RSITriggerMax = 0
	if got := buys(p, day, time.Time{}); !slices.Equal(got, []int{canonTrigger}) {
		t.Fatalf("RSITriggerMax 0: buys = %v, want [%d]", got, canonTrigger)
	}
}

func TestStopATR(t *testing.T) {
	p, day := testParams(), canonical()
	p.StopATR = 0.3
	sig := run(p, day, canonTrigger, time.Time{})
	if sig.Kind != model.SignalBuy || math.Abs(sig.StopLoss-(99.5-0.6)) > eps {
		t.Fatalf("StopATR 0.3: Kind %v StopLoss %v, want 99.5 − 0.3 × 2", sig.Kind, sig.StopLoss)
	}
	p.StopATR = 0
	if got := buys(p, day, time.Time{}); len(got) != 0 {
		t.Fatalf("StopATR 0: buys = %v, want none", got)
	}
}

func TestMaxStopPct(t *testing.T) {
	// 0.5 daily ATRs (2) is 1.0 under the 99.5 close; a 0.5% cap cuts it to 0.4975.
	p, day := testParams(), canonical()
	p.MaxStopPct = 0.5
	sig := run(p, day, canonTrigger, time.Time{})
	want := 99.5 - 0.005*99.5
	if sig.Kind != model.SignalBuy || math.Abs(sig.StopLoss-want) > eps ||
		math.Abs(sig.TakeProfit-(99.5+2*0.005*99.5)) > eps || !strings.Contains(sig.EntryReason, "урезан до 0.50%") {
		t.Fatalf("MaxStopPct 0.5: Kind %v stop/target %v/%v (%q), want %v and 2 R of the capped stop",
			sig.Kind, sig.StopLoss, sig.TakeProfit, sig.EntryReason, want)
	}
	p.MaxStopPct = 2 // 1.99 over the 1.0 ATR distance: no cap
	if sig := run(p, day, canonTrigger, time.Time{}); math.Abs(sig.StopLoss-98.5) > eps || strings.Contains(sig.EntryReason, "урезан") {
		t.Fatalf("MaxStopPct 2: stop %v (%q), want 98.5 uncapped", sig.StopLoss, sig.EntryReason)
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
	// Tuesday ends flat at 23:00; Wednesday's 06:00 hour falls to 99.0, crossing 30 down, and the
	// 07:00 hour turns the stochastic up. The hour before the RSI cross belongs to Tuesday: no signal.
	p := testParams()
	eve := path("2026-09-15 20:00", 100, 100, 100, 100, 100)
	morning := path("2026-09-16 06:00", 100, 99.0, 99.5)
	day := append(eve, morning...)
	r := rsiOf(p, day)
	k, d := stochOf(p, day)
	if !crossDown(r[3], r[4], p.RSIArm) || !crossOver(k, d, 5) {
		t.Fatalf("fixture: RSI %v, %%K %v, %%D %v", r, k, d)
	}
	if got := buys(p, day, time.Time{}); len(got) != 0 {
		t.Fatalf("buys = %v, want none", got)
	}
}

func TestVolumeFilter(t *testing.T) {
	day := canonical()
	evening := shifted(day, 8*time.Hour) // the RSI hour at 20:00: not a main-session hour
	cases := []struct {
		name string
		day  []bar
		mult float64
		loud time.Time
		want bool
	}{
		{"loud RSI hour passes", day, 1.5, loudFrom(day, canonRSI), true},
		{"quiet RSI hour fails", day, 1.5, time.Time{}, false},
		{"loud trigger hour alone fails", day, 1.5, loudFrom(day, canonTrigger), false},
		{"filter off buys on a quiet hour", day, 0, time.Time{}, true},
		{"evening RSI hour is out of the main session", evening, 1.5, loudFrom(evening, canonRSI), false},
		{"evening RSI hour, filter off", evening, 0, time.Time{}, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			p := testParams()
			p.VolumeMult = tc.mult
			if got := run(p, tc.day, canonTrigger, tc.loud).Kind == model.SignalBuy; got != tc.want {
				t.Fatalf("buy = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestNoVolumesNoEntry(t *testing.T) {
	day := canonical()
	md := fixture(warm(day)[:warmBars+canonTrigger+1], loudFrom(day, canonRSI))
	md.Volumes = nil
	p := testParams()
	p.VolumeMult = 1.5
	if sig := NewWithParams("TEST", step, p).Decide(md); sig.Kind == model.SignalBuy {
		t.Fatal("bought without volumes")
	}
}

func TestMinRoomATR(t *testing.T) {
	// The day's low 99.0 + ATR 2 − the 99.5 trigger close = 0.75 ATR of room.
	day := canonical()
	md := fixture(warm(day)[:warmBars+canonTrigger+1], time.Time{})
	if room := roomLeft(barsOf(md), warmBars+canonTrigger, 2); math.Abs(room-0.75) > 1e-6 {
		t.Fatalf("fixture: room %v, want 0.75", room)
	}
	p := testParams()
	p.MinRoomATR = 0.8
	if got := buys(p, day, time.Time{}); len(got) != 0 {
		t.Fatalf("MinRoomATR 0.8: buys = %v, want none", got)
	}
	p.MinRoomATR = 0.7
	if got := buys(p, day, time.Time{}); !slices.Equal(got, []int{canonTrigger}) {
		t.Fatalf("MinRoomATR 0.7: buys = %v, want [%d]", got, canonTrigger)
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
	// Nine flat hours from 06:00 bring %K and %D down to 0 after a rising warm-up, then the
	// canonical dip at 15:00 and the turn at 16:00.
	day := path("2026-09-15 06:00", 100, append(slices.Repeat([]float64{100}, 9), 99.0, 99.5, 100.0)...)
	const trigger = 10
	// Flat warm-up: the EMA(200) is about 100, the 99.5 trigger close sits under it.
	if got := buys(p, day, time.Time{}); len(got) != 0 {
		t.Fatalf("close under EMA: buys = %v, want none", got)
	}
	// Warm-up rising 90 → 100: the EMA lags about 2 points, under the 99.5 close.
	bars := risingWarm(day, 90)
	s := NewWithParams("TEST", step, p)
	var got []int
	var sig model.Signal
	for i := range day {
		if g := s.Decide(fixture(bars[:warmBars+i+1], time.Time{})); g.Kind == model.SignalBuy {
			got, sig = append(got, i), g
		}
	}
	if !slices.Equal(got, []int{trigger}) || !strings.Contains(sig.EntryReason, "над EMA(200)") {
		t.Fatalf("close over EMA: buys = %v, reason %q, want [%d]", got, sig.EntryReason, trigger)
	}
	p.TrendEMA = 1000 // more bars than the window holds: the EMA never warms
	if g := NewWithParams("TEST", step, p).Decide(fixture(bars[:warmBars+trigger+1], time.Time{})); g.Kind == model.SignalBuy {
		t.Fatal("cold EMA: bought")
	}
}

func TestWeekendAndLastBarOfDate(t *testing.T) {
	p := testParams()
	saturday := shifted(canonical(), 4*24*time.Hour)
	if got := buys(p, saturday, time.Time{}); len(got) != 0 {
		t.Fatalf("Saturday: buys = %v, want none", got)
	}
	last := shifted(canonical()[:canonTrigger+1], 10*time.Hour) // the trigger hour at 23:00
	if got := last[canonTrigger].t; !got.Equal(at("2026-09-15 23:00")) {
		t.Fatalf("fixture: trigger hour at %v", got)
	}
	if got := buys(p, last, time.Time{}); len(got) != 0 {
		t.Fatalf("23:00 trigger: buys = %v, want none", got)
	}
}

func TestWarmupZerosNeverSignal(t *testing.T) {
	// Only the day's 7 hours: the RSI and stochastic warm-up zeros are no cross.
	p := testParams()
	day := canonical()
	s := NewWithParams("TEST", step, p)
	for i := range day {
		if sig := s.Decide(fixture(day[:i+1], time.Time{})); sig.Kind == model.SignalBuy {
			t.Fatalf("bought at %d on warm-up zeros", i)
		}
	}
}
