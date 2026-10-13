package config

import "strings"

// DropBounceConfig — вселенные drop_bounce для будущего живого раннера, своя на каждый
// таймфрейм: литерал тикера калибруется под таймфрейм отдельно, и тикер торгуется только на
// тех, что прошли калибровку (скилл drop-bounce-calibrate записывает их сюда). Живого раннера
// пока нет — конфиг только хранит списки. Ни одна переменная не required: ошибка загрузки
// конфига оборвала бы InitApp целиком.
type DropBounceConfig struct {
	TickersMinutes15 []string `config:"DROP_BOUNCE_TICKERS_M15,backend=env"`
	TickersMinutes30 []string `config:"DROP_BOUNCE_TICKERS_M30,backend=env"`
	TickersHour1     []string `config:"DROP_BOUNCE_TICKERS_H1,backend=env"`
}

// NewDropBounceConfig: все вселенные пусты — стратегия ничего не торгует, пока их не заполнят.
func NewDropBounceConfig() *DropBounceConfig {
	return &DropBounceConfig{}
}

// Enabled — задана ли хотя бы одна вселенная. DROP_BOUNCE_TICKERS_M15= в env приходит как [""].
func (c *DropBounceConfig) Enabled() bool {
	for _, list := range [][]string{c.TickersMinutes15, c.TickersMinutes30, c.TickersHour1} {
		for _, t := range list {
			if strings.TrimSpace(t) != "" {
				return true
			}
		}
	}
	return false
}
