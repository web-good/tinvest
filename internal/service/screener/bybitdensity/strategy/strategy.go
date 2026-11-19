// Package strategy detects persistent Bybit order book levels and trades pinbars in a paper session.
package strategy

import (
	"time"

	"tinvest/internal/service/screener/bybitdensity/model"
)

type Config struct {
	MinNotional, MinMultiple, MaxDistance float64
	Persistence, Freshness                time.Duration
	PaperNotional, SpotFee, LinearFee     float64
	StopTicks, EntryZoneTicks             int64
	RewardRisk, MinTarget                 float64
}

func DefaultConfig() Config {
	return Config{
		MinNotional: 100000, MinMultiple: 5, MaxDistance: .01,
		Persistence: time.Minute, Freshness: 3 * time.Second,
		PaperNotional: 250, SpotFee: .001, LinearFee: .00055,
		StopTicks: 2, RewardRisk: 2, MinTarget: .003,
	}
}

type priceLevel struct {
	side  model.Side
	ticks int64
}

type Strategy struct {
	candidates               []model.Signal
	instrument               model.Instrument
	cfg                      Config
	book                     *model.Book
	needSnapshot, tradeReady bool
	episodes                 map[priceLevel]*episode
	sent                     map[priceLevel]time.Time
	nextCleanup              time.Time
	trades                   tradeIDs
	lastCandle               time.Time
	pending                  *setup
	position                 *position
	executor                 Executor
	orderID                  uint64
	outstanding              *submittedOrder
}

func New(instrument model.Instrument, cfg Config) *Strategy {
	return NewWithExecutor(instrument, cfg, &paperExecutor{instrument: instrument, spent: make(map[priceLevel]float64)})
}

// NewWithExecutor routes decisions for one instrument to the given executor.
func NewWithExecutor(instrument model.Instrument, cfg Config, executor Executor) *Strategy {
	return &Strategy{
		instrument:   instrument,
		cfg:          cfg,
		needSnapshot: true,
		episodes:     make(map[priceLevel]*episode),
		sent:         make(map[priceLevel]time.Time),
		executor:     executor,
	}
}

// Expire ends observation and cancels any pending entry on loss of freshness.
func (s *Strategy) Expire(now time.Time) {
	if s.book != nil && !s.fresh(s.book, now) {
		s.clearObservation()
	}
	if s.pending != nil && !now.Before(s.pending.end.Add(time.Minute)) {
		s.pending = nil
	}
	if !now.Before(s.nextCleanup) {
		for level, until := range s.sent {
			if !now.Before(until) {
				delete(s.sent, level)
			}
		}
		s.nextCleanup = now.Add(time.Minute)
	}
}

func (s *Strategy) clearObservation() {
	clear(s.episodes)
	s.candidates = s.candidates[:0]
	s.book = nil
	s.pending = nil
}

func (s *Strategy) fresh(b *model.Book, now time.Time) bool {
	return !b.At.IsZero() && !b.Received.IsZero() && now.Sub(b.At).Abs() <= s.cfg.Freshness && now.Sub(b.Received).Abs() <= s.cfg.Freshness
}

func (s *Strategy) Apply(e model.Event, now time.Time) ([]model.Signal, []model.Fill) {
	s.Expire(now)
	if e.Reset {
		s.clearObservation()
		s.needSnapshot = true
		s.tradeReady = false
	}
	if e.TradeReady {
		s.tradeReady = true
	}
	var signals []model.Signal
	if b := e.Book; b != nil && (!s.needSnapshot || b.Snapshot) {
		if len(b.Bids) == 0 || len(b.Asks) == 0 || b.Bids[0].Ticks >= b.Asks[0].Ticks {
			s.clearObservation()
			s.needSnapshot = true
			return nil, nil
		}
		s.needSnapshot = false
		s.executor.ObserveBook(b)
		if !s.fresh(b, now) {
			s.clearObservation()
			return nil, nil
		}
		if b.Snapshot || s.book != nil && (b.At.Sub(s.book.At) > s.cfg.Freshness || b.Received.Sub(s.book.Received) > s.cfg.Freshness) {
			s.clearObservation()
		}
		s.book = b
		s.findCandidates(b)
		if s.tradeReady {
			signals = s.observe(b, now)
		}
	}
	for _, trade := range e.Trades {
		s.observeTrade(trade)
	}
	if e.Candle != nil {
		s.onCandle(*e.Candle, now)
	}
	if s.book == nil || (e.Book == nil && e.Candle == nil) {
		return signals, nil
	}
	return signals, s.execute(now)
}
