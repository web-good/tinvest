package tbank

import (
	"testing"

	"tinvest/internal/service/trading_strategy/drop_bounce/strategy/core"
)

// Литералы прибиты снимками: правка любого числа меняет то, что гоняет бэктест T на своём
// таймфрейме.
func TestLiteralsArePinned(t *testing.T) {
	cases := []struct {
		name string
		got  core.Params
		want core.Params
	}{
		{"Minutes5", ParamsMinutes5(), core.Params{
			VolDailyMult:   0.4,
			VolAvgPeriod:   14,
			RSIPeriod:      6,
			RSIExit:        70,
			StopDailyATR:   1,
			DailyATRPeriod: 14,
			StartHour:      7,
		}},
		{"Minutes15", ParamsMinutes15(), core.Params{
			VolDailyMult:   0.1,
			VolAvgPeriod:   12,
			RSIPeriod:      4,
			RSIExit:        51,
			StopDailyATR:   1.02,
			DailyATRPeriod: 8,
			StartHour:      7,
		}},
		{"Minutes30", ParamsMinutes30(), core.Params{
			VolDailyMult:   0.21,
			VolAvgPeriod:   6,
			RSIPeriod:      4,
			RSIExit:        64,
			StopDailyATR:   1,
			DailyATRPeriod: 7,
			StartHour:      7,
		}},
		{"Hour1", ParamsHour1(), core.Params{
			VolDailyMult:   0.29,
			VolAvgPeriod:   7,
			RSIPeriod:      6,
			RSIExit:        77,
			StopDailyATR:   0.79,
			DailyATRPeriod: 7,
			StartHour:      7,
		}},
	}
	for _, c := range cases {
		if c.got != c.want {
			t.Errorf("%s: got %+v, want %+v", c.name, c.got, c.want)
		}
	}
}
