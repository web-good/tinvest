package config

import "strings"

// PDLRSIConfig подключает pdl_rsi к живому раннеру счёта rsi_pullback: счёт, токен и
// расписание у стратегий общие (RSI_PULLBACK_*), здесь — только своё (docs/pdl_rsi/live.md).
// Ни одна переменная не required — по той же причине, что у RSIPullbackConfig: ошибка
// загрузки конфига оборвала бы InitApp целиком.
type PDLRSIConfig struct {
	Tickers       []string `config:"PDL_RSI_TICKERS,backend=env"`
	BuyPct        float64  `config:"PDL_RSI_BUY_PCT,backend=env"`
	TradeEnabled  bool     `config:"PDL_RSI_TRADE_ENABLED,backend=env"`
	NotifyEnabled bool     `config:"PDL_RSI_NOTIFY_ENABLED,backend=env"`
}

// NewPDLRSIConfig: вселенная по умолчанию пуста — стратегия не подключается, пока её не
// включат в env; торговля выключена, чтобы отсутствующий флаг не выставил ордер.
func NewPDLRSIConfig() *PDLRSIConfig {
	return &PDLRSIConfig{BuyPct: 5}
}

// Enabled — задана ли вселенная. Пустые строки не считаются: PDL_RSI_TICKERS= в env
// приходит как [""].
func (c *PDLRSIConfig) Enabled() bool {
	for _, t := range c.Tickers {
		if strings.TrimSpace(t) != "" {
			return true
		}
	}
	return false
}
