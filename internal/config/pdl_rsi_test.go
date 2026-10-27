package config

import (
	"context"
	"testing"

	"github.com/heetch/confita"
	"github.com/heetch/confita/backend/env"
)

// Без переменных pdl_rsi конфиг грузится, стратегия и торговля выключены.
func TestPDLRSIConfigLoadsWithoutItsEnvVars(t *testing.T) {
	t.Setenv("PDL_RSI_TICKERS", "")
	t.Setenv("PDL_RSI_BUY_PCT", "")
	t.Setenv("PDL_RSI_TRADE_ENABLED", "")
	t.Setenv("PDL_RSI_NOTIFY_ENABLED", "")

	cfg := NewPDLRSIConfig()
	if err := confita.NewLoader(env.NewBackend()).Load(context.Background(), cfg); err != nil {
		t.Fatalf("Load без PDL_RSI_* = %v, want nil", err)
	}
	if cfg.Enabled() || cfg.TradeEnabled {
		t.Fatalf("без переменных cfg = %+v, want стратегия и торговля выключены", cfg)
	}
}

func TestPDLRSIConfigDefaultsOff(t *testing.T) {
	c := NewPDLRSIConfig()
	if c.Enabled() || c.TradeEnabled || c.NotifyEnabled || c.BuyPct != 5 {
		t.Fatalf("дефолты = %+v, want выключено, BuyPct 5", c)
	}
	// PDL_RSI_TICKERS= в env приходит как [""] — это не вселенная
	c.Tickers = []string{"", " "}
	if c.Enabled() {
		t.Fatal("вселенная из пустых строк включила стратегию")
	}
	c.Tickers = []string{"SVCB"}
	if !c.Enabled() {
		t.Fatal("вселенная SVCB не включила стратегию")
	}
}
