package config

// RSIPullbackConfig configures the live rsi_pullback runner. Trade and notify are
// independent: both off = dry-run to log only.
//
// AccountID и Token НЕ помечены `required` намеренно. confita трактует required как
// «непустое значение» и возвращает ошибку загрузки, а она обрывает InitApp — то есть
// приложение не поднимается целиком, вместе с воркерами живого reversion, ведущими
// реальные позиции. Пока у стратегии нет своего счёта, её раннер должен просто не
// запускаться (см. Ready), а не ронять всё остальное.
type RSIPullbackConfig struct {
	AccountID     string   `config:"RSI_PULLBACK_ACCOUNT_ID,backend=env"`
	Token         string   `config:"RSI_PULLBACK_TOKEN,backend=env"`
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
//
// Вселенная по умолчанию — UGLD, T и GAZP. Walk-forward прошёл только UGLD
// (docs/rsi_pullback/live.md §10, риск 3); T и GAZP торгуются как принятый владельцем риск,
// а не по недосмотру.
func NewRSIPullbackConfig() *RSIPullbackConfig {
	return &RSIPullbackConfig{
		Tickers:  []string{"UGLD", "T", "GAZP"},
		BuyPct:   5,
		Schedule: "1,31 6-23 * * *",
	}
}

// Ready reports whether the runner has its own account to trade on. Without both the
// account id and the token there is nothing to run: the portfolio call would target no
// account and the gRPC client would carry an empty token, turning every pass into a stream
// of auth failures. The caller skips the worker instead — the rest of the application,
// including the live reversion workers, must keep running.
func (c *RSIPullbackConfig) Ready() bool {
	return c.AccountID != "" && c.Token != ""
}
