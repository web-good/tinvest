package live

import (
	"testing"
	"time"

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

func TestAccountEntrySchedule(t *testing.T) {
	msk, err := time.LoadLocation("Europe/Moscow")
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name, strategy string
		at             time.Time
		allowed        bool
	}{
		{"pullback before", "rsi_pullback", time.Date(2026, 6, 5, 13, 59, 59, 0, msk), true},
		{"pullback cutoff", "rsi_pullback", time.Date(2026, 6, 5, 14, 0, 0, 0, msk), false},
		{"zone cutoff UTC", "rsi_zone", time.Date(2026, 6, 5, 11, 0, 0, 0, time.UTC), false},
		{"zone saturday", "rsi_zone", time.Date(2026, 6, 6, 12, 0, 0, 0, msk), false},
		{"pullback sunday", "rsi_pullback", time.Date(2026, 6, 7, 12, 0, 0, 0, msk), false},
		{"zone monday", "rsi_zone", time.Date(2026, 6, 8, 14, 0, 0, 0, msk), true},
		{"other strategy", "gap_fade", time.Date(2026, 6, 6, 12, 0, 0, 0, msk), true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := accountEntryAllowed(tc.strategy, tc.at); got != tc.allowed {
				t.Errorf("got %v, want %v", got, tc.allowed)
			}
		})
	}
}
