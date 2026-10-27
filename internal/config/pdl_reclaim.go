package config

import "strings"

// PDLReclaimConfig — вселенная pdl_reclaim для будущего живого раннера. Стратегия работает только
// на 15m; тикер попадает сюда, только если прошёл калибровку. Живого раннера пока нет — конфиг
// только хранит список. Переменная не required: ошибка загрузки конфига оборвала бы InitApp целиком.
type PDLReclaimConfig struct {
	TickersMinutes15 []string `config:"PDL_RECLAIM_TICKERS_M15,backend=env"`
}

// NewPDLReclaimConfig: вселенная пуста — стратегия ничего не торгует, пока её не заполнят.
func NewPDLReclaimConfig() *PDLReclaimConfig {
	return &PDLReclaimConfig{}
}

// Enabled — задан ли хоть один тикер. PDL_RECLAIM_TICKERS_M15= в env приходит как [""].
func (c *PDLReclaimConfig) Enabled() bool {
	for _, t := range c.TickersMinutes15 {
		if strings.TrimSpace(t) != "" {
			return true
		}
	}
	return false
}
