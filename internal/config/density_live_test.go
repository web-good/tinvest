package config

import (
	"context"
	"math"
	"testing"

	"github.com/heetch/confita"
	"github.com/heetch/confita/backend/env"
)

func TestDensityLiveDefaultsAreDisabledAndMinimal(t *testing.T) {
	c := NewDensityLiveConfig()
	if c.Enabled || c.TradeEnabled || len(c.Tickers) != 2 || c.EntryTicks != 1 || c.StopTicks != 2 || c.RewardRisk != 2 || c.StateDir != "data/state/density" {
		t.Fatal(c)
	}
	if err := c.Validate(); err != nil {
		t.Fatal(err)
	}
}

func TestMalformedDensityLiveDoesNotBreakMainConfigLoad(t *testing.T) {
	t.Setenv("DENSITY_LIVE_RR", "not-a-number")
	cfg := &Config{AppName: "test", DensityLive: NewDensityLiveConfig()}
	if err := confita.NewLoader(env.NewBackend()).Load(context.Background(), cfg); err != nil {
		t.Fatalf("optional density broke mandatory config: %v", err)
	}
}
func TestDensityLiveRejectsInvalidLimits(t *testing.T) {
	for _, edit := range []func(*DensityLiveConfig){func(c *DensityLiveConfig) { c.MinRemaining = math.NaN() }, func(c *DensityLiveConfig) { c.Tickers = nil }, func(c *DensityLiveConfig) { c.EntryTicks = 0 }, func(c *DensityLiveConfig) { c.StateDir = "" }} {
		c := NewDensityLiveConfig()
		edit(c)
		if err := c.Validate(); err == nil {
			t.Fatal("invalid density live accepted")
		}
	}
}

func TestDensityLiveEntryPercentageEnvironment(t *testing.T) {
	t.Setenv("DENSITY_LIVE_ENTRY_PCT", "0.3")
	c := NewDensityLiveConfig()
	if err := confita.NewLoader(env.NewBackend()).Load(context.Background(), c); err != nil {
		t.Fatal(err)
	}
	if c.EntryPct != .3 {
		t.Fatalf("entry percentage = %v", c.EntryPct)
	}
	for _, pct := range []float64{-1, math.NaN(), math.Inf(1), 101} {
		c.EntryPct = pct
		if c.Validate() == nil {
			t.Fatalf("accepted entry percentage %v", pct)
		}
	}
}
