package density

import (
	"math"
	"sort"
	"time"
)

const tradingDistanceTicks = 25

type key struct {
	id    string
	price Price
}
type observation struct {
	since, last time.Time
	bookTime    time.Time
	signal      Signal
}
type Detector struct {
	cfg                     Config
	maxDistanceTicks        int64
	averageVolumeMultiplier float64
	observations            map[key]observation
	timestamps              map[string]time.Time
	instruments             map[string]Instrument
}

// SetInstruments installs a fresh history snapshot and restarts confirmations.
func (d *Detector) SetInstruments(instruments []Instrument) {
	d.instruments = make(map[string]Instrument, len(instruments))
	for _, i := range instruments {
		i.Supports = append([]Support(nil), i.Supports...)
		d.instruments[i.ID] = i
	}
	clear(d.observations)
}

func NewDetector(cfg Config) *Detector {
	return newDetector(cfg, tradingDistanceTicks, 1)
}

func newScreenerDetector(cfg Config) *Detector {
	return newDetector(cfg, cfg.AlertDistanceTicks, cfg.AlertM5VolumeMultiplier)
}

func newDetector(cfg Config, maxDistanceTicks int64, averageVolumeMultiplier float64) *Detector {
	return &Detector{cfg: cfg, maxDistanceTicks: maxDistanceTicks, averageVolumeMultiplier: averageVolumeMultiplier, observations: make(map[key]observation), timestamps: make(map[string]time.Time)}
}
func (d *Detector) Reset(id string) {
	for k := range d.observations {
		if k.id == id {
			delete(d.observations, k)
		}
	}
	delete(d.timestamps, id)
}
func (d *Detector) Expire(now time.Time) {
	for k, v := range d.observations {
		if now.Sub(v.last) > d.cfg.Freshness || now.Sub(v.bookTime) > d.cfg.Freshness || (!v.signal.ProfileUntil.IsZero() && !now.Before(v.signal.ProfileUntil)) {
			delete(d.observations, k)
		}
	}
}

// Candidates returns all currently observed levels, including those still accumulating age.
func (d *Detector) Candidates() []Signal {
	result := make([]Signal, 0, len(d.observations))
	for _, o := range d.observations {
		result = append(result, o.signal)
	}
	return result
}
func value(p Price) float64 { return float64(p.Units) + float64(p.Nano)/1e9 }
func comparePrice(a, b Price) int {
	if a.Units < b.Units || (a.Units == b.Units && a.Nano < b.Nano) {
		return -1
	}
	if a == b {
		return 0
	}
	return 1
}
func valid(p Price) bool { return p.Units >= 0 && p.Nano >= 0 && p.Nano < 1e9 && value(p) > 0 }
func withinTickDistance(best, level, step Price, maxDistanceTicks int64) bool {
	if maxDistanceTicks <= 0 {
		return best == level
	}
	const nanosPerUnit = 1_000_000_000
	units := best.Units - level.Units
	nanos := int64(best.Nano) - int64(level.Nano)
	if nanos < 0 {
		units--
		nanos += nanosPerUnit
	}
	// Compare whole units in groups of maxDistanceTicks before comparing the remainder.
	// This keeps the inclusive boundary exact without scaling the full prices.
	if groups := units / maxDistanceTicks; groups != step.Units {
		return groups < step.Units
	}
	return (units%maxDistanceTicks)*nanosPerUnit+nanos <= int64(step.Nano)*maxDistanceTicks
}

