package config

type PortfolioYieldConfig struct {
	SnapshotPath     string  `config:"PORTFOLIO_SNAPSHOT_PATH"`
	ManualStartValue float64 `config:"PORTFOLIO_YTD_START_VALUE"`
}

func NewPortfolioYieldConfig() *PortfolioYieldConfig {
	return &PortfolioYieldConfig{
		SnapshotPath:     "./cache/portfolio_snapshots.json",
		ManualStartValue: 0,
	}
}
