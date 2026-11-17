package config

import (
	"fmt"
	"math"
	"strings"
	"time"
)

// DensityLiveConfig joins the existing RSI account; it has no separate credentials.
type DensityLiveConfig struct {
	FollowScreener bool          `config:"DENSITY_LIVE_FOLLOW_SCREENER,backend=env"`
	Enabled        bool          `config:"DENSITY_LIVE_ENABLED,backend=env"`
	TradeEnabled   bool          `config:"DENSITY_LIVE_TRADE_ENABLED,backend=env"`
	Tickers        []string      `config:"DENSITY_LIVE_TICKERS,backend=env"`
	StateDir       string        `config:"DENSITY_LIVE_STATE_DIR,backend=env"`
	EntryTicks     int64         `config:"DENSITY_LIVE_ENTRY_TICKS,backend=env"`
	EntryPct       float64       `config:"DENSITY_LIVE_ENTRY_PCT,backend=env"`
	StopTicks      int64         `config:"DENSITY_LIVE_STOP_TICKS,backend=env"`
	RewardRisk     float64       `config:"DENSITY_LIVE_RR,backend=env"`
	MinRemaining   float64       `config:"DENSITY_LIVE_MIN_REMAINING,backend=env"`
	EntryTTL       time.Duration `config:"DENSITY_LIVE_ENTRY_TTL,backend=env"`
	MaxHold        time.Duration `config:"DENSITY_LIVE_MAX_HOLD,backend=env"`
	Cooldown       time.Duration `config:"DENSITY_LIVE_COOLDOWN,backend=env"`
}

func NewDensityLiveConfig() *DensityLiveConfig {
	return &DensityLiveConfig{Tickers: []string{"T", "SBER"}, StateDir: "data/state/density", EntryTicks: 1, StopTicks: 2, RewardRisk: 2, MinRemaining: .5, EntryTTL: 2 * time.Minute, MaxHold: 10 * time.Minute, Cooldown: 5 * time.Minute}
}
func (c *DensityLiveConfig) Validate() error {
	if c == nil {
		return fmt.Errorf("density live configuration missing")
	}
	if math.IsNaN(c.EntryPct) || math.IsInf(c.EntryPct, 0) || c.EntryPct < 0 || c.EntryPct > 100 {
		return fmt.Errorf("invalid DENSITY_LIVE_ENTRY_PCT: must be between 0 and 100")
	}
	finite := func(v float64) bool { return !math.IsNaN(v) && !math.IsInf(v, 0) }

	if len(c.Tickers) < 1 || len(c.Tickers) > 10 || strings.TrimSpace(c.StateDir) == "" || c.EntryTicks < 1 || c.EntryTicks > 1000000 || c.StopTicks < 1 || c.StopTicks > 1000000 || !finite(c.RewardRisk) || c.RewardRisk <= 0 || c.RewardRisk > 100 || !finite(c.MinRemaining) || c.MinRemaining <= 0 || c.MinRemaining > 1 || c.EntryTTL <= 0 || c.MaxHold <= 0 || c.Cooldown < 0 {
		return fmt.Errorf("invalid density live limits")
	}
	seen := map[string]bool{}
	for _, ticker := range c.Tickers {
		ticker = strings.ToUpper(strings.TrimSpace(ticker))
		if ticker == "" || seen[ticker] {
			return fmt.Errorf("density live tickers must be nonempty and unique")
		}
		seen[ticker] = true
	}
	return nil
}
