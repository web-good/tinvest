package strategy

import (
	"math"
	"slices"
	"time"

	"tinvest/internal/service/screener/bybitdensity/model"
)

const tradingDistanceTicks = 25

type episode struct {
	lastSell                time.Time
	initialQty              float64
	signal                  model.Signal
	firstReceipt, confirmed time.Time
	seen, delivered         bool
	entryPlaced             bool
}

func (s *Strategy) findCandidates(b *model.Book, now time.Time) {
	s.candidates = s.candidates[:0]
	s.alertCandidates = s.alertCandidates[:0]
	levels := b.Bids
	maxDistanceTicks := int64(tradingDistanceTicks)
	if !s.cfg.ZoneAlerts {
		maxDistanceTicks = max(maxDistanceTicks, s.cfg.AlertDistanceTicks)
	}
	for i, l := range levels {
		ticks := levels[0].Ticks - l.Ticks
		if ticks > maxDistanceTicks {
			break
		}
		price := s.instrument.Price(l.Ticks)
		value := price * l.Qty
		distance := float64(ticks) / float64(levels[0].Ticks)
		if value < s.cfg.MinNotional || len(levels) < 6 {
			continue
		}
		tradingEligible := ticks <= tradingDistanceTicks && s.volumeAllows(l.Qty, now)
		alertEligible := !s.cfg.ZoneAlerts && ticks <= s.cfg.AlertDistanceTicks && s.alertVolumeAllows(l.Qty, now)
		if !tradingEligible && !alertEligible {
			continue
		}
		multiple := value / s.medianNeighbors(levels, i)
		if multiple < s.cfg.MinMultiple {
			continue
		}
		signal := model.Signal{Key: s.instrument.Key, Side: model.Bid, Ticks: l.Ticks, Price: price, Qty: l.Qty, Notional: value, Distance: distance, Multiple: multiple, AverageVolume: s.averageVolume.Quantity, HasAverageVolume: s.cfg.RequireAverageVolume}
		if tradingEligible {
			s.candidates = append(s.candidates, signal)
		}
		if alertEligible {
			s.alertCandidates = append(s.alertCandidates, signal)
		}
	}
}

func (s *Strategy) volumeAllows(quantity float64, now time.Time) bool {
	return !s.cfg.RequireAverageVolume || s.averageVolume.Allows(quantity, now)
}

func (s *Strategy) alertVolumeAllows(quantity float64, now time.Time) bool {
	return !s.cfg.RequireAverageVolume || s.averageVolume.AllowsMultiple(quantity, now, s.cfg.AlertM5VolumeMultiplier)
}

func (s *Strategy) observe(b *model.Book, now time.Time, episodes map[priceLevel]*episode, candidates []model.Signal, sent map[priceLevel]time.Time) []model.Signal {
	for _, ep := range episodes {
		ep.seen = false
	}
	var signals []model.Signal
	for _, sig := range candidates {
		k := priceLevel{sig.Side, sig.Ticks}
		ep := episodes[k]
		if ep == nil {
			ep = &episode{firstReceipt: b.Received}
			episodes[k] = ep
			sig.Since = b.At
		} else {
			sig.Since, sig.TradeQty = ep.signal.Since, ep.signal.TradeQty
		}
		sig.Observed = min(b.At.Sub(sig.Since), b.Received.Sub(ep.firstReceipt))
		sig.TradeCoverage = sig.Observed
		ep.seen, ep.signal = true, sig
		if ep.confirmed.IsZero() && sig.Observed >= s.cfg.Persistence {
			ep.confirmed = b.At
		}
		if sent != nil && !ep.confirmed.IsZero() && !ep.delivered && !now.Before(sent[k]) {
			signals = append(signals, sig)
		}
	}
	for k, ep := range episodes {
		if !ep.seen {
			delete(episodes, k)
		}
	}
	return signals
}

func levelsOn(b *model.Book, side model.Side) []model.Level {
	if side == model.Bid {
		return b.Bids
	}
	return b.Asks
}

func absTicks(n int64) int64 {
	if n < 0 {
		return -n
	}
	return n
}

// On a sorted side, the nearest remaining neighbor is adjacent to the selected interval.
func (s *Strategy) medianNeighbors(levels []model.Level, i int) float64 {
	var neighbors [10]float64
	n := min(len(levels)-1, len(neighbors))
	left, right := i-1, i+1
	for j := range n {
		var next model.Level
		if right == len(levels) || (left >= 0 && absTicks(levels[i].Ticks-levels[left].Ticks) <= absTicks(levels[right].Ticks-levels[i].Ticks)) {
			next = levels[left]
			left--
		} else {
			next = levels[right]
			right++
		}
		neighbors[j] = s.instrument.Price(next.Ticks) * next.Qty
	}
	slices.Sort(neighbors[:n])
	return (neighbors[n/2] + neighbors[(n-1)/2]) / 2
}

const tradeWindow = 8192

type tradeIDs struct {
	ids  []string
	seen map[string]struct{}
	next int
}

func (t *tradeIDs) add(id string) bool {
	if id == "" {
		return false
	}
	if _, ok := t.seen[id]; ok {
		return false
	}
	if t.seen == nil {
		t.seen = make(map[string]struct{})
	}
	if len(t.ids) == tradeWindow {
		delete(t.seen, t.ids[t.next])
		t.ids[t.next] = id
		t.next = (t.next + 1) % tradeWindow
	} else {
		t.ids = append(t.ids, id)
	}
	t.seen[id] = struct{}{}
	return true
}

func (s *Strategy) observeTrade(trade model.Trade) {
	if !s.tradeReady || trade.Qty <= 0 || math.IsNaN(trade.Qty) || math.IsInf(trade.Qty, 0) || !s.trades.add(trade.ID) {
		return
	}
	level := priceLevel{model.Bid, trade.Ticks}
	if ep := s.episodes[level]; ep != nil && !trade.At.Before(ep.signal.Since) {
		ep.signal.TradeQty += trade.Qty
	}
	if s.cfg.ZoneAlerts {
		if !trade.Sell || s.zoneWidth == 0 {
			return
		}
		level.ticks -= level.ticks % s.zoneWidth
	}
	if ep := s.alertEpisodes[level]; ep != nil && !trade.At.Before(ep.signal.Since) {
		ep.signal.TradeQty += trade.Qty
		if trade.At.After(ep.lastSell) {
			ep.lastSell = trade.At
		}
	}
}

func (s *Strategy) Current(signal model.Signal, now time.Time) (model.Signal, bool) {
	k := priceLevel{signal.Side, signal.Ticks}
	ep := s.alertEpisodes[k]
	if ep == nil || (!s.cfg.ZoneAlerts && !s.alertVolumeAllows(ep.signal.Qty, now) || s.cfg.ZoneAlerts && ep.signal.Distance > approachDistance) || signal.Key != s.instrument.Key || s.book == nil || !s.fresh(s.book, now) || !ep.signal.Since.Equal(signal.Since) || ep.confirmed.IsZero() || ep.delivered || now.Before(s.sent[k]) {
		return model.Signal{}, false
	}
	return ep.signal, true
}

func (s *Strategy) MarkSent(signal model.Signal, now time.Time) {
	k := priceLevel{signal.Side, signal.Ticks}
	s.sent[k] = now.Add(30 * time.Minute)
	if ep := s.alertEpisodes[k]; ep != nil && ep.signal.Since.Equal(signal.Since) {
		ep.delivered = true
	}
}
