package core

import (
	"strings"
	"testing"
	"time"

	"tinvest/internal/service/trading_strategy/scalping/model"
	"tinvest/pkg/indicators"
)

// hourRSIAt is RSI(period) of hourly closes built by hand from bars started before end.
func hourRSIAt(bars []bar, end string, period int) float64 {
	var closes []float64
	var last time.Time
	for _, b := range bars {
		t := at(b.t)
		if !t.Before(at(end)) {
			break
		}
		h := t.Truncate(time.Hour)
		if len(closes) == 0 || !h.Equal(last) {
			closes = append(closes, b.c)
			last = h
		} else {
			closes[len(closes)-1] = b.c
		}
	}
	return indicators.RSISeries(closes, period)[len(closes)-1]
}

// slide is Monday 00:00–09:55, rising +1.2 in even hours and falling −1.32 in odd ones (100 down
// to 99.4): hourly RSI(6) on the 10:00 hour is low but not zero, so a threshold can be set on
// either side of it.
func slide() []bar {
	var out []bar
	c := 100.0
	for t := at("2026-09-14 00:00"); t.Before(at("2026-09-14 10:00")); t = t.Add(5 * time.Minute) {
		if t.In(mskLoc).Hour()%2 == 0 {
			c += 0.1
		} else {
			c -= 0.11
		}
		out = append(out, bar{t.Format("2006-01-02 15:04"), c, c, c, c})
	}
	return out
}

func TestHourRSIGate(t *testing.T) {
	bars := append(slide(), halfA()...)
	rsi := hourRSIAt(bars, "2026-09-14 10:30", 6)
	if rsi <= 0 || rsi >= 50 {
		t.Fatalf("fixture hourly RSI %v, want inside (0, 50)", rsi)
	}
	p := DefaultParams()
	p.HourRSIMax = rsi + 0.5
	sig := newS(p).Decide(fixture(bars))
	if sig.Kind != model.SignalBuy || !strings.Contains(sig.EntryReason, "RSI(6) 1h") {
		t.Fatalf("hourly RSI %.2f < %.2f: %v %q, want Buy with 1h RSI", rsi, p.HourRSIMax, sig.Kind, sig.EntryReason)
	}
	p.HourRSIMax = rsi
	if sig := newS(p).Decide(fixture(bars)); sig.Kind != model.SignalNone {
		t.Fatalf("hourly RSI %.2f = max: Kind = %v, want None (strictly below)", rsi, sig.Kind)
	}
}

// The hourly gate replaces the half-hour one: RSIEntryMax no longer matters after the switch.
func TestHourRSIReplaces30mGate(t *testing.T) {
	p := DefaultParams()
	p.RSIEntryMax, p.HourRSIMax = 0.001, 50 // slide's half-hour RSI is above 0.001, its hourly below 50
	bars := append(slide(), halfA()...)
	if sig := newS(p).Decide(fixture(bars)); sig.Kind != model.SignalBuy {
		t.Fatalf("RSIEntryMax tiny, hourly gate on: Kind = %v, want Buy", sig.Kind)
	}
	p.HourRSIMax = 0 // back to the half-hour gate
	if sig := newS(p).Decide(fixture(bars)); sig.Kind != model.SignalNone {
		t.Fatalf("hourly gate off, RSIEntryMax tiny: Kind = %v, want None", sig.Kind)
	}
}

// The morning 5m gate stays RSI(6) of 5m ≤ RSIEntryMax; the hourly one does not touch it.
func TestHourRSIKeeps5mGate(t *testing.T) {
	p := m5()
	p.HourRSIMax = 0.001
	if sig := decide(p, []bar{pre, barA}); sig.Kind != model.SignalBuy {
		t.Fatalf("5m signal, hourly max tiny: Kind = %v, want Buy", sig.Kind)
	}
	if sig := newS(p).Decide(fixture(rally())); sig.Kind != model.SignalNone {
		t.Fatalf("5m RSI above 50: Kind = %v, want None", sig.Kind)
	}
}

func TestHourRSINeedsHistory(t *testing.T) {
	bars := append(flat("2026-09-14 07:00", "2026-09-14 09:55", 102), halfA()...) // four hours
	if sig := newS(DefaultParams()).Decide(fixture(bars)); sig.Kind != model.SignalNone {
		t.Fatalf("four hours for RSI(6): Kind = %v, want None", sig.Kind)
	}
}

// With Confirm the hour is read when the pattern half-hour closed: the confirming half-hour's rally
// in the same hour does not count.
func TestHourRSIConfirmReadsPatternTime(t *testing.T) {
	bars := append(slide(), halfA()...)
	bars = append(bars, half("2026-09-14 10:30", 99.3, 100.5, 99.3, 100.4)...)
	atPattern := hourRSIAt(bars, "2026-09-14 10:30", 6)
	full := hourRSIAt(bars, "2026-09-14 11:00", 6)
	if full <= atPattern+1 {
		t.Fatalf("fixture: full hour RSI %v must be above pattern-time %v", full, atPattern)
	}
	p := DefaultParams()
	p.Confirm = 1
	p.HourRSIMax = (atPattern + full) / 2
	if sig := newS(p).Decide(fixture(bars)); sig.Kind != model.SignalBuy {
		t.Fatalf("pattern-time RSI %.2f < %.2f: Kind = %v, want Buy", atPattern, p.HourRSIMax, sig.Kind)
	}
}

func TestExplainHourRSI(t *testing.T) {
	out := newS(DefaultParams()).Explain(fixture(append(slide(), halfA()...)))
	if !strings.Contains(out, "RSI(6) 1h") {
		t.Fatalf("Explain misses the hourly RSI:\n%s", out)
	}
}
