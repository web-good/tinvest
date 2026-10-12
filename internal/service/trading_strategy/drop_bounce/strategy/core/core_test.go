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
	t          time.Time
	o, h, l, c float64
}

// step15 is the timeframe the fixtures are built on unless a test names another.
const step15 = 15 * time.Minute

// path builds consecutive 15m bars from start (MSK): each bar opens at the previous close (the
// first at its own close) and its high/low span the open and the close.
func path(start string, closes []float64) []bar {
	t0 := at(start)
	out := make([]bar, len(closes))
	prev := closes[0]
	for i, c := range closes {
		out[i] = bar{t0.Add(time.Duration(i) * step15), prev, max(prev, c), min(prev, c), c}
		prev = c
	}
	return out
}

// steps returns n closes, each pct percent from the previous one, starting from `from` (excluded).
func steps(from float64, n int, pct float64) []float64 {
	out := make([]float64, n)
	for i := range out {
		from *= 1 + pct/100
		out[i] = from
	}
	return out
}

// flatN returns n closes equal to p.
func flatN(n int, p float64) []float64 {
	out := make([]float64, n)
	for i := range out {
		out[i] = p
	}
	return out
}

// fixtureATR is the daily ATR every fixture carries: its dailies range exactly this much.
const fixtureATR = 2.0

// fixtureDayVol is the volume of every fixture daily, so the average daily volume is exactly it.
const fixtureDayVol = 1000

// quietVol is the volume of every fixture bar a test does not set: far below any threshold.
const quietVol = 10

// fixture builds the market data over bars, every bar trading quietVol, with 20 weekday dailies
// of range fixtureATR and volume fixtureDayVol before the first bar.
func fixture(bars []bar) strategy.MarketData {
	var md strategy.MarketData
	for _, b := range bars {
		md.Times = append(md.Times, b.t)
		md.Opens = append(md.Opens, b.o)
		md.Highs = append(md.Highs, b.h)
		md.Lows = append(md.Lows, b.l)
		md.Closes = append(md.Closes, b.c)
		md.Volumes = append(md.Volumes, quietVol)
	}
	md.Price = md.Closes[len(md.Closes)-1]
	withDailies(&md, dailies(bars[0].t, 20, fixtureATR))
	return md
}

// dailies returns n completed weekday daily bars ending the weekday before `before`, each
// spanning `rng` around a close of 100, so their ATR is exactly rng.
func dailies(before time.Time, n int, rng float64) []bar {
	d := time.Date(before.In(mskLoc).Year(), before.In(mskLoc).Month(), before.In(mskLoc).Day(), 0, 0, 0, 0, mskLoc)
	out := make([]bar, 0, n)
	for len(out) < n {
		d = d.AddDate(0, 0, -1)
		if wd := d.Weekday(); wd == time.Saturday || wd == time.Sunday {
			continue
		}
		out = append([]bar{{d, 100, 100 + rng/2, 100 - rng/2, 100}}, out...)
	}
	return out
}

// withDailies replaces the daily series of md with ds, each day trading fixtureDayVol.
func withDailies(md *strategy.MarketData, ds []bar) {
	md.DailyHighs, md.DailyLows, md.DailyCloses, md.DailyTimes, md.DailyVolumes = nil, nil, nil, nil, nil
	for _, d := range ds {
		md.DailyTimes = append(md.DailyTimes, d.t)
		md.DailyHighs = append(md.DailyHighs, d.h)
		md.DailyLows = append(md.DailyLows, d.l)
		md.DailyCloses = append(md.DailyCloses, d.c)
		md.DailyVolumes = append(md.DailyVolumes, fixtureDayVol)
	}
}

// redAt is 30 flat bars at 100 followed by a red bar 100 → 99.5 starting at t (MSK).
func redAt(t string) []bar {
	end := at(t)
	bars := path(end.Add(-30*step15).Format("2006-01-02 15:04"), flatN(30, 100))
	return append(bars, bar{end, 100, 100, 99.5, 99.5})
}

// spikeMD is the fixture over bars with the last bar trading vol.
func spikeMD(bars []bar, vol int64) strategy.MarketData {
	md := fixture(bars)
	md.Volumes[len(md.Volumes)-1] = vol
	return md
}

// spike decides on bars whose last bar traded vol.
func spike(p Params, bars []bar, vol int64) model.Signal {
	return NewWithParams("TEST", step15, p).Decide(spikeMD(bars, vol))
}

