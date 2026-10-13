package config

import (
	"context"
	"slices"
	"testing"

	"github.com/heetch/confita"
	"github.com/heetch/confita/backend/env"
)

// Без переменных drop_bounce конфиг грузится, все вселенные пусты.
func TestDropBounceConfigLoadsWithoutItsEnvVars(t *testing.T) {
	t.Setenv("DROP_BOUNCE_TICKERS_M15", "")
	t.Setenv("DROP_BOUNCE_TICKERS_M30", "")
	t.Setenv("DROP_BOUNCE_TICKERS_H1", "")

	cfg := NewDropBounceConfig()
	if err := confita.NewLoader(env.NewBackend()).Load(context.Background(), cfg); err != nil {
		t.Fatalf("Load без DROP_BOUNCE_* = %v, want nil", err)
	}
	if cfg.Enabled() {
		t.Fatalf("без переменных cfg = %+v, want выключено", cfg)
	}
}

// Каждая переменная попадает в вселенную своего таймфрейма.
func TestDropBounceConfigLoadsPerTimeframe(t *testing.T) {
	t.Setenv("DROP_BOUNCE_TICKERS_M15", "GAZP,T")
	t.Setenv("DROP_BOUNCE_TICKERS_M30", "")
	t.Setenv("DROP_BOUNCE_TICKERS_H1", "PLZL")

	cfg := NewDropBounceConfig()
	if err := confita.NewLoader(env.NewBackend()).Load(context.Background(), cfg); err != nil {
		t.Fatalf("Load = %v", err)
	}
	if !slices.Equal(cfg.TickersMinutes15, []string{"GAZP", "T"}) || !slices.Equal(cfg.TickersHour1, []string{"PLZL"}) {
		t.Fatalf("cfg = %+v, want M15 [GAZP T], H1 [PLZL]", cfg)
	}
	if !cfg.Enabled() {
		t.Fatal("заданные вселенные не включили стратегию")
	}
}

func TestDropBounceEnabledIgnoresBlankTickers(t *testing.T) {
	c := NewDropBounceConfig()
	c.TickersMinutes30 = []string{"", " "}
	if c.Enabled() {
		t.Fatal("вселенная из пустых строк включила стратегию")
	}
}
