package config

// RSIPullbackConfig configures the live rsi_pullback runner. Trade and notify are
// independent: both off = dry-run to log only.
type RSIPullbackConfig struct {
	AccountID     string   `config:"RSI_PULLBACK_ACCOUNT_ID,required,backend=env"`
	Token         string   `config:"RSI_PULLBACK_TOKEN,required,backend=env"`
	Tickers       []string `config:"RSI_PULLBACK_TICKERS,backend=env"`
	BuyPct        float64  `config:"RSI_PULLBACK_BUY_PCT,backend=env"`
	TradeEnabled  bool     `config:"RSI_PULLBACK_TRADE_ENABLED,backend=env"`
	NotifyEnabled bool     `config:"RSI_PULLBACK_NOTIFY_ENABLED,backend=env"`
	Schedule      string   `config:"RSI_PULLBACK_SCHEDULE,backend=env"`
}

// NewRSIPullbackConfig returns the config pre-seeded with safe defaults. confita
// overrides any field whose env var is set; unset fields keep these values.
// TradeEnabled defaults to false so a missing flag never places real orders.
//
// Schedule фиксирует три решения. Раз в полчаса — потому что рабочий таймфрейм
// стратегии 30 минут. Минута :01/:31, а не :00/:30 — запас на то, чтобы закрывшийся бар
// успел прийти из API с IsComplete=true. Все семь дней недели — позиция едет через
// выходные, и бэктест отрабатывает выходы на выходных барах MOEX; входы на выходных
// закрывает само ядро (tradingDay). Ночью 00:00-06:00 MSK торгов нет.
func NewRSIPullbackConfig() *RSIPullbackConfig {
	return &RSIPullbackConfig{
		Tickers:  []string{"UGLD", "T", "GAZP"},
		BuyPct:   5,
		Schedule: "1,31 6-23 * * *",
	}
}