func (d *Detector) Update(b Book, now time.Time) []Signal {
	d.Expire(now)
	if b.InstrumentID == "" || b.Lot <= 0 || len(b.Bids) == 0 || len(b.Asks) == 0 || b.Time.IsZero() || now.Sub(b.Time) > d.cfg.Freshness || b.Time.After(now.Add(d.cfg.Freshness)) {
		d.Reset(b.InstrumentID)
		return nil
	}
	if !b.Time.After(d.timestamps[b.InstrumentID]) {
		return nil
	}
	for i, l := range b.Bids {
		if !valid(l.Price) || l.Quantity <= 0 || (i > 0 && comparePrice(b.Bids[i-1].Price, l.Price) <= 0) {
			d.Reset(b.InstrumentID)
			return nil
		}
	}
	for i, l := range b.Asks {
		if !valid(l.Price) || l.Quantity <= 0 || (i > 0 && comparePrice(b.Asks[i-1].Price, l.Price) >= 0) {
			d.Reset(b.InstrumentID)
			return nil
		}
	}
	instrument := d.instruments[b.InstrumentID]
	bestBid := b.Bids[0].Price
	best := value(bestBid)
	if !valid(instrument.PriceStep) || comparePrice(b.Asks[0].Price, bestBid) <= 0 {
		d.Reset(b.InstrumentID)
		return nil
	}
	d.timestamps[b.InstrumentID] = b.Time
	seen := make(map[key]bool)
	var signals []Signal
	average := instrument.AverageVolume
	for i, l := range b.Bids {
		if !withinTickDistance(bestBid, l.Price, instrument.PriceStep, d.maxDistanceTicks) {
			break
		}
		if d.cfg.RequireAverageVolume && !average.AllowsMultiple(float64(l.Quantity), now, d.averageVolumeMultiplier) {
			continue
		}
		price := value(l.Price)
		distance := (best - price) / best
		amount := price * float64(l.Quantity) * float64(b.Lot)
		threshold, support, allowed := d.historicalFilter(b.InstrumentID, price, now)
		if !allowed || amount < threshold || math.IsInf(amount, 0) || math.IsNaN(amount) {
			continue
		}
		type neighbor struct {
			delta  float64
			amount float64
		}
		var ns []neighbor
		for j, n := range b.Bids {
			if i != j {
				a := value(n.Price) * float64(n.Quantity) * float64(b.Lot)
				if a > 0 && !math.IsInf(a, 0) {
					ns = append(ns, neighbor{math.Abs(value(n.Price) - price), a})
				}
			}
		}
		sort.SliceStable(ns, func(i, j int) bool { return ns[i].delta < ns[j].delta })
		if len(ns) > d.cfg.MaxNeighbors {
			ns = ns[:d.cfg.MaxNeighbors]
		}
		if len(ns) < d.cfg.MinNeighbors {
			continue
		}
		amounts := make([]float64, len(ns))
		for i, n := range ns {
			amounts[i] = n.amount
		}
		sort.Float64s(amounts)
		median := amounts[len(amounts)/2]
		if len(amounts)%2 == 0 {
			median = (median + amounts[len(amounts)/2-1]) / 2
		}
		multiple := amount / median
		if multiple < d.cfg.MinMultiple {
			continue
		}
		k := key{b.InstrumentID, l.Price}
		seen[k] = true
		o, ok := d.observations[k]
		if !ok {
			o.since = now
		}
		o.last = now
		o.bookTime = b.Time
		o.signal = Signal{InstrumentID: b.InstrumentID, Price: l.Price, Quantity: l.Quantity, Rubles: amount, Multiple: multiple, Distance: distance, Since: o.since}
		o.signal.MinRubles, o.signal.SupportPrice, o.signal.SupportTouches = threshold, support.Price, support.Touches
		if d.cfg.TurnoverFraction > 0 || d.cfg.Support.Enabled {
			o.signal.ProfileUntil = d.instruments[b.InstrumentID].ProfileAt.Add(ProfileLifetime)
		}
		if d.cfg.RequireAverageVolume {
			o.signal.MinLots = average.Quantity
			o.signal.HasAverageVolume = true
			if o.signal.ProfileUntil.IsZero() || average.Until.Before(o.signal.ProfileUntil) {
				o.signal.ProfileUntil = average.Until
			}
		}
		d.observations[k] = o
		if now.Sub(o.since) >= d.cfg.Persistence {
			signals = append(signals, o.signal)
		}
	}
	for k := range d.observations {
		if k.id == b.InstrumentID && !seen[k] {
			delete(d.observations, k)
		}
	}
	return signals
}
