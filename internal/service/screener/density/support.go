package density

import (
	"math"
	"sort"
	"time"
)

// ProfileLifetime bounds how long historical entry filters may be reused.
const ProfileLifetime = 24 * time.Hour

type SupportConfig struct {
	Enabled                                         bool
	LookbackDays, PivotBars, MinTouches, BounceBars int
	MinSeparation                                   time.Duration
	BounceFraction, Tolerance                       float64
}

func DefaultSupportConfig() SupportConfig {
	return SupportConfig{LookbackDays: 60, PivotBars: 2, MinTouches: 2, BounceBars: 6, MinSeparation: 6 * time.Hour, BounceFraction: .005, Tolerance: .002}
}

type SupportCandle struct {
	Time             time.Time
	Low, High, Close Price
	Complete         bool
}

type Support struct {
	Price   Price
	Touches int
}

// FindSupports groups confirmed swing lows, discarding touches before the last
// close below a zone. All confirmation candles must already have finished.
func FindSupports(input []SupportCandle, cfg SupportConfig, now time.Time) []Support {
	if cfg.PivotBars < 1 || cfg.BounceBars < cfg.PivotBars || cfg.MinTouches < 2 || cfg.LookbackDays < 1 || cfg.Tolerance <= 0 || cfg.BounceFraction <= 0 {
		return nil
	}
	candles := append([]SupportCandle(nil), input...)
	sort.Slice(candles, func(i, j int) bool { return candles[i].Time.Before(candles[j].Time) })
	clean := make([]SupportCandle, 0, len(candles))
	cutoff := now.AddDate(0, 0, -cfg.LookbackDays)
	// Identical replays count once; conflicting copies of an hour are rejected.
	for i := 0; i < len(candles); {
		c := candles[i]
		j := i + 1
		conflict := false
		for j < len(candles) && candles[j].Time.Equal(c.Time) {
			if candles[j] != c {
				conflict = true
			}
			j++
		}
		i = j
		if conflict || !c.Complete || c.Time.IsZero() || c.Time.Before(cutoff) || c.Time.Add(time.Hour).After(now) || !valid(c.Low) || !valid(c.High) || !valid(c.Close) || comparePrice(c.Low, c.Close) > 0 || comparePrice(c.Close, c.High) > 0 {
			continue
		}
		clean = append(clean, c)
	}
	candles = clean
	type zone struct {
		price   Price
		touches []time.Time
	}
	var zones []zone
	for i := cfg.PivotBars; i+cfg.BounceBars < len(candles); i++ {
		c := candles[i]
		low := value(c.Low)
		pivot := true
		for j := 1; j <= cfg.PivotBars; j++ {
			if comparePrice(c.Low, candles[i-j].Low) >= 0 || comparePrice(c.Low, candles[i+j].Low) >= 0 {
				pivot = false
				break
			}
		}
		if !pivot {
			continue
		}
		bounced := false
		for j := i + 1; j <= i+cfg.BounceBars; j++ {
			closePrice := value(candles[j].Close)
			if closePrice < low*(1-cfg.Tolerance) {
				break
			}
			if closePrice >= low*(1+cfg.BounceFraction) {
				bounced = true
				break
			}
		}
		if !bounced {
			continue
		}
		best := -1
		distance := math.Inf(1)
		for j, z := range zones {
			delta := math.Abs(low-value(z.price)) / value(z.price)
			if delta <= cfg.Tolerance+1e-12 && delta < distance {
				best = j
				distance = delta
			}
		}
		if best < 0 {
			zones = append(zones, zone{price: c.Low, touches: []time.Time{c.Time}})
		} else {
			zones[best].touches = append(zones[best].touches, c.Time)
		}
	}
	var result []Support
	for _, z := range zones {
		var broken, last time.Time
		for _, c := range candles {
			if value(c.Close) < value(z.price)*(1-cfg.Tolerance) {
				broken = c.Time
			}
		}
		count := 0
		for _, at := range z.touches {
			if !at.After(broken) || (!last.IsZero() && at.Sub(last) < cfg.MinSeparation) {
				continue
			}
			count++
			last = at
		}
		if count >= cfg.MinTouches {
			result = append(result, Support{Price: z.price, Touches: count})
		}
	}
	return result
}

func (d *Detector) profile(id string, now time.Time) (Instrument, bool) {
	i, ok := d.instruments[id]
	return i, ok && !i.ProfileAt.IsZero() && !now.Before(i.ProfileAt) && now.Sub(i.ProfileAt) < ProfileLifetime
}

func (d *Detector) historicalFilter(id string, price float64, now time.Time) (float64, Support, bool) {
	threshold := d.cfg.MinRubles
	if d.cfg.TurnoverFraction == 0 && !d.cfg.Support.Enabled {
		return threshold, Support{}, true
	}
	i, ok := d.profile(id, now)
	if !ok {
		return 0, Support{}, false
	}
	if d.cfg.TurnoverFraction > 0 {
		if i.Turnover <= 0 || math.IsNaN(i.Turnover) || math.IsInf(i.Turnover, 0) {
			return 0, Support{}, false
		}
		threshold = math.Max(threshold, i.Turnover*d.cfg.TurnoverFraction)
	}
	if !d.cfg.Support.Enabled {
		return threshold, Support{}, true
	}
	best := Support{}
	distance := math.Inf(1)
	for _, s := range i.Supports {
		if !valid(s.Price) || s.Touches < d.cfg.Support.MinTouches {
			continue
		}
		delta := math.Abs(price-value(s.Price)) / value(s.Price)
		if delta <= d.cfg.Support.Tolerance+1e-12 && delta < distance {
			best = s
			distance = delta
		}
	}
	return threshold, best, best.Touches > 0
}
