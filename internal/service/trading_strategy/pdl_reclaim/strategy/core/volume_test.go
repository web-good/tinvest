package core

import (
	"strings"
	"testing"
	"time"

	"tinvest/internal/service/trading_strategy/scalping/model"
	"tinvest/internal/service/trading_strategy/scalping/strategy"
)

// withVolumes sets every bar's volume from vol(bar start).
func withVolumes(md strategy.MarketData, vol func(time.Time) int64) strategy.MarketData {
	md.Volumes = make([]int64, len(md.Times))
	for i, t := range md.Times {
		md.Volumes[i] = vol(t)
	}
	return md
}

// priorDays returns, for each of the `days` weekdays before Monday 2026-09-14 (newest last), the
// bars of slot(date) flat at 102, plus one Saturday 2026-09-12 session of the same slot.
func priorDays(days int, slot func(date string) []bar) []bar {
	var dates []string
	for d := at("2026-09-14 00:00").AddDate(0, 0, -1); len(dates) < days; d = d.AddDate(0, 0, -1) {
		if !isWeekend(d) {
			dates = append([]string{d.Format("2006-01-02")}, dates...)
		}
	}
	var out []bar
	for _, d := range dates {
		out = append(out, slot(d)...)
	}
	return append(out, slot("2026-09-12")...)
}

// The 5m slot bars of past days (10:00) carry volume 100; their 09:55 neighbours and the Saturday
// session carry 10000, so a filter reading anything but the same weekday slot is visible.
func vol5m(signal int64) func(time.Time) int64 {
	return func(t time.Time) int64 {
		switch {
		case dayKey(t) == dayKey(at("2026-09-14 00:00")):
			if minuteOf(t) == 10*60 {
				return signal
			}
			return 10000
		case isWeekend(t.In(mskLoc)):
			return 10000
		case minuteOf(t) == 10*60:
			return 100
		}
		return 10000
	}
}

func slot5m(date string) []bar {
	return []bar{{date + " 09:55", 102, 102, 102, 102}, {date + " 10:00", 102, 102, 102, 102}}
}

func volMD(days int, signal int64) strategy.MarketData {
	bars := append(priorDays(days, slot5m), warm([]bar{pre, barA})...)
	return withVolumes(fixture(bars), vol5m(signal))
}

func TestVolume5mSameSlot(t *testing.T) {
	p := m5()
	p.VolumeMult, p.VolDays = 1, 3
	if sig := newS(p).Decide(volMD(3, 150)); sig.Kind != model.SignalBuy {
		t.Fatalf("volume 150 > slot average 100: Kind = %v, want Buy", sig.Kind)
	}
	if ex := newS(p).Explain(volMD(3, 150)); !strings.Contains(ex, "✓ объём свечи паттерна 150 > 1.00 × среднее слота 10:00 за 3 будних дней 100") {
		t.Fatalf("Explain misses the volume gate:\n%s", ex)
	}
	if sig := newS(p).Decide(volMD(3, 100)); sig.Kind != model.SignalNone {
		t.Fatalf("volume 100 = slot average: Kind = %v, want None (strictly above)", sig.Kind)
	}
	p.VolumeMult = 2
	if sig := newS(p).Decide(volMD(3, 150)); sig.Kind != model.SignalNone {
		t.Fatalf("volume 150 ≤ 2 × 100: Kind = %v, want None", sig.Kind)
	}
}

func TestVolumeFailsClosed(t *testing.T) {
	p := m5()
	p.VolumeMult, p.VolDays = 1, 3
	if sig := newS(p).Decide(volMD(2, 150)); sig.Kind != model.SignalNone {
		t.Fatalf("two past days for VolDays 3: Kind = %v, want None", sig.Kind)
	}
	md := volMD(3, 150)
	md.Volumes = nil
	if sig := newS(p).Decide(md); sig.Kind != model.SignalNone {
		t.Fatalf("no volumes: Kind = %v, want None", sig.Kind)
	}
}

func TestVolumeOffIgnoresVolumes(t *testing.T) {
	md := volMD(3, 1)
	if sig := newS(m5()).Decide(md); sig.Kind != model.SignalBuy {
		t.Fatalf("VolumeMult 0: Kind = %v, want Buy", sig.Kind)
	}
}

