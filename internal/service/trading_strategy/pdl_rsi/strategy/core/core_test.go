package core

import (
	"testing"
	"time"
)

const step = time.Hour

// at parses "2006-01-02 15:04" as MSK.
func at(s string) time.Time {
	t, err := time.ParseInLocation("2006-01-02 15:04", s, mskLoc)
	if err != nil {
		panic(err)
	}
	return t
}

func TestDefaultParams(t *testing.T) {
	want := Params{HourRSIPeriod: 6, RSIArm: 30, RSIFloor: 22, RSITriggerMax: 50, RSIExit: 70,
		StochK: 5, StochSmooth: 3, StochD: 3, StochBars: 4, StochCrossMax: 40,
		VolumeMult: 0, VolBars: 20, MinRoomATR: 0, StopATR: 0.5, TargetR: 2, TrendEMA: 200, DailyATRPeriod: 14}
	if got := DefaultParams(); got != want {
		t.Fatalf("DefaultParams() = %+v, want %+v", got, want)
	}
}

func TestLookbackCoversWarmup(t *testing.T) {
	s := NewWithParams("X", step, DefaultParams())
	// The volume baseline days, a weekend of up to 22 hours a day (02:00–23:00) and the current
	// day; twice the trend EMA period.
	if dayBars != 18 || longDayBars != 22 {
		t.Fatalf("dayBars/longDayBars = %d/%d, want 18/22", dayBars, longDayBars)
	}
	if lb := s.Lookback(); lb < (3+2)*18+2*22 || lb < 2*200 {
		t.Errorf("Lookback = %d too small", lb)
	}
	small, big := DefaultParams(), DefaultParams()
	small.TrendEMA, big.TrendEMA, big.VolBars = 0, 0, 200
	if NewWithParams("X", step, big).Lookback() <= NewWithParams("X", step, small).Lookback() {
		t.Error("Lookback must grow with VolBars")
	}
}
