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
// Вселенная по умолчанию — UGLD, T, GAZP, DOMRF, FESH, WUSH, LENT, RENI, NVTK и LSNGP. С
// 2026-08-17 это ВЕСЬ реестр: решением владельца в неё заведены три последних тикера, у которых
// литерал был, а решения не было. Walk-forward прошёл только
// UGLD (docs/rsi_pullback/live.md §10, риск 3); остальные торгуются как принятый владельцем
// риск, а не по недосмотру. У DOMRF риск самый крупный: 8.6 месяца истории, вся она — один
// пост-IPO аптренд, out-of-sample по существу один фолд на 8 сделках (подробности и замеры —
// в доке пакета strategy/domrf). FESH заведён 2026-08-13 (риск 7): истории у него как раз
// достаточно, но штатный протокол его НЕ подтверждает — тематические сетки на четырёх фолдах
// дают pooled OOS PF 1.029 по входу и 0.953 по тренду, а хорошие 2.366 принадлежат
// конфигурации, выбранной человеком, видевшим всю историю (док пакета strategy/fesh). WUSH
// LENT заведён 2026-08-14 последним, и у него риск ДРУГОЙ ПРИРОДЫ, чем у остальных: помимо
// непройденной планки (entry 1.355, trend 1.777 при неустойчивой ведущей оси — четыре разных
// значения RSILower на четырёх фолдах) у него самая тонкая ликвидность вселенной — 67 млн ₽
// среднего дневного оборота при медиане будних дней всего 38 млн, тогда как у WUSH медиана
// 253 млн. Это ограничение исполнения, а не статистики: половина торговых дней тоньше гейта
// отбора скринера в 50 млн, и BUY_PCT=5 на крупном счёте упрётся в стакан здесь раньше, чем на
// любом другом тикере. Числа литерала — pooled PF 2.487 на 120 сделках при четырёх фолдах выше
// планки — принадлежат конфигурации, собранной по лидербордам человеком, видевшим всю историю
// (док пакета strategy/lent). WUSH
// заведён 2026-08-14 на тех же основаниях и с тем же типом риска: истории 36 месяцев хватает,
// но объявленную заранее планку тикер не взял ДВАЖДЫ (тема trend дала 1.317, после расширения
// оси входа 1.377 при пороге 1.5), а числа принятого литерала — pooled PF 2.030 на 144 сделках
// с прибыльными четырьмя фолдами — принадлежат конфигурации, подобранной вручную по всей
// истории, то есть не out-of-sample (док пакета strategy/wush). Отдельная особенность WUSH:
// оба окна протокола у него ПАДАЮЩИЕ (train −58.9%, holdout −48.0%), поэтому лонговый результат
// здесь не завышен режимом — но и просадка литерала выше остальных, 14.85% за 36 месяцев.
//
// RENI, NVTK и LSNGP заведены 2026-08-17 одним решением владельца — до этого дня каждый из них
// имел откалиброванный литерал и не имел решения о вселенной. Тип риска у всех трёх тот же, что у
// FESH, WUSH и LENT: истории на штатный протокол хватает, но протокол тикер не подтверждает, а
// числа принятой точки собраны человеком, видевшим тестовые окна. Различия, которые надо знать
// (полный разбор — live.md §10, риски 11, 12 и 13, и доки пакетов):
//   - RENI (риск 11) из троих ближе всех к планке: шесть тем из девяти выше 1.5, обязательная
//     пара взята по PF (entry 1.955, trend 1.920) и провалена только по устойчивости ведущей оси.
//     Перекалибровка 2026-08-16 закрыла обе прежние оговорки тикера. Оборот 91 млн ₽/день —
//     второй снизу в каталоге после LENT.
//   - NVTK (риск 12) — противоположный край: слепые темы дали ХУДШИЙ результат каталога (одна из
//     девяти выше 1.5, entry 1.218, trend 1.044), и весь разрыв с принятой точкой (pooled 2.823
//     на 93 сделках) — работа человека. Взамен это самый ликвидный тикер вселенной, то есть
//     единственный из троих, где риск исполнения не добавляется к статистическому.
//   - LSNGP (риск 13) — единственный, у кого ВСЕ девять тем выше 1.5, и одновременно единственный,
//     у кого в истории НЕТ падающего окна: оба окна протокола растут (+46.7% / +11.2%), так что
//     против падающего рынка конфигурация не проверялась вовсе. Ликвидность вторая снизу после
//     LENT (медиана 41 млн ₽), а RSIUpper 55 делает сделку внутридневной — медиана удержания
//     4 бара против 12 у LENT.
func NewRSIPullbackConfig() *RSIPullbackConfig {
	return &RSIPullbackConfig{
		Tickers:  []string{"UGLD", "T", "GAZP", "DOMRF", "FESH", "WUSH", "LENT", "RENI", "NVTK", "LSNGP"},
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
