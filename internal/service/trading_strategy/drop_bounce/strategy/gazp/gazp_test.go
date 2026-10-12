package gazp

import (
	"testing"

	"tinvest/internal/service/trading_strategy/drop_bounce/strategy/core"
)

// Литералы прибиты снимками: правка любого числа меняет то, что гоняет бэктест GAZP на своём
// таймфрейме.
func TestLiteralsArePinned(t *testing.T) {
	cases := []struct {
		name string
		got  core.Params
		want core.Params
	}{
		{"Minutes5", ParamsMinutes5(), core.Params{
			MinDropPct:     0.6,
			RSIPeriod:      6,
			RSIEntry:       30,
			RSIExit:        70,
			ExitInProfit:   0,
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
			GreenWaitBars:  0,
		}},
		{"Minutes15", ParamsMinutes15(), core.Params{
			MinDropPct:     1.3,
			RSIPeriod:      6,
			RSIEntry:       30,
			RSIExit:        70,
			ExitInProfit:   0,
			StochPeriod:    14,
			StochSmooth:    3,
			StochExit:      80,
			ProfitPct:      1,
			StopDailyATR:   0.7,
			DailyATRPeriod: 14,
			MaxDayRangeATR: 0,
			VolMult:        1.4,
			VolBars:        20,
			StartHour:      8,
			GreenWaitBars:  0,
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
			GreenWaitBars:  0,
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
			GreenWaitBars:  0,
		}},
	}
	for _, c := range cases {
		if c.got != c.want {
			t.Errorf("литерал GAZP %s изменился:\n got: %+v\nwant: %+v", c.name, c.got, c.want)
		}
	}
}
