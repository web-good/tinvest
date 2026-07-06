package config

// ReversionConfig configures the live reversion runner. Trade and notify are
// independent: both off = dry-run to log only.
type ReversionConfig struct {
	AccountID     string   `config:"REVERSION_ACCOUNT_ID,required,backend=env"`
	Token         string   `config:"REVERSION_TOKEN,required,backend=env"`
	Tickers       []string `config:"REVERSION_TICKERS,backend=env"`
	BuyPct        float64  `config:"REVERSION_BUY_PCT,backend=env"`
	TradeEnabled  bool     `config:"REVERSION_TRADE_ENABLED,backend=env"`
	NotifyEnabled bool     `config:"REVERSION_NOTIFY_ENABLED,backend=env"`
}

// NewReversionConfig returns the config pre-seeded with safe defaults. confita
// overrides any field whose env var is set; unset fields keep these values.
// TradeEnabled defaults to false so a missing flag never places real orders.
func NewReversionConfig() *ReversionConfig {
	return &ReversionConfig{
		Tickers: []string{"UGLD", "EUTR", "NVTK"},
		BuyPct:  10,
	}
}
