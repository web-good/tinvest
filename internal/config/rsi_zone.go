package config

import "strings"

// RSIZoneConfig подключает rsi_zone к живому раннеру счёта rsi_pullback: счёт, токен и
// расписание у стратегий общие (RSI_PULLBACK_*), здесь — только своё (docs/rsi_zone/live.md).
// Ни одна переменная не required — по той же причине, что у RSIPullbackConfig: ошибка
// загрузки конфига оборвала бы InitApp целиком.
type RSIZoneConfig struct {
	Tickers       []string `config:"RSI_ZONE_TICKERS,backend=env"`
	BuyPct        float64  `config:"RSI_ZONE_BUY_PCT,backend=env"`
	TradeEnabled  bool     `config:"RSI_ZONE_TRADE_ENABLED,backend=env"`
	NotifyEnabled bool     `config:"RSI_ZONE_NOTIFY_ENABLED,backend=env"`
}

// NewRSIZoneConfig: вселенная по умолчанию пуста — стратегия не подключается, пока её не
// включат в env; торговля выключена, чтобы отсутствующий флаг не выставил ордер.
func NewRSIZoneConfig() *RSIZoneConfig {
	return &RSIZoneConfig{BuyPct: 5}
}

// Enabled — задана ли вселенная. Пустые строки не считаются: RSI_ZONE_TICKERS= в env
// приходит как [""].
func (c *RSIZoneConfig) Enabled() bool {
	for _, t := range c.Tickers {
		if strings.TrimSpace(t) != "" {
			return true
		}
	}
	return false
}
