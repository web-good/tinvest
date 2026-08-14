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
// Вселенная по умолчанию — UGLD, T, GAZP, DOMRF, FESH и WUSH. Walk-forward прошёл только UGLD
// (docs/rsi_pullback/live.md §10, риск 3); остальные четыре торгуются как принятый владельцем
// риск, а не по недосмотру. У DOMRF риск самый крупный: 8.6 месяца истории, вся она — один
// пост-IPO аптренд, out-of-sample по существу один фолд на 8 сделках (подробности и замеры —
// в доке пакета strategy/domrf). FESH заведён 2026-08-13 (риск 7): истории у него как раз
// достаточно, но штатный протокол его НЕ подтверждает — тематические сетки на четырёх фолдах
// дают pooled OOS PF 1.029 по входу и 0.953 по тренду, а хорошие 2.366 принадлежат
// конфигурации, выбранной человеком, видевшим всю историю (док пакета strategy/fesh). WUSH
// заведён 2026-08-14 на тех же основаниях и с тем же типом риска: истории 36 месяцев хватает,
// но объявленную заранее планку тикер не взял ДВАЖДЫ (тема trend дала 1.317, после расширения
// оси входа 1.377 при пороге 1.5), а числа принятого литерала — pooled PF 2.030 на 144 сделках
// с прибыльными четырьмя фолдами — принадлежат конфигурации, подобранной вручную по всей
// истории, то есть не out-of-sample (док пакета strategy/wush). Отдельная особенность WUSH:
// оба окна протокола у него ПАДАЮЩИЕ (train −58.9%, holdout −48.0%), поэтому лонговый результат
// здесь не завышен режимом — но и просадка литерала выше остальных, 14.85% за 36 месяцев.
func NewRSIPullbackConfig() *RSIPullbackConfig {
	return &RSIPullbackConfig{
		Tickers:  []string{"UGLD", "T", "GAZP", "DOMRF", "FESH", "WUSH"},
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