// Engulfing B: the volume read is the green bar's, not the red one's.
func TestVolume5mEntryBReadsGreenBar(t *testing.T) {
	p := m5()
	p.VolumeMult, p.VolDays = 1, 3
	bars := append(priorDays(3, slot5m), warm([]bar{redB, greenB})...)
	md := withVolumes(fixture(bars), vol5m(150)) // red 09:55 today carries 10000
	if sig := newS(p).Decide(md); sig.Kind != model.SignalBuy || !strings.HasPrefix(sig.EntryReason, "B") {
		t.Fatalf("got %v %q, want Buy B", sig.Kind, sig.EntryReason)
	}
	md = withVolumes(fixture(bars), vol5m(50))
	if sig := newS(p).Decide(md); sig.Kind != model.SignalNone {
		t.Fatalf("green bar below average: Kind = %v, want None", sig.Kind)
	}
}

// On 30m the half-hour volume (sum of its six 5m bars) is compared with the same half-hour of past
// days: past 10:00 half-hours sum 600, their 09:30 neighbours 60000.
func TestVolume30mSameSlot(t *testing.T) {
	p := DefaultParams()
	p.VolumeMult, p.VolDays = 1, 3
	slot := func(date string) []bar {
		return append(half(date+" 09:30", 102, 102, 102, 102), half(date+" 10:00", 102, 102, 102, 102)...)
	}
	vol := func(signal int64) func(time.Time) int64 {
		return func(t time.Time) int64 {
			inSlot := minuteOf(halfStart(t)) == 10*60
			switch {
			case dayKey(t) == dayKey(at("2026-09-14 00:00")):
				if inSlot {
					return signal
				}
				return 10000
			case isWeekend(t.In(mskLoc)):
				return 10000
			case inSlot:
				return 100
			}
			return 10000
		}
	}
	bars := append(priorDays(3, slot), append(morning(), halfA()...)...)
	if sig := newS(p).Decide(withVolumes(fixture(bars), vol(101))); sig.Kind != model.SignalBuy {
		t.Fatalf("half-hour 606 > 600: Kind = %v, want Buy", sig.Kind)
	}
	if sig := newS(p).Decide(withVolumes(fixture(bars), vol(100))); sig.Kind != model.SignalNone {
		t.Fatalf("half-hour 600 = 600: Kind = %v, want None", sig.Kind)
	}
}

func TestLookbackCoversVolDays(t *testing.T) {
	p := DefaultParams()
	base := newS(p).Lookback()
	p.VolumeMult, p.VolDays = 1, 20
	if l := newS(p).Lookback(); l < (20*7/5+3)*dayBars || l <= base {
		t.Fatalf("Lookback %d with VolDays 20, want ≥ %d", l, (20*7/5+3)*dayBars)
	}
}

func TestEntry5mOff(t *testing.T) {
	p := m5()
	p.Entry5m = 0
	if sig := decide(p, []bar{pre, barA}); sig.Kind != model.SignalNone {
		t.Fatalf("Entry5m 0: Kind = %v, want None", sig.Kind)
	}
	// half-hours still trade with the 5m morning off
	p = DefaultParams()
	p.Entry5m = 0
	if sig := newS(p).Decide(fixture(append(morning(), halfA()...))); sig.Kind != model.SignalBuy {
		t.Fatalf("Entry5m 0, 30m A: Kind = %v, want Buy", sig.Kind)
	}
}

func TestEntry30mOff(t *testing.T) {
	p := DefaultParams()
	p.Entry30m = 0
	if sig := newS(p).Decide(fixture(append(morning(), halfA()...))); sig.Kind != model.SignalNone {
		t.Fatalf("Entry30m 0: Kind = %v, want None", sig.Kind)
	}
	b, q := barA, pre
	b.t, q.t = "2026-09-14 09:55", "2026-09-14 09:50"
	if sig := decide(p, []bar{q, b}); sig.Kind != model.SignalBuy {
		t.Fatalf("Entry30m 0, 5m A at 09:55: Kind = %v, want Buy", sig.Kind)
	}
}

// The flags gate entries only: a held position still exits.
func TestEntryFlagsKeepExits(t *testing.T) {
	p := quiet()
	p.Entry5m, p.Entry30m = 0, 0
	sig := held(p, []bar{fillBar, {"2026-09-14 10:10", 99.4, 99.4, 98.3, 98.5}}, pos0())
	if sig.Reason != "SL" {
		t.Fatalf("Reason = %q, want SL", sig.Reason)
	}
}
