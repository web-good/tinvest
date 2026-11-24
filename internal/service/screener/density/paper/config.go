// Package paper simulates density trades using market data only. It has no broker order API.
package paper

import (
	"fmt"
	"math"
	"time"
)

type Config struct {
	ScreenerStrategy                                bool // Use screener filters and fixed percentage exits.
	FixedLimitEntry                                 bool
	CandleEntries                                   bool
	RestrictEntryHours                              bool  // Allow buys only at 07:00–12:00 and 15:00–18:00 Moscow time.
	BookExecution                                   bool  // Execute against visible books without a trade subscription.
	IgnoreCapital                                   bool  // Price-only simulation does not allocate a virtual balance.
	MaxEntryLots                                    int64 // Zero keeps allocation-based sizing; live simulation uses one lot.
	InitialCash, PositionRubles, Commission         float64
	MaxPositions                                    int
	EntryTicks, EntryZoneTicks, StopTicks           int64
	RewardRisk, MinRemaining                        float64
	Latency, Freshness, EntryTTL, MaxHold, Cooldown time.Duration
}

func DefaultConfig() Config {
	return Config{InitialCash: 100000, PositionRubles: 10000, Commission: .0005, MaxPositions: 3, EntryTicks: 1, StopTicks: 2, RewardRisk: 2, MinRemaining: .5, Latency: 250 * time.Millisecond, Freshness: 3 * time.Second, EntryTTL: 2 * time.Minute, MaxHold: 10 * time.Minute, Cooldown: 5 * time.Minute}
}
func (c Config) fixedLimitEntry() bool {
	return c.FixedLimitEntry || c.ScreenerStrategy
}

func (c Config) Validate() error {
	if c.EntryZoneTicks < 0 || c.EntryZoneTicks > 1000000 {
		return fmt.Errorf("invalid paper entry zone ticks")
	}
	if c.IgnoreCapital && c.MaxEntryLots < 1 {
		return fmt.Errorf("price-only simulation requires a fixed maximum entry size")
	}
	if c.MaxEntryLots < 0 {
		return fmt.Errorf("maximum entry lots cannot be negative")
	}
	positive := func(v float64) bool { return v > 0 && !math.IsNaN(v) && !math.IsInf(v, 0) }
	if !positive(c.InitialCash) || !positive(c.PositionRubles) || c.PositionRubles > c.InitialCash || c.MaxPositions < 1 || c.EntryTicks < 1 || c.StopTicks < 1 || c.EntryTicks > 1000000 || c.StopTicks > 1000000 || !positive(c.RewardRisk) || c.RewardRisk > 100 || !positive(c.MinRemaining) || c.MinRemaining > 1 || math.IsNaN(c.Commission) || math.IsInf(c.Commission, 0) || c.Commission < 0 || c.Commission >= 1 || c.Latency < 0 || c.Freshness <= 0 || c.EntryTTL <= c.Latency || (c.MaxHold != 0 && c.MaxHold <= c.Latency) || c.Cooldown < 0 {
		return fmt.Errorf("invalid paper configuration")
	}
	return nil
}
