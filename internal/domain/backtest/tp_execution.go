package backtest

import (
	"fmt"
	"math"

	"tinvest/internal/service/trading_strategy/scalping/model"
)

// TPExecution controls fill assumptions without changing strategy signals.
// Zero values preserve historical target fills.
type TPExecution struct {
	Mode        string
	SlippageBPS float64
}

// Validate rejects unsupported execution assumptions before loading market data.
func (e TPExecution) Validate(strategyName string) error {
	if e.Mode != "" && e.Mode != "touch" && e.Mode != "bar-close" {
		return fmt.Errorf("-tp-fill must be touch or bar-close, got %q", e.Mode)
	}
	if math.IsNaN(e.SlippageBPS) || math.IsInf(e.SlippageBPS, 0) || e.SlippageBPS < 0 || e.SlippageBPS >= 10000 {
		return fmt.Errorf("-tp-slippage-bps must be finite and in [0, 10000)")
	}
	if strategyName != "rsi_pullback" && (e.Mode == "bar-close" || e.SlippageBPS != 0) {
		return fmt.Errorf("custom TP execution requires -strategy rsi_pullback")
	}
	return nil
}

func (e TPExecution) price(sig model.Signal, c Candle) float64 {
	price := c.Close
	if e.Mode != "bar-close" && sig.TakeProfit > 0 {
		price = sig.TakeProfit
		if !sig.ExactTargetFill {
			price = max(price, c.Open)
		}
	}
	return price * (1 - e.SlippageBPS/10000)
}

// Description records that fills are modeled, not measured broker executions.
func (e TPExecution) Description() string {
	mode := e.Mode
	if mode == "" {
		mode = "touch"
	}
	return fmt.Sprintf("TP fill: %s; adverse slippage: %g bps (scenario assumption; bar-close approximates delayed live execution)", mode, e.SlippageBPS)
}
