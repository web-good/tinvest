package backtest

import (
	"math"
	"testing"

	"tinvest/internal/service/trading_strategy/scalping/model"
	"tinvest/internal/service/trading_strategy/scalping/strategy"
)

func TestTPExecutionPrice(t *testing.T) {
	for _, tc := range []struct {
		name, mode        string
		open, target, bps float64
		exact             bool
		want              float64
	}{
		{name: "default touch", open: 100, target: 105, want: 105},
		{name: "close after wick", mode: "bar-close", open: 100, target: 105, want: 102},
		{name: "gap", open: 107, target: 105, want: 107},
		{name: "exact", open: 107, target: 105, exact: true, want: 105},
		{name: "touch slippage", open: 100, target: 105, bps: 10, want: 104.895},
		{name: "close slippage", mode: "bar-close", open: 100, target: 105, bps: 10, want: 101.898},
		{name: "no target", open: 100, want: 102},
	} {
		t.Run(tc.name, func(t *testing.T) {
			e := TPExecution{Mode: tc.mode, SlippageBPS: tc.bps}
			got := e.price(model.Signal{TakeProfit: tc.target, ExactTargetFill: tc.exact}, Candle{Open: tc.open, High: 108, Low: 99, Close: 102})
			if math.Abs(got-tc.want) > 1e-9 {
				t.Fatalf("price=%v want %v", got, tc.want)
			}
		})
	}
}

func TestTPExecutionValidation(t *testing.T) {
	for _, tc := range []struct {
		name, mode, strategy string
		bps                  float64
		valid                bool
	}{
		{name: "zero legacy", strategy: "reversion", valid: true},
		{name: "touch", mode: "touch", strategy: "rsi_pullback", valid: true},
		{name: "close", mode: "bar-close", strategy: "rsi_pullback", bps: 10, valid: true},
		{name: "unknown", mode: "typo", strategy: "rsi_pullback"},
		{name: "foreign", mode: "bar-close", strategy: "rsi_zone"},
		{name: "foreign slippage", strategy: "rsi_zone", bps: 5},
		{name: "negative", strategy: "rsi_pullback", bps: -1},
		{name: "nan", strategy: "rsi_pullback", bps: math.NaN()},
		{name: "inf", strategy: "rsi_pullback", bps: math.Inf(1)},
		{name: "100 percent", strategy: "rsi_pullback", bps: 10000},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := (TPExecution{Mode: tc.mode, SlippageBPS: tc.bps}).Validate(tc.strategy)
			if (err == nil) != tc.valid {
				t.Fatalf("err=%v valid=%v", err, tc.valid)
			}
		})
	}
}

func TestTPExecutionRunAndTrace(t *testing.T) {
	for _, mode := range []string{"touch", "bar-close"} {
		t.Run(mode, func(t *testing.T) {
			candles := flatCandles([]float64{100, 102, 100, 101})
			candles[1].Open = 100
			candles[1].High = 106
			cfg := Config{InitialCash: 10000, Fraction: 1, Lot: 1, TPExecution: TPExecution{Mode: mode}}
			makeStrategy := func(seen *int64) scriptedStrategy {
				return scriptedStrategy{lookback: 1, decide: func(md strategy.MarketData) model.Signal {
					if md.Position == nil && md.Price == 100 {
						return model.Signal{Kind: model.SignalBuy, TakeProfit: 105}
					}
					if md.Position != nil && md.Price == 102 {
						return model.Signal{Kind: model.SignalSell, Reason: "TP", TakeProfit: 105}
					}
					if md.Position != nil && md.Price == 101 {
						*seen = md.Position.Quantity
					}
					return model.Signal{}
				}}
			}
			var runQty, traceQty int64
			res := Run(makeStrategy(&runQty), candles, nil, nil, cfg)
			Trace(makeStrategy(&traceQty), candles, nil, nil, cfg, candles[3].Time)
			want := 105.0
			if mode == "bar-close" {
				want = 102
			}
			if len(res.Trades) != 1 || res.Trades[0].ExitPrice != want {
				t.Fatalf("trades=%+v", res.Trades)
			}
			if runQty != int64(want) || traceQty != runQty {
				t.Fatalf("Run qty=%v Trace qty=%v want=%v", runQty, traceQty, want)
			}
		})
	}
}

func TestTPExecutionCommissionAndOtherExit(t *testing.T) {
	for _, reason := range []string{"TP", "RSI", "SL"} {
		t.Run(reason, func(t *testing.T) {
			s := scriptedStrategy{lookback: 1, decide: func(md strategy.MarketData) model.Signal {
				if md.Position == nil {
					return model.Signal{Kind: model.SignalBuy}
				}
				return model.Signal{Kind: model.SignalSell, Reason: reason, TakeProfit: 105, StopLoss: 95}
			}}
			candles := flatCandles([]float64{100, 102})
			candles[1].Open = 100
			candles[1].High = 106
			candles[1].Low = 94
			cfg := Config{InitialCash: 10000, Fraction: 1, Lot: 1, Commission: 0.001, TPExecution: TPExecution{Mode: "bar-close", SlippageBPS: 10}}
			res := Run(s, candles, nil, nil, cfg)
			tr := res.Trades[0]
			price := 102.0
			if reason == "TP" {
				price = 101.898
			}
			if reason == "SL" {
				price = 95
			}
			if math.Abs(tr.ExitPrice-price) > 1e-9 {
				t.Fatalf("price=%v want %v", tr.ExitPrice, price)
			}
			want := float64(tr.Quantity) * (price*(1-cfg.Commission) - 100*(1+cfg.Commission))
			if math.Abs(tr.PnL-want) > 1e-8 {
				t.Fatalf("PnL=%v want %v", tr.PnL, want)
			}
		})
	}
}