// held decides with a position bought at entry on the bar starting at entryTime, with the daily
// ATR frozen at entry equal to fixtureATR.
func held(p Params, bars []bar, entry float64, entryTime time.Time) model.Signal {
	md := fixture(bars)
	md.Position = &strategy.Position{PurchasePrice: entry, Quantity: 1, EntryTime: entryTime, EntryATR: fixtureATR}
	return NewWithParams("TEST", step15, p).Decide(md)
}

func TestDefaultParams(t *testing.T) {
	want := Params{VolDailyMult: 0.4, VolAvgPeriod: 14, RSIPeriod: 6, RSIExit: 70, StopDailyATR: 1.0, DailyATRPeriod: 14, StartHour: 8}
	if got := DefaultParams(); got != want {
		t.Fatalf("DefaultParams() = %+v, want %+v", got, want)
	}
}

// Monday 2026-09-14 15:30: a red bar trading 0.4 of an average day buys at the next slot.
func TestEnterBuysRedBarOnVolumeSpike(t *testing.T) {
	bars := redAt("2026-09-14 15:30")
	sig := spike(DefaultParams(), bars, 400)
	if sig.Kind != model.SignalBuy {
		t.Fatalf("kind = %v, want Buy", sig.Kind)
	}
	if math.Abs(sig.ATR-fixtureATR) > eps {
		t.Fatalf("ATR = %v, want %v", sig.ATR, fixtureATR)
	}
	if want := at("2026-09-14 15:45"); !sig.FillBy.Equal(want) {
		t.Fatalf("FillBy = %v, want %v", sig.FillBy, want)
	}
	if !strings.HasPrefix(sig.EntryReason, "VOLUME:") || !strings.Contains(sig.EntryReason, "0.40×") {
		t.Fatalf("entry reason = %q, want VOLUME with the 0.40× ratio", sig.EntryReason)
	}
}

func TestEnterVolumeThreshold(t *testing.T) {
	bars := redAt("2026-09-14 15:30")
	if sig := spike(DefaultParams(), bars, 399); sig.Kind != model.SignalNone {
		t.Fatalf("volume 399 < 0.4×1000: kind = %v, want None", sig.Kind)
	}
	if sig := spike(DefaultParams(), bars, 400); sig.Kind != model.SignalBuy {
		t.Fatalf("volume exactly at the threshold: kind = %v, want Buy", sig.Kind)
	}
	p := DefaultParams()
	p.VolDailyMult = 0
	if sig := spike(p, bars, 1_000_000); sig.Kind != model.SignalNone {
		t.Fatalf("VolDailyMult 0: kind = %v, want None", sig.Kind)
	}
}

func TestEnterNeedsRedBar(t *testing.T) {
	bars := redAt("2026-09-14 15:30")
	last := len(bars) - 1
	bars[last] = bar{bars[last].t, 100, 100.5, 100, 100.5} // green
	if sig := spike(DefaultParams(), bars, 1000); sig.Kind != model.SignalNone {
		t.Fatalf("green bar: kind = %v, want None", sig.Kind)
	}
	bars[last] = bar{bars[last].t, 100, 100, 100, 100} // doji
	if sig := spike(DefaultParams(), bars, 1000); sig.Kind != model.SignalNone {
		t.Fatalf("doji: kind = %v, want None", sig.Kind)
	}
}

// Volume is the signal itself, so without a known daily average there is no entry.
func TestEnterNeedsAverageDailyVolume(t *testing.T) {
	bars := redAt("2026-09-14 15:30")
	s := NewWithParams("TEST", step15, DefaultParams())

	md := spikeMD(bars, 1000)
	md.DailyVolumes = nil
	if sig := s.Decide(md); sig.Kind != model.SignalNone {
		t.Fatalf("no daily volumes: kind = %v, want None", sig.Kind)
	}
	md = spikeMD(bars, 1000)
	md.DailyVolumes = md.DailyVolumes[1:]
	if sig := s.Decide(md); sig.Kind != model.SignalNone {
		t.Fatalf("misaligned daily volumes: kind = %v, want None", sig.Kind)
	}

	// The stop is off so that only the volume history matters (the ATR needs 15 dailies).
	p := DefaultParams()
	p.StopDailyATR = 0
	md = spikeMD(bars, 1000)
	withDailies(&md, dailies(bars[0].t, 13, fixtureATR))
	if sig := NewWithParams("TEST", step15, p).Decide(md); sig.Kind != model.SignalNone {
		t.Fatalf("13 days of volume < VolAvgPeriod 14: kind = %v, want None", sig.Kind)
	}
	withDailies(&md, dailies(bars[0].t, 14, fixtureATR))
	if sig := NewWithParams("TEST", step15, p).Decide(md); sig.Kind != model.SignalBuy {
		t.Fatalf("14 days of volume: kind = %v, want Buy", sig.Kind)
	}
}

