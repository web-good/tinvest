package config

import (
	"context"
	"slices"
	"testing"

	"github.com/heetch/confita"
	"github.com/heetch/confita/backend/env"
)

// Без переменной drop_bounce конфиг грузится, вселенная пуста.
func TestDropBounceConfigLoadsWithoutItsEnvVar(t *testing.T) {
	t.Setenv("DROP_BOUNCE_TICKERS_M15", "")

	cfg := NewDropBounceConfig()
	if err := confita.NewLoader(env.NewBackend()).Load(context.Background(), cfg); err != nil {
		t.Fatalf("Load без DROP_BOUNCE_TICKERS_M15 = %v, want nil", err)
	}
	if cfg.Enabled() {
		t.Fatalf("без переменной cfg = %+v, want выключено", cfg)
	}
}

func TestDropBounceConfigLoadsMinutes15(t *testing.T) {
	t.Setenv("DROP_BOUNCE_TICKERS_M15", "GAZP,T")

	cfg := NewDropBounceConfig()
	if err := confita.NewLoader(env.NewBackend()).Load(context.Background(), cfg); err != nil {
		t.Fatalf("Load = %v", err)
	}
	if !slices.Equal(cfg.TickersMinutes15, []string{"GAZP", "T"}) {
		t.Fatalf("cfg = %+v, want H1 [GAZP T]", cfg)
	}
	if !cfg.Enabled() {
		t.Fatal("заданная вселенная не включила стратегию")
	}
}

func TestDropBounceEnabledIgnoresBlankTickers(t *testing.T) {
	c := NewDropBounceConfig()
	c.TickersMinutes15 = []string{"", " "}
	if c.Enabled() {
		t.Fatal("вселенная из пустых строк включила стратегию")
	}
}
