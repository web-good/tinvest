package core

import (
	"testing"
	"time"
	bt "tinvest/internal/domain/backtest"
)

func TestEngineActualFillSetsATRLevels(t *testing.T) {
	for _, tc := range []struct {
		name   string
		after  []bar
		reason string
		exit   float64
	}{
		{"target on fill bar", []bar{{105, 110, 104, 109}}, "TP", 109},
		{"gap through stop", []bar{{105, 108, 104, 104}, {100, 102, 99, 101}}, "SL", 100},
		{"gap through target", []bar{{105, 108, 104, 104}, {112, 113, 111, 112}}, "TP", 112},
	} {
		t.Run(tc.name, func(t *testing.T) {
			bars := make([]bar, 30)
			for i := range bars {
				bars[i] = bar{99, 100, 98, 99}
			}
			bars = append(bars, formation()...)
			bars = append(bars, bar{103, 104, 98, 98})
			bars = append(bars, tc.after...)
			md := snapshot(bars...)
			candles := make([]bt.Candle, len(bars))
			for i, b := range bars {
				candles[i] = bt.Candle{Time: md.Times[i], Open: b.o, High: b.h, Low: b.l, Close: b.c}
			}
			var daily []bt.Candle
			for i := range md.DailyTimes {
				daily = append(daily, bt.Candle{Time: md.DailyTimes[i], High: 101, Low: 99, Close: 100})
			}
			result := bt.Run(NewWithParams("TEST", 30*time.Minute, testParams()), candles, daily, nil, bt.Config{InitialCash: 100000, Fraction: 1, Lot: 1, EntryAtNextOpen: true})
			if len(result.Trades) != 1 {
				t.Fatalf("trades: %+v", result.Trades)
			}
			trade := result.Trades[0]
			if trade.EntryPrice != 105 || trade.StopLoss != 103 || trade.ExitPrice != tc.exit || trade.Reason != tc.reason {
				t.Fatalf("trade: %+v", trade)
			}
		})
	}
}
