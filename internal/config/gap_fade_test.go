package config

import (
	"context"
	"testing"

	"github.com/heetch/confita"
	"github.com/heetch/confita/backend/env"
)

// Без переменных gap_fade конфиг грузится, стратегия и торговля выключены.
func TestGapFadeConfigLoadsWithoutItsEnvVars(t *testing.T) {
	t.Setenv("GAP_FADE_TICKERS", "")
	t.Setenv("GAP_FADE_BUY_PCT", "")
	t.Setenv("GAP_FADE_TRADE_ENABLED", "")
	t.Setenv("GAP_FADE_NOTIFY_ENABLED", "")

	cfg := NewGapFadeConfig()
	if err := confita.NewLoader(env.NewBackend()).Load(context.Background(), cfg); err != nil {
		t.Fatalf("Load без GAP_FADE_* = %v, want nil", err)
	}
	if cfg.Enabled() || cfg.TradeEnabled {
		t.Fatalf("без переменных cfg = %+v, want стратегия и торговля выключены", cfg)
	}
}

func TestNewGapFadeConfig_Defaults(t *testing.T) {
	c := NewGapFadeConfig()
	if c.Enabled() || c.TradeEnabled || c.NotifyEnabled || c.BuyPct != 5 {
		t.Fatalf("дефолты = %+v, want выключено, BuyPct 5", c)
	}
}

func TestGapFadeEnabledIgnoresBlankTickers(t *testing.T) {
	c := NewGapFadeConfig()
	c.Tickers = []string{"", " "}
	if c.Enabled() {
		t.Fatal("вселенная из пустых строк включила стратегию")
	}
	c.Tickers = []string{"SBER"}
	if !c.Enabled() {
		t.Fatal("вселенная SBER не включила стратегию")
	}
}
