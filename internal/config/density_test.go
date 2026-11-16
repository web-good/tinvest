package config

import (
	"context"
	"testing"
	"time"

	"github.com/heetch/confita"
	"github.com/heetch/confita/backend/env"
)

func TestDensityConfigDestination(t *testing.T) {
	cfg := NewDensityConfig()
	if cfg.Enabled {
		t.Fatal("density must be disabled by default")
	}
	if err := cfg.Validate(); err != nil {
		t.Fatalf("disabled: %v", err)
	}
	cfg.Enabled = true
	if err := cfg.Validate(); err == nil {
		t.Fatal("enabled without chat ID must fail")
	}
	cfg.ChatID = -100123
	if err := cfg.Validate(); err != nil {
		t.Fatalf("enabled with chat ID: %v", err)
	}
}

func TestDensityConfigLimits(t *testing.T) {
	cfg := NewDensityConfig()
	if cfg.MaxBooks != 80 || cfg.MaxTrades != 10 || cfg.BookDepth != 50 || cfg.MinRubles != 5_000_000 || cfg.MaxDistance != .01 {
		t.Fatalf("unexpected defaults: %+v", cfg)
	}
	for _, change := range []func(*DensityConfig){
		func(c *DensityConfig) { c.MaxBooks = 81 }, func(c *DensityConfig) { c.MaxTrades = 0 },
		func(c *DensityConfig) { c.MinNeighbors = 0 }, func(c *DensityConfig) { c.Freshness = 0 },
		func(c *DensityConfig) { c.MinMultiple = -1 }, func(c *DensityConfig) { c.MaxDistance = 2 },
	} {
		c := NewDensityConfig()
		change(c)
		if c.Validate() == nil {
			t.Fatalf("invalid config accepted: %+v", c)
		}
	}
}

func TestDensityConfigLoadsEnvironment(t *testing.T) {
	t.Setenv("DENSITY_ENABLED", "true")
	t.Setenv("DENSITY_CHAT_ID", "-100123")
	t.Setenv("DENSITY_FRESHNESS", "4s")
	cfg := NewDensityConfig()
	if err := confita.NewLoader(env.NewBackend()).Load(context.Background(), cfg); err != nil {
		t.Fatal(err)
	}
	if !cfg.Enabled || cfg.ChatID != -100123 || cfg.Freshness != 4*time.Second {
		t.Fatalf("config %+v", cfg)
	}
}

func TestDensityHistoryFilterEnvironmentAndValidation(t *testing.T) {
	t.Setenv("DENSITY_TURNOVER_FRACTION", "0.005")
	t.Setenv("DENSITY_SUPPORT_ENABLED", "true")
	t.Setenv("DENSITY_SUPPORT_LOOKBACK_DAYS", "60")
	t.Setenv("DENSITY_SUPPORT_TOLERANCE", "0.002")
	c := NewDensityConfig()
	if err := confita.NewLoader(env.NewBackend()).Load(context.Background(), c); err != nil {
		t.Fatal(err)
	}
	if c.TurnoverFraction != .005 || !c.SupportEnabled || c.SupportLookbackDays != 60 || c.SupportTolerance != .002 {
		t.Fatalf("history config %+v", c)
	}
	if err := c.Validate(); err != nil {
		t.Fatal(err)
	}
	for _, change := range []func(*DensityConfig){
		func(c *DensityConfig) { c.TurnoverFraction = -.1 },
		func(c *DensityConfig) { c.SupportLookbackDays = 91 },
		func(c *DensityConfig) { c.SupportTolerance = 0 },
		func(c *DensityConfig) { c.SupportMinTouches = 1 },
		func(c *DensityConfig) { c.SupportBounceBars = 1 },
	} {
		copy := *c
		change(&copy)
		if copy.Validate() == nil {
			t.Fatal("invalid history config accepted")
		}
	}
}
