package core

import (
	"testing"
	"time"

	"tinvest/internal/service/trading_strategy/scalping/strategy"
)

const step = 15 * time.Minute

// at parses "2006-01-02 15:04" as MSK.
func at(s string) time.Time {
	t, err := time.ParseInLocation("2006-01-02 15:04", s, mskLoc)
	if err != nil {
		panic(err)
	}
	return t
}

// mdAt is MarketData with closes c at the given MSK start times.
func mdAt(times []string, closes []float64) strategy.MarketData {
	var md strategy.MarketData
	for k, ts := range times {
		md.Times = append(md.Times, at(ts))
		md.Closes = append(md.Closes, closes[k])
	}
	return md
}

func TestDefaultParams(t *testing.T) {
	want := Params{RSIPeriod: 6, RSIArm: 30, RSIEntry: 30, RSIGate: 50, RSIBreakeven: 50, RSIExit: 70,
		ZoneATR: 0.4, VolumeMult: 1.5, VolBars: 20, MinRoomATR: 0.6, DailyATRPeriod: 14}
	if got := DefaultParams(); got != want {
		t.Fatalf("DefaultParams() = %+v, want %+v", got, want)
	}
}

func TestHourBarsCompleteOnTheQuarterToBar(t *testing.T) {
	times := []string{"2026-09-15 10:00", "2026-09-15 10:15", "2026-09-15 10:30", "2026-09-15 10:45",
		"2026-09-15 11:00", "2026-09-15 11:15", "2026-09-15 11:30", "2026-09-15 11:45"}
	closes := []float64{1, 2, 3, 4, 5, 6, 7, 8}
	hs := NewWithParams("X", step, DefaultParams()).hourBars(mdAt(times, closes))
	if len(hs) != 2 {
		t.Fatalf("hours = %d, want 2", len(hs))
	}
	if hs[0].first != 0 || hs[0].done != 3 || hs[0].close != closes[3] {
		t.Errorf("hour 0 = %+v", hs[0])
	}
	if hs[1].first != 4 || hs[1].done != 7 || hs[1].close != closes[7] {
		t.Errorf("hour 1 = %+v", hs[1])
	}
}

func TestHourBarsMissingCloseBar(t *testing.T) {
	times := []string{"2026-09-15 10:00", "2026-09-15 10:15", "2026-09-15 10:30", "2026-09-15 11:00"}
	hs := NewWithParams("X", step, DefaultParams()).hourBars(mdAt(times, []float64{1, 2, 3, 4}))
	if len(hs) != 1 {
		t.Fatalf("hours = %d, want 1 (the 11:00 hour is still forming)", len(hs))
	}
	if hs[0].done != 3 || hs[0].last != 2 {
		t.Errorf("hour 0 = %+v, want done 3, last 2", hs[0])
	}
}

func TestLookbackCoversHoldAndWarmup(t *testing.T) {
	s := NewWithParams("X", step, DefaultParams())
	if lb := s.Lookback(); lb < (holdDays+3)*dayBars || lb < 4*(4*DefaultParams().RSIPeriod+2+64) {
		t.Errorf("Lookback = %d too small", lb)
	}
	big := DefaultParams()
	big.VolBars = 200
	if NewWithParams("X", step, big).Lookback() <= s.Lookback() {
		t.Error("Lookback must grow with VolBars")
	}
}
