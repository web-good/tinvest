// Package live executes density signals using confirmed broker orders.
package live

import (
	"fmt"
	"math"
	"time"
	"tinvest/internal/service/screener/density"
)

const Nano int64 = 1000000000

type Config struct {
	EntryTicks, StopTicks                                int64
	EntryPct                                             float64
	RewardRisk, MinRemaining                             float64
	Freshness, EntryTTL, MaxHold, Cooldown, PollInterval time.Duration
}

func DefaultConfig() Config {
	return Config{EntryTicks: 1, StopTicks: 2, RewardRisk: 2, MinRemaining: .5, Freshness: 3 * time.Second, EntryTTL: 2 * time.Minute, MaxHold: 10 * time.Minute, Cooldown: 5 * time.Minute, PollInterval: 2 * time.Second}
}

func (c Config) Validate() error {
	if math.IsNaN(c.EntryPct) || math.IsInf(c.EntryPct, 0) || c.EntryPct < 0 || c.EntryPct > 100 {
		return fmt.Errorf("invalid live entry percentage")
	}
	if c.EntryTicks < 1 || c.EntryTicks > 1000000 || c.StopTicks < 1 || c.StopTicks > 1000000 || math.IsNaN(c.RewardRisk) || math.IsInf(c.RewardRisk, 0) || c.RewardRisk <= 0 || c.RewardRisk > 100 || math.IsNaN(c.MinRemaining) || c.MinRemaining <= 0 || c.MinRemaining > 1 || c.Freshness <= 0 || c.EntryTTL <= 0 || c.MaxHold <= 0 || c.Cooldown < 0 || c.PollInterval < time.Second {
		return fmt.Errorf("invalid live configuration")
	}
	return nil
}

func Price(p density.Price) int64 {
	if p.Units < 0 || p.Units > 100000000 || p.Nano < 0 || p.Nano >= int32(Nano) {
		return 0
	}
	return p.Units*Nano + int64(p.Nano)
}

func Plan(c Config, i density.Instrument, s density.Signal, b density.Book, now time.Time) (*Episode, error) {
	if !s.ProfileUntil.IsZero() && !now.Before(s.ProfileUntil) {
		return nil, fmt.Errorf("density history profile expired")
	}
	step, level := Price(i.PriceStep), Price(s.Price)
	if err := c.Validate(); err != nil {
		return nil, err
	}
	if i.ID == "" || i.Ticker == "" || i.Lot <= 0 || step <= 0 || step > 1000000*Nano || level <= 0 || level%step != 0 || i.ID != s.InstrumentID || !validBook(b, now, c.Freshness) || b.InstrumentID != i.ID || b.Lot != i.Lot {
		return nil, fmt.Errorf("invalid entry metadata or book")
	}
	if c.EntryTicks > (math.MaxInt64-level)/step || c.StopTicks >= (level+step-1)/step {
		return nil, fmt.Errorf("entry or stop offset exceeds price range")
	}
	entry, err := density.EntryPrice(level, step, c.EntryTicks, c.EntryPct)
	if err != nil {
		return nil, err
	}
	stop := level - c.StopTicks*step
	ticks := math.Ceil(float64(entry-stop) / float64(step) * c.RewardRisk)
	if stop <= 0 || entry >= Price(b.Asks[0].Price) || ticks > float64(math.MaxInt64-entry)/float64(step) {
		return nil, fmt.Errorf("entry crosses ask or invalid protection prices")
	}
	var qty int64
	for _, l := range b.Bids {
		if Price(l.Price) == level {
			qty = l.Quantity
			break
		}
	}
	if qty <= 0 || entry > math.MaxInt64/i.Lot {
		return nil, fmt.Errorf("density missing or lot value overflows")
	}
	return &Episode{Instrument: i, Density: level, DensityLots: qty, EntryPrice: entry, StopPrice: stop, TakePrice: entry + int64(ticks)*step, Created: now, Entry: Order{Kind: "buy", Lots: 1}}, nil
}

func validBook(b density.Book, now time.Time, fresh time.Duration) bool {
	if b.Time.IsZero() || b.Time.After(now) || now.Sub(b.Time) > fresh || b.Lot <= 0 || len(b.Bids) == 0 || len(b.Asks) == 0 {
		return false
	}
	for side, levels := range [][]density.Level{b.Bids, b.Asks} {
		for j, l := range levels {
			p := Price(l.Price)
			if p <= 0 || l.Quantity <= 0 {
				return false
			}
			if j > 0 {
				prev := Price(levels[j-1].Price)
				if side == 0 && p >= prev || side == 1 && p <= prev {
					return false
				}
			}
		}
	}
	return Price(b.Bids[0].Price) < Price(b.Asks[0].Price)
}
