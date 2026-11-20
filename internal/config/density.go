package config

import (
	"errors"
	"math"
	"time"
)

// DensityConfig controls the order book density screener.
type DensityConfig struct {
	AlertM5VolumeMultiplier float64       `config:"DENSITY_ALERT_M5_VOLUME_MULTIPLIER,backend=env"`
	TurnoverFraction        float64       `config:"DENSITY_TURNOVER_FRACTION,backend=env"`
	SupportEnabled          bool          `config:"DENSITY_SUPPORT_ENABLED,backend=env"`
	SupportLookbackDays     int           `config:"DENSITY_SUPPORT_LOOKBACK_DAYS,backend=env"`
	SupportPivotBars        int           `config:"DENSITY_SUPPORT_PIVOT_BARS,backend=env"`
	SupportMinTouches       int           `config:"DENSITY_SUPPORT_MIN_TOUCHES,backend=env"`
	SupportMinSeparation    time.Duration `config:"DENSITY_SUPPORT_MIN_SEPARATION,backend=env"`
	SupportBounceFraction   float64       `config:"DENSITY_SUPPORT_BOUNCE_FRACTION,backend=env"`
	SupportBounceBars       int           `config:"DENSITY_SUPPORT_BOUNCE_BARS,backend=env"`
	SupportTolerance        float64       `config:"DENSITY_SUPPORT_TOLERANCE,backend=env"`
	Enabled                 bool          `config:"DENSITY_ENABLED,backend=env"`
	ChatID                  int64         `config:"DENSITY_CHAT_ID,backend=env"`
	TopicID                 int           `config:"DENSITY_TOPIC_ID,backend=env"`
	MaxBooks                int           `config:"DENSITY_MAX_BOOKS,backend=env"`
	MaxTrades               int           `config:"DENSITY_MAX_TRADES,backend=env"`
	BookDepth               int           `config:"DENSITY_BOOK_DEPTH,backend=env"`
	MinRubles               float64       `config:"DENSITY_MIN_RUBLES,backend=env"`
	MinMultiple             float64       `config:"DENSITY_MIN_MULTIPLE,backend=env"`
	MinNeighbors            int           `config:"DENSITY_MIN_NEIGHBORS,backend=env"`
	MaxNeighbors            int           `config:"DENSITY_MAX_NEIGHBORS,backend=env"`
	Persistence             time.Duration `config:"DENSITY_PERSISTENCE,backend=env"`
	Freshness               time.Duration `config:"DENSITY_FRESHNESS,backend=env"`
	MinTurnover             float64       `config:"DENSITY_MIN_TURNOVER,backend=env"`
	MaxLotPrice             float64       `config:"DENSITY_MAX_LOT_PRICE,backend=env"`
	HistoryInterval         time.Duration `config:"DENSITY_HISTORY_INTERVAL,backend=env"`
}

func NewDensityConfig() *DensityConfig {
	return &DensityConfig{AlertM5VolumeMultiplier: .5, MaxBooks: 80, MaxTrades: 10, BookDepth: 50, MinRubles: 5_000_000, MinMultiple: 5, MinNeighbors: 5, MaxNeighbors: 10, Persistence: time.Minute, Freshness: 3 * time.Second, MinTurnover: 100_000_000, MaxLotPrice: 30_000, HistoryInterval: time.Second, SupportLookbackDays: 60, SupportPivotBars: 2, SupportMinTouches: 2, SupportMinSeparation: 6 * time.Hour, SupportBounceFraction: .005, SupportBounceBars: 6, SupportTolerance: .002}
}
func (c *DensityConfig) Validate() error {
	if c.Enabled && c.ChatID == 0 {
		return errors.New("DENSITY_CHAT_ID is required when DENSITY_ENABLED is true")
	}
	positive := func(v float64) bool { return v > 0 && !math.IsInf(v, 0) && !math.IsNaN(v) }
	if !positive(c.AlertM5VolumeMultiplier) {
		return errors.New("DENSITY_ALERT_M5_VOLUME_MULTIPLIER must be finite and positive")
	}
	if math.IsNaN(c.TurnoverFraction) || math.IsInf(c.TurnoverFraction, 0) || c.TurnoverFraction < 0 || c.TurnoverFraction > 1 {
		return errors.New("DENSITY_TURNOVER_FRACTION must be between 0 and 1")
	}
	if c.SupportEnabled && (c.SupportLookbackDays < 1 || c.SupportLookbackDays > 90 || c.SupportPivotBars < 1 || c.SupportPivotBars > 24 || c.SupportMinTouches < 2 || c.SupportMinSeparation < time.Hour || c.SupportBounceBars < c.SupportPivotBars || c.SupportBounceBars > 48 || !positive(c.SupportBounceFraction) || c.SupportBounceFraction > 1 || !positive(c.SupportTolerance) || c.SupportTolerance > .1) {
		return errors.New("invalid density support settings")
	}
	if c.TopicID < 0 || c.MaxBooks < 1 || c.MaxBooks > 80 || c.MaxTrades < 1 || c.MaxTrades > 10 || c.BookDepth < 1 || c.BookDepth > 50 || c.MinNeighbors < 1 || c.MaxNeighbors < c.MinNeighbors || c.MaxNeighbors >= c.BookDepth || c.Persistence <= 0 || c.Freshness <= 0 || c.HistoryInterval <= 0 || !positive(c.MinRubles) || !positive(c.MinMultiple) || !positive(c.MinTurnover) || !positive(c.MaxLotPrice) {
		return errors.New("invalid density screener limits")
	}
	return nil
}
