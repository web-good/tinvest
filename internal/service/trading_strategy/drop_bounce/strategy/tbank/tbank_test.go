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
			MinDropPct:     0.5,
			RSIPeriod:      3,
			RSIEntry:       20,
			RSIExit:        70,
			ExitInProfit:   0,
			StochPeriod:    14,
			StochSmooth:    3,
			StochExit:      0,
			ProfitPct:      0.5,
			StopDailyATR:   1.15,
			DailyATRPeriod: 14,
			MaxDayRangeATR: 0,
			VolMult:        0,
			VolBars:        20,
			StartHour:      8,
		}},
		{"Minutes15", ParamsMinutes15(), core.Params{
			MinDropPct:     1,
			RSIPeriod:      6,
			RSIEntry:       30,
			RSIExit:        70,
			ExitInProfit:   1,
			StochPeriod:    14,
			StochSmooth:    3,
			StochExit:      80,
			ProfitPct:      1,
			StopDailyATR:   0.5,
			DailyATRPeriod: 14,
			MaxDayRangeATR: 0,
			VolMult:        1.5,
			VolBars:        20,
			StartHour:      8,
			GreenWaitBars:  3,
		}},
		{"Minutes30", ParamsMinutes30(), core.Params{
			MinDropPct:     0.6,
			RSIPeriod:      6,
			RSIEntry:       30,
			RSIExit:        70,
			ExitInProfit:   1,
			StochPeriod:    14,
			StochSmooth:    3,
			StochExit:      80,
			ProfitPct:      1,
			StopDailyATR:   0.5,
			DailyATRPeriod: 14,
			MaxDayRangeATR: 0,
			VolMult:        0,
			VolBars:        20,
			StartHour:      8,
		}},
		{"Hour1", ParamsHour1(), core.Params{
			MinDropPct:     0.6,
			RSIPeriod:      6,
			RSIEntry:       30,
			RSIExit:        70,
			ExitInProfit:   1,
			StochPeriod:    14,
			StochSmooth:    3,
			StochExit:      80,
			ProfitPct:      1,
			StopDailyATR:   0.5,
			DailyATRPeriod: 14,
			MaxDayRangeATR: 0,
			VolMult:        0,
			VolBars:        20,
			StartHour:      8,
		}},
	}
	for _, c := range cases {
		if c.got != c.want {
			t.Errorf("литерал T %s изменился:\n got: %+v\nwant: %+v", c.name, c.got, c.want)
		}
	}
}
