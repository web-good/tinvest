package config

import "strings"

// GapFadeConfig подключает gap_fade к живому раннеру счёта rsi_pullback гостем: счёт, токен и
// расписание общие (RSI_PULLBACK_*), здесь — только своё (docs/gap_fade/live.md). Ни одна
// переменная не required — ошибка загрузки конфига оборвала бы InitApp целиком.
type GapFadeConfig struct {
	Tickers       []string `config:"GAP_FADE_TICKERS,backend=env"`
	BuyPct        float64  `config:"GAP_FADE_BUY_PCT,backend=env"`
	TradeEnabled  bool     `config:"GAP_FADE_TRADE_ENABLED,backend=env"`
	NotifyEnabled bool     `config:"GAP_FADE_NOTIFY_ENABLED,backend=env"`
}

// NewGapFadeConfig: вселенная пуста — стратегия не подключается, пока её не включат в env;
// торговля выключена, чтобы отсутствующий флаг не выставил ордер.
func NewGapFadeConfig() *GapFadeConfig {
	return &GapFadeConfig{BuyPct: 5}
}

// Enabled — задана ли вселенная. GAP_FADE_TICKERS= в env приходит как [""].
func (c *GapFadeConfig) Enabled() bool {
	for _, t := range c.Tickers {
		if strings.TrimSpace(t) != "" {
			return true
		}
	}
	return false
}
