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

func TestDensityLiveEntryZoneTicksEnvironment(t *testing.T) {
	t.Setenv("DENSITY_LIVE_ENTRY_ZONE_TICKS", "18")
	c := NewDensityLiveConfig()
	if err := confita.NewLoader(env.NewBackend()).Load(context.Background(), c); err != nil {
		t.Fatal(err)
	}
	if c.EntryZoneTicks != 18 {
		t.Fatalf("entry zone ticks = %v", c.EntryZoneTicks)
	}
	for _, ticks := range []int64{-1, 1000001} {
		c.EntryZoneTicks = ticks
		if c.Validate() == nil {
			t.Fatalf("accepted entry zone ticks %d", ticks)
		}
	}
}

func TestDensityLiveFollowScreenerEnvironment(t *testing.T) {
	t.Setenv("DENSITY_LIVE_FOLLOW_SCREENER", "true")
	c := NewDensityLiveConfig()
	if err := confita.NewLoader(env.NewBackend()).Load(context.Background(), c); err != nil {
		t.Fatal(err)
	}
	if !c.FollowScreener {
		t.Fatal("follow screener flag not loaded")
	}
}

func TestDensitySimulationTopicEnvironment(t *testing.T) {
	t.Setenv("DENSITY_SIMULATION_TOPIC_ID", "42")
	c := NewDensityLiveConfig()
	if err := confita.NewLoader(env.NewBackend()).Load(context.Background(), c); err != nil {
		t.Fatal(err)
	}
	if c.SimulationTopicID != 42 {
		t.Fatalf("simulation topic = %d", c.SimulationTopicID)
	}
	c.SimulationTopicID = -1
	if c.Validate() == nil {
		t.Fatal("accepted a negative simulation topic")
	}
}
