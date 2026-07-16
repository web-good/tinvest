package config

// NewsConfig — настройки новостного дайджеста.
type NewsConfig struct {
	// FeedURL — RSS-лента новостей рынка; дефолт — «Лента всех новостей
	// акций» smart-lab (агрегатор Интерфакс/Reuters/эмитенты).
	FeedURL string `config:"NEWS_FEED_URL"`
}

func NewNewsConfig() *NewsConfig {
	return &NewsConfig{FeedURL: "https://smart-lab.ru/news/rss/"}
}