// Weekend and zero-volume days stay out of the Wilder average.
func TestAvgDailyVolumeWilder(t *testing.T) {
	md := strategy.MarketData{
		DailyTimes: []time.Time{
			at("2026-09-07 00:00"), // Mon 100
			at("2026-09-08 00:00"), // Tue 0 — no trading, dropped
			at("2026-09-09 00:00"), // Wed 200
			at("2026-09-10 00:00"), // Thu 300
			at("2026-09-12 00:00"), // Sat 1e6 — weekend, dropped
			at("2026-09-14 00:00"), // Mon 400
		},
		DailyVolumes: []int64{100, 0, 200, 300, 1_000_000, 400},
	}
	p := DefaultParams()
	p.VolAvgPeriod = 3
	// seed SMA(100, 200, 300) = 200, then 200 + (400 − 200)/3
	want := 200 + 200.0/3
	if got := NewWithParams("TEST", step15, p).avgDailyVolume(md); math.Abs(got-want) > 1e-9 {
		t.Fatalf("avgDailyVolume = %v, want %v", got, want)
	}
	p.VolAvgPeriod = 5 // only four usable days
	if got := NewWithParams("TEST", step15, p).avgDailyVolume(md); got != 0 {
		t.Fatalf("too few days: avgDailyVolume = %v, want 0", got)
	}
}

func TestEnterWindow(t *testing.T) {
	cases := []struct {
		at   string
		want model.SignalKind
	}{
		{"2026-09-14 07:45", model.SignalNone}, // before StartHour 8
		{"2026-09-14 08:00", model.SignalBuy},
		{"2026-09-19 12:00", model.SignalNone}, // Saturday
		{"2026-09-14 23:30", model.SignalBuy},  // fill at 23:45, same date
		{"2026-09-14 23:45", model.SignalNone}, // fill would be tomorrow
	}
	for _, c := range cases {
		if sig := spike(DefaultParams(), redAt(c.at), 1000); sig.Kind != c.want {
			t.Fatalf("%s: kind = %v, want %v", c.at, sig.Kind, c.want)
		}
	}
}

func TestEnterStopGuards(t *testing.T) {
	bars := redAt("2026-09-14 15:30")
	md := spikeMD(bars, 1000)
	md.DailyHighs = nil // the daily ATR becomes unknown, the daily volumes stay
	if sig := NewWithParams("TEST", step15, DefaultParams()).Decide(md); sig.Kind != model.SignalNone {
		t.Fatalf("stop on, ATR unknown: kind = %v, want None", sig.Kind)
	}
	p := DefaultParams()
	p.StopDailyATR = 0
	if sig := NewWithParams("TEST", step15, p).Decide(md); sig.Kind != model.SignalBuy {
		t.Fatalf("stop off, ATR unknown: kind = %v, want Buy", sig.Kind)
	}
	p.StopDailyATR = 50 // 99.5 − 50×2 < 0
	if sig := spike(p, bars, 1000); sig.Kind != model.SignalNone {
		t.Fatalf("stop below zero: kind = %v, want None", sig.Kind)
	}
}

// The step only sets the fill slot.
func TestEnterOnOtherSteps(t *testing.T) {
	sig := NewWithParams("TEST", 5*time.Minute, DefaultParams()).Decide(spikeMD(redAt("2026-09-14 15:30"), 1000))
	if want := at("2026-09-14 15:35"); sig.Kind != model.SignalBuy || !sig.FillBy.Equal(want) {
		t.Fatalf("5m: kind %v FillBy %v, want Buy by %v", sig.Kind, sig.FillBy, want)
	}
}

func TestLookback(t *testing.T) {
	if got := NewWithParams("TEST", step15, DefaultParams()).Lookback(); got != 64 {
		t.Fatalf("default Lookback = %d, want 64", got)
	}
	p := DefaultParams()
	p.RSIPeriod = 20
	if got := NewWithParams("TEST", step15, p).Lookback(); got != 82 {
		t.Fatalf("RSIPeriod 20: Lookback = %d, want 82", got)
	}
}
