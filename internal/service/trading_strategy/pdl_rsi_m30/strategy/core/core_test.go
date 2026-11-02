package core

import (
	"testing"
	"time"
)

const step = 30 * time.Minute

// at parses "2006-01-02 15:04" as MSK.
func at(s string) time.Time {
	t, err := time.ParseInLocation("2006-01-02 15:04", s, mskLoc)
	if err != nil {
		panic(err)
	}
	return t
}

func TestDefaultParams(t *testing.T) {
	want := Params{RSIPeriod: 6, RSIArm: 30, RSIFloor: 22, RSIExit: 70,
		StochK: 14, StochSmooth: 1, StochD: 3, VolumeMult: 0, VolBars: 20, MinRoomATR: 0, StopATR: 0.5, TargetR: 2, TrendEMA: 200, DailyATRPeriod: 14}
	if got := DefaultParams(); got != want {
		t.Fatalf("DefaultParams() = %+v, want %+v", got, want)
	}
}

func TestLookbackCoversWarmup(t *testing.T) {
	s := NewWithParams("X", step, DefaultParams())
	// The volume baseline days, a weekend of up to 44 half-hours a day (02:00–23:30) and the current
	// day; twice the trend EMA period.
	if dayBars != 36 || longDayBars != 44 || sessionBars != 18 {
		t.Fatalf("dayBars/longDayBars/sessionBars = %d/%d/%d, want 36/44/18", dayBars, longDayBars, sessionBars)
	}
	if lb := s.Lookback(); lb < (3+2)*36+2*44 || lb < 2*200 {
		t.Errorf("Lookback = %d too small", lb)
	}
	small, big := DefaultParams(), DefaultParams()
	small.TrendEMA, big.TrendEMA, big.VolBars = 0, 0, 200
	if NewWithParams("X", step, big).Lookback() <= NewWithParams("X", step, small).Lookback() {
		t.Error("Lookback must grow with VolBars")
	}
}
