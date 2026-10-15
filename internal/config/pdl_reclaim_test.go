package config

import (
	"context"
	"slices"
	"testing"

	"github.com/heetch/confita"
	"github.com/heetch/confita/backend/env"
)

// Без переменной pdl_reclaim конфиг грузится, вселенная пуста.
func TestPDLReclaimConfigLoadsWithoutItsEnvVar(t *testing.T) {
	t.Setenv("PDL_RECLAIM_TICKERS_M5", "")

	cfg := NewPDLReclaimConfig()
	if err := confita.NewLoader(env.NewBackend()).Load(context.Background(), cfg); err != nil {
		t.Fatalf("Load без PDL_RECLAIM_TICKERS_M5 = %v, want nil", err)
	}
	if cfg.Enabled() {
		t.Fatalf("без переменной cfg = %+v, want выключено", cfg)
	}
}

func TestPDLReclaimConfigLoadsMinutes5(t *testing.T) {
	t.Setenv("PDL_RECLAIM_TICKERS_M5", "GAZP,SBER")

	cfg := NewPDLReclaimConfig()
	if err := confita.NewLoader(env.NewBackend()).Load(context.Background(), cfg); err != nil {
		t.Fatalf("Load = %v", err)
	}
	if !slices.Equal(cfg.TickersMinutes5, []string{"GAZP", "SBER"}) {
		t.Fatalf("cfg = %+v, want [GAZP SBER]", cfg)
	}
	if !cfg.Enabled() {
		t.Fatal("заданная вселенная не включила стратегию")
	}
}

func TestPDLReclaimEnabledIgnoresBlankTickers(t *testing.T) {
	c := NewPDLReclaimConfig()
	c.TickersMinutes5 = []string{"", " "}
	if c.Enabled() {
		t.Fatal("вселенная из пустых строк включила стратегию")
	}
}
