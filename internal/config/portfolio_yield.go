package config

type PortfolioYieldConfig struct {
	ManualStartValue float64 `config:"PORTFOLIO_YTD_START_VALUE"`
}

func NewPortfolioYieldConfig() *PortfolioYieldConfig {
	return &PortfolioYieldConfig{
		ManualStartValue: 0,
	}
}
