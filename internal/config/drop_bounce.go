package config

import "strings"

// DropBounceConfig — вселенная drop_bounce для будущего живого раннера. Стратегия работает только
// на 30m (сигнал на 1h, подтверждение MACD на 30m); тикер попадает сюда, только если прошёл калибровку (скилл drop-bounce-calibrate
// записывает его). Живого раннера пока нет — конфиг только хранит список. Переменная не
// required: ошибка загрузки конфига оборвала бы InitApp целиком.
type DropBounceConfig struct {
	TickersMinutes30 []string `config:"DROP_BOUNCE_TICKERS_M30,backend=env"`
}

// NewDropBounceConfig: вселенная пуста — стратегия ничего не торгует, пока её не заполнят.
func NewDropBounceConfig() *DropBounceConfig {
	return &DropBounceConfig{}
}

// Enabled — задан ли хоть один тикер. DROP_BOUNCE_TICKERS_M30= в env приходит как [""].
func (c *DropBounceConfig) Enabled() bool {
	for _, t := range c.TickersMinutes30 {
		if strings.TrimSpace(t) != "" {
			return true
		}
	}
	return false
}
