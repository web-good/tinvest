package plzl

import (
	"testing"

	"tinvest/internal/service/trading_strategy/drop_bounce/strategy/core"
)

// Литералы прибиты снимками: правка любого числа меняет то, что гоняет бэктест PLZL на своём
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
			VolDailyMult:   0.17,
			VolAvgPeriod:   6,
			RSIPeriod:      6,
			RSIExit:        69,
			StopDailyATR:   0.7,
			DailyATRPeriod: 8,
			StartHour:      7,
		}},
		{"Minutes30", ParamsMinutes30(), core.Params{
			VolDailyMult:   0.2,
			VolAvgPeriod:   10,
			RSIPeriod:      6,
			RSIExit:        64,
			StopDailyATR:   1.12,
			DailyATRPeriod: 8,
			StartHour:      7,
		}},
		{"Hour1", ParamsHour1(), core.Params{
			VolDailyMult:   0.34,
			VolAvgPeriod:   13,
			RSIPeriod:      6,
			RSIExit:        70,
			StopDailyATR:   0.97,
			DailyATRPeriod: 9,
			StartHour:      7,
		}},
	}
	for _, c := range cases {
		if c.got != c.want {
			t.Errorf("%s: got %+v, want %+v", c.name, c.got, c.want)
		}
	}
}
