package live

import (
	"testing"

	"tinvest/internal/config"
)

// TestEveryLiveTickerBlocksMiddayBuys пинит решение владельца 2026-10-14: вся боевая вселенная
// rsi_pullback не покупает с 11:30 до 16:00 MSK (окно по закрытию сигнального бара, все будни).
// Тикер, добавленный без окна, торговал бы по правилам, которые замер не проверял.
func TestEveryLiveTickerBlocksMiddayBuys(t *testing.T) {
	for _, ticker := range config.NewRSIPullbackConfig().Tickers {
		p, ok := realParams[ticker]
		if !ok {
			continue // отсутствие в реестре ловят тесты registry_test.go
		}
		if p.EntryBlockFrom != 1130 || p.EntryBlockTo != 1600 || p.EntryBlockMonThu != 0 {
			t.Errorf("%s: окно запрета покупок %d–%d (MonThu=%d), want 1130–1600 во все будни",
				ticker, p.EntryBlockFrom, p.EntryBlockTo, p.EntryBlockMonThu)
		}
	}
}
