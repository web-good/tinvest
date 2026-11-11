package config

import "strings"

// FVGImbalanceConfig configures the FVG guest of the shared account runner.
type FVGImbalanceConfig struct {
	Tickers       []string `config:"FVG_IMBALANCE_TICKERS,backend=env"`
	BuyPct        float64  `config:"FVG_IMBALANCE_BUY_PCT,backend=env"`
	TradeEnabled  bool     `config:"FVG_IMBALANCE_TRADE_ENABLED,backend=env"`
	NotifyEnabled bool     `config:"FVG_IMBALANCE_NOTIFY_ENABLED,backend=env"`
}

func NewFVGImbalanceConfig() *FVGImbalanceConfig { return &FVGImbalanceConfig{BuyPct: 5} }

func (c *FVGImbalanceConfig) Enabled() bool {
	for _, ticker := range c.Tickers {
		if strings.TrimSpace(ticker) != "" {
			return true
		}
	}
	return false
}
